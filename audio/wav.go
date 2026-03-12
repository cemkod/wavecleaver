package audio

import (
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"

	"wavecleaver/model"
)

// LoadWAV reads a WAV file and returns a Sample with mono float64 data normalized to [-1, 1].
func LoadWAV(path string) (*model.Sample, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open wav: %w", err)
	}
	defer f.Close()

	dec := wav.NewDecoder(f)
	if !dec.IsValidFile() {
		return nil, fmt.Errorf("invalid wav file: %s", path)
	}

	buf, err := dec.FullPCMBuffer()
	if err != nil {
		return nil, fmt.Errorf("decode wav: %w", err)
	}

	numChannels := int(dec.NumChans)
	bitDepth := int(dec.BitDepth)
	sampleRate := int(dec.SampleRate)

	// Convert integer samples to float64 normalized [-1, 1], mix to mono.
	numFrames := len(buf.Data) / numChannels
	samples := make([]float64, numFrames)
	maxVal := math.Pow(2, float64(bitDepth-1))

	for i := 0; i < numFrames; i++ {
		var sum float64
		for ch := 0; ch < numChannels; ch++ {
			sum += float64(buf.Data[i*numChannels+ch])
		}
		samples[i] = (sum / float64(numChannels)) / maxVal
	}

	return &model.Sample{
		Samples:    samples,
		SampleRate: sampleRate,
		FileName:   filepath.Base(path),
	}, nil
}

// WriteWAV writes float64 samples as a 32-bit WAV file.
func WriteWAV(path string, samples []float64, sampleRate int) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create wav: %w", err)
	}
	defer f.Close()

	enc := wav.NewEncoder(f, sampleRate, 32, 1, 1) // 32-bit, mono, PCM

	buf := &audio.IntBuffer{
		Data:           make([]int, len(samples)),
		Format:         &audio.Format{SampleRate: sampleRate, NumChannels: 1},
		SourceBitDepth: 32,
	}

	maxVal := math.Pow(2, 31) - 1
	for i, s := range samples {
		clamped := math.Max(-1, math.Min(1, s))
		buf.Data[i] = int(clamped * maxVal)
	}

	if err := enc.Write(buf); err != nil {
		return fmt.Errorf("write wav: %w", err)
	}
	return enc.Close()
}
