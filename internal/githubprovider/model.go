package githubprovider

import (
	"fmt"

	"github.com/cayleygraph/quad"
	"github.com/google/go-github/v30/github"
	"go.uber.org/zap"
	"moul.io/depviz/v3/internal/dvmodel"
	"moul.io/depviz/v3/internal/dvparser"
	"moul.io/multipmuri"
	"moul.io/multipmuri/pmbodyparser"
)

func fromIssues(issues []*github.Issue, logger *zap.Logger) dvmodel.Batch {
	batch := dvmodel.Batch{}
	for _, issue := range issues {
		err := fromIssue(&batch, issue)
		if err != nil {
			logger.Warn("parse issue", zap.String("url", issue.GetHTMLURL()), zap.Error(err))
			continue
		}
	}
	return batch
}

func fromIssue(batch *dvmodel.Batch, input *github.Issue) error {
	entity, err := dvparser.ParseTarget(input.GetHTMLURL())
	if err != nil {
		return fmt.Errorf("parse target: %w", err)
	}

	//
	// the issue
	//

	issue := dvmodel.Task{
		ID:           quad.IRI(entity.String()),
		LocalID:      entity.LocalID(),
		CreatedAt:    input.CreatedAt,
		UpdatedAt:    input.UpdatedAt,
		Title:        input.GetTitle(),
		Description:  input.GetBody(),
		Driver:       dvmodel.Driver_GitHub,
		IsLocked:     input.GetLocked(),
		CompletedAt:  input.ClosedAt,
		NumComments:  int32(input.GetComments()),
		NumUpvotes:   int32(*input.Reactions.PlusOne),
		NumDownvotes: int32(*input.Reactions.MinusOne),
	}
	if input.PullRequestLinks != nil { // is PR
		issue.Kind = dvmodel.Task_MergeRequest
	} else { // is issue
		issue.Kind = dvmodel.Task_Issue
	}
	switch state := input.GetState(); state {
	case "open":
		issue.State = dvmodel.Task_Open
	case "closed":
		issue.State = dvmodel.Task_Closed
	default:
		return fmt.Errorf("unsupported state: %q", state)
	}

	//
	// relationships
	//

	// author
	author, err := fromUser(batch, input.User)
	if err != nil {
		return fmt.Errorf("from user: %w", err)
	}
	issue.HasAuthor = author.ID

	// repo
	repo, err := fromRepoURL(batch, multipmuri.RepoEntity(entity).String())
	if err != nil {
		return fmt.Errorf("from repo URL: %w", err)
	}
	issue.HasOwner = repo.ID

	// milestone
	if input.Milestone != nil {
		milestone, err := fromMilestone(batch, input.Milestone)
		if err != nil {
			return fmt.Errorf("from milestone: %w", err)
		}
		issue.HasMilestone = milestone.ID
	}

	// assignees
	for _, assignee := range input.Assignees {
		assigneeRet, err := fromUser(batch, assignee)
		if err != nil {
			return fmt.Errorf("from user: %w", err)
		}
		issue.HasAssignee = append(issue.HasAssignee, assigneeRet.ID)
	}

	// reviewers
	// FIXME: TODO: HasReviewer

	// projects
	// FIXME: TODO

	// labels
	for _, label := range input.Labels {
		labelRet, err := fromLabel(batch, label)
		if err != nil {
			return fmt.Errorf("from label: %w", err)
		}
		issue.HasLabel = append(issue.HasLabel, labelRet.ID)
	}

	// parse body
	relationships, errs := pmbodyparser.RelParseString(entity, issue.Description)
	if len(errs) > 0 {
		for _, err := range errs {
			return fmt.Errorf("pmbodyparser error: %w", err)
		}
	}
	for _, relationship := range relationships {
		switch relationship.Kind {
		case pmbodyparser.Blocks,
			pmbodyparser.Fixes,
			pmbodyparser.Closes,
			pmbodyparser.Addresses:
			issue.IsBlocking = append(issue.IsBlocking, quad.IRI(relationship.Target.String()))
		case pmbodyparser.DependsOn:
			issue.IsDependingOn = append(issue.IsDependingOn, quad.IRI(relationship.Target.String()))
		case pmbodyparser.RelatedWith:
			issue.IsRelatedWith = append(issue.IsRelatedWith, quad.IRI(relationship.Target.String()))
		case pmbodyparser.PartOf:
			issue.IsPartOf = append(issue.IsPartOf, quad.IRI(relationship.Target.String()))
		case pmbodyparser.ParentOf:
			issue.HasPart = append(issue.HasPart, quad.IRI(relationship.Target.String()))
		default:
			return fmt.Errorf("unsupported pmbodyparser.Kind: %v", relationship.Kind)
		}
	}

	batch.Tasks = append(batch.Tasks, &issue)
	return nil
}

