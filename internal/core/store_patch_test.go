package core

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
)

var errTestRollback = errors.New("test rollback")

func TestWithTxCommit(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	called := false
	err = s.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		called = true
		return nil
	})
	if err != nil {
		t.Fatalf("WithTx commit: %v", err)
	}
	if !called {
		t.Error("fn not called")
	}
}

func TestWithTxRollback(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	err = s.WithTx(ctx, func(ctx context.Context, tx *sql.Tx) error {
		return errTestRollback
	})
	if !errors.Is(err, errTestRollback) {
		t.Fatalf("expected errTestRollback, got %v", err)
	}
}

func TestApplyBoardSourcePatchRollsBackAllWrites(t *testing.T) {
	ctx := context.Background()
	s, err := OpenStore(ctx, t.TempDir()+"/state.db")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	err = s.ApplyBoardSourcePatch(ctx, DefaultBoardID, BoardSourcePatch{
		Creates: []BoardSourceCreate{{
			Kind:   "task",
			Title:  "Should not survive rollback",
			Status: "draft",
		}},
		LinkCreates: []BoardSourceLinkCreate{{
			FromID: "task:missing-from",
			ToID:   "task:missing-to",
			Kind:   "blocks",
		}},
	})
	if err == nil {
		t.Fatal("expected patch error")
	}
	snap, err := s.Snapshot(ctx, DefaultBoardID)
	if err != nil {
		t.Fatal(err)
	}
	for _, node := range snap.Nodes {
		if strings.Contains(node.Title, "Should not survive rollback") {
			t.Fatalf("node %q survived failed patch", node.ID)
		}
	}
}
