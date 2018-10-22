package main

import (
	"fmt"
	"log"
	"strings"
)

func (i Issue) GetRelativeURL(target string) string {
	if strings.Contains(target, "://") {
		return normalizeURL(target)
	}

	if target[0] == '#' {
		return fmt.Sprintf("%s/issues/%s", i.Repository.URL, target[1:])
	}

	target = strings.Replace(target, "#", "/issues/", -1)

	parts := strings.Split(target, "/")
	if strings.Contains(parts[0], ".") && isDNSName(parts[0]) {
		return fmt.Sprintf("https://%s", target)
	}

	return fmt.Sprintf("%s/%s", strings.TrimRight(i.Repository.Provider.URL, "/"), target)
}

func (i *Issue) PostLoad() {
	i.ParentIDs = []string{}
	i.ChildIDs = []string{}
	i.DuplicateIDs = []string{}
	for _, rel := range i.Parents {
		i.ParentIDs = append(i.ParentIDs, rel.ID)
	}
	for _, rel := range i.Children {
		i.ChildIDs = append(i.ChildIDs, rel.ID)
	}
	for _, rel := range i.Duplicates {
		i.DuplicateIDs = append(i.DuplicateIDs, rel.ID)
	}
}

func (i Issue) IsClosed() bool {
	return i.State == "closed"
}

func (i Issue) IsReady() bool {
	return !i.IsOrphan && len(i.Parents) == 0 // FIXME: switch parents with children?
}

func (i Issue) MatchesWithATarget(targets Targets) bool {
	return i.matchesWithATarget(targets, 0)
}

func (i Issue) matchesWithATarget(targets Targets, depth int) bool {
	if depth > 100 {
		log.Printf("circular dependency or too deep graph (>100), skipping this node. (issue=%s)", i)
		return false
	}

	for _, target := range targets {
		if target.Issue() != "" { // issue-mode
			if target.Canonical() == i.URL {
				return true
			}
		} else { // project-mode
			if i.RepositoryID == target.ProjectURL() {
				return true
			}
		}
	}

	for _, parent := range i.Parents {
		if parent.matchesWithATarget(targets, depth+1) {
			return true
		}
	}

	for _, child := range i.Children {
		if child.matchesWithATarget(targets, depth+1) {
			return true
		}
	}

	return false
}
