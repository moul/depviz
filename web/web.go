package web

import (
	"bytes"
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
	"moul.io/depviz/graph"
	"moul.io/depviz/model"
	"moul.io/depviz/sql"
)

type Options struct {
	SQL    sql.Options `mapstructure:"sql"` // inherited with sql.GetOptions()
	Bind   string      `mapstructure:"bind"`
	GenDoc bool        `mapstructure:"gendoc"`
	// Targets []multipmuri.Entity `mapstructure:"targets"` // parsed from Args
}

func Web(opts *Options) error {
	r := chi.NewRouter()

	//r.Use(middleware.RequestID)
	//r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	//r.Use(middleware.URLFormat)
	r.Use(middleware.Timeout(5 * time.Second))

	h := handler{opts: opts}

	r.Route("/api", func(r chi.Router) {
		r.Route("/", func(r chi.Router) {
			r.Use(render.SetContentType(render.ContentTypeJSON))
			r.Get("/issues.json", h.webListIssues)
		})
		r.Get("/graph/dot", h.webDotIssues)
		r.Get("/graph/image", h.webImageIssues)
	})

	workDir, _ := os.Getwd()
	filesDir := filepath.Join(workDir, "static")
	FileServer(r, "/", http.Dir(filesDir))

	if opts.GenDoc {
		fmt.Println(docgen.MarkdownRoutesDoc(r, docgen.MarkdownOpts{
			ProjectPath: "moul.io/depviz",
			Intro:       "Welcome to depviz generated docs.",
		}))
		return nil
	}

	log.Printf("Listening on %s", opts.Bind)
	return http.ListenAndServe(opts.Bind, r)
}

type handler struct {
	opts *Options
}

// webListIssues loads the issues stored in the database and writes them to the http response.
func (h *handler) webListIssues(w http.ResponseWriter, r *http.Request) {
	db, err := sql.FromOpts(&h.opts.SQL)
	if err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}

	issues, err := sql.LoadAllIssues(db)
	if err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}

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

func (h *handler) webGraphviz(r *http.Request) (string, error) {
	args := strings.Split(r.URL.Query().Get("targets"), ",")
	targets, err := model.ParseTargets(args)
	if err != nil {
		return "", err
	}
	opts := graph.Options{
		SQL:     h.opts.SQL,
		Targets: targets,
		// FIXME: add more options
	}
	return graph.Graph(&opts)
}

func (h *handler) webDotIssues(w http.ResponseWriter, r *http.Request) {
	out, err := h.webGraphviz(r)
	if err != nil {
		_ = render.Render(w, r, ErrRender(err))
		return
	}

	_, _ = w.Write([]byte(out))
}

func (h *handler) webImageIssues(w http.ResponseWriter, r *http.Request) {
	out, err := h.webGraphviz(r)
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
