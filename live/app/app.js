const assetVersion = 'v4.1.18-dev';
const sampleURL = `./sample.depviz?v=${assetVersion}`;
const githubTokenStorageKey = 'depviz.githubToken';
const githubFineGrainedTokenURL = 'https://github.com/settings/personal-access-tokens/new';
const views = new Set(['brief', 'graph', 'table', 'kanban']);

const dom = {
  shell: document.getElementById('shell'),
  input: document.getElementById('sourceInput'),
  syntax: document.getElementById('syntaxLayer'),
  sourcePaneTitle: document.getElementById('sourcePaneTitle'),
  sourcePaneSubtitle: document.getElementById('sourcePaneSubtitle'),
  resetSource: document.getElementById('resetSourceBtn'),
  status: document.getElementById('status'),
  lineCount: document.getElementById('lineCount'),
  error: document.getElementById('errorText'),
  boardTitle: document.getElementById('boardTitle'),
  boardMeta: document.getElementById('boardMeta'),
  filter: document.getElementById('filterInput'),
  stats: document.getElementById('stats'),
  suggestions: document.getElementById('suggestionPanel'),
  edgeInspector: document.getElementById('edgeInspector'),
  graphZoomLabel: document.getElementById('graphZoomLabel'),
  brief: document.getElementById('briefView'),
  graph: document.getElementById('graphView'),
  graphFocus: document.getElementById('graphFocusPanel'),
  graphCanvas: document.getElementById('graphCanvas'),
  graphDriver: document.getElementById('graphDriverSelect'),
  graphConnectedToggle: document.getElementById('graphConnectedToggle'),
  graphUnlinkedToggle: document.getElementById('graphUnlinkedToggle'),
  table: document.getElementById('tableView'),
  itemInspector: document.getElementById('itemInspector'),
  githubToken: document.getElementById('githubTokenInput'),
  backendGithubLogin: document.getElementById('backendGithubLoginBtn'),
  backendLogout: document.getElementById('backendLogoutBtn'),
  backendAuthState: document.getElementById('backendAuthState'),
  settings: document.getElementById('settingsBtn'),
  userPanel: document.getElementById('userPanel'),
  userPanelTitle: document.getElementById('userPanelTitle'),
  userPanelMeta: document.getElementById('userPanelMeta'),
  workspacePanel: document.getElementById('workspacePanel'),
  workspaceSummary: document.getElementById('workspaceSummary'),
  boardList: document.getElementById('boardList'),
  loadGitHubPresets: document.getElementById('loadGitHubPresetsBtn'),
  githubPresetList: document.getElementById('githubPresetList'),
  createBoardForm: document.getElementById('createBoardForm'),
  newBoardName: document.getElementById('newBoardName'),
  newBoardDescription: document.getElementById('newBoardDescription'),
  addBoardItemForm: document.getElementById('addBoardItemForm'),
  addBoardLinkForm: document.getElementById('addBoardLinkForm'),
  newItemKind: document.getElementById('newItemKind'),
  newItemRef: document.getElementById('newItemRef'),
  newItemTitle: document.getElementById('newItemTitle'),
  newItemStatus: document.getElementById('newItemStatus'),
  newItemOwner: document.getElementById('newItemOwner'),
  newItemDescription: document.getElementById('newItemDescription'),
  newLinkFrom: document.getElementById('newLinkFrom'),
  newLinkKind: document.getElementById('newLinkKind'),
  newLinkTo: document.getElementById('newLinkTo'),
  syncBoard: document.getElementById('syncBoardBtn'),
  workspaceSuggestionList: document.getElementById('workspaceSuggestionList'),
  debugPanel: document.getElementById('debugPanel'),
  syncPanel: document.getElementById('syncPanel'),
  connectGithub: document.getElementById('connectGithubBtn'),
  pasteGithubToken: document.getElementById('pasteGithubTokenBtn'),
  forgetGithubToken: document.getElementById('forgetGithubTokenBtn'),
  githubAuthState: document.getElementById('githubAuthState'),
  hydrateGithub: document.getElementById('hydrateGithubBtn'),
  showExternal: document.getElementById('showExternal'),
  showLocal: document.getElementById('showLocal'),
  showClosed: document.getElementById('showClosed'),
  filterChips: document.getElementById('filterChips'),
  newItemTimeHorizon: document.getElementById('newItemTimeHorizon'),
  newItemPriority: document.getElementById('newItemPriority'),
  newItemLabels: document.getElementById('newItemLabels'),
  newLinkNotes: document.getElementById('newLinkNotes'),
  bulkImportText: document.getElementById('bulkImportText'),
  bulkImportKind: document.getElementById('bulkImportKind'),
  kanban: document.getElementById('kanbanView'),
};

const state = {
  mode: 'stateless',
  view: 'graph',
  filter: '',
  showExternal: true,
  showLocal: true,
  showClosed: true,
  githubRefresh: [],
  githubFailures: [],
  backendSession: { available: false, authenticated: false, github_oauth_configured: false, github_app_configured: false },
  userPanelOpen: false,
  workspaceTab: 'views',
  currentBoardID: 'default',
  boards: [],
  githubPresets: { repos: [], orgs: [], projects: [], loaded: false },
  lastSync: null,
  selectedEdgeID: '',
  selectedNodeID: '',
  graphDriver: 'pairs',
  graphZoom: 1,
  graphLayout: { width: 900, height: 620 },
  showGraphAllConnected: false,
  showGraphUnlinked: false,
  dismissedSuggestionIDs: new Set(),
  activeChipFilters: new Set(),
  data: emptyExport(),
  inspectorEditMode: false,
  undoStack: [],
  boardFilter: '',
  sourceBase: '',
  sourceDirty: false,
  sourceSnapshot: null,
  paletteOpen: false,
  paletteQuery: '',
  paletteSelected: 0,
  syncIndicator: 'idle',
  linkingFrom: null,
  linkingKind: 'blocked_by',
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

async function boot() {
  dom.githubToken.value = sessionStorage.getItem(githubTokenStorageKey) || '';
  refreshGitHubAuthUI();
  wireEvents();
  setView(readURLView(), { persist: false, renderNow: false });
  state.currentBoardID = readURLBoard() || state.currentBoardID;
  readURLFilters();
  const urlNode = readURLNode();
  if (urlNode) state.selectedNodeID = urlNode;
  state.graphDriver = readURLDriver();
  if (dom.graphDriver) dom.graphDriver.value = state.graphDriver;
  const hashed = readHash();
  if (hashed) {
    setMode('stateless', { renderNow: false });
    dom.input.value = hashed;
    update();
    return;
  }
  await refreshBackendSession();
  if (state.backendSession.available) {
    await setMode('stateful', { renderNow: true });
    return;
  }
  setMode('stateless', { renderNow: false });
  loadSample();
}

function wireEvents() {
  dom.input.addEventListener('input', updateSourceInput);
  dom.input.addEventListener('scroll', syncHighlightScroll);
  dom.filter.addEventListener('input', () => {
    state.filter = dom.filter.value.toLowerCase();
    writeURLFilters();
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
  for (const btn of document.querySelectorAll('[data-mode]')) {
    btn.addEventListener('click', () => {
      setMode(btn.dataset.mode, { renderNow: true });
    });
  }
  document.getElementById('sampleBtn').addEventListener('click', loadSample);
  dom.resetSource.addEventListener('click', resetStatefulSourcePreview);
  document.getElementById('shareBtn').addEventListener('click', shareLink);
  document.getElementById('exportBtn').addEventListener('click', exportJSON);
  document.getElementById('fileInput').addEventListener('change', readFile);
  dom.connectGithub.addEventListener('click', connectGitHub);
  dom.backendGithubLogin.addEventListener('click', signInWithBackendGitHub);
  dom.backendLogout.addEventListener('click', signOutBackend);
  dom.settings.addEventListener('click', toggleUserPanel);
  document.querySelectorAll('[data-workspace-tab]').forEach((btn) => {
    btn.addEventListener('click', () => setWorkspaceTab(btn.dataset.workspaceTab));
  });
  dom.loadGitHubPresets.addEventListener('click', loadGitHubPresets);
  dom.createBoardForm.addEventListener('submit', createBoard);
  dom.addBoardItemForm.addEventListener('submit', addBoardItem);
  dom.addBoardLinkForm.addEventListener('submit', addBoardLink);
  dom.syncBoard.addEventListener('click', syncCurrentBoard);
  dom.boardList.addEventListener('click', handleBoardListClick);
  document.getElementById('boardFilterInput')?.addEventListener('input', (e) => {
    state.boardFilter = e.target.value.toLowerCase();
    renderManagePanel();
  });
  dom.githubPresetList.addEventListener('click', handlePresetClick);
  dom.pasteGithubToken.addEventListener('click', pasteGitHubToken);
  dom.forgetGithubToken.addEventListener('click', forgetGitHubToken);
  dom.githubToken.addEventListener('input', () => {
    persistGitHubToken();
    refreshGitHubAuthUI();
  });
  dom.hydrateGithub.addEventListener('click', hydrateGitHub);
  dom.suggestions.addEventListener('click', handleSuggestionClick);
  dom.workspaceSuggestionList.addEventListener('click', handleSuggestionClick);
  dom.edgeInspector.addEventListener('click', handleEdgeInspectorClick);
  dom.itemInspector.addEventListener('click', handleItemInspectorClick);
  dom.brief.addEventListener('click', handleBriefClick);
  dom.table.addEventListener('click', handleNodePickClick);
  dom.graphCanvas.addEventListener('click', handleGraphClick);
  document.getElementById('graphView').addEventListener('click', handleGraphControlClick);
  dom.graphFocus.addEventListener('click', handleGraphFocusClick);
  dom.graphDriver.addEventListener('change', () => {
    state.graphDriver = dom.graphDriver.value;
    writeURLDriver(state.graphDriver);
    render();
  });
  if (dom.filterChips) {
    dom.filterChips.addEventListener('click', (e) => {
      const btn = e.target.closest('[data-chip-type]');
      if (!btn) return;
      const key = `${btn.dataset.chipType}:${btn.dataset.chipValue}`;
      if (state.activeChipFilters.has(key)) state.activeChipFilters.delete(key);
      else state.activeChipFilters.add(key);
      writeURLFilters();
      render();
    });
  }
  document.addEventListener('keydown', handleGraphKeydown);
  document.addEventListener('keydown', (e) => {
    if ((e.ctrlKey || e.metaKey) && e.key === 'k') {
      e.preventDefault();
      if (state.paletteOpen) closePalette(); else openPalette();
      return;
    }
    if (e.key === 'Escape' && state.paletteOpen) {
      closePalette();
      return;
    }
    if (e.key === 'Escape' && state.linkingFrom) {
      state.linkingFrom = null;
      render();
      return;
    }
    if ((e.ctrlKey || e.metaKey) && e.key === 'z' && !e.shiftKey) {
      e.preventDefault();
      undoLastOp();
      return;
    }
    if (inputFocused()) return;
    if (e.key === 'n') {
      e.preventDefault();
      setWorkspaceTab('actions');
      setTimeout(() => (dom.newItemTitle || dom.newItemRef).focus(), 50);
      return;
    }
    if (e.key === 'l') {
      e.preventDefault();
      setWorkspaceTab('actions');
      setTimeout(() => dom.newLinkFrom.focus(), 50);
      return;
    }
    if (e.key === '/' || e.key === 's') {
      e.preventDefault();
      if (dom.filter) dom.filter.focus();
      return;
    }
    if (e.key === 'Escape') {
      if (state.selectedNodeID) {
        state.selectedNodeID = '';
        state.inspectorEditMode = false;
        render();
      }
    }
  });
  document.getElementById('bulkImportForm')?.addEventListener('submit', handleBulkImport);
  document.getElementById('resetPreviewBtn')?.addEventListener('click', () => {
    resetStatefulSourcePreview();
  });
  document.getElementById('applySourceBtn')?.addEventListener('click', applySourcePatch);
  document.getElementById('saveCurrentViewBtn')?.addEventListener('click', () => {
    const name = prompt('Name for this saved view:');
    if (name) saveCurrentView(name.trim());
  });
  document.getElementById('commandPalette')?.addEventListener('click', (e) => {
    if (e.target.classList.contains('paletteBackdrop')) closePalette();
  });
  const paletteInput = document.getElementById('paletteInput');
  paletteInput?.addEventListener('input', (e) => {
    state.paletteQuery = e.target.value;
    state.paletteSelected = 0;
    renderPalette();
  });
  paletteInput?.addEventListener('keydown', (e) => {
    const q = (state.paletteQuery || '').toLowerCase();
    const cmds = paletteCommands().filter((c) => !q || c.label.toLowerCase().includes(q) || (c.hint || '').toLowerCase().includes(q));
    const visible = cmds.slice(0, 12);
    if (e.key === 'ArrowDown') { e.preventDefault(); state.paletteSelected = Math.min(state.paletteSelected + 1, visible.length - 1); renderPalette(); }
    if (e.key === 'ArrowUp') { e.preventDefault(); state.paletteSelected = Math.max(state.paletteSelected - 1, 0); renderPalette(); }
    if (e.key === 'Enter') { e.preventDefault(); if (visible[state.paletteSelected]) { visible[state.paletteSelected].action(); closePalette(); } }
    if (e.key === 'Escape') { e.preventDefault(); closePalette(); }
  });
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
  if (state.mode !== 'stateless') return;
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

function updateSourceInput() {
  if (state.mode === 'stateful') {
    updateStatefulSourcePreview();
    return;
  }
  update();
}

async function setMode(mode, options = {}) {
  const next = mode === 'stateful' && state.backendSession.available ? 'stateful' : 'stateless';
  state.mode = next;
  document.querySelectorAll('[data-mode]').forEach((item) => {
    item.classList.toggle('active', item.dataset.mode === next);
    if (item.dataset.mode === 'stateful') item.disabled = !state.backendSession.available;
  });
  dom.shell.classList.toggle('statefulMode', next === 'stateful');
  syncModeVisibility();
  syncSourcePaneMode();
  refreshBackendAuthUI();
  if (next === 'stateful') {
    await loadBackendBoard();
    return;
  }
  if (options.renderNow !== false) update();
}

function syncModeVisibility() {
  document.querySelectorAll('.statelessOnly').forEach((item) => {
    item.classList.toggle('hidden', state.mode !== 'stateless');
  });
  document.querySelectorAll('.statefulOnly').forEach((item) => {
    const show = state.mode === 'stateful' && state.backendSession.authenticated;
    item.classList.toggle('hidden', !show || (item === dom.userPanel && !state.userPanelOpen));
  });
  dom.settings.classList.toggle('active', state.userPanelOpen && state.mode === 'stateful' && state.backendSession.authenticated);
  renderWorkspaceTabs();
}

async function loadBackendBoard() {
  if (!state.backendSession.authenticated) {
    state.data = emptyExport();
    setStatefulSourceFromSnapshot(state.data.snapshot);
    dom.status.textContent = state.backendSession.github_oauth_configured ? 'sign in for stateful graph' : 'stateful backend needs oauth config';
    dom.error.textContent = '';
    state.userPanelOpen = false;
    renderAuthGate();
    render();
    return;
  }
  setSyncIndicator('syncing');
  dom.status.textContent = 'loading stateful graph';
  dom.error.textContent = '';
  state.githubRefresh = [];
  state.githubFailures = [];
  try {
    if (state.boards.length === 0) await refreshBoards();
    const board = encodeURIComponent(state.currentBoardID || 'default');
    const res = await fetch(`./api/export?board=${board}`, { credentials: 'same-origin' });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    state.data = normalizeExport(await res.json());
    setStatefulSourceFromSnapshot(state.data.snapshot);
    if (state.selectedNodeID && !nodeByID(state.selectedNodeID)) {
      state.selectedNodeID = '';
      writeURLNode('');
    }
    setSyncIndicator('done');
    dom.status.textContent = 'stateful backend graph';
  } catch (err) {
    setSyncIndicator('failed');
    state.data = emptyExport();
    setStatefulSourceFromSnapshot(state.data.snapshot);
    dom.error.textContent = err.message;
    dom.status.textContent = 'stateful load failed';
  }
  render();
}

function syncSourcePaneMode() {
  dom.sourcePaneTitle.textContent = state.mode === 'stateful' ? 'Board source' : 'Input';
  dom.sourcePaneSubtitle.textContent = state.mode === 'stateful'
    ? 'Editable preview of the current board. Changes update the UI locally and are not saved yet.'
    : 'DepViz Flow, JSONL, or export JSON.';
  dom.input.setAttribute('aria-label', state.mode === 'stateful'
    ? 'Editable DepViz Flow preview for this board'
    : 'DepViz Flow, JSONL, or export JSON');
  dom.resetSource.classList.toggle('hidden', state.mode !== 'stateful');
  updateSourceDirtyState();
}

function setStatefulSourceFromSnapshot(snapshot) {
	if (state.mode !== 'stateful') return;
	const source = snapshotToFlow(snapshot || emptyExport().snapshot);
	state.sourceBase = source;
	state.sourceDirty = false;
	state.sourceSnapshot = normalizeExport(buildExportFromSnapshot(snapshot || emptyExport().snapshot));
	dom.input.value = source;
	updateHighlight(source);
	dom.lineCount.textContent = `${countLines(source)} lines`;
  updateSourceDirtyState();
}

function resetStatefulSourcePreview() {
  if (state.mode !== 'stateful') return;
  dom.input.value = state.sourceBase;
  updateStatefulSourcePreview();
  dom.status.textContent = 'preview reset to server board';
}

function updateStatefulSourcePreview() {
  const text = dom.input.value;
  updateHighlight(text);
  dom.lineCount.textContent = `${countLines(text)} lines`;
  state.sourceDirty = text !== state.sourceBase;
  updateSourceDirtyState();
  try {
    const preview = parseInput(text);
    if (!preview.snapshot.board.id || preview.snapshot.board.id === 'default') {
      preview.snapshot.board.id = state.currentBoardID || 'default';
    }
    state.data = preview;
    dom.error.textContent = '';
    dom.status.textContent = state.sourceDirty ? 'stateful source preview - unsaved' : 'stateful backend graph';
    render();
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'source preview error';
  }
}

function updateSourceDirtyState() {
	dom.shell.classList.toggle('sourceDirty', state.mode === 'stateful' && state.sourceDirty);
	dom.shell.classList.toggle('sourcePreviewMode', state.mode === 'stateful');
	renderSourceDirtyIndicator();
}

async function refreshBoards() {
  if (!state.backendSession.authenticated) return;
  const res = await fetch('./api/boards', { credentials: 'same-origin' });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  const payload = await res.json();
  state.boards = sortBoards(Array.isArray(payload.boards) ? payload.boards.map(normalizeBoard) : []);
  if (!state.boards.some((board) => board.id === state.currentBoardID)) {
    state.currentBoardID = preferredBoardID(state.boards);
    writeURLBoard(state.currentBoardID);
  }
  renderManagePanel();
}

function toggleUserPanel() {
  state.userPanelOpen = !state.userPanelOpen;
  syncModeVisibility();
  renderUserPanel();
}

function setWorkspaceTab(tab) {
  state.workspaceTab = ['views', 'actions', 'presets', 'suggestions', 'debug', 'sync'].includes(tab) ? tab : 'views';
  renderWorkspaceTabs();
  if (state.workspaceTab === 'presets') renderGitHubPresets();
  if (state.workspaceTab === 'suggestions') renderWorkspaceSuggestions();
  if (state.workspaceTab === 'debug') renderDebugPanel();
  if (state.workspaceTab === 'sync') renderSyncPanel();
}

async function createBoard(event) {
  event.preventDefault();
  const name = dom.newBoardName.value.trim();
  const description = dom.newBoardDescription.value.trim();
  if (!name) {
    dom.status.textContent = 'view name required';
    dom.newBoardName.focus();
    return;
  }
  try {
    await createBoardFromPreset({ name, description, preset: 'empty' });
    dom.newBoardName.value = '';
    dom.newBoardDescription.value = '';
  } catch (err) {
    dom.status.textContent = 'view create failed';
    dom.error.textContent = err.message;
  }
}

async function createBoardFromPreset(input) {
  const res = await fetch('./api/boards', {
    method: 'POST',
    credentials: 'same-origin',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(input),
  });
  if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
  const payload = await res.json();
  const board = normalizeBoard(payload.board || {});
  state.currentBoardID = board.id || state.currentBoardID;
  writeURLBoard(state.currentBoardID);
  await refreshBoards();
  if (['repo', 'org', 'my-work'].includes(input.preset)) {
    try {
      await syncBoard(board.id || state.currentBoardID, { quiet: true });
    } catch (err) {
      dom.error.textContent = `initial sync failed: ${err.message}`;
    }
  }
  await loadBackendBoard();
  dom.status.textContent = 'view created';
  return board;
}

async function addBoardItem(event) {
  event.preventDefault();
  const ref = dom.newItemRef.value.trim();
  const title = dom.newItemTitle.value.trim();
  if (!ref && !title) {
    dom.status.textContent = 'item text required';
    dom.newItemRef.focus();
    return;
  }
  const status = dom.newItemStatus ? dom.newItemStatus.value.trim() : '';
  const owner = dom.newItemOwner ? dom.newItemOwner.value.trim() : '';
  const description = dom.newItemDescription ? dom.newItemDescription.value.trim() : '';
  const timeHorizon = dom.newItemTimeHorizon ? dom.newItemTimeHorizon.value : '';
  const priority = dom.newItemPriority ? dom.newItemPriority.value : '';
  const labelsRaw = dom.newItemLabels ? dom.newItemLabels.value.trim() : '';
  const itemLabels = labelsRaw ? labelsRaw.split(',').map((l) => l.trim()).filter(Boolean) : [];
  try {
    const res = await fetch('./api/board-items', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        board_id: state.currentBoardID || 'default',
        kind: dom.newItemKind.value,
        ref,
        title,
        status,
        owner,
        description,
        time_horizon: timeHorizon,
        priority,
        labels: itemLabels,
      }),
    });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    const addPayload = await res.json();
    const newNodeID = addPayload.node?.id || '';
    if (newNodeID) pushUndo({ type: 'add-node', nodeID: newNodeID });
    dom.newItemRef.value = '';
    dom.newItemTitle.value = '';
    if (dom.newItemStatus) dom.newItemStatus.value = '';
    if (dom.newItemOwner) dom.newItemOwner.value = '';
    if (dom.newItemDescription) dom.newItemDescription.value = '';
    if (dom.newItemTimeHorizon) dom.newItemTimeHorizon.value = '';
    if (dom.newItemPriority) dom.newItemPriority.value = '';
    if (dom.newItemLabels) dom.newItemLabels.value = '';
    await loadBackendBoard();
    await refreshBoards();
    dom.status.textContent = 'item added';
  } catch (err) {
    dom.status.textContent = 'item add failed';
    dom.error.textContent = err.message;
  }
}

async function addBoardLink(event) {
  event.preventDefault();
  const from = dom.newLinkFrom.value.trim();
  const to = dom.newLinkTo.value.trim();
  if (!from || !to) {
    dom.status.textContent = 'two item refs required';
    (from ? dom.newLinkTo : dom.newLinkFrom).focus();
    return;
  }
  try {
    const res = await fetch('./api/board-links', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        board_id: state.currentBoardID || 'default',
        from,
        to,
        kind: dom.newLinkKind.value,
        notes: dom.newLinkNotes ? dom.newLinkNotes.value.trim() : '',
      }),
    });
    if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
    dom.newLinkFrom.value = '';
    dom.newLinkTo.value = '';
    if (dom.newLinkNotes) dom.newLinkNotes.value = '';
    await loadBackendBoard();
    await refreshBoards();
    dom.status.textContent = 'link added';
  } catch (err) {
    dom.status.textContent = 'link add failed';
    dom.error.textContent = err.message;
  }
}

