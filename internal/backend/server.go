package backend

import (
	"bytes"
	"context"
	"crypto/subtle"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"moul.io/depviz/v4/internal/core"
	"moul.io/depviz/v4/live"
)

// ActivityBus tracks in-flight and recently completed background operations.
type ActivityBus struct {
	mu         sync.Mutex
	activities []*Activity
	nextID     int
}

// Activity represents a single background operation.
type Activity struct {
	ID        string     `json:"id"`
	Kind      string     `json:"kind"`
	Label     string     `json:"label"`
	Status    string     `json:"status"`
	Done      int        `json:"done"`
	Total     int        `json:"total"`
	Detail    string     `json:"detail"`
	StartedAt time.Time  `json:"started_at"`
	EndedAt   *time.Time `json:"ended_at,omitempty"`
	Error     string     `json:"error,omitempty"`
}

func (b *ActivityBus) Start(kind, label string) *Activity {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.nextID++
	a := &Activity{
		ID:        fmt.Sprintf("%d", b.nextID),
		Kind:      kind,
		Label:     label,
		Status:    "running",
		StartedAt: time.Now().UTC(),
	}
	b.activities = append(b.activities, a)
	return a
}

func (b *ActivityBus) Update(a *Activity, done, total int, detail string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if a.Status != "running" {
		return
	}
	if done > 0 {
		a.Done = done
	}
	if total > 0 {
		a.Total = total
	}
	if detail != "" {
		a.Detail = detail
	}
}

func (b *ActivityBus) Finish(a *Activity, done, total int, detail string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now().UTC()
	a.Status = "done"
	a.EndedAt = &now
	if done > 0 {
		a.Done = done
	}
	if total > 0 {
		a.Total = total
	}
	if detail != "" {
		a.Detail = detail
	}
}

func (b *ActivityBus) Fail(a *Activity, errMsg string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now().UTC()
	a.Status = "failed"
	a.EndedAt = &now
	a.Error = errMsg
}

func (b *ActivityBus) Active() []*Activity {
	b.mu.Lock()
	defer b.mu.Unlock()
	cutoff := time.Now().UTC().Add(-8 * time.Second)
	var out []*Activity
	var kept []*Activity
	for _, a := range b.activities {
		if a.Status == "running" || (a.EndedAt != nil && a.EndedAt.After(cutoff)) {
			out = append(out, a)
			kept = append(kept, a)
		}
	}
	b.activities = kept
	return out
}

type ctxKeyActivity struct{}

func withActivity(ctx context.Context, a *Activity) context.Context {
	return context.WithValue(ctx, ctxKeyActivity{}, a)
}

func activityFromCtx(ctx context.Context) *Activity {
	a, _ := ctx.Value(ctxKeyActivity{}).(*Activity)
	return a
}

const sessionCookieName = "depviz_session"

type Config struct {
	Addr                    string
	BaseURL                 string
	GitHubClientID          string
	GitHubClientSecret      string
	GitHubAppID             string
	GitHubAppPrivateKeyFile string
	GitHubWebhookSecret     string
	SessionTTL              time.Duration
	// BasicAuthUser/BasicAuthPass gate every route except /api/health when set.
	// Without them a deployed instance is world-readable: sessions only exist via
	// GitHub OAuth, so an instance with no OAuth app configured has no other gate.
	BasicAuthUser string
	BasicAuthPass string
}

type Server struct {
	cfg        Config
	store      *core.Store
	client     *http.Client
	activities *ActivityBus
}

func NewServer(store *core.Store, cfg Config) *Server {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 30 * 24 * time.Hour
	}
	return &Server{
		cfg:        cfg,
		store:      store,
		client:     http.DefaultClient,
		activities: &ActivityBus{},
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/export", s.handleExport)
	mux.HandleFunc("/api/boards", s.handleBoards)
	mux.HandleFunc("/api/board-items", s.handleBoardItems)
	mux.HandleFunc("/api/board-links", s.handleBoardLinks)
	mux.HandleFunc("/api/activities", s.handleActivities)
	mux.HandleFunc("/api/board-sync", s.handleBoardSync)
	mux.HandleFunc("/api/github/orgs", s.handleGitHubOrgs)
	mux.HandleFunc("/api/github/projects", s.handleGitHubProjects)
	mux.HandleFunc("/api/github/repos", s.handleGitHubRepos)
	mux.HandleFunc("/api/github/webhook", s.handleGitHubWebhook)
	mux.HandleFunc("/api/github/create-issue", s.handleCreateGitHubIssue)
	mux.HandleFunc("/api/github/update-issue", s.handleUpdateGitHubIssue)
	mux.HandleFunc("/api/github/comment", s.handleCreateGitHubComment)
	mux.HandleFunc("/api/board-source/apply", s.handleBoardSourceApply)
	mux.HandleFunc("/api/suggestions/dismiss", s.handleDismissSuggestion)
	mux.HandleFunc("/api/board-sync-logs", s.handleBoardSyncLogs)
	mux.HandleFunc("/api/board-views", s.handleBoardViews)
	mux.HandleFunc("/api/overrides", s.handleOverrides)
	mux.HandleFunc("/api/auth/github/start", s.handleGitHubStart)
	mux.HandleFunc("/api/auth/github/callback", s.handleGitHubCallback)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)
	mux.HandleFunc("/api/workspaces", s.handleWorkspaces)
	mux.Handle("/app/", http.StripPrefix("/app", http.FileServer(http.FS(live.AppFS()))))
	mux.Handle("/", http.FileServer(http.FS(live.SiteFS())))
	return s.withBasicAuth(mux)
}

// isPublicPath reports whether a path stays reachable when the instance is gated
// by basic auth. Two things stay open, and both are board-data-free by
// construction: /api/health (booleans only, so deploy health checks and the
// post-deploy contract keep working) and everything served from live.SiteFS —
// the landing page, whose whole job is to say what this instance is. The board
// lives under /app/ and the rest of /api/, which stay behind the gate.
func isPublicPath(p string) bool {
	if p == "/api/health" {
		return true
	}
	return !strings.HasPrefix(p, "/api/") && !strings.HasPrefix(p, "/app/")
}

