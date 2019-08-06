package sql

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"moul.io/depviz/cli"
)

type dumpOptions struct {
	sql Options `mapstructure:"sql"`
	// FIXME: add --anonymize
}

type dumpCommand struct{ opts dumpOptions }

func (cmd *dumpCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	cc := &cobra.Command{
		Use:   "dump",
		Short: "Print all issues stored in the database, formatted as JSON",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			opts.sql = GetOptions(commands)
			return runDump(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	commands["sql"].ParseFlags(cc.Flags())
	return cc
}
func (cmd *dumpCommand) LoadDefaultOptions() error { return viper.Unmarshal(&cmd.opts) }
func (cmd *dumpCommand) ParseFlags(flags *pflag.FlagSet) {
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}
func runDump(opts *dumpOptions) error {
	db, err := FromOpts(&opts.sql)
	if err != nil {
		return err
	}

	issues, err := LoadAllIssues(db)
	if err != nil {
		return err
	}

	out, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
