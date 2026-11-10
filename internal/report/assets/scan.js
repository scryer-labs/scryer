(() => {
  "use strict";
  const byId = id => document.getElementById(id);
  const report = JSON.parse(byId("scan-data").textContent);
  const projects = report.projects || [];
  const picker = byId("project");
  let projectIndex = 0;
  let component = "";
  let file = "";
  let symbol = "";
  let initialize = true;

  function element(tag, text, className) {
    const node = document.createElement(tag);
    if (text != null) node.textContent = text;
    if (className) node.className = className;
    return node;
  }

  function button(label, action, pressed) {
    const node = element("button", label);
    node.type = "button";
    if (pressed != null) node.setAttribute("aria-pressed", String(pressed));
    node.addEventListener("click", action);
    return node;
  }

  function location(source) {
    if (!source?.file) return "";
    const line = source.start?.line ?? source.line;
    const column = source.start?.column ?? source.column;
    return [source.file, line, column].filter(value => value != null).join(":");
  }

  function selectedProject(index) {
    projectIndex = index;
    picker.value = String(index);
    component = "";
    file = "";
    symbol = "";
    initialize = true;
    render();
  }

  function cards(id, entries, active, select, empty) {
    const lane = byId(id).querySelector(".cards");
    const scroll = lane.scrollTop;
    lane.replaceChildren();
    if (!entries.length) lane.append(element("p", empty, "muted"));
    for (const entry of entries) {
      const card = button(entry.label, () => {
        select(entry.id);
        byId(id).querySelector('[aria-pressed="true"]')?.focus({preventScroll: true});
      }, entry.id === active);
      if (entry.caption) card.append(element("small", entry.caption));
      lane.append(card);
    }
    lane.scrollTop = scroll;
    const current = lane.querySelector('[aria-pressed="true"]');
    if (current) {
      const viewport = lane.getBoundingClientRect(), selected = current.getBoundingClientRect();
      if (selected.top < viewport.top || selected.bottom > viewport.bottom) {
        lane.scrollTop += selected.top - viewport.top - 8;
      }
    }
  }

  function detailList(title, values, item) {
    const detail = byId("detail");
    detail.append(element("h3", title));
    if (!values.length) {
      detail.append(element("p", "No matching facts in this report.", "muted"));
      return;
    }
    const list = element("ul");
    for (const value of values) {
      const row = element("li");
      item(row, value);
      list.append(row);
    }
    detail.append(list);
  }

  function renderWorkspace(project) {
    const area = byId("workspace");
    const deps = report.dependencies;
    area.replaceChildren();
    area.hidden = !deps;
    if (!deps) return;
    area.append(element("strong", "Workspace references"));
    area.append(element("p", deps.scope, "muted"));
    if (deps.unsupported?.length) area.append(element("p", "Unsupported: " + deps.unsupported.join(", "), "muted"));
    const relations = (deps.references || []).filter(ref => ref.from === project.path || ref.to === project.path);
    if (!relations.length) area.append(element("span", " · No references for this project.", "muted"));
    for (const ref of relations) {
      const other = ref.from === project.path ? ref.to : ref.from;
      const index = projects.findIndex(item => item.path === other);
      const label = ref.from + " → " + (ref.to || ref.repository?.id || "unknown") + " [" + ref.status + "]";
      area.append(index >= 0 ? button(label, () => selectedProject(index)) : element("p", label));
    }
  }

  function render() {
    const project = projects[projectIndex] || {};
    const facts = project.facts || {};
    const payload = kind => (facts.engineFacts || []).find(outcome => outcome.engine === "cpp" && outcome.fact === kind)?.data || {};
    const arch = payload("architecture");
    const source = payload("files");
    const graph = payload("call-graph");
    const components = arch.components || [];
    const symbols = graph.symbols || [];
    const calls = graph.calls || [];
    const referencedFiles = new Set([...symbols.map(item => item.definition?.file), ...calls.map(item => item.site?.file)].filter(Boolean));
    const files = source.files || Array.from(referencedFiles, path => ({path}));
    if (initialize) {
      initialize = false;
      const initialFile = files.find(item => item.componentIds?.length && referencedFiles.has(item.path)) || files[0];
      component = initialFile?.componentIds?.[0] || "";
      file = initialFile?.path || "";
      symbol = symbols.find(item => item.definition?.file === file)?.symbolId || "";
      for (const lane of document.querySelectorAll(".cards")) lane.scrollTop = 0;
    }
    const componentNames = new Map(components.map(item => [item.componentId, item.name]));
    const symbolsById = new Map(symbols.map(item => [item.symbolId, item]));
    const filePaths = new Set(files.map(item => item.path));
    const active = components.find(item => item.componentId === component);
    const selectedFiles = files.filter(item => !component || (item.componentIds || []).includes(component));
    const selectedSymbols = symbols.filter(item => item.definition?.file === file);
    const activeSymbol = symbolsById.get(symbol);
    const outgoing = calls.filter(item => item.site?.file === file && (!symbol || item.callerSymbolId === symbol));

    byId("repository").textContent = facts.repository?.id || facts.root || project.error || "";
    renderWorkspace(project);
    cards("architecture", [{id: "", label: "All source files", caption: files.length + " files"}, ...components.map(item => ({
      id: item.componentId, label: item.name, caption: [item.kind, item.type, location(item)].filter(Boolean).join(" · ")
    }))], component, value => {
      component = value;
      file = "";
      symbol = "";
      render();
    }, "Architecture facts were not selected.");
    cards("files", selectedFiles.map(item => ({
      id: item.path, label: item.path,
      caption: (item.componentIds || []).map(id => componentNames.get(id) || id).join(", ") || "No declared component"
    })), file, value => {
      file = value;
      symbol = "";
      render();
    }, "No files in this selection.");
    cards("calls", selectedSymbols.map(item => ({
      id: item.symbolId, label: item.name || item.kind, caption: location(item.definition)
    })), symbol, value => {
      symbol = value;
      render();
    }, file ? "No function definitions in this file." : "Select a file to inspect its functions.");

    const detail = byId("detail");
    detail.replaceChildren(element("h2", activeSymbol?.name || activeSymbol?.kind || file || active?.name || facts.name || project.path || "Project details"));
    if (activeSymbol) detail.append(element("p", location(activeSymbol.definition), "muted"));
    if (project.error) detail.append(element("p", project.error));
    if (active) {
      detailList("Component links", (arch.links || []).filter(item => item.fromComponentIds?.includes(component)), (row, item) => {
        row.append(document.createTextNode(item.from + " → " + item.to + " · " + location(item)));
        for (const id of item.toComponentIds || []) {
          row.append(button("Open " + (componentNames.get(id) || id), () => {
            component = id; file = ""; symbol = ""; render();
          }));
        }
      });
    }
    if (file && !symbol) {
      detailList("Include references", (source.references || []).filter(item => item.file === file), (row, item) => {
        row.append(document.createTextNode(location(item) + " → " + (item.to || item.target) + " [" + item.resolution + "]"));
        for (const candidate of item.candidates || []) {
          const evidence = (candidate.searchPaths || []).map(dir => location(dir)).filter(Boolean);
          row.append(element("small", candidate.path + (evidence.length ? " via " + evidence.join(", ") : "")));
        }
        if (item.to && filePaths.has(item.to)) row.append(button("Open file", () => {
          component = ""; file = item.to; render();
        }));
      });
    }
    if (file) {
      detailList("Syntactic call sites", outgoing, (row, item) => {
        row.append(document.createTextNode(location(item.site) + " → " + item.callee + " [" + item.resolution + "]"));
        const candidate = symbolsById.get(item.candidateSymbolId);
        if (candidate) row.append(button("Open candidate", () => {
          component = ""; file = candidate.definition.file; symbol = candidate.symbolId; render();
        }));
      });
    }
    if (!active && !file && !project.error) detail.append(element("p", "Select a component or file to inspect its relationships.", "muted"));
    if (byId("facts").parentElement.open) showRawFacts();
    requestAnimationFrame(drawConnections);
  }

  function showRawFacts() {
    byId("facts").textContent = JSON.stringify(projects[projectIndex]?.facts || {}, null, 2);
  }

  function drawConnections() {
    const svg = byId("connections");
    svg.replaceChildren();
    const box = svg.getBoundingClientRect();
    const ids = ["architecture", "files", "calls"];
    for (let i = 0; i < ids.length - 1; i++) {
      const left = byId(ids[i]).querySelector('[aria-pressed="true"]');
      const right = byId(ids[i + 1]).querySelector('[aria-pressed="true"]');
      if (!left || !right) continue;
      const a = left.getBoundingClientRect();
      const b = right.getBoundingClientRect();
      const x1 = a.right - box.left, x2 = b.left - box.left;
      const y1 = a.top + a.height / 2 - box.top, y2 = b.top + b.height / 2 - box.top;
      const aViewport = left.parentElement.getBoundingClientRect();
      const bViewport = right.parentElement.getBoundingClientRect();
      if (y1 + box.top < aViewport.top || y1 + box.top > aViewport.bottom ||
          y2 + box.top < bViewport.top || y2 + box.top > bViewport.bottom) continue;
      const middle = (x1 + x2) / 2;
      const curve = document.createElementNS("http://www.w3.org/2000/svg", "path");
      curve.setAttribute("d", `M ${x1} ${y1} C ${middle} ${y1}, ${middle} ${y2}, ${x2} ${y2}`);
      svg.append(curve);
    }
  }

  projects.forEach((project, i) => {
    const option = element("option", project.facts?.name || project.path || "Project " + (i + 1));
    option.value = String(i);
    picker.append(option);
  });
  if (report.root) byId("scope").textContent = report.root;
  picker.disabled = projects.length < 2;
  picker.addEventListener("change", () => selectedProject(Number(picker.value)));
  byId("facts").parentElement.addEventListener("toggle", event => {
    if (event.currentTarget.open) showRawFacts();
  });
  window.addEventListener("resize", drawConnections);
  document.querySelectorAll(".cards").forEach(lane => lane.addEventListener("scroll", drawConnections));
  render();
})();
