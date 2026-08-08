# Target search (reverse / goal-driven parameters)

How to go from **desired outcomes** (e.g. max drawdown ≤ 15%, Sharpe ≥ 1.0, CAGR ≥ 12%) to **strategy parameters** that get as close as possible — and why this is not a true mathematical inverse of a backtest.

Companion: [Parameter search](./parameter-search.md) for grid / random / TPE / staged methods.

---

## What people mean by “reverse configure”

You do **not** solve:

```text
f(params) = Results   ⇒   params = f⁻¹(targetResults)
```

A backtest is a many-to-one, noisy, path-dependent map. Many unrelated configs can share similar Sharpe; small param changes can jump drawdown. There is no unique inverse.

What you **can** do (and what ML / ops research already does):

1. **Constraint satisfaction** — find any params where all hard rules hold  
2. **Goal programming** — minimize distance to soft targets when constraints conflict  
3. **Constrained optimization** — maximize Sharpe **subject to** DD ≤ 15%  
4. **Multi-objective** — Pareto front of return vs risk (not a single “best”)

In backgommon terms: still **search the parameter space**, but score trials by **feasibility + distance to targets**, not only raw Sharpe.

---

## Hard constraints vs soft targets

| Kind | Example | Behavior |
|------|---------|----------|
| **Hard constraint** | `max_drawdown ≤ 0.15` | Infeasible trials are discarded or ranked last |
| **Soft target** | `sharpe_ratio ≈ 1.2` | Penalize `|sharpe - 1.2|` in a distance score |
| **Bound target** | `cagr ≥ 0.12` | Soft floor, or hard constraint if you require it |

Example goal set:

```text
Hard:  max_drawdown <= 0.15
Hard:  total_trades >= 30          (optional: avoid lucky low-trade Sharpes)
Soft:  sharpe_ratio -> 1.5
Soft:  cagr -> 0.15
```

Search then:

1. Sample / suggest configs (random, TPE, staged, …)  
2. Run backtest (Shared data — same as grid)  
3. Mark **feasible** if all hard constraints pass  
4. Rank feasible by primary metric **or** by **goal distance** (lower better)  
5. Optionally re-check top-K on a **holdout** window  

---

## Goal distance (how “closeness” is scored)

A simple, interpretable score used by `pkg/sweep` goals:

```text
distance = Σ_i  w_i * normalized_error_i(target_i, actual_i)
```

Examples of `normalized_error`:

- Sharpe target 1.5, actual 1.2 → `|1.2 - 1.5| / max(|1.5|, ε)`  
- CAGR target 0.15, actual 0.10 → absolute or relative gap  
- One-sided soft floors: error = 0 if `actual ≥ target`, else gap  

**Hard constraints** are applied first: if DD > 15%, the trial is infeasible regardless of Sharpe.

You can also use **constrained ranking**: among feasible trials only, maximize Sharpe (classic “best under risk budget”).

---

## Is it possible? Yes — with the right framing

| Question | Answer |
|----------|--------|
| Can I dial outcomes and get params? | **Search** for params whose results meet / approach those outcomes |
| Guaranteed exact match? | **No** — targets may be unreachable for this strategy/data |
| Empty feasible set? | Loosen constraints, expand space, or change strategy |
| Replace sweep? | No — reverse mode **is** a scoring layer on top of search |

---

## How this fits the framework

```text
Targets / constraints ──► Goals
Search space (axes)   ──► Space + Suggestor (random / TPE / staged)
Shared TimeseriesTable ──► RunTrial (unchanged strategy)
Results               ──► Feasible? + distance / constrained Sharpe
```

CLI-shaped usage (portfolio momentum example):

```bash
go run ./strategies/portfolio_momentum_sweep \
  --cache-format binary --cache-bin ... \
  --start 2000-01-01 --end 2017-12-31 \
  --search random --trials 500 \
  --max-dd 0.15 --min-sharpe 1.0 --min-cagr 0.12
```

Meaning: among random samples, prefer trials that **satisfy** DD / Sharpe / CAGR bounds; rank survivors by Sharpe (or by goal distance if configured).

---

## Recommended workflow

1. **Define hard risk first** — e.g. DD ≤ 15% (non-negotiable).  
2. **Add soft return goals** — Sharpe / CAGR targets.  
3. **Search on a train window** with budget 500–2000.  
4. **If zero feasible** — relax one constraint at a time; plot how many trials pass each rule.  
5. **Validate** top feasible configs on a holdout period.  
6. **Optional:** TPE with the same goals so the sampler spends budget near the feasible region.  
7. **Optional fidelity:** short-window screen, then full-window confirm (only if ranks correlate).

---

## Staged reverse search

Useful when the space is large:

1. Fix conservative risk params; search signal params for feasible DD + Sharpe floor.  
2. Freeze signal; search risk / sizing to reduce DD further or lift CAGR.  
3. Small joint refine around the winner.

Same idea as staged search in [parameter-search.md](./parameter-search.md), with feasibility as the gate.

---

## Limitations

- **Unreachable targets** — no optimizer invents edge the strategy/data do not have.  
- **Overfitting** — meeting tight targets in-sample is easy to fake; use holdout.  
- **Multiple objectives** — “max Sharpe and min DD and max CAGR” needs priorities or a Pareto view; a single distance scalar hides tradeoffs.  
- **Ill-posed inverse** — do not expect a unique parameter vector for a metric tuple.

---

## Summary

**Reverse configuration** in this framework means:

> Budgeted parameter search scored by **constraints + optional distance to metric targets**, not a closed-form inverse of the backtest.

Keep using Shared data and the same `RunTrial`. Change only how trials are **suggested** and **ranked**. For method details (random vs TPE vs grid), see [parameter-search.md](./parameter-search.md).
