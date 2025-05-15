package handler

import (
	"encoding/json"
	"net/http"
	"strings"

	"kanban-backend/internal/models"
	"kanban-backend/internal/storage"
	"kanban-backend/internal/auth"

	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	UserStore storage.UserStore
}

func NewAuthHandler(userStore storage.UserStore) *AuthHandler {
	return &AuthHandler{UserStore: userStore}
}

// Register godoc
// @Summary Регистрация нового пользователя
// @Description Регистрирует нового пользователя по username и password
// @Tags auth
// @Accept json
// @Produce json
// @Param user body models.UserRegisterPayload true "Данные для регистрации"
// @Success 201 {object} map[string]string "Пользователь успешно зарегистрирован"
// @Failure 400 {object} map[string]string "Ошибка валидации или пользователь уже существует"
// @Router /register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var payload models.UserRegisterPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	defer r.Body.Close()

	payload.Username = strings.TrimSpace(payload.Username)
	if payload.Username == "" || payload.Password == "" {
		respondWithError(w, http.StatusBadRequest, "Username and password required")
		return
	}

	user := &models.User{Username: payload.Username}
	_, err := h.UserStore.CreateUser(r.Context(), user, payload.Password)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, "User already exists or DB error: "+err.Error())
		return
	}
	respondWithJSON(w, http.StatusCreated, map[string]string{"message": "User registered successfully"})
}

// Login godoc
// @Summary Аутентификация пользователя
// @Description Возвращает JWT-токен при успешном логине
// @Tags auth
// @Accept json
// @Produce json
// @Param credentials body models.UserLoginPayload true "Данные для входа"
// @Success 200 {object} map[string]string "Успешная аутентификация"
// @Failure 401 {object} map[string]string "Неверные имя пользователя или пароль"
// @Router /login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var payload models.UserLoginPayload
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		respondWithError(w, http.StatusBadRequest, "Invalid JSON: "+err.Error())
		return
	}
	defer r.Body.Close()

	user, err := h.UserStore.GetUserByUsername(r.Context(), payload.Username)
	if err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(payload.Password)); err != nil {
		respondWithError(w, http.StatusUnauthorized, "Invalid username or password")
		return
	}
	token, err := auth.GenerateJWT(user.ID, user.Username)
	if err != nil {
		respondWithError(w, http.StatusInternalServerError, "Failed to generate token")
		return
	}
	respondWithJSON(w, http.StatusOK, map[string]string{"token": token})
} 