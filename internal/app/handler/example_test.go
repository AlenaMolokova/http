package handler_test

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/gorilla/mux"
)

// Примеры в этом файле демонстрируют использование обработчиков URL сервиса.
// Для работы с реальными данными необходимо заменить моки на реальные реализации сервисов.

// MockURLShortener - мок для сервиса сокращения URL
type MockURLShortener struct{}

func (m *MockURLShortener) ShortenURL(ctx context.Context, url string, userID string) (models.ShortenResult, error) {
	return models.ShortenResult{
		ShortURL: "http://localhost:8080/abc123",
		IsNew:    true,
	}, nil
}

// MockBatchURLShortener - мок для сервиса пакетного сокращения URL
type MockBatchURLShortener struct{}

func (m *MockBatchURLShortener) ShortenBatch(ctx context.Context, batch []models.BatchShortenRequest, userID string) ([]models.BatchShortenResponse, error) {
	resp := []models.BatchShortenResponse{
		{
			CorrelationID: "1",
			ShortURL:      "http://localhost:8080/abc123",
		},
	}
	return resp, nil
}

// MockURLGetter - мок для сервиса получения оригинальных URL
type MockURLGetter struct{}

func (m *MockURLGetter) Get(ctx context.Context, shortID string) (string, bool) {
	return "https://example.com/original/url", true
}

// MockURLFetcher - мок для сервиса получения URL пользователя
type MockURLFetcher struct{}

func (m *MockURLFetcher) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	urls := []models.UserURL{
		{
			ShortURL:    "http://localhost:8080/abc123",
			OriginalURL: "https://example.com/original/url",
		},
	}
	return urls, nil
}

// MockURLDeleter - мок для сервиса удаления URL
type MockURLDeleter struct{}

func (m *MockURLDeleter) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	return nil
}

// MockPinger - мок для сервиса проверки соединения с хранилищем
type MockPinger struct{}

func (m *MockPinger) Ping(ctx context.Context) error {
	return nil
}

// addAuthCookies добавляет корректные аутентификационные cookies к запросу
// используя ту же логику, что и в auth пакете
func addAuthCookies(req *http.Request, userID string) {
	// Секретный ключ для подписи (тот же, что в auth пакете)
	secretKey := []byte("your-secret-key-change-this-in-production")

	// Создаем HMAC подпись
	h := hmac.New(sha256.New, secretKey)
	h.Write([]byte(userID))
	signature := hex.EncodeToString(h.Sum(nil))

	// Добавляем все необходимые cookies
	req.AddCookie(&http.Cookie{Name: "user_id_id", Value: userID})
	req.AddCookie(&http.Cookie{Name: "user_id_sign", Value: signature})
	req.AddCookie(&http.Cookie{Name: "user_id", Value: "1"})
}

// ExampleShortenHandler_HandleShortenURL демонстрирует использование обработчика для сокращения URL в текстовом формате.
func ExampleShortenHandler_HandleShortenURL() {
	// Создаем моки сервисов
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	baseURL := "http://localhost:8080"

	// Создаем обработчик
	handler := handler.NewShortenHandler(shortener, batchShortener, baseURL)

	// Создаем тестовый запрос
	body := "https://example.com/very/long/url"
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(body))
	req.Header.Set("Content-Type", "text/plain")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.HandleShortenURL(rr, req)

	// Проверяем код ответа
	fmt.Printf("Код ответа: %d\n", rr.Code)
	fmt.Printf("Сокращенный URL получен: %t\n", len(rr.Body.String()) > 0)

	// Не забываем закрыть тело ответа
	_ = rr.Result().Body.Close()

	// Output:
	// Код ответа: 201
	// Сокращенный URL получен: true
}

