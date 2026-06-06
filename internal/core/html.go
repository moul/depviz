package core

import (
	"context"
	"encoding/json"
	"html/template"
	"io"
)

func (s *Store) RenderHTML(ctx context.Context, boardID string, w io.Writer) error {
	payload, err := s.BuildExport(ctx, boardID)
	if err != nil {
		return err
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return htmlTemplate.Execute(w, struct {
		Title string
		Data  template.JS
	}{
		Title: "DepViz: " + payload.Snapshot.Board.Name,
		Data:  template.JS(data),
	})
}

var htmlTemplate = template.Must(template.New("html").Parse(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}}</title>
  <style>
    :root { color-scheme: light; --bg:#f7f8fb; --panel:#ffffff; --ink:#20242c; --muted:#677084; --line:#d9deea; --accent:#0f766e; --warn:#b45309; --closed:#6b7280; }
    * { box-sizing: border-box; }
    body { margin:0; font:14px/1.45 ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; background:var(--bg); color:var(--ink); }
    header { padding:16px 20px 12px; border-bottom:1px solid var(--line); background:var(--panel); display:flex; gap:16px; align-items:center; justify-content:space-between; flex-wrap:wrap; }
    h1 { font-size:20px; margin:0; font-weight:650; }
    .meta { color:var(--muted); display:flex; gap:12px; flex-wrap:wrap; }
    .toolbar { display:flex; gap:8px; align-items:center; flex-wrap:wrap; }
    button, select { border:1px solid var(--line); background:var(--panel); color:var(--ink); border-radius:6px; padding:7px 10px; font:inherit; }
    button.active { border-color:var(--accent); color:var(--accent); font-weight:650; }
    main { display:grid; grid-template-columns:260px 1fr; min-height:calc(100vh - 74px); }
    aside { border-right:1px solid var(--line); background:var(--panel); padding:16px; }
    section { padding:16px; min-width:0; }
    .summary { display:grid; grid-template-columns:repeat(5,minmax(110px,1fr)); gap:10px; margin-bottom:14px; }
    .stat { background:var(--panel); border:1px solid var(--line); border-radius:8px; padding:10px; }
    .stat strong { display:block; font-size:22px; }
    label { display:block; font-size:12px; font-weight:650; color:var(--muted); margin:14px 0 6px; text-transform:uppercase; }
    .search { width:100%; border:1px solid var(--line); border-radius:6px; padding:8px 9px; font:inherit; }
    .check { display:flex; gap:8px; align-items:center; margin:8px 0; color:var(--ink); }
    .canvas { background:var(--panel); border:1px solid var(--line); border-radius:8px; min-height:560px; overflow:auto; position:relative; }
    .graph { position:relative; min-width:900px; min-height:620px; }
    .card { position:absolute; width:210px; min-height:74px; padding:10px; border:1px solid var(--line); background:#fff; border-radius:8px; box-shadow:0 2px 10px rgba(20,24,35,.06); transition:transform .18s ease, opacity .18s ease; }
    .card.note { border-color:#0f766e; background:#eefdfa; }
    .card.closed { color:var(--closed); background:#f3f4f6; }
    .card .id { font-size:12px; color:var(--muted); margin-bottom:4px; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; }
    .card .title { font-weight:650; }
    .card .state { margin-top:6px; font-size:12px; color:var(--muted); }
    svg.edges { position:absolute; inset:0; pointer-events:none; overflow:visible; }
    table { width:100%; border-collapse:collapse; background:var(--panel); border:1px solid var(--line); border-radius:8px; overflow:hidden; }
    th, td { padding:9px 10px; border-bottom:1px solid var(--line); text-align:left; vertical-align:top; }
    th { color:var(--muted); font-size:12px; text-transform:uppercase; background:#fbfcfe; }
    tr:last-child td { border-bottom:0; }
    a { color:#0b5cad; text-decoration:none; }
    .brief { background:var(--panel); border:1px solid var(--line); border-radius:8px; padding:14px; margin-bottom:14px; }
    .brief h2 { margin:0 0 8px; font-size:16px; }
    .brief ul { margin:8px 0 0; padding-left:18px; }
    @media (max-width: 850px) { main { grid-template-columns:1fr; } aside { border-right:0; border-bottom:1px solid var(--line); } .summary { grid-template-columns:repeat(2,1fr); } }
  </style>
</head>
<body>
<header>
  <div>
    <h1 id="title"></h1>
    <div class="meta" id="meta"></div>
  </div>
  <div class="toolbar">
    <button data-view="graph" class="active">Graph</button>
    <button data-view="table">Table</button>
  </div>
</header>
<main>
  <aside>
    <label for="q">Filter</label>
    <input class="search" id="q" placeholder="id, title, label, state">
    <label>Sources</label>
    <div class="check"><input type="checkbox" id="showExternal" checked><span>External cards</span></div>
    <div class="check"><input type="checkbox" id="showLocal" checked><span>Local notes</span></div>
    <label>State</label>
    <div class="check"><input type="checkbox" id="showClosed"><span>Closed / merged</span></div>
  </aside>
  <section>
    <div class="summary" id="summary"></div>
    <div class="brief" id="brief"></div>
    <div class="canvas" id="graphView"><div class="graph" id="graph"></div></div>
    <div id="tableView" style="display:none"></div>
  </section>
</main>
<script id="depviz-data" type="application/json">{{.Data}}</script>
<script>
const data = JSON.parse(document.getElementById('depviz-data').textContent);
const state = { view: 'graph', q: '', showExternal: true, showLocal: true, showClosed: false };
const nodes = data.snapshot.nodes || [];
const edges = data.snapshot.edges || [];
document.getElementById('title').textContent = data.snapshot.board.name || 'DepViz';
document.getElementById('meta').textContent = String(nodes.length) + ' nodes - ' + String(edges.length) + ' edges';
for (const btn of document.querySelectorAll('button[data-view]')) btn.onclick = () => { state.view = btn.dataset.view; document.querySelectorAll('button[data-view]').forEach(b => b.classList.toggle('active', b === btn)); render(); };
document.getElementById('q').oninput = e => { state.q = e.target.value.toLowerCase(); render(); };
for (const id of ['showExternal','showLocal','showClosed']) document.getElementById(id).onchange = e => { state[id] = e.target.checked; render(); };
function isLocal(n) { return n.kind === 'note' || n.id.startsWith('note:'); }
function isClosed(n) { return ['closed','done','merged','cancelled','canceled','resolved'].includes((n.state||'').toLowerCase()); }
function labels(n) { try { return (JSON.parse(n.DataJSON || n.data_json || '{}').labels || []).join(' '); } catch { return ''; } }
function visibleNodes() {
  return nodes.filter(n => {
    if (!state.showClosed && isClosed(n)) return false;
    if (!state.showLocal && isLocal(n)) return false;
    if (!state.showExternal && !isLocal(n)) return false;
    const hay = [n.id, n.title, n.state, n.kind, labels(n)].join(' ').toLowerCase();
    return !state.q || hay.includes(state.q);
  });
}
function render() {
  renderSummary();
  renderBrief();
  document.getElementById('graphView').style.display = state.view === 'graph' ? '' : 'none';
  document.getElementById('tableView').style.display = state.view === 'table' ? '' : 'none';
  if (state.view === 'graph') renderGraph(); else renderTable();
}
function renderSummary() {
  const b = data.brief.counts || data.brief.Counts || {};
  document.getElementById('summary').innerHTML = [['Nodes',b.nodes||b.Nodes||0],['Ready',b.ready||b.Ready||0],['Blocked',b.blocked||b.Blocked||0],['Local',b.local_only||b.LocalOnly||0],['Stale',b.stale||b.Stale||0]].map(x => '<div class="stat"><strong>' + x[1] + '</strong>' + x[0] + '</div>').join('');
}
function renderBrief() {
  const next = data.brief.next_move || data.brief.NextMove;
  document.getElementById('brief').innerHTML = '<h2>Next move</h2>' + (next ? '<div><strong>' + esc(next.id||next.ID) + '</strong> ' + esc(next.title||next.Title) + '<br><span class="meta">' + esc(next.reason||next.Reason||'') + '</span></div>' : '<div class="meta">none</div>');
}
function renderGraph() {
  const list = visibleNodes();
  const visible = new Set(list.map(n => n.id));
  const graph = document.getElementById('graph');
  graph.innerHTML = '<svg class="edges" width="1800" height="1200"><defs><marker id="arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0,0 L0,6 L7,3 z" fill="#9aa4b2"></path></marker></defs></svg>';
  const cols = 4, xGap = 250, yGap = 130;
  const pos = {};
  list.forEach((n, i) => { const col = i % cols, row = Math.floor(i / cols); pos[n.id] = { x: 30 + col * xGap, y: 30 + row * yGap }; });
  const svg = graph.querySelector('svg');
  for (const e of edges) {
    if (!visible.has(e.from_id || e.FromID) || !visible.has(e.to_id || e.ToID)) continue;
    const from = pos[e.from_id || e.FromID], to = pos[e.to_id || e.ToID];
    if (!from || !to) continue;
    const line = document.createElementNS('http://www.w3.org/2000/svg','line');
    line.setAttribute('x1', from.x + 210); line.setAttribute('y1', from.y + 38);
    line.setAttribute('x2', to.x); line.setAttribute('y2', to.y + 38);
    line.setAttribute('stroke', '#9aa4b2'); line.setAttribute('stroke-width', '1.4'); line.setAttribute('marker-end', 'url(#arrow)');
    svg.appendChild(line);
  }
  for (const n of list) {
    const p = pos[n.id];
    const div = document.createElement('div');
    div.className = 'card ' + (isLocal(n) ? 'note ' : '') + (isClosed(n) ? 'closed' : '');
    div.style.transform = 'translate(' + p.x + 'px, ' + p.y + 'px)';
    div.innerHTML = '<div class="id">' + link(n) + '</div><div class="title">' + esc(n.title) + '</div><div class="state">' + esc(n.kind) + ' - ' + esc(n.state || '') + '</div>';
    graph.appendChild(div);
  }
}
function renderTable() {
  const rows = visibleNodes().map(n => '<tr><td>' + link(n) + '</td><td>' + esc(n.title) + '</td><td>' + esc(n.kind) + '</td><td>' + esc(n.state||'') + '</td><td>' + esc(labels(n)) + '</td></tr>').join('');
  document.getElementById('tableView').innerHTML = '<table><thead><tr><th>ID</th><th>Title</th><th>Kind</th><th>State</th><th>Labels</th></tr></thead><tbody>' + rows + '</tbody></table>';
}
function link(n) { const id = esc(n.id); return n.url ? '<a href="' + esc(n.url) + '">' + id + '</a>' : id; }
function esc(s) { return String(s || '').replace(/[&<>"']/g, c => ({'&':'&amp;','<':'&lt;','>':'&gt;','"':'&quot;',"'":'&#39;'}[c])); }
render();
</script>
</body>
</html>`))
