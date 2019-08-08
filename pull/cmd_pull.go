package pull // import "moul.io/depviz/pull"

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
	return commands["pull"].(*pullCommand).opts
}

func Commands() cli.Commands {
	return cli.Commands{"pull": &pullCommand{}}
}

type pullCommand struct {
	opts Options
}

func (cmd *pullCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	cc := &cobra.Command{
		Use:   "pull",
		Short: "Pull issues and update database without outputting graph",
		Args:  cobra.MinimumNArgs(1),
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
			return Pull(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	commands["sql"].ParseFlags(cc.Flags())
	return cc
}

func (cmd *pullCommand) LoadDefaultOptions() error {
	return viper.Unmarshal(&cmd.opts)
}

func (cmd *pullCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&cmd.opts.GithubToken, "github-token", "", "", "GitHub Token with 'issues' access")
	flags.StringVarP(&cmd.opts.GitlabToken, "gitlab-token", "", "", "GitLab Token with 'issues' access")
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}
