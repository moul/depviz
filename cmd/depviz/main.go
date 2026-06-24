package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"moul.io/depviz/v4/internal/backend"
	"moul.io/depviz/v4/internal/core"
	"moul.io/depviz/v4/live"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "depviz: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		usage()
		return nil
	}
	dbPath := os.Getenv("DEPVIZ_DB")
	if dbPath == "" {
		dbPath = core.DefaultDBPath
	}
	cmd := args[0]
	args = args[1:]
	switch cmd {
	case "help", "-h", "--help":
		usage()
		return nil
	case "init":
		s, err := core.OpenStore(ctx, dbPath)
		if err != nil {
			return err
		}
		defer s.Close()
		fmt.Printf("initialized %s\n", s.Path())
		return nil
	case "ingest":
		return runIngest(ctx, dbPath, args)
	case "board":
		return runBoard(ctx, dbPath, args)
	case "edge":
		return runEdge(ctx, dbPath, args)
	case "query":
		return runQuery(ctx, dbPath, args)
	case "brief":
		return runBrief(ctx, dbPath, args)
	case "gen":
		return runGen(ctx, dbPath, args)
	case "sync":
		return runSync(ctx, dbPath, args)
	case "live":
		return runLive(ctx, args)
	case "backup":
		return runBackup(ctx, dbPath, args)
	case "server":
		return runServer(ctx, dbPath, args)
	default:
		return fmt.Errorf("unknown command %q", cmd)
	}
}

