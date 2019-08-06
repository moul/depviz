package github // import "moul.io/depviz/github"

import (
	"context"
	"log"
	"sync"

	"github.com/google/go-github/github"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"moul.io/depviz/model"
	"moul.io/multipmuri"
)

type multipmuriMinimalInterface interface {
	Owner() string
	Hostname() string
	Repo() string
}

func Pull(input multipmuri.Entity, wg *sync.WaitGroup, token string, db *gorm.DB, out chan<- []*model.Issue) {
	defer wg.Done()
	target, ok := input.(multipmuriMinimalInterface)
	if !ok {
		zap.L().Warn("invalid input in github.Pull")
		return
	}

	// create client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// queries
	totalIssues := 0
	callOpts := &github.IssueListByRepoOptions{State: "all"}
	var lastEntry Issue
	if err := db.Where("repository_id = ?", target.ProjectURL()).Order("updated_at desc").First(&lastEntry).Error; err == nil {
		callOpts.Since = lastEntry.UpdatedAt
	} else {
		zap.L().Warn("failed to get last entry", zap.String("repo", target.ProjectURL()), zap.Error(err))
	}

	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, target.Namespace(), target.Project(), callOpts)
		if err != nil {
			log.Fatal(err)
			return
		}
		totalIssues += len(issues)
		zap.L().Debug("paginate",
			zap.String("provider", "github"),
			zap.String("repo", target.ProjectURL()),
			zap.Int("new-issues", len(issues)),
			zap.Int("total-issues", totalIssues),
		)
		normalizedIssues := []*Issue{}
		for _, issue := range issues {
			normalizedIssues = append(normalizedIssues, fromGithubIssue(issue))
		}
		out <- normalizedIssues
		if resp.NextPage == 0 {
			break
		}
		callOpts.Page = resp.NextPage
	}
	if rateLimits, _, err := client.RateLimits(ctx); err == nil {
		zap.L().Debug("github API rate limiting", zap.Stringer("limit", rateLimits.GetCore()))
	}
}
