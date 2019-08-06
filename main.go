package main // import "moul.io/depviz"

import (
	"fmt"
	"os"
	"strings"

	_ "github.com/mattn/go-sqlite3" // required by gorm
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	//"moul.io/depviz/airtable"
	"moul.io/depviz/cli"
	"moul.io/depviz/sql"

	//"moul.io/depviz/web"
	"moul.io/depviz/graph"
	"moul.io/depviz/pull"
)

func main() {
	// rand.Seed(time.Now().UnixNano())
	defer func() {
		_ = zap.L().Sync()
	}()
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newRootCommand() *cobra.Command {
	var (
		verbose bool
		cfgFile string
	)

	cmd := &cobra.Command{
		Use: os.Args[0],
	}
	cmd.PersistentFlags().BoolP("help", "h", false, "print usage")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode")
	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./.depviz.yml)")
	// FIXME: cmd.Version = ...

	// Add commands
	commands := cli.Commands{}
	for name, command := range sql.Commands() {
		commands[name] = command
	}
	for name, command := range pull.Commands() {
		commands[name] = command
	}
	for name, command := range graph.Commands() {
		commands[name] = command
	}
	/*
		for name, command := range airtable.Commands() {
			commands[name] = command
		}
		for name, command := range web.Commands() {
			commands[name] = command
		}
	*/

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// configure zap
		config := zap.NewDevelopmentConfig()
		if verbose {
			os.Setenv("DEPVIZ_DEBUG", "1") // FIXME: can be done in a more gopher way
			config.Level.SetLevel(zapcore.DebugLevel)
		} else {
			config.Level.SetLevel(zapcore.InfoLevel)
		}
		config.DisableStacktrace = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		l, err := config.Build()
		if err != nil {
			return errors.Wrap(err, "failed to configure logger")
		}
		zap.ReplaceGlobals(l)
		zap.L().Debug("logger initialized")

		// configure viper
		if cfgFile != "" {
			viper.SetConfigFile(cfgFile)
		} else {
			viper.AddConfigPath(".")
			viper.SetConfigName(".depviz")
		}
		//viper.SetEnvPrefix("DEPVIZ")
		viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
		viper.AutomaticEnv()
		if err := viper.MergeInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
				return errors.Wrap(err, "cannot read config")
			}
		}

		for _, command := range commands {
			if err := command.LoadDefaultOptions(); err != nil {
				return err
			}
		}

		return nil
	}
	for name, command := range commands {
		if strings.Contains(name, " ") { // do not add commands where level > 1
			continue
		}
		cmd.AddCommand(command.CobraCommand(commands))
	}
	return cmd
}
