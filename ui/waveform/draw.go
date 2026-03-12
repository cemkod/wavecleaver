// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 WaveCleaver contributors

package waveform

import (
	"image"
	"image/color"
	"math"

	"wavecleaver/model"
)

var (
	colorWaveform    = color.RGBA{R: 0x4C, G: 0xAF, B: 0x50, A: 0xFF}
	colorBackground  = color.RGBA{R: 0x1E, G: 0x1E, B: 0x2E, A: 0xFF}
	colorCycleLine   = color.RGBA{R: 0xFF, G: 0xA0, B: 0x00, A: 0xA0}
	colorCandidateBg = color.RGBA{R: 0xFF, G: 0x40, B: 0x80, A: 0x28}
	colorSelectionBg = color.RGBA{R: 0x42, G: 0xA5, B: 0xF5, A: 0x30}
	colorCenterLine  = color.RGBA{R: 0xFF, G: 0xFF, B: 0xFF, A: 0x20}
)

// DrawWaveformImage renders the waveform into dst (pixel-based, no Gio deps).
// vp.Width and vp.Height must be set to dst's pixel dimensions before calling.
func DrawWaveformImage(dst *image.RGBA, vp *Viewport,
	samples []float64, cycles *model.Cycles,
	sel *model.Selection, fc *model.FrameCandidates) {

	b := dst.Bounds()
	width := b.Dx()
	height := b.Dy()
	if width <= 0 || height <= 0 {
		return
	}

	// Background
	fillRect(dst, b, colorBackground)

	if len(samples) == 0 {
		return
	}

	// Center line
	centerY := height / 2
	fillRect(dst, image.Rect(0, centerY, width, centerY+1), colorCenterLine)

	// Selection highlights
	if cycles != nil && sel != nil {
		for i, boundary := range cycles.Boundaries {
			if sel.IsSelected(i) {
				x0 := int(vp.SampleToPixelX(float64(boundary.Start)))
				x1 := int(vp.SampleToPixelX(float64(boundary.End)))
				if x1 > 0 && x0 < width {
					if x0 < 0 {
						x0 = 0
					}
					if x1 > width {
						x1 = width
					}
					blendRect(dst, image.Rect(x0, 0, x1, height), colorSelectionBg)
				}
			}
		}
	}

	// Waveform
	samplesPerPixel := vp.VisibleSamples() / float64(width)
	if samplesPerPixel > 1 {
		drawEnvelope(dst, vp, samples)
	} else {
		drawLines(dst, vp, samples)
	}

	// Candidate cycle background highlights
	if cycles != nil && fc != nil {
		for _, c := range fc.Candidates {
			boundary := cycles.Boundaries[c.CycleIndex]
			x0 := int(vp.SampleToPixelX(float64(boundary.Start)))
			x1 := int(vp.SampleToPixelX(float64(boundary.End)))
			if x1 > 0 && x0 < width {
				if x0 < 0 {
					x0 = 0
				}
				if x1 > width {
					x1 = width
				}
				blendRect(dst, image.Rect(x0, 0, x1, height), colorCandidateBg)
			}
		}
	}

	// Cycle boundary lines
	if cycles != nil {
		for _, boundary := range cycles.Boundaries {
			px := int(vp.SampleToPixelX(float64(boundary.Start)))
			if px >= 0 && px < width {
				drawVLine(dst, px, colorCycleLine)
			}
		}
	}
}

// drawLines draws the waveform as antialiased Wu line segments (zoomed in).
func drawLines(dst *image.RGBA, vp *Viewport, samples []float64) {
	b := dst.Bounds()
	height := b.Dy()
	centerY := float64(height) / 2
	halfH := centerY

	startSample := int(math.Floor(vp.StartSample))
	endSample := int(math.Ceil(vp.EndSample))
	if startSample < 0 {
		startSample = 0
	}
	if endSample > len(samples) {
		endSample = len(samples)
	}
	if startSample >= endSample {
		return
	}

	for i := startSample; i < endSample-1; i++ {
		x0 := float64(vp.SampleToPixelX(float64(i)))
		y0 := centerY - samples[i]*halfH
		x1 := float64(vp.SampleToPixelX(float64(i + 1)))
		y1 := centerY - samples[i+1]*halfH
		drawWuLine(dst, x0, y0, x1, y1, colorWaveform)
	}
}

// drawEnvelope draws a filled min/max envelope per pixel column (zoomed out).
func drawEnvelope(dst *image.RGBA, vp *Viewport, samples []float64) {
	b := dst.Bounds()
	width := b.Dx()
	height := b.Dy()
	centerY := float64(height) / 2
	halfH := centerY

	for px := 0; px < width; px++ {
		s0 := int(vp.PixelXToSample(float32(px)))
		s1 := int(vp.PixelXToSample(float32(px + 1)))
		if s0 < 0 {
			s0 = 0
		}
		if s1 > len(samples) {
			s1 = len(samples)
		}
		if s0 >= s1 {
			continue
		}

		minV, maxV := samples[s0], samples[s0]
		for _, v := range samples[s0:s1] {
			if v < minV {
				minV = v
			}
			if v > maxV {
				maxV = v
			}
		}

		// maxV → smaller Y (top of screen), minV → larger Y (bottom)
		yTop := int(centerY - maxV*halfH)
		yBot := int(centerY - minV*halfH)
		if yTop > yBot {
			yTop, yBot = yBot, yTop
		}
		if yTop < 0 {
			yTop = 0
		}
		if yBot >= height {
			yBot = height - 1
		}
		off := dst.PixOffset(px, yTop)
		for y := yTop; y <= yBot; y++ {
			dst.Pix[off] = colorWaveform.R
			dst.Pix[off+1] = colorWaveform.G
			dst.Pix[off+2] = colorWaveform.B
			dst.Pix[off+3] = colorWaveform.A
			off += dst.Stride
		}
	}
}