function handleBoardListClick(event) {
  const btn = event.target.closest('[data-board-id]');
  if (!btn) return;
  state.currentBoardID = btn.dataset.boardId || 'default';
  writeURLBoard(state.currentBoardID);
  renderManagePanel();
  loadBackendBoard();
}

async function loadGitHubPresets() {
  dom.status.textContent = 'loading github presets';
  dom.githubPresetList.innerHTML = '<div class="emptyState">Loading GitHub repositories, orgs, and projects...</div>';
  try {
    const [reposRes, orgsRes, projectsRes] = await Promise.all([
      fetch('./api/github/repos', { credentials: 'same-origin' }),
      fetch('./api/github/orgs', { credentials: 'same-origin' }),
      fetch('./api/github/projects', { credentials: 'same-origin' }),
    ]);
    if (!reposRes.ok) throw new Error(`repos ${reposRes.status}`);
    if (!orgsRes.ok) throw new Error(`orgs ${orgsRes.status}`);
    if (!projectsRes.ok) throw new Error(`projects ${projectsRes.status}`);
    const reposPayload = await reposRes.json();
    const orgsPayload = await orgsRes.json();
    const projectsPayload = await projectsRes.json();
    state.githubPresets = {
      repos: Array.isArray(reposPayload.repos) ? reposPayload.repos : [],
      orgs: Array.isArray(orgsPayload.orgs) ? orgsPayload.orgs : [],
      projects: Array.isArray(projectsPayload.projects) ? projectsPayload.projects : [],
      loaded: true,
    };
    renderGitHubPresets();
    dom.status.textContent = 'github presets loaded';
  } catch (err) {
    dom.githubPresetList.innerHTML = `<div class="emptyState">Could not load GitHub presets: ${esc(err.message)}</div>`;
    dom.status.textContent = 'github presets failed';
  }
}

function renderGitHubPresets() {
  const builtinItems = presetButton({
    type: 'my-work',
    id: 'my-work',
    label: 'My Work',
    meta: 'assigned, authored, mentioned',
  });
  const repos = state.githubPresets.repos.slice(0, 24);
  const orgs = state.githubPresets.orgs.slice(0, 24);
  const projects = state.githubPresets.projects.slice(0, 24);
  if (!repos.length && !orgs.length && !projects.length) {
    dom.githubPresetList.innerHTML = `${builtinItems}<div class="emptyState">No repos, orgs, or projects loaded yet</div>`;
    return;
  }
  const repoItems = repos.map((repo) => presetButton({
    type: 'repo',
    id: repo.full_name || repo.FullName || '',
    label: repo.full_name || repo.FullName || repo.name || repo.Name || 'repo',
    meta: repo.private ? 'private repo' : 'repo',
  })).join('');
  const orgItems = orgs.map((org) => presetButton({
    type: 'org',
    id: org.login || org.Login || '',
    label: org.login || org.Login || 'org',
    meta: 'org',
  })).join('');
  const projectItems = projects.map((project) => presetButton({
    type: 'project',
    id: project.id || project.ID || '',
    label: project.title || project.Title || 'project',
    meta: `project${project.owner || project.Owner ? ` - ${project.owner || project.Owner}` : ''}`,
  })).join('');
  dom.githubPresetList.innerHTML = `${builtinItems}${repoItems}${orgItems}${projectItems}`;
}

function presetButton(item) {
  return `<button type="button" data-preset-type="${esc(item.type)}" data-preset-id="${esc(item.id)}">
    <strong>${esc(item.label)}</strong>
    <span>${esc(item.meta)}</span>
  </button>`;
}

function handlePresetClick(event) {
  const btn = event.target.closest('[data-preset-type]');
  if (!btn) return;
  const type = btn.dataset.presetType;
  const id = btn.dataset.presetId;
  if (!id) return;
  const name = type === 'repo' ? id : type === 'my-work' ? 'My Work' : `${id} org`;
  const owner = type === 'org' ? id : type === 'repo' ? id.split('/')[0] : '';
  createBoardFromPreset({
    name: type === 'project' ? btn.querySelector('strong')?.textContent || 'GitHub project' : name,
    description: type === 'repo' ? `GitHub repo ${id}` : type === 'project' ? `GitHub project ${id}` : type === 'my-work' ? 'GitHub work involving me' : `GitHub org ${id}`,
    preset: type,
    provider: 'github',
    owner,
    repo: type === 'repo' ? id : '',
    source_id: type === 'project' ? id : '',
  }).catch((err) => {
    dom.status.textContent = 'preset view failed';
    dom.error.textContent = err.message;
  });
}

async function syncCurrentBoard() {
  await syncBoard(state.currentBoardID || 'default');
}

