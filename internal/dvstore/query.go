package dvstore

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph/path"
	"github.com/cayleygraph/cayley/schema"
	"github.com/cayleygraph/quad"
	"moul.io/depviz/internal/dvmodel"
	"moul.io/multipmuri"
)

func LastUpdatedIssueInRepo(ctx context.Context, h *cayley.Handle, entity multipmuri.Entity) (time.Time, error) {
	type multipmuriMinimalInterface interface {
		Repo() *multipmuri.GitHubRepo
	}
	entityWithRepo, ok := entity.(multipmuriMinimalInterface)
	if !ok {
		return time.Time{}, fmt.Errorf("invalid entity: %q", entity.String())
	}
	repo := entityWithRepo.Repo()

	// g.V("<https://github.com/moul/depviz-test>").In().Has("<rdf:type>", "<dv:Task>").Has("<schema:kind>", 1).Out("<schema:updatedAt>").all()
	chain := path.StartPath(h, quad.IRI(repo.String())).
		In().
		Has(quad.IRI("rdf:type"), quad.IRI("dv:Task")).
		Has(quad.IRI("schema:kind"), quad.Int(dvmodel.Task_Issue)).
		Out(quad.IRI("schema:updatedAt")).
		Iterate(ctx)
	since := time.Time{}

	values, err := chain.Paths(false).AllValues(h)
	if err != nil {
		return time.Time{}, err
	}

	for _, value := range values {
		typed := quad.NativeOf(value).(time.Time)
		if since.Before(typed) {
			since = typed
		}
	}
	return since, nil
}

type LoadTasksFilters struct {
	Targets             []multipmuri.Entity
	WithClosed          bool
	WithoutIsolated     bool
	WithoutPRs          bool
	WithoutExternalDeps bool
}

func LoadTasks(h *cayley.Handle, schema *schema.Config, filters LoadTasksFilters) (dvmodel.Tasks, error) {
	if filters.Targets == nil || len(filters.Targets) == 0 {
		return nil, fmt.Errorf("missing filter.targets")
	}

	ctx := context.TODO()

	// fetch and filter
	paths := []*path.Path{}
	for _, target := range filters.Targets {
		// FIXME: handle different target types (for now only repo)
		p := path.StartPath(h, quad.IRI(target.String())).
			In().
			Has(quad.IRI("rdf:type"), quad.IRI("dv:Task"))
		if filters.WithoutPRs {
			p = p.Has(quad.IRI("schema:kind"), quad.Int(dvmodel.Task_Issue))
		} else {
			p = p.Has(quad.IRI("schema:kind"), quad.Int(dvmodel.Task_Issue), quad.Int(dvmodel.Task_MergeRequest))
		}
		if !filters.WithClosed {
			p = p.Has(quad.IRI("schema:state"), quad.Int(dvmodel.Task_Open))
		}
		// FIXME: reverse depends/blocks

		paths = append(paths, p)
	}

	p := paths[0]
	for _, path := range paths[1:] {
		p = p.Or(path)
	}

	if !filters.WithoutExternalDeps {
		p = p.Or(p.Both(quad.IRI("isDependingOn"), quad.IRI("isBlocking")))
	}

	allTasks := dvmodel.Tasks{}
	err := schema.LoadPathTo(ctx, h, &allTasks, p)
	if err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}
	tasks := dvmodel.Tasks{}
	for _, task := range allTasks {
		if !filters.WithoutIsolated {
			tasks = append(tasks, task)
			continue
		}
		if len(task.IsDependingOn) > 0 || len(task.IsBlocking) > 0 {
			tasks = append(tasks, task)
		}
	}

	sort.Slice(tasks[:], func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	// fmt.Println(godev.PrettyJSON(tasks))

	return tasks, nil
}
