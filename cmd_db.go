package main

import (
	"encoding/json"
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type dbOptions struct {
}

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
	opts := &dbOptions{}
	cmd := &cobra.Command{
		Use: "dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(opts); err != nil {
				return err
			}
			return dbDump(opts)
		},
	}
	dbSetupFlags(cmd.Flags(), opts)
	return cmd
}

func dbDump(opts *dbOptions) error {
	issues, err := loadIssues(db, nil)
	if err != nil {
		return errors.Wrap(err, "failed to load issues")
	}
	out, err := json.MarshalIndent(issues.ToSlice(), "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func canonicalTargets(input []string) []string {
	output := []string{}
	base := Issue{RepoURL: "https://github.com/moul/depviz", URL: "https://github.com/moul/depviz/issues/1"}
	for _, target := range input {
		output = append(output, base.GetRelativeIssueURL(target))
	}
	return output
}

func loadIssues(db *gorm.DB, targets []string) (Issues, error) {
	query := db.Model(Issue{})
	if len(targets) > 0 {
		query = query.Where("repo_url IN (?)", canonicalTargets(targets))
	}

	/*var count int
	if err := query.Count(&count).Error; err != nil {
		return nil, err
	}*/

	perPage := 100
	var issues []*Issue
	for page := 0; ; page++ {
		var newIssues []*Issue
		if err := query.Limit(perPage).Offset(perPage * page).Find(&newIssues).Error; err != nil {
			return nil, err
		}
		issues = append(issues, newIssues...)
		if len(newIssues) < perPage {
			break
		}
	}
	slice := IssueSlice(issues)
	return slice.ToMap(), nil
}
