// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 WaveCleaver contributors

package ui

import (
	"image/color"
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/widget"
)

// Toolbar holds the Load/Export buttons, frame size/count selects, and status label.
type Toolbar struct {
	loadBtn      *widget.Button
	exportBtn    *widget.Button
	frameSizeDD  *widget.Select
	frameCountDD *widget.Select
	statusLabel  *widget.Label

	OnLoad              func()
	OnExport            func()
	OnFrameCountChanged func(n int)

	content fyne.CanvasObject
}

func NewToolbar() *Toolbar {
	tb := &Toolbar{}

	tb.loadBtn = widget.NewButton("Load WAV", func() {
		if tb.OnLoad != nil {
			tb.OnLoad()
		}
	})

	tb.exportBtn = widget.NewButton("Export Wavetable", func() {
		if tb.OnExport != nil {
			tb.OnExport()
		}
	})

	frameSizes := []string{"256", "512", "1024", "2048", "4096"}
	tb.frameSizeDD = widget.NewSelect(frameSizes, func(_ string) {
		// value is read at export time via FrameSizeValue()
	})
	tb.frameSizeDD.SetSelected("2048")

	frameCounts := []string{"8", "16", "32", "64", "128", "256"}
	tb.frameCountDD = widget.NewSelect(frameCounts, func(s string) {
		if tb.OnFrameCountChanged != nil {
			if n, err := strconv.Atoi(s); err == nil {
				tb.OnFrameCountChanged(n)
			}
		}
	})
	tb.frameCountDD.SetSelected("64")

	tb.statusLabel = widget.NewLabel("")

	bgRect := canvas.NewRectangle(color.RGBA{R: 0x2D, G: 0x2D, B: 0x3D, A: 0xFF})
	tb.content = container.NewStack(
		bgRect,
		container.NewHBox(
			tb.loadBtn,
			tb.exportBtn,
			tb.frameSizeDD,
			tb.frameCountDD,
			layout.NewSpacer(),
			tb.statusLabel,
		),
	)

	return tb
}

// Content returns the toolbar's canvas object for embedding in a layout.
func (tb *Toolbar) Content() fyne.CanvasObject {
	return tb.content
}

// SetStatus updates the status label text.
func (tb *Toolbar) SetStatus(text string) {
	tb.statusLabel.SetText(text)
}

// FrameSizeValue returns the currently selected frame size.
func (tb *Toolbar) FrameSizeValue() int {
	if n, err := strconv.Atoi(tb.frameSizeDD.Selected); err == nil {
		return n
	}
	return 2048
}

// FrameCountValue returns the currently selected frame count.
func (tb *Toolbar) FrameCountValue() int {
	if n, err := strconv.Atoi(tb.frameCountDD.Selected); err == nil {
		return n
	}
	return 64
}
