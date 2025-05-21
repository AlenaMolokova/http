package service_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Моки для тестирования
type mockStorage struct {
	urls     map[string]string            // shortID -> originalURL
	userURLs map[string]map[string]string // userID -> shortID -> originalURL
	deleted  map[string]bool              // shortID -> isDeleted
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		urls:     make(map[string]string),
		userURLs: make(map[string]map[string]string),
		deleted:  make(map[string]bool),
	}
}

func (m *mockStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	m.urls[shortID] = originalURL
	if _, ok := m.userURLs[userID]; !ok {
		m.userURLs[userID] = make(map[string]string)
	}
	m.userURLs[userID][shortID] = originalURL
	return nil
}

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

func (m *mockStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	for shortID, url := range m.urls {
		if url == originalURL && !m.deleted[shortID] {
			return shortID, nil
		}
	}
	return "", nil
}

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
	return result, nil
}

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

func (m *mockStorage) Ping(ctx context.Context) error {
	return nil
}

// Мок-генератор коротких идентификаторов
type mockGenerator struct {
	counter int
}

func (g *mockGenerator) Generate() string {
	g.counter++
	return fmt.Sprintf("short%d", g.counter)
}

// Примеры использования сервиса сокращения URL
func Example_shortenURL() {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Сокращаем URL
	userID := "user123"
	originalURL := "https://very-long-url.com/with/path/and?query=parameters"

	result, err := svc.ShortenURL(ctx, originalURL, userID)
	if err != nil {
		fmt.Printf("Ошибка при сокращении URL: %v\n", err)
		return
	}

	fmt.Printf("Короткий URL: %s\n", result.ShortURL)
	fmt.Printf("Это новый URL: %v\n", result.IsNew)

	// Восстанавливаем оригинальный URL по короткому идентификатору
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

func Example_shortenBatch() {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Подготавливаем пакет URL для сокращения
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

func Example_getUserURLs() {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Сокращаем несколько URL для пользователя
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

	// Получаем все URL пользователя
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

func Example_deleteURLs() {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Сокращаем несколько URL для пользователя
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

	// Удаляем один из URL
	shortIDs := []string{"short2"}
	err := svc.DeleteURLs(ctx, shortIDs, userID)
	if err != nil {
		fmt.Printf("Ошибка при удалении URL: %v\n", err)
		return
	}

	// Проверяем, что URL удален
	_, found := svc.Get(ctx, "short2")
	fmt.Printf("URL short2 доступен: %v\n", found)

	// Получаем оставшиеся URL пользователя
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

// Тестовые примеры со стандартными утверждениями Go
func TestShortenURL(t *testing.T) {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Тестируем сокращение URL
	userID := "user123"
	originalURL := "https://very-long-url.com/with/path"

	result, err := svc.ShortenURL(ctx, originalURL, userID)

	assert.NoError(t, err)
	assert.Equal(t, "http://example.com/short1", result.ShortURL)
	assert.True(t, result.IsNew)

	// Тестируем повторное сокращение того же URL
	result, err = svc.ShortenURL(ctx, originalURL, userID)

	assert.NoError(t, err)
	assert.Equal(t, "http://example.com/short1", result.ShortURL)
	assert.False(t, result.IsNew)
}

func TestShortenBatch(t *testing.T) {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Подготавливаем пакет URL для сокращения
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

	// Проверяем корреляцию между запросами и ответами
	correlationMap := make(map[string]string)
	for _, result := range results {
		correlationMap[result.CorrelationID] = result.ShortURL
	}

	assert.Contains(t, correlationMap, "id1")
	assert.Contains(t, correlationMap, "id2")
	assert.NotEqual(t, correlationMap["id1"], correlationMap["id2"])
}

func TestGetURLsByUserID(t *testing.T) {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Сокращаем несколько URL для пользователя
	userID := "user123"
	urls := []string{
		"https://example1.com",
		"https://example2.com",
	}

	for _, url := range urls {
		_, err := svc.ShortenURL(ctx, url, userID)
		require.NoError(t, err)
	}

	// Получаем URL пользователя
	userURLs, err := svc.GetURLsByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, userURLs, 2)

	// Проверяем кэширование
	// Вызываем второй раз - должно вернуться из кэша
	cachedURLs, err := svc.GetURLsByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, cachedURLs, 2)
	assert.Equal(t, userURLs, cachedURLs)

	// Проверяем, что для несуществующего пользователя возвращается пустой список
	emptyURLs, err := svc.GetURLsByUserID(ctx, "nonexistent")
	require.NoError(t, err)
	require.Empty(t, emptyURLs)
}

func TestDeleteURLs(t *testing.T) {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Сокращаем несколько URL для пользователя
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

	// Кэшируем URL пользователя
	_, err := svc.GetURLsByUserID(ctx, userID)
	require.NoError(t, err)

	// Удаляем один из URL
	shortIDs := []string{"short2"}
	err = svc.DeleteURLs(ctx, shortIDs, userID)
	require.NoError(t, err)

	// Проверяем, что URL действительно удален
	_, found := svc.Get(ctx, "short2")
	assert.False(t, found)

	// Проверяем, что кэш обновлен
	userURLs, err := svc.GetURLsByUserID(ctx, userID)
	require.NoError(t, err)
	require.Len(t, userURLs, 2)

	// Проверяем, что другие URL пользователя не затронуты
	_, found = svc.Get(ctx, "short1")
	assert.True(t, found)
	_, found = svc.Get(ctx, "short3")
	assert.True(t, found)
}

func TestPing(t *testing.T) {
	// Создаем тестовый контекст и зависимости
	ctx := context.Background()
	storage := newMockStorage()
	generator := &mockGenerator{}
	baseURL := "http://example.com"

	// Инициализируем сервис
	svc := service.NewService(storage, storage, storage, storage, storage, storage, generator, baseURL)

	// Проверяем соединение с хранилищем
	err := svc.Ping(ctx)
	assert.NoError(t, err)
}

// Example_httpHandlers демонстрирует использование HTTP хендлеров с сервисом сокращения URL.
// Пример показывает, как можно реализовать обработчики для сокращения URL через текстовый
// и JSON интерфейсы, а также получения оригинального URL по короткому идентификатору.
func Example_httpHandlers() {
	// Создаем тестовый сервер и клиент
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
			return
		}
		defer r.Body.Close()

		url := string(body)
		if url == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Извлекаем user ID из cookie или создаем новый
		userID := "user123" // В реальном приложении это может быть из cookie или JWT

		result, err := svc.ShortenURL(r.Context(), url, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		statusCode := http.StatusCreated
		if !result.IsNew {
			statusCode = http.StatusConflict
		}

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(statusCode)
		w.Write([]byte(result.ShortURL))
	}

	// Пример HTTP хендлера для получения оригинального URL
	getHandler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		// Извлекаем shortID из пути URL
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
			return
		}
		defer r.Body.Close()

		if request.URL == "" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		// Извлекаем user ID из cookie или создаем новый
		userID := "user123" // В реальном приложении это может быть из cookie или JWT

		result, err := svc.ShortenURL(r.Context(), request.URL, userID)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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
		json.NewEncoder(w).Encode(response)
	}

	// Примеры использования HTTP хендлеров

	// 1. Сокращение URL через текстовый интерфейс
	reqBody := "https://example.com/very/long/path"
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(reqBody))
	w := httptest.NewRecorder()

	shortenHandler(w, req)

	resp := w.Result()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Статус код: %d\n", resp.StatusCode)
	fmt.Printf("Короткий URL: %s\n", body)

	// 2. Получение оригинального URL
	shortID := "short1" // Предполагается, что этот ID был создан в предыдущем запросе
	req = httptest.NewRequest(http.MethodGet, "/"+shortID, nil)
	w = httptest.NewRecorder()

	getHandler(w, req)

	resp = w.Result()
	fmt.Printf("Статус код: %d\n", resp.StatusCode)
	fmt.Printf("Location: %s\n", resp.Header.Get("Location"))

	// 3. Сокращение URL через JSON API
	reqJSON := `{"url": "https://another-example.com"}`
	req = httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBufferString(reqJSON))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()

	apiShortenHandler(w, req)

	resp = w.Result()
	body, _ = io.ReadAll(resp.Body)
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
