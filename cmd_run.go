package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type runOptions struct {
	// pull
	PullOpts     pullOptions
	NoPull       bool
	ReposToFetch []string

	// db
	DBOpts    dbOptions
	GraphOpts graphOptions

	// run
	AdditionalPulls []string
	Destination     string

	//Preview     bool
}

func (opts runOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func runSetupFlags(flags *pflag.FlagSet, opts *runOptions) {
	flags.BoolVarP(&opts.NoPull, "no-pull", "f", false, "do not pull new issues before runing")
	flags.StringVarP(&opts.Destination, "destination", "", "-", "destination ('-' for stdout)")
	flags.StringSliceVarP(&opts.AdditionalPulls, "additional-pull", "", []string{}, "additional pull that won't necessarily be displayed on the graph")
	//flags.BoolVarP(&opts.Preview, "preview", "p", false, "preview result")
	viper.BindPFlags(flags)
}

func newRunCommand() *cobra.Command {
	opts := &runOptions{}
	cmd := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(opts); err != nil {
				return err
			}
			if err := viper.Unmarshal(&opts.PullOpts); err != nil {
				return err
			}
			if err := viper.Unmarshal(&opts.DBOpts); err != nil {
				return err
			}
			if err := viper.Unmarshal(&opts.GraphOpts); err != nil {
				return err
			}
			opts.PullOpts.DBOpts = opts.DBOpts
			opts.PullOpts.Targets = append(args, opts.AdditionalPulls...)
			opts.GraphOpts.Targets = args
			return run(opts)
		},
	}
	runSetupFlags(cmd.Flags(), opts)
	pullSetupFlags(cmd.Flags(), &opts.PullOpts)
	dbSetupFlags(cmd.Flags(), &opts.DBOpts)
	return cmd
}

func graphviz(opts *graphOptions) (string, error) {
	if opts.Targets == nil || len(opts.Targets) < 1 || opts.Targets[0] == "" {
		return "", fmt.Errorf("you need to specify at least one target")
	}
	issues, err := loadIssues(db, nil)
	if err != nil {
		return "", errors.Wrap(err, "failed to load issues")
	}

	if err := issues.prepare(false); err != nil {
		return "", errors.Wrap(err, "failed to prepare issues")
	}

	if !opts.ShowClosed {
		issues.HideClosed()
	}
	issues.filterByTargets(opts.Targets)
	if opts.ShowOrphans {
		logger().Warn("--show-orphans is deprecated and will be removed")
	}

	return graphvizRender(issues, opts)
}

func run(opts *runOptions) error {
	logger().Debug("run", zap.Stringer("opts", *opts))
	if !opts.NoPull {
		if err := pull(&opts.PullOpts); err != nil {
			return err
		}
	}

	out, err := graphviz(&opts.GraphOpts)
	if err != nil {
		return err
	}

	var dest io.WriteCloser
	switch opts.Destination {
	case "-":
		dest = os.Stdout
	default:
		var err error
		dest, err = os.Create(opts.Destination)
		if err != nil {
			return err
		}
		defer dest.Close()
	}
	fmt.Fprintln(dest, out)

	return nil
}
