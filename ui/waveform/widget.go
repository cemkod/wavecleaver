// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 WaveCleaver contributors

package waveform

import (
	"image"
	"math"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"

	"wavecleaver/model"
)

// Widget is the waveform display with zoom/pan/selection interaction.
type Widget struct {
	widget.BaseWidget
	Viewport *Viewport

	mu      sync.RWMutex
	samples []float64
	cycles  *model.Cycles
	sel     *model.Selection
	fc      *model.FrameCandidates

	dragging    bool
	dragStartX  float32
	dragMoved   bool
	pressButton desktop.MouseButton

	scrollSlider  *widget.Slider
	settingSlider bool

	OnSelectionChanged func()
}

const dragThreshold = float32(3)

func NewWidget() *Widget {
	w := &Widget{
		Viewport: NewViewport(),
	}
	w.scrollSlider = widget.NewSlider(0, 1)
	w.scrollSlider.Step = 0.001
	w.scrollSlider.OnChanged = func(v float64) {
		if w.settingSlider {
			return
		}
		if w.Viewport.TotalSamples == 0 {
			return
		}
		current := w.Viewport.StartSample / float64(w.Viewport.TotalSamples)
		w.Viewport.PanByFraction(v - current)
		w.Refresh()
	}
	w.ExtendBaseWidget(w)
	return w
}

// Update stores new data and triggers a redraw.
func (w *Widget) Update(samples []float64, cycles *model.Cycles, sel *model.Selection, fc *model.FrameCandidates) {
	w.mu.Lock()
	w.samples = samples
	w.cycles = cycles
	w.sel = sel
	w.fc = fc
	w.mu.Unlock()
	w.updateSlider()
	w.Refresh()
}

func (w *Widget) updateSlider() {
	if w.Viewport.TotalSamples == 0 {
		return
	}
	w.settingSlider = true
	w.scrollSlider.SetValue(w.Viewport.StartSample / float64(w.Viewport.TotalSamples))
	w.settingSlider = false
}

// CreateRenderer implements fyne.Widget.
func (w *Widget) CreateRenderer() fyne.WidgetRenderer {
	raster := canvas.NewRaster(func(width, height int) image.Image {
		w.mu.RLock()
		samples := w.samples
		cycles := w.cycles
		sel := w.sel
		fc := w.fc
		w.mu.RUnlock()

		w.Viewport.Width = float32(width)
		w.Viewport.Height = float32(height)

		img := image.NewRGBA(image.Rect(0, 0, width, height))
		DrawWaveformImage(img, w.Viewport, samples, cycles, sel, fc)
		return img
	})

	return &waveformRenderer{
		widget: w,
		raster: raster,
		slider: w.scrollSlider,
	}
}

// --- desktop.Mouseable ---

func (w *Widget) MouseDown(ev *desktop.MouseEvent) {
	w.dragging = true
	w.dragStartX = ev.Position.X
	w.dragMoved = false
	w.pressButton = ev.Button
}

func (w *Widget) MouseUp(ev *desktop.MouseEvent) {
	if !w.dragMoved && w.pressButton == desktop.MouseButtonPrimary {
		w.mu.RLock()
		cycles := w.cycles
		sel := w.sel
		w.mu.RUnlock()

		if cycles != nil && sel != nil {
			samplePos := w.Viewport.PixelXToSample(w.dragStartX)
			cycleIdx := HitTestCycle(samplePos, cycles)
			if cycleIdx >= 0 {
				if ev.Modifier&desktop.ShiftModifier != 0 {
					sel.SelectRange(cycleIdx)
				} else {
					sel.Toggle(cycleIdx)
				}
				w.Refresh()
				if w.OnSelectionChanged != nil {
					w.OnSelectionChanged()
				}
			}
		}
	}
	w.dragging = false
	w.dragMoved = false
}

// --- fyne.Draggable ---

func (w *Widget) Dragged(ev *fyne.DragEvent) {
	if !w.dragging {
		return
	}
	dist := ev.Position.X - w.dragStartX
	if abs32(dist) > dragThreshold {
		w.dragMoved = true
	}
	if w.dragMoved {
		w.Viewport.Pan(ev.Dragged.DX)
		w.updateSlider()
		w.Refresh()
	}
}

func (w *Widget) DragEnd() {
	w.dragging = false
	w.dragMoved = false
}

// --- fyne.Scrollable ---

func (w *Widget) Scrolled(ev *fyne.ScrollEvent) {
	factor := math.Exp(-float64(ev.Scrolled.DY) * 0.04)
	w.Viewport.Zoom(factor, ev.Position.X)
	w.updateSlider()
	w.Refresh()
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// --- renderer ---

type waveformRenderer struct {
	widget *Widget
	raster *canvas.Raster
	slider *widget.Slider
}

func (r *waveformRenderer) Layout(size fyne.Size) {
	const sliderH = float32(16)
	rasterH := size.Height - sliderH
	if rasterH < 0 {
		rasterH = 0
	}
	r.raster.Resize(fyne.NewSize(size.Width, rasterH))
	r.raster.Move(fyne.NewPos(0, 0))
	r.slider.Resize(fyne.NewSize(size.Width, sliderH))
	r.slider.Move(fyne.NewPos(0, rasterH))
}

func (r *waveformRenderer) MinSize() fyne.Size {
	return fyne.NewSize(200, 80)
}

func (r *waveformRenderer) Refresh() {
	canvas.Refresh(r.raster)
}

func (r *waveformRenderer) Destroy() {}

func (r *waveformRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.raster, r.slider}
}
