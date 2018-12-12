package repo

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/jinzhu/gorm"
	gitlab "github.com/xanzy/go-gitlab"
	"go.uber.org/zap"
)

func gitlabPull(target Target, wg *sync.WaitGroup, token string, db *gorm.DB, out chan []*Issue) {
	defer wg.Done()
	client := gitlab.NewClient(nil, token)
	client.SetBaseURL(fmt.Sprintf("%s/api/v4", target.ProviderURL()))
	total := 0
	gitlabOpts := &gitlab.ListProjectIssuesOptions{
		ListOptions: gitlab.ListOptions{
			PerPage: 30,
			Page:    1,
		},
	}

	var lastEntry Issue
	if err := db.Where("repository_id = ?", target.ProjectURL()).Order("updated_at desc").First(&lastEntry).Error; err == nil {
		gitlabOpts.UpdatedAfter = &lastEntry.UpdatedAt
	}

	// FIXME: fetch PRs

	for {
		issues, resp, err := client.Issues.ListProjectIssues(target.Path(), gitlabOpts)
		if err != nil {
			zap.L().Error("failed to pull issues", zap.Error(err))
			return
		}
		total += len(issues)
		zap.L().Debug("paginate",
			zap.String("provider", "gitlab"),
			zap.String("repo", target.ProjectURL()),
			zap.Int("new-issues", len(issues)),
			zap.Int("total-issues", total),
		)
		normalizedIssues := []*Issue{}
		for _, issue := range issues {
			normalizedIssues = append(normalizedIssues, fromGitlabIssue(issue))
		}
		out <- normalizedIssues
		if resp.NextPage == 0 {
			break
		}
		gitlabOpts.ListOptions.Page = resp.NextPage
	}
}

func fromGitlabIssue(input *gitlab.Issue) *Issue {
	repoURL := input.Links.Project
	if repoURL == "" {
		repoURL = strings.Replace(input.WebURL, fmt.Sprintf("/issues/%d", input.IID), "", -1)
	}

	//out, _ := json.MarshalIndent(input, "", "  ")
	//fmt.Println(string(out))

	repo := fromGitlabRepositoryURL(repoURL)
	issue := &Issue{
		Base: Base{
			ID:        input.WebURL,
			CreatedAt: *input.CreatedAt,
			UpdatedAt: *input.UpdatedAt,
		},
		Repository: repo,
		Title:      input.Title,
		State:      input.State,
		Body:       input.Description,
		IsPR:       false,
		URL:        input.WebURL,
		IsLocked:   false, // not supported on GitLab
		Comments:   0,     // not supported directly
		Upvotes:    input.Upvotes,
		Downvotes:  input.Downvotes,
		Labels:     make([]*Label, 0),
		Assignees:  make([]*Account, 0),
		Author:     fromGitlabFakeUser(repo.Provider, input.Author),
		Milestone:  fromGitlabMilestone(repo, input.Milestone),
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
	if input.ClosedAt != nil {
		issue.CompletedAt = *input.ClosedAt
	}
	for _, label := range input.Labels {
		issue.Labels = append(issue.Labels, fromGitlabLabelname(repo, label))
	}
	//issue.Assignees = append(issue.Assignees, fromGitlabFakeUser(input.Assignee))
	for _, assignee := range input.Assignees {
		issue.Assignees = append(issue.Assignees, fromGitlabFakeUser(repo.Provider, assignee))
	}
	return issue
}

func fromGitlabLabelname(repository *Repository, name string) *Label {
	url := fmt.Sprintf("%s/labels/%s", repository.URL, name)
	return &Label{
		Base: Base{
			ID: url,
		},
		Name:  name,
		Color: "aaaacc",
		URL:   url,
		// Description:
	}
}

type gitlabFakeUser struct {
	ID        int    `json:"id"`
	State     string `json:"state"`
	WebURL    string `json:"web_url"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	Username  string `json:"username"`
}

func fromGitlabFakeUser(provider *Provider, input gitlabFakeUser) *Account {
	name := input.Name
	if name == "" {
		name = input.Username
	}
	account := Account{
		Base: Base{
			ID: input.WebURL,
			// UpdatedAt:
			// CreatedAt:
		},
		Provider: &Provider{
			Base: Base{
				ID: "gitlab", // FIXME: support multiple gitlab instances
			},
			Driver: GitlabDriver,
		},
		// Email:
		FullName: name,
		Login:    input.Username,
		URL:      input.WebURL,
		// State: // FIXME: investigate what to do with this

		// Location:
		// Company:
		// Blog:
		AvatarURL: input.AvatarURL,
	}

	return &account
}

func fromGitlabRepositoryURL(input string) *Repository {
	u, err := url.Parse(input)
	if err != nil {
		zap.L().Warn("invalid repository URL", zap.String("URL", input))
		return nil
	}
	providerURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	return &Repository{
		Base: Base{
			ID: input,
		},
		URL: input,
		Provider: &Provider{
			Base: Base{
				ID: "gitlab", // FIXME: support multiple gitlab instances
			},
			URL:    providerURL,
			Driver: GitlabDriver,
		},
	}
}

func fromGitlabMilestone(repository *Repository, input *gitlab.Milestone) *Milestone {
	if input == nil {
		return nil
	}
	url := fmt.Sprintf("%s/milestones/%d", repository.URL, input.ID)
	milestone := Milestone{
		Base: Base{
			ID:        url,
			CreatedAt: *input.CreatedAt,
			UpdatedAt: *input.UpdatedAt,
		},
		URL:         url,
		Title:       input.Title,
		Description: input.Description,
	}
	if input.DueDate != nil {
		milestone.DueOn = time.Time(*input.DueDate)
	}
	// startdate // FIXME: todo
	// state // FIXME: todo
	return &milestone
}
