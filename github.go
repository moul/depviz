package main

import (
	"context"
	"log"
	"strings"
	"sync"

	"github.com/google/go-github/github"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

func githubPull(target Target, wg *sync.WaitGroup, opts *pullOptions, out chan []*Issue) {
	defer wg.Done()
	ctx := context.Background()
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: opts.GithubToken})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	total := 0
	callOpts := &github.IssueListByRepoOptions{State: "all"}

	var lastEntry Issue
	if err := db.Where("repository_id = ?", target.ProjectURL()).Order("updated_at desc").First(&lastEntry).Error; err == nil {
		callOpts.Since = lastEntry.UpdatedAt
	} else {
		logger().Warn("failed to get last entry", zap.Error(err))
	}

	for {
		issues, resp, err := client.Issues.ListByRepo(ctx, target.Namespace(), target.Project(), callOpts)
		if err != nil {
			log.Fatal(err)
			return
		}
		total += len(issues)
		logger().Debug("paginate",
			zap.String("provider", "github"),
			zap.String("repo", target.ProjectURL()),
			zap.Int("new-issues", len(issues)),
			zap.Int("total-issues", total),
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
		logger().Debug("github API rate limiting", zap.Stringer("limit", rateLimits.GetCore()))
	}
}

func fromGithubUser(input *github.User) *Account {
	name := input.GetName()
	if name == "" {
		name = input.GetLogin()
	}
	return &Account{
		Base: Base{
			ID:        input.GetLogin(),
			CreatedAt: input.GetCreatedAt().Time,
			UpdatedAt: input.GetUpdatedAt().Time,
		},
		Provider: &Provider{
			Base: Base{
				ID: "github", // FIXME: support multiple github instances
			},
			Driver: GithubDriver,
		},
		URL:       input.GetURL(),
		Location:  input.GetLocation(),
		Company:   input.GetCompany(),
		Blog:      input.GetBlog(),
		Email:     input.GetEmail(),
		AvatarURL: input.GetAvatarURL(),
		Login:     input.GetLogin(),
		FullName:  name,
	}
}

func fromGithubRepository(input *github.Repository) *Repository {
	panic("not implemented")
}

func fromGithubRepositoryURL(input string) *Repository {
	return &Repository{
		Base: Base{
			ID: input,
		},
		URL: input,
		Provider: &Provider{
			Base: Base{
				ID: "github", // FIXME: support multiple github instances
			},
			Driver: GithubDriver,
		},
	}
}

func fromGithubMilestone(input *github.Milestone) *Milestone {
	if input == nil {
		return nil
	}
	parts := strings.Split(input.GetHTMLURL(), "/")
	return &Milestone{
		Base: Base{
			ID:        input.GetURL(), // FIXME: make it smaller
			CreatedAt: input.GetCreatedAt(),
			UpdatedAt: input.GetUpdatedAt(),
		},
		URL:         input.GetURL(),
		Title:       input.GetTitle(),
		Description: input.GetDescription(),
		ClosedAt:    input.GetClosedAt(),
		DueOn:       input.GetDueOn(),
		Creator:     fromGithubUser(input.GetCreator()),
		Repository:  fromGithubRepositoryURL(strings.Join(parts[0:len(parts)-2], "/")),
	}
}

func fromGithubLabel(input *github.Label) *Label {
	if input == nil {
		return nil
	}
	return &Label{
		Base: Base{
			ID: input.GetURL(), // FIXME: make it smaller
		},
		Name:        input.GetName(),
		Color:       input.GetColor(),
		Description: input.GetDescription(),
		URL:         input.GetURL(),
	}
}

func fromGithubIssue(input *github.Issue) *Issue {
	parts := strings.Split(input.GetHTMLURL(), "/")
	url := strings.Replace(input.GetHTMLURL(), "/pull/", "/issues/", -1)

	issue := &Issue{
		Base: Base{
			ID:        url,
			CreatedAt: input.GetCreatedAt(),
			UpdatedAt: input.GetUpdatedAt(),
		},
		CompletedAt: input.GetClosedAt(),
		Repository:  fromGithubRepositoryURL(strings.Join(parts[0:len(parts)-2], "/")),
		Title:       input.GetTitle(),
		State:       input.GetState(),
		Body:        input.GetBody(),
		IsPR:        input.PullRequestLinks != nil,
		URL:         url,
		IsLocked:    input.GetLocked(),
		Comments:    input.GetComments(),
		Upvotes:     *input.Reactions.PlusOne,
		Downvotes:   *input.Reactions.MinusOne,
		Labels:      make([]*Label, 0),
		Assignees:   make([]*Account, 0),
		Author:      fromGithubUser(input.User),
		Milestone:   fromGithubMilestone(input.Milestone),

		/*
			IsOrphan    bool      `json:"is-orphan"`
			IsHidden    bool      `json:"is-hidden"`
			BaseWeight  int       `json:"base-weight"`
			Weight      int       `json:"weight"`
			IsEpic      bool      `json:"is-epic"`
			HasEpic     bool      `json:"has-epic"`

			// internal
			Parents    []*Issue `json:"-" gorm:"-"`
			Children   []*Issue `json:"-" gorm:"-"`
			Duplicates []*Issue `json:"-" gorm:"-"`
		*/

	}
	for _, label := range input.Labels {
		issue.Labels = append(issue.Labels, fromGithubLabel(&label))
	}
	for _, assignee := range input.Assignees {
		issue.Assignees = append(issue.Assignees, fromGithubUser(assignee))
	}
	return issue
}
