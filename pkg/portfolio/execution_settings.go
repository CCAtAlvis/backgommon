package portfolio

// ExecutionSettings defines parameters related to order execution realism,
// such as brokerage fees and slippage.
type ExecutionSettings struct {
	// --- Slippage Model ---

	// SlippageMode defines the model used to simulate slippage on order execution.
	// Supported values: "None", "FixedPoints", "PercentOfPrice".
	// If "None" or empty, slippage is not simulated.
	// e.g., "PercentOfPrice"
	SlippageMode string

	// FixedSlippageAmount is the fixed amount of adverse price movement (slippage) applied per share/contract.
	// Used when SlippageModel is "FixedPoints".
	// For buy orders, execution price is OrderPrice + FixedSlippageAmount.
	// For sell orders, execution price is OrderPrice - FixedSlippageAmount.
	// e.g., 0.02 (for 2 cents slippage per share)
	FixedSlippageAmount float64

	// PercentSlippageRate is the slippage applied as a percentage of the order price.
	// Used when SlippageModel is "PercentOfPrice". Expressed as a decimal.
	// For buy orders, execution price is OrderPrice * (1 + PercentSlippageRate).
	// For sell orders, execution price is OrderPrice * (1 - PercentSlippageRate).
	// e.g., 0.0005 (for 0.05% slippage)
	PercentSlippageRate float64

	// --- Order Fill & Execution Logic ---

	// OrderFillAssumption defines at what price point within a candle/bar orders are assumed to be filled.
	// Supported values: "CurrentBarClose" (default), "NextBarOpen", "MidPrice", "WorstCaseWithinBar".
	// - "CurrentBarClose": Assumes fill at the closing price of the current bar/tick where the order is generated.
	// - "NextBarOpen": Assumes fill at the opening price of the *next* bar/tick.
	// - "MidPrice": Assumes fill at the (High+Low)/2 of the current bar (if applicable, otherwise Close).
	// - "WorstCaseWithinBar": For buy, fill at High; for sell, fill at Low of the current bar.
	// e.g., "CurrentBarClose"
	OrderFillAssumption string

	// EnablePartialFills determines if orders can be partially filled.
	// Note: Full implementation of partial fills based on simulated volume is complex.
	// Initially, this might just mean an order can be partially filled if it exceeds max position size
	// or available capital, rather than being rejected entirely.
	// Default: false (orders are either fully filled or rejected).
	// e.g., false
	EnablePartialFills bool // Optional, marked for later thought by user

	// MarketImpactModel defines how large orders might affect the execution price.
	// Supported values: "None" (default), "Linear", "SquareRoot".
	// Note: Implementation of market impact models is advanced.
	// Default: "None".
	// e.g., "None"
	MarketImpactModel string // Optional, marked for later thought by user
}

// NewDefaultExecutionSettings creates ExecutionSettings with sensible defaults.
func NewDefaultExecutionSettings() *ExecutionSettings {
	return &ExecutionSettings{
		SlippageMode:        "None",
		FixedSlippageAmount: 0.0,
		PercentSlippageRate: 0.0,
		OrderFillAssumption: "CurrentBarClose", // Defaulting to CurrentBarClose
		EnablePartialFills:  false,
		MarketImpactModel:   "None",
	}
}