async function syncBoard(boardID, options = {}) {
  if (!options.quiet) {
    dom.status.textContent = 'syncing github view';
    dom.syncBoard.disabled = true;
  }
  setSyncIndicator('syncing');
  try {
    const res = await fetch('./api/board-sync', {
      method: 'POST',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ board_id: boardID, limit: 100 }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    state.lastSync = await res.json();
    if (!options.quiet) {
      await loadBackendBoard();
      await refreshBoards();
      dom.status.textContent = `synced ${state.lastSync.items || 0} GitHub items`;
    }
    setSyncIndicator('done');
    return state.lastSync;
  } catch (err) {
    setSyncIndicator('failed');
    if (!options.quiet) {
      dom.status.textContent = 'github sync failed';
      dom.error.textContent = err.message;
    }
    throw err;
  } finally {
    dom.syncBoard.disabled = false;
  }
}

async function responseErrorMessage(res) {
  let detail = '';
  try {
    const data = await res.clone().json();
    detail = data.error || data.message || '';
  } catch (_) {
    detail = (await res.text()).trim();
  }
  return detail || `${res.status} ${res.statusText}`;
}

function renderManagePanel() {
  if (!dom.boardList) return;
  renderWorkspaceSummary();
  if (!state.boards.length) {
    dom.boardList.innerHTML = '<div class="emptyState">No saved views yet</div>';
  } else {
    const filterQuery = state.boardFilter || '';
    const filteredBoards = filterQuery
      ? state.boards.filter((b) => {
          const hay = [b.name || '', b.scope_query || '', b.description || ''].join(' ').toLowerCase();
          return hay.includes(filterQuery);
        })
      : state.boards;
    const usefulBoards = filteredBoards.filter((board) => !isDraftBoard(board) || board.id === state.currentBoardID);
    const draftBoards = filteredBoards.filter((board) => isDraftBoard(board) && board.id !== state.currentBoardID);
    const boardButtons = usefulBoards.map(renderBoardListButton).join('');
    const draftList = draftBoards.length
      ? `<details class="draftGroup"><summary>Draft views <strong>${draftBoards.length}</strong></summary>${draftBoards.map(renderBoardListButton).join('')}</details>`
      : '';
    dom.boardList.innerHTML = boardButtons + draftList;
  }
  if (state.githubPresets.loaded) renderGitHubPresets();
  renderDebugPanel();
  renderAuthGate();
  renderWorkspaceSuggestions();
  if (state.mode === 'stateful' && state.backendSession.authenticated) {
    loadArchivedNodes();
    loadSavedViews();
  }
}

function renderBoardListButton(board) {
      const active = board.id === state.currentBoardID;
      const metrics = board.metrics || {};
      const draft = isDraftBoard(board);
      const scope = board.scope_query || 'local view';
      const description = board.description || scope;
      const syncError = draft ? '' : (metrics.sync_error ? `<span class="boardSyncError" title="${esc(metrics.sync_error)}">${esc(metrics.sync_error.slice(0, 60))}</span>` : '');
      return `<button class="${[active ? 'active' : '', draft ? 'draftBoard' : ''].filter(Boolean).join(' ')}" type="button" data-board-id="${esc(board.id)}">
        <span class="boardListTitle">
          <strong>${esc(board.name || board.id)}</strong>
          <span class="freshnessBadge ${syncClass(metrics)}">${esc(syncLabel(metrics))}</span>
        </span>
        <span class="boardScope">${esc(scope)}</span>
        <span class="boardDescription">${esc(description)}</span>
        <span class="boardMetrics">
          <span><strong>${esc(metrics.items || 0)}</strong> items</span>
          <span><strong>${esc(metrics.links || 0)}</strong> links</span>
          <span><strong>${esc(metrics.open || 0)}</strong> open</span>
        </span>
        ${syncError}
      </button>`;
}

function renderWorkspaceTabs() {
  document.querySelectorAll('[data-workspace-tab]').forEach((btn) => {
    btn.classList.toggle('active', btn.dataset.workspaceTab === state.workspaceTab);
  });
  document.querySelectorAll('.workspaceTab').forEach((panel) => {
    panel.classList.toggle('hidden', panel.id !== `workspace${capitalize(state.workspaceTab)}`);
  });
}

function renderWorkspaceSummary() {
  if (!dom.workspaceSummary) return;
  const board = state.data.snapshot.board || {};
  const scope = board.scope_query || 'local view';
  const counts = state.data.brief.counts || {};
  dom.workspaceSummary.textContent = `${board.name || state.currentBoardID || 'Default'} - ${scope} - ${counts.nodes || 0} items`;
}

function renderUserPanel() {
  const session = state.backendSession || {};
  const account = session.account || {};
  const provider = session.github_app_configured ? 'GitHub App' : 'DepViz';
  dom.userPanelTitle.textContent = account.login ? `${provider}: @${account.login}` : provider;
  dom.userPanelMeta.textContent = [
    session.github_app_configured ? 'App auth configured' : 'OAuth mode',
    session.github_webhook_configured ? 'webhook active' : 'webhook pending',
  ].join(' - ');
  renderOnboardingChecklist();
}

function renderOnboardingChecklist() {
  const el = document.getElementById('onboardingPanel');
  if (!el) return;
  if (!state.backendSession.authenticated) { el.classList.add('hidden'); return; }
  const hasBoards = state.boards.some((b) => !isDraftBoard(b));
  const hasSynced = state.boards.some((b) => b.metrics && b.metrics.items > 0);
  const hasLocalItem = (state.data.snapshot.nodes || []).some((n) => n.kind === 'task' || n.kind === 'note');
  const hasLinks = (state.data.snapshot.edges || []).length > 0;
  if (hasSynced && hasBoards && hasLocalItem && hasLinks) { el.classList.add('hidden'); return; }
  el.classList.remove('hidden');
  el.innerHTML = `<div class="onboardingChecklist">
    <h3>Get started</h3>
    <ol>
      <li class="checkItem ${hasBoards ? 'done' : ''}">Choose a repo or org in Sources tab</li>
      <li class="checkItem ${hasSynced ? 'done' : ''}">Sync a board</li>
      <li class="checkItem ${hasLocalItem ? 'done' : ''}">Add a local planning item</li>
      <li class="checkItem ${hasLinks ? 'done' : ''}">Inspect a dependency link</li>
    </ol>
  </div>`;
}

function renderAuthGate() {
  const el = document.getElementById('authGatePanel');
  if (!el) return;
  const session = state.backendSession || {};
  if (session.authenticated) { el.classList.add('hidden'); return; }
  el.classList.remove('hidden');
  const hasOAuth = session.github_oauth_configured;
  el.innerHTML = `<div class="authGate">
    <h2>Sign in to use live mode</h2>
    <p>DepViz stateful mode lets you manage dependency boards backed by GitHub.</p>
    ${hasOAuth
      ? `<button type="button" class="primaryAction" id="authGateSignIn">Sign in with GitHub</button>`
      : `<div class="envVars">DEPVIZ_GITHUB_CLIENT_ID<br>DEPVIZ_GITHUB_CLIENT_SECRET</div><p>Configure these env vars to enable GitHub OAuth.</p>`}
    <a href="#" data-mode="stateless">Use stateless mode instead</a>
  </div>`;
  document.getElementById('authGateSignIn')?.addEventListener('click', signInWithBackendGitHub);
  el.querySelector('[data-mode="stateless"]')?.addEventListener('click', (e) => {
    e.preventDefault();
    setMode('stateless', { renderNow: true });
  });
}

function renderDebugPanel() {
  if (!dom.debugPanel) return;
  const board = state.data.snapshot.board || {};
  const counts = state.data.brief.counts || {};
  const session = state.backendSession || {};
  const account = session.account || {};
  const selectedNode = state.selectedNodeID ? nodeByID(state.selectedNodeID) : null;
  const selectedEdge = state.selectedEdgeID ? edgeByID(state.selectedEdgeID) : null;
  const selectedNodeJSON = selectedNode ? JSON.stringify(nodeData(selectedNode), null, 2) : null;
  const selectedEdgeJSON = selectedEdge ? JSON.stringify(selectedEdge, null, 2) : null;
  dom.debugPanel.innerHTML = `<dl>
    <div><dt>Board name</dt><dd>${esc(board.name || state.currentBoardID)}</dd></div>
    <div><dt>Board id</dt><dd>${esc(state.currentBoardID)}</dd></div>
    <div><dt>Scope</dt><dd>${esc(board.scope_query || 'none')}</dd></div>
    <div><dt>Nodes</dt><dd>${esc(String(counts.nodes || 0))}</dd></div>
    <div><dt>Last sync</dt><dd>${esc(currentBoardSyncLabel())}</dd></div>
    <div><dt>OAuth</dt><dd>${session.github_oauth_configured ? 'configured' : 'not configured'}</dd></div>
    <div><dt>GitHub App</dt><dd>${session.github_app_configured ? 'configured' : 'not configured'}</dd></div>
    <div><dt>Webhook</dt><dd>${session.github_webhook_configured ? 'configured' : 'not configured'}</dd></div>
    <div><dt>User</dt><dd>${account.login ? `@${esc(account.login)}` : 'not signed in'}</dd></div>
    <div><dt>Mode</dt><dd>${esc(state.mode)}</dd></div>
  </dl>
  ${selectedNodeJSON ? `<details><summary>Selected node data <button type="button" onclick="navigator.clipboard.writeText(${JSON.stringify(selectedNodeJSON)}).catch(()=>{})">Copy JSON</button></summary><pre>${esc(selectedNodeJSON)}</pre></details>` : ''}
  ${selectedEdgeJSON ? `<details><summary>Selected edge <button type="button" onclick="navigator.clipboard.writeText(${JSON.stringify(selectedEdgeJSON)}).catch(()=>{})">Copy JSON</button></summary><pre>${esc(selectedEdgeJSON)}</pre></details>` : ''}
  <div class="debugGitHubSetup">
    <h4>GitHub App Setup</h4>
    <dl>
      <div><dt>Webhook URL <button type="button" onclick="navigator.clipboard.writeText('${esc(`${location.origin}/api/github/webhook`)}').catch(()=>{})">Copy</button></dt><dd><code>${esc(`${location.origin}/api/github/webhook`)}</code></dd></div>
      <div><dt>OAuth callback URL <button type="button" onclick="navigator.clipboard.writeText('${esc(`${location.origin}/api/auth/github/callback`)}').catch(()=>{})">Copy</button></dt><dd><code>${esc(`${location.origin}/api/auth/github/callback`)}</code></dd></div>
    </dl>
    <div class="debugChecklist">
      <div class="checkItem ${session.github_oauth_configured ? 'done' : ''}">OAuth app configured ${session.github_oauth_configured ? '✅' : '❌'}</div>
      <div class="checkItem ${session.github_app_configured ? 'done' : ''}">GitHub App configured ${session.github_app_configured ? '✅' : '❌'}</div>
      <div class="checkItem ${session.github_webhook_configured ? 'done' : ''}">Webhook receiving ${session.github_webhook_configured ? '✅' : '❌'}</div>
    </div>
  </div>`;
}

function renderSyncPanel() {
  if (!dom.syncPanel) return;
  const board = currentBoard();
  const metrics = board.metrics || {};
  const session = state.backendSession || {};
  const lastSync = state.lastSync || {};
  const syncStatus = metrics.sync_status || 'unknown';
  const syncError = metrics.sync_error || '';
  const statusClass = syncStatus === 'ok' ? 'syncOk' : syncStatus === 'running' ? 'syncRunning' : syncStatus === 'failed' ? 'syncFailed' : 'syncUnknown';
  const provider = session.github_app_configured ? 'GitHub App' : 'OAuth user';
  const webhookState = session.github_webhook_configured ? 'configured' : 'not configured';
  const items = Number(metrics.items || 0);
  const links = Number(metrics.links || 0);
  const open = Number(metrics.open || 0);
  dom.syncPanel.innerHTML = `<dl class="syncList">
    <div><dt>View</dt><dd>${esc(board.name || state.currentBoardID)}</dd></div>
    <div><dt>Scope</dt><dd>${esc(board.scope_query || 'local')}</dd></div>
    <div><dt>Status</dt><dd><span class="syncStatus ${statusClass}">${esc(syncStatus)}</span>${syncError ? ` — <em>${esc(syncError)}</em>` : ''}</dd></div>
    <div><dt>Last sync</dt><dd>${esc(currentBoardSyncLabel())}</dd></div>
    ${metrics.last_sync_at ? `<div><dt>Synced at</dt><dd>${esc(metrics.last_sync_at)}</dd></div>` : ''}
    <div><dt>Items</dt><dd>${esc(String(items))} total · ${esc(String(open))} open · ${esc(String(links))} links</dd></div>
    <div><dt>Token mode</dt><dd>${esc(lastSync.mode || provider)}</dd></div>
    <div><dt>Webhook</dt><dd>${esc(webhookState)}</dd></div>
    <div><dt>GitHub App</dt><dd>${session.github_app_configured ? 'configured' : 'not configured'}</dd></div>
  </dl>
  <div class="syncActions">
    <button type="button" id="syncFromPanelBtn">Sync now</button>
  </div>
  <div class="syncLogsSection">
    <h4>Sync history</h4>
    <div id="syncLogsList" class="syncLogsList"></div>
  </div>`;
  document.getElementById('syncFromPanelBtn')?.addEventListener('click', syncCurrentBoard);
  loadSyncLogs();
}

async function loadSyncLogs() {
  const el = document.getElementById('syncLogsList');
  if (!el) return;
  try {
    const board = encodeURIComponent(state.currentBoardID || 'default');
    const res = await fetch(`./api/board-sync-logs?board_id=${board}`, { credentials: 'same-origin' });
    if (!res.ok) return;
    const data = await res.json();
    const logs = Array.isArray(data.logs) ? data.logs : [];
    if (!logs.length) { el.innerHTML = '<div class="emptyState">No sync history yet</div>'; return; }
    el.innerHTML = logs.map((log) => `<div class="syncLogEntry ${log.status === 'ok' ? 'syncLogOk' : 'syncLogFailed'}">
      <div class="syncLogHead">
        <strong>${esc(log.status)}</strong>
        <span>${esc(log.mode || 'unknown')}</span>
        <span>${esc(log.started_at ? new Date(log.started_at).toLocaleString() : '')}</span>
      </div>
      <div class="syncLogMeta">
        ${log.items_synced ? `${log.items_synced} items · ` : ''}${log.edges_synced ? `${log.edges_synced} links · ` : ''}${log.rate_limit_remaining ? `${log.rate_limit_remaining} API calls remaining` : ''}
      </div>
      ${log.error ? `<div class="syncLogError">${esc(log.error)}</div>` : ''}
    </div>`).join('');
  } catch (_) {}
}

function currentBoard() {
  return state.boards.find((board) => board.id === state.currentBoardID) || normalizeBoard(state.data.snapshot.board || {});
}

function currentBoardSyncLabel() {
  const metrics = currentBoard().metrics || {};
  return `${syncLabel(metrics)}${metrics.sync_error ? ` - ${metrics.sync_error}` : ''}`;
}

function renderWorkspaceSuggestions() {
  if (!dom.workspaceSuggestionList) return;
  const suggestions = suggestedEdges(state.data.snapshot).slice(0, 12);
  if (!suggestions.length) {
    dom.workspaceSuggestionList.innerHTML = '<div class="emptyState">No suggestions for the current view</div>';
    return;
  }
  dom.workspaceSuggestionList.innerHTML = suggestions.map((edge) => `<div>
    <strong>${esc(edge.from_id || edge.from)} -> ${esc(edge.to_id || edge.to)}</strong>
    <span>${esc(relationLabel(edge.kind || 'related'))}</span>
    <button type="button" class="primaryAction" data-suggestion-action="promote" data-edge-id="${esc(edgeSelectionID(edge))}">Accept</button>
  </div>`).join('');
}

function capitalize(value) {
  value = String(value || '');
  return value.charAt(0).toUpperCase() + value.slice(1);
}

function freshnessLabel(value) {
  const age = ageMs(value);
  if (!Number.isFinite(age)) return 'unknown';
  const minute = 60 * 1000;
  const hour = 60 * minute;
  const day = 24 * hour;
  if (age < hour) return 'fresh';
  if (age < day) return `${Math.max(1, Math.floor(age / hour))}h`;
  if (age < 14 * day) return `${Math.max(1, Math.floor(age / day))}d`;
  return 'stale';
}

function freshnessClass(value) {
  const age = ageMs(value);
  if (!Number.isFinite(age)) return 'freshnessUnknown';
  const day = 24 * 60 * 60 * 1000;
  if (age < day) return 'freshnessFresh';
  if (age < 7 * day) return 'freshnessRecent';
  return 'freshnessStale';
}

function syncLabel(metrics = {}) {
  if (Number(metrics.items || 0) === 0) return 'draft';
  const status = String(metrics.sync_status || '').toLowerCase();
  if (status === 'running') return 'syncing';
  if (status === 'failed') return 'sync failed';
  if (status === 'never') return 'never synced';
  if (metrics.last_sync_at) return freshnessLabel(metrics.last_sync_at);
  return freshnessLabel(metrics.last_activity_at);
}

function syncClass(metrics = {}) {
  if (Number(metrics.items || 0) === 0) return 'freshnessDraft';
  const status = String(metrics.sync_status || '').toLowerCase();
  if (status === 'running') return 'freshnessRecent';
  if (status === 'failed') return 'freshnessStale';
  if (metrics.last_sync_at) return freshnessClass(metrics.last_sync_at);
  return freshnessClass(metrics.last_activity_at);
}

function sortBoards(boards) {
  return boards.slice().sort((a, b) => boardRank(a) - boardRank(b) || boardActivityScore(b) - boardActivityScore(a) || String(a.name).localeCompare(String(b.name)));
}

function boardRank(board) {
  const metrics = board.metrics || {};
  if ((metrics.items || 0) > 0 && (metrics.open || 0) > 0) return 0;
  if ((metrics.items || 0) > 0) return 1;
  if (String(metrics.sync_status || '') === 'failed') return 2;
  return 3;
}

function boardActivityScore(board) {
  return Date.parse(board.metrics?.last_activity_at || board.updated_at || '') || 0;
}

function preferredBoardID(boards) {
  const active = boards.find((board) => !isDraftBoard(board));
  return active?.id || boards[0]?.id || 'default';
}

function isDraftBoard(board) {
  return Number(board.metrics?.items || 0) === 0;
}

function ageMs(value) {
  const parsed = Date.parse(value || '');
  if (!Number.isFinite(parsed)) return Number.POSITIVE_INFINITY;
  return Math.max(0, Date.now() - parsed);
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
  refreshBackendAuthUI();
}

async function refreshBackendSession() {
  try {
    const res = await fetch('./api/session', { credentials: 'same-origin' });
    if (!res.ok) throw new Error(String(res.status));
    state.backendSession = { available: true, ...await res.json() };
  } catch {
    state.backendSession = { available: false, authenticated: false, github_oauth_configured: false, github_app_configured: false };
  }
  refreshBackendAuthUI();
  document.querySelector('[data-mode="stateful"]').disabled = !state.backendSession.available;
}

function refreshBackendAuthUI() {
  const session = state.backendSession || {};
  const showLogin = session.available && !session.authenticated;
  dom.backendGithubLogin.classList.toggle('hidden', !showLogin);
  dom.backendGithubLogin.disabled = showLogin && !session.github_oauth_configured;
  dom.backendGithubLogin.textContent = session.github_oauth_configured ? 'Sign in' : 'GitHub auth missing';
  dom.backendLogout.classList.toggle('hidden', !session.available || !session.authenticated);
  dom.backendAuthState.classList.toggle('hidden', !session.available);
  syncModeVisibility();
  const provider = session.github_app_configured ? 'GitHub App' : 'DepViz';
  if (!session.available) {
    dom.backendAuthState.textContent = '';
    renderUserPanel();
    return;
  }
  if (session.authenticated && session.account) {
    dom.backendAuthState.textContent = `${provider}: @${session.account.login || 'account'}`;
    renderUserPanel();
    return;
  }
  dom.backendAuthState.textContent = session.github_oauth_configured ? `${provider}: signed out` : 'GitHub auth: not configured';
  renderUserPanel();
}

function signInWithBackendGitHub() {
  if (!state.backendSession.github_oauth_configured) {
    dom.status.textContent = 'github auth is not configured';
    dom.error.textContent = 'Set DEPVIZ_GITHUB_CLIENT_ID and DEPVIZ_GITHUB_CLIENT_SECRET on the backend service.';
    return;
  }
  const returnTo = `${location.pathname}${location.search}${location.hash}`;
  location.href = `./api/auth/github/start?return_to=${encodeURIComponent(returnTo)}`;
}

async function signOutBackend() {
  try {
    await fetch('./api/auth/logout', { method: 'POST', credentials: 'same-origin' });
    await refreshBackendSession();
    dom.status.textContent = 'signed out';
    if (state.mode === 'stateful') await loadBackendBoard();
  } catch {
    dom.status.textContent = 'sign out failed';
  }
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
      assignees: (issue.assignees || []).map(githubPersonData),
      author: githubPersonData(issue.user),
      reviewers: review.requested || [],
      milestone: issue.milestone?.title || '',
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

function githubPersonData(person) {
  if (!person) return null;
  if (typeof person === 'string') return { login: person, avatar_url: githubAvatarURL(person) };
  const login = person.login || person.name || '';
  if (!login) return null;
  return {
    login,
    avatar_url: person.avatar_url || githubAvatarURL(login),
    html_url: person.html_url || `https://github.com/${encodeURIComponent(login)}`,
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
    const tokenRE = /"(?:\\.|[^"\\])*"|(?:gh:)?[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+[#!]\d+|[A-Za-z][A-Za-z0-9_.-]*[#!]\d+|[#][0-9]+|![0-9]+|@[A-Za-z0-9_.:-]+|<-|->|~>|--|\b(?:depviz|repo|board|note|task|strategy|initiative|bet|project|workstream|risk|decision|question|metric|as|depends|on|blocks|addresses|mentions|relates|to|closes|fixes|resolves|and)\b/g;
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
      else if (/^(depviz|repo|board|note|task|strategy|initiative|bet|project|workstream|risk|decision|question|metric|as|depends|on|blocks|addresses|mentions|relates|to|closes|fixes|resolves|and)$/.test(token)) cls = 'tok-keyword';
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

function snapshotToFlow(snapshot) {
  const board = snapshot.board || {};
  const nodes = Array.isArray(snapshot.nodes) ? snapshot.nodes : [];
  const edges = Array.isArray(snapshot.edges) ? snapshot.edges : [];
  const repos = Array.from(new Set(nodes.map((node) => parseGitHubNodeID(node.id)?.repo).filter(Boolean))).sort();
  const defaultRepo = repos[0] || '';
  const lines = [
    '// Stateful board preview. Edit to preview changes locally; persistence comes next.',
    `board ${flowQuote(board.name || board.id || 'DepViz board')}`,
  ];
  for (const repo of repos) lines.push(`repo ${repo}`);
  if (nodes.length) lines.push('');
  for (const node of [...nodes].sort(compareNodesForSource)) {
    lines.push(flowNodeLine(node, defaultRepo));
  }
  if (edges.length) lines.push('');
  for (const edge of [...edges].sort(compareEdgesForSource)) {
    const from = nodeRefForFlow(edge.from_id, defaultRepo);
    const to = nodeRefForFlow(edge.to_id, defaultRepo);
    lines.push(`${from} ${flowRelationVerb(edge.kind)} ${to}`);
  }
  return `${lines.join('\n')}\n`;
}

function flowNodeLine(node, defaultRepo) {
  const ref = parseGitHubNodeID(node.id);
  const data = nodeData(node);
  const title = node.title && node.title !== node.id ? ` ${flowQuote(node.title)}` : '';
  const status = node.state ? ` [${node.state}]` : '';
  const owner = node.owner ? ` +${flowToken(node.owner)}` : '';
  const labels = Array.isArray(data.labels) ? data.labels : [];
  const labelText = labels.map((label) => ` @${flowToken(label)}`).join('');
  if (ref) return `${nodeRefForFlow(node.id, defaultRepo)}${title}${status}${owner}${labelText}`;
  const [kind, slugID] = localFlowParts(node);
  return `${kind} ${slugID}${title}${status}${owner}${labelText}`;
}

function flowRelationVerb(kind) {
  const normalized = String(kind || 'relates_to').toLowerCase();
  if (normalized === 'blocked_by' || normalized === 'depends_on' || normalized === 'depends') return 'depends on';
  if (normalized === 'relates_to' || normalized === 'related_to') return 'relates to';
  return normalized.replace(/_/g, ' ');
}

function flowQuote(value) {
  return JSON.stringify(String(value || ''));
}

function flowToken(value) {
  return String(value || '').trim().replace(/\s+/g, '-').replace(/[^A-Za-z0-9_.:-]/g, '-');
}

function localFlowParts(node) {
  const validKinds = localFlowKinds();
  const [prefix, rest] = String(node.id || '').split(':', 2);
  const kind = validKinds.has(node.kind) ? node.kind : (validKinds.has(prefix) ? prefix : 'task');
  const slugID = flowToken(rest || slug(node.title || node.id || 'item'));
  return [kind, slugID || 'item'];
}

function nodeRefForFlow(nodeID, defaultRepo) {
  const ref = parseGitHubNodeID(nodeID);
  if (!ref) return String(nodeID || '').replace(/\s+/g, '-');
  if (ref.repo === defaultRepo) return `${ref.marker}${ref.number}`;
  return `${ref.repo}${ref.marker}${ref.number}`;
}

function compareNodesForSource(a, b) {
  if (isClosed(a) !== isClosed(b)) return isClosed(a) ? 1 : -1;
  return String(a.id || '').localeCompare(String(b.id || ''));
}

function compareEdgesForSource(a, b) {
  return relationSignature(a).localeCompare(relationSignature(b));
}

function localFlowKinds() {
  return new Set(['note', 'task', 'strategy', 'initiative', 'bet', 'project', 'workstream', 'risk', 'decision', 'question', 'metric']);
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
    return /^(depviz|repo|board|note|task|strategy|initiative|bet|project|workstream|risk|decision|question|metric)\b/.test(trimmed) || /\b(depends\s+on|blocks|addresses|mentions|relates\s+to|relates|closes|fixes|resolves)\b/i.test(trimmed) || /(?:^|\s)(?:gh:)?[\w.-]+\/[\w.-]+[#!]\d+\b/.test(trimmed) || /\s(?:->|<-|~>)\s/.test(trimmed);
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
    if (/^(note|task|strategy|initiative|bet|project|workstream|risk|decision|question|metric)\b/i.test(line)) {
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
  const kindMatch = /^(note|task|strategy|initiative|bet|project|workstream|risk|decision|question|metric)\b/i.exec(line);
  const kind = kindMatch ? kindMatch[1].toLowerCase() : 'task';
  const rest = line.replace(/^(note|task|strategy|initiative|bet|project|workstream|risk|decision|question|metric)\s+/i, '');
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
    time_horizon: readAttribute(tail, 'horizon'),
    priority: readAttribute(tail, 'priority'),
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
  if (/^(note|task|strategy|initiative|bet|project|workstream|risk|decision|question|metric):[A-Za-z0-9_.:-]+$/.test(token)) return { id: token, kind: 'local' };
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

function readAttribute(text, key) {
  const re = new RegExp(`\\b${key}:([A-Za-z0-9_.-]+)`, 'i');
  const match = re.exec(text);
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
    metrics: normalizeBoardMetrics(board.metrics || board.Metrics || {}),
  };
}

function normalizeBoardMetrics(metrics) {
  return {
    items: Number(metrics.items || metrics.Items || 0),
    links: Number(metrics.links || metrics.Links || 0),
    open: Number(metrics.open || metrics.Open || 0),
    closed: Number(metrics.closed || metrics.Closed || 0),
    local: Number(metrics.local || metrics.Local || 0),
    external: Number(metrics.external || metrics.External || 0),
    last_activity_at: metrics.last_activity_at || metrics.LastActivityAt || '',
    last_sync_at: metrics.last_sync_at || metrics.LastSyncAt || '',
    sync_status: metrics.sync_status || metrics.SyncStatus || '',
    sync_error: metrics.sync_error || metrics.SyncError || '',
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

function collectNodeChips(nodes) {
  const labelCounts = new Map();
  const assigneeCounts = new Map();
  const kindCounts = new Map();
  const statusCounts = new Map();
  const milestoneCounts = new Map();
  const repoCounts = new Map();
  const ownerCounts = new Map();
  for (const node of nodes) {
    for (const label of labels(node)) {
      labelCounts.set(label, (labelCounts.get(label) || 0) + 1);
    }
    for (const person of nodePeople(node)) {
      assigneeCounts.set(person.login, (assigneeCounts.get(person.login) || 0) + 1);
    }
    if (node.kind) {
      kindCounts.set(node.kind, (kindCounts.get(node.kind) || 0) + 1);
    }
    if (node.state) {
      statusCounts.set(node.state, (statusCounts.get(node.state) || 0) + 1);
    }
    const nd = nodeData(node);
    if (nd.milestone) {
      milestoneCounts.set(nd.milestone, (milestoneCounts.get(nd.milestone) || 0) + 1);
    }
    const gh = parseGitHubNodeID(node.id);
    if (gh && gh.repo) {
      repoCounts.set(gh.repo, (repoCounts.get(gh.repo) || 0) + 1);
    }
    if (node.owner) {
      ownerCounts.set(node.owner, (ownerCounts.get(node.owner) || 0) + 1);
    }
  }
  const chips = [];
  for (const [kind, count] of [...kindCounts.entries()].sort((a, b) => b[1] - a[1])) {
    chips.push({ type: 'kind', value: kind, label: kind, count });
  }
  for (const [status, count] of [...statusCounts.entries()].sort((a, b) => b[1] - a[1])) {
    chips.push({ type: 'status', value: status, label: status, count });
  }
  for (const [label, count] of [...labelCounts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 20)) {
    chips.push({ type: 'label', value: label, label, count });
  }
  for (const [login, count] of [...assigneeCounts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 10)) {
    chips.push({ type: 'assignee', value: login, label: `@${login}`, count });
  }
  for (const [milestone, count] of [...milestoneCounts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 5)) {
    chips.push({ type: 'milestone', value: milestone, label: `🏁 ${milestone}`, count });
  }
  for (const [repo, count] of [...repoCounts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 5)) {
    chips.push({ type: 'repo', value: repo, label: repo, count });
  }
  for (const [owner, count] of [...ownerCounts.entries()].sort((a, b) => b[1] - a[1]).slice(0, 5)) {
    chips.push({ type: 'owner', value: owner, label: `+${owner}`, count });
  }
  return chips;
}

function renderFilterChips() {
  if (!dom.filterChips) return;
  const chips = collectNodeChips(state.data.snapshot.nodes);
  if (!chips.length) { dom.filterChips.innerHTML = ''; return; }
  dom.filterChips.innerHTML = chips.map((chip) => {
    const key = `${chip.type}:${chip.value}`;
    const active = state.activeChipFilters.has(key);
    return `<button type="button" class="filterChip ${active ? 'active' : ''}" data-chip-type="${esc(chip.type)}" data-chip-value="${esc(chip.value)}" title="${esc(chip.type)}: ${esc(chip.value)}">${emojiHTML(chip.label)}${chip.count > 1 ? ` <span>${chip.count}</span>` : ''}</button>`;
  }).join('');
}

function render() {
  if (state.mode === 'stateful' && !state.backendSession.authenticated) {
    renderStatefulSignedOut();
    return;
  }
  const { snapshot, brief } = state.data;
  const nodes = visibleNodes(snapshot.nodes);
  const graphSelection = graphVisibleNodeSelection(snapshot, nodes);
  dom.shell.classList.toggle('emptyBoard', snapshot.nodes.length === 0);
  dom.boardTitle.textContent = snapshot.board.name || 'Default';
  dom.boardMeta.textContent = `${snapshot.nodes.length} nodes - ${snapshot.edges.length} edges`;
  renderFilterChips();
  renderWorkspaceSummary();
  renderStats(brief.counts || {}, snapshot);
  renderSuggestions(snapshot);
  renderGraphFocusPanel(snapshot);
  renderEdgeInspector(snapshot);
  renderItemInspector(snapshot);
  renderWorkspaceSuggestions();
  renderDebugPanel();
  dom.brief.classList.toggle('hidden', state.view !== 'brief');
  dom.graph.classList.toggle('hidden', state.view !== 'graph');
  dom.table.classList.toggle('hidden', state.view !== 'table');
  if (dom.kanban) dom.kanban.classList.toggle('hidden', state.view !== 'kanban');
  if (state.view === 'brief') renderBrief(brief);
  if (state.view === 'graph') renderGraph(snapshot, graphSelection.nodes, graphSelection.hidden);
  if (state.view === 'table') renderTable(nodes);
  if (state.view === 'kanban') renderKanbanView();
}

function renderStatefulSignedOut() {
  const session = state.backendSession || {};
  dom.shell.classList.remove('emptyBoard');
  dom.boardTitle.textContent = 'Stateful DepViz';
  dom.boardMeta.textContent = session.github_oauth_configured
    ? 'Sign in to load your persisted boards and GitHub-backed graph.'
    : 'Backend is running, but GitHub OAuth is not configured yet.';
  dom.stats.innerHTML = '';
  dom.suggestions.innerHTML = '';
  dom.graphFocus.innerHTML = '';
  dom.edgeInspector.classList.add('hidden');
  dom.itemInspector.classList.add('hidden');
  dom.brief.classList.remove('hidden');
  dom.graph.classList.add('hidden');
  dom.table.classList.add('hidden');
  if (dom.kanban) dom.kanban.classList.add('hidden');
  const loginButton = session.github_oauth_configured
    ? '<button type="button" class="primaryAction" data-auth-action="signin">Sign in with GitHub</button>'
    : '<button type="button" class="primaryAction" data-auth-action="signin" disabled>GitHub OAuth missing</button>';
  const configHint = session.github_oauth_configured
    ? 'GitHub will return here after authorization and DepViz will load your saved views.'
    : 'Set DEPVIZ_GITHUB_CLIENT_ID and DEPVIZ_GITHUB_CLIENT_SECRET on the depviz-live service, then restart it.';
  dom.brief.innerHTML = `<section class="authGate">
    <div>
      <span class="emptyBoardKicker">Stateful workspace</span>
      <h3>${session.github_oauth_configured ? 'Connect your GitHub account' : 'GitHub OAuth is not configured'}</h3>
      <p>${esc(configHint)}</p>
    </div>
    <div class="emptyBoardActions">
      ${loginButton}
      <button type="button" data-auth-action="stateless">Use stateless mode</button>
    </div>
  </section>`;
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
  dom.stats.innerHTML = values.map(([label, value]) => `<div class="stat"><span>${label}</span><strong>${value}</strong></div>`).join('');
}

function renderBrief(brief) {
  if ((brief.counts?.nodes || 0) === 0) {
    renderEmptyBoardBrief();
    return;
  }
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

function renderEmptyBoardBrief() {
  const board = state.data.snapshot.board || {};
  const usefulBoard = state.boards.find((item) => !isDraftBoard(item) && item.id !== state.currentBoardID);
  const canSync = Boolean(board.scope_query && board.scope_query !== 'local');
  const scope = board.scope_query || 'local draft';
  dom.brief.innerHTML = `<section class="emptyBoardPanel">
    <div>
      <span class="emptyBoardKicker">${esc(scope)}</span>
      <h3>${esc(board.name || 'Empty view')}</h3>
      <p>This view has no items yet. Start from a GitHub source or add a first issue, PR, task, or note.</p>
    </div>
    <div class="emptyBoardActions">
      <button type="button" class="primaryAction" data-empty-action="add-item">Add first item</button>
      <button type="button" data-empty-action="sources">Choose source</button>
      ${canSync ? '<button type="button" data-empty-action="sync">Sync source</button>' : ''}
      ${usefulBoard ? `<button type="button" data-empty-action="useful" data-board-id="${esc(usefulBoard.id)}">Open ${esc(usefulBoard.name || usefulBoard.id)}</button>` : ''}
    </div>
  </section>`;
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
  return `<section class="briefSection ${wide ? 'wide' : ''}"><h3><span>${esc(title)}</span><strong>${items.length}</strong></h3>${body}</section>`;
}

function renderItem(item) {
  const id = esc(item.id || item.ID || '');
  const title = esc(item.title || item.Title || '');
  const url = item.url || item.URL || '';
  const reason = esc(item.reason || item.Reason || '');
  const label = url ? `<a href="${esc(url)}">${id}</a>` : id;
  const badges = badgesHTML(item.badges || plainBadges(item));
  const klass = ['item', isClosed({ state: item.state || item.State }) ? 'closedItem' : ''].filter(Boolean).join(' ');
  return `<div class="${klass}" data-node-id="${esc(item.id || item.ID || '')}"><div class="itemHead"><strong>${label} ${title}</strong>${badges}</div><div class="reason">${reason}</div></div>`;
}

function graphVisibleNodeSelection(snapshot, nodes) {
  const visible = new Set(nodes.map((node) => node.id));
  const connected = new Set(graphLayoutEdges(snapshot, visible).flatMap((edge) => [edge.from, edge.to]));
  const connectedNodes = nodes.filter((node) => connected.has(node.id));
  const unlinkedNodes = nodes.filter((node) => !connected.has(node.id));
  if (state.graphDriver === 'backlog') return { nodes: unlinkedNodes, hidden: { connected: connectedNodes.length, unlinked: 0 } };
  if (state.graphDriver === 'focus') return { nodes, hidden: { connected: 0, unlinked: unlinkedNodes.length } };
  if (state.showGraphUnlinked) return { nodes, hidden: { connected: 0, unlinked: 0 } };
  if (state.showGraphAllConnected) return { nodes: connectedNodes.length ? connectedNodes : nodes.slice(0, 24), hidden: { connected: 0, unlinked: unlinkedNodes.length } };

  const overviewIDs = new Set();
  for (const edge of graphFocusEdges(snapshot)) {
    overviewIDs.add(edge.from_id);
    overviewIDs.add(edge.to_id);
  }
  if (state.selectedNodeID) overviewIDs.add(state.selectedNodeID);
  const edgeLimit = state.graphDriver === 'focus' ? 4 : state.graphDriver === 'pairs' ? 12 : 8;
  for (const edge of graphOverviewEdges(snapshot, visible).slice(0, edgeLimit)) {
    if (overviewIDs.size >= 14 && !overviewIDs.has(edge.from) && !overviewIDs.has(edge.to)) break;
    overviewIDs.add(edge.from);
    overviewIDs.add(edge.to);
  }
  if (overviewIDs.size === 0) {
    for (const node of connectedNodes.slice(0, 12)) overviewIDs.add(node.id);
  }
  const graphNodes = nodes.filter((node) => overviewIDs.has(node.id));
  return {
    nodes: graphNodes.length ? graphNodes : nodes.slice(0, Math.min(nodes.length, 12)),
    hidden: {
      connected: Math.max(0, connectedNodes.length - graphNodes.length),
      unlinked: unlinkedNodes.length,
    },
  };
}

function graphOverviewEdges(snapshot, visible) {
  return graphLayoutEdges(snapshot, visible)
    .sort((a, b) => graphRelationScore(b, snapshot) - graphRelationScore(a, snapshot) || String(a.to).localeCompare(String(b.to)))
    .slice(0, 8);
}

function graphRelationScore(edge, snapshot) {
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const to = nodes.get(edge.to);
  const from = nodes.get(edge.from);
  let score = graphDegree(graphLayoutEdges(snapshot, new Set(snapshot.nodes.map((node) => node.id))), edge.from) + graphDegree(graphLayoutEdges(snapshot, new Set(snapshot.nodes.map((node) => node.id))), edge.to);
  if (to && !isClosed(to)) score += 8;
  if (from && !isClosed(from)) score += 4;
  if (state.selectedNodeID && (edge.from === state.selectedNodeID || edge.to === state.selectedNodeID)) score += 20;
  return score;
}

function renderGraph(snapshot, nodes, hidden = {}) {
  const hints = {
    pairs: 'Grouped by dependency direction. Blocked items on left, blockers on right.',
    focus: 'Select an item to see its neighbors and dependency chain.',
    backlog: 'Items not yet linked to anything. Use to triage and connect.',
    cluster: 'Items grouped by label, repo, or kind.',
  };
  const hintEl = document.getElementById('graphDriverHint');
  if (hintEl) hintEl.textContent = hints[state.graphDriver] || '';
  if (dom.graphDriver && dom.graphDriver.value !== state.graphDriver) dom.graphDriver.value = state.graphDriver;
  if (state.graphDriver === 'pairs') {
    renderGraphPairs(snapshot, nodes, hidden);
    return;
  }
  if (state.graphDriver === 'focus') {
    renderGraphFocusDriver(snapshot, nodes, hidden);
    return;
  }
  if (state.graphDriver === 'backlog') {
    renderGraphBacklog(snapshot, nodes, hidden);
    return;
  }
  if (state.graphDriver === 'cluster') {
    renderGraphCluster(snapshot, nodes);
    return;
  }
  const hiddenConnected = Number(hidden.connected || 0);
  const hiddenUnlinked = Number(hidden.unlinked || 0);
  if (dom.graphConnectedToggle) {
    dom.graphConnectedToggle.textContent = state.showGraphAllConnected ? 'Overview only' : `Show more connected${hiddenConnected ? ` (${hiddenConnected})` : ''}`;
    dom.graphConnectedToggle.classList.toggle('active', state.showGraphAllConnected);
    dom.graphConnectedToggle.disabled = hiddenConnected === 0 && !state.showGraphAllConnected;
  }
  if (dom.graphUnlinkedToggle) {
    dom.graphUnlinkedToggle.textContent = state.showGraphUnlinked ? 'Hide unlinked' : `Show unlinked${hiddenUnlinked ? ` (${hiddenUnlinked})` : ''}`;
    dom.graphUnlinkedToggle.classList.toggle('active', state.showGraphUnlinked);
    dom.graphUnlinkedToggle.disabled = hiddenUnlinked === 0 && !state.showGraphUnlinked;
  }
  const visible = new Set(nodes.map((node) => node.id));
  const selectedEdge = edgeByID(state.selectedEdgeID);
  const selectedEndpoints = selectedEdge ? new Set([selectedEdge.from_id, selectedEdge.to_id]) : new Set();
  const focusEdges = graphFocusEdges(snapshot);
  const focusNodeIDs = new Set(focusEdges.flatMap((edge) => [edge.from_id, edge.to_id]));
  const layout = graphLayout(snapshot, nodes);
  state.graphLayout = { width: layout.width, height: layout.height };
  const zoom = graphZoom();
  const positions = layout.positions;
  const hiddenSummary = hiddenConnected > 0 || hiddenUnlinked > 0
    ? `<div class="graphHiddenSummary">
        ${hiddenConnected > 0 ? `<span><strong>${hiddenConnected}</strong> connected items hidden</span> <button type="button" data-graph-action="toggle-connected">Show connected</button>` : ''}
        ${hiddenUnlinked > 0 ? `<span><strong>${hiddenUnlinked}</strong> unlinked items hidden</span> <button type="button" data-graph-action="toggle-unlinked">Show backlog</button>` : ''}
      </div>`
    : '';
  let html = `${hiddenSummary}<div class="graphScale" style="width:${Math.ceil(layout.width * zoom)}px;min-height:${Math.ceil(layout.height * zoom)}px">
  <div class="graphInner" style="width:${layout.width}px;min-height:${layout.height}px;transform:scale(${zoom})">
    ${graphColumnHeaders(layout)}
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
    const inFocus = focusEdges.some((focusEdge) => edgeSelectionID(focusEdge) === edgeID);
    const kind = esc(edge.kind || 'edge');
    const authority = esc(edge.authority || '');
    const line = graphEdgeLine(from, to);
    html += `<line class="graphEdgeHit" data-edge-id="${esc(edgeID)}" x1="${line.x1}" y1="${line.y1}" x2="${line.x2}" y2="${line.y2}"></line>`;
    html += `<line class="${edgeClasses(edge)}${selected ? ' selectedEdge' : ''}${inFocus ? ' focusEdge' : ''}" data-edge-id="${esc(edgeID)}" x1="${line.x1}" y1="${line.y1}" x2="${line.x2}" y2="${line.y2}" marker-end="url(#${selected ? 'arrowSelected' : (soft ? 'arrowSoft' : 'arrowHard')})"><title>${kind}${authority ? ` - ${authority}` : ''}</title></line>`;
  }
  html += '</svg>';
  for (const node of nodes) {
    const pos = positions.get(node.id);
    const klass = [
      'nodeCard',
      ...nodeCardClasses(node),
      selectedEndpoints.has(node.id) ? 'selectedEndpoint' : '',
      node.id === state.selectedNodeID ? 'selectedNode' : '',
      focusNodeIDs.has(node.id) && node.id !== state.selectedNodeID ? 'focusNode' : '',
      state.selectedNodeID && !focusNodeIDs.has(node.id) && node.id !== state.selectedNodeID ? 'dimNode' : '',
      isBlocked(node.id, snapshot) ? 'blocked' : 'ready',
    ].filter(Boolean).join(' ');
    const id = node.url ? `<a href="${esc(node.url)}">${esc(node.id)}</a>` : esc(node.id);
    const ref = nodeReferenceLabel(node);
    const kind = nodeKindLabel(node);
    html += `<article class="${klass}" data-node-id="${esc(node.id)}" style="transform:translate(${pos.x}px, ${pos.y}px)">
      <div class="nodeTop"><span class="nodeKind">${esc(kind)}</span><span class="nodeRef">${esc(ref)}</span></div>
      <div class="nodeTitle">${emojiHTML(node.title)}</div>
      <div class="nodeId">${id}</div>
      ${badgesHTML(nodeBadges(node))}
      ${nodeSignalsHTML(node)}
    </article>`;
  }
  html += '</div></div>';
  dom.graphCanvas.innerHTML = html;
  dom.graphZoomLabel.textContent = `${Math.round(zoom * 100)}%`;
}

function renderGraphPairs(snapshot, nodes, hidden = {}) {
  if (dom.graphConnectedToggle) {
    dom.graphConnectedToggle.textContent = hidden.connected ? `Show more connected (${hidden.connected})` : 'Connected shown';
    dom.graphConnectedToggle.classList.remove('active');
    dom.graphConnectedToggle.disabled = !hidden.connected;
  }
  if (dom.graphUnlinkedToggle) {
    dom.graphUnlinkedToggle.textContent = hidden.unlinked ? `Show unlinked (${hidden.unlinked})` : 'Backlog hidden';
    dom.graphUnlinkedToggle.classList.toggle('active', state.showGraphUnlinked);
    dom.graphUnlinkedToggle.disabled = !hidden.unlinked && !state.showGraphUnlinked;
  }
  state.graphLayout = { width: 900, height: 620 };
  dom.graphZoomLabel.textContent = 'pairs';
  const visible = new Set(nodes.map((node) => node.id));
  const nodesByID = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const groups = graphPairGroups(snapshot, visible);
  const hiddenConnected = Number(hidden.connected || 0);
  const hiddenUnlinked = Number(hidden.unlinked || 0);
  const hiddenSummary = hiddenConnected > 0 || hiddenUnlinked > 0
    ? `<div class="graphHiddenSummary">
        ${hiddenConnected > 0 ? `<span><strong>${hiddenConnected}</strong> connected items hidden</span> <button type="button" data-graph-action="toggle-connected">Show connected</button>` : ''}
        ${hiddenUnlinked > 0 ? `<span><strong>${hiddenUnlinked}</strong> unlinked items hidden</span> <button type="button" data-graph-action="toggle-unlinked">Show backlog</button>` : ''}
      </div>`
    : '';
  if (!groups.length) {
    dom.graphCanvas.innerHTML = `${hiddenSummary}<div class="graphEmpty">No visible relations for this driver.</div>`;
    return;
  }

  function groupTier(group) {
    if (group.edges.every((e) => !isSoftEdge(e) && Number(e.confidence || 1) >= 1)) return 'Confirmed';
    if (group.edges.some((e) => /inferred/i.test(e.authority || ''))) return 'Inferred';
    return 'Suggested';
  }

  const tierOrder = ['Confirmed', 'Inferred', 'Suggested'];
  const sortedGroups = groups.slice().sort((a, b) => {
    const ta = tierOrder.indexOf(groupTier(a));
    const tb = tierOrder.indexOf(groupTier(b));
    return ta - tb;
  });

  let currentTier = '';
  const pairHTML = sortedGroups.map((group) => {
    const from = nodesByID.get(group.fromID) || placeholderNode(group.fromID);
    const selected = group.edges.some((edge) => edgeSelectionID(edge) === state.selectedEdgeID);
    const tier = groupTier(group);
    let tierHeader = '';
    if (tier !== currentTier) {
      currentTier = tier;
      tierHeader = `<h3 class="pairTierLabel">${esc(tier)}</h3>`;
    }
    const articleHTML = `<article class="graphPairGroup ${selected ? 'selectedPair' : ''}">
      ${renderGraphPairNode(from, 'Source')}
      <div class="graphPairTargets">
        ${group.edges.map((edge) => {
          const to = nodesByID.get(edge.to_id) || placeholderNode(edge.to_id);
          const edgeID = edgeSelectionID(edge);
          const isSuggested = isSuggestedEdge(edge) && !hasOfficialEquivalent(state.data.snapshot, edge);
          const actionsHTML = isSuggested
            ? `<div class="pairTargetActions">
                <button type="button" class="primaryAction" data-suggestion-action="promote" data-edge-id="${esc(edgeID)}">Accept</button>
                <button type="button" data-suggestion-action="dismiss" data-edge-id="${esc(edgeID)}">Hide</button>
              </div>`
            : '';
          return `<div class="graphPairTarget" data-edge-id="${esc(edgeID)}">
            <span class="graphPairRelation"><strong>${esc(relationLabel(edge.kind))}</strong><em>${esc(confidenceLabel(edge))} · ${esc(edge.authority || 'local')}</em></span>
            ${renderGraphPairNode(to, 'Target')}
            ${actionsHTML}
          </div>`;
        }).join('')}
      </div>
    </article>`;
    return tierHeader + articleHTML;
  }).join('');

  dom.graphCanvas.innerHTML = `${hiddenSummary}<div class="graphPairs">${pairHTML}</div>`;
}

function renderGraphFocusDriver(snapshot, nodes, hidden = {}) {
  const node = nodeByID(state.selectedNodeID) || autoFocusGraphNode(snapshot, nodes);
  if (dom.graphConnectedToggle) {
    dom.graphConnectedToggle.textContent = node ? 'Focus selected' : 'No focus';
    dom.graphConnectedToggle.classList.remove('active');
    dom.graphConnectedToggle.disabled = true;
  }
  if (dom.graphUnlinkedToggle) {
    dom.graphUnlinkedToggle.textContent = hidden.unlinked ? `Backlog (${hidden.unlinked})` : 'Backlog empty';
    dom.graphUnlinkedToggle.classList.remove('active');
    dom.graphUnlinkedToggle.disabled = true;
  }
  state.graphLayout = { width: 900, height: 620 };
  dom.graphZoomLabel.textContent = 'focus';
  if (!node) {
    dom.graphCanvas.innerHTML = '<div class="graphEmpty">Select an item from Relation pairs to inspect its neighborhood.</div>';
    return;
  }
  const blockers = graphBlockingNeighbors(snapshot, node.id, 'blockers');
  const unlocks = graphBlockingNeighbors(snapshot, node.id, 'blocked');
  const related = graphRelatedNeighbors(snapshot, node.id);
  dom.graphCanvas.innerHTML = `<div class="graphFocusDriver">
    <section class="graphFocusHero">${renderGraphPairNode(node, 'Selected')}</section>
    ${renderGraphFocusLane('Blockers', blockers, 'No active blockers')}
    ${renderGraphFocusLane('Unlocks', unlocks, 'Blocks nothing active')}
    ${renderGraphFocusLane('Related', related, 'No related items')}
  </div>`;
}

function renderGraphFocusLane(title, nodes, emptyText) {
  return `<section class="graphFocusLane">
    <h3>${esc(title)} <strong>${nodes.length}</strong></h3>
    <div>${nodes.length ? nodes.slice(0, 12).map((node) => renderGraphPairNode(node, title.slice(0, -1) || title)).join('') : `<div class="graphEmpty">${esc(emptyText)}</div>`}</div>
  </section>`;
}

function autoFocusGraphNode(snapshot, nodes) {
  const visible = new Set(nodes.map((node) => node.id));
  const firstEdge = graphPairEdges(snapshot, visible)[0];
  if (firstEdge) return nodeByID(firstEdge.from_id) || nodeByID(firstEdge.to_id);
  return nodes.find((node) => !isClosed(node)) || nodes[0] || null;
}

function renderGraphBacklog(snapshot, nodes, hidden = {}) {
  if (dom.graphConnectedToggle) {
    dom.graphConnectedToggle.textContent = hidden.connected ? `${hidden.connected} linked hidden` : 'Linked hidden';
    dom.graphConnectedToggle.classList.remove('active');
    dom.graphConnectedToggle.disabled = true;
  }
  if (dom.graphUnlinkedToggle) {
    dom.graphUnlinkedToggle.textContent = `${nodes.length} backlog items`;
    dom.graphUnlinkedToggle.classList.add('active');
    dom.graphUnlinkedToggle.disabled = true;
  }
  state.graphLayout = { width: 900, height: 620 };
  dom.graphZoomLabel.textContent = 'backlog';
  if (!nodes.length) {
    dom.graphCanvas.innerHTML = '<div class="graphEmpty">No unlinked items match the current filters.</div>';
    return;
  }
  dom.graphCanvas.innerHTML = `<div class="graphBacklog">
    ${nodes.sort(graphNodeSort).slice(0, 80).map((node) => renderGraphPairNode(node, 'Unlinked')).join('')}
  </div>`;
}

function renderGraphCluster(snapshot, nodes) {
  if (dom.graphConnectedToggle) { dom.graphConnectedToggle.textContent = 'N/A'; dom.graphConnectedToggle.disabled = true; }
  if (dom.graphUnlinkedToggle) { dom.graphUnlinkedToggle.textContent = 'N/A'; dom.graphUnlinkedToggle.disabled = true; }
  state.graphLayout = { width: 900, height: 620 };
  dom.graphZoomLabel.textContent = 'cluster';
  if (!nodes.length) { dom.graphCanvas.innerHTML = '<div class="graphEmpty">No visible items for cluster view.</div>'; return; }
  const groups = {};
  for (const n of nodes) {
    const nd = nodeData(n);
    const gh = parseGitHubNodeID(n.id);
    const key = (Array.isArray(nd.labels) && nd.labels[0]) || (gh && gh.repo) || n.kind || 'other';
    if (!groups[key]) groups[key] = [];
    groups[key].push(n);
  }
  dom.graphCanvas.innerHTML = `<div class="graphClusters">${Object.entries(groups).map(([key, gnodes]) => `
    <div class="clusterGroup">
      <div class="clusterGroupLabel">${esc(key)}</div>
      <div class="clusterGroupCards">${gnodes.map((n) => renderGraphPairNode(n, 'Card')).join('')}</div>
    </div>`).join('')}</div>`;
}

function graphPairGroups(snapshot, visible) {
  const limit = state.showGraphAllConnected ? 24 : 10;
  const grouped = new Map();
  for (const edge of graphPairEdges(snapshot, visible)) {
    if (!grouped.has(edge.from_id)) grouped.set(edge.from_id, []);
    grouped.get(edge.from_id).push(edge);
  }
  return Array.from(grouped, ([fromID, edges]) => ({ fromID, edges: edges.slice(0, 5) }))
    .sort((a, b) => graphRawEdgeScore(b.edges[0]) - graphRawEdgeScore(a.edges[0]) || String(a.fromID).localeCompare(String(b.fromID)))
    .slice(0, limit);
}

function graphPairEdges(snapshot, visible) {
  const seen = new Set();
  const edges = [];
  for (const edge of snapshot.edges || []) {
    if (!visible.has(edge.from_id) || !visible.has(edge.to_id)) continue;
    const key = `${edge.from_id}\x00${edge.to_id}\x00${edge.kind || ''}`;
    if (seen.has(key)) continue;
    seen.add(key);
    edges.push(edge);
  }
  return edges.sort((a, b) => graphRawEdgeScore(b) - graphRawEdgeScore(a) || String(a.from_id).localeCompare(String(b.from_id)));
}

function graphRawEdgeScore(edge) {
  let score = Number(edge.confidence || 1) * 10;
  if (!isSoftEdge(edge)) score += 10;
  if (!isNonBlockingEdgeKind(String(edge.kind || '').toLowerCase())) score += 6;
  if (state.selectedNodeID && (edge.from_id === state.selectedNodeID || edge.to_id === state.selectedNodeID)) score += 20;
  return score;
}

function renderGraphPairNode(node, label) {
  return `<button type="button" class="graphPairNode" data-node-id="${esc(node.id)}">
    <span>${esc(label)} · ${esc(nodeKindLabel(node))} ${esc(nodeReferenceLabel(node))}</span>
    <strong>${emojiHTML(node.title || node.id)}</strong>
    ${badgesHTML(nodeBadges(node).slice(0, 4))}
    ${nodeSignalsHTML(node)}
  </button>`;
}

function graphColumnHeaders(layout) {
  return (layout.columns || []).map((column) => `<div class="graphColumnHeader" style="transform:translate(${column.x}px, ${column.y}px)">
    <strong>${esc(column.title)}</strong>
    <span>${esc(column.count)} item${column.count === 1 ? '' : 's'}</span>
  </div>`).join('');
}

function renderGraphFocusPanel(snapshot) {
  if (!dom.graphFocus) return;
  const node = nodeByID(state.selectedNodeID);
  dom.graphFocus.classList.toggle('hidden', state.view !== 'graph' || state.graphDriver === 'focus' || !node);
  if (state.view !== 'graph' || state.graphDriver === 'focus' || !node) {
    dom.graphFocus.innerHTML = '';
    return;
  }
  const blockers = graphBlockingNeighbors(snapshot, node.id, 'blockers');
  const blocked = graphBlockingNeighbors(snapshot, node.id, 'blocked');
  const related = graphRelatedNeighbors(snapshot, node.id).slice(0, 4);
  const blockerHTML = graphFocusNodeList(blockers, 'No active blockers');
  const blockedHTML = graphFocusNodeList(blocked, 'Blocks nothing active');
  const relatedHTML = graphFocusNodeList(related, 'No extra links');
  dom.graphFocus.innerHTML = `<section class="graphFocusBox">
    <div class="graphFocusHead">
      <div>
        <span>${esc(nodeKindLabel(node))} ${esc(nodeReferenceLabel(node))}</span>
        <strong>${emojiHTML(node.title || node.id)}</strong>
      </div>
      <button type="button" data-graph-focus-action="clear">Clear</button>
    </div>
    <div class="graphFocusColumns">
      <div><h3>Blockers</h3>${blockerHTML}</div>
      <div><h3>Unlocks</h3>${blockedHTML}</div>
      <div><h3>Related</h3>${relatedHTML}</div>
    </div>
  </section>`;
}

function graphFocusNodeList(nodes, emptyText) {
  if (!nodes.length) return `<div class="graphFocusEmpty">${esc(emptyText)}</div>`;
  return nodes.slice(0, 5).map((node) => `<button type="button" class="graphFocusNode" data-node-id="${esc(node.id)}">
    <span>${esc(nodeReferenceLabel(node))}</span>
    <strong>${emojiHTML(node.title || node.id)}</strong>
  </button>`).join('');
}

function graphFocusEdges(snapshot) {
  const id = state.selectedNodeID;
  if (!id) return [];
  return (snapshot.edges || []).filter((edge) => edge.from_id === id || edge.to_id === id);
}

function graphBlockingNeighbors(snapshot, nodeID, direction) {
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const out = [];
  for (const edge of snapshot.edges || []) {
    const [blocked, blocker] = edgeBlockedAndBlocker(edge);
    if (!blocked || !blocker) continue;
    const candidateID = direction === 'blockers' && blocked === nodeID ? blocker : direction === 'blocked' && blocker === nodeID ? blocked : '';
    if (!candidateID) continue;
    const node = nodes.get(candidateID);
    if (node && !isClosed(node)) out.push(node);
  }
  return out.sort(graphNodeSort);
}

function graphRelatedNeighbors(snapshot, nodeID) {
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const seen = new Set();
  const out = [];
  for (const edge of snapshot.edges || []) {
    if (!isNonBlockingEdgeKind(String(edge.kind || '').toLowerCase())) continue;
    const otherID = edge.from_id === nodeID ? edge.to_id : edge.to_id === nodeID ? edge.from_id : '';
    if (!otherID || seen.has(otherID)) continue;
    const node = nodes.get(otherID);
    if (node) {
      seen.add(otherID);
      out.push(node);
    }
  }
  return out.sort(graphNodeSort);
}

function handleGraphFocusClick(event) {
  const nodeButton = event.target.closest('[data-node-id]');
  if (nodeButton && dom.graphFocus.contains(nodeButton)) {
    selectNode(nodeButton.dataset.nodeId || '');
    requestAnimationFrame(scrollSelectedNodeIntoView);
    return;
  }
  const button = event.target.closest('[data-graph-focus-action]');
  if (!button) return;
  if (button.dataset.graphFocusAction === 'clear') {
    state.selectedNodeID = '';
    state.selectedEdgeID = '';
    render();
  }
}

function handleGraphControlClick(event) {
  const button = event.target.closest('[data-graph-action]');
  if (!button) return;
  applyGraphAction(button.dataset.graphAction);
}

function handleGraphKeydown(event) {
  if (state.view !== 'graph' || isTypingTarget(event.target)) return;
  const key = event.key.toLowerCase();
  if (key === 'f') {
    event.preventDefault();
    applyGraphAction('fit');
  }
  if (key === '+' || key === '=') {
    event.preventDefault();
    applyGraphAction('in');
  }
  if (key === '-' || key === '_') {
    event.preventDefault();
    applyGraphAction('out');
  }
  if (key === '0') {
    event.preventDefault();
    applyGraphAction('reset');
  }
  if (key === 'escape' && state.selectedEdgeID) {
    event.preventDefault();
    state.selectedEdgeID = '';
    render();
    dom.status.textContent = 'edge selection cleared';
  }
}

function isTypingTarget(target) {
  const tag = String(target?.tagName || '').toLowerCase();
  return tag === 'input' || tag === 'textarea' || tag === 'select' || Boolean(target?.isContentEditable);
}

function applyGraphAction(action) {
  if (action === 'fit') {
    fitGraphToCanvas();
    return;
  }
  if (action === 'toggle-connected') state.showGraphAllConnected = !state.showGraphAllConnected;
  if (action === 'toggle-unlinked') state.showGraphUnlinked = !state.showGraphUnlinked;
  if (action === 'reset') state.graphZoom = 1;
  if (action === 'in') state.graphZoom = graphZoom(state.graphZoom + 0.15);
  if (action === 'out') state.graphZoom = graphZoom(state.graphZoom - 0.15);
  render();
}

function fitGraphToCanvas() {
  const width = Math.max(1, state.graphLayout.width);
  const height = Math.max(1, state.graphLayout.height);
  const next = Math.min(
    (dom.graphCanvas.clientWidth - 28) / width,
    (dom.graphCanvas.clientHeight - 28) / height,
  );
  state.graphZoom = graphZoom(next);
  render();
  dom.graphCanvas.scrollTo({ top: 0, left: 0 });
}

function graphZoom(value = state.graphZoom) {
  return Math.max(0.35, Math.min(1.8, Number(value) || 1));
}

function handleGraphClick(event) {
  const suggBtn = event.target.closest('[data-suggestion-action]');
  if (suggBtn && dom.graphCanvas.contains(suggBtn)) {
    handleSuggestionClick(event);
    return;
  }
  const nodeTarget = event.target.closest?.('[data-node-id]');
  if (nodeTarget && dom.graphCanvas.contains(nodeTarget)) {
    selectNode(nodeTarget.dataset.nodeId || '');
    requestAnimationFrame(scrollSelectedNodeIntoView);
    return;
  }
  const edgeTarget = event.target.closest?.('[data-edge-id]');
  if (edgeTarget && dom.graphCanvas.contains(edgeTarget)) {
    const edgeID = edgeTarget.dataset.edgeId || '';
    if (!edgeByID(edgeID)) return;
    state.selectedEdgeID = edgeID;
    state.selectedNodeID = '';
    render();
    dom.status.textContent = 'edge selected';
    return;
  }
}

function graphLayout(snapshot, nodes) {
  const cardWidth = 204;
  const cardHeight = 104;
  const xGap = 246;
  const yGap = 104;
  const padX = 26;
  const padY = 72;
  const visible = new Set(nodes.map((node) => node.id));
  const layoutEdges = graphLayoutEdges(snapshot, visible);
  const connected = new Set(layoutEdges.flatMap((edge) => [edge.from, edge.to]));
  const connectedNodes = nodes.filter((node) => connected.has(node.id)).sort(graphNodeSort);
  const isolatedNodes = nodes.filter((node) => !connected.has(node.id)).sort(graphNodeSort);
  const positions = new Map();

  if (connectedNodes.length === 0) {
    placeNodeGrid(isolatedNodes, positions, { x: padX, y: padY, cols: graphGridColumns(isolatedNodes.length), xGap, yGap });
    const layout = graphLayoutSize(positions, cardWidth, cardHeight, padX, padY);
    layout.columns = [{ x: padX, y: 24, title: 'Unlinked work', count: isolatedNodes.length }];
    return layout;
  }

  const ranks = graphRanks(connectedNodes, layoutEdges);
  const columns = new Map();
  for (const node of connectedNodes) {
    const rank = ranks.get(node.id) || 0;
    if (!columns.has(rank)) columns.set(rank, []);
    columns.get(rank).push(node);
  }
  const orderedRanks = Array.from(columns.keys()).sort((a, b) => a - b);
  const columnMeta = [];
  for (const [index, rank] of orderedRanks.entries()) {
    const column = columns.get(rank).sort((a, b) => graphDegree(layoutEdges, b.id) - graphDegree(layoutEdges, a.id) || graphNodeSort(a, b));
    const x = padX + index * xGap;
    placeNodeGrid(column, positions, { x, y: padY, cols: 1, xGap, yGap });
    columnMeta.push({ x, y: 24, title: graphRankTitle(index, orderedRanks.length), count: column.length });
  }

  if (isolatedNodes.length > 0) {
    const isolatedCols = graphGridColumns(isolatedNodes.length);
    const isolatedX = padX + orderedRanks.length * xGap + 38;
    placeNodeGrid(isolatedNodes, positions, { x: isolatedX, y: padY, cols: isolatedCols, xGap: 236, yGap: 102 });
    columnMeta.push({ x: isolatedX, y: 24, title: 'Unlinked', count: isolatedNodes.length });
  }

  const layout = graphLayoutSize(positions, cardWidth, cardHeight, padX, padY);
  layout.columns = columnMeta;
  return layout;
}

function graphRankTitle(index, total) {
  if (total <= 1) return 'Connected work';
  if (index === 0) return 'Upstream';
  if (index === total - 1) return 'Downstream';
  return `Layer ${index + 1}`;
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
  const cardWidth = 204;
  const centerY = 50;
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
    return `<tr class="${klass}" data-node-id="${esc(node.id)}">
      <td><div class="tableItem"><div class="tableItemID">${id}</div><div class="tableItemTitle">${emojiHTML(node.title)}</div></div></td>
      <td>${badgesHTML(nodeBadges(node))}</td>
      <td>${emojiHTML(labelText)}</td>
    </tr>`;
  }).join('');
  const empty = '<tr><td colspan="3"><div class="reason">no visible cards</div></td></tr>';
  dom.table.innerHTML = `<table class="workTable">
    <colgroup><col class="itemCol"><col class="signalCol"><col class="labelCol"></colgroup>
    <thead><tr><th>Item</th><th>Signals</th><th>Labels</th></tr></thead>
    <tbody>${rows || empty}</tbody>
  </table>`;
}

function handleNodePickClick(event) {
  const target = event.target.closest('[data-node-id]');
  if (!target) return;
  selectNode(target.dataset.nodeId || '');
}

function handleBriefClick(event) {
  const authAction = event.target.closest('[data-auth-action]');
  if (authAction) {
    handleAuthGateAction(authAction);
    return;
  }
  const emptyAction = event.target.closest('[data-empty-action]');
  if (emptyAction) {
    handleEmptyBoardAction(emptyAction);
    return;
  }
  handleNodePickClick(event);
}

function handleAuthGateAction(target) {
  const action = target.dataset.authAction || '';
  if (action === 'signin') {
    signInWithBackendGitHub();
    return;
  }
  if (action === 'stateless') {
    setMode('stateless', { renderNow: true });
  }
}

function handleEmptyBoardAction(target) {
  const action = target.dataset.emptyAction || '';
  if (action === 'add-item') {
    setWorkspaceTab('actions');
    dom.newItemRef.focus();
    return;
  }
  if (action === 'sources') {
    setWorkspaceTab('presets');
    loadGitHubPresets();
    return;
  }
  if (action === 'sync') {
    syncCurrentBoard();
    return;
  }
  if (action === 'useful' && target.dataset.boardId) {
    state.currentBoardID = target.dataset.boardId;
    writeURLBoard(state.currentBoardID);
    loadBackendBoard();
  }
}

function selectNode(nodeID) {
  if (!nodeByID(nodeID)) return;
  state.selectedNodeID = nodeID;
  writeURLNode(nodeID);
  state.selectedEdgeID = '';
  render();
  dom.status.textContent = 'item selected';
}

function renderSuggestions(snapshot) {
  const nodes = new Map(snapshot.nodes.map((node) => [node.id, node]));
  const suggestions = suggestedEdges(snapshot).filter((edge) => !state.dismissedSuggestionIDs.has(edgeSelectionID(edge)));
  dom.suggestions.classList.toggle('hidden', suggestions.length === 0);
  if (suggestions.length === 0) {
    dom.suggestions.innerHTML = '';
    return;
  }
  const limit = state.view === 'graph' ? 1 : 8;
  const rows = suggestions.slice(0, limit).map((edge) => renderSuggestion(edge, nodes)).join('');
  const more = suggestions.length > limit ? `<div class="suggestionMore">${suggestions.length - limit} more in Links</div>` : '';
  dom.suggestions.innerHTML = `<section class="suggestionBox" aria-label="Suggested relations">
    <div class="suggestionHead">
      <div>
        <strong>Suggested relations</strong>
        <span>${suggestions.length} soft edge${suggestions.length === 1 ? '' : 's'} from GitHub or low-confidence sources</span>
      </div>
      <button type="button" data-suggestion-action="review-all">Review all</button>
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
  return `<article class="suggestionRow ${selected ? 'selectedSuggestion' : ''}" data-edge-id="${esc(edgeID)}">
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
        <button type="button" class="dangerAction" data-edge-action="delete-edge">Delete link</button>
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

function renderItemInspector(snapshot) {
  if (!dom.itemInspector) return;
  const node = nodeByID(state.selectedNodeID);
  dom.itemInspector.classList.toggle('hidden', !node || state.mode !== 'stateful');
  if (!node || state.mode !== 'stateful') {
    dom.itemInspector.innerHTML = '';
    return;
  }
  const incoming = (snapshot.edges || []).filter((edge) => edge.to_id === node.id);
  const outgoing = (snapshot.edges || []).filter((edge) => edge.from_id === node.id);
  const data = nodeData(node);
  const github = parseGitHubNodeID(node.id);
  const labelsHTML = labels(node).slice(0, 10).map((label) => `<span>${emojiHTML(label)}</span>`).join('');
  const local = isLocal(node);
  const kindBadgeHTML = `<span class="badge type-${esc(badgeClass(node.kind || 'task'))}">${esc(nodeKindLabel(node))}</span>`;
  const stateBadge = lifecycleBadge(node.state);
  const stateBadgeHTML = stateBadge ? `<span class="badge ${esc(stateBadge.kind)}">${esc(stateBadge.text)}</span>` : '';
  const timeHorizonBadgeHTML = data.time_horizon ? `<span class="badge horizon-${esc(data.time_horizon)}">${esc(data.time_horizon)}</span>` : '';
  const priorityBadgeHTML = data.priority ? `<span class="badge priority-${esc(data.priority)}">${esc(data.priority)}</span>` : '';
  const editBtn = local ? `<button type="button" data-item-action="edit">Edit</button>` : '';
  const headSection = `<div class="inspectorHead">
    <div>
      <div class="inspectorBadges">${kindBadgeHTML}${stateBadgeHTML}${timeHorizonBadgeHTML}${priorityBadgeHTML}</div>
      <strong>${emojiHTML(node.title || node.id)}</strong>
    </div>
    <div>
      ${editBtn}
      <button type="button" data-item-action="close">×</button>
    </div>
  </div>`;
  const createdAt = data.created_at ? ` · Created ${formatDate(data.created_at)}` : '';
  const updatedAt = node.updated_at ? ` · Updated ${formatDate(node.updated_at)}` : '';
  const metaSection = `<div class="inspectorMeta">
    <span>${esc(node.id)}</span>
    ${node.owner ? `<span>@${esc(node.owner)}</span>` : ''}
    ${github ? `<span>${esc(github.repo)}</span>` : ''}
    ${(createdAt || updatedAt) ? `<span class="inspectorDates">${esc((createdAt + updatedAt).trim())}</span>` : ''}
  </div>`;
  const labelsSection = labelsHTML ? `<div class="inspectorLabels">${labelsHTML}</div>` : '';
  const descriptionSection = data.description ? `<div class="inspectorDescription">${emojiHTML(data.description)}</div>` : '';
  const editFormSection = local && state.inspectorEditMode ? `<form class="inspectorEditForm" id="inspectorEditForm">
    <label>Title<input type="text" name="title" value="${esc(node.title || '')}"></label>
    <label>Status
      <select name="status">
        <option value="draft"${node.state === 'draft' ? ' selected' : ''}>Draft</option>
        <option value="active"${node.state === 'active' ? ' selected' : ''}>Active</option>
        <option value="blocked"${node.state === 'blocked' ? ' selected' : ''}>Blocked</option>
        <option value="at-risk"${node.state === 'at-risk' ? ' selected' : ''}>At-risk</option>
        <option value="paused"${node.state === 'paused' ? ' selected' : ''}>Paused</option>
        <option value="open"${node.state === 'open' ? ' selected' : ''}>Open</option>
        <option value="done"${node.state === 'done' ? ' selected' : ''}>Done</option>
        <option value="rejected"${node.state === 'rejected' ? ' selected' : ''}>Rejected</option>
        <option value="local"${node.state === 'local' ? ' selected' : ''}>Local</option>
      </select>
    </label>
    <label>Owner<input type="text" name="owner" value="${esc(node.owner || '')}"></label>
    <label>Description<textarea name="description" rows="3">${esc(data.description || '')}</textarea></label>
    <label>Time horizon
      <select name="time_horizon">
        <option value="">No horizon</option>
        <option value="now"${data.time_horizon === 'now' ? ' selected' : ''}>Now</option>
        <option value="next"${data.time_horizon === 'next' ? ' selected' : ''}>Next</option>
        <option value="later"${data.time_horizon === 'later' ? ' selected' : ''}>Later</option>
        <option value="quarter"${data.time_horizon === 'quarter' ? ' selected' : ''}>This quarter</option>
        <option value="year"${data.time_horizon === 'year' ? ' selected' : ''}>This year</option>
        <option value="someday"${data.time_horizon === 'someday' ? ' selected' : ''}>Someday</option>
      </select>
    </label>
    <label>Priority
      <select name="priority">
        <option value="">No priority</option>
        <option value="critical"${data.priority === 'critical' ? ' selected' : ''}>Critical</option>
        <option value="high"${data.priority === 'high' ? ' selected' : ''}>High</option>
        <option value="medium"${data.priority === 'medium' ? ' selected' : ''}>Medium</option>
        <option value="low"${data.priority === 'low' ? ' selected' : ''}>Low</option>
      </select>
    </label>
    <label>Labels<input type="text" name="labels" value="${esc((data.labels || []).join(', '))}"></label>
    <label>Convert to
      <select name="convert_kind">
        <option value="">Keep current (${esc(node.kind)})</option>
        <option value="strategy">Strategy</option>
        <option value="initiative">Initiative</option>
        <option value="bet">Bet</option>
        <option value="project">Project</option>
        <option value="workstream">Workstream</option>
        <option value="risk">Risk</option>
        <option value="decision">Decision</option>
        <option value="question">Question</option>
        <option value="metric">Metric</option>
        <option value="task">Task</option>
        <option value="note">Note</option>
      </select>
    </label>
    <div class="inspectorFormActions">
      <button type="submit" class="primaryAction">Save</button>
      <button type="button" data-item-action="cancel-edit">Cancel</button>
    </div>
  </form>` : '';
  const duplicateBtn = local ? `<button type="button" data-item-action="duplicate">Duplicate</button>` : '';
  const deleteLabel = local ? 'Archive' : 'Remove';
  const deleteAction = local ? 'archive-node' : 'delete-node';
  const createGHIssueSection = local && state.backendSession.authenticated ? `
    <div class="inspectorGitHubCreate">
      <details>
        <summary>Create GitHub Issue</summary>
        <form class="inlineForm" id="createGHIssueForm">
          <input type="text" name="repo" placeholder="owner/repo" required>
          <input type="text" name="title" value="${esc(node.title || '')}" required>
          <textarea name="body" rows="2" placeholder="Description (optional)">${esc(data.description || '')}</textarea>
          <input type="text" name="labels" placeholder="Labels (comma-separated, optional)">
          <input type="text" name="assignees" placeholder="Assignees (comma-separated logins, optional)">
          <label class="inspectorCheckbox"><input type="checkbox" name="archive_local"> Archive local node after creating issue</label>
          <button type="submit" class="primaryAction">Create issue</button>
        </form>
      </details>
    </div>` : '';
  const githubStateSection = !local && github && state.backendSession.authenticated ? `
    <div class="inspectorGitHubActions">
      <div class="inspectorSectionLabel">GitHub Actions</div>
      ${node.state === 'open' || node.state === 'active' || node.state === 'draft' ? `<button type="button" data-item-action="close-github-issue">Close issue</button>` : `<button type="button" data-item-action="reopen-github-issue">Reopen issue</button>`}
    </div>` : '';
  const commentSection = !local && github && state.backendSession.authenticated ? `
    <div class="inspectorCommentCompose">
      <div class="inspectorSectionLabel">Add comment</div>
      <textarea id="inspectorCommentBody" rows="3" placeholder="Add a comment..."></textarea>
      <button type="button" data-item-action="submit-comment">Comment</button>
    </div>` : '';
  const linkCreateSection = `<div class="inspectorLinkCreate">
    <select id="inspectorLinkKind">
      <option value="blocked_by">depends on</option>
      <option value="blocks">blocks</option>
      <option value="relates_to">relates to</option>
      <option value="addresses">addresses</option>
    </select>
    <input id="inspectorLinkTarget" type="text" placeholder="node-id or title fragment">
    <button type="button" data-item-action="create-link-from-inspector">Add link</button>
  </div>`;
  const actionsSection = `<div class="inspectorActions">
    <div class="inspectorPrimaryActions">
      ${node.url ? `<a href="${esc(node.url)}" target="_blank" rel="noreferrer">Open GitHub</a>` : ''}
      <button type="button" data-item-action="link-from">Link from</button>
      <button type="button" data-item-action="link-to">Link to</button>
      ${duplicateBtn}
    </div>
    <div class="inspectorDangerActions">
      <button type="button" class="dangerAction" data-item-action="${deleteAction}">${esc(deleteLabel)}</button>
    </div>
  </div>`;
  dom.itemInspector.innerHTML = `<section class="inspectorBox">
    ${headSection}
    ${metaSection}
    ${labelsSection}
    ${descriptionSection}
    ${editFormSection}
    ${actionsSection}
    ${createGHIssueSection}
    ${githubStateSection}
    ${commentSection}
    ${linkCreateSection}
    <div class="inspectorSection inspectorLinks">
      ${outgoing.length ? `<div class="inspectorLinkGroup"><div class="linkGroupLabel">Blocks / Out</div>${renderInspectorLinks('Out', outgoing)}</div>` : ''}
      ${incoming.length ? `<div class="inspectorLinkGroup"><div class="linkGroupLabel">Blocked by / In</div>${renderInspectorLinks('In', incoming)}</div>` : ''}
      ${(!outgoing.length && !incoming.length) ? '<div class="emptyState">No links</div>' : ''}
    </div>
    <details class="inspectorRaw">
      <summary>Raw data</summary>
      <pre>${esc(JSON.stringify(data, null, 2))}</pre>
    </details>
  </section>`;
  // Wire edit form submit via delegation
  const editForm = document.getElementById('inspectorEditForm');
  if (editForm) {
    editForm.addEventListener('submit', (e) => { e.preventDefault(); saveNodeEdit(); });
  }
  const createGHForm = document.getElementById('createGHIssueForm');
  if (createGHForm) {
    createGHForm.addEventListener('submit', (e) => {
      e.preventDefault();
      const fd = new FormData(createGHForm);
      const labelsStr = fd.get('labels') || '';
      const assigneesStr = fd.get('assignees') || '';
      const archiveLocal = fd.get('archive_local') === 'on';
      const lbls = labelsStr ? labelsStr.split(',').map((l) => l.trim()).filter(Boolean) : [];
      const asgns = assigneesStr ? assigneesStr.split(',').map((a) => a.trim()).filter(Boolean) : [];
      createGitHubIssueFromNode(node.id, fd.get('repo'), fd.get('title'), fd.get('body'), lbls, asgns, archiveLocal);
    });
  }
}

function renderInspectorLinks(label, edges) {
  if (!edges.length) return `<div class="inspectorLink empty">${esc(label)}: none</div>`;
  return edges.slice(0, 8).map((edge) => {
    const otherID = label === 'Out' ? edge.to_id : edge.from_id;
    const other = nodeByID(otherID) || placeholderNode(otherID);
    return `<button type="button" class="inspectorLink" data-node-id="${esc(otherID)}">
      <span>${esc(label)} / ${esc(relationLabel(edge.kind))}</span>
      <strong>${esc(shortNodeLabel(other))}</strong>
    </button>`;
  }).join('');
}

function handleItemInspectorClick(event) {
  const nodeButton = event.target.closest('[data-node-id]');
  if (nodeButton && dom.itemInspector.contains(nodeButton)) {
    selectNode(nodeButton.dataset.nodeId || '');
    return;
  }
  const button = event.target.closest('[data-item-action]');
  if (!button) return;
  if (button.dataset.itemAction === 'close') {
    state.selectedNodeID = '';
    state.inspectorEditMode = false;
    writeURLNode('');
    render();
    return;
  }
  if (button.dataset.itemAction === 'edit') {
    state.inspectorEditMode = true;
    render();
    return;
  }
  if (button.dataset.itemAction === 'cancel-edit') {
    state.inspectorEditMode = false;
    render();
    return;
  }
  const node = nodeByID(state.selectedNodeID);
  if (!node) return;
  if (button.dataset.itemAction === 'delete-node') {
    deleteNodeFromBoard(state.selectedNodeID);
    return;
  }
  if (button.dataset.itemAction === 'archive-node') {
    archiveNodeFromBoard(state.selectedNodeID);
    return;
  }
  if (button.dataset.itemAction === 'duplicate') {
    duplicateNode(state.selectedNodeID);
    return;
  }
  if (button.dataset.itemAction === 'link-from') {
    dom.newLinkFrom.value = node.id;
    setWorkspaceTab('actions');
    dom.newLinkTo.focus();
  }
  if (button.dataset.itemAction === 'link-to') {
    dom.newLinkTo.value = node.id;
    setWorkspaceTab('actions');
    dom.newLinkFrom.focus();
  }
  if (button.dataset.itemAction === 'submit-comment') {
    const gh = parseGitHubNodeID(state.selectedNodeID);
    if (!gh) return;
    const bodyEl = document.getElementById('inspectorCommentBody');
    const commentBody = bodyEl ? bodyEl.value.trim() : '';
    if (!commentBody) return;
    submitGitHubComment(gh.repo, Number(gh.number), commentBody);
    return;
  }
  if (button.dataset.itemAction === 'close-github-issue' || button.dataset.itemAction === 'reopen-github-issue') {
    const gh = parseGitHubNodeID(state.selectedNodeID);
    if (!gh) return;
    const newState = button.dataset.itemAction === 'close-github-issue' ? 'closed' : 'open';
    closeOrReopenGitHubIssue(gh.repo, Number(gh.number), newState);
    return;
  }
  if (button.dataset.itemAction === 'create-link-from-inspector') {
    const kindEl = document.getElementById('inspectorLinkKind');
    const targetEl = document.getElementById('inspectorLinkTarget');
    const kind = kindEl ? kindEl.value : 'blocked_by';
    const targetStr = targetEl ? targetEl.value.trim() : '';
    if (!targetStr) return;
    const target = resolveNodeByRef(targetStr);
    if (!target) { dom.error.textContent = `Cannot find node: ${targetStr}`; return; }
    addBoardLinkDirect(state.selectedNodeID, kind, target.id);
    return;
  }
}

async function saveNodeEdit() {
  const form = document.getElementById('inspectorEditForm');
  if (!form) return;
  const nodeID = state.selectedNodeID;
  if (!nodeID) return;
  const data = Object.fromEntries(new FormData(form));
  const beforeNode = nodeByID(nodeID);
  const beforeNodeData = nodeData(beforeNode || {});
  const beforeData = {
    title: beforeNode?.title || '',
    status: beforeNode?.state || '',
    owner: beforeNode?.owner || '',
    description: beforeNodeData.description || '',
    time_horizon: beforeNodeData.time_horizon || '',
    priority: beforeNodeData.priority || '',
    labels: beforeNodeData.labels || [],
  };
  try {
    // If convert_kind is set, do a kind conversion first
    if (data.convert_kind) {
      const res2 = await fetch('./api/board-items', {
        method: 'PATCH',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ node_id: nodeID, kind: data.convert_kind }),
      });
      if (!res2.ok) throw new Error(await responseErrorMessage(res2));
    }
    const labelsStr = data.labels || '';
    const labelsList = labelsStr ? labelsStr.split(',').map((l) => l.trim()).filter(Boolean) : [];
    const res = await fetch('./api/board-items', {
      method: 'PATCH',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        node_id: nodeID,
        title: data.title || '',
        status: data.status || '',
        owner: data.owner || '',
        description: data.description || '',
        time_horizon: data.time_horizon || '',
        priority: data.priority || '',
        labels: labelsList,
      }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    pushUndo({ type: 'edit-node', nodeID, before: beforeData });
    state.inspectorEditMode = false;
    await loadBackendBoard();
    dom.status.textContent = 'node updated';
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'update failed';
  }
}

async function deleteNodeFromBoard(nodeID) {
  if (!nodeID) return;
  const node = nodeByID(nodeID);
  const label = node ? (node.title || node.id) : nodeID;
  if (!confirm(`Remove "${label.slice(0, 60)}" from this board?`)) return;
  try {
    const nodeSnap = node;
    const res = await fetch('./api/board-items', {
      method: 'DELETE',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ board_id: state.currentBoardID || 'default', node_id: nodeID }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    if (nodeSnap) pushUndo({ type: 'remove-node', snapshot: nodeSnap });
    state.selectedNodeID = '';
    state.selectedEdgeID = '';
    state.inspectorEditMode = false;
    await loadBackendBoard();
    await refreshBoards();
    dom.status.textContent = 'node removed';
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'remove failed';
  }
}

async function archiveNodeFromBoard(nodeID) {
  if (!nodeID) return;
  const node = nodeByID(nodeID);
  const label = node ? (node.title || node.id) : nodeID;
  if (!confirm(`Archive "${label.slice(0, 60)}"? You can restore it later.`)) return;
  try {
    const res = await fetch('./api/board-items', {
      method: 'DELETE',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ board_id: state.currentBoardID || 'default', node_id: nodeID, soft: true }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    pushUndo({ type: 'restore-node', nodeID });
    state.selectedNodeID = '';
    state.inspectorEditMode = false;
    await loadBackendBoard();
    dom.status.textContent = 'node archived';
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'archive failed';
  }
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
  if (button.dataset.edgeAction === 'delete-edge') deleteLinkFromBoard(edgeID);
}

async function deleteLinkFromBoard(edgeID) {
  if (!edgeID) return;
  const edge = edgeByID(edgeID);
  if (!confirm(`Delete this link (${edge ? relationLabel(edge.kind) : edgeID})?`)) return;
  try {
    const edgeSnap = edge;
    const res = await fetch('./api/board-links', {
      method: 'DELETE',
      credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ edge_id: edgeID }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    if (edgeSnap) pushUndo({ type: 'delete-link', snapshot: edgeSnap });
    state.selectedEdgeID = '';
    await loadBackendBoard();
    dom.status.textContent = 'link deleted';
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'delete failed';
  }
}

function handleSuggestionClick(event) {
  const button = event.target.closest('[data-suggestion-action]');
  if (!button) return;
  const edgeID = button.dataset.edgeId || '';
  const action = button.dataset.suggestionAction;
  if (action === 'review-all') {
    setWorkspaceTab('suggestions');
    dom.status.textContent = 'all suggested relations visible in Links';
    return;
  }
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

function scrollSelectedNodeIntoView() {
  const node = dom.graphCanvas.querySelector('.selectedNode');
  if (!node) return;
  node.scrollIntoView({ block: 'center', inline: 'center' });
}

async function dismissSuggestedEdge(edgeID) {
  state.dismissedSuggestionIDs.add(edgeID);
  if (state.selectedEdgeID === edgeID) state.selectedEdgeID = '';
  render();
  dom.status.textContent = 'suggested relation hidden';
  if (state.mode === 'stateful' && state.backendSession.authenticated) {
    try {
      await fetch('./api/suggestions/dismiss', {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ edge_id: edgeID, board_id: state.currentBoardID || 'default' }),
      });
    } catch (_) {}
  }
}

async function promoteSuggestedEdge(edgeID) {
  const edge = edgeByID(edgeID);
  if (!edge) return;
  if (state.mode === 'stateful' && state.backendSession.authenticated) {
    try {
      const res = await fetch('./api/board-links', {
        method: 'POST',
        credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          board_id: state.currentBoardID || 'default',
          from: edge.from_id,
          to: edge.to_id,
          kind: edge.kind || 'relates_to',
          note: evidenceText(edge),
        }),
      });
      if (!res.ok) throw new Error(`${res.status} ${res.statusText}`);
      state.dismissedSuggestionIDs.delete(edgeID);
      await loadBackendBoard();
      await refreshBoards();
      dom.status.textContent = 'suggested relation saved';
      return;
    } catch (err) {
      dom.error.textContent = err.message;
      dom.status.textContent = 'promotion failed';
      return;
    }
  }
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

function nodeByID(nodeID) {
  return (state.data.snapshot.nodes || []).find((node) => node.id === nodeID);
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
    const nd = nodeData(node);
    const hay = [node.id, node.title, node.state, node.kind, badgeText(nodeBadges(node)), labels(node).join(' '), nodePeople(node).map((person) => person.login).join(' '), nd.milestone || '', nd.description || '', nd.owner || '', nd.time_horizon || '', nd.priority || ''].join(' ').toLowerCase();
    if (state.filter && !hay.includes(state.filter)) return false;
    if (state.activeChipFilters.size > 0) {
      const byType = {};
      for (const key of state.activeChipFilters) {
        const colonIdx = key.indexOf(':');
        const type = key.slice(0, colonIdx);
        const value = key.slice(colonIdx + 1);
        if (!byType[type]) byType[type] = new Set();
        byType[type].add(value);
      }
      for (const [type, values] of Object.entries(byType)) {
        if (type === 'kind' && !values.has(node.kind)) return false;
        if (type === 'label' && !labels(node).some((l) => values.has(l))) return false;
        if (type === 'assignee' && !nodePeople(node).some((p) => values.has(p.login))) return false;
        if (type === 'status' && !values.has(node.state)) return false;
        if (type === 'milestone') { const nd2 = nodeData(node); if (!values.has(nd2.milestone || '')) return false; }
        if (type === 'repo') { const gh2 = parseGitHubNodeID(node.id); if (!gh2 || !values.has(gh2.repo)) return false; }
        if (type === 'owner' && !values.has(node.owner || '')) return false;
      }
    }
    return true;
  });
}

function setView(view, options = {}) {
  const next = views.has(view) ? view : 'graph';
  state.view = next;
  document.querySelectorAll('[data-view]').forEach((item) => {
    item.classList.toggle('active', item.dataset.view === next);
  });
  if (options.persist !== false) writeURLView(next);
  if (options.renderNow !== false) render();
}

function readURLView() {
  const view = new URLSearchParams(location.search).get('view') || 'graph';
  return views.has(view) ? view : 'graph';
}

function writeURLView(view) {
  const url = new URL(location.href);
  if (view === 'graph') url.searchParams.delete('view');
  else url.searchParams.set('view', view);
  history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
}

function readURLBoard() {
  return new URLSearchParams(location.search).get('board') || '';
}

function writeURLBoard(boardID) {
  const url = new URL(location.href);
  if (!boardID || boardID === 'default') url.searchParams.delete('board');
  else url.searchParams.set('board', boardID);
  history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
}

function readURLNode() {
  return new URLSearchParams(location.search).get('node') || '';
}

function writeURLNode(nodeID) {
  const url = new URL(location.href);
  if (!nodeID) url.searchParams.delete('node');
  else url.searchParams.set('node', nodeID);
  history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
}

function readURLFilters() {
  const params = new URLSearchParams(location.search);
  const filter = params.get('filter') || '';
  if (filter) {
    state.filter = filter;
    if (dom.filter) dom.filter.value = filter;
  }
  const chips = params.get('chips') || '';
  if (chips) {
    state.activeChipFilters = parseChipFilterParam(chips);
  }
}

function writeURLFilters() {
  const url = new URL(location.href);
  if (state.filter) url.searchParams.set('filter', state.filter);
  else url.searchParams.delete('filter');
  if (state.activeChipFilters.size > 0) url.searchParams.set('chips', formatChipFilterParam(state.activeChipFilters));
  else url.searchParams.delete('chips');
  history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
}

function readURLDriver() {
  const driver = new URLSearchParams(location.search).get('driver') || 'pairs';
  return ['pairs', 'focus', 'backlog', 'cluster'].includes(driver) ? driver : 'pairs';
}

function writeURLDriver(driver) {
  const url = new URL(location.href);
  if (driver === 'pairs') url.searchParams.delete('driver');
  else url.searchParams.set('driver', driver);
  history.replaceState(null, '', `${url.pathname}${url.search}${url.hash}`);
}

function formatChipFilterParam(filters) {
  return Array.from(filters).sort().map(encodeURIComponent).join(',');
}

function parseChipFilterParam(value) {
  return new Set(String(value || '').split(',').filter(Boolean).map((item) => {
    try {
      return decodeURIComponent(item);
    } catch (_) {
      return item;
    }
  }));
}

function shareLink() {
  if (state.mode !== 'stateless') return;
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

function nodeKindLabel(node) {
  if (node.kind === 'pr') return 'PR';
  if (node.kind === 'issue') return 'Issue';
  const strategyKinds = new Set(['strategy', 'initiative', 'bet', 'project', 'workstream', 'risk', 'decision', 'question', 'metric']);
  if (strategyKinds.has(node.kind)) return capitalize(node.kind);
  if (isLocal(node)) return capitalize(node.kind || 'Note');
  return capitalize(node.kind || 'Task');
}

function nodeReferenceLabel(node) {
  const ref = parseGitHubNodeID(node.id);
  if (ref) return `${ref.marker}${ref.number}`;
  const id = String(node.id || '');
  if (id.startsWith('note:') || id.startsWith('task:')) return 'local';
  const strategyPrefixes = ['strategy:', 'initiative:', 'bet:', 'project:', 'workstream:', 'risk:', 'decision:', 'question:', 'metric:'];
  if (strategyPrefixes.some((p) => id.startsWith(p))) return 'local';
  return id.replace(/^gh:/, '').slice(0, 28);
}

function nodeBadges(node) {
  const data = nodeData(node);
  const badges = [];
  const strategyKinds = new Set(['strategy', 'initiative', 'bet', 'project', 'workstream', 'risk', 'decision', 'question', 'metric']);
  if (strategyKinds.has(node.kind)) {
    badges.push(typeBadge(node));
    badges.push(lifecycleBadge(node.state));
    return badges.filter(Boolean);
  }
  if (isLocal(node)) {
    badges.push(badge('type-local', node.kind === 'task' ? '▣ task' : 'note'));
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
  if (kind === 'pr') return badge('type-pr', 'PR');
  if (kind === 'issue') return badge('type-issue', 'issue');
  if (kind === 'note') return badge('type-local', 'note');
  const strategyKinds = new Set(['strategy', 'initiative', 'bet', 'project', 'workstream', 'risk', 'decision', 'question', 'metric']);
  if (strategyKinds.has(kind)) return badge(`type-${kind}`, capitalize(kind));
  return badge('type-task', '▣ task');
}

function lifecycleBadge(value) {
  const lifecycle = String(value || 'unknown').toLowerCase();
  if (lifecycle === 'merged') return badge('life-merged', '🟣 merged');
  if (lifecycle === 'draft') return badge('life-draft', '🚧 draft');
  if (lifecycle === 'active') return badge('life-active', '🟢 active');
  if (lifecycle === 'open') return badge('life-open', '🟢 open');
  if (lifecycle === 'blocked') return badge('life-blocked', '🔴 blocked');
  if (lifecycle === 'at-risk') return badge('life-atrisk', '🟡 at-risk');
  if (lifecycle === 'paused') return badge('life-paused', '⏸ paused');
  if (lifecycle === 'rejected') return badge('life-rejected', '❌ rejected');
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

function nodeSignalsHTML(node) {
  const people = nodePeople(node).slice(0, 5);
  const labelItems = labels(node).slice(0, 5);
  const milestone = nodeData(node).milestone || '';
  const peopleHTML = people.length
    ? `<div class="nodePeople">${people.map((person) => `<img src="${esc(person.avatar_url || githubAvatarURL(person.login))}" alt="@${esc(person.login)}" title="@${esc(person.login)}" loading="lazy">`).join('')}</div>`
    : '';
  const labelsHTML = labelItems.length
    ? `<div class="nodeLabels">${labelItems.map((label) => `<span>${emojiHTML(label)}</span>`).join('')}</div>`
    : '';
  const milestoneHTML = milestone ? `<div class="nodeMilestone">${emojiHTML(milestone)}</div>` : '';
  return peopleHTML || labelsHTML || milestoneHTML ? `<div class="nodeSignals">${peopleHTML}${labelsHTML}${milestoneHTML}</div>` : '';
}

function nodePeople(node) {
  const data = nodeData(node);
  const people = [];
  const add = (person) => {
    const normalized = githubPersonData(person);
    if (!normalized || people.some((item) => item.login === normalized.login)) return;
    people.push(normalized);
  };
  (Array.isArray(data.assignees) ? data.assignees : []).forEach(add);
  (Array.isArray(data.reviewers) ? data.reviewers : []).forEach(add);
  add(data.author);
  add(node.owner);
  return people;
}

function githubAvatarURL(login) {
  return login ? `https://github.com/${encodeURIComponent(login)}.png?size=40` : '';
}

function nodeData(node) {
  try {
    return JSON.parse(node.data_json || '{}');
  } catch {
    return {};
  }
}

function isClosed(node) {
  return ['closed', 'done', 'merged', 'cancelled', 'canceled', 'resolved', 'rejected'].includes(String(node.state || '').toLowerCase());
}

function isLocal(node) {
  const localPrefixes = ['note:', 'task:', 'strategy:', 'initiative:', 'bet:', 'project:',
    'workstream:', 'risk:', 'decision:', 'question:', 'metric:'];
  return node.kind === 'note' || localPrefixes.some((p) => String(node.id || '').startsWith(p));
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

const emojiShortcodes = {
  package: '📦',
  bug: '🐛',
  zap: '⚡',
  sparkles: '✨',
  fire: '🔥',
  warning: '⚠️',
  wrench: '🔧',
  hammer: '🔨',
  construction: '🚧',
  rocket: '🚀',
  memo: '📝',
  book: '📖',
  books: '📚',
  lock: '🔒',
  key: '🔑',
  shield: '🛡️',
  test_tube: '🧪',
  white_check_mark: '✅',
  x: '❌',
  question: '❓',
  bulb: '💡',
  recycle: '♻️',
  art: '🎨',
  lipstick: '💄',
  mag: '🔍',
  chart_with_upwards_trend: '📈',
  hourglass: '⌛',
  tada: '🎉',
  eyes: '👀',
  pray: '🙏',
};

function emojiHTML(value) {
  const text = String(value || '');
  const parts = [];
  let index = 0;
  for (const match of text.matchAll(/:([a-z0-9_+\-]+):/gi)) {
    parts.push(esc(text.slice(index, match.index)));
    const emoji = emojiShortcodes[String(match[1] || '').toLowerCase()];
    parts.push(emoji ? `<span class="emojiShortcode" title="${esc(match[0])}">${emoji}</span>` : esc(match[0]));
    index = match.index + match[0].length;
  }
  parts.push(esc(text.slice(index)));
  return parts.join('');
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

// --- Commit B: undo stack ---

function pushUndo(op) {
  state.undoStack.push(op);
  if (state.undoStack.length > 20) state.undoStack.shift();
}

async function undoLastOp() {
  const op = state.undoStack.pop();
  if (!op) { dom.status.textContent = 'nothing to undo'; return; }
  try {
    if (op.type === 'add-node') {
      await fetch('./api/board-items', {
        method: 'DELETE', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ board_id: state.currentBoardID || 'default', node_id: op.nodeID }),
      });
    } else if (op.type === 'remove-node') {
      const nd = nodeData(op.snapshot || {});
      await fetch('./api/board-items', {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          board_id: state.currentBoardID || 'default',
          kind: op.snapshot.kind || 'task',
          title: op.snapshot.title || '',
          status: op.snapshot.state || '',
          owner: op.snapshot.owner || '',
          description: nd.description || '',
        }),
      });
    } else if (op.type === 'edit-node') {
      await fetch('./api/board-items', {
        method: 'PATCH', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ node_id: op.nodeID, ...op.before }),
      });
    } else if (op.type === 'delete-link') {
      await fetch('./api/board-links', {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          board_id: state.currentBoardID || 'default',
          from: op.snapshot.from_id,
          to: op.snapshot.to_id,
          kind: op.snapshot.kind,
        }),
      });
    } else if (op.type === 'restore-node') {
      await fetch('./api/board-items', {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'restore', node_id: op.nodeID }),
      });
    }
    await loadBackendBoard();
    dom.status.textContent = `undone: ${op.type.replace(/-/g, ' ')}`;
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'undo failed';
  }
}

