package core

import (
	"context"
	"database/sql"
	"errors"
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
