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
	"kanban-backend/internal/auth"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors" // <--- ДОБАВЬ ЭТОТ ИМПОРТ
	httpSwagger "github.com/swaggo/http-swagger"
)

// ... (остальные твои комментарии Swagger)

func main() {
	cfg := config.Load()
	log.Printf("Конфигурация загружена: Port=%s, DB DSN (скрыто)", cfg.ServerPort)

	dbStore := postgres.NewTaskStore()
	ctx, cancelDbConnect := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelDbConnect()

	if err := dbStore.Connect(ctx, cfg.DatabaseDSN); err != nil {
		log.Fatalf("Не удалось подключиться к базе данных: %v", err)
	}
	defer func() {
		if err := dbStore.Close(); err != nil {
			log.Printf("Ошибка при закрытии соединения с БД: %v", err)
		}
	}()

	migrateCtx, cancelMigrate := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancelMigrate()
	if err := dbStore.Migrate(migrateCtx); err != nil {
		log.Fatalf("Не удалось выполнить миграцию базы данных: %v", err)
	}

	// --- UserStore и AuthHandler ---
	userStore := postgres.NewUserStore(dbStore.DB()) // Получаем *sql.DB из TaskStore
	if err := userStore.Migrate(migrateCtx); err != nil {
		log.Fatalf("Не удалось выполнить миграцию users: %v", err)
	}
	authHandler := handler.NewAuthHandler(userStore)

	taskHandler := handler.NewTaskHandler(dbStore)

	r := chi.NewRouter()

	// --- НАСТРОЙКА CORS ---
	// Это должно быть одним из первых middleware
	corsMiddleware := cors.New(cors.Options{
		// AllowedOrigins: []string{"*"}, // Разрешить все источники (менее безопасно для продакшена)
		AllowedOrigins:   []string{"http://localhost:3000"}, // Конкретно твой фронтенд
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true,
		MaxAge:           300, // Максимальное время кеширования preflight запроса в секундах
	})
	r.Use(corsMiddleware.Handler) // <--- ПРИМЕНИТЬ CORS MIDDLEWARE

	// Остальные Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))

	// Маршруты API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Auth middleware только для задач
		r.Group(func(r chi.Router) {
			r.Use(auth.JWTAuthMiddleware)
			r.Get("/tasks", taskHandler.GetTasks)
			r.Post("/tasks", taskHandler.CreateTask)
			r.Get("/tasks/{taskID}", taskHandler.GetTask)
			r.Delete("/tasks/{taskID}", taskHandler.DeleteTask)
		})
		// --- Auth routes ---
		r.Post("/register", authHandler.Register)
		r.Post("/login", authHandler.Login)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
	))
	log.Printf("Swagger UI доступен по адресу: http://localhost%s/swagger/index.html", cfg.ServerPort)


	server := &http.Server{
		Addr:         cfg.ServerPort,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Запуск сервера на портА %s...", cfg.ServerPort)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	<-done
	log.Println("Получен сигнал завершения. Начинаем graceful shutdown...")

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		log.Fatalf("Ошибка graceful shutdown: %v", err)
	}

	log.Println("Сервер успешно остановлен.")
}