// --- Commit C: keyboard helpers ---

function inputFocused() {
  const t = document.activeElement?.tagName?.toLowerCase();
  return t === 'input' || t === 'textarea' || t === 'select' || Boolean(document.activeElement?.isContentEditable);
}

// --- Commit D: duplicate node ---

async function duplicateNode(nodeID) {
  try {
    const res = await fetch('./api/board-items', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ action: 'duplicate', board_id: state.currentBoardID || 'default', node_id: nodeID }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    const payload = await res.json();
    const newID = payload.node?.id || '';
    if (newID) pushUndo({ type: 'add-node', nodeID: newID });
    await loadBackendBoard();
    dom.status.textContent = 'node duplicated';
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'duplicate failed';
  }
}

// --- Commit E: bulk import ---

async function handleBulkImport(e) {
  e.preventDefault();
  const raw = dom.bulkImportText ? dom.bulkImportText.value.trim() : '';
  if (!raw) return;
  const defaultKind = dom.bulkImportKind ? dom.bulkImportKind.value || 'task' : 'task';
  const lines = raw.split('\n').map((l) => l.trim()).filter(Boolean);
  const tasks = lines.map((line) => {
    let kind = defaultKind;
    let title = line;
    let done = false;
    const checkboxMatch = line.match(/^[-*]\s+\[([x ])\]\s+(.+)$/i);
    if (checkboxMatch) {
      done = checkboxMatch[1].toLowerCase() === 'x';
      title = checkboxMatch[2];
    } else {
      title = line.replace(/^[-*#•]\s+/, '');
    }
    const kindPrefixMatch = title.match(/^(strategy|initiative|bet|project|workstream|risk|decision|question|metric|task|note):\s+(.+)$/i);
    if (kindPrefixMatch) {
      kind = kindPrefixMatch[1].toLowerCase();
      title = kindPrefixMatch[2];
    }
    return { kind, title, status: done ? 'done' : '' };
  });
  const boardID = state.currentBoardID || 'default';
  let created = 0;
  for (const t of tasks) {
    try {
      const res = await fetch('./api/board-items', {
        method: 'POST', credentials: 'same-origin',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ board_id: boardID, kind: t.kind, title: t.title, status: t.status }),
      });
      if (res.ok) created++;
    } catch (_) {}
  }
  if (dom.bulkImportText) dom.bulkImportText.value = '';
  await loadBackendBoard();
  dom.status.textContent = `${created} node${created !== 1 ? 's' : ''} imported`;
}

