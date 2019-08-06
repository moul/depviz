package gitlab // import "moul.io/depviz/gitlab"

import (
	"fmt"
	"sync"

	"github.com/jinzhu/gorm"
	gitlab "github.com/xanzy/go-gitlab"
	"go.uber.org/zap"
	"moul.io/depviz/model"
	"moul.io/multipmuri"
)

func Pull(input multipmuri.Entity, wg *sync.WaitGroup, token string, db *gorm.DB, out chan<- []*model.Issue) {
	defer wg.Done()
	// parse input
	type multipmuriMinimalInterface interface {
		RepoEntity() *multipmuri.GitLabRepo
	}
	target, ok := input.(multipmuriMinimalInterface)
	if !ok {
		zap.L().Warn("invalid input", zap.String("input", fmt.Sprintf("%v", input)))
		return
	}
	repo := target.RepoEntity()

	// create client
	client := gitlab.NewClient(nil, token)
	if err := client.SetBaseURL(fmt.Sprintf("%s/api/v4", repo.ServiceEntity().Canonical())); err != nil {
		zap.L().Error("failed to configure GitLab client", zap.Error(err))
		return
	}
	total := 0
	gitlabOpts := &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 30,
			Page:    1,
		},
	}

	var lastEntry model.Issue
	if err := db.Where("repository_id = ?", repo.Canonical()).Order("updated_at desc").First(&lastEntry).Error; err == nil {
		gitlabOpts.UpdatedAfter = &lastEntry.UpdatedAt
	}

	// FIXME: fetch PRs

	for {
		path := fmt.Sprintf("%s/%s", repo.Owner(), repo.Repo())
		issues, resp, err := client.Issues.ListProjectIssues(path, gitlabOpts)
		if err != nil {
			zap.L().Error("failed to pull issues", zap.Error(err))
			return
		}
		total += len(issues)
		zap.L().Debug("paginate",
			zap.String("provider", "gitlab"),
			zap.String("repo", repo.Canonical()),
			zap.Int("new-issues", len(issues)),
			zap.Int("total-issues", total),
		)
		normalizedIssues := []*model.Issue{}
		for _, issue := range issues {
			normalizedIssues = append(normalizedIssues, FromIssue(issue))
		}
		out <- normalizedIssues
		if resp.NextPage == 0 {
			break
		}
		gitlabOpts.ListOptions.Page = resp.NextPage
	}
}
