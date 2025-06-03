// Package handler реализует обработчики HTTP-запросов для приложения.
package handler

import (
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/gorilla/mux"
)

// RedirectHandler обрабатывает запросы на перенаправление по короткому URL.
type RedirectHandler struct {
	redirector models.URLGetter
}

// NewRedirectHandler создает новый обработчик для перенаправления по коротким URL.
//
// Параметры:
//   - redirector: сервис для получения оригинальных URL
//
// Возвращает:
//   - *RedirectHandler: новый обработчик
func NewRedirectHandler(redirector models.URLGetter) *RedirectHandler {
	return &RedirectHandler{
		redirector: redirector,
	}
}

// HandleRedirect обрабатывает запросы на перенаправление по короткому URL.
// Поддерживает HTTP методы GET.
// Извлекает идентификатор из URL-пути и перенаправляет на оригинальный URL.
//
// Коды ответа:
//   - 307 Temporary Redirect: успешное перенаправление
//   - 410 Gone: URL был удален или не существует
func (h *RedirectHandler) HandleRedirect(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	vars := mux.Vars(r)
	id := vars["id"]

	originalURL, found := h.redirector.Get(ctx, id)
	if !found {
		http.Error(w, "Gone", http.StatusGone)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
