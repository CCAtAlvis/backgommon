# Portfolio settings, costs & periodic cash flows

Backgommon simulates a brokerage account: margin, fees, taxes, scheduled deposits, and interest. This guide explains **what each setting does**, **when it runs**, and **how it interacts with order fill pricing**.

For fill timing and slippage mechanics, also read [Order fill pricing](./order-fill.md).

**Want custom brokerage, taxes, or cash rules?** See [Extending the framework](./extending-the-framework.md) — every cost and cash function is interface-based and injectable.

---

## Mental model: one bar in the runner

```
Bar T
│
├─ Portfolio.ProcessPeriodicEvents(T)     SIP, idle interest, leverage cost, mgmt fee
├─ OnDayStart / OnDayEnd (if calendar day changed)
├─ UpdatePositions(prices)
├─ RiskManager.CheckDrawdown(...)       may block entries or liquidate
├─ RiskManager.CheckPositionExits(...)  stop / take-profit / trailing
├─ Strategy.OnTick → orders
├─ FillPricer + slippage → order.Price
└─ Portfolio.ProcessOrder → cash & positions
```

Costs are applied **inside `ProcessOrder`** after the runner sets the fill price (including slippage).

### Strict vs lenient order processing

By default the runner is **lenient**: if risk validation or `ProcessOrder` rejects an order (e.g. insufficient cash), that order is skipped, a warning is logged, and the backtest continues. `Results.SkippedOrders` records how many were dropped.

Use `runner.WithStrictOrderProcessing(true)` for fail-fast / CI runs — the first rejected order aborts `Start()`.

Strategies should still size with `Portfolio.EstimateEntryCash` so multi-entry batches account for margin + brokerage + taxes and avoid skips.

---

## Portfolio.Settings fields

Configure via `portfolio.Settings` when creating the portfolio:

```go
pf := portfolio.New(&portfolio.Settings{
    InitialCapital: 100_000,
    DefaultLeverage: 1.0,
    Execution: portfolio.ExecutionSettings{
        OrderFillAssumption: "NextBarOpen",
        SlippageMode:        "PercentOfPrice",
        PercentSlippageRate: 0.0005,
    },
    FixedBrokerageFee:    20,
    PercentBrokerageRate: 0.0003,
    EnableTaxes:          true,
    BuyTaxRate:           0.001,
    SellTaxRate:          0.001,
    STCapitalGainsTaxRate: 0.15,
    LTCapitalGainsTaxRate: 0.10,
    ShortTermHoldingPeriod: 365 * 24 * time.Hour,
    SIPAmount:            5000,
    SIPFrequency:         30 * 24 * time.Hour,
    IdleCashInterestAnnualRate: 0.04,
    IdleCashInterestFrequency:  24 * time.Hour,
})
```

### Core setup

| Field | Purpose |
|-------|---------|
| `InitialCapital` | Starting cash |
| `EnableShorts` | Reject short entry orders when `false` |
| `DefaultLeverage` | Used when `Order.Leverage` is 0 or unset |
| `CashReserveRate` | Reserved for future sizing rules (not enforced yet) |

### Brokerage

Applied on **every** entry and exit fill:

```
brokerage = FixedBrokerageFee + notional × PercentBrokerageRate
```

where `notional = quantity × fillPrice`.

### Transaction taxes

When `EnableTaxes` is true:

- **Buy (long entry):** `notional × BuyTaxRate`
- **Sell (long exit):** `notional × SellTaxRate`

### Capital gains tax

Applied on **realized profit** for each exit slice (partial or full):

- If holding period `< ShortTermHoldingPeriod` → `STCapitalGainsTaxRate`
- Otherwise → `LTCapitalGainsTaxRate`
- Losses are not taxed (rate applies only when profit > 0)

Holding period is measured from position open time to order `FilledAt` (set by the runner).

### Periodic cash flows

| Field | Behavior |
|-------|----------|
| `SIPAmount` + `SIPFrequency` | Adds cash every interval |
| `IdleCashInterestAnnualRate` + `IdleCashInterestFrequency` | Accrues interest on uninvested cash |
| `LeverageCostAnnualRate` + `LeverageCostFrequency` | Charges interest on borrowed notional for leveraged longs |
| `EnableManagementFee` + rates/frequency | Deducts fee from total equity |

Annual rates are converted to per-period rates using the configured frequency.

### Execution settings (embedded)

See [Order fill pricing](./order-fill.md). Key fields:

