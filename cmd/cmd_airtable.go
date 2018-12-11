package main

import (
	"encoding/json"
	"fmt"

	"github.com/brianloveswords/airtable"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/pkg/airtabledb"
	"moul.io/depviz/pkg/repo"
)

type airtableOptions struct {
	IssuesTableName       string `mapstructure:"airtable-issues-table-name"`
	RepositoriesTableName string `mapstructure:"airtable-repositories-table-name"`
	LabelsTableName       string `mapstructure:"airtable-labels-table-name"`
	MilestonesTableName   string `mapstructure:"airtable-milestones-table-name"`
	ProvidersTableName    string `mapstructure:"airtable-providers-table-name"`
	AccountsTableName     string `mapstructure:"airtable-accounts-table-name"`
	BaseID                string `mapstructure:"airtable-base-id"`
	Token                 string `mapstructure:"airtable-token"`
	DestroyInvalidRecords bool   `mapstructure:"airtable-destroy-invalid-records"`
	TableNames            []string

	Targets []repo.Target `mapstructure:"targets"`
}

func (opts airtableOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

type airtableCommand struct {
	opts airtableOptions
}

func (cmd *airtableCommand) LoadDefaultOptions() error {
	if err := viper.Unmarshal(&cmd.opts); err != nil {
		return err
	}
	return nil
}

func (cmd *airtableCommand) ParseFlags(flags *pflag.FlagSet) {
	cmd.opts.TableNames = make([]string, airtabledb.NumTables)

	flags.StringVarP(&cmd.opts.IssuesTableName, "airtable-issues-table-name", "", "Issues and PRs", "Airtable issues table name")
	cmd.opts.TableNames[airtabledb.IssueIndex] = cmd.opts.IssuesTableName
	flags.StringVarP(&cmd.opts.RepositoriesTableName, "airtable-repositories-table-name", "", "Repositories", "Airtable repositories table name")
	cmd.opts.TableNames[airtabledb.RepositoryIndex] = cmd.opts.RepositoriesTableName
	flags.StringVarP(&cmd.opts.AccountsTableName, "airtable-accounts-table-name", "", "Accounts", "Airtable accounts table name")
	cmd.opts.TableNames[airtabledb.AccountIndex] = cmd.opts.AccountsTableName
	flags.StringVarP(&cmd.opts.LabelsTableName, "airtable-labels-table-name", "", "Labels", "Airtable labels table name")
	cmd.opts.TableNames[airtabledb.LabelIndex] = cmd.opts.LabelsTableName
	flags.StringVarP(&cmd.opts.MilestonesTableName, "airtable-milestones-table-name", "", "Milestones", "Airtable milestones table nfame")
	cmd.opts.TableNames[airtabledb.MilestoneIndex] = cmd.opts.MilestonesTableName
	flags.StringVarP(&cmd.opts.ProvidersTableName, "airtable-providers-table-name", "", "Providers", "Airtable providers table name")
	cmd.opts.TableNames[airtabledb.ProviderIndex] = cmd.opts.ProvidersTableName
	flags.StringVarP(&cmd.opts.BaseID, "airtable-base-id", "", "", "Airtable base ID")
	flags.StringVarP(&cmd.opts.Token, "airtable-token", "", "", "Airtable token")
	flags.BoolVarP(&cmd.opts.DestroyInvalidRecords, "airtable-destroy-invalid-records", "", false, "Destroy invalid records")
	viper.BindPFlags(flags)
}

func (cmd *airtableCommand) NewCobraCommand(dc map[string]DepvizCommand) *cobra.Command {
	cc := &cobra.Command{
		Use: "airtable",
	}
	cc.AddCommand(cmd.airtableSyncCommand())
	return cc
}

func (cmd *airtableCommand) airtableSyncCommand() *cobra.Command {
	cc := &cobra.Command{
		Use: "sync",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			var err error
			if opts.Targets, err = repo.ParseTargets(args); err != nil {
				return errors.Wrap(err, "invalid targets")
			}
			return airtableSync(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	return cc
}

func airtableSync(opts *airtableOptions) error {
	if opts.BaseID == "" || opts.Token == "" {
		return fmt.Errorf("missing token or baseid, check '-h'")
	}

	//
	// prepare
	//

	// load issues
	issues, err := loadIssues(nil)
	if err != nil {
		return errors.Wrap(err, "failed to load issues")
	}
	filtered := issues.FilterByTargets(opts.Targets)
	zap.L().Debug("fetch db entries", zap.Int("count", len(filtered)))

	// unique entries
	features := make([]map[string]repo.Feature, airtabledb.NumTables)

	for _, issue := range filtered {
		// providers
		//providerMap[issue.Repository.Provider.ID] = issue.Repository.Provider
		features[airtabledb.ProviderIndex][issue.Repository.Provider.ID] = issue.Repository.Provider

		// labels
		for _, label := range issue.Labels {
			//labelMap[label.ID] = label
			features[airtabledb.LabelIndex][label.ID] = label
		}

		// accounts
		if issue.Repository.Owner != nil {
			//accountMap[issue.Repository.Owner.ID] = issue.Repository.Owner
			features[airtabledb.AccountIndex][issue.Repository.Owner.ID] = issue.Repository.Owner
		}
		//accountMap[issue.Author.ID] = issue.Author
		features[airtabledb.AccountIndex][issue.Author.ID] = issue.Author
		for _, assignee := range issue.Assignees {
			//accountMap[assignee.ID] = assignee
			features[airtabledb.AccountIndex][assignee.ID] = assignee
		}
		if issue.Milestone != nil && issue.Milestone.Creator != nil {
			//accountMap[issue.Milestone.Creator.ID] = issue.Milestone.Creator
			features[airtabledb.AccountIndex][issue.Milestone.Creator.ID] = issue.Milestone.Creator
		}

		// repositories
		//repositoryMap[issue.Repository.ID] = issue.Repository
		features[airtabledb.RepositoryIndex][issue.Repository.ID] = issue.Repository
		// FIXME: find external repositories based on depends-on links

		// milestones
		if issue.Milestone != nil {
			//milestoneMap[issue.Milestone.ID] = issue.Milestone
			features[airtabledb.MilestoneIndex][issue.Milestone.ID] = issue.Milestone
		}

		// issue
		//issueMap[issue.ID] = issue
		features[airtabledb.IssueIndex][issue.ID] = issue
		// FIXME: find external issues based on depends-on links
	}

	// init client
	at := airtable.Client{
		APIKey:  opts.Token,
		BaseID:  opts.BaseID,
		Limiter: airtable.RateLimiter(5),
	}

	// fetch remote data
	cache := airtabledb.NewDB()
	for tableKind, tableName := range opts.TableNames {
		table := at.Table(tableName)
		records := cache.Tables[tableKind]
		if err := table.List(&records, &airtable.Options{}); err != nil {
			return err
		}
	}

	unmatched := airtabledb.NewDB()

	//
	// compute fields
	//

	for tableKind, featureMap := range features {
		for _, dbEntry := range featureMap {
			matched := false
			dbRecord := dbEntry.ToRecord(cache)
			for idx, atEntry := range cache.Tables[tableKind] {
				if atEntry.Fields.ID == dbEntry.GetID() {
					if atEntry.Equals(dbRecord) {
						cache.Tables[tableKind][idx].State = airtabledb.StateUnchanged
					} else {
						cache.Tables[tableKind][idx].Fields = dbRecord.Fields
						cache.Tables[tableKind][idx].State = airtabledb.StateChanged
					}
					matched = true
					break
				}
			}
			if !matched {
				unmatched.Tables[tableKind] = append(unmatched.Tables[tableKind], dbRecord)
			}
		}
	}

	//
	// update airtable
	//
	for tableKind, tableName := range opts.TableNames {
		table := at.Table(tableName)
		for _, entry := range unmatched.Tables[tableKind] {
			zap.L().Debug("create airtable entry", zap.String("type", tableName), zap.Stringer("entry", entry))
			if err := table.Create(&entry); err != nil {
				return err
			}
			entry.State = airtabledb.StateNew
			cache.Tables[tableKind] = append(cache.Tables[tableKind], entry)
		}
		for _, entry := range cache.Tables[tableKind] {
			var err error
			switch entry.State {
			case airtabledb.StateUnknown:
				err = table.Delete(&entry)
				zap.L().Debug("delete airtable entry", zap.String("type", tableName), zap.Stringer("entry", entry), zap.Error(err))
			case airtabledb.StateChanged:
				err = table.Update(&entry)
				zap.L().Debug("update airtable entry", zap.String("type", tableName), zap.Stringer("entry", entry), zap.Error(err))
			case airtabledb.StateUnchanged:
				zap.L().Debug("unchanged airtable entry", zap.String("type", tableName), zap.Stringer("entry", entry), zap.Error(err))
				// do nothing
			case airtabledb.StateNew:
				zap.L().Debug("new airtable entry", zap.String("type", tableName), zap.Stringer("entry", entry), zap.Error(err))
				// do nothing
			}
		}
	}

	//
	// debug
	//
	for tableKind, tableName := range opts.TableNames {
		fmt.Println("-------", tableName)
		for _, entry := range cache.Tables[tableKind] {
			fmt.Println(entry.ID, airtabledb.StateString[entry.State], entry.Fields.ID)
		}
	}
	fmt.Println("-------")

	return nil
}
