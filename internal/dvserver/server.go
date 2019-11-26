package dvserver

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"strings"
	"time"

	"github.com/cayleygraph/cayley"
	"github.com/cayleygraph/cayley/schema"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	packr "github.com/gobuffalo/packr/v2"
	"github.com/gogo/gateway"
	grpc_middleware "github.com/grpc-ecosystem/go-grpc-middleware"
	grpc_zap "github.com/grpc-ecosystem/go-grpc-middleware/logging/zap"
	grpc_recovery "github.com/grpc-ecosystem/go-grpc-middleware/recovery"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/oklog/run"
	"github.com/rs/cors"
	chilogger "github.com/treastech/logger"
	cache "github.com/victorspringer/http-cache"
	"github.com/victorspringer/http-cache/adapter/memory"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"moul.io/depviz/internal/chiutil"
)

type Opts struct {
	Logger             *zap.Logger
	HTTPBind           string
	GRPCBind           string
	CORSAllowedOrigins string
	ShutdownTimeout    time.Duration
	RequestTimeout     time.Duration
	WithoutRecovery    bool
	WithPprof          bool
	Godmode            bool
	WithoutCache       bool
	BasicAuth          string
	Realm              string
}

type Service interface {
	DepvizServiceServer
	Run() error
	Close()
	HTTPListenerAddr() string
	GRPCListenerAddr() string
}

type service struct {
	ctx              context.Context
	schema           *schema.Config
	h                *cayley.Handle
	opts             Opts
	workers          run.Group
	grpcServer       *grpc.Server
	grpcListenerAddr string
	httpListenerAddr string
}

var _ DepvizServiceServer = (*service)(nil)

