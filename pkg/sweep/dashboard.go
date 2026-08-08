package sweep

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

func writeDashboardHTML(path string, summary *Summary) error {
	payload := *summary
	trials := make([]SummaryTrial, len(summary.Trials))
	copy(trials, summary.Trials)
	for i := range trials {
		trials[i].ParamsMap = nil
	}
	payload.Trials = trials

	dataJSON, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal dashboard data: %w", err)
	}
	// Prevent </script> breakouts and keep % safe (no fmt verbs on payload).
	safe := strings.ReplaceAll(string(dataJSON), "<", `\u003c`)

	html := strings.Replace(dashboardTemplate, "__SWEEP_DATA__", safe, 1)
	return os.WriteFile(path, []byte(html), 0644)
}

// dashboardTemplate is a self-contained SPA. Data is injected at __SWEEP_DATA__.
const dashboardTemplate = `<!DOCTYPE html>
<html><head><meta charset="UTF-8"><meta name="viewport" content="width=device-width,initial-scale=1">
<title>Sweep Dashboard</title>
<script src="https://cdn.plot.ly/plotly-latest.min.js"></script>
<style>
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:'JetBrains Mono','Fira Code','SF Mono',monospace;background:#0a0e17;color:#c8d3e0;min-height:100vh}
::-webkit-scrollbar{width:6px;height:6px}::-webkit-scrollbar-track{background:#111827}::-webkit-scrollbar-thumb{background:#374151;border-radius:3px}
header{padding:14px 20px;border-bottom:1px solid #1e293b;display:flex;justify-content:space-between;align-items:center;gap:16px;flex-wrap:wrap;background:#0f1623;position:sticky;top:0;z-index:20}
header h1{font-size:1.05rem;font-weight:600;color:#e2e8f0}
header .meta{font-size:0.75rem;color:#64748b}
.tabs{display:flex;gap:6px;flex-wrap:wrap}
.tab{padding:6px 12px;border:1px solid #1e293b;background:#111827;color:#94a3b8;border-radius:4px;cursor:pointer;font-size:0.75rem;text-transform:uppercase;letter-spacing:0.4px}
.tab:hover{background:#1e293b;color:#e2e8f0}
.tab.active{background:#1e3a5f;border-color:#3b82f6;color:#93c5fd}
.container{padding:16px 20px;max-width:100%}
.toolbar{display:flex;gap:12px;flex-wrap:wrap;align-items:center;margin-bottom:14px}
input[type=search],select{background:#0a0e17;border:1px solid #1e293b;color:#e2e8f0;padding:8px 10px;border-radius:4px;font:inherit;font-size:0.8rem;min-width:180px}
input[type=search]:focus,select:focus{outline:1px solid #3b82f6}
.filters{display:flex;gap:8px;flex-wrap:wrap;margin-bottom:12px}
.chip{font-size:0.7rem;color:#94a3b8;display:flex;align-items:center;gap:6px;background:#111827;border:1px solid #1e293b;padding:4px 8px;border-radius:4px}
.chip select{min-width:90px;padding:4px 6px}
.btn{padding:6px 12px;border:1px solid #1e293b;background:#1e293b;color:#e2e8f0;border-radius:4px;cursor:pointer;font:inherit;font-size:0.75rem}
.btn:hover{border-color:#3b82f6}
.btn.primary{background:#1e3a5f;border-color:#3b82f6;color:#93c5fd}
.panel{background:#111827;border:1px solid #1e293b;border-radius:6px;margin-bottom:16px;overflow:hidden}
.panel-head{padding:12px 16px;font-size:0.8rem;font-weight:600;color:#94a3b8;text-transform:uppercase;letter-spacing:0.5px;border-bottom:1px solid #1e293b}
.panel-body{padding:16px}
.table-wrap{overflow:auto;max-height:70vh;border:1px solid #1e293b;border-radius:6px}
table{width:100%;border-collapse:collapse;font-size:0.75rem}
th,td{padding:8px 10px;border-bottom:1px solid #1e293b;text-align:left;white-space:nowrap}
th{position:sticky;top:0;background:#0f1623;color:#64748b;cursor:pointer;user-select:none;z-index:1}
th:hover{color:#93c5fd}
tr.data-row{cursor:pointer}
tr.data-row:hover{background:#1e293b}
tr.selected{background:#172554}
.g{color:#34d399}.r{color:#f87171}.muted{color:#64748b}
.metrics{display:grid;grid-template-columns:repeat(auto-fit,minmax(140px,1fr));gap:10px}
.m{padding:12px;background:#0a0e17;border:1px solid #1e293b;border-radius:4px}
.m .lbl{font-size:0.65rem;color:#64748b;text-transform:uppercase;letter-spacing:0.5px;margin-bottom:4px}
.m .val{font-size:1rem;font-weight:600;color:#e2e8f0}
.params{display:grid;grid-template-columns:repeat(auto-fit,minmax(220px,1fr));gap:8px;font-size:0.75rem}
.param{padding:8px 10px;background:#0a0e17;border:1px solid #1e293b;border-radius:4px}
.param.axis{border-color:#3b82f6}
.param .k{color:#64748b;margin-bottom:2px}
.param .v{color:#e2e8f0;word-break:break-all}
.axis-group{margin-bottom:18px}
.chart{height:260px;margin-top:8px}
.compare-bar{position:sticky;bottom:0;background:#0f1623;border-top:1px solid #1e293b;padding:10px 16px;display:flex;gap:10px;align-items:center;flex-wrap:wrap;z-index:15}
.hidden{display:none!important}
.badge{display:inline-block;padding:2px 6px;border-radius:3px;font-size:0.65rem;background:#1e293b;color:#94a3b8}
.badge.ok{background:#064e3b;color:#34d399}
.badge.err{background:#7f1d1d;color:#fca5a5}
td.diff{background:#172554}
.note{font-size:0.75rem;color:#64748b;margin:8px 0}
a{color:#93c5fd}
</style>
</head>
<body>
<header>
  <div>
    <h1>Sweep Dashboard</h1>
    <div class="meta" id="hdr-meta"></div>
  </div>
  <div class="tabs">
    <button class="tab active" data-view="grid">Grid</button>
    <button class="tab" data-view="detail">Detail</button>
    <button class="tab" data-view="axis">Axis Analysis</button>
    <button class="tab" data-view="compare">Compare</button>
  </div>
</header>
<div class="container">
  <section id="view-grid">
    <div class="toolbar">
      <input type="search" id="search" placeholder="Search params / metrics…">
      <span class="muted" id="grid-count"></span>
    </div>
    <div class="filters" id="axis-filters"></div>
    <div class="table-wrap"><table><thead id="grid-head"></thead><tbody id="grid-body"></tbody></table></div>
    <div class="compare-bar" id="compare-bar">
      <span id="sel-count">0 selected</span>
      <button class="btn primary" id="btn-goto-compare">Compare selected</button>
      <button class="btn" id="btn-clear-sel">Clear</button>
      <span class="note" id="sel-warn"></span>
    </div>
  </section>

  <section id="view-detail" class="hidden">
    <div class="toolbar">
      <button class="btn" id="btn-back-grid">← Grid</button>
      <a class="btn primary" id="btn-open-report" href="#" target="_blank">Open full report</a>
      <button class="btn" id="btn-add-compare">Add to compare</button>
    </div>
    <div class="panel"><div class="panel-head">Metrics</div><div class="panel-body"><div class="metrics" id="detail-metrics"></div></div></div>
    <div class="panel"><div class="panel-head">Varying parameters</div><div class="panel-body"><div class="params" id="detail-axes"></div></div></div>
    <div class="panel"><div class="panel-head" style="cursor:pointer" id="fixed-toggle">Fixed parameters <span class="badge">toggle</span></div><div class="panel-body hidden" id="detail-fixed"></div></div>
  </section>

  <section id="view-axis" class="hidden">
    <div class="toolbar">
      <label class="muted">Axis <select id="axis-select"></select></label>
      <label class="muted">Metric <select id="axis-metric"></select></label>
    </div>
    <div id="axis-groups"></div>
  </section>

  <section id="view-compare" class="hidden">
    <div class="toolbar">
      <button class="btn" id="btn-clear-compare">Clear selection</button>
      <span class="muted" id="compare-note"></span>
    </div>
    <div class="panel"><div class="panel-head">Parameter differences</div><div class="panel-body" id="compare-params"></div></div>
    <div class="panel"><div class="panel-head">Metrics</div><div class="panel-body" id="compare-metrics"></div></div>
    <div class="panel"><div class="panel-head">Reports</div><div class="panel-body" id="compare-links"></div></div>
  </section>
</div>
<script>
const DATA = __SWEEP_DATA__;
const MAX_COMPARE = 8;
const PRIMARY_METRICS = ['sharpe_ratio','cagr','max_drawdown','returns','total_trades'];

const state = {
  view: 'grid',
  sortKey: DATA.sort_by || 'sharpe_ratio',
  sortAsc: (DATA.sort_by === 'max_drawdown'),
  search: '',
  filters: {},
  selected: new Set(),
  detailIndex: null,
};

function fmtVal(v) {
  if (v == null) return '—';
  if (typeof v === 'number') {
    if (!isFinite(v)) return String(v);
    if (Math.abs(v) >= 1000) return v.toFixed(2);
    if (Math.abs(v) < 0.0001 && v !== 0) return v.toExponential(2);
    return Number(v.toPrecision(4)).toString();
  }
  if (typeof v === 'boolean') return v ? 'true' : 'false';
  return String(v);
}
function fmtMetric(key, v) {
  if (v == null || !isFinite(v)) return '—';
  if (key === 'max_drawdown' || key === 'returns' || key === 'cagr' || key === 'win_rate') return (v*100).toFixed(2)+'%';
  if (key === 'final_capital') return v.toFixed(2);
  if (key === 'total_trades' || key === 'winning_trades' || key === 'losing_trades') return String(Math.round(v));
  return v.toFixed(3);
}
function metricClass(key, v) {
  if (v == null || !isFinite(v)) return 'muted';
  if (key === 'max_drawdown') return v > 0.15 ? 'r' : (v > 0.05 ? '' : 'g');
  if (key === 'returns' || key === 'cagr' || key === 'sharpe_ratio' || key === 'sortino_ratio' || key === 'profit_factor') return v >= 0 ? 'g' : 'r';
  return '';
}
function canon(v) { return JSON.stringify(v); }
function axisNames() { return (DATA.axes||[]).map(a => a.name); }
function trialByIndex(idx) { return DATA.trials.find(t => t.index === idx); }

DATA.trials.forEach(t => {
  if (typeof t.params === 'string') {
    try { t.params = JSON.parse(t.params); } catch(e) { t.params = {}; }
  }
});

function matchesSearch(t, q) {
  if (!q) return true;
  q = q.toLowerCase();
  const blob = JSON.stringify(t.params||{}) + ' ' + JSON.stringify(t.metrics||{}) + ' ' + (t.error||'') + ' trial_'+String(t.index).padStart(3,'0');
  return blob.toLowerCase().includes(q);
}
function matchesFilters(t) {
  for (const [k, v] of Object.entries(state.filters)) {
    if (!v) continue;
    if (canon(t.params[k]) !== v) return false;
  }
  return true;
}
function filteredTrials() {
  return DATA.trials.filter(t => matchesSearch(t, state.search) && matchesFilters(t));
}

function setView(name) {
  state.view = name;
  document.querySelectorAll('.tab').forEach(el => el.classList.toggle('active', el.dataset.view === name));
  ['grid','detail','axis','compare'].forEach(v => {
    document.getElementById('view-'+v).classList.toggle('hidden', v !== name);
  });
  if (name === 'axis') renderAxis();
  if (name === 'compare') renderCompare();
  if (name === 'detail') renderDetail();
}

function renderHeader() {
  const n = DATA.trials.length;
  const axes = axisNames().join(', ') || '(none)';
  document.getElementById('hdr-meta').textContent =
    n + ' trials · sort=' + (DATA.sort_by||'') + ' · axes: ' + axes;
}

function renderFilters() {
  const box = document.getElementById('axis-filters');
  box.innerHTML = '';
  (DATA.axes||[]).forEach(ax => {
    const div = document.createElement('div');
    div.className = 'chip';
    const sel = document.createElement('select');
    const all = document.createElement('option');
    all.value = '';
    all.textContent = ax.name + ': all';
    sel.appendChild(all);
    ax.values.forEach(v => {
      const opt = document.createElement('option');
      opt.value = canon(v);
      opt.textContent = fmtVal(v);
      if (state.filters[ax.name] === canon(v)) opt.selected = true;
      sel.appendChild(opt);
    });
    sel.addEventListener('change', () => {
      if (sel.value) state.filters[ax.name] = sel.value; else delete state.filters[ax.name];
      renderGrid();
    });
    div.appendChild(sel);
    box.appendChild(div);
  });
}

function sortTrials(list) {
  const key = state.sortKey;
  const asc = state.sortAsc;
  const axisSet = new Set(axisNames());
  return list.slice().sort((a,b) => {
    let av, bv, aOK=true, bOK=true;
    if (key === 'index') { av=a.index; bv=b.index; }
    else if (key === 'duration_ms') { av=a.duration_ms; bv=b.duration_ms; }
    else if (axisSet.has(key)) { av=a.params[key]; bv=b.params[key]; }
    else {
      av = a.metrics ? a.metrics[key] : null;
      bv = b.metrics ? b.metrics[key] : null;
      aOK = av != null && isFinite(av);
      bOK = bv != null && isFinite(bv);
      if (aOK !== bOK) return aOK ? -1 : 1;
    }
    if (typeof av === 'string' || typeof bv === 'string') {
      const cmp = String(av).localeCompare(String(bv), undefined, {numeric:true});
      return asc ? cmp : -cmp;
    }
    if (av === bv) return a.index - b.index;
    if (av == null) return 1;
    if (bv == null) return -1;
    return asc ? (av - bv) : (bv - av);
  });
}

function renderGrid() {
  const axes = axisNames();
  const metrics = PRIMARY_METRICS.filter(m => (DATA.metric_keys||[]).includes(m) || DATA.trials.some(t => t.metrics && t.metrics[m] != null));
  const cols = ['sel','index', ...axes, ...metrics, 'duration_ms', 'status'];
  const head = document.getElementById('grid-head');
  head.innerHTML = '<tr>' + cols.map(c => {
    if (c === 'sel') return '<th></th>';
    const label = c === 'duration_ms' ? 'ms' : c;
    const mark = state.sortKey === c ? (state.sortAsc ? ' ↑' : ' ↓') : '';
    return '<th data-sort="'+c+'">'+label+mark+'</th>';
  }).join('') + '</tr>';

  head.querySelectorAll('th[data-sort]').forEach(th => {
    th.addEventListener('click', () => {
      const k = th.dataset.sort;
      if (state.sortKey === k) state.sortAsc = !state.sortAsc;
      else {
        state.sortKey = k;
        state.sortAsc = (k === 'max_drawdown' || k === 'index' || k === 'duration_ms');
      }
      renderGrid();
    });
  });

  let rows = sortTrials(filteredTrials());
  document.getElementById('grid-count').textContent = rows.length + ' / ' + DATA.trials.length + ' shown';

  const body = document.getElementById('grid-body');
  body.innerHTML = rows.map(t => {
    const checked = state.selected.has(t.index) ? 'checked' : '';
    const status = t.error
      ? '<span class="badge err">err</span>'
      : '<span class="badge ok">ok</span>';
    const cells = cols.map(c => {
      if (c === 'sel') return '<td><input type="checkbox" data-idx="'+t.index+'" '+checked+' onclick="event.stopPropagation()"></td>';
      if (c === 'index') return '<td>trial_'+String(t.index).padStart(3,'0')+'</td>';
      if (c === 'duration_ms') return '<td class="muted">'+t.duration_ms+'</td>';
      if (c === 'status') return '<td>'+status+'</td>';
      if (axes.includes(c)) return '<td>'+fmtVal(t.params[c])+'</td>';
      const v = t.metrics ? t.metrics[c] : null;
      return '<td class="'+metricClass(c,v)+'">'+fmtMetric(c,v)+'</td>';
    }).join('');
    return '<tr class="data-row'+(state.selected.has(t.index)?' selected':'')+'" data-idx="'+t.index+'">'+cells+'</tr>';
  }).join('');

  body.querySelectorAll('tr.data-row').forEach(tr => {
    tr.addEventListener('click', () => {
      state.detailIndex = Number(tr.dataset.idx);
      setView('detail');
    });
  });
  body.querySelectorAll('input[type=checkbox]').forEach(cb => {
    cb.addEventListener('change', () => {
      const idx = Number(cb.dataset.idx);
      if (cb.checked) {
        if (state.selected.size >= MAX_COMPARE) {
          cb.checked = false;
          document.getElementById('sel-warn').textContent = 'Max '+MAX_COMPARE+' trials for compare';
          return;
        }
        state.selected.add(idx);
      } else state.selected.delete(idx);
      updateSelBar();
      trHighlight();
    });
  });
  updateSelBar();
}

function trHighlight() {
  document.querySelectorAll('#grid-body tr').forEach(tr => {
    tr.classList.toggle('selected', state.selected.has(Number(tr.dataset.idx)));
  });
}
function updateSelBar() {
  document.getElementById('sel-count').textContent = state.selected.size + ' selected';
  if (state.selected.size <= MAX_COMPARE) document.getElementById('sel-warn').textContent = '';
}

function renderDetail() {
  const t = trialByIndex(state.detailIndex);
  if (!t) {
    document.getElementById('detail-metrics').innerHTML = '<div class="note">Select a trial from the grid.</div>';
    return;
  }
  const axes = new Set(axisNames());
  const metricsEl = document.getElementById('detail-metrics');
  const keys = (DATA.metric_keys && DATA.metric_keys.length) ? DATA.metric_keys : PRIMARY_METRICS;
  metricsEl.innerHTML = keys.map(k => {
    const v = t.metrics ? t.metrics[k] : null;
    return '<div class="m"><div class="lbl">'+k+'</div><div class="val '+metricClass(k,v)+'">'+fmtMetric(k,v)+'</div></div>';
  }).join('') +
  '<div class="m"><div class="lbl">duration</div><div class="val">'+t.duration_ms+' ms</div></div>' +
  (t.error ? '<div class="m"><div class="lbl">error</div><div class="val r">'+t.error+'</div></div>' : '');

  const axEl = document.getElementById('detail-axes');
  axEl.innerHTML = axisNames().map(k =>
    '<div class="param axis"><div class="k">'+k+'</div><div class="v">'+fmtVal(t.params[k])+'</div></div>'
  ).join('') || '<div class="note">No varying axes detected.</div>';

  const fixedEl = document.getElementById('detail-fixed');
  const fixedKeys = Object.keys(t.params||{}).filter(k => !axes.has(k)).sort();
  fixedEl.innerHTML = '<div class="params">'+fixedKeys.map(k =>
    '<div class="param"><div class="k">'+k+'</div><div class="v">'+fmtVal(t.params[k])+'</div></div>'
  ).join('')+'</div>';

  const a = document.getElementById('btn-open-report');
  a.href = t.report_html || '#';
  a.style.pointerEvents = t.report_html ? '' : 'none';
}

function bestDirection(metric) {
  return metric === 'max_drawdown' ? 'min' : 'max';
}

function renderAxis() {
  const axisSel = document.getElementById('axis-select');
  const metricSel = document.getElementById('axis-metric');
  if (!axisSel.options.length) {
    (DATA.axes||[]).forEach(ax => {
      const o = document.createElement('option');
      o.value = ax.name; o.textContent = ax.name;
      axisSel.appendChild(o);
    });
    const mkeys = (DATA.metric_keys && DATA.metric_keys.length) ? DATA.metric_keys : PRIMARY_METRICS;
    mkeys.forEach(k => {
      const o = document.createElement('option');
      o.value = k; o.textContent = k;
      if (k === DATA.sort_by) o.selected = true;
      metricSel.appendChild(o);
    });
  }
  const axis = axisSel.value || (DATA.axes[0] && DATA.axes[0].name);
  const metric = metricSel.value || DATA.sort_by || 'sharpe_ratio';
  if (!axis) {
    document.getElementById('axis-groups').innerHTML = '<div class="note">No varying axes to analyze.</div>';
    return;
  }
  const others = axisNames().filter(n => n !== axis);
  const groups = new Map();
  DATA.trials.forEach(t => {
    if (t.error || !t.metrics) return;
    const keyParts = others.map(k => k+'='+canon(t.params[k]));
    const gkey = keyParts.join('|') || '_all';
    if (!groups.has(gkey)) groups.set(gkey, {label: keyParts.join(', ') || '(all trials)', trials: []});
    groups.get(gkey).trials.push(t);
  });

  const dir = bestDirection(metric);
  const box = document.getElementById('axis-groups');
  box.innerHTML = '';
  let gi = 0;
  groups.forEach(g => {
    if (g.trials.length < 2) return;
    gi++;
    const byVal = new Map();
    g.trials.forEach(t => {
      const ck = canon(t.params[axis]);
      byVal.set(ck, t);
    });
    const entries = [...byVal.entries()].map(([ck,t]) => ({
      label: fmtVal(t.params[axis]),
      value: t.metrics[metric],
      trial: t
    })).sort((a,b) => String(a.label).localeCompare(String(b.label), undefined, {numeric:true}));

    let best = null;
    entries.forEach(e => {
      if (e.value == null || !isFinite(e.value)) return;
      if (!best) best = e;
      else if (dir === 'max' ? e.value > best.value : e.value < best.value) best = e;
    });

    const wrap = document.createElement('div');
    wrap.className = 'axis-group panel';
    wrap.innerHTML = '<div class="panel-head">Held constant: '+(g.label||'(none)')+'</div><div class="panel-body">'+
      '<table><thead><tr><th>'+axis+'</th><th>'+metric+'</th><th>trial</th></tr></thead><tbody>'+
      entries.map(e => {
        const hi = best && e.trial.index === best.trial.index;
        return '<tr'+(hi?' style="background:#172554"':'')+'><td>'+e.label+'</td><td class="'+metricClass(metric,e.value)+'">'+fmtMetric(metric,e.value)+(hi?' ★':'')+'</td><td>trial_'+String(e.trial.index).padStart(3,'0')+'</td></tr>';
      }).join('')+
      '</tbody></table><div class="chart" id="chart-'+gi+'"></div></div>';
    box.appendChild(wrap);

    if (window.Plotly) {
      Plotly.newPlot('chart-'+gi, [{
        type: 'bar',
        x: entries.map(e => e.label),
        y: entries.map(e => e.value),
        marker: {color: entries.map(e => best && e.trial.index===best.trial.index ? '#3b82f6' : '#334155')}
      }], {
        margin:{t:10,r:10,b:40,l:50},
        paper_bgcolor:'#111827', plot_bgcolor:'#0a0e17',
        font:{color:'#94a3b8', size:11},
        yaxis:{gridcolor:'#1e293b', title: metric},
        xaxis:{title: axis}
      }, {displayModeBar:false, responsive:true});
    }
  });
  if (!gi) box.innerHTML = '<div class="note">No groups with 2+ trials for this axis (need a full slice with other axes held fixed).</div>';
}

function renderCompare() {
  const idxs = [...state.selected].sort((a,b)=>a-b);
  const note = document.getElementById('compare-note');
  if (!idxs.length) {
    note.textContent = 'Select trials via checkboxes on the Grid tab.';
    document.getElementById('compare-params').innerHTML = '';
    document.getElementById('compare-metrics').innerHTML = '';
    document.getElementById('compare-links').innerHTML = '';
    return;
  }
  note.textContent = idxs.length + ' trials · max ' + MAX_COMPARE;
  const trials = idxs.map(trialByIndex).filter(Boolean);

  const allKeys = new Set();
  trials.forEach(t => Object.keys(t.params||{}).forEach(k => allKeys.add(k)));
  const diffKeys = [...allKeys].filter(k => {
    const vals = new Set(trials.map(t => canon(t.params[k])));
    return vals.size > 1;
  }).sort();

  let phtml = '<div class="table-wrap"><table><thead><tr><th>param</th>'+
    trials.map(t => '<th>trial_'+String(t.index).padStart(3,'0')+'</th>').join('')+'</tr></thead><tbody>';
  (diffKeys.length ? diffKeys : []).forEach(k => {
    phtml += '<tr><td>'+k+'</td>'+trials.map(t => '<td class="diff">'+fmtVal(t.params[k])+'</td>').join('')+'</tr>';
  });
  if (!diffKeys.length) phtml += '<tr><td colspan="'+(trials.length+1)+'" class="muted">No differing params among selection</td></tr>';
  phtml += '</tbody></table></div>';
  document.getElementById('compare-params').innerHTML = phtml;

  const mkeys = (DATA.metric_keys && DATA.metric_keys.length) ? DATA.metric_keys : PRIMARY_METRICS;
  let mhtml = '<div class="table-wrap"><table><thead><tr><th>metric</th>'+
    trials.map(t => '<th>trial_'+String(t.index).padStart(3,'0')+'</th>').join('')+'</tr></thead><tbody>';
  mkeys.forEach(k => {
    const vals = trials.map(t => t.metrics ? t.metrics[k] : null);
    let bestIdx = -1;
    const dir = bestDirection(k);
    vals.forEach((v,i) => {
      if (v == null || !isFinite(v)) return;
      if (bestIdx < 0) bestIdx = i;
      else if (dir === 'max' ? v > vals[bestIdx] : v < vals[bestIdx]) bestIdx = i;
    });
    mhtml += '<tr><td>'+k+'</td>'+vals.map((v,i) => {
      const hi = i === bestIdx;
      return '<td class="'+metricClass(k,v)+(hi?' diff':'')+'">'+fmtMetric(k,v)+(hi?' ★':'')+'</td>';
    }).join('')+'</tr>';
  });
  mhtml += '</tbody></table></div>';
  document.getElementById('compare-metrics').innerHTML = mhtml;

  document.getElementById('compare-links').innerHTML = '<div class="params">'+trials.map(t =>
    '<div class="param"><div class="k">trial_'+String(t.index).padStart(3,'0')+'</div><div class="v"><a href="'+(t.report_html||'#')+'" target="_blank">'+ (t.report_html||'no report') +'</a></div></div>'
  ).join('')+'</div>';
}

document.querySelectorAll('.tab').forEach(tab => tab.addEventListener('click', () => setView(tab.dataset.view)));
document.getElementById('search').addEventListener('input', e => { state.search = e.target.value; renderGrid(); });
document.getElementById('btn-clear-sel').addEventListener('click', () => { state.selected.clear(); updateSelBar(); renderGrid(); });
document.getElementById('btn-goto-compare').addEventListener('click', () => setView('compare'));
document.getElementById('btn-back-grid').addEventListener('click', () => setView('grid'));
document.getElementById('btn-add-compare').addEventListener('click', () => {
  if (state.detailIndex == null) return;
  if (state.selected.size >= MAX_COMPARE && !state.selected.has(state.detailIndex)) {
    alert('Max '+MAX_COMPARE+' trials for compare');
    return;
  }
  state.selected.add(state.detailIndex);
  updateSelBar();
});
document.getElementById('fixed-toggle').addEventListener('click', () => {
  document.getElementById('detail-fixed').classList.toggle('hidden');
});
document.getElementById('axis-select').addEventListener('change', renderAxis);
document.getElementById('axis-metric').addEventListener('change', renderAxis);
document.getElementById('btn-clear-compare').addEventListener('click', () => {
  state.selected.clear();
  updateSelBar();
  renderCompare();
  renderGrid();
});

renderHeader();
renderFilters();
renderGrid();
</script>
</body></html>
`
