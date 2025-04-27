package models

// Task представляет собой задачу в списке дел.
// swagger:model Task
type Task struct {
	// Уникальный идентификатор задачи
	// example: 1
	ID int `json:"id"`

	// Название задачи
	// required: true
	// example: Купить молоко
	Title string `json:"title" validate:"required"` // Добавим validate для примера, хотя сейчас не используем валидатор

	// Описание задачи (опционально)
	// example: Нежирное, 1 литр
	Description string `json:"description,omitempty"`

	// Статус задачи (например, "pending", "completed")
	// example: pending
	Status string `json:"status,omitempty"` // Пока не используется в логике, но добавим для полноты
}

// CreateTaskRequest описывает тело запроса для создания задачи.
// Используется, чтобы не требовать ID при создании.
// swagger:parameters createTask
type CreateTaskRequest struct {
	// В теле запроса ожидается объект TaskCreatePayload
	// in: body
	// required: true
	Body TaskCreatePayload `json:"body"`
}

// TaskCreatePayload определяет поля для создания новой задачи.
// swagger:model TaskCreatePayload
type TaskCreatePayload struct {
	// Название задачи
	// required: true
	// example: Помыть посуду
	Title string `json:"title" validate:"required"`

	// Описание задачи (опционально)
	// example: Сразу после ужина
	Description string `json:"description,omitempty"`
}

// TaskIDParameter описывает параметр ID задачи в пути URL.
// swagger:parameters getTask deleteTask
type TaskIDParameter struct {
	// Идентификатор задачи
	// in: path
	// name: taskID
	// required: true
	// example: 5
	TaskID int `json:"taskID"`
}
