package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/docgen"
	"github.com/go-chi/render"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"moul.io/depviz/warehouse"
)

type webOptions struct {
	// web specific
	Bind       string
	ShowRoutes bool
}

func (opts webOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

type webCommand struct {
	opts webOptions
}

func (cmd *webCommand) LoadDefaultOptions() error {
	if err := viper.Unmarshal(&cmd.opts); err != nil {
		return err
	}
	return nil
}

func (cmd *webCommand) ParseFlags(flags *pflag.FlagSet) {
	flags.StringVarP(&cmd.opts.Bind, "bind", "b", ":2020", "web server bind address")
	flags.BoolVarP(&cmd.opts.ShowRoutes, "show-routes", "", false, "display available routes and quit")
	if err := viper.BindPFlags(flags); err != nil {
		zap.L().Warn("failed to bind flags using Viper", zap.Error(err))
	}
}

func (cmd *webCommand) NewCobraCommand(dc map[string]DepvizCommand) *cobra.Command {
	cc := &cobra.Command{
		Use:   "web",
		Short: "Run depviz as a web server",
		RunE: func(_ *cobra.Command, args []string) error {
			opts := cmd.opts
			return web(&opts)
		},
	}
	cmd.ParseFlags(cc.Flags())
	return cc
}

// webListIssues loads the issues stored in the database and writes them to the http response.
func webListIssues(w http.ResponseWriter, r *http.Request) {
	issues, err := warehouse.Load(db, nil)
	if err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}

	list := []render.Renderer{}
	for _, issue := range issues {
		if issue.IsHidden {
			continue
		}
		list = append(list, issue)
	}

	if err := render.RenderList(w, r, list); err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}
}

func webGraphviz(r *http.Request) (string, error) {
	targets, err := warehouse.ParseTargets(strings.Split(r.URL.Query().Get("targets"), ","))
	if err != nil {
		return "", err
	}
	opts := &graphOptions{
		Targets:    targets,
		ShowClosed: r.URL.Query().Get("show-closed") == "1",
	}
	issues, err := warehouse.Load(db, nil)
	if err != nil {
		return "", err
	}
	filtered := issues.FilterByTargets(targets)
	return graphviz(filtered, opts)
}

func webDotIssues(w http.ResponseWriter, r *http.Request) {
	out, err := webGraphviz(r)
	if err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}

	_, _ = w.Write([]byte(out))
}

func webImageIssues(w http.ResponseWriter, r *http.Request) {
	out, err := webGraphviz(r)
	if err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}

	binary, err := exec.LookPath("dot")
	if err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}

	cmd := exec.Command(binary, "-Tsvg") // guardrails-disable-line
	cmd.Stdin = bytes.NewBuffer([]byte(out))
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}
}

func web(opts *webOptions) error {
	r := chi.NewRouter()

	//r.Use(middleware.RequestID)
	//r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	//r.Use(middleware.URLFormat)
	r.Use(middleware.Timeout(5 * time.Second))

	r.Route("/api", func(r chi.Router) {
		r.Route("/", func(r chi.Router) {
			r.Use(render.SetContentType(render.ContentTypeJSON))
			r.Get("/issues.json", webListIssues)
		})
		r.Get("/graph/dot", webDotIssues)
		r.Get("/graph/image", webImageIssues)
	})

	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "web")
	FileServer(r, "/", http.Dir(filesDir))

	if opts.ShowRoutes {
		fmt.Println(docgen.MarkdownRoutesDoc(r, docgen.MarkdownOpts{
			ProjectPath: "moul.io/depviz",
			Intro:       "Welcome to depviz generated docs.",
		}))
		return nil
	}

	log.Printf("Listening on %s", opts.Bind)
	return http.ListenAndServe(opts.Bind, r)
}
