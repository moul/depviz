package main // import "moul.io/depviz"

import (
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
	"moul.io/depviz/cli"
)

func main() {
	defer zap.L().Sync()
	rootCmd := cli.NewRootCommand()
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
