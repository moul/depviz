package dvmodel

import (
	"github.com/cayleygraph/quad"
	"go.uber.org/zap"
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
			uniqueDeps[dep] = nil
		}
		for _, dep := range task.IsBlocking {
			uniqueDeps[dep] = nil
		}
		for _, dep := range task.IsRelatedWith {
			uniqueDeps[dep] = nil
		}
		for _, dep := range task.IsPartOf {
			uniqueDeps[dep] = nil
		}
		for _, dep := range task.HasPart {
			uniqueDeps[dep] = nil
		}
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
