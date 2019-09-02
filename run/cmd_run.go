package run // import "moul.io/depviz/run"

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/cli"
	"moul.io/depviz/graph"
	"moul.io/depviz/model"
	"moul.io/depviz/pull"
	"moul.io/depviz/sql"
)

type Options struct {
	Graph graph.Options
	Pull  pull.Options
}

func (opts Options) Validate() error {
	if err := opts.Graph.Validate(); err != nil {
		return err
	}
	if err := opts.Pull.Validate(); err != nil {
		return err
	}
	return nil
}

func GetOptions(commands cli.Commands) Options {
	return commands["run"].(*runCommand).opts
}

func Commands() cli.Commands {
	return cli.Commands{"run": &runCommand{}}
}

type runCommand struct {
	opts Options
}

func (cmd *runCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	cc := &cobra.Command{
		Use:   "run",
		Short: "'pull' + 'graph' in a unique command",
		Args: func(c *cobra.Command, args []string) error {
			// FIXME: if no args, then run the whole database
			if err := cobra.MinimumNArgs(1)(c, args); err != nil {
				return err
			}
			return nil
		},
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			opts.Pull = pull.GetOptions(commands)
			opts.Graph = graph.GetOptions(commands)
			opts.Pull.SQL = sql.GetOptions(commands)
			opts.Graph.SQL = opts.Pull.SQL
			targets, err := model.ParseTargets(args)
			if err != nil {
				return err
			}
			opts.Pull.Targets = targets
			opts.Graph.Targets = targets
			if err := opts.Validate(); err != nil {
				return err
			}
			return Run(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	commands["sql"].ParseFlags(cc.Flags())
	commands["graph"].ParseFlags(cc.Flags())
	commands["pull"].ParseFlags(cc.Flags())
	return cc
}

func (cmd *runCommand) LoadDefaultOptions() error {
	return viper.Unmarshal(&cmd.opts)
}

func (cmd *runCommand) ParseFlags(flags *pflag.FlagSet) {
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}

func Run(opts *Options) error {
	if err := pull.Pull(&opts.Pull); err != nil {
		return err
	}
	graph, err := graph.Graph(&opts.Graph)
	if err != nil {
		return err
	}
	fmt.Println(graph)
	return nil
}
