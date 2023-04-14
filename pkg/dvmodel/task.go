package dvmodel

import (
	"strings"

	"go.uber.org/zap"

	"github.com/cayleygraph/quad"
	"github.com/goccy/go-graphviz/cgraph"
)

func (t *Task) AllDeps() []quad.IRI {
	if len(t.IsDependingOn) < 1 && len(t.IsBlocking) < 1 {
		return nil
	}
	allDeps := make([]quad.IRI, len(t.IsDependingOn)+len(t.IsBlocking))
	copy(allDeps, t.IsDependingOn)
	n := len(t.IsDependingOn)
	for i, dep := range t.IsBlocking {
		allDeps[n+i] = dep
	}
	return allDeps
}

func FilterIsolatedTasks(in []Task, logger *zap.Logger) []Task {
	uniqueDeps := map[quad.IRI]*Task{}

	for _, task := range in {
		for _, dep := range task.IsDependingOn {
			uniqueDeps[task.ID] = nil
			uniqueDeps[dep] = nil
		}
		for _, dep := range task.IsBlocking {
			uniqueDeps[task.ID] = nil
			uniqueDeps[dep] = nil
		}
		for _, dep := range task.IsRelatedWith {
			uniqueDeps[task.ID] = nil
			uniqueDeps[dep] = nil
		}

		// FIXME: need to determine the relationship between the two and print it decently
		//for _, dep := range task.IsPartOf {
		//	uniqueDeps[task.ID] = nil
		//	uniqueDeps[dep] = nil
		//}
		//for _, dep := range task.HasPart {
		//	uniqueDeps[task.ID] = nil
		//	uniqueDeps[dep] = nil
		//}
	}

	for _, task := range in {
		taskCopy := task
		if _, found := uniqueDeps[task.ID]; found {
			uniqueDeps[task.ID] = &taskCopy
		}
	}

	out := make([]Task, len(uniqueDeps))
	i := 0
	for key, dep := range uniqueDeps {
		if dep == nil {
			logger.Warn("nil dep", zap.Any("key", key))
		} else {
			out[i] = *dep
			i++
		}
	}

	return out
}

func (t *Task) MarshalCSV() []string {
	if t == nil {
		return nil
	}
	return []string{
		t.ID.String(),
		t.CreatedAt.String(),
		t.UpdatedAt.String(),
		t.LocalID,
		t.Kind.String(),
		t.Title,
		t.Description,
		t.Driver.String(),
		t.State.String(),
		t.EstimatedDuration,
		t.HasAuthor.String(),
		t.HasOwner.String(),
		// t.IsDependingOn,
		// t.IsBlocking,
	}
}

type fmtLabel struct {
	label string
	style string
	color string
}

// special depviz labels, used to colorize nodes in the graphviz generation
// TODO: determine a way to create 'themes' with custom config with the following format
var depvizLabels = [...]fmtLabel{
	{"focus", "filled,bold,rounded", "#ffeeee"},
	{"vision", "filled,rounded", "#eeeeee"},
}

// ApplyLabel apply modifications to the Node based on the label of the task
func (t Task) ApplyLabel(node *cgraph.Node) {
	if t.Driver == Driver_GitHub {
		for _, label := range t.HasLabel {
			s, _ := strings.CutPrefix(label.String(), t.HasOwner.String()+"/labels/")
			for _, dl := range depvizLabels {
				if s == dl.label {
					node.SetStyle(cgraph.NodeStyle(dl.style))
					node.SetColor(dl.color)
					return
				}
			}
		}
	}
}
