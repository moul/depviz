package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type renderOptions struct {
	// fetch
	FetchOpts  fetchOptions
	ForceFetch bool

	// db
	DBOpts dbOptions

	// render
	RenderType  string
	ShowClosed  bool `mapstructure:"show-closed"`
	ShowOrphans bool
	EpicLabel   string
	Destination string
	//Preview     bool
}

func (opts renderOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func renderSetupFlags(flags *pflag.FlagSet, opts *renderOptions) {
	flags.BoolVarP(&opts.ForceFetch, "fetch", "f", false, "force fetch before rendering")
	flags.StringVarP(&opts.RenderType, "type", "t", "roadmap", "graph type ('roadmap', 'orphans')")
	flags.BoolVarP(&opts.ShowClosed, "show-closed", "", false, "show closed issues")
	flags.BoolVarP(&opts.ShowOrphans, "show-orphans", "", false, "show issues not linked to an epic")
	flags.StringVarP(&opts.EpicLabel, "epic-label", "", "", "label used for epics (empty means issues with dependencies but without dependants)")
	flags.StringVarP(&opts.Destination, "destination", "", "-", "destination ('-' for stdout)")
	//flags.BoolVarP(&opts.Preview, "preview", "p", false, "preview result")
	viper.BindPFlags(flags)
}

func newRenderCommand() *cobra.Command {
	opts := &renderOptions{}
	cmd := &cobra.Command{
		Use: "render",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(opts); err != nil {
				return err
			}
			if err := viper.Unmarshal(&opts.FetchOpts); err != nil {
				return err
			}
			if err := viper.Unmarshal(&opts.DBOpts); err != nil {
				return err
			}
			opts.FetchOpts.DBOpts = opts.DBOpts
			return render(opts)
		},
	}
	renderSetupFlags(cmd.Flags(), opts)
	fetchSetupFlags(cmd.Flags(), &opts.FetchOpts)
	dbSetupFlags(cmd.Flags(), &opts.DBOpts)
	return cmd
}

func render(opts *renderOptions) error {
	logger().Debug("render", zap.Stringer("opts", *opts))
	if opts.ForceFetch || !dbExists(&opts.DBOpts) {
		if err := fetch(&opts.FetchOpts); err != nil {
			return err
		}
	}

	issues, err := dbLoad(&opts.DBOpts)
	if err != nil {
		return errors.Wrap(err, "failed to load issues")
	}

	if err = issues.prepare(); err != nil {
		return errors.Wrap(err, "failed to prepare issues")
	}

	var out string
	switch opts.RenderType {
	case "roadmap":
		out, err = roadmapGraph(issues, opts)
	case "orphans":
		out, err = orphansGraph(issues, opts)
	default:
		err = fmt.Errorf("unknown graph type: %q", opts.RenderType)
	}
	if err != nil {
		return errors.Wrap(err, "failed to render graph")
	}

	var dest io.WriteCloser
	switch opts.Destination {
	case "-":
		dest = os.Stdout
	default:
		var err error
		dest, err = os.Create(opts.Destination)
		if err != nil {
			return err
		}
		defer dest.Close()
	}
	fmt.Fprintln(dest, out)

	return nil
}
