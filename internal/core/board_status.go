package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

type BoardStatusBrief struct {
	BoardName     string             `json:"board_name"`
	Counts        BoardStatusCounts  `json:"counts"`
	Statuses      []BoardStatusCount `json:"statuses"`
	Deltas        []BoardStatusDelta `json:"deltas,omitempty"`
	Pullable      []BriefItem        `json:"pullable"`
	Blocked       []BriefItem        `json:"blocked"`
	Untriaged     []BriefItem        `json:"untriaged"`
	SnapshotCheck *SnapshotFreshness `json:"snapshot_check,omitempty"`
	Warnings      []string           `json:"warnings,omitempty"`
}

type BoardStatusCounts struct {
	Nodes       int `json:"nodes"`
	Open        int `json:"open"`
	Closed      int `json:"closed"`
	Edges       int `json:"edges"`
	Pullable    int `json:"pullable"`
	Blocked     int `json:"blocked"`
	Untriaged   int `json:"untriaged"`
	DroppedEdge int `json:"dropped_edges"`
}

type BoardStatusCount struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type BoardStatusDelta struct {
	Status string `json:"status"`
	Now    int    `json:"now"`
	Before int    `json:"before"`
	Delta  int    `json:"delta"`
}

type SnapshotFreshness struct {
	Checked        bool   `json:"checked"`
	LiveReady      []int  `json:"live_ready"`
	SnapshotReady  []int  `json:"snapshot_ready"`
	Disagreement   bool   `json:"disagreement"`
	Message        string `json:"message"`
	SnapshotAgeSec int    `json:"snapshot_age_sec,omitempty"`
}

func (s *Store) BuildBoardStatusBrief(ctx context.Context, boardID string) (BoardStatusBrief, error) {
	snap, err := s.Snapshot(ctx, boardID)
	if err != nil {
		return BoardStatusBrief{}, err
	}
	nodes := map[string]Node{}
	for _, n := range snap.Nodes {
		nodes[n.ID] = n
	}
	blockersByNode, droppedEdges := boardStatusBlockers(snap.Edges, nodes)
	statusCounts := map[string]int{}
	var pullable, blocked, untriaged []BriefItem
	openCount := 0
	closedCount := 0
	for _, n := range snap.Nodes {
		if n.IsClosed() {
			closedCount++
			continue
		}
		openCount++
		statuses := statusLabels(n)
		if len(statuses) == 0 {
			untriaged = append(untriaged, boardStatusItem(n, "open issue has no status:* label", 0))
			statusCounts["untriaged"]++
			continue
		}
		for _, status := range statuses {
			statusCounts[status]++
		}
		if !hasString(statuses, "ready") {
			continue
		}
		active := activeBlockers(n.ID, nodes, blockersByNode)
		if len(active) > 0 {
			blocked = append(blocked, boardStatusItem(n, fmt.Sprintf("status:ready but blocked by %s", strings.Join(active, ", ")), len(active)))
			continue
		}
		pullable = append(pullable, boardStatusItem(n, "status:ready and no live blocker", 0))
	}
	sortBriefItems(pullable)
	sortBriefItems(blocked)
	sortBriefItems(untriaged)
	statuses := boardStatusCounts(statusCounts)
	b := BoardStatusBrief{
		BoardName: snap.Board.Name,
		Counts: BoardStatusCounts{
			Nodes:       len(snap.Nodes),
			Open:        openCount,
			Closed:      closedCount,
			Edges:       len(snap.Edges),
			Pullable:    len(pullable),
			Blocked:     len(blocked),
			Untriaged:   len(untriaged),
			DroppedEdge: droppedEdges,
		},
		Statuses:  statuses,
		Deltas:    deltas(statuses, s.lastBoardStatusCounts(ctx, boardID)),
		Pullable:  limitItems(pullable, 12),
		Blocked:   limitItems(blocked, 12),
		Untriaged: limitItems(untriaged, 12),
	}
	if repo := boardStatusRepo(snap.Nodes); repo == "1789-tech/job-board" {
		check := checkBoardSnapshot(ctx, repo, pullable)
		b.SnapshotCheck = &check
		if check.Disagreement {
			b.Warnings = append(b.Warnings, check.Message)
		}
	}
	return b, nil
}

