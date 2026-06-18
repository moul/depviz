package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"moul.io/depviz/v4/internal/core"
	"moul.io/depviz/v4/live"
)

const sessionCookieName = "depviz_session"

type Config struct {
	Addr               string
	BaseURL            string
	GitHubClientID     string
	GitHubClientSecret string
	SessionTTL         time.Duration
}

type Server struct {
	cfg    Config
	store  *core.Store
	client *http.Client
}

func NewServer(store *core.Store, cfg Config) *Server {
	if cfg.SessionTTL <= 0 {
		cfg.SessionTTL = 30 * 24 * time.Hour
	}
	return &Server{
		cfg:    cfg,
		store:  store,
		client: http.DefaultClient,
	}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/health", s.handleHealth)
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/export", s.handleExport)
	mux.HandleFunc("/api/auth/github/start", s.handleGitHubStart)
	mux.HandleFunc("/api/auth/github/callback", s.handleGitHubCallback)
	mux.HandleFunc("/api/auth/logout", s.handleLogout)
	mux.Handle("/", http.FileServer(http.FS(live.AppFS())))
	return mux
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":                      true,
		"github_oauth_configured": s.githubOAuthConfigured(),
	})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
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
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	account, ok, err := s.accountForRequest(r)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	out := map[string]any{
		"authenticated":           ok,
		"github_oauth_configured": s.githubOAuthConfigured(),
	}
	if ok {
		out["account"] = account
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleGitHubStart(w http.ResponseWriter, r *http.Request) {
	if !s.githubOAuthConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "github oauth is not configured"})
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
		"scope":        {"repo read:user user:email"},
		"state":        {state},
	}
	http.Redirect(w, r, "https://github.com/login/oauth/authorize?"+params.Encode(), http.StatusFound)
}

func (s *Server) handleGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if !s.githubOAuthConfigured() {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{"error": "github oauth is not configured"})
		return
	}
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	state := strings.TrimSpace(r.URL.Query().Get("state"))
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

func (s *Server) accountForRequest(r *http.Request) (core.Account, bool, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil {
		return core.Account{}, false, nil
	}
	return s.store.AccountForWebSession(r.Context(), cookie.Value)
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

func (s *Server) doJSON(req *http.Request, out any) error {
	res, err := s.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("%s: %s", res.Status, strings.TrimSpace(string(data)))
	}
	return json.Unmarshal(data, out)
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
