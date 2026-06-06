package core

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestFixtureBriefGolden(t *testing.T) {
	ctx := context.Background()
	s := openFixtureStore(t, ctx)
	var got bytes.Buffer
	brief, err := s.BuildBrief(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	if err := RenderBrief(&got, brief); err != nil {
		t.Fatal(err)
	}
	assertGolden(t, "simple.brief.txt", got.Bytes())
}

func TestFixtureExportGolden(t *testing.T) {
	ctx := context.Background()
	s := openFixtureStore(t, ctx)
	payload, err := s.BuildExport(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	normalizeExportForGolden(&payload)
	got, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	got = append(got, '\n')
	assertGolden(t, "simple.export.json", got)
}

func openFixtureStore(t *testing.T, ctx context.Context) *Store {
	t.Helper()
	s, err := OpenStore(ctx, filepath.Join(t.TempDir(), "state.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Fatal(err)
		}
	})
	f, err := os.Open(filepath.Join("..", "..", "testdata", "simple", "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if _, err := s.IngestEvents(ctx, f, DefaultBoardID); err != nil {
		t.Fatal(err)
	}
	return s
}

func normalizeExportForGolden(payload *Export) {
	payload.Snapshot.Board.UpdatedAt = time.Time{}
	for i := range payload.Snapshot.Nodes {
		payload.Snapshot.Nodes[i].UpdatedAt = time.Time{}
	}
	for i := range payload.Snapshot.Edges {
		payload.Snapshot.Edges[i].ObservedAt = time.Time{}
	}
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := filepath.Join("..", "..", "testdata", "golden", name)
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, got, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}
