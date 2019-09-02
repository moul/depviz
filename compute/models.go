package compute

import (
	"moul.io/depviz/model"
	"moul.io/multipmuri"
	"moul.io/multipmuri/pmbodyparser"
)

//
// ComputedIssue
//

type ComputedIssue struct {
	model.Issue
	DirectMatchWithTarget bool
	Hidden                bool
	DependsOn             []string
	Relationships         pmbodyparser.Relationships
	Errs                  []error
}

func (i ComputedIssue) MultipmuriEntity() multipmuri.Entity {
	// FIXME: can be optimized by creating object directly
	entity, err := multipmuri.DecodeString(i.URL)
	if err != nil {
		panic(err)
	}
	return entity
}

func newComputedIssue(issue *model.Issue) *ComputedIssue {
	return &ComputedIssue{
		Issue:     *issue,
		DependsOn: []string{},
		Errs:      []error{},
	}
}

func (i *ComputedIssue) parseBody() {
	relationships, errs := pmbodyparser.RelParseString(
		i.MultipmuriEntity(),
		i.Body,
	)
	if errs != nil && len(errs) > 0 {
		i.Errs = append(i.Errs, errs...)
	}
	i.Relationships = relationships
}

//
// ComputedMilestone
//

type ComputedMilestone struct {
	model.Milestone
	DirectMatchWithTarget bool
	Hidden                bool
	DependsOn             []string
}

func (m ComputedMilestone) MultipmuriEntity() multipmuri.Entity {
	// FIXME: can be optimized by creating object directly
	entity, err := multipmuri.DecodeString(m.URL)
	if err != nil {
		panic(err)
	}
	return entity
}

func newComputedMilestone(milestone *model.Milestone) *ComputedMilestone {
	return &ComputedMilestone{
		Milestone: *milestone,
		DependsOn: []string{},
	}
}

func (c *Computed) getOrCreateMilestone(input *model.Milestone) *ComputedMilestone {
	if _, found := c.mmap[input.URL]; !found {
		c.mmap[input.URL] = newComputedMilestone(input)
	}
	return c.mmap[input.URL]
}

//
// ComputedRepo
//

type ComputedRepo struct {
	model.Repository
	DirectMatchWithTarget bool
	Hidden                bool
	DependsOn             []string
}

func (r ComputedRepo) MultipmuriEntity() multipmuri.Entity {
	// FIXME: can be optimized by creating object directly
	entity, err := multipmuri.DecodeString(r.URL)
	if err != nil {
		panic(err)
	}
	return entity
}

func newComputedRepo(repo *model.Repository) *ComputedRepo {
	return &ComputedRepo{
		Repository: *repo,
		DependsOn:  []string{},
	}
}

func (c *Computed) getOrCreateRepo(input *model.Repository) *ComputedRepo {
	if _, found := c.rmap[input.URL]; !found {
		c.rmap[input.URL] = newComputedRepo(input)
	}
	return c.rmap[input.URL]
}
