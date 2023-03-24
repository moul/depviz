package dvcore

import (
	"fmt"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/schema"
	"go.uber.org/zap"
	"moul.io/depviz/v3/internal/dvparser"
	"moul.io/multipmuri"
)

type FetchOpts struct {
	Logger *zap.Logger
	Schema *schema.Config

	GitHubToken string
	Resync      bool
}

func Fetch(h *cayley.Handle, args []string, opts FetchOpts) error {
	targets, err := dvparser.ParseTargets(args)
	if err != nil {
		return fmt.Errorf("parse targets: %w", err)
	}

	_, err = PullAndSave(targets, h, opts.Schema, opts.GitHubToken, opts.Resync, opts.Logger)
	return err
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
