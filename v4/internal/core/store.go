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
	}
	for _, stmt := range stmts {
		if _, err := s.db.ExecContext(ctx, stmt); err != nil {
			return err
		}
	}
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
	payload, _ := json.Marshal(map[string]any{"source": "local", "text": text})
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
		Confidence:   1,
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
		boards = append(boards, b)
	}
	return boards, rows.Err()
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
		WHERE bi.board_id = ?
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
		_, err := s.AddEdge(ctx, ev.Board, ev.From, ev.To, ev.Kind, ev.Authority, map[string]any{"event": json.RawMessage(data)})
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
