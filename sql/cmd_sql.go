package sql

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/cli"
)

type Options struct {
	Config  string `mapstructure:"config"`
	Verbose bool   `mapstructure:"verbose"`
}

func (opts Options) Validate() error {
	return nil
}

func (opts Options) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func GetOptions(commands cli.Commands) Options {
	return commands["sql"].(*sqlCommand).opts
}

func Commands() cli.Commands {
	return cli.Commands{
		"sql":      &sqlCommand{},
		"sql dump": &dumpCommand{},
		"sql info": &infoCommand{},
		// FIXME: "sql flush"
	}
}

type sqlCommand struct{ opts Options }

func (cmd *sqlCommand) LoadDefaultOptions() error { return viper.Unmarshal(&cmd.opts) }

func (cmd *sqlCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&cmd.opts.Config, "sql-config", "", "sqlite://$HOME/.depviz.db", "sql connection string")
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}

func (cmd *sqlCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	command := &cobra.Command{
		Use:   "sql",
		Short: "Manager sql",
	}
	command.AddCommand(commands["sql dump"].CobraCommand(commands))
	command.AddCommand(commands["sql info"].CobraCommand(commands))
	return command
}
