package dsp

import (
	"math"

	"wavecleaver/model"
	"wavecleaver/util"
)

const comparisonLen = 256

func cycleRMS(samples []float64, b model.CycleBoundary) float64 {
	slice := samples[b.Start:b.End]
	var sum float64
	for _, v := range slice {
		sum += v * v
	}
	return math.Sqrt(sum / float64(len(slice)))
}

func normalizeCycle(samples []float64, b model.CycleBoundary) []float64 {
	slice := samples[b.Start:b.End]
	peak := util.AbsMaxFloat64(slice)
	out := make([]float64, len(slice))
	if peak == 0 {
		return out
	}
	for i, v := range slice {
		out[i] = v / peak
	}
	return out
}

func cycleSimilarity(a, b []float64) float64 {
	ra := util.Resample(a, comparisonLen)
	rb := util.Resample(b, comparisonLen)
	var sum float64
	for i := range ra {
		d := ra[i] - rb[i]
		sum += d * d
	}
	return math.Sqrt(sum / float64(comparisonLen))
}

const (
	MinSimilarityThreshold = 0.005
	MaxSimilarityThreshold = 0.8
)

// GenerateFrameCandidatesTargeted binary-searches the similarity threshold to
// produce approximately targetCount frame candidates.
func GenerateFrameCandidatesTargeted(samples []float64, cycles *model.Cycles, targetCount int, rmsThreshold float64) *model.FrameCandidates {
	if cycles == nil || len(cycles.Boundaries) == 0 || targetCount <= 0 {
		return &model.FrameCandidates{}
	}
	lo, hi := MinSimilarityThreshold, MaxSimilarityThreshold
	var best *model.FrameCandidates
	for i := 0; i < 20; i++ {
		mid := (lo + hi) / 2
		fc := GenerateFrameCandidates(samples, cycles, mid, rmsThreshold)
		best = fc
		if fc.Count() == targetCount {
			return fc
		}
		if fc.Count() > targetCount {
			lo = mid // too many → raise threshold
		} else {
			hi = mid // too few  → lower threshold
		}
		if hi-lo < 1e-6 {
			break
		}
	}
	return best
}

// GenerateFrameCandidates selects representative cycles whose RMSE difference
// from the previous selected cycle exceeds threshold. Cycles with RMS below
// rmsThreshold are skipped (e.g. silent/near-silent regions).
func GenerateFrameCandidates(samples []float64, cycles *model.Cycles, threshold, rmsThreshold float64) *model.FrameCandidates {
	if cycles == nil || len(cycles.Boundaries) == 0 {
		return &model.FrameCandidates{}
	}

	fc := &model.FrameCandidates{}
	var lastNorm []float64

	for i, b := range cycles.Boundaries {
		if b.End > len(samples) || b.Start >= b.End {
			continue
		}
		if cycleRMS(samples, b) < rmsThreshold {
			continue
		}
		norm := normalizeCycle(samples, b)
		if lastNorm == nil || cycleSimilarity(norm, lastNorm) > threshold {
			fc.Candidates = append(fc.Candidates, model.FrameCandidate{
				CycleIndex: i,
				Normalized: norm,
			})
			lastNorm = norm
		}
	}

	return fc
}
