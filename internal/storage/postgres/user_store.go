package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"kanban-backend/internal/models"
)

// UserStore реализует storage.UserStore для PostgreSQL.
type UserStore struct {
	db *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{db: db}
}

func (s *UserStore) CreateUser(ctx context.Context, user *models.User, password string) (int, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return 0, fmt.Errorf("bcrypt error: %w", err)
	}
	user.CreatedAt = time.Now()
	query := `INSERT INTO users (username, password_hash, created_at) VALUES ($1, $2, $3) RETURNING id`
	var id int
	err = s.db.QueryRowContext(ctx, query, user.Username, string(hash), user.CreatedAt).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("user not created: %w", err)
		}
		return 0, fmt.Errorf("db error: %w", err)
	}
	user.ID = id
	user.PasswordHash = string(hash)
	return id, nil
}

func (s *UserStore) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	query := `SELECT id, username, password_hash, created_at FROM users WHERE username = $1`
	user := &models.User{}
	err := s.db.QueryRowContext(ctx, query, username).Scan(&user.ID, &user.Username, &user.PasswordHash, &user.CreatedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("db error: %w", err)
	}
	return user, nil
}

func (s *UserStore) Migrate(ctx context.Context) error {
	query := `
	CREATE TABLE IF NOT EXISTS users (
		id SERIAL PRIMARY KEY,
		username VARCHAR(64) NOT NULL UNIQUE,
		password_hash VARCHAR(255) NOT NULL,
		created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
	);`
	_, err := s.db.ExecContext(ctx, query)
	return err
} 