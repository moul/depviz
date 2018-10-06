package main

import (
	"encoding/json"
	"time"

	airtable "github.com/fabioberger/airtable-go"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type airtableOptions struct {
	AirtableTableName string `mapstructure:"airtable-table-name"`
	AirtableBaseID    string `mapstructure:"airtable-base-id"`
	AirtableToken     string `mapstructure:"airtable-token"`
	Targets           []string
}

func (opts airtableOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func airtableSetupFlags(flags *pflag.FlagSet, opts *airtableOptions) {
	flags.StringVarP(&opts.AirtableTableName, "airtable-table-name", "", "Issues and PRs", "Airtable table name")
	flags.StringVarP(&opts.AirtableBaseID, "airtable-base-id", "", "", "Airtable base ID")
	flags.StringVarP(&opts.AirtableToken, "airtable-token", "", "", "Airtable token")
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
	opts := &airtableOptions{}
	cmd := &cobra.Command{
		Use: "sync",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(opts); err != nil {
				return err
			}
			opts.Targets = args
			return airtableSync(opts)
		},
	}
	airtableSetupFlags(cmd.Flags(), opts)
	return cmd
}

func airtableSync(opts *airtableOptions) error {
	issues, err := loadIssues(db, nil)
	if err != nil {
		return errors.Wrap(err, "failed to load issues")
	}
	if err := issues.prepare(); err != nil {
		return errors.Wrap(err, "failed to prepare issues")
	}
	issues.filterByTargets(opts.Targets)
	logger().Debug("fetch db entries", zap.Int("count", len(issues)))

	at, err := airtable.New(opts.AirtableToken, opts.AirtableBaseID)
	if err != nil {
		return err
	}

	alreadyInAirtable := map[string]bool{}

	records := []airtableRecord{}
	if err := at.ListRecords(opts.AirtableTableName, &records); err != nil {
		return err
	}
	logger().Debug("fetched airtable records", zap.Int("count", len(records)))

	for _, record := range records {
		alreadyInAirtable[record.Fields.ID] = true
		if issue, found := issues[record.Fields.ID]; !found {
			logger().Debug("destroying airtable record", zap.String("ID", record.Fields.ID))
			if err := at.DestroyRecord(opts.AirtableTableName, record.ID); err != nil {
				return errors.Wrap(err, "failed to destroy record")
			}
		} else {
			if issue.Hidden {
				continue
			}
			// FIXME: check if entry changed before updating
			logger().Debug("updating airtable record", zap.String("ID", issue.URL))
			if err := at.UpdateRecord(opts.AirtableTableName, record.ID, issue.ToAirtableRecord().Fields.Map(), &record); err != nil {
				return errors.Wrap(err, "failed to update record")
			}
		}
	}

	for _, issue := range issues {
		if issue.Hidden {
			continue
		}
		if _, found := alreadyInAirtable[issue.URL]; found {
			continue
		}
		logger().Debug("creating airtable record", zap.String("ID", issue.URL))
		if err := at.CreateRecord(opts.AirtableTableName, issue.ToAirtableRecord()); err != nil {
			return err
		}
	}
	return nil
}

func (i Issue) ToAirtableRecord() airtableRecord {
	return airtableRecord{
		ID: "",
		Fields: airtableIssue{
			ID:      i.URL,
			Created: i.CreatedAt,
			Updated: i.UpdatedAt,
			Title:   i.Title,
		},
	}
}

type airtableRecord struct {
	ID     string        `json:"id,omitempty"`
	Fields airtableIssue `json:"fields,omitempty"`
}

type airtableIssue struct {
	ID      string
	Created time.Time
	Updated time.Time
	Title   string
}

func (a airtableIssue) Map() map[string]interface{} {
	return map[string]interface{}{
		"ID":      a.ID,
		"Created": a.Created,
		"Updated": a.Updated,
		"Title":   a.Title,
	}
}
