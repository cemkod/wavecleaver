package dsp

import (
	"math"
	"math/cmplx"

	"gonum.org/v1/gonum/dsp/fourier"

	"wavecleaver/util"
)

// Autocorrelation computes the FFT-based autocorrelation of x.
// Returns the autocorrelation array (same length as input, zero-padded to power of 2).
func Autocorrelation(x []float64) []float64 {
	n := len(x)
	padded := util.NextPow2(n * 2)

	// Zero-pad input
	buf := make([]float64, padded)
	copy(buf, x)

	fft := fourier.NewFFT(padded)
	freq := fft.Coefficients(nil, buf)

	// Power spectrum: |FFT(x)|^2
	for i := range freq {
		mag := cmplx.Abs(freq[i])
		freq[i] = complex(mag*mag, 0)
	}

	// IFFT to get autocorrelation
	result := fft.Sequence(nil, freq)

	// Normalize by lag-0 value
	if result[0] != 0 {
		norm := result[0]
		for i := range result {
			result[i] /= norm
		}
	}

	return result[:n]
}

// FindF0 estimates the fundamental frequency from an autocorrelation.
// Searches for the first peak between minLag and maxLag (in samples).
// Returns the lag of the peak, or -1 if no peak found.
func FindF0Lag(ac []float64, minLag, maxLag int) int {
	if minLag < 1 {
		minLag = 1
	}
	if maxLag >= len(ac) {
		maxLag = len(ac) - 1
	}
	if minLag >= maxLag {
		return -1
	}

	// Find the highest peak in the valid range
	bestLag := -1
	bestVal := 0.0
	threshold := 0.2 // minimum autocorrelation value to consider

	for i := minLag; i <= maxLag; i++ {
		if ac[i] > threshold && ac[i] > bestVal {
			// Check it's a local peak
			if (i == minLag || ac[i] >= ac[i-1]) && (i == maxLag || ac[i] >= ac[i+1]) {
				bestVal = ac[i]
				bestLag = i
			}
		}
	}

	return bestLag
}

// EstimateF0 estimates F0 for a windowed segment of audio.
// minFreq/maxFreq define the search range in Hz.
// Returns F0 in Hz, or 0 if detection fails.
func EstimateF0(window []float64, sampleRate int, minFreq, maxFreq float64) float64 {
	// Apply Hann window
	windowed := make([]float64, len(window))
	for i, s := range window {
		w := 0.5 * (1 - math.Cos(2*math.Pi*float64(i)/float64(len(window)-1)))
		windowed[i] = s * w
	}

	ac := Autocorrelation(windowed)

	minLag := int(float64(sampleRate) / maxFreq)
	maxLag := int(float64(sampleRate) / minFreq)

	lag := FindF0Lag(ac, minLag, maxLag)
	if lag <= 0 {
		return 0
	}

	// Parabolic interpolation for sub-sample accuracy
	if lag > 0 && lag < len(ac)-1 {
		a := ac[lag-1]
		b := ac[lag]
		c := ac[lag+1]
		delta := 0.5 * (a - c) / (a - 2*b + c)
		if !math.IsNaN(delta) && !math.IsInf(delta, 0) {
			return float64(sampleRate) / (float64(lag) + delta)
		}
	}

	return float64(sampleRate) / float64(lag)
}
