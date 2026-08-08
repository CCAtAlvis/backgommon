// gemini

// package portfolio

// Settings contains portfolio-specific settings
type Settings struct {
	// InitialCapital is the starting cash balance of the portfolio.
	// e.g., 50000.0
	InitialCapital float64
	// AllowShorts determines if short selling is permitted.
	// e.g., true
	AllowShorts bool
	// MaxPositions is the maximum number of concurrent open positions the portfolio can hold.
	// e.g., 20
	MaxPositions int

	// DefaultLeverage is the leverage to apply to an order if not specified in the order itself.
	// This is still subject to MaxLeverage defined in risk.Settings.
	// e.g., 1.0 (no leverage), 2.0 (2x leverage)
	DefaultLeverage float64

	// SIPAmount is the amount for Systematic Investment Plan contributions.
	// If > 0, SIPFrequency must also be set to a non-zero duration.
	// e.g., 1000.0
	SIPAmount float64
	// SIPFrequency is the interval at which SIPAmount is added to the portfolio's cash.
	// e.g., 30 * 24 * time.Hour (for monthly contributions)
	SIPFrequency time.Duration

	// CostOfLeverageRate is the interest rate charged on borrowed capital due to leverage.
	// This rate is applied per CostOfLeveragePeriod.
	// e.g., 0.0002 (for 0.02% per period)
	CostOfLeverageRate float64
	// CostOfLeveragePeriod is the time interval for which CostOfLeverageRate is applied.
	// e.g., 24 * time.Hour (for daily cost)
	CostOfLeveragePeriod time.Duration

	// SafeInterestRate is the annual interest rate earned on idle (uninvested) cash.
	// The portfolio logic will need to accrue this periodically.
	// e.g., 0.03 (for 3% annual rate)
	SafeInterestRate float64
	// SafeInterestAccrualPeriod is how often the safe interest is calculated and added to cash.
	// The SafeInterestRate will be proportionally adjusted to this period.
	// e.g., 24 * time.Hour (for daily accrual)
	SafeInterestAccrualPeriod time.Duration

	// IsTaxEnabled flags whether trading taxes are applied.
	// e.g., false
	IsTaxEnabled bool
	// BuySideTaxPercent is the tax rate applied to the total value of a buy order.
	// Expressed as a decimal. e.g., 0.0020 (for 0.20%)
	BuySideTaxPercent float64
	// SellSideTaxPercent is the tax rate applied to the total value of a sell order.
	// Expressed as a decimal. e.g., 0.0020 (for 0.20%)
	SellSideTaxPercent float64
	// STCGTaxPercent is the Short Term Capital Gains tax rate applied to realized profits from trades.
	// Expressed as a decimal. e.g., 0.15 (for 15%)
	STCGTaxPercent float64

	// IsPocketEnabled flags whether the profit "pocketing" mechanism is active.
	// This allows setting aside a portion of profits from successful trades.
	// e.g., false
	IsPocketEnabled bool
	// MinProfitForPocket is the minimum realized profit a single position must achieve to trigger pocketing.
	// e.g., 100000.0
	MinProfitForPocket float64
	// PocketPercent is the percentage of the realized profit (above MinProfitForPocket, or total profit) to be "pocketed".
	// Pocketed funds might be considered unavailable for further trading or subject to different rules.
	// Expressed as a decimal. e.g., 0.1 (for 10%)
	PocketPercent float64

	// IsManagementFeeEnabled flags whether management fees are deducted from the portfolio.
	// e.g., false
	IsManagementFeeEnabled bool
	// ManagementFeePercent is the (typically annual) management fee rate based on total portfolio value.
	// Expressed as a decimal. e.g., 0.01 (for 1%)
	ManagementFeePercent float64
	// ManagementFeeFrequency is how often the management fee is calculated and deducted.
	// The ManagementFeePercent will be proportionally adjusted to this period.
	// e.g., 365 * 24 * time.Hour (for annual deduction)
	ManagementFeeFrequency time.Duration
}




// package risk

