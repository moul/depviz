package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

type dbOptions struct {
	Path string `mapstructure:"db-path"`
}

func dbSetupFlags(flags *pflag.FlagSet, opts *dbOptions) {
	flags.StringVarP(&opts.Path, "db-path", "", "./depviz.db", "depviz database path")
	viper.BindPFlags(flags)
}

func newDBCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "db",
	}
	cmd.AddCommand(newDBDumpCommand())
	return cmd
}

func newDBDumpCommand() *cobra.Command {
	opts := &dbOptions{}
	cmd := &cobra.Command{
		Use: "dump",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(opts); err != nil {
				return err
			}
			return dbDump(opts)
		},
	}
	dbSetupFlags(cmd.Flags(), opts)
	return cmd
}

func dbDump(opts *dbOptions) error {
	log.Printf("dbDump(%v)", *opts)
	issues, err := dbLoad(opts)
	if err != nil {
		return err
	}
	out, err := json.MarshalIndent(issues, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(out))
	return nil
}

func dbExists(opts *dbOptions) bool {
	log.Printf("dbExists(%v)", *opts)
	_, err := os.Stat(opts.Path)
	return err == nil
}

func dbLoad(opts *dbOptions) (Issues, error) {
	log.Printf("dbLoad(%v)", *opts)
	var issues []*Issue
	content, err := ioutil.ReadFile(opts.Path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open db file")
	}
	if err := json.Unmarshal(content, &issues); err != nil {
		return nil, errors.Wrap(err, "failed to parse db file")
	}
	m := make(Issues)
	for _, issue := range issues {
		m[issue.NodeName()] = issue
	}
	return m, nil
}
