package backend

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"moul.io/depviz/v4/internal/core"
)

func TestHealthAndAnonymousSession(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, Config{Addr: "127.0.0.1:0", BaseURL: "https://depviz.example"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	res, err := http.Get(ts.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var health struct {
		OK                    bool `json:"ok"`
		GitHubOAuthConfigured bool `json:"github_oauth_configured"`
	}
	if err := json.NewDecoder(res.Body).Decode(&health); err != nil {
		t.Fatal(err)
	}
	if !health.OK {
		t.Fatal("health ok=false")
	}
	if health.GitHubOAuthConfigured {
		t.Fatal("github oauth should not be configured")
	}

	res, err = http.Get(ts.URL + "/api/session")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	var session struct {
		Authenticated bool `json:"authenticated"`
	}
	if err := json.NewDecoder(res.Body).Decode(&session); err != nil {
		t.Fatal(err)
	}
	if session.Authenticated {
		t.Fatal("anonymous session is authenticated")
	}

	res, err = http.Get(ts.URL + "/api/export")
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous export status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
}

func TestAuthenticatedExport(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	account, err := store.UpsertOAuthAccount(ctx, core.OAuthAccountInput{
		Provider:   "github",
		ExternalID: "42",
		Login:      "moul",
	})
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.CreateWebSession(ctx, account.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(store, Config{Addr: "127.0.0.1:0", BaseURL: "https://depviz.example"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	req, err := http.NewRequest(http.MethodGet, ts.URL+"/api/export", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	var payload struct {
		Snapshot struct {
			Board struct {
				ID string `json:"id"`
			} `json:"board"`
		} `json:"snapshot"`
	}
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.Snapshot.Board.ID != core.DefaultBoardID {
		t.Fatalf("board id = %q, want %q", payload.Snapshot.Board.ID, core.DefaultBoardID)
	}
}

func TestAuthenticatedBoards(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	account, err := store.UpsertOAuthAccount(ctx, core.OAuthAccountInput{
		Provider:   "github",
		ExternalID: "42",
		Login:      "moul",
	})
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.CreateWebSession(ctx, account.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(store, Config{})

	req := httptest.NewRequest(http.MethodPost, "/api/boards", strings.NewReader(`{"name":"Roadmap","description":"Product graph","preset":"repo","provider":"github","repo":"moul/depviz"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("create status = %d, want %d body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	var created struct {
		Board core.Board `json:"board"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Board.ID != "roadmap" {
		t.Fatalf("board id = %q, want roadmap", created.Board.ID)
	}
	if created.Board.ScopeQuery != "repo:moul/depviz" {
		t.Fatalf("scope = %q, want repo:moul/depviz", created.Board.ScopeQuery)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/boards", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list status = %d, want %d", rec.Code, http.StatusOK)
	}
	var listed struct {
		Boards []core.Board `json:"boards"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed.Boards) != 2 {
		t.Fatalf("boards = %d, want default plus created", len(listed.Boards))
	}
	if listed.Boards[0].Metrics == nil {
		t.Fatalf("board metrics missing")
	}

	req = httptest.NewRequest(http.MethodPost, "/api/board-items", strings.NewReader(`{"board_id":"roadmap","kind":"github","ref":"https://github.com/moul/depviz/issues/691","title":"Backend work"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("github item status = %d, want %d body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/board-items", strings.NewReader(`{"board_id":"roadmap","kind":"task","title":"Write daemon sync"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("task item status = %d, want %d body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	req = httptest.NewRequest(http.MethodPost, "/api/board-links", strings.NewReader(`{"board_id":"roadmap","from":"#691","to":"Write daemon sync","kind":"blocked_by"}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("link status = %d, want %d body=%s", rec.Code, http.StatusCreated, rec.Body.String())
	}
	payload, err := store.BuildExport(ctx, "roadmap")
	if err != nil {
		t.Fatal(err)
	}
	if len(payload.Snapshot.Nodes) != 2 {
		t.Fatalf("nodes = %d, want 2", len(payload.Snapshot.Nodes))
	}
	if len(payload.Snapshot.Edges) != 1 {
		t.Fatalf("edges = %d, want 1", len(payload.Snapshot.Edges))
	}
	boards, err := store.BoardList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var roadmap core.Board
	for _, board := range boards {
		if board.ID == "roadmap" {
			roadmap = board
		}
	}
	if roadmap.Metrics == nil || roadmap.Metrics.Items != 2 || roadmap.Metrics.Links != 1 {
		t.Fatalf("roadmap metrics = %+v, want 2 items and 1 link", roadmap.Metrics)
	}
	if err := store.RecordBoardSync(ctx, "roadmap", "ok", map[string]any{"items": 2, "links": 1}); err != nil {
		t.Fatal(err)
	}
	boards, err = store.BoardList(ctx)
	if err != nil {
		t.Fatal(err)
	}
	for _, board := range boards {
		if board.ID == "roadmap" {
			roadmap = board
		}
	}
	if roadmap.Metrics == nil || roadmap.Metrics.SyncStatus != "ok" || roadmap.Metrics.LastSyncAt.IsZero() {
		t.Fatalf("sync metrics = %+v, want ok sync status", roadmap.Metrics)
	}
}

func TestGitHubDiscoveryRequiresConnectedOAuth(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	account, err := store.UpsertOAuthAccount(ctx, core.OAuthAccountInput{
		Provider:   "github",
		ExternalID: "42",
		Login:      "moul",
		TokenJSON:  `{}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.CreateWebSession(ctx, account.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(store, Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/github/repos", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestGitHubStartRequiresConfig(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, Config{})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/github/start", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusServiceUnavailable)
	}
}

func TestGitHubStartRequestsDiscoveryScopes(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, Config{
		BaseURL:            "https://depviz.example",
		GitHubClientID:     "client-id",
		GitHubClientSecret: "client-secret",
	})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/github/start?return_to=/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	scopes := strings.Fields(location.Query().Get("scope"))
	for _, want := range []string{"repo", "read:user", "user:email", "read:org", "read:project"} {
		if !contains(scopes, want) {
			t.Fatalf("scopes = %v, missing %q", scopes, want)
		}
	}
}

func TestGitHubAppStartOmitsOAuthScopes(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, Config{
		BaseURL:                 "https://depviz.example",
		GitHubClientID:          "client-id",
		GitHubClientSecret:      "client-secret",
		GitHubAppID:             "123",
		GitHubAppPrivateKeyFile: "/tmp/depviz-test-key.pem",
	})
	req := httptest.NewRequest(http.MethodGet, "/api/auth/github/start?return_to=/", nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusFound {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusFound)
	}
	location, err := url.Parse(rec.Header().Get("Location"))
	if err != nil {
		t.Fatal(err)
	}
	if scope := location.Query().Get("scope"); scope != "" {
		t.Fatalf("scope = %q, want empty for GitHub App auth", scope)
	}
}

func TestGitHubPublicSyncFallbackOnlyForOAuthRepoOrOrg(t *testing.T) {
	repoBoard := core.Board{ScopeQuery: "repo:gnolang/gno"}
	if !canRetryGitHubPublicSync(errors.New("401 Unauthorized: Bad credentials"), "github-oauth-user", repoBoard) {
		t.Fatal("expected public retry for repo view with bad OAuth token")
	}
	if canRetryGitHubPublicSync(errors.New("401 Unauthorized: Bad credentials"), "github-app-installation", repoBoard) {
		t.Fatal("did not expect public retry for installation token failures")
	}
	if canRetryGitHubPublicSync(errors.New("403 Forbidden"), "github-oauth-user", repoBoard) {
		t.Fatal("did not expect public retry for non-401 failures")
	}
	if canRetryGitHubPublicSync(errors.New("401 Unauthorized: Bad credentials"), "github-oauth-user", core.Board{ScopeQuery: "my-work"}) {
		t.Fatal("did not expect public retry for personal views")
	}
}

func TestGitHubWebhookSignature(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	body := []byte(`{"zen":"Non-blocking is better than blocking.","installation":{"id":42,"target_type":"User","repository_selection":"all","account":{"id":7,"login":"moul","type":"User","html_url":"https://github.com/moul"}}}`)
	secret := "test-secret"
	srv := NewServer(store, Config{
		GitHubClientID:          "client-id",
		GitHubClientSecret:      "client-secret",
		GitHubAppID:             "123",
		GitHubAppPrivateKeyFile: "/tmp/depviz-test-key.pem",
		GitHubWebhookSecret:     secret,
	})
	req := httptest.NewRequest(http.MethodPost, "/api/github/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "ping")
	req.Header.Set("X-Hub-Signature-256", "sha256=bad")
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("bad signature status = %d, want %d", rec.Code, http.StatusUnauthorized)
	}

	req = httptest.NewRequest(http.MethodPost, "/api/github/webhook", strings.NewReader(string(body)))
	req.Header.Set("X-GitHub-Event", "ping")
	req.Header.Set("X-Hub-Signature-256", testSignature(body, secret))
	rec = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("good signature status = %d, want %d body=%s", rec.Code, http.StatusOK, rec.Body.String())
	}
}

func TestLogoutClearsSessionCookie(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	account, err := store.UpsertOAuthAccount(ctx, core.OAuthAccountInput{
		Provider:   "github",
		ExternalID: "42",
		Login:      "moul",
	})
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := store.CreateWebSession(ctx, account.ID, 0)
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(store, Config{})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	if _, ok, err := store.AccountForWebSession(ctx, token); err != nil || ok {
		t.Fatalf("session after logout ok=%v err=%v, want false nil", ok, err)
	}
}

func testSignature(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
