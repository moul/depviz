package main

import (
	"fmt"
	"regexp"
	"strconv"
)

type IssueSlice []*Issue

func (s IssueSlice) Unique() IssueSlice {
	return s.ToMap().ToSlice()
}

type Issues map[string]*Issue

func (m Issues) ToSlice() IssueSlice {
	slice := IssueSlice{}
	for _, issue := range m {
		slice = append(slice, issue)
	}
	return slice
}

func (s IssueSlice) ToMap() Issues {
	m := Issues{}
	for _, issue := range s {
		m[issue.URL] = issue
	}
	return m
}

func (issues Issues) prepare() error {
	var (
		dependsOnRegex, _        = regexp.Compile(`(?i)(require|requires|blocked by|block by|depend on|depends on|parent of) ([a-z0-9:/_.-]+|[a-z0-9/_-]*#[0-9]+)`)
		blocksRegex, _           = regexp.Compile(`(?i)(blocks|block|address|addresses|part of|child of|fix|fixes) ([a-z0-9:/_.-]+|[a-z0-9/_-]*#[0-9]+)`)
		isDuplicateRegex, _      = regexp.Compile(`(?i)(duplicates|duplicate|dup of|dup|duplicate of) ([a-z0-9:/_.-]+|[a-z0-9/_-]*#[0-9]+)`)
		weightMultiplierRegex, _ = regexp.Compile(`(?i)(depviz.weight_multiplier[:= ]+)([0-9]+)`)
		baseWeightRegex, _       = regexp.Compile(`(?i)(depviz.base_weight[:= ]+)([0-9]+)`)
		hideFromRoadmapRegex, _  = regexp.Compile(`(?i)(depviz.hide_from_roadmap)`) // FIXME: use label
	)

	for _, issue := range issues {
		issue.DependsOn = make([]*Issue, 0)
		issue.Blocks = make([]*Issue, 0)
		issue.IsOrphan = true
		issue.weightMultiplier = 1
		issue.BaseWeight = 1
	}
	for _, issue := range issues {
		if issue.Body == "" {
			continue
		}

		if match := isDuplicateRegex.FindStringSubmatch(issue.Body); match != nil {
			issue.Duplicates = append(issue.Duplicates, issue.GetRelativeIssueURL(match[len(match)-1]))
		}

		if match := weightMultiplierRegex.FindStringSubmatch(issue.Body); match != nil {
			issue.weightMultiplier, _ = strconv.Atoi(match[len(match)-1])
		}

		if match := hideFromRoadmapRegex.FindStringSubmatch(issue.Body); match != nil {
			delete(issues, issue.URL)
			continue
		}

		if match := baseWeightRegex.FindStringSubmatch(issue.Body); match != nil {
			issue.BaseWeight, _ = strconv.Atoi(match[len(match)-1])
		}

		for _, match := range dependsOnRegex.FindAllStringSubmatch(issue.Body, -1) {
			num := issue.GetRelativeIssueURL(match[len(match)-1])
			dep, found := issues[num]
			//fmt.Println(issue.URL, num, found, match[len(match)-1])
			if !found {
				issue.Errors = append(issue.Errors, fmt.Errorf("parent %q not found", num))
				continue
			}
			issue.DependsOn = append(issue.DependsOn, dep)
			issues[num].Blocks = append(dep.Blocks, issue)
			issue.IsOrphan = false
			issues[num].IsOrphan = false
		}

		for _, match := range blocksRegex.FindAllStringSubmatch(issue.Body, -1) {
			num := issue.GetRelativeIssueURL(match[len(match)-1])
			dep, found := issues[num]
			if !found {
				issue.Errors = append(issue.Errors, fmt.Errorf("child %q not found", num))
				continue
			}
			issues[num].DependsOn = append(dep.DependsOn, issue)
			issue.Blocks = append(issue.Blocks, dep)
			issue.IsOrphan = false
			issues[num].IsOrphan = false
		}
	}
	for _, issue := range issues {
		if len(issue.Duplicates) > 0 {
			issue.Hidden = true
		}
		if issue.IsPR() {
			issue.Hidden = true
		}
	}
	issues.processEpicLinks()
	return nil
}

func (issues Issues) processEpicLinks() {
	for _, issue := range issues {
		issue.LinkedWithEpic = !issue.Hidden && (issue.IsEpic() || issue.BlocksAnEpic() || issue.DependsOnAnEpic())

	}
}

func (issues Issues) HideClosed() {
	for _, issue := range issues {
		if issue.IsClosed() {
			issue.Hidden = true
		}
	}
}

func (issues Issues) HideOrphans() {
	for _, issue := range issues {
		if issue.IsOrphan || !issue.LinkedWithEpic {
			issue.Hidden = true
		}
	}
}

func (issues Issues) HasOrphans() bool {
	for _, issue := range issues {
		if !issue.Hidden && issue.IsOrphan {
			return true
		}
	}
	return false
}

func (issues Issues) HasNonOrphans() bool {
	for _, issue := range issues {
		if !issue.Hidden && !issue.IsOrphan && issue.LinkedWithEpic {
			return true
		}
	}
	return false
}