func New(ctx context.Context, h *cayley.Handle, schema *schema.Config, opts Opts) (Service, error) {
	if opts.Logger == nil {
		opts.Logger = zap.NewNop()
	}
	if opts.HTTPBind == "" {
		opts.HTTPBind = ":0"
	}
	if opts.GRPCBind == "" {
		opts.GRPCBind = ":0"
	}
	if opts.CORSAllowedOrigins == "" {
		opts.CORSAllowedOrigins = "*"
	}
	if opts.Realm == "" {
		opts.Realm = "DepViz"
	}

	svc := service{
		ctx:    ctx,
		h:      h,
		schema: schema,
		opts:   opts,
	}

	var (
		grpcLogger = opts.Logger.Named("gRPC")
		httpLogger = opts.Logger.Named("HTTP")
	)

	{ // local gRPC server
		serverStreamOpts := []grpc.StreamServerInterceptor{
			grpc_zap.StreamServerInterceptor(grpcLogger),
			// auth
		}
		serverUnaryOpts := []grpc.UnaryServerInterceptor{
			grpc_zap.UnaryServerInterceptor(grpcLogger),
			// auth
		}

		if !opts.WithoutRecovery {
			serverStreamOpts = append([]grpc.StreamServerInterceptor{grpc_recovery.StreamServerInterceptor()}, serverStreamOpts...)
			serverStreamOpts = append(serverStreamOpts, grpc_recovery.StreamServerInterceptor())
			serverUnaryOpts = append([]grpc.UnaryServerInterceptor{grpc_recovery.UnaryServerInterceptor()}, serverUnaryOpts...)
			serverUnaryOpts = append(serverUnaryOpts, grpc_recovery.UnaryServerInterceptor())
		}

		grpcServer := grpc.NewServer(
			grpc.StreamInterceptor(grpc_middleware.ChainStreamServer(serverStreamOpts...)),
			grpc.UnaryInterceptor(grpc_middleware.ChainUnaryServer(serverUnaryOpts...)),
		)
		RegisterDepvizServiceServer(grpcServer, &svc)
		svc.grpcServer = grpcServer
	}

	if opts.HTTPBind != "" || opts.GRPCBind != "" { // grpcbind is required for grpc-gateway (for now)
		grpcListener, err := net.Listen("tcp", opts.GRPCBind)
		if err != nil {
			return nil, fmt.Errorf("start gRPC listener: %w", err)
		}
		svc.grpcListenerAddr = grpcListener.Addr().String()

		svc.workers.Add(func() error {
			grpcLogger.Debug("starting gRPC server", zap.String("bind", opts.GRPCBind))
			return svc.grpcServer.Serve(grpcListener)
		}, func(error) {
			if err := grpcListener.Close(); err != nil {
				grpcLogger.Warn("close gRPC listener", zap.Error(err))
			}
		})
	}

	if opts.HTTPBind != "" {
		r := chi.NewRouter()
		if opts.BasicAuth != "" {
			r.Use(basicAuth(opts.BasicAuth, opts.Realm))
		}
		cors := cors.New(cors.Options{
			AllowedOrigins:   strings.Split(opts.CORSAllowedOrigins, ","),
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: true,
			MaxAge:           300,
		})
		r.Use(cors.Handler)
		r.Use(chilogger.Logger(httpLogger))
		r.Use(middleware.Recoverer)
		r.Use(middleware.Timeout(opts.RequestTimeout))
		r.Use(middleware.RealIP)
		r.Use(middleware.RequestID)
		gwmux := runtime.NewServeMux(
			runtime.WithMarshalerOption(runtime.MIMEWildcard, &gateway.JSONPb{
				EmitDefaults: false,
				Indent:       "  ",
				OrigName:     true,
			}),
			runtime.WithProtoErrorHandler(runtime.DefaultHTTPProtoErrorHandler),
		)
		grpcOpts := []grpc.DialOption{grpc.WithInsecure()}
		if err := RegisterDepvizServiceHandlerFromEndpoint(ctx, gwmux, svc.grpcListenerAddr, grpcOpts); err != nil {
			return nil, fmt.Errorf("register service on gateway: %w", err)
		}

		var handler http.Handler = gwmux

		// api endpoints
		if !opts.WithoutCache {
			// FIXME: invalidate cache
			memcached, err := memory.NewAdapter(
				memory.AdapterWithAlgorithm(memory.LRU),
				memory.AdapterWithCapacity(10000000),
			)
			if err != nil {
				return nil, fmt.Errorf("memory cache: %w", err)
			}

			cacheClient, err := cache.NewClient(
				cache.ClientWithAdapter(memcached),
				cache.ClientWithTTL(10*time.Minute),
				cache.ClientWithRefreshKey("opn"),
			)
			if err != nil {
				return nil, fmt.Errorf("cache client: %w", err)
			}

			handler = cacheClient.Middleware(handler)
		}
		r.Mount("/api", http.StripPrefix("/api", handler))

		// pprof endpoints
		if opts.WithPprof {
			r.HandleFunc("/debug/pprof/*", pprof.Index)
			r.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
			r.HandleFunc("/debug/pprof/profile", pprof.Profile)
			r.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
			r.HandleFunc("/debug/pprof/trace", pprof.Trace)
		}

		// static content
		box := packr.New("web", "../../web")
		chiutil.FileServer(r, "/", box)

		// pages
		r.Get("/", homepage(box, opts))

		http.DefaultServeMux = http.NewServeMux() // disables default handlers registere by importing net/http/pprof for security reasons
		listener, err := net.Listen("tcp", opts.HTTPBind)
		if err != nil {
			return nil, fmt.Errorf("start HTTP listener: %w", err)
		}
		svc.httpListenerAddr = listener.Addr().String()
		srv := http.Server{
			Handler: r,
		}
		svc.workers.Add(func() error {
			httpLogger.Debug("starting HTTP server", zap.String("bind", opts.HTTPBind))
			return srv.Serve(listener)
		}, func(error) {
			ctx, cancel := context.WithTimeout(ctx, opts.ShutdownTimeout)
			if err := srv.Shutdown(ctx); err != nil {
				httpLogger.Warn("shutdown HTTP server", zap.Error(err))
			}
			defer cancel()
			if err := listener.Close(); err != nil {
				httpLogger.Warn("close HTTP listener", zap.Error(err))
			}
		})
	}

	// FIXME: add grpc-web support?

	return &svc, nil
}

func (s *service) Run() error {
	return s.workers.Run()
}

func (s *service) Close() {
	s.grpcServer.GracefulStop()
}

func (s service) HTTPListenerAddr() string { return s.httpListenerAddr }
func (s service) GRPCListenerAddr() string { return s.grpcListenerAddr }