// --- Commit F: kanban view ---

function renderKanbanView() {
  const el = document.getElementById('kanbanView');
  if (!el) return;
  const lanes = [
    { status: 'draft', label: '🚧 Draft' },
    { status: 'active', label: '🟢 Active' },
    { status: 'open', label: '🟢 Open' },
    { status: 'blocked', label: '🔴 Blocked' },
    { status: 'at-risk', label: '🟡 At-risk' },
    { status: 'paused', label: '⏸ Paused' },
    { status: 'local', label: '📝 Local' },
    { status: 'done', label: '⚫ Done' },
    { status: 'closed', label: '⚫ Closed' },
    { status: 'rejected', label: '❌ Rejected' },
  ];
  const nodes = visibleNodes(state.data.snapshot.nodes);
  if (!nodes.length) {
    el.innerHTML = '<div class="emptyState"><p>No items to show.</p></div>';
    return;
  }
  const byStatus = {};
  for (const n of nodes) {
    const s = (n.state || 'local').toLowerCase();
    if (!byStatus[s]) byStatus[s] = [];
    byStatus[s].push(n);
  }
  const laneStatuses = new Set(lanes.map((l) => l.status));
  for (const s of Object.keys(byStatus)) {
    if (!laneStatuses.has(s)) lanes.push({ status: s, label: capitalize(s) });
  }
  const activeLanes = lanes.filter((l) => byStatus[l.status]?.length > 0);
  if (!activeLanes.length) {
    el.innerHTML = '<div class="emptyState"><p>No items to show.</p></div>';
    return;
  }
  el.innerHTML = `<div class="kanbanBoard">${activeLanes.map((lane) => {
    const cards = byStatus[lane.status] || [];
    return `<div class="kanbanLane">
      <div class="kanbanLaneHead"><strong>${esc(lane.label)}</strong><span class="kanbanCount">${cards.length}</span></div>
      <div class="kanbanCards">${cards.map((n) => `
        <div class="kanbanCard${state.selectedNodeID === n.id ? ' selected' : ''}" data-node-id="${esc(n.id)}">
          <div class="kanbanCardKind">${esc(nodeKindLabel(n))}</div>
          <div class="kanbanCardTitle">${emojiHTML(n.title || n.id)}</div>
          ${n.owner ? `<div class="kanbanCardOwner">@${esc(n.owner)}</div>` : ''}
        </div>`).join('')}
      </div>
    </div>`;
  }).join('')}</div>`;
  el.querySelectorAll('[data-node-id]').forEach((card) => {
    card.addEventListener('click', () => {
      state.selectedNodeID = card.dataset.nodeId;
      render();
    });
  });
}

