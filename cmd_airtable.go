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

	Targets []Target `mapstructure:"targets"`
}

var globalAirtableOptions airtableOptions

func (opts airtableOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func airtableSetupFlags(flags *pflag.FlagSet, opts *airtableOptions) {
	flags.StringVarP(&opts.IssuesTableName, "airtable-issues-table-name", "", "Issues and PRs", "Airtable issues table name")
	flags.StringVarP(&opts.RepositoriesTableName, "airtable-repositories-table-name", "", "Repositories", "Airtable repositories table name")
	flags.StringVarP(&opts.AccountsTableName, "airtable-accounts-table-name", "", "Accounts", "Airtable accounts table name")
	flags.StringVarP(&opts.LabelsTableName, "airtable-labels-table-name", "", "Labels", "Airtable labels table name")
	flags.StringVarP(&opts.MilestonesTableName, "airtable-milestones-table-name", "", "Milestones", "Airtable milestones table nfame")
	flags.StringVarP(&opts.ProvidersTableName, "airtable-providers-table-name", "", "Providers", "Airtable providers table name")
	flags.StringVarP(&opts.BaseID, "airtable-base-id", "", "", "Airtable base ID")
	flags.StringVarP(&opts.Token, "airtable-token", "", "", "Airtable token")
	flags.BoolVarP(&opts.DestroyInvalidRecords, "airtable-destroy-invalid-records", "", false, "Destroy invalid records")
	viper.BindPFlags(flags)
}

func newAirtableCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "airtable",
	}
	cmd.AddCommand(newAirtableSyncCommand())
	return cmd
}

func newAirtableSyncCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := globalAirtableOptions
			var err error
			if opts.Targets, err = ParseTargets(args); err != nil {
				return errors.Wrap(err, "invalid targets")
			}
			return airtableSync(&opts)
		},
	}
	airtableSetupFlags(cmd.Flags(), &globalAirtableOptions)
	return cmd
}

