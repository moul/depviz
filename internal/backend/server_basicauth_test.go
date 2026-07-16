package backend

import (
	"context"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"moul.io/depviz/v4/internal/core"
)

func newBasicAuthTestServer(t *testing.T, cfg Config) *httptest.Server {
	t.Helper()
	ctx := context.Background()
	store, err := core.OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = store.Close() })
	cfg.Addr = "127.0.0.1:0"
	cfg.BaseURL = "https://depviz.example"
	ts := httptest.NewServer(NewServer(store, cfg).Handler())
	t.Cleanup(ts.Close)
	return ts
}

// get issues a request without following the redirect the SPA handler may emit.
func get(t *testing.T, url, user, pass string) *http.Response {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	if user != "" || pass != "" {
		req.SetBasicAuth(user, pass)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = res.Body.Close() })
	return res
}

func TestBasicAuthGate(t *testing.T) {
	gated := Config{BasicAuthUser: "demo", BasicAuthPass: "s3cret"}

	tests := []struct {
		name       string
		cfg        Config
		path       string
		user, pass string
		want       int
	}{
		// Unset config must not change today's behavior.
		{"unset config leaves SPA open", Config{}, "/", "", "", http.StatusOK},
		{"unset config leaves health open", Config{}, "/api/health", "", "", http.StatusOK},

		{"gated SPA rejects anonymous", gated, "/", "", "", http.StatusUnauthorized},
		{"gated API rejects anonymous", gated, "/api/export", "", "", http.StatusUnauthorized},
		{"gated rejects wrong password", gated, "/", "demo", "wrong", http.StatusUnauthorized},
		{"gated rejects wrong user", gated, "/", "nope", "s3cret", http.StatusUnauthorized},
		{"gated allows correct credentials", gated, "/", "demo", "s3cret", http.StatusOK},
		// Health stays open so deploy checks keep working; it exposes no board data.
		{"gated leaves health open", gated, "/api/health", "", "", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := newBasicAuthTestServer(t, tt.cfg)
			res := get(t, ts.URL+tt.path, tt.user, tt.pass)
			if res.StatusCode != tt.want {
				t.Fatalf("GET %s = %d, want %d", tt.path, res.StatusCode, tt.want)
			}
		})
	}
}

// A 401 without WWW-Authenticate makes browsers show a blank error instead of a
// login prompt, which would make a gated instance look broken.
func TestBasicAuthChallengeHeader(t *testing.T) {
	ts := newBasicAuthTestServer(t, Config{BasicAuthUser: "demo", BasicAuthPass: "s3cret"})
	res := get(t, ts.URL+"/", "", "")
	if got := res.Header.Get("WWW-Authenticate"); got == "" {
		t.Fatal("gated 401 must send a WWW-Authenticate challenge, got none")
	}
}
