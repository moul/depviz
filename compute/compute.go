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
	Issues     []*ComputedIssue
	Milestones []*ComputedMilestone
	Repos      []*ComputedRepo

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
	for _, issue := range computed.Issues {
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
	for _, milestone := range computed.Milestones {
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
	for _, repo := range computed.Repos {
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

func newComputed() Computed {
	return Computed{
		Issues:     make([]*ComputedIssue, 0),
		Milestones: make([]*ComputedMilestone, 0),
		Repos:      make([]*ComputedRepo, 0),

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
		computed.Milestones = append(computed.Milestones, milestone)
	}
	sort.Slice(computed.Milestones, func(i, j int) bool {
		return computed.Milestones[i].URL < computed.Milestones[j].URL
	})
	// repos
	for _, repo := range computed.rmap {
		sort.Strings(repo.DependsOn)
		computed.Repos = append(computed.Repos, repo)
	}
	sort.Slice(computed.Repos, func(i, j int) bool {
		return computed.Repos[i].URL < computed.Repos[j].URL
	})
	// issues
	for _, issue := range computed.imap {
		sort.Strings(issue.DependsOn)
		computed.Issues = append(computed.Issues, issue)
	}
	sort.Slice(computed.Issues, func(i, j int) bool {
		return computed.Issues[i].URL < computed.Issues[j].URL
	})
}
