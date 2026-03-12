package waveform

import (
	"wavecleaver/model"
)

// HitTestCycle returns the cycle index at the given sample position, or -1.
func HitTestCycle(samplePos float64, cycles *model.Cycles) int {
	if cycles == nil {
		return -1
	}
	pos := int(samplePos)
	for i, b := range cycles.Boundaries {
		if pos >= b.Start && pos < b.End {
			return i
		}
	}
	return -1
}