// Settings contains risk management settings
type Settings struct {
	// MaxDrawdown is the maximum allowed percentage drop in total portfolio value
	// from its peak before triggering a potential stop or alert.
	// Expressed as a decimal. e.g., 0.30 (for 30% drawdown)
	MaxDrawdown float64
	// MaxLeverage is the absolute maximum leverage allowed for any single order.
	// e.g., 5.0 (for 5x leverage)
	MaxLeverage float64
	// MaxPositionSize is the maximum size a single position can represent as a percentage of the total portfolio value.
	// Expressed as a decimal. e.g., 0.20 (for 20% of portfolio value)
	MaxPositionSize float64
	// MinPositionSize is the minimum absolute value required for opening a new position.
	// e.g., 100.0 (currency units)
	MinPositionSize float64

	// UseStopLoss flags whether stop-loss orders should be automatically considered or generated.
	// e.g., true
	UseStopLoss bool
	// UseTakeProfit flags whether take-profit orders should be automatically considered or generated.
	// e.g., true
	UseTakeProfit bool
	// UseTrailingStop flags whether trailing stop mechanisms should be active for positions.
	// e.g., true
	UseTrailingStop bool
	// DefaultStopLoss is the default stop-loss percentage from the entry price if not otherwise specified.
	// Expressed as a decimal. e.g., 0.15 (for 15% loss from entry)
	DefaultStopLoss float64
	// DefaultTakeProfit is the default take-profit percentage from the entry price if not otherwise specified.
	// Expressed as a decimal. e.g., 0.20 (for 20% gain from entry)
	DefaultTakeProfit float64
	// DefaultTrailingStop is the default percentage for a trailing stop-loss.
	// The stop price will trail the market price by this percentage.
	// Expressed as a decimal. e.g., 0.05 (for 5% trail)
	DefaultTrailingStop float64

	// RiskPerTradePercent is the maximum percentage of total portfolio capital to risk on a single trade.
	// This is used for position sizing, typically in conjunction with a stop-loss distance.
	// For example, if portfolio is $100k and RiskPerTradePercent is 0.01 (1%), then max risk for a trade is $1k.
	// If stop-loss is 10% away from entry, position size would be $1k / 10% = $10k.
	// Expressed as a decimal. e.g., 0.01 (for 1% of portfolio capital)
	RiskPerTradePercent float64

	// MinGainPercent is the minimum target gain percentage for a position.
	// This can be used by strategy logic or as a secondary check for take-profit levels.
	// Expressed as a decimal. e.g., 0.19 (for 19% gain)
	MinGainPercent float64

	// IsAccountLockEnabled flags whether the account locking mechanism is active.
	// If true, trading might be temporarily suspended if the account value drops by AccountLockPercent.
	// e.g., false
	IsAccountLockEnabled bool
	// AccountLockPercent is the percentage drop in total account value from a recent peak
	// that would trigger a temporary account lock (if IsAccountLockEnabled is true).
	// Expressed as a decimal. e.g., 0.02 (for a 2% drop from peak)
	AccountLockPercent float64
}


**I. Portfolio Operational & Financial Settings**
(Likely to reside in `portfolio.Settings` or a dedicated financial settings struct)

*   **Core Portfolio Setup:**
    *   `InitialCapital` (float64): The starting cash balance of the portfolio.
    *   `AllowShorts` (bool): Determines if short selling is permitted.
    *   `DefaultLeverage` (float64): Leverage to apply to an order if not specified in the order itself. This would be capped by `MaxLeverage` from risk settings.
*   **Periodic Contributions/Withdrawals:**
    *   `SIPAmount` (float64): Amount for Systematic Investment Plan contributions.
    *   `SIPFrequency` (time.Duration): The interval at which `SIPAmount` is added to the portfolio's cash (e.g., monthly, weekly).
*   **Interest & Costs on Capital:**
    *   `SafeInterestRateAnnual` (float64): Annual interest rate earned on uninvested cash.
    *   `SafeInterestAccrualFrequency` (time.Duration): How often the safe interest is calculated and added to cash (e.g., daily, monthly).
    *   `CostOfLeverageRateAnnual` (float64): Annual interest rate charged on borrowed capital due to leverage.
    *   `CostOfLeverageAccrualFrequency` (time.Duration): How often the cost of leverage is calculated and deducted.
*   **Taxation:**
    *   `IsTaxEnabled` (bool): Master switch to enable/disable tax calculations.
    *   `BuySideTaxPercent` (float64): Tax rate applied to the total value of a buy order.
    *   `SellSideTaxPercent` (float64): Tax rate applied to the total value of a sell order.
    *   `ShortTermCapitalGainsTaxPercent` (float64): Tax rate for short-term capital gains.
    *   `LongTermCapitalGainsTaxPercent` (float64): Tax rate for long-term capital gains.
    *   `ShortTermHoldingPeriod` (time.Duration): Duration to define a holding as "short-term" for tax purposes.
*   **Profit Management:**
    *   `IsPocketEnabled` (bool): Flags whether the profit "pocketing" mechanism is active.
    *   `MinProfitForPocket` (float64): Minimum realized profit a single position must achieve for pocketing.
    *   `PocketPercent` (float64): Percentage of realized profit to be "pocketed".
*   **Management Fees:**
    *   `IsManagementFeeEnabled` (bool): Master switch for management fees.
    *   `ManagementFeePercentAnnual` (float64): Annual management fee rate based on total portfolio value.
    *   `ManagementFeeFrequency` (time.Duration): How often the management fee is calculated and deducted.

