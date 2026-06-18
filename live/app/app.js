const assetVersion = 'v4.1.11-dev';
const sampleURL = `./sample.depviz?v=${assetVersion}`;
const githubTokenStorageKey = 'depviz.githubToken';
const githubFineGrainedTokenURL = 'https://github.com/settings/personal-access-tokens/new';
const views = new Set(['brief', 'graph', 'table']);

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
  suggestions: document.getElementById('suggestionPanel'),
  edgeInspector: document.getElementById('edgeInspector'),
  brief: document.getElementById('briefView'),
  graph: document.getElementById('graphView'),
  graphCanvas: document.getElementById('graphCanvas'),
  table: document.getElementById('tableView'),
  githubToken: document.getElementById('githubTokenInput'),
  connectGithub: document.getElementById('connectGithubBtn'),
  pasteGithubToken: document.getElementById('pasteGithubTokenBtn'),
  forgetGithubToken: document.getElementById('forgetGithubTokenBtn'),
  githubAuthState: document.getElementById('githubAuthState'),
  hydrateGithub: document.getElementById('hydrateGithubBtn'),
  showExternal: document.getElementById('showExternal'),
  showLocal: document.getElementById('showLocal'),
  showClosed: document.getElementById('showClosed'),
};

const state = {
  view: 'brief',
  filter: '',
  showExternal: true,
  showLocal: true,
  showClosed: true,
  githubRefresh: [],
  githubFailures: [],
  selectedEdgeID: '',
  dismissedSuggestionIDs: new Set(),
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
  dom.githubToken.value = sessionStorage.getItem(githubTokenStorageKey) || '';
  refreshGitHubAuthUI();
  wireEvents();
  setView(readURLView(), { persist: false, renderNow: false });
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
      setView(btn.dataset.view, { persist: true, renderNow: true });
    });
  }
  document.getElementById('sampleBtn').addEventListener('click', loadSample);
  document.getElementById('shareBtn').addEventListener('click', shareLink);
  document.getElementById('exportBtn').addEventListener('click', exportJSON);
  document.getElementById('fileInput').addEventListener('change', readFile);
  dom.connectGithub.addEventListener('click', connectGitHub);
  dom.pasteGithubToken.addEventListener('click', pasteGitHubToken);
  dom.forgetGithubToken.addEventListener('click', forgetGitHubToken);
  dom.githubToken.addEventListener('input', () => {
    persistGitHubToken();
    refreshGitHubAuthUI();
  });
  dom.hydrateGithub.addEventListener('click', hydrateGitHub);
  dom.suggestions.addEventListener('click', handleSuggestionClick);
  dom.edgeInspector.addEventListener('click', handleEdgeInspectorClick);
  dom.graphCanvas.addEventListener('click', handleGraphClick);
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
  state.githubRefresh = [];
  state.githubFailures = [];
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

function persistGitHubToken() {
  const token = githubToken();
  if (token) sessionStorage.setItem(githubTokenStorageKey, token);
  else sessionStorage.removeItem(githubTokenStorageKey);
}

function githubToken() {
  return dom.githubToken.value.trim();
}

function setGitHubToken(token) {
  dom.githubToken.value = token.trim();
  persistGitHubToken();
  refreshGitHubAuthUI();
}

function refreshGitHubAuthUI() {
  const connected = Boolean(githubToken());
  dom.githubAuthState.textContent = connected ? 'GitHub: token' : 'GitHub: public';
  dom.forgetGithubToken.disabled = !connected;
}

function connectGitHub() {
  const refs = githubRefsForSnapshot(state.data.snapshot);
  const url = buildGitHubTokenURL(refs);
  const popup = window.open(url, '_blank');
  if (popup) popup.opener = null;
  const owners = uniqueGitHubOwners(refs);
  dom.status.textContent = owners.length > 1
    ? `GitHub token page opened; choose access for ${owners.length} owners`
    : 'GitHub token page opened';
}

async function pasteGitHubToken() {
  if (!navigator.clipboard || !navigator.clipboard.readText) {
    dom.status.textContent = 'clipboard unavailable; use token fallback';
    dom.githubToken.focus();
    return;
  }
  try {
    const token = extractGitHubToken(await navigator.clipboard.readText());
    if (!token) {
      dom.status.textContent = 'clipboard has no GitHub token';
      dom.githubToken.focus();
      return;
    }
    setGitHubToken(token);
    dom.status.textContent = 'GitHub token ready for this tab';
  } catch (err) {
    dom.status.textContent = 'clipboard blocked; use token fallback';
    dom.githubToken.focus();
  }
}

function forgetGitHubToken() {
  setGitHubToken('');
  dom.status.textContent = 'GitHub token forgotten';
}

function buildGitHubTokenURL(refs) {
  const repos = [...new Set(refs.map((ref) => ref.repo))].filter(Boolean);
  const owners = uniqueGitHubOwners(refs);
  const description = repos.length > 0
    ? `Read issues and pull requests for ${repos.slice(0, 6).join(', ')} from DepViz Live.`
    : 'Read issues and pull requests from DepViz Live.';
  const params = new URLSearchParams({
    name: 'DepViz Live',
    description,
    expires_in: '30',
    metadata: 'read',
    issues: 'read',
    pull_requests: 'read',
    checks: 'read',
    statuses: 'read',
  });
  if (owners.length === 1) params.set('target_name', owners[0]);
  return `${githubFineGrainedTokenURL}?${params.toString()}`;
}

function uniqueGitHubOwners(refs) {
  return [...new Set(refs.map((ref) => ref.repo.split('/')[0]).filter(Boolean))];
}

function extractGitHubToken(text) {
  const trimmed = String(text || '').trim();
  const prefixed = trimmed.match(/\b(?:github_pat_[A-Za-z0-9_]+|gh[pousr]_[A-Za-z0-9_]+)\b/);
  if (prefixed) return prefixed[0];
  if (/^[A-Za-z0-9_]{20,}$/.test(trimmed)) return trimmed;
  return '';
}

async function hydrateGitHub() {
  const refs = githubRefsForSnapshot(state.data.snapshot);
  if (refs.length === 0) {
    dom.status.textContent = 'no GitHub refs';
    return;
  }

  dom.hydrateGithub.disabled = true;
  dom.status.textContent = `refreshing ${refs.length} GitHub refs`;
  dom.error.textContent = '';
  try {
    const token = githubToken();
    const updates = [];
    const failures = [];
    for (const ref of refs) {
      try {
        updates.push(await fetchGitHubNode(ref, token));
      } catch (err) {
        failures.push({ id: ref.id, message: err.message });
      }
    }
    state.githubFailures = failures;
    if (updates.length > 0) {
      state.data = mergeHydratedNodes(state.data, updates);
      state.githubRefresh = githubRefreshItems(updates);
    }
    if (updates.length > 0 || failures.length > 0) {
      render();
    }
    const statusParts = [failures.length > 0
      ? `refreshed ${updates.length}/${refs.length} GitHub refs`
      : `refreshed ${updates.length} GitHub refs`];
    const closedHidden = !state.showClosed ? updates.filter(isClosed).length : 0;
    const publicFallbacks = updates.filter((node) => nodeData(node).auth_fallback).length;
    if (closedHidden > 0) statusParts.push(`${closedHidden} closed in refresh summary`);
    if (publicFallbacks > 0) statusParts.push(`${publicFallbacks} via public fallback`);
    dom.status.textContent = statusParts.join('; ');
    dom.error.textContent = failures.slice(0, 2).map((item) => `${item.id}: ${item.message}`).join('  ');
  } finally {
    dom.hydrateGithub.disabled = false;
  }
}

