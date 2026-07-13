package core

import (
	"bytes"
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

func TestBoardStatusBriefOnlyListsReadyWithoutLiveBlockers(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	nodes := []Node{
		boardStatusTestNode(1, "Ready unblocked", "open", "status:ready"),
		boardStatusTestNode(2, "Ready blocked", "open", "status:ready"),
		boardStatusTestNode(3, "Live blocker", "open", "status:active"),
		boardStatusTestNode(4, "Parked work", "open", "status:parked"),
		boardStatusTestNode(5, "Blocked work", "open", "status:blocked"),
		boardStatusTestNode(6, "Review work", "open", "status:review"),
		boardStatusTestNode(7, "Untriaged work", "open"),
		boardStatusTestNode(8, "Closed blocker", "closed", "status:active"),
		boardStatusTestNode(9, "Ready with closed blocker", "open", "status:ready"),
	}
	for _, n := range nodes {
		if err := s.UpsertNode(ctx, n); err != nil {
			t.Fatal(err)
		}
		if err := s.AddNodeToBoard(ctx, DefaultBoardID, n.ID, "issue", ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.AddEdgeWithConfidence(ctx, DefaultBoardID, nodes[1].ID, nodes[2].ID, "blocked_by", "github-inferred", 0.75, ExtractedEdge{Line: "Depends on: #3"}); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddEdgeWithConfidence(ctx, DefaultBoardID, nodes[8].ID, nodes[7].ID, "blocked_by", "github-inferred", 0.75, ExtractedEdge{Line: "Depends on: #8"}); err != nil {
		t.Fatal(err)
	}

	brief, err := s.BuildBoardStatusBrief(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	if brief.Counts.Pullable != 2 {
		t.Fatalf("pullable = %d, want 2: %+v", brief.Counts.Pullable, brief.Pullable)
	}
	if got := brief.Pullable[0].ID + "," + brief.Pullable[1].ID; got != "gh:1789-tech/job-board#1,gh:1789-tech/job-board#9" {
		t.Fatalf("pullable IDs = %s", got)
	}
	if len(brief.Blocked) != 1 || brief.Blocked[0].ID != "gh:1789-tech/job-board#2" {
		t.Fatalf("blocked = %+v, want only #2", brief.Blocked)
	}
	if len(brief.Untriaged) != 1 || brief.Untriaged[0].ID != "gh:1789-tech/job-board#7" {
		t.Fatalf("untriaged = %+v, want only #7", brief.Untriaged)
	}
	for _, item := range brief.Pullable {
		switch item.ID {
		case "gh:1789-tech/job-board#4", "gh:1789-tech/job-board#5", "gh:1789-tech/job-board#6", "gh:1789-tech/job-board#8":
			t.Fatalf("non-ready or closed item listed as pullable: %+v", item)
		}
	}
}

func TestBoardStatusBriefSuppressesProseSoftBlockers(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for _, n := range []Node{
		boardStatusTestNode(1, "Ready target", "open", "status:ready"),
		boardStatusTestNode(59, "Sibling", "open", "status:active"),
	} {
		if err := s.UpsertNode(ctx, n); err != nil {
			t.Fatal(err)
		}
		if err := s.AddNodeToBoard(ctx, DefaultBoardID, n.ID, "issue", ""); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := s.AddEdgeWithConfidence(ctx, DefaultBoardID, "gh:1789-tech/job-board#1", "gh:1789-tech/job-board#59", "blocked_by", "github-inferred", 0.75, ExtractedEdge{Line: "sibling #59"}); err != nil {
		t.Fatal(err)
	}
	brief, err := s.BuildBoardStatusBrief(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	if brief.Counts.Pullable != 1 || brief.Pullable[0].ID != "gh:1789-tech/job-board#1" {
		t.Fatalf("pullable = %+v, want #1 despite prose edge", brief.Pullable)
	}
	if brief.Counts.DroppedEdge != 1 {
		t.Fatalf("dropped edges = %d, want 1", brief.Counts.DroppedEdge)
	}
}

func TestRenderBoardStatusBriefHeadlinesHistogram(t *testing.T) {
	var out bytes.Buffer
	brief := BoardStatusBrief{
		BoardName: "Default",
		Counts:    BoardStatusCounts{Open: 1},
		Statuses:  []BoardStatusCount{{Status: "ready", Count: 0}, {Status: "active", Count: 1}},
		Deltas:    []BoardStatusDelta{{Status: "active", Now: 1, Before: 0, Delta: 1}},
		Pullable:  []BriefItem{},
		Blocked:   []BriefItem{},
		Untriaged: []BriefItem{},
	}
	if err := RenderBoardStatusBrief(&out, brief); err != nil {
		t.Fatal(err)
	}
	got := out.String()
	for _, want := range []string{
		"Status histogram (day-over-day delta)",
		"status:active 1 (+1)",
		"Pullable now\n  none",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("rendered brief missing %q:\n%s", want, got)
		}
	}
}

func boardStatusTestNode(number int, title, state string, labels ...string) Node {
	payload, _ := json.Marshal(map[string]any{
		"source": "github",
		"kind":   "issue",
		"repo":   "example/job-board",
		"number": number,
		"labels": labels,
	})
	id := "gh:1789-tech/job-board#" + strconv.Itoa(number)
	return Node{
		ID:         id,
		Kind:       "issue",
		Title:      title,
		State:      state,
		DataJSON:   string(payload),
		URL:        "https://github.com/1789-tech/job-board/issues/" + strconv.Itoa(number),
		ExternalID: "#" + strconv.Itoa(number),
	}
}
