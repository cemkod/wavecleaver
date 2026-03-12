// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 WaveCleaver contributors

package main

import (
	"fyne.io/fyne/v2"
	fyneapp "fyne.io/fyne/v2/app"

	wavecleaverApp "wavecleaver/app"
)

func main() {
	a := fyneapp.NewWithID("io.wavecleaver.app")
	w := a.NewWindow("WaveCleaver")
	w.Resize(fyne.NewSize(1200, 600))
	controller := wavecleaverApp.New(w)
	w.SetContent(controller.SetupUI())
	w.ShowAndRun()
}
