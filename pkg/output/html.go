package output

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"github.com/CCAtAlvis/backgommon/pkg/types"
)

// ExportHTMLReport generates the full interactive HTML report at the given path.
// Kept for backward compatibility — prefer [ExportRun] for full output including
// results.json and config.json.
func ExportHTMLReport(results *types.Results, curve []types.AccountValue, path string) error {
	data := ReportData{
		Results:     results,
		EquityCurve: curve,
	}
	_, periods := CalculateDrawdownPeriods(curve)
	return exportHTMLReportFull(data, periods, path)
}

// equityCurveHoldings is the per-bar holdings snapshot serialised into the HTML
// template's JavaScript data array. The JS tooltip builder reads Cash and
// Holdings to render a hover-card showing portfolio composition at any point.
type equityCurveHoldings struct {
	Cash     float64                 `json:"cash"`
	Holdings []types.HoldingSnapshot `json:"h"`
}

// exportHTMLReportFull renders the self-contained interactive HTML report
// (Plotly equity/drawdown charts, trade tables, order tables, settings panel)
// and writes it to the given path. All data is JSON-encoded inline in <script>
// tags so the report has no external dependencies beyond the Plotly CDN.
func exportHTMLReportFull(data ReportData, periods []DrawdownPeriod, path string) error {
	if data.Results == nil {
		return fmt.Errorf("results is nil")
	}

	n := len(data.EquityCurve)
	dates := make([]string, n)
	values := make([]float64, n)
	logValues := make([]float64, n)
	drawdownPcts := make([]float64, n)
	eqHoldings := make([]equityCurveHoldings, n)
	peak := 0.0

	for i, pt := range data.EquityCurve {
		dates[i] = pt.Time.Format("2006-01-02")
		values[i] = pt.Value
		if pt.Value > 0 {
			logValues[i] = math.Log10(pt.Value)
		}
		if pt.Value > peak {
			peak = pt.Value
		}
		if peak > 0 {
			drawdownPcts[i] = ((pt.Value - peak) / peak) * 100
		}
		eqHoldings[i] = equityCurveHoldings{Cash: pt.Cash, Holdings: pt.Holdings}
	}

	trades := buildTradeRecords(data.ClosedPositions, true)
	holdings := buildHoldingRecords(data.OpenPositions)
	orders := buildOrderRecords(data.ClosedPositions, data.OpenPositions)

	datesJSON, _ := json.Marshal(dates)
	valuesJSON, _ := json.Marshal(values)
	logValuesJSON, _ := json.Marshal(logValues)
	drawdownJSON, _ := json.Marshal(drawdownPcts)
	eqHoldingsJSON, _ := json.Marshal(eqHoldings)
	periodsJSON, _ := json.Marshal(periods)
	tradesJSON, _ := json.Marshal(trades)
	holdingsJSON, _ := json.Marshal(holdings)
	ordersJSON, _ := json.Marshal(orders)
	settingsJSON := []byte("{}")
	if data.Settings != nil {
		settingsJSON, _ = json.Marshal(data.Settings)
	}
	ta := buildTradeAnalysis(data.ClosedPositions, data.OpenPositions)
	taJSON, _ := json.Marshal(ta)

	r := data.Results
	winRate := 0.0
	if r.TotalTrades > 0 {
		winRate = float64(r.WinningTrades) / float64(r.TotalTrades) * 100
	}

	costsHTML := ""
	if r.TotalCosts > 0 {
		costsHTML = fmt.Sprintf(`<div class="m"><div class="lbl">Total Costs</div><div class="val r">%s</div></div>
<div class="m"><div class="lbl">Brokerage</div><div class="val">%s</div></div>
<div class="m"><div class="lbl">Transaction Tax</div><div class="val">%s</div></div>
<div class="m"><div class="lbl">Capital Gains Tax</div><div class="val">%s</div></div>`,
			fmtNum(r.TotalCosts), fmtNum(r.TotalBrokerage),
			fmtNum(r.TotalTransactionTax), fmtNum(r.TotalCapitalGainsTax))
	}

	html := fmt.Sprintf(htmlTemplate,
		r.StartTime.Format("2006-01-02"), r.EndTime.Format("2006-01-02"),
		fmtNum(r.InitialCapital), fmtNum(r.FinalCapital),
		pnlCls(r.Returns), fmtRet(r.Returns),
		pnlCls(r.CAGR), r.CAGR*100,
		r.MaxDrawdown*100,
		r.SharpeRatio, r.SortinoRatio,
		r.AnnualizedStdDev*100, winRate, r.ProfitFactor,
		r.RiskFreeRate*100, r.RiskFreeRateSource,
		costsHTML,
		string(datesJSON), string(valuesJSON), string(logValuesJSON),
		string(drawdownJSON), string(eqHoldingsJSON), string(periodsJSON),
		string(tradesJSON), string(holdingsJSON), string(ordersJSON),
		string(settingsJSON), string(taJSON),
	)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(html), 0644)
}

