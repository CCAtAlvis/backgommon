package sweep

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/core"
	"github.com/CCAtAlvis/backgommon/pkg/outmgr"
	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// Stage is one phase of a staged (coordinate) search.
// Space.Base should already embed frozen params from prior stages.
type Stage[T any] struct {
	Name      string
	Space     Space[T]
	MaxTrials int
	Suggestor Suggestor[T] // nil → random
}

// StagedConfig runs stages sequentially: the best config of stage i becomes
// the Base of stage i+1's space (axes from that stage only).
type StagedConfig[T any] struct {
	Base      T // initial frozen defaults
	Stages    []Stage[T]
	Workers   int
	OutputDir string
	SortBy    string
	Progress  bool
	Data      DataMode[T]
	RunTrial  func(cfg T, data *types.TimeseriesTable[core.Candle], reportDir string) (*types.Results, error)
	Goals     *Goals
	Seed      int64
}

// RunStaged executes each stage and returns the last stage's Result (ranked).
// Intermediate stage outputs are written under OutputDir/staged_<ts>/stage_N_*.
func RunStaged[T any](cfg StagedConfig[T]) (*Result[T], error) {
	if len(cfg.Stages) == 0 {
		return nil, fmt.Errorf("sweep: StagedConfig.Stages must not be empty")
	}
	if cfg.RunTrial == nil {
		return nil, fmt.Errorf("sweep: RunTrial is required")
	}
	if cfg.OutputDir == "" {
		return nil, fmt.Errorf("sweep: OutputDir is required")
	}

	ts := time.Now().Format("2006-01-02T15-04-05.000")
	root := filepath.Join(cfg.OutputDir, "staged_"+ts)
	base := cfg.Base
	var last *Result[T]

	seed := cfg.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}

	for i, st := range cfg.Stages {
		if st.MaxTrials <= 0 {
			return nil, fmt.Errorf("sweep: stage %d (%s): MaxTrials must be > 0", i, st.Name)
		}
		space := st.Space
		space.Base = base
		suggestor := st.Suggestor
		if suggestor == nil {
			suggestor = SuggestRandom[T](rand.New(rand.NewSource(seed + int64(i)*997)))
		}
		stageDir := filepath.Join(root, fmt.Sprintf("stage_%d", i))
		if cfg.Progress {
			name := st.Name
			if name == "" {
				name = fmt.Sprintf("%d", i)
			}
			outmgr.Printf("sweep staged: stage %q trials=%d suggestor=%s\n", name, st.MaxTrials, suggestor.Name())
		}
		res, err := RunSearch(SearchConfig[T]{
			Space:     space,
			Suggestor: suggestor,
			MaxTrials: st.MaxTrials,
			Workers:   cfg.Workers,
			OutputDir: stageDir,
			SortBy:    cfg.SortBy,
			Progress:  cfg.Progress,
			Data:      cfg.Data,
			RunTrial:  cfg.RunTrial,
			Goals:     cfg.Goals,
			Seed:      seed + int64(i)*997,
		})
		if err != nil {
			return nil, fmt.Errorf("sweep staged stage %d: %w", i, err)
		}
		last = res
		if len(res.Trials) == 0 {
			return nil, fmt.Errorf("sweep staged stage %d: no trials", i)
		}
		best := res.Trials[0]
		if best.Err != nil {
			return res, fmt.Errorf("sweep staged stage %d: best trial failed: %w", i, best.Err)
		}
		base = best.Params
		if cfg.Progress {
			outmgr.Printf("sweep staged: stage %d best → next base (index %d)\n", i, best.Index)
		}
	}
	return last, nil
}

// FreezeBase returns a Space that searches only `active` axes while starting from base.
func FreezeBase[T any](base T, active ...CategoricalAxis[T]) Space[T] {
	return SpaceFromAxes(base, active...)
}
