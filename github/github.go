package github // import "moul.io/depviz/github"

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/google/go-github/v28/github"
	"github.com/jinzhu/gorm"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"moul.io/depviz/model"
	"moul.io/multipmuri"
)

func Pull(input multipmuri.Entity, wg *sync.WaitGroup, token string, db *gorm.DB, out chan<- []*model.Issue) {
	defer wg.Done()
	type multipmuriMinimalInterface interface {
		Repo() *multipmuri.GitHubRepo
	}
	target, ok := input.(multipmuriMinimalInterface)
	if !ok {
		zap.L().Warn("invalid input", zap.String("input", fmt.Sprintf("%v", input.String())))
		return
	}
	repo := target.Repo()

	// create client
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// queries
	totalIssues := 0
	callOpts := &github.IssueListByRepoOptions{State: "all"}
	var lastEntry model.Issue
	if err := db.Where("repository_id = ?", repo.String()).Order("updated_at desc").First(&lastEntry).Error; err == nil {
		callOpts.Since = lastEntry.UpdatedAt
	} else {
		zap.L().Warn("failed to get last entry", zap.String("repo", repo.String()), zap.Error(err))
	}

	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, repo.OwnerID(), repo.RepoID(), callOpts)
		if err != nil {
			log.Fatal(err)
			return
		}
		totalIssues += len(issues)
		zap.L().Debug("paginate",
			zap.String("provider", "github"),
			zap.String("repo", repo.String()),
			zap.Int("new-issues", len(issues)),
			zap.Int("total-issues", totalIssues),
		)
		normalizedIssues := []*model.Issue{}
		for _, issue := range issues {
			normalizedIssues = append(normalizedIssues, FromIssue(issue))
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
