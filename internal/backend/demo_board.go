package backend

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"moul.io/depviz/v4/internal/core"
)

const defaultDemoBoardMaxAge = 30 * time.Minute

type demoBoardPayload struct {
	Snapshot            core.Snapshot          `json:"snapshot"`
	Brief               core.BoardStatusBrief  `json:"brief"`
	BriefType           string                 `json:"brief_type"`
	SnapshotGeneratedAt string                 `json:"snapshot_generated_at"`
	SnapshotStale       bool                   `json:"snapshot_stale"`
	SnapshotAgeSec      int                    `json:"snapshot_age_sec,omitempty"`
	Warnings            []string               `json:"warnings,omitempty"`
}

type hermesBoardSnapshot struct {
	GeneratedAt string             `json:"generated_at"`
	Queue       []hermesBoardIssue `json:"queue"`
	Open        []hermesBoardIssue `json:"open"`
	Done        []hermesBoardIssue `json:"done"`
}

type hermesBoardIssue struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	URL     string `json:"url"`
	Project string `json:"project"`
	Status  string `json:"status"`
	Prio    string `json:"prio"`
	Type    string `json:"type"`
	Rank    int    `json:"rank"`
}

func (s *Server) demoBoardConfigured() bool {
	return strings.TrimSpace(s.cfg.DemoBoardSnapshotFile) != ""
}

func (s *Server) handleDemoBoard(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", http.MethodGet)
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	if !s.demoBoardConfigured() {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "demo board not configured"})
		return
	}
	if !s.basicAuthAuthorized(r) {
		writeJSON(w, http.StatusForbidden, map[string]string{"error": "demo board requires basic auth"})
		return
	}
	payload, err := s.loadDemoBoard()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, payload)
}

func (s *Server) loadDemoBoard() (demoBoardPayload, error) {
	f, err := os.Open(s.cfg.DemoBoardSnapshotFile)
	if err != nil {
		return demoBoardPayload{}, fmt.Errorf("open demo board snapshot: %w", err)
	}
	defer f.Close()
	return parseDemoBoard(f, s.demoBoardMaxAge())
}

func (s *Server) demoBoardMaxAge() time.Duration {
	if s.cfg.DemoBoardMaxAge > 0 {
		return s.cfg.DemoBoardMaxAge
	}
	return defaultDemoBoardMaxAge
}

func parseDemoBoard(r io.Reader, maxAge time.Duration) (demoBoardPayload, error) {
	var hermes hermesBoardSnapshot
	if err := json.NewDecoder(r).Decode(&hermes); err != nil {
		return demoBoardPayload{}, fmt.Errorf("parse demo board snapshot: %w", err)
	}
	generatedAt, err := time.Parse(time.RFC3339, hermes.GeneratedAt)
	if err != nil {
		return demoBoardPayload{}, fmt.Errorf("parse generated_at: %w", err)
	}
	snap := hermes.toDepVizSnapshot(generatedAt)
	brief := core.BuildBoardStatusBriefFromSnapshot(snap, nil)
	age := time.Since(generatedAt)
	stale := maxAge > 0 && age > maxAge
	warnings := append([]string{}, brief.Warnings...)
	if stale {
		msg := fmt.Sprintf("snapshot stale: generated %s", generatedAt.UTC().Format(time.RFC3339))
		warnings = append(warnings, msg)
		brief.Warnings = append(brief.Warnings, msg)
		brief.SnapshotCheck = &core.SnapshotFreshness{
			Checked:        true,
			Disagreement:   false,
			Message:        msg,
			SnapshotAgeSec: int(age.Seconds()),
		}
	}
	return demoBoardPayload{
		Snapshot:            snap,
		Brief:               brief,
		BriefType:           "board-status",
		SnapshotGeneratedAt: generatedAt.UTC().Format(time.RFC3339),
		SnapshotStale:       stale,
		SnapshotAgeSec:      int(age.Seconds()),
		Warnings:            warnings,
	}, nil
}

func (h hermesBoardSnapshot) toDepVizSnapshot(generatedAt time.Time) core.Snapshot {
	issues := map[int]hermesBoardIssue{}
	for _, rows := range [][]hermesBoardIssue{h.Open, h.Done, h.Queue} {
		for _, issue := range rows {
			if issue.Number == 0 {
				continue
			}
			issues[issue.Number] = issue
		}
	}
	numbers := make([]int, 0, len(issues))
	for n := range issues {
		numbers = append(numbers, n)
	}
	sort.Ints(numbers)
	nodes := make([]core.Node, 0, len(numbers))
	for _, number := range numbers {
		nodes = append(nodes, hermesIssueToNode(issues[number], generatedAt))
	}
	return core.Snapshot{
		Board: core.Board{
			ID:          "1789-job-board",
			Name:        "1789 job-board",
			Description: "Private 1789 delegation board snapshot",
			ScopeQuery:  "repo:1789-tech/job-board",
			UpdatedAt:   generatedAt,
			Metrics: &core.BoardMetrics{
				Items:      len(nodes),
				Open:       countOpen(nodes),
				Closed:     len(nodes) - countOpen(nodes),
				External:   len(nodes),
				LastSyncAt: generatedAt,
				SyncStatus: "snapshot",
			},
		},
		Nodes: nodes,
		Edges: nil,
	}
}

func hermesIssueToNode(issue hermesBoardIssue, generatedAt time.Time) core.Node {
	status := strings.TrimPrefix(issue.Status, "status:")
	state := status
	if state == "" {
		state = "open"
	}
	labels := []string{}
	addLabel := func(prefix, value string) {
		value = strings.TrimSpace(strings.TrimPrefix(value, prefix))
		if value != "" {
			labels = append(labels, prefix+value)
		}
	}
	addLabel("status:", status)
	addLabel("project:", issue.Project)
	addLabel("prio:", issue.Prio)
	addLabel("type:", issue.Type)
	data, _ := json.Marshal(map[string]any{
		"repo":   "1789-tech/job-board",
		"number": issue.Number,
		"labels": labels,
		"rank":   issue.Rank,
	})
	return core.Node{
		ID:         fmt.Sprintf("gh:1789-tech/job-board#%d", issue.Number),
		Kind:       "issue",
		Title:      issue.Title,
		State:      state,
		DataJSON:   string(data),
		UpdatedAt:  generatedAt,
		URL:        issue.URL,
		SourceID:   "github:1789-tech/job-board",
		ExternalID: fmt.Sprintf("#%d", issue.Number),
	}
}

func countOpen(nodes []core.Node) int {
	open := 0
	for _, n := range nodes {
		if !n.IsClosed() {
			open++
		}
	}
	return open
}
