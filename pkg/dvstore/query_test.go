package dvstore

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"moul.io/depviz/v3/pkg/dvmodel"
	"moul.io/depviz/v3/pkg/dvparser"
	"moul.io/depviz/v3/pkg/multipmuri"
	"moul.io/depviz/v3/pkg/testutil"
	"moul.io/godev"

	"github.com/stretchr/testify/assert"
)

func TestLoadTasks(t *testing.T) {
	tests := []struct {
		golden      string
		name        string
		filters     dvmodel.Filters
		expectedErr error
	}{
		{"all-depviz-test", "theworld", dvmodel.Filters{TheWorld: true}, nil},
		{"all-depviz-test", "theworld-light", dvmodel.Filters{TheWorld: true, WithoutPRs: true, WithClosed: false, WithoutExternalDeps: true}, nil},
		{"all-depviz-test", "theworld-with-closed", dvmodel.Filters{TheWorld: true, WithClosed: true}, nil},
		{"all-depviz-test", "theworld-without-prs", dvmodel.Filters{TheWorld: true, WithoutPRs: true}, nil},
		{"all-depviz-test", "theworld-without-isolated", dvmodel.Filters{TheWorld: true, WithoutIsolated: true}, nil},
		{"all-depviz-test", "theworld-without-external-deps", dvmodel.Filters{TheWorld: true, WithoutExternalDeps: true}, nil},
		{"all-depviz-test", "theworld-all-flags", dvmodel.Filters{TheWorld: true, WithClosed: true, WithoutPRs: true, WithoutIsolated: true, WithoutExternalDeps: true}, nil},
		{"all-depviz-test", "moul-depviz-test", dvmodel.Filters{Targets: parseTargets(t, "moul/depviz-test")}, nil},
		{"all-depviz-test", "moulbot-depviz-test", dvmodel.Filters{Targets: parseTargets(t, "moul-bot/depviz-test")}, nil},
		{"all-depviz-test", "moul-and-moulbot-depviz-test", dvmodel.Filters{Targets: parseTargets(t, "moul/depviz-test, moul-bot/depviz-test")}, nil},
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
				err := os.WriteFile(gp, []byte(actual), 0o644)
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

			g, err := os.ReadFile(gp)
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
