package dvcore

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/schema"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"github.com/gobuffalo/packr/v2"
	gschema "github.com/gorilla/schema"
	"go.uber.org/zap"
	"moul.io/depviz/internal/chiutil"
	"moul.io/depviz/internal/dvparser"
	"moul.io/depviz/internal/dvstore"
)

func Server(bind string, godmode bool, h *cayley.Handle, logger *zap.Logger, schema *schema.Config) error {
	logger.Debug("Server called", zap.String("bind", bind), zap.Bool("godmode", godmode))

	r := chi.NewRouter()

	//r.Use(middleware.RequestID)
	//r.Use(middleware.RealIP)
	//r.Use(middleware.URLFormat)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(5 * time.Second))
	// FIXME: add caching

	a := api{
		logger: logger,
		h:      h,
		schema: schema,
	}

	r.Route("/api", func(r chi.Router) {
		r.Route("/", func(r chi.Router) {
			r.Use(render.SetContentType(render.ContentTypeJSON))
			if godmode {
				r.Get("/store/dump.json", a.storeDumpJSON)
			}
			r.Get("/graph.json", a.graphJSON)
		})
	})

	box := packr.New("static", "./static")
	chiutil.FileServer(r, "/", box)

	{ // print listeners and routes
		logger.Info("HTTP API listening", zap.String("bind", bind))
		walkFunc := func(method string, route string, handler http.Handler, middlewares ...func(http.Handler) http.Handler) error {
			logger.Debug(fmt.Sprintf("  %s %s", method, route))
			return nil
		}
		if err := chi.Walk(r, walkFunc); err != nil {
			logger.Warn("chi walk", zap.Error(err))
		}
	}

	return http.ListenAndServe(bind, r)
}

type api struct {
	logger *zap.Logger
	h      *cayley.Handle
	schema *schema.Config
}

func (api *api) storeDumpJSON(w http.ResponseWriter, r *http.Request) {
	dump, err := getStoreDump(api.h, api.schema)
	if err != nil {
		_ = render.Render(w, r, chiutil.ErrRender(err))
		return
	}

	if err := render.Render(w, r, dump); err != nil {
		_ = render.Render(w, r, chiutil.ErrRender(err))
		return
	}
}

func (api *api) graphJSON(w http.ResponseWriter, r *http.Request) {
	// parsing
	var decoder = gschema.NewDecoder()
	type Opts struct {
		Targets             string `schema:"targets"`
		WithClosed          bool   `schema:"with-closed"`
		WithoutIsolated     bool   `schema:"without-isolated"`
		WithoutPRs          bool   `schema:"without-prs"`
		WithoutExternalDeps bool   `schema:"without-external-deps"`
	}
	opts := Opts{}

	err := decoder.Decode(&opts, r.URL.Query())
	if err != nil {
		_ = render.Render(w, r, chiutil.ErrRender(err))
		return
	}

	// validation
	if opts.Targets == "" {
		_ = render.Render(w, r, chiutil.ErrRender(fmt.Errorf("missing ?targets=")))
		return
	}
	filters := dvstore.LoadTasksFilters{
		WithClosed:          opts.WithClosed,
		WithoutIsolated:     opts.WithoutIsolated,
		WithoutPRs:          opts.WithoutPRs,
		WithoutExternalDeps: opts.WithoutExternalDeps,
	}
	targets, err := dvparser.ParseTargets(strings.Split(opts.Targets, ","))
	if err != nil {
		_ = render.Render(w, r, chiutil.ErrRender(err))
		return
	}
	filters.Targets = targets

	// load tasks
	tasks, err := dvstore.LoadTasks(api.h, api.schema, filters)
	if err != nil {
		_ = render.Render(w, r, chiutil.ErrRender(err))
		return
	}

	// return JSON
	if err := render.Render(w, r, tasks); err != nil {
		_ = render.Render(w, r, chiutil.ErrRender(err))
		return
	}
}
