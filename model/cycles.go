package model

// CycleBoundary represents a single cycle's start and end sample indices.
type CycleBoundary struct {
	Start int
	End   int
}

// Cycles holds the detected cycle boundaries for a sample.
type Cycles struct {
	Boundaries []CycleBoundary
}

// Count returns the number of detected cycles.
func (c *Cycles) Count() int {
	if c == nil {
		return 0
	}
	return len(c.Boundaries)
}
