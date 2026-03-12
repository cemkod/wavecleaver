// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 WaveCleaver contributors

package candidates

import (
	"fmt"
	"image"
	"image/color"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"wavecleaver/model"
)

const cellPx = 80 // logical cell size in pixels

var (
	colorPanelBg    = color.RGBA{R: 0x13, G: 0x13, B: 0x1E, A: 0xFF}
	colorCellBg     = color.RGBA{R: 0x1E, G: 0x1E, B: 0x2E, A: 0xFF}
	colorCellBorder = color.RGBA{R: 0x44, G: 0x44, B: 0x66, A: 0xFF}
	colorMiniWave   = color.RGBA{R: 0x4C, G: 0xAF, B: 0x50, A: 0xFF}
)

// Panel renders the frame candidates thumbnail strip.
type Panel struct {
	widget.BaseWidget

	OnPlayToggle        func(playing bool)
	OnNoteChanged       func(note string)
	OnVolumeChanged     func(volume float64)
	OnSweepSpeedChanged func(speed float64)

	mu           sync.RWMutex
	fc           *model.FrameCandidates
	playing      bool
	note         string
	scrollOffset float64
	lastW, lastH int // last rendered raster pixel dimensions
}

func NewPanel() *Panel {
	p := &Panel{note: "C3"}
	p.ExtendBaseWidget(p)
	return p
}

// SetPlaying updates the playing state and refreshes the button icon.
func (p *Panel) SetPlaying(v bool) {
	p.mu.Lock()
	p.playing = v
	p.mu.Unlock()
	p.Refresh()
}

// Update stores new frame candidates and triggers a redraw.
func (p *Panel) Update(fc *model.FrameCandidates) {
	p.mu.Lock()
	p.fc = fc
	p.scrollOffset = 0
	p.mu.Unlock()
	p.Refresh()
}

// CreateRenderer implements fyne.Widget.
func (p *Panel) CreateRenderer() fyne.WidgetRenderer {
	header := canvas.NewText("Frame Candidates: 0", color.RGBA{R: 0xAA, G: 0xAA, B: 0xCC, A: 0xFF})
	header.TextSize = 12

	raster := canvas.NewRaster(func(w, h int) image.Image {
		p.mu.Lock()
		fc := p.fc
		scrollOffset := p.scrollOffset
		p.lastW = w
		p.lastH = h
		p.mu.Unlock()

		img := image.NewRGBA(image.Rect(0, 0, w, h))
		drawPanelImage(img, fc, scrollOffset)
		return img
	})

	var playBtn *widget.Button
	playBtn = widget.NewButtonWithIcon("", theme.MediaPlayIcon(), func() {
		p.mu.Lock()
		nowPlaying := !p.playing
		p.playing = nowPlaying
		p.mu.Unlock()
		if nowPlaying {
			playBtn.SetIcon(theme.MediaStopIcon())
		} else {
			playBtn.SetIcon(theme.MediaPlayIcon())
		}
		if p.OnPlayToggle != nil {
			p.OnPlayToggle(nowPlaying)
		}
	})
	playBtn.Importance = widget.LowImportance

	noteSelect := widget.NewSelect([]string{"C1", "G1", "C2", "G2", "C3", "G3", "C4", "G4"}, func(s string) {
		p.mu.Lock()
		p.note = s
		p.mu.Unlock()
		if p.OnNoteChanged != nil {
			p.OnNoteChanged(s)
		}
	})
	noteSelect.Selected = "C3"

	volSlider := widget.NewSlider(0.0, 1.0)
	volSlider.Value = 0.3
	volSlider.Step = 0.01
	volSlider.OnChanged = func(v float64) {
		if p.OnVolumeChanged != nil {
			p.OnVolumeChanged(v)
		}
	}

	spdSlider := widget.NewSlider(0.25, 2.0)
	spdSlider.Value = 1.0
	spdSlider.Step = 0.05
	spdSlider.OnChanged = func(v float64) {
		if p.OnSweepSpeedChanged != nil {
			p.OnSweepSpeedChanged(v)
		}
	}

	labelColor := color.RGBA{R: 0xAA, G: 0xAA, B: 0xCC, A: 0xFF}
	noteLabel := canvas.NewText("Note", labelColor)
	noteLabel.TextSize = 9
	volLabel := canvas.NewText("Vol", labelColor)
	volLabel.TextSize = 9
	spdLabel := canvas.NewText("Speed", labelColor)
	spdLabel.TextSize = 9

	return &panelRenderer{
		panel:      p,
		header:     header,
		raster:     raster,
		playBtn:    playBtn,
		noteSelect: noteSelect,
		volSlider:  volSlider,
		spdSlider:  spdSlider,
		noteLabel:  noteLabel,
		volLabel:   volLabel,
		spdLabel:   spdLabel,
	}
}

// Scrolled implements fyne.Scrollable.
func (p *Panel) Scrolled(ev *fyne.ScrollEvent) {
	p.mu.Lock()
	fc := p.fc
	p.scrollOffset -= float64(ev.Scrolled.DY) * 20
	if p.scrollOffset < 0 {
		p.scrollOffset = 0
	}
	if fc != nil && p.lastW > 0 && p.lastH > 0 {
		cols := p.lastW / cellPx
		if cols < 1 {
			cols = 1
		}
		cellH := p.lastW / cols
		rows := (fc.Count() + cols - 1) / cols
		totalH := rows * cellH
		maxScroll := float64(totalH - p.lastH)
		if maxScroll < 0 {
			maxScroll = 0
		}
		if p.scrollOffset > maxScroll {
			p.scrollOffset = maxScroll
		}
	}
	p.mu.Unlock()
	p.Refresh()
}

