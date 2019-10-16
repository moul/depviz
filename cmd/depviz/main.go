package main // import "moul.io/depviz"

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/kv/bolt"
	"github.com/cayleygraph/cayley/schema"
	"github.com/peterbourgon/ff"
	"github.com/peterbourgon/ff/ffcli"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"moul.io/depviz/internal/dvcore"
	"moul.io/depviz/internal/dvstore"
)

var (
	logger       *zap.Logger
	schemaConfig *schema.Config

	globalFlags     = flag.NewFlagSet("depviz", flag.ExitOnError)
	globalStorePath = globalFlags.String("store-path", os.Getenv("HOME")+"/.depviz", "store path")
	globalDebug     = globalFlags.Bool("debug", false, "debug mode")

	airtableFlags     = flag.NewFlagSet("airtable", flag.ExitOnError)
	airtableToken     = airtableFlags.String("token", "", "airtable token")
	airtableBaseID    = airtableFlags.String("base-id", "", "base ID")
	airtableOwnersTab = airtableFlags.String("owners", "Owners", `"Owners" tab name`)
	airtableTasksTab  = airtableFlags.String("tasks", "Tasks", `"Tasks" tab name`)
	airtableTopicsTab = airtableFlags.String("topics", "Topics", `"Topics" tab name`)

	serverFlags   = flag.NewFlagSet("server", flag.ExitOnError)
	serverBind    = serverFlags.String("bind", ":8000", "server bind address")
	serverGodmode = serverFlags.Bool("godmode", false, "enable dangerous API calls")

	runFlags            = flag.NewFlagSet("run", flag.ExitOnError)
	runNoPull           = runFlags.Bool("no-pull", false, "don't pull providers (graph only)")
	runNoGraph          = runFlags.Bool("no-graph", false, "don't generate graph (pull only)")
	runResync           = runFlags.Bool("resync", false, "resync already synced content")
	runGitHubToken      = runFlags.String("github-token", "", "GitHub token")
	runGitLabToken      = runFlags.String("gitlab-token", "", "GitLab token")
	runNoPert           = runFlags.Bool("no-pert", false, "disable PERT computing")
	runFormat           = runFlags.String("format", "dot", "output format")
	runVertical         = runFlags.Bool("vertical", false, "vertical mode")
	runHidePRs          = runFlags.Bool("hide-prs", false, "hide PRs")
	runHideExternalDeps = runFlags.Bool("hide-external-deps", false, "hide dependencies outside of the specified targets")
	runHideIsolated     = runFlags.Bool("hide-isolated", false, "hide isolated tasks")
	runShowClosed       = runFlags.Bool("show-closed", false, "show closed tasks")
)

func main() {
	log.SetFlags(0)

	defer func() {
		if logger != nil {
			_ = logger.Sync()
		}
	}()

	root := &ffcli.Command{
		Usage:    "depviz [global flags] <subcommand> [flags] [args...]",
		FlagSet:  globalFlags,
		Options:  []ff.Option{ff.WithEnvVarNoPrefix()},
		LongHelp: "More info here: https://moul.io/depviz",
		Subcommands: []*ffcli.Command{
			{
				Name:      "airtable",
				ShortHelp: "manage airtable sync",
				Usage:     "airtable [flags] <subcommand>",
				FlagSet:   airtableFlags,
				Options:   []ff.Option{ff.WithEnvVarNoPrefix()},
				Subcommands: []*ffcli.Command{
					{Name: "info", Exec: execAirtableInfo, ShortHelp: "get metrics"},
					{Name: "sync", Exec: execAirtableSync, ShortHelp: "sync store with Airtable"},
				},
				Exec: func([]string) error { return flag.ErrHelp },
			}, {
				Name:      "store",
				ShortHelp: "manage the data store",
				Subcommands: []*ffcli.Command{
					{Name: "dump-quads", Exec: execStoreDumpQuads},
					{Name: "dump-json", Exec: execStoreDumpJSON},
					{Name: "info", Exec: execStoreInfo},
					// restore-quads
					// restore-json
				},
				Exec: func([]string) error { return flag.ErrHelp },
			}, {
				Name:      "run",
				ShortHelp: "sync target urls and draw a graph",
				Usage:     "run [flags] [url...]",
				Exec:      execRun,
				FlagSet:   runFlags,
				Options:   []ff.Option{ff.WithEnvVarNoPrefix()},
			}, {
				Name:      "server",
				ShortHelp: "start a depviz server with depviz API",
				FlagSet:   serverFlags,
				Options:   []ff.Option{ff.WithEnvVarNoPrefix()},
				Exec:      execServer,
			},
		},
		Exec: func([]string) error { return flag.ErrHelp },
	}

	if err := root.Run(os.Args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatalf("fatal: %+v", err)
	}
}

func globalPreRun() error {
	rand.Seed(time.Now().UnixNano())

	config := zap.NewDevelopmentConfig()
	config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	if *globalDebug {
		config.Level.SetLevel(zap.DebugLevel)
		config.DisableStacktrace = false
	} else {
		config.Level.SetLevel(zap.InfoLevel)
		config.DisableStacktrace = true
	}
	var err error
	logger, err = config.Build()
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	logger.Debug("logger initialized")

	schemaConfig = dvstore.Schema()
	return nil
}

func execAirtableSync(args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	opts := dvcore.AirtableOpts{
		Token:     *airtableToken,
		BaseID:    *airtableBaseID,
		OwnersTab: *airtableOwnersTab,
		TasksTab:  *airtableTasksTab,
		TopicsTab: *airtableTopicsTab,
	}

	return dvcore.AirtableSync(store, opts)
}

func execAirtableInfo(args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	opts := dvcore.AirtableOpts{
		Token:     *airtableToken,
		BaseID:    *airtableBaseID,
		OwnersTab: *airtableOwnersTab,
		TasksTab:  *airtableTasksTab,
		TopicsTab: *airtableTopicsTab,
	}

	return dvcore.AirtableInfo(opts)
}

func execStoreDumpQuads(args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	return dvcore.StoreDumpQuads(store)
}

func execStoreDumpJSON(args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	return dvcore.StoreDumpJSON(store, schemaConfig)
}

func execStoreInfo(args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	return dvcore.StoreInfo(store)
}

func execRun(args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	opts := dvcore.RunOpts{
		Logger:           logger,
		Schema:           schemaConfig,
		Vertical:         *runVertical,
		NoPert:           *runNoPert,
		NoGraph:          *runNoGraph,
		NoPull:           *runNoPull,
		Format:           *runFormat,
		GitHubToken:      *runGitHubToken,
		Resync:           *runResync,
		GitLabToken:      *runGitLabToken,
		ShowClosed:       *runShowClosed,
		HideIsolated:     *runHideIsolated,
		HidePRs:          *runHidePRs,
		HideExternalDeps: *runHideExternalDeps,
	}
	return dvcore.Run(store, args, opts)
}

func execServer(args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	return dvcore.Server(*serverBind, *serverGodmode, store, logger, schemaConfig)
}

func storeFromArgs() (*cayley.Handle, error) {
	if _, err := os.Stat(*globalStorePath); err != nil {
		if err := graph.InitQuadStore("bolt", *globalStorePath, nil); err != nil {
			return nil, fmt.Errorf("create quad store: %w", err)
		}
	}
	store, err := cayley.NewGraph("bolt", *globalStorePath, nil)
	if err != nil {
		return nil, fmt.Errorf("load STORE: %w", err)
	}

	return store, nil
}
