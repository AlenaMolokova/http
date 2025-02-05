package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/gorilla/mux"
)

func TestHandleShortenURL_Success(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewSimpleGenerator(8)
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
	generator := generator.NewSimpleGenerator(8)
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
	generator := generator.NewSimpleGenerator(8)
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

func TestHandleRedirect_Success(t *testing.T) {
	cfg := &config.Config{BaseURL: "http://localhost:8080"}
	storage := memory.NewMemoryStorage()
	generator := generator.NewSimpleGenerator(8)
	service := service.NewURLService(storage, generator, cfg.BaseURL)
	handler := NewHandler(service)

	router := mux.NewRouter()
	router.HandleFunc("/{id}", handler.HandleRedirect).Methods(http.MethodGet)

	// Сохраняем тестовый URL в хранилище
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
	generator := generator.NewSimpleGenerator(8)
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
