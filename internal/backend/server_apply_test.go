package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"moul.io/depviz/v4/internal/core"
)

func TestHandleBoardSourceApplyAnonymous(t *testing.T) {
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	srv := NewServer(store, Config{Addr: "127.0.0.1:0", BaseURL: "https://depviz.example"})
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	body, _ := json.Marshal(map[string]any{"board_id": "default", "dry_run": true})
	res, err := http.Post(ts.URL+"/api/board-source/apply", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous apply status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
}

func TestHandleBoardSourceApplyDryRun(t *testing.T) {
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

	payload := map[string]any{
		"board_id": "default",
		"dry_run":  true,
		"creates": []map[string]any{
			{"kind": "task", "title": "Test task", "status": "todo"},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/board-source/apply", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("dry_run apply status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	var out struct {
		OK      bool           `json:"ok"`
		Summary map[string]int `json:"summary"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !out.OK {
		t.Error("dry_run apply ok=false")
	}
	if out.Summary["created"] != 1 {
		t.Errorf("summary.created = %d, want 1", out.Summary["created"])
	}
}

func TestHandleBoardSourceApplyCommit(t *testing.T) {
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

	payload := map[string]any{
		"board_id": "default",
		"dry_run":  false,
		"creates": []map[string]any{
			{"kind": "task", "title": "Test task commit", "status": "todo"},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/board-source/apply", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		t.Fatalf("apply status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	var out struct {
		OK bool `json:"ok"`
	}
	if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if !out.OK {
		t.Error("apply ok=false")
	}
}

func TestHandleBoardSourceApplyMissingBoard(t *testing.T) {
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

	payload := map[string]any{
		"board_id": "missing-board",
		"dry_run":  false,
		"creates": []map[string]any{
			{"kind": "task", "title": "Test task commit", "status": "todo"},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/board-source/apply", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: token})

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("missing board apply status = %d, want %d", res.StatusCode, http.StatusNotFound)
	}
}
