package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "kanban-backend/docs" // swagger generated docs
	"kanban-backend/internal/config"
	"kanban-backend/internal/handler"
	"kanban-backend/internal/storage/postgres"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	httpSwagger "github.com/swaggo/http-swagger" // http-swagger middleware
	// Не забудьте godotenv, если используете .env файл
	// "github.com/joho/godotenv"
)

// @title Task Manager API
// @version 1.0
// @description Этот сервис предоставляет API для управления задачами.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1
// @schemes http https
func main() {
	// (Опционально) Загрузка .env файла, если он есть
	// err := godotenv.Load()
	// if err != nil && !os.IsNotExist(err) {
	//     log.Printf("Warning: could not load .env file: %v", err)
	// }

	// 1. Загрузка конфигурации
	cfg := config.Load()
	log.Printf("Конфигурация загружена: Port=%s, DB DSN (скрыто)", cfg.ServerPort)

	// 2. Инициализация хранилища (PostgreSQL)
	dbStore := postgres.NewTaskStore()
	ctx, cancelDbConnect := context.WithTimeout(context.Background(), 15*time.Second) // Таймаут на подключение
	defer cancelDbConnect()

	if err := dbStore.Connect(ctx, cfg.DatabaseDSN); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer func() {
		if err := dbStore.Close(); err != nil {
			log.Printf("Ошибка при закрытии соединения с БД: %v", err)
		}
	}()

	// 3. Выполнение миграций (просто и неидемпотентно, но для примера сойдет)
	// В реальном проекте лучше использовать инструменты миграций (migrate, goose и т.д.)
	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelMigrate()
	if err := dbStore.Migrate(migrateCtx); err != nil {
		log.Fatalf("Не удалось выполнить миграцию базы данных: %v", err)
	}

	// 4. Инициализация обработчиков
	taskHandler := handler.NewTaskHandler(dbStore)

	// 5. Настройка роутера Chi
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)                 // Добавляет ID к каждому запросу
	r.Use(middleware.RealIP)                    // Получает реальный IP клиента
	r.Use(middleware.Logger)                    // Логгирует HTTP-запросы
	r.Use(middleware.Recoverer)                 // Восстанавливается после паник
	r.Use(middleware.Timeout(60 * time.Second)) // Устанавливает таймаут на запрос

	// Маршруты API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/tasks", taskHandler.GetTasks)               // GET /api/v1/tasks
		r.Post("/tasks", taskHandler.CreateTask)            // POST /api/v1/tasks
		r.Get("/tasks/{taskID}", taskHandler.GetTask)       // GET /api/v1/tasks/123
		r.Delete("/tasks/{taskID}", taskHandler.DeleteTask) // DELETE /api/v1/tasks/123
		// Можно добавить PUT /api/v1/tasks/{taskID} для обновления
	})

	// Маршрут для Swagger UI
	// URL будет /swagger/index.html
	r.Get("/swagger/*", httpSwagger.Handler(
		// Используем относительный URL. Браузер сам правильно сформирует полный путь.
		httpSwagger.URL("/swagger/doc.json"),
	))
	// Можно также обновить лог для ясности:
	log.Printf("Swagger UI доступен по адресу: http://localhost%s/swagger/index.html (или используйте ваш актуальный хост/IP)", cfg.ServerPort)
	// Если порт не localhost:8080, замените его или используйте относительный путь /swagger/doc.json
	log.Printf("Swagger UI доступен по адресу: http://localhost%s/swagger/index.html", cfg.ServerPort)

	// 6. Настройка и запуск HTTP сервера
	server := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Канал для сигналов завершения
	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	// Запуск сервера в горутине
	go func() {
		log.Printf("Запуск сервера на порту %s...", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	// Ожидание сигнала завершения
	<-done
	log.Println("Получен сигнал завершения. Начинаем graceful shutdown...")

	// Graceful shutdown с таймаутом
	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Ошибка graceful shutdown: %v", err)
	}

	log.Println("Сервер успешно остановлен.")
}
