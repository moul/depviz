package backend

import (
	"bytes"
	"context"
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"moul.io/depviz/v4/internal/core"
)

func (s *Server) handleGitHubWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.githubAppConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "github app is not configured"})
		return
	}
	if s.cfg.GitHubWebhookSecret == "" {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "github webhook secret is not configured"})
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}
	if !validGitHubSignature(body, r.Header.Get("X-Hub-Signature-256"), s.cfg.GitHubWebhookSecret) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "invalid github webhook signature"})
		return
	}
	event := r.Header.Get("X-GitHub-Event")
	switch event {
	case "ping", "installation", "installation_repositories":
		if err := s.ingestGitHubInstallationWebhook(r.Context(), body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	case "issues":
		if err := s.ingestGitHubIssueWebhook(r.Context(), body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	case "pull_request":
		if err := s.ingestGitHubPullRequestWebhook(r.Context(), body); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "event": event})
}

func (s *Server) syncGitHubInstallation(ctx context.Context, installationID int64) error {
	jwt, err := s.githubAppJWT()
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fmt.Sprintf("https://api.github.com/app/installations/%d", installationID), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+jwt)
	var installation githubInstallationPayload
	if err := s.doJSON(req, &installation); err != nil {
		return err
	}
	raw, _ := json.Marshal(installation)
	return s.upsertGitHubInstallation(ctx, installation, raw)
}

func (s *Server) githubInstallationAccessToken(ctx context.Context, installationID int64) (githubInstallationToken, error) {
	jwt, err := s.githubAppJWT()
	if err != nil {
		return githubInstallationToken{}, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, fmt.Sprintf("https://api.github.com/app/installations/%d/access_tokens", installationID), bytes.NewReader([]byte(`{}`)))
	if err != nil {
		return githubInstallationToken{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("Authorization", "Bearer "+jwt)
	var token githubInstallationToken
	if err := s.doJSON(req, &token); err != nil {
		return githubInstallationToken{}, err
	}
	if token.Token == "" {
		return githubInstallationToken{}, errors.New("github did not return an installation token")
	}
	return token, nil
}

func (s *Server) githubAppJWT() (string, error) {
	key, err := loadGitHubAppPrivateKey(s.cfg.GitHubAppPrivateKeyFile)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iat": now.Add(-30 * time.Second).Unix(),
		"exp": now.Add(9 * time.Minute).Unix(),
		"iss": s.cfg.GitHubAppID,
	}
	headerJSON, _ := json.Marshal(header)
	claimsJSON, _ := json.Marshal(claims)
	unsigned := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	sum := sha256.Sum256([]byte(unsigned))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		return "", err
	}
	return unsigned + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func (s *Server) ingestGitHubInstallationWebhook(ctx context.Context, body []byte) error {
	var payload struct {
		Installation githubInstallationPayload `json:"installation"`
		Repositories []githubRepo              `json:"repositories"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	if payload.Installation.ID == 0 {
		return nil
	}
	if err := s.upsertGitHubInstallation(ctx, payload.Installation, body); err != nil {
		return err
	}
	for _, repo := range payload.Repositories {
		raw, _ := json.Marshal(repo)
		if _, err := s.store.UpsertWorkspace(ctx, core.Workspace{
			Provider:   "github",
			ExternalID: fmt.Sprint(repo.ID),
			Kind:       "repo",
			Name:       repo.FullName,
			URL:        repo.HTMLURL,
			DataJSON:   string(raw),
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) upsertGitHubInstallation(ctx context.Context, installation githubInstallationPayload, raw []byte) error {
	if _, err := s.store.UpsertGitHubInstallation(ctx, core.GitHubInstallation{
		InstallationID: installation.ID,
		AccountLogin:   installation.Account.Login,
		AccountID:      installation.Account.ID,
		AccountType:    installation.Account.Type,
		TargetType:     installation.TargetType,
		RepositoryMode: installation.RepositorySelection,
		HTMLURL:        installation.HTMLURL,
		RawJSON:        string(raw),
	}); err != nil {
		return err
	}
	kind := strings.ToLower(installation.Account.Type)
	if kind == "" {
		kind = "account"
	}
	_, err := s.store.UpsertWorkspace(ctx, core.Workspace{
		Provider:   "github",
		ExternalID: fmt.Sprint(installation.Account.ID),
		Kind:       kind,
		Name:       installation.Account.Login,
		URL:        installation.Account.HTMLURL,
		DataJSON:   string(raw),
	})
	return err
}

func validGitHubSignature(body []byte, signature, secret string) bool {
	if !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(body)
	expected := "sha256=" + hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func loadGitHubAppPrivateKey(path string) (*rsa.PrivateKey, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	block, _ := pem.Decode(data)
	if block == nil {
		return nil, errors.New("github app private key is not PEM")
	}
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	key, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, errors.New("github app private key is not RSA")
	}
	return key, nil
}

func parseInstallationID(value string) int64 {
	id, _ := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	return id
}

type githubInstallationToken struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type githubInstallationPayload struct {
	ID                  int64  `json:"id"`
	TargetType          string `json:"target_type"`
	RepositorySelection string `json:"repository_selection"`
	HTMLURL             string `json:"html_url"`
	Account             struct {
		ID        int64  `json:"id"`
		Login     string `json:"login"`
		Type      string `json:"type"`
		HTMLURL   string `json:"html_url"`
		AvatarURL string `json:"avatar_url"`
	} `json:"account"`
}

func (s *Server) ingestGitHubIssueWebhook(ctx context.Context, body []byte) error {
	var payload struct {
		Action string `json:"action"`
		Issue  struct {
			ID      int64  `json:"id"`
			Number  int    `json:"number"`
			Title   string `json:"title"`
			State   string `json:"state"`
			HTMLURL string `json:"html_url"`
			User    struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"issue"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	if payload.Issue.Number == 0 || payload.Repository.FullName == "" {
		return nil
	}
	state := payload.Issue.State
	if state == "" {
		state = "open"
	}
	nodeID := fmt.Sprintf("gh:%s#%d", payload.Repository.FullName, payload.Issue.Number)
	raw, _ := json.Marshal(payload.Issue)
	return s.store.UpsertNode(ctx, core.Node{
		ID:       nodeID,
		Kind:     "issue",
		Title:    payload.Issue.Title,
		State:    state,
		Owner:    payload.Issue.User.Login,
		DataJSON: string(raw),
	})
}

func (s *Server) ingestGitHubPullRequestWebhook(ctx context.Context, body []byte) error {
	var payload struct {
		Action      string `json:"action"`
		PullRequest struct {
			ID      int64  `json:"id"`
			Number  int    `json:"number"`
			Title   string `json:"title"`
			State   string `json:"state"`
			Draft   bool   `json:"draft"`
			HTMLURL string `json:"html_url"`
			User    struct {
				Login string `json:"login"`
			} `json:"user"`
		} `json:"pull_request"`
		Repository struct {
			FullName string `json:"full_name"`
		} `json:"repository"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return err
	}
	if payload.PullRequest.Number == 0 || payload.Repository.FullName == "" {
		return nil
	}
	state := payload.PullRequest.State
	if state == "" {
		state = "open"
	}
	nodeID := fmt.Sprintf("gh:%s!%d", payload.Repository.FullName, payload.PullRequest.Number)
	raw, _ := json.Marshal(payload.PullRequest)
	return s.store.UpsertNode(ctx, core.Node{
		ID:       nodeID,
		Kind:     "pull_request",
		Title:    payload.PullRequest.Title,
		State:    state,
		Owner:    payload.PullRequest.User.Login,
		DataJSON: string(raw),
	})
}
