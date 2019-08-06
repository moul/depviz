package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Command interface {
	CobraCommand(Commands) *cobra.Command
	LoadDefaultOptions() error
	ParseFlags(*pflag.FlagSet)
}

type Commands map[string]Command
