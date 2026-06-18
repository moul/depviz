package backend

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
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
