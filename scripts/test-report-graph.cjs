const {test} = require('node:test');
const assert = require('node:assert/strict');
const {layoutGraph, visibleGraph} = require('../src/main/resources/analyze-report/graph-model.js');
const nodes = names => names.map(id => ({id}));
const edges = pairs => pairs.map(([caller, callee]) => ({caller, callee, kind: 'DIRECT'}));

test('branching calls remain caller to callee and all vertices receive finite positions', () => {
  const symbols = nodes(['A', 'B', 'C', 'D', 'E']);
  const calls = edges([['A', 'B'], ['B', 'C'], ['A', 'D'], ['B', 'E']]);
  const layout = layoutGraph(symbols, calls);
  assert.equal(layout.positions.size, symbols.length);
  for (const edge of calls) assert.ok(layout.positions.get(edge.caller).x < layout.positions.get(edge.callee).x);
  for (const position of layout.positions.values()) assert.ok(Number.isFinite(position.x) && Number.isFinite(position.y));
  assert.deepEqual(visibleGraph(symbols, calls).edges, calls);
});

test('cycles, self calls, shared nodes and disconnected components remain finite and complete', () => {
  const symbols = nodes(['A', 'B', 'C', 'isolated']);
  const calls = edges([['A', 'B'], ['B', 'A'], ['B', 'B'], ['A', 'C'], ['B', 'C']]);
  const layout = layoutGraph(symbols, calls);
  assert.equal(layout.positions.get('A').x, layout.positions.get('B').x);
  assert.ok(layout.positions.get('C').x > layout.positions.get('A').x);
  assert.deepEqual(new Set(visibleGraph(symbols, calls).nodes.map(node => node.id)), new Set(symbols.map(node => node.id)));
  assert.equal(visibleGraph(symbols, calls).edges.length, calls.length);
});

test('collapsing calls preserves children reachable through another expanded caller', () => {
  const symbols = nodes(['A', 'B', 'C', 'D']);
  const calls = edges([['A', 'B'], ['B', 'C'], ['A', 'D'], ['D', 'C']]);
  const collapsed = visibleGraph(symbols, calls, new Set(['B']));
  assert.ok(collapsed.nodes.some(node => node.id === 'C'));
  assert.ok(collapsed.edges.some(edge => edge.caller === 'D' && edge.callee === 'C'));
  assert.ok(!collapsed.edges.some(edge => edge.caller === 'B'));
  const allCollapsed = visibleGraph(symbols, calls, new Set(['A']));
  assert.deepEqual(allCollapsed.nodes.map(node => node.id), ['A']);
  assert.deepEqual(allCollapsed.edges, []);
  assert.deepEqual(visibleGraph(symbols, calls, new Set()).edges, calls);
});

test('rootless recursive components can collapse and restore without disappearing entirely', () => {
  const symbols = nodes(['A', 'B', 'C']);
  const calls = edges([['A', 'B'], ['B', 'C'], ['C', 'A']]);
  const result = visibleGraph(symbols, calls, new Set(['A']));
  assert.deepEqual(result.nodes.map(node => node.id), ['A']);
  assert.equal(visibleGraph(symbols, calls).nodes.length, 3);
});

test('one-hop focus and search subsets exclude unrelated vertices and dangling edges', () => {
  const symbols = nodes(['A', 'B', 'C', 'D']);
  const calls = edges([['A', 'B'], ['B', 'C'], ['C', 'D']]);
  const focus = visibleGraph(symbols, calls, new Set(), 'B');
  assert.deepEqual(focus.nodes.map(node => node.id), ['A', 'B', 'C']);
  const filtered = visibleGraph(nodes(['B', 'C']), calls);
  assert.deepEqual(filtered.edges, [calls[1]]);
});

test('deep chains use iterative traversal and preserve distinct edge kinds', () => {
  const symbols = nodes(Array.from({length: 12000}, (_, index) => String(index)));
  const calls = edges(symbols.slice(1).map((symbol, index) => [String(index), symbol.id]));
  calls.push({...calls[0], kind: 'REFLECTION'});
  const layout = layoutGraph(symbols, calls);
  assert.equal(layout.positions.size, 12000);
  assert.ok(layout.positions.get('11999').x > layout.positions.get('0').x);
  assert.equal(visibleGraph(symbols, calls).edges.length, calls.length);
  assert.deepEqual(visibleGraph([], []).nodes, []);
});