function githubRefsForSnapshot(snapshot) {
  const seen = new Set();
  return snapshot.nodes.flatMap((node) => {
    const ref = parseGitHubNodeID(node.id);
    if (!ref || seen.has(ref.id)) return [];
    seen.add(ref.id);
    return [ref];
  });
}

function parseGitHubNodeID(id) {
  const match = /^gh:([^#!]+)([#!])([0-9]+)$/.exec(id || '');
  if (!match) return null;
  return {
    id,
    repo: match[1],
    marker: match[2],
    number: match[3],
  };
}

async function fetchGitHubNode(ref, token) {
  const issueResult = await fetchGitHubJSON(ref, token, `/repos/${ref.repo}/issues/${ref.number}`);
  if (!issueResult.ok) throw new Error(githubFetchError(issueResult.res, token));
  const issue = issueResult.data;
  const isPR = Boolean(issue.pull_request);
  const marker = ref.marker === '!' || isPR ? '!' : '#';
  let authFallback = issueResult.authFallback;
  let pr = null;
  let reviews = [];
  let checkRuns = null;
  let status = null;
  const metadataErrors = [];

  if (isPR) {
    const prResult = await fetchOptionalGitHubJSON(ref, token, `/repos/${ref.repo}/pulls/${ref.number}`);
    authFallback = authFallback || prResult.authFallback;
    pr = prResult.data;
    if (prResult.error) metadataErrors.push(`pr:${prResult.error}`);

    const reviewsResult = await fetchOptionalGitHubJSON(ref, token, `/repos/${ref.repo}/pulls/${ref.number}/reviews`);
    authFallback = authFallback || reviewsResult.authFallback;
    reviews = Array.isArray(reviewsResult.data) ? reviewsResult.data : [];
    if (reviewsResult.error) metadataErrors.push(`review:${reviewsResult.error}`);

    const headSHA = pr?.head?.sha || '';
    if (headSHA) {
      const checkRunsResult = await fetchOptionalGitHubJSON(ref, token, `/repos/${ref.repo}/commits/${headSHA}/check-runs?per_page=100`);
      authFallback = authFallback || checkRunsResult.authFallback;
      checkRuns = checkRunsResult.data;
      if (checkRunsResult.error) metadataErrors.push(`checks:${checkRunsResult.error}`);

      const statusResult = await fetchOptionalGitHubJSON(ref, token, `/repos/${ref.repo}/commits/${headSHA}/status`);
      authFallback = authFallback || statusResult.authFallback;
      status = statusResult.data;
      if (statusResult.error) metadataErrors.push(`status:${statusResult.error}`);
    }
  }

  const lifecycle = githubLifecycle(issue, pr);
  const review = summarizeGitHubReviews(reviews, pr);
  const ci = summarizeGitHubCI(checkRuns, status);
  return normalizeNode({
    id: ref.id,
    kind: isPR ? 'pr' : 'issue',
    title: issue.title || ref.id,
    state: lifecycle.state,
    owner: issue.assignee?.login || issue.user?.login || '',
    data_json: JSON.stringify({
      hydrated: true,
      auth_fallback: authFallback,
      lifecycle,
      review,
      ci,
      metadata_errors: metadataErrors,
      labels: (issue.labels || []).map((label) => label.name || label),
      number: issue.number,
      repo: ref.repo,
      source: 'github-browser',
      head_sha: pr?.head?.sha || '',
      base_ref: pr?.base?.ref || '',
      head_ref: pr?.head?.ref || '',
    }),
    updated_at: issue.updated_at || '',
    board_role: 'card',
    url: issue.html_url || githubURL(ref.repo, marker, ref.number),
    source_id: `github:${ref.repo}`,
    external_id: `${marker}${ref.number}`,
  });
}

async function fetchOptionalGitHubJSON(ref, token, path) {
  const result = await fetchGitHubJSON(ref, token, path);
  if (result.ok) return result;
  return { ...result, data: null, error: githubFetchError(result.res, token) };
}

async function fetchGitHubJSON(ref, token, path) {
  let authFallback = false;
  let res = await githubFetch(path, token);
  if (!res.ok && token && [401, 403, 404].includes(res.status)) {
    const publicRes = await githubFetch(path, '');
    if (publicRes.ok) {
      res = publicRes;
      authFallback = true;
    }
  }
  if (!res.ok) return { ok: false, res, data: null, authFallback };
  return { ok: true, res, data: await res.json(), authFallback };
}

async function githubFetch(path, token) {
  const headers = {
    Accept: 'application/vnd.github+json',
    'X-GitHub-Api-Version': '2022-11-28',
  };
  if (token) headers.Authorization = `Bearer ${token}`;
  return fetch(`https://api.github.com${path}`, { headers });
}

function githubFetchError(res, token) {
  if (res.status === 401) return token ? '401; token rejected' : '401; connect GitHub';
  if (res.status === 403) return token
    ? '403; token lacks access or rate-limited'
    : '403; connect GitHub for private refs or higher limits';
  if (res.status === 404) return token
    ? '404; missing or token lacks repo access'
    : '404; missing/private; connect GitHub if private';
  return `${res.status} ${res.statusText}`;
}

function mergeHydratedNodes(data, updates) {
  const hydrated = new Map(updates.map((node) => [node.id, node]));
  const snapshot = {
    ...data.snapshot,
    nodes: data.snapshot.nodes.map((node) => hydrated.get(node.id) || node),
    edges: data.snapshot.edges,
  };
  return buildExportFromSnapshot(snapshot);
}

function githubLifecycle(issue, pr) {
  if (pr?.merged_at || pr?.merged) {
    return { state: 'merged', phase: 'done', merged: true, draft: false };
  }
  if (pr?.draft) {
    return { state: 'draft', phase: 'review', merged: false, draft: true };
  }
  const state = issue.state || pr?.state || 'unknown';
  const phase = state === 'open' ? 'active' : 'done';
  return { state, phase, merged: false, draft: false };
}

function summarizeGitHubReviews(reviews, pr) {
  const requested = [
    ...(pr?.requested_reviewers || []).map((reviewer) => reviewer.login).filter(Boolean),
    ...(pr?.requested_teams || []).map((team) => team.slug || team.name).filter(Boolean),
  ];
  const latestByUser = new Map();
  for (const review of reviews || []) {
    const user = review.user?.login || '';
    const stateName = String(review.state || '').toUpperCase();
    if (!user || !stateName || stateName === 'PENDING') continue;
    latestByUser.set(user, stateName);
  }
  const latest = Array.from(latestByUser.values());
  const approvals = latest.filter((stateName) => stateName === 'APPROVED').length;
  const changesRequested = latest.filter((stateName) => stateName === 'CHANGES_REQUESTED').length;
  const comments = latest.filter((stateName) => stateName === 'COMMENTED').length;
  let stateName = 'none';
  if (changesRequested > 0) stateName = 'changes_requested';
  else if (approvals > 0) stateName = 'approved';
  else if (requested.length > 0) stateName = 'requested';
  else if (comments > 0) stateName = 'commented';
  return {
    state: stateName,
    approvals,
    changes_requested: changesRequested,
    comments,
    requested,
    total: reviews?.length || 0,
  };
}

function summarizeGitHubCI(checkRunsPayload, statusPayload) {
  const runs = checkRunsPayload?.check_runs || [];
  const statusState = Number(statusPayload?.total_count || 0) > 0 ? String(statusPayload?.state || '').toLowerCase() : '';
  const failedConclusions = new Set(['failure', 'cancelled', 'timed_out', 'action_required', 'startup_failure']);
  const okConclusions = new Set(['success', 'neutral', 'skipped']);
  let failed = 0;
  let pending = 0;
  let passed = 0;
  for (const run of runs) {
    const status = String(run.status || '').toLowerCase();
    const conclusion = String(run.conclusion || '').toLowerCase();
    if (failedConclusions.has(conclusion)) failed++;
    else if (status !== 'completed' || !conclusion) pending++;
    else if (okConclusions.has(conclusion)) passed++;
  }
  if (['failure', 'error'].includes(statusState)) failed++;
  if (statusState === 'pending') pending++;

  let stateName = 'none';
  if (failed > 0) stateName = 'failure';
  else if (pending > 0) stateName = 'pending';
  else if (runs.length > 0 || statusState === 'success') stateName = 'success';

  return {
    state: stateName,
    total: runs.length,
    passed,
    failed,
    pending,
    status_state: statusState || 'none',
  };
}

function githubRefreshItems(nodes) {
  return nodes.map((node) => briefItem(node, {
    reason: `${node.kind} ${node.state}${isClosed(node) && !state.showClosed ? '; closed hidden by filter' : ''}`,
  })).slice(0, 12);
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
    if (/^\s*(#\s|\/\/)/.test(line)) return `<span class="tok-comment">${esc(line)}</span>`;
    const tokenRE = /"(?:\\.|[^"\\])*"|(?:gh:)?[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+[#!]\d+|[A-Za-z][A-Za-z0-9_.-]*[#!]\d+|[#][0-9]+|![0-9]+|@[A-Za-z0-9_.:-]+|<-|->|~>|--|\b(?:depviz|repo|board|note|task|as|depends|on|blocks|addresses|mentions|relates|to|closes|fixes|resolves|and)\b/g;
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
      else if (/^(depviz|repo|board|note|task|as|depends|on|blocks|addresses|mentions|relates|to|closes|fixes|resolves|and)$/.test(token)) cls = 'tok-keyword';
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
    return /^(depviz|repo|board|note|task)\b/.test(trimmed) || /\b(depends\s+on|blocks|addresses|mentions|relates\s+to|relates|closes|fixes|resolves)\b/i.test(trimmed) || /(?:^|\s)(?:gh:)?[\w.-]+\/[\w.-]+[#!]\d+\b/.test(trimmed) || /\s(?:->|<-|~>)\s/.test(trimmed);
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
  const relationRE = /\b(depends\s+on|depends|blocks|addresses|mentions|relates\s+to|relates|closes|fixes|resolves)\b/ig;
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
  if (['fixes', 'resolves'].includes(normalized)) return 'closes';
  if (normalized === 'relates' || normalized === 'relates to') return 'relates_to';
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
      ready.push(briefItem(node, {
        reason: readyReason(node, blockedByNode.get(node.id)),
        impact: activeBlockedCount(node.id, nodes, blockedByNode),
      }));
    } else if (active.length > 0) {
      blocked++;
    }
    if (isLocal(node)) {
      localOnly.push(briefItem(node, { reason: 'local-only planning card' }));
    }
    if (isPlaceholder(node) && !isLocal(node)) {
      stale.push(briefItem(node, { reason: 'placeholder external ref; refresh GitHub or sync/export a wider scope' }));
      continue;
    }
    const updated = Date.parse(node.updated_at || '');
    if (!isLocal(node) && Number.isFinite(updated) && updated < cutoff) {
      stale.push(briefItem(node, { reason: 'not updated in 30+ days' }));
    }
  }

  for (const blockerID of blockedByNode.keys()) {
    const node = nodes.get(blockerID);
    if (!node || isClosed(node)) continue;
    const impact = activeBlockedCount(blockerID, nodes, blockedByNode);
    if (impact === 0) continue;
    blockers.push(briefItem(node, {
      impact,
      reason: `blocks ${impact} active card${impact === 1 ? '' : 's'}`,
    }));
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

function briefItem(node, extra = {}) {
  return {
    id: node.id,
    title: node.title,
    kind: node.kind,
    state: node.state,
    url: node.url,
    badges: nodeBadges(node),
    ...extra,
  };
}

function render() {
  const { snapshot, brief } = state.data;
  const nodes = visibleNodes(snapshot.nodes);
  dom.boardTitle.textContent = snapshot.board.name || 'Default';
  dom.boardMeta.textContent = `${snapshot.nodes.length} nodes - ${snapshot.edges.length} edges`;
  renderStats(brief.counts || {}, snapshot);
  renderSuggestions(snapshot);
  renderEdgeInspector(snapshot);
  dom.brief.classList.toggle('hidden', state.view !== 'brief');
  dom.graph.classList.toggle('hidden', state.view !== 'graph');
  dom.table.classList.toggle('hidden', state.view !== 'table');
  if (state.view === 'brief') renderBrief(brief);
  if (state.view === 'graph') renderGraph(snapshot, nodes);
  if (state.view === 'table') renderTable(nodes);
}

function renderStats(counts, snapshot) {
  const values = [
    ['Nodes', counts.nodes || 0],
    ['Suggested', suggestedEdges(snapshot).length],
    ['Ready', counts.ready || 0],
    ['Blocked', counts.blocked || 0],
    ['Local', counts.local_only || 0],
    ['Stale', counts.stale || 0],
  ];
  dom.stats.innerHTML = values.map(([label, value]) => `<div class="stat"><strong>${value}</strong>${label}</div>`).join('');
}

function renderBrief(brief) {
  const githubRefresh = state.githubRefresh.length
    ? briefSection('GitHub refresh', state.githubRefresh, false)
    : '';
  const githubDiagnostics = githubDiagnosticItems(state.data.snapshot);
  dom.brief.innerHTML = `<div class="briefGrid">
    ${githubRefresh}
    ${githubDiagnostics.length ? briefSection('GitHub diagnostics', githubDiagnostics, false) : ''}
    ${briefSection('Next move', brief.next_move ? [brief.next_move] : [], true)}
    ${briefSection('Ready now', brief.ready || [], false)}
    ${briefSection('Blocking most work', brief.blockers || [], false)}
    ${briefSection('Local-only', brief.local_only || [], false)}
    ${briefSection('Stale external state', brief.stale || [], false)}
  </div>`;
}

function githubDiagnosticItems(snapshot) {
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const items = [];
  for (const failure of state.githubFailures) {
    const node = nodes.get(failure.id) || placeholderNode(failure.id);
    items.push(briefItem(node, { reason: `refresh failed: ${failure.message}` }));
  }
  for (const node of snapshot.nodes) {
    if (isLocal(node)) continue;
    const data = nodeData(node);
    if (isUnhydratedExternalRef(node)) {
      items.push(briefItem(node, { reason: 'unhydrated title/state; refresh GitHub or sync/export a wider scope' }));
      continue;
    }
    if (Array.isArray(data.metadata_errors) && data.metadata_errors.length > 0) {
      items.push(briefItem(node, { reason: `partial GitHub metadata: ${data.metadata_errors.slice(0, 2).join(', ')}` }));
      continue;
    }
    if (data.auth_fallback) {
      items.push(briefItem(node, { reason: 'token lacked scope; refreshed through public GitHub fallback' }));
    }
  }
  dedupeBriefItems(items);
  sortItems(items);
  return items.slice(0, 12);
}

function isUnhydratedExternalRef(node) {
  const data = nodeData(node);
  if (data.hydrated) return false;
  if (!parseGitHubNodeID(node.id)) return isPlaceholder(node);
  return isPlaceholder(node) || node.title === node.id || String(node.state || '').toLowerCase() === 'unknown';
}

function dedupeBriefItems(items) {
  const seen = new Set();
  for (let index = items.length - 1; index >= 0; index--) {
    const item = items[index];
    const key = `${item.id}\x00${item.reason}`;
    if (seen.has(key)) items.splice(index, 1);
    else seen.add(key);
  }
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
  const badges = badgesHTML(item.badges || plainBadges(item));
  const klass = ['item', isClosed({ state: item.state || item.State }) ? 'closedItem' : ''].filter(Boolean).join(' ');
  return `<div class="${klass}"><div class="itemHead"><strong>${label} ${title}</strong>${badges}</div><div class="reason">${reason}</div></div>`;
}

function renderGraph(snapshot, nodes) {
  const visible = new Set(nodes.map((node) => node.id));
  const selectedEdge = edgeByID(state.selectedEdgeID);
  const selectedEndpoints = selectedEdge ? new Set([selectedEdge.from_id, selectedEdge.to_id]) : new Set();
  const layout = graphLayout(snapshot, nodes);
  const positions = layout.positions;
  let html = `<div class="graphInner" style="width:${layout.width}px;min-height:${layout.height}px">
    <svg class="edgeLayer" width="${layout.width}" height="${layout.height}" aria-hidden="true">
      <defs>
        <marker id="arrowHard" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0,0 L0,6 L7,3 z" fill="#667085"></path></marker>
        <marker id="arrowSoft" markerWidth="8" markerHeight="8" refX="6" refY="3" orient="auto"><path d="M0,0 L0,6 L7,3 z" fill="#b8c0cc"></path></marker>
        <marker id="arrowSelected" markerWidth="9" markerHeight="9" refX="7" refY="3.5" orient="auto"><path d="M0,0 L0,7 L8,3.5 z" fill="#dc2626"></path></marker>
      </defs>`;
  for (const edge of snapshot.edges) {
    if (!visible.has(edge.from_id) || !visible.has(edge.to_id)) continue;
    const from = positions.get(edge.from_id);
    const to = positions.get(edge.to_id);
    if (!from || !to) continue;
    const soft = isSoftEdge(edge);
    const edgeID = edgeSelectionID(edge);
    const selected = edgeID === state.selectedEdgeID;
    const kind = esc(edge.kind || 'edge');
    const authority = esc(edge.authority || '');
    const line = graphEdgeLine(from, to);
    html += `<line class="graphEdgeHit" data-edge-id="${esc(edgeID)}" x1="${line.x1}" y1="${line.y1}" x2="${line.x2}" y2="${line.y2}"></line>`;
    html += `<line class="${edgeClasses(edge)}${selected ? ' selectedEdge' : ''}" data-edge-id="${esc(edgeID)}" x1="${line.x1}" y1="${line.y1}" x2="${line.x2}" y2="${line.y2}" marker-end="url(#${selected ? 'arrowSelected' : (soft ? 'arrowSoft' : 'arrowHard')})"><title>${kind}${authority ? ` - ${authority}` : ''}</title></line>`;
  }
  html += '</svg>';
  for (const node of nodes) {
    const pos = positions.get(node.id);
    const klass = ['nodeCard', ...nodeCardClasses(node), selectedEndpoints.has(node.id) ? 'selectedEndpoint' : '', isBlocked(node.id, snapshot) ? 'blocked' : 'ready'].filter(Boolean).join(' ');
    const id = node.url ? `<a href="${esc(node.url)}">${esc(node.id)}</a>` : esc(node.id);
    html += `<article class="${klass}" data-node-id="${esc(node.id)}" style="transform:translate(${pos.x}px, ${pos.y}px)">
      <div class="nodeId">${id}</div>
      <div class="nodeTitle">${esc(node.title)}</div>
      ${badgesHTML(nodeBadges(node))}
    </article>`;
  }
  html += '</div>';
  dom.graphCanvas.innerHTML = html;
}

function handleGraphClick(event) {
  const target = event.target.closest?.('[data-edge-id]');
  if (!target || !dom.graphCanvas.contains(target)) return;
  const edgeID = target.dataset.edgeId || '';
  if (!edgeByID(edgeID)) return;
  state.selectedEdgeID = edgeID;
  render();
  dom.status.textContent = 'edge selected';
}

function graphLayout(snapshot, nodes) {
  const cardWidth = 218;
  const cardHeight = 88;
  const xGap = 282;
  const yGap = 126;
  const padX = 26;
  const padY = 30;
  const visible = new Set(nodes.map((node) => node.id));
  const layoutEdges = graphLayoutEdges(snapshot, visible);
  const connected = new Set(layoutEdges.flatMap((edge) => [edge.from, edge.to]));
  const connectedNodes = nodes.filter((node) => connected.has(node.id)).sort(graphNodeSort);
  const isolatedNodes = nodes.filter((node) => !connected.has(node.id)).sort(graphNodeSort);
  const positions = new Map();

  if (connectedNodes.length === 0) {
    placeNodeGrid(isolatedNodes, positions, { x: padX, y: padY, cols: graphGridColumns(isolatedNodes.length), xGap, yGap });
    return graphLayoutSize(positions, cardWidth, cardHeight, padX, padY);
  }

  const ranks = graphRanks(connectedNodes, layoutEdges);
  const columns = new Map();
  for (const node of connectedNodes) {
    const rank = ranks.get(node.id) || 0;
    if (!columns.has(rank)) columns.set(rank, []);
    columns.get(rank).push(node);
  }
  const orderedRanks = Array.from(columns.keys()).sort((a, b) => a - b);
  for (const [index, rank] of orderedRanks.entries()) {
    const column = columns.get(rank).sort((a, b) => graphDegree(layoutEdges, b.id) - graphDegree(layoutEdges, a.id) || graphNodeSort(a, b));
    placeNodeGrid(column, positions, { x: padX + index * xGap, y: padY, cols: 1, xGap, yGap });
  }

  if (isolatedNodes.length > 0) {
    const isolatedCols = graphGridColumns(isolatedNodes.length);
    const isolatedX = padX + orderedRanks.length * xGap + 38;
    placeNodeGrid(isolatedNodes, positions, { x: isolatedX, y: padY, cols: isolatedCols, xGap: 236, yGap: 102 });
  }

  return graphLayoutSize(positions, cardWidth, cardHeight, padX, padY);
}

function graphLayoutEdges(snapshot, visible) {
  const seen = new Set();
  const out = [];
  for (const edge of snapshot.edges || []) {
    if (!visible.has(edge.from_id) || !visible.has(edge.to_id)) continue;
    const relation = graphLayoutRelation(edge);
    if (!relation || relation.from === relation.to) continue;
    const key = `${relation.from}\x00${relation.to}`;
    if (seen.has(key)) continue;
    seen.add(key);
    out.push(relation);
  }
  return out;
}

function graphLayoutRelation(edge) {
  const kind = String(edge.kind || '').toLowerCase().trim();
  if (['blocked_by', 'depends_on', 'depends', 'after', 'blocks', 'unblocks', 'precedes'].includes(kind)) {
    const [blocked, blocker] = edgeBlockedAndBlockerRaw(edge);
    return { from: blocker, to: blocked };
  }
  if (isNonBlockingEdgeKind(kind)) return { from: edge.from_id, to: edge.to_id };
  return { from: edge.from_id, to: edge.to_id };
}

function graphRanks(nodes, edges) {
  const nodeIDs = new Set(nodes.map((node) => node.id));
  const incoming = new Map();
  for (const node of nodes) incoming.set(node.id, []);
  for (const edge of edges) {
    if (!nodeIDs.has(edge.from) || !nodeIDs.has(edge.to)) continue;
    incoming.get(edge.to).push(edge.from);
  }
  const memo = new Map();
  const visiting = new Set();
  const rankOf = (id) => {
    if (memo.has(id)) return memo.get(id);
    if (visiting.has(id)) return 0;
    visiting.add(id);
    const rank = Math.min(10, Math.max(0, ...Array.from(incoming.get(id) || [], (parent) => rankOf(parent) + 1)));
    visiting.delete(id);
    memo.set(id, rank);
    return rank;
  };
  for (const node of nodes) rankOf(node.id);
  return memo;
}

function placeNodeGrid(nodes, positions, opts) {
  nodes.forEach((node, index) => {
    positions.set(node.id, {
      x: opts.x + (index % opts.cols) * opts.xGap,
      y: opts.y + Math.floor(index / opts.cols) * opts.yGap,
    });
  });
}

function graphLayoutSize(positions, cardWidth, cardHeight, padX, padY) {
  const points = Array.from(positions.values());
  const maxX = Math.max(0, ...points.map((point) => point.x));
  const maxY = Math.max(0, ...points.map((point) => point.y));
  return {
    positions,
    width: Math.max(900, maxX + cardWidth + padX + 34),
    height: Math.max(620, maxY + cardHeight + padY + 34),
  };
}

function graphGridColumns(count) {
  if (count <= 1) return 1;
  if (count <= 6) return 2;
  if (count <= 24) return 3;
  return 4;
}

function graphDegree(edges, id) {
  return edges.reduce((total, edge) => total + (edge.from === id || edge.to === id ? 1 : 0), 0);
}

function graphNodeSort(a, b) {
  if (isClosed(a) !== isClosed(b)) return isClosed(a) ? 1 : -1;
  if (isPlaceholder(a) !== isPlaceholder(b)) return isPlaceholder(a) ? 1 : -1;
  if (String(a.kind) !== String(b.kind)) return String(a.kind).localeCompare(String(b.kind));
  return String(a.id).localeCompare(String(b.id));
}

function graphEdgeLine(from, to) {
  const cardWidth = 218;
  const centerY = 39;
  if (from.x === to.x) {
    return { x1: from.x + cardWidth / 2, y1: from.y + centerY, x2: to.x + cardWidth / 2, y2: to.y + centerY };
  }
  if (from.x <= to.x) {
    return { x1: from.x + cardWidth, y1: from.y + centerY, x2: to.x, y2: to.y + centerY };
  }
  return { x1: from.x, y1: from.y + centerY, x2: to.x + cardWidth, y2: to.y + centerY };
}

function renderTable(nodes) {
  const rows = nodes.map((node) => {
    const id = node.url ? `<a href="${esc(node.url)}">${esc(node.id)}</a>` : esc(node.id);
    const klass = nodeCardClasses(node).filter(Boolean).join(' ');
    const labelText = labels(node).join(', ');
    return `<tr class="${klass}">
      <td><div class="tableItem"><div class="tableItemID">${id}</div><div class="tableItemTitle">${esc(node.title)}</div></div></td>
      <td>${badgesHTML(nodeBadges(node))}</td>
      <td>${esc(labelText)}</td>
    </tr>`;
  }).join('');
  const empty = '<tr><td colspan="3"><div class="reason">no visible cards</div></td></tr>';
  dom.table.innerHTML = `<table class="workTable">
    <colgroup><col class="itemCol"><col class="signalCol"><col class="labelCol"></colgroup>
    <thead><tr><th>Item</th><th>Signals</th><th>Labels</th></tr></thead>
    <tbody>${rows || empty}</tbody>
  </table>`;
}

function renderSuggestions(snapshot) {
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const suggestions = suggestedEdges(snapshot).filter((edge) => !state.dismissedSuggestionIDs.has(edgeSelectionID(edge)));
  dom.suggestions.classList.toggle('hidden', suggestions.length === 0);
  if (suggestions.length === 0) {
    dom.suggestions.innerHTML = '';
    return;
  }
  const rows = suggestions.slice(0, 8).map((edge) => renderSuggestion(edge, nodes)).join('');
  const more = suggestions.length > 8 ? `<div class="suggestionMore">${suggestions.length - 8} more hidden by the compact panel</div>` : '';
  dom.suggestions.innerHTML = `<section class="suggestionBox" aria-label="Suggested relations">
    <div class="suggestionHead">
      <strong>Suggested relations</strong>
      <span>${suggestions.length} soft edge${suggestions.length === 1 ? '' : 's'} from GitHub or low-confidence sources</span>
    </div>
    <div class="suggestionList">${rows}</div>
    ${more}
  </section>`;
}

function renderSuggestion(edge, nodes) {
  const from = nodes.get(edge.from_id) || placeholderNode(edge.from_id);
  const to = nodes.get(edge.to_id) || placeholderNode(edge.to_id);
  const edgeID = edgeSelectionID(edge);
  const selected = edgeID === state.selectedEdgeID;
  const evidence = evidenceText(edge);
  const kind = relationLabel(edge.kind);
  const confidence = confidenceLabel(edge);
  return `<article class="suggestionRow ${selected ? 'selectedSuggestion' : ''}">
    <div class="suggestionMain">
      <div class="suggestionRelation">
        <strong>${esc(shortNodeLabel(from))}</strong>
        <span>${esc(kind)}</span>
        <strong>${esc(shortNodeLabel(to))}</strong>
      </div>
      <div class="suggestionEvidence">${esc(evidence || 'no evidence line captured')}</div>
    </div>
    <div class="suggestionMeta">
      <span class="badge suggestionBadge">${esc(confidence)}</span>
      <span class="badge suggestionBadge">${esc(edge.authority || 'soft')}</span>
    </div>
    <div class="suggestionActions">
      <button type="button" data-suggestion-action="focus" data-edge-id="${esc(edgeID)}">Focus</button>
      <button type="button" class="primaryAction" data-suggestion-action="promote" data-edge-id="${esc(edgeID)}">Promote</button>
      <button type="button" data-suggestion-action="dismiss" data-edge-id="${esc(edgeID)}">Hide</button>
    </div>
  </article>`;
}

function renderEdgeInspector(snapshot) {
  const edge = edgeByID(state.selectedEdgeID);
  dom.edgeInspector.classList.toggle('hidden', !edge);
  if (!edge) {
    dom.edgeInspector.innerHTML = '';
    return;
  }
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const from = nodes.get(edge.from_id) || placeholderNode(edge.from_id);
  const to = nodes.get(edge.to_id) || placeholderNode(edge.to_id);
  const evidence = parseEvidence(edge);
  const evidenceLine = evidenceText(edge);
  const evidenceJSON = Object.keys(evidence).length ? JSON.stringify(evidence, null, 2) : '';
  const confidence = confidenceLabel(edge);
  const authority = edge.authority || 'local';
  const suggested = isSuggestedEdge(edge) && !hasOfficialEquivalent(snapshot, edge);
  const suggestionActions = suggested
    ? `<button type="button" class="primaryAction" data-edge-action="promote">Promote</button>
      <button type="button" data-edge-action="hide">Hide suggestion</button>`
    : '';
  dom.edgeInspector.innerHTML = `<section class="edgeBox" aria-label="Selected edge">
    <div class="edgeHead">
      <div>
        <strong>Selected edge</strong>
        <span>${esc(relationLabel(edge.kind))}</span>
      </div>
      <div class="edgeActions">
        ${suggestionActions}
        <button type="button" data-edge-action="locate">Locate in graph</button>
        <button type="button" data-edge-action="clear">Clear</button>
      </div>
    </div>
    <div class="edgeRoute">
      <div class="edgeEndpoint">
        <span>From</span>
        <strong>${esc(shortNodeLabel(from))}</strong>
      </div>
      <div class="edgeArrow">${esc(relationLabel(edge.kind))}</div>
      <div class="edgeEndpoint">
        <span>To</span>
        <strong>${esc(shortNodeLabel(to))}</strong>
      </div>
    </div>
    <div class="edgeSignals">
      <span class="badge edgeBadge">${esc(authority)}</span>
      <span class="badge edgeBadge">${esc(confidence)}</span>
      <span class="badge edgeBadge">${isSoftEdge(edge) ? 'soft' : 'official'}</span>
    </div>
    ${evidenceLine ? `<div class="edgeEvidence"><span>Evidence</span><code>${esc(evidenceLine)}</code></div>` : ''}
    ${evidenceJSON ? `<details class="edgeJSON"><summary>Raw evidence</summary><pre>${esc(evidenceJSON)}</pre></details>` : ''}
  </section>`;
}

function handleEdgeInspectorClick(event) {
  const button = event.target.closest('[data-edge-action]');
  if (!button) return;
  const edgeID = state.selectedEdgeID;
  if (button.dataset.edgeAction === 'clear') {
    state.selectedEdgeID = '';
    render();
    dom.status.textContent = 'edge selection cleared';
  }
  if (button.dataset.edgeAction === 'locate') {
    setView('graph', { persist: true, renderNow: true });
    requestAnimationFrame(scrollSelectedEdgeIntoView);
    dom.status.textContent = 'selected edge located';
  }
  if (button.dataset.edgeAction === 'promote') promoteSuggestedEdge(edgeID);
  if (button.dataset.edgeAction === 'hide') dismissSuggestedEdge(edgeID);
}

function handleSuggestionClick(event) {
  const button = event.target.closest('[data-suggestion-action]');
  if (!button) return;
  const edgeID = button.dataset.edgeId || '';
  const action = button.dataset.suggestionAction;
  if (action === 'focus') focusSuggestedEdge(edgeID);
  if (action === 'promote') promoteSuggestedEdge(edgeID);
  if (action === 'dismiss') dismissSuggestedEdge(edgeID);
}

function focusSuggestedEdge(edgeID) {
  if (!edgeByID(edgeID)) return;
  state.selectedEdgeID = edgeID;
  setView('graph', { persist: true, renderNow: true });
  requestAnimationFrame(scrollSelectedEdgeIntoView);
  dom.status.textContent = 'suggested relation focused';
}

function scrollSelectedEdgeIntoView() {
  const endpoints = dom.graphCanvas.querySelectorAll('.selectedEndpoint');
  if (endpoints.length === 0) return;
  const target = endpoints[Math.floor((endpoints.length - 1) / 2)];
  target.scrollIntoView({ block: 'center', inline: 'center' });
}

function dismissSuggestedEdge(edgeID) {
  state.dismissedSuggestionIDs.add(edgeID);
  if (state.selectedEdgeID === edgeID) state.selectedEdgeID = '';
  render();
  dom.status.textContent = 'suggested relation hidden for this session';
}

function promoteSuggestedEdge(edgeID) {
  const edge = edgeByID(edgeID);
  if (!edge) return;
  try {
    dom.input.value = promoteInputText(dom.input.value, edge);
    state.dismissedSuggestionIDs.delete(edgeID);
    update();
    dom.status.textContent = 'suggested relation promoted locally';
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'promotion failed';
  }
}

function promoteInputText(text, edge) {
  const trimmed = stripMarkdownFence(text.trim());
  if (looksLikeJSON(trimmed)) {
    try {
      return promoteJSONText(trimmed, edge);
    } catch (err) {
      if (!trimmed.includes('\n')) throw err;
    }
  }
  if (looksLikeFlow(trimmed)) return appendFlowLine(text, edgePromotionLine(edge, trimmed));
  return appendJSONLEdge(text, officialEdge(edge));
}

function promoteJSONText(text, edge) {
  const parsed = JSON.parse(text);
  if (parsed.snapshot && parsed.brief) {
    const snapshot = promoteSnapshotEdge(parsed.snapshot, edge);
    return `${JSON.stringify({ ...parsed, snapshot, brief: buildBrief(snapshot) }, null, 2)}\n`;
  }
  if (parsed.nodes || parsed.edges) {
    return `${JSON.stringify(promoteSnapshotEdge(parsed, edge), null, 2)}\n`;
  }
  if (parsed.type || parsed.id || parsed.from || parsed.to) {
    return appendJSONLEdge(text, officialEdge(edge));
  }
  throw new Error('JSON promotion needs a DepViz export, snapshot, or event stream');
}

function promoteSnapshotEdge(snapshot, edge) {
  let promoted = false;
  const edges = (snapshot.edges || []).map((item) => {
    if (item.id === edge.id || sameRelation(item, edge)) {
      promoted = true;
      return officialEdge({ ...edge, ...item });
    }
    return item;
  });
  if (!promoted) edges.push(officialEdge(edge));
  return { ...snapshot, edges };
}

function officialEdge(edge) {
  const evidence = {
    promoted_from: {
      authority: edge.authority || '',
      confidence: Number(edge.confidence || 1),
      evidence: parseEvidence(edge),
    },
  };
  return normalizeEdge({
    ...edge,
    confidence: 1,
    authority: 'local',
    evidence_json: JSON.stringify(evidence),
  });
}

function appendFlowLine(text, line) {
  const lines = text.split(/\r?\n/);
  const isFence = lines.length >= 2 && /^```\w*\s*$/.test(lines[0].trim()) && /^```\s*$/.test(lines[lines.length - 1].trim());
  if (isFence) {
    lines.splice(lines.length - 1, 0, line);
    return `${lines.join('\n')}\n`;
  }
  return `${text.trimEnd()}\n${line}\n`;
}

function appendJSONLEdge(text, edge) {
  const event = {
    type: 'edge',
    from: edge.from_id,
    to: edge.to_id,
    kind: edge.kind,
    authority: 'local',
    confidence: 1,
  };
  return `${text.trimEnd()}\n${JSON.stringify(event)}\n`;
}

function edgePromotionLine(edge, text) {
  const repo = defaultRepoFromFlow(text);
  const from = flowRefForID(edge.from_id, repo);
  const to = flowRefForID(edge.to_id, repo);
  const kind = String(edge.kind || '').toLowerCase();
  if (['blocked_by', 'depends_on', 'depends', 'after'].includes(kind)) return `${from} depends on ${to}`;
  if (['blocks', 'unblocks'].includes(kind)) return `${from} blocks ${to}`;
  if (kind === 'addresses') return `${from} addresses ${to}`;
  if (kind === 'mentions') return `${from} mentions ${to}`;
  if (kind === 'closes') return `${from} closes ${to}`;
  if (kind === 'relates_to' || kind === 'related_to') return `${from} relates to ${to}`;
  return `${from} -> ${to}`;
}

function defaultRepoFromFlow(text) {
  const match = text.match(/^\s*repo\s+([A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+)/im);
  return match ? match[1] : '';
}

function flowRefForID(id, defaultRepo) {
  const ref = parseGitHubNodeID(id);
  if (ref) {
    const marker = ref.marker;
    if (defaultRepo && ref.repo === defaultRepo) return `${marker}${ref.number}`;
    return `${ref.repo}${marker}${ref.number}`;
  }
  return id;
}

function suggestedEdges(snapshot) {
  return (snapshot.edges || [])
    .filter(isSuggestedEdge)
    .filter((edge) => !hasOfficialEquivalent(snapshot, edge))
    .sort((a, b) => Number(b.confidence || 0) - Number(a.confidence || 0) || String(a.id).localeCompare(String(b.id)));
}

function isSuggestedEdge(edge) {
  const confidence = Number(edge.confidence || 1);
  return isSoftEdge(edge) && (confidence < 1 || /inferred|soft|suggest/i.test(edge.authority || ''));
}

function hasOfficialEquivalent(snapshot, edge) {
  return (snapshot.edges || []).some((other) => sameRelation(other, edge) && !isSuggestedEdge(other));
}

function sameRelation(a, b) {
  return relationSignature(a) === relationSignature(b);
}

function relationSignature(edge) {
  const kind = String(edge.kind || '').toLowerCase();
  if (['blocked_by', 'depends_on', 'depends', 'after', 'blocks', 'unblocks', 'precedes'].includes(kind)) {
    const [blocked, blocker] = edgeBlockedAndBlockerRaw(edge);
    return `blocking:${blocked}->${blocker}`;
  }
  return `${kind}:${edge.from_id}->${edge.to_id}`;
}

function edgeBlockedAndBlockerRaw(edge) {
  const kind = String(edge.kind || '').toLowerCase().trim();
  if (['blocked_by', 'depends_on', 'depends', 'after'].includes(kind)) return [edge.from_id, edge.to_id];
  if (['blocks', 'unblocks', 'precedes'].includes(kind)) return [edge.to_id, edge.from_id];
  return [edge.from_id, edge.to_id];
}

function edgeByID(edgeID) {
  return (state.data.snapshot.edges || []).find((edge) => edgeSelectionID(edge) === edgeID);
}

function edgeSelectionID(edge) {
  return edge.id || relationSignature(edge);
}

function relationLabel(kind) {
  const normalized = String(kind || 'relates').toLowerCase();
  if (normalized === 'blocked_by') return 'depends on';
  if (normalized === 'depends_on') return 'depends on';
  if (normalized === 'relates_to' || normalized === 'related_to') return 'relates to';
  return normalized.replace(/_/g, ' ');
}

function shortNodeLabel(node) {
  const ref = parseGitHubNodeID(node.id);
  const defaultRepo = defaultRepoFromFlow(stripMarkdownFence(dom.input.value.trim()));
  const id = ref ? flowRefForID(node.id, defaultRepo) : node.id;
  const title = node.title && node.title !== node.id ? ` ${node.title}` : '';
  return `${id}${title}`.trim();
}

function confidenceLabel(edge) {
  const confidence = Math.round(Number(edge.confidence || 1) * 100);
  return `${confidence}%`;
}

function evidenceText(edge) {
  const evidence = parseEvidence(edge);
  return evidence.line || evidence.note || evidence.event?.evidence?.line || evidence.event?.evidence?.note || '';
}

function parseEvidence(edge) {
  try {
    return JSON.parse(edge.evidence_json || '{}');
  } catch {
    return {};
  }
}

function visibleNodes(nodes) {
  return nodes.filter((node) => {
    if (!state.showClosed && isClosed(node)) return false;
    if (!state.showLocal && isLocal(node)) return false;
    if (!state.showExternal && !isLocal(node)) return false;
    const hay = [node.id, node.title, node.state, node.kind, badgeText(nodeBadges(node)), labels(node).join(' ')].join(' ').toLowerCase();
    return !state.filter || hay.includes(state.filter);
  });
}

function setView(view, options = {}) {
  const next = views.has(view) ? view : 'brief';
  state.view = next;
  document.querySelectorAll('[data-view]').forEach((item) => {
    item.classList.toggle('active', item.dataset.view === next);
  });
  if (options.persist !== false) writeURLView(next);
  if (options.renderNow !== false) render();
}

function readURLView() {
  const view = new URLSearchParams(location.search).get('view') || 'brief';
  return views.has(view) ? view : 'brief';
}

function writeURLView(view) {
  const url = new URL(location.href);
  if (view === 'brief') url.searchParams.delete('view');
  else url.searchParams.set('view', view);
  history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
}

function shareLink() {
  const encoded = encodeBase64URL(dom.input.value);
  const url = new URL(location.href);
  url.hash = `data=${encoded}`;
  history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
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
  const padded = text.replace(/-/g, '+').replace(/_/g, '/') + '='.repeat((4 - (text.length % 4)) % 4);
  const binary = atob(padded);
  return new TextDecoder().decode(Uint8Array.from(binary, (char) => char.charCodeAt(0)));
}

function countLines(text) {
  if (!text) return 0;
  return text.split(/\r?\n/).length;
}

function nodeCardClasses(node) {
  const data = nodeData(node);
  const lifecycle = data.lifecycle?.state || node.state || 'unknown';
  const ci = data.ci?.state || 'none';
  const review = data.review?.state || 'none';
  return [
    isLocal(node) ? 'note' : '',
    isClosed(node) ? 'closed' : '',
    `kind-${badgeClass(node.kind || 'task')}`,
    `life-${badgeClass(lifecycle)}`,
    node.kind === 'pr' ? `ci-${badgeClass(ci)}` : '',
    node.kind === 'pr' ? `review-${badgeClass(review)}` : '',
  ];
}

function nodeBadges(node) {
  const data = nodeData(node);
  const badges = [];
  if (isLocal(node)) {
    badges.push(badge('type-local', '📝 note'));
    return badges;
  }
  badges.push(typeBadge(node));
  badges.push(lifecycleBadge(data.lifecycle?.state || node.state));
  if (node.kind === 'pr') {
    badges.push(reviewBadge(data.review || {}));
    badges.push(ciBadge(data.ci || {}));
  }
  if (data.auth_fallback) badges.push(badge('auth-public', '🌐 public'));
  return badges.filter(Boolean);
}

function plainBadges(item) {
  const kind = item.kind || item.Kind || 'task';
  const state = item.state || item.State || 'unknown';
  return [typeBadge({ kind }), lifecycleBadge(state)].filter(Boolean);
}

function typeBadge(node) {
  const kind = node.kind || 'task';
  if (kind === 'pr') return badge('type-pr', '🔀 PR');
  if (kind === 'issue') return badge('type-issue', '📌 issue');
  if (kind === 'note') return badge('type-local', '📝 note');
  return badge('type-task', '▣ task');
}

function lifecycleBadge(value) {
  const lifecycle = String(value || 'unknown').toLowerCase();
  if (lifecycle === 'merged') return badge('life-merged', '🟣 merged');
  if (lifecycle === 'draft') return badge('life-draft', '🚧 draft');
  if (lifecycle === 'open') return badge('life-open', '🟢 open');
  if (isClosed({ state: lifecycle })) return badge('life-closed', '⚫ closed');
  if (lifecycle === 'local') return badge('life-local', '📝 local');
  return badge('life-unknown', '◇ unknown');
}

function reviewBadge(review) {
  const reviewState = String(review.state || 'none').toLowerCase();
  if (reviewState === 'approved') return badge('review-approved', `✅ review${review.approvals ? ` ${review.approvals}` : ''}`);
  if (reviewState === 'changes_requested') return badge('review-changes', `✋ changes${review.changes_requested ? ` ${review.changes_requested}` : ''}`);
  if (reviewState === 'requested') return badge('review-requested', '👀 review');
  if (reviewState === 'commented') return badge('review-commented', '💬 comments');
  return badge('review-none', '👀 no review');
}

function ciBadge(ci) {
  const ciState = String(ci.state || 'none').toLowerCase();
  if (ciState === 'success') return badge('ci-success', `🟢 ci ok${ci.total ? ` ${ci.total}` : ''}`);
  if (ciState === 'failure') return badge('ci-failure', `🔴 ci fail${ci.failed ? ` ${ci.failed}` : ''}`);
  if (ciState === 'pending') return badge('ci-pending', `🟡 ci wait${ci.pending ? ` ${ci.pending}` : ''}`);
  return badge('ci-none', '⚪ no ci');
}

function badge(kind, text) {
  return { kind, text };
}

function badgesHTML(badges) {
  if (!badges || badges.length === 0) return '';
  return `<div class="badges">${badges.map((item) => `<span class="badge ${esc(item.kind)}">${esc(item.text)}</span>`).join('')}</div>`;
}

function badgeText(badges) {
  return (badges || []).map((item) => item.text || '').join(' ');
}

function badgeClass(value) {
  return String(value || 'unknown').toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-+|-+$/g, '') || 'unknown';
}

function labels(node) {
  return nodeData(node).labels || [];
}

function nodeData(node) {
  try {
    return JSON.parse(node.data_json || '{}');
  } catch {
    return {};
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
  if (isSoftEdge(edge)) return ['', ''];
  if (isNonBlockingEdgeKind(kind)) return ['', ''];
  if (['blocked_by', 'depends_on', 'depends', 'after'].includes(kind)) return [edge.from_id, edge.to_id];
  if (['blocks', 'unblocks', 'precedes'].includes(kind)) return [edge.to_id, edge.from_id];
  return [edge.from_id, edge.to_id];
}

function edgeClasses(edge) {
  const kind = String(edge.kind || 'edge').toLowerCase().replace(/[^a-z0-9_-]+/g, '-');
  return ['graphEdge', isSoftEdge(edge) ? 'softEdge' : 'hardEdge', `edge-${kind}`].join(' ');
}

function isSoftEdge(edge) {
  const kind = String(edge.kind || '').toLowerCase().trim();
  const authority = String(edge.authority || '').toLowerCase();
  const confidence = Number(edge.confidence || 1);
  return confidence < 1 || authority.includes('inferred') || authority.includes('soft') || isNonBlockingEdgeKind(kind);
}

function isNonBlockingEdgeKind(kind) {
  return ['addresses', 'mentions', 'relates_to', 'related_to', 'closes'].includes(kind);
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
