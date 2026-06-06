const sampleURL = './sample.depviz';

const dom = {
  input: document.getElementById('sourceInput'),
  syntax: document.getElementById('syntaxLayer'),
  status: document.getElementById('status'),
  lineCount: document.getElementById('lineCount'),
  error: document.getElementById('errorText'),
  boardTitle: document.getElementById('boardTitle'),
  boardMeta: document.getElementById('boardMeta'),
  filter: document.getElementById('filterInput'),
  stats: document.getElementById('stats'),
  brief: document.getElementById('briefView'),
  graph: document.getElementById('graphView'),
  graphCanvas: document.getElementById('graphCanvas'),
  table: document.getElementById('tableView'),
  showExternal: document.getElementById('showExternal'),
  showLocal: document.getElementById('showLocal'),
  showClosed: document.getElementById('showClosed'),
};

const state = {
  view: 'brief',
  filter: '',
  showExternal: true,
  showLocal: true,
  showClosed: false,
  data: emptyExport(),
};

function emptyExport() {
  return {
    snapshot: {
      board: { id: 'default', name: 'Default' },
      nodes: [],
      edges: [],
    },
    brief: {
      board_name: 'Default',
      next_move: null,
      ready: [],
      blockers: [],
      local_only: [],
      stale: [],
      counts: { nodes: 0, edges: 0, ready: 0, blocked: 0, local_only: 0, stale: 0 },
    },
  };
}

function boot() {
  wireEvents();
  const hashed = readHash();
  if (hashed) {
    dom.input.value = hashed;
    update();
    return;
  }
  loadSample();
}

function wireEvents() {
  dom.input.addEventListener('input', update);
  dom.input.addEventListener('scroll', syncHighlightScroll);
  dom.filter.addEventListener('input', () => {
    state.filter = dom.filter.value.toLowerCase();
    render();
  });
  for (const key of ['showExternal', 'showLocal', 'showClosed']) {
    dom[key].addEventListener('change', () => {
      state[key] = dom[key].checked;
      render();
    });
  }
  for (const btn of document.querySelectorAll('[data-view]')) {
    btn.addEventListener('click', () => {
      state.view = btn.dataset.view;
      document.querySelectorAll('[data-view]').forEach((item) => {
        item.classList.toggle('active', item === btn);
      });
      render();
    });
  }
  document.getElementById('sampleBtn').addEventListener('click', loadSample);
  document.getElementById('shareBtn').addEventListener('click', shareLink);
  document.getElementById('exportBtn').addEventListener('click', exportJSON);
  document.getElementById('fileInput').addEventListener('change', readFile);
}

async function loadSample() {
  const res = await fetch(sampleURL);
  dom.input.value = await res.text();
  update();
}

function readFile(event) {
  const file = event.target.files && event.target.files[0];
  if (!file) return;
  const reader = new FileReader();
  reader.onload = () => {
    dom.input.value = String(reader.result || '');
    update();
  };
  reader.readAsText(file);
}

function update() {
  const text = dom.input.value;
  updateHighlight(text);
  dom.lineCount.textContent = `${countLines(text)} lines`;
  try {
    state.data = parseInput(text);
    dom.error.textContent = '';
    dom.status.textContent = 'stateless work graph';
  } catch (err) {
    state.data = emptyExport();
    dom.error.textContent = err.message;
    dom.status.textContent = 'input error';
  }
  render();
}

function updateHighlight(text) {
  dom.syntax.innerHTML = `${highlightInput(text)}\n`;
  syncHighlightScroll();
}

function syncHighlightScroll() {
  dom.syntax.scrollTop = dom.input.scrollTop;
  dom.syntax.scrollLeft = dom.input.scrollLeft;
}

function highlightInput(text) {
  const trimmed = text.trim();
  if (!trimmed) return '';
  if (looksLikeJSON(trimmed)) return highlightJSON(text);
  return highlightFlow(text);
}

function looksLikeJSON(text) {
  if (text.startsWith('{') || text.startsWith('[')) return true;
  return text.split(/\r?\n/).some((line) => line.trim().startsWith('{'));
}

function highlightJSON(text) {
  const tokenRE = /"(?:\\u[\da-fA-F]{4}|\\[^u]|[^\\"])*"|-?\d+(?:\.\d+)?(?:[eE][+-]?\d+)?|\b(?:true|false|null)\b|[{}\[\],:]/g;
  let out = '';
  let last = 0;
  let match;
  while ((match = tokenRE.exec(text)) !== null) {
    out += esc(text.slice(last, match.index));
    const token = match[0];
    const next = text.slice(tokenRE.lastIndex).match(/^\s*(.)/);
    let cls = 'tok-punct';
    if (token.startsWith('"')) {
      cls = next && next[1] === ':' ? 'tok-key' : 'tok-string';
    } else if (/^-?\d/.test(token)) {
      cls = 'tok-number';
    } else if (/^(true|false|null)$/.test(token)) {
      cls = 'tok-literal';
    }
    out += `<span class="${cls}">${esc(token)}</span>`;
    last = tokenRE.lastIndex;
  }
  out += esc(text.slice(last));
  return out;
}