func runIngest(ctx context.Context, dbPath string, args []string) error {
	if len(args) < 2 || args[0] != "events" {
		return errors.New("usage: depviz ingest events <path> [--board default]")
	}
	fs := flag.NewFlagSet("ingest events", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	board := fs.String("board", core.DefaultBoardID, "board id")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	f, err := os.Open(args[1])
	if err != nil {
		return err
	}
	defer f.Close()
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	count, err := s.IngestEvents(ctx, f, *board)
	if err != nil {
		return err
	}
	fmt.Printf("ingested %d events into board %s\n", count, *board)
	return nil
}

func runBoard(ctx context.Context, dbPath string, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: depviz board list | depviz board note <board> <text>")
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	switch args[0] {
	case "list":
		boards, err := s.BoardList(ctx)
		if err != nil {
			return err
		}
		for _, b := range boards {
			fmt.Printf("%s\t%s\n", b.ID, b.Name)
		}
		return nil
	case "note":
		if len(args) < 3 {
			return errors.New("usage: depviz board note <board> <text>")
		}
		n, err := s.CreateNote(ctx, args[1], strings.Join(args[2:], " "))
		if err != nil {
			return err
		}
		fmt.Printf("created %s %s\n", n.ID, n.Title)
		return nil
	default:
		return fmt.Errorf("unknown board command %q", args[0])
	}
}

func runEdge(ctx context.Context, dbPath string, args []string) error {
	if len(args) == 0 || args[0] != "add" {
		return errors.New("usage: depviz edge add <from> <to> --kind blocked_by [--board default]")
	}
	if len(args) < 3 {
		return errors.New("usage: depviz edge add <from> <to> --kind blocked_by [--board default]")
	}
	fs := flag.NewFlagSet("edge add", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	kind := fs.String("kind", "blocked_by", "edge kind")
	board := fs.String("board", core.DefaultBoardID, "board id")
	if err := fs.Parse(args[3:]); err != nil {
		return err
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	e, err := s.AddEdge(ctx, *board, args[1], args[2], *kind, "local", map[string]string{"created_by": "depviz edge add"})
	if err != nil {
		return err
	}
	fmt.Printf("created %s %s -> %s (%s)\n", e.ID, e.FromID, e.ToID, e.Kind)
	return nil
}

func runQuery(ctx context.Context, dbPath string, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: depviz query ready|blockers [--board default]")
	}
	fs := flag.NewFlagSet("query", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	board := fs.String("board", core.DefaultBoardID, "board id")
	if err := fs.Parse(args[1:]); err != nil {
		return err
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	brief, err := s.BuildBrief(ctx, *board)
	if err != nil {
		return err
	}
	switch args[0] {
	case "ready":
		for _, item := range brief.Ready {
			fmt.Printf("%s\t%s\t%s\n", item.ID, item.State, item.Title)
		}
	case "blockers":
		for _, item := range brief.Blockers {
			fmt.Printf("%s\t%d\t%s\n", item.ID, item.Impact, item.Title)
		}
	default:
		return fmt.Errorf("unknown query %q", args[0])
	}
	return nil
}

func runBrief(ctx context.Context, dbPath string, args []string) error {
	fs := flag.NewFlagSet("brief", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	board := fs.String("board", core.DefaultBoardID, "board id")
	if err := fs.Parse(args); err != nil {
		return err
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	brief, err := s.BuildBrief(ctx, *board)
	if err != nil {
		return err
	}
	return core.RenderBrief(os.Stdout, brief)
}

func runGen(ctx context.Context, dbPath string, args []string) error {
	if len(args) == 0 {
		return errors.New("usage: depviz gen html|json --board default --out dist/depviz.html")
	}
	switch args[0] {
	case "html":
		return runGenHTML(ctx, dbPath, args[1:])
	case "json":
		return runGenJSON(ctx, dbPath, args[1:])
	default:
		return fmt.Errorf("unknown gen target %q", args[0])
	}
}

func runGenHTML(ctx context.Context, dbPath string, args []string) error {
	fs := flag.NewFlagSet("gen html", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	board := fs.String("board", core.DefaultBoardID, "board id")
	view := fs.String("view", "graph", "initial view")
	out := fs.String("out", "dist/depviz.html", "output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *view != "graph" && *view != "table" {
		return fmt.Errorf("unsupported view %q", *view)
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := s.RenderHTML(ctx, *board, f); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", *out)
	return nil
}

func runGenJSON(ctx context.Context, dbPath string, args []string) error {
	fs := flag.NewFlagSet("gen json", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	board := fs.String("board", core.DefaultBoardID, "board id")
	out := fs.String("out", "dist/depviz.json", "output file")
	if err := fs.Parse(args); err != nil {
		return err
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := os.MkdirAll(filepath.Dir(*out), 0o755); err != nil {
		return err
	}
	f, err := os.Create(*out)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := s.RenderJSON(ctx, *board, f); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", *out)
	return nil
}

func runSync(ctx context.Context, dbPath string, args []string) error {
	if len(args) < 2 || args[0] != "github" {
		return errors.New("usage: depviz sync github owner/repo [--limit 200]")
	}
	fs := flag.NewFlagSet("sync github", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	limit := fs.Int("limit", 200, "max issues and PRs to import")
	if err := fs.Parse(args[2:]); err != nil {
		return err
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	count, err := core.SyncGitHub(ctx, s, core.GitHubSyncOptions{Repo: args[1], Limit: *limit})
	if err != nil {
		return err
	}
	fmt.Printf("synced %d GitHub cards from %s\n", count, args[1])
	return nil
}

func runLive(ctx context.Context, args []string) error {
	_ = ctx
	fs := flag.NewFlagSet("live", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	addr := fs.String("addr", "127.0.0.1:8686", "listen address")
	if err := fs.Parse(args); err != nil {
		return err
	}
	fmt.Printf("serving DepViz Live at http://%s\n", *addr)
	return http.ListenAndServe(*addr, http.FileServer(http.FS(live.AppFS())))
}

func runBackup(ctx context.Context, dbPath string, args []string) error {
	fs := flag.NewFlagSet("backup", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	outDir := fs.String("out", "backups", "output directory for backup files")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if _, err := os.Stat(dbPath); err != nil {
		return fmt.Errorf("database not found at %s", dbPath)
	}
	ts := time.Now().UTC().Format("20060102T150405Z")
	outFile := filepath.Join(*outDir, "state-"+ts+"-manual.db")
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		return err
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := s.Backup(ctx, outFile); err != nil {
		return err
	}
	fmt.Printf("backup written to: %s\n", outFile)
	return nil
}

func runServer(ctx context.Context, dbPath string, args []string) error {
	fs := flag.NewFlagSet("server", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)
	addr := fs.String("addr", envDefault("DEPVIZ_ADDR", "127.0.0.1:8766"), "listen address")
	baseURL := fs.String("base-url", os.Getenv("DEPVIZ_BASE_URL"), "public base URL")
	if err := fs.Parse(args); err != nil {
		return err
	}
	s, err := core.OpenStore(ctx, dbPath)
	if err != nil {
		return err
	}
	defer s.Close()
	cfg := backend.Config{
		Addr:                    *addr,
		BaseURL:                 *baseURL,
		GitHubClientID:          os.Getenv("DEPVIZ_GITHUB_CLIENT_ID"),
		GitHubClientSecret:      os.Getenv("DEPVIZ_GITHUB_CLIENT_SECRET"),
		GitHubAppID:             os.Getenv("DEPVIZ_GITHUB_APP_ID"),
		GitHubAppPrivateKeyFile: os.Getenv("DEPVIZ_GITHUB_PRIVATE_KEY_FILE"),
		GitHubWebhookSecret:     os.Getenv("DEPVIZ_GITHUB_WEBHOOK_SECRET"),
		SessionTTL:              30 * 24 * time.Hour,
	}
	srv := backend.NewServer(s, cfg)
	fmt.Printf("serving DepViz backend at http://%s\n", *addr)
	return http.ListenAndServe(*addr, srv.Handler())
}

func envDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func usage() {
	fmt.Println(`depviz - local-first work graph

Usage:
  depviz init
  depviz ingest events <path>
  depviz sync github owner/repo [--limit 200]
  depviz board list
  depviz board note <board> <text>
  depviz edge add <from> <to> --kind blocked_by
  depviz query ready|blockers
  depviz brief
  depviz gen html --board default --view graph --out dist/depviz.html
  depviz gen json --board default --out dist/depviz.json
  depviz live --addr 127.0.0.1:8686
  depviz backup [--out backups]
  depviz server --addr 127.0.0.1:8766 --base-url https://depviz.moul.io

Environment:
  DEPVIZ_DB                    override .depviz/state.db
  DEPVIZ_ADDR                  default server listen address
  DEPVIZ_BASE_URL              public server URL for OAuth callbacks
  DEPVIZ_GITHUB_CLIENT_ID      GitHub OAuth app client id
  DEPVIZ_GITHUB_CLIENT_SECRET  GitHub OAuth app client secret
  DEPVIZ_GITHUB_APP_ID         GitHub App id
  DEPVIZ_GITHUB_PRIVATE_KEY_FILE
                               GitHub App private key PEM path
  DEPVIZ_GITHUB_WEBHOOK_SECRET GitHub App webhook secret`)
}
