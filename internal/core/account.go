package core

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

type OAuthAccountInput struct {
	Provider   string
	ExternalID string
	Login      string
	Name       string
	AvatarURL  string
	HTMLURL    string
	Scopes     []string
	TokenJSON  string
}

func (s *Store) CreateOAuthState(ctx context.Context, provider, redirectURI string, ttl time.Duration) (string, error) {
	provider = strings.TrimSpace(provider)
	if provider == "" {
		return "", errors.New("oauth provider is required")
	}
	if redirectURI == "" {
		redirectURI = "/"
	}
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	state, err := randomToken(32)
	if err != nil {
		return "", err
	}
	now := nowUTC()
	_, err = s.db.ExecContext(ctx, `INSERT INTO oauth_states(state, provider, redirect_uri, expires_at, created_at)
		VALUES(?, ?, ?, ?, ?)`, state, provider, redirectURI, formatTime(now.Add(ttl)), formatTime(now))
	return state, err
}

func (s *Store) ConsumeOAuthState(ctx context.Context, provider, state string) (string, error) {
	var redirectURI, expires string
	err := s.db.QueryRowContext(ctx, `SELECT redirect_uri, expires_at FROM oauth_states WHERE state = ? AND provider = ?`, state, provider).
		Scan(&redirectURI, &expires)
	if err != nil {
		return "", errors.New("invalid oauth state")
	}
	_, _ = s.db.ExecContext(ctx, `DELETE FROM oauth_states WHERE state = ?`, state)
	if exp := parseTime(expires); exp.IsZero() || exp.Before(nowUTC()) {
		return "", errors.New("expired oauth state")
	}
	if redirectURI == "" {
		redirectURI = "/"
	}
	return redirectURI, nil
}

func (s *Store) UpsertOAuthAccount(ctx context.Context, in OAuthAccountInput) (Account, error) {
	in.Provider = strings.TrimSpace(in.Provider)
	in.ExternalID = strings.TrimSpace(in.ExternalID)
	in.Login = strings.TrimSpace(in.Login)
	if in.Provider == "" || in.ExternalID == "" || in.Login == "" {
		return Account{}, errors.New("provider, external id, and login are required")
	}
	now := nowUTC()
	accountID := stableID("account", in.Provider, in.ExternalID)
	var created string
	err := s.db.QueryRowContext(ctx, `SELECT created_at FROM accounts WHERE id = ?`, accountID).Scan(&created)
	if err != nil {
		created = formatTime(now)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO accounts(id, primary_provider, login, name, avatar_url, html_url, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			login=excluded.login,
			name=excluded.name,
			avatar_url=excluded.avatar_url,
			html_url=excluded.html_url,
			updated_at=excluded.updated_at`,
		accountID, in.Provider, in.Login, in.Name, in.AvatarURL, in.HTMLURL, created, formatTime(now))
	if err != nil {
		return Account{}, err
	}
	scopesJSON, _ := json.Marshal(in.Scopes)
	if in.TokenJSON == "" {
		in.TokenJSON = `{}`
	}
	connectionID := stableID("oauth", in.Provider, in.ExternalID)
	_, err = s.db.ExecContext(ctx, `INSERT INTO oauth_connections(id, account_id, provider, external_id, login, scopes_json, token_json, created_at, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(provider, external_id) DO UPDATE SET
			account_id=excluded.account_id,
			login=excluded.login,
			scopes_json=excluded.scopes_json,
			token_json=excluded.token_json,
			updated_at=excluded.updated_at`,
		connectionID, accountID, in.Provider, in.ExternalID, in.Login, string(scopesJSON), in.TokenJSON, created, formatTime(now))
	if err != nil {
		return Account{}, err
	}
	return s.AccountByID(ctx, accountID)
}

func (s *Store) AccountByID(ctx context.Context, accountID string) (Account, error) {
	var a Account
	var created, updated string
	err := s.db.QueryRowContext(ctx, `SELECT id, primary_provider, login, name, avatar_url, html_url, created_at, updated_at
		FROM accounts WHERE id = ?`, accountID).
		Scan(&a.ID, &a.PrimaryProvider, &a.Login, &a.Name, &a.AvatarURL, &a.HTMLURL, &created, &updated)
	if err != nil {
		return Account{}, err
	}
	a.CreatedAt = parseTime(created)
	a.UpdatedAt = parseTime(updated)
	return a, nil
}

func (s *Store) CreateWebSession(ctx context.Context, accountID string, ttl time.Duration) (string, time.Time, error) {
	if accountID == "" {
		return "", time.Time{}, errors.New("account id is required")
	}
	if ttl <= 0 {
		ttl = 30 * 24 * time.Hour
	}
	token, err := randomToken(32)
	if err != nil {
		return "", time.Time{}, err
	}
	now := nowUTC()
	expires := now.Add(ttl)
	_, err = s.db.ExecContext(ctx, `INSERT INTO web_sessions(token_hash, account_id, expires_at, created_at)
		VALUES(?, ?, ?, ?)`, sessionHash(token), accountID, formatTime(expires), formatTime(now))
	return token, expires, err
}

func (s *Store) AccountForWebSession(ctx context.Context, token string) (Account, bool, error) {
	if token == "" {
		return Account{}, false, nil
	}
	var accountID, expires string
	err := s.db.QueryRowContext(ctx, `SELECT account_id, expires_at FROM web_sessions WHERE token_hash = ?`, sessionHash(token)).
		Scan(&accountID, &expires)
	if err != nil {
		return Account{}, false, nil
	}
	if exp := parseTime(expires); exp.IsZero() || exp.Before(nowUTC()) {
		_, _ = s.db.ExecContext(ctx, `DELETE FROM web_sessions WHERE token_hash = ?`, sessionHash(token))
		return Account{}, false, nil
	}
	account, err := s.AccountByID(ctx, accountID)
	if err != nil {
		return Account{}, false, err
	}
	return account, true, nil
}

func (s *Store) DeleteWebSession(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM web_sessions WHERE token_hash = ?`, sessionHash(token))
	return err
}

func (s *Store) UpsertGitHubCache(ctx context.Context, accountID, repo, refID, payloadJSON, etag string, ttl time.Duration) error {
	if accountID == "" || repo == "" || refID == "" {
		return errors.New("account id, repo, and ref id are required")
	}
	if payloadJSON == "" {
		payloadJSON = `{}`
	}
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	now := nowUTC()
	id := stableID("github-cache", accountID, repo, refID)
	_, err := s.db.ExecContext(ctx, `INSERT INTO github_cache(id, account_id, repo, ref_id, payload_json, etag, fetched_at, expires_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(account_id, repo, ref_id) DO UPDATE SET
			payload_json=excluded.payload_json,
			etag=excluded.etag,
			fetched_at=excluded.fetched_at,
			expires_at=excluded.expires_at`,
		id, accountID, repo, refID, payloadJSON, etag, formatTime(now), formatTime(now.Add(ttl)))
	return err
}

func randomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func sessionHash(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