// withBasicAuth gates the whole instance when BasicAuthUser/BasicAuthPass are set,
// except for the paths isPublicPath allows.
func (s *Server) withBasicAuth(next http.Handler) http.Handler {
	if s.cfg.BasicAuthUser == "" && s.cfg.BasicAuthPass == "" {
		return next
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isPublicPath(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		user, pass, ok := r.BasicAuth()
		userOK := subtle.ConstantTimeCompare([]byte(user), []byte(s.cfg.BasicAuthUser)) == 1
		passOK := subtle.ConstantTimeCompare([]byte(pass), []byte(s.cfg.BasicAuthPass)) == 1
		if !ok || !userOK || !passOK {
			w.Header().Set("WWW-Authenticate", `Basic realm="depviz", charset="UTF-8"`)
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                        true,
		"github_oauth_configured":   s.githubOAuthConfigured(),
		"github_app_configured":     s.githubAppConfigured(),
		"github_webhook_configured": s.githubWebhookConfigured(),
	})
}

func (s *Server) handleActivities(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache")
	writeJSON(w, http.StatusOK, map[string]any{"activities": s.activities.Active()})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if _, ok := s.requireAccount(w, r); !ok {
		return
	}
	board := r.URL.Query().Get("board")
	if board == "" {
		board = core.DefaultBoardID
	}
	payload, err := s.store.BuildExport(r.Context(), board)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	format := r.URL.Query().Get("format")
	if format == "flow" {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, snapshotToFlowText(payload))
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func snapshotToFlowText(payload any) string {
	data, _ := json.Marshal(payload)
	var export struct {
		Snapshot struct {
			Board struct {
				Name string `json:"name"`
				ID   string `json:"id"`
			} `json:"board"`
			Nodes []struct {
				ID    string `json:"id"`
				Kind  string `json:"kind"`
				Title string `json:"title"`
				State string `json:"state"`
				Owner string `json:"owner"`
			} `json:"nodes"`
			Edges []struct {
				FromID    string `json:"from_id"`
				ToID      string `json:"to_id"`
				Kind      string `json:"kind"`
				Authority string `json:"authority"`
			} `json:"edges"`
		} `json:"snapshot"`
	}
	if err := json.Unmarshal(data, &export); err != nil {
		return ""
	}
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("board %q\n", export.Snapshot.Board.Name))
	for _, n := range export.Snapshot.Nodes {
		if strings.HasPrefix(n.ID, "gh:") {
			continue
		}
		sb.WriteString(fmt.Sprintf("%s %s", n.Kind, strings.ReplaceAll(n.ID, n.Kind+":", "")))
		if n.Title != "" && n.Title != n.ID {
			sb.WriteString(fmt.Sprintf(" %q", n.Title))
		}
		if n.State != "" && n.State != "open" {
			sb.WriteString(fmt.Sprintf(" [%s]", n.State))
		}
		sb.WriteString("\n")
	}
	for _, e := range export.Snapshot.Edges {
		if e.Authority == "local" || e.Authority == "user" {
			verb := e.Kind
			if e.Kind == "blocked_by" {
				verb = "depends on"
			}
			if e.Kind == "relates_to" {
				verb = "relates to"
			}
			sb.WriteString(fmt.Sprintf("%s %s %s\n", e.FromID, verb, e.ToID))
		}
	}
	return sb.String()
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	account, ok, err := s.accountForRequest(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out := map[string]any{
		"authenticated":             ok,
		"github_oauth_configured":   s.githubOAuthConfigured(),
		"github_app_configured":     s.githubAppConfigured(),
		"github_webhook_configured": s.githubWebhookConfigured(),
	}
	if ok {
		out["account"] = account
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleBoards(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAccount(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		boards, err := s.store.BoardList(r.Context())
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"boards": boards})
	case http.MethodPost:
		var in struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Preset      string `json:"preset"`
			Provider    string `json:"provider"`
			Owner       string `json:"owner"`
			Repo        string `json:"repo"`
			SourceID    string `json:"source_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		configJSON, _ := json.Marshal(map[string]any{
			"preset":    strings.TrimSpace(in.Preset),
			"provider":  strings.TrimSpace(in.Provider),
			"owner":     strings.TrimSpace(in.Owner),
			"repo":      strings.TrimSpace(in.Repo),
			"source_id": strings.TrimSpace(in.SourceID),
		})
		scope := strings.TrimSpace(in.Preset)
		if in.Repo != "" {
			scope = "repo:" + strings.TrimSpace(in.Repo)
		} else if in.Owner != "" {
			scope = "org:" + strings.TrimSpace(in.Owner)
		} else if strings.TrimSpace(in.Preset) == "my-work" {
			scope = "my-work"
		}
		board, err := s.store.CreateBoardWithConfig(r.Context(), in.Name, in.Description, scope, string(configJSON))
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"board": board})
	default:
		w.Header().Set("Allow", "GET, POST")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleBoardItems(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAccount(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodPost:
		var in struct {
			BoardID     string   `json:"board_id"`
			Action      string   `json:"action"`
			NodeID      string   `json:"node_id"`
			Kind        string   `json:"kind"`
			Ref         string   `json:"ref"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Owner       string   `json:"owner"`
			Description string   `json:"description"`
			TimeHorizon string   `json:"time_horizon"`
			Priority    string   `json:"priority"`
			Labels      []string `json:"labels"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		boardID := strings.TrimSpace(in.BoardID)
		if boardID == "" {
			boardID = core.DefaultBoardID
		}
		action := strings.TrimSpace(in.Action)
		if action == "duplicate" {
			nodeID := strings.TrimSpace(in.NodeID)
			if nodeID == "" {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id is required for duplicate"})
				return
			}
			node, err := s.store.DuplicateNode(r.Context(), boardID, nodeID)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"node": node})
			return
		}
		if action == "restore" {
			nodeID := strings.TrimSpace(in.NodeID)
			if err := s.store.RestoreNode(r.Context(), nodeID); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
			return
		}
		kind := strings.ToLower(strings.TrimSpace(in.Kind))
		ref := strings.TrimSpace(in.Ref)
		title := strings.TrimSpace(in.Title)
		strategyKinds := map[string]bool{
			"strategy": true, "initiative": true, "bet": true, "project": true,
			"workstream": true, "risk": true, "decision": true, "question": true,
			"metric": true,
		}
		if kind == "" || kind == "auto" {
			if _, ok := parseGitHubRef(ref); ok {
				kind = "github"
			} else if title != "" {
				kind = "task"
			} else {
				kind = "note"
			}
		}
		switch {
		case kind == "github":
			gh, ok := parseGitHubRef(ref)
			if !ok {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "expected GitHub URL, owner/repo#123, or owner/repo!123"})
				return
			}
			node, err := s.store.AddGitHubRefToBoard(r.Context(), boardID, gh.repo, gh.marker, gh.number, title)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"node": node})
		case kind == "note":
			text := title
			if text == "" {
				text = ref
			}
			node, err := s.store.CreateNote(r.Context(), boardID, text)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"node": node})
		case kind == "task" || strategyKinds[kind]:
			text := title
			if text == "" {
				text = ref
			}
			node, err := s.store.CreateStrategyNode(r.Context(), boardID, kind, text, in.Status, in.Owner, in.Description, in.TimeHorizon, in.Priority, in.Labels)
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusCreated, map[string]any{"node": node})
		default:
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "kind must be github, task, note, strategy, initiative, bet, project, workstream, risk, decision, question, metric, or auto"})
		}
	case http.MethodPatch:
		var in struct {
			NodeID      string    `json:"node_id"`
			Title       *string   `json:"title"`
			Status      *string   `json:"status"`
			Owner       *string   `json:"owner"`
			Description *string   `json:"description"`
			TimeHorizon *string   `json:"time_horizon"`
			Priority    *string   `json:"priority"`
			Labels      *[]string `json:"labels"`
			Kind        string    `json:"kind"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		nodeID := strings.TrimSpace(in.NodeID)
		if nodeID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id is required"})
			return
		}
		if in.Kind != "" {
			node, err := s.store.ConvertNodeKind(r.Context(), nodeID, strings.TrimSpace(strings.ToLower(in.Kind)))
			if err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"node": node})
			return
		}
		node, err := s.store.UpdateNodeFields(r.Context(), nodeID, core.NodeFieldUpdate{
			Title:       in.Title,
			Status:      in.Status,
			Owner:       in.Owner,
			Description: in.Description,
			TimeHorizon: in.TimeHorizon,
			Priority:    in.Priority,
			Labels:      in.Labels,
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"node": node})
	case http.MethodDelete:
		var in struct {
			BoardID string `json:"board_id"`
			NodeID  string `json:"node_id"`
			Soft    bool   `json:"soft"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		boardID := strings.TrimSpace(in.BoardID)
		if boardID == "" {
			boardID = core.DefaultBoardID
		}
		nodeID := strings.TrimSpace(in.NodeID)
		if nodeID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id is required"})
			return
		}
		if in.Soft {
			if err := s.store.ArchiveNode(r.Context(), nodeID); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		} else {
			if err := s.store.RemoveNodeFromBoard(r.Context(), boardID, nodeID); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	case http.MethodGet:
		boardID := r.URL.Query().Get("board_id")
		if boardID == "" {
			boardID = core.DefaultBoardID
		}
		archived := r.URL.Query().Get("archived")
		if archived == "true" {
			nodes, err := s.store.ListArchivedNodes(r.Context(), boardID)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, map[string]any{"nodes": nodes})
			return
		}
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "use archived=true"})
	default:
		w.Header().Set("Allow", "GET, POST, PATCH, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleBoardLinks(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAccount(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodPost:
		var in struct {
			BoardID string `json:"board_id"`
			From    string `json:"from"`
			To      string `json:"to"`
			Kind    string `json:"kind"`
			Note    string `json:"note"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		boardID := strings.TrimSpace(in.BoardID)
		if boardID == "" {
			boardID = core.DefaultBoardID
		}
		snap, err := s.store.Snapshot(r.Context(), boardID)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		from, err := resolveBoardNodeRef(snap, in.From)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "from: " + err.Error()})
			return
		}
		to, err := resolveBoardNodeRef(snap, in.To)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "to: " + err.Error()})
			return
		}
		kind := strings.TrimSpace(in.Kind)
		if kind == "" {
			kind = "blocked_by"
		}
		edge, err := s.store.AddEdge(r.Context(), boardID, from, to, kind, "user", map[string]any{"note": strings.TrimSpace(in.Note), "source": "manual"})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"edge": edge})
	case http.MethodDelete:
		var in struct {
			EdgeID string `json:"edge_id"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		edgeID := strings.TrimSpace(in.EdgeID)
		if edgeID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "edge_id is required"})
			return
		}
		if err := s.store.DeleteEdge(r.Context(), edgeID); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		w.Header().Set("Allow", "POST, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleBoardSync(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	var in struct {
		BoardID string `json:"board_id"`
		Limit   int    `json:"limit"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	boardID := strings.TrimSpace(in.BoardID)
	if boardID == "" {
		boardID = core.DefaultBoardID
	}
	snap, err := s.store.Snapshot(r.Context(), boardID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	limit := in.Limit
	if limit <= 0 || limit > 100 {
		limit = 100
	}
	_ = s.store.RecordBoardSync(r.Context(), boardID, "running", map[string]any{"scope": snap.Board.ScopeQuery, "limit": limit})
	act := s.activities.Start("sync", "Syncing "+boardScopeLabel(snap.Board))
	token, tokenMode, err := s.githubTokenForBoardSync(r.Context(), account.ID, snap.Board)
	if err != nil {
		s.activities.Fail(act, err.Error())
		_ = s.store.RecordBoardSync(r.Context(), boardID, "failed", map[string]any{"scope": snap.Board.ScopeQuery, "limit": limit, "error": err.Error()})
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	syncCtx := withActivity(r.Context(), act)
	count, edges, err := s.syncGitHubBoardScope(syncCtx, token, account.Login, boardID, snap.Board, limit)
	if err != nil && canRetryGitHubPublicSync(err, tokenMode, snap.Board) {
		count, edges, err = s.syncGitHubBoardScope(syncCtx, "", account.Login, boardID, snap.Board, limit)
		tokenMode = "github-public-rest"
	}
	if err != nil {
		err = friendlyGitHubSyncError(err, tokenMode, snap.Board.ScopeQuery)
		s.activities.Fail(act, err.Error())
		_ = s.store.RecordBoardSync(r.Context(), boardID, "failed", map[string]any{"scope": snap.Board.ScopeQuery, "limit": limit, "mode": tokenMode, "error": err.Error()})
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	s.activities.Finish(act, count, 0, fmt.Sprintf("%d items, %d links", count, edges))
	_ = s.store.RecordBoardSync(r.Context(), boardID, "ok", map[string]any{"scope": snap.Board.ScopeQuery, "limit": limit, "mode": tokenMode, "items": count, "links": edges})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "scope": snap.Board.ScopeQuery, "mode": tokenMode, "items": count, "links": edges})
}

func (s *Server) handleGitHubStart(w http.ResponseWriter, r *http.Request) {
	if !s.githubOAuthConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "github auth is not configured"})
		return
	}
	returnTo := safeReturnPath(r.URL.Query().Get("return_to"))
	state, err := s.store.CreateOAuthState(r.Context(), "github", returnTo, 10*time.Minute)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	params := url.Values{
		"client_id":    {s.cfg.GitHubClientID},
		"redirect_uri": {s.callbackURL()},
		"state":        {state},
	}
	if !s.githubAppConfigured() {
		params.Set("scope", "repo read:user user:email read:org read:project")
	}
	http.Redirect(w, r, "https://github.com/login/oauth/authorize?"+params.Encode(), http.StatusFound)
}

