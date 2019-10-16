package graph // import "moul.io/depviz/graph"

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
	"moul.io/depviz/compute"
	"moul.io/depviz/sql"
	"moul.io/graphman"
	"moul.io/graphman/viz"
	"moul.io/multipmuri"
)

type Options struct {
	SQL             sql.Options         `mapstructure:"sql"`     // inherited with sql.GetOptions()
	Targets         []multipmuri.Entity `mapstructure:"targets"` // parsed from Args
	ShowClosed      bool                `mapstructure:"show-closed"`
	ShowOrphans     bool                `mapstructure:"show-orphans"`
	ShowPRs         bool                `mapstructure:"show-prs"`
	ShowAllRelated  bool                `mapstructure:"show-all-related"`
	NoPertEstimates bool                `mapstructure:"no-pert-estimates"`
	Vertical        bool                `mapstructure:"vertical"`
	Format          string              `mapstructure:"format"`
}

func (opts Options) Validate() error {
	if err := opts.SQL.Validate(); err != nil {
		return err
	}
	switch format := opts.Format; format {
	case "dot", "graphman-pert":
	default:
		return fmt.Errorf("invalid format: %q", format)
	}
	return nil
}

func (opts Options) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func PrintGraph(opts *Options) error {
	zap.L().Debug("PrintGraph", zap.Stringer("opts", *opts))

	str, err := Graph(opts)
	if err != nil {
		return err
	}

	fmt.Println(str)
	return nil
}

func Graph(opts *Options) (string, error) {
	zap.L().Debug("Graph", zap.Stringer("opts", *opts))

	db, err := sql.FromOpts(&opts.SQL)
	if err != nil {
		return "", err
	}

	computed, err := compute.LoadIssuesByTargets(db, opts.Targets)
	if err != nil {
		return "", err
	}
	if !opts.ShowClosed {
		computed.FilterClosed()
	}
	// FIXME: if !opts.ShowOrphans { computed.FilterOrphans() }
	// FIXME: if !opts.ShowAllRelated { computed.FilterAllRelated()
	// FIXME: if !opts.ShowPRs { computed.FilterPRs()

	// initialize graph config
	config := graphman.PertConfig{
		Actions: []graphman.PertAction{},
		States:  []graphman.PertState{},
	}
	config.Opts.NoSimplify = false

	// process computed issues
	for _, issue := range computed.Issues() {
		// fmt.Println(issue.Hidden, issue.URL)
		if issue.Hidden {
			continue
		}
		config.Actions = append(
			config.Actions,
			graphman.PertAction{
				ID:        issue.URL,
				Title:     issue.Title,
				DependsOn: issue.DependsOn,
				// Estimate
				// FIXME: set style based on type, active, etc
			},
		)
	}
	for _, milestone := range computed.Milestones() {
		if milestone.Hidden {
			continue
		}
		config.States = append(
			config.States,
			graphman.PertState{
				ID:        milestone.URL,
				Title:     milestone.Title,
				DependsOn: milestone.DependsOn,
			},
		)
	}
	if len(computed.Repos()) > 1 {
		// FIXME: alternative layout with repo a bordered subgraph
		for _, repo := range computed.Repos() {
			if repo.Hidden {
				continue
			}
			config.States = append(
				config.States,
				graphman.PertState{
					ID:        repo.URL,
					Title:     fmt.Sprintf("Repo %q", repo.URL),
					DependsOn: repo.DependsOn,
				},
			)
		}
	}

	if opts.Format == "graphman-pert" {
		out, err := yaml.Marshal(config)
		if err != nil {
			return "", err
		}
		return string(out), nil
	}

	// initialize graph from config
	graph := graphman.FromPertConfig(config)
	if !opts.NoPertEstimates {
		_ = graphman.ComputePert(graph)
		//for _, e := range graph.Edges() {log.Println("*", e)}
		shortestPath, _ := graph.FindShortestPath("Start", "Finish")
		for _, edge := range shortestPath {
			edge.Dst().SetColor("red")
			edge.SetColor("red")
		}
	}

	// graph fine tuning
	graph.GetVertex("Start").SetColor("blue")
	graph.GetVertex("Finish").SetColor("blue")
	if opts.Vertical {
		graph.Attrs["rankdir"] = "TB"
	}
	//graph.Attrs["size"] = "\"11,11\""
	graph.Attrs["overlap"] = "false"
	graph.Attrs["pack"] = "true"
	graph.Attrs["splines"] = "true"
	// graph.Attrs["layout"] = "neato"
	graph.Attrs["sep"] = "0.1"
	// graph.Attrs["start"] = "random"
	// FIXME: hightlight critical paths
	// FIXME: highlight other infos
	// FIXME: highlight target

	// graphviz
	s, err := viz.ToGraphviz(graph, &viz.Opts{
		CommentsInLabel: true,
	})
	if err != nil {
		return "", err
	}

	return s, nil
}
