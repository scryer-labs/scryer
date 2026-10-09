(() => {
  'use strict';
  const app = window.ScryerReport;
  const model = window.ScryerGraphModel;
  const host = document.getElementById('graph-host');
  const svgNamespace = 'http://www.w3.org/2000/svg';
  const graphState = {selected: null, collapsed: new Set(), focus: false, snapshot: app.state.snapshot, scale: 1, x: 0, y: 0};
  let canvas, world, layout, visible;
  const elements = new Map();
  const movedPositions = new Map();
  let refreshEdges = () => {};
  const svgNode = (name, attributes = {}, text) => {
    const element = document.createElementNS(svgNamespace, name);
    for (const [key, value] of Object.entries(attributes)) element.setAttribute(key, String(value));
    if (text !== undefined) element.textContent = text;
    return element;
  };
  const control = (label, action, title = label) => {
    const button = app.node('button', label); button.type = 'button'; button.title = title; button.setAttribute('aria-label', title);
    button.addEventListener('click', action); return button;
  };
  const transform = () => world?.setAttribute('transform', `translate(${graphState.x} ${graphState.y}) scale(${graphState.scale})`);
  function fit() {
    if (!canvas || !layout) return;
    const width = canvas.clientWidth; const height = canvas.clientHeight;
    graphState.scale = Math.min(1, (width - 40) / layout.width, (height - 40) / layout.height);
    graphState.x = (width - layout.width * graphState.scale) / 2;
    graphState.y = (height - layout.height * graphState.scale) / 2;
    transform();
  }
  function zoom(factor, point = null) {
    if (!canvas) return;
    const center = point || {x: canvas.clientWidth / 2, y: canvas.clientHeight / 2};
    const next = Math.max(.015, Math.min(2.5, graphState.scale * factor));
    const ratio = next / graphState.scale;
    graphState.x = center.x - (center.x - graphState.x) * ratio;
    graphState.y = center.y - (center.y - graphState.y) * ratio;
    graphState.scale = next; transform();
  }
  function updateSelection() {
    const current = app.snapshot().nodes.find(node => node.id === graphState.selected);
    elements.forEach((group, id) => group.classList.toggle('selected', id === graphState.selected));
    world?.querySelectorAll('.graph-edge').forEach(edge => edge.classList.toggle('highlighted', edge.dataset.caller === graphState.selected || edge.dataset.callee === graphState.selected));
    const inspector = document.getElementById('graph-inspector');
    if (inspector) {
      inspector.replaceChildren();
      if (current) {
        const information = app.node('div');
        information.append(app.node('strong', app.label(current.signature)), app.node('span', `${current.role} · ${current.impact} · ${app.statusFor(current).replaceAll('_', ' ')}`, 'source-path'));
        inspector.append(information, control('Details →', () => app.selectSymbol(current.id, app.state.snapshot)));
      } else inspector.append(app.node('span', 'Select a node to highlight its incoming/outgoing calls and inspect its evidence.', 'muted small'));
    }
    const selectedVisible = visible?.nodes.some(node => node.id === graphState.selected);
    const collapse = document.getElementById('collapse-calls');
    if (collapse) { collapse.disabled = !selectedVisible || !app.snapshot().edges.some(edge => edge.caller === graphState.selected); collapse.textContent = graphState.collapsed.has(graphState.selected) ? 'Expand calls' : 'Collapse calls'; }
    const focus = document.getElementById('focus-calls');
    if (focus) { focus.disabled = !current; focus.textContent = graphState.focus ? 'Show full scope' : 'Neighbors only'; focus.setAttribute('aria-pressed', String(graphState.focus)); }
  }
  function select(id) { graphState.selected = id; updateSelection(); }
  function draw() {
    if (app.state.panel !== 'impact') return;
    if (graphState.snapshot !== app.state.snapshot) {
      movedPositions.clear(); graphState.snapshot = app.state.snapshot; graphState.selected = null; graphState.collapsed.clear(); graphState.focus = false;
    }
    const graph = app.snapshot();
    const filtered = app.filteredNodes();
    visible = model.visibleGraph(filtered, graph.edges, graphState.collapsed, graphState.focus ? graphState.selected : null);
    layout = model.layoutGraph(visible.nodes, visible.edges);
    for (const [id, position] of movedPositions) {
      if (layout.positions.has(id)) layout.positions.set(id, {...position});
    }
    elements.clear(); host.replaceChildren(); host.className = 'graph-host interactive';
    const toolbar = app.node('div', null, 'graph-toolbar');
    const zoomControls = app.node('div', null, 'graph-controls');
    zoomControls.append(control('−', () => zoom(1 / 1.2), 'Zoom out'), control('+', () => zoom(1.2), 'Zoom in'), control('Fit', fit, 'Fit all visible symbols'));
    const navigation = app.node('div', null, 'graph-controls');
    const collapse = control('Collapse calls', () => {
      if (!graphState.selected) return;
      if (graphState.collapsed.has(graphState.selected)) graphState.collapsed.delete(graphState.selected); else graphState.collapsed.add(graphState.selected);
      draw();
    }); collapse.id = 'collapse-calls';
    const focus = control('Neighbors only', () => { graphState.focus = !graphState.focus; draw(); }); focus.id = 'focus-calls';
    navigation.append(collapse, focus, control('Restore graph', () => { graphState.collapsed.clear(); graphState.focus = false; movedPositions.clear(); draw(); }));
    toolbar.append(zoomControls, navigation); host.append(toolbar);
    const viewport = app.node('div', null, 'graph-viewport');
    canvas = svgNode('svg', {class: 'impact-svg', tabindex: 0, role: 'group', 'aria-label': 'Static impact graph. Drag background to pan; drag nodes to rearrange. Plus and minus zoom; zero fits the graph. Tab selects nodes; Enter opens details.'});
    viewport.append(canvas); host.append(viewport);
    const width = viewport.clientWidth || 900;
    const height = viewport.clientHeight || 520;
    canvas.setAttribute('viewBox', `0 0 ${width} ${height}`);
    const defs = svgNode('defs');
    const marker = svgNode('marker', {id: 'call-arrow', viewBox: '0 0 10 10', refX: 9, refY: 5, markerWidth: 6, markerHeight: 6, orient: 'auto-start-reverse'});
    marker.append(svgNode('path', {d: 'M 0 0 L 10 5 L 0 10 z', fill: 'var(--muted)'})); defs.append(marker); canvas.append(defs);
    world = svgNode('g'); canvas.append(world);
    const edgeElements = [];
    function geometry(edge) {
      const from = layout.positions.get(edge.caller); const to = layout.positions.get(edge.callee);
      const sx = from.x + 260, sy = from.y + 44;
      const backward = to.x <= from.x;
      const tx = to.x + (backward ? 260 : 0), ty = to.y + 44;
      const curve = edge.caller === edge.callee
        ? `M ${sx} ${sy - 15} C ${sx + 74} ${sy - 74}, ${sx + 74} ${sy + 74}, ${sx} ${sy + 15}`
        : backward ? `M ${sx} ${sy} C ${sx + 64} ${sy}, ${tx + 64} ${ty}, ${tx} ${ty}`
          : `M ${sx} ${sy} C ${(sx + tx) / 2} ${sy}, ${(sx + tx) / 2} ${ty}, ${tx} ${ty}`;
      return {curve, x: (sx + tx) / 2 + (backward ? 50 : 0), y: (sy + ty) / 2 - 7};
    }
    visible.edges.forEach(edge => {
      const position = geometry(edge);
      const path = svgNode('path', {d: position.curve, class: `graph-edge ${edge.kind === 'DIRECT' ? '' : 'potential'}`, 'marker-end': 'url(#call-arrow)', 'data-caller': edge.caller, 'data-callee': edge.callee});
      path.append(svgNode('title', {}, `${edge.caller} → ${edge.callee} · ${edge.kind}`)); world.append(path);
      const label = edge.kind !== 'DIRECT' ? svgNode('text', {x: position.x, y: position.y, class: 'edge-kind'}, edge.kind.toLowerCase().replaceAll('_', ' ')) : null;
      if (label) world.append(label);
      edgeElements.push({edge, path, label});
    });
    refreshEdges = () => edgeElements.forEach(({edge, path, label}) => {
      const position = geometry(edge);
      path.setAttribute('d', position.curve);
      if (label) { label.setAttribute('x', position.x); label.setAttribute('y', position.y); }
    });
    visible.nodes.forEach((symbol, index) => {
      const position = layout.positions.get(symbol.id);
      const kind = symbol.impact === 'CHANGED' ? 'changed' : symbol.impact === 'CALLER' ? 'caller' : 'indirect';
      const group = svgNode('g', {transform: `translate(${position.x} ${position.y})`, class: `graph-node ${kind} ${symbol.role === 'TEST' ? 'test' : ''}`, tabindex: 0, role: 'button', 'aria-label': `${symbol.signature} · ${symbol.impact} · ${symbol.role}`, 'data-symbol': symbol.id});
      group.style.setProperty('--reveal-delay', `${Math.min(index, 12) * 18}ms`);
      const title = symbol.signature.split('#').pop();
      const shorten = (text, limit) => text.length > limit ? `${text.slice(0, limit - 1)}…` : text;
      const status = app.statusFor(symbol);
      const statusText = app.state.snapshot === 'before' ? 'BEFORE · static only' : symbol.role === 'TEST' ? 'TEST · attribution unknown' : status.replaceAll('_', ' ');
      group.append(svgNode('title', {}, `${symbol.signature}\n${symbol.path}:${symbol.line}\n${symbol.impact} · ${symbol.role}\n${statusText}`),
        svgNode('rect', {width: 260, height: 88, rx: 10}),
        svgNode('text', {x: 14, y: 22, class: 'node-title'}, shorten(title, 33)),
        svgNode('text', {x: 14, y: 40, class: 'node-owner'}, shorten(app.label(symbol.signature).split('#')[0], 38)),
        svgNode('text', {x: 14, y: 61, class: 'node-role'}, `${symbol.impact.replaceAll('_', ' ')} · ${symbol.role}`),
        svgNode('circle', {cx: 18, cy: 76, r: 3, class: `node-status ${status.toLowerCase().replaceAll('_', '-')}`}),
        svgNode('text', {x: 27, y: 79, class: 'node-evidence'}, `${graphState.collapsed.has(symbol.id) ? '⊕ ' : ''}${statusText}`));
      group.addEventListener('click', () => select(symbol.id));
      group.addEventListener('dblclick', () => app.selectSymbol(symbol.id, app.state.snapshot));
      group.addEventListener('keydown', event => {
        if (event.key === 'Enter' || event.key === ' ') { event.preventDefault(); event.stopPropagation(); select(symbol.id); app.selectSymbol(symbol.id, app.state.snapshot); }
      });
      elements.set(symbol.id, group); world.append(group);
    });
    if (!visible.nodes.length) viewport.append(app.node('div', 'No visible scope symbols. Reset filters or restore the graph.', 'graph-empty'));
    const inspector = app.node('div', null, 'graph-inspector'); inspector.id = 'graph-inspector'; inspector.setAttribute('aria-live', 'polite'); host.append(inspector);
    host.append(app.node('div', 'Drag background to pan; drag nodes to rearrange · wheel / + − to zoom · select a node, then Details or Enter · dashed edges are potential dispatch/reference/reflection calls', 'graph-help'));
    document.getElementById('graph-count').textContent = `${visible.nodes.length}/${graph.nodes.length} symbols · ${visible.edges.length}/${graph.edges.length} edges visible (filters / collapsed calls)`;
    let pointer = null;
    canvas.addEventListener('pointerdown', event => {
      if (event.button !== 0) return;
      const symbol = event.target.closest('.graph-node')?.dataset.symbol;
      if (symbol) select(symbol);
      pointer = {id: event.pointerId, symbol, x: event.clientX, y: event.clientY}; canvas.setPointerCapture(event.pointerId); canvas.classList.add('panning');
    });
    canvas.addEventListener('pointermove', event => {
      if (!pointer) return;
      const dx = event.clientX - pointer.x, dy = event.clientY - pointer.y;
      if (pointer.symbol) {
        const position = layout.positions.get(pointer.symbol);
        position.x = Math.max(16, position.x + dx / graphState.scale);
        position.y = Math.max(16, position.y + dy / graphState.scale);
        movedPositions.set(pointer.symbol, {...position});
        layout.width = Math.max(layout.width, position.x + 300);
        layout.height = Math.max(layout.height, position.y + 128);
        elements.get(pointer.symbol).setAttribute('transform', `translate(${position.x} ${position.y})`);
        refreshEdges();
      } else { graphState.x += dx; graphState.y += dy; transform(); }
      pointer.x = event.clientX; pointer.y = event.clientY;
    });
    const release = () => { pointer = null; canvas.classList.remove('panning'); };
    canvas.addEventListener('pointerup', release); canvas.addEventListener('pointercancel', release);
    canvas.addEventListener('wheel', event => {
      event.preventDefault(); const bounds = canvas.getBoundingClientRect();
      zoom(event.deltaY < 0 ? 1.12 : 1 / 1.12, {x: event.clientX - bounds.left, y: event.clientY - bounds.top});
    }, {passive: false});
    canvas.addEventListener('keydown', event => {
      if (event.key === '+' || event.key === '=') zoom(1.2);
      else if (event.key === '-') zoom(1 / 1.2);
      else if (event.key === '0') fit();
      else if (['ArrowLeft', 'ArrowRight', 'ArrowUp', 'ArrowDown'].includes(event.key)) {
        graphState.x += event.key === 'ArrowLeft' ? 40 : event.key === 'ArrowRight' ? -40 : 0;
        graphState.y += event.key === 'ArrowUp' ? 40 : event.key === 'ArrowDown' ? -40 : 0; transform();
      } else return;
      event.preventDefault();
    });
    fit(); updateSelection();
  }
  app.graphRenderer = draw;
  app.highlightSymbol = id => { graphState.selected = id; updateSelection(); };
  document.addEventListener('scryer:render', draw);
  let observedWidth = 0;
  new ResizeObserver(() => {
    if (canvas && app.state.panel === 'impact' && host.clientWidth > 0 && host.clientWidth !== observedWidth) {
      observedWidth = host.clientWidth;
      canvas.setAttribute('viewBox', `0 0 ${canvas.clientWidth} ${canvas.clientHeight}`); fit();
    }
  }).observe(host);
  draw();
})();