func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if !s.githubOAuthConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "github auth is not configured"})
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
	installationID := parseInstallationID(r.URL.Query().Get("installation_id"))
	if code == "" || state == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "missing code or state"})
		return
	}
	returnTo, err := s.store.ConsumeOAuthState(r.Context(), "github", state)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	token, err := s.exchangeGitHubCode(r.Context(), code)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	user, err := s.fetchGitHubUser(r.Context(), token.AccessToken)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	tokenJSON, _ := json.Marshal(token)
	account, err := s.store.UpsertOAuthAccount(r.Context(), core.OAuthAccountInput{
		Provider:   "github",
		ExternalID: fmt.Sprint(user.ID),
		Login:      user.Login,
		Name:       user.Name,
		AvatarURL:  user.AvatarURL,
		HTMLURL:    user.HTMLURL,
		Scopes:     token.Scopes(),
		TokenJSON:  string(tokenJSON),
	})
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	workspace, err := s.store.UpsertWorkspace(r.Context(), core.Workspace{
		Provider:   "github",
		ExternalID: fmt.Sprint(user.ID),
		Kind:       "user",
		Name:       user.Login,
		URL:        user.HTMLURL,
	})
	if err == nil {
		_ = s.store.UpsertWorkspaceMembership(r.Context(), workspace.ID, account.ID, "owner", "github")
	}
	if installationID != 0 && s.githubAppConfigured() {
		_ = s.syncGitHubInstallation(r.Context(), installationID)
	}
	session, expires, err := s.store.CreateWebSession(r.Context(), account.ID, s.cfg.SessionTTL)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    session,
		Path:     "/",
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(s.cfg.BaseURL, "https://"),
	})
	http.Redirect(w, r, returnTo, http.StatusFound)
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	cookie, err := r.Cookie(sessionCookieName)
	if err == nil {
		_ = s.store.DeleteWebSession(r.Context(), cookie.Value)
	}
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   strings.HasPrefix(s.cfg.BaseURL, "https://"),
	})
	writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
}