// --- Commit I: date formatting ---

function formatDate(v) {
  if (!v) return '';
  try {
    const d = new Date(v);
    if (isNaN(d.getTime())) return '';
    return d.toLocaleDateString('en', { month: 'short', day: 'numeric', year: 'numeric' });
  } catch { return String(v); }
}

// --- Commit J: load archived nodes ---

async function loadArchivedNodes() {
  const container = document.getElementById('archivedNodesList');
  if (!container) return;
  try {
    const board = encodeURIComponent(state.currentBoardID || 'default');
    const res = await fetch(`./api/board-items?board_id=${board}&archived=true`, { credentials: 'same-origin' });
    if (!res.ok) return;
    const payload = await res.json();
    const nodes = Array.isArray(payload.nodes) ? payload.nodes : [];
    if (!nodes.length) {
      container.innerHTML = '<div class="emptyState" style="padding:8px">No archived nodes</div>';
      return;
    }
    container.innerHTML = nodes.map((n) => `<div class="archivedNode">
      <span>${esc(n.title || n.id)}</span>
      <button type="button" data-restore-id="${esc(n.id)}">Restore</button>
    </div>`).join('');
    container.querySelectorAll('[data-restore-id]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        const res2 = await fetch('./api/board-items', {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ action: 'restore', node_id: btn.dataset.restoreId }),
        });
        if (res2.ok) {
          await loadBackendBoard();
          dom.status.textContent = 'node restored';
        }
      });
    });
  } catch (_) {}
}

