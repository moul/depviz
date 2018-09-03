package main // import "moul.io/depviz"

import (
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func main() {
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

var (
	//verbose bool
	cfgFile string
)

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "depviz",
	}
	cmd.PersistentFlags().BoolP("help", "h", false, "Print usage")
	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./.depviz.yml)")
	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			viper.AddConfigPath(".")
			viper.SetConfigName(".depviz")
		}
		if err := viper.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return errors.Wrap(err, "cannot read config")
			}
		}
		return nil
	}
	cmd.AddCommand(
		newFetchCommand(),
		newRenderCommand(),
		newDBCommand(),
	)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
