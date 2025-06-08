package handler_test

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/gorilla/mux"
)

// MockURLShortener - мок для сервиса сокращения URL.
type MockURLShortener struct{}

// ShortenURL имитирует сокращение URL для тестов.
func (m *MockURLShortener) ShortenURL(ctx context.Context, url string, userID string) (models.ShortenResult, error) {
	return models.ShortenResult{
		ShortURL: "http://localhost:8080/abc123",
		IsNew:    true,
	}, nil
}

// MockBatchURLShortener - мок для сервиса пакетного сокращения URL.
type MockBatchURLShortener struct{}

// ShortenBatch имитирует пакетное сокращение URL для тестов.
func (m *MockBatchURLShortener) ShortenBatch(ctx context.Context, batch []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	resp := []models.BatchShortenResponse{
		{
			CorrelationID: "1",
			ShortURL:      "http://localhost:8080/abc123",
		},
	}
	return resp, nil
}

// MockURLGetter - мок для сервиса получения оригинальных URL.
type MockURLGetter struct{}

// Get имитирует получение оригинального URL по короткому ID для тестов.
func (m *MockURLGetter) Get(ctx context.Context, shortID string) (string, bool) {
	return "https://example.com/original/url", true
}

// MockURLFetcher - мок для сервиса получения URL пользователя.
type MockURLFetcher struct{}

// GetURLsByUserID имитирует получение всех URL пользователя для тестов.
func (m *MockURLFetcher) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	urls := []models.UserURL{
		{
			ShortURL:    "http://localhost:8080/abc123",
			OriginalURL: "https://example.com/original/url",
		},
	}
	return urls, nil
}

// MockURLDeleter - мок для сервиса удаления URL.
type MockURLDeleter struct{}

// DeleteURLs имитирует удаление URL для тестов.
func (m *MockURLDeleter) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	return nil
}

// MockPinger - мок для сервиса проверки соединения с хранилищем.
type MockPinger struct{}

// Ping имитирует проверку соединения с хранилищем для тестов.
func (m *MockPinger) Ping(ctx context.Context) error {
	return nil
}

// MockPingerError - мок для сервиса проверки соединения с ошибкой.
type MockPingerError struct{}

// Ping имитирует ошибку соединения с хранилищем для тестов.
func (m *MockPingerError) Ping(ctx context.Context) error {
	return fmt.Errorf("database connection failed")
}

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

// ExampleShortenHandler_HandleShortenURL демонстрирует использование обработчика для сокращения URL в текстовом формате.
func ExampleShortenHandler_HandleShortenURL() {
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	baseURL := "http://localhost:8080"

	handler := handler.NewShortenHandler(shortener, batchShortener, baseURL)

	body := "https://example.com/very/long/url"
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()

	handler.HandleShortenURL(rr, req)

	resp := rr.Result()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Ошибка чтения ответа: %v\n", err)
		return
	}
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)
	fmt.Printf("Сокращенный URL получен: %t\n", len(bodyBytes) > 0)

	// Output:
	// Код ответа: 201
	// Сокращенный URL получен: true
}

// ExampleShortenHandler_HandleShortenURL_invalidInput демонстрирует обработку невалидного ввода.
func ExampleShortenHandler_HandleShortenURL_invalidInput() {
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	baseURL := "http://localhost:8080"

	handler := handler.NewShortenHandler(shortener, batchShortener, baseURL)

	body := "invalid-url"
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	rr := httptest.NewRecorder()

	handler.HandleShortenURL(rr, req)

	resp := rr.Result()
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)

	// Output:
	// Код ответа: 400
}

// ExampleShortenHandler_HandleShortenURLJSON демонстрирует использование обработчика для сокращения URL в формате JSON.
func ExampleShortenHandler_HandleShortenURLJSON() {
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	baseURL := "http://localhost:8080"

	handler := handler.NewShortenHandler(shortener, batchShortener, baseURL)

	reqBody := `{"url":"https://example.com/very/long/url"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler.HandleShortenURLJSON(rr, req)

	resp := rr.Result()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Ошибка чтения ответа: %v\n", err)
		return
	}
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	var response map[string]string
	if err := json.Unmarshal(bodyBytes, &response); err != nil {
		fmt.Printf("Ошибка разбора JSON ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)
	fmt.Printf("Результат содержит URL: %t\n", response["result"] != "")

	// Output:
	// Код ответа: 201
	// Результат содержит URL: true
}

// ExampleShortenHandler_HandleBatchShortenURL демонстрирует использование обработчика для пакетного сокращения URL.
func ExampleShortenHandler_HandleBatchShortenURL() {
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	baseURL := "http://localhost:8080"

	handler := handler.NewShortenHandler(shortener, batchShortener, baseURL)

	reqBody := `[{"correlation_id":"1","original_url":"https://example.com/url1"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler.HandleBatchShortenURL(rr, req)

	resp := rr.Result()
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)
	fmt.Printf("Тип содержимого: %s\n", resp.Header.Get("Content-Type"))

	// Output:
	// Код ответа: 201
	// Тип содержимого: application/json
}

// ExampleRedirectHandler_HandleRedirect демонстрирует использование обработчика для перенаправления по короткому URL.
func ExampleRedirectHandler_HandleRedirect() {
	getter := &MockURLGetter{}

	handler := handler.NewRedirectHandler(getter)

	r := mux.NewRouter()
	r.HandleFunc("/{id}", handler.HandleRedirect)

	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)

	rr := httptest.NewRecorder()

	r.ServeHTTP(rr, req)

	resp := rr.Result()
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)
	fmt.Printf("Location заголовок: %s\n", resp.Header.Get("Location"))

	// Output:
	// Код ответа: 307
	// Location заголовок: https://example.com/original/url
}

