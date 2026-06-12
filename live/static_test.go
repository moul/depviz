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
