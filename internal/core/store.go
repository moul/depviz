package core

import (
	"bufio"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	db   *sql.DB
	path string
}

type NodeFieldUpdate struct {
	Title       *string
	Status      *string
	Owner       *string
	Description *string
	TimeHorizon *string
	Priority    *string
	Labels      *[]string
}

func OpenStore(ctx context.Context, path string) (*Store, error) {
	if path == "" {
		path = DefaultDBPath
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, path: path}
	if _, err := db.ExecContext(ctx, "PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.Migrate(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	if err := s.EnsureDefaultBoard(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

func (s *Store) Path() string {
	return s.path
}

func (s *Store) Backup(ctx context.Context, outPath string) error {
	if strings.TrimSpace(outPath) == "" {
		return errors.New("backup output path is required")
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0o755); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `VACUUM INTO ?`, outPath)
	return err
}

func (s *Store) Migrate(ctx context.Context) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS sources (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL DEFAULT '',
			capabilities_json TEXT NOT NULL DEFAULT '{}',
			sync_json TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS nodes (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			title TEXT NOT NULL,
			state TEXT NOT NULL,
			owner TEXT NOT NULL DEFAULT '',
			data_json TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS source_refs (
			id TEXT PRIMARY KEY,
			node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
			source_id TEXT NOT NULL REFERENCES sources(id) ON DELETE CASCADE,
			external_id TEXT NOT NULL,
			url TEXT NOT NULL DEFAULT '',
			sync_cursor TEXT NOT NULL DEFAULT '',
			last_seen_at TEXT NOT NULL,
			UNIQUE(source_id, external_id)
		)`,
		`CREATE TABLE IF NOT EXISTS boards (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			description TEXT NOT NULL DEFAULT '',
			scope_query TEXT NOT NULL DEFAULT '',
			parent_board_id TEXT NOT NULL DEFAULT '',
			config_json TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS board_items (
			board_id TEXT NOT NULL REFERENCES boards(id) ON DELETE CASCADE,
			node_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
			role TEXT NOT NULL DEFAULT '',
			local_state TEXT NOT NULL DEFAULT '',
			sort_key TEXT NOT NULL DEFAULT '',
			data_json TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL,
			PRIMARY KEY(board_id, node_id)
		)`,
		`CREATE TABLE IF NOT EXISTS edges (
			id TEXT PRIMARY KEY,
			from_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
			to_id TEXT NOT NULL REFERENCES nodes(id) ON DELETE CASCADE,
			kind TEXT NOT NULL,
			scope_board_id TEXT NOT NULL DEFAULT '',
			confidence REAL NOT NULL DEFAULT 1.0,
			authority TEXT NOT NULL DEFAULT 'local',
			evidence_json TEXT NOT NULL DEFAULT '{}',
			observed_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS events (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT NOT NULL,
			object_id TEXT NOT NULL DEFAULT '',
			data_json TEXT NOT NULL,
			observed_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS field_values (
			id TEXT PRIMARY KEY,
			owner_type TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			namespace TEXT NOT NULL,
			key TEXT NOT NULL,
			value_json TEXT NOT NULL,
			authority TEXT NOT NULL DEFAULT 'local',
			source_id TEXT NOT NULL DEFAULT '',
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS accounts (
			id TEXT PRIMARY KEY,
			primary_provider TEXT NOT NULL,
			login TEXT NOT NULL,
			name TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			html_url TEXT NOT NULL DEFAULT '',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_connections (
			id TEXT PRIMARY KEY,
			account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			provider TEXT NOT NULL,
			external_id TEXT NOT NULL,
			login TEXT NOT NULL,
			scopes_json TEXT NOT NULL DEFAULT '[]',
			token_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(provider, external_id)
		)`,
		`CREATE TABLE IF NOT EXISTS oauth_states (
			state TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			redirect_uri TEXT NOT NULL DEFAULT '/',
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS web_sessions (
			token_hash TEXT PRIMARY KEY,
			account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS github_cache (
			id TEXT PRIMARY KEY,
			account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			repo TEXT NOT NULL,
			ref_id TEXT NOT NULL,
			payload_json TEXT NOT NULL,
			etag TEXT NOT NULL DEFAULT '',
			fetched_at TEXT NOT NULL,
			expires_at TEXT NOT NULL,
			UNIQUE(account_id, repo, ref_id)
		)`,
		`CREATE TABLE IF NOT EXISTS workspaces (
			id TEXT PRIMARY KEY,
			provider TEXT NOT NULL,
			external_id TEXT NOT NULL,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			url TEXT NOT NULL DEFAULT '',
			data_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(provider, external_id)
		)`,
		`CREATE TABLE IF NOT EXISTS workspace_memberships (
			workspace_id TEXT NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
			account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			role TEXT NOT NULL DEFAULT 'member',
			source TEXT NOT NULL DEFAULT 'github',
			updated_at TEXT NOT NULL,
			PRIMARY KEY(workspace_id, account_id)
		)`,
		`CREATE TABLE IF NOT EXISTS personal_overrides (
			id TEXT PRIMARY KEY,
			account_id TEXT NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
			owner_type TEXT NOT NULL,
			owner_id TEXT NOT NULL,
			data_json TEXT NOT NULL DEFAULT '{}',
			updated_at TEXT NOT NULL,
			UNIQUE(account_id, owner_type, owner_id)
		)`,
		`CREATE TABLE IF NOT EXISTS github_installations (
			id TEXT PRIMARY KEY,
			installation_id INTEGER NOT NULL UNIQUE,
			account_login TEXT NOT NULL DEFAULT '',
			account_id INTEGER NOT NULL DEFAULT 0,
			account_type TEXT NOT NULL DEFAULT '',
			target_type TEXT NOT NULL DEFAULT '',
			repository_mode TEXT NOT NULL DEFAULT '',
			html_url TEXT NOT NULL DEFAULT '',
			raw_json TEXT NOT NULL DEFAULT '{}',
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
	// Additive migrations (ALTER TABLE — idempotent via ignored errors)
	_, _ = s.db.ExecContext(ctx, `ALTER TABLE nodes ADD COLUMN archived_at TEXT`)
	_, _ = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS dismissed_suggestions (
		account_id TEXT NOT NULL,
		board_id TEXT NOT NULL,
		edge_id TEXT NOT NULL,
		dismissed_at TEXT NOT NULL,
		PRIMARY KEY (account_id, board_id, edge_id)
	)`)
	_, _ = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS sync_logs (
		id TEXT PRIMARY KEY,
		board_id TEXT NOT NULL,
		started_at TEXT NOT NULL,
		completed_at TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL,
		items_synced INTEGER NOT NULL DEFAULT 0,
		edges_synced INTEGER NOT NULL DEFAULT 0,
		mode TEXT NOT NULL DEFAULT '',
		error TEXT NOT NULL DEFAULT '',
		rate_limit_remaining INTEGER NOT NULL DEFAULT 0,
		rate_limit_reset TEXT NOT NULL DEFAULT ''
	)`)
	_, _ = s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS board_views (
		id TEXT PRIMARY KEY,
		board_id TEXT NOT NULL,
		name TEXT NOT NULL,
		config_json TEXT NOT NULL DEFAULT '{}',
		created_at TEXT NOT NULL
	)`)
	return nil
}

func (s *Store) EnsureDefaultBoard(ctx context.Context) error {
	now := formatTime(nowUTC())
	if err := s.UpsertSource(ctx, Source{
		ID:           LocalSourceID,
		Kind:         "local",
		Name:         "Local DepViz",
		Capabilities: `{"write":"local"}`,
		Sync:         `{}`,
		UpdatedAt:    nowUTC(),
	}); err != nil {
		return err
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO boards(id, name, description, scope_query, parent_board_id, config_json, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO NOTHING`,
		DefaultBoardID, "Default", "Default local DepViz board", "", "", `{}`, now)
	return err
}

func (s *Store) UpsertSource(ctx context.Context, src Source) error {
	if src.UpdatedAt.IsZero() {
		src.UpdatedAt = nowUTC()
	}
	if src.Capabilities == "" {
		src.Capabilities = `{}`
	}
	if src.Sync == "" {
		src.Sync = `{}`
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO sources(id, kind, name, url, capabilities_json, sync_json, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind=excluded.kind,
			name=excluded.name,
			url=excluded.url,
			capabilities_json=excluded.capabilities_json,
			sync_json=excluded.sync_json,
			updated_at=excluded.updated_at`,
		src.ID, src.Kind, src.Name, src.URL, src.Capabilities, src.Sync, formatTime(src.UpdatedAt))
	return err
}

func (s *Store) UpsertNode(ctx context.Context, n Node) error {
	if n.ID == "" {
		return errors.New("node id is required")
	}
	if n.Kind == "" {
		n.Kind = "task"
	}
	if n.Title == "" {
		n.Title = n.ID
	}
	if n.State == "" {
		n.State = "open"
	}
	if n.DataJSON == "" {
		n.DataJSON = `{}`
	}
	if n.UpdatedAt.IsZero() {
		n.UpdatedAt = nowUTC()
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO nodes(id, kind, title, state, owner, data_json, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind=excluded.kind,
			title=excluded.title,
			state=excluded.state,
			owner=excluded.owner,
			data_json=excluded.data_json,
			updated_at=excluded.updated_at`,
		n.ID, n.Kind, n.Title, n.State, n.Owner, n.DataJSON, formatTime(n.UpdatedAt))
	return err
}

func (s *Store) UpsertSourceRef(ctx context.Context, nodeID, sourceID, externalID, url string) error {
	if sourceID == "" || externalID == "" {
		return nil
	}
	id := stableID("ref", sourceID, externalID)
	_, err := s.db.ExecContext(ctx, `INSERT INTO source_refs(id, node_id, source_id, external_id, url, sync_cursor, last_seen_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(source_id, external_id) DO UPDATE SET
			node_id=excluded.node_id,
			url=excluded.url,
			last_seen_at=excluded.last_seen_at`,
		id, nodeID, sourceID, externalID, url, "", formatTime(nowUTC()))
	return err
}

func (s *Store) AddNodeToBoard(ctx context.Context, boardID, nodeID, role, localState string) error {
	if boardID == "" {
		boardID = DefaultBoardID
	}
	if role == "" {
		role = "card"
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO board_items(board_id, node_id, role, local_state, sort_key, data_json, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(board_id, node_id) DO UPDATE SET
			role=excluded.role,
			local_state=CASE WHEN excluded.local_state != '' THEN excluded.local_state ELSE board_items.local_state END,
			updated_at=excluded.updated_at`,
		boardID, nodeID, role, localState, "", `{}`, formatTime(nowUTC()))
	return err
}

func (s *Store) CreateNote(ctx context.Context, boardID, text string) (Node, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return Node{}, errors.New("note text is required")
	}
	id, err := s.availableNoteID(ctx, text)
	if err != nil {
		return Node{}, err
	}
	payload, _ := json.Marshal(map[string]any{"source": "local", "text": text, "created_at": formatTime(nowUTC())})
	n := Node{
		ID:        id,
		Kind:      "note",
		Title:     text,
		State:     "local",
		DataJSON:  string(payload),
		UpdatedAt: nowUTC(),
	}
	if err := s.UpsertNode(ctx, n); err != nil {
		return Node{}, err
	}
	if err := s.UpsertSourceRef(ctx, n.ID, LocalSourceID, n.ID, ""); err != nil {
		return Node{}, err
	}
	if err := s.AddNodeToBoard(ctx, boardID, n.ID, "note", "local"); err != nil {
		return Node{}, err
	}
	if err := s.RecordEvent(ctx, "depviz.note.v1", n.ID, payload); err != nil {
		return Node{}, err
	}
	return n, nil
}

func (s *Store) AddEdge(ctx context.Context, boardID, fromID, toID, kind, authority string, evidence any) (Edge, error) {
	return s.AddEdgeWithConfidence(ctx, boardID, fromID, toID, kind, authority, 1, evidence)
}

func (s *Store) AddEdgeWithConfidence(ctx context.Context, boardID, fromID, toID, kind, authority string, confidence float64, evidence any) (Edge, error) {
	if fromID == "" || toID == "" {
		return Edge{}, errors.New("edge from and to are required")
	}
	if kind == "" {
		kind = "blocked_by"
	}
	if boardID == "" {
		boardID = DefaultBoardID
	}
	if authority == "" {
		authority = "local"
	}
	if confidence <= 0 {
		confidence = 1
	}
	if err := s.ensureNodeInBoard(ctx, boardID, fromID); err != nil {
		return Edge{}, err
	}
	if err := s.ensureNodeInBoard(ctx, boardID, toID); err != nil {
		return Edge{}, err
	}
	evidenceJSON := `{}`
	if evidence != nil {
		b, err := json.Marshal(evidence)
		if err != nil {
			return Edge{}, err
		}
		evidenceJSON = string(b)
	}
	e := Edge{
		ID:           stableID("edge", boardID, fromID, toID, kind),
		FromID:       fromID,
		ToID:         toID,
		Kind:         kind,
		ScopeBoardID: boardID,
		Confidence:   confidence,
		Authority:    authority,
		EvidenceJSON: evidenceJSON,
		ObservedAt:   nowUTC(),
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO edges(id, from_id, to_id, kind, scope_board_id, confidence, authority, evidence_json, observed_at)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			kind=excluded.kind,
			confidence=excluded.confidence,
			authority=excluded.authority,
			evidence_json=excluded.evidence_json,
			observed_at=excluded.observed_at`,
		e.ID, e.FromID, e.ToID, e.Kind, e.ScopeBoardID, e.Confidence, e.Authority, e.EvidenceJSON, formatTime(e.ObservedAt))
	if err != nil {
		return Edge{}, err
	}
	payload, _ := json.Marshal(e)
	return e, s.RecordEvent(ctx, "depviz.edge.v1", e.ID, payload)
}

func (s *Store) ensureNodeInBoard(ctx context.Context, boardID, nodeID string) error {
	exists, err := s.nodeExists(ctx, nodeID)
	if err != nil {
		return err
	}
	if !exists {
		n, sourceID, externalID, url := placeholderNode(nodeID)
		if err := s.UpsertSource(ctx, Source{ID: sourceID, Kind: sourceKind(sourceID), Name: sourceID, URL: sourceURL(sourceID), Capabilities: `{}`, Sync: `{}`, UpdatedAt: nowUTC()}); err != nil {
			return err
		}
		if err := s.UpsertNode(ctx, n); err != nil {
			return err
		}
		if err := s.UpsertSourceRef(ctx, n.ID, sourceID, externalID, url); err != nil {
			return err
		}
	}
	itemExists, err := s.boardItemExists(ctx, boardID, nodeID)
	if err != nil {
		return err
	}
	if !itemExists {
		role := "card"
		if strings.HasPrefix(nodeID, "note:") {
			role = "note"
		}
		return s.AddNodeToBoard(ctx, boardID, nodeID, role, "")
	}
	return nil
}

func (s *Store) nodeExists(ctx context.Context, nodeID string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE id = ?`, nodeID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) nodeByID(ctx context.Context, nodeID string) (Node, error) {
	var n Node
	var updated string
	err := s.db.QueryRowContext(ctx, `SELECT id, kind, title, state, owner, data_json, updated_at FROM nodes WHERE id = ?`, nodeID).
		Scan(&n.ID, &n.Kind, &n.Title, &n.State, &n.Owner, &n.DataJSON, &updated)
	if err != nil {
		return Node{}, err
	}
	n.UpdatedAt = parseTime(updated)
	return n, nil
}

func (s *Store) availableNodeID(ctx context.Context, base string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		base = "task:untitled"
	}
	for i := 0; i < 1000; i++ {
		id := base
		if i > 0 {
			id = fmt.Sprintf("%s-%d", base, i+1)
		}
		exists, err := s.nodeExists(ctx, id)
		if err != nil {
			return "", err
		}
		if !exists {
			return id, nil
		}
	}
	return "", errors.New("could not allocate node id")
}

func (s *Store) boardItemExists(ctx context.Context, boardID, nodeID string) (bool, error) {
	var count int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM board_items WHERE board_id = ? AND node_id = ?`, boardID, nodeID).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func (s *Store) RecordEvent(ctx context.Context, eventType, objectID string, payload []byte) error {
	if len(payload) == 0 {
		payload = []byte(`{}`)
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO events(type, object_id, data_json, observed_at)
		VALUES(?, ?, ?, ?)`, eventType, objectID, string(payload), formatTime(nowUTC()))
	return err
}

func (s *Store) BoardList(ctx context.Context) ([]Board, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, description, scope_query, parent_board_id, config_json, updated_at FROM boards ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var boards []Board
	for rows.Next() {
		var b Board
		var updated string
		if err := rows.Scan(&b.ID, &b.Name, &b.Description, &b.ScopeQuery, &b.ParentBoardID, &b.ConfigJSON, &updated); err != nil {
			return nil, err
		}
		b.UpdatedAt = parseTime(updated)
		metrics, err := s.BoardMetrics(ctx, b.ID)
		if err != nil {
			return nil, err
		}
		b.Metrics = &metrics
		boards = append(boards, b)
	}
	return boards, rows.Err()
}

func (s *Store) BoardMetrics(ctx context.Context, boardID string) (BoardMetrics, error) {
	if boardID == "" {
		boardID = DefaultBoardID
	}
	var m BoardMetrics
	var latestNode string
	err := s.db.QueryRowContext(ctx, `SELECT
			COUNT(DISTINCT n.id),
			COUNT(DISTINCT CASE WHEN lower(n.state) NOT IN ('closed', 'done', 'merged', 'cancelled', 'canceled', 'resolved', 'rejected') THEN n.id END),
			COUNT(DISTINCT CASE WHEN lower(n.state) IN ('closed', 'done', 'merged', 'cancelled', 'canceled', 'resolved', 'rejected') THEN n.id END),
			COUNT(DISTINCT CASE WHEN n.kind = 'note' OR n.id LIKE 'note:%' OR bi.local_state != '' THEN n.id END),
			COUNT(DISTINCT CASE WHEN COALESCE(sr.source_id, '') != '' AND sr.source_id != ? THEN n.id END),
			COALESCE(MAX(n.updated_at), '')
		FROM board_items bi
		JOIN nodes n ON n.id = bi.node_id
		LEFT JOIN source_refs sr ON sr.node_id = n.id
		WHERE bi.board_id = ? AND (n.archived_at IS NULL OR n.archived_at = '')`, LocalSourceID, boardID).Scan(&m.Items, &m.Open, &m.Closed, &m.Local, &m.External, &latestNode)
	if err != nil {
		return BoardMetrics{}, err
	}
	var linkCount int
	var latestEdge string
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*), COALESCE(MAX(observed_at), '') FROM edges WHERE scope_board_id = ? OR scope_board_id = ''`, boardID).Scan(&linkCount, &latestEdge); err != nil {
		return BoardMetrics{}, err
	}
	var boardUpdated string
	if err := s.db.QueryRowContext(ctx, `SELECT updated_at FROM boards WHERE id = ?`, boardID).Scan(&boardUpdated); err != nil {
		return BoardMetrics{}, err
	}
	m.Links = linkCount
	m.LastActivityAt = maxParsedTime(boardUpdated, latestNode, latestEdge)
	syncAt, syncStatus, syncError, err := s.BoardSyncState(ctx, boardID)
	if err != nil {
		return BoardMetrics{}, err
	}
	m.LastSyncAt = syncAt
	m.SyncStatus = syncStatus
	m.SyncError = syncError
	return m, nil
}

func (s *Store) BoardSyncState(ctx context.Context, boardID string) (time.Time, string, string, error) {
	if boardID == "" {
		boardID = DefaultBoardID
	}
	var payload string
	var observed string
	err := s.db.QueryRowContext(ctx, `SELECT data_json, observed_at FROM events
		WHERE type = 'depviz.board_sync.v1' AND object_id = ?
		ORDER BY observed_at DESC, seq DESC LIMIT 1`, boardID).Scan(&payload, &observed)
	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, "never", "", nil
	}
	if err != nil {
		return time.Time{}, "", "", err
	}
	var data struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}
	_ = json.Unmarshal([]byte(payload), &data)
	if data.Status == "" {
		data.Status = "unknown"
	}
	return parseTime(observed), data.Status, data.Error, nil
}

func (s *Store) RecordBoardSync(ctx context.Context, boardID, status string, details any) error {
	if boardID == "" {
		boardID = DefaultBoardID
	}
	payload := map[string]any{"status": strings.TrimSpace(status)}
	if payload["status"] == "" {
		payload["status"] = "unknown"
	}
	if details != nil {
		if b, err := json.Marshal(details); err == nil {
			_ = json.Unmarshal(b, &payload)
			payload["status"] = strings.TrimSpace(status)
			if payload["status"] == "" {
				payload["status"] = "unknown"
			}
		}
	}
	data, _ := json.Marshal(payload)
	return s.RecordEvent(ctx, "depviz.board_sync.v1", boardID, data)
}

func (s *Store) CreateBoard(ctx context.Context, name, description string) (Board, error) {
	return s.CreateBoardWithConfig(ctx, name, description, "", `{}`)
}

func (s *Store) CreateBoardWithConfig(ctx context.Context, name, description, scopeQuery, configJSON string) (Board, error) {
	name = strings.TrimSpace(name)
	description = strings.TrimSpace(description)
	scopeQuery = strings.TrimSpace(scopeQuery)
	if name == "" {
		return Board{}, errors.New("board name is required")
	}
	if configJSON == "" {
		configJSON = `{}`
	}
	if !json.Valid([]byte(configJSON)) {
		return Board{}, errors.New("board config must be valid json")
	}
	base := slug(name)
	if base == "" {
		base = "board"
	}
	id := base
	for i := 0; i < 1000; i++ {
		if i > 0 {
			id = fmt.Sprintf("%s-%d", base, i+1)
		}
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM boards WHERE id = ?`, id).Scan(&count); err != nil {
			return Board{}, err
		}
		if count == 0 {
			break
		}
		if i == 999 {
			return Board{}, errors.New("could not allocate board id")
		}
	}
	board := Board{
		ID:          id,
		Name:        name,
		Description: description,
		ScopeQuery:  scopeQuery,
		ConfigJSON:  configJSON,
		UpdatedAt:   nowUTC(),
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO boards(id, name, description, scope_query, parent_board_id, config_json, updated_at)
		VALUES(?, ?, ?, ?, ?, ?, ?)`,
		board.ID, board.Name, board.Description, board.ScopeQuery, board.ParentBoardID, board.ConfigJSON, formatTime(board.UpdatedAt))
	return board, err
}

func (s *Store) AddTaskToBoard(ctx context.Context, boardID, title string) (Node, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return Node{}, errors.New("task title is required")
	}
	id, err := s.availableNodeID(ctx, "task:"+slug(title))
	if err != nil {
		return Node{}, err
	}
	payload, _ := json.Marshal(map[string]any{"source": "local", "text": title, "created_at": formatTime(nowUTC())})
	n := Node{
		ID:        id,
		Kind:      "task",
		Title:     title,
		State:     "open",
		DataJSON:  string(payload),
		UpdatedAt: nowUTC(),
	}
	if err := s.UpsertNode(ctx, n); err != nil {
		return Node{}, err
	}
	if err := s.UpsertSourceRef(ctx, n.ID, LocalSourceID, n.ID, ""); err != nil {
		return Node{}, err
	}
	if err := s.AddNodeToBoard(ctx, boardID, n.ID, "card", "local"); err != nil {
		return Node{}, err
	}
	if err := s.RecordEvent(ctx, "depviz.task.v1", n.ID, payload); err != nil {
		return Node{}, err
	}
	return n, nil
}

// CreateStrategyNode creates a local non-GitHub planning node with an extended type.
// kind must be one of: strategy, initiative, bet, project, workstream, risk, decision, question, metric, task, note
// status must be one of: draft, active, blocked, at-risk, paused, done, rejected, open, closed
func (s *Store) CreateStrategyNode(ctx context.Context, boardID, kind, title, status, owner, description, timeHorizon, priority string, labels []string) (Node, error) {
	kind = strings.TrimSpace(strings.ToLower(kind))
	title = strings.TrimSpace(title)
	status = strings.TrimSpace(strings.ToLower(status))
	owner = strings.TrimSpace(owner)
	description = strings.TrimSpace(description)
	if title == "" {
		return Node{}, errors.New("title is required")
	}
	validKinds := map[string]bool{
		"strategy": true, "initiative": true, "bet": true, "project": true,
		"workstream": true, "risk": true, "decision": true, "question": true,
		"metric": true, "task": true, "note": true,
	}
	if !validKinds[kind] {
		kind = "task"
	}
	if status == "" {
		if kind == "note" {
			status = "local"
		} else {
			status = "draft"
		}
	}
	prefix := kind + ":"
	id, err := s.availableNodeID(ctx, prefix+slug(title))
	if err != nil {
		return Node{}, err
	}
	payload, _ := json.Marshal(map[string]any{
		"source":       "local",
		"kind":         kind,
		"text":         title,
		"description":  description,
		"owner":        owner,
		"time_horizon": timeHorizon,
		"priority":     priority,
		"labels":       labels,
		"created_at":   formatTime(nowUTC()),
	})
	n := Node{
		ID:        id,
		Kind:      kind,
		Title:     title,
		State:     status,
		Owner:     owner,
		DataJSON:  string(payload),
		UpdatedAt: nowUTC(),
	}
	if err := s.UpsertNode(ctx, n); err != nil {
		return Node{}, err
	}
	if err := s.UpsertSourceRef(ctx, n.ID, LocalSourceID, n.ID, ""); err != nil {
		return Node{}, err
	}
	role := "card"
	if kind == "note" {
		role = "note"
	}
	if err := s.AddNodeToBoard(ctx, boardID, n.ID, role, "local"); err != nil {
		return Node{}, err
	}
	evPayload, _ := json.Marshal(n)
	return n, s.RecordEvent(ctx, "depviz.strategy_node.v1", n.ID, evPayload)
}

// UpdateNodeFields updates editable fields of a node.
// Nil fields are left untouched; present empty values clear optional fields.
func (s *Store) UpdateNodeFields(ctx context.Context, nodeID string, update NodeFieldUpdate) (Node, error) {
	n, err := s.nodeByID(ctx, nodeID)
	if err != nil {
		return Node{}, fmt.Errorf("node not found: %w", err)
	}
	if update.Title != nil {
		title := strings.TrimSpace(*update.Title)
		if title == "" {
			return Node{}, errors.New("title is required")
		}
		n.Title = title
	}
	if update.Status != nil {
		status := strings.TrimSpace(strings.ToLower(*update.Status))
		if status == "" {
			return Node{}, errors.New("status is required")
		}
		n.State = status
	}
	if update.Owner != nil {
		n.Owner = strings.TrimSpace(*update.Owner)
	}
	var data map[string]any
	if n.DataJSON != "" {
		_ = json.Unmarshal([]byte(n.DataJSON), &data)
	}
	if data == nil {
		data = map[string]any{}
	}
	if update.Description != nil {
		description := strings.TrimSpace(*update.Description)
		if description == "" {
			delete(data, "description")
		} else {
			data["description"] = description
		}
	}
	if update.Owner != nil {
		if n.Owner == "" {
			delete(data, "owner")
		} else {
			data["owner"] = n.Owner
		}
	}
	if update.TimeHorizon != nil {
		timeHorizon := strings.TrimSpace(*update.TimeHorizon)
		if timeHorizon == "" {
			delete(data, "time_horizon")
		} else {
			data["time_horizon"] = timeHorizon
		}
	}
	if update.Priority != nil {
		priority := strings.TrimSpace(*update.Priority)
		if priority == "" {
			delete(data, "priority")
		} else {
			data["priority"] = priority
		}
	}
	if update.Labels != nil {
		cleaned := make([]string, 0, len(*update.Labels))
		for _, label := range *update.Labels {
			label = strings.TrimSpace(label)
			if label != "" {
				cleaned = append(cleaned, label)
			}
		}
		if len(cleaned) == 0 {
			delete(data, "labels")
		} else {
			data["labels"] = cleaned
		}
	}
	n.UpdatedAt = nowUTC()
	merged, _ := json.Marshal(data)
	n.DataJSON = string(merged)
	if err := s.UpsertNode(ctx, n); err != nil {
		return Node{}, err
	}
	evPayload, _ := json.Marshal(n)
	return n, s.RecordEvent(ctx, "depviz.node_update.v1", n.ID, evPayload)
}

// RemoveNodeFromBoard removes a node from a board. If the node is local-only and has no
// other board references, it is also deleted from the nodes table.
func (s *Store) RemoveNodeFromBoard(ctx context.Context, boardID, nodeID string) error {
	if boardID == "" {
		boardID = DefaultBoardID
	}
	_, err := s.db.ExecContext(ctx, `DELETE FROM board_items WHERE board_id = ? AND node_id = ?`, boardID, nodeID)
	if err != nil {
		return err
	}
	// Also delete edges scoped to this board that reference the node
	_, err = s.db.ExecContext(ctx, `DELETE FROM edges WHERE scope_board_id = ? AND (from_id = ? OR to_id = ?)`, boardID, nodeID, nodeID)
	if err != nil {
		return err
	}
	// If local node and no other board references, delete from nodes too
	isLocal := strings.HasPrefix(nodeID, "note:") || strings.HasPrefix(nodeID, "task:") ||
		strings.HasPrefix(nodeID, "strategy:") || strings.HasPrefix(nodeID, "initiative:") ||
		strings.HasPrefix(nodeID, "bet:") || strings.HasPrefix(nodeID, "project:") ||
		strings.HasPrefix(nodeID, "workstream:") || strings.HasPrefix(nodeID, "risk:") ||
		strings.HasPrefix(nodeID, "decision:") || strings.HasPrefix(nodeID, "question:") ||
		strings.HasPrefix(nodeID, "metric:")
	if isLocal {
		var count int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM board_items WHERE node_id = ?`, nodeID).Scan(&count); err == nil && count == 0 {
			_, _ = s.db.ExecContext(ctx, `DELETE FROM source_refs WHERE node_id = ?`, nodeID)
			_, _ = s.db.ExecContext(ctx, `DELETE FROM nodes WHERE id = ?`, nodeID)
		}
	}
	payload, _ := json.Marshal(map[string]any{"board_id": boardID, "node_id": nodeID})
	return s.RecordEvent(ctx, "depviz.node_remove.v1", nodeID, payload)
}

// DeleteEdge removes an edge by ID.
func (s *Store) DeleteEdge(ctx context.Context, edgeID string) error {
	if edgeID == "" {
		return errors.New("edge id is required")
	}
	res, err := s.db.ExecContext(ctx, `DELETE FROM edges WHERE id = ?`, edgeID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("edge not found")
	}
	payload, _ := json.Marshal(map[string]any{"edge_id": edgeID})
	return s.RecordEvent(ctx, "depviz.edge_delete.v1", edgeID, payload)
}

// DuplicateNode creates a copy of an existing node with "Copy of " prefix.
func (s *Store) DuplicateNode(ctx context.Context, boardID, nodeID string) (Node, error) {
	src, err := s.nodeByID(ctx, nodeID)
	if err != nil {
		return Node{}, fmt.Errorf("source node not found: %w", err)
	}
	var data map[string]any
	_ = json.Unmarshal([]byte(src.DataJSON), &data)
	if data == nil {
		data = map[string]any{}
	}
	description, _ := data["description"].(string)
	timeHorizon, _ := data["time_horizon"].(string)
	priority, _ := data["priority"].(string)
	var labelsList []string
	if ls, ok := data["labels"].([]interface{}); ok {
		for _, l := range ls {
			if lStr, ok := l.(string); ok {
				labelsList = append(labelsList, lStr)
			}
		}
	}
	return s.CreateStrategyNode(ctx, boardID, src.Kind, "Copy of "+src.Title, src.State, src.Owner, description, timeHorizon, priority, labelsList)
}

// ConvertNodeKind changes the kind of a local node.
func (s *Store) ConvertNodeKind(ctx context.Context, nodeID, newKind string) (Node, error) {
	n, err := s.nodeByID(ctx, nodeID)
	if err != nil {
		return Node{}, fmt.Errorf("node not found: %w", err)
	}
	validKinds := map[string]bool{
		"strategy": true, "initiative": true, "bet": true, "project": true,
		"workstream": true, "risk": true, "decision": true, "question": true,
		"metric": true, "task": true, "note": true,
	}
	if !validKinds[newKind] {
		return Node{}, errors.New("invalid kind")
	}
	n.Kind = newKind
	n.UpdatedAt = nowUTC()
	var data map[string]any
	_ = json.Unmarshal([]byte(n.DataJSON), &data)
	if data == nil {
		data = map[string]any{}
	}
	data["kind"] = newKind
	merged, _ := json.Marshal(data)
	n.DataJSON = string(merged)
	if err := s.UpsertNode(ctx, n); err != nil {
		return Node{}, err
	}
	evPayload, _ := json.Marshal(n)
	return n, s.RecordEvent(ctx, "depviz.node_convert.v1", n.ID, evPayload)
}

// ArchiveNode soft-archives a node by setting archived_at.
func (s *Store) ArchiveNode(ctx context.Context, nodeID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE nodes SET archived_at = ? WHERE id = ?`, formatTime(nowUTC()), nodeID)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"node_id": nodeID})
	return s.RecordEvent(ctx, "depviz.node_archive.v1", nodeID, payload)
}

// RestoreNode clears the archived_at field, making the node visible again.
func (s *Store) RestoreNode(ctx context.Context, nodeID string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE nodes SET archived_at = NULL WHERE id = ?`, nodeID)
	if err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"node_id": nodeID})
	return s.RecordEvent(ctx, "depviz.node_restore.v1", nodeID, payload)
}

// ListArchivedNodes returns nodes that have been soft-archived on a board.
func (s *Store) ListArchivedNodes(ctx context.Context, boardID string) ([]Node, error) {
	if boardID == "" {
		boardID = DefaultBoardID
	}
	rows, err := s.db.QueryContext(ctx, `SELECT n.id, n.kind, n.title, n.state, n.owner, n.data_json, n.updated_at
		FROM board_items bi
		JOIN nodes n ON n.id = bi.node_id
		WHERE bi.board_id = ? AND n.archived_at IS NOT NULL AND n.archived_at != ''
		ORDER BY n.updated_at DESC`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var nodes []Node
	for rows.Next() {
		var n Node
		var updated string
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.State, &n.Owner, &n.DataJSON, &updated); err != nil {
			return nil, err
		}
		n.UpdatedAt = parseTime(updated)
		nodes = append(nodes, n)
	}
	return nodes, rows.Err()
}

func (s *Store) AddGitHubRefToBoard(ctx context.Context, boardID, repo, marker, number, title string) (Node, error) {
	repo = strings.TrimSpace(repo)
	marker = strings.TrimSpace(marker)
	number = strings.TrimSpace(number)
	if repo == "" || number == "" {
		return Node{}, errors.New("github repo and number are required")
	}
	if marker != "!" {
		marker = "#"
	}
	id := "gh:" + repo + marker + number
	if err := s.ensureNodeInBoard(ctx, boardID, id); err != nil {
		return Node{}, err
	}
	if strings.TrimSpace(title) != "" {
		kind := "issue"
		if marker == "!" {
			kind = "pr"
		}
		n := Node{
			ID:        id,
			Kind:      kind,
			Title:     strings.TrimSpace(title),
			State:     "open",
			DataJSON:  `{"source":"github","manual":true}`,
			UpdatedAt: nowUTC(),
		}
		if err := s.UpsertNode(ctx, n); err != nil {
			return Node{}, err
		}
	}
	n, err := s.nodeByID(ctx, id)
	if err != nil {
		return Node{}, err
	}
	payload, _ := json.Marshal(map[string]any{"board": boardID, "repo": repo, "marker": marker, "number": number})
	return n, s.RecordEvent(ctx, "depviz.github_ref.v1", id, payload)
}

func (s *Store) Snapshot(ctx context.Context, boardID string) (Snapshot, error) {
	if boardID == "" {
		boardID = DefaultBoardID
	}
	board, err := s.board(ctx, boardID)
	if err != nil {
		return Snapshot{}, err
	}
	rows, err := s.db.QueryContext(ctx, `SELECT
			n.id, n.kind, n.title, n.state, n.owner, n.data_json, n.updated_at,
			bi.role, bi.local_state,
			COALESCE(sr.url, ''), COALESCE(sr.source_id, ''), COALESCE(sr.external_id, '')
		FROM board_items bi
		JOIN nodes n ON n.id = bi.node_id
		LEFT JOIN source_refs sr ON sr.node_id = n.id
		WHERE bi.board_id = ? AND (n.archived_at IS NULL OR n.archived_at = '')
		GROUP BY n.id
		ORDER BY n.updated_at DESC, n.id`, boardID)
	if err != nil {
		return Snapshot{}, err
	}
	defer rows.Close()
	var nodes []Node
	for rows.Next() {
		var n Node
		var updated string
		if err := rows.Scan(&n.ID, &n.Kind, &n.Title, &n.State, &n.Owner, &n.DataJSON, &updated, &n.BoardRole, &n.LocalState, &n.URL, &n.SourceID, &n.ExternalID); err != nil {
			return Snapshot{}, err
		}
		n.UpdatedAt = parseTime(updated)
		nodes = append(nodes, n)
	}
	if err := rows.Err(); err != nil {
		return Snapshot{}, err
	}
	edgeRows, err := s.db.QueryContext(ctx, `SELECT id, from_id, to_id, kind, scope_board_id, confidence, authority, evidence_json, observed_at
		FROM edges
		WHERE scope_board_id = ? OR scope_board_id = ''
		ORDER BY observed_at DESC, id`, boardID)
	if err != nil {
		return Snapshot{}, err
	}
	defer edgeRows.Close()
	var edges []Edge
	for edgeRows.Next() {
		var e Edge
		var observed string
		if err := edgeRows.Scan(&e.ID, &e.FromID, &e.ToID, &e.Kind, &e.ScopeBoardID, &e.Confidence, &e.Authority, &e.EvidenceJSON, &observed); err != nil {
			return Snapshot{}, err
		}
		e.ObservedAt = parseTime(observed)
		edges = append(edges, e)
	}
	return Snapshot{Board: board, Nodes: nodes, Edges: edges}, edgeRows.Err()
}

func (s *Store) IngestEvents(ctx context.Context, r io.Reader, defaultBoard string) (int, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 1024), 1024*1024)
	count := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if err := s.IngestEvent(ctx, []byte(line), defaultBoard); err != nil {
			return count, fmt.Errorf("line %d: %w", count+1, err)
		}
		count++
	}
	return count, scanner.Err()
}

func (s *Store) IngestEvent(ctx context.Context, data []byte, defaultBoard string) error {
	var ev struct {
		Type       string          `json:"type"`
		ID         string          `json:"id"`
		Kind       string          `json:"kind"`
		Title      string          `json:"title"`
		State      string          `json:"state"`
		Owner      string          `json:"owner"`
		Source     string          `json:"source"`
		ExternalID string          `json:"external_id"`
		URL        string          `json:"url"`
		Board      string          `json:"board"`
		Role       string          `json:"role"`
		From       string          `json:"from"`
		To         string          `json:"to"`
		Confidence float64         `json:"confidence"`
		Authority  string          `json:"authority"`
		Evidence   json.RawMessage `json:"evidence"`
	}
	if err := json.Unmarshal(data, &ev); err != nil {
		return err
	}
	if ev.Board == "" {
		ev.Board = defaultBoard
	}
	if ev.Board == "" {
		ev.Board = DefaultBoardID
	}
	switch ev.Type {
	case "depviz.edge.v1", "edge":
		if ev.Authority == "" {
			ev.Authority = "event"
		}
		if ev.Kind == "" {
			ev.Kind = "blocked_by"
		}
		_, err := s.AddEdgeWithConfidence(ctx, ev.Board, ev.From, ev.To, ev.Kind, ev.Authority, ev.Confidence, map[string]any{"event": json.RawMessage(data)})
		return err
	case "depviz.note.v1", "note":
		if ev.Title == "" {
			ev.Title = ev.ID
		}
		if ev.ID == "" {
			n, err := s.CreateNote(ctx, ev.Board, ev.Title)
			if err != nil {
				return err
			}
			ev.ID = n.ID
			return nil
		}
		ev.Kind = "note"
		ev.State = "local"
		return s.ingestNode(ctx, ev, data)
	case "depviz.node.v1", "node", "":
		return s.ingestNode(ctx, ev, data)
	case "depviz.source.v1", "source":
		return s.UpsertSource(ctx, Source{ID: ev.ID, Kind: ev.Kind, Name: ev.Title, URL: ev.URL, Capabilities: `{}`, Sync: `{}`, UpdatedAt: nowUTC()})
	default:
		return fmt.Errorf("unsupported event type %q", ev.Type)
	}
}

func (s *Store) ingestNode(ctx context.Context, ev struct {
	Type       string          `json:"type"`
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Title      string          `json:"title"`
	State      string          `json:"state"`
	Owner      string          `json:"owner"`
	Source     string          `json:"source"`
	ExternalID string          `json:"external_id"`
	URL        string          `json:"url"`
	Board      string          `json:"board"`
	Role       string          `json:"role"`
	From       string          `json:"from"`
	To         string          `json:"to"`
	Confidence float64         `json:"confidence"`
	Authority  string          `json:"authority"`
	Evidence   json.RawMessage `json:"evidence"`
}, data []byte) error {
	if ev.ID == "" {
		return errors.New("node id is required")
	}
	if ev.Source == "" {
		if strings.HasPrefix(ev.ID, "note:") {
			ev.Source = LocalSourceID
		} else {
			ev.Source = "events"
		}
	}
	if ev.ExternalID == "" {
		ev.ExternalID = ev.ID
	}
	if err := s.UpsertSource(ctx, Source{ID: ev.Source, Kind: sourceKind(ev.Source), Name: ev.Source, Capabilities: `{}`, Sync: `{}`, UpdatedAt: nowUTC()}); err != nil {
		return err
	}
	n := Node{
		ID:        ev.ID,
		Kind:      ev.Kind,
		Title:     ev.Title,
		State:     ev.State,
		Owner:     ev.Owner,
		DataJSON:  string(data),
		UpdatedAt: nowUTC(),
	}
	if err := s.UpsertNode(ctx, n); err != nil {
		return err
	}
	if err := s.UpsertSourceRef(ctx, ev.ID, ev.Source, ev.ExternalID, ev.URL); err != nil {
		return err
	}
	if err := s.AddNodeToBoard(ctx, ev.Board, ev.ID, ev.Role, ""); err != nil {
		return err
	}
	return s.RecordEvent(ctx, "depviz.node.v1", ev.ID, data)
}

func (s *Store) board(ctx context.Context, boardID string) (Board, error) {
	var b Board
	var updated string
	err := s.db.QueryRowContext(ctx, `SELECT id, name, description, scope_query, parent_board_id, config_json, updated_at FROM boards WHERE id = ?`, boardID).
		Scan(&b.ID, &b.Name, &b.Description, &b.ScopeQuery, &b.ParentBoardID, &b.ConfigJSON, &updated)
	if err != nil {
		return Board{}, err
	}
	b.UpdatedAt = parseTime(updated)
	return b, nil
}

func (s *Store) availableNoteID(ctx context.Context, text string) (string, error) {
	base := "note:" + slug(text)
	if base == "note:" {
		base = "note:untitled"
	}
	for i := 0; i < 1000; i++ {
		id := base
		if i > 0 {
			id = fmt.Sprintf("%s-%d", base, i+1)
		}
		var exists int
		if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM nodes WHERE id = ?`, id).Scan(&exists); err != nil {
			return "", err
		}
		if exists == 0 {
			return id, nil
		}
	}
	return "", errors.New("could not allocate note id")
}

func stableID(parts ...string) string {
	h := sha1.New()
	for _, p := range parts {
		_, _ = h.Write([]byte(p))
		_, _ = h.Write([]byte{0})
	}
	return parts[0] + ":" + hex.EncodeToString(h.Sum(nil))[:16]
}

var slugRE = regexp.MustCompile(`[^a-z0-9]+`)
var githubNodeRE = regexp.MustCompile(`^gh:([^#!]+)([#!])([0-9]+)$`)

func slug(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = slugRE.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")
	if len(s) > 64 {
		s = strings.Trim(s[:64], "-")
	}
	return s
}

func sourceKind(id string) string {
	if strings.HasPrefix(id, "github:") {
		return "github"
	}
	if id == LocalSourceID {
		return "local"
	}
	return "events"
}

func placeholderNode(id string) (Node, string, string, string) {
	now := nowUTC()
	if strings.HasPrefix(id, "note:") {
		return Node{ID: id, Kind: "note", Title: strings.TrimPrefix(id, "note:"), State: "local", DataJSON: `{"placeholder":true}`, UpdatedAt: now}, LocalSourceID, id, ""
	}
	if m := githubNodeRE.FindStringSubmatch(id); m != nil {
		repo := m[1]
		marker := m[2]
		number := m[3]
		kind := "issue"
		path := "issues"
		if marker == "!" {
			kind = "pr"
			path = "pull"
		}
		sourceID := "github:" + repo
		url := fmt.Sprintf("https://github.com/%s/%s/%s", repo, path, number)
		externalID := marker + number
		return Node{ID: id, Kind: kind, Title: id, State: "open", DataJSON: `{"placeholder":true}`, UpdatedAt: now}, sourceID, externalID, url
	}
	return Node{ID: id, Kind: "task", Title: id, State: "open", DataJSON: `{"placeholder":true}`, UpdatedAt: now}, LocalSourceID, id, ""
}

func sourceURL(sourceID string) string {
	if strings.HasPrefix(sourceID, "github:") {
		return "https://github.com/" + strings.TrimPrefix(sourceID, "github:")
	}
	return ""
}

type BoardView struct {
	ID         string `json:"id"`
	BoardID    string `json:"board_id"`
	Name       string `json:"name"`
	ConfigJSON string `json:"config_json"`
	CreatedAt  string `json:"created_at"`
}

func (s *Store) SaveBoardView(ctx context.Context, boardID, name string, config map[string]any) (BoardView, error) {
	id := fmt.Sprintf("view-%d", time.Now().UnixNano())
	now := formatTime(nowUTC())
	configJSON, _ := json.Marshal(config)
	_, err := s.db.ExecContext(ctx, `INSERT INTO board_views(id, board_id, name, config_json, created_at) VALUES(?, ?, ?, ?, ?)`,
		id, boardID, name, string(configJSON), now)
	if err != nil {
		return BoardView{}, err
	}
	return BoardView{ID: id, BoardID: boardID, Name: name, ConfigJSON: string(configJSON), CreatedAt: now}, nil
}

func (s *Store) ListBoardViews(ctx context.Context, boardID string) ([]BoardView, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, board_id, name, config_json, created_at FROM board_views WHERE board_id=? ORDER BY created_at DESC`, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BoardView
	for rows.Next() {
		var v BoardView
		if err := rows.Scan(&v.ID, &v.BoardID, &v.Name, &v.ConfigJSON, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (s *Store) DeleteBoardView(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM board_views WHERE id=?`, id)
	return err
}

type SyncLog struct {
	ID                 string `json:"id"`
	BoardID            string `json:"board_id"`
	StartedAt          string `json:"started_at"`
	CompletedAt        string `json:"completed_at"`
	Status             string `json:"status"`
	ItemsSynced        int    `json:"items_synced"`
	EdgesSynced        int    `json:"edges_synced"`
	Mode               string `json:"mode"`
	Error              string `json:"error"`
	RateLimitRemaining int    `json:"rate_limit_remaining"`
	RateLimitReset     string `json:"rate_limit_reset"`
}

func (s *Store) AddSyncLog(ctx context.Context, log SyncLog) error {
	if log.ID == "" {
		log.ID = fmt.Sprintf("sync-%d", time.Now().UnixNano())
	}
	if log.StartedAt == "" {
		log.StartedAt = formatTime(nowUTC())
	}
	_, err := s.db.ExecContext(ctx, `INSERT OR REPLACE INTO sync_logs(id, board_id, started_at, completed_at, status, items_synced, edges_synced, mode, error, rate_limit_remaining, rate_limit_reset)
		VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		log.ID, log.BoardID, log.StartedAt, log.CompletedAt, log.Status, log.ItemsSynced, log.EdgesSynced, log.Mode, log.Error, log.RateLimitRemaining, log.RateLimitReset)
	return err
}

func (s *Store) GetSyncLogs(ctx context.Context, boardID string, limit int) ([]SyncLog, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT id, board_id, started_at, completed_at, status, items_synced, edges_synced, mode, error, rate_limit_remaining, rate_limit_reset FROM sync_logs WHERE board_id=? ORDER BY started_at DESC LIMIT ?`, boardID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SyncLog
	for rows.Next() {
		var l SyncLog
		if err := rows.Scan(&l.ID, &l.BoardID, &l.StartedAt, &l.CompletedAt, &l.Status, &l.ItemsSynced, &l.EdgesSynced, &l.Mode, &l.Error, &l.RateLimitRemaining, &l.RateLimitReset); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CancelSyncLog marks a queued or running sync log entry as cancelled.
func (s *Store) CancelSyncLog(ctx context.Context, id string) error {
	now := formatTime(nowUTC())
	_, err := s.db.ExecContext(ctx, `UPDATE sync_logs SET status = 'cancelled', completed_at = ? WHERE id = ? AND status IN ('queued', 'running')`, now, id)
	return err
}

func (s *Store) DismissSuggestion(ctx context.Context, accountID, boardID, edgeID string) error {
	now := formatTime(nowUTC())
	_, err := s.db.ExecContext(ctx, `INSERT INTO dismissed_suggestions(account_id, board_id, edge_id, dismissed_at)
		VALUES(?, ?, ?, ?)
		ON CONFLICT(account_id, board_id, edge_id) DO UPDATE SET dismissed_at=excluded.dismissed_at`,
		accountID, boardID, edgeID, now)
	return err
}

func (s *Store) RestoreSuggestion(ctx context.Context, accountID, boardID, edgeID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM dismissed_suggestions WHERE account_id=? AND board_id=? AND edge_id=?`,
		accountID, boardID, edgeID)
	return err
}

func (s *Store) ListDismissedSuggestions(ctx context.Context, accountID, boardID string) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT edge_id FROM dismissed_suggestions WHERE account_id=? AND board_id=?`, accountID, boardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		t = nowUTC()
	}
	return t.UTC().Format(time.RFC3339)
}

func parseTime(s string) time.Time {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return time.Time{}
	}
	return t
}

func maxParsedTime(values ...string) time.Time {
	var out time.Time
	for _, value := range values {
		t := parseTime(value)
		if t.After(out) {
			out = t
		}
	}
	return out
}

func sortBriefItems(items []BriefItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Impact != items[j].Impact {
			return items[i].Impact > items[j].Impact
		}
		if items[i].BlockerCount != items[j].BlockerCount {
			return items[i].BlockerCount < items[j].BlockerCount
		}
		return items[i].ID < items[j].ID
	})
}

// BoardUpdatedAt returns the updated_at timestamp of the given board.
func (s *Store) BoardUpdatedAt(ctx context.Context, boardID string) (time.Time, error) {
	var updated string
	err := s.db.QueryRowContext(ctx, `SELECT updated_at FROM boards WHERE id = ?`, boardID).Scan(&updated)
	if err != nil {
		return time.Time{}, err
	}
	return parseTime(updated), nil
}

// WithTx runs fn inside a single SQLite transaction, rolling back on error.
func (s *Store) WithTx(ctx context.Context, fn func(context.Context, *sql.Tx) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(ctx, tx); err != nil {
		return err
	}
	return tx.Commit()
}
