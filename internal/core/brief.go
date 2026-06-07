package core

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"
)

func (s *Store) BuildBrief(ctx context.Context, boardID string) (Brief, error) {
	snap, err := s.Snapshot(ctx, boardID)
	if err != nil {
		return Brief{}, err
	}
	nodes := map[string]Node{}
	for _, n := range snap.Nodes {
		nodes[n.ID] = n
	}
	blockersByNode := map[string]map[string]bool{}
	blockedByNode := map[string]map[string]bool{}
	for _, e := range snap.Edges {
		blocked, blocker := edgeBlockedAndBlocker(e)
		if blocked == "" || blocker == "" {
			continue
		}
		if _, ok := nodes[blocked]; !ok {
			continue
		}
		if _, ok := nodes[blocker]; !ok {
			continue
		}
		if blockersByNode[blocked] == nil {
			blockersByNode[blocked] = map[string]bool{}
		}
		if blockedByNode[blocker] == nil {
			blockedByNode[blocker] = map[string]bool{}
		}
		blockersByNode[blocked][blocker] = true
		blockedByNode[blocker][blocked] = true
	}
	var ready, blockers, localOnly, stale []BriefItem
	blockedCount := 0
	cutoff := nowUTC().Add(-30 * 24 * time.Hour)
	for _, n := range snap.Nodes {
		if n.IsClosed() {
			continue
		}
		activeBlockers := activeBlockers(n.ID, nodes, blockersByNode)
		if len(activeBlockers) == 0 && !n.IsPlaceholder() {
			ready = append(ready, BriefItem{
				ID:     n.ID,
				Title:  n.Title,
				Kind:   n.Kind,
				State:  n.State,
				URL:    n.URL,
				Reason: readyReason(n, blockedByNode[n.ID]),
				Impact: activeBlockedCount(n.ID, nodes, blockedByNode),
			})
		} else {
			blockedCount++
		}
		if n.IsLocalOnly() {
			localOnly = append(localOnly, BriefItem{ID: n.ID, Title: n.Title, Kind: n.Kind, State: n.State, Reason: "local-only planning card"})
		}
		if n.IsPlaceholder() && !n.IsLocalOnly() {
			stale = append(stale, BriefItem{ID: n.ID, Title: n.Title, Kind: n.Kind, State: n.State, URL: n.URL, Reason: "placeholder external ref; sync a wider scope"})
			continue
		}
		if !n.IsLocalOnly() && !n.UpdatedAt.IsZero() && n.UpdatedAt.Before(cutoff) {
			stale = append(stale, BriefItem{ID: n.ID, Title: n.Title, Kind: n.Kind, State: n.State, URL: n.URL, Reason: "not updated in 30+ days"})
		}
	}
	for blockerID := range blockedByNode {
		n := nodes[blockerID]
		if n.IsClosed() {
			continue
		}
		count := activeBlockedCount(blockerID, nodes, blockedByNode)
		if count == 0 {
			continue
		}
		blockers = append(blockers, BriefItem{
			ID:     n.ID,
			Title:  n.Title,
			Kind:   n.Kind,
			State:  n.State,
			URL:    n.URL,
			Impact: count,
			Reason: fmt.Sprintf("blocks %d active card%s", count, plural(count)),
		})
	}
	sortBriefItems(ready)
	sortBriefItems(blockers)
	sortBriefItems(localOnly)
	sortBriefItems(stale)
	b := Brief{
		BoardName: snap.Board.Name,
		Ready:     limitItems(ready, 12),
		Blockers:  limitItems(blockers, 12),
		LocalOnly: limitItems(localOnly, 12),
		Stale:     limitItems(stale, 12),
		Counts: BriefCounts{
			Nodes:     len(snap.Nodes),
			Edges:     len(snap.Edges),
			Ready:     len(ready),
			Blocked:   blockedCount,
			LocalOnly: len(localOnly),
			Stale:     len(stale),
		},
	}
	if len(ready) > 0 {
		next := ready[0]
		b.NextMove = &next
	}
	return b, nil
}

