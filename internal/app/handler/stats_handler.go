// Package handler содержит HTTP-обработчики для различных эндпоинтов.
package handler

import (
	"encoding/json"
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
)

// StatsHandler обрабатывает запросы на получение статистики сервиса.
type StatsHandler struct {
	statsProvider models.StatsProvider
}

// NewStatsHandler создает новый экземпляр StatsHandler.
func NewStatsHandler(statsProvider models.StatsProvider) *StatsHandler {
	return &StatsHandler{
		statsProvider: statsProvider,
	}
}

// HandleStats обрабатывает GET-запрос на получение статистики сервиса.
// Возвращает JSON с количеством сокращенных URL и пользователей.
func (h *StatsHandler) HandleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats, err := h.statsProvider.GetStats(r.Context())
	if err != nil {
		logrus.WithError(err).Error("Failed to get stats")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(stats); err != nil {
		logrus.WithError(err).Error("Failed to encode stats response")
		return
	}
}
