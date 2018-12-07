package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"moul.io/depviz/pkg/repo"
	"moul.io/zapgorm"
)

func main() {
	defer zap.L().Sync()
	rootCmd := newRootCommand()
	if err := rootCmd.Execute(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

var (
	verbose bool
	cfgFile string
	dbPath  string
	db      *gorm.DB
)

type DepvizCommand interface {
	NewCobraCommand(map[string]DepvizCommand) *cobra.Command
	LoadDefaultOptions() error
	ParseFlags(*pflag.FlagSet)
}

func newRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use: "depviz",
	}
	rootCmd.PersistentFlags().BoolP("help", "h", false, "print usage")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode")
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./.depviz.yml)")
	rootCmd.PersistentFlags().StringVarP(&dbPath, "db-path", "", "$HOME/.depviz.db", "database path")

	// Add commands here.
	cmds := map[string]DepvizCommand {
		"pull": &pullCommand{},
		"db": &dbCommand{},
		"airtable": &airtableCommand{},
		"graph": &graphCommand{},
		"run": &runCommand{},
		"web": &webCommand{},
	}

	rootCmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// configure zap
		config := zap.NewDevelopmentConfig()
		if verbose {
			config.Level.SetLevel(zapcore.DebugLevel)
		} else {
			config.Level.SetLevel(zapcore.InfoLevel)
		}
		config.DisableStacktrace = true
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		l, err := config.Build()
		if err != nil {
			return err
		}
		zap.ReplaceGlobals(l)

		// configure viper
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

		for _, cmd := range cmds {
			if err := cmd.LoadDefaultOptions(); err != nil {
				return err
			}
		}

		// configure sql
		dbPath = os.ExpandEnv(dbPath)
		db, err = gorm.Open("sqlite3", dbPath)
		if err != nil {
			return err
		}
		log.SetOutput(ioutil.Discard)
		db.Callback().Create().Remove("gorm:update_time_stamp")
		db.Callback().Update().Remove("gorm:update_time_stamp")
		log.SetOutput(os.Stderr)
		db.SetLogger(zapgorm.New(zap.L().Named("vendor.gorm")))
		db = db.Set("gorm:auto_preload", true)
		db = db.Set("gorm:association_autoupdate", true)
		db.BlockGlobalUpdate(true)
		db.SingularTable(true)
		db.LogMode(verbose)
		if err := db.AutoMigrate(
			repo.Issue{},
			repo.Label{},
			repo.Account{},
			repo.Milestone{},
			repo.Repository{},
			repo.Provider{},
		).Error; err != nil {
			return err
		}

		return nil
	}
	for _, cmd := range cmds {
		rootCmd.AddCommand(cmd.NewCobraCommand(cmds))
	}
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return rootCmd
}
