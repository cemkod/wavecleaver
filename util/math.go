package util

import "math"

// Lerp linearly interpolates between a and b by t.
func Lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

// Clamp constrains v to [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// ClampInt constrains v to [lo, hi].
func ClampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Resample resamples src to dst length using linear interpolation.
func Resample(src []float64, dstLen int) []float64 {
	if len(src) == 0 || dstLen == 0 {
		return nil
	}
	dst := make([]float64, dstLen)
	ratio := float64(len(src)-1) / float64(dstLen-1)
	for i := range dst {
		pos := float64(i) * ratio
		idx := int(pos)
		frac := pos - float64(idx)
		if idx+1 < len(src) {
			dst[i] = Lerp(src[idx], src[idx+1], frac)
		} else {
			dst[i] = src[len(src)-1]
		}
	}
	return dst
}

// NextPow2 returns the smallest power of 2 >= n.
func NextPow2(n int) int {
	if n <= 1 {
		return 1
	}
	p := 1
	for p < n {
		p <<= 1
	}
	return p
}

// MaxFloat64 returns the maximum value in a slice.
func MaxFloat64(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

// AbsMaxFloat64 returns the maximum absolute value in a slice.
func AbsMaxFloat64(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	m := math.Abs(s[0])
	for _, v := range s[1:] {
		a := math.Abs(v)
		if a > m {
			m = a
		}
	}
	return m
}
