package model

// FrameCandidate represents a single representative cycle.
type FrameCandidate struct {
	CycleIndex   int       // index into Cycles.Boundaries
	Normalized   []float64 // variable-length, peak-normalized (similarity comparison)
	PhaseAligned []float64 // 2048-sample, phase+zero aligned (display and export)
}

// FrameCandidates holds the reduced set of representative cycles.
type FrameCandidates struct {
	Candidates []FrameCandidate
}

// Count returns the number of frame candidates.
func (fc *FrameCandidates) Count() int {
	if fc == nil {
		return 0
	}
	return len(fc.Candidates)
}