// fmtNum formats a number using Indian number conventions — values ≥ 1 Crore
// (10,000,000) display as "X.XX Cr", values ≥ 1 Lakh (100,000) as "X.XX L",
// and smaller values as plain decimals.
//
// NOTE: This is a localization assumption baked into the report. See
// docs/notes/code-review-observations.md #7 for discussion.
func fmtNum(v float64) string {
	if v >= 10000000 {
		return fmt.Sprintf("%.2f Cr", v/10000000)
	}
	if v >= 100000 {
		return fmt.Sprintf("%.2f L", v/100000)
	}
	return fmt.Sprintf("%.2f", v)
}

// fmtRet formats a return value (decimal, e.g. 0.12) as a percentage string
// with a leading "+" for non-negative values ("+12.00%") and plain sign for
// negative values ("-3.50%").
func fmtRet(v float64) string {
	if v >= 0 {
		return fmt.Sprintf("+%.2f%%", v*100)
	}
	return fmt.Sprintf("%.2f%%", v*100)
}

// pnlCls returns the CSS class for profit/loss colouring: "g" (green) for
// non-negative values, "r" (red) for negative.
func pnlCls(v float64) string {
	if v >= 0 {
		return "g"
	}
	return "r"
}

const htmlTemplate = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Backtest Report</title>
<script src="https://cdn.plot.ly/plotly-latest.min.js"></script>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'JetBrains Mono','Fira Code','SF Mono',monospace;background:#0a0e17;color:#c8d3e0;min-height:100vh}
::-webkit-scrollbar{width:6px;height:6px}::-webkit-scrollbar-track{background:#111827}::-webkit-scrollbar-thumb{background:#374151;border-radius:3px}
.container{width:100%%;max-width:100%%;padding:16px 24px}
header{padding:16px 24px;border-bottom:1px solid #1e293b;display:flex;justify-content:space-between;align-items:center;background:#0f1623}
header h1{font-size:1.1rem;font-weight:600;color:#e2e8f0}
header .period{font-size:0.85rem;color:#64748b}
.panel{background:#111827;border:1px solid #1e293b;border-radius:6px;margin-bottom:16px;overflow:hidden}
.panel-head{display:flex;align-items:center;padding:12px 16px;cursor:pointer;font-size:0.85rem;font-weight:600;color:#94a3b8;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #1e293b;user-select:none;transition:background 0.15s}
.panel-head:hover{background:#1e293b}
.panel-head .arr{margin-right:10px;font-size:0.7rem;transition:transform 0.2s}
.panel.closed .panel-body{display:none}
.panel.closed .arr{transform:rotate(-90deg)}
.panel-body{padding:16px}
.metrics{display:grid;grid-template-columns:repeat(auto-fit,minmax(160px,1fr));gap:12px}
.m{padding:12px;background:#0a0e17;border:1px solid #1e293b;border-radius:4px}
.m .lbl{font-size:0.65rem;color:#64748b;text-transform:uppercase;letter-spacing:0.5px;margin-bottom:4px}
.m .val{font-size:1.1rem;font-weight:700;font-variant-numeric:tabular-nums}
.m .note{font-size:0.6rem;color:#475569;margin-top:2px}
.g{color:#22c55e}.r{color:#ef4444}
.chart-box{height:480px;margin:8px 0}
table{width:100%%;border-collapse:collapse;font-size:0.8rem}
th{padding:8px 10px;text-align:right;color:#64748b;font-weight:600;text-transform:uppercase;font-size:0.7rem;letter-spacing:0.3px;border-bottom:1px solid #1e293b;position:sticky;top:0;background:#111827;z-index:1}
td{padding:7px 10px;text-align:right;border-bottom:1px solid #1a2233}
th:first-child,td:first-child{text-align:left}
tr:hover td{background:#1e293b}
.filter-row{display:flex;flex-wrap:wrap;gap:12px;align-items:center;margin-bottom:12px;padding:10px;background:#0a0e17;border-radius:4px;border:1px solid #1e293b}
.filter-row label{font-size:0.75rem;color:#64748b}
.filter-row input,.filter-row select{background:#111827;border:1px solid #374151;color:#e2e8f0;padding:4px 8px;border-radius:3px;font-family:inherit;font-size:0.8rem}
.btn{background:#2563eb;color:#fff;border:none;border-radius:3px;padding:5px 14px;font-size:0.75rem;font-weight:600;cursor:pointer;font-family:inherit}
.btn:hover{background:#1d4ed8}
.btn-sm{padding:3px 8px;font-size:0.7rem}
.pag{display:flex;gap:4px;align-items:center;margin-left:auto}
.pag button{background:#1e293b;border:1px solid #374151;color:#94a3b8;padding:3px 8px;border-radius:3px;cursor:pointer;font-size:0.75rem;font-family:inherit}
.pag button:disabled{opacity:0.4;cursor:default}
.pag button.act{background:#2563eb;color:#fff;border-color:#2563eb}
.ctrl-row{display:flex;align-items:center;gap:16px;margin-bottom:10px;flex-wrap:wrap}
.ctrl-row label{font-size:0.75rem;color:#64748b}
.sortable{cursor:pointer;position:relative;padding-right:16px}
.sortable:hover{color:#e2e8f0}
.sortable::after{content:'⇅';position:absolute;right:2px;font-size:0.65rem;color:#475569}
.sortable.asc::after{content:'↑';color:#22c55e}.sortable.desc::after{content:'↓';color:#ef4444}
.grid3{display:grid;grid-template-columns:repeat(auto-fit,minmax(280px,1fr));gap:12px}
.card{background:#0a0e17;border:1px solid #1e293b;border-radius:4px;padding:14px}
.card h3{font-size:0.8rem;color:#64748b;margin-bottom:10px;text-transform:uppercase;letter-spacing:0.3px}
.card p{font-size:0.85rem;margin:5px 0;color:#c8d3e0}
.controls{padding:10px;background:#0a0e17;border:1px solid #1e293b;border-radius:4px;margin-bottom:12px}
.controls label{font-size:0.75rem;color:#94a3b8;cursor:pointer}
.expand-btn{background:none;border:1px solid #374151;color:#94a3b8;border-radius:3px;padding:2px 8px;cursor:pointer;font-size:0.7rem;font-family:inherit;transition:all 0.15s}
.expand-btn:hover{background:#1e293b;color:#e2e8f0}
.expand-btn.open{background:#1e3a5f;border-color:#2563eb;color:#93c5fd}
.order-detail{background:#0a0e17;border:1px solid #1e293b;border-radius:4px;padding:10px 12px;margin:4px 0}
.order-detail table{font-size:0.75rem}
.order-detail th{background:#0a0e17;font-size:0.65rem;padding:5px 8px}
.order-detail td{padding:5px 8px;border-bottom:1px solid #111827}
.trade-id{font-size:0.65rem;color:#475569;font-family:monospace}
@media(max-width:768px){.container{padding:8px}.metrics{grid-template-columns:1fr 1fr}.grid3{grid-template-columns:1fr}}
</style></head><body>

<header>
<h1>BACKTEST REPORT</h1>
<span class="period">%s → %s</span>
</header>

<div class="container">

<div class="panel" id="pMetrics">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>Performance Overview</div>
<div class="panel-body">
<div class="metrics">
<div class="m"><div class="lbl">Initial</div><div class="val">%s</div></div>
<div class="m"><div class="lbl">Final</div><div class="val">%s</div></div>
<div class="m"><div class="lbl">Return</div><div class="val %s">%s</div></div>
<div class="m"><div class="lbl">CAGR</div><div class="val %s">%.2f%%%%</div></div>
<div class="m"><div class="lbl">Max Drawdown</div><div class="val r">%.2f%%%%</div></div>
<div class="m"><div class="lbl">Sharpe</div><div class="val">%.3f</div></div>
<div class="m"><div class="lbl">Sortino</div><div class="val">%.3f</div></div>
<div class="m"><div class="lbl">Std Dev (Ann)</div><div class="val">%.2f%%%%</div></div>
<div class="m"><div class="lbl">Win Rate</div><div class="val">%.1f%%%%</div></div>
<div class="m"><div class="lbl">Profit Factor</div><div class="val">%.2f</div></div>
<div class="m"><div class="lbl">Risk-Free Rate</div><div class="val">%.1f%%%%</div><div class="note">%s</div></div>
%s</div>
</div></div>

<div class="panel" id="pDD">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>Drawdown Analysis</div>
<div class="panel-body">
<div class="grid3">
<div class="card"><h3>Maximum Drawdown</h3><p>Value: <span id="mxV" class="r"></span></p><p>Start: <span id="mxS"></span></p><p>End: <span id="mxE"></span></p><p>Duration: <span id="mxD"></span></p><p>Peak: <span id="mxP"></span></p><p>Trough: <span id="mxT"></span></p></div>
<div class="card"><h3>Longest Drawdown</h3><p>Value: <span id="lgV" class="r"></span></p><p>Start: <span id="lgS"></span></p><p>End: <span id="lgE"></span></p><p>Duration: <span id="lgD"></span></p><p>Peak: <span id="lgP"></span></p><p>Trough: <span id="lgT"></span></p></div>
<div class="card"><h3>Time in Drawdown</h3><p>Total: <span id="ddTot"></span></p><p>Percentage: <span id="ddPct"></span>%%%% of time</p></div>
</div>
</div></div>

<div class="panel" id="pTrade">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>Trade Analysis</div>
<div class="panel-body">
<div class="grid3">
<div class="card"><h3>Winning Trades</h3><p>Count: <span id="wC"></span></p><p>Total P&L: <span id="wPnL" class="g"></span></p><p>Avg Return: <span id="wAvg"></span>%%%%</p><p>Avg Holding: <span id="wDays"></span> days</p></div>
<div class="card"><h3>Losing Trades</h3><p>Count: <span id="lC"></span></p><p>Total P&L: <span id="lPnL" class="r"></span></p><p>Avg Return: <span id="lAvg"></span>%%%%</p><p>Avg Holding: <span id="lDays"></span> days</p></div>
<div class="card"><h3>Open Positions</h3><p>Profit: <span id="oP"></span></p><p>Loss: <span id="oL"></span></p><p>Unrealized: <span id="oU"></span></p></div>
</div>
</div></div>

<div class="panel" id="pChart">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>Equity Charts</div>
<div class="panel-body">
<div class="controls"><label><input type="checkbox" id="showShortDD"> Show drawdowns &lt; 30 days</label></div>
<div id="eqChart" class="chart-box"></div>
<div id="logChart" class="chart-box"></div>
</div></div>

<div class="panel" id="pHoldings">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>Current Holdings</div>
<div class="panel-body">
<table><thead><tr><th>#</th><th>Instrument</th><th>Qty</th><th>Entry</th><th>Current</th><th>Value</th><th>P&L</th><th>P&L %%%%</th></tr></thead><tbody id="hTbl"></tbody></table>
</div></div>

<div class="panel" id="pTrades">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>All Trades</div>
<div class="panel-body">
<div class="filter-row">
<label>Instrument <input type="text" id="tF1" placeholder="search"></label>
<label>Open <input type="date" id="tF2"> - <input type="date" id="tF3"></label>
<label>Close <input type="date" id="tF4"> - <input type="date" id="tF5"></label>
<label>PnL%%%% <input type="number" id="tF6" style="width:55px" placeholder="min"> - <input type="number" id="tF7" style="width:55px" placeholder="max"></label>
<button class="btn" id="expCsv">Export CSV</button>
</div>
<div class="ctrl-row">
<label>Show <select id="tPP"><option value="25">25</option><option value="50" selected>50</option><option value="100">100</option><option value="250">250</option></select></label>
<label><input type="checkbox" id="tAll"> All</label>
<div class="pag" id="tPag"></div>
</div>
<table><thead><tr><th></th><th>#</th><th>Instrument</th><th>Peak Qty</th><th>Open</th><th>Close</th><th>Avg Entry</th><th>Exit</th><th>PnL</th><th>PnL/Share</th><th>ROI %%%%</th><th>Days</th><th>Orders</th></tr></thead><tbody id="tTbl"></tbody></table>
</div></div>

<div class="panel" id="pOrders">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>All Orders</div>
<div class="panel-body">
<div class="filter-row">
<label>Instrument <input type="text" id="oF1" placeholder="search"></label>
<label>Type <select id="oF2"><option value="">All</option><option value="ENTRY">Entry</option><option value="EXIT">Exit</option></select></label>
<label>Trade ID <input type="text" id="oF5" placeholder="filter"></label>
<label>Date <input type="date" id="oF3"> - <input type="date" id="oF4"></label>
<button class="btn" id="expOrdCsv">Export CSV</button>
</div>
<div class="ctrl-row">
<label>Show <select id="oPP"><option value="50">50</option><option value="100" selected>100</option><option value="250">250</option></select></label>
<label><input type="checkbox" id="oAll"> All</label>
<div class="pag" id="oPag"></div>
</div>
<table><thead><tr><th>#</th><th>Trade ID</th><th>Instrument</th><th>Type</th><th>Side</th><th>Qty</th><th>Price</th><th>Date</th></tr></thead><tbody id="oTbl"></tbody></table>
</div></div>

<div class="panel" id="pSettings">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>Settings</div>
<div class="panel-body"><table><tbody id="sTbl"></tbody></table></div>
</div>

<div class="panel" id="pDDtbl">
<div class="panel-head" onclick="tog(this)"><span class="arr">▼</span>Drawdown Periods (>5%%%%)</div>
<div class="panel-body">
<table><thead><tr><th>#</th><th class="sortable" data-s="s">Start</th><th class="sortable" data-s="e">End</th><th class="sortable" data-s="d">Duration</th><th class="sortable" data-s="p">Drawdown</th><th>Peak</th><th>Trough</th></tr></thead><tbody id="ddTbl"></tbody></table>
</div></div>

</div>

<script>
const D=%s,V=%s,LV=%s,DD=%s,EH=%s,DP=%s,AT=%s,H=%s,AO=%s,S=%s,TA=%s;
const initVal=V[0]||1;

function tog(el){el.parentElement.classList.toggle('closed')}

// Build trade number lookup: trade_id -> "T-001" style label
const tradeNumMap={};
AT.forEach((t,i)=>{tradeNumMap[t.trade_id]='T-'+(i+1).toString().padStart(3,'0');});
function tNum(id){return tradeNumMap[id]||id;}

// --- Drawdown details ---
if(DP&&DP.length){
const mx=DP.reduce((a,b)=>b.drawdown_pct<a.drawdown_pct?b:a,DP[0]);
const lg=DP.reduce((a,b)=>b.duration_days>a.duration_days?b:a,DP[0]);
function setD(p,d){document.getElementById(p+'V').textContent=d.drawdown_pct.toFixed(2)+'%%';document.getElementById(p+'S').textContent=d.start_date;document.getElementById(p+'E').textContent=d.end_date;document.getElementById(p+'D').textContent=Math.round(d.duration_days)+' days';document.getElementById(p+'P').textContent=d.peak_value.toLocaleString();document.getElementById(p+'T').textContent=d.trough_value.toLocaleString();}
setD('mx',mx);setD('lg',lg);
const sig=DP.filter(p=>p.drawdown_pct<-5);
const totDD=sig.reduce((s,p)=>s+p.duration_days,0);
const totDays=D.length>1?(new Date(D[D.length-1])-new Date(D[0]))/86400000:1;
document.getElementById('ddTot').textContent=Math.round(totDD)+' days ('+(totDD/365).toFixed(1)+' yrs)';
document.getElementById('ddPct').textContent=(totDD/totDays*100).toFixed(1);
const ddTbl=document.getElementById('ddTbl');
function renderDDTbl(list){ddTbl.innerHTML='';list.forEach((p,i)=>{const r=document.createElement('tr');r.innerHTML='<td>'+(i+1)+'</td><td>'+p.start_date+'</td><td>'+p.end_date+'</td><td>'+Math.round(p.duration_days)+' d</td><td class="r">'+p.drawdown_pct.toFixed(2)+'%%</td><td>'+p.peak_value.toLocaleString()+'</td><td>'+p.trough_value.toLocaleString()+'</td>';ddTbl.appendChild(r);});}
renderDDTbl(sig);
document.querySelectorAll('#pDDtbl .sortable').forEach(th=>{th.onclick=()=>{const f=th.dataset.s;const dir=th.classList.contains('asc')?'desc':'asc';document.querySelectorAll('#pDDtbl .sortable').forEach(t=>{t.classList.remove('asc','desc')});th.classList.add(dir);const sorted=[...sig].sort((a,b)=>{let c=0;if(f==='s')c=new Date(a.start_date)-new Date(b.start_date);else if(f==='e')c=new Date(a.end_date)-new Date(b.end_date);else if(f==='d')c=a.duration_days-b.duration_days;else c=a.drawdown_pct-b.drawdown_pct;return dir==='asc'?c:-c;});renderDDTbl(sorted);};});}

// --- Trade analysis ---
document.getElementById('wC').textContent=TA.realized_profits?.count||0;
document.getElementById('wPnL').textContent=(TA.realized_profits?.total_pnl||0).toLocaleString();
document.getElementById('wAvg').textContent=(TA.realized_profits?.avg_pnl_percent||0).toFixed(2);
document.getElementById('wDays').textContent=Math.round(TA.realized_profits?.avg_holding_days||0);
document.getElementById('lC').textContent=TA.realized_losses?.count||0;
document.getElementById('lPnL').textContent=(TA.realized_losses?.total_pnl||0).toLocaleString();
document.getElementById('lAvg').textContent=(TA.realized_losses?.avg_pnl_percent||0).toFixed(2);
document.getElementById('lDays').textContent=Math.round(TA.realized_losses?.avg_holding_days||0);
document.getElementById('oP').textContent=TA.profit_positions||0;
document.getElementById('oL').textContent=TA.loss_positions||0;
document.getElementById('oU').textContent=((TA.unrealized_profit||0)+(TA.unrealized_loss||0)).toLocaleString();

// --- Settings ---
const sTbl=document.getElementById('sTbl');
Object.entries(S).forEach(([k,v])=>{const r=document.createElement('tr');r.innerHTML='<td>'+k+'</td><td>'+v+'</td>';sTbl.appendChild(r);});

// --- Holdings ---
const hTbl=document.getElementById('hTbl');
H.sort((a,b)=>b.value-a.value).forEach((h,i)=>{const c=h.pnl_percent>=0?'g':'r';const r=document.createElement('tr');r.innerHTML='<td>'+(i+1)+'</td><td>'+h.instrument+'</td><td>'+h.quantity+'</td><td>'+h.open_price.toFixed(2)+'</td><td>'+h.current_price.toFixed(2)+'</td><td>'+h.value.toFixed(0)+'</td><td class="'+c+'">'+h.pnl.toFixed(0)+'</td><td class="'+c+'">'+h.pnl_percent.toFixed(2)+'%%</td>';hTbl.appendChild(r);});

// --- Rich Tooltip Builder ---
function buildTooltip(i){
const date=D[i],val=V[i],gain=((val-initVal)/initVal*100).toFixed(2);
const eh=EH[i]||{};
let txt='<b>'+date+'</b><br>Value: '+val.toLocaleString()+'<br>Gain: '+gain+'%%<br>Cash: '+(eh.cash||0).toLocaleString()+'<br>Drawdown: '+DD[i].toFixed(2)+'%%';
const hs=eh.h;
if(hs&&hs.length>0){
txt+='<br><br><b>Holdings ('+hs.length+'):</b>';
const sorted=[...hs].sort((a,b)=>(b.p*b.q)-(a.p*a.q));
const show=sorted.slice(0,15);
show.forEach(h=>{const w=(h.p*h.q/val*100).toFixed(1);const pnl=((h.p-h.e)/h.e*100).toFixed(1);txt+='<br>'+h.i+': '+w+'%% ('+(pnl>=0?'+':'')+pnl+'%%)';});
if(sorted.length>15)txt+='<br>... +'+(sorted.length-15)+' more';
}
return txt;
}
const tooltipText=D.map((_,i)=>buildTooltip(i));

// --- Charts ---
const plotCfg={paper_bgcolor:'#111827',plot_bgcolor:'#111827',font:{color:'#94a3b8',family:'JetBrains Mono,monospace',size:11},margin:{t:30,b:40,l:60,r:60}};
const axCfg={gridcolor:'#1e293b',linecolor:'#1e293b',zerolinecolor:'#1e293b',showspikes:true,spikemode:'across',spikesnap:'cursor',spikedash:'solid',spikethickness:1,spikecolor:'#475569'};

function mkDDShapes(ps,short){const sh=[],an=[];ps.forEach((p,i)=>{if(!short&&p.duration_days<30)return;sh.push({type:'line',x0:p.start_date,y0:p.peak_value,x1:p.end_date,y1:p.peak_value,line:{color:'rgba(239,68,68,0.3)',width:1.5},layer:'below'});an.push({x:p.start_date,y:p.peak_value,xref:'x',yref:'y',text:'▾',showarrow:false,font:{color:'#ef4444',size:20},hovertext:'DD #'+(i+1)+' | '+p.drawdown_pct.toFixed(1)+'%% | '+Math.round(p.duration_days)+'d'});an.push({x:p.end_date,y:p.peak_value,xref:'x',yref:'y',text:'▴',showarrow:false,font:{color:'#22c55e',size:20},hovertext:'Recovery #'+(i+1)});});return{shapes:sh,annotations:an};}

const eqTrace={name:'Equity',x:D,y:V,type:'scatter',line:{color:'#3b82f6',width:1.5},hovertemplate:'%%{text}<extra></extra>',text:tooltipText};
const ddTr={name:'Drawdown',x:D,y:DD,type:'scatter',fill:'tozeroy',fillcolor:'rgba(239,68,68,0.08)',line:{color:'rgba(239,68,68,0.6)',width:1},yaxis:'y2',hoverinfo:'skip'};
const ly1={...plotCfg,xaxis:{...axCfg},yaxis:{...axCfg,title:'Value'},yaxis2:{...axCfg,title:'DD %%',overlaying:'y',side:'right',range:[Math.min(...DD)*1.2,0],fixedrange:true},showlegend:true,legend:{bgcolor:'transparent',font:{size:10}},hovermode:'x unified'};

Plotly.newPlot('eqChart',[eqTrace,ddTr],ly1,{responsive:true});
Plotly.newPlot('logChart',[{...eqTrace,name:'Log₁₀ Equity',y:LV,line:{color:'#8b5cf6',width:1.5}},ddTr],{...ly1,yaxis:{...axCfg,title:'Log₁₀'}},{responsive:true});

function updCharts(s){const{shapes,annotations}=mkDDShapes(DP,s);Plotly.relayout('eqChart',{shapes,annotations});const ls=shapes.map(x=>({...x,y0:Math.log10(Math.max(x.y0,1)),y1:Math.log10(Math.max(x.y1,1))}));const la=annotations.map(a=>({...a,y:Math.log10(Math.max(a.y,1))}));Plotly.relayout('logChart',{shapes:ls,annotations:la});}
document.getElementById('showShortDD').onchange=e=>updCharts(e.target.checked);
updCharts(false);

// --- Trades Table with Expandable Orders ---
let tFilt=AT,tPage=1,tPP=50,tShowAll=false;
const tF1=document.getElementById('tF1'),tF2=document.getElementById('tF2'),tF3=document.getElementById('tF3'),tF4=document.getElementById('tF4'),tF5=document.getElementById('tF5'),tF6=document.getElementById('tF6'),tF7=document.getElementById('tF7');
function tFilter(){const q=tF1.value.trim().toLowerCase();tFilt=AT.filter(t=>{if(q&&!t.instrument.toLowerCase().includes(q))return false;if(tF2.value&&t.open_date<tF2.value)return false;if(tF3.value&&t.open_date>tF3.value)return false;if(tF4.value&&t.close_date<tF4.value)return false;if(tF5.value&&t.close_date>tF5.value)return false;if(tF6.value!==''&&t.pnl_percent<+tF6.value)return false;if(tF7.value!==''&&t.pnl_percent>+tF7.value)return false;return true;});tPage=1;}

function renderOrderTimeline(orders){
let h='<div class="order-detail"><table><thead><tr><th>#</th><th>Date</th><th>Action</th><th>Qty</th><th>Price</th><th>Running Qty</th><th>Avg Buy</th><th>Realized PnL</th><th>Unrealized PnL</th></tr></thead><tbody>';
orders.forEach((o,i)=>{const ac=o.action==='BUY'?'g':'r';const pc=o.running_pnl>=0?'g':'r';const uc=(o.unrealized_pnl||0)>=0?'g':'r';h+='<tr><td>'+(i+1)+'</td><td>'+o.date+'</td><td class="'+ac+'">'+o.action+'</td><td>'+o.qty+'</td><td>'+o.price.toFixed(2)+'</td><td>'+o.running_qty+'</td><td>'+(o.avg_price>0?o.avg_price.toFixed(2):'-')+'</td><td class="'+pc+'">'+o.running_pnl.toFixed(2)+'</td><td class="'+uc+'">'+(o.unrealized_pnl||0).toFixed(2)+'</td></tr>';});
h+='</tbody></table></div>';return h;
}

function tRender(){const tb=document.getElementById('tTbl');tb.innerHTML='';let list=tFilt;if(!tShowAll){const s=(tPage-1)*tPP;list=tFilt.slice(s,s+tPP);}list.forEach((t,i)=>{const idx=tShowAll?i+1:(tPage-1)*tPP+i+1;const c=t.pnl>=0?'g':'r';const tid=tNum(t.trade_id);const uid='exp_'+idx;const tr=document.createElement('tr');tr.style.cursor='pointer';tr.innerHTML='<td><button class="expand-btn" data-uid="'+uid+'">▶</button></td><td class="trade-id">'+tid+'</td><td>'+t.instrument+'</td><td>'+t.quantity+'</td><td>'+t.open_date+'</td><td>'+t.close_date+'</td><td>'+t.avg_entry.toFixed(2)+'</td><td>'+t.exit_price.toFixed(2)+'</td><td class="'+c+'">'+t.pnl.toFixed(0)+'</td><td class="'+c+'">'+t.pnl_per_share.toFixed(2)+'</td><td class="'+c+'">'+t.pnl_percent.toFixed(2)+'%%</td><td>'+Math.round(t.holding_days)+'</td><td>'+(t.orders?t.orders.length:0)+'</td>';tb.appendChild(tr);
const detailRow=document.createElement('tr');detailRow.style.display='none';detailRow.id=uid;const detailTd=document.createElement('td');detailTd.colSpan=13;detailTd.innerHTML=renderOrderTimeline(t.orders||[]);detailRow.appendChild(detailTd);tb.appendChild(detailRow);
});
tb.querySelectorAll('.expand-btn').forEach(btn=>{btn.onclick=e=>{e.stopPropagation();const uid=btn.dataset.uid;const dr=document.getElementById(uid);if(dr.style.display==='none'){dr.style.display='';btn.textContent='▼';btn.classList.add('open');}else{dr.style.display='none';btn.textContent='▶';btn.classList.remove('open');}};});
tPagRender();}
function tPagRender(){const el=document.getElementById('tPag');el.innerHTML='';if(tShowAll)return;const tot=Math.ceil(tFilt.length/tPP);if(tot<=1)return;const mk=(t,p,d)=>{const b=document.createElement('button');b.textContent=t;b.disabled=d;if(p===tPage)b.classList.add('act');b.onclick=()=>{tPage=p;tRender();};return b;};el.appendChild(mk('‹',tPage-1,tPage===1));let s=Math.max(1,tPage-2),e=Math.min(tot,tPage+2);for(let i=s;i<=e;i++)el.appendChild(mk(i,i,false));el.appendChild(mk('›',tPage+1,tPage===tot));}
[tF1,tF2,tF3,tF4,tF5,tF6,tF7].forEach(el=>el.oninput=()=>{tFilter();tRender();});
document.getElementById('tPP').onchange=function(){tPP=+this.value;tPage=1;tRender();};
document.getElementById('tAll').onchange=function(){tShowAll=this.checked;tRender();};
document.getElementById('expCsv').onclick=()=>{let rows=[['#','TradeID','Instrument','PeakQty','Open','Close','AvgEntry','Exit','PnL','PnL/Share','ROI%%','Days','Orders']];tFilt.forEach((t,i)=>rows.push([i+1,tNum(t.trade_id),t.instrument,t.quantity,t.open_date,t.close_date,t.avg_entry.toFixed(2),t.exit_price.toFixed(2),t.pnl.toFixed(2),t.pnl_per_share.toFixed(2),t.pnl_percent.toFixed(2),Math.round(t.holding_days),t.orders?t.orders.length:0]));dl('trades.csv',rows);};
tFilter();tRender();

// --- Orders Table ---
let oFilt=AO,oPage=1,oPP=100,oShowAll=false;
const oF1=document.getElementById('oF1'),oF2=document.getElementById('oF2'),oF3=document.getElementById('oF3'),oF4=document.getElementById('oF4'),oF5=document.getElementById('oF5');
function oFilter(){const q=oF1.value.trim().toLowerCase();const tp=oF2.value;const tid=oF5.value.trim().toLowerCase();oFilt=AO.filter(o=>{if(q&&!o.instrument.toLowerCase().includes(q))return false;if(tp&&o.type!==tp)return false;if(tid&&!tNum(o.trade_id).toLowerCase().includes(tid))return false;if(oF3.value&&o.date<oF3.value)return false;if(oF4.value&&o.date>oF4.value)return false;return true;});oPage=1;}
function oRender(){const tb=document.getElementById('oTbl');tb.innerHTML='';let list=oFilt;if(!oShowAll){const s=(oPage-1)*oPP;list=oFilt.slice(s,s+oPP);}list.forEach((o,i)=>{const idx=oShowAll?i+1:(oPage-1)*oPP+i+1;const tc=o.type==='ENTRY'?'g':'r';const r=document.createElement('tr');r.innerHTML='<td>'+idx+'</td><td class="trade-id">'+tNum(o.trade_id)+'</td><td>'+o.instrument+'</td><td class="'+tc+'">'+o.type+'</td><td>'+o.side+'</td><td>'+o.quantity+'</td><td>'+o.price.toFixed(2)+'</td><td>'+o.date+'</td>';tb.appendChild(r);});oPagRender();}
function oPagRender(){const el=document.getElementById('oPag');el.innerHTML='';if(oShowAll)return;const tot=Math.ceil(oFilt.length/oPP);if(tot<=1)return;const mk=(t,p,d)=>{const b=document.createElement('button');b.textContent=t;b.disabled=d;if(p===oPage)b.classList.add('act');b.onclick=()=>{oPage=p;oRender();};return b;};el.appendChild(mk('‹',oPage-1,oPage===1));let s=Math.max(1,oPage-2),e=Math.min(tot,oPage+2);for(let i=s;i<=e;i++)el.appendChild(mk(i,i,false));el.appendChild(mk('›',oPage+1,oPage===tot));}
[oF1,oF2,oF3,oF4,oF5].forEach(el=>el.oninput=()=>{oFilter();oRender();});
document.getElementById('oPP').onchange=function(){oPP=+this.value;oPage=1;oRender();};
document.getElementById('oAll').onchange=function(){oShowAll=this.checked;oRender();};
document.getElementById('expOrdCsv').onclick=()=>{let rows=[['#','TradeID','Instrument','Type','Side','Qty','Price','Date']];oFilt.forEach((o,i)=>rows.push([i+1,tNum(o.trade_id),o.instrument,o.type,o.side,o.quantity,o.price.toFixed(2),o.date]));dl('orders.csv',rows);};
oFilter();oRender();

function dl(name,rows){const csv=rows.map(r=>r.map(c=>'"'+String(c).replace(/"/g,'""')+'"').join(',')).join('\n');const b=new Blob([csv],{type:'text/csv'});const a=document.createElement('a');a.href=URL.createObjectURL(b);a.download=name;a.click();}
</script></body></html>`
