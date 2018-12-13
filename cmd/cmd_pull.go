package main

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/pkg/issues"
)

type pullOptions struct {
	// pull
	GithubToken string `mapstructure:"github-token"`
	GitlabToken string `mapstructure:"gitlab-token"`
	// includeExternalDeps bool

	Targets issues.Targets `mapstructure:"targets"`
}

func (opts pullOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

type pullCommand struct {
	opts pullOptions
}

func (cmd *pullCommand) LoadDefaultOptions() error {
	if err := viper.Unmarshal(&cmd.opts); err != nil {
		return err
	}
	return nil
}

func (cmd *pullCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&cmd.opts.GithubToken, "github-token", "", "", "GitHub Token with 'issues' access")
	flags.StringVarP(&cmd.opts.GitlabToken, "gitlab-token", "", "", "GitLab Token with 'issues' access")
	viper.BindPFlags(flags)
}

func (cmd *pullCommand) NewCobraCommand(dc map[string]DepvizCommand) *cobra.Command {
	cc := &cobra.Command{
		Use:   "pull",
		Short: "Pull issues and update database without outputting graph",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			var err error
			if opts.Targets, err = issues.ParseTargets(args); err != nil {
				return errors.Wrap(err, "invalid targets")
			}
			return pullAndCompute(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	return cc
}

func pullAndCompute(opts *pullOptions) error {
	zap.L().Debug("pull", zap.Stringer("opts", *opts))
	if os.Getenv("DEPVIZ_NOPULL") != "1" {
		if err := issues.PullAndCompute(opts.GithubToken, opts.GitlabToken, db, opts.Targets); err != nil {
			return errors.Wrap(err, "failed to pull")
		}
	} else {
		if err := issues.Compute(db); err != nil {
			return errors.Wrap(err, "failed to compute")
		}
	}
	return nil
}
