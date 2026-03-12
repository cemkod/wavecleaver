package model

// Sample holds a loaded audio sample.
type Sample struct {
	Samples    []float64 // mono samples normalized to [-1, 1]
	SampleRate int
	FileName   string
}
