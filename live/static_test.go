package live

import (
	"io/fs"
	"testing"
)

func TestAppFSContainsLiveAssets(t *testing.T) {
	fsys := AppFS()
	for _, name := range []string{"index.html", "style.css", "app.js", "sample.events.jsonl"} {
		info, err := fs.Stat(fsys, name)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
}