func fromUser(batch *dvmodel.Batch, input *github.User) (*dvmodel.Owner, error) {
	entity, err := dvparser.ParseTarget(input.GetHTMLURL())
	if err != nil {
		return nil, err
	}

	name := input.GetName()
	if name == "" {
		name = input.GetLogin()
	}
	description := ""
	if location := input.GetLocation(); location != "" {
		description += fmt.Sprintf("Location: %s\n", location)
	}
	if company := input.GetCompany(); company != "" {
		description += fmt.Sprintf("Company: %s\n", company)
	}
	if email := input.GetEmail(); email != "" {
		description += fmt.Sprintf("Email: %s\n", email)
	}
	user := dvmodel.Owner{
		ID:          quad.IRI(entity.String()),
		LocalID:     entity.LocalID(),
		Kind:        dvmodel.Owner_User,
		FullName:    name,
		ShortName:   input.GetLogin(),
		Driver:      dvmodel.Driver_GitHub,
		Homepage:    input.GetBlog(),
		AvatarURL:   input.GetAvatarURL(),
		ForkStatus:  dvmodel.Owner_UnknownForkStatus,
		Description: description,
	}
	if input.CreatedAt != nil {
		created := input.GetCreatedAt().Time
		user.CreatedAt = &created
	}
	if input.UpdatedAt != nil {
		updated := input.GetUpdatedAt().Time
		user.UpdatedAt = &updated
	}
	batch.Owners = append(batch.Owners, &user)
	return &user, nil
}

func fromMilestone(batch *dvmodel.Batch, input *github.Milestone) (*dvmodel.Task, error) {
	entity, err := dvparser.ParseTarget(input.GetHTMLURL())
	if err != nil {
		return nil, err
	}

	milestone := dvmodel.Task{
		ID:          quad.IRI(entity.String()),
		LocalID:     entity.LocalID(),
		Kind:        dvmodel.Task_Milestone,
		CreatedAt:   input.CreatedAt,
		UpdatedAt:   input.UpdatedAt,
		Title:       input.GetTitle(),
		Description: input.GetDescription(),
		Driver:      dvmodel.Driver_GitHub,
	}
	switch state := input.GetState(); state {
	case "open":
		milestone.State = dvmodel.Task_Open
	case "closed":
		milestone.State = dvmodel.Task_Closed
	default:
		return nil, fmt.Errorf("unsupported state: %q", state)
	}
	if input.DueOn != nil {
		dueOn := input.GetDueOn()
		milestone.DueOn = &dueOn
	}
	if input.ClosedAt != nil {
		completedAt := input.GetClosedAt()
		milestone.CompletedAt = &completedAt
	}

	//
	// Relationships
	//

	// author
	author, err := fromUser(batch, input.Creator)
	if err != nil {
		return nil, err
	}
	milestone.HasAuthor = author.ID

	// repo
	repo := multipmuri.RepoEntity(entity)
	milestone.HasOwner = quad.IRI(repo.String())

	batch.Tasks = append(batch.Tasks, &milestone)
	return &milestone, err
}

func fromRepoURL(batch *dvmodel.Batch, url string) (*dvmodel.Owner, error) {
	entity, err := dvparser.ParseTarget(url)
	if err != nil {
		return nil, err
	}

	repo := dvmodel.Owner{
		ID:      quad.IRI(entity.String()),
		LocalID: entity.LocalID(),
		Kind:    dvmodel.Owner_Repo,
		Driver:  dvmodel.Driver_GitHub,
	}

	// repo owner
	repoOwner := multipmuri.OwnerEntity(entity)
	repo.HasOwner = quad.IRI(repoOwner.String())

	batch.Owners = append(batch.Owners, &repo)
	return &repo, err
}

func fromLabel(batch *dvmodel.Batch, input *github.Label) (*dvmodel.Topic, error) {
	entity, err := dvparser.ParseTarget(input.GetURL())
	if err != nil {
		return nil, err
	}

	topic := dvmodel.Topic{
		ID:          quad.IRI(entity.String()),
		LocalID:     entity.LocalID(),
		Title:       input.GetName(),
		Color:       "#" + input.GetColor(),
		Description: input.GetDescription(),
	}

	repo := multipmuri.RepoEntity(entity)
	topic.HasOwner = quad.IRI(repo.String())

	batch.Topics = append(batch.Topics, &topic)
	return &topic, nil
}
