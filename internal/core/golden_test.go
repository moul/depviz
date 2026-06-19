package core

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
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

func TestRealisticGnoLast100Golden(t *testing.T) {
	payload := readExportFixture(t, "realistic", "gno-last-100", "export.json")
	wantCounts := BriefCounts{
		Nodes:   103,
		Edges:   11,
		Ready:   72,
		Blocked: 0,
		Stale:   22,
	}
	if payload.Brief.Counts != wantCounts {
		t.Fatalf("counts = %+v, want %+v", payload.Brief.Counts, wantCounts)
	}
	if len(payload.Snapshot.Nodes) != wantCounts.Nodes {
		t.Fatalf("nodes = %d, want %d", len(payload.Snapshot.Nodes), wantCounts.Nodes)
	}
	if len(payload.Snapshot.Edges) != wantCounts.Edges {
		t.Fatalf("edges = %d, want %d", len(payload.Snapshot.Edges), wantCounts.Edges)
	}
	kinds := map[string]int{}
	for _, edge := range payload.Snapshot.Edges {
		kinds[edge.Kind]++
		if edge.Authority != "github-inferred" {
			t.Fatalf("edge %s authority = %q, want github-inferred", edge.ID, edge.Authority)
		}
		if edge.Confidence <= 0 || edge.Confidence >= 1 {
			t.Fatalf("edge %s confidence = %v, want soft confidence", edge.ID, edge.Confidence)
		}
	}
	if kinds["blocked_by"] != 5 || kinds["closes"] != 6 {
		t.Fatalf("edge kinds = %+v, want blocked_by=5 closes=6", kinds)
	}
	var rendered bytes.Buffer
	if err := RenderBrief(&rendered, payload.Brief); err != nil {
		t.Fatal(err)
	}
	assertFixtureBytes(t, filepath.Join("realistic", "gno-last-100", "brief.txt"), rendered.Bytes())
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

func readExportFixture(t *testing.T, parts ...string) Export {
	t.Helper()
	path := fixturePath(parts...)
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var payload Export
	if err := json.Unmarshal(b, &payload); err != nil {
		t.Fatal(err)
	}
	return payload
}

func normalizeExportForGolden(payload *Export) {
	payload.Snapshot.Board.UpdatedAt = time.Time{}
	sort.Slice(payload.Snapshot.Nodes, func(i, j int) bool {
		return payload.Snapshot.Nodes[i].ID < payload.Snapshot.Nodes[j].ID
	})
	for i := range payload.Snapshot.Nodes {
		payload.Snapshot.Nodes[i].UpdatedAt = time.Time{}
	}
	sort.Slice(payload.Snapshot.Edges, func(i, j int) bool {
		return payload.Snapshot.Edges[i].ID < payload.Snapshot.Edges[j].ID
	})
	for i := range payload.Snapshot.Edges {
		payload.Snapshot.Edges[i].ObservedAt = time.Time{}
	}
}

func assertFixtureBytes(t *testing.T, name string, got []byte) {
	t.Helper()
	want, err := os.ReadFile(fixturePath(name))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func assertGolden(t *testing.T, name string, got []byte) {
	t.Helper()
	path := fixturePath("golden", name)
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
	if strings.HasSuffix(name, ".json") {
		gotJSON, gotErr := normalizeGoldenJSON(got)
		wantJSON, wantErr := normalizeGoldenJSON(want)
		if gotErr != nil || wantErr != nil {
			t.Fatalf("%s json normalization failed got=%v want=%v", name, gotErr, wantErr)
		}
		if !reflect.DeepEqual(gotJSON, wantJSON) {
			t.Fatalf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
		}
		return
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s mismatch\n--- got ---\n%s\n--- want ---\n%s", name, got, want)
	}
}

func normalizeGoldenJSON(data []byte) (any, error) {
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return nil, err
	}
	return normalizeEmbeddedJSON(value), nil
}

func normalizeEmbeddedJSON(value any) any {
	switch v := value.(type) {
	case map[string]any:
		out := make(map[string]any, len(v))
		for key, child := range v {
			if s, ok := child.(string); ok && strings.HasSuffix(key, "_json") && json.Valid([]byte(s)) {
				var embedded any
				if err := json.Unmarshal([]byte(s), &embedded); err == nil {
					out[key] = normalizeEmbeddedJSON(embedded)
					continue
				}
			}
			out[key] = normalizeEmbeddedJSON(child)
		}
		return out
	case []any:
		out := make([]any, len(v))
		for i, child := range v {
			out[i] = normalizeEmbeddedJSON(child)
		}
		return out
	default:
		return value
	}
}

func fixturePath(parts ...string) string {
	return filepath.Join(append([]string{"..", "..", "testdata"}, parts...)...)
}
