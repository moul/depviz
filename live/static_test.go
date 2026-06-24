package live

import (
	"io/fs"
	"os"
	"strings"
	"testing"
)

func TestAppFSContainsLiveAssets(t *testing.T) {
	fsys := AppFS()
	for _, name := range []string{"index.html", "style.css", "app.js", "sample.depviz", "sample.events.jsonl", "logo.svg", "logo-512.png", "favicon.svg", "favicon.png"} {
		info, err := fs.Stat(fsys, name)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
}

func TestLiveAssetsExposeDepVizBranding(t *testing.T) {
	fsys := AppFS()
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	style, err := fs.ReadFile(fsys, "style.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"svg favicon", string(index), `href="./favicon.svg"`},
		{"png favicon", string(index), `href="./favicon.png"`},
		{"touch icon", string(index), `href="./logo-512.png"`},
		{"brand logo", string(index), `src="./logo.svg"`},
		{"emoji fallback", string(index), `alt="🔗"`},
		{"brand style", string(style), `.brandLogo`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeSuggestedRelationsUI(t *testing.T) {
	fsys := AppFS()
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := fs.ReadFile(fsys, "style.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"panel mount", string(index), `id="suggestionPanel"`},
		{"promote action", string(app), `data-suggestion-action="promote"`},
		{"local promotion", string(app), `suggested relation promoted locally`},
		{"graph highlight", string(style), `.graphEdge.selectedEdge`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeRelationAwareGraphLayout(t *testing.T) {
	fsys := AppFS()
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"layout helper", string(app), `function graphLayout(snapshot, nodes)`},
		{"rank helper", string(app), `function graphRanks(nodes, edges)`},
		{"isolated pool", string(app), `const isolatedNodes = nodes.filter`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeEdgeInspectorWorkflow(t *testing.T) {
	fsys := AppFS()
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := fs.ReadFile(fsys, "style.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"inspector mount", string(index), `id="edgeInspector"`},
		{"edge hit target", string(app), `class="graphEdgeHit"`},
		{"edge locate", string(app), `data-edge-action="locate"`},
		{"edge locate scroll", string(app), `function scrollSelectedEdgeIntoView()`},
		{"inspector promote", string(app), `data-edge-action="promote"`},
		{"edge actions", string(style), `.edgeActions`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeGraphZoomControls(t *testing.T) {
	fsys := AppFS()
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := fs.ReadFile(fsys, "style.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"toolbar", string(index), `class="graphToolbar"`},
		{"fit action", string(index), `data-graph-action="fit"`},
		{"zoom helper", string(app), `function graphZoom(`},
		{"fit helper", string(app), `function fitGraphToCanvas()`},
		{"keyboard helper", string(app), `function handleGraphKeydown(event)`},
		{"typing guard", string(app), `function isTypingTarget(target)`},
		{"scale wrapper", string(style), `.graphScale`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeBackendSessionUI(t *testing.T) {
	fsys := AppFS()
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"login button", string(index), `id="backendGithubLoginBtn"`},
		{"logout button", string(index), `id="backendLogoutBtn"`},
		{"session fetch", string(app), `function refreshBackendSession()`},
		{"github start", string(app), `./api/auth/github/start`},
		{"logout api", string(app), `./api/auth/logout`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeStatefulMode(t *testing.T) {
	fsys := AppFS()
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := fs.ReadFile(fsys, "style.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"stateful button", string(index), `data-mode="stateful"`},
		{"stateless button", string(index), `data-mode="stateless"`},
		{"backend export", string(app), `./api/export?board=${board}`},
		{"mode setter", string(app), `async function setMode(`},
		{"user button", string(index), `id="settingsBtn"`},
		{"workspace panel", string(index), `id="workspacePanel"`},
		{"workspace tabs", string(index), `data-workspace-tab="views"`},
		{"suggestions tab", string(index), `data-workspace-tab="suggestions"`},
		{"boards endpoint", string(app), `./api/boards`},
		{"board items endpoint", string(app), `./api/board-items`},
		{"board links endpoint", string(app), `./api/board-links`},
		{"board sync endpoint", string(app), `./api/board-sync`},
		{"board metrics", string(app), `boardMetrics`},
		{"freshness labels", string(app), `freshnessLabel`},
		{"board url persistence", string(app), `writeURLBoard`},
		{"board sorting", string(app), `sortBoards`},
		{"board scope", string(app), `boardScope`},
		{"draft grouping", string(app), `draftGroup`},
		{"structured stats", string(app), `<span>${label}</span><strong>${value}</strong>`},
		{"empty board render", string(app), `renderEmptyBoardBrief`},
		{"empty board action", string(app), `data-empty-action="add-item"`},
		{"empty board style", string(style), `.emptyBoardPanel`},
		{"stateful top aligned", string(style), `align-content: start`},
		{"draft board style", string(style), `.draftBoard`},
		{"draft group style", string(style), `.draftGroup`},
		{"visual shadow token", string(style), `--shadow`},
		{"graph column headers", string(app), `graphColumnHeaders`},
		{"graph column style", string(style), `.graphColumnHeader`},
		{"graph focus panel", string(index), `id="graphFocusPanel"`},
		{"graph focus render", string(app), `renderGraphFocusPanel`},
		{"graph focus style", string(style), `.graphFocusBox`},
		{"graph focus edges", string(app), `focusEdge`},
		{"graph overview selection", string(app), `graphVisibleNodeSelection`},
		{"graph driver dropdown", string(index), `id="graphDriverSelect"`},
		{"relation pairs driver", string(app), `renderGraphPairs`},
		{"relation pair grouping", string(app), `graphPairGroups`},
		{"relation pairs style", string(style), `.graphPairGroup`},
		{"focus driver", string(app), `renderGraphFocusDriver`},
		{"backlog driver", string(app), `renderGraphBacklog`},
		{"focus driver style", string(style), `.graphFocusDriver`},
		{"backlog driver style", string(style), `.graphBacklog`},
		{"node visual signals", string(app), `nodeSignalsHTML`},
		{"node avatar style", string(style), `.nodePeople`},
		{"emoji shortcode rendering", string(app), `emojiShortcodes`},
		{"package shortcode", string(app), `package: '📦'`},
		{"emoji shortcode style", string(style), `.emojiShortcode`},
		{"connected toggle", string(index), `id="graphConnectedToggle"`},
		{"connected toggle action", string(app), `toggle-connected`},
		{"unlinked toggle", string(index), `id="graphUnlinkedToggle"`},
		{"hidden graph summary", string(style), `.graphHiddenSummary`},
		{"compact graph suggestions", string(app), `data-suggestion-action="review-all"`},
		{"node card top row", string(app), `nodeTop`},
		{"item inspector", string(index), `id="itemInspector"`},
		{"item inspector render", string(app), `function renderItemInspector`},
		{"stateful suggestion promotion", string(app), `suggested relation saved`},
		{"status filter chips", string(app), `statusCounts`},
		{"chip url encoding", string(app), `formatChipFilterParam`},
		{"graph driver whitelist", string(app), `['pairs', 'focus', 'backlog', 'cluster']`},
		{"github presets", string(index), `id="githubPresetList"`},
		{"add item form", string(index), `id="addBoardItemForm"`},
		{"add link form", string(index), `id="addBoardLinkForm"`},
		{"stateful layout", string(style), `.shell.statefulMode`},
		{"workspace layout", string(style), `.workspacePanel`},
		{"stateful source reset", string(index), `id="resetSourceBtn"`},
		{"stateful source preview", string(app), `updateStatefulSourcePreview`},
		{"stateful source generator", string(app), `snapshotToFlow`},
		{"stateful source dirty style", string(style), `.shell.sourceDirty .editorWrap`},
		{"stateful source stays visible", string(style), `.shell.statefulMode .inputPane`},
		{"stateful auth gate", string(app), `renderStatefulSignedOut`},
		{"stateful auth action", string(app), `data-auth-action="signin"`},
		{"stateful auth style", string(style), `.authGate`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeUXBacklog709Features(t *testing.T) {
	fsys := AppFS()
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := fs.ReadFile(fsys, "style.css")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"command palette mount", string(index), `id="commandPalette"`},
		{"palette input", string(index), `id="paletteInput"`},
		{"board filter input", string(index), `id="boardFilterInput"`},
		{"sync indicator", string(index), `id="syncIndicator"`},
		{"auth gate panel", string(index), `id="authGatePanel"`},
		{"onboarding panel", string(index), `id="onboardingPanel"`},
		{"link mode indicator", string(index), `id="linkModeIndicator"`},
		{"command palette style", string(style), `.commandPalette`},
		{"auth gate style", string(style), `.authGate`},
		{"onboarding checklist style", string(style), `.onboardingChecklist`},
		{"sync indicator style", string(style), `.syncIndicator`},
		{"open palette func", string(app), `function openPalette()`},
		{"close palette func", string(app), `function closePalette()`},
		{"render palette func", string(app), `function renderPalette()`},
		{"palette commands func", string(app), `function paletteCommands()`},
		{"auth gate render", string(app), `function renderAuthGate()`},
		{"onboarding render", string(app), `function renderOnboardingChecklist()`},
		{"sync indicator func", string(app), `function setSyncIndicator(`},
		{"source dirty indicator", string(app), `function renderSourceDirtyIndicator()`},
		{"github issue create", string(app), `function createGitHubIssueFromNode(`},
		{"read attribute func", string(app), `function readAttribute(`},
		{"resolve node by ref", string(app), `function resolveNodeByRef(`},
		{"add board link direct", string(app), `function addBoardLinkDirect(`},
		{"write url node", string(app), `function writeURLNode(`},
		{"compute source diff", string(app), `function computeSourceDiff(`},
		{"graph driver hint", string(style), `.graphDriverHint`},
		{"inspector link group", string(style), `.inspectorLinkGroup`},
		{"link target style", string(style), `.linkTarget`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeGitHubDiagnostics(t *testing.T) {
	fsys := AppFS()
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"diagnostics section", string(app), `GitHub diagnostics`},
		{"api error detail", string(app), `responseErrorMessage`},
		{"sync error json", string(app), `data.error || data.message`},
		{"refresh failures", string(app), `state.githubFailures`},
		{"placeholder guidance", string(app), `refresh GitHub or sync/export a wider scope`},
		{"partial metadata", string(app), `partial GitHub metadata`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeHardening713Features(t *testing.T) {
	fsys := AppFS()
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	index, err := fs.ReadFile(fsys, "index.html")
	if err != nil {
		t.Fatal(err)
	}
	storeGo, err := os.ReadFile("../internal/core/store.go")
	if err != nil {
		t.Fatal(err)
	}
	serverGo, err := os.ReadFile("../internal/backend/server.go")
	if err != nil {
		t.Fatal(err)
	}
	mainGo, err := os.ReadFile("../cmd/depviz/main.go")
	if err != nil {
		t.Fatal(err)
	}
	docsFile, err := os.ReadFile("../docs/meta-repo-strategy.md")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"WithTx helper", string(storeGo), `func (s *Store) WithTx(`},
		{"BoardUpdatedAt helper", string(storeGo), `func (s *Store) BoardUpdatedAt(`},
		{"workspaces route", string(serverGo), `/api/workspaces`},
		{"base_updated_at in server", string(serverGo), `base_updated_at`},
		{"schema_migrations table", string(storeGo), `schema_migrations`},
		{"requireBoardAccess helper", string(serverGo), `func (s *Server) requireBoardAccess(`},
		{"restore command", string(mainGo), `case "restore"`},
		{"timedRender in app", string(app), `function timedRender(`},
		{"renderTimings in app", string(app), `renderTimings`},
		{"workspaceSwitcher in index", string(index), `workspaceSwitcher`},
		{"meta-repo docs exist", string(docsFile), `Meta-Repo Strategy`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}

func TestLiveAssetsExposeCockpitNext711Features(t *testing.T) {
	fsys := AppFS()
	app, err := fs.ReadFile(fsys, "app.js")
	if err != nil {
		t.Fatal(err)
	}
	style, err := fs.ReadFile(fsys, "style.css")
	if err != nil {
		t.Fatal(err)
	}
	serverGo, err := os.ReadFile("../internal/backend/server.go")
	if err != nil {
		t.Fatal(err)
	}
	mainGo, err := os.ReadFile("../cmd/depviz/main.go")
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name string
		body string
		want string
	}{
		{"source-apply route", string(serverGo), `/api/board-source/apply`},
		{"update-issue route", string(serverGo), `/api/github/update-issue`},
		{"comment route", string(serverGo), `/api/github/comment`},
		{"dismiss route", string(serverGo), `/api/suggestions/dismiss`},
		{"board-views route", string(serverGo), `/api/board-views`},
		{"patchModal css", string(style), `.patchModal`},
		{"clusterGroup css", string(style), `.clusterGroup`},
		{"savedViews css", string(style), `.savedViews`},
		{"backup command", string(mainGo), `case "backup"`},
		{"handleDismissSuggestion", string(serverGo), `handleDismissSuggestion`},
		{"renderSourcePatchModal", string(app), `function renderSourcePatchModal(`},
		{"computeSourcePatch", string(app), `function computeSourcePatch(`},
		{"handleCreateGitHubComment", string(serverGo), `handleCreateGitHubComment`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}
