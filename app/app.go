// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 WaveCleaver contributors

package app

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"

	"wavecleaver/audio"
	"wavecleaver/dsp"
	"wavecleaver/model"
	"wavecleaver/ui"
	"wavecleaver/util"

	"github.com/sqweek/dialog"
)

// App holds all application state and wires UI callbacks.
type App struct {
	fyneWin fyne.Window
	win     *ui.UIWindow

	Sample          *model.Sample
	Cycles          *model.Cycles
	Selection       *model.Selection
	FrameCandidates *model.FrameCandidates

	medianF0    float64
	previewNote string
	playback    *audio.PlaybackController
	statusText  string
	analyzing   bool
}

func New(w fyne.Window) *App {
	return &App{
		fyneWin:     w,
		win:         ui.NewWindow(),
		Selection:   model.NewSelection(),
		playback:    audio.NewPlaybackController(),
		previewNote: "C3",
	}
}

var noteFreqs = map[string]float64{
	"C1": 32.703, "G1": 49.000,
	"C2": 65.406, "G2": 98.000,
	"C3": 130.813, "G3": 196.000,
	"C4": 261.626, "G4": 392.000,
}

func noteToFreq(note string) float64 {
	if f, ok := noteFreqs[note]; ok {
		return f
	}
	return 130.813
}

// SetupUI wires callbacks and returns the root canvas object.
func (a *App) SetupUI() fyne.CanvasObject {
	a.win.Toolbar.OnLoad = func() { go a.loadFile() }
	a.win.Toolbar.OnExport = func() { go a.exportWavetable() }
	a.win.Toolbar.OnFrameCountChanged = func(n int) {
		if !a.analyzing {
			a.regenerateCandidates()
			a.refreshAll()
		}
	}
	a.win.Waveform.OnSelectionChanged = func() {
		a.win.InfoLabel.SetText(a.buildInfoText())
	}
	a.win.Candidates.OnPlayToggle = func(playing bool) {
		if playing {
			go a.startPreview()
		} else {
			go a.stopPreview()
		}
	}
	a.win.Candidates.OnVolumeChanged = func(v float64) {
		a.playback.SetVolume(v)
	}
	a.win.Candidates.OnSweepSpeedChanged = func(v float64) {
		a.playback.SetSpeed(v)
	}
	a.win.Candidates.OnNoteChanged = func(note string) {
		a.previewNote = note
		if a.playback.IsPlaying() {
			go a.startPreview()
		}
	}
	a.playback.OnDone = func() {
		fyne.Do(func() { a.win.Candidates.SetPlaying(false) })
	}
	return a.win.Content()
}

func (a *App) refreshAll() {
	var samples []float64
	if a.Sample != nil {
		samples = a.Sample.Samples
	}
	samples, cycles, sel, fc := samples, a.Cycles, a.Selection, a.FrameCandidates
	status, info := a.statusText, a.buildInfoText()
	fyne.Do(func() {
		a.win.Update(samples, cycles, sel, fc, status, info)
	})
}

func (a *App) loadFile() {
	path, err := dialog.File().
		Filter("WAV files", "wav").
		Title("Open WAV File").
		Load()
	if err != nil {
		if err.Error() != "Cancelled" {
			log.Printf("dialog error: %v", err)
		}
		return
	}

	a.statusText = "Loading..."
	a.refreshAll()

	sample, err := audio.LoadWAV(path)
	if err != nil {
		a.statusText = fmt.Sprintf("Error: %v", err)
		a.refreshAll()
		log.Printf("load error: %v", err)
		return
	}

	a.Sample = sample
	a.Cycles = nil
	a.FrameCandidates = nil
	a.Selection = model.NewSelection()
	a.win.Waveform.Viewport.Reset(len(sample.Samples))
	a.statusText = fmt.Sprintf("Loaded: %s", sample.FileName)
	a.stopPreview()
	fyne.Do(func() { a.win.Candidates.SetPlaying(false) })
	a.refreshAll()

	a.analyze()
}

func (a *App) regenerateCandidates() {
	if a.Sample == nil || a.Cycles == nil {
		return
	}
	target := a.win.Toolbar.FrameCountValue()
	a.FrameCandidates = dsp.GenerateFrameCandidatesTargeted(a.Sample.Samples, a.Cycles, target, 0.01)
	for i := range a.FrameCandidates.Candidates {
		a.FrameCandidates.Candidates[i].PhaseAligned = dsp.PhaseAlign(util.Resample(a.FrameCandidates.Candidates[i].Normalized, 2048))
	}
	a.statusText = fmt.Sprintf("%s — %d cycles, %d frames",
		a.Sample.FileName, a.Cycles.Count(), a.FrameCandidates.Count())
}

