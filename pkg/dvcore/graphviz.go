package dvcore

import (
	"fmt"
	"strings"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/quad"
	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	"go.uber.org/zap"
	"moul.io/depviz/v3/pkg/dvmodel"
	"moul.io/depviz/v3/pkg/dvparser"
	"moul.io/depviz/v3/pkg/dvstore"
)

type GraphvizOpts struct {
	*GenOpts

	// graphviz
	Label string
	Type  string
}

func GenGraphviz(h *cayley.Handle, args []string, opts GraphvizOpts) error {
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	opts.Logger.Debug("Gen called", zap.Strings("args", args), zap.Any("opts", opts))

	targets, err := dvparser.ParseTargets(args)
	if err != nil {
		return fmt.Errorf("parse targets: %w", err)
	}

	filters := dvstore.LoadTasksFilters{
		Targets:             targets,
		WithClosed:          opts.ShowClosed,
		WithoutIsolated:     opts.HideIsolated,
		WithoutPRs:          opts.HidePRs,
		WithoutExternalDeps: opts.HideExternalDeps,
	}
	tasks, err := dvstore.LoadTasks(h, opts.Schema, filters, opts.Logger)
	if err != nil {
		return fmt.Errorf("load tasks: %w", err)
	}

	roadmap := make(map[string]dvmodel.Task)
	for _, t := range tasks {
		if t.Kind != 1 {
			continue
		}
		roadmap[fmtIRI(t.ID)] = t
	}

	g := graphviz.New()
	graph, err := g.Graph()
	if err != nil {
		return fmt.Errorf("create graph: %w", err)
	}

	graph.SetRankDir("LR")
	graph.SetLabel(opts.Label)
	defer func() {
		if graph.Close() != nil {
			panic("graph.Close() failed")
		}
		g.Close()
	}()

	nodes := make(map[string]*cgraph.Node)

	for _, task := range roadmap {
		node, err := graph.CreateNode(fmtIRI(task.ID))
		if err != nil {
			return fmt.Errorf("create node: %w", err)
		}

		node.SetLabel(task.Title)
		node.SetHref(fmtIRI(task.ID))
		node.SetShape("box")
		node.SetStyle("rounded")

		task.ApplyLabel(node)

		nodes[fmtIRI(task.ID)] = node
	}

	for _, task := range roadmap {
		for _, dependentID := range task.IsBlocking {
			dependent := roadmap[fmtIRI(dependentID)]
			name := task.ID + dependent.ID
			edge, err := graph.CreateEdge(fmtIRI(name), nodes[fmtIRI(task.ID)], nodes[fmtIRI(dependent.ID)])
			if err != nil {
				return fmt.Errorf("create dependent edge: %w", err)
			}
			_ = edge
		}
		for _, dependingID := range task.IsDependingOn {
			depending := roadmap[fmtIRI(dependingID)]
			name := depending.ID + task.ID
			edge, err := graph.CreateEdge(fmtIRI(name), nodes[fmtIRI(depending.ID)], nodes[fmtIRI(task.ID)])
			if err != nil {
				return fmt.Errorf("create depending edge: %w", err)
			}
			_ = edge
		}
	}
	return g.RenderFilename(graph, graphviz.Format(opts.Type), "graph."+opts.Type)
}

func fmtIRI(s quad.IRI) string {
	return strings.Replace(strings.Replace(s.String(), "<", "", -1), ">", "", -1)
}
