package main // import "moul.io/depviz"

import (
	"fmt"
	"os"

	"go.uber.org/zap"

	"moul.io/depviz/cli"
)

func main() {
	defer func() {
		_ = zap.L().Sync()
	}()
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
