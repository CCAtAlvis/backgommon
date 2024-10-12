package runner

import (
	"encoding/json"
	"os"
	"time"

	"github.com/CCAtAlvis/backgommon/src/structs"
	"github.com/CCAtAlvis/backgommon/src/structs/order"
	"github.com/CCAtAlvis/backgommon/src/types"
	"gonum.org/v1/plot"
	"gonum.org/v1/plot/plotter"
	"gonum.org/v1/plot/vg"
)

type TickHandler func(r *Runner, row map[string]structs.Candle) []order.Order
type ExecuteOrders func(r *Runner, orders []order.Order)
type HandleLongEntry func(r *Runner, existingPosition *structs.Position, order order.Order)
type HandleLongExit func(r *Runner, existingPosition *structs.Position, order order.Order)
type HandleShortEntry func(r *Runner, existingPosition *structs.Position, order order.Order)
type HandleShortExit func(r *Runner, existingPosition *structs.Position, order order.Order)
type HandleTax func(r *Runner, existingPosition *structs.Position, order order.Order)

type RunnerFuntions struct {
	TickHandler      TickHandler
	ExecuteOrders    ExecuteOrders
	HandleLongEntry  HandleLongEntry
	HandleLongExit   HandleLongExit
	HandleShortEntry HandleShortEntry
	HandleShortExit  HandleShortExit
	HandleTax        HandleTax
}

type Runner struct {
	table       *types.TimeseriesTable[structs.Candle]
	CurrentTime time.Time
	Settings    structs.Settings
	AccountData structs.AccountData

	Functions RunnerFuntions
}

func GetDefaultRunnerFunctions(tickHandler TickHandler) RunnerFuntions {
	runnerFuncs := RunnerFuntions{
		TickHandler:      tickHandler,
		ExecuteOrders:    executeOrders,
		HandleLongEntry:  handleLongEntry,
		HandleLongExit:   handleLongExit,
		HandleShortEntry: handleShortEntry,
		HandleShortExit:  handleShortExit,
	}

	return runnerFuncs
}

func GetRunnerWithDefaults(tickHandler TickHandler) *Runner {
	runnerFuncs := GetDefaultRunnerFunctions(tickHandler)

	return &Runner{
		Functions: runnerFuncs,
	}
}

func (r *Runner) SetTable(table *types.TimeseriesTable[structs.Candle]) {
	r.table = table
}

func (r *Runner) Start(tickHandler TickHandler) {

	for _, val := range r.table.Rows() {
		if !r.Settings.StartDate.IsZero() && val.Timestamp.Before(r.Settings.StartDate) {
			continue
		}
		if !r.Settings.EndDate.IsZero() && val.Timestamp.After(r.Settings.EndDate) {
			continue
		}

		row, _ := val.Get()
		r.CurrentTime = val.Timestamp

		orders := r.Functions.TickHandler(r, row)
		r.Functions.ExecuteOrders(r, orders)
	}

	// save equity curve as json
	{
		jsonStr, _ := json.Marshal(r.AccountData.EquityCurve)
		_ = os.WriteFile("equity_curve.json", jsonStr, 0644)

		// save closed positions as json
		jsonStr, _ = json.Marshal(r.AccountData.ClosedPositions)
		_ = os.WriteFile("closed_positions.json", jsonStr, 0644)

		// visualize equity curve
		plt := plot.New()
		plt.Title.Text = "Equity Curve"
		plt.X.Label.Text = "Date"
		plt.Y.Label.Text = "Value"

		pts := make(plotter.XYs, len(r.AccountData.EquityCurve))
		for i, v := range r.AccountData.EquityCurve {
			// x, _ := strconv.ParseFloat(v.Date, 64)
			pts[i].X = float64(i)
			pts[i].Y = v.Value
		}
		line, _ := plotter.NewLine(pts)
		plt.Add(line)
		plt.Save(12*vg.Inch, 8*vg.Inch, "equity_curve.png")

		// visualize log equity curve
		pts = make(plotter.XYs, len(r.AccountData.EquityCurve))
		for i, v := range r.AccountData.EquityCurve {
			// x, _ := strconv.ParseFloat(v.Date, 64)
			pts[i].X = float64(i)
			pts[i].Y = v.LogValue
		}

		plt = plot.New()
		plt.Title.Text = "Log Equity Curve"
		plt.X.Label.Text = "Date"
		plt.Y.Label.Text = "Value"

		line, _ = plotter.NewLine(pts)
		plt.Add(line)
		plt.Save(12*vg.Inch, 8*vg.Inch, "log_equity_curve.png")
	}
}
