package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"log" // Используем стандартный логгер для простоты
	"time"

	"kanban-backend/internal/models" // Замените на свой путь

	_ "github.com/lib/pq" // Драйвер PostgreSQL
)

// TaskStore реализует интерфейс storage.TaskStore для PostgreSQL.
type TaskStore struct {
	db *sql.DB
}

// NewTaskStore создает новый экземпляр TaskStore.
func NewTaskStore() *TaskStore {
	return &TaskStore{}
}

// Connect устанавливает соединение с базой данных PostgreSQL.
func (s *TaskStore) Connect(ctx context.Context, dsn string) error {
	var err error
	// Используем PingContext с таймаутом для проверки соединения
	const maxRetries = 5
	const delay = 2 * time.Second

	for i := 0; i < maxRetries; i++ {
		log.Printf("Попытка подключения к PostgreSQL (попытка %d/%d)...", i+1, maxRetries)
		s.db, err = sql.Open("postgres", dsn)
		if err != nil {
			return fmt.Errorf("ошибка при открытии соединения с БД: %w", err)
		}

		pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err = s.db.PingContext(pingCtx)
		cancel() // Важно отменить контекст

		if err == nil {
			log.Println("Успешное подключение к PostgreSQL")
			return nil // Успешное подключение
		}

		log.Printf("Не удалось подключиться к PostgreSQL: %v. Повтор через %v...", err, delay)
		s.db.Close() // Закрываем неудачное соединение
		select {
		case <-time.After(delay): // Ждем перед следующей попыткой
		case <-ctx.Done(): // Если основной контекст отменен, выходим
			return fmt.Errorf("подключение к БД отменено: %w", ctx.Err())
		}
	}

	return fmt.Errorf("не удалось подключиться к PostgreSQL после %d попыток: %w", maxRetries, err)
}

// Close закрывает соединение с базой данных.
func (s *TaskStore) Close() error {
	if s.db != nil {
		log.Println("Закрытие соединения с PostgreSQL...")
		return s.db.Close()
	}
	return nil
}

// Migrate создает необходимую таблицу, если она не существует.
func (s *TaskStore) Migrate(ctx context.Context) error {
	query := `
    CREATE TABLE IF NOT EXISTS tasks (
        id SERIAL PRIMARY KEY,
        title VARCHAR(255) NOT NULL,
        description TEXT,
        status VARCHAR(50) DEFAULT 'pending',
        created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
    );`

	migrateCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	_, err := s.db.ExecContext(migrateCtx, query)
	if err != nil {
		return fmt.Errorf("ошибка при миграции базы данных: %w", err)
	}
	log.Println("Миграция базы данных успешно выполнена (или таблица уже существует)")
	return nil
}

// CreateTask добавляет новую задачу в базу данных.
func (s *TaskStore) CreateTask(ctx context.Context, task *models.Task) (int, error) {
	query := `INSERT INTO tasks (title, description, status, user_id) VALUES ($1, $2, $3, $4) RETURNING id`
	if task.Status == "" {
		task.Status = "pending"
	}
	createCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	err := s.db.QueryRowContext(createCtx, query, task.Title, task.Description, task.Status, task.UserID).Scan(&task.ID)
	if err != nil {
		return 0, fmt.Errorf("ошибка при создании задачи: %w", err)
	}
	log.Printf("Задача создана с ID: %d", task.ID)
	return task.ID, nil
}

// GetTaskByID получает задачу по ее ID и user_id.
func (s *TaskStore) GetTaskByID(ctx context.Context, id int, userID int) (*models.Task, error) {
	query := `SELECT id, title, description, status, user_id FROM tasks WHERE id = $1 AND user_id = $2`
	task := &models.Task{}
	getCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	row := s.db.QueryRowContext(getCtx, query, id, userID)
	err := row.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.UserID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("задача с ID %d не найдена: %w", id, err)
		}
		return nil, fmt.Errorf("ошибка при получении задачи %d: %w", id, err)
	}
	return task, nil
}

// GetAllTasks получает все задачи пользователя из базы данных.
func (s *TaskStore) GetAllTasks(ctx context.Context, userID int) ([]models.Task, error) {
	query := `SELECT id, title, description, status, user_id FROM tasks WHERE user_id = $1 ORDER BY created_at DESC`
	tasks := []models.Task{}
	getCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	rows, err := s.db.QueryContext(getCtx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка задач: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var task models.Task
		err := rows.Scan(&task.ID, &task.Title, &task.Description, &task.Status, &task.UserID)
		if err != nil {
			log.Printf("Ошибка сканирования строки задачи: %v", err)
			continue
		}
		tasks = append(tasks, task)
	}
	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("ошибка при итерации по результатам задач: %w", err)
	}
	return tasks, nil
}

// DeleteTask удаляет задачу по ее ID и user_id.
func (s *TaskStore) DeleteTask(ctx context.Context, id int, userID int) error {
	query := `DELETE FROM tasks WHERE id = $1 AND user_id = $2`
	deleteCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	result, err := s.db.ExecContext(deleteCtx, query, id, userID)
	if err != nil {
		return fmt.Errorf("ошибка при удалении задачи %d: %w", id, err)
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Printf("Не удалось получить количество удаленных строк для задачи %d: %v", id, err)
	}
	if rowsAffected == 0 {
		return fmt.Errorf("задача с ID %d не найдена для удаления", id)
	}
	log.Printf("Задача с ID %d удалена", id)
	return nil
}

func (s *TaskStore) DB() *sql.DB {
	return s.db
}
