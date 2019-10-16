package compute // import "moul.io/depviz/compute"

import (
	"fmt"
	"sort"

	"moul.io/depviz/model"
	"moul.io/multipmuri"
	"moul.io/multipmuri/pmbodyparser"
)

//
// Computed
//

type Computed struct {
	AllIssues     []*ComputedIssue
	AllMilestones []*ComputedMilestone
	AllRepos      []*ComputedRepo

	// internal
	mmap map[string]*ComputedMilestone
	imap map[string]*ComputedIssue
	rmap map[string]*ComputedRepo
}

func Compute(input model.Issues) Computed {
	computed := newComputed()
	for _, issue := range input {
		// issue
		issue := newComputedIssue(issue)
		issue.parseBody()
		computed.imap[issue.URL] = issue

		// repo
		repo := computed.getOrCreateRepo(issue.Repository)
		repo.DependsOn = append(repo.DependsOn, issue.URL)

		// milestone
		if issue.Milestone != nil {
			milestone := computed.getOrCreateMilestone(issue.Milestone)
			milestone.DependsOn = append(milestone.DependsOn, issue.URL)
			// FIXME: a milestone belongs to a repo
		}
	}
	for _, milestone := range computed.mmap {
		repo := computed.getOrCreateRepo(milestone.Repository)
		repo.DependsOn = append(repo.DependsOn, milestone.URL)
	}
	for _, issue := range computed.imap {
		for _, relationship := range issue.Relationships {
			switch relationship.Kind {
			case pmbodyparser.Blocks, pmbodyparser.Fixes, pmbodyparser.Closes, pmbodyparser.Addresses, pmbodyparser.PartOf:
				if relatedIssue, found := computed.imap[relationship.Target.String()]; found {
					relatedIssue.DependsOn = append(relatedIssue.DependsOn, issue.URL)
				} else {
					issue.Errs = append(issue.Errs, fmt.Errorf("is dependent of a missing issue: %q", relationship.Target.String()))
					// FIXME: create dummy issue?
				}
			case pmbodyparser.DependsOn, pmbodyparser.ParentOf:
				issue.DependsOn = append(issue.DependsOn, relationship.Target.String())
			case pmbodyparser.RelatedWith:
				// nothing to do (for now)
			default:
				panic(fmt.Errorf("unsupported pmbodyparser.Kind: %q", relationship.Kind))
			}
		}
	}

	computed.mapsToSlices()
	return computed
}

func (computed *Computed) FilterByTargets(targets []multipmuri.Entity) {
	for _, issue := range computed.AllIssues {
		issueEntity := issue.MultipmuriEntity()
		for _, target := range targets {
			if issueEntity.Equals(target) || target.Contains(issueEntity) {
				issue.DirectMatchWithTarget = true
				break
			}
		}
		if !issue.DirectMatchWithTarget {
			issue.Hidden = true
		}
	}
	for _, milestone := range computed.AllMilestones {
		milestoneEntity := milestone.MultipmuriEntity()
		for _, target := range targets {
			if milestoneEntity.Equals(target) || target.Contains(milestoneEntity) {
				milestone.DirectMatchWithTarget = true
				break
			}
		}
		if !milestone.DirectMatchWithTarget {
			milestone.Hidden = true
		}
	}
	for _, repo := range computed.AllRepos {
		repoEntity := repo.MultipmuriEntity()
		for _, target := range targets {
			if repoEntity.Equals(target) || target.Contains(repoEntity) {
				repo.DirectMatchWithTarget = true
				break
			}
		}
		if !repo.DirectMatchWithTarget {
			repo.Hidden = true
		}
	}
	// FIXME: check for "indirect" matches too
}

func (computed *Computed) IssueByURL(url string) *ComputedIssue {
	for _, issue := range computed.AllIssues {
		if issue.URL == url {
			return issue
		}
	}
	return nil
}

func (computed *Computed) FilterClosed() {
	for _, issue := range computed.AllIssues {
		if issue.State == "closed" {
			issue.Hidden = true
		}
	}

	for _, milestone := range computed.AllMilestones {
		hasDeps := false
		for _, dep := range milestone.DependsOn {
			issue := computed.IssueByURL(dep)
			if issue == nil {
				// if we have at least one unknown dependency, we need to keep the whole object
				hasDeps = true
				break
			}
			if !issue.Hidden {
				hasDeps = true
				break
			}
		}
		if !hasDeps {
			milestone.Hidden = true
		}
	}
}

func newComputed() Computed {
	return Computed{
		AllIssues:     make([]*ComputedIssue, 0),
		AllMilestones: make([]*ComputedMilestone, 0),
		AllRepos:      make([]*ComputedRepo, 0),

		mmap: map[string]*ComputedMilestone{},
		imap: map[string]*ComputedIssue{},
		rmap: map[string]*ComputedRepo{},
	}
}

func (computed *Computed) mapsToSlices() {
	// generated sorted Computed object
	// milestones
	for _, milestone := range computed.mmap {
		sort.Strings(milestone.DependsOn)
		computed.AllMilestones = append(computed.AllMilestones, milestone)
	}
	sort.Slice(computed.AllMilestones, func(i, j int) bool {
		return computed.AllMilestones[i].URL < computed.AllMilestones[j].URL
	})
	// repos
	for _, repo := range computed.rmap {
		sort.Strings(repo.DependsOn)
		computed.AllRepos = append(computed.AllRepos, repo)
	}
	sort.Slice(computed.AllRepos, func(i, j int) bool {
		return computed.AllRepos[i].URL < computed.AllRepos[j].URL
	})
	// issues
	for _, issue := range computed.imap {
		sort.Strings(issue.DependsOn)
		computed.AllIssues = append(computed.AllIssues, issue)
	}
	sort.Slice(computed.AllIssues, func(i, j int) bool {
		return computed.AllIssues[i].URL < computed.AllIssues[j].URL
	})
}

func (computed *Computed) Repos() []*ComputedRepo {
	enabled := []*ComputedRepo{}
	for _, repo := range computed.AllRepos {
		if !repo.Hidden {
			enabled = append(enabled, repo)
		}
	}
	return enabled
}

func (computed *Computed) Milestones() []*ComputedMilestone {
	enabled := []*ComputedMilestone{}
	for _, milestone := range computed.AllMilestones {
		if !milestone.Hidden {
			enabled = append(enabled, milestone)
		}
	}
	return enabled
}

func (computed *Computed) Issues() []*ComputedIssue {
	enabled := []*ComputedIssue{}
	for _, issue := range computed.AllIssues {
		if !issue.Hidden {
			enabled = append(enabled, issue)
		}
	}
	return enabled
}
