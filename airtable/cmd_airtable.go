package airtable

import (
	"encoding/json"
	"fmt"

	"github.com/brianloveswords/airtable"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"moul.io/depviz/airtabledb"
	"moul.io/depviz/warehouse"
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

	Targets []warehouse.Target `mapstructure:"targets"`
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

	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind flags using Viper", zap.Error(err))
	}
}

func (cmd *airtableCommand) NewCobraCommand(dc map[string]DepvizCommand) *cobra.Command {
	cc := &cobra.Command{
		Use:   "airtable",
		Short: "Upload issue info stored in database to airtable spreadsheets",
	}
	cc.AddCommand(cmd.airtableSyncCommand())
	return cc
}

func (cmd *airtableCommand) airtableSyncCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:   "sync",
		Short: "Upload issue info stored in database to airtable spreadsheets",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			var err error
			if opts.Targets, err = warehouse.ParseTargets(args); err != nil {
				return errors.Wrap(err, "invalid targets")
			}
			return airtableSync(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	return cc
}

// airtableSync pushes issue info to the airtable base specified in opts.
// Repository info is loaded from the targets specified in opts.
func airtableSync(opts *airtableOptions) error {
	if opts.BaseID == "" || opts.Token == "" {
		return fmt.Errorf("missing token or baseid, check '-h'")
	}

	//
	// prepare
	//

	loadedIssues, err := warehouse.Load(db, nil)
	if err != nil {
		return errors.Wrap(err, "failed to load issues")
	}
	loadedIssues = loadedIssues.FilterByTargets(opts.Targets)
	zap.L().Debug("fetch db entries", zap.Int("count", len(loadedIssues)))

	issueFeatures := make([]map[string]warehouse.Feature, airtabledb.NumTables)
	for i := range issueFeatures {
		issueFeatures[i] = make(map[string]warehouse.Feature)
	}

	// Parse the loaded issues into the issueFeature map.
	for _, issue := range loadedIssues {
		// providers
		issueFeatures[airtabledb.ProviderIndex][issue.Repository.Provider.ID] = issue.Repository.Provider

		// labels
		for _, label := range issue.Labels {
			issueFeatures[airtabledb.LabelIndex][label.ID] = label
		}

		// accounts
		if issue.Repository.Owner != nil {
			issueFeatures[airtabledb.AccountIndex][issue.Repository.Owner.ID] = issue.Repository.Owner
		}

		issueFeatures[airtabledb.AccountIndex][issue.Author.ID] = issue.Author
		for _, assignee := range issue.Assignees {
			issueFeatures[airtabledb.AccountIndex][assignee.ID] = assignee
		}
		if issue.Milestone != nil && issue.Milestone.Creator != nil {
			issueFeatures[airtabledb.AccountIndex][issue.Milestone.Creator.ID] = issue.Milestone.Creator
		}

		// repositories
		issueFeatures[airtabledb.RepositoryIndex][issue.Repository.ID] = issue.Repository
		// FIXME: find external repositories based on depends-on links

		// milestones
		if issue.Milestone != nil {
			issueFeatures[airtabledb.MilestoneIndex][issue.Milestone.ID] = issue.Milestone
		}

		// issue
		issueFeatures[airtabledb.IssueIndex][issue.ID] = issue
		// FIXME: find external issues based on depends-on links
	}

	client := airtable.Client{
		APIKey:  opts.Token,
		BaseID:  opts.BaseID,
		Limiter: airtable.RateLimiter(5),
	}

	// cache stores issueFeatures inserted into the airtable base.
	cache := airtabledb.NewDB()

	// Store already existing issueFeatures into the cache.
	for tableKind, tableName := range opts.TableNames {
		table := client.Table(tableName)
		if err := cache.Tables[tableKind].Fetch(table); err != nil {
			return err
		}
	}

	// unmatched stores new issueFeatures (exist in the loaded issues but not the airtable base).
	unmatched := airtabledb.NewDB()

	// Insert new issueFeatures into unmatched and mark altered cache issueFeatures with airtabledb.StateChanged.
	for tableKind, featureMap := range issueFeatures {
		for _, dbEntry := range featureMap {
			matched := false
			dbRecord := dbEntry.ToRecord(cache)
			for idx := 0; idx < cache.Tables[tableKind].Len(); idx++ {
				t := cache.Tables[tableKind]
				if t.GetFieldID(idx) == dbEntry.GetID() {
					if t.RecordsEqual(idx, dbRecord) {
						t.SetState(idx, airtabledb.StateUnchanged)
					} else {
						t.CopyFields(idx, dbRecord)
						t.SetState(idx, airtabledb.StateChanged)
					}
					matched = true
					break
				}
			}
			if !matched {
				unmatched.Tables[tableKind].Append(dbRecord)
			}
		}
	}

	// Add new issueFeatures from unmatched to cache.
	// Then, push new and altered issueFeatures from cache to airtable base.
	for tableKind, tableName := range opts.TableNames {
		table := client.Table(tableName)
		ut := unmatched.Tables[tableKind]
		ct := cache.Tables[tableKind]
		for i := 0; i < ut.Len(); i++ {
			zap.L().Debug("create airtable entry", zap.String("type", tableName), zap.String("entry", ut.StringAt(i)))
			if err := table.Create(ut.GetPtr(i)); err != nil {
				return err
			}
			ut.SetState(i, airtabledb.StateNew)
			ct.Append(ut.Get(i))
		}
		for i := 0; i < ct.Len(); i++ {
			var err error
			switch ct.GetState(i) {
			case airtabledb.StateUnknown:
				err = table.Delete(ct.GetPtr(i))
				zap.L().Debug("delete airtable entry", zap.String("type", tableName), zap.String("entry", ct.StringAt(i)), zap.Error(err))
			case airtabledb.StateChanged:
				err = table.Update(ct.GetPtr(i))
				zap.L().Debug("update airtable entry", zap.String("type", tableName), zap.String("entry", ct.StringAt(i)), zap.Error(err))
			case airtabledb.StateUnchanged:
				zap.L().Debug("unchanged airtable entry", zap.String("type", tableName), zap.String("entry", ct.StringAt(i)), zap.Error(err))
				// do nothing
			case airtabledb.StateNew:
				zap.L().Debug("new airtable entry", zap.String("type", tableName), zap.String("entry", ct.StringAt(i)), zap.Error(err))
				// do nothing
			}
		}
	}

	for tableKind, tableName := range opts.TableNames {
		fmt.Println("-------", tableName)
		ct := cache.Tables[tableKind]
		for i := 0; i < ct.Len(); i++ {
			fmt.Println(ct.GetID(i), airtabledb.StateString[ct.GetState(i)], ct.GetFieldID(i))
		}
	}
	fmt.Println("-------")

	return nil
}
