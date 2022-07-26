package dvserver

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"moul.io/depviz/v3/internal/dvcore"
	"moul.io/depviz/v3/internal/dvmodel"
	"moul.io/depviz/v3/internal/dvparser"
	"moul.io/depviz/v3/internal/dvstore"
)

func (s *service) Graph(ctx context.Context, in *Graph_Input) (*Graph_Output, error) {
	s.opts.Logger.Debug("graph", zap.Any("in", in))
	// validation
	if len(in.Targets) == 0 {
		return nil, fmt.Errorf("targets is required")
	}

	filters := dvstore.LoadTasksFilters{
		WithClosed:          in.WithClosed,
		WithoutIsolated:     in.WithoutIsolated,
		WithoutPRs:          in.WithoutPRs,
		WithoutExternalDeps: in.WithoutExternalDeps,
	}
	if len(in.Targets) == 1 && in.Targets[0] == "world" {
		filters.TheWorld = true
	} else {
		targets, err := dvparser.ParseTargets(in.Targets)
		if err != nil {
			return nil, fmt.Errorf("parse targets: %w", err)
		}
		filters.Targets = targets
	}

	// load tasks
	tasks, err := dvstore.LoadTasks(s.h, s.schema, filters, s.opts.Logger)
	if err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}
	
	// fetch if not already in db
	if len(tasks) == 0 {
		_, err := dvcore.PullAndSave(filters.Targets, s.h, s.schema, s.opts.GitHubToken, false, s.opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("pull: %w", err)
		}
		tasks, err = dvstore.LoadTasks(s.h, s.schema, filters, s.opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("load tasks: %w", err)
		}
	}

	// fetch if not already in db
	if len(tasks) == 0 {
		_, err := dvcore.PullAndSave(filters.Targets, s.h, s.schema, s.opts.GitHubToken, false, s.opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("pull: %w", err)
		}
		tasks, err = dvstore.LoadTasks(s.h, s.schema, filters, s.opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("load tasks: %w", err)
		}
	}

	// build output
	ret := Graph_Output{
		Tasks: make([]*dvmodel.Task, len(tasks)),
	}
	for idx, task := range tasks {
		clone := task
		ret.Tasks[idx] = &clone
	}
	return &ret, nil
}

func (s *service) StoreDump(ctx context.Context, in *StoreDump_Input) (*StoreDump_Output, error) {
	if !s.opts.Godmode {
		return nil, fmt.Errorf("permission denied (--god-mode required)")
	}
	batch, err := dvcore.GetStoreDump(ctx, s.h, s.schema)
	if err != nil {
		return nil, fmt.Errorf("store dump: %w", err)
	}

	ret := StoreDump_Output{
		Batch: batch,
	}
	return &ret, nil
}

func (s *service) Ping(context.Context, *Ping_Input) (*Ping_Output, error) {
	return &Ping_Output{Message: "pong"}, nil
}

func (s *service) Status(context.Context, *Status_Input) (*Status_Output, error) {
	return &Status_Output{EverythingIsOK: true}, nil
}
