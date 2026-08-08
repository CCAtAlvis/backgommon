# `execution` Package

**Purpose:** Built-in order **fill price** simulation for the runner (`FillPricer` implementations).

**User guide:** [Order fill pricing](../../order-fill.md) — read that first.

---

## For runner users

```go
runner.WithFillMode(execution.FillCurrentClose)
runner.WithFillPricer(execution.NewFuncFillPricer(myFn))
```

## Fill modes

See [order-fill.md](../../order-fill.md#choosing-a-fill-mode) for the full table.

## Files

- `fill.go` — `StandardFillPricer`, `FuncFillPricer`, `FillMode`, `FillModeFromString`

## Interface

`FillPricer` is defined in `pkg/interfaces/fill.go`. This package provides default implementations.
