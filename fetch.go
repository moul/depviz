package main

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"log"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type fetchOptions struct {
	// db
	DBOpts dbOptions

	// fetch
	Repos       []string
	GithubToken string `mapstructure:"github-token"`
	// includeExternalDeps bool
}

func (opts fetchOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func fetchSetupFlags(flags *pflag.FlagSet, opts *fetchOptions) {
	flags.StringSliceVarP(&opts.Repos, "repos", "r", []string{}, "list of repositories to aggregate issues from") // FIXME: get the default value dynamically from .git, if present
	flags.StringVarP(&opts.GithubToken, "github-token", "", "", "GitHub Token with 'issues' access")
	viper.BindPFlags(flags)
}

func newFetchCommand() *cobra.Command {
	opts := &fetchOptions{}
	cmd := &cobra.Command{
		Use: "fetch",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(opts); err != nil {
				return err
			}
			return fetch(opts)
		},
	}
	fetchSetupFlags(cmd.Flags(), opts)
	dbSetupFlags(cmd.Flags(), &opts.DBOpts)
	return cmd
}

func fetch(opts *fetchOptions) error {
	logger().Debug("fetch", zap.Stringer("opts", *opts))
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: opts.GithubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	var (
		wg        sync.WaitGroup
		allIssues []*github.Issue
		out       = make(chan []*github.Issue, 100)
	)
	wg.Add(len(opts.Repos))
	for _, repo := range opts.Repos {
		parts := strings.Split(repo, "/")
		organization := parts[0]
		repo := parts[1]

		go func(repo string) {
			total := 0
			defer wg.Done()
			opts := &github.IssueListByRepoOptions{State: "all"}
			for {
				issues, resp, err := client.Issues.ListByRepo(ctx, organization, repo, opts)
				if err != nil {
					log.Fatal(err)
					return
				}
				total += len(issues)
				logger().Debug("paginate",
					zap.String("repo", repo),
					zap.Int("new-issues", len(issues)),
					zap.Int("total-issues", total),
				)
				out <- issues
				if resp.NextPage == 0 {
					break
				}
				opts.Page = resp.NextPage
			}
		}(repo)
	}
	wg.Wait()
	close(out)
	for issues := range out {
		allIssues = append(allIssues, issues...)
	}

	issuesJson, _ := json.MarshalIndent(allIssues, "", "  ")
	rateLimits, _, err := client.RateLimits(ctx)
	if err != nil {
		return err
	}
	logger().Debug("github API rate limiting", zap.Stringer("limit", rateLimits.GetCore()))
	return errors.Wrap(ioutil.WriteFile(opts.DBOpts.Path, issuesJson, 0644), "failed to write db file")
}
