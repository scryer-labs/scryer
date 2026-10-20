(() => {
  'use strict';
  const doc = JSON.parse(document.getElementById('data').textContent);
  const {projects, dependencies} = doc.facts;
  const svg = document.getElementById('map');
  const ns = 'http://www.w3.org/2000/svg';
  const node = (tag, attributes = {}, text) => {
    const e = document.createElementNS(ns, tag);
    for (const [k,v] of Object.entries(attributes)) e.setAttribute(k,v);
    if (text !== undefined) e.textContent = text;
    return e;
  };
  const short = id => id.split(':').at(-1);
  const internal = new Set(projects.map(p => p.id));
  const ids = [...new Set([...internal, ...dependencies.map(e => e.target)])].sort();
  document.getElementById('summary').textContent = `${projects.length} projects · ${dependencies.length} declared relationships`;
  document.getElementById('semantics').textContent = 'Repository facts only. No code is executed, no upgrade strategy is selected, and no test confidence is inferred.';
  for (const note of doc.notes) { const li = document.createElement('li'); li.textContent = note; document.getElementById('notes').append(li); }
  if (doc.suggestion) {
    document.getElementById('title').textContent = 'Suggested refactor phases';
    document.getElementById('summary').textContent += ` · ${doc.suggestion.phases.length} phases · ${doc.suggestion.blocked.length} blocked`;
    document.getElementById('semantics').textContent = 'Dependency-based suggestions, not an execution plan or safety guarantee. Columns use internal declared edges; arrows still point from consumer to dependency.';
  }
  const positions = new Map();
  const labels = [];
  function initialPositions() {
    positions.clear(); labels.length = 0;
    if (doc.suggestion) {
      const groups = doc.suggestion.phases.map(p => [p.id, p.projects]);
      if (doc.suggestion.blocked.length) groups.push(['Blocked', doc.suggestion.blocked]);
      const external = ids.filter(id => !internal.has(id));
      if (external.length) groups.push(['External', external]);
      groups.forEach(([label, members], col) => {
        labels.push({label,x:col*300,y:0});
        members.forEach((id,row) => positions.set(id,{x:col*300,y:40+row*95}));
      });
    } else ids.forEach((id,i) => positions.set(id,{x:(i%4)*300,y:Math.floor(i/4)*95}));
  }
  let world, scale = 1, x = 20, y = 20, selected = null;
  const transform = () => world.setAttribute('transform', `translate(${x} ${y}) scale(${scale})`);
  function fit() {
    const visible = [...positions.entries()].filter(([id]) => matches(id)).map(([,p]) => p);
    if (doc.suggestion) visible.push(...labels.map(p => ({x:p.x,y:p.y+35})));
    if (!visible.length) return;
    const minX=Math.min(...visible.map(p=>p.x)),minY=Math.min(...visible.map(p=>p.y))-35;
    const width=Math.max(...visible.map(p=>p.x))-minX+270,height=Math.max(...visible.map(p=>p.y))-minY+100;
    scale=Math.min(1,(svg.clientWidth-40)/width,(svg.clientHeight-40)/height);
    x=(svg.clientWidth-width*scale)/2-minX*scale;y=(svg.clientHeight-height*scale)/2-minY*scale;transform();
  }
  const matches = id => id.toLowerCase().includes(document.getElementById('search').value.toLowerCase());
  function inspect(id) {
    selected=id;
    const outgoing=dependencies.filter(e=>e.source===id).map(e=>e.target);
    const incoming=dependencies.filter(e=>e.target===id).map(e=>e.source);
    document.getElementById('details').textContent=`${id}\n${internal.has(id)?'Internal project':'External target'}\nDepends on: ${outgoing.join(', ')||'none observed'}\nDirect consumers: ${incoming.join(', ')||'none observed'}`;
    world.querySelectorAll('.node').forEach(e=>e.classList.toggle('selected',e.dataset.id===id));
    world.querySelectorAll('.edge').forEach(e=>e.classList.toggle('active',e.dataset.source===id||e.dataset.target===id));
  }
  function draw() {
    svg.replaceChildren();
    const defs=node('defs'),marker=node('marker',{id:'arrow',viewBox:'0 0 10 10',refX:9,refY:5,markerWidth:7,markerHeight:7,orient:'auto'});
    marker.append(node('path',{d:'M0 0 L10 5 L0 10 Z',fill:'#778bab'}));defs.append(marker);svg.append(defs);
    world=node('g');svg.append(world);transform();
    for(const label of labels) world.append(node('text',{class:'phase',x:label.x,y:label.y},label.label));
    const edges=[];
    function updateEdges() {
      for(const {e,path} of edges) {
        const a=positions.get(e.source),b=positions.get(e.target);
        const ax=a.x+125,ay=a.y+28,bx=b.x+125,by=b.y+28;
        const dx=bx-ax,dy=by-ay;
        const fraction=dx||dy?Math.min(dx?125/Math.abs(dx):Infinity,dy?28/Math.abs(dy):Infinity):0;
        const sx=ax+dx*fraction,sy=ay+dy*fraction,tx=bx-dx*fraction,ty=by-dy*fraction;
        path.setAttribute('d',e.source===e.target?`M${ax} ${ay-28} C${ax+170} ${ay-110} ${ax+170} ${ay+110} ${ax+125} ${ay}`:`M${sx} ${sy} L${tx} ${ty}`);
      }
    }
    for(const e of dependencies) {
      if(!matches(e.source)||!matches(e.target)) continue;
      const path=node('path',{class:'edge','marker-end':'url(#arrow)'});path.dataset.source=e.source;path.dataset.target=e.target;
      path.append(node('title',{},`${e.source} → ${e.target}`));world.append(path);edges.push({e,path});
    }
    for(const id of ids) {
      if(!matches(id)) continue;
      const p=positions.get(id),g=node('g',{class:`node ${internal.has(id)?'':'external'}`,transform:`translate(${p.x} ${p.y})`,tabindex:0,role:'button','aria-label':id});g.dataset.id=id;
      g.append(node('rect',{width:250,height:56,rx:10}),node('title',{},id),node('text',{x:12,y:32},short(id).length>29?short(id).slice(0,26)+'…':short(id)));
      g.addEventListener('click',()=>inspect(id));g.addEventListener('keydown',e=>{if(e.key==='Enter'||e.key===' '){e.preventDefault();inspect(id);}});
      g.addEventListener('pointerdown',e=>{
        e.stopPropagation();if(e.button!==0)return;g.setPointerCapture(e.pointerId);
        let px=e.clientX,py=e.clientY;
        const move=ev=>{p.x+=(ev.clientX-px)/scale;p.y+=(ev.clientY-py)/scale;px=ev.clientX;py=ev.clientY;g.setAttribute('transform',`translate(${p.x} ${p.y})`);updateEdges();};
        const stop=()=>{g.removeEventListener('pointermove',move);};g.addEventListener('pointermove',move);g.addEventListener('pointerup',stop,{once:true});g.addEventListener('pointercancel',stop,{once:true});
      });world.append(g);
    }
    updateEdges();if(selected)inspect(selected);
    const body=document.getElementById('edges');body.replaceChildren();
    for(const e of dependencies.filter(e=>matches(e.source)&&matches(e.target))){const row=document.createElement('tr');for(const value of [e.source,e.target,e.scopes.map(s=>s||'unspecified').join(', '),e.internal?'internal':'external']){const cell=document.createElement('td');cell.textContent=value;row.append(cell);}body.append(row);}
  }
  function zoom(factor){const next=Math.min(4,Math.max(.01,scale*factor)),ratio=next/scale;const cx=svg.clientWidth/2,cy=svg.clientHeight/2;x=cx-(cx-x)*ratio;y=cy-(cy-y)*ratio;scale=next;transform();}
  svg.addEventListener('wheel',e=>{e.preventDefault();zoom(e.deltaY<0?1.1:1/1.1);},{passive:false});
  svg.addEventListener('pointerdown',e=>{if(e.button!==0)return;svg.setPointerCapture(e.pointerId);let px=e.clientX,py=e.clientY;const move=ev=>{x+=ev.clientX-px;y+=ev.clientY-py;px=ev.clientX;py=ev.clientY;transform();};const stop=()=>svg.removeEventListener('pointermove',move);svg.addEventListener('pointermove',move);svg.addEventListener('pointerup',stop,{once:true});svg.addEventListener('pointercancel',stop,{once:true});});
  document.getElementById('fit').onclick=fit;document.getElementById('in').onclick=()=>zoom(1.2);document.getElementById('out').onclick=()=>zoom(1/1.2);
  document.getElementById('reset').onclick=()=>{initialPositions();draw();fit();};document.getElementById('search').oninput=()=>{draw();fit();};
  window.addEventListener('resize',fit);initialPositions();draw();fit();
})();
