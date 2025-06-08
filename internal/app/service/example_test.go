package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockStorage имитирует хранилище для тестирования.
type mockStorage struct {
	urls     map[string]string            // shortID -> originalURL
	userURLs map[string]map[string]string // userID -> shortID -> originalURL
	deleted  map[string]bool              // shortID -> isDeleted
}

// newMockStorage создает новый экземпляр mockStorage.
func newMockStorage() *mockStorage {
	return &mockStorage{
		urls:     make(map[string]string),
		userURLs: make(map[string]map[string]string),
		deleted:  make(map[string]bool),
	}
}

// Save сохраняет URL в хранилище.
func (m *mockStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	m.urls[shortID] = originalURL
	if _, ok := m.userURLs[userID]; !ok {
		m.userURLs[userID] = make(map[string]string)
	}
	m.userURLs[userID][shortID] = originalURL
	return nil
}

// SaveBatch сохраняет пакет URL в хранилище.
func (m *mockStorage) SaveBatch(ctx context.Context, batch map[string]string, userID string) error {
	for shortID, originalURL := range batch {
		m.urls[shortID] = originalURL
		if _, ok := m.userURLs[userID]; !ok {
			m.userURLs[userID] = make(map[string]string)
		}
		m.userURLs[userID][shortID] = originalURL
	}
	return nil
}

// Get получает оригинальный URL по короткому ID.
func (m *mockStorage) Get(ctx context.Context, shortID string) (string, bool) {
	originalURL, ok := m.urls[shortID]
	if !ok {
		return "", false
	}
	if m.deleted[shortID] {
		return originalURL, false
	}
	return originalURL, true
}

// FindByOriginalURL находит короткий ID по оригинальному URL.
func (m *mockStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	for shortID, url := range m.urls {
		if url == originalURL && !m.deleted[shortID] {
			return shortID, nil
		}
	}
	return "", nil
}

// GetURLsByUserID возвращает все URL пользователя.
func (m *mockStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	userURLs, ok := m.userURLs[userID]
	if !ok {
		return []models.UserURL{}, nil
	}

	result := make([]models.UserURL, 0, len(userURLs))
	for shortID, originalURL := range userURLs {
		if !m.deleted[shortID] {
			result = append(result, models.UserURL{
				ShortURL:    shortID,
				OriginalURL: originalURL,
			})
		}
	}

	// Сортируем результат для предсказуемого порядка вывода
	sort.Slice(result, func(i, j int) bool {
		return result[i].ShortURL < result[j].ShortURL
	})

	return result, nil
}

// DeleteURLs помечает URL как удаленные.
func (m *mockStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	for _, shortID := range shortIDs {
		if userURLs, ok := m.userURLs[userID]; ok {
			if _, exists := userURLs[shortID]; exists {
				m.deleted[shortID] = true
			}
		}
	}
	return nil
}

// Ping имитирует проверку соединения с хранилищем.
func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

// mockGenerator имитирует генератор коротких идентификаторов.
type mockGenerator struct {
	counter int
}

// Generate создает новый короткий идентификатор.
func (g *mockGenerator) Generate() string {
	g.counter++
	return fmt.Sprintf("short%d", g.counter)
}

// Example_shortenURL демонстрирует сокращение одного URL с использованием сервиса.
func Example_shortenURL() {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	userID := "user123"
	originalURL := "https://very-long-url.com/with/path/and?query=parameters"

	result, err := svc.ShortenURL(ctx, originalURL, userID)
	if err != nil {
		fmt.Printf("Ошибка при сокращении URL: %v\n", err)
		return
	}

	fmt.Printf("Короткий URL: %s\n", result.ShortURL)
	fmt.Printf("Это новый URL: %v\n", result.IsNew)

	shortID := strings.TrimPrefix(result.ShortURL, baseURL+"/")
	originalURL, found := svc.Get(ctx, shortID)

	if found {
		fmt.Printf("Найден оригинальный URL: %s\n", originalURL)
	} else {
		fmt.Println("URL не найден")
	}

	// Output:
	// Короткий URL: http://example.com/short1
	// Это новый URL: true
	// Найден оригинальный URL: https://very-long-url.com/with/path/and?query=parameters
}

// Example_shortenBatch демонстрирует пакетное сокращение URL с использованием сервиса.
func Example_shortenBatch() {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	userID := "user123"
	batch := []models.BatchShortenRequest{
		{
			CorrelationID: "id1",
			OriginalURL:   "https://example1.com",
		},
		{
			CorrelationID: "id2",
			OriginalURL:   "https://example2.com",
		},
	}

	results, err := svc.ShortenBatch(ctx, batch, userID)
	if err != nil {
		fmt.Printf("Ошибка при пакетном сокращении URL: %v\n", err)
		return
	}

	for _, result := range results {
		fmt.Printf("Correlation ID: %s, Short URL: %s\n", result.CorrelationID, result.ShortURL)
	}

	// Output:
	// Correlation ID: id1, Short URL: http://example.com/short1
	// Correlation ID: id2, Short URL: http://example.com/short2
}

