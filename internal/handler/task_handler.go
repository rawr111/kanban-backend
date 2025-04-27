package handler

import (
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strconv"
	"strings"

	"kanban-backend/internal/models"  // Замените на свой путь
	"kanban-backend/internal/storage" // Замените

	"github.com/go-chi/chi/v5"
)

// TaskHandler обрабатывает HTTP запросы, связанные с задачами.
type TaskHandler struct {
	Store storage.TaskStore
}

// NewTaskHandler создает новый экземпляр TaskHandler.
func NewTaskHandler(store storage.TaskStore) *TaskHandler {
	return &TaskHandler{Store: store}
}

// --- Вспомогательные функции ---

func respondWithError(w http.ResponseWriter, code int, message string) {
	respondWithJSON(w, code, map[string]string{"error": message})
}

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	response, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Ошибка маршалинга JSON: %v", err)
		// Отправляем простой текстовый ответ в случае ошибки маршалинга
		http.Error(w, `{"error":"Internal Server Error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, err = w.Write(response)
	if err != nil {
		log.Printf("Ошибка записи ответа: %v", err)
	}
}

// --- Обработчики ---

// CreateTask godoc
// @Summary Создать новую задачу
// @Description Добавляет новую задачу в список
// @Tags tasks
// @Accept json
// @Produce json
// @Param task body models.TaskCreatePayload true "Данные для создания задачи"
// @Success 201 {object} models.Task "Задача успешно создана"
// @Failure 400 {object} map[string]string "Неверный формат запроса"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /tasks [post]
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	var payload models.TaskCreatePayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Неверный формат JSON: "+err.Error())
		return
	}
	defer r.Body.Close()

	// Простая валидация
	if strings.TrimSpace(payload.Title) == "" {
		respondWithError(w, http.StatusBadRequest, "Название задачи не может быть пустым")
		return
	}

	task := models.Task{
		Title:       payload.Title,
		Description: payload.Description,
		Status:      "pending", // Статус по умолчанию
	}

	id, err := h.Store.CreateTask(r.Context(), &task)
	if err != nil {
		log.Printf("Ошибка при создании задачи в хранилище: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Не удалось создать задачу")
		return
	}

	task.ID = id // Устанавливаем ID, возвращенный из БД
	log.Printf("Задача успешно создана: ID=%d, Title=%s", task.ID, task.Title)
	respondWithJSON(w, http.StatusCreated, task)
}

// GetTasks godoc
// @Summary Получить список всех задач
// @Description Возвращает все задачи, хранящиеся в системе
// @Tags tasks
// @Produce json
// @Success 200 {array} models.Task "Список задач"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /tasks [get]
func (h *TaskHandler) GetTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.Store.GetAllTasks(r.Context())
	if err != nil {
		log.Printf("Ошибка при получении списка задач: %v", err)
		respondWithError(w, http.StatusInternalServerError, "Не удалось получить список задач")
		return
	}

	// Если задач нет, возвращаем пустой массив, а не null
	if tasks == nil {
		tasks = []models.Task{}
	}

	respondWithJSON(w, http.StatusOK, tasks)
}

// GetTask godoc
// @Summary Получить задачу по ID
// @Description Возвращает детали конкретной задачи по её идентификатору
// @Tags tasks
// @Produce json
// @Param taskID path int true "ID задачи"
// @Success 200 {object} models.Task "Найденная задача"
// @Failure 400 {object} map[string]string "Неверный ID задачи"
// @Failure 404 {object} map[string]string "Задача не найдена"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /tasks/{taskID} [get]
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "taskID")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Неверный формат ID задачи")
		return
	}

	task, err := h.Store.GetTaskByID(r.Context(), id)
	if err != nil {
		// Проверяем, была ли ошибка "не найдено"
		if errors.Is(err, sql.ErrNoRows) || strings.Contains(err.Error(), "не найдена") {
			log.Printf("Задача с ID %d не найдена", id)
			respondWithError(w, http.StatusNotFound, "Задача не найдена")
		} else {
			log.Printf("Ошибка при получении задачи %d: %v", id, err)
			respondWithError(w, http.StatusInternalServerError, "Не удалось получить задачу")
		}
		return
	}

	respondWithJSON(w, http.StatusOK, task)
}

// DeleteTask godoc
// @Summary Удалить задачу по ID
// @Description Удаляет задачу с указанным идентификатором
// @Tags tasks
// @Param taskID path int true "ID задачи для удаления"
// @Success 204 "Задача успешно удалена"
// @Failure 400 {object} map[string]string "Неверный ID задачи"
// @Failure 404 {object} map[string]string "Задача не найдена для удаления"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /tasks/{taskID} [delete]
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "taskID")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "Неверный формат ID задачи")
		return
	}

	err = h.Store.DeleteTask(r.Context(), id)
	if err != nil {
		// Проверяем специфичную ошибку "не найдено" из нашего хранилища
		if strings.Contains(err.Error(), "не найдена для удаления") {
			log.Printf("Попытка удаления несуществующей задачи с ID %d", id)
			respondWithError(w, http.StatusNotFound, "Задача не найдена для удаления")
		} else {
			log.Printf("Ошибка при удалении задачи %d: %v", id, err)
			respondWithError(w, http.StatusInternalServerError, "Не удалось удалить задачу")
		}
		return
	}

	log.Printf("Задача с ID %d успешно удалена", id)
	w.WriteHeader(http.StatusNoContent) // Успешное удаление - 204 No Content
}
