package workflow // import "moul.io/depviz/workflow"
/*
import (
	"encoding/json"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/model"
)

type runOptions struct {
	GraphOptions    graphOptions `mapstructure:"graph"`
	PullOptions     pullOptions  `mapstructure:"pull"`
	AdditionalPulls []string     `mapstructure:"additional-pulls"`
	NoPull          bool         `mapstructure:"no-pull"`
}

func (opts runOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

type runCommand struct {
	opts runOptions
}

func (cmd *runCommand) LoadDefaultOptions() error {
	if err := viper.Unmarshal(&cmd.opts); err != nil {
		return err
	}
	return nil
}

func (cmd *runCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&cmd.opts.NoPull, "no-pull", "", false, "do not pull new issues before running")
	flags.StringSliceVarP(&cmd.opts.AdditionalPulls, "additional-pulls", "", []string{}, "additional pull that won't necessarily be displayed on the graph")
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind flags using Viper", zap.Error(err))
	}
}

func (cmd *runCommand) NewCobraCommand(dc map[string]DepvizCommand) *cobra.Command {
	cc := &cobra.Command{
		Use:   "run",
		Short: "Pull issues, update database, and output a graph of relationships between issues",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			opts.GraphOptions = dc["graph"].(*graphCommand).opts
			opts.PullOptions = dc["pull"].(*pullCommand).opts

			targets, err := model.ParseTargets(args)
			if err != nil {
				return errors.Wrap(err, "invalid targets")
			}
			additionalPulls, err := model.ParseTargets(opts.AdditionalPulls)
			if err != nil {
				return errors.Wrap(err, "invalid targets")
			}
			opts.PullOptions.Targets = append(targets, additionalPulls...)
			opts.GraphOptions.Targets = targets
			return run(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	dc["graph"].ParseFlags(cc.Flags())
	dc["pull"].ParseFlags(cc.Flags())
	return cc
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
*/
