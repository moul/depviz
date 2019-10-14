package github // import "moul.io/depviz/github"

import (
	"strings"

	"github.com/google/go-github/v28/github"
	"moul.io/depviz/model"
	"moul.io/multipmuri"
)

func FromUser(input *github.User) *model.Account {
	entity, err := model.ParseTarget(input.GetHTMLURL())
	if err != nil {
		panic(err)
	}

	name := input.GetName()
	if name == "" {
		name = input.GetLogin()
	}
	return &model.Account{
		Base: model.Base{
			ID:        input.GetLogin(),
			CreatedAt: input.GetCreatedAt().Time,
			UpdatedAt: input.GetUpdatedAt().Time,
			URL:       input.GetURL(),
		},
		// Type: "user"
		Provider:  FromServiceURL(multipmuri.ServiceEntity(entity).String()),
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
	entity, err := model.ParseTarget(input)
	if err != nil {
		panic(err)
	}
	owner := multipmuri.OwnerEntity(entity)
	return &model.Repository{
		Base: model.Base{
			ID:  input,
			URL: input,
		},
		Provider: FromServiceURL(multipmuri.ServiceEntity(entity).String()),
		Owner:    FromOwnerURL(owner.String()),
	}
}

func FromOwnerURL(input string) *model.Account {
	entity, err := model.ParseTarget(input)
	if err != nil {
		panic(err)
	}
	return &model.Account{
		Base: model.Base{
			ID:  input,
			URL: input,
		},
		Provider: FromServiceURL(multipmuri.ServiceEntity(entity).String()),
	}
}

func FromMilestone(input *github.Milestone) *model.Milestone {
	if input == nil {
		return nil
	}
	parts := strings.Split(input.GetHTMLURL(), "/")
	return &model.Milestone{
		Base: model.Base{
			ID:        input.GetHTMLURL(), // FIXME: make it smaller
			CreatedAt: input.GetCreatedAt(),
			UpdatedAt: input.GetUpdatedAt(),
			URL:       input.GetHTMLURL(),
		},
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
			ID:  input.GetURL(), // FIXME: make it smaller
			URL: input.GetURL(),
		},
		Name:        input.GetName(),
		Color:       input.GetColor(),
		Description: input.GetDescription(),
	}
}

func FromIssue(input *github.Issue) *model.Issue {
	entity, err := model.ParseTarget(input.GetHTMLURL())
	if err != nil {
		panic(err)
	}
	repo := multipmuri.RepoEntity(entity)
	owner := multipmuri.OwnerEntity(entity)
	service := multipmuri.ServiceEntity(entity)

	issue := &model.Issue{
		Base: model.Base{
			ID:        entity.String(),
			URL:       entity.String(),
			CreatedAt: input.GetCreatedAt(),
			UpdatedAt: input.GetUpdatedAt(),
		},
		CompletedAt:     input.GetClosedAt(),
		Repository:      FromRepositoryURL(repo.String()),
		RepositoryOwner: FromOwnerURL(owner.String()),
		Service:         FromServiceURL(service.String()),
		Title:           input.GetTitle(),
		State:           input.GetState(),
		Body:            input.GetBody(),
		IsPR:            input.PullRequestLinks != nil,
		IsLocked:        input.GetLocked(),
		NumComments:     input.GetComments(),
		NumUpvotes:      *input.Reactions.PlusOne,
		NumDownvotes:    *input.Reactions.MinusOne,
		Labels:          make([]*model.Label, 0),
		Assignees:       make([]*model.Account, 0),
		Author:          FromUser(input.User),
		Milestone:       FromMilestone(input.Milestone),
	}
	for _, label := range input.Labels {
		issue.Labels = append(issue.Labels, FromLabel(&label))
	}
	for _, assignee := range input.Assignees {
		issue.Assignees = append(issue.Assignees, FromUser(assignee))
	}
	return issue
}

func FromServiceURL(input string) *model.Provider {
	return &model.Provider{
		Base: model.Base{
			ID:  input,
			URL: input,
		},
		Driver: string(model.GithubDriver),
	}
}
