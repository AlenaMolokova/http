// Package handler реализует обработчики HTTP-запросов для приложения.
package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
)

// PingHandler обрабатывает запросы на проверку соединения с хранилищем.
type PingHandler struct {
	pinger models.Pinger
}

// NewPingHandler создает новый обработчик для проверки соединения с хранилищем.
//
// Параметры:
//   - pinger: сервис для проверки соединения
//
// Возвращает:
//   - *PingHandler: новый обработчик
func NewPingHandler(pinger models.Pinger) *PingHandler {
	return &PingHandler{
		pinger: pinger,
	}
}

// HandlePing обрабатывает запросы на проверку соединения с хранилищем данных.
// Поддерживает HTTP методы GET.
// Проверяет доступность базы данных.
//
// Коды ответа:
//   - 200 OK: соединение с базой данных установлено или хранилище не требует проверки соединения
//   - 500 Internal Server Error: ошибка соединения с базой данных
func (h *PingHandler) HandlePing(w http.ResponseWriter, r *http.Request) {
	// Проверяем только GET запросы
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	ctx := r.Context()

	// Добавляем таймаут для операции ping
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := h.pinger.Ping(ctx)
	if err != nil {
		logrus.WithError(err).Error("Ping failed")

		// Проверяем специальные случаи для файлового и in-memory хранилищ
		if err.Error() == "file storage does not support database connection check" ||
			err.Error() == "memory storage does not support database connection check" {
			w.WriteHeader(http.StatusOK)
			w.Header().Set("Content-Type", "text/plain")
			if _, writeErr := w.Write([]byte("Storage does not require database connection")); writeErr != nil {
				logrus.WithError(writeErr).Error("Failed to write response")
			}
			return
		}

		http.Error(w, "Database connection error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "text/plain")
	if _, writeErr := w.Write([]byte("Database connection is OK")); writeErr != nil {
		logrus.WithError(writeErr).Error("Failed to write response")
	}
}
