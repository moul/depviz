package core

import (
	"context"
	"strings"
	"testing"
)

func TestBriefWithLocalNoteAndBlockedIssue(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	events := strings.NewReader(`
{"type":"node","id":"gh:moul/depviz2#1","kind":"issue","title":"Bootstrap SQLite work graph","state":"open","source":"github:moul/depviz2","external_id":"#1","url":"https://github.com/moul/depviz2/issues/1"}
{"type":"node","id":"gh:moul/depviz2#2","kind":"issue","title":"Render static HTML","state":"open","source":"github:moul/depviz2","external_id":"#2","url":"https://github.com/moul/depviz2/issues/2"}
{"type":"edge","from":"gh:moul/depviz2#2","to":"gh:moul/depviz2#1","kind":"blocked_by"}
`)
	if _, err := s.IngestEvents(ctx, events, DefaultBoardID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.CreateNote(ctx, DefaultBoardID, "Decide first implementation slice"); err != nil {
		t.Fatal(err)
	}
	brief, err := s.BuildBrief(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	if brief.Counts.Nodes != 3 {
		t.Fatalf("nodes = %d, want 3", brief.Counts.Nodes)
	}
	if brief.Counts.Blocked != 1 {
		t.Fatalf("blocked = %d, want 1", brief.Counts.Blocked)
	}
	if brief.NextMove == nil {
		t.Fatal("expected next move")
	}
	if brief.NextMove.ID != "gh:moul/depviz2#1" {
		t.Fatalf("next = %s, want gh:moul/depviz2#1", brief.NextMove.ID)
	}
	if len(brief.LocalOnly) != 1 {
		t.Fatalf("local notes = %d, want 1", len(brief.LocalOnly))
	}
}

func TestPlaceholderRefsDoNotBecomeReadyWork(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.CreateNote(ctx, DefaultBoardID, "Local decision"); err != nil {
		t.Fatal(err)
	}
	if _, err := s.AddEdge(ctx, DefaultBoardID, "gh:moul/depviz2#99", "note:local-decision", "blocked_by", "test", nil); err != nil {
		t.Fatal(err)
	}
	brief, err := s.BuildBrief(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range brief.Ready {
		if item.ID == "gh:moul/depviz2#99" {
			t.Fatalf("placeholder was listed as ready: %+v", item)
		}
	}
	if len(brief.Stale) != 1 || brief.Stale[0].ID != "gh:moul/depviz2#99" {
		t.Fatalf("stale placeholders = %+v, want gh:moul/depviz2#99", brief.Stale)
	}
}

func TestNonBlockingEdgesDoNotBlockBrief(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	events := strings.NewReader(`
{"type":"node","id":"gh:moul/depviz#1","kind":"issue","title":"Product question","state":"open","source":"github:moul/depviz","external_id":"#1","url":"https://github.com/moul/depviz/issues/1"}
{"type":"node","id":"gh:moul/depviz#2","kind":"issue","title":"Implementation detail","state":"open","source":"github:moul/depviz","external_id":"#2","url":"https://github.com/moul/depviz/issues/2"}
{"type":"edge","from":"gh:moul/depviz#1","to":"gh:moul/depviz#2","kind":"addresses"}
{"type":"edge","from":"gh:moul/depviz#2","to":"gh:moul/depviz#1","kind":"closes"}
`)
	if _, err := s.IngestEvents(ctx, events, DefaultBoardID); err != nil {
		t.Fatal(err)
	}
	brief, err := s.BuildBrief(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	if brief.Counts.Blocked != 0 {
		t.Fatalf("blocked = %d, want 0", brief.Counts.Blocked)
	}
	if brief.Counts.Ready != 2 {
		t.Fatalf("ready = %d, want 2", brief.Counts.Ready)
	}
}

func TestIngestEventPreservesEdgeConfidence(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	events := strings.NewReader(`
{"type":"node","id":"gh:moul/depviz#1","kind":"issue","title":"Implementation detail","state":"open","source":"github:moul/depviz","external_id":"#1","url":"https://github.com/moul/depviz/issues/1"}
{"type":"node","id":"gh:moul/depviz#2","kind":"issue","title":"Closing PR","state":"open","source":"github:moul/depviz","external_id":"#2","url":"https://github.com/moul/depviz/issues/2"}
{"type":"edge","from":"gh:moul/depviz#2","to":"gh:moul/depviz#1","kind":"closes","authority":"github-inferred","confidence":0.7}
`)
	if _, err := s.IngestEvents(ctx, events, DefaultBoardID); err != nil {
		t.Fatal(err)
	}
	snap, err := s.Snapshot(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Edges) != 1 {
		t.Fatalf("edges = %+v, want one edge", snap.Edges)
	}
	if snap.Edges[0].Confidence != 0.7 {
		t.Fatalf("confidence = %v, want 0.7", snap.Edges[0].Confidence)
	}
	if snap.Edges[0].Authority != "github-inferred" {
		t.Fatalf("authority = %q, want github-inferred", snap.Edges[0].Authority)
	}
}