// ExampleShortenHandler_HandleShortenURLJSON демонстрирует использование обработчика для сокращения URL в формате JSON.
func ExampleShortenHandler_HandleShortenURLJSON() {
	// Создаем моки сервисов
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	baseURL := "http://localhost:8080"

	// Создаем обработчик
	handler := handler.NewShortenHandler(shortener, batchShortener, baseURL)

	// Создаем JSON-запрос
	reqBody := `{"url":"https://example.com/very/long/url"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.HandleShortenURLJSON(rr, req)

	// Проверяем код ответа и содержимое ответа
	fmt.Printf("Код ответа: %d\n", rr.Code)

	var response map[string]string
	json.Unmarshal(rr.Body.Bytes(), &response)
	fmt.Printf("Результат содержит сокращенный URL: %t\n", response["result"] != "")

	// Не забываем закрыть тело ответа
	_ = rr.Result().Body.Close()

	// Output:
	// Код ответа: 201
	// Результат содержит сокращенный URL: true
}

// ExampleShortenHandler_HandleBatchShortenURL демонстрирует использование обработчика для пакетного сокращения URL.
func ExampleShortenHandler_HandleBatchShortenURL() {
	// Создаем моки сервисов
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	baseURL := "http://localhost:8080"

	// Создаем обработчик
	handler := handler.NewShortenHandler(shortener, batchShortener, baseURL)

	// Создаем JSON-запрос для пакетного сокращения
	reqBody := `[{"correlation_id":"1","original_url":"https://example.com/url1"}]`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.HandleBatchShortenURL(rr, req)

	// Проверяем код ответа
	fmt.Printf("Код ответа: %d\n", rr.Code)
	fmt.Printf("Тип содержимого: %s\n", rr.Header().Get("Content-Type"))

	// Не забываем закрыть тело ответа
	_ = rr.Result().Body.Close()

	// Output:
	// Код ответа: 201
	// Тип содержимого: application/json
}

// ExampleRedirectHandler_HandleRedirect демонстрирует использование обработчика для перенаправления по короткому URL.
func ExampleRedirectHandler_HandleRedirect() {
	// Создаем моки сервисов
	getter := &MockURLGetter{}
	fetcher := &MockURLFetcher{}
	baseURL := "http://localhost:8080"

	// Создаем обработчик
	handler := handler.NewRedirectHandler(getter, fetcher, baseURL)

	// Создаем маршрутизатор Gorilla Mux для получения параметров из URL
	r := mux.NewRouter()
	r.HandleFunc("/{id}", handler.HandleRedirect)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodGet, "/abc123", nil)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Обрабатываем запрос через маршрутизатор
	r.ServeHTTP(rr, req)

	// Проверяем код ответа и заголовок Location
	fmt.Printf("Код ответа: %d\n", rr.Code)
	fmt.Printf("Location заголовок: %s\n", rr.Header().Get("Location"))

	// Не забываем закрыть тело ответа
	_ = rr.Result().Body.Close()

	// Output:
	// Код ответа: 307
	// Location заголовок: https://example.com/original/url
}

// ExampleUserURLsHandler_HandleGetUserURLs демонстрирует использование обработчика для получения URL пользователя.
func ExampleUserURLsHandler_HandleGetUserURLs() {
	// Создаем мок сервиса
	fetcher := &MockURLFetcher{}

	// Создаем обработчик
	handler := handler.NewUserURLsHandler(fetcher)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

	// Добавляем корректные аутентификационные cookies
	addAuthCookies(req, "test-user")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.HandleGetUserURLs(rr, req)

	// Проверяем код ответа и содержимое ответа
	fmt.Printf("Код ответа: %d\n", rr.Code)
	fmt.Printf("Тип содержимого: %s\n", rr.Header().Get("Content-Type"))

	var urls []models.UserURL
	json.Unmarshal(rr.Body.Bytes(), &urls)
	fmt.Printf("Количество URL пользователя: %d\n", len(urls))

	// Не забываем закрыть тело ответа
	_ = rr.Result().Body.Close()

	// Output:
	// Код ответа: 200
	// Тип содержимого: application/json
	// Количество URL пользователя: 1
}

// ExampleDeleteHandler_HandleDeleteURLs демонстрирует использование обработчика для удаления URL.
func ExampleDeleteHandler_HandleDeleteURLs() {
	// Создаем мок сервиса
	deleter := &MockURLDeleter{}

	// Создаем обработчик
	handler := handler.NewDeleteHandler(deleter)

	// Создаем JSON-запрос для удаления URL
	reqBody := `["abc123", "def456"]`
	req := httptest.NewRequest(http.MethodDelete, "/api/user/urls", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	// Добавляем корректные аутентификационные cookies
	addAuthCookies(req, "test-user")

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.HandleDeleteURLs(rr, req)

	// Проверяем код ответа
	fmt.Printf("Код ответа: %d\n", rr.Code)

	// Не забываем закрыть тело ответа
	_ = rr.Result().Body.Close()

	// Output:
	// Код ответа: 202
}

// ExamplePingHandler_HandlePing демонстрирует использование обработчика для проверки соединения с хранилищем.
func ExamplePingHandler_HandlePing() {
	// Создаем мок сервиса
	pinger := &MockPinger{}

	// Создаем обработчик
	handler := handler.NewPingHandler(pinger)

	// Создаем тестовый запрос
	req := httptest.NewRequest(http.MethodGet, "/ping", nil)

	// Создаем ResponseRecorder для записи ответа
	rr := httptest.NewRecorder()

	// Вызываем обработчик
	handler.HandlePing(rr, req)

	// Проверяем код ответа
	fmt.Printf("Код ответа: %d\n", rr.Code)

	// Не забываем закрыть тело ответа
	_ = rr.Result().Body.Close()

	// Output:
	// Код ответа: 200
}

// ExampleURLHandler демонстрирует создание и использование комбинированного обработчика URL.
func ExampleURLHandler() {
	// Создаем моки всех сервисов
	shortener := &MockURLShortener{}
	batchShortener := &MockBatchURLShortener{}
	getter := &MockURLGetter{}
	fetcher := &MockURLFetcher{}
	deleter := &MockURLDeleter{}
	pinger := &MockPinger{}
	baseURL := "http://localhost:8080"

	// Создаем комбинированный обработчик
	handler := handler.NewURLHandler(shortener, batchShortener, getter, fetcher, deleter, pinger, baseURL)

	// Пример использования для сокращения URL в формате JSON
	reqBody := `{"url":"https://example.com/very/long/url"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()

	handler.HandleShortenURLJSON(rr, req)

	fmt.Printf("Код ответа при сокращении URL: %d\n", rr.Code)

	// Не забываем закрыть тело ответа
	_ = rr.Result().Body.Close()

	// Output:
	// Код ответа при сокращении URL: 201
}
