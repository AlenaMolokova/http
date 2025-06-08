// Package handler реализует обработчики HTTP-запросов для приложения.
package handler

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/auth"
	"github.com/AlenaMolokova/http/internal/app/models"
)

// DeleteHandler обрабатывает запросы на удаление URL.
type DeleteHandler struct {
	deleter models.URLDeleter
}

// NewDeleteHandler создает новый обработчик для удаления URL.
//
// Параметры:
//   - deleter: сервис для удаления URL
//
// Возвращает:
//   - *DeleteHandler: новый обработчик
func NewDeleteHandler(deleter models.URLDeleter) *DeleteHandler {
	return &DeleteHandler{
		deleter: deleter,
	}
}

// HandleDeleteURLs обрабатывает запросы на удаление URL.
// Поддерживает HTTP методы DELETE.
// Принимает массив идентификаторов URL для удаления в теле запроса.
// Удаление выполняется асинхронно.
//
// Коды ответа:
//   - 202 Accepted: запрос на удаление принят
//   - 400 Bad Request: неверный формат запроса
//   - 401 Unauthorized: пользователь не авторизован
//   - 500 Internal Server Error: внутренняя ошибка сервера
func (h *DeleteHandler) HandleDeleteURLs(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var shortIDs []string
	if err := json.NewDecoder(r.Body).Decode(&shortIDs); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return
	}
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			log.Printf("Failed to close request body: %v", closeErr)
		}
	}()

	if len(shortIDs) == 0 {
		http.Error(w, "Empty list of URLs", http.StatusBadRequest)
		return
	}

	if err := h.deleter.DeleteURLs(ctx, shortIDs, userID); err != nil {
		http.Error(w, "Failed to delete URLs", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
