package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"moul.io/depviz/pkg/issues"
)

type dbOptions struct{}

func (opts dbOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

type dbCommand struct {
	opts dbOptions
}

func (cmd *dbCommand) LoadDefaultOptions() error {
	if err := viper.Unmarshal(&cmd.opts); err != nil {
		return err
	}
	return nil
}

func (cmd *dbCommand) NewCobraCommand(dc map[string]DepvizCommand) *cobra.Command {
	cc := &cobra.Command{
		Use: "db",
	}
	cc.AddCommand(cmd.dbDumpCommand())
	return cc
}

func (cmd *dbCommand) ParseFlags(flags *pflag.FlagSet) {
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind flags using Viper", zap.Error(err))
	}
}

func (cmd *dbCommand) dbDumpCommand() *cobra.Command {
	cc := &cobra.Command{
		Use:   "dump",
		Short: "Print all issues stored in the database, formatted as JSON",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			return dbDump(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	return cc
}

func dbDump(opts *dbOptions) error {
	query := db.Model(issues.Issue{}).Order("created_at")
	perPage := 100
	var allIssues []*issues.Issue
	for page := 0; ; page++ {
		var newIssues []*issues.Issue
		if err := query.Limit(perPage).Offset(perPage * page).Find(&newIssues).Error; err != nil {
			return err
		}
		allIssues = append(allIssues, newIssues...)
		if len(newIssues) < perPage {
			break
		}
	}

	for _, issue := range allIssues {
		issue.PostLoad()
	}

	out, err := json.MarshalIndent(allIssues, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}