func (a *App) analyze() {
	if a.Sample == nil || a.analyzing {
		return
	}
	a.analyzing = true
	a.statusText = "Analyzing pitch..."
	a.refreshAll()

	pe := dsp.EstimatePitchEnvelope(a.Sample.Samples, a.Sample.SampleRate, 20, 5000)
	a.medianF0 = pe.MedianF0
	cycles := dsp.DetectCycles(a.Sample.Samples, a.Sample.SampleRate, pe)

	a.Cycles = cycles
	a.analyzing = false
	a.regenerateCandidates()
	a.refreshAll()
}

func (a *App) exportWavetable() {
	if a.Sample == nil || a.Cycles == nil || a.FrameCandidates.Count() == 0 {
		a.statusText = "Load a file first"
		a.refreshAll()
		return
	}

	path, filterIdx, err := dialog.File().
		Filter("Surge XT Wavetable .wt", "wt").
		Filter("WAV Wavetable .wav", "wav").
		Filter("Individual frames (folder of .wav)", "wav").
		Title("Export Wavetable").
		SaveWithFilter()
	if err != nil {
		if err.Error() != "Cancelled" {
			log.Printf("save dialog error: %v", err)
		}
		return
	}

	if filterIdx == 2 {
		path = strings.TrimSuffix(path, ".wav")
		path = strings.TrimSuffix(path, ".WAV")

		a.statusText = "Exporting frames..."
		a.refreshAll()

		err = audio.ExportIndividualFrames(path, a.Sample, a.FrameCandidates, a.win.Toolbar.FrameSizeValue())
		if err != nil {
			a.statusText = fmt.Sprintf("Export error: %v", err)
			a.refreshAll()
			log.Printf("export error: %v", err)
			return
		}

		a.statusText = fmt.Sprintf("Exported %d frames to %s/", a.FrameCandidates.Count(), path)
		a.refreshAll()
		return
	}

	if filepath.Ext(path) == "" {
		if filterIdx == 0 {
			path += ".wt"
		} else {
			path += ".wav"
		}
	}

	a.statusText = "Exporting..."
	a.refreshAll()

	err = audio.ExportWavetable(path, a.Sample, a.FrameCandidates, a.win.Toolbar.FrameSizeValue())
	if err != nil {
		a.statusText = fmt.Sprintf("Export error: %v", err)
		a.refreshAll()
		log.Printf("export error: %v", err)
		return
	}

	a.statusText = fmt.Sprintf("Exported %d frames to %s", a.FrameCandidates.Count(), path)
	a.refreshAll()
}

func (a *App) startPreview() {
	if a.FrameCandidates.Count() == 0 {
		fyne.Do(func() { a.win.Candidates.SetPlaying(false) })
		return
	}
	frames := make([][]float64, a.FrameCandidates.Count())
	for i, c := range a.FrameCandidates.Candidates {
		frames[i] = c.PhaseAligned
	}
	freq := noteToFreq(a.previewNote)
	if err := a.playback.Start(frames, freq); err != nil {
		log.Printf("playback error: %v", err)
		fyne.Do(func() { a.win.Candidates.SetPlaying(false) })
	}
}

func (a *App) stopPreview() { a.playback.Stop() }

func (a *App) buildInfoText() string {
	if a.Sample == nil {
		return "Load a WAV file to get started. Left-click cycles to select, drag to pan, scroll to zoom."
	}

	info := fmt.Sprintf("%s | %d Hz | %d samples",
		a.Sample.FileName,
		a.Sample.SampleRate,
		len(a.Sample.Samples))

	if a.Cycles != nil {
		info += fmt.Sprintf(" | %d cycles", a.Cycles.Count())
	}
	if a.Selection.Count() > 0 {
		info += fmt.Sprintf(" | %d selected", a.Selection.Count())
	}

	vp := a.win.Waveform.Viewport
	if vp.TotalSamples > 0 {
		zoom := float64(vp.TotalSamples) / vp.VisibleSamples()
		info += fmt.Sprintf(" | Zoom: %.1fx", zoom)
	}

	return info
}
