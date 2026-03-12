// SPDX-License-Identifier: GPL-3.0-or-later
// Copyright (C) 2026 WaveCleaver contributors

package audio

import (
	"encoding/binary"
	"io"
	"math"
	"sync"
	"sync/atomic"

	"github.com/ebitengine/oto/v3"
)

const (
	playbackSampleRate = 44100
	sweepSecs          = 4.0
	rampSamples        = int(0.01 * playbackSampleRate) // 10 ms fade in/out
)

// wavetableReader is an io.Reader that generates one linear sweep through frames.
type wavetableReader struct {
	frames    [][]float64
	freq      float64
	onDone    func()
	mu        sync.Mutex
	phase     float64 // [0, frameSize)
	framePos  float64 // [0, numFrames); stops at numFrames
	samplePos int
	stopped   int32  // atomic; 1 = stop immediately
	volume    uint64 // atomic; float64 bits via math.Float64bits
	speedMult uint64 // atomic; float64 bits; 1.0 = default speed
}

func (r *wavetableReader) stop() { atomic.StoreInt32(&r.stopped, 1) }

func (r *wavetableReader) Read(buf []byte) (int, error) {
	if atomic.LoadInt32(&r.stopped) != 0 {
		return 0, io.EOF
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	n := len(buf) / 4 // float32 = 4 bytes per sample
	frameSize := float64(len(r.frames[0]))
	numFrames := float64(len(r.frames))
	phaseInc := r.freq / playbackSampleRate * frameSize
	speed := math.Float64frombits(atomic.LoadUint64(&r.speedMult))
	framePosInc := numFrames / (sweepSecs * playbackSampleRate) * speed
	// Fade-out ramp expressed in frame-position units so it adapts to speed changes.
	rampFrames := framePosInc * float64(rampSamples)

	for i := 0; i < n; i++ {
		if r.framePos >= numFrames {
			// Sweep complete — zero remaining bytes and signal EOF.
			for j := i * 4; j < len(buf); j++ {
				buf[j] = 0
			}
			if r.onDone != nil {
				go r.onDone()
				r.onDone = nil
			}
			return n * 4, io.EOF
		}

		fi0 := int(r.framePos)
		fi1 := (fi0 + 1) % len(r.frames)
		frac := r.framePos - float64(fi0)

		pi := int(r.phase)
		pi1 := (pi + 1) % len(r.frames[fi0])
		pfrac := r.phase - float64(pi)

		sA := r.frames[fi0][pi] + (r.frames[fi0][pi1]-r.frames[fi0][pi])*pfrac
		sB := r.frames[fi1][pi] + (r.frames[fi1][pi1]-r.frames[fi1][pi])*pfrac
		out := (sA + (sB-sA)*frac) * math.Float64frombits(atomic.LoadUint64(&r.volume))

		// 10 ms fade-in / fade-out to eliminate clicks.
		if r.samplePos < rampSamples {
			out *= float64(r.samplePos) / float64(rampSamples)
		} else if remaining := numFrames - r.framePos; remaining < rampFrames {
			out *= remaining / rampFrames
		}

		binary.LittleEndian.PutUint32(buf[i*4:], math.Float32bits(float32(out)))

		r.phase += phaseInc
		for r.phase >= frameSize {
			r.phase -= frameSize
		}
		r.framePos += framePosInc
		r.samplePos++
	}

	return n * 4, nil
}

// PlaybackController manages a single oto player for wavetable preview.
type PlaybackController struct {
	OnDone func() // called when sweep finishes naturally; may be called from any goroutine

	mu        sync.Mutex
	ctx       *oto.Context
	player    *oto.Player
	reader    *wavetableReader
	vol       uint64 // atomic; float64 bits; persists across Start/Stop
	speedMult uint64 // atomic; float64 bits; persists across Start/Stop
}

func NewPlaybackController() *PlaybackController {
	c := &PlaybackController{}
	atomic.StoreUint64(&c.vol, math.Float64bits(0.3))
	atomic.StoreUint64(&c.speedMult, math.Float64bits(1.0))
	return c
}

// Start begins a single sweep through frames at the given frequency.
// Must be called from a non-UI goroutine.
func (c *PlaybackController) Start(frames [][]float64, freq float64) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.stopLocked()

	if c.ctx == nil {
		opts := &oto.NewContextOptions{
			SampleRate:   playbackSampleRate,
			ChannelCount: 1,
			Format:       oto.FormatFloat32LE,
		}
		ctx, readyCh, err := oto.NewContext(opts)
		if err != nil {
			return err
		}
		<-readyCh
		c.ctx = ctx
	}

	reader := &wavetableReader{
		frames: frames,
		freq:   freq,
		onDone: func() {
			c.mu.Lock()
			c.reader = nil
			c.player = nil
			c.mu.Unlock()
			if c.OnDone != nil {
				c.OnDone()
			}
		},
	}
	atomic.StoreUint64(&reader.volume, atomic.LoadUint64(&c.vol))
	atomic.StoreUint64(&reader.speedMult, atomic.LoadUint64(&c.speedMult))
	c.reader = reader
	p := c.ctx.NewPlayer(reader)
	p.Play()
	c.player = p
	return nil
}

// Stop halts playback immediately.
func (c *PlaybackController) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopLocked()
}

func (c *PlaybackController) stopLocked() {
	if c.reader != nil {
		c.reader.stop()
		c.reader = nil
	}
	if c.player != nil {
		c.player.Pause()
		c.player = nil
	}
}

// SetVolume sets playback volume in [0, 1] and applies it immediately if playing.
func (c *PlaybackController) SetVolume(v float64) {
	atomic.StoreUint64(&c.vol, math.Float64bits(v))
	c.mu.Lock()
	r := c.reader
	c.mu.Unlock()
	if r != nil {
		atomic.StoreUint64(&r.volume, math.Float64bits(v))
	}
}

// SetSpeed sets the sweep speed multiplier and applies it immediately if playing.
func (c *PlaybackController) SetSpeed(v float64) {
	atomic.StoreUint64(&c.speedMult, math.Float64bits(v))
	c.mu.Lock()
	r := c.reader
	c.mu.Unlock()
	if r != nil {
		atomic.StoreUint64(&r.speedMult, math.Float64bits(v))
	}
}

// IsPlaying reports whether playback is currently active.
func (c *PlaybackController) IsPlaying() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reader != nil
}
