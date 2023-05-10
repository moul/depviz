package dvmodel

import (
	"moul.io/depviz/v3/pkg/multipmuri"
)

type Filters struct {
	Targets             []multipmuri.Entity
	TheWorld            bool
	WithClosed          bool
	WithoutIsolated     bool
	WithoutPRs          bool
	WithoutExternalDeps bool
	WithFetch           bool
}