// Example_getUserURLs демонстрирует получение всех URL пользователя.
func Example_getUserURLs() {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	userID := "user123"
	urls := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
	}

	for _, url := range urls {
		_, err := svc.ShortenURL(ctx, url, userID)
		if err != nil {
			fmt.Printf("Ошибка при сокращении URL: %v\n", err)
			return
		}
	}

	userURLs, err := svc.GetURLsByUserID(ctx, userID)
	if err != nil {
		fmt.Printf("Ошибка при получении URL пользователя: %v\n", err)
		return
	}

	fmt.Printf("Найдено %d URL для пользователя %s:\n", len(userURLs), userID)
	for _, url := range userURLs {
		fmt.Printf("Короткий URL: %s, Оригинальный URL: %s\n", url.ShortURL, url.OriginalURL)
	}

	// Output:
	// Найдено 3 URL для пользователя user123:
	// Короткий URL: http://example.com/short1, Оригинальный URL: https://example1.com
	// Короткий URL: http://example.com/short2, Оригинальный URL: https://example2.com
	// Короткий URL: http://example.com/short3, Оригинальный URL: https://example3.com
}

// Example_deleteURLs демонстрирует удаление URL пользователя.
func Example_deleteURLs() {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	userID := "user123"
	urls := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
	}

	for _, url := range urls {
		_, err := svc.ShortenURL(ctx, url, userID)
		if err != nil {
			fmt.Printf("Ошибка при сокращении URL: %v\n", err)
			return
		}
	}

	shortIDs := []string{"short2"}
	err := svc.DeleteURLs(ctx, shortIDs, userID)
	if err != nil {
		fmt.Printf("Ошибка при удалении URL: %v\n", err)
		return
	}

	_, found := svc.Get(ctx, "short2")
	fmt.Printf("URL short2 доступен: %v\n", found)

	userURLs, err := svc.GetURLsByUserID(ctx, userID)
	if err != nil {
		fmt.Printf("Ошибка при получении URL пользователя: %v\n", err)
		return
	}

	fmt.Printf("Осталось %d URL для пользователя %s\n", len(userURLs), userID)

	// Output:
	// URL short2 доступен: false
	// Осталось 2 URL для пользователя user123
}

// TestShortenURL тестирует сокращение одного URL.
func TestShortenURL(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	userID := "user123"
	originalURL := "https://very-long-url.com/with/path"

	result, err := svc.ShortenURL(ctx, originalURL, userID)

	assert.NoError(t, err)
	assert.Equal(t, "http://example.com/short1", result.ShortURL)
	assert.True(t, result.IsNew)

	result, err = svc.ShortenURL(ctx, originalURL, userID)

	assert.NoError(t, err)
	assert.Equal(t, "http://example.com/short1", result.ShortURL)
	assert.False(t, result.IsNew)
}

// TestShortenBatch тестирует пакетное сокращение URL.
func TestShortenBatch(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	userID := "user123"
	batch := []models.BatchShortenRequest{
		{
			CorrelationID: "id1",
			OriginalURL:   "https://example1.com",
		},
		{
			CorrelationID: "id2",
			OriginalURL:   "https://example2.com",
		},
	}

	results, err := svc.ShortenBatch(ctx, batch, userID)

	require.NoError(t, err)
	require.Len(t, results, 2)

	correlationMap := make(map[string]string)
	for _, result := range results {
		correlationMap[result.CorrelationID] = result.ShortURL
	}

	assert.Contains(t, correlationMap, "id1")
	assert.Contains(t, correlationMap, "id2")
	assert.NotEqual(t, correlationMap["id1"], correlationMap["id2"])
}

// TestGetURLsByUserID тестирует получение всех URL пользователя.
func TestGetURLsByUserID(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	userID := "user123"
	urls := []string{
		"https://example1.com",
		"https://example2.com",
	}

	for _, url := range urls {
		_, err := svc.ShortenURL(ctx, url, userID)
		require.NoError(t, err)
	}

	userURLs, err := svc.GetURLsByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, userURLs, 2)

	cachedURLs, err := svc.GetURLsByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, cachedURLs, 2)
	assert.Equal(t, userURLs, cachedURLs)

	emptyURLs, err := svc.GetURLsByUserID(ctx, "nonexistent")
	require.NoError(t, err)
	require.Empty(t, emptyURLs)
}