func (s *Server) handleWorkspaces(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	workspaces, err := s.store.ListWorkspacesForAccount(r.Context(), account.ID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	if workspaces == nil {
		workspaces = []core.Workspace{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": workspaces})
}

func (s *Server) handleGitHubRepos(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	token, ok := s.githubAccessTokenForAccount(w, r, account.ID)
	if !ok {
		return
	}
	var repos []githubRepo
	path := "/user/repos?affiliation=owner,collaborator,organization_member&sort=updated&per_page=100"
	if err := s.doGitHubREST(r.Context(), token, path, &repos); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	for _, repo := range repos {
		raw, _ := json.Marshal(repo)
		workspace, err := s.store.UpsertWorkspace(r.Context(), core.Workspace{
			Provider:   "github",
			ExternalID: fmt.Sprint(repo.ID),
			Kind:       "repo",
			Name:       repo.FullName,
			URL:        repo.HTMLURL,
			DataJSON:   string(raw),
		})
		if err == nil {
			_ = s.store.UpsertWorkspaceMembership(r.Context(), workspace.ID, account.ID, "member", "github")
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"repos": repos})
}

func (s *Server) handleGitHubOrgs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	token, ok := s.githubAccessTokenForAccount(w, r, account.ID)
	if !ok {
		return
	}
	var orgs []githubOrg
	if err := s.doGitHubREST(r.Context(), token, "/user/orgs?per_page=100", &orgs); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	for _, org := range orgs {
		raw, _ := json.Marshal(org)
		workspace, err := s.store.UpsertWorkspace(r.Context(), core.Workspace{
			Provider:   "github",
			ExternalID: fmt.Sprint(org.ID),
			Kind:       "org",
			Name:       org.Login,
			URL:        org.HTMLURL,
			DataJSON:   string(raw),
		})
		if err == nil {
			_ = s.store.UpsertWorkspaceMembership(r.Context(), workspace.ID, account.ID, "member", "github")
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"orgs": orgs})
}

func (s *Server) handleGitHubProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	token, ok := s.githubAccessTokenForAccount(w, r, account.ID)
	if !ok {
		return
	}
	query := `query DepVizProjects {
		viewer {
			projectsV2(first: 50, orderBy: {field: UPDATED_AT, direction: DESC}) {
				nodes { id title url updatedAt }
			}
			organizations(first: 50) {
				nodes {
					login
					projectsV2(first: 50, orderBy: {field: UPDATED_AT, direction: DESC}) {
						nodes { id title url updatedAt }
					}
				}
			}
		}
	}`
	var gql githubProjectsGraphQL
	if err := s.doGitHubGraphQL(r.Context(), token, query, &gql); err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	projects := make([]githubProject, 0, len(gql.Viewer.ProjectsV2.Nodes))
	for _, project := range gql.Viewer.ProjectsV2.Nodes {
		project.Owner = account.Login
		projects = append(projects, project)
	}
	for _, org := range gql.Viewer.Organizations.Nodes {
		for _, project := range org.ProjectsV2.Nodes {
			project.Owner = org.Login
			projects = append(projects, project)
		}
	}
	for _, project := range projects {
		raw, _ := json.Marshal(project)
		workspace, err := s.store.UpsertWorkspace(r.Context(), core.Workspace{
			Provider:   "github",
			ExternalID: project.ID,
			Kind:       "project",
			Name:       project.Title,
			URL:        project.URL,
			DataJSON:   string(raw),
		})
		if err == nil {
			_ = s.store.UpsertWorkspaceMembership(r.Context(), workspace.ID, account.ID, "member", "github")
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

func (s *Server) handleOverrides(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		nodeID := r.URL.Query().Get("node_id")
		if nodeID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id is required"})
			return
		}
		override, err := s.store.GetPersonalOverride(r.Context(), account.ID, "node", nodeID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"override": override})
	case http.MethodPost:
		var in struct {
			OwnerType string          `json:"owner_type"`
			OwnerID   string          `json:"owner_id"`
			Data      json.RawMessage `json:"data"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if len(in.Data) == 0 {
			in.Data = json.RawMessage(`{}`)
		}
		override, err := s.store.UpsertPersonalOverride(r.Context(), core.PersonalOverride{
			AccountID: account.ID,
			OwnerType: in.OwnerType,
			OwnerID:   in.OwnerID,
			DataJSON:  string(in.Data),
		})
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"override": override})
	case http.MethodDelete:
		nodeID := r.URL.Query().Get("node_id")
		if nodeID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "node_id is required"})
			return
		}
		if err := s.store.DeletePersonalOverride(r.Context(), account.ID, "node", nodeID); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) accountForRequest(r *http.Request) (core.Account, bool, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return core.Account{}, false, nil
	}
	return s.store.AccountForWebSession(r.Context(), cookie.Value)
}

func (s *Server) requireAccount(w http.ResponseWriter, r *http.Request) (core.Account, bool) {
	account, ok, err := s.accountForRequest(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return core.Account{}, false
	}
	if !ok {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return core.Account{}, false
	}
	return account, true
}

func (s *Server) requireBoardAccess(w http.ResponseWriter, r *http.Request, boardID string, sess core.Account) bool {
	if sess.ID == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		return false
	}
	if strings.TrimSpace(boardID) == "" {
		boardID = core.DefaultBoardID
	}
	if _, err := s.store.BoardUpdatedAt(r.Context(), boardID); err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "board not found"})
		return false
	}
	return true
}

func (s *Server) githubAccessTokenForAccount(w http.ResponseWriter, r *http.Request, accountID string) (string, bool) {
	conn, ok, err := s.store.OAuthConnectionForAccount(r.Context(), accountID, "github")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return "", false
	}
	if !ok {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "github account is not connected"})
		return "", false
	}
	var token githubToken
	if err := json.Unmarshal([]byte(conn.TokenJSON), &token); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stored github token is unreadable"})
		return "", false
	}
	if token.AccessToken == "" {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "github account is not connected"})
		return "", false
	}
	return token.AccessToken, true
}

func (s *Server) githubTokenForBoardSync(ctx context.Context, accountID string, board core.Board) (string, string, error) {
	if owner := githubOwnerForBoard(board); owner != "" && s.githubAppConfigured() {
		token, ok, err := s.githubInstallationTokenForOwner(ctx, owner)
		if err != nil {
			return "", "", err
		}
		if ok {
			return token, "github-app-installation", nil
		}
	}
	conn, ok, err := s.store.OAuthConnectionForAccount(ctx, accountID, "github")
	if err != nil {
		return "", "", err
	}
	if !ok {
		return "", "", errors.New("github account is not connected; sign in with GitHub before syncing this view")
	}
	var token githubToken
	if err := json.Unmarshal([]byte(conn.TokenJSON), &token); err != nil {
		return "", "", errors.New("stored github token is unreadable; reconnect GitHub")
	}
	if token.AccessToken == "" {
		return "", "", errors.New("github account is not connected; reconnect GitHub")
	}
	return token.AccessToken, "github-oauth-user", nil
}

func (s *Server) githubInstallationTokenForOwner(ctx context.Context, owner string) (string, bool, error) {
	installations, err := s.store.GitHubInstallations(ctx)
	if err != nil {
		return "", false, err
	}
	for _, installation := range installations {
		if !strings.EqualFold(installation.AccountLogin, owner) || installation.InstallationID == 0 {
			continue
		}
		token, err := s.githubInstallationAccessToken(ctx, installation.InstallationID)
		if err != nil {
			return "", false, fmt.Errorf("github app installation token for %s failed: %w", owner, err)
		}
		return token.Token, true, nil
	}
	return "", false, nil
}

func githubOwnerForBoard(board core.Board) string {
	if repo := repoForBoard(board); repo != "" {
		return strings.Split(repo, "/")[0]
	}
	return orgForBoard(board)
}

func friendlyGitHubSyncError(err error, tokenMode, scope string) error {
	msg := strings.TrimSpace(err.Error())
	switch {
	case strings.Contains(msg, "401 Unauthorized"):
		if tokenMode == "github-oauth-user" {
			return fmt.Errorf("github OAuth token is no longer valid for %s; sign out and sign in again, or install the DepViz GitHub App on this owner: %s", scope, msg)
		}
		return fmt.Errorf("github app token was rejected for %s: %s", scope, msg)
	case strings.Contains(msg, "403 Forbidden"):
		return fmt.Errorf("github denied access for %s; check GitHub App permissions/installation or OAuth scopes: %s", scope, msg)
	case strings.Contains(msg, "404 Not Found"):
		return fmt.Errorf("github could not read %s; the repo may be private, missing, or the DepViz GitHub App is not installed on that owner: %s", scope, msg)
	default:
		return err
	}
}

func canRetryGitHubPublicSync(err error, tokenMode string, board core.Board) bool {
	if tokenMode != "github-oauth-user" || githubOwnerForBoard(board) == "" {
		return false
	}
	return strings.Contains(err.Error(), "401 Unauthorized")
}

func (s *Server) exchangeGitHubCode(ctx context.Context, code string) (githubToken, error) {
	body, _ := json.Marshal(map[string]string{
		"client_id":     s.cfg.GitHubClientID,
		"client_secret": s.cfg.GitHubClientSecret,
		"code":          code,
		"redirect_uri":  s.callbackURL(),
	})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://github.com/login/oauth/access_token", bytes.NewReader(body))
	if err != nil {
		return githubToken{}, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	var token githubToken
	if err := s.doJSON(req, &token); err != nil {
		return githubToken{}, err
	}
	if token.Error != "" {
		return githubToken{}, errors.New(token.ErrorDescription)
	}
	if token.AccessToken == "" {
		return githubToken{}, errors.New("github did not return an access token")
	}
	return token, nil
}

func (s *Server) fetchGitHubUser(ctx context.Context, accessToken string) (githubUser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return githubUser{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	var user githubUser
	if err := s.doJSON(req, &user); err != nil {
		return githubUser{}, err
	}
	if user.ID == 0 || user.Login == "" {
		return githubUser{}, errors.New("github user response is missing id or login")
	}
	return user, nil
}

func (s *Server) doGitHubREST(ctx context.Context, accessToken, path string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com"+path, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	return s.doJSON(req, out)
}

func (s *Server) doGitHubGraphQL(ctx context.Context, accessToken, query string, out any) error {
	body, _ := json.Marshal(map[string]string{"query": query})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)
	var envelope struct {
		Data   json.RawMessage `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := s.doJSON(req, &envelope); err != nil {
		return err
	}
	if len(envelope.Errors) > 0 {
		return errors.New(envelope.Errors[0].Message)
	}
	return json.Unmarshal(envelope.Data, out)
}

func (s *Server) syncGitHubRepoBoard(ctx context.Context, accessToken, boardID, repo string, limit int) (int, int, error) {
	sourceID := "github:" + repo
	if err := s.store.UpsertSource(ctx, core.Source{
		ID:           sourceID,
		Kind:         "github",
		Name:         repo,
		URL:          "https://github.com/" + repo,
		Capabilities: `{"read":true,"write":"github-app"}`,
		Sync:         `{"mode":"oauth-rest"}`,
		UpdatedAt:    time.Now().UTC(),
	}); err != nil {
		return 0, 0, err
	}
	if a := activityFromCtx(ctx); a != nil {
		s.activities.Update(a, 0, 0, "fetching "+repo)
	}
	var issues []githubIssueREST
	path := fmt.Sprintf("/repos/%s/issues?state=all&sort=updated&direction=desc&per_page=%d", repo, limit)
	if err := s.doGitHubREST(ctx, accessToken, path, &issues); err != nil {
		return 0, 0, err
	}
	count := 0
	edgeCount := 0
	for _, issue := range issues {
		node, err := s.upsertGitHubIssueREST(ctx, boardID, repo, issue)
		if err != nil {
			return count, edgeCount, err
		}
		count++
		if a := activityFromCtx(ctx); a != nil {
			s.activities.Update(a, count, len(issues), "")
		}
		for _, edge := range core.ExtractDependencyEdges(repo, node.ID, issue.Body) {
			if _, err := s.store.AddEdgeWithConfidence(ctx, boardID, edge.From, edge.To, edge.Kind, "github-inferred", edge.Confidence, edge); err != nil {
				return count, edgeCount, err
			}
			edgeCount++
		}
	}
	return count, edgeCount, nil
}

func (s *Server) upsertGitHubIssueREST(ctx context.Context, boardID, repo string, issue githubIssueREST) (core.Node, error) {
	sourceID := "github:" + repo
	if err := s.store.UpsertSource(ctx, core.Source{
		ID:           sourceID,
		Kind:         "github",
		Name:         repo,
		URL:          "https://github.com/" + repo,
		Capabilities: `{"read":true,"write":"github-app"}`,
		Sync:         `{"mode":"oauth-rest"}`,
		UpdatedAt:    time.Now().UTC(),
	}); err != nil {
		return core.Node{}, err
	}
	marker := "#"
	kind := "issue"
	if issue.PullRequest.URL != "" {
		marker = "!"
		kind = "pr"
	}
	id := fmt.Sprintf("gh:%s%s%d", repo, marker, issue.Number)
	payload, _ := json.Marshal(map[string]any{
		"source":     "github",
		"kind":       kind,
		"repo":       repo,
		"number":     issue.Number,
		"labels":     issue.LabelNames(),
		"assignees":  issue.AssigneePeople(),
		"author":     issue.AuthorPerson(),
		"milestone":  issue.Milestone.Title,
		"body":       issue.Body,
		"synced_at":  time.Now().UTC().Format(time.RFC3339),
		"html_url":   issue.HTMLURL,
		"api_url":    issue.URL,
		"repository": repo,
		"draft":      issue.Draft,
	})
	node := core.Node{
		ID:        id,
		Kind:      kind,
		Title:     issue.Title,
		State:     strings.ToLower(issue.State),
		Owner:     firstString(issue.AssigneeNames()),
		DataJSON:  string(payload),
		UpdatedAt: parseGitHubRESTTime(issue.UpdatedAt),
	}
	if err := s.store.UpsertNode(ctx, node); err != nil {
		return core.Node{}, err
	}
	if err := s.store.UpsertSourceRef(ctx, id, sourceID, fmt.Sprintf("%s%d", marker, issue.Number), issue.HTMLURL); err != nil {
		return core.Node{}, err
	}
	if err := s.store.AddNodeToBoard(ctx, boardID, id, kind, ""); err != nil {
		return core.Node{}, err
	}
	return node, nil
}

func (s *Server) syncGitHubBoardScope(ctx context.Context, accessToken, login, boardID string, board core.Board, limit int) (int, int, error) {
	if repo := repoForBoard(board); repo != "" {
		return s.syncGitHubRepoBoard(ctx, accessToken, boardID, repo, limit)
	}
	if owner := orgForBoard(board); owner != "" {
		return s.syncGitHubOrgBoard(ctx, accessToken, boardID, owner, limit)
	}
	if board.ScopeQuery == "my-work" || boardPreset(board) == "my-work" {
		return s.syncGitHubMyWorkBoard(ctx, accessToken, login, boardID, limit)
	}
	return 0, 0, errors.New("sync currently supports repo, org, and my-work views")
}

func (s *Server) syncGitHubOrgBoard(ctx context.Context, accessToken, boardID, owner string, limit int) (int, int, error) {
	if a := activityFromCtx(ctx); a != nil {
		s.activities.Update(a, 0, 0, "fetching repos for "+owner)
	}
	var repos []githubRepo
	if err := s.doGitHubREST(ctx, accessToken, fmt.Sprintf("/orgs/%s/repos?sort=updated&direction=desc&per_page=20", owner), &repos); err != nil {
		return 0, 0, err
	}
	totalItems := 0
	totalLinks := 0
	perRepo := limit / maxInt(1, len(repos))
	if perRepo < 5 {
		perRepo = 5
	}
	if perRepo > 30 {
		perRepo = 30
	}
	for _, repo := range repos {
		if repo.FullName == "" {
			continue
		}
		items, links, err := s.syncGitHubRepoBoard(ctx, accessToken, boardID, repo.FullName, perRepo)
		if err != nil {
			return totalItems, totalLinks, err
		}
		totalItems += items
		totalLinks += links
		if a := activityFromCtx(ctx); a != nil {
			s.activities.Update(a, totalItems, 0, "")
		}
	}
	return totalItems, totalLinks, nil
}

func (s *Server) syncGitHubMyWorkBoard(ctx context.Context, accessToken, login, boardID string, limit int) (int, int, error) {
	if login == "" {
		return 0, 0, errors.New("github login is required for my-work sync")
	}
	if a := activityFromCtx(ctx); a != nil {
		s.activities.Update(a, 0, 0, "fetching issues for "+login)
	}
	query := url.QueryEscape(fmt.Sprintf("involves:%s archived:false sort:updated-desc", login))
	var payload struct {
		Items []githubSearchIssue `json:"items"`
	}
	if err := s.doGitHubREST(ctx, accessToken, fmt.Sprintf("/search/issues?q=%s&per_page=%d", query, limit), &payload); err != nil {
		return 0, 0, err
	}
	count := 0
	edges := 0
	for _, item := range payload.Items {
		repo := item.RepoFullName()
		if repo == "" {
			continue
		}
		node, err := s.upsertGitHubIssueREST(ctx, boardID, repo, githubIssueREST{
			Number:      item.Number,
			Title:       item.Title,
			State:       item.State,
			URL:         item.URL,
			HTMLURL:     item.HTMLURL,
			Body:        item.Body,
			UpdatedAt:   item.UpdatedAt,
			PullRequest: item.PullRequest,
		})
		if err != nil {
			return count, edges, err
		}
		count++
		if a := activityFromCtx(ctx); a != nil {
			s.activities.Update(a, count, len(payload.Items), "")
		}
		for _, edge := range core.ExtractDependencyEdges(repo, node.ID, item.Body) {
			if _, err := s.store.AddEdgeWithConfidence(ctx, boardID, edge.From, edge.To, edge.Kind, "github-inferred", edge.Confidence, edge); err != nil {
				return count, edges, err
			}
			edges++
		}
	}
	return count, edges, nil
}

func (s *Server) doJSON(req *http.Request, out any) error {
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 20<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("%s %s: %s", res.Status, req.URL.Path, responseBodySummary(data))
	}
	if err := json.Unmarshal(data, out); err != nil {
		return fmt.Errorf("expected JSON from %s%s, got %s: %w", req.URL.Host, req.URL.Path, responseBodySummary(data), err)
	}
	return nil
}

func responseBodySummary(data []byte) string {
	body := strings.TrimSpace(string(data))
	if body == "" {
		return "empty response"
	}
	var envelope struct {
		Message string `json:"message"`
		Error   string `json:"error"`
	}
	if json.Unmarshal(data, &envelope) == nil {
		if envelope.Message != "" {
			return envelope.Message
		}
		if envelope.Error != "" {
			return envelope.Error
		}
	}
	prefixLen := len(body)
	if prefixLen > 120 {
		prefixLen = 120
	}
	if strings.Contains(strings.ToLower(body[:prefixLen]), "<html") || strings.HasPrefix(strings.ToLower(body), "<!doctype") {
		return "non-JSON HTML response"
	}
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 240 {
		body = body[:240] + "..."
	}
	return body
}

func (s *Server) callbackURL() string {
	base := strings.TrimRight(s.cfg.BaseURL, "/")
	if base == "" {
		base = "http://" + s.cfg.Addr
	}
	return base + "/api/auth/github/callback"
}

func (s *Server) githubOAuthConfigured() bool {
	return s.cfg.GitHubClientID != "" && s.cfg.GitHubClientSecret != ""
}

func (s *Server) githubAppConfigured() bool {
	return s.cfg.GitHubAppID != "" && s.cfg.GitHubClientID != "" && s.cfg.GitHubClientSecret != "" && s.cfg.GitHubAppPrivateKeyFile != ""
}

func (s *Server) githubWebhookConfigured() bool {
	return s.githubAppConfigured() && s.cfg.GitHubWebhookSecret != ""
}

type githubToken struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (g githubToken) Scopes() []string {
	if g.Scope == "" {
		return nil
	}
	parts := strings.Split(g.Scope, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part != "" {
			out = append(out, part)
		}
	}
	return out
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
}

type githubRepo struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	FullName  string `json:"full_name"`
	Private   bool   `json:"private"`
	HTMLURL   string `json:"html_url"`
	UpdatedAt string `json:"updated_at"`
	Owner     struct {
		ID    int64  `json:"id"`
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"owner"`
}

type githubIssueREST struct {
	ID        int64  `json:"id"`
	Number    int    `json:"number"`
	Title     string `json:"title"`
	State     string `json:"state"`
	URL       string `json:"url"`
	HTMLURL   string `json:"html_url"`
	Body      string `json:"body"`
	UpdatedAt string `json:"updated_at"`
	Labels    []struct {
		Name string `json:"name"`
	} `json:"labels"`
	Assignees []struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"assignees"`
	User struct {
		Login     string `json:"login"`
		AvatarURL string `json:"avatar_url"`
		HTMLURL   string `json:"html_url"`
	} `json:"user"`
	Milestone struct {
		Title string `json:"title"`
	} `json:"milestone"`
	PullRequest githubPullRequestRef `json:"pull_request"`
	Draft       bool                 `json:"draft"`
}

type githubPullRequestRef struct {
	URL     string `json:"url"`
	HTMLURL string `json:"html_url"`
}

type githubSearchIssue struct {
	Number        int                  `json:"number"`
	Title         string               `json:"title"`
	State         string               `json:"state"`
	URL           string               `json:"url"`
	HTMLURL       string               `json:"html_url"`
	Body          string               `json:"body"`
	UpdatedAt     string               `json:"updated_at"`
	RepositoryURL string               `json:"repository_url"`
	PullRequest   githubPullRequestRef `json:"pull_request"`
}

func (g githubSearchIssue) RepoFullName() string {
	return strings.TrimPrefix(g.RepositoryURL, "https://api.github.com/repos/")
}

func (g githubIssueREST) LabelNames() []string {
	out := make([]string, 0, len(g.Labels))
	for _, label := range g.Labels {
		if label.Name != "" {
			out = append(out, label.Name)
		}
	}
	return out
}

func (g githubIssueREST) AssigneeNames() []string {
	out := make([]string, 0, len(g.Assignees))
	for _, assignee := range g.Assignees {
		if assignee.Login != "" {
			out = append(out, assignee.Login)
		}
	}
	return out
}

func (g githubIssueREST) AssigneePeople() []map[string]string {
	out := make([]map[string]string, 0, len(g.Assignees))
	for _, assignee := range g.Assignees {
		if assignee.Login != "" {
			out = append(out, map[string]string{
				"login":      assignee.Login,
				"avatar_url": assignee.AvatarURL,
				"html_url":   assignee.HTMLURL,
			})
		}
	}
	return out
}

func (g githubIssueREST) AuthorPerson() map[string]string {
	if g.User.Login == "" {
		return nil
	}
	return map[string]string{
		"login":      g.User.Login,
		"avatar_url": g.User.AvatarURL,
		"html_url":   g.User.HTMLURL,
	}
}

type githubOrg struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	HTMLURL   string `json:"html_url"`
	AvatarURL string `json:"avatar_url"`
}

type githubProject struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	UpdatedAt string `json:"updated_at"`
	Owner     string `json:"owner"`
}

