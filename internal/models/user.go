package models

import "time"

// User представляет пользователя системы.
type User struct {
	ID           int       `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Не возвращаем хеш пароля в API
	CreatedAt    time.Time `json:"created_at"`
}

// UserRegisterPayload для регистрации пользователя.
type UserRegisterPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// UserLoginPayload для логина пользователя.
type UserLoginPayload struct {
	Username string `json:"username"`
	Password string `json:"password"`
} 