// TestDeleteURLs тестирует удаление URL пользователя.
func TestDeleteURLs(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	userID := "user123"
	urls := []string{
		"https://example1.com",
		"https://example2.com",
		"https://example3.com",
	}

	for _, url := range urls {
		_, err := svc.ShortenURL(ctx, url, userID)
		require.NoError(t, err)
	}

	_, err := svc.GetURLsByUserID(ctx, userID)
	require.NoError(t, err)

	shortIDs := []string{"short2"}
	err = svc.DeleteURLs(ctx, shortIDs, userID)
	require.NoError(t, err)

	_, found := svc.Get(ctx, "short2")
	assert.False(t, found)

	userURLs, err := svc.GetURLsByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, userURLs, 2)

	_, found = svc.Get(ctx, "short1")
	assert.True(t, found)
	_, found = svc.Get(ctx, "short3")
	assert.True(t, found)
}

// TestPing тестирует проверку соединения с хранилищем.
func TestPing(t *testing.T) {
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	err := svc.Ping(ctx)
	assert.NoError(t, err)
}

// Example_httpHandlers демонстрирует использование HTTP хендлеров с сервисом сокращения URL.
// Пример показывает, как можно реализовать обработчики для сокращения URL через текстовый
// и JSON интерфейсы, а также получения оригинального URL по короткому идентификатору.
func Example_httpHandlers() {
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Пример HTTP хендлера для сокращения URL
	shortenHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Ошибка чтения тела запроса: %v", err)
			return
		}
		defer func() {
			if closeErr := r.Body.Close(); closeErr != nil {
				log.Printf("Ошибка закрытия тела запроса: %v", closeErr)
			}
		}()

		url := string(body)
		if url == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		userID := "user123" // В реальном приложении это может быть из cookie или JWT

		result, err := svc.ShortenURL(r.Context(), url, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Ошибка при сокращении URL: %v", err)
			return
		}

		statusCode := http.StatusCreated
		if !result.IsNew {
			statusCode = http.StatusConflict
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		if _, err := w.Write([]byte(result.ShortURL)); err != nil {
			log.Printf("Ошибка записи ответа: %v", err)
			return
		}
	}

	// Пример HTTP хендлера для получения оригинального URL
	getHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		path := r.URL.Path
		shortID := strings.TrimPrefix(path, "/")

		originalURL, found := svc.Get(r.Context(), shortID)
		if !found {
			w.WriteHeader(http.StatusGone)
			return
		}

		w.Header().Set("Location", originalURL)
		w.WriteHeader(http.StatusTemporaryRedirect)
	}

	// Пример HTTP хендлера для API сокращения URL
	apiShortenHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		var request struct {
			URL string `json:"url"`
		}

		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			log.Printf("Ошибка декодирования JSON: %v", err)
			return
		}
		defer func() {
			if closeErr := r.Body.Close(); closeErr != nil {
				log.Printf("Ошибка закрытия тела запроса: %v", closeErr)
			}
		}()

		if request.URL == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		userID := "user123" // В реальном приложении это может быть из cookie или JWT

		result, err := svc.ShortenURL(r.Context(), request.URL, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			log.Printf("Ошибка при сокращении URL: %v", err)
			return
		}

		statusCode := http.StatusCreated
		if !result.IsNew {
			statusCode = http.StatusConflict
		}

		response := struct {
			Result string `json:"result"`
		}{
			Result: result.ShortURL,
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Ошибка кодирования ответа: %v", err)
			return
		}
	}

	// Примеры использования HTTP хендлеров

	// 1. Сокращение URL через текстовый интерфейс
	reqBody := "https://example.com/very/long/path"
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	shortenHandler(w, req)

	resp := w.Result()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка чтения ответа: %v", err)
		return
	}
	if err := resp.Body.Close(); err != nil {
		log.Printf("Ошибка закрытия тела ответа: %v", err)
		return
	}
	fmt.Printf("Статус код: %d\n", resp.StatusCode)
	fmt.Printf("Короткий URL: %s\n", body)

	// 2. Получение оригинального URL
	shortID := "short1"
	req = httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w = httptest.NewRecorder()

	getHandler(w, req)

	resp = w.Result()
	if err := resp.Body.Close(); err != nil {
		log.Printf("Ошибка закрытия тела ответа: %v", err)
		return
	}
	fmt.Printf("Статус код: %d\n", resp.StatusCode)
	fmt.Printf("Location: %s\n", resp.Header.Get("Location"))

	// 3. Сокращение URL через JSON API
	reqJSON := `{"url": "https://another-example.com"}`
	req = httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	apiShortenHandler(w, req)

	resp = w.Result()
	body, err = io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Ошибка чтения ответа: %v", err)
		return
	}
	if err := resp.Body.Close(); err != nil {
		log.Printf("Ошибка закрытия тела ответа: %v", err)
		return
	}
	fmt.Printf("Статус код API: %d\n", resp.StatusCode)
	fmt.Printf("Ответ API: %s\n", body)

	// Output:
	// Статус код: 201
	// Короткий URL: http://example.com/short1
	// Статус код: 307
	// Location: https://example.com/very/long/path
	// Статус код API: 201
	// Ответ API: {"result":"http://example.com/short2"}
}
