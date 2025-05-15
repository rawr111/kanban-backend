package storage

import (
	"context"
	"kanban-backend/internal/models"
)

// UserStore определяет методы для работы с пользователями.
type UserStore interface {
	CreateUser(ctx context.Context, user *models.User, password string) (int, error)
	GetUserByUsername(ctx context.Context, username string) (*models.User, error)
} 