package main // import "moul.io/depviz/cmd/depviz"

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"os/signal"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/graph"
	_ "github.com/cayleygraph/cayley/graph/kv/bolt"
	"github.com/cayleygraph/cayley/schema"
	"github.com/oklog/run"
	"github.com/peterbourgon/ff/v3/ffcli"
	"go.uber.org/zap"
	"moul.io/banner"
	"moul.io/depviz/v3/pkg/dvcore"
	"moul.io/depviz/v3/pkg/dvparser"
	"moul.io/depviz/v3/pkg/dvserver"
	"moul.io/depviz/v3/pkg/dvstore"
	"moul.io/srand"
	"moul.io/u"
	"moul.io/zapconfig"
)

var (
	logger       *zap.Logger
	schemaConfig *schema.Config

	globalFlags          = flag.NewFlagSet("depviz", flag.ExitOnError)
	globalStorePath      = globalFlags.String("store-path", os.Getenv("HOME")+"/.depviz", "store path")
	globalDebug          = globalFlags.Bool("debug", false, "debug mode")
	globalWithStacktrace = globalFlags.Bool("with-stacktrace", false, "show stacktrace on warns, errors and worse")

	airtableFlags     = flag.NewFlagSet("airtable", flag.ExitOnError)
	airtableToken     = airtableFlags.String("token", "", "airtable token")
	airtableBaseID    = airtableFlags.String("base-id", "", "base ID")
	airtableOwnersTab = airtableFlags.String("owners", "Owners", `"Owners" tab name`)
	airtableTasksTab  = airtableFlags.String("tasks", "Tasks", `"Tasks" tab name`)
	airtableTopicsTab = airtableFlags.String("topics", "Topics", `"Topics" tab name`)

	serverFlags              = flag.NewFlagSet("server", flag.ExitOnError)
	serverHTTPBind           = serverFlags.String("http-bind", ":8000", "HTTP bind address")
	serverGRPCBInd           = serverFlags.String("grpc-bind", ":9000", "gRPC bind address")
	serverRequestTimeout     = serverFlags.Duration("request-timeout", 5*time.Second, "request timeout")   // nolint:gomnd
	serverShutdownTimeout    = serverFlags.Duration("shutdowm-timeout", 6*time.Second, "shutdown timeout") // nolint:gomnd
	serverCORSAllowedOrigins = serverFlags.String("cors-allowed-origins", "*", "allowed CORS origins")
	serverGitHubToken        = serverFlags.String("github-token", "", "GitHub token")
	serverNoAutoUpdate       = serverFlags.Bool("no-auto-update", false, "don't auto-update projects in background")
	serverGodmode            = serverFlags.Bool("godmode", false, "enable dangerous API calls")
	serverWithPprof          = serverFlags.Bool("with-pprof", false, "enable pprof endpoints")
	serverWithoutRecovery    = serverFlags.Bool("without-recovery", false, "disable panic recovery (dev)")
	serverWithoutCache       = serverFlags.Bool("without-cache", false, "disable HTTP caching")
	serverAuth               = serverFlags.String("auth", "", "authentication password")
	serverRealm              = serverFlags.String("realm", "DepViz", "server Realm")
	serverAutoUpdateInterval = serverFlags.Duration("auto-update-interval", 2*time.Minute, "time between two auto-updates") // nolint:gomnd
	serverGitHubClientID     = serverFlags.String("github-client-id", "", "GitHub client ID")
	serverGitHubClientSecret = serverFlags.String("github-client-secret", "", "GitHub client secret")

	genFlags            = flag.NewFlagSet("gen", flag.ExitOnError)
	genNoGraph          = genFlags.Bool("no-graph", false, "don't generate graph (pull only)")
	genNoPert           = genFlags.Bool("no-pert", false, "disable PERT computing")
	genVertical         = genFlags.Bool("vertical", false, "vertical mode")
	genHidePRs          = genFlags.Bool("hide-prs", false, "hide PRs")
	genHideExternalDeps = genFlags.Bool("hide-external-deps", false, "hide dependencies outside of the specified targets")
	genHideIsolated     = genFlags.Bool("hide-isolated", false, "hide isolated tasks")
	genShowClosed       = genFlags.Bool("show-closed", false, "show closed tasks")

	fetchFlags       = flag.NewFlagSet("fetch", flag.ExitOnError)
	fetchGitHubToken = fetchFlags.String("github-token", "", "GitHub token")
	fetchResync      = fetchFlags.Bool("resync", false, "resync already synced content")
)

func main() {
	err := Main(os.Args)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatalf("fatal: %+v", err)
	}
}

func Main(args []string) error {
	log.SetFlags(0)

	defer func() {
		if logger != nil {
			_ = logger.Sync()
		}
	}()

	root := &ffcli.Command{
		ShortUsage: "depviz [global flags] <subcommand> [flags] [args...]",
		FlagSet:    globalFlags,
		LongHelp:   "More info here: https://moul.io/depviz",
		Subcommands: []*ffcli.Command{
			{
				Name:       "airtable",
				ShortHelp:  "manage airtable sync",
				ShortUsage: "airtable [flags] <subcommand>",
				FlagSet:    airtableFlags,
				Subcommands: []*ffcli.Command{
					{Name: "info", Exec: execAirtableInfo, ShortHelp: "get metrics"},
					{Name: "sync", Exec: execAirtableSync, ShortHelp: "sync store with Airtable"},
				},
				Exec: func(context.Context, []string) error { return flag.ErrHelp },
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
				Exec: func(context.Context, []string) error { return flag.ErrHelp },
			}, {
				Name:      "server",
				ShortHelp: "start a depviz server with depviz API",
				FlagSet:   serverFlags,
				Exec:      execServer,
			}, {
				Name:      "gen",
				ShortHelp: "use the db to generate outputs, without requiring any fetch",
				Subcommands: []*ffcli.Command{
					{Name: "graphviz", Exec: execGenGraphviz, ShortHelp: "generate graphviz output"},
					{Name: "json", Exec: execGenJSON, ShortHelp: "generate JSON output"},
					{Name: "csv", Exec: execGenCSV, ShortHelp: "generate CSV output"},
				},
				FlagSet: genFlags,
			}, {
				Name:      "fetch",
				ShortHelp: "fetch data from providers",
				Exec:      execFetch,
				FlagSet:   fetchFlags,
			},
		},
		Exec: func(context.Context, []string) error { return flag.ErrHelp },
	}

	return root.ParseAndRun(context.Background(), args[1:])
}

