# Extending the framework

Backgommon is designed so **every simulation rule can be replaced** without forking the repo. Defaults live in `Standard*` types; you inject custom behavior via **interfaces** and **`Func*` wrappers** (same pattern as `FillPricer`).

This guide lists every extension point, where to hook it, and minimal examples.

---

## Design principles

| Principle | What it means |
|-----------|----------------|
| **Interfaces where consumed** | `CostCalculator` lives in `pkg/portfolio` because portfolio applies costs. Avoids import cycles with `pkg/interfaces`. |
| **Defaults are zero-config** | `portfolio.New(settings)` uses `StandardCostCalculator`, `StandardCashFlowCalculator`, etc. |
| **Override one method** | `FuncCostCalculator` delegates unset funcs to `StandardCostCalculator`. |
| **Replace the whole engine** | Implement `interfaces.RiskManager` or `interfaces.Strategy` entirely if needed. |

---

## Extension points (quick reference)

| What | Interface | Package | Inject via |
|------|-----------|---------|------------|
| Fill price | `interfaces.FillPricer` | `interfaces` / `execution` | `runner.WithFillPricer`, `WithFillMode` |
| Slippage | `execution.SlippageModel` | `execution` | `runner.WithSlippageModel` |
| Brokerage & taxes | `portfolio.CostCalculator` | `portfolio` | `portfolio.WithCostCalculator` |
| Cash on entry/exit | `portfolio.CashFlowCalculator` | `portfolio` | `portfolio.WithCashFlowCalculator` |
| SIP / interest / fees | `portfolio.PeriodicEventsModel` | `portfolio` | `portfolio.WithPeriodicEventsModel` |
| Position sizing | `risk.PositionSizer` | `risk` | `risk.WithPositionSizer` |
| Drawdown rules | `risk.DrawdownPolicy` | `risk` | `risk.WithDrawdownPolicy` |
| Full risk layer | `interfaces.RiskManager` | your app | `runner.WithRiskManager` |
| Indicators | `interfaces.Indicator` | `indicators` | `WithIndicators` / custom types |

See also: [Order fill pricing](./order-fill.md), [Portfolio settings & costs](./portfolio-and-costs.md).

---

## 1. Costs: brokerage and taxes

### Interface

```go
type CostCalculator interface {
    Brokerage(ctx TradeCostContext) float64
    TransactionTax(ctx TradeCostContext) float64
    CapitalGainsTax(ctx TradeCostContext) float64
}
```

`TradeCostContext` includes `Order`, `Notional`, `RealizedProfit`, `HoldingPeriod`, and `Settings`.

### Override only brokerage (NSE flat fee example)

```go
calc := &portfolio.FuncCostCalculator{
    BrokerageFn: func(ctx portfolio.TradeCostContext) float64 {
        if ctx.Notional < 100_000 {
            return 20
        }
        return 20 + ctx.Notional*0.0003
    },
}

pf := portfolio.New(settings, portfolio.WithCostCalculator(calc))
```

Unset methods (`TransactionTaxFn`, `CapitalGainsTaxFn`) still use the standard settings-based logic.

### Full custom tax model

Implement `CostCalculator` on your own struct and pass it with `WithCostCalculator`.

---

## 2. Cash flow: margin and proceeds

### Interface

```go
type CashFlowCalculator interface {
    EntryCashOutflow(ctx EntryCashContext) float64
    ExitCashInflow(ctx ExitCashContext) float64
    PositionEquityContribution(ctx PositionEquityContext) float64
}
```

Default: `StandardCashFlowCalculator` (uses your `CostCalculator` for fees).

### When to override

- Custom margin rules (portfolio margin, SPAN-style approximations)
- Different equity marking for leveraged books
- Cross-currency cash (future)

```go
pf := portfolio.New(settings, portfolio.WithCashFlowCalculator(&MyMarginModel{
    Costs: calc, // optional: compose with custom CostCalculator
}))
```

---

## 3. Periodic events (SIP, interest, fees)

### Interface

```go
type PeriodicEventsModel interface {
    Apply(ctx PeriodicContext) PeriodicResult
}
```

