package core

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestOAuthAccountAndWebSession(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	state, err := s.CreateOAuthState(ctx, "github", "/graph", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	redirect, err := s.ConsumeOAuthState(ctx, "github", state)
	if err != nil {
		t.Fatal(err)
	}
	if redirect != "/graph" {
		t.Fatalf("redirect = %q, want /graph", redirect)
	}
	if _, err := s.ConsumeOAuthState(ctx, "github", state); err == nil {
		t.Fatal("expected consumed oauth state to be invalid")
	}

	account, err := s.UpsertOAuthAccount(ctx, OAuthAccountInput{
		Provider:   "github",
		ExternalID: "42",
		Login:      "moul",
		Name:       "Manfred",
		Scopes:     []string{"repo", "read:user"},
		TokenJSON:  `{"access_token":"redacted"}`,
	})
	if err != nil {
		t.Fatal(err)
	}
	token, _, err := s.CreateWebSession(ctx, account.ID, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := s.AccountForWebSession(ctx, token)
	if err != nil {
		t.Fatal(err)
	}
	if !ok {
		t.Fatal("session did not authenticate")
	}
	if got.Login != "moul" {
		t.Fatalf("login = %q, want moul", got.Login)
	}
}
