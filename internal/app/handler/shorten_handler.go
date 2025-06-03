// Package handler реализует обработчики HTTP-запросов для приложения.
package handler

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/AlenaMolokova/http/internal/app/auth"
	"github.com/AlenaMolokova/http/internal/app/models"
)

// ShortenHandler обрабатывает запросы на сокращение URL.
type ShortenHandler struct {
	shortener models.URLShortener
	batch     models.BatchURLShortener
	baseURL   string
}

// NewShortenHandler создает новый обработчик для сокращения URL.
//
// Параметры:
//   - shortener: сервис для сокращения отдельных URL
//   - batch: сервис для пакетного сокращения URL
//   - baseURL: базовый URL сервиса
//
// Возвращает:
//   - *ShortenHandler: новый обработчик
func NewShortenHandler(shortener models.URLShortener, batch models.BatchURLShortener, baseURL string) *ShortenHandler {
	return &ShortenHandler{
		shortener: shortener,
		batch:     batch,
		baseURL:   baseURL,
	}
}

// HandleShortenURL обрабатывает запросы на сокращение URL в текстовом формате.
// Поддерживает HTTP методы POST.
// Принимает URL в теле запроса в виде текста.
// Возвращает сокращенный URL в теле ответа.
//
// Коды ответа:
//   - 201 Created: URL успешно сокращен (новый URL)
//   - 409 Conflict: URL уже был сокращен ранее
//   - 400 Bad Request: неверный формат запроса
//   - 500 Internal Server Error: внутренняя ошибка сервера
func (h *ShortenHandler) HandleShortenURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != "" && !strings.Contains(contentType, "text/plain") {
		http.Error(w, "Content-Type must be text/plain", http.StatusBadRequest)
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			log.Printf("Failed to close request body: %v", closeErr)
		}
	}()

	originalURL := strings.TrimSpace(string(body))
	if originalURL == "" {
		http.Error(w, "Empty URL", http.StatusBadRequest)
		return
	}

	if _, err := url.ParseRequestURI(originalURL); err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}

	result, err := h.shortener.ShortenURL(ctx, originalURL, userID)
	if err != nil {
		cleanErr := strings.TrimSpace(err.Error())
		http.Error(w, cleanErr, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	if result.IsNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	if _, err := io.WriteString(w, result.ShortURL); err != nil {
		log.Printf("Failed to write response: %v", err)
		return
	}
}

// HandleShortenURLJSON обрабатывает запросы на сокращение URL в формате JSON.
// Поддерживает HTTP методы POST.
// Принимает JSON-объект с полем "url" в теле запроса.
// Возвращает JSON-объект с полем "result", содержащим сокращенный URL.
//
// Коды ответа:
//   - 201 Created: URL успешно сокращен (новый URL)
//   - 409 Conflict: URL уже был сокращен ранее
//   - 400 Bad Request: неверный формат запроса
//   - 500 Internal Server Error: внутренняя ошибка сервера
func (h *ShortenHandler) HandleShortenURLJSON(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	if r.Body == nil {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			log.Printf("Failed to close request body: %v", closeErr)
		}
	}()

	w.Header().Set("Content-Type", "application/json")

	var req models.ShortenRequest
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON format"}); encErr != nil {
			log.Printf("Failed to encode error response: %v", encErr)
			return
		}
		return
	}

	if req.URL == "" {
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "URL cannot be empty"}); encErr != nil {
			log.Printf("Failed to encode error response: %v", encErr)
			return
		}
		return
	}

	if _, err := url.Parse(req.URL); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid URL format"}); encErr != nil {
			log.Printf("Failed to encode error response: %v", encErr)
			return
		}
		return
	}

	result, err := h.shortener.ShortenURL(ctx, req.URL, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Failed to shorten URL"}); encErr != nil {
			log.Printf("Failed to encode error response: %v", encErr)
			return
		}
		return
	}

	resp := models.ShortenResponse{Result: result.ShortURL}
	if result.IsNew {
		w.WriteHeader(http.StatusCreated)
	} else {
		w.WriteHeader(http.StatusConflict)
	}
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		log.Printf("Failed to encode response: %v", encErr)
		return
	}
}

// HandleBatchShortenURL обрабатывает запросы на пакетное сокращение URL.
// Поддерживает HTTP методы POST.
// Принимает массив JSON-объектов с полями "correlation_id" и "original_url" в теле запроса.
// Возвращает массив JSON-объектов с полями "correlation_id" и "short_url".
//
// Коды ответа:
//   - 201 Created: URLs успешно сокращены
//   - 400 Bad Request: неверный формат запроса
//   - 500 Internal Server Error: внутренняя ошибка сервера
func (h *ShortenHandler) HandleBatchShortenURL(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	userID, err := auth.GetUserIDFromCookie(r)
	if err != nil {
		userID = auth.GenerateUserID()
		auth.SetUserIDCookie(w, userID)
	}

	if r.Body == nil {
		http.Error(w, "Empty request body", http.StatusBadRequest)
		return
	}
	defer func() {
		if closeErr := r.Body.Close(); closeErr != nil {
			log.Printf("Failed to close request body: %v", closeErr)
		}
	}()

	w.Header().Set("Content-Type", "application/json")

	var req []models.BatchShortenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid JSON format"}); encErr != nil {
			log.Printf("Failed to encode error response: %v", encErr)
			return
		}
		return
	}

	if len(req) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Empty batch"}); encErr != nil {
			log.Printf("Failed to encode error response: %v", encErr)
			return
		}
		return
	}

	for _, item := range req {
		if item.OriginalURL == "" {
			w.WriteHeader(http.StatusBadRequest)
			if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "URL cannot be empty"}); encErr != nil {
				log.Printf("Failed to encode error response: %v", encErr)
				return
			}
			return
		}
		if _, err := url.Parse(item.OriginalURL); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Invalid URL format"}); encErr != nil {
				log.Printf("Failed to encode error response: %v", encErr)
				return
			}
			return
		}
	}

	resp, err := h.batch.ShortenBatch(ctx, req, userID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		if encErr := json.NewEncoder(w).Encode(map[string]string{"error": "Failed to shorten batch"}); encErr != nil {
			log.Printf("Failed to encode error response: %v", encErr)
			return
		}
		return
	}

	w.WriteHeader(http.StatusCreated)
	if encErr := json.NewEncoder(w).Encode(resp); encErr != nil {
		log.Printf("Failed to encode response: %v", encErr)
		return
	}
}
