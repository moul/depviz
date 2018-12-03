package main

import (
	"encoding/json"
	"moul.io/depviz/pkg/repo"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type runOptions struct {
	GraphOptions    graphOptions `mapstructure:"graph"`
	PullOptions     pullOptions  `mapstructure:"pull"`
	AdditionalPulls []string     `mapstructure:"additional-pulls"`
	NoPull          bool         `mapstructure:"no-pull"`
}

var globalRunOptions runOptions

func (opts runOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func runSetupFlags(flags *pflag.FlagSet, opts *runOptions) {
	flags.BoolVarP(&opts.NoPull, "no-pull", "", false, "do not pull new issues before running")
	flags.StringSliceVarP(&opts.AdditionalPulls, "additional-pulls", "", []string{}, "additional pull that won't necessarily be displayed on the graph")
	viper.BindPFlags(flags)
}

func newRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "run",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := globalRunOptions
			opts.GraphOptions = globalGraphOptions
			opts.PullOptions = globalPullOptions

			targets, err := repo.ParseTargets(args)
			if err != nil {
				return errors.Wrap(err, "invalid targets")
			}
			additionalPulls, err := repo.ParseTargets(opts.AdditionalPulls)
			if err != nil {
				return errors.Wrap(err, "invalid targets")
			}
			opts.PullOptions.Targets = append(targets, additionalPulls...)
			opts.GraphOptions.Targets = targets
			return run(&opts)
		},
	}
	runSetupFlags(cmd.Flags(), &globalRunOptions)
	graphSetupFlags(cmd.Flags(), &globalGraphOptions)
	pullSetupFlags(cmd.Flags(), &globalPullOptions)
	return cmd
}

func run(opts *runOptions) error {
	if !opts.NoPull {
		if err := pullAndCompute(&opts.PullOptions); err != nil {
			return errors.Wrap(err, "failed to pull")
		}
	}
	if err := graph(&opts.GraphOptions); err != nil {
		return errors.Wrap(err, "failed to graph")
	}
	return nil
}
