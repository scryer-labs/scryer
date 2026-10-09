(() => {
  'use strict';
  const report = JSON.parse(document.getElementById('scryer-data').textContent);
  const state = {dataset: 0, panel: 'evidence', snapshot: 'after', search: '', status: 'ALL', impact: 'ALL', role: 'ALL'};
  const $ = id => document.getElementById(id);
  const words = value => String(value || 'unknown').replaceAll('_', ' ').toLowerCase();
  const node = (tag, text, className) => {
    const element = document.createElement(tag);
    if (text !== undefined && text !== null) element.textContent = String(text);
    if (className) element.className = className;
    return element;
  };
  const shortSignature = signature => {
    const separator = signature.indexOf('#');
    return separator < 0 ? signature : signature.slice(0, separator).split('.').pop() + signature.slice(separator);
  };
  const signatures = [...new Set([...report.before.nodes, ...report.after.nodes].map(symbol => symbol.signature))];
  const labelCounts = new Map();
  signatures.forEach(signature => labelCounts.set(shortSignature(signature), (labelCounts.get(shortSignature(signature)) || 0) + 1));
  const label = signature => labelCounts.get(shortSignature(signature)) > 1 ? signature : shortSignature(signature);
  const sourceLabel = (path, line) => `${(path || 'unknown').split('/').pop()}:${line}`;
  const sourceNode = (path, line) => {
    const element = node('span', sourceLabel(path, line), 'source-path');
    element.title = `${path}:${line}`;
    return element;
  };
  const replace = (element, children) => element.replaceChildren(...children);
  const dataset = () => report.matching.datasets[state.dataset];
  const snapshot = () => report[state.snapshot];
  const percent = value => value === null ? 'Unknown' : `${value.toFixed(1)}%`;
  const badge = (value, type) => node('span', words(value), `tag ${type || words(value).replaceAll(' ', '-')}`);
  const impactClass = value => value === 'CHANGED' ? 'changed' : value === 'CALLER' ? 'caller' : 'indirect';
  let indexedDataset = -1;
  let indexedMethods = new Map();
  const methodIndex = () => {
    if (indexedDataset !== state.dataset) {
      indexedDataset = state.dataset;
      indexedMethods = new Map(dataset().methods.map(method => [method.symbol, method]));
    }
    return indexedMethods;
  };
  const statusFor = (symbol, snapshotName = state.snapshot) => {
    if (snapshotName !== 'after' || symbol.role === 'TEST') return 'NOT_APPLICABLE';
    return methodIndex().get(symbol.id)?.status || 'UNKNOWN';
  };
  const matches = item => {
    const text = [item.symbol, item.id, item.signature, item.path].filter(Boolean).join(' ').toLowerCase();
    const status = item.status || statusFor(item);
    return text.includes(state.search) && (state.role === 'ALL' || item.role === state.role)
      && (state.impact === 'ALL' || item.impact === state.impact)
      && ((state.panel === 'impact' && state.snapshot === 'before') || state.status === 'ALL' || (state.status === 'GAPS' ? ['NOT_EXECUTED', 'PARTIALLY_EXECUTED', 'UNKNOWN'].includes(status) : status === state.status));
  };
  const filteredNodes = () => snapshot().nodes.filter(matches);
  const symbolButton = (symbol, text, snapshotName = state.snapshot) => {
    const button = node('button', label(text), 'symbol-button');
    button.type = 'button';
    button.title = text;
    button.addEventListener('click', () => selectSymbol(symbol, snapshotName));
    return button;
  };
  function simpleTable(headers, rows) {
    const wrap = node('div', null, 'table-scroll');
    const table = node('table');
    const head = node('thead');
    const heading = node('tr');
    headers.forEach(text => heading.append(node('th', text)));
    head.append(heading);
    const body = node('tbody');
    rows.forEach(values => {
      const row = node('tr');
      values.forEach(value => { const cell = node('td'); cell.append(value instanceof Node ? value : node('span', value ?? 'unknown')); row.append(cell); });
      body.append(row);
    });
    table.append(head, body); wrap.append(table); return wrap;
  }
  function facts(values) {
    const list = node('dl', null, 'facts');
    values.forEach(([label, value]) => list.append(node('dt', label), node('dd', value ?? 'unknown')));
    return list;
  }
  function resetFilters() {
    state.search = ''; state.status = 'ALL'; state.impact = 'ALL'; state.role = 'ALL';
    $('search-input').value = ''; $('status-filter').value = 'ALL'; $('impact-filter').value = 'ALL'; $('role-filter').value = 'ALL';
    render();
  }
  function showPanel(panel) {
    state.panel = panel;
    for (const name of ['evidence', 'impact', 'changes', 'execution']) {
      $(`${name}-panel`).hidden = name !== panel;
      $(`tab-${name}`).setAttribute('aria-selected', String(name === panel));
      $(`tab-${name}`).tabIndex = name === panel ? 0 : -1;
    }
    $('filters').hidden = !['evidence', 'impact'].includes(panel);
    render();
  }
  function renderMetrics() {
    const current = dataset();
    const metrics = [
      ['Not executed', current.counts.NOT_EXECUTED, 'Matched methods with no hits', 'not-executed', 'NOT_EXECUTED'],
      ['Partially executed', current.counts.PARTIALLY_EXECUTED, 'Missed instructions or branches', 'partial', 'PARTIALLY_EXECUTED'],
      ['Unknown', current.counts.UNKNOWN, 'Evidence cannot be established', 'unknown', 'UNKNOWN'],
      ['Method execution', percent(current.methodExecutionPercent), `${current.withHits}/${current.assessed} assessed · unknown excluded`, 'execution', 'ALL']
    ];
    replace($('metrics'), metrics.map(([label, value, explanation, className, filter]) => {
      const card = node('button', null, `metric ${className}`); card.type = 'button';
      card.append(node('span', label, 'metric-label'), node('span', value, 'metric-value'), node('span', explanation, 'metric-sub'));
      card.setAttribute('aria-label', `${label}: ${value}. Filter methods`);
      card.addEventListener('click', () => { resetFilters(); state.status = filter; $('status-filter').value = filter; showPanel('evidence'); });
      return card;
    }));
  }
  function renderMethods() {
    const impactOrder = {CHANGED: 0, CALLER: 1, POTENTIAL_INDIRECT: 2};
    const evidenceOrder = {NOT_EXECUTED: 0, UNKNOWN: 1, PARTIALLY_EXECUTED: 2, EXECUTED: 3};
    const methods = dataset().methods.filter(matches).sort((left, right) =>
      impactOrder[left.impact] - impactOrder[right.impact] || evidenceOrder[left.status] - evidenceOrder[right.status] || left.symbol.localeCompare(right.symbol));
    replace($('method-table'), methods.map(method => {
      const row = node('tr');
      const name = node('td');
      name.append(symbolButton(method.symbol, method.signature, 'after'), sourceNode(method.path, method.line));
      const impact = node('td'); impact.append(badge(method.impact, impactClass(method.impact)));
      const evidence = node('td'); evidence.append(badge(method.status));
      row.append(name, impact, evidence, node('td', method.instructions ? `${method.instructions.covered} / ${method.instructions.missed}` : 'unknown', 'numeric'), node('td', method.branches?.missed ?? 'unknown', 'numeric'));
      return row;
    }));
    $('method-count').textContent = `${methods.length} / ${dataset().methods.length} production methods`;
    $('method-empty').hidden = methods.length !== 0;
  }
  function renderImpact() {
    const graph = snapshot();
    const visible = filteredNodes();
    const ids = new Set(visible.map(symbol => symbol.id));
    const edges = graph.edges.filter(edge => ids.has(edge.caller) && ids.has(edge.callee));
    $('graph-count').textContent = `${visible.length}/${graph.nodes.length} symbols · ${edges.length}/${graph.edges.length} edges shown`;
    if (!window.ScryerReport?.graphRenderer) {
      replace($('graph-host'), visible.length ? visible.map(symbol => {
        const card = node('button', null, 'graph-card'); card.type = 'button';
        card.append(node('strong', symbol.name || label(symbol.signature) || symbol.id), node('span', symbol.owner || symbol.module || '', 'source-path'), badge(symbol.impact, impactClass(symbol.impact)), node('span', words(symbol.role), 'source-path'));
        card.addEventListener('click', () => selectSymbol(symbol.id, state.snapshot)); return card;
      }) : [node('div', 'No matching scope symbols. Reset filters or select another snapshot.', 'empty')]);
    }
    const names = new Map(graph.nodes.map(symbol => [symbol.id, symbol.signature]));
    replace($('edge-table'), edges.map(edge => {
      const row = node('tr');
      const caller = node('td'); caller.append(symbolButton(edge.caller, names.get(edge.caller) || edge.caller));
      const callee = node('td'); callee.append(symbolButton(edge.callee, names.get(edge.callee) || edge.callee));
      row.append(caller, node('td', words(edge.kind)), callee); return row;
    }));
    $('boundary-summary').textContent = `Call boundaries · ${graph.boundaries.length} scope/unknown caller · ${graph.totalBoundaryCount} repository sites`;
    replace($('boundary-list'), graph.boundaries.map(boundary => {
      const item = node('div', null, 'boundary');
      item.append(node('strong', boundary.reason), node('span', `${boundary.path}:${boundary.line} · ${boundary.caller || 'Unknown caller'}`, 'source-path'), node('code', boundary.expression)); return item;
    }));
    document.dispatchEvent(new CustomEvent('scryer:render'));
  }
  function renderChanges() {
    replace($('file-changes'), [simpleTable(['Status', 'Before path', 'After path', 'Line ranges'], report.files.map(file => [file.status, file.beforePath || '(added)', file.afterPath || '(deleted)', file.lines.map(range => `before ${range.before.start} +${range.before.count} → after ${range.after.start} +${range.after.count}`).join('; ')]))]);
    replace($('declaration-changes'), report.changes.length ? report.changes.map(change => {
      const block = node('article', null, 'change-block');
      block.append(node('h3', `${change.kind} · ${label((change.after || change.before).signature)}`));
      const pair = node('div', null, 'source-pair');
      for (const [label, method] of [['Before', change.before], ['After', change.after]]) {
        const column = node('div'); column.append(node('h4', label.toUpperCase()));
        column.append(node('span', method ? `${sourceLabel(method.path, method.lines.start)} · ${method.lines.count} line(s)` : 'Declaration absent', 'source-path'));
        column.append(node('pre', method?.source || '(absent)')); pair.append(column);
      }
      block.append(pair); return block;
    }) : [node('div', 'No method changes identified. File/class context impact is not analyzed yet.', 'empty')]);
  }
  function renderExecution() {
    const execution = report.execution;
    replace($('execution-facts'), [facts([['After SHA', execution.afterSha], ['Command status', execution.status], ['Command', execution.command.join(' ') || '(not executed)'], ['Target JAVA_HOME', execution.javaHome], ['Exit code', execution.exitCode], ['Duration', `${execution.durationMillis}ms`], ['Run log', execution.log], ['Artifact manifest', report.evidence.manifest]])]);
    replace($('test-reports'), report.evidence.tests.length ? report.evidence.tests.map(test => {
      const block = node('div', null, 'report-block'); block.append(node('h4', test.artifact.source));
      block.append(simpleTable(['Class', 'Test record', 'Status', 'Seconds'], test.cases.map(record => [record.className, record.name, record.status, record.seconds]))); return block;
    }) : [node('div', 'No fresh usable test XML reports. Test counts are unknown.', 'empty')]);
    replace($('coverage-reports'), report.evidence.coverage.length ? report.evidence.coverage.map(coverage => {
      const block = node('details', null, 'report-block'); block.append(node('summary', `${coverage.format} · ${coverage.artifact.source} · ${coverage.classes.length} classes`));
      block.append(simpleTable(['Class', 'Match', 'Class ID', 'Compiled output'], coverage.classes.map(owner => [owner.name, owner.match, owner.classId, owner.compiledFile])));
      coverage.classes.forEach(owner => {
        const detail = node('details'); detail.append(node('summary', `${owner.name} · ${owner.methods.length} raw method records`));
        detail.append(simpleTable(['Binary symbol', 'Instructions hit/missed', 'Branches hit/missed', 'Lines hit/missed'], owner.methods.map(method => [method.symbol, `${method.instructions.covered}/${method.instructions.missed}`, `${method.branches.covered}/${method.branches.missed}`, `${method.lines.covered}/${method.lines.missed}`]))); block.append(detail);
      });
      coverage.notes.forEach(note => block.append(node('p', note, 'muted small'))); return block;
    }) : [node('div', 'Coverage unavailable. This is not 0% coverage.', 'empty')]);
    replace($('artifact-table'), report.evidence.artifacts.map(artifact => {
      const row = node('tr'); row.append(node('td', artifact.source, 'mono'), node('td', artifact.retained, 'mono'), node('td', artifact.sha256, 'mono')); return row;
    }));
  }
  function selectSymbol(id, snapshotName) {
    const graph = report[snapshotName];
    const symbol = graph.nodes.find(item => item.id === id);
    if (!symbol) return;
    const content = $('symbol-content'); content.replaceChildren();
    const title = node('h2', symbol.name || label(symbol.signature) || symbol.id); title.id = 'symbol-title';
    content.append(title, node('p', symbol.owner || symbol.module || '', 'symbol-owner'));
    const tags = node('div', null, 'symbol-tags'); tags.append(badge(symbol.impact, impactClass(symbol.impact)), badge(symbol.role), badge(snapshotName)); content.append(tags);
    function section(label) { const part = node('section', null, 'detail-section'); part.append(node('h3', label)); content.append(part); return part; }
    section('Identity & source').append(facts([['Signature', symbol.signature], ['Binary symbol', symbol.id], ['Source', symbol.path ? `${symbol.path}:${symbol.line}` : 'unavailable'], ['Module', symbol.module]]));
    const method = snapshotName === 'after' ? methodIndex().get(id) : null;
    const evidence = section('After method execution evidence');
    if (method) {
      evidence.append(badge(method.status), node('p', method.reason));
      evidence.append(facts([['Dataset', dataset().artifact?.source || 'unavailable'], ['Instruction hits', method.instructions?.covered], ['Missed instructions', method.instructions?.missed], ['Missed branches', method.branches?.missed]]));
      const routes = section('Static test-route candidates · attribution unknown');
      method.routes.forEach(route => routes.append(node('p', `${words(route.kind)} · ${route.test} · ${route.passedClassRecords} passed class records`)));
      if (!method.routes.length) routes.append(node('p', 'No static test route found. Framework/HTTP boundaries can still produce real execution hits.'));
    } else evidence.append(node('p', snapshotName === 'before' ? 'Coverage was collected only for after. It is not attributed to this before symbol.' : symbol.role === 'TEST' ? 'Test symbol; production-method coverage metrics do not apply. A test class or call edge is not execution evidence.' : 'No production matching record; source-role or method evidence remains unknown.'));
    const calls = section('Resolved static calls · caller → callee');
    graph.edges.filter(edge => edge.caller === id || edge.callee === id).forEach(edge => {
      const other = edge.caller === id ? edge.callee : edge.caller;
      const button = node('button', `${edge.caller === id ? 'Calls' : 'Called by'} · ${label(graph.nodes.find(node => node.id === other)?.signature || other)} · ${words(edge.kind)}`);
      button.title = other; button.type = 'button'; button.addEventListener('click', () => selectSymbol(other, snapshotName)); calls.append(button);
    });
    if (calls.childElementCount === 1) calls.append(node('p', 'No resolved scope edges. Unresolved calls remain boundaries.'));
    const boundaries = graph.boundaries.filter(boundary => boundary.caller === id);
    if (boundaries.length) { const sectionNode = section('Call boundaries'); boundaries.forEach(boundary => sectionNode.append(node('p', `${boundary.reason} · ${boundary.path}:${boundary.line}`), node('pre', boundary.expression))); }
    const change = report.changes.find(change => change.before?.signature === symbol.signature || change.after?.signature === symbol.signature);
    if (change) { const source = section('Changed declaration'); source.append(node('p', 'Before'), node('pre', change.before?.source || '(absent)'), node('p', 'After'), node('pre', change.after?.source || '(absent)')); }
    if (!$('symbol-dialog').open) $('symbol-dialog').showModal();
    window.ScryerReport?.highlightSymbol?.(id);
  }
  function render() {
    $('status-filter').disabled = state.panel === 'impact' && state.snapshot === 'before';
    renderMetrics(); renderMethods(); renderImpact();
  }

  const repoName = report.repository.replaceAll('\\', '/').split('/').filter(Boolean).pop();
  $('project-name').textContent = repoName || 'Repository analysis';
  document.title = `Scryer · ${repoName || 'Analysis report'}`;
  $('repository').textContent = report.repository;
  $('before-sha').textContent = report.beforeSha.slice(0, 12); $('before-sha').title = report.beforeSha;
  $('after-sha').textContent = report.afterSha.slice(0, 12); $('after-sha').title = report.afterSha;
  $('change-count').textContent = `${report.files.length} changed files · ${report.changes.length} changed methods`;
  $('run-status').textContent = `Command ${words(report.execution.status)}`;
  $('run-status').className = `badge ${report.execution.status === 'SUCCEEDED' ? 'good' : report.execution.status === 'SKIPPED' ? 'unknown' : 'bad'}`;
  $('generated-at').textContent = report.generatedAt;
  $('schema-version').textContent = report.schemaVersion;
  $('evidence-count').textContent = dataset().methods.length;
  $('impact-count').textContent = report.after.nodes.length;
  $('changes-count').textContent = report.changes.length;
  replace($('dataset-select'), report.matching.datasets.map((item, index) => { const option = node('option', item.artifact?.source || 'Coverage unavailable'); option.value = index; return option; }));
  const notes = [...new Set([...report.notes, ...report.before.notes, ...report.after.notes, ...report.execution.notes, ...report.evidence.notes, ...report.matching.notes, ...report.matching.removed.map(symbol => `Removed before symbol: ${symbol}; after coverage not applicable.`), ...report.before.unresolvedChanges.map(symbol => `Before unresolved change: ${symbol}`), ...report.matching.unresolvedChanges.map(symbol => `After unresolved change: ${symbol}`)])];
  $('limits-count').textContent = notes.length;
  replace($('limits-list'), notes.map(note => node('p', note)));
  $('dataset-select').addEventListener('change', event => { state.dataset = Number(event.target.value); $('evidence-count').textContent = dataset().methods.length; $('symbol-dialog').close(); render(); });
  $('snapshot-select').addEventListener('change', event => { state.snapshot = event.target.value; $('symbol-dialog').close(); render(); });
  $('search-input').addEventListener('input', event => { state.search = event.target.value.toLowerCase(); render(); });
  for (const key of ['status', 'impact', 'role']) $(`${key}-filter`).addEventListener('change', event => { state[key] = event.target.value; render(); });
  $('reset-filters').addEventListener('click', resetFilters);
  document.querySelectorAll('[data-panel]').forEach(tab => tab.addEventListener('click', () => showPanel(tab.dataset.panel)));
  document.querySelector('[role=tablist]').addEventListener('keydown', event => {
    const tabs = [...document.querySelectorAll('[data-panel]')]; const index = tabs.indexOf(document.activeElement);
    if (index < 0 || !['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
    event.preventDefault();
    const next = event.key === 'Home' ? 0 : event.key === 'End' ? tabs.length - 1 : (index + (event.key === 'ArrowRight' ? 1 : -1) + tabs.length) % tabs.length;
    tabs[next].focus(); showPanel(tabs[next].dataset.panel);
  });
  $('close-symbol').addEventListener('click', () => $('symbol-dialog').close());
  $('theme-button').addEventListener('click', () => { document.documentElement.dataset.theme = document.documentElement.dataset.theme === 'dark' ? 'light' : 'dark'; });
  $('download-json').addEventListener('click', () => {
    const url = URL.createObjectURL(new Blob([JSON.stringify(report, null, 2) + '\n'], {type: 'application/json'}));
    const link = node('a'); link.href = url; link.download = 'scryer-analysis.json'; link.click(); setTimeout(() => URL.revokeObjectURL(url), 1000);
  });
  window.ScryerReport = {report, state, node, label, dataset, snapshot, filteredNodes, statusFor, selectSymbol, render, graphRenderer: null};
  renderChanges(); renderExecution(); render();
})();
