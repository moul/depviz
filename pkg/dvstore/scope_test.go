package dvstore

import (
	"testing"

	"github.com/Doozers/gl/pkg/funct"
	"github.com/cayleygraph/quad"
	"golang.org/x/exp/slices"
	"moul.io/depviz/v3/pkg/dvmodel"
	"moul.io/depviz/v3/pkg/dvparser"
	"moul.io/depviz/v3/pkg/testutil"
)

type recFunc func(task dvmodel.Task, remaining uint8, target string) bool

func TestScopeIssue(t *testing.T) {

	tests := []struct {
		target  string
		golden  string
		name    string
		filters dvmodel.Filters
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
		},
	}

	logger := testutil.Logger(t)
	for _, testptr := range tests {
		test := testptr
		store, closeFunc := TestingGoldenStore(t, test.golden)
		defer closeFunc()
		targetEntity, err := dvparser.ParseTarget(test.target)
		if err != nil {
			t.Fatal(err)
		}

		test.filters.Scope = targetEntity

		tasks, err := LoadTasks(store, schemaConfig, test.filters, logger)
		if err != nil {
			return
		}

		mtasks := map[string]dvmodel.Task{}
		for _, task := range tasks {
			mtasks[task.ID.String()] = task
		}

		var rec recFunc
		rec = func(_task dvmodel.Task, _remaining uint8, _target string) bool {
			if _task.ID.String() == _target {
				return true
			}

			if _remaining == 0 {
				return false
			}

			if slices.Contains(funct.Map(_task.IsDependingOn, func(t quad.IRI) string {
				return t.String()
			}), _target) {
				return true
			}
			for _, dep := range _task.IsDependingOn {
				if t, ok := mtasks[dep.String()]; ok {
					if rec(t, _remaining-1, _target) {
						return true
					}
				}
			}
			return false
		}

		for _, task := range tasks {
			rec1 := rec(task, uint8(test.filters.ScopeSize), "<"+test.target+">")
			rec2 := rec(mtasks["<"+test.target+">"], uint8(test.filters.ScopeSize), task.ID.String())

			if rec1 == false && rec2 == false {
				t.Errorf("task %s should be in the scope of %s", task.ID.String(), test.target)
			}
		}
	}
}
