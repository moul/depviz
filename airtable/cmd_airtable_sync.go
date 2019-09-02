package airtable

import (
	"fmt"
	"log"

	"github.com/brianloveswords/airtable"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/airtabledb"
	"moul.io/depviz/airtablemodel"
	"moul.io/depviz/cli"
	"moul.io/depviz/compute"
	"moul.io/depviz/model"
	"moul.io/depviz/sql"
	"moul.io/multipmuri"
)

type SyncOptions struct {
	Airtable              Options             `mapstructure:"airtable"`
	SQL                   sql.Options         `mapstructure:"sql"`     // inherited with sql.GetOptions()
	Targets               []multipmuri.Entity `mapstructure:"targets"` // parsed from Args
	DestroyInvalidRecords bool                `mapstructure:"airtable-destroy-invalid-records"`
}

type syncCommand struct{ opts SyncOptions }

func (cmd *syncCommand) CobraCommand(commands cli.Commands) *cobra.Command {
	cc := &cobra.Command{
		Use:   "sync",
		Short: "Upload issue info stored in database to airtable spreadsheets",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			targets, err := model.ParseTargets(args)
			if err != nil {
				return err
			}
			opts.Targets = targets
			opts.SQL = sql.GetOptions(commands)
			opts.Airtable = GetOptions(commands)
			return Sync(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	commands["airtable"].ParseFlags(cc.Flags())
	commands["sql"].ParseFlags(cc.Flags())
	return cc
}

func (cmd *syncCommand) LoadDefaultOptions() error { return viper.Unmarshal(&cmd.opts) }

func (cmd *syncCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.BoolVarP(&cmd.opts.DestroyInvalidRecords, "airtable-destroy-invalid-records", "", false, "Destroy invalid records")

	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind viper flags", zap.Error(err))
	}
}

//
// implementation
//

// airtableSync pushes issue info to the airtable base specified in opts.
// Repository info is loaded from the targets specified in opts.
func Sync(opts *SyncOptions) error {
	tableNames := make([]string, airtablemodel.NumTables)
	tableNames[airtablemodel.AccountIndex] = opts.Airtable.AccountsTableName
	tableNames[airtablemodel.IssueIndex] = opts.Airtable.IssuesTableName
	tableNames[airtablemodel.LabelIndex] = opts.Airtable.LabelsTableName
	tableNames[airtablemodel.MilestoneIndex] = opts.Airtable.MilestonesTableName
	tableNames[airtablemodel.ProviderIndex] = opts.Airtable.ProvidersTableName
	tableNames[airtablemodel.RepositoryIndex] = opts.Airtable.RepositoriesTableName

	if opts.Airtable.BaseID == "" || opts.Airtable.Token == "" {
		return fmt.Errorf("missing token or baseid, check '-h'")
	}

	//
	// prepare
	//
	db, err := sql.FromOpts(&opts.SQL)
	if err != nil {
		return err
	}

	loadedIssues, err := sql.LoadAllIssues(db)
	if err != nil {
		return errors.Wrap(err, "failed to load issues")
	}
	zap.L().Debug("fetch db entries", zap.Int("count", len(loadedIssues)))

	// compute and filter issues
	computed := compute.Compute(loadedIssues)
	computed.FilterByTargets(opts.Targets)
	zap.L().Debug("fetch db entries", zap.Int("count", len(computed.Issues())))

	issueFeatures := make([]map[string]model.Feature, airtablemodel.NumTables)
	for i := range issueFeatures {
		issueFeatures[i] = make(map[string]model.Feature)
	}

	// Parse the loaded issues into the issueFeature map.
	for _, issue := range computed.Issues() {
		if issue.Hidden {
			continue
		}
		// providers
		issueFeatures[airtablemodel.ProviderIndex][issue.Repository.Provider.ID] = issue.Repository.Provider

		// labels
		for _, label := range issue.Labels {
			issueFeatures[airtablemodel.LabelIndex][label.ID] = label
		}

		// accounts
		if issue.Repository.Owner != nil {
			issueFeatures[airtablemodel.AccountIndex][issue.Repository.Owner.ID] = issue.Repository.Owner
		}

		issueFeatures[airtablemodel.AccountIndex][issue.Author.ID] = issue.Author
		for _, assignee := range issue.Assignees {
			issueFeatures[airtablemodel.AccountIndex][assignee.ID] = assignee
		}
		if issue.Milestone != nil && issue.Milestone.Creator != nil {
			issueFeatures[airtablemodel.AccountIndex][issue.Milestone.Creator.ID] = issue.Milestone.Creator
		}

		// repositories
		issueFeatures[airtablemodel.RepositoryIndex][issue.Repository.ID] = issue.Repository
		// FIXME: find external repositories based on depends-on links

		// milestones
		if issue.Milestone != nil {
			issueFeatures[airtablemodel.MilestoneIndex][issue.Milestone.ID] = issue.Milestone
		}

		// issue
		issueFeatures[airtablemodel.IssueIndex][issue.ID] = issue
		// FIXME: find external issues based on depends-on links
	}

	if opts.Airtable.RateLimiter == 0 {
		opts.Airtable.RateLimiter = 5
	}
	client := airtable.Client{
		APIKey:  opts.Airtable.Token,
		BaseID:  opts.Airtable.BaseID,
		Limiter: airtable.RateLimiter(opts.Airtable.RateLimiter),
	}

	// cache stores issueFeatures inserted into the airtable base.
	cache := airtablemodel.NewDB()

	// Store already existing issueFeatures into the cache.
	for tableKind, tableName := range tableNames {
		table := client.Table(tableName)
		if err := cache.Tables[tableKind].Fetch(table); err != nil {
			return err
		}
	}

	// unmatched stores new issueFeatures (exist in the loaded issues but not the airtable base).
	unmatched := airtablemodel.NewDB()

	// Add new issueFeatures from unmatched to cache.
	// Then, push new and altered issueFeatures from cache to airtable base.
	for tableKind, tableName := range tableNames {
		ut := unmatched.Tables[tableKind]
		table := client.Table(tableName)

		for _, dbEntry := range issueFeatures[tableKind] {
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
				ut.Append(dbRecord)
			}
		}

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
				if opts.DestroyInvalidRecords {
					err = table.Delete(ct.GetPtr(i))
					zap.L().Debug("delete airtable entry", zap.String("type", tableName), zap.String("entry", ct.StringAt(i)), zap.Error(err))
				} else {
					zap.L().Debug("unknown airtable entry, doing nothing", zap.String("type", tableName), zap.String("entry", ct.StringAt(i)))
				}
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

	for tableKind, tableName := range tableNames {
		ct := cache.Tables[tableKind]
		log.Println(tableName)
		for i := 0; i < ct.Len(); i++ {
			log.Println("-", ct.GetID(i), airtabledb.StateString[ct.GetState(i)], ct.GetFieldID(i))
		}
	}

	return nil
}