func RenderBoardStatusBrief(w io.Writer, b BoardStatusBrief) error {
	write := func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, format, args...)
	}
	write("DepViz board-status brief: %s\n", b.BoardName)
	write("Status histogram (day-over-day delta)\n")
	if len(b.Statuses) == 0 {
		write("  none\n")
	} else {
		deltasByStatus := map[string]BoardStatusDelta{}
		for _, d := range b.Deltas {
			deltasByStatus[d.Status] = d
		}
		for _, s := range b.Statuses {
			suffix := " (delta n/a)"
			if d, ok := deltasByStatus[s.Status]; ok {
				suffix = fmt.Sprintf(" (%+d)", d.Delta)
			}
			write("  status:%s %d%s\n", s.Status, s.Count, suffix)
		}
	}
	write("\n")
	write("Open: %d - closed: %d - pullable: %d - blocked ready: %d - untriaged: %d - edges: %d - dropped edges: %d\n\n",
		b.Counts.Open, b.Counts.Closed, b.Counts.Pullable, b.Counts.Blocked, b.Counts.Untriaged, b.Counts.Edges, b.Counts.DroppedEdge)
	writeSection(w, "Pullable now", b.Pullable, false, true)
	writeSection(w, "Blocked ready", b.Blocked, false, true)
	writeSection(w, "Untriaged", b.Untriaged, false, true)
	if b.SnapshotCheck != nil {
		write("Snapshot freshness\n")
		write("  %s\n", b.SnapshotCheck.Message)
		write("\n")
	}
	if len(b.Warnings) > 0 {
		write("Warnings\n")
		for _, warning := range b.Warnings {
			write("  %s\n", warning)
		}
	}
	return nil
}

func (s *Store) RecordBoardStatusHistogram(ctx context.Context, boardID string, statuses []BoardStatusCount) error {
	if boardID == "" {
		boardID = DefaultBoardID
	}
	counts := map[string]int{}
	for _, status := range statuses {
		counts[status.Status] = status.Count
	}
	payload, _ := json.Marshal(map[string]any{
		"board_id": boardID,
		"counts":   counts,
	})
	return s.RecordEvent(ctx, "depviz.board_status_histogram.v1", boardID, payload)
}

func (s *Store) lastBoardStatusCounts(ctx context.Context, boardID string) map[string]int {
	var payload string
	err := s.db.QueryRowContext(ctx, `SELECT data_json FROM events
		WHERE type = ? AND object_id = ?
		ORDER BY seq DESC LIMIT 1`, "depviz.board_status_histogram.v1", boardID).Scan(&payload)
	if err != nil {
		return nil
	}
	var data struct {
		Counts map[string]int `json:"counts"`
	}
	if err := json.Unmarshal([]byte(payload), &data); err != nil {
		return nil
	}
	return data.Counts
}

func boardStatusBlockers(edges []Edge, nodes map[string]Node) (map[string]map[string]bool, int) {
	blockersByNode := map[string]map[string]bool{}
	dropped := 0
	for _, e := range edges {
		blocked, blocker := boardStatusBlockedAndBlocker(e)
		if blocked == "" || blocker == "" {
			continue
		}
		blockedNode, blockedOK := nodes[blocked]
		blockerNode, blockerOK := nodes[blocker]
		if !blockedOK || !blockerOK || blockedNode.IsClosed() || blockerNode.IsClosed() || !boardStatusExplicitBlocker(e) {
			dropped++
			continue
		}
		if blockersByNode[blocked] == nil {
			blockersByNode[blocked] = map[string]bool{}
		}
		blockersByNode[blocked][blocker] = true
	}
	return blockersByNode, dropped
}

func boardStatusBlockedAndBlocker(e Edge) (blocked string, blocker string) {
	switch strings.ToLower(strings.TrimSpace(e.Kind)) {
	case "blocked_by", "depends_on", "depends":
		return e.FromID, e.ToID
	case "blocks":
		return e.ToID, e.FromID
	default:
		return "", ""
	}
}

func boardStatusExplicitBlocker(e Edge) bool {
	if !edgeIsSoft(e) {
		return true
	}
	var evidence struct {
		Line string `json:"line"`
	}
	if err := json.Unmarshal([]byte(e.EvidenceJSON), &evidence); err != nil {
		return false
	}
	line := strings.TrimSpace(evidence.Line)
	return explicitDependencyLineRE.MatchString(line) || checkboxDependencyLineRE.MatchString(line)
}

var (
	explicitDependencyLineRE = regexp.MustCompile(`(?i)^\s*(?:[-*]\s*)?(?:blocked by|depends on)\s*:\s+`)
	checkboxDependencyLineRE = regexp.MustCompile(`(?i)^\s*[-*]\s+\[[ xX]\]\s+(?:blocked by|depends on)?\s*(?:gh:)?(?:[A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)?[#!][0-9]+`)
)

