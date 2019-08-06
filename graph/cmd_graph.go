package graph // import "moul.io/depviz/graph"

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/cli"
	"moul.io/depviz/sql"
	"moul.io/multipmuri"
)

func GetOptions(commands cli.Commands) Options {
	return commands["graph"].(*graphCommand).opts
}

func Commands() cli.Commands {
	return cli.Commands{"graph": &graphCommand{}}
}

type graphCommand struct {
	opts Options
}

func (cmd *graphCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	cc := &cobra.Command{
		Use:   "graph",
		Short: "Output graph of relationships between all issues stored in database",
		Args:  cobra.MinimumNArgs(1), // FIXME: if no args, then graph the whole database
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			opts.SQL = sql.GetOptions(commands)
			opts.Targets = []multipmuri.Entity{}
			defaultContext := multipmuri.NewGitHubService("")
			for _, arg := range args {
				entity, err := defaultContext.RelDecodeString(arg)
				if err != nil {
					return err
				}
				opts.Targets = append(opts.Targets, entity)
			}
			return Graph(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	commands["sql"].ParseFlags(cc.Flags())
	return cc
}

func (cmd *graphCommand) LoadDefaultOptions() error {
	return viper.Unmarshal(&cmd.opts)
}

func (cmd *graphCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&cmd.opts.ShowClosed, "show-closed", "", false, "show closed issues/PRs")
	flags.BoolVarP(&cmd.opts.ShowOrphans, "show-orphans", "", false, "show orphans issues/PRs")
	flags.BoolVarP(&cmd.opts.ShowPRs, "show-prs", "", false, "show PRs")
	flags.BoolVarP(&cmd.opts.ShowAllRelated, "show-all-related", "", false, "show related from other repos")
	flags.BoolVarP(&cmd.opts.Vertical, "vertical", "", false, "display graph vertically instead of horizontally")
	flags.BoolVarP(&cmd.opts.NoPert, "no-pert", "", false, "do not compute Pert")
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}
