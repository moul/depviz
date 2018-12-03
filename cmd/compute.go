package main

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"moul.io/depviz/pkg/repo"

	"go.uber.org/zap"
)

// @TODO: Move all of this into repo?

var (
	childrenRegex, _    = regexp.Compile(`(?i)(require|requires|blocked by|block by|depend on|depends on|parent of) ([a-z0-9:/_.-]+issues/[0-9]+|[a-z0-9:/_.-]+#[0-9]+|[a-z0-9/_-]*#[0-9]+)`)
	parentsRegex, _     = regexp.Compile(`(?i)(blocks|block|address|addresses|part of|child of|fix|fixes) ([a-z0-9:/_.-]+issues/[0-9]+|[a-z0-9:/_.-]+#[0-9]+|[a-z0-9/_-]*#[0-9]+)`)
	isDuplicateRegex, _ = regexp.Compile(`(?i)(duplicates|duplicate|dup of|dup|duplicate of) ([a-z0-9:/_.-]+issues/[0-9]+|[a-z0-9:/_.-]+#[0-9]+|[a-z0-9/_-]*#[0-9]+)`)
	//weightMultiplierRegex, _ = regexp.Compile(`(?i)(depviz.weight_multiplier[:= ]+)([0-9]+)`)
	weightRegex, _ = regexp.Compile(`(?i)(depviz.base_weight|depviz.weight)[:= ]+([0-9]+)`)
	hideRegex, _   = regexp.Compile(`(?i)(depviz.hide)`) // FIXME: use label
)

func compute(opts *pullOptions) error {
	zap.L().Debug("compute", zap.Stringer("opts", *opts))
	issues, err := loadIssues(nil)
	if err != nil {
		return err
	}

	for _, issue := range issues {
		// reset default values
		issue.Errors = []string{}
		issue.Parents = []*repo.Issue{}
		issue.Children = []*repo.Issue{}
		issue.Duplicates = []*repo.Issue{}
		issue.Weight = 0
		issue.IsHidden = false
		issue.IsEpic = false
		issue.HasEpic = false
		issue.IsOrphan = true
	}

	for _, issue := range issues {
		if issue.Body == "" {
			continue
		}

		// is epic
		for _, label := range issue.Labels {
			// FIXME: get epic labels dynamically based on a configuration filein the repo
			if label.Name == "epic" || label.Name == "t/epic" {
				issue.IsEpic = true
			}
		}

		// hidden
		if match := hideRegex.FindStringSubmatch(issue.Body); match != nil {
			issue.IsHidden = true
			continue
		}

		// duplicates
		if match := isDuplicateRegex.FindStringSubmatch(issue.Body); match != nil {
			canonical := issue.GetRelativeURL(match[len(match)-1])
			rel := issues.Get(canonical)
			if rel == nil {
				issue.Errors = append(issue.Errors, fmt.Errorf("duplicate %q not found", canonical).Error())
				continue
			}
			issue.Duplicates = append(issue.Duplicates, rel)
			issue.IsHidden = true
			continue
		}

		// weight
		if match := weightRegex.FindStringSubmatch(issue.Body); match != nil {
			issue.Weight, _ = strconv.Atoi(match[len(match)-1])
		}

		// children
		for _, match := range childrenRegex.FindAllStringSubmatch(issue.Body, -1) {
			canonical := issue.GetRelativeURL(match[len(match)-1])
			child := issues.Get(canonical)
			if child == nil {
				issue.Errors = append(issue.Errors, fmt.Errorf("children %q not found", canonical).Error())
				continue
			}
			issue.Children = append(issue.Children, child)
			issue.IsOrphan = false
			child.Parents = append(child.Parents, issue)
			child.IsOrphan = false
		}

		// parents
		for _, match := range parentsRegex.FindAllStringSubmatch(issue.Body, -1) {
			canonical := issue.GetRelativeURL(match[len(match)-1])
			parent := issues.Get(canonical)
			if parent == nil {
				issue.Errors = append(issue.Errors, fmt.Errorf("parent %q not found", canonical).Error())
				continue
			}
			issue.Parents = append(issue.Parents, parent)
			issue.IsOrphan = false
			parent.Children = append(parent.Children, issue)
			parent.IsOrphan = false
		}
	}

	for _, issue := range issues {
		if issue.IsEpic {
			issue.HasEpic = true
			continue
		}
		// has epic
		issue.HasEpic, err = computeHasEpic(issue, 0)
		if err != nil {
			issue.Errors = append(issue.Errors, err.Error())
		}
	}

	for _, issue := range issues {
		issue.PostLoad()

		issue.ParentIDs = uniqueStrings(issue.ParentIDs)
		sort.Strings(issue.ParentIDs)
		issue.ChildIDs = uniqueStrings(issue.ChildIDs)
		sort.Strings(issue.ChildIDs)
		issue.DuplicateIDs = uniqueStrings(issue.DuplicateIDs)
		sort.Strings(issue.DuplicateIDs)
	}

	for _, issue := range issues {
		// TODO: add a "if changed" to preserve some CPU and time
		if err := db.Set("gorm:association_autoupdate", false).Save(issue).Error; err != nil {
			return err
		}
	}

	return nil
}

func computeHasEpic(i *repo.Issue, depth int) (bool, error) {
	if depth > 100 {
		return false, fmt.Errorf("very high blocking depth (>100), do not continue. (issue=%s)", i.URL)
	}
	if i.IsHidden {
		return false, nil
	}
	for _, parent := range i.Parents {
		if parent.IsEpic {
			return true, nil
		}
		parentHasEpic, err := computeHasEpic(parent, depth + 1)
		if err != nil {
			return false, nil
		}
		if parentHasEpic {
			return true, nil
		}
	}
	return false, nil
}

func uniqueStrings(input []string) []string {
	u := make([]string, 0, len(input))
	m := make(map[string]bool)

	for _, val := range input {
		if _, ok := m[val]; !ok {
			m[val] = true
			u = append(u, val)
		}
	}

	return u
}
