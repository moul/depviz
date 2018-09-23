package main // import "moul.io/depviz"

import (
	"fmt"
	"os"
	"strings"

	"github.com/jinzhu/gorm"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"moul.io/zapgorm"
)

func main() {
	defer logger().Sync()
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

func newRootCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "depviz",
	}
	cmd.PersistentFlags().BoolP("help", "h", false, "print usage")
	cmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose mode")
	cmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "config file (default is ./.depviz.yml)")
	cmd.PersistentFlags().StringVarP(&dbPath, "db-path", "", "$HOME/.depviz.db", "database path")

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
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

		// configure sql
		dbPath = os.ExpandEnv(dbPath)
		db, err = gorm.Open("sqlite3", dbPath)
		if err != nil {
			return err
		}
		db.SetLogger(zapgorm.New(zap.L().Named("vendor.gorm")))
		db = db.Set("gorm:auto_preload", true)
		db = db.Set("gorm:association_autoupdate", true)
		db.BlockGlobalUpdate(true)
		db.SingularTable(true)
		db.LogMode(verbose)
		if err := db.AutoMigrate(
			Issue{},
			IssueLabel{},
			Profile{},
		).Error; err != nil {
			return err
		}

		return nil
	}
	cmd.AddCommand(
		newPullCommand(),
		newRunCommand(),
		newDBCommand(),
	)
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	return cmd
}
