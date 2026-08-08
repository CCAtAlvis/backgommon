package sweep

import (
	"math"
	"math/rand"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// Observation is one evaluated trial for adaptive suggestors (TPE).
type Observation struct {
	Indices []int
	Metric  float64 // higher is better for the suggestor
	OK      bool    // false if trial failed or was infeasible under hard goals
}

// Suggestor proposes the next encoded samples. Return ok=false when search should stop.
type Suggestor[T any] interface {
	Suggest(space Space[T], history []Observation, batchHint int) (indices [][]int, ok bool)
	Name() string
}

// RandomSuggestor samples uniformly with replacement.
type RandomSuggestor[T any] struct {
	RNG *rand.Rand
}

// SuggestRandom returns a uniform random suggestor.
func SuggestRandom[T any](rng *rand.Rand) *RandomSuggestor[T] {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return &RandomSuggestor[T]{RNG: rng}
}

func (r *RandomSuggestor[T]) Name() string { return "random" }

func (r *RandomSuggestor[T]) Suggest(space Space[T], history []Observation, batchHint int) ([][]int, bool) {
	if batchHint < 1 {
		batchHint = 1
	}
	out := make([][]int, batchHint)
	for i := 0; i < batchHint; i++ {
		_, idx := space.SampleRandom(r.RNG)
		out[i] = idx
	}
	return out, true
}

func metricForSuggestor[T any](trial Trial[T], sortBy string, goals *Goals) (float64, bool) {
	if trial.Err != nil || trial.Results == nil {
		return 0, false
	}
	gs := EvaluateGoals(trial.Results, goals)
	if goals != nil && len(goals.Constraints) > 0 && !gs.Feasible {
		return 0, false
	}
	if goals != nil && len(goals.Targets) > 0 && goals.RankFeasibleBy == "" {
		if math.IsInf(gs.Distance, 0) {
			return 0, false
		}
		return -gs.Distance, true
	}
	rankMetric := sortBy
	if goals != nil && goals.RankFeasibleBy != "" {
		rankMetric = goals.RankFeasibleBy
	}
	v, ok := metricValue(trial, rankMetric)
	if !ok {
		return 0, false
	}
	if rankMetric == "max_drawdown" || rankMetric == "drawdown" {
		return -v, true
	}
	return v, true
}

var _ *types.Results
