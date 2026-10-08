/* Pure graph operations. Iterative SCC traversal keeps recursive call graphs finite. */
((root) => {
  'use strict';
  function components(nodes, edges) {
    const ids = new Set(nodes.map(node => node.id));
    const outgoing = new Map(nodes.map(node => [node.id, []]));
    const incoming = new Map(nodes.map(node => [node.id, []]));
    edges.forEach(edge => {
      if (ids.has(edge.caller) && ids.has(edge.callee)) {
        outgoing.get(edge.caller).push(edge.callee);
        incoming.get(edge.callee).push(edge.caller);
      }
    });
    const visited = new Set();
    const finished = [];
    for (const id of ids) {
      if (visited.has(id)) continue;
      const pending = [{id, exit: false}];
      while (pending.length) {
        const item = pending.pop();
        if (item.exit) { finished.push(item.id); continue; }
        if (visited.has(item.id)) continue;
        visited.add(item.id); pending.push({id: item.id, exit: true});
        outgoing.get(item.id).forEach(target => { if (!visited.has(target)) pending.push({id: target, exit: false}); });
      }
    }
    const byId = new Map();
    const groups = [];
    for (const id of finished.reverse()) {
      if (byId.has(id)) continue;
      const group = [];
      const pending = [id];
      while (pending.length) {
        const current = pending.pop();
        if (byId.has(current)) continue;
        byId.set(current, groups.length); group.push(current);
        incoming.get(current).forEach(target => { if (!byId.has(target)) pending.push(target); });
      }
      groups.push(group.sort());
    }
    const dag = groups.map(() => new Set());
    const indegree = groups.map(() => 0);
    edges.forEach(edge => {
      const from = byId.get(edge.caller); const to = byId.get(edge.callee);
      if (from !== undefined && to !== undefined && from !== to && !dag[from].has(to)) { dag[from].add(to); indegree[to]++; }
    });
    return {groups, byId, dag, indegree, outgoing};
  }
  function visibleGraph(nodes, edges, collapsed = new Set(), focus = null) {
    let selected = nodes;
    if (focus) {
      const neighbors = new Set([focus]);
      edges.forEach(edge => { if (edge.caller === focus) neighbors.add(edge.callee); if (edge.callee === focus) neighbors.add(edge.caller); });
      selected = nodes.filter(node => neighbors.has(node.id));
    }
    const ids = new Set(selected.map(node => node.id));
    const selectedEdges = edges.filter(edge => ids.has(edge.caller) && ids.has(edge.callee));
    const model = components(selected, selectedEdges);
    const anchors = model.groups.filter((_, index) => model.indegree[index] === 0).map(group => group[0]);
    const visible = new Set();
    const pending = [...anchors];
    while (pending.length) {
      const id = pending.pop();
      if (visible.has(id)) continue;
      visible.add(id);
      if (!collapsed.has(id)) pending.push(...model.outgoing.get(id));
    }
    return {nodes: selected.filter(node => visible.has(node.id)), edges: selectedEdges.filter(edge => visible.has(edge.caller) && visible.has(edge.callee) && !collapsed.has(edge.caller))};
  }
  function layoutGraph(nodes, edges) {
    const model = components(nodes, edges);
    const indegree = [...model.indegree];
    const levels = model.groups.map(() => 0);
    const pending = model.groups.map((_, index) => index).filter(index => indegree[index] === 0);
    for (let index = 0; index < pending.length; index++) {
      const current = pending[index];
      model.dag[current].forEach(target => {
        levels[target] = Math.max(levels[target], levels[current] + 1);
        if (--indegree[target] === 0) pending.push(target);
      });
    }
    const columns = new Map();
    nodes.forEach(node => {
      const level = levels[model.byId.get(node.id)];
      if (!columns.has(level)) columns.set(level, []);
      columns.get(level).push(node);
    });
    const positions = new Map();
    let maxRows = 0; let maxColumn = 0;
    columns.forEach((column, level) => {
      column.sort((left, right) => left.id.localeCompare(right.id));
      column.forEach((node, row) => positions.set(node.id, {x: 32 + level * 356, y: 32 + row * 116}));
      maxRows = Math.max(maxRows, column.length); maxColumn = Math.max(maxColumn, level);
    });
    return {positions, width: nodes.length ? 356 * maxColumn + 356 : 356, height: Math.max(160, 116 * maxRows + 44), groups: model.groups};
  }
  const api = {layoutGraph, visibleGraph};
  if (typeof module !== 'undefined' && module.exports) module.exports = api;
  else root.ScryerGraphModel = api;
})(typeof window !== 'undefined' ? window : this);
