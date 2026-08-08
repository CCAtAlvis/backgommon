# Parameter search (grid, random, TPE, and friends)

How to explore strategy parameters in backgommon without always running a full Cartesian grid.

Related: [Target / reverse search](./target-search.md) — start from desired Sharpe / drawdown / CAGR and search for configs that meet those outcomes.

---

## The problem

A full grid is:

```text
N = size(axis1) × size(axis2) × … × size(axisK)
```

With ~10 axes at 2–4 values each, **N easily hits 50k–100k+** full backtests. At ~1s/trial and 6 workers that is hours of wall time, and most trials are uninformative if only a few parameters dominate.

[`pkg/sweep`](../pkg/sweep) still supports full grids via `Expand` + `Run`. It also supports **budgeted search** via `RunSearch` so you evaluate hundreds or thousands of trials instead of the entire product.

**Data loading:** with `DataMode.Shared`, the market table is loaded **once**. Search vs grid does not multiply load cost or duplicate the candle universe per trial.

---

## Methods

### 1. Full grid (`Expand` + `Run`)

Try every combination on discrete axes.

| | |
|--|--|
| **Pros** | Exhaustive on that grid; easy heatmaps; reproducible |
| **Cons** | Cost = product of axis sizes |
| **Use when** | Axes are tiny, or you need a complete 1–2 axis slice |

### 2. Random search (`SuggestRandom` + `RunSearch`)

Sample `MaxTrials` configs uniformly from the same categorical axes (or ranges).

Classic result (Bergstra & Bengio): when only a subset of params matter, **random often beats an equally sized grid** because the grid wastes budget on irrelevant axes.

| | |
|--|--|
| **Pros** | Trivial; fully parallel; no sequential dependency |
| **Cons** | No memory of good regions; can miss a sharp peak if budget is tiny |
| **Use when** | First upgrade from a huge grid; budget ~200–2000 trials |

### 3. Latin Hypercube / Sobol (planned / optional)

Space-filling designs: better coverage than naive random for the same `K`. Still non-adaptive and parallel.

### 4. Bayesian optimization (GP-EI, etc.)

Fit a surrogate of `params → metric`, pick the next point that balances explore vs exploit.

| | |
|--|--|
| **Pros** | Sample-efficient on smooth, low-dimensional **continuous** spaces |
| **Cons** | Suggest step can be slow; discrete/conditional params are awkward; sequential unless batched |
| **Use when** | Few continuous knobs; not the first tool for large discrete trading grids |

### 5. TPE (`SuggestTPE` + `RunSearch`)

Tree-structured Parzen Estimator (Optuna-style): model “good” vs “bad” regions and sample more where good density is high. Works well on **categorical** axes.

| | |
|--|--|
| **Pros** | Practical on discrete spaces; can suggest batches of size `Workers` |
| **Cons** | Adaptive (needs history); can overfit noisy in-sample metrics |
| **Use when** | After random, when you want the budget to concentrate on promising regions |

### 6. Successive halving / Hyperband-lite (`Fidelity` in `RunSearch`)

Run many configs on a **cheap fidelity** (shorter end date / fewer bars), keep top fraction, re-run survivors on the full window.

| | |
|--|--|
| **Pros** | Large wall-clock savings if early ranking correlates with full-run ranking |
| **Cons** | If short window disagrees with full history, you discard real winners — **measure correlation first** |

### 7. Evolutionary / genetic / CMA-ES

Population mutates toward higher fitness. Robust and parallel within a generation; usually needs more trials than good TPE on smooth low-dim problems.

### 8. Staged / coordinate search (`RunStaged`)

Search Tier A (signal) with Tier B (risk) fixed → freeze A → search B → optional small joint refine.

| | |
|--|--|
| **Pros** | Matches how traders already think; turns a product into a sum of smaller products |
| **Cons** | Can miss A×B interactions (fix with a short joint search) |

---

## Walk-forward (required for honest results)

Optimizing Sharpe on the **same** 2000–2017 window you report will overfit.

Recommended pattern:

1. **Train / search** on e.g. 2000–2012  
2. **Validate** top configs on 2013–2017 (or rolling folds)  
3. Report test metrics, not only search-best train Sharpe  

Target constraints (max DD, min Sharpe) should be checked on the **validation** window too — see [target-search.md](./target-search.md).

---

## API sketch

```text
// Full grid (existing)
cases := sweep.Expand(base, axes...)
sweep.Run(sweep.Config[T]{ Cases: cases, Data: Shared, RunTrial: ... })

// Budgeted search
space := sweep.SpaceFromAxes(base, namedAxes...)
sweep.RunSearch(sweep.SearchConfig[T]{
  Space: space,
  Suggestor: sweep.SuggestRandom[T](rng), // or SuggestTPE[T](...)
  MaxTrials: 500,
  Workers: 6,
  Data: Shared,
  RunTrial: ...,
  Goals: optional constraints / targets,
  Fidelity: optional short window then promote,
})
```

---

## Rough cost intuition

| Approach | Trials | Wall @ ~0.9s/trial, 6 workers |
|----------|--------|-------------------------------|
| Full grid ~93k | 93312 | ~3.9 h |
| Random/TPE budget 500 | 500 | ~1.3 min |
| Random/TPE budget 2000 | 2000 | ~5 min |
| Staged A then B | hundreds–low thousands | minutes |

Quality is not a guaranteed global grid optimum; with a fixed validation discipline you usually get **near-best under the objective with 50–100× less compute**.

---

## CLI example (`portfolio_momentum_sweep`)

```bash
# Full grid (existing)
go run ./strategies/portfolio_momentum_sweep --sweep --workers 6 ...

# Random search, 500 trials
go run ./strategies/portfolio_momentum_sweep --sweep --search random --trials 500 --workers 6 ...

# TPE search
go run ./strategies/portfolio_momentum_sweep --sweep --search tpe --trials 500 --seed 42 ...

# Staged Tier A then Tier B
go run ./strategies/portfolio_momentum_sweep --sweep --search staged --trials 400 ...

# Target constraints (reverse / goal mode) — see target-search.md
go run ./strategies/portfolio_momentum_sweep --sweep --search random --trials 500 \
  --max-dd 0.15 --min-sharpe 1.0 --min-cagr 0.12

# Fidelity screen (mid-window) then promote top-K to full window
go run ./strategies/portfolio_momentum_sweep --sweep --search tpe --trials 500 --fidelity ...
```

Shared BGC/SQLite load is unchanged: one table, all trials reuse it.

---

## What search does *not* change

- Strategy logic (still receives a concrete config)  
- Shared table loading / no per-trial candle duplication  
- Portfolio / risk / fill semantics  

Search only changes **which configs get evaluated**.