type AirtableDB struct {
	Providers    ProviderRecords
	Labels       LabelRecords
	Accounts     AccountRecords
	Repositories RepositoryRecords
	Milestones   MilestoneRecords
	Issues       IssueRecords
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
	logger().Debug("fetch db entries", zap.Int("count", len(filtered)))

	// unique entries
	var (
		providerMap   = make(map[string]*Provider)
		labelMap      = make(map[string]*Label)
		accountMap    = make(map[string]*Account)
		repositoryMap = make(map[string]*Repository)
		milestoneMap  = make(map[string]*Milestone)
		issueMap      = make(map[string]*Issue)
	)
	for _, issue := range filtered {
		// providers
		providerMap[issue.Repository.Provider.ID] = issue.Repository.Provider

		// labels
		for _, label := range issue.Labels {
			labelMap[label.ID] = label
		}

		// accounts
		if issue.Repository.Owner != nil {
			accountMap[issue.Repository.Owner.ID] = issue.Repository.Owner
		}
		accountMap[issue.Author.ID] = issue.Author
		for _, assignee := range issue.Assignees {
			accountMap[assignee.ID] = assignee
		}
		if issue.Milestone != nil && issue.Milestone.Creator != nil {
			accountMap[issue.Milestone.Creator.ID] = issue.Milestone.Creator
		}

		// repositories
		repositoryMap[issue.Repository.ID] = issue.Repository
		// FIXME: find external repositories based on depends-on links

		// milestones
		if issue.Milestone != nil {
			milestoneMap[issue.Milestone.ID] = issue.Milestone
		}

		// issue
		issueMap[issue.ID] = issue
		// FIXME: find external issues based on depends-on links
	}

	// init client
	at := airtable.Client{
		APIKey:  opts.Token,
		BaseID:  opts.BaseID,
		Limiter: airtable.RateLimiter(5),
	}

	// fetch remote data
	cache := AirtableDB{}
	table := at.Table(opts.ProvidersTableName)
	if err := table.List(&cache.Providers, &airtable.Options{}); err != nil {
		return err
	}
	table = at.Table(opts.LabelsTableName)
	if err := table.List(&cache.Labels, &airtable.Options{}); err != nil {
		return err
	}
	table = at.Table(opts.AccountsTableName)
	if err := table.List(&cache.Accounts, &airtable.Options{}); err != nil {
		return err
	}
	table = at.Table(opts.RepositoriesTableName)
	if err := table.List(&cache.Repositories, &airtable.Options{}); err != nil {
		return err
	}
	table = at.Table(opts.MilestonesTableName)
	if err := table.List(&cache.Milestones, &airtable.Options{}); err != nil {
		return err
	}
	table = at.Table(opts.IssuesTableName)
	if err := table.List(&cache.Issues, &airtable.Options{}); err != nil {
		return err
	}

	unmatched := AirtableDB{
		Providers:    ProviderRecords{},
		Labels:       LabelRecords{},
		Accounts:     AccountRecords{},
		Repositories: RepositoryRecords{},
		Milestones:   MilestoneRecords{},
		Issues:       IssueRecords{},
	}

	//
	// compute fields
	//

	// providers
	for _, dbEntry := range providerMap {
		matched := false
		dbRecord := dbEntry.ToRecord(cache)
		for idx, atEntry := range cache.Providers {
			if atEntry.Fields.ID == dbEntry.ID {
				if atEntry.Equals(dbRecord) {
					cache.Providers[idx].State = airtableStateUnchanged
				} else {
					cache.Providers[idx].Fields = dbRecord.Fields
					cache.Providers[idx].State = airtableStateChanged
				}
				matched = true
				break
			}
		}
		if !matched {
			unmatched.Providers = append(unmatched.Providers, *dbRecord)
		}
	}

	// labels
	for _, dbEntry := range labelMap {
		matched := false
		dbRecord := dbEntry.ToRecord(cache)
		for idx, atEntry := range cache.Labels {
			if atEntry.Fields.ID == dbEntry.ID {
				if atEntry.Equals(dbRecord) {
					cache.Labels[idx].State = airtableStateUnchanged
				} else {
					cache.Labels[idx].Fields = dbRecord.Fields
					cache.Labels[idx].State = airtableStateChanged
				}
				matched = true
				break
			}
		}
		if !matched {
			unmatched.Labels = append(unmatched.Labels, *dbRecord)
		}
	}

	// accounts
	for _, dbEntry := range accountMap {
		matched := false
		dbRecord := dbEntry.ToRecord(cache)
		for idx, atEntry := range cache.Accounts {
			if atEntry.Fields.ID == dbEntry.ID {
				if atEntry.Equals(dbRecord) {
					cache.Accounts[idx].State = airtableStateUnchanged
				} else {
					cache.Accounts[idx].Fields = dbRecord.Fields
					cache.Accounts[idx].State = airtableStateChanged
				}
				matched = true
				break
			}
		}
		if !matched {
			unmatched.Accounts = append(unmatched.Accounts, *dbRecord)
		}
	}

	// repositories
	for _, dbEntry := range repositoryMap {
		matched := false
		dbRecord := dbEntry.ToRecord(cache)
		for idx, atEntry := range cache.Repositories {
			if atEntry.Fields.ID == dbEntry.ID {
				if atEntry.Equals(dbRecord) {
					cache.Repositories[idx].State = airtableStateUnchanged
				} else {
					cache.Repositories[idx].Fields = dbRecord.Fields
					cache.Repositories[idx].State = airtableStateChanged
				}
				matched = true
				break
			}
		}
		if !matched {
			unmatched.Repositories = append(unmatched.Repositories, *dbRecord)
		}
	}

	// milestones
	for _, dbEntry := range milestoneMap {
		matched := false
		dbRecord := dbEntry.ToRecord(cache)
		for idx, atEntry := range cache.Milestones {
			if atEntry.Fields.ID == dbEntry.ID {
				if atEntry.Equals(dbRecord) {
					cache.Milestones[idx].State = airtableStateUnchanged
				} else {
					cache.Milestones[idx].Fields = dbRecord.Fields
					cache.Milestones[idx].State = airtableStateChanged
				}
				matched = true
				break
			}
		}
		if !matched {
			unmatched.Milestones = append(unmatched.Milestones, *dbRecord)
		}
	}

	// issues
	for _, dbEntry := range issueMap {
		matched := false
		dbRecord := dbEntry.ToRecord(cache)
		for idx, atEntry := range cache.Issues {
			if atEntry.Fields.ID == dbEntry.ID {
				if atEntry.Equals(dbRecord) {
					cache.Issues[idx].State = airtableStateUnchanged
				} else {
					cache.Issues[idx].Fields = dbRecord.Fields
					cache.Issues[idx].State = airtableStateChanged
				}
				matched = true
				break
			}
		}
		if !matched {
			unmatched.Issues = append(unmatched.Issues, *dbRecord)
		}
	}

	//
	// update airtable
	//

	// providers
	table = at.Table(opts.ProvidersTableName)
	for _, entry := range unmatched.Providers {
		logger().Debug("create airtable entry", zap.String("type", "provider"), zap.Stringer("entry", entry))
		if err := table.Create(&entry); err != nil {
			return err
		}
		entry.State = airtableStateNew
		cache.Providers = append(cache.Providers, entry)
	}
	for _, entry := range cache.Providers {
		var err error
		switch entry.State {
		case airtableStateUnknown:
			err = table.Delete(&entry)
			logger().Debug("delete airtable entry", zap.String("type", "provider"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateChanged:
			err = table.Update(&entry)
			logger().Debug("update airtable entry", zap.String("type", "provider"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateUnchanged:
			logger().Debug("unchanged airtable entry", zap.String("type", "provider"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		case airtableStateNew:
			logger().Debug("new airtable entry", zap.String("type", "provider"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		}
	}

	// labels
	table = at.Table(opts.LabelsTableName)
	for _, entry := range unmatched.Labels {
		logger().Debug("create airtable entry", zap.String("type", "label"), zap.Stringer("entry", entry))
		if err := table.Create(&entry); err != nil {
			return err
		}
		entry.State = airtableStateNew
		cache.Labels = append(cache.Labels, entry)
	}
	for _, entry := range cache.Labels {
		var err error
		switch entry.State {
		case airtableStateUnknown:
			err = table.Delete(&entry)
			logger().Debug("delete airtable entry", zap.String("type", "label"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateChanged:
			err = table.Update(&entry)
			logger().Debug("update airtable entry", zap.String("type", "label"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateUnchanged:
			logger().Debug("unchanged airtable entry", zap.String("type", "label"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		case airtableStateNew:
			logger().Debug("new airtable entry", zap.String("type", "label"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		}
	}

	// accounts
	table = at.Table(opts.AccountsTableName)
	for _, entry := range unmatched.Accounts {
		logger().Debug("create airtable entry", zap.String("type", "account"), zap.Stringer("entry", entry))
		if err := table.Create(&entry); err != nil {
			return err
		}
		entry.State = airtableStateNew
		cache.Accounts = append(cache.Accounts, entry)
	}
	for _, entry := range cache.Accounts {
		var err error
		switch entry.State {
		case airtableStateUnknown:
			err = table.Delete(&entry)
			logger().Debug("delete airtable entry", zap.String("type", "account"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateChanged:
			err = table.Update(&entry)
			logger().Debug("update airtable entry", zap.String("type", "account"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateUnchanged:
			logger().Debug("unchanged airtable entry", zap.String("type", "account"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		case airtableStateNew:
			logger().Debug("new airtable entry", zap.String("type", "account"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		}
	}

	// repositories
	table = at.Table(opts.RepositoriesTableName)
	for _, entry := range unmatched.Repositories {
		logger().Debug("create airtable entry", zap.String("type", "repository"), zap.Stringer("entry", entry))
		if err := table.Create(&entry); err != nil {
			return err
		}
		entry.State = airtableStateNew
		cache.Repositories = append(cache.Repositories, entry)
	}
	for _, entry := range cache.Repositories {
		var err error
		switch entry.State {
		case airtableStateUnknown:
			err = table.Delete(&entry)
			logger().Debug("delete airtable entry", zap.String("type", "repository"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateChanged:
			err = table.Update(&entry)
			logger().Debug("update airtable entry", zap.String("type", "repository"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateUnchanged:
			logger().Debug("unchanged airtable entry", zap.String("type", "repository"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		case airtableStateNew:
			logger().Debug("new airtable entry", zap.String("type", "repository"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		}
	}

	// milestones
	table = at.Table(opts.MilestonesTableName)
	for _, entry := range unmatched.Milestones {
		logger().Debug("create airtable entry", zap.String("type", "milestone"), zap.Stringer("entry", entry))
		if err := table.Create(&entry); err != nil {
			return err
		}
		entry.State = airtableStateNew
		cache.Milestones = append(cache.Milestones, entry)
	}
	for _, entry := range cache.Milestones {
		var err error
		switch entry.State {
		case airtableStateUnknown:
			err = table.Delete(&entry)
			logger().Debug("delete airtable entry", zap.String("type", "milestone"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateChanged:
			err = table.Update(&entry)
			logger().Debug("update airtable entry", zap.String("type", "milestone"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateUnchanged:
			logger().Debug("unchanged airtable entry", zap.String("type", "milestone"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		case airtableStateNew:
			logger().Debug("new airtable entry", zap.String("type", "milestone"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		}
	}

	// issues
	table = at.Table(opts.IssuesTableName)
	for _, entry := range unmatched.Issues {
		logger().Debug("create airtable entry", zap.String("type", "issue"), zap.Stringer("entry", entry))
		if err := table.Create(&entry); err != nil {
			return err
		}
		entry.State = airtableStateNew
		cache.Issues = append(cache.Issues, entry)
	}
	for _, entry := range cache.Issues {
		var err error
		switch entry.State {
		case airtableStateUnknown:
			err = table.Delete(&entry)
			logger().Debug("delete airtable entry", zap.String("type", "issue"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateChanged:
			err = table.Update(&entry)
			logger().Debug("update airtable entry", zap.String("type", "issue"), zap.Stringer("entry", entry), zap.Error(err))
		case airtableStateUnchanged:
			logger().Debug("unchanged airtable entry", zap.String("type", "issue"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		case airtableStateNew:
			logger().Debug("new airtable entry", zap.String("type", "issue"), zap.Stringer("entry", entry), zap.Error(err))
			// do nothing
		}
	}

	//
	// debug
	//
	fmt.Println("------- providers")
	for _, entry := range cache.Providers {
		fmt.Println(entry.ID, airtableStateString[entry.State], entry.Fields.ID)
	}
	fmt.Println("------- labels")
	for _, entry := range cache.Labels {
		fmt.Println(entry.ID, airtableStateString[entry.State], entry.Fields.ID)
	}
	fmt.Println("------- accounts")
	for _, entry := range cache.Accounts {
		fmt.Println(entry.ID, airtableStateString[entry.State], entry.Fields.ID)
	}
	fmt.Println("------- repositories")
	for _, entry := range cache.Repositories {
		fmt.Println(entry.ID, airtableStateString[entry.State], entry.Fields.ID)
	}
	fmt.Println("------- milestones")
	for _, entry := range cache.Milestones {
		fmt.Println(entry.ID, airtableStateString[entry.State], entry.Fields.ID)
	}
	fmt.Println("------- issues")
	for _, entry := range cache.Issues {
		fmt.Println(entry.ID, airtableStateString[entry.State], entry.Fields.ID)
	}
	fmt.Println("-------")

	return nil
}
