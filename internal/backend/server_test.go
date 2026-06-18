package backend

import (
	"context"
	"encoding/json"
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

func contains(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}
