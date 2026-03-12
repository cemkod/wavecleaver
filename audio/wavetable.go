package audio

import (
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"

	"wavecleaver/model"
	"wavecleaver/util"
)

// ExportWavetable resamples each frame candidate to frameSize samples and writes
// either a Surge .wt file or a Serum-compatible WAV depending on the path extension.
func ExportWavetable(path string, sample *model.Sample, fc *model.FrameCandidates, frameSize int) error {
	if fc.Count() == 0 {
		return fmt.Errorf("no frame candidates to export")
	}

	var frames []float64
	for _, c := range fc.Candidates {
		frame := c.PhaseAligned
		if len(frame) == 0 {
			frame = util.Resample(c.Normalized, frameSize)
		}
		frames = append(frames, frame...)
	}

	if len(frames) == 0 {
		return fmt.Errorf("no valid frames to export")
	}

	numFrames := len(frames) / frameSize
	if strings.EqualFold(filepath.Ext(path), ".wt") {
		return writeWavetableWT(path, frames, numFrames, frameSize)
	}
	return writeWavetableWAV(path, frames, sample.SampleRate, numFrames, frameSize)
}

// writeWavetableWT writes a Surge XT .wt binary wavetable file.
// Format: "vawt" magic + uint32 samplesPerTable + uint16 numTables + uint16 flags + float32 samples.
// flags=0 means wavetable mode with float32 data.
func writeWavetableWT(path string, samples []float64, numFrames, frameSize int) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create wavetable: %w", err)
	}
	defer f.Close()

	write := func(data interface{}) {
		if err != nil {
			return
		}
		err = binary.Write(f, binary.LittleEndian, data)
	}

	f.Write([]byte("vawt"))
	write(uint32(frameSize))
	write(uint16(numFrames))
	write(uint16(0)) // flags: wavetable mode, float32 data

	if err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	for _, s := range samples {
		clamped := math.Max(-1, math.Min(1, s))
		if err := binary.Write(f, binary.LittleEndian, float32(clamped)); err != nil {
			return fmt.Errorf("write sample: %w", err)
		}
	}

	return nil
}

// writeWavetableWAV writes a WAV file with a CLM chunk for wavetable compatibility.
func writeWavetableWAV(path string, samples []float64, sampleRate int, numFrames, frameSize int) error {
	f, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create wavetable: %w", err)
	}
	defer f.Close()

	numSamples := len(samples)
	bitsPerSample := 32
	bytesPerSample := bitsPerSample / 8
	dataSize := numSamples * bytesPerSample

	// CLM chunk: "clm " + size(4) + null-terminated string "<!><frameSize> 0 0 0\x00"
	clmText := fmt.Sprintf("<!>%d 0 0 0", frameSize)
	clmData := append([]byte(clmText), 0) // null terminate
	clmChunkSize := len(clmData)

	// RIFF header sizes
	fmtChunkSize := 16
	totalSize := 4 + // "WAVE"
		8 + fmtChunkSize + // fmt chunk
		8 + clmChunkSize + // clm chunk
		8 + dataSize // data chunk

	// Pad CLM chunk to even size
	clmPad := clmChunkSize % 2

	totalSize += clmPad

	// RIFF header
	write := func(data interface{}) {
		if err != nil {
			return
		}
		err = binary.Write(f, binary.LittleEndian, data)
	}

	f.Write([]byte("RIFF"))
	write(uint32(totalSize))
	f.Write([]byte("WAVE"))

	// fmt chunk
	f.Write([]byte("fmt "))
	write(uint32(fmtChunkSize))
	write(uint16(3)) // IEEE float format
	write(uint16(1)) // mono
	write(uint32(sampleRate))
	write(uint32(sampleRate * bytesPerSample)) // byte rate
	write(uint16(bytesPerSample))              // block align
	write(uint16(bitsPerSample))

	// CLM chunk (Serum wavetable marker)
	f.Write([]byte("clm "))
	write(uint32(clmChunkSize))
	f.Write(clmData)
	if clmPad > 0 {
		f.Write([]byte{0})
	}

	// data chunk
	f.Write([]byte("data"))
	write(uint32(dataSize))

	if err != nil {
		return fmt.Errorf("write header: %w", err)
	}

	// Write 32-bit float samples
	for _, s := range samples {
		clamped := math.Max(-1, math.Min(1, s))
		if err := binary.Write(f, binary.LittleEndian, float32(clamped)); err != nil {
			return fmt.Errorf("write sample: %w", err)
		}
	}

	return nil
}
