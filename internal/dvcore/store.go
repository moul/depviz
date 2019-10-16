package dvcore

import (
	"context"
	"fmt"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/schema"
	"moul.io/depviz/internal/dvmodel"
	"moul.io/godev"
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

func getStoreDump(h *cayley.Handle, schema *schema.Config) (*dvmodel.Batch, error) {
	dump := dvmodel.Batch{}
	ctx := context.TODO()

	if err := schema.LoadTo(ctx, h, &dump.Owners); err != nil {
		return nil, fmt.Errorf("load owners: %w", err)
	}
	if err := schema.LoadTo(ctx, h, &dump.Tasks); err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}
	if err := schema.LoadTo(ctx, h, &dump.Topics); err != nil {
		return nil, fmt.Errorf("load topics: %w", err)
	}

	return &dump, nil
}

func StoreDumpJSON(h *cayley.Handle, schema *schema.Config) error {
	dump, err := getStoreDump(h, schema)
	if err != nil {
		return err
	}

	fmt.Println(godev.PrettyJSON(dump))
	return nil
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
