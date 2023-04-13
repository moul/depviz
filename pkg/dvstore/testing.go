package dvstore

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	// required by cayley
	_ "github.com/cayleygraph/cayley/graph/kv/bolt"
	"github.com/cayleygraph/quad"
	// required by cayley
	_ "github.com/cayleygraph/quad/gml"
	// required by cayley
	_ "github.com/cayleygraph/quad/graphml"
	// required by cayley
	_ "github.com/cayleygraph/quad/json"
	// required by cayley
	_ "github.com/cayleygraph/quad/jsonld"
	// required by cayley
	_ "github.com/cayleygraph/quad/nquads"
	// required by cayley
	_ "github.com/cayleygraph/quad/pquads"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const GoldenFormat = "pquads"

func TestingGoldenDumpPath(t *testing.T, name string) string {
	return filepath.Join("..", "dvstore", "testdata", "golden."+name+".pq")
}

func TestingGoldenJSONPath(t *testing.T, name string) string {
	nameWithoutSlashes := strings.Replace(name, "/", "_", -1)
	return filepath.Join("..", "dvstore", "testdata", "golden."+nameWithoutSlashes+".json")
}

func TestingGoldenStore(t *testing.T, name string) (*cayley.Handle, func()) {
	t.Helper()

	store, closeFunc := TestingStore(t)

	gp := TestingGoldenDumpPath(t, name)
	f, err := os.Open(gp)
	assert.NoError(t, err, name)
	defer f.Close()

	qw, err := store.NewQuadWriter()
	assert.NoError(t, err, name)
	assert.NotNil(t, qw, name)
	defer qw.Close()

	format := quad.FormatByName(GoldenFormat)
	assert.NotNil(t, format, name)

	qr := format.Reader(f)
	assert.NotNil(t, qr, name)
	defer qr.Close()

	n, err := quad.CopyBatch(qw, qr, quad.DefaultBatch)
	assert.NoError(t, err, name)
	assert.Greater(t, n, 0, name)

	return store, closeFunc
}

func TestingStore(t *testing.T) (*cayley.Handle, func()) {
	t.Helper()

	dir, err := os.MkdirTemp("", "depviz")
	require.NoError(t, err)

	err = graph.InitQuadStore("bolt", dir, nil)
	require.NoError(t, err)

	store, err := cayley.NewGraph("bolt", dir, nil)
	require.NoError(t, err)

	closeFunc := func() {
		if store != nil {
			_ = store.Close()
		}
		_ = os.RemoveAll(dir)
	}

	return store, closeFunc
}
