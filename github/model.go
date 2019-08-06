package github // import "moul.io/depviz/github"

import (
	"strings"

	"github.com/google/go-github/github"
	"moul.io/depviz/model"
)

func FromUser(input *github.User) *model.Account {
	name := input.GetName()
	if name == "" {
		name = input.GetLogin()
	}
	return &model.Account{
		Base: model.Base{
			ID:        input.GetLogin(),
			CreatedAt: input.GetCreatedAt().Time,
			UpdatedAt: input.GetUpdatedAt().Time,
		},
		Provider:  githubProvider(),
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

func FromRepositoryURL(input string) *model.Repository {
	return &model.Repository{
		Base: model.Base{
			ID: input,
		},
		URL:      input,
		Provider: githubProvider(),
	}
}

func FromMilestone(input *github.Milestone) *model.Milestone {
	if input == nil {
		return nil
	}
	parts := strings.Split(input.GetHTMLURL(), "/")
	return &model.Milestone{
		Base: model.Base{
			ID:        input.GetURL(), // FIXME: make it smaller
			CreatedAt: input.GetCreatedAt(),
			UpdatedAt: input.GetUpdatedAt(),
		},
		URL:         input.GetURL(),
		Title:       input.GetTitle(),
		Description: input.GetDescription(),
		ClosedAt:    input.GetClosedAt(),
		DueOn:       input.GetDueOn(),
		Creator:     FromUser(input.GetCreator()),
		Repository:  FromRepositoryURL(strings.Join(parts[0:len(parts)-2], "/")),
	}
}

func FromLabel(input *github.Label) *model.Label {
	if input == nil {
		return nil
	}
	return &model.Label{
		Base: model.Base{
			ID: input.GetURL(), // FIXME: make it smaller
		},
		Name:        input.GetName(),
		Color:       input.GetColor(),
		Description: input.GetDescription(),
		URL:         input.GetURL(),
	}
}

func FromIssue(input *github.Issue) *model.Issue {
	parts := strings.Split(input.GetHTMLURL(), "/")
	url := strings.Replace(input.GetHTMLURL(), "/pull/", "/issues/", -1)

	issue := &model.Issue{
		Base: model.Base{
			ID:        url,
			CreatedAt: input.GetCreatedAt(),
			UpdatedAt: input.GetUpdatedAt(),
		},
		CompletedAt: input.GetClosedAt(),
		Repository:  FromRepositoryURL(strings.Join(parts[0:len(parts)-2], "/")),
		Title:       input.GetTitle(),
		State:       input.GetState(),
		Body:        input.GetBody(),
		IsPR:        input.PullRequestLinks != nil,
		URL:         url,
		IsLocked:    input.GetLocked(),
		Comments:    input.GetComments(),
		Upvotes:     *input.Reactions.PlusOne,
		Downvotes:   *input.Reactions.MinusOne,
		Labels:      make([]*model.Label, 0),
		Assignees:   make([]*model.Account, 0),
		Author:      FromUser(input.User),
		Milestone:   FromMilestone(input.Milestone),

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
		issue.Labels = append(issue.Labels, FromLabel(&label))
	}
	for _, assignee := range input.Assignees {
		issue.Assignees = append(issue.Assignees, FromUser(assignee))
	}
	return issue
}

func githubProvider() *model.Provider {
	return &model.Provider{
		Base: model.Base{
			ID: "github", // FIXME: support multiple github instances
		},
		Driver: string(model.GithubDriver),
	}
}