type githubProjectsGraphQL struct {
	Viewer struct {
		ProjectsV2 struct {
			Nodes []githubProject `json:"nodes"`
		} `json:"projectsV2"`
		Organizations struct {
			Nodes []struct {
				Login      string `json:"login"`
				ProjectsV2 struct {
					Nodes []githubProject `json:"nodes"`
				} `json:"projectsV2"`
			} `json:"nodes"`
		} `json:"organizations"`
	} `json:"viewer"`
}

func (s *Server) handleCreateGitHubIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	var in struct {
		BoardID      string   `json:"board_id"`
		NodeID       string   `json:"node_id"`
		Repo         string   `json:"repo"`
		Title        string   `json:"title"`
		Body         string   `json:"body"`
		Labels       []string `json:"labels"`
		Assignees    []string `json:"assignees"`
		Milestone    int      `json:"milestone"`
		ArchiveLocal bool     `json:"archive_local"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if in.Repo == "" || in.Title == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo and title are required"})
		return
	}
	parts := strings.SplitN(in.Repo, "/", 2)
	if len(parts) != 2 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo must be owner/repo"})
		return
	}
	token, _, err := s.githubTokenForBoardSync(r.Context(), account.ID, core.Board{})
	if err != nil || token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no github token available; sign in first"})
		return
	}
	issueBody := map[string]any{"title": in.Title, "body": in.Body}
	if len(in.Labels) > 0 {
		issueBody["labels"] = in.Labels
	}
	if len(in.Assignees) > 0 {
		issueBody["assignees"] = in.Assignees
	}
	if in.Milestone > 0 {
		issueBody["milestone"] = in.Milestone
	}
	payload, _ := json.Marshal(issueBody)
	act := s.activities.Start("github-write", "Creating GitHub issue")
	apiURL := "https://api.github.com/repos/" + parts[0] + "/" + parts[1] + "/issues"
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		s.activities.Fail(act, err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.activities.Fail(act, err.Error())
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var ghResult map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&ghResult)
	if resp.StatusCode >= 300 {
		errMsg, _ := ghResult["message"].(string)
		var friendlyErr string
		switch resp.StatusCode {
		case 401:
			friendlyErr = "github: unauthorized - token may be expired or invalid"
		case 403:
			friendlyErr = "github: forbidden - token lacks write permission for this repository"
		case 404:
			friendlyErr = "github: repository not found or no access"
		case 429:
			friendlyErr = "github: rate limit exceeded"
		default:
			friendlyErr = "github: " + errMsg
		}
		s.activities.Fail(act, friendlyErr)
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": friendlyErr})
		return
	}
	issueURL, _ := ghResult["html_url"].(string)
	issueNumber := fmt.Sprint(ghResult["number"])
	issueNumber = strings.TrimSuffix(issueNumber, ".0")
	boardID := strings.TrimSpace(in.BoardID)
	if boardID == "" {
		boardID = core.DefaultBoardID
	}
	node, err := s.store.AddGitHubRefToBoard(r.Context(), boardID, in.Repo, "#", issueNumber, in.Title)
	if err != nil {
		s.activities.Fail(act, "depviz import failed: "+err.Error())
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github issue created, but depviz import failed: " + err.Error()})
		return
	}
	s.activities.Finish(act, 1, 1, "Created #"+issueNumber)
	if strings.TrimSpace(in.NodeID) != "" {
		_, _ = s.store.AddEdge(r.Context(), boardID, strings.TrimSpace(in.NodeID), node.ID, "addresses", "user", map[string]any{"source": "github-create-issue"})
	}
	if in.ArchiveLocal && strings.TrimSpace(in.NodeID) != "" {
		_ = s.store.ArchiveNode(r.Context(), strings.TrimSpace(in.NodeID))
	}
	writeJSON(w, http.StatusOK, map[string]any{"url": issueURL, "number": issueNumber, "node": node})
}

func (s *Server) handleUpdateGitHubIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	var in struct {
		Repo        string   `json:"repo"`
		IssueNumber int      `json:"issue_number"`
		Title       string   `json:"title"`
		Body        string   `json:"body"`
		State       string   `json:"state"`
		Labels      []string `json:"labels"`
		Assignees   []string `json:"assignees"`
		Milestone   int      `json:"milestone"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if in.Repo == "" || in.IssueNumber == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo and issue_number are required"})
		return
	}
	token, _, err := s.githubTokenForBoardSync(r.Context(), account.ID, core.Board{})
	if err != nil || token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no github token available; sign in first"})
		return
	}
	patchBody := map[string]any{}
	if in.Title != "" {
		patchBody["title"] = in.Title
	}
	if in.Body != "" {
		patchBody["body"] = in.Body
	}
	if in.State != "" {
		patchBody["state"] = in.State
	}
	if len(in.Labels) > 0 {
		patchBody["labels"] = in.Labels
	}
	if len(in.Assignees) > 0 {
		patchBody["assignees"] = in.Assignees
	}
	payload, _ := json.Marshal(patchBody)
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d", in.Repo, in.IssueNumber)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPatch, apiURL, bytes.NewReader(payload))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var ghResult map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&ghResult)
	if resp.StatusCode >= 300 {
		errMsg, _ := ghResult["message"].(string)
		switch resp.StatusCode {
		case 401:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: unauthorized - token may be expired or invalid"})
		case 403:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: forbidden - token lacks write permission for this repository"})
		case 404:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: repository not found or no access"})
		case 429:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: rate limit exceeded"})
		default:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: " + errMsg})
		}
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "issue": ghResult})
}

