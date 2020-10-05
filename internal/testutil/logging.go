package testutil

import (
	"flag"
	"testing"

	"go.uber.org/zap"
	"moul.io/zapconfig"
)

var debug = flag.Bool("debug", false, "more verbose logging")

func Logger(t *testing.T) *zap.Logger {
	t.Helper()
	if !*debug {
		return zap.NewNop()
	}

	return zapconfig.Configurator{}.MustBuild()
}