`PeriodicContext` includes cash, equity, borrowed notional, settings, and **state timestamps** (`LastSIPTime`, etc.). Return updated `PeriodicResult.State` so the portfolio tracks scheduling correctly.

### Example: custom SIP ladder

```go
model := &portfolio.FuncPeriodicEventsModel{
    ApplyFn: func(ctx portfolio.PeriodicContext) portfolio.PeriodicResult {
        delta := 0.0
        state := ctx.State
        if ctx.Time.Month() != state.LastSIPTime.Month() {
            delta = ctx.Settings.SIPAmount * 2 // double in new month
            state.LastSIPTime = ctx.Time
        }
        return portfolio.PeriodicResult{CashDelta: delta, State: state}
    },
}
pf := portfolio.New(settings, portfolio.WithPeriodicEventsModel(model))
```

---

## 4. Slippage

### Interface

```go
type SlippageModel interface {
    AdjustPrice(ctx SlippageContext) float64
}
```

Runs **after** `FillPricer` in the runner.

### Example: volatility-based slippage

```go
slip := &execution.FuncSlippageModel{
    AdjustFn: func(ctx execution.SlippageContext) float64 {
        spread := (ctx.Order.Price) * 0.001 // use bar range in real impl
        if ctx.Order.Type == portfolio.Entry && ctx.Order.Side == portfolio.Long {
            return ctx.BasePrice + spread
        }
        return ctx.BasePrice - spread
    },
}

r := runner.New(strat,
    runner.WithPortfolio(pf),
    runner.WithSlippageModel(slip),
    // ...
)
```

Default: `StandardSlippageModel` reads `portfolio.Settings.Execution` (`SlippageMode`, `PercentSlippageRate`, …).

---

## 5. Risk: position sizing & drawdown

### PositionSizer

```go
m := risk.New(riskSettings, risk.WithPositionSizer(&risk.FuncPositionSizer{
    SuggestFn: func(ctx risk.PositionSizeContext) int {
        return int(ctx.Equity * 0.05 / ctx.EntryPrice) // 5% notional
    },
}))
qty := m.SuggestedQuantity(pf.Value(), price)
```

### DrawdownPolicy

```go
m := risk.New(riskSettings, risk.WithDrawdownPolicy(&MyDrawdownPolicy{}))
```

`DrawdownPolicyContext` carries equity, peak, lock-until, and settings. Return `DrawdownPolicyResult` with `BlockNewEntries`, `LiquidateAll`, updated `PeakEquity`, and `LockUntil`.

### Full risk replacement

Implement `interfaces.RiskManager` (validate, exits, drawdown) and pass to `runner.WithRiskManager`.

---

## 6. Fill pricing (existing pattern)

Already documented in [order-fill.md](./order-fill.md):

- `runner.WithFillMode(execution.FillNextBarOpen)`
- `runner.WithFillPricer(custom)`
- `execution.NewFuncFillPricer(fn)`

---

## 7. Deprecated package functions

These still work but call the standard implementation only — **they cannot be overridden**:

- `portfolio.ComputeBrokerage` → use `CostCalculator`
- `execution.ApplySlippage` → use `SlippageModel`

Prefer injection at construction time so the runner and portfolio share the same rules.

---

## Composition example (gp-trade / NSE)

```go
settings := &portfolio.Settings{ /* brokerage, execution, taxes */ }

costs := &portfolio.FuncCostCalculator{
    BrokerageFn: nseBrokerageFn,
}

pf := portfolio.New(settings,
    portfolio.WithCostCalculator(costs),
)

rm := risk.New(riskSettings,
    risk.WithPositionSizer(mySizer),
)

r := runner.New(strat,
    runner.WithPortfolio(pf),
    runner.WithRiskManager(rm),
    runner.WithSlippageModel(&execution.FuncSlippageModel{AdjustFn: nseSlippage}),
    runner.WithData(table),
)
```

---

## Related

- [Portfolio settings & costs](./portfolio-and-costs.md) — field reference for `Settings`
- [Writing a strategy](./writing-a-strategy.md)
- [pkg/interfaces](./pkg/interfaces/interfaces.md) — strategy & runner contracts
