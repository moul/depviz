package dvstore

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	_ "github.com/cayleygraph/quad/json"
	"github.com/stretchr/testify/assert"
	"moul.io/depviz/v3/internal/dvparser"
	"moul.io/depviz/v3/internal/testutil"
	"moul.io/godev"
	"moul.io/multipmuri"
)

func TestLoadTasks(t *testing.T) {
	tests := []struct {
		golden      string
		name        string
		filters     LoadTasksFilters
		expectedErr error
	}{
		{"all-depviz-test", "theworld", LoadTasksFilters{TheWorld: true}, nil},
		{"all-depviz-test", "theworld-light", LoadTasksFilters{TheWorld: true, WithoutPRs: true, WithClosed: false, WithoutExternalDeps: true}, nil},
		{"all-depviz-test", "theworld-with-closed", LoadTasksFilters{TheWorld: true, WithClosed: true}, nil},
		{"all-depviz-test", "theworld-without-prs", LoadTasksFilters{TheWorld: true, WithoutPRs: true}, nil},
		{"all-depviz-test", "theworld-without-isolated", LoadTasksFilters{TheWorld: true, WithoutIsolated: true}, nil},
		{"all-depviz-test", "theworld-without-external-deps", LoadTasksFilters{TheWorld: true, WithoutExternalDeps: true}, nil},
		{"all-depviz-test", "theworld-all-flags", LoadTasksFilters{TheWorld: true, WithClosed: true, WithoutPRs: true, WithoutIsolated: true, WithoutExternalDeps: true}, nil},
		{"all-depviz-test", "moul-depviz-test", LoadTasksFilters{Targets: parseTargets(t, "moul/depviz-test")}, nil},
		{"all-depviz-test", "moulbot-depviz-test", LoadTasksFilters{Targets: parseTargets(t, "moul-bot/depviz-test")}, nil},
		{"all-depviz-test", "moul-and-moulbot-depviz-test", LoadTasksFilters{Targets: parseTargets(t, "moul/depviz-test, moul-bot/depviz-test")}, nil},
	}
	alreadySeen := map[string]bool{}
	for _, testptr := range tests {
		test := testptr
		name := fmt.Sprintf("%s/%s", test.golden, test.name)
		gp := TestingGoldenJSONPath(t, name)
		if _, found := alreadySeen[gp]; found {
			t.Fatalf("duplicate key: %q (golden files conflict)", gp)
		}
		alreadySeen[gp] = true

		t.Run(name, func(t *testing.T) {
			t.Parallel()
			logger := testutil.Logger(t)
			store, close := TestingGoldenStore(t, test.golden)
			defer close()
			tasks, err := LoadTasks(store, schemaConfig, test.filters, logger)
			assert.Equal(t, test.expectedErr, err)
			if err != nil {
				return
			}
			assert.NotNil(t, tasks)
			assert.NoError(t, err)

			actual := godev.JSON(test.filters) + "\n"
			for _, task := range tasks {
				actual += godev.JSON(task) + "\n"
			}

			if testutil.UpdateGolden() {
				t.Logf("update golden file: %s", gp)
				err := ioutil.WriteFile(gp, []byte(actual), 0644)
				assert.NoError(t, err, name)
			}

			{ // check for duplicates
				duplicateMap := map[string]int{}
				hasDuplicates := false
				for _, task := range tasks {
					if _, found := duplicateMap[string(task.ID)]; !found {
						duplicateMap[string(task.ID)] = 0
					} else {
						hasDuplicates = true
					}
					duplicateMap[string(task.ID)]++
				}
				if !assert.False(t, hasDuplicates) {
					fmt.Println(godev.PrettyJSON(duplicateMap))
				}
			}
			t.Log("\n" + tasks.DebugTree(true, false))

			g, err := ioutil.ReadFile(gp)
			assert.NoError(t, err, name)
			assert.Equal(t, len(string(g)), len(actual), gp)
		})
	}
}

func parseTargets(t *testing.T, input string) []multipmuri.Entity {
	t.Helper()
	targets, err := dvparser.ParseTargets(strings.Split(input, ", "))
	if !assert.NoError(t, err) {
		t.Fatalf("parse targets: %v", input)
	}
	return targets
}