**II. Risk Management Settings**
(Likely to reside in `risk.Settings`)

*   **Portfolio-Level Risk:**
    *   `MaxPortfolioDrawdownPercent` (float64): Maximum allowed percentage drop in total portfolio value from its peak.
(we need to define multiple options that can happen after max drawdown is reached, what should the framework do in that case.. expand more on that. also another parameter we can introduce is PortfolioDrawdown, this will be available to the user as to what is the current drawdown of the portfolio)
*   **Position-Level Risk:**
    *   `MaxLeverage` (float64): Absolute maximum leverage allowed for any single order.
    *   `MaxPositionSizeAsPercentOfPortfolio` (float64): Maximum size a single position can represent as a percentage of the total portfolio value at the time of entry. *(Clarifies existing `MaxPositionSize`)*
    *   `RiskPerTradeAsPercentOfPortfolio` (float64): Maximum percentage of total portfolio capital to risk on a single trade, used for position sizing. (e.g., 1% of portfolio capital = $1000 risk if portfolio is $100k). *(From your `RiskOfPosition` and `MaxRiskPerPosition`)*
(we need something more here i think... Initially, the `RiskPerTradePositionSize` setting, combined with the stop-loss, was designed to determine the position size. For example, with a 2% risk and a 10% stop-loss, we could calculate the number of shares to buy. The goal was to automate this calculation based on the user's risk settings.
The current setting is good, but we need to expand it and explore alternative settings to achieve the same result. We need to figure out how to approach this and how to implement it effectively.)

*   **Order-Level Risk Controls (Defaults & Enables):**
    *   `UseStopLoss` (bool): Enable automatic stop-loss generation/consideration.
    *   `DefaultStopLossPercent` (float64): Default stop-loss percentage from the entry price.
    *   `UseTakeProfit` (bool): Enable automatic take-profit generation/consideration.
    *   `DefaultTakeProfitPercent` (float64): Default take-profit percentage from the entry price.
    *   `MinGainPercentForTakeProfit` (float64): Explicit minimum target gain percentage for a position, can inform take-profit levels.
    *   `UseTrailingStop` (bool): Enable trailing stop mechanisms.
    *   `DefaultTrailingStopPercent` (float64): Default percentage for a trailing stop-loss.
*   **Account Safety:**
    *   `IsAccountLockEnabled` (bool): Enable temporary suspension of trading if account value drops significantly.
    *   `AccountLockTriggerPercent` (float64): Percentage drop in total account value (e.g., from a recent peak or start) to trigger the lock.
    *   `AccountLockDuration` (time.Duration): How long the account remains locked if triggered.
(Initially, `isAccountLockedEnabled` was used to set aside a percentage of an account as free cash. For example, setting it to 2% would earmark 2% of your portfolio. Trading would only occur on the remaining 98%. When the account was "unlocked," the 2% would be added back, creating a 2% cash safety net.

First, is this functionality necessary or beneficial for the framework? Second, if it is necessary, we can enable the two settings I mentioned earlier: `isAccountLockedEnabled` and `accountLockPercentage`. These are the two settings we could implement.)

`RiskFreeRateForMetrics` (float64):
Use: The annualized risk-free rate used specifically for calculating performance metrics like Sharpe Ratio, Treynor Ratio, Jensen's Alpha. Can be different from the rate idle cash earns.
Example: 0.02 (for 2%)

**III. Execution Realism & Brokerage Settings**
(Could be a new `ExecutionSettings` struct, or parts in `PortfolioSettings`/`RunnerSettings`)

*   **Brokerage / Commission Model:**
    *   `FixedBrokeragePerTrade` (float64): Fixed fee per trade.
    *   `PercentBrokerageRate` (float64): Brokerage as a percentage of trade value.
*   **Slippage Model:**
    *   `SlippageType` (enum/string): e.g., "FixedPoints", "PercentOfPrice", "VolumeBased", "None".
    *   `FixedSlippagePoints` (float64): Fixed price points of slippage per trade/share.
    *   `PercentSlippageRate` (float64): Slippage as a percentage of the order price.
*   **Order Fill & Execution Logic:**
    *   `OrderFillAssumption` (enum/string): e.g., "NextBarOpen", "CurrentBarClose" (default), "MidPrice", "WorstCaseWithinBar".
    *   `AllowPartialFills` (bool): Whether orders can be partially filled (e.g., based on available volume at a price level – more advanced) (optional-will think on this later).
    *   `MarketImpactModel` (enum/string): For simulating price impact of large orders. (e.g., "None", "Linear", "SquareRoot" - very advanced) (optional-will think on this later).