function highlightFlow(text) {
  return text.split(/(\r?\n)/).map((line) => {
    if (/^\s*#\s/.test(line)) return `<span class="tok-comment">${esc(line)}</span>`;
    const tokenRE = /"(?:\\.|[^"\\])*"|(?:gh:)?[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+[#!]\d+|[A-Za-z][A-Za-z0-9_.-]*[#!]\d+|[#][0-9]+|![0-9]+|@[A-Za-z0-9_.:-]+|<-|->|~>|--|\b(?:depviz|repo|board|note|task|as|depends|on|blocks|addresses|and)\b/g;
    let out = '';
    let last = 0;
    let match;
    while ((match = tokenRE.exec(line)) !== null) {
      out += esc(line.slice(last, match.index));
      const token = match[0];
      let cls = 'tok-ref';
      if (token.startsWith('"')) cls = 'tok-string';
      else if (token.startsWith('@')) cls = 'tok-tag';
      else if (['<-', '->', '~>', '--'].includes(token)) cls = 'tok-arrow';
      else if (/^(depviz|repo|board|note|task|as|depends|on|blocks|addresses|and)$/.test(token)) cls = 'tok-keyword';
      out += `<span class="${cls}">${esc(token)}</span>`;
      last = tokenRE.lastIndex;
    }
    out += esc(line.slice(last));
    return out;
  }).join('');
}

function parseInput(text) {
  const trimmed = stripMarkdownFence(text.trim());
  if (!trimmed) return emptyExport();
  if (trimmed.startsWith('{')) {
    try {
      const parsed = JSON.parse(trimmed);
      if (parsed.snapshot && parsed.brief) return normalizeExport(parsed);
      if (parsed.nodes || parsed.edges) return buildExportFromSnapshot(parsed);
      if (parsed.type || parsed.id || parsed.from || parsed.to) return buildExportFromEvents([parsed]);
      throw new Error('JSON must be a DepViz export, snapshot, or event');
    } catch (err) {
      if (!trimmed.includes('\n')) throw err;
    }
  }
  if (looksLikeFlow(trimmed)) return buildExportFromEvents(parseFlow(trimmed));
  return buildExportFromEvents(parseJSONL(trimmed));
}

function stripMarkdownFence(text) {
  const lines = text.split(/\r?\n/);
  if (lines.length >= 2 && /^```\w*\s*$/.test(lines[0].trim()) && /^```\s*$/.test(lines[lines.length - 1].trim())) {
    return lines.slice(1, -1).join('\n').trim();
  }
  return text;
}

