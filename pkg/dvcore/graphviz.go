package dvcore

import (
	"fmt"
	"os"
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
	File  string
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

	filters := dvmodel.Filters{
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
		// TODO: handle more once implemented
		if t.Kind == dvmodel.Task_Issue || t.Kind == dvmodel.Task_MergeRequest {
			roadmap[fmtIRI(t.ID)] = t
		}
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
		for _, dependingID := range task.IsDependingOn {
			depending, ok := roadmap[fmtIRI(dependingID)]
			if !ok {
				continue
			}
			name := depending.ID + task.ID
			edge, err := graph.CreateEdge(fmtIRI(name), nodes[fmtIRI(depending.ID)], nodes[fmtIRI(task.ID)])
			if err != nil {
				return fmt.Errorf("create depending edge: %w", err)
			}
			_ = edge
		}
		for _, dependentID := range task.IsBlocking {
			dependent, ok := roadmap[fmtIRI(dependentID)]
			if !ok {
				continue
			}
			name := task.ID + dependent.ID
			edge, err := graph.CreateEdge(fmtIRI(name), nodes[fmtIRI(task.ID)], nodes[fmtIRI(dependent.ID)])
			if err != nil {
				return fmt.Errorf("create dependent edge: %w", err)
			}
			_ = edge
		}
		for _, relatedID := range task.IsRelatedWith {
			related, ok := roadmap[fmtIRI(relatedID)]
			if !ok {
				continue
			}
			name := task.ID + related.ID
			edge, err := graph.CreateEdge(fmtIRI(name), nodes[fmtIRI(task.ID)], nodes[fmtIRI(related.ID)])
			if err != nil {
				return fmt.Errorf("create related edge: %w", err)
			}

			name = related.ID + task.ID
			edge, err = graph.CreateEdge(fmtIRI(name), nodes[fmtIRI(related.ID)], nodes[fmtIRI(task.ID)])
			if err != nil {
				return fmt.Errorf("create related edge: %w", err)
			}
			_ = edge
		}
		// TODO: define best relationship for both following
		for _, partID := range task.IsPartOf {
			part, ok := roadmap[fmtIRI(partID)]
			if !ok {
				continue
			}
			name := task.ID + part.ID
			edge, err := graph.CreateEdge(fmtIRI(name), nodes[fmtIRI(task.ID)], nodes[fmtIRI(part.ID)])
			if err != nil {
				return fmt.Errorf("create dependent edge: %w", err)
			}
			_ = edge
		}
		for _, partID := range task.HasPart {
			part, ok := roadmap[fmtIRI(partID)]
			if !ok {
				continue
			}
			name := part.ID + task.ID
			edge, err := graph.CreateEdge(fmtIRI(name), nodes[fmtIRI(part.ID)], nodes[fmtIRI(task.ID)])
			if err != nil {
				return fmt.Errorf("create dependent edge: %w", err)
			}
			_ = edge
		}
	}
	if opts.File == "" {
		return g.Render(graph, graphviz.Format(opts.Type), os.Stdout)
	}
	return g.RenderFilename(graph, graphviz.Format(opts.Type), opts.File)
}

func fmtIRI(s quad.IRI) string {
	return strings.Replace(strings.Replace(s.String(), "<", "", -1), ">", "", -1)
}
