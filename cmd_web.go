package main

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
)

type webOptions struct {
	// web specific
	Bind       string
	ShowRoutes bool

	// db
	DBOpts dbOptions
}

func (opts webOptions) String() string {
	out, _ := json.Marshal(opts)
	return string(out)
}

func webSetupFlags(flags *pflag.FlagSet, opts *webOptions) {
	flags.StringVarP(&opts.Bind, "bind", "b", ":2020", "web server bind address")
	flags.BoolVarP(&opts.ShowRoutes, "show-routes", "", false, "display available routes and quit")
	viper.BindPFlags(flags)
}

func newWebCommand() *cobra.Command {
	opts := &webOptions{}
	cmd := &cobra.Command{
		Use: "web",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := viper.Unmarshal(opts); err != nil {
				return err
			}
			if err := viper.Unmarshal(&opts.DBOpts); err != nil {
				return err
			}
			return web(opts)
		},
	}
	webSetupFlags(cmd.Flags(), opts)
	dbSetupFlags(cmd.Flags(), &opts.DBOpts)
	return cmd
}

func (i *Issue) Render(w http.ResponseWriter, r *http.Request) error {
	return nil
}

func webListIssues(w http.ResponseWriter, r *http.Request) {
	issues, err := loadIssues(db, nil)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	list := []render.Renderer{}
	for _, issue := range issues {
		list = append(list, issue)
	}

	if err := render.RenderList(w, r, list); err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}
}

func webGraphviz(r *http.Request) (string, error) {
	opts := &graphOptions{
		Targets:    strings.Split(r.URL.Query().Get("targets"), ","),
		ShowClosed: r.URL.Query().Get("show-closed") == "1",
	}
	return graphviz(opts)
}

func webDotIssues(w http.ResponseWriter, r *http.Request) {
	out, err := webGraphviz(r)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	w.Write([]byte(out))
}

func webImageIssues(w http.ResponseWriter, r *http.Request) {
	out, err := webGraphviz(r)
	if err != nil {
		render.Render(w, r, ErrRender(err))
		return
	}

	cmd := exec.Command("dot", "-Tsvg")
	cmd.Stdin = bytes.NewBuffer([]byte(out))
	cmd.Stdout = w

	if err := cmd.Run(); err != nil {
		render.Render(w, r, ErrRender(err))
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
