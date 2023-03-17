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
	"go.uber.org/zap"
	"moul.io/depviz/v3/internal/dvmodel"
	"moul.io/multipmuri"
)

func LastUpdatedIssueInRepo(ctx context.Context, h *cayley.Handle, entity multipmuri.Entity) (time.Time, error) { // nolint:interfacer
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

	since = since.Add(time.Second) // in order to skip the last one
	// FIXME: find a better approach

	return since, nil
}

type LoadTasksFilters struct {
	Targets             []multipmuri.Entity
	TheWorld            bool
	WithClosed          bool
	WithoutIsolated     bool
	WithoutPRs          bool
	WithoutExternalDeps bool
	WithFetch           bool
}

func LoadTasks(h *cayley.Handle, schema *schema.Config, filters LoadTasksFilters, logger *zap.Logger) (dvmodel.Tasks, error) {
	if (filters.Targets == nil || len(filters.Targets) == 0) && !filters.TheWorld {
		return nil, fmt.Errorf("missing filter.targets")
	}

	ctx := context.TODO()

	// fetch targets
	paths := []*path.Path{}
	if filters.TheWorld {
		paths = append(paths, path.StartPath(h))
	} else {
		for _, target := range filters.Targets {
			// FIXME: handle different target types (for now only repo)
			p := path.StartPath(h, quad.IRI(target.String())).
				Both().
				Has(quad.IRI("rdf:type"), quad.IRI("dv:Task"))

			// FIXME: reverse depends/blocks
			paths = append(paths, p)
		}
	}
	p := paths[0]
	for _, path := range paths[1:] {
		p = p.Or(path)
	}

	// filters
	kinds := []quad.Value{
		quad.Int(dvmodel.Task_Issue),
		quad.Int(dvmodel.Task_Milestone),
		quad.Int(dvmodel.Task_Epic),
		quad.Int(dvmodel.Task_Story),
		quad.Int(dvmodel.Task_Card),
	}
	if !filters.WithoutPRs {
		kinds = append(kinds, quad.Int(dvmodel.Task_MergeRequest))
	}
	p = p.Has(quad.IRI("schema:kind"), kinds...)
	if !filters.WithClosed {
		p = p.Has(quad.IRI("schema:state"), quad.Int(dvmodel.Task_Open))
	}

	if !filters.WithoutExternalDeps {
		p = p.Or(p.Both(
			quad.IRI("isDependingOn"),
			quad.IRI("isBlocking"),
			quad.IRI("IsRelatedWith"),
			quad.IRI("IsPartOf"),
			quad.IRI("HasPart"),
		))
	}

	tasks := dvmodel.Tasks{}
	p = p.Limit(300) // nolint:gomnd
	err := schema.LoadPathTo(ctx, h, &tasks, p)
	if err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}

	if filters.WithoutIsolated {
		tasks = dvmodel.FilterIsolatedTasks(tasks, logger)
	}

	{ // remove duplicates
		// FIXME: remove duplicates from the query itself
		taskMap := map[quad.IRI]dvmodel.Task{}
		for _, task := range tasks {
			taskMap[task.ID] = task
		}
		tasks = make(dvmodel.Tasks, len(taskMap))
		i := 0
		for _, task := range taskMap {
			tasks[i] = task
			i++
		}
	}

	sort.Slice(tasks[:], func(i, j int) bool {
		return tasks[i].ID < tasks[j].ID
	})

	// fmt.Println(godev.PrettyJSON(tasks))

	return tasks, nil
}
