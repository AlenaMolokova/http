// Package handler реализует обработчики HTTP-запросов для приложения.
package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/auth"
	"github.com/AlenaMolokova/http/internal/app/models"
)

// UserURLsHandler обрабатывает запросы на получение URL, принадлежащих пользователю.
type UserURLsHandler struct {
	fetcher models.URLFetcher
}

// NewUserURLsHandler создает новый обработчик для получения URL пользователя.
//
// Параметры:
//   - fetcher: сервис для получения URL пользователя
//
// Возвращает:
//   - *UserURLsHandler: новый обработчик
func NewUserURLsHandler(fetcher models.URLFetcher) *UserURLsHandler {
	return &UserURLsHandler{
		fetcher: fetcher,
	}
}

// HandleGetUserURLs обрабатывает запросы на получение всех URL, принадлежащих пользователю.
// Поддерживает HTTP методы GET.
// Извлекает идентификатор пользователя из cookie и возвращает список его URL.
//
// Коды ответа:
//   - 200 OK: список URL успешно получен
//   - 204 No Content: у пользователя нет сохраненных URL
//   - 500 Internal Server Error: внутренняя ошибка сервера
func (h *UserURLsHandler) HandleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	urls, err := h.fetcher.GetURLsByUserID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user URLs", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	encoder := json.NewEncoder(w)
	if encErr := encoder.Encode(urls); encErr != nil {
		log.Printf("Failed to encode response: %v", encErr)
		return
	}
}
