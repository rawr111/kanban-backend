# --- Этап сборки ---
    FROM golang:1.23-alpine AS builder

    WORKDIR /app
    
    # Копируем файлы модулей и скачиваем зависимости
    COPY go.mod go.sum ./
    RUN go mod download
    
    # Копируем исходный код
    COPY . .
    
    # Устанавливаем swag CLI для генерации документации внутри контейнера (опционально, если не генерировать локально)
    # RUN go install github.com/swaggo/swag/cmd/swag@latest
    # RUN swag init -g cmd/server/main.go --output ./docs
    
    # Собираем приложение
    # CGO_ENABLED=0 отключает Cgo, что делает бинарник статически скомпонованным (лучше для Alpine)
    # -ldflags="-w -s" уменьшает размер бинарника, удаляя отладочную информацию
    RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o /server ./cmd/server/main.go
    
    # --- Финальный этап ---
    FROM alpine:latest
    
    WORKDIR /app
    
    # Копируем только скомпилированный бинарник из этапа сборки
    COPY --from=builder /server /app/server
    
    # Копируем сгенерированную документацию Swagger (если она не встроена в бинарник)
    # http-swagger использует сгенерированный docs/docs.go, который компилируется в бинарник,
    # так что копировать папку docs обычно не нужно. Оставим для примера.
    # COPY --from=builder /app/docs /app/docs
    
    # Открываем порт, на котором будет работать сервер
    EXPOSE 8080
    
    # Команда для запуска приложения при старте контейнера
    # Используем массив, чтобы правильно обрабатывать сигналы
    CMD ["/app/server"]
    
    # Можно передать аргументы через CMD или ENTRYPOINT, если нужно переопределить DSN/порт
    # CMD ["/app/server", "-port", ":8080", "-db-dsn", "postgresql://user:pass@dbhost:5432/dbname?sslmode=disable"]