package risk

// Option configures a risk Manager at construction time.
type Option func(*Manager)

// WithPositionSizer replaces the default position sizing model.
func WithPositionSizer(s PositionSizer) Option {
	return func(m *Manager) {
		m.positionSizer = s
	}
}

// WithDrawdownPolicy replaces the default drawdown policy.
func WithDrawdownPolicy(p DrawdownPolicy) Option {
	return func(m *Manager) {
		m.drawdownPolicy = p
	}
}

// WithLogger sets a logger for risk events (drawdown warnings, exit triggers).
// Pass nil to silence all risk output. Default is nil (silent).
func WithLogger(l Logger) Option {
	return func(m *Manager) {
		m.logger = l
	}
}

// positionSizerOrDefault returns the injected PositionSizer, falling back to
// StandardPositionSizer when none was provided via WithPositionSizer.
func (m *Manager) positionSizerOrDefault() PositionSizer {
	if m.positionSizer != nil {
		return m.positionSizer
	}
	return StandardPositionSizer{}
}

// drawdownPolicyOrDefault returns the injected DrawdownPolicy, falling back to
// StandardDrawdownPolicy when none was provided via WithDrawdownPolicy.
func (m *Manager) drawdownPolicyOrDefault() DrawdownPolicy {
	if m.drawdownPolicy != nil {
		return m.drawdownPolicy
	}
	return StandardDrawdownPolicy{}
}
