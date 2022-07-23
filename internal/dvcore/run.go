package dvcore

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	"github.com/cayleygraph/cayley/schema"
	"go.uber.org/zap"
	yaml "gopkg.in/yaml.v2"
	"moul.io/depviz/v3/internal/dvmodel"
	"moul.io/depviz/v3/internal/dvparser"
	"moul.io/depviz/v3/internal/dvstore"
	"moul.io/depviz/v3/internal/githubprovider"
	"moul.io/graphman"
	"moul.io/graphman/viz"
	"moul.io/multipmuri"
)

type RunOpts struct {
	// global

	NoPull  bool
	NoGraph bool
	Logger  *zap.Logger
	Schema  *schema.Config

	// pull

	GitHubToken string
	// GitLabToken string
	// TrelloToken string
	// JiraToken string
	Resync bool

	// graph

	Format           string
	Vertical         bool
	NoPert           bool
	ShowClosed       bool
	HideIsolated     bool
	HidePRs          bool
	HideExternalDeps bool
}

func Run(h *cayley.Handle, args []string, opts RunOpts) error {
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	opts.Logger.Debug("Run called", zap.Strings("args", args), zap.Any("opts", opts))

	// FIXME: support the world

	targets, err := dvparser.ParseTargets(args)
	if err != nil {
		return fmt.Errorf("parse targets: %w", err)
	}

	if !opts.NoPull {
		_, err := PullAndSave(targets, h, opts.Schema, opts.GitHubToken, opts.Resync, opts.Logger)
		if err != nil {
			return fmt.Errorf("pull: %w", err)
		}
	}

	if !opts.NoGraph { // nolint:nestif
		// load tasks
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

		// graph
		pertConfig := graphmanPertConfig(tasks, opts)

		switch opts.Format {
		case "json":
			out, err := json.MarshalIndent(tasks, "", "  ")
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		case "graphman-pert":
			out, err := yaml.Marshal(pertConfig)
			if err != nil {
				return err
			}
			fmt.Println(string(out))
			return nil
		case "dot":
			// graph from PERT config
			graph := graphman.FromPertConfig(*pertConfig)

			// initialize graph from config
			if !opts.NoPert {
				result := graphman.ComputePert(graph)
				shortestPath, distance := graph.FindShortestPath("Start", "Finish")
				opts.Logger.Debug("pert result", zap.Any("result", result), zap.Int64("distance", distance))

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
			graph.Attrs["overlap"] = "false"
			graph.Attrs["pack"] = "true"
			graph.Attrs["splines"] = "true"
			graph.Attrs["sep"] = "0.1"
			// graph.Attrs["layout"] = "neato"
			// graph.Attrs["size"] = "\"11,11\""
			// graph.Attrs["start"] = "random"
			// FIXME: hightlight critical paths
			// FIXME: highlight other infos
			// FIXME: highlight target

			// graphviz
			s, err := viz.ToGraphviz(graph, &viz.Opts{
				CommentsInLabel: true,
			})
			if err != nil {
				return fmt.Errorf("graphviz: %w", err)
			}

			fmt.Println(s)
			return nil
		case "quads":
			return fmt.Errorf("not implemented")
		default:
			return fmt.Errorf("unsupported graph format: %q", opts.Format)
		}
	}

	return nil
}

func PullAndSave(targets []multipmuri.Entity, h *cayley.Handle, schema *schema.Config, githubToken string, resync bool, logger *zap.Logger) (bool, error) {
	batches := pullBatches(targets, h, githubToken, resync, logger)
	if len(batches) > 0 {
		err := saveBatches(h, schema, batches)
		if err != nil {
			return false, fmt.Errorf("save batches: %w", err)
		}
		return true, nil
	}
	return false, nil
}

func pullBatches(targets []multipmuri.Entity, h *cayley.Handle, githubToken string, resync bool, logger *zap.Logger) []dvmodel.Batch {
	// FIXME: handle the special '@me' target
	var (
		wg      sync.WaitGroup
		batches = []dvmodel.Batch{}
		out     = make(chan dvmodel.Batch)
		ctx     = context.Background()
	)

	// parallel fetches
	wg.Add(len(targets))
	for _, target := range targets {
		switch provider := target.Provider(); provider { // nolint:exhaustive
		case multipmuri.GitHubProvider:
			go func(repo multipmuri.Entity) {
				defer wg.Done()

				ghOpts := githubprovider.Opts{
					// FIXME: Since: lastUpdated,
					Logger: logger.Named("github"),
				}

				if !resync {
					since, err := dvstore.LastUpdatedIssueInRepo(ctx, h, repo)
					if err != nil {
						logger.Warn("failed to get last updated issue", zap.Error(err))
					}
					if !since.IsZero() && since.Unix() > 0 {
						ghOpts.Since = &since
					}
				}

				githubprovider.FetchRepo(ctx, repo, githubToken, out, ghOpts)
			}(target)
		default:
			// FIXME: clean context-based exit
			panic(fmt.Sprintf("unsupported provider: %v", provider))
		}
	}
	go func() {
		wg.Wait()
		close(out)
	}()

	for batch := range out {
		batches = append(batches, batch)
	}

	return batches
}

func saveBatches(h *cayley.Handle, schema *schema.Config, batches []dvmodel.Batch) error {
	ctx := context.TODO()

	tx := cayley.NewTransaction()
	dw := graph.NewTxWriter(tx, graph.Delete)
	iw := graph.NewTxWriter(tx, graph.Add)

	for _, batch := range batches {
		for _, owner := range batch.Owners {
			var working dvmodel.Owner
			if err := schema.LoadTo(ctx, h, &working, owner.ID); err == nil {
				_, _ = schema.WriteAsQuads(dw, working)
			}

			working = *owner
			if _, err := schema.WriteAsQuads(iw, working); err != nil {
				return fmt.Errorf("write as quads: %w", err)
			}
		}
		for _, task := range batch.Tasks {
			var working dvmodel.Task
			if err := schema.LoadTo(ctx, h, &working, task.ID); err == nil {
				_, _ = schema.WriteAsQuads(dw, working)
			}

			working = *task
			if _, err := schema.WriteAsQuads(iw, working); err != nil {
				return fmt.Errorf("write as quads: %w", err)
			}
		}
		for _, topic := range batch.Topics {
			var working dvmodel.Topic
			if err := schema.LoadTo(ctx, h, &working, topic.ID); err == nil {
				_, _ = schema.WriteAsQuads(dw, working)
			}

			working = *topic
			if _, err := schema.WriteAsQuads(iw, working); err != nil {
				return fmt.Errorf("write as quads: %w", err)
			}
		}
	}

	if err := h.ApplyTransaction(tx); err != nil {
		return fmt.Errorf("apply tx: %w", err)
	}
	return nil
}

func graphmanPertConfig(tasks []dvmodel.Task, opts RunOpts) *graphman.PertConfig {
	opts.Logger.Debug("graphTargets", zap.Int("tasks", len(tasks)), zap.Any("opts", opts))

	// initialize graph config
	config := graphman.PertConfig{
		Actions: []graphman.PertAction{},
		States:  []graphman.PertState{},
	}
	config.Opts.NoSimplify = false

	// process tasks
	for _, task := range tasks {
		// compute dependsOn
		dependsOn := []string{}
		for _, dep := range task.IsDependingOn {
			dependsOn = append(dependsOn, string(dep))
		}
		// FIXME: compute reverse dependsOn

		switch task.Kind { // nolint:exhaustive
		case dvmodel.Task_Issue, dvmodel.Task_MergeRequest:
			config.Actions = append(
				config.Actions,
				graphman.PertAction{
					ID:        string(task.ID),
					Title:     task.Title,
					DependsOn: dependsOn,
					// FIXME: Estimate
					// FIXME: set style based on type, active, etc
				},
			)
		case dvmodel.Task_Milestone:
			config.States = append(
				config.States,
				graphman.PertState{
					ID:        string(task.ID),
					Title:     task.Title,
					DependsOn: dependsOn,
					// FIXME: auto estimate (PERT)
					// FIXME: DependsOn: milestone.DependsOn
					// FIXME: styling
				},
			)
		default:
			opts.Logger.Warn("unsupported task kind", zap.Stringer("kind", task.Kind))
		}
	}
	// FIXME: if len(unique(repos)) > 1 -> add PertState for each repo with DependsOn

	return &config
}
