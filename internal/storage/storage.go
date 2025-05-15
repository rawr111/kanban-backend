package storage

import (
	"context"

	"kanban-backend/internal/models" // Замените на свой путь
)

// TaskStore определяет методы для взаимодействия с хранилищем задач.
type TaskStore interface {
	Connect(ctx context.Context, dsn string) error
	Close() error
	CreateTask(ctx context.Context, task *models.Task) (int, error)
	GetTaskByID(ctx context.Context, id int, userID int) (*models.Task, error)
	GetAllTasks(ctx context.Context, userID int) ([]models.Task, error)
	DeleteTask(ctx context.Context, id int, userID int) error
}