| Field | Values |
|-------|--------|
| `OrderFillAssumption` | `CurrentBarClose`, `NextBarOpen`, `MidPrice`, `WorstCaseWithinBar`, … |
| `SlippageMode` | `None`, `FixedPoints`, `PercentOfPrice` |
| `FixedSlippageAmount` | Points added (buys) or subtracted (sells) per share |
| `PercentSlippageRate` | Decimal fraction of price (e.g. `0.001` = 0.1%) |

When you pass a `*portfolio.Portfolio` to `runner.WithPortfolio`, the runner **automatically** picks the fill mode from `Execution.OrderFillAssumption` unless you override with `runner.WithFillMode`.

Slippage is applied **after** the fill pricer, using the same execution settings.

---

## Cash accounting (long positions)

### Entry

```
cash -= (qty × price / leverage) + brokerage + buyTax
```

### Exit

```
cash += marginReleased + realizedPnL − brokerage − sellTax − capitalGainsTax
marginReleased = qty × openPrice / leverage   // for the exited quantity
realizedPnL      = qty × (exitPrice − openPrice) × leverage
```

### Portfolio value (equity)

```
equity = cash + Σ (marginDeployed + unrealizedPnL) per open position
marginDeployed = qty × openPrice / leverage
```

This keeps leverage consistent: at 1× leverage, equity equals cash plus mark-to-market position value.

---

## Order conventions

### `Quantity: 0` on exit = full close

```go
portfolio.NewOrder("RELIANCE", portfolio.Long, portfolio.Exit, 0, 1)
```

The portfolio resolves quantity to the full open size before validation.

### Exit side matches position side

For a long position, exit orders use `Side: Long`, `Type: Exit` — not the opposite side. Risk exits follow the same convention via `portfolio.ExitOrderForPosition`.

---

## Risk manager integration

### Drawdown limits

Configure in `risk.Settings`:

```go
risk.New(&risk.Settings{
    MaxPortfolioDrawdownRate: 0.20,
    MaxDrawdownMode:          risk.StopNewTrades,
    DrawdownLockDuration:     7 * 24 * time.Hour,
})
```

| Mode | Behavior |
|------|----------|
| `NoAction` | Log only; strategy decides |
| `AlertOnly` | Warn when breached |
| `StopNewTrades` | Block new entries; optional lock duration |
| `LiquidateAllPositions` | Flatten all open positions on breach |

The runner calls `CheckDrawdown` each bar after marking positions.

### Position sizing helper

When stop-loss is enabled, size entries from risk budget:

```go
qty := riskManager.SuggestedQuantity(portfolio.Value(), entryPrice)
// Uses RiskPerTradeRate × equity / (entryPrice × DefaultStopLossRate)
```

This is a **helper** — your strategy must call it and pass the quantity into `NewOrder`.

---

## Day lifecycle hooks

Strategies embedding `strategy.BaseStrategy` receive:

- `OnDayStart(t)` — first bar of a new calendar day (UTC midnight boundary)
- `OnDayEnd(t)` — previous day’s close when the day rolls

Override these in your strategy struct; the runner invokes them automatically.

---

## What is not implemented yet

| Feature | Status |
|---------|--------|
| Profit pocketing | Settings exist; logic deferred |
| Short selling cash model | Entries rejected unless `EnableShorts`; full margin model TBD |
| Partial fills from volume | `EnablePartialFills` reserved |
| Market impact | `MarketImpactModel` reserved |

---

## Example: realistic Indian equity costs

```go
settings := &portfolio.Settings{
    InitialCapital:       500_000,
    FixedBrokerageFee:    20,
    PercentBrokerageRate: 0.0003, // ~0.03%
    EnableTaxes:          true,
    BuyTaxRate:           0.001,
    SellTaxRate:          0.001,
    STCapitalGainsTaxRate: 0.15,
    Execution: portfolio.ExecutionSettings{
        OrderFillAssumption: "NextBarOpen",
        SlippageMode:        "PercentOfPrice",
        PercentSlippageRate: 0.0005,
    },
}
```

Pair with `runner.WithPortfolio(portfolio.New(settings))` — no extra fill configuration required unless you want to override the mode.

---

## Related docs

- [Extending the framework](./extending-the-framework.md) — override `CostCalculator`, `CashFlowCalculator`, `PeriodicEventsModel`
- [Order fill pricing](./order-fill.md) — fill modes, `FillContext`, custom pricers
- [Writing a strategy](./writing-a-strategy.md) — order types and callbacks
- [Getting started](./getting-started.md) — minimal runnable backtest
