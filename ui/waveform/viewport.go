package waveform

import (
	"wavecleaver/util"
)

// Viewport manages zoom/pan state and coordinate mapping.
type Viewport struct {
	// View range in sample space
	StartSample float64
	EndSample   float64

	// Total sample count
	TotalSamples int

	// Pixel dimensions of the waveform area
	Width  float32
	Height float32
}

func NewViewport() *Viewport {
	return &Viewport{}
}

// Reset sets the viewport to show all samples.
func (v *Viewport) Reset(totalSamples int) {
	v.TotalSamples = totalSamples
	v.StartSample = 0
	v.EndSample = float64(totalSamples)
}

// VisibleSamples returns the number of samples currently visible.
func (v *Viewport) VisibleSamples() float64 {
	return v.EndSample - v.StartSample
}

// SampleToPixelX converts a sample index to a pixel X coordinate.
func (v *Viewport) SampleToPixelX(sample float64) float32 {
	if v.VisibleSamples() == 0 {
		return 0
	}
	t := (sample - v.StartSample) / v.VisibleSamples()
	return float32(t) * v.Width
}

// PixelXToSample converts a pixel X coordinate to a sample index.
func (v *Viewport) PixelXToSample(px float32) float64 {
	if v.Width == 0 {
		return v.StartSample
	}
	t := float64(px) / float64(v.Width)
	return v.StartSample + t*v.VisibleSamples()
}

// Zoom zooms in/out centered on a pixel X position.
// factor > 1 zooms in, factor < 1 zooms out.
func (v *Viewport) Zoom(factor float64, centerPx float32) {
	centerSample := v.PixelXToSample(centerPx)

	newVisible := v.VisibleSamples() / factor
	minVisible := 64.0 // don't zoom in past 64 samples
	maxVisible := float64(v.TotalSamples)

	newVisible = util.Clamp(newVisible, minVisible, maxVisible)

	// Keep center sample at the same pixel position
	t := float64(centerPx) / float64(v.Width)
	v.StartSample = centerSample - t*newVisible
	v.EndSample = v.StartSample + newVisible

	v.clampToRange()
}

// Pan shifts the view by deltaPx pixels.
func (v *Viewport) Pan(deltaPx float32) {
	deltaSamples := float64(deltaPx) / float64(v.Width) * v.VisibleSamples()
	v.StartSample -= deltaSamples
	v.EndSample -= deltaSamples
	v.clampToRange()
}

// PanByFraction pans the view by a fraction of total content (d in [-1, 1]).
func (v *Viewport) PanByFraction(d float64) {
	delta := d * float64(v.TotalSamples)
	v.StartSample += delta
	v.EndSample += delta
	v.clampToRange()
}

func (v *Viewport) clampToRange() {
	visible := v.VisibleSamples()
	if v.StartSample < 0 {
		v.StartSample = 0
		v.EndSample = visible
	}
	if v.EndSample > float64(v.TotalSamples) {
		v.EndSample = float64(v.TotalSamples)
		v.StartSample = v.EndSample - visible
	}
	if v.StartSample < 0 {
		v.StartSample = 0
	}
}
