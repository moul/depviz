package main

import (
	"encoding/json"
	"fmt"
	"moul.io/depviz/pkg/repo"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type dbOptions struct{}

var globalDBOptions dbOptions

func (opts dbOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func dbSetupFlags(flags *pflag.FlagSet, opts *dbOptions) {
	viper.BindPFlags(flags)
}

func newDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "db",
	}
	cmd.AddCommand(newDBDumpCommand())
	return cmd
}

func newDBDumpCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			opts := globalDBOptions
			return dbDump(&opts)
		},
	}
	dbSetupFlags(cmd.Flags(), &globalDBOptions)
	return cmd
}

func dbDump(opts *dbOptions) error {
	issues := []*repo.Issue{}
	if err := db.Find(&issues).Error; err != nil {
		return err
	}

	for _, issue := range issues {
		issue.PostLoad()
	}

	out, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func loadIssues(targets []string) (repo.Issues, error) {
	query := db.Model(repo.Issue{}).Order("created_at")
	if len(targets) > 0 {
		return nil, fmt.Errorf("not implemented")
		// query = query.Where("repo_url IN (?)", canonicalTargets(targets))
		// OR WHERE parents IN ....
		// etc
	}

	perPage := 100
	var issues []*repo.Issue
	for page := 0; ; page++ {
		var newIssues []*repo.Issue
		if err := query.Limit(perPage).Offset(perPage * page).Find(&newIssues).Error; err != nil {
			return nil, err
		}
		issues = append(issues, newIssues...)
		if len(newIssues) < perPage {
			break
		}
	}

	for _, issue := range issues {
		issue.PostLoad()
	}

	return repo.Issues(issues), nil
}

// FIXME: try to use gorm hooks to auto preload/postload items
