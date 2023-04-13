package dvcore

import (
	"bytes"
	"os"
	"testing"

	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/quad"
	"github.com/stretchr/testify/assert"
	"moul.io/depviz/v3/pkg/dvstore"
	"moul.io/depviz/v3/pkg/testutil"
	"moul.io/multipmuri"
)

func TestPullAndSave(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test (--short)")
	}
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		t.Skip("missing GITHUB_TOKEN")
	}
	schema := dvstore.Schema()
	logger := testutil.Logger(t)

	tests := []struct {
		name    string
		targets []multipmuri.Entity
	}{
		{
			"moul-depviz-test",
			[]multipmuri.Entity{
				multipmuri.NewGitHubRepo("github.com", "moul", "depviz-test"),
			},
		},
		{
			"moulbot-depviz-test",
			[]multipmuri.Entity{
				multipmuri.NewGitHubRepo("github.com", "moul-bot", "depviz-test"),
			},
		},
		{
			"all-depviz-test",
			[]multipmuri.Entity{
				multipmuri.NewGitHubRepo("github.com", "moul", "depviz-test"),
				multipmuri.NewGitHubRepo("github.com", "moul-bot", "depviz-test"),
			},
		},
	}

	for _, test := range tests {
		store, close := dvstore.TestingStore(t)
		defer close()
		changed, err := PullAndSave(test.targets, store, schema, githubToken, false, logger)
		assert.NoError(t, err, test.name)
		assert.True(t, changed, test.name)
		changed, err = PullAndSave(test.targets, store, schema, githubToken, false, logger)
		assert.NoError(t, err, test.name)
		assert.False(t, changed, test.name)
		changed, err = PullAndSave(test.targets, store, schema, githubToken, true, logger)
		assert.NoError(t, err, test.name)
		assert.True(t, changed, test.name)

		var b bytes.Buffer
		qr := graph.NewQuadStoreReader(store.QuadStore)
		assert.NotNil(t, qr, test.name)
		defer qr.Close()

		format := quad.FormatByName(dvstore.GoldenFormat)
		assert.NotNil(t, format, test.name)

		qw := format.Writer(&b)
		assert.NotNil(t, qw, test.name)
		defer qw.Close()

		n, err := quad.Copy(qw, qr)
		assert.Greater(t, n, 0, test.name)
		assert.NoError(t, err, test.name)

		gp := dvstore.TestingGoldenDumpPath(t, test.name)
		if testutil.UpdateGolden() {
			t.Logf("update golden file: %s", gp)
			err := os.WriteFile(gp, b.Bytes(), 0o644)
			assert.NoError(t, err, test.name)
		}

		g, err := os.ReadFile(gp)
		assert.NoError(t, err, test.name)
		assert.Equal(t, string(g), b.String())
	}
}
