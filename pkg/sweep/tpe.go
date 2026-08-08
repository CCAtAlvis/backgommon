package sweep

import (
	"math"
	"math/rand"
	"sort"
	"time"
)

// TPESuggestor is a simple categorical Tree-structured Parzen Estimator.
// It splits history into good/bad by gamma quantile and samples axes from the
// good empirical distribution (with Laplace smoothing toward uniform).
type TPESuggestor[T any] struct {
	RNG    *rand.Rand
	Gamma  float64 // fraction considered "good" (default 0.15)
	NStartup int   // random samples before TPE kicks in (default 20)
	EICandidates int // candidates drawn from good dist per suggestion (default 24)
}

// SuggestTPE returns a TPE suggestor.
func SuggestTPE[T any](rng *rand.Rand) *TPESuggestor[T] {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return &TPESuggestor[T]{
		RNG:          rng,
		Gamma:        0.15,
		NStartup:     20,
		EICandidates: 24,
	}
}

func (t *TPESuggestor[T]) Name() string { return "tpe" }

func (t *TPESuggestor[T]) Suggest(space Space[T], history []Observation, batchHint int) ([][]int, bool) {
	if batchHint < 1 {
		batchHint = 1
	}
	gamma := t.Gamma
	if gamma <= 0 || gamma >= 1 {
		gamma = 0.15
	}
	startup := t.NStartup
	if startup < 1 {
		startup = 20
	}
	nCand := t.EICandidates
	if nCand < 1 {
		nCand = 24
	}

	okHist := make([]Observation, 0, len(history))
	for _, h := range history {
		if h.OK && len(h.Indices) == len(space.Axes) {
			okHist = append(okHist, h)
		}
	}

	out := make([][]int, 0, batchHint)
	for b := 0; b < batchHint; b++ {
		if len(okHist) < startup {
			_, idx := space.SampleRandom(t.RNG)
			out = append(out, idx)
			continue
		}
		good, bad := splitGoodBad(okHist, gamma)
		bestScore := math.Inf(-1)
		var best []int
		for c := 0; c < nCand; c++ {
			cand := sampleFromHist(t.RNG, space, good)
			lGood := logDensity(cand, space, good)
			lBad := logDensity(cand, space, bad)
			score := lGood - lBad
			if score > bestScore {
				bestScore = score
				best = cand
			}
		}
		if best == nil {
			_, best = space.SampleRandom(t.RNG)
		}
		out = append(out, best)
	}
	return out, true
}

func splitGoodBad(hist []Observation, gamma float64) (good, bad []Observation) {
	cp := append([]Observation(nil), hist...)
	sort.Slice(cp, func(i, j int) bool { return cp[i].Metric > cp[j].Metric })
	nGood := int(math.Ceil(float64(len(cp)) * gamma))
	if nGood < 1 {
		nGood = 1
	}
	if nGood > len(cp) {
		nGood = len(cp)
	}
	return cp[:nGood], cp[nGood:]
}

func sampleFromHist[T any](rng *rand.Rand, space Space[T], hist []Observation) []int {
	if len(hist) == 0 {
		_, idx := space.SampleRandom(rng)
		return idx
	}
	// Independent categorical sample from empirical counts per axis.
	out := make([]int, len(space.Axes))
	for a, ax := range space.Axes {
		counts := make([]float64, len(ax.Choices))
		for _, h := range hist {
			if a < len(h.Indices) {
				i := h.Indices[a]
				if i >= 0 && i < len(counts) {
					counts[i]++
				}
			}
		}
		// Laplace smooth
		for i := range counts {
			counts[i]++
		}
		out[a] = weightedChoice(rng, counts)
	}
	return out
}

func logDensity[T any](indices []int, space Space[T], hist []Observation) float64 {
	if len(hist) == 0 {
		// uniform
		ll := 0.0
		for _, ax := range space.Axes {
			ll += -math.Log(float64(len(ax.Choices)))
		}
		return ll
	}
	ll := 0.0
	for a, ax := range space.Axes {
		counts := make([]float64, len(ax.Choices))
		total := 0.0
		for _, h := range hist {
			if a < len(h.Indices) {
				i := h.Indices[a]
				if i >= 0 && i < len(counts) {
					counts[i]++
					total++
				}
			}
		}
		for i := range counts {
			counts[i]++ // Laplace
			total++
		}
		idx := indices[a]
		if idx < 0 || idx >= len(counts) {
			ll += -math.Log(float64(len(ax.Choices)))
			continue
		}
		ll += math.Log(counts[idx] / total)
	}
	return ll
}

func weightedChoice(rng *rand.Rand, weights []float64) int {
	sum := 0.0
	for _, w := range weights {
		sum += w
	}
	if sum <= 0 {
		return rng.Intn(len(weights))
	}
	x := rng.Float64() * sum
	acc := 0.0
	for i, w := range weights {
		acc += w
		if x <= acc {
			return i
		}
	}
	return len(weights) - 1
}
