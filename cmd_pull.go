package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/google/go-github/github"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	gitlab "github.com/xanzy/go-gitlab"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type pullOptions struct {
	// db
	DBOpts dbOptions

	// pull
	Repos       []string
	GithubToken string `mapstructure:"github-token"`
	GitlabToken string `mapstructure:"gitlab-token"`
	// includeExternalDeps bool

	Targets []string
}

func (opts pullOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func pullSetupFlags(flags *pflag.FlagSet, opts *pullOptions) {
	flags.StringVarP(&opts.GithubToken, "github-token", "", "", "GitHub Token with 'issues' access")
	flags.StringVarP(&opts.GitlabToken, "gitlab-token", "", "", "GitLab Token with 'issues' access")
	viper.BindPFlags(flags)
}

func newPullCommand() *cobra.Command {
	opts := &pullOptions{}
	cmd := &cobra.Command{
		Use: "pull",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(opts); err != nil {
				return err
			}
			opts.Targets = args
			return pull(opts)
		},
	}
	pullSetupFlags(cmd.Flags(), opts)
	dbSetupFlags(cmd.Flags(), &opts.DBOpts)
	return cmd
}

func pull(opts *pullOptions) error {
	logger().Debug("pull", zap.Stringer("opts", *opts))

	var (
		wg        sync.WaitGroup
		allIssues []*Issue
		out       = make(chan []*Issue, 100)
	)

	repos := getReposFromTargets(opts.Targets)

	wg.Add(len(repos))
	for _, repoURL := range repos {
		repo := NewRepo(repoURL)
		switch repo.Provider() {
		case GitHubProvider:
			ctx := context.Background()
			ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: opts.GithubToken})
			tc := oauth2.NewClient(ctx, ts)
			client := github.NewClient(tc)

			go func(repo Repo) {
				total := 0
				defer wg.Done()
				opts := &github.IssueListByRepoOptions{State: "all"}
				for {
					issues, resp, err := client.Issues.ListByRepo(ctx, repo.Namespace(), repo.Project(), opts)
					if err != nil {
						log.Fatal(err)
						return
					}
					total += len(issues)
					logger().Debug("paginate",
						zap.String("provider", "github"),
						zap.String("repo", repo.Canonical()),
						zap.Int("new-issues", len(issues)),
						zap.Int("total-issues", total),
					)
					normalizedIssues := []*Issue{}
					for _, issue := range issues {
						normalizedIssues = append(normalizedIssues, FromGitHubIssue(issue))
					}
					out <- normalizedIssues
					if resp.NextPage == 0 {
						break
					}
					opts.Page = resp.NextPage
				}
				if rateLimits, _, err := client.RateLimits(ctx); err == nil {
					logger().Debug("github API rate limiting", zap.Stringer("limit", rateLimits.GetCore()))
				}
			}(repo)
		case GitLabProvider:
			go func(repo Repo) {
				client := gitlab.NewClient(nil, opts.GitlabToken)
				client.SetBaseURL(fmt.Sprintf("%s/api/v4", repo.SiteURL()))

				//projectID := url.QueryEscape(repo.RepoPath())
				projectID := repo.RepoPath()
				total := 0
				defer wg.Done()
				opts := &gitlab.ListProjectIssuesOptions{
					ListOptions: gitlab.ListOptions{
						PerPage: 30,
						Page:    1,
					},
				}
				for {
					issues, resp, err := client.Issues.ListProjectIssues(projectID, opts)
					if err != nil {
						logger().Error("failed to pull issues", zap.Error(err))
						return
					}
					total += len(issues)
					logger().Debug("paginate",
						zap.String("provider", "gitlab"),
						zap.String("repo", repo.Canonical()),
						zap.Int("new-issues", len(issues)),
						zap.Int("total-issues", total),
					)
					normalizedIssues := []*Issue{}
					for _, issue := range issues {
						normalizedIssues = append(normalizedIssues, FromGitLabIssue(issue))
					}
					out <- normalizedIssues
					if resp.NextPage == 0 {
						break
					}
					opts.ListOptions.Page = resp.NextPage
				}
			}(repo)
		default:
			panic("should not happen")
		}
	}
	wg.Wait()
	close(out)
	for issues := range out {
		allIssues = append(allIssues, issues...)
	}

	for _, issue := range allIssues {
		if err := db.Save(issue).Error; err != nil {
			return err
		}
	}
	return nil
}