func globalPreRun() error {
	rand.Seed(srand.MustSecure())

	config := zapconfig.Configurator{}
	if *globalDebug {
		config.SetLevel(zap.DebugLevel)
	} else {
		config.SetLevel(zap.InfoLevel)
	}
	if *globalWithStacktrace {
		config.EnableStacktrace()
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

func execAirtableSync(ctx context.Context, args []string) error {
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

func execAirtableInfo(ctx context.Context, args []string) error {
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

func execStoreDumpQuads(ctx context.Context, args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	return dvcore.StoreDumpQuads(store)
}

func execStoreDumpJSON(ctx context.Context, args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	batch, err := dvcore.GetStoreDump(ctx, store, schemaConfig)
	if err != nil {
		return fmt.Errorf("get store dump: %w", err)
	}

	fmt.Println(u.PrettyJSON(batch))
	return nil
}

func execStoreInfo(ctx context.Context, args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	return dvcore.StoreInfo(store)
}

func execServer(ctx context.Context, args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	var (
		g   run.Group
		svc dvserver.Service
	)

	{ // server
		store, err := storeFromArgs()
		if err != nil {
			return fmt.Errorf("init store: %w", err)
		}

		targets, err := dvparser.ParseTargets(args)
		if err != nil {
			return fmt.Errorf("parse targets: %w", err)
		}

		opts := dvserver.Opts{
			Logger:             logger,
			HTTPBind:           *serverHTTPBind,
			GRPCBind:           *serverGRPCBInd,
			CORSAllowedOrigins: *serverCORSAllowedOrigins,
			RequestTimeout:     *serverRequestTimeout,
			ShutdownTimeout:    *serverShutdownTimeout,
			WithPprof:          *serverWithPprof,
			WithoutRecovery:    *serverWithoutRecovery,
			WithoutCache:       *serverWithoutCache,
			Auth:               *serverAuth,
			Realm:              *serverRealm,
			Godmode:            *serverGodmode,
			GitHubToken:        *serverGitHubToken,
			NoAutoUpdate:       *serverNoAutoUpdate,
			AutoUpdateTargets:  targets,
			AutoUpdateInterval: *serverAutoUpdateInterval,
			GitHubClientID:     *serverGitHubClientID,
			GitHubClientSecret: *serverGitHubClientSecret,
		}
		svc, err = dvserver.New(ctx, store, schemaConfig, opts)
		if err != nil {
			return fmt.Errorf("init server: %w", err)
		}

		g.Add(
			svc.Run,
			func(error) { svc.Close() },
		)
	}

	{ // signal handling
		ctx, cancel := context.WithCancel(ctx)
		g.Add(func() error {
			sigch := make(chan os.Signal, 1)
			signal.Notify(sigch, os.Interrupt)
			select {
			case <-sigch:
			case <-ctx.Done():
			}
			return nil
		}, func(error) {
			cancel()
		})
	}

	fmt.Fprintln(os.Stderr, banner.Inline("depviz"))
	logger.Info("server started",
		zap.String("http-bind", svc.HTTPListenerAddr()),
		zap.String("grpc-bind", svc.GRPCListenerAddr()),
	)

	if err := g.Run(); err != nil {
		return fmt.Errorf("group terminated: %w", err)
	}
	return nil
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

func execGenGraphviz(ctx context.Context, args []string) error {
	return fmt.Errorf("not implemented yet")
}

func execGenJSON(ctx context.Context, args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	opts := dvcore.GenOpts{
		Logger:           logger,
		Schema:           schemaConfig,
		Vertical:         *genVertical,
		NoPert:           *genNoPert,
		NoGraph:          *genNoGraph,
		ShowClosed:       *genShowClosed,
		HideIsolated:     *genHideIsolated,
		HidePRs:          *genHidePRs,
		HideExternalDeps: *genHideExternalDeps,
		Format:           "json",
	}

	return dvcore.Gen(store, args, opts)
}

func execGenCSV(ctx context.Context, args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	opts := dvcore.GenOpts{
		Logger:           logger,
		Schema:           schemaConfig,
		Vertical:         *genVertical,
		NoPert:           *genNoPert,
		NoGraph:          *genNoGraph,
		ShowClosed:       *genShowClosed,
		HideIsolated:     *genHideIsolated,
		HidePRs:          *genHidePRs,
		HideExternalDeps: *genHideExternalDeps,
		Format:           "csv",
	}

	return dvcore.Gen(store, args, opts)
}

func execFetch(ctx context.Context, args []string) error {
	if err := globalPreRun(); err != nil {
		return err
	}

	store, err := storeFromArgs()
	if err != nil {
		return fmt.Errorf("init store: %w", err)
	}

	opts := dvcore.FetchOpts{
		Logger:      logger,
		Schema:      schemaConfig,
		GitHubToken: *fetchGitHubToken,
		Resync:      *fetchResync,
	}
	return dvcore.Fetch(store, args, opts)
}
