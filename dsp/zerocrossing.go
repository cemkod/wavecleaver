package dsp

import (
	"wavecleaver/model"
)

// DetectCycles finds cycle boundaries using F0 estimates and positive-going zero crossings.
func DetectCycles(samples []float64, sampleRate int, pe *PitchEnvelope) *model.Cycles {
	if len(samples) == 0 || len(pe.F0s) == 0 {
		return &model.Cycles{}
	}

	var boundaries []int

	// Find the first positive-going zero crossing
	pos := findNextZeroCrossing(samples, 0, len(samples))
	if pos < 0 {
		return &model.Cycles{}
	}
	boundaries = append(boundaries, pos)

	for pos < len(samples)-1 {
		f0 := pe.F0AtSample(pos)
		if f0 <= 0 {
			// No pitch detected, skip forward
			pos += sampleRate / 100 // 10ms hop
			next := findNextZeroCrossing(samples, pos, len(samples))
			if next < 0 {
				break
			}
			pos = next
			boundaries = append(boundaries, pos)
			continue
		}

		// Expected period in samples
		period := float64(sampleRate) / f0
		expectedNext := pos + int(period)

		// Search for nearest positive-going zero crossing within ±period/4
		searchRadius := int(period / 4)
		searchStart := expectedNext - searchRadius
		searchEnd := expectedNext + searchRadius
		if searchStart < pos+1 {
			searchStart = pos + 1
		}
		if searchEnd > len(samples) {
			searchEnd = len(samples)
		}

		next := findNearestZeroCrossing(samples, searchStart, searchEnd, expectedNext)
		if next < 0 {
			// Fallback: search wider
			next = findNextZeroCrossing(samples, expectedNext-searchRadius, len(samples))
			if next < 0 {
				break
			}
		}

		boundaries = append(boundaries, next)
		pos = next
	}

	// Convert boundary list to cycle pairs
	var cycles []model.CycleBoundary
	for i := 0; i < len(boundaries)-1; i++ {
		cycles = append(cycles, model.CycleBoundary{
			Start: boundaries[i],
			End:   boundaries[i+1],
		})
	}

	return &model.Cycles{Boundaries: cycles}
}

// findNextZeroCrossing finds the next positive-going zero crossing starting from pos.
func findNextZeroCrossing(samples []float64, start, end int) int {
	if start < 0 {
		start = 0
	}
	for i := start; i < end-1; i++ {
		if samples[i] <= 0 && samples[i+1] > 0 {
			return i + 1 // sample index just after crossing
		}
	}
	return -1
}

// findNearestZeroCrossing finds the positive-going zero crossing nearest to target
// within [start, end).
func findNearestZeroCrossing(samples []float64, start, end, target int) int {
	if start < 0 {
		start = 0
	}
	if end > len(samples) {
		end = len(samples)
	}

	bestPos := -1
	bestDist := end - start + 1

	for i := start; i < end-1; i++ {
		if samples[i] <= 0 && samples[i+1] > 0 {
			dist := i + 1 - target
			if dist < 0 {
				dist = -dist
			}
			if dist < bestDist {
				bestDist = dist
				bestPos = i + 1
			}
		}
	}

	return bestPos
}
