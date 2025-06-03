package handler

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
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

// addAuthCookies добавляет корректные аутентификационные cookies к запросу,
// используя ту же логику, что и в auth пакете.
func addAuthCookies(req *http.Request, userID string) {
	secretKey := []byte("your-secret-key-change-this-in-production")
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(userID))
	signature := hex.EncodeToString(h.Sum(nil))

	req.AddCookie(&http.Cookie{Name: "user_id_id", Value: userID})
	req.AddCookie(&http.Cookie{Name: "user_id_sign", Value: signature})
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "1"})
}

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

	handler.ShortenHandler.HandleShortenURL(w, req)

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

	handler.ShortenHandler.HandleShortenURL(w, req)

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

	handler.ShortenHandler.HandleShortenURL(w, req)

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

	handler.ShortenHandler.HandleShortenURLJSON(w, req)

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

	handler.ShortenHandler.HandleShortenURLJSON(w, req)

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

	handler.ShortenHandler.HandleShortenURLJSON(w, req)

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
	router.HandleFunc("/{id}", handler.RedirectHandler.HandleRedirect).Methods(http.MethodGet)

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
	router.HandleFunc("/{id}", handler.RedirectHandler.HandleRedirect).Methods(http.MethodGet)

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

	handler.ShortenHandler.HandleBatchShortenURL(w, req)

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

	handler.ShortenHandler.HandleBatchShortenURL(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandleDeleteURLsValidInput(t *testing.T) {
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
	handler := NewDeleteHandler(serviceImpl)

	userID := "test-user"
	shortID := generator.Generate()
	if err := urlStorage.AsURLSaver().Save(context.Background(), shortID, "https://example.com", userID); err != nil {
		t.Fatalf("Failed to save URL: %v", err)
	}

	requestBody := []string{shortID}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	addAuthCookies(req, userID)

	w := httptest.NewRecorder()

	handler.HandleDeleteURLs(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected 202, got %d", w.Code)
	}
}

func TestHandleDeleteURLsUnauthorized(t *testing.T) {
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
	handler := NewDeleteHandler(serviceImpl)

	requestBody := []string{"abc123"}
	jsonBody, _ := json.Marshal(requestBody)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	// Не добавляем куки для симуляции неавторизованного запроса

	w := httptest.NewRecorder()

	handler.HandleDeleteURLs(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected 401, got %d", w.Code)
	}
}

func TestHandleDeleteURLsInvalidJSON(t *testing.T) {
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
	handler := NewDeleteHandler(serviceImpl)

	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	addAuthCookies(req, "test-user")

	w := httptest.NewRecorder()

	handler.HandleDeleteURLs(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected 400, got %d", w.Code)
	}
}

func TestHandlePingSuccess(t *testing.T) {
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
	handler := NewPingHandler(serviceImpl)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	w := httptest.NewRecorder()

	handler.HandlePing(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestHandleGetUserURLsValidInput(t *testing.T) {
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
	handler := NewUserURLsHandler(serviceImpl)

	userID := "test-user"
	shortID := generator.Generate()
	if err := urlStorage.AsURLSaver().Save(context.Background(), shortID, "https://example.com", userID); err != nil {
		t.Fatalf("Failed to save URL: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	addAuthCookies(req, userID)

	w := httptest.NewRecorder()

	handler.HandleGetUserURLs(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response []models.UserURL
	err = json.NewDecoder(w.Body).Decode(&response)
	if err != nil {
		t.Errorf("Failed to decode response: %v", err)
	}

	if len(response) != 1 {
		t.Errorf("Expected 1 URL, got %d", len(response))
	}
}

func TestHandleGetUserURLsNoContent(t *testing.T) {
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
	handler := NewUserURLsHandler(serviceImpl)

	userID := "test-user"
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	addAuthCookies(req, userID)

	w := httptest.NewRecorder()

	handler.HandleGetUserURLs(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", w.Code)
	}
}

func TestHandleGetUserURLsNoAuth(t *testing.T) {
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
	handler := NewUserURLsHandler(serviceImpl)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)
	// Не добавляем куки

	w := httptest.NewRecorder()

	handler.HandleGetUserURLs(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected 204, got %d", w.Code)
	}
	if len(w.Header().Get("Set-Cookie")) == 0 {
		t.Errorf("Expected Set-Cookie header, got none")
	}
}