func boardStatusItem(n Node, reason string, blockerCount int) BriefItem {
	return BriefItem{
		ID:           n.ID,
		Title:        n.Title,
		Kind:         n.Kind,
		State:        n.State,
		URL:          n.URL,
		Reason:       reason,
		BlockerCount: blockerCount,
	}
}

func statusLabels(n Node) []string {
	var statuses []string
	for _, label := range n.Labels() {
		if status, ok := strings.CutPrefix(label, "status:"); ok && strings.TrimSpace(status) != "" {
			statuses = append(statuses, strings.TrimSpace(status))
		}
	}
	sort.Strings(statuses)
	return statuses
}

func boardStatusCounts(counts map[string]int) []BoardStatusCount {
	statuses := make([]BoardStatusCount, 0, len(counts))
	for status, count := range counts {
		statuses = append(statuses, BoardStatusCount{Status: status, Count: count})
	}
	sort.Slice(statuses, func(i, j int) bool {
		return statusRank(statuses[i].Status) < statusRank(statuses[j].Status)
	})
	return statuses
}

func statusRank(status string) int {
	order := []string{"ready", "active", "review", "blocked", "parked", "done", "untriaged"}
	for i, candidate := range order {
		if status == candidate {
			return i
		}
	}
	if status == "" {
		return len(order)
	}
	return len(order) + int(status[0])
}

func deltas(now []BoardStatusCount, before map[string]int) []BoardStatusDelta {
	if len(before) == 0 {
		return nil
	}
	seen := map[string]bool{}
	var out []BoardStatusDelta
	for _, current := range now {
		previous := before[current.Status]
		out = append(out, BoardStatusDelta{Status: current.Status, Now: current.Count, Before: previous, Delta: current.Count - previous})
		seen[current.Status] = true
	}
	for status, previous := range before {
		if seen[status] {
			continue
		}
		out = append(out, BoardStatusDelta{Status: status, Now: 0, Before: previous, Delta: -previous})
	}
	sort.Slice(out, func(i, j int) bool {
		return statusRank(out[i].Status) < statusRank(out[j].Status)
	})
	return out
}

func hasString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func boardStatusRepo(nodes []Node) string {
	repos := map[string]int{}
	for _, n := range nodes {
		var payload struct {
			Repo string `json:"repo"`
		}
		if json.Unmarshal([]byte(n.DataJSON), &payload) == nil && payload.Repo != "" {
			repos[payload.Repo]++
		}
	}
	bestRepo := ""
	bestCount := 0
	for repo, count := range repos {
		if count > bestCount {
			bestRepo = repo
			bestCount = count
		}
	}
	return bestRepo
}

func checkBoardSnapshot(ctx context.Context, repo string, pullable []BriefItem) SnapshotFreshness {
	liveReady := issueNumbers(pullable)
	check := SnapshotFreshness{Checked: true, LiveReady: liveReady}
	cmd := exec.CommandContext(ctx, "gh", "api", "--header", "Accept: application/vnd.github.raw", "/repos/moul/1789.tech/contents/hermes/state/board-snapshot.json")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		check.Message = fmt.Sprintf("snapshot unavailable: %s", strings.TrimSpace(stderr.String()))
		return check
	}
	var payload struct {
		GeneratedAt string `json:"generated_at"`
		Queue       []struct {
			Number int    `json:"number"`
			Status string `json:"status"`
		} `json:"queue"`
	}
	if err := json.Unmarshal(out, &payload); err != nil {
		check.Message = "snapshot unavailable: could not parse board-snapshot.json"
		return check
	}
	for _, item := range payload.Queue {
		if item.Status == "ready" && item.Number > 0 {
			check.SnapshotReady = append(check.SnapshotReady, item.Number)
		}
	}
	sort.Ints(check.SnapshotReady)
	check.Disagreement = !sameInts(check.LiveReady, check.SnapshotReady)
	if check.Disagreement {
		check.Message = fmt.Sprintf("snapshot ready %v disagrees with live pullable %v", check.SnapshotReady, check.LiveReady)
	} else {
		check.Message = fmt.Sprintf("snapshot agrees with live pullable %v", check.LiveReady)
	}
	return check
}

func issueNumbers(items []BriefItem) []int {
	var numbers []int
	for _, item := range items {
		n, ok := issueNumber(item.ID)
		if ok {
			numbers = append(numbers, n)
		}
	}
	sort.Ints(numbers)
	return numbers
}

func issueNumber(id string) (int, bool) {
	hash := strings.LastIndex(id, "#")
	if hash < 0 {
		return 0, false
	}
	var n int
	_, err := fmt.Sscanf(id[hash+1:], "%d", &n)
	return n, err == nil
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
