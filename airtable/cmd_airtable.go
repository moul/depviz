package airtable // import "moul.io/depviz/airtable"

import (
	"encoding/json"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/airtablemodel"
	"moul.io/depviz/cli"
)

//
// Options
//

type Options struct {
	IssuesTableName       string `mapstructure:"airtable-issues-table-name"`
	RepositoriesTableName string `mapstructure:"airtable-repositories-table-name"`
	LabelsTableName       string `mapstructure:"airtable-labels-table-name"`
	MilestonesTableName   string `mapstructure:"airtable-milestones-table-name"`
	ProvidersTableName    string `mapstructure:"airtable-providers-table-name"`
	AccountsTableName     string `mapstructure:"airtable-accounts-table-name"`
	BaseID                string `mapstructure:"airtable-base-id"`
	Token                 string `mapstructure:"airtable-token"`
	RateLimiter           int    `mapstructure:"airtable-ratelimiter"`
}

func (opts Options) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func (opts *Options) tableNames() []string {
	tableNames := make([]string, airtablemodel.NumTables)
	tableNames[airtablemodel.AccountIndex] = opts.AccountsTableName
	tableNames[airtablemodel.IssueIndex] = opts.IssuesTableName
	tableNames[airtablemodel.LabelIndex] = opts.LabelsTableName
	tableNames[airtablemodel.MilestoneIndex] = opts.MilestonesTableName
	tableNames[airtablemodel.ProviderIndex] = opts.ProvidersTableName
	tableNames[airtablemodel.RepositoryIndex] = opts.RepositoriesTableName
	return tableNames
}

//
// Command
//

func GetOptions(commands cli.Commands) Options {
	return commands["airtable"].(*airtableCommand).opts
}

func Commands() cli.Commands {
	return cli.Commands{
		"airtable":      &airtableCommand{},
		"airtable sync": &syncCommand{},
		"airtable info": &infoCommand{},
	}
}

type airtableCommand struct{ opts Options }

func (cmd *airtableCommand) LoadDefaultOptions() error { return viper.Unmarshal(&cmd.opts) }

func (cmd *airtableCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&cmd.opts.IssuesTableName, "airtable-issues-table-name", "", "Issues and PRs", "Airtable issues table name")
	flags.StringVarP(&cmd.opts.RepositoriesTableName, "airtable-repositories-table-name", "", "Repositories", "Airtable repositories table name")
	flags.StringVarP(&cmd.opts.AccountsTableName, "airtable-accounts-table-name", "", "Accounts", "Airtable accounts table name")
	flags.StringVarP(&cmd.opts.LabelsTableName, "airtable-labels-table-name", "", "Labels", "Airtable labels table name")
	flags.StringVarP(&cmd.opts.MilestonesTableName, "airtable-milestones-table-name", "", "Milestones", "Airtable milestones table nfame")
	flags.StringVarP(&cmd.opts.ProvidersTableName, "airtable-providers-table-name", "", "Providers", "Airtable providers table name")
	flags.StringVarP(&cmd.opts.BaseID, "airtable-base-id", "", "", "Airtable base ID")
	flags.StringVarP(&cmd.opts.Token, "airtable-token", "", "", "Airtable token")

	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}

func (cmd *airtableCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	command := &cobra.Command{
		Use:   "airtable",
		Short: "Manager airtable",
	}
	command.AddCommand(commands["airtable sync"].CobraCommand(commands))
	command.AddCommand(commands["airtable info"].CobraCommand(commands))
	return command
}
