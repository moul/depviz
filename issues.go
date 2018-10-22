package main

type Issues []*Issue

func (issues Issues) Get(id string) *Issue {
	for _, issue := range issues {
		if issue.ID == id {
			return issue
		}
	}
	return nil
}

func (issues Issues) FilterByTargets(targets []Target) Issues {
	filtered := Issues{}

	for _, issue := range issues {
		if issue.MatchesWithATarget(targets) {
			filtered = append(filtered, issue)
		}
	}

	return filtered
}
