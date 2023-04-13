package dvserver

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"

	"go.uber.org/zap"
	"google.golang.org/grpc/metadata"
	"moul.io/depviz/v3/pkg/dvcore"
	"moul.io/depviz/v3/pkg/dvmodel"
	"moul.io/depviz/v3/pkg/dvparser"
	"moul.io/depviz/v3/pkg/dvstore"
)

func gitHubOAuth(opts Opts, httpLogger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if opts.GitHubClientID == "" || opts.GitHubClientSecret == "" {
			httpLogger.Error("GitHub OAuth: missing client ID or secret")
			http.Error(w, "missing client ID or secret", http.StatusInternalServerError)
			return
		}

		code, err := io.ReadAll(r.Body)
		if err != nil {
			httpLogger.Error("get body", zap.Error(err))
			http.Error(w, "failed to retrieve body", http.StatusInternalServerError)
			return
		}

		if len(code) < 1 {
			httpLogger.Error("Url Param 'code' is missing")
			http.Error(w, "Url Param 'code' is missing", http.StatusBadRequest)
			return
		}
		httpLogger.Info("github code received successfully")

		//  maybe switch to env variables
		data := []byte(fmt.Sprintf(`{
				"client_id":     "%s",
				"client_secret": "%s",
				"code":          "%s"
			}`, opts.GitHubClientID, opts.GitHubClientSecret, string(code)))
		req, err := http.NewRequestWithContext(context.Background(), "POST", "https://github.com/login/oauth/access_token", bytes.NewBuffer(data))
		if err != nil {
			httpLogger.Error("create request", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		req.Header.Add("Content-Type", "application/json")
		req.Header.Add("Accept", "application/json")

		gitHubResponse, err := http.DefaultClient.Do(req)
		if err != nil {
			httpLogger.Error("request token to github", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		defer gitHubResponse.Body.Close()

		token, err := io.ReadAll(gitHubResponse.Body)
		if err != nil {
			httpLogger.Error("get body", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}

		_, err = w.Write(token)
		if err != nil {
			httpLogger.Error("write response", zap.Error(err))
			http.Error(w, "internal server error", http.StatusInternalServerError)
			return
		}
	}
}

func getToken(ctx context.Context) (string, error) {
	md, _ := metadata.FromIncomingContext(ctx)

	var gitHubToken string
	if md["authorization"] == nil {
		return "", nil
	}
	// skip "Basic"
	if len(md["authorization"][0]) <= len("Basic ") {
		return "", fmt.Errorf("invalid authorization header")
	}
	gitHubToken = md["authorization"][1][6:]
	// prevent empty token (skip prefix)
	if gitHubToken == base64.StdEncoding.EncodeToString([]byte("depviz:")) {
		return gitHubToken, nil
	}
	bytesGithubToken, err := base64.StdEncoding.DecodeString(gitHubToken)
	if err != nil {
		return "", fmt.Errorf("invalid github token: %w", err)
	}
	// len("depviz:") = 6
	gitHubToken = string(bytesGithubToken[7:])

	return gitHubToken, nil
}

func (s *service) Graph(ctx context.Context, in *Graph_Input) (*Graph_Output, error) {
	s.opts.Logger.Debug("graph", zap.Any("in", in))

	// retrieve token
	gitHubToken, err := getToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("get token: %w", err)
	}

	// validation
	if len(in.Targets) == 0 {
		return nil, fmt.Errorf("targets is required")
	}

	filters := dvstore.LoadTasksFilters{
		WithClosed:          in.WithClosed,
		WithoutIsolated:     in.WithoutIsolated,
		WithoutPRs:          in.WithoutPRs,
		WithoutExternalDeps: in.WithoutExternalDeps,
		WithFetch:           in.WithFetch,
	}
	if len(in.Targets) == 1 && in.Targets[0] == "world" {
		filters.TheWorld = true
	} else {
		targets, err := dvparser.ParseTargets(in.Targets)
		if err != nil {
			return nil, fmt.Errorf("parse targets: %w", err)
		}
		filters.Targets = targets
	}

	// load tasks
	if filters.WithFetch && gitHubToken != "" {
		_, err := dvcore.PullAndSave(filters.Targets, s.h, s.schema, gitHubToken, false, s.opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("pull: %w", err)
		}
	}

	var tasks dvmodel.Tasks
	tasks, err = dvstore.LoadTasks(s.h, s.schema, filters, s.opts.Logger)
	if err != nil {
		return nil, fmt.Errorf("load tasks: %w", err)
	}

	// fetch if not already in db
	if len(tasks) == 0 {
		_, err := dvcore.PullAndSave(filters.Targets, s.h, s.schema, s.opts.GitHubToken, false, s.opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("pull: %w", err)
		}
		tasks, err = dvstore.LoadTasks(s.h, s.schema, filters, s.opts.Logger)
		if err != nil {
			return nil, fmt.Errorf("load tasks: %w", err)
		}
	}

	// build output
	ret := Graph_Output{
		Tasks: make([]*dvmodel.Task, len(tasks)),
	}
	for idx, task := range tasks {
		clone := task
		ret.Tasks[idx] = &clone
	}
	return &ret, nil
}

func (s *service) StoreDump(ctx context.Context, in *StoreDump_Input) (*StoreDump_Output, error) {
	if !s.opts.Godmode {
		return nil, fmt.Errorf("permission denied (--god-mode required)")
	}
	batch, err := dvcore.GetStoreDump(ctx, s.h, s.schema)
	if err != nil {
		return nil, fmt.Errorf("store dump: %w", err)
	}

	ret := StoreDump_Output{
		Batch: batch,
	}
	return &ret, nil
}

func (s *service) Ping(context.Context, *Ping_Input) (*Ping_Output, error) {
	return &Ping_Output{Message: "pong"}, nil
}

func (s *service) Status(context.Context, *Status_Input) (*Status_Output, error) {
	return &Status_Output{EverythingIsOK: true}, nil
}
