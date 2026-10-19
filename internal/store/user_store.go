package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/find-work/tools-web-backend/internal/model"
)

type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) Create(ctx context.Context, username, passwordHash, role string) (*model.User, error) {
	now := time.Now()
	res, err := s.db.ExecContext(ctx, `
INSERT INTO users (username, password_hash, role, created_at, updated_at)
VALUES (?, ?, ?, ?, ?)
`, username, passwordHash, role, now, now)
	if err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, err
	}
	return &model.User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		Role:         role,
		CreatedAt:    now,
		UpdatedAt:    now,
	}, nil
}

func (s *UserStore) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	var u model.User
	err := s.db.QueryRowContext(ctx, `
SELECT id, username, password_hash, role, created_at, updated_at
FROM users WHERE username = ?
`, username).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (s *UserStore) GetByID(ctx context.Context, id int64) (*model.User, error) {
	var u model.User
	err := s.db.QueryRowContext(ctx, `
SELECT id, username, password_hash, role, created_at, updated_at
FROM users WHERE id = ?
`, id).Scan(&u.ID, &u.Username, &u.PasswordHash, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
