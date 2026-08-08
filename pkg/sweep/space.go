package sweep

import (
	"fmt"
	"math/rand"
)

// CategoricalAxis is one named parameter with discrete Mutator choices.
type CategoricalAxis[T any] struct {
	Name    string
	Choices []Mutator[T]
}

// Space describes a discrete search space around a base config.
type Space[T any] struct {
	Base T
	Axes []CategoricalAxis[T]
}

// SpaceFromAxes builds a Space from named categorical axes.
// Empty choice lists are skipped. Names may be empty.
func SpaceFromAxes[T any](base T, axes ...CategoricalAxis[T]) Space[T] {
	out := Space[T]{Base: base}
	for _, ax := range axes {
		if len(ax.Choices) == 0 {
			continue
		}
		out.Axes = append(out.Axes, ax)
	}
	return out
}

// SpaceFromMutatorAxes converts Expand-style [][]Mutator into a Space with
// synthetic names axis_0, axis_1, ...
func SpaceFromMutatorAxes[T any](base T, axes ...[]Mutator[T]) Space[T] {
	named := make([]CategoricalAxis[T], 0, len(axes))
	for i, ax := range axes {
		if len(ax) == 0 {
			continue
		}
		named = append(named, CategoricalAxis[T]{
			Name:    fmt.Sprintf("axis_%d", i),
			Choices: ax,
		})
	}
	return SpaceFromAxes(base, named...)
}

// NumCombinations returns the Cartesian product size (0 if no axes).
func (s Space[T]) NumCombinations() int {
	if len(s.Axes) == 0 {
		return 1
	}
	n := 1
	for _, ax := range s.Axes {
		n *= len(ax.Choices)
		if n <= 0 {
			return 0
		}
	}
	return n
}

// Decode applies one choice index per axis to a copy of Base.
func (s Space[T]) Decode(indices []int) (T, error) {
	cfg := s.Base
	if len(indices) != len(s.Axes) {
		return cfg, fmt.Errorf("sweep: Decode expected %d indices, got %d", len(s.Axes), len(indices))
	}
	for i, ax := range s.Axes {
		idx := indices[i]
		if idx < 0 || idx >= len(ax.Choices) {
			return cfg, fmt.Errorf("sweep: axis %q index %d out of range [0,%d)", ax.Name, idx, len(ax.Choices))
		}
		ax.Choices[idx](&cfg)
	}
	return cfg, nil
}

// SampleRandom draws one uniform categorical sample. Returns config and choice indices.
func (s Space[T]) SampleRandom(rng *rand.Rand) (T, []int) {
	if rng == nil {
		rng = rand.New(rand.NewSource(1))
	}
	indices := make([]int, len(s.Axes))
	for i, ax := range s.Axes {
		indices[i] = rng.Intn(len(ax.Choices))
	}
	cfg, err := s.Decode(indices)
	if err != nil {
		// Should not happen with Intn bounds.
		return s.Base, indices
	}
	return cfg, indices
}

// SampleRandomN draws n samples (with replacement). Dedup is caller's choice.
func (s Space[T]) SampleRandomN(rng *rand.Rand, n int) ([]T, [][]int) {
	cfgs := make([]T, n)
	idxs := make([][]int, n)
	for i := 0; i < n; i++ {
		cfgs[i], idxs[i] = s.SampleRandom(rng)
	}
	return cfgs, idxs
}
