package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage"
	"github.com/gorilla/mux"
)

func TestHandleShortenURLValidInput(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.HandleShortenURL(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	responseBody := w.Body.String()
	if !strings.Contains(responseBody, cfg.BaseURL+"/") {
		t.Errorf("Expected shortened URL starting with %s, got %s", cfg.BaseURL, responseBody)
	}
}

func TestHandleShortenURLInvalidContentType(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleShortenURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleShortenURLEmptyBody(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.HandleShortenURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleShortenURLJSONValidInput(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	requestBody := models.ShortenRequest{URL: "https://example.com"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleShortenURLJSON(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var response models.ShortenResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Errorf("Expected shortened URL, got %s", response.Result)
	}
}

func TestHandleShortenURLJSONInvalidJSON(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleShortenURLJSON(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleShortenURLJSONEmptyURL(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	requestBody := models.ShortenRequest{URL: ""}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleShortenURLJSON(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleRedirectValidID(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	router := mux.NewRouter()
	router.HandleFunc("/{id}", handler.HandleRedirect).Methods(http.MethodGet)

	shortID := generator.Generate()
	originalURL := "https://example.com"
	userID := "test-user"
	if err := urlStorage.AsURLSaver().Save(context.Background(), shortID, originalURL, userID); err != nil {
		t.Fatalf("Failed to save URL: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusTemporaryRedirect {
		t.Errorf("Expected 307, got %d", w.Code)
	}

	if location := w.Header().Get("Location"); location != originalURL {
		t.Errorf("Expected redirect to %s, got %s", originalURL, location)
	}
}

func TestHandleRedirectNotFound(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	router := mux.NewRouter()
	router.HandleFunc("/{id}", handler.HandleRedirect).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusGone {
		t.Errorf("Expected 410, got %d", w.Code)
	}
}

func TestHandleBatchShortenURLValidInput(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	requestBatch := []models.BatchShortenRequest{
		{CorrelationID: "1", OriginalURL: "https://example1.com"},
		{CorrelationID: "2", OriginalURL: "https://example2.com"},
	}
	jsonBody, _ := json.Marshal(requestBatch)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleBatchShortenURL(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	var response []models.BatchShortenResponse
	err = json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(response) != 2 {
		t.Errorf("Expected 2 items in response, got %d", len(response))
	}

	for _, item := range response {
		if item.CorrelationID != "1" && item.CorrelationID != "2" {
			t.Errorf("Unexpected correlation_id: %s", item.CorrelationID)
		}
		if !strings.Contains(item.ShortURL, "http://localhost:8080/") {
			t.Errorf("Expected shortened URL, got %s", item.ShortURL)
		}
	}
}

func TestHandleBatchShortenURLEmptyBatch(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	urlStorage, err := storage.NewStorage("", "")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}
	generator := generator.NewGenerator(8)
	serviceImpl := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		generator,
		cfg.BaseURL,
	)
	handler := NewURLHandler(serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, serviceImpl, cfg.BaseURL)

	requestBatch := []models.BatchShortenRequest{}
	jsonBody, _ := json.Marshal(requestBatch)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleBatchShortenURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}
