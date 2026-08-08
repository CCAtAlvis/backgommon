package risk

import (
	"time"

	"github.com/CCAtAlvis/backgommon/pkg/interfaces"
)

// CheckDrawdown implements [interfaces.RiskManager]. It evaluates the
// portfolio's current equity against its historical peak using the configured
// [DrawdownPolicy] and returns the resulting action (block new entries,
// liquidate, etc.). As a side-effect it updates the Manager's internal peak
// equity, lock-until time, and entry-blocking flag.
func (m *Manager) CheckDrawdown(portfolioManager interfaces.PortfolioManager, now time.Time) interfaces.DrawdownAction {
	result := m.drawdownPolicyOrDefault().Evaluate(DrawdownPolicyContext{
		Equity:     portfolioManager.Value(),
		PeakEquity: m.peakEquity,
		Now:        now,
		LockUntil:  m.lockUntil,
		Settings:   m.settings,
		Logger:     m.logger,
	})

	m.peakEquity = result.PeakEquity
	m.lockUntil = result.LockUntil
	m.blockNewEntries = result.BlockNewEntries

	return toInterfaceAction(result)
}

// SuggestedQuantity delegates to the configured [PositionSizer] to compute
// how many units/shares to buy given the current equity and expected entry
// price. Returns 0 when sizing cannot be determined (e.g. missing stop-loss
// configuration).
func (m *Manager) SuggestedQuantity(equity, entryPrice float64) int {
	return m.positionSizerOrDefault().SuggestQuantity(PositionSizeContext{
		Equity:     equity,
		EntryPrice: entryPrice,
		Settings:   m.settings,
	})
}

// Deprecated: DrawdownResult is retained only for backward compatibility with
// existing tests. New code should use [DrawdownPolicyResult] instead, which
// carries the same fields plus the lock-until time.
type DrawdownResult struct {
	CurrentRate     float64
	PeakEquity      float64
	BlockNewEntries bool
	LiquidateAll    bool
	Breached        bool
}

// checkDrawdown exposes policy evaluation for tests.
func (m *Manager) checkDrawdown(portfolioManager interfaces.PortfolioManager, now time.Time) DrawdownResult {
	result := m.drawdownPolicyOrDefault().Evaluate(DrawdownPolicyContext{
		Equity:     portfolioManager.Value(),
		PeakEquity: m.peakEquity,
		Now:        now,
		LockUntil:  m.lockUntil,
		Settings:   m.settings,
		Logger:     m.logger,
	})
	m.peakEquity = result.PeakEquity
	m.lockUntil = result.LockUntil
	m.blockNewEntries = result.BlockNewEntries
	return DrawdownResult{
		CurrentRate:     result.DrawdownRate,
		PeakEquity:      result.PeakEquity,
		BlockNewEntries: result.BlockNewEntries,
		LiquidateAll:    result.LiquidateAll,
		Breached:        result.Breached,
	}
}
