package backend

import (
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
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

func TestDemoBoardPutRequiresBasicAuthAndDoesNotWriteAnonymously(t *testing.T) {
	snapshot := writeDemoSnapshot(t, time.Now().UTC())
	before, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	ts := newBasicAuthTestServer(t, Config{
		BasicAuthUser:         "demo",
		BasicAuthPass:         "s3cret",
		DemoBoardSnapshotFile: snapshot,
	})

	res := put(t, ts.URL+"/api/demo-board", "", "", demoSnapshotJSON(time.Now().UTC(), 220, "Anonymous write"))
	if res.StatusCode != http.StatusUnauthorized {
		t.Fatalf("anonymous PUT status = %d, want %d", res.StatusCode, http.StatusUnauthorized)
	}
	after, err := os.ReadFile(snapshot)
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("anonymous PUT changed the demo board snapshot")
	}
}

func TestDemoBoardPutReplacesSnapshot(t *testing.T) {
	snapshot := writeDemoSnapshot(t, time.Now().UTC())
	ts := newBasicAuthTestServer(t, Config{
		BasicAuthUser:         "demo",
		BasicAuthPass:         "s3cret",
		DemoBoardSnapshotFile: snapshot,
	})

	res := put(t, ts.URL+"/api/demo-board", "demo", "s3cret", demoSnapshotJSON(time.Now().UTC(), 220, "Updated private brief"))
	if res.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(res.Body)
		t.Fatalf("authenticated PUT status = %d, want %d: %s", res.StatusCode, http.StatusOK, body)
	}

	res = get(t, ts.URL+"/api/demo-board", "demo", "s3cret")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("GET after PUT status = %d, want %d", res.StatusCode, http.StatusOK)
	}
	var payload demoBoardPayload
	if err := json.NewDecoder(res.Body).Decode(&payload); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, node := range payload.Snapshot.Nodes {
		if node.ExternalID == "#220" && node.State == "ready" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("snapshot nodes after PUT = %+v", payload.Snapshot.Nodes)
	}
	if payload.Brief.Counts.Pullable != 1 {
		t.Fatalf("pullable after PUT = %d, want 1", payload.Brief.Counts.Pullable)
	}
}

func TestDemoBoardSnapshotDoneRowsAreClosed(t *testing.T) {
	generatedAt := time.Now().UTC()
	payload, err := parseDemoBoard(strings.NewReader(`{
  "generated_at": "`+generatedAt.Format(time.RFC3339)+`",
  "queue": [
    {"number": 112, "title": "Claimed private brief", "url": "https://github.com/1789-tech/job-board/issues/112", "project": "depviz", "status": "active", "prio": "normal", "type": "build", "rank": 1},
    {"number": 100, "title": "Closed duplicate in queue", "url": "https://github.com/1789-tech/job-board/issues/100", "project": "depviz", "status": "ready", "prio": "normal", "type": "build", "rank": 2}
  ],
  "open": [
    {"number": 112, "title": "Claimed private brief", "url": "https://github.com/1789-tech/job-board/issues/112", "project": "depviz", "status": "active", "prio": "normal", "type": "build", "rank": 1},
    {"number": 113, "title": "Untriaged open private brief", "url": "https://github.com/1789-tech/job-board/issues/113", "project": "boussole", "prio": "normal", "type": "brief", "rank": 3}
  ],
  "done": [
    {"number": 100, "title": "Done private brief", "url": "https://github.com/1789-tech/job-board/issues/100", "project": "depviz", "ClosedAt": "2026-07-09T11:04:00Z"},
    {"number": 101, "title": "Done lower-case private brief", "url": "https://github.com/1789-tech/job-board/issues/101", "project": "depviz", "closed_at": "2026-07-08T08:06:05Z"}
  ]
}`), time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if payload.Brief.Counts.Open != 2 {
		t.Fatalf("open = %d, want 2", payload.Brief.Counts.Open)
	}
	if payload.Brief.Counts.Closed != 2 {
		t.Fatalf("closed = %d, want 2", payload.Brief.Counts.Closed)
	}
	if payload.Brief.Counts.Untriaged != 1 {
		t.Fatalf("untriaged = %d, want 1", payload.Brief.Counts.Untriaged)
	}
	for _, item := range payload.Brief.Untriaged {
		if strings.HasSuffix(item.ID, "#100") || strings.HasSuffix(item.ID, "#101") {
			t.Fatalf("closed issue listed as untriaged: %+v", item)
		}
	}
	for _, node := range payload.Snapshot.Nodes {
		if node.ExternalID == "#100" && node.State != "closed" {
			t.Fatalf("done row duplicated in queue should stay closed: %+v", node)
		}
	}
}

func writeDemoSnapshot(t *testing.T, generatedAt time.Time) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "board-snapshot.json")
	if err := os.WriteFile(path, []byte(demoSnapshotJSON(generatedAt, 112, "Ready private brief")), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func demoSnapshotJSON(generatedAt time.Time, number int, title string) string {
	return `{
  "generated_at": "` + generatedAt.Format(time.RFC3339) + `",
  "queue": [
    {"number": ` + strconv.Itoa(number) + `, "title": "` + title + `", "url": "https://github.com/1789-tech/job-board/issues/` + strconv.Itoa(number) + `", "project": "depviz", "status": "ready", "prio": "normal", "type": "build", "rank": 1}
  ],
  "open": [
    {"number": ` + strconv.Itoa(number) + `, "title": "` + title + `", "url": "https://github.com/1789-tech/job-board/issues/` + strconv.Itoa(number) + `", "project": "depviz", "status": "ready", "prio": "normal", "type": "build", "rank": 1},
    {"number": 113, "title": "Blocked private brief", "url": "https://github.com/1789-tech/job-board/issues/113", "project": "boussole", "status": "blocked", "prio": "normal", "type": "brief", "rank": 2}
  ],
  "done": [
    {"number": 90, "title": "Done private brief", "url": "https://github.com/1789-tech/job-board/issues/90", "project": "career", "status": "done", "prio": "normal", "type": "build", "rank": 3}
  ]
}`
}
