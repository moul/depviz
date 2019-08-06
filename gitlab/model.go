package gitlab // import "moul.io/depviz/gitlab"

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"moul.io/depviz/model"

	gitlab "github.com/xanzy/go-gitlab"
	"go.uber.org/zap"
)

func FromIssue(input *gitlab.Issue) *model.Issue {
	repoURL := input.Links.Project
	if repoURL == "" {
		repoURL = strings.Replace(input.WebURL, fmt.Sprintf("/issues/%d", input.IID), "", -1)
	}

	//out, _ := json.MarshalIndent(input, "", "  ")
	//fmt.Println(string(out))

	repo := FromRepositoryURL(repoURL)
	issue := &model.Issue{
		Base: model.Base{
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
		Labels:     make([]*model.Label, 0),
		Assignees:  make([]*model.Account, 0),
		Author:     FromIssueAuthor(repo.Provider, input.Author),
		Milestone:  FromMilestone(repo, input.Milestone),
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
		issue.Labels = append(issue.Labels, FromLabelname(repo, label))
	}
	//issue.Assignees = append(issue.Assignees, FromIssueAssignee(input.Assignee))
	for _, assignee := range input.Assignees {
		issue.Assignees = append(issue.Assignees, FromIssueAssignee(repo.Provider, assignee))
	}
	return issue
}

func FromLabelname(repository *model.Repository, name string) *model.Label {
	url := fmt.Sprintf("%s/labels/%s", repository.URL, name)
	return &model.Label{
		Base: model.Base{
			ID: url,
		},
		Name:  name,
		Color: "aaaacc",
		URL:   url,
		// Description:
	}
}

func FromIssueAssignee(provider *model.Provider, input *gitlab.IssueAssignee) *model.Account {
	author := gitlab.IssueAuthor(*input)
	return FromIssueAuthor(provider, &author)
}

func FromIssueAuthor(provider *model.Provider, input *gitlab.IssueAuthor) *model.Account {
	name := input.Name
	if name == "" {
		name = input.Username
	}
	account := model.Account{
		Base: model.Base{
			ID: input.WebURL,
			// UpdatedAt:
			// CreatedAt:
		},
		Provider: &model.Provider{
			Base: model.Base{
				ID: "gitlab", // FIXME: support multiple gitlab instances
			},
			Driver: string(model.GitlabDriver),
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

func FromRepositoryURL(input string) *model.Repository {
	u, err := url.Parse(input)
	if err != nil {
		zap.L().Warn("invalid repository URL", zap.String("URL", input))
		return nil
	}
	providerURL := fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	return &model.Repository{
		Base: model.Base{
			ID: input,
		},
		URL: input,
		Provider: &model.Provider{
			Base: model.Base{
				ID: "gitlab", // FIXME: support multiple gitlab instances
			},
			URL:    providerURL,
			Driver: string(model.GitlabDriver),
		},
	}
}

func FromMilestone(repository *model.Repository, input *gitlab.Milestone) *model.Milestone {
	if input == nil {
		return nil
	}
	url := fmt.Sprintf("%s/milestones/%d", repository.URL, input.ID)
	milestone := model.Milestone{
		Base: model.Base{
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
