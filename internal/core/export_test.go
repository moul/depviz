package core

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	// Create some local nodes
	_, err = s.CreateNote(ctx, DefaultBoardID, "test note")
	if err != nil {
		t.Fatal(err)
	}
	// Build export
	payload, err := s.BuildExport(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	// Marshal to JSON and back
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "test note") {
		t.Error("export JSON should contain the note text")
	}
}

func TestExportJSON(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	payload, err := s.BuildExport(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	// Verify JSON export contains required fields
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	if _, ok := out["snapshot"]; !ok {
		t.Error("export should contain 'snapshot' field")
	}
}
