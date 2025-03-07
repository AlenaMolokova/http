package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/gorilla/mux"
)

func TestHandleShortenURL_Success(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.HandleShortenURL(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected 201, got %d", w.Code)
	}

	responseBody := w.Body.String()
	if !strings.Contains(responseBody, "http://localhost:8080/") {
		t.Errorf("Expected shortened URL, got %s", responseBody)
	}
}

func TestHandleShortenURL_InvalidContentType(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleShortenURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleShortenURL_EmptyBody(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(""))
	req.Header.Set("Content-Type", "text/plain")
	w := httptest.NewRecorder()

	handler.HandleShortenURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleShortenURLJSON_Success(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

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
	err := json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if !strings.Contains(response.Result, "http://localhost:8080/") {
		t.Errorf("Expected shortened URL, got %s", response.Result)
	}
}

func TestHandleShortenURLJSON_InvalidJSON(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	handler.HandleShortenURLJSON(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleShortenURLJSON_EmptyURL(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

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

func TestHandleRedirect_Success(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

	router := mux.NewRouter()
	router.HandleFunc("/{id}", handler.HandleRedirect).Methods(http.MethodGet)

	shortID := generator.Generate()
	originalURL := "https://example.com"
	storage.Save(shortID, originalURL)

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

func TestHandleRedirect_NotFound(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

	router := mux.NewRouter()
	router.HandleFunc("/{id}", handler.HandleRedirect).Methods(http.MethodGet)

	req := httptest.NewRequest(http.MethodGet, "/nonexistent", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleBatchShortenURL_Success(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

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
	err := json.NewDecoder(w.Body).Decode(&response)
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

func TestHandleBatchShortenURL_EmptyBatch(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

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
