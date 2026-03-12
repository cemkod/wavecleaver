package dsp

import "math"

// PitchEnvelope holds per-window F0 estimates.
type PitchEnvelope struct {
	F0s        []float64 // F0 in Hz per window
	MedianF0   float64   // median of valid estimates
	WindowSize int
	HopSize    int
}

// EstimatePitchEnvelope computes F0 across the sample using a sliding window.
// It runs two passes:
//  1. Wide-range detection to find the median F0.
//  2. Narrow-range re-detection constrained to ±20% of the median period,
//     which eliminates octave-jump errors at the source.
func EstimatePitchEnvelope(samples []float64, sampleRate int, minFreq, maxFreq float64) *PitchEnvelope {
	windowSize := 4096
	hopSize := windowSize / 2

	windows := collectWindows(samples, windowSize, hopSize)

	// Pass 1: wide-range detection
	f0s := make([]float64, len(windows))
	for i, w := range windows {
		f0s[i] = EstimateF0(w, sampleRate, minFreq, maxFreq)
	}

	median := medianF0(f0s)

	// Pass 2: re-detect with lag range locked around the median period
	if median > 0 {
		const band = 0.20 // ±20% allows vibrato/drift but blocks octave jumps
		narrowMin := median / (1 + band)
		narrowMax := median * (1 + band)
		// Clamp to original bounds
		narrowMin = math.Max(narrowMin, minFreq)
		narrowMax = math.Min(narrowMax, maxFreq)

		for i, w := range windows {
			f := EstimateF0(w, sampleRate, narrowMin, narrowMax)
			if f > 0 {
				f0s[i] = f
			} else {
				// Narrow search failed (e.g. noisy region) — keep pass-1
				// value only if it's close to the median, otherwise zero it
				if f0s[i] > 0 {
					ratio := f0s[i] / median
					if ratio < (1-band) || ratio > (1+band) {
						f0s[i] = 0
					}
				}
			}
		}
		median = medianF0(f0s)
	}

	return &PitchEnvelope{
		F0s:        f0s,
		MedianF0:   median,
		WindowSize: windowSize,
		HopSize:    hopSize,
	}
}

func collectWindows(samples []float64, windowSize, hopSize int) [][]float64 {
	var windows [][]float64
	for start := 0; start+windowSize <= len(samples); start += hopSize {
		windows = append(windows, samples[start:start+windowSize])
	}
	return windows
}

// medianF0 returns the median of non-zero F0 values.
func medianF0(f0s []float64) float64 {
	var valid []float64
	for _, f := range f0s {
		if f > 0 {
			valid = append(valid, f)
		}
	}
	if len(valid) == 0 {
		return 0
	}
	for i := 1; i < len(valid); i++ {
		for j := i; j > 0 && valid[j-1] > valid[j]; j-- {
			valid[j-1], valid[j] = valid[j], valid[j-1]
		}
	}
	return valid[len(valid)/2]
}

// F0AtSample returns the interpolated F0 estimate at a given sample position.
func (pe *PitchEnvelope) F0AtSample(sampleIdx int) float64 {
	if len(pe.F0s) == 0 {
		return 0
	}

	windowIdx := float64(sampleIdx-pe.WindowSize/2) / float64(pe.HopSize)
	if windowIdx < 0 {
		return pe.F0s[0]
	}
	if int(windowIdx) >= len(pe.F0s)-1 {
		return pe.F0s[len(pe.F0s)-1]
	}

	idx := int(windowIdx)
	frac := windowIdx - float64(idx)

	a, b := pe.F0s[idx], pe.F0s[idx+1]
	if a == 0 && b == 0 {
		return 0
	}
	if a == 0 {
		return b
	}
	if b == 0 {
		return a
	}

	return a + (b-a)*frac
}