async function loadSavedViews() {
  const el = document.getElementById('savedViewsList');
  if (!el || !state.backendSession.authenticated) return;
  try {
    const board = encodeURIComponent(state.currentBoardID || 'default');
    const res = await fetch(`./api/board-views?board_id=${board}`, { credentials: 'same-origin' });
    if (!res.ok) return;
    const data = await res.json();
    const views = Array.isArray(data.views) ? data.views : [];
    if (!views.length) { el.innerHTML = '<div class="emptyState">No saved filters yet</div>'; return; }
    el.innerHTML = views.map((v) => {
      let cfg = {};
      try { cfg = JSON.parse(v.config_json || '{}'); } catch (_) {}
      const desc = [cfg.driver, cfg.view, cfg.filter_text].filter(Boolean).join(' · ');
      return `<div class="savedViewItem">
        <div>
          <strong>${esc(v.name)}</strong>
          ${desc ? `<span class="savedViewDesc">${esc(desc)}</span>` : ''}
        </div>
        <div class="savedViewActions">
          <button type="button" data-apply-view-id="${esc(v.id)}" data-apply-view-config="${esc(v.config_json)}">Apply</button>
          <button type="button" data-delete-view-id="${esc(v.id)}">Delete</button>
        </div>
      </div>`;
    }).join('');
    el.querySelectorAll('[data-apply-view-id]').forEach((btn) => {
      btn.addEventListener('click', () => {
        try {
          const cfg = JSON.parse(btn.dataset.applyViewConfig || '{}');
          if (cfg.driver) { state.graphDriver = cfg.driver; if (dom.graphDriver) dom.graphDriver.value = cfg.driver; }
          if (cfg.view) setView(cfg.view, { persist: true, renderNow: false });
          if (cfg.filter_text !== undefined) { state.filter = cfg.filter_text; if (dom.filter) dom.filter.value = cfg.filter_text; }
          render();
          const nameEl = btn.closest('.savedViewItem')?.querySelector('strong');
          dom.status.textContent = `view "${nameEl ? nameEl.textContent : cfg.driver || ''}" applied`;
        } catch (_) {}
      });
    });
    el.querySelectorAll('[data-delete-view-id]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        await fetch(`./api/board-views?id=${encodeURIComponent(btn.dataset.deleteViewId)}`, { method: 'DELETE', credentials: 'same-origin' });
        loadSavedViews();
      });
    });
  } catch (_) {}
}