// drawWuLine draws an antialiased line using Wu's algorithm.
func drawWuLine(dst *image.RGBA, x0, y0, x1, y1 float64, c color.RGBA) {
	steep := math.Abs(y1-y0) > math.Abs(x1-x0)
	if steep {
		x0, y0 = y0, x0
		x1, y1 = y1, x1
	}
	if x0 > x1 {
		x0, x1 = x1, x0
		y0, y1 = y1, y0
	}
	dx := x1 - x0
	dy := y1 - y0
	gradient := 1.0
	if dx != 0 {
		gradient = dy / dx
	}

	bounds := dst.Bounds()

	// First endpoint
	xend := math.Round(x0)
	yend := y0 + gradient*(xend-x0)
	xgap := rfpart(x0 + 0.5)
	xpxl1 := int(xend)
	ypxl1 := int(math.Floor(yend))
	if steep {
		plotWu(dst, bounds, ypxl1, xpxl1, rfpart(yend)*xgap, c)
		plotWu(dst, bounds, ypxl1+1, xpxl1, fpart(yend)*xgap, c)
	} else {
		plotWu(dst, bounds, xpxl1, ypxl1, rfpart(yend)*xgap, c)
		plotWu(dst, bounds, xpxl1, ypxl1+1, fpart(yend)*xgap, c)
	}
	intery := yend + gradient

	// Second endpoint
	xend = math.Round(x1)
	yend = y1 + gradient*(xend-x1)
	xgap = fpart(x1 + 0.5)
	xpxl2 := int(xend)
	ypxl2 := int(math.Floor(yend))
	if steep {
		plotWu(dst, bounds, ypxl2, xpxl2, rfpart(yend)*xgap, c)
		plotWu(dst, bounds, ypxl2+1, xpxl2, fpart(yend)*xgap, c)
	} else {
		plotWu(dst, bounds, xpxl2, ypxl2, rfpart(yend)*xgap, c)
		plotWu(dst, bounds, xpxl2, ypxl2+1, fpart(yend)*xgap, c)
	}

	// Main loop
	if steep {
		for x := xpxl1 + 1; x <= xpxl2-1; x++ {
			plotWu(dst, bounds, int(math.Floor(intery)), x, rfpart(intery), c)
			plotWu(dst, bounds, int(math.Floor(intery))+1, x, fpart(intery), c)
			intery += gradient
		}
	} else {
		for x := xpxl1 + 1; x <= xpxl2-1; x++ {
			plotWu(dst, bounds, x, int(math.Floor(intery)), rfpart(intery), c)
			plotWu(dst, bounds, x, int(math.Floor(intery))+1, fpart(intery), c)
			intery += gradient
		}
	}
}

func plotWu(dst *image.RGBA, b image.Rectangle, x, y int, brightness float64, c color.RGBA) {
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	blendPixel(dst, x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: uint8(brightness * float64(c.A))})
}

func fpart(x float64) float64  { return x - math.Floor(x) }
func rfpart(x float64) float64 { return 1 - fpart(x) }

// fillRect fills a rectangle with a solid opaque color (fast path).
func fillRect(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	b := dst.Bounds()
	r = r.Intersect(b)
	if r.Empty() {
		return
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		off := dst.PixOffset(r.Min.X, y)
		for x := r.Min.X; x < r.Max.X; x++ {
			dst.Pix[off] = c.R
			dst.Pix[off+1] = c.G
			dst.Pix[off+2] = c.B
			dst.Pix[off+3] = c.A
			off += 4
		}
	}
}

// blendRect alpha-composites a semi-transparent rectangle over existing pixels.
func blendRect(dst *image.RGBA, r image.Rectangle, c color.RGBA) {
	b := dst.Bounds()
	r = r.Intersect(b)
	if r.Empty() {
		return
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		for x := r.Min.X; x < r.Max.X; x++ {
			blendPixel(dst, x, y, c)
		}
	}
}

// blendPixel alpha-composites src (non-premultiplied) over the existing pixel.
// Assumes the background is fully opaque (A=255) after fillRect.
func blendPixel(dst *image.RGBA, x, y int, c color.RGBA) {
	off := dst.PixOffset(x, y)
	sa := uint32(c.A)
	invSA := uint32(255 - sa)
	dst.Pix[off] = uint8((uint32(c.R)*sa + uint32(dst.Pix[off])*invSA) / 255)
	dst.Pix[off+1] = uint8((uint32(c.G)*sa + uint32(dst.Pix[off+1])*invSA) / 255)
	dst.Pix[off+2] = uint8((uint32(c.B)*sa + uint32(dst.Pix[off+2])*invSA) / 255)
	dst.Pix[off+3] = uint8((sa*255 + uint32(dst.Pix[off+3])*invSA) / 255)
}

// drawVLine draws a vertical line blended over existing pixels.
func drawVLine(dst *image.RGBA, x int, c color.RGBA) {
	b := dst.Bounds()
	if x < b.Min.X || x >= b.Max.X {
		return
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		blendPixel(dst, x, y, c)
	}
}
