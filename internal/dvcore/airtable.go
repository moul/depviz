package dvcore

import (
	"fmt"

	"github.com/cayleygraph/cayley"
)

type AirtableOpts struct {
	Token     string
	BaseID    string
	OwnersTab string
	TasksTab  string
	TopicsTab string
}

func AirtableSync(db *cayley.Handle, opts AirtableOpts) error {
	fmt.Println(db, opts)
	return fmt.Errorf("not implemented")
}

func AirtableInfo(opts AirtableOpts) error {
	fmt.Println(opts)
	return fmt.Errorf("not implemented")
}
