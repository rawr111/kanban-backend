package config

import (
	"flag"
	"os"
)

// Config содержит конфигурацию приложения.
type Config struct {
	ServerPort  string
	DatabaseDSN string
}

// Load загружает конфигурацию из флагов командной строки или переменных окружения.
func Load() *Config {
	cfg := &Config{}

	// Флаги командной строки (имеют приоритет)
	flag.StringVar(&cfg.ServerPort, "port", os.Getenv("PORT"), "HTTP server port (e.g., :8080)")
	flag.StringVar(&cfg.DatabaseDSN, "db-dsn", os.Getenv("DATABASE_DSN"), "PostgreSQL DSN (e.g., postgresql://user:password@host:port/dbname?sslmode=disable)")
	flag.Parse()

	// Значения по умолчанию, если не заданы ни флаги, ни переменные окружения
	if cfg.ServerPort == "" {
		cfg.ServerPort = ":8080" // Порт по умолчанию
	}
	if cfg.DatabaseDSN == "" {
		// DSN по умолчанию для локального запуска с docker-compose
		cfg.DatabaseDSN = "postgresql://user:password@localhost:5432/taskdb?sslmode=disable"
	}

	return cfg
}