func (s *Server) handleCreateGitHubComment(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	var in struct {
		Repo        string `json:"repo"`
		IssueNumber int    `json:"issue_number"`
		Body        string `json:"body"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if in.Repo == "" || in.IssueNumber == 0 || strings.TrimSpace(in.Body) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "repo, issue_number, and body are required"})
		return
	}
	token, _, err := s.githubTokenForBoardSync(r.Context(), account.ID, core.Board{})
	if err != nil || token == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "no github token available; sign in first"})
		return
	}
	payload, _ := json.Marshal(map[string]string{"body": in.Body})
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/issues/%d/comments", in.Repo, in.IssueNumber)
	req, err := http.NewRequestWithContext(r.Context(), http.MethodPost, apiURL, bytes.NewReader(payload))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github+json")
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()
	var ghResult map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&ghResult)
	if resp.StatusCode >= 300 {
		errMsg, _ := ghResult["message"].(string)
		switch resp.StatusCode {
		case 401:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: unauthorized - token may be expired or invalid"})
		case 403:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: forbidden - token lacks write permission for this repository"})
		case 404:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: repository not found or no access"})
		case 429:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: rate limit exceeded"})
		default:
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "github: " + errMsg})
		}
		return
	}
	commentURL, _ := ghResult["html_url"].(string)
	commentID := fmt.Sprint(ghResult["id"])
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "url": commentURL, "id": commentID})
}

func (s *Server) handleBoardSourceApply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	var in struct {
		BoardID string `json:"board_id"`
		DryRun  bool   `json:"dry_run"`
		Creates []struct {
			Kind        string   `json:"kind"`
			Title       string   `json:"title"`
			Status      string   `json:"status"`
			Owner       string   `json:"owner"`
			Description string   `json:"description"`
			TimeHorizon string   `json:"time_horizon"`
			Priority    string   `json:"priority"`
			Labels      []string `json:"labels"`
		} `json:"creates"`
		Updates []struct {
			NodeID      string `json:"node_id"`
			Title       string `json:"title"`
			Status      string `json:"status"`
			Owner       string `json:"owner"`
			Description string `json:"description"`
		} `json:"updates"`
		Deletes []struct {
			NodeID string `json:"node_id"`
		} `json:"deletes"`
		LinkCreates []struct {
			FromID string `json:"from_id"`
			ToID   string `json:"to_id"`
			Kind   string `json:"kind"`
			Notes  string `json:"notes"`
		} `json:"link_creates"`
		LinkDeletes []struct {
			EdgeID string `json:"edge_id"`
		} `json:"link_deletes"`
		BaseUpdatedAt string `json:"base_updated_at"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	boardID := strings.TrimSpace(in.BoardID)
	if boardID == "" {
		boardID = core.DefaultBoardID
	}
	if !s.requireBoardAccess(w, r, boardID, account) {
		return
	}
	if strings.TrimSpace(in.BaseUpdatedAt) != "" {
		serverUpdatedAt, err := s.store.BoardUpdatedAt(r.Context(), boardID)
		if err == nil {
			baseTime, parseErr := time.Parse(time.RFC3339, in.BaseUpdatedAt)
			if parseErr == nil && serverUpdatedAt.After(baseTime) {
				writeJSON(w, http.StatusConflict, map[string]any{
					"error":             "conflict",
					"server_updated_at": serverUpdatedAt.UTC().Format(time.RFC3339),
				})
				return
			}
		}
	}
	total := len(in.Creates) + len(in.Updates) + len(in.Deletes) + len(in.LinkCreates) + len(in.LinkDeletes)
	if total > 100 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "too many operations (max 100)"})
		return
	}
	var errs []string
	for _, c := range in.Creates {
		if strings.TrimSpace(c.Title) == "" {
			errs = append(errs, "create: title is required")
		}
	}
	snap, err := s.store.Snapshot(r.Context(), boardID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	nodes := map[string]bool{}
	for _, node := range snap.Nodes {
		nodes[node.ID] = true
	}
	edges := map[string]bool{}
	for _, edge := range snap.Edges {
		edges[edgeSelectionID(edge)] = true
	}
	for _, u := range in.Updates {
		nodeID := strings.TrimSpace(u.NodeID)
		if nodeID == "" {
			errs = append(errs, "update: node_id is required")
			continue
		}
		if !nodes[nodeID] {
			errs = append(errs, "update: node not found: "+nodeID)
		}
		if strings.TrimSpace(u.Title) == "" {
			errs = append(errs, "update: title is required for "+nodeID)
		}
		if strings.TrimSpace(u.Status) == "" {
			errs = append(errs, "update: status is required for "+nodeID)
		}
	}
	for _, d := range in.Deletes {
		nodeID := strings.TrimSpace(d.NodeID)
		if nodeID == "" {
			errs = append(errs, "delete: node_id is required")
			continue
		}
		if !nodes[nodeID] {
			errs = append(errs, "delete: node not found: "+nodeID)
		}
		if !isLocalBoardNodeID(nodeID) {
			errs = append(errs, "delete: refusing to remove GitHub-backed node: "+nodeID)
		}
	}
	for _, ld := range in.LinkDeletes {
		edgeID := strings.TrimSpace(ld.EdgeID)
		if edgeID == "" {
			errs = append(errs, "link delete: edge_id is required")
			continue
		}
		if !edges[edgeID] {
			errs = append(errs, "link delete: edge not found: "+edgeID)
		}
	}
	for _, lc := range in.LinkCreates {
		from := strings.TrimSpace(lc.FromID)
		to := strings.TrimSpace(lc.ToID)
		if from == "" || to == "" {
			errs = append(errs, "link create: from_id and to_id are required")
			continue
		}
		if !nodes[from] {
			errs = append(errs, "link create: from node not found: "+from)
		}
		if !nodes[to] {
			errs = append(errs, "link create: to node not found: "+to)
		}
	}
	if len(errs) > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]any{"errors": errs})
		return
	}
	summary := map[string]int{
		"created":       len(in.Creates),
		"updated":       len(in.Updates),
		"deleted":       len(in.Deletes),
		"links_added":   len(in.LinkCreates),
		"links_removed": len(in.LinkDeletes),
	}
	if in.DryRun {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "summary": summary, "errors": []string{}})
		return
	}
	patch := core.BoardSourcePatch{}
	for _, c := range in.Creates {
		patch.Creates = append(patch.Creates, core.BoardSourceCreate{
			Kind:        c.Kind,
			Title:       c.Title,
			Status:      c.Status,
			Owner:       c.Owner,
			Description: c.Description,
			TimeHorizon: c.TimeHorizon,
			Priority:    c.Priority,
			Labels:      c.Labels,
		})
	}
	for _, u := range in.Updates {
		patch.Updates = append(patch.Updates, core.BoardSourceUpdate{
			NodeID:      u.NodeID,
			Title:       u.Title,
			Status:      u.Status,
			Owner:       u.Owner,
			Description: u.Description,
		})
	}
	for _, d := range in.Deletes {
		patch.Deletes = append(patch.Deletes, core.BoardSourceDelete{NodeID: d.NodeID})
	}
	for _, lc := range in.LinkCreates {
		patch.LinkCreates = append(patch.LinkCreates, core.BoardSourceLinkCreate{
			FromID: lc.FromID,
			ToID:   lc.ToID,
			Kind:   lc.Kind,
			Notes:  lc.Notes,
		})
	}
	for _, ld := range in.LinkDeletes {
		patch.LinkDeletes = append(patch.LinkDeletes, core.BoardSourceLinkDelete{EdgeID: ld.EdgeID})
	}
	act := s.activities.Start("patch-apply", "Applying source patch")
	if err := s.store.ApplyBoardSourcePatch(r.Context(), boardID, patch); err != nil {
		s.activities.Fail(act, err.Error())
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	s.activities.Finish(act, total, total, fmt.Sprintf("%d ops applied", total))
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "summary": summary, "errors": []string{}})
}

