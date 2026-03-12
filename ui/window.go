// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 WaveCleaver contributors

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/widget"

	"wavecleaver/model"
	"wavecleaver/ui/candidates"
	"wavecleaver/ui/waveform"
)

// UIWindow composes the main layout: toolbar + waveform + candidates + info bar.
type UIWindow struct {
	Toolbar    *Toolbar
	Waveform   *waveform.Widget
	Candidates *candidates.Panel
	InfoLabel  *widget.Label
}

func NewWindow() *UIWindow {
	return &UIWindow{
		Toolbar:    NewToolbar(),
		Waveform:   waveform.NewWidget(),
		Candidates: candidates.NewPanel(),
		InfoLabel:  widget.NewLabel("Load a WAV file to get started."),
	}
}

// Content builds and returns the root canvas object for the window.
func (w *UIWindow) Content() fyne.CanvasObject {
	split := container.NewVSplit(w.Waveform, w.Candidates)
	split.Offset = 0.33

	return container.NewBorder(
		w.Toolbar.Content(), // top
		w.InfoLabel,         // bottom
		nil, nil,            // left, right
		split,               // center
	)
}

// Update pushes new state to all sub-widgets.
func (w *UIWindow) Update(samples []float64, cycles *model.Cycles, sel *model.Selection, fc *model.FrameCandidates, statusText, infoText string) {
	w.Waveform.Update(samples, cycles, sel, fc)
	w.Candidates.Update(fc)
	w.Toolbar.SetStatus(statusText)
	w.InfoLabel.SetText(infoText)
}