async function saveCurrentView(name) {
  const config = {
    driver: state.graphDriver,
    view: state.view,
    filter_text: state.filter,
    active_chips: Array.from(state.activeChipFilters),
    show_closed: state.showClosed,
    show_external: state.showExternal,
    show_local: state.showLocal,
  };
  try {
    const res = await fetch('./api/board-views', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ board_id: state.currentBoardID || 'default', name, config }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    dom.status.textContent = `view "${name}" saved`;
    loadSavedViews();
  } catch (err) {
    dom.error.textContent = err.message;
  }
}

function resolveNodeByRef(ref) {
  const nodes = state.data.snapshot.nodes || [];
  const trimmed = ref.trim();
  const byID = nodes.find((n) => n.id === trimmed);
  if (byID) return byID;
  const numMatch = /^#?(\d+)$/.exec(trimmed);
  if (numMatch) return nodes.find((n) => n.external_id === `#${numMatch[1]}` || n.external_id === numMatch[1]);
  const lower = trimmed.toLowerCase();
  return nodes.find((n) => (n.title || '').toLowerCase().includes(lower));
}

async function addBoardLinkDirect(from, kind, to) {
  try {
    const res = await fetch('./api/board-links', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ board_id: state.currentBoardID || 'default', from, to, kind }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    await loadBackendBoard();
    dom.status.textContent = 'link added';
  } catch (err) {
    dom.error.textContent = err.message;
  }
}

async function createGitHubIssueFromNode(nodeID, repo, title, body, labels = [], assignees = [], archiveLocal = false) {
	try {
		const res = await fetch('./api/github/create-issue', {
			method: 'POST', credentials: 'same-origin',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ board_id: state.currentBoardID || 'default', node_id: nodeID, repo, title, body, labels, assignees, archive_local: archiveLocal }),
		});
		if (!res.ok) throw new Error(await responseErrorMessage(res));
		const data = await res.json();
		dom.status.textContent = `GitHub issue #${data.number} created`;
		await loadBackendBoard();
		await refreshBoards();
		if (data.url) window.open(data.url, '_blank', 'noreferrer');
	} catch (err) {
		dom.error.textContent = err.message;
	}
}

async function closeOrReopenGitHubIssue(repo, issueNumber, issueState) {
  try {
    const res = await fetch('./api/github/update-issue', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repo, issue_number: issueNumber, state: issueState }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    dom.status.textContent = issueState === 'closed' ? 'issue closed' : 'issue reopened';
    await loadBackendBoard();
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'github update failed';
  }
}

async function submitGitHubComment(repo, issueNumber, body) {
  try {
    const res = await fetch('./api/github/comment', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ repo, issue_number: issueNumber, body }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    const data = await res.json();
    dom.status.textContent = 'comment posted';
    if (data.url) {
      const el = document.getElementById('inspectorCommentBody');
      if (el) el.value = '';
    }
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'comment failed';
  }
}

function setSyncIndicator(status) {
  state.syncIndicator = status;
  const el = document.getElementById('syncIndicator');
  if (!el) return;
  el.className = `syncIndicator sync${capitalize(status)}`;
  if (status === 'done' || status === 'failed') {
    setTimeout(() => setSyncIndicator('idle'), 3000);
  }
}

function paletteCommands() {
  return [
    { id: 'new-item', label: 'New item', hint: 'Create a new node', action: () => { setWorkspaceTab('actions'); setTimeout(() => (dom.newItemTitle || dom.newItemRef)?.focus(), 50); } },
    { id: 'new-link', label: 'New link', hint: 'Create a new link', action: () => { setWorkspaceTab('actions'); setTimeout(() => dom.newLinkFrom?.focus(), 50); } },
    { id: 'view-graph', label: 'Switch to Relations view', action: () => setView('graph') },
    { id: 'view-brief', label: 'Switch to Overview', action: () => setView('brief') },
    { id: 'view-table', label: 'Switch to Table', action: () => setView('table') },
    { id: 'view-kanban', label: 'Switch to Board', action: () => setView('kanban') },
    { id: 'sync', label: 'Sync board', hint: 'Re-sync from GitHub', action: () => syncCurrentBoard() },
    { id: 'export', label: 'Export JSON', action: () => exportJSON() },
    { id: 'search', label: 'Search / filter', action: () => dom.filter?.focus() },
    ...(state.boards || []).map((b) => ({
      id: 'board-' + b.id,
      label: 'Switch to: ' + (b.name || b.id),
      hint: b.scope_query || '',
      action: () => { state.currentBoardID = b.id; writeURLBoard(b.id); loadBackendBoard(); },
    })),
  ];
}

function openPalette() {
  state.paletteOpen = true;
  state.paletteQuery = '';
  state.paletteSelected = 0;
  document.getElementById('commandPalette')?.classList.remove('hidden');
  const input = document.getElementById('paletteInput');
  if (input) { input.value = ''; input.focus(); }
  renderPalette();
}

function closePalette() {
  state.paletteOpen = false;
  document.getElementById('commandPalette')?.classList.add('hidden');
}

function renderPalette() {
  const q = (state.paletteQuery || '').toLowerCase();
  const cmds = paletteCommands().filter((c) =>
    !q || c.label.toLowerCase().includes(q) || (c.hint || '').toLowerCase().includes(q)
  );
  const el = document.getElementById('paletteResults');
  if (!el) return;
  const visible = cmds.slice(0, 12);
  el.innerHTML = visible.map((c, i) => `
    <div class="paletteItem ${i === state.paletteSelected ? 'selected' : ''}" data-palette-index="${i}" role="option">
      <span class="paletteLabel">${esc(c.label)}</span>
      ${c.hint ? `<span class="paletteHint">${esc(c.hint)}</span>` : ''}
    </div>`).join('');
  el.querySelectorAll('[data-palette-index]').forEach((item) => {
    item.addEventListener('click', () => {
      const idx = Number(item.dataset.paletteIndex);
      if (visible[idx]) { visible[idx].action(); closePalette(); }
    });
  });
}

function computeSourceDiff(base, current) {
  const baseNodes = new Set((base?.snapshot?.nodes || []).map((n) => n.id));
  const currNodes = new Set((current?.snapshot?.nodes || []).map((n) => n.id));
  const added = [...currNodes].filter((id) => !baseNodes.has(id)).length;
  const removed = [...baseNodes].filter((id) => !currNodes.has(id)).length;
  const baseEdges = (base?.snapshot?.edges || []).length;
  const currEdges = (current?.snapshot?.edges || []).length;
  const edgeDiff = currEdges - baseEdges;
  return { added, removed, edgeDiff };
}
function renderSourceDirtyIndicator() {
  const el = document.getElementById('sourcePreviewMeta');
  if (!el) return;
  if (!state.sourceDirty) { el.classList.add('hidden'); return; }
  el.classList.remove('hidden');
  const diff = state.sourceSnapshot ? computeSourceDiff(state.sourceSnapshot, state.data) : null;
  const diffText = diff ? `+${diff.added} nodes  -${diff.removed}  ${diff.edgeDiff >= 0 ? '+' : ''}${diff.edgeDiff} edges` : '';
  const summaryEl = document.getElementById('sourceDiffSummary');
  if (summaryEl) summaryEl.textContent = diffText;
}

function computeSourcePatch(baseNodes, baseEdges, editedNodes, editedEdges) {
  const baseNodeMap = new Map((baseNodes || []).map((n) => [n.id, n]));
  const editedNodeMap = new Map((editedNodes || []).map((n) => [n.id, n]));
  const baseEdgeMap = new Map((baseEdges || []).map((e) => [edgeSelectionID(e), e]));
  const editedEdgeMap = new Map((editedEdges || []).map((e) => [edgeSelectionID(e), e]));
  const creates = [];
  const updates = [];
  const deletes = [];
  const link_creates = [];
  const link_deletes = [];
  for (const [id, node] of editedNodeMap) {
    if (!baseNodeMap.has(id)) {
      if (isLocal(node)) {
        creates.push({ kind: node.kind, title: node.title, status: node.state, owner: node.owner, description: nodeData(node).description || '', time_horizon: nodeData(node).time_horizon || '', priority: nodeData(node).priority || '', labels: nodeData(node).labels || [] });
      }
    } else {
      const base = baseNodeMap.get(id);
      if (isLocal(node) && (node.title !== base.title || node.state !== base.state || node.owner !== base.owner)) {
        updates.push({ node_id: id, title: node.title, status: node.state, owner: node.owner, description: nodeData(node).description || '' });
      }
    }
  }
  for (const [id, node] of baseNodeMap) {
    if (!editedNodeMap.has(id) && isLocal(node)) {
      deletes.push({ node_id: id });
    }
  }
  for (const [id, edge] of editedEdgeMap) {
    if (!baseEdgeMap.has(id)) {
      link_creates.push({ from_id: edge.from_id, to_id: edge.to_id, kind: edge.kind, notes: '' });
    }
  }
  for (const [id, edge] of baseEdgeMap) {
    if (!editedEdgeMap.has(id) && (edge.authority === 'local' || edge.authority === 'user')) {
      link_deletes.push({ edge_id: id });
    }
  }
  return { creates, updates, deletes, link_creates, link_deletes };
}

function renderSourcePatchModal(patch, onApply) {
  const existing = document.getElementById('sourcePatchModal');
  if (existing) existing.remove();
  const modal = document.createElement('div');
  modal.id = 'sourcePatchModal';
  modal.className = 'patchModal';
  const totalOps = patch.creates.length + patch.updates.length + patch.deletes.length + patch.link_creates.length + patch.link_deletes.length;
  modal.innerHTML = `<div class="patchBackdrop"></div>
  <div class="patchBox">
    <h3>Preview changes</h3>
    <div class="patchSummary">
      ${patch.creates.length ? `<div class="patchSection creates">Creates: ${patch.creates.length} item${patch.creates.length !== 1 ? 's' : ''}</div>` : ''}
      ${patch.updates.length ? `<div class="patchSection updates">Updates: ${patch.updates.length} item${patch.updates.length !== 1 ? 's' : ''}</div>` : ''}
      ${patch.deletes.length ? `<div class="patchSection deletes danger">Removes: ${patch.deletes.length} item${patch.deletes.length !== 1 ? 's' : ''}</div>` : ''}
      ${patch.link_creates.length ? `<div class="patchSection link_creates">Links added: ${patch.link_creates.length}</div>` : ''}
      ${patch.link_deletes.length ? `<div class="patchSection link_deletes danger">Links removed: ${patch.link_deletes.length}</div>` : ''}
      ${totalOps === 0 ? '<div class="patchSection">No changes detected</div>' : ''}
    </div>
    <div class="patchDetails">
      ${patch.creates.map((c) => `<div>+ ${esc(c.kind)}: ${esc(c.title)}</div>`).join('')}
      ${patch.updates.map((u) => `<div>~ ${esc(u.node_id)}: ${esc(u.title)}</div>`).join('')}
      ${patch.deletes.map((d) => `<div>- ${esc(d.node_id)}</div>`).join('')}
      ${patch.link_creates.map((l) => `<div>+ link: ${esc(l.from_id)} → ${esc(l.to_id)}</div>`).join('')}
      ${patch.link_deletes.map((l) => `<div>- link: ${esc(l.edge_id)}</div>`).join('')}
    </div>
    <div class="patchActions">
      <button class="primaryAction" id="confirmApplyBtn">Apply</button>
      <button id="cancelApplyBtn">Cancel</button>
    </div>
  </div>`;
  document.body.appendChild(modal);
  document.getElementById('cancelApplyBtn').addEventListener('click', () => modal.remove());
  modal.querySelector('.patchBackdrop').addEventListener('click', () => modal.remove());
  document.getElementById('confirmApplyBtn').addEventListener('click', () => {
    modal.remove();
    if (onApply) onApply();
  });
}

async function applySourcePatch() {
  if (!state.sourceDirty || state.mode !== 'stateful') return;
  const baseSnap = state.sourceSnapshot?.snapshot || { nodes: [], edges: [] };
  const currSnap = state.data?.snapshot || { nodes: [], edges: [] };
  const patch = computeSourcePatch(baseSnap.nodes, baseSnap.edges, currSnap.nodes, currSnap.edges);
  const boardID = state.currentBoardID || 'default';
  try {
    const res = await fetch('./api/board-source/apply', {
      method: 'POST', credentials: 'same-origin',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ board_id: boardID, dry_run: true, ...patch }),
    });
    if (!res.ok) throw new Error(await responseErrorMessage(res));
    renderSourcePatchModal(patch, async () => {
      try {
        const res2 = await fetch('./api/board-source/apply', {
          method: 'POST', credentials: 'same-origin',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ board_id: boardID, dry_run: false, ...patch }),
        });
        if (!res2.ok) throw new Error(await responseErrorMessage(res2));
        state.sourceDirty = false;
        await loadBackendBoard();
        dom.status.textContent = 'changes applied';
      } catch (err) {
        dom.error.textContent = err.message;
        dom.status.textContent = 'apply failed';
      }
    });
  } catch (err) {
    dom.error.textContent = err.message;
    dom.status.textContent = 'preview failed';
  }
}

boot();
