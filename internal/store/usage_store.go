package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type UsageStore struct {
	db *sql.DB
}

func NewUsageStore(db *sql.DB) *UsageStore {
	return &UsageStore{db: db}
}

func (s *UsageStore) Get(ctx context.Context, toolKey string) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx,
		`SELECT use_count FROM tool_usage WHERE tool_key = ?`, toolKey,
	).Scan(&count)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, err
	}
	return count, nil
}

func (s *UsageStore) Increment(ctx context.Context, toolKey string) (int64, error) {
	now := time.Now()
	_, err := s.db.ExecContext(ctx, `
INSERT INTO tool_usage (tool_key, use_count, updated_at)
VALUES (?, 1, ?)
ON DUPLICATE KEY UPDATE
  use_count = use_count + 1,
  updated_at = VALUES(updated_at)
`, toolKey, now)
	if err != nil {
		return 0, fmt.Errorf("increment usage: %w", err)
	}
	return s.Get(ctx, toolKey)
}
