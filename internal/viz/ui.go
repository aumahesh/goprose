package viz

// uiHTML is the single-page web UI served at /.
const uiHTML = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>ProSe Simulator</title>
<style>
* { box-sizing: border-box; margin: 0; padding: 0; }
body { font-family: 'Segoe UI', system-ui, sans-serif; background: #1a1a2e; color: #e0e0e0; height: 100vh; display: flex; flex-direction: column; }

#toolbar {
  display: flex; align-items: center; gap: 12px; padding: 10px 16px;
  background: #16213e; border-bottom: 1px solid #0f3460; flex-shrink: 0;
}
#toolbar h1 { font-size: 1rem; font-weight: 600; color: #e94560; margin-right: 8px; }
button {
  padding: 6px 14px; border: none; border-radius: 4px; cursor: pointer;
  font-size: 0.875rem; font-weight: 500; transition: opacity 0.15s;
}
button:hover { opacity: 0.85; }
#btn-step  { background: #0f3460; color: #e0e0e0; }
#btn-auto  { background: #0f3460; color: #e0e0e0; }
#btn-auto.active { background: #e94560; }
#btn-reset { background: #333; color: #e0e0e0; }
#btn-compile { background: #1a472a; color: #e0e0e0; }
#step-counter { color: #a0a0c0; font-size: 0.875rem; margin-left: auto; }
label { font-size: 0.875rem; color: #a0a0c0; display: flex; align-items: center; gap: 6px; }
input[type=range] { width: 80px; }

#layout { display: flex; flex: 1; overflow: hidden; }

#graph-panel { flex: 1; position: relative; overflow: hidden; }
svg#graph { width: 100%; height: 100%; }

.node circle {
  fill: #16213e; stroke: #0f3460; stroke-width: 2;
  cursor: grab; transition: fill 0.3s;
}
.node.fired circle { fill: #e94560; stroke: #ff6b6b; }
.node text { fill: #e0e0e0; font-size: 11px; pointer-events: none; text-anchor: middle; }
.node .node-id { font-weight: bold; font-size: 13px; fill: #ffffff; }
.link { stroke: #0f3460; stroke-width: 1.5; }

#sidebar {
  width: 320px; background: #16213e; border-left: 1px solid #0f3460;
  display: flex; flex-direction: column; flex-shrink: 0;
}
#sidebar h2 { padding: 10px 14px; font-size: 0.875rem; color: #a0a0c0; border-bottom: 1px solid #0f3460; flex-shrink: 0; }
#log-entries { flex: 1; overflow-y: auto; padding: 8px; font-size: 0.78rem; }
.log-entry {
  padding: 6px 8px; margin-bottom: 4px; border-radius: 4px;
  background: #0f3460; border-left: 3px solid #e94560;
  line-height: 1.5;
}
.log-entry .step-num { color: #a0a0c0; }
.log-entry .node-label { color: #e94560; font-weight: 600; }
.log-entry .guard-text { color: #80b3ff; font-family: monospace; font-size: 0.75rem; }
.log-entry .change { color: #90ee90; }
.log-entry.no-fire { border-left-color: #555; background: #0a0a1a; }
</style>
</head>
<body>
<div id="toolbar">
  <h1>ProSe Sim</h1>
  <button id="btn-step">Step</button>
  <button id="btn-auto">Auto</button>
  <label>Speed: <input type="range" id="speed" min="1" max="20" value="5"> <span id="speed-val">5</span>/s</label>
  <button id="btn-reset">Reset</button>
  <button id="btn-compile">Generate Go ↓</button>
  <span id="step-counter">Step: 0</span>
</div>
<div id="layout">
  <div id="graph-panel"><svg id="graph"></svg></div>
  <div id="sidebar">
    <h2>Execution Log</h2>
    <div id="log-entries"></div>
  </div>
</div>

<script src="https://d3js.org/d3.v7.min.js"></script>
<script>
const state = { nodes: [], edges: [], varNames: [], step: 0, lastFiredID: null };
let simulation = null;
let autoInterval = null;
let nodeElements = null, linkElements = null;

async function api(path, method) {
  const res = await fetch(path, method ? { method } : {});
  return res.json();
}

function formatValue(v) {
  if (v === null || v === undefined) return '—';
  if (typeof v === 'string') return v === '' ? '""' : '"' + v + '"';
  return String(v);
}

function buildNodeLabel(node) {
  return state.varNames.map(k => k + ':' + formatValue(node.state[k])).join('  ');
}

function initGraph(data) {
  state.nodes = data.nodes;
  state.edges = data.edges;
  state.varNames = data.varNames;
  state.step = data.step;
  document.getElementById('step-counter').textContent = 'Step: ' + state.step;

  const svg = d3.select('#graph');
  svg.selectAll('*').remove();

  const width = svg.node().clientWidth || 800;
  const height = svg.node().clientHeight || 600;

  const g = svg.append('g');

  // Zoom/pan
  svg.call(d3.zoom().scaleExtent([0.3, 4]).on('zoom', e => g.attr('transform', e.transform)));

  // Arrow marker
  svg.append('defs').append('marker')
    .attr('id', 'arrow').attr('viewBox', '0 -5 10 10').attr('refX', 20).attr('refY', 0)
    .attr('markerWidth', 6).attr('markerHeight', 6).attr('orient', 'auto')
    .append('path').attr('d', 'M0,-5L10,0L0,5').attr('fill', '#0f3460');

  const simNodes = state.nodes.map(n => ({ id: n.id, ...n }));
  const simLinks = state.edges.map(e => ({ source: e.source, target: e.target }));

  simulation = d3.forceSimulation(simNodes)
    .force('link', d3.forceLink(simLinks).id(d => d.id).distance(120))
    .force('charge', d3.forceManyBody().strength(-300))
    .force('center', d3.forceCenter(width / 2, height / 2))
    .force('collision', d3.forceCollide(50));

  linkElements = g.append('g').selectAll('line')
    .data(simLinks).join('line').attr('class', 'link');

  nodeElements = g.append('g').selectAll('g')
    .data(simNodes).join('g')
    .attr('class', 'node')
    .call(d3.drag()
      .on('start', (e, d) => { if (!e.active) simulation.alphaTarget(0.3).restart(); d.fx = d.x; d.fy = d.y; })
      .on('drag', (e, d) => { d.fx = e.x; d.fy = e.y; })
      .on('end', (e, d) => { if (!e.active) simulation.alphaTarget(0); d.fx = null; d.fy = null; }));

  nodeElements.append('circle').attr('r', 36);
  nodeElements.append('text').attr('class', 'node-id').attr('dy', 0);

  // State variables label (below node ID)
  nodeElements.append('text').attr('class', 'node-vars').attr('dy', 16).style('font-size', '9px').style('fill', '#a0c4ff');

  updateLabels();

  simulation.on('tick', () => {
    linkElements
      .attr('x1', d => d.source.x).attr('y1', d => d.source.y)
      .attr('x2', d => d.target.x).attr('y2', d => d.target.y);
    nodeElements.attr('transform', d => 'translate(' + d.x + ',' + d.y + ')');
  });
}

function updateLabels() {
  if (!nodeElements) return;
  nodeElements.select('.node-id').text(d => d.id);
  nodeElements.select('.node-vars').text(d => {
    const nodeData = state.nodes.find(n => n.id === d.id);
    if (!nodeData) return '';
    return state.varNames.map(k => k + ':' + formatValue(nodeData.state[k])).join(' ');
  });
}

function updateGraph(data, lastFiredID) {
  state.nodes = data.nodes;
  state.step = data.step;
  state.lastFiredID = lastFiredID || null;
  document.getElementById('step-counter').textContent = 'Step: ' + state.step;

  if (!nodeElements) return;

  // Update state for label rendering and highlight
  nodeElements.classed('fired', d => d.id === state.lastFiredID);
  updateLabels();

  // Sync simulation node data
  if (simulation) {
    simulation.nodes().forEach(sn => {
      const updated = data.nodes.find(n => n.id === sn.id);
      if (updated) Object.assign(sn, updated);
    });
  }
}

function addLogEntry(result) {
  const container = document.getElementById('log-entries');
  const el = document.createElement('div');
  el.className = 'log-entry' + (result.fired ? '' : ' no-fire');

  if (!result.fired) {
    el.innerHTML = '<span class="step-num">step ' + result.step + '</span> — no guard fired on <span class="node-label">' + result.nodeId + '</span>';
  } else {
    const changes = (result.changes || []).map(c =>
      c.var + ': ' + formatValue(c.from) + ' → ' + formatValue(c.to)
    ).join(', ');
    el.innerHTML =
      '<span class="step-num">step ' + result.step + '</span>' +
      ' · <span class="node-label">' + result.nodeId + '</span>' +
      ' guard ' + result.guardIndex +
      ' <span class="guard-text">(' + escapeHTML(result.guardText) + ')</span>' +
      (changes ? '<br><span class="change">  ' + escapeHTML(changes) + '</span>' : '');
  }

  container.insertBefore(el, container.firstChild);
  // Keep last 200 entries
  while (container.children.length > 200) container.removeChild(container.lastChild);
}

function escapeHTML(s) {
  return s.replace(/&/g,'&amp;').replace(/</g,'&lt;').replace(/>/g,'&gt;');
}

async function doStep() {
  const data = await api('/api/step', 'POST');
  updateGraph(data, data.result && data.result.fired ? data.result.nodeId : null);
  addLogEntry(data.result);
}

async function doReset() {
  stopAuto();
  const data = await api('/api/reset', 'POST');
  document.getElementById('log-entries').innerHTML = '';
  updateGraph(data, null);
}

function startAuto() {
  const speed = parseInt(document.getElementById('speed').value, 10);
  const ms = Math.max(50, Math.round(1000 / speed));
  autoInterval = setInterval(doStep, ms);
  document.getElementById('btn-auto').classList.add('active');
  document.getElementById('btn-auto').textContent = 'Stop';
}

function stopAuto() {
  if (autoInterval) { clearInterval(autoInterval); autoInterval = null; }
  document.getElementById('btn-auto').classList.remove('active');
  document.getElementById('btn-auto').textContent = 'Auto';
}

document.getElementById('btn-step').addEventListener('click', doStep);
document.getElementById('btn-auto').addEventListener('click', () => {
  if (autoInterval) stopAuto(); else startAuto();
});
document.getElementById('btn-reset').addEventListener('click', doReset);
document.getElementById('speed').addEventListener('input', function() {
  document.getElementById('speed-val').textContent = this.value;
  if (autoInterval) { stopAuto(); startAuto(); }
});
document.getElementById('btn-compile').addEventListener('click', () => {
  window.location.href = '/api/compile';
});

// Initial load
api('/api/state').then(data => initGraph(data));
</script>
</body>
</html>`
