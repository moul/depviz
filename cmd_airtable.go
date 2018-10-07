package main

import (
	"encoding/json"
	"reflect"
	"sort"
	"time"

	"github.com/brianloveswords/airtable"
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
	if err := issues.prepare(true); err != nil {
		return errors.Wrap(err, "failed to prepare issues")
	}
	issues.filterByTargets(opts.Targets)
	logger().Debug("fetch db entries", zap.Int("count", len(issues)))

	at := airtable.Client{
		APIKey:  opts.AirtableToken,
		BaseID:  opts.AirtableBaseID,
		Limiter: airtable.RateLimiter(5),
	}
	table := at.Table(opts.AirtableTableName)

	alreadyInAirtable := map[string]bool{}

	records := []airtableRecord{}
	if err := table.List(&records, &airtable.Options{}); err != nil {
		return err
	}
	logger().Debug("fetched airtable records", zap.Int("count", len(records)))

	// create new records
	for _, record := range records {
		alreadyInAirtable[record.Fields.URL] = true
	}
	for _, issue := range issues {
		if issue.Hidden {
			continue
		}
		if _, found := alreadyInAirtable[issue.URL]; found {
			continue
		}
		logger().Debug("creating airtable record without slices", zap.String("URL", issue.URL))
		r := minimalAirtableRecord{
			Fields: minimalAirtableIssue{
				URL:    issue.URL,
				Errors: "initialization",
			},
		}
		if err := table.Create(&r); err != nil {
			return err
		}
		records = append(records, airtableRecord{
			ID: r.ID,
			Fields: airtableIssue{
				URL: issue.URL,
			},
		})
	}

	// update/destroy existing ones
	for _, record := range records {
		if issue, found := issues[record.Fields.URL]; !found {
			logger().Debug("destroying airtable record", zap.String("URL", record.Fields.URL))
			if err := table.Delete(&record); err != nil {
				return errors.Wrap(err, "failed to destroy record")
			}
		} else {
			if issue.Hidden {
				continue
			}

			if issue.ToAirtableRecord().Fields.Equals(record.Fields) {
				continue
			}

			logger().Debug("updating airtable record", zap.String("URL", issue.URL))
			record.Fields = issue.ToAirtableRecord().Fields
			if err := table.Update(&record); err != nil {
				logger().Warn("failed to update record, retrying without slices", zap.String("URL", issue.URL), zap.Error(err))
				record := minimalAirtableRecord{
					ID: record.ID,
					Fields: minimalAirtableIssue{
						URL: issue.URL,
					},
				}
				if typedErr, ok := err.(airtable.ErrClientRequest); ok {
					record.Fields.Errors = typedErr.Err.Error()
				} else {
					record.Fields.Errors = err.Error()
				}
				if err := table.Update(&record); err != nil {
					logger().Error("failed to update record without slices", zap.String("URL", issue.URL), zap.Error(err))
				}
			}
		}
	}

	return nil
}

type airtableRecord struct {
	ID     string        `json:"id,omitempty"`
	Fields airtableIssue `json:"fields,omitempty"`
}

type minimalAirtableRecord struct {
	ID     string               `json:"id,omitempty"`
	Fields minimalAirtableIssue `json:"fields,omitempty"`
}

func (ai airtableIssue) String() string {
	out, _ := json.Marshal(ai)
	return string(out)
}

func (i Issue) ToAirtableRecord() airtableRecord {
	typ := "issue"
	if i.IsPR {
		typ = "pull-request"
	}
	labels := []string{}
	for _, label := range i.Labels {
		labels = append(labels, label.ID)
	}
	assignees := []string{}
	for _, assignee := range i.Assignees {
		assignees = append(assignees, assignee.ID)
	}

	return airtableRecord{
		ID: "",
		Fields: airtableIssue{
			URL:       i.URL,
			Created:   i.CreatedAt,
			Updated:   i.UpdatedAt,
			Title:     i.Title,
			Type:      typ,
			Labels:    labels,
			Assignees: assignees,
			Provider:  string(i.Provider),
			RepoURL:   i.RepoURL,
			Body:      i.Body,
			State:     i.State,
			Locked:    i.Locked,
			Author:    i.AuthorID,
			Comments:  i.Comments,
			Milestone: i.Milestone,
			Upvotes:   i.Upvotes,
			Downvotes: i.Downvotes,
			Errors:    "",
		},
	}
}

type airtableIssue struct {
	URL       string
	Created   time.Time
	Updated   time.Time
	Title     string
	Provider  string
	State     string
	Body      string
	RepoURL   string
	Type      string
	Locked    bool
	Author    string
	Comments  int
	Milestone string
	Upvotes   int
	Downvotes int
	Labels    []string
	Assignees []string
	Errors    string
}

type minimalAirtableIssue struct {
	URL    string
	Errors string
}

func (ai airtableIssue) Equals(other airtableIssue) bool {
	sameSlice := func(a, b []string) bool {
		if a == nil {
			a = []string{}
		}
		if b == nil {
			b = []string{}
		}
		sort.Strings(a)
		sort.Strings(b)
		return reflect.DeepEqual(a, b)
	}
	return ai.URL == other.URL &&
		ai.Created.Truncate(time.Millisecond).UTC() == other.Created.Truncate(time.Millisecond).UTC() &&
		ai.Updated.Truncate(time.Millisecond).UTC() == other.Updated.Truncate(time.Millisecond).UTC() &&
		ai.Title == other.Title &&
		ai.Provider == other.Provider &&
		ai.State == other.State &&
		ai.Body == other.Body &&
		ai.RepoURL == other.RepoURL &&
		ai.Type == other.Type &&
		ai.Locked == other.Locked &&
		ai.Author == other.Author &&
		ai.Comments == other.Comments &&
		ai.Milestone == other.Milestone &&
		ai.Upvotes == other.Upvotes &&
		ai.Downvotes == other.Downvotes &&
		sameSlice(ai.Labels, other.Labels) &&
		sameSlice(ai.Assignees, other.Assignees) &&
		ai.Errors == other.Errors
}