func RenderBrief(w io.Writer, b Brief) error {
	write := func(format string, args ...any) {
		_, _ = fmt.Fprintf(w, format, args...)
	}
	write("DepViz brief: %s\n", b.BoardName)
	write("Nodes: %d - edges: %d - ready: %d - blocked: %d - local: %d\n\n", b.Counts.Nodes, b.Counts.Edges, b.Counts.Ready, b.Counts.Blocked, b.Counts.LocalOnly)
	if b.NextMove != nil {
		write("Next move\n")
		writeItem(w, *b.NextMove, true)
		write("\n")
	}
	writeSection(w, "Ready now", b.Ready, false, true)
	writeSection(w, "Blocking most work", b.Blockers, true, true)
	writeSection(w, "Local-only", b.LocalOnly, false, true)
	writeSection(w, "Stale external state", b.Stale, false, false)
	return nil
}

func writeSection(w io.Writer, title string, items []BriefItem, showImpact bool, trailingBlank bool) {
	_, _ = fmt.Fprintf(w, "%s\n", title)
	if len(items) == 0 {
		_, _ = fmt.Fprintln(w, "  none")
		if trailingBlank {
			_, _ = fmt.Fprintln(w)
		}
		return
	}
	for _, item := range items {
		writeItem(w, item, showImpact)
	}
	if trailingBlank {
		_, _ = fmt.Fprintln(w)
	}
}

func writeItem(w io.Writer, item BriefItem, showImpact bool) {
	line := fmt.Sprintf("  %s %s", item.ID, item.Title)
	if showImpact && item.Impact > 0 {
		line += fmt.Sprintf(" (%s)", item.Reason)
	} else if item.Reason != "" {
		line += fmt.Sprintf(" - %s", item.Reason)
	}
	_, _ = fmt.Fprintln(w, line)
	if item.URL != "" {
		_, _ = fmt.Fprintf(w, "    %s\n", item.URL)
	}
}

func edgeBlockedAndBlocker(e Edge) (blocked string, blocker string) {
	switch strings.ToLower(strings.TrimSpace(e.Kind)) {
	case "addresses", "mentions", "relates_to", "related_to":
		return "", ""
	case "blocked_by", "depends_on", "depends", "after":
		return e.FromID, e.ToID
	case "blocks", "unblocks", "precedes":
		return e.ToID, e.FromID
	default:
		return e.FromID, e.ToID
	}
}

func activeBlockers(nodeID string, nodes map[string]Node, blockersByNode map[string]map[string]bool) []string {
	var blockers []string
	for blockerID := range blockersByNode[nodeID] {
		blocker, ok := nodes[blockerID]
		if ok && !blocker.IsClosed() {
			blockers = append(blockers, blockerID)
		}
	}
	sort.Strings(blockers)
	return blockers
}

func activeBlockedCount(nodeID string, nodes map[string]Node, blockedByNode map[string]map[string]bool) int {
	count := 0
	for blockedID := range blockedByNode[nodeID] {
		blocked, ok := nodes[blockedID]
		if ok && !blocked.IsClosed() {
			count++
		}
	}
	return count
}

func readyReason(n Node, blocked map[string]bool) string {
	impact := len(blocked)
	switch {
	case n.IsLocalOnly() && impact > 0:
		return fmt.Sprintf("local note, unlocks %d card%s", impact, plural(impact))
	case n.IsLocalOnly():
		return "local note with no active blockers"
	case impact > 0:
		return fmt.Sprintf("no active blockers, unlocks %d card%s", impact, plural(impact))
	default:
		return "no active blockers"
	}
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

func limitItems(items []BriefItem, n int) []BriefItem {
	if items == nil {
		return []BriefItem{}
	}
	if len(items) <= n {
		return items
	}
	return items[:n]
}
