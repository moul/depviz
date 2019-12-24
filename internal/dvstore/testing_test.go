package dvstore

import (
	"context"
	"testing"

	"github.com/cayleygraph/cayley/schema"
	"github.com/stretchr/testify/assert"
)

var schemaConfig *schema.Config

func init() {
	schemaConfig = Schema()
}

func TestTestingGoldenStore(t *testing.T) {
	store, close := TestingGoldenStore(t, "all-depviz-test")
	assert.NotNil(t, store)
	defer close()

	ctx := context.Background()
	it := store.QuadsAllIterator()
	count := 0
	for it.Next(ctx) {
		count++
	}
	// FIXME: check if contain some specific data
	assert.Greater(t, count, 0)
}
