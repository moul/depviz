package dvcore

import (
	"context"
	"fmt"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/schema"
	"moul.io/depviz/internal/dvmodel"
)

func StoreDumpQuads(h *cayley.Handle) error {
	fmt.Println("quads:")
	ctx := context.Background()
	it := h.QuadsAllIterator()
	for it.Next(ctx) {
		fmt.Println(h.Quad(it.Result()))
	}

	return nil
}

func GetStoreDump(ctx context.Context, h *cayley.Handle, schema *schema.Config) (*dvmodel.Batch, error) {
	owners := []dvmodel.Owner{}
	if err := schema.LoadTo(ctx, h, &owners); err != nil {
		return nil, fmt.Errorf("load owners: %w", err)
	}
	tasks := []dvmodel.Task{}
	if err := schema.LoadTo(ctx, h, &tasks); err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}
	topics := []dvmodel.Topic{}
	if err := schema.LoadTo(ctx, h, &topics); err != nil {
		return nil, fmt.Errorf("load topics: %w", err)
	}

	dump := dvmodel.Batch{
		Owners: make([]*dvmodel.Owner, len(owners)),
		Tasks:  make([]*dvmodel.Task, len(tasks)),
		Topics: make([]*dvmodel.Topic, len(topics)),
	}
	for idx, owner := range owners {
		clone := owner
		dump.Owners[idx] = &clone
	}
	for idx, task := range tasks {
		clone := task
		dump.Tasks[idx] = &clone
	}
	for idx, topic := range topics {
		clone := topic
		dump.Topics[idx] = &clone
	}

	return &dump, nil
}

func StoreInfo(h *cayley.Handle) error {
	fmt.Println(h)
	// FIXME: amount of quads
	// FIXME: amount of owners, tasks, topics
	// FIXME: amount of relationships
	// FIXME: last refresh
	// FIXME: db size
	// FIXME: db location
	return fmt.Errorf("not implemented")
}
