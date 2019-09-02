package web // import "moul.io/depviz/web"

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/cli"
	"moul.io/depviz/sql"
)

func GetOptions(commands cli.Commands) Options {
	return commands["web"].(*webCommand).opts
}

func Commands() cli.Commands {
	return cli.Commands{"web": &webCommand{}}
}

type webCommand struct {
	opts Options
}

func (cmd *webCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	cc := &cobra.Command{
		Use:   "web",
		Short: "Output web of relationships between all issues stored in database",
		Args:  cobra.MaximumNArgs(0),
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			opts.SQL = sql.GetOptions(commands)
			return Web(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	commands["sql"].ParseFlags(cc.Flags())
	return cc
}

func (cmd *webCommand) LoadDefaultOptions() error {
	return viper.Unmarshal(&cmd.opts)
}

func (cmd *webCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&cmd.opts.Bind, "bind", "b", ":2020", "HTTP server bind address")
	flags.BoolVarP(&cmd.opts.GenDoc, "gendoc", "", false, "generate Markdown documentation and exit")
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}