func (s *Server) handleBoardViews(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAccount(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		boardID := r.URL.Query().Get("board_id")
		if boardID == "" {
			boardID = core.DefaultBoardID
		}
		views, err := s.store.ListBoardViews(r.Context(), boardID)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if views == nil {
			views = []core.BoardView{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"views": views})
	case http.MethodPost:
		var in struct {
			BoardID    string         `json:"board_id"`
			Name       string         `json:"name"`
			Visibility string         `json:"visibility"`
			Config     map[string]any `json:"config"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		if strings.TrimSpace(in.Name) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name is required"})
			return
		}
		boardID := strings.TrimSpace(in.BoardID)
		if boardID == "" {
			boardID = core.DefaultBoardID
		}
		view, err := s.store.SaveBoardView(r.Context(), boardID, in.Name, in.Visibility, in.Config)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"view": view})
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
			return
		}
		if err := s.store.DeleteBoardView(r.Context(), id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		w.Header().Set("Allow", "GET, POST, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleBoardSyncLogs(w http.ResponseWriter, r *http.Request) {
	if _, ok := s.requireAccount(w, r); !ok {
		return
	}
	switch r.Method {
	case http.MethodGet:
		boardID := r.URL.Query().Get("board_id")
		if boardID == "" {
			boardID = core.DefaultBoardID
		}
		logs, err := s.store.GetSyncLogs(r.Context(), boardID, 20)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		if logs == nil {
			logs = []core.SyncLog{}
		}
		writeJSON(w, http.StatusOK, map[string]any{"logs": logs})
	case http.MethodDelete:
		id := r.URL.Query().Get("id")
		if id == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "id is required"})
			return
		}
		if err := s.store.CancelSyncLog(r.Context(), id); err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		w.Header().Set("Allow", "GET, DELETE")
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func (s *Server) handleDismissSuggestion(w http.ResponseWriter, r *http.Request) {
	account, ok := s.requireAccount(w, r)
	if !ok {
		return
	}
	switch r.Method {
	case http.MethodPost:
		var in struct {
			EdgeID  string `json:"edge_id"`
			BoardID string `json:"board_id"`
			Restore bool   `json:"restore"`
		}
		if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&in); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
		boardID := strings.TrimSpace(in.BoardID)
		if boardID == "" {
			boardID = core.DefaultBoardID
		}
		edgeID := strings.TrimSpace(in.EdgeID)
		if edgeID == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": "edge_id is required"})
			return
		}
		if in.Restore {
			if err := s.store.RestoreSuggestion(r.Context(), account.ID, boardID, edgeID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		} else {
			if err := s.store.DismissSuggestion(r.Context(), account.ID, boardID, edgeID); err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
		}
		writeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	default:
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
	}
}

func edgeSelectionID(edge core.Edge) string {
	if edge.ID != "" {
		return edge.ID
	}
	kind := strings.ToLower(strings.TrimSpace(edge.Kind))
	if kind == "blocked_by" || kind == "depends_on" || kind == "depends" || kind == "after" {
		return "blocking:" + edge.FromID + "->" + edge.ToID
	}
	if kind == "blocks" || kind == "unblocks" || kind == "precedes" {
		return "blocking:" + edge.ToID + "->" + edge.FromID
	}
	return kind + ":" + edge.FromID + "->" + edge.ToID
}

func isLocalBoardNodeID(nodeID string) bool {
	for _, prefix := range []string{"note:", "task:", "strategy:", "initiative:", "bet:", "project:", "workstream:", "risk:", "decision:", "question:", "metric:"} {
		if strings.HasPrefix(nodeID, prefix) {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func safeReturnPath(value string) string {
	if value == "" {
		return "/"
	}
	u, err := url.Parse(value)
	if err != nil || u.IsAbs() || !strings.HasPrefix(value, "/") {
		return "/"
	}
	return value
}

type parsedGitHubRef struct {
	repo   string
	marker string
	number string
}

var (
	githubShortRefRE = regexp.MustCompile(`^(?:gh:)?([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)([#!])([0-9]+)$`)
	githubURLRefRE   = regexp.MustCompile(`^https://github\.com/([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)/(issues|pull)/([0-9]+)(?:[/?#].*)?$`)
	githubBareRefRE  = regexp.MustCompile(`^([#!])([0-9]+)$`)
)

func parseGitHubRef(value string) (parsedGitHubRef, bool) {
	value = strings.TrimSpace(value)
	if m := githubShortRefRE.FindStringSubmatch(value); m != nil {
		return parsedGitHubRef{repo: m[1], marker: m[2], number: m[3]}, true
	}
	if m := githubURLRefRE.FindStringSubmatch(value); m != nil {
		marker := "#"
		if m[2] == "pull" {
			marker = "!"
		}
		return parsedGitHubRef{repo: m[1], marker: marker, number: m[3]}, true
	}
	return parsedGitHubRef{}, false
}

func resolveBoardNodeRef(snap core.Snapshot, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("node reference is required")
	}
	if gh, ok := parseGitHubRef(value); ok {
		return "gh:" + gh.repo + gh.marker + gh.number, nil
	}
	if m := githubBareRefRE.FindStringSubmatch(value); m != nil {
		repo := repoForBoard(snap.Board)
		if repo == "" {
			return "", errors.New("short GitHub refs require a repo-scoped view")
		}
		return "gh:" + repo + m[1] + m[2], nil
	}
	lower := strings.ToLower(value)
	for _, node := range snap.Nodes {
		if node.ID == value || strings.ToLower(node.Title) == lower {
			return node.ID, nil
		}
	}
	return value, nil
}

func repoForBoard(board core.Board) string {
	if strings.HasPrefix(board.ScopeQuery, "repo:") {
		return strings.TrimSpace(strings.TrimPrefix(board.ScopeQuery, "repo:"))
	}
	var cfg struct {
		Repo string `json:"repo"`
	}
	if board.ConfigJSON != "" && json.Unmarshal([]byte(board.ConfigJSON), &cfg) == nil {
		return strings.TrimSpace(cfg.Repo)
	}
	return ""
}

func orgForBoard(board core.Board) string {
	if strings.HasPrefix(board.ScopeQuery, "org:") {
		return strings.TrimSpace(strings.TrimPrefix(board.ScopeQuery, "org:"))
	}
	var cfg struct {
		Owner string `json:"owner"`
	}
	if board.ConfigJSON != "" && json.Unmarshal([]byte(board.ConfigJSON), &cfg) == nil {
		if repoForBoard(board) == "" {
			return strings.TrimSpace(cfg.Owner)
		}
	}
	return ""
}

func boardPreset(board core.Board) string {
	var cfg struct {
		Preset string `json:"preset"`
	}
	if board.ConfigJSON != "" && json.Unmarshal([]byte(board.ConfigJSON), &cfg) == nil {
		return strings.TrimSpace(cfg.Preset)
	}
	return ""
}

func boardScopeLabel(board core.Board) string {
	if repo := repoForBoard(board); repo != "" {
		return repo
	}
	if org := orgForBoard(board); org != "" {
		return org
	}
	if board.ScopeQuery == "my-work" || boardPreset(board) == "my-work" {
		return "my work"
	}
	if board.Name != "" {
		return board.Name
	}
	return board.ScopeQuery
}

func parseGitHubRESTTime(value string) time.Time {
	t, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return time.Now().UTC().Truncate(time.Second)
	}
	return t.UTC().Truncate(time.Second)
}

func firstString(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
