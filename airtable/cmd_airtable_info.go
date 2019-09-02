package airtable

import (
	"fmt"

	"github.com/brianloveswords/airtable"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/airtablemodel"
	"moul.io/depviz/cli"
)

type InfoOptions struct {
	Airtable Options `mapstructure:"airtable"`
}

type infoCommand struct{ opts InfoOptions }

func (cmd *infoCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	cc := &cobra.Command{
		Use:   "info",
		Short: "Print info about airtable",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			opts.Airtable = GetOptions(commands)
			return Info(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	commands["airtable"].ParseFlags(cc.Flags())
	return cc
}

func (cmd *infoCommand) LoadDefaultOptions() error { return viper.Unmarshal(&cmd.opts) }

func (cmd *infoCommand) ParseFlags(flags *pflag.FlagSet) {
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}

func Info(opts *InfoOptions) error {
	if opts.Airtable.BaseID == "" || opts.Airtable.Token == "" {
		return fmt.Errorf("missing token or baseid, check '-h'")
	}

	if opts.Airtable.RateLimiter == 0 {
		opts.Airtable.RateLimiter = 5
	}
	client := airtable.Client{
		APIKey:  opts.Airtable.Token,
		BaseID:  opts.Airtable.BaseID,
		Limiter: airtable.RateLimiter(opts.Airtable.RateLimiter),
	}

	cache := airtablemodel.NewDB()

	for tableKind, tableName := range opts.Airtable.tableNames() {
		table := client.Table(tableName)
		if err := cache.Tables[tableKind].Fetch(table); err != nil {
			return err
		}
		fmt.Printf("- %s: %d\n", tableName, cache.Tables[tableKind].Len())
	}

	return nil
}
