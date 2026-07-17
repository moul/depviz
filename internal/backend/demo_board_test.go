package backend

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDemoBoardRequiresBasicAuth(t *testing.T) {
	snapshot := writeDemoSnapshot(t, time.Now().UTC())
	ts := newBasicAuthTestServer(t, Config{
		BasicAuthUser:         "demo",
		BasicAuthPass:         "s3cret",
		DemoBoardSnapshotFile: snapshot,
	})

	res := get(t, ts.URL+"/api/demo-board", "", "")
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}

	res = get(t, ts.URL+"/api/demo-board", "demo", "s3cret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("authenticated status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	var payload demoBoardPayload
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if payload.BriefType != "board-status" {
		t.Fatalf("brief_type = %q, want board-status", payload.BriefType)
	}
	if payload.Brief.Counts.Pullable != 1 {
		t.Fatalf("pullable = %d, want 1", payload.Brief.Counts.Pullable)
	}
	if payload.Snapshot.Board.ScopeQuery != "repo:1789-tech/job-board" {
		t.Fatalf("scope = %q", payload.Snapshot.Board.ScopeQuery)
	}
}

func TestDemoBoardNotServedWithoutBasicAuthGate(t *testing.T) {
	snapshot := writeDemoSnapshot(t, time.Now().UTC())
	ts := newBasicAuthTestServer(t, Config{DemoBoardSnapshotFile: snapshot})

	res := get(t, ts.URL+"/api/demo-board", "", "")
	if res.StatusCode != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusForbidden)
	}
}

func TestDemoBoardMarksStaleSnapshots(t *testing.T) {
	snapshot := writeDemoSnapshot(t, time.Now().UTC().Add(-2*time.Hour))
	ts := newBasicAuthTestServer(t, Config{
		BasicAuthUser:         "demo",
		BasicAuthPass:         "s3cret",
		DemoBoardSnapshotFile: snapshot,
		DemoBoardMaxAge:       time.Minute,
	})

	res := get(t, ts.URL+"/api/demo-board", "demo", "s3cret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	var payload demoBoardPayload
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	if !payload.SnapshotStale {
		t.Fatal("snapshot_stale = false, want true")
	}
	if payload.Brief.SnapshotCheck == nil || payload.Brief.SnapshotCheck.SnapshotAgeSec == 0 {
		t.Fatalf("snapshot check missing age: %+v", payload.Brief.SnapshotCheck)
	}
}

func writeDemoSnapshot(t *testing.T, generatedAt time.Time) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "board-snapshot.json")
	data := `{
  "generated_at": "` + generatedAt.Format(time.RFC3339) + `",
  "queue": [
    {"number": 112, "title": "Ready private brief", "url": "https://github.com/1789-tech/job-board/issues/112", "project": "depviz", "status": "ready", "prio": "normal", "type": "build", "rank": 1}
  ],
  "open": [
    {"number": 112, "title": "Ready private brief", "url": "https://github.com/1789-tech/job-board/issues/112", "project": "depviz", "status": "ready", "prio": "normal", "type": "build", "rank": 1},
    {"number": 113, "title": "Blocked private brief", "url": "https://github.com/1789-tech/job-board/issues/113", "project": "boussole", "status": "blocked", "prio": "normal", "type": "brief", "rank": 2}
  ],
  "done": [
    {"number": 90, "title": "Done private brief", "url": "https://github.com/1789-tech/job-board/issues/90", "project": "career", "status": "done", "prio": "normal", "type": "build", "rank": 3}
  ]
}`
	if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}