function looksLikeFlow(text) {
  return text.split(/\r?\n/).some((line) => {
    const trimmed = line.trim();
    return /^(depviz|repo|board|note|task)\b/.test(trimmed) || /\b(depends\s+on|blocks|addresses)\b/i.test(trimmed) || /(?:^|\s)(?:gh:)?[\w.-]+\/[\w.-]+[#!]\d+\b/.test(trimmed) || /\s(?:->|<-|~>)\s/.test(trimmed);
  });
}

function parseJSONL(text) {
  return text.split(/\r?\n/).flatMap((line, index) => {
    const trimmed = line.trim();
    if (!trimmed || trimmed.startsWith('#')) return [];
    try {
      return [JSON.parse(trimmed)];
    } catch (err) {
      throw new Error(`line ${index + 1}: ${err.message}`);
    }
  });
}

function parseFlow(text) {
  const ctx = {
    defaultRepo: '',
    aliases: new Map(),
    localAliases: new Map(),
    events: [],
    nodes: new Set(),
  };
  for (const [index, rawLine] of text.split(/\r?\n/).entries()) {
    const lineNo = index + 1;
    const line = stripInlineComment(rawLine).trim();
    if (!line) continue;
    if (/^depviz\b/i.test(line)) continue;
    if (/^repo\b/i.test(line)) {
      parseFlowRepo(ctx, line, lineNo);
      continue;
    }
    if (/^board\b/i.test(line)) {
      parseFlowBoard(ctx, line, lineNo);
      continue;
    }
    if (/^note\b/i.test(line) || /^task\b/i.test(line)) {
      parseFlowLocalNode(ctx, line, lineNo);
      continue;
    }
    if (parseFlowRelation(ctx, line, lineNo)) continue;
    if (parseFlowEdge(ctx, line, lineNo)) continue;
    if (parseFlowGitHubNode(ctx, line, lineNo)) continue;
    throw new Error(`line ${lineNo}: unsupported DepViz Flow statement`);
  }
  return ctx.events;
}

function parseFlowRepo(ctx, line, lineNo) {
  const match = /^repo\s+([A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+)(?:\s+as\s+([A-Za-z][A-Za-z0-9_.-]*))?\s*$/i.exec(line);
  if (!match) throw new Error(`line ${lineNo}: expected repo owner/name [as alias]`);
  const repo = match[1];
  const alias = match[2] || '';
  if (!ctx.defaultRepo) ctx.defaultRepo = repo;
  if (alias) ctx.aliases.set(alias, repo);
  ctx.events.push({ type: 'source', id: `github:${repo}`, kind: 'github', title: repo, url: `https://github.com/${repo}` });
}

function parseFlowBoard(ctx, line, lineNo) {
  const title = readQuoted(line.replace(/^board\s+/i, ''), lineNo) || line.replace(/^board\s+/i, '').trim();
  if (!title) throw new Error(`line ${lineNo}: board needs a title`);
  ctx.events.push({ type: 'board', id: 'default', title });
}

function parseFlowLocalNode(ctx, line, lineNo) {
  const kind = /^note\b/i.test(line) ? 'note' : 'task';
  const rest = line.replace(/^(note|task)\s+/i, '');
  const match = /^([A-Za-z][A-Za-z0-9_.:-]*)(?:\s+(.*))?$/.exec(rest);
  if (!match) throw new Error(`line ${lineNo}: expected ${kind} slug "title"`);
  const slugID = match[1];
  const tail = match[2] || '';
  const title = readQuoted(tail, lineNo) || slugID;
  const id = `${kind}:${slugID}`;
  ctx.localAliases.set(slugID, id);
  upsertFlowNode(ctx, {
    type: kind === 'note' ? 'note' : 'node',
    id,
    kind,
    title,
    state: kind === 'note' ? 'local' : readState(tail) || 'open',
    source: kind === 'note' ? 'local' : 'flow',
    external_id: id,
    labels: readLabels(tail),
  });
}

function parseFlowGitHubNode(ctx, line, lineNo) {
  const refMatch = /^(\S+)(?:\s+(.*))?$/.exec(line);
  if (!refMatch) return false;
  const ref = resolveFlowRef(ctx, refMatch[1], lineNo);
  if (!ref || ref.kind === 'local') return false;
  const tail = refMatch[2] || '';
  const title = readQuoted(tail, lineNo) || ref.id;
  upsertFlowNode(ctx, {
    type: 'node',
    id: ref.id,
    kind: ref.kind,
    title,
    state: readState(tail) || 'unknown',
    source: `github:${ref.repo}`,
    external_id: `${ref.marker}${ref.number}`,
    url: githubURL(ref.repo, ref.marker, ref.number),
    labels: readLabels(tail),
    owner: readOwner(tail),
  });
  return true;
}

function parseFlowEdge(ctx, line, lineNo) {
  const match = /^(\S+)\s+(->|<-|~>)\s+(\S+)(?:\s+(.*))?$/.exec(line);
  if (!match) return false;
  const left = resolveFlowRef(ctx, match[1], lineNo);
  const arrow = match[2];
  const right = resolveFlowRef(ctx, match[3], lineNo);
  const tail = match[4] || '';
  ensureFlowRefNode(ctx, left);
  ensureFlowRefNode(ctx, right);
  if (arrow === '<-') {
    ctx.events.push({ type: 'edge', from: left.id, to: right.id, kind: 'blocked_by', authority: 'flow', evidence: { line, note: readQuoted(tail, lineNo) || tail.trim() } });
  } else {
    ctx.events.push({ type: 'edge', from: left.id, to: right.id, kind: 'blocks', authority: arrow === '~>' ? 'flow-soft' : 'flow', confidence: arrow === '~>' ? 0.5 : 1, evidence: { line, note: readQuoted(tail, lineNo) || tail.trim() } });
  }
  return true;
}

function parseFlowRelation(ctx, line, lineNo) {
  const match = /^(\S+)\s+(.+)$/.exec(line);
  if (!match) return false;
  const chunks = relationChunks(match[2]);
  if (chunks.length === 0) return false;

  const subject = resolveFlowRef(ctx, match[1], lineNo);
  ensureFlowRefNode(ctx, subject);
  for (const chunk of chunks) {
    const targets = parseFlowRefList(ctx, chunk.refs, lineNo);
    for (const target of targets) {
      ensureFlowRefNode(ctx, target);
      if (chunk.verb === 'depends_on') {
        ctx.events.push({ type: 'edge', from: target.id, to: subject.id, kind: 'blocks', authority: 'flow', evidence: { line, relation: 'depends_on' } });
        continue;
      }
      ctx.events.push({ type: 'edge', from: subject.id, to: target.id, kind: chunk.verb, authority: 'flow', evidence: { line, relation: chunk.verb } });
    }
  }
  return true;
}

function relationChunks(text) {
  const relationRE = /\b(depends\s+on|depends|blocks|addresses)\b/ig;
  const matches = Array.from(text.matchAll(relationRE));
  if (matches.length === 0 || matches[0].index !== 0) return [];
  return matches.map((match, index) => {
    const start = match.index + match[0].length;
    const end = index + 1 < matches.length ? matches[index + 1].index : text.length;
    const refs = text.slice(start, end).trim().replace(/\band\s*$/i, '').trim();
    return { verb: normalizeRelationVerb(match[0]), refs };
  });
}

function normalizeRelationVerb(verb) {
  const normalized = verb.toLowerCase().replace(/\s+/g, ' ').trim();
  if (normalized === 'depends' || normalized === 'depends on') return 'depends_on';
  return normalized;
}

function parseFlowRefList(ctx, text, lineNo) {
  const refs = text
    .replace(/\band\b/gi, ',')
    .split(',')
    .map((part) => part.trim().replace(/[.;]$/, ''))
    .filter(Boolean)
    .map((token) => resolveFlowRef(ctx, token, lineNo));
  if (refs.length === 0) throw new Error(`line ${lineNo}: relation needs at least one ref`);
  return refs;
}

function resolveFlowRef(ctx, token, lineNo) {
  if (/^(note|task):[A-Za-z0-9_.:-]+$/.test(token)) return { id: token, kind: 'local' };
  if (ctx.localAliases.has(token)) return { id: ctx.localAliases.get(token), kind: 'local' };
  const local = /^([A-Za-z][A-Za-z0-9_.:-]*)$/.exec(token);
  if (local && !ctx.aliases.has(local[1])) return { id: `task:${local[1]}`, kind: 'local' };
  const canonical = /^gh:([A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+)([#!])(\d+)$/.exec(token);
  if (canonical) return flowGitHubRef(canonical[1], canonical[2], canonical[3]);
  const repoRef = /^([A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+)([#!])(\d+)$/.exec(token);
  if (repoRef) return flowGitHubRef(repoRef[1], repoRef[2], repoRef[3]);
  const aliasRef = /^([A-Za-z][A-Za-z0-9_.-]*)([#!])(\d+)$/.exec(token);
  if (aliasRef && ctx.aliases.has(aliasRef[1])) return flowGitHubRef(ctx.aliases.get(aliasRef[1]), aliasRef[2], aliasRef[3]);
  const shorthand = /^([#!])(\d+)$/.exec(token);
  if (shorthand) {
    if (!ctx.defaultRepo) throw new Error(`line ${lineNo}: ${token} needs a default repo`);
    return flowGitHubRef(ctx.defaultRepo, shorthand[1], shorthand[2]);
  }
  throw new Error(`line ${lineNo}: cannot resolve ref ${token}`);
}

function flowGitHubRef(repo, marker, number) {
  return {
    id: `gh:${repo}${marker}${number}`,
    kind: marker === '!' ? 'pr' : 'issue',
    repo,
    marker,
    number,
  };
}

function ensureFlowRefNode(ctx, ref) {
  if (ctx.nodes.has(ref.id)) return;
  if (ref.kind === 'local') {
    const [kind, slugID] = ref.id.split(':', 2);
    upsertFlowNode(ctx, { type: kind === 'note' ? 'note' : 'node', id: ref.id, kind, title: slugID, state: kind === 'note' ? 'local' : 'open', source: kind === 'note' ? 'local' : 'flow', external_id: ref.id });
    return;
  }
  upsertFlowNode(ctx, {
    type: 'node',
    id: ref.id,
    kind: ref.kind,
    title: ref.id,
    state: 'unknown',
    source: `github:${ref.repo}`,
    external_id: `${ref.marker}${ref.number}`,
    url: githubURL(ref.repo, ref.marker, ref.number),
  });
}

function upsertFlowNode(ctx, event) {
  if (ctx.nodes.has(event.id)) {
    const index = ctx.events.findIndex((item) => item.id === event.id && (item.type === 'node' || item.type === 'note'));
    if (index >= 0) ctx.events[index] = { ...ctx.events[index], ...event };
    return;
  }
  ctx.nodes.add(event.id);
  ctx.events.push(event);
}

function githubURL(repo, marker, number) {
  return `https://github.com/${repo}/${marker === '!' ? 'pull' : 'issues'}/${number}`;
}

function stripInlineComment(line) {
  let quoted = false;
  for (let i = 0; i < line.length; i++) {
    const char = line[i];
    if (char === '"' && line[i - 1] !== '\\') quoted = !quoted;
    if (!quoted && char === '/' && line[i + 1] === '/') return line.slice(0, i);
    if (!quoted && char === '#' && /\s/.test(line[i + 1] || '') && (i === 0 || /\s/.test(line[i - 1]))) return line.slice(0, i);
  }
  return line;
}

function readQuoted(text, lineNo) {
  const match = /"((?:\\.|[^"\\])*)"/.exec(text);
  if (!match) return '';
  try {
    return JSON.parse(`"${match[1]}"`);
  } catch (err) {
    throw new Error(`line ${lineNo}: invalid quoted string`);
  }
}

function readState(text) {
  const match = /\[([A-Za-z0-9_.:-]+)\]/.exec(text);
  return match ? match[1] : '';
}

function readLabels(text) {
  return Array.from(text.matchAll(/@([A-Za-z0-9_.:-]+)/g), (match) => match[1]);
}

function readOwner(text) {
  const match = /(?:^|\s)\+([A-Za-z0-9_.-]+)/.exec(text);
  return match ? match[1] : '';
}

function buildExportFromEvents(events) {
  const board = {
    id: 'default',
    name: 'Default',
    description: '',
    scope_query: '',
    parent_board_id: '',
    config_json: '{}',
  };
  const nodes = new Map();
  const edges = [];

  for (const event of events) {
    const type = event.type || 'node';
    const boardID = event.board || board.id;
    if (type === 'source' || type === 'depviz.source.v1') continue;
    if (type === 'board' || type === 'depviz.board.v1') {
      board.id = event.id || board.id;
      board.name = event.title || event.name || board.name;
      board.description = event.description || board.description;
      continue;
    }
    if (type === 'edge' || type === 'depviz.edge.v1') {
      const edge = normalizeEdge({
        id: event.id || stableEdgeID(boardID, event.from, event.to, event.kind || 'blocked_by'),
        from_id: event.from,
        to_id: event.to,
        kind: event.kind || 'blocked_by',
        scope_board_id: boardID,
        confidence: event.confidence || 1,
        authority: event.authority || 'event',
        evidence_json: JSON.stringify({ event }),
        observed_at: event.observed_at || '',
      });
      if (edge.from_id && edge.to_id) {
        edges.push(edge);
        ensureNode(nodes, edge.from_id);
        ensureNode(nodes, edge.to_id);
      }
      continue;
    }
    const isNote = type === 'note' || type === 'depviz.note.v1' || event.kind === 'note';
    const id = event.id || (isNote ? `note:${slug(event.title || 'untitled')}` : '');
    if (!id) throw new Error('node event is missing id');
    const node = normalizeNode({
      id,
      kind: isNote ? 'note' : event.kind || 'task',
      title: event.title || id,
      state: isNote ? 'local' : event.state || 'open',
      owner: event.owner || '',
      data_json: JSON.stringify(event),
      updated_at: event.updated_at || '',
      board_role: event.role || (isNote ? 'note' : 'card'),
      local_state: event.local_state || '',
      url: event.url || '',
      source_id: event.source || (isNote ? 'local' : 'events'),
      external_id: event.external_id || id,
    });
    nodes.set(node.id, node);
  }

  return buildExportFromSnapshot({
    board,
    nodes: Array.from(nodes.values()),
    edges,
  });
}

function buildExportFromSnapshot(snapshot) {
  const normalized = {
    snapshot: {
      board: normalizeBoard(snapshot.board || {}),
      nodes: (snapshot.nodes || []).map(normalizeNode),
      edges: (snapshot.edges || []).map(normalizeEdge),
    },
    brief: null,
  };
  normalized.brief = buildBrief(normalized.snapshot);
  return normalized;
}

function normalizeExport(payload) {
  const out = {
    snapshot: {
      board: normalizeBoard(payload.snapshot.board || {}),
      nodes: (payload.snapshot.nodes || []).map(normalizeNode),
      edges: (payload.snapshot.edges || []).map(normalizeEdge),
    },
    brief: payload.brief || null,
  };
  if (!out.brief || !out.brief.counts) out.brief = buildBrief(out.snapshot);
  return out;
}

function normalizeBoard(board) {
  return {
    id: board.id || board.ID || 'default',
    name: board.name || board.Name || 'Default',
    description: board.description || board.Description || '',
    scope_query: board.scope_query || board.ScopeQuery || '',
    parent_board_id: board.parent_board_id || board.ParentBoardID || '',
    config_json: board.config_json || board.ConfigJSON || '{}',
    updated_at: board.updated_at || board.UpdatedAt || '',
  };
}

function normalizeNode(node) {
  const id = node.id || node.ID || '';
  const kind = node.kind || node.Kind || (id.startsWith('note:') ? 'note' : 'task');
  return {
    id,
    kind,
    title: node.title || node.Title || id,
    state: node.state || node.State || (kind === 'note' ? 'local' : 'open'),
    owner: node.owner || node.Owner || '',
    data_json: node.data_json || node.DataJSON || '{}',
    updated_at: node.updated_at || node.UpdatedAt || '',
    board_role: node.board_role || node.BoardRole || '',
    local_state: node.local_state || node.LocalState || '',
    url: node.url || node.URL || '',
    source_id: node.source_id || node.SourceID || '',
    external_id: node.external_id || node.ExternalID || '',
  };
}

function normalizeEdge(edge) {
  return {
    id: edge.id || edge.ID || '',
    from_id: edge.from_id || edge.FromID || edge.from || '',
    to_id: edge.to_id || edge.ToID || edge.to || '',
    kind: edge.kind || edge.Kind || 'blocked_by',
    scope_board_id: edge.scope_board_id || edge.ScopeBoardID || 'default',
    confidence: edge.confidence || edge.Confidence || 1,
    authority: edge.authority || edge.Authority || 'local',
    evidence_json: edge.evidence_json || edge.EvidenceJSON || '{}',
    observed_at: edge.observed_at || edge.ObservedAt || '',
  };
}

function ensureNode(nodes, id) {
  if (!id || nodes.has(id)) return;
  nodes.set(id, placeholderNode(id));
}

function placeholderNode(id) {
  const gh = /^gh:([^#!]+)([#!])([0-9]+)$/.exec(id);
  if (gh) {
    const repo = gh[1];
    const marker = gh[2];
    const number = gh[3];
    const isPR = marker === '!';
    return normalizeNode({
      id,
      kind: isPR ? 'pr' : 'issue',
      title: id,
      state: 'open',
      data_json: '{"placeholder":true}',
      url: `https://github.com/${repo}/${isPR ? 'pull' : 'issues'}/${number}`,
      source_id: `github:${repo}`,
      external_id: `${marker}${number}`,
    });
  }
  return normalizeNode({
    id,
    kind: id.startsWith('note:') ? 'note' : 'task',
    title: id.startsWith('note:') ? id.slice(5) : id,
    state: id.startsWith('note:') ? 'local' : 'open',
    data_json: '{"placeholder":true}',
    source_id: id.startsWith('note:') ? 'local' : '',
    external_id: id,
  });
}

function buildBrief(snapshot) {
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const blockersByNode = new Map();
  const blockedByNode = new Map();

  for (const edge of snapshot.edges) {
    const [blocked, blocker] = edgeBlockedAndBlocker(edge);
    if (!nodes.has(blocked) || !nodes.has(blocker)) continue;
    mapSet(blockersByNode, blocked, blocker);
    mapSet(blockedByNode, blocker, blocked);
  }

  const ready = [];
  const blockers = [];
  const localOnly = [];
  const stale = [];
  let blocked = 0;
  const cutoff = Date.now() - 30 * 24 * 60 * 60 * 1000;

  for (const node of snapshot.nodes) {
    if (isClosed(node)) continue;
    const active = activeBlockers(node.id, nodes, blockersByNode);
    if (active.length === 0 && !isPlaceholder(node)) {
      ready.push({
        id: node.id,
        title: node.title,
        kind: node.kind,
        state: node.state,
        url: node.url,
        reason: readyReason(node, blockedByNode.get(node.id)),
        impact: activeBlockedCount(node.id, nodes, blockedByNode),
      });
    } else {
      blocked++;
    }
    if (isLocal(node)) {
      localOnly.push({ id: node.id, title: node.title, kind: node.kind, state: node.state, reason: 'local-only planning card' });
    }
    if (isPlaceholder(node) && !isLocal(node)) {
      stale.push({ id: node.id, title: node.title, kind: node.kind, state: node.state, url: node.url, reason: 'placeholder external ref; sync a wider scope' });
      continue;
    }
    const updated = Date.parse(node.updated_at || '');
    if (!isLocal(node) && Number.isFinite(updated) && updated < cutoff) {
      stale.push({ id: node.id, title: node.title, kind: node.kind, state: node.state, url: node.url, reason: 'not updated in 30+ days' });
    }
  }

  for (const blockerID of blockedByNode.keys()) {
    const node = nodes.get(blockerID);
    if (!node || isClosed(node)) continue;
    const impact = activeBlockedCount(blockerID, nodes, blockedByNode);
    if (impact === 0) continue;
    blockers.push({
      id: node.id,
      title: node.title,
      kind: node.kind,
      state: node.state,
      url: node.url,
      impact,
      reason: `blocks ${impact} active card${impact === 1 ? '' : 's'}`,
    });
  }

  sortItems(ready);
  sortItems(blockers);
  sortItems(localOnly);
  sortItems(stale);

  return {
    board_name: snapshot.board.name || 'Default',
    next_move: ready[0] || null,
    ready: ready.slice(0, 12),
    blockers: blockers.slice(0, 12),
    local_only: localOnly.slice(0, 12),
    stale: stale.slice(0, 12),
    counts: {
      nodes: snapshot.nodes.length,
      edges: snapshot.edges.length,
      ready: ready.length,
      blocked,
      local_only: localOnly.length,
      stale: stale.length,
    },
  };
}

function render() {
  const { snapshot, brief } = state.data;
  const nodes = visibleNodes(snapshot.nodes);
  dom.boardTitle.textContent = snapshot.board.name || 'Default';
  dom.boardMeta.textContent = `${snapshot.nodes.length} nodes - ${snapshot.edges.length} edges`;
  renderStats(brief.counts || {});
  dom.brief.classList.toggle('hidden', state.view !== 'brief');
  dom.graph.classList.toggle('hidden', state.view !== 'graph');
  dom.table.classList.toggle('hidden', state.view !== 'table');
  if (state.view === 'brief') renderBrief(brief);
  if (state.view === 'graph') renderGraph(snapshot, nodes);
  if (state.view === 'table') renderTable(nodes);
}

function renderStats(counts) {
  const values = [
    ['Nodes', counts.nodes || 0],
    ['Ready', counts.ready || 0],
    ['Blocked', counts.blocked || 0],
    ['Local', counts.local_only || 0],
    ['Stale', counts.stale || 0],
  ];
  dom.stats.innerHTML = values.map(([label, value]) => `<div class="stat"><strong>${value}</strong>${label}</div>`).join('');
}

function renderBrief(brief) {
  dom.brief.innerHTML = `<div class="briefGrid">
    ${briefSection('Next move', brief.next_move ? [brief.next_move] : [], true)}
    ${briefSection('Ready now', brief.ready || [], false)}
    ${briefSection('Blocking most work', brief.blockers || [], false)}
    ${briefSection('Local-only', brief.local_only || [], false)}
    ${briefSection('Stale external state', brief.stale || [], false)}
  </div>`;
}

function briefSection(title, items, wide) {
  const body = items.length ? items.map(renderItem).join('') : '<div class="reason">none</div>';
  return `<section class="briefSection ${wide ? 'wide' : ''}"><h3>${esc(title)}</h3>${body}</section>`;
}

function renderItem(item) {
  const id = esc(item.id || item.ID || '');
  const title = esc(item.title || item.Title || '');
  const url = item.url || item.URL || '';
  const reason = esc(item.reason || item.Reason || '');
  const label = url ? `<a href="${esc(url)}">${id}</a>` : id;
  return `<div class="item"><strong>${label} ${title}</strong><div class="reason">${reason}</div></div>`;
}

function renderGraph(snapshot, nodes) {
  const visible = new Set(nodes.map((node) => node.id));
  const positions = new Map();
  const cols = 4;
  const xGap = 252;
  const yGap = 130;
  nodes.forEach((node, index) => {
    positions.set(node.id, {
      x: 26 + (index % cols) * xGap,
      y: 30 + Math.floor(index / cols) * yGap,
    });
  });
  const height = Math.max(620, 100 + Math.ceil(nodes.length / cols) * yGap);
  const width = Math.max(900, 70 + Math.min(cols, Math.max(nodes.length, 1)) * xGap);
  let html = `<div class="graphInner" style="width:${width}px;min-height:${height}px">
    <svg class="edgeLayer" width="${width}" height="${height}" aria-hidden="true">
      <defs><marker id="arrow" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0,0 L0,6 L7,3 z" fill="#98a2b3"></path></marker></defs>`;
  for (const edge of snapshot.edges) {
    if (!visible.has(edge.from_id) || !visible.has(edge.to_id)) continue;
    const from = positions.get(edge.from_id);
    const to = positions.get(edge.to_id);
    if (!from || !to) continue;
    html += `<line x1="${from.x + 218}" y1="${from.y + 39}" x2="${to.x}" y2="${to.y + 39}" stroke="#98a2b3" stroke-width="1.5" marker-end="url(#arrow)"></line>`;
  }
  html += '</svg>';
  for (const node of nodes) {
    const pos = positions.get(node.id);
    const klass = ['nodeCard', isLocal(node) ? 'note' : '', isClosed(node) ? 'closed' : '', isBlocked(node.id, snapshot) ? 'blocked' : 'ready'].filter(Boolean).join(' ');
    const id = node.url ? `<a href="${esc(node.url)}">${esc(node.id)}</a>` : esc(node.id);
    html += `<article class="${klass}" style="transform:translate(${pos.x}px, ${pos.y}px)">
      <div class="nodeId">${id}</div>
      <div class="nodeTitle">${esc(node.title)}</div>
      <div class="nodeState">${esc(node.kind)} - ${esc(node.state)}</div>
    </article>`;
  }
  html += '</div>';
  dom.graphCanvas.innerHTML = html;
}

function renderTable(nodes) {
  const rows = nodes.map((node) => {
    const id = node.url ? `<a href="${esc(node.url)}">${esc(node.id)}</a>` : esc(node.id);
    return `<tr><td>${id}</td><td>${esc(node.title)}</td><td>${esc(node.kind)}</td><td>${esc(node.state)}</td><td>${esc(labels(node).join(', '))}</td></tr>`;
  }).join('');
  dom.table.innerHTML = `<table><thead><tr><th>ID</th><th>Title</th><th>Kind</th><th>State</th><th>Labels</th></tr></thead><tbody>${rows}</tbody></table>`;
}

function visibleNodes(nodes) {
  return nodes.filter((node) => {
    if (!state.showClosed && isClosed(node)) return false;
    if (!state.showLocal && isLocal(node)) return false;
    if (!state.showExternal && !isLocal(node)) return false;
    const hay = [node.id, node.title, node.state, node.kind, labels(node).join(' ')].join(' ').toLowerCase();
    return !state.filter || hay.includes(state.filter);
  });
}

function shareLink() {
  const encoded = encodeBase64URL(dom.input.value);
  history.replaceState(null, '', `#data=${encoded}`);
  navigator.clipboard?.writeText(location.href);
  dom.status.textContent = 'share link copied';
}

function exportJSON() {
  const blob = new Blob([`${JSON.stringify(state.data, null, 2)}\n`], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const link = document.createElement('a');
  link.href = url;
  link.download = 'depviz-live.json';
  link.click();
  URL.revokeObjectURL(url);
}

function readHash() {
  const match = /^#data=(.+)$/.exec(location.hash);
  if (!match) return '';
  try {
    return decodeBase64URL(match[1]);
  } catch {
    return '';
  }
}

function encodeBase64URL(text) {
  const bytes = new TextEncoder().encode(text);
  let binary = '';
  for (const byte of bytes) binary += String.fromCharCode(byte);
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/g, '');
}

function decodeBase64URL(text) {
  const padded = text.replace(/-/g, '+').replace(/_/g, '/') + '==='.slice((text.length + 3) % 4);
  const binary = atob(padded);
  return new TextDecoder().decode(Uint8Array.from(binary, (char) => char.charCodeAt(0)));
}

function countLines(text) {
  if (!text) return 0;
  return text.split(/\r?\n/).length;
}

function labels(node) {
  try {
    return JSON.parse(node.data_json || '{}').labels || [];
  } catch {
    return [];
  }
}

function isClosed(node) {
  return ['closed', 'done', 'merged', 'cancelled', 'canceled', 'resolved'].includes(String(node.state || '').toLowerCase());
}

function isLocal(node) {
  return node.kind === 'note' || String(node.id || '').startsWith('note:');
}

function isPlaceholder(node) {
  try {
    return Boolean(JSON.parse(node.data_json || '{}').placeholder);
  } catch {
    return false;
  }
}

function isBlocked(nodeID, snapshot) {
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  return snapshot.edges.some((edge) => {
    const [blocked, blocker] = edgeBlockedAndBlocker(edge);
    const blockerNode = nodes.get(blocker);
    return blocked === nodeID && blockerNode && !isClosed(blockerNode);
  });
}

function edgeBlockedAndBlocker(edge) {
  const kind = String(edge.kind || '').toLowerCase().trim();
  if (['addresses', 'mentions', 'relates_to'].includes(kind)) return ['', ''];
  if (['blocked_by', 'depends_on', 'depends', 'after'].includes(kind)) return [edge.from_id, edge.to_id];
  if (['blocks', 'unblocks', 'precedes'].includes(kind)) return [edge.to_id, edge.from_id];
  return [edge.from_id, edge.to_id];
}

function activeBlockers(nodeID, nodes, blockersByNode) {
  return Array.from(blockersByNode.get(nodeID) || []).filter((id) => {
    const node = nodes.get(id);
    return node && !isClosed(node);
  }).sort();
}

function activeBlockedCount(nodeID, nodes, blockedByNode) {
  return Array.from(blockedByNode.get(nodeID) || []).filter((id) => {
    const node = nodes.get(id);
    return node && !isClosed(node);
  }).length;
}

function readyReason(node, blocked) {
  const impact = blocked ? blocked.size : 0;
  if (isLocal(node) && impact > 0) return `local note, unlocks ${impact} card${impact === 1 ? '' : 's'}`;
  if (isLocal(node)) return 'local note with no active blockers';
  if (impact > 0) return `no active blockers, unlocks ${impact} card${impact === 1 ? '' : 's'}`;
  return 'no active blockers';
}

function mapSet(map, key, value) {
  if (!map.has(key)) map.set(key, new Set());
  map.get(key).add(value);
}

function sortItems(items) {
  items.sort((a, b) => {
    if ((a.impact || 0) !== (b.impact || 0)) return (b.impact || 0) - (a.impact || 0);
    return String(a.id).localeCompare(String(b.id));
  });
}

function stableEdgeID(board, from, to, kind) {
  return `edge:${slug([board, from, to, kind].join('-'))}`;
}

function slug(text) {
  return String(text || '').toLowerCase().trim().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '').slice(0, 64) || 'untitled';
}

function esc(value) {
  return String(value || '').replace(/[&<>"']/g, (char) => ({
    '&': '&amp;',
    '<': '&lt;',
    '>': '&gt;',
    '"': '&quot;',
    "'": '&#39;',
  })[char]);
}

boot();
