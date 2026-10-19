package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type SessionStore struct {
	db *sql.DB
}

func NewSessionStore(db *sql.DB) *SessionStore {
	return &SessionStore{db: db}
}

func (s *SessionStore) Create(ctx context.Context, id string, userID int64, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
INSERT INTO sessions (id, user_id, expires_at, created_at)
VALUES (?, ?, ?, ?)
`, id, userID, expiresAt, time.Now())
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}
	return nil
}

func (s *SessionStore) GetUserID(ctx context.Context, id string) (int64, time.Time, error) {
	var userID int64
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx, `
SELECT user_id, expires_at FROM sessions WHERE id = ?
`, id).Scan(&userID, &expiresAt)
	if err == sql.ErrNoRows {
		return 0, time.Time{}, nil
	}
	if err != nil {
		return 0, time.Time{}, err
	}
	return userID, expiresAt, nil
}

func (s *SessionStore) Delete(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, id)
	return err
}

func (s *SessionStore) DeleteExpired(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now())
	return err
}