func drawPanelImage(dst *image.RGBA, fc *model.FrameCandidates, scrollOffset float64) {
	b := dst.Bounds()
	w := b.Dx()
	h := b.Dy()

	fillRect(dst, b, colorPanelBg)

	if fc == nil || fc.Count() == 0 {
		return
	}

	cols := w / cellPx
	if cols < 1 {
		cols = 1
	}
	cellW := w / cols
	cellH := cellW
	offset := int(scrollOffset)

	for i := range fc.Candidates {
		col := i % cols
		row := i / cols
		x0 := col * cellW
		y0 := row*cellH - offset
		x1 := x0 + cellW
		y1 := y0 + cellH

		if y1 < 0 || y0 >= h {
			continue
		}

		// Cell background (clipped to visible area)
		cy0 := y0
		if cy0 < 0 {
			cy0 = 0
		}
		cy1 := y1
		if cy1 > h {
			cy1 = h
		}
		fillRect(dst, image.Rect(x0, cy0, x1, cy1), colorCellBg)

		// Bottom border
		by0 := y1 - 1
		if by0 < 0 {
			by0 = 0
		}
		if by0 < h {
			by1 := y1
			if by1 > h {
				by1 = h
			}
			fillRect(dst, image.Rect(x0, by0, x1, by1), colorCellBorder)
		}

		// Right border
		rx0 := x1 - 1
		if rx0 >= 0 && rx0 < w {
			fillRect(dst, image.Rect(rx0, cy0, x1, cy1), colorCellBorder)
		}

		// Mini waveform
		drawMiniWaveform(dst, fc.Candidates[i].PhaseAligned, x0, y0, cellW, cellH)
	}
}

func drawMiniWaveform(dst *image.RGBA, samples []float64, x0, y0, w, h int) {
	if len(samples) < 2 || w <= 0 || h <= 0 {
		return
	}
	b := dst.Bounds()
	centerY := float64(y0) + float64(h)/2
	halfH := float64(h)/2 - 4

	for i := 0; i < len(samples)-1; i++ {
		px0 := x0 + int(float64(i)/float64(len(samples)-1)*float64(w))
		py0 := int(centerY - samples[i]*halfH)
		px1 := x0 + int(float64(i+1)/float64(len(samples)-1)*float64(w))
		py1 := int(centerY - samples[i+1]*halfH)
		drawBresenhamLine(dst, b, px0, py0, px1, py1, colorMiniWave)
	}
}

func drawBresenhamLine(dst *image.RGBA, b image.Rectangle, x0, y0, x1, y1 int, c color.RGBA) {
	dx := absInt(x1 - x0)
	dy := absInt(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx - dy

	for {
		if x0 >= b.Min.X && x0 < b.Max.X && y0 >= b.Min.Y && y0 < b.Max.Y {
			dst.SetRGBA(x0, y0, c)
		}
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x0 += sx
		}
		if e2 < dx {
			err += dx
			y0 += sy
		}
	}
}

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

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

// --- renderer ---

type panelRenderer struct {
	panel      *Panel
	header     *canvas.Text
	raster     *canvas.Raster
	playBtn    *widget.Button
	noteSelect *widget.Select
	volSlider  *widget.Slider
	spdSlider  *widget.Slider
	noteLabel  *canvas.Text
	volLabel   *canvas.Text
	spdLabel   *canvas.Text
}

func (r *panelRenderer) Layout(size fyne.Size) {
	const headerH = float32(46)
	// Bottom row (y=24, height=20): controls right-to-left with 4px gaps.
	r.playBtn.Resize(fyne.NewSize(24, 20))
	r.playBtn.Move(fyne.NewPos(size.Width-28, 24))
	r.volSlider.Resize(fyne.NewSize(100, 20))
	r.volSlider.Move(fyne.NewPos(size.Width-132, 24))
	r.spdSlider.Resize(fyne.NewSize(110, 20))
	r.spdSlider.Move(fyne.NewPos(size.Width-246, 24))
	r.noteSelect.Resize(fyne.NewSize(72, 20))
	r.noteSelect.Move(fyne.NewPos(size.Width-322, 24))
	// Top row (y=6): header text left, labels above their sliders.
	r.header.Resize(fyne.NewSize(size.Width-326, 14))
	r.header.Move(fyne.NewPos(4, 6))
	r.noteLabel.Move(fyne.NewPos(size.Width-322, 6))
	r.volLabel.Move(fyne.NewPos(size.Width-132, 6))
	r.spdLabel.Move(fyne.NewPos(size.Width-246, 6))
	r.raster.Resize(fyne.NewSize(size.Width, size.Height-headerH))
	r.raster.Move(fyne.NewPos(0, headerH))
}

func (r *panelRenderer) MinSize() fyne.Size {
	return fyne.NewSize(100, 100)
}

func (r *panelRenderer) Refresh() {
	r.panel.mu.RLock()
	fc := r.panel.fc
	playing := r.panel.playing
	r.panel.mu.RUnlock()

	count := 0
	if fc != nil {
		count = fc.Count()
	}
	r.header.Text = fmt.Sprintf("Frame Candidates: %d", count)
	r.header.Refresh()
	if playing {
		r.playBtn.SetIcon(theme.MediaStopIcon())
	} else {
		r.playBtn.SetIcon(theme.MediaPlayIcon())
	}
	r.playBtn.Refresh()
	canvas.Refresh(r.raster)
}

func (r *panelRenderer) Destroy() {}

func (r *panelRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.raster, r.header, r.playBtn, r.noteSelect, r.volSlider, r.spdSlider, r.noteLabel, r.volLabel, r.spdLabel}
}

// Compile-time interface checks.
var _ fyne.Scrollable = (*Panel)(nil)
var _ fyne.Widget = (*Panel)(nil)
