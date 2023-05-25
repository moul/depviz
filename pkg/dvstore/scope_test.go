package dvstore

import (
	"testing"

	"moul.io/depviz/v3/pkg/dvparser"

	"github.com/stretchr/testify/assert"
	"golang.org/x/exp/slices"

	"moul.io/depviz/v3/pkg/dvmodel"
	"moul.io/depviz/v3/pkg/testutil"
)

func TestScopeIssue(t *testing.T) {

	tests := []struct {
		target             string
		golden             string
		name               string
		filters            dvmodel.Filters
		expectedDependency []string
	}{

		{
			"https://github.com/moul/depviz-test/issues/6",
			"all-depviz-test",
			"theworld",
			dvmodel.Filters{
				Targets:             nil,
				TheWorld:            true,
				WithClosed:          true,
				WithoutIsolated:     false,
				WithoutPRs:          true,
				WithoutExternalDeps: true,
				WithFetch:           false,
				ScopeSize:           1,
			},
			[]string{
				"<https://github.com/moul/depviz-test/issues/2>",
				"<https://github.com/moul/depviz-test/issues/3>",
				"<https://github.com/moul/depviz-test/issues/5>",
				"<https://github.com/moul/depviz-test/issues/6>",
				"<https://github.com/moul/depviz-test/issues/7>",
				"<https://github.com/moul/depviz-test/issues/10>",
			}},
	}

	logger := testutil.Logger(t)
	for _, testptr := range tests {
		test := testptr
		store, close := TestingGoldenStore(t, test.golden)
		defer close()
		targetEntity, err := dvparser.ParseTarget(test.target)
		if err != nil {
			t.Fatal(err)
		}

		test.filters.Scope = targetEntity

		tasks, err := LoadTasks(store, schemaConfig, test.filters, logger)
		if err != nil {
			return
		}

		for _, task := range tasks {
			assert.Equal(t, true, slices.Contains(test.expectedDependency, task.ID.String()), "unexpected dependency %q", task.ID.String())
		}
	}
}
