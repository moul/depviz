package dvmodel

import (
	"fmt"
)

func (tasks Tasks) DebugTree(useLocalID bool, withRelationships bool) string {
	out := ""
	for _, task := range tasks {
		out += fmt.Sprintf("%s\n", task.DebugLine(useLocalID))
		if withRelationships {
			if task.HasAuthor != "" {
				out += fmt.Sprintf("    hasAuthor     --> %s\n", task.HasAuthor)
			}
			if task.HasOwner != "" {
				out += fmt.Sprintf("    hasOwner      --> %s\n", task.HasOwner)
			}
			if task.HasMilestone != "" {
				out += fmt.Sprintf("    hasMilestone  --> %s\n", task.HasMilestone)
			}
			for _, dep := range task.IsDependingOn {
				out += fmt.Sprintf("    isDependingOn --> %s\n", dep)
			}
			for _, dep := range task.IsBlocking {
				out += fmt.Sprintf("    isBlocking    --> %s\n", dep)
			}
			for _, dep := range task.HasLabel {
				out += fmt.Sprintf("    hasLabel      --> %s\n", dep)
			}
			for _, dep := range task.HasAssignee {
				out += fmt.Sprintf("    hasAssignee   --> %s\n", dep)
			}
			for _, dep := range task.HasReviewer {
				out += fmt.Sprintf("    hasReviewer   --> %s\n", dep)
			}
			for _, dep := range task.IsRelatedWith {
				out += fmt.Sprintf("    isRelatedWith --> %s\n", dep)
			}
			for _, dep := range task.IsPartOf {
				out += fmt.Sprintf("    isPartOf      --> %s\n", dep)
			}
			for _, dep := range task.HasPart {
				out += fmt.Sprintf("    hasPart       --> %s\n", dep)
			}
		}
	}
	return out
}
