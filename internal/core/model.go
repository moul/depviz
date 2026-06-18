package core

import (
	"encoding/json"
	"strings"
	"time"
)

const (
	DefaultBoardID = "default"
	LocalSourceID  = "local"
	DefaultDBPath  = ".depviz/state.db"
)

type Source struct {
	ID           string    `json:"id"`
	Kind         string    `json:"kind"`
	Name         string    `json:"name"`
	URL          string    `json:"url"`
	Capabilities string    `json:"capabilities_json"`
	Sync         string    `json:"sync_json"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type Node struct {
	ID         string    `json:"id"`
	Kind       string    `json:"kind"`
	Title      string    `json:"title"`
	State      string    `json:"state"`
	Owner      string    `json:"owner"`
	DataJSON   string    `json:"data_json"`
	UpdatedAt  time.Time `json:"updated_at"`
	BoardRole  string    `json:"board_role"`
	LocalState string    `json:"local_state"`
	URL        string    `json:"url"`
	SourceID   string    `json:"source_id"`
	ExternalID string    `json:"external_id"`
}

type Edge struct {
	ID           string    `json:"id"`
	FromID       string    `json:"from_id"`
	ToID         string    `json:"to_id"`
	Kind         string    `json:"kind"`
	ScopeBoardID string    `json:"scope_board_id"`
	Confidence   float64   `json:"confidence"`
	Authority    string    `json:"authority"`
	EvidenceJSON string    `json:"evidence_json"`
	ObservedAt   time.Time `json:"observed_at"`
}

type Board struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	ScopeQuery    string    `json:"scope_query"`
	ParentBoardID string    `json:"parent_board_id"`
	ConfigJSON    string    `json:"config_json"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type Snapshot struct {
	Board Board  `json:"board"`
	Nodes []Node `json:"nodes"`
	Edges []Edge `json:"edges"`
}

type Export struct {
	Snapshot Snapshot `json:"snapshot"`
	Brief    Brief    `json:"brief"`
}

type Brief struct {
	BoardName string      `json:"board_name"`
	NextMove  *BriefItem  `json:"next_move"`
	Ready     []BriefItem `json:"ready"`
	Blockers  []BriefItem `json:"blockers"`
	LocalOnly []BriefItem `json:"local_only"`
	Stale     []BriefItem `json:"stale"`
	Counts    BriefCounts `json:"counts"`
}

type BriefCounts struct {
	Nodes     int `json:"nodes"`
	Edges     int `json:"edges"`
	Ready     int `json:"ready"`
	Blocked   int `json:"blocked"`
	LocalOnly int `json:"local_only"`
	Stale     int `json:"stale"`
}

type BriefItem struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	Kind         string `json:"kind"`
	State        string `json:"state"`
	URL          string `json:"url,omitempty"`
	Reason       string `json:"reason,omitempty"`
	Impact       int    `json:"impact,omitempty"`
	BlockerCount int    `json:"blocker_count,omitempty"`
}

type Account struct {
	ID              string    `json:"id"`
	PrimaryProvider string    `json:"primary_provider"`
	Login           string    `json:"login"`
	Name            string    `json:"name"`
	AvatarURL       string    `json:"avatar_url"`
	HTMLURL         string    `json:"html_url"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func (n Node) IsClosed() bool {
	switch strings.ToLower(strings.TrimSpace(n.State)) {
	case "closed", "done", "merged", "cancelled", "canceled", "resolved":
		return true
	default:
		return false
	}
}

func (n Node) IsLocalOnly() bool {
	return n.Kind == "note" || strings.HasPrefix(n.ID, "note:")
}

func (n Node) IsPlaceholder() bool {
	var payload struct {
		Placeholder bool `json:"placeholder"`
	}
	if n.DataJSON == "" {
		return false
	}
	if err := json.Unmarshal([]byte(n.DataJSON), &payload); err != nil {
		return false
	}
	return payload.Placeholder
}

func (n Node) Labels() []string {
	var payload struct {
		Labels []string `json:"labels"`
	}
	if n.DataJSON == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(n.DataJSON), &payload); err != nil {
		return nil
	}
	return payload.Labels
}

func nowUTC() time.Time {
	return time.Now().UTC().Truncate(time.Second)
}
