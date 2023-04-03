package dvstore

import (
	"github.com/cayleygraph/cayley/schema"
	"github.com/moul/depviz/v3/internal/dvmodel"
)

func Schema() *schema.Config {
	config := schema.NewConfig()
	// temporarily forced to register it globally :(
	schema.RegisterType("dv:Owner", dvmodel.Owner{})
	schema.RegisterType("dv:Task", dvmodel.Task{})
	schema.RegisterType("dv:Topic", dvmodel.Topic{})
	return config
}
