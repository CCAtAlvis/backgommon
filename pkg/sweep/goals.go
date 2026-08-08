package sweep

import (
	"fmt"
	"math"

	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// ConstraintOp compares a metric to a bound.
type ConstraintOp string

const (
	OpLT  ConstraintOp = "lt"
	OpLTE ConstraintOp = "lte"
	OpGT  ConstraintOp = "gt"
	OpGTE ConstraintOp = "gte"
)

// Constraint is a hard rule on a results metric (e.g. max_drawdown lte 0.15).
type Constraint struct {
	Metric string
	Op     ConstraintOp
	Value  float64
}

// Target is a soft goal used for distance scoring (e.g. sharpe_ratio → 1.5).
// If OneSided is true, error is 0 when actual is on the "good" side of Value:
// for metrics where higher is better (default), actual >= Value is perfect;
// for max_drawdown, actual <= Value is perfect.
type Target struct {
	Metric   string
	Value    float64
	Weight   float64 // default 1
	OneSided bool
}

// Goals combines hard constraints and soft targets for reverse / target search.
type Goals struct {
	Constraints []Constraint
	Targets     []Target
	// RankFeasibleBy, when non-empty, ranks feasible trials by this metric
	// (same names as Rank). When empty and Targets is non-empty, ranks by
	// ascending goal distance. When both empty, falls back to SearchConfig.SortBy.
	RankFeasibleBy string
}

// GoalScore is the evaluation of one trial against Goals.
type GoalScore struct {
	Feasible bool
	Distance float64 // lower is better; only meaningful when Targets is non-empty
}

// EvaluateGoals checks constraints and computes soft target distance.
func EvaluateGoals(results *types.Results, g *Goals) GoalScore {
	if g == nil {
		return GoalScore{Feasible: true, Distance: 0}
	}
	if results == nil {
		return GoalScore{Feasible: false, Distance: math.Inf(1)}
	}
	for _, c := range g.Constraints {
		v, ok := resultsMetric(results, c.Metric)
		if !ok || !constraintHolds(v, c.Op, c.Value) {
			return GoalScore{Feasible: false, Distance: math.Inf(1)}
		}
	}
	dist := 0.0
	for _, t := range g.Targets {
		v, ok := resultsMetric(results, t.Metric)
		if !ok {
			return GoalScore{Feasible: false, Distance: math.Inf(1)}
		}
		w := t.Weight
		if w == 0 {
			w = 1
		}
		dist += w * targetError(t, v)
	}
	return GoalScore{Feasible: true, Distance: dist}
}

func constraintHolds(actual float64, op ConstraintOp, bound float64) bool {
	switch op {
	case OpLT:
		return actual < bound
	case OpLTE:
		return actual <= bound
	case OpGT:
		return actual > bound
	case OpGTE:
		return actual >= bound
	default:
		return false
	}
}

func targetError(t Target, actual float64) float64 {
	higherBetter := t.Metric != "max_drawdown" && t.Metric != "drawdown"
	if t.OneSided {
		if higherBetter {
			if actual >= t.Value {
				return 0
			}
			return (t.Value - actual) / math.Max(math.Abs(t.Value), 1e-9)
		}
		if actual <= t.Value {
			return 0
		}
		return (actual - t.Value) / math.Max(math.Abs(t.Value), 1e-9)
	}
	return math.Abs(actual-t.Value) / math.Max(math.Abs(t.Value), 1e-9)
}

func resultsMetric(r *types.Results, metric string) (float64, bool) {
	if r == nil {
		return 0, false
	}
	switch metric {
	case "sharpe_ratio", "sharpe":
		return r.SharpeRatio, true
	case "sortino_ratio", "sortino":
		return r.SortinoRatio, true
	case "cagr":
		return r.CAGR, true
	case "returns", "return":
		return r.Returns, true
	case "max_drawdown", "drawdown":
		return r.MaxDrawdown, true
	case "profit_factor":
		return r.ProfitFactor, true
	case "final_capital":
		return r.FinalCapital, true
	case "total_trades", "trades":
		return float64(r.TotalTrades), true
	default:
		if r.Metrics != nil {
			if v, ok := r.Metrics[metric]; ok {
				return v, true
			}
		}
		return 0, false
	}
}

// MaxDrawdownLTE is a helper constraint: max_drawdown <= value (fraction).
func MaxDrawdownLTE(value float64) Constraint {
	return Constraint{Metric: "max_drawdown", Op: OpLTE, Value: value}
}

// MinSharpe is a helper constraint: sharpe_ratio >= value.
func MinSharpe(value float64) Constraint {
	return Constraint{Metric: "sharpe_ratio", Op: OpGTE, Value: value}
}

// MinCAGR is a helper constraint: cagr >= value (decimal).
func MinCAGR(value float64) Constraint {
	return Constraint{Metric: "cagr", Op: OpGTE, Value: value}
}

// ValidateGoals returns an error if Goals is malformed.
func ValidateGoals(g *Goals) error {
	if g == nil {
		return nil
	}
	for i, c := range g.Constraints {
		switch c.Op {
		case OpLT, OpLTE, OpGT, OpGTE:
		default:
			return fmt.Errorf("goals: constraint[%d] invalid op %q", i, c.Op)
		}
		if c.Metric == "" {
			return fmt.Errorf("goals: constraint[%d] missing metric", i)
		}
	}
	for i, t := range g.Targets {
		if t.Metric == "" {
			return fmt.Errorf("goals: target[%d] missing metric", i)
		}
	}
	return nil
}
