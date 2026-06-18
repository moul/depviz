package live

import (
	"io/fs"
	"strings"
	"testing"
)

func TestAppFSContainsLiveAssets(t *testing.T) {
	fsys := AppFS()
	for _, name := range []string{"index.html", "style.css", "app.js", "sample.depviz", "sample.events.jsonl"} {
		info, err := fs.Stat(fsys, name)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
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
		{"scale wrapper", string(style), `.graphScale`},
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
		{"refresh failures", string(app), `state.githubFailures`},
		{"placeholder guidance", string(app), `refresh GitHub or sync/export a wider scope`},
		{"partial metadata", string(app), `partial GitHub metadata`},
	} {
		if !strings.Contains(tc.body, tc.want) {
			t.Fatalf("%s: missing %q", tc.name, tc.want)
		}
	}
}