// ExampleUserURLsHandler_HandleGetUserURLs демонстрирует использование обработчика для получения URL пользователя.
func ExampleUserURLsHandler_HandleGetUserURLs() {
	fetcher := &MockURLFetcher{}

	handler := handler.NewUserURLsHandler(fetcher)

	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	addAuthCookies(req, "test-user")

	rr := httptest.NewRecorder()

	handler.HandleGetUserURLs(rr, req)

	resp := rr.Result()
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("Ошибка чтения ответа: %v\n", err)
		return
	}
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	var urls []models.UserURL
	if err := json.Unmarshal(bodyBytes, &urls); err != nil {
		fmt.Printf("Ошибка разбора JSON ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)
	fmt.Printf("Тип содержимого: %s\n", resp.Header.Get("Content-Type"))
	fmt.Printf("Количество URL пользователя: %d\n", len(urls))

	// Output:
	// Код ответа: 200
	// Тип содержимого: application/json
	// Количество URL пользователя: 1
}

// ExampleDeleteHandler_HandleDeleteURLs демонстрирует использование обработчика для удаления URL.
func ExampleDeleteHandler_HandleDeleteURLs() {
	deleter := &MockURLDeleter{}

	handler := handler.NewDeleteHandler(deleter)

	reqBody := `["abc123", "def456"]`
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	addAuthCookies(req, "test-user")

	rr := httptest.NewRecorder()

	handler.HandleDeleteURLs(rr, req)

	resp := rr.Result()
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)

	// Output:
	// Код ответа: 202
}

// ExampleDeleteHandler_HandleDeleteURLs_invalidJSON демонстрирует обработку невалидного JSON.
func ExampleDeleteHandler_HandleDeleteURLs_invalidJSON() {
	deleter := &MockURLDeleter{}

	handler := handler.NewDeleteHandler(deleter)

	reqBody := `invalid json`
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	addAuthCookies(req, "test-user")

	rr := httptest.NewRecorder()

	handler.HandleDeleteURLs(rr, req)

	resp := rr.Result()
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)

	// Output:
	// Код ответа: 400
}

// ExamplePingHandler_HandlePing демонстрирует использование обработчика для проверки соединения с хранилищем.
func ExamplePingHandler_HandlePing() {
	pinger := &MockPinger{}

	handler := handler.NewPingHandler(pinger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	rr := httptest.NewRecorder()

	handler.HandlePing(rr, req)

	resp := rr.Result()
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)

	// Output:
	// Код ответа: 200
}

// ExamplePingHandler_HandlePing_error демонстрирует обработку ошибки соединения.
func ExamplePingHandler_HandlePing_error() {
	pinger := &MockPingerError{}

	handler := handler.NewPingHandler(pinger)

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	rr := httptest.NewRecorder()

	handler.HandlePing(rr, req)

	resp := rr.Result()
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа: %d\n", resp.StatusCode)

	// Output:
	// Код ответа: 500
}

// ExampleURLHandler демонстрирует создание и использование комбинированного обработчика URL.
func ExampleURLHandler() {
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	getter := &MockURLGetter{}
	fetcher := &MockURLFetcher{}
	deleter := &MockURLDeleter{}
	pinger := &MockPinger{}
	baseURL := "http://localhost:8080"

	handler := handler.NewURLHandler(shortener, batchShortener, getter, fetcher, deleter, pinger, baseURL)

	reqBody := `{"url":"https://example.com/very/long/url"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler.ShortenHandler.HandleShortenURLJSON(rr, req)

	resp := rr.Result()
	if err := resp.Body.Close(); err != nil {
		fmt.Printf("Ошибка закрытия тела ответа: %v\n", err)
		return
	}

	fmt.Printf("Код ответа при сокращении URL: %d\n", resp.StatusCode)

	// Output:
	// Код ответа при сокращении URL: 201
}
