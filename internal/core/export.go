package core

import (
	"context"
	"encoding/json"
	"io"
)

func (s *Store) BuildExport(ctx context.Context, boardID string) (Export, error) {
	snap, err := s.Snapshot(ctx, boardID)
	if err != nil {
		return Export{}, err
	}
	brief, err := s.BuildBrief(ctx, boardID)
	if err != nil {
		return Export{}, err
	}
	return Export{Snapshot: snap, Brief: brief}, nil
}

func (s *Store) RenderJSON(ctx context.Context, boardID string, w io.Writer) error {
	payload, err := s.BuildExport(ctx, boardID)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(payload)
}
