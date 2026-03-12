package dsp

import (
	"math"
	"math/cmplx"

	"gonum.org/v1/gonum/dsp/fourier"
)

// PhaseAlign aligns a power-of-2 wavetable frame so that it starts at a
// positive-going zero crossing of the composite waveform. Two-step process:
// (1) FFT-rotate the fundamental to phase 0 to locate the correct zero crossing,
// (2) small cyclic shift to snap frame[0] to the actual zero crossing.
func PhaseAlign(frame []float64) []float64 {
	n := len(frame)

	// Step 1: FFT-based fundamental phase alignment
	fft := fourier.NewFFT(n)
	coeff := fft.Coefficients(nil, frame)
	// Rotate to sine phase: target phase(bin[1]) = -π/2 so rising zero crossing lands at t=0
	phi := cmplx.Phase(coeff[1]) + math.Pi/2
	for k := range coeff {
		coeff[k] *= cmplx.Exp(complex(0, -float64(k)*phi))
	}
	raw := fft.Sequence(nil, coeff)
	aligned := make([]float64, n)
	for i, v := range raw {
		aligned[i] = v / float64(n) // normalize IFFT scale
	}

	// Step 2: snap to nearest positive-going zero crossing near sample 0
	// Search within ±N/8 of sample 0 (±45° of fundamental)
	window := n / 8
	offset := findNearestZeroCrossingOffset(aligned, window)
	if offset == 0 {
		return aligned
	}
	return cyclicRotate(aligned, offset)
}

// findNearestZeroCrossingOffset returns the index of the positive-going zero
// crossing nearest to sample 0, searching within [0, window) and wrapping
// to check [n-window, n). Returns 0 if none found.
func findNearestZeroCrossingOffset(frame []float64, window int) int {
	n := len(frame)
	bestDist := n
	bestIdx := 0
	found := false

	check := func(i int) {
		j := (i + 1) % n
		if frame[i] <= 0 && frame[j] > 0 {
			dist := i
			if dist > n/2 {
				dist = n - dist // wrap distance
			}
			if !found || dist < bestDist {
				bestDist = dist
				bestIdx = j // start of rising edge
				found = true
			}
		}
	}

	for i := 0; i < window; i++ {
		check(i)
	}
	for i := n - window; i < n; i++ {
		check(i)
	}
	return bestIdx
}

func cyclicRotate(frame []float64, offset int) []float64 {
	n := len(frame)
	out := make([]float64, n)
	copy(out, frame[offset:])
	copy(out[n-offset:], frame[:offset])
	return out
}
