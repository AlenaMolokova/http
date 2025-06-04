// Package file содержит тесты для файлового хранилища URL.
package file

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// waitForFileOperation ждёт завершения асинхронных операций с файлом
func waitForFileOperation() {
	time.Sleep(50 * time.Millisecond)
}

// TestNewFileStorage тестирует создание нового файлового хранилища.
// Проверяет корректную инициализацию, загрузку данных из файла и обработку ошибок.
func TestNewFileStorage(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, filePath, storage.filePath)
	assert.NotNil(t, storage.urls)

	testData := []models.UserURL{
		{ShortURL: "abc123", OriginalURL: "https://example.com", UserID: "user1", IsDeleted: false},
		{ShortURL: "def456", OriginalURL: "https://test.com", UserID: "user2", IsDeleted: true},
	}
	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)
	err = os.WriteFile(filePath, jsonData, 0644)
	require.NoError(t, err)

	storage, err = NewFileStorage(filePath)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Len(t, storage.urls, 2)
	assert.Equal(t, "https://example.com", storage.urls["abc123"].OriginalURL)
	assert.Equal(t, "https://test.com", storage.urls["def456"].OriginalURL)
	assert.True(t, storage.urls["def456"].IsDeleted)

	invalidPath := filepath.Join(tmpDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte("invalid json"), 0644)
	require.NoError(t, err)
	_, err = NewFileStorage(invalidPath)
	assert.Error(t, err)
}

// TestFileStorage_Save тестирует сохранение URL в файловое хранилище.
// Проверяет, что данные корректно сохраняются в память и записываются в файл.
func TestFileStorage_Save(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	assert.Equal(t, "https://example.com", storage.urls["abc123"].OriginalURL)
	assert.Equal(t, "user1", storage.urls["abc123"].UserID)
	assert.False(t, storage.urls["abc123"].IsDeleted)

	// Ждём завершения асинхронного сохранения
	waitForFileOperation()

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var urls []models.UserURL
	err = json.Unmarshal(data, &urls)
	require.NoError(t, err)

	assert.Len(t, urls, 1)
	assert.Equal(t, "abc123", urls[0].ShortURL)
	assert.Equal(t, "https://example.com", urls[0].OriginalURL)
	assert.Equal(t, "user1", urls[0].UserID)
}

// TestFileStorage_JSONSerialization тестирует сериализацию и десериализацию UserURL.
// Проверяет, что поле IsDeleted с тегом omitempty корректно обрабатывается.
func TestFileStorage_JSONSerialization(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	// Ждём завершения асинхронного сохранения
	waitForFileOperation()

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var urls []models.UserURL
	err = json.Unmarshal(data, &urls)
	require.NoError(t, err)

	assert.Len(t, urls, 1)
	assert.False(t, urls[0].IsDeleted)

	// Проверяем, что IsDeleted не включается в JSON, если false
	var rawJSON []map[string]interface{}
	err = json.Unmarshal(data, &rawJSON)
	require.NoError(t, err)
	assert.NotContains(t, rawJSON[0], "is_deleted")
}

// TestFileStorage_FindByOriginalURL тестирует поиск короткого URL по оригинальному URL.
// Проверяет корректность поиска для существующих, несуществующих и удалённых URL.
func TestFileStorage_FindByOriginalURL(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	// Ждём завершения асинхронного сохранения
	waitForFileOperation()

	shortID, err := storage.FindByOriginalURL(ctx, "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "abc123", shortID)

	shortID, err = storage.FindByOriginalURL(ctx, "https://nonexistent.com")
	require.NoError(t, err)
	assert.Empty(t, shortID)

	storage.mu.Lock()
	url := storage.urls["abc123"]
	url.IsDeleted = true
	storage.urls["abc123"] = url
	storage.isDirty = true
	storage.mu.Unlock()

	err = storage.saveToFile()
	require.NoError(t, err)

	shortID, err = storage.FindByOriginalURL(ctx, "https://example.com")
	require.NoError(t, err)
	assert.Empty(t, shortID)
}

// TestFileStorage_SaveBatch тестирует пакетное сохранение URL.
// Проверяет, что несколько URL корректно сохраняются в память и записываются в файл.
func TestFileStorage_SaveBatch(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	batch := map[string]string{
		"abc123": "https://example.com",
		"def456": "https://test.com",
	}

	err = storage.SaveBatch(ctx, batch, "user1")
	require.NoError(t, err)

	assert.Len(t, storage.urls, 2)
	assert.Equal(t, "https://example.com", storage.urls["abc123"].OriginalURL)
	assert.Equal(t, "https://test.com", storage.urls["def456"].OriginalURL)

	// Ждём завершения асинхронного сохранения
	waitForFileOperation()

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var urls []models.UserURL
	err = json.Unmarshal(data, &urls)
	require.NoError(t, err)

	assert.Len(t, urls, 2)
}

// TestFileStorage_Get тестирует получение оригинального URL по короткому идентификатору.
// Проверяет случаи существующих, несуществующих и удалённых URL.
func TestFileStorage_Get(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "def456", "https://test.com", "user1")
	require.NoError(t, err)

	// Ждём завершения асинхронного сохранения
	waitForFileOperation()

	storage.mu.Lock()
	url := storage.urls["def456"]
	url.IsDeleted = true
	storage.urls["def456"] = url
	storage.isDirty = true
	storage.mu.Unlock()

	err = storage.saveToFile()
	require.NoError(t, err)

	originalURL, exists := storage.Get(ctx, "abc123")
	assert.True(t, exists)
	assert.Equal(t, "https://example.com", originalURL)

	originalURL, exists = storage.Get(ctx, "nonexistent")
	assert.False(t, exists)
	assert.Empty(t, originalURL)

	originalURL, exists = storage.Get(ctx, "def456")
	assert.False(t, exists)
	assert.Empty(t, originalURL)
}

// TestFileStorage_GetURLsByUserID тестирует получение всех URL пользователя.
// Проверяет фильтрацию по пользователю и исключение удалённых URL.
func TestFileStorage_GetURLsByUserID(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "def456", "https://test.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "ghi789", "https://other.com", "user2")
	require.NoError(t, err)

	// Ждём завершения асинхронного сохранения
	waitForFileOperation()

	storage.mu.Lock()
	url := storage.urls["def456"]
	url.IsDeleted = true
	storage.urls["def456"] = url
	storage.isDirty = true
	storage.mu.Unlock()

	err = storage.saveToFile()
	require.NoError(t, err)

	urls, err := storage.GetURLsByUserID(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "https://example.com", urls[0].OriginalURL)

	urls, err = storage.GetURLsByUserID(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, urls)
}

// TestFileStorage_DeleteURLs тестирует удаление URL по коротким идентификаторам.
// Проверяет, что только URL указанного пользователя помечаются как удалённые.
func TestFileStorage_DeleteURLs(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "def456", "https://test.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "ghi789", "https://other.com", "user2")
	require.NoError(t, err)

	// Ждём завершения асинхронного сохранения после создания
	waitForFileOperation()

	err = storage.DeleteURLs(ctx, []string{"abc123", "ghi789"}, "user1")
	require.NoError(t, err)

	// Ждём завершения асинхронного сохранения после удаления
	waitForFileOperation()

	assert.True(t, storage.urls["abc123"].IsDeleted)
	assert.False(t, storage.urls["def456"].IsDeleted)
	assert.False(t, storage.urls["ghi789"].IsDeleted)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var urls []models.UserURL
	err = json.Unmarshal(data, &urls)
	require.NoError(t, err)

	for _, url := range urls {
		if url.ShortURL == "abc123" {
			assert.True(t, url.IsDeleted)
		}
		if url.ShortURL == "def456" {
			assert.False(t, url.IsDeleted)
		}
		if url.ShortURL == "ghi789" {
			assert.False(t, url.IsDeleted)
		}
	}
}

// TestFileStorage_Ping тестирует проверку соединения с хранилищем.
// Проверяет, что файловая реализация возвращает ошибку, так как не поддерживает подключение к базе.
func TestFileStorage_Ping(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	err = storage.Ping(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file storage does not support database connection check")
}

// TestFileStorage_saveToFile тестирует внутреннюю функцию сохранения данных в файл.
// Проверяет корректность записи данных и обработку ошибок при недоступном пути.
func TestFileStorage_saveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	storage.urls["abc123"] = models.UserURL{
		ShortURL:    "abc123",
		OriginalURL: "https://example.com",
		UserID:      "user1",
		IsDeleted:   false,
	}
	storage.urls["def456"] = models.UserURL{
		ShortURL:    "def456",
		OriginalURL: "https://test.com",
		UserID:      "user2",
		IsDeleted:   true,
	}

	err = storage.saveToFile()
	require.NoError(t, err)

	assert.False(t, storage.isDirty)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var urls []models.UserURL
	err = json.Unmarshal(data, &urls)
	require.NoError(t, err)

	assert.Len(t, urls, 2)

	// Тест ошибки записи
	storage.filePath = "/nonexistent/urls.json"
	err = storage.saveToFile()
	assert.Error(t, err)
}

// TestFileStorage_scheduleSave тестирует планирование сохранения данных в файл.
// Проверяет, что сохранение происходит только при наличии изменений и учитывает контекст.
func TestFileStorage_scheduleSave(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	// Без изменений файл не создаётся
	storage.scheduleSave(context.Background())
	_, err = os.Stat(filePath)
	assert.Error(t, err)

	// С изменениями файл создаётся
	storage.mu.Lock()
	storage.isDirty = true
	storage.urls["abc123"] = models.UserURL{
		ShortURL:    "abc123",
		OriginalURL: "https://example.com",
		UserID:      "user1",
	}
	storage.mu.Unlock()

	storage.scheduleSave(context.Background())
	time.Sleep(100 * time.Millisecond)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var urls []models.UserURL
	err = json.Unmarshal(data, &urls)
	require.NoError(t, err)

	assert.Len(t, urls, 1)
	assert.Equal(t, "https://example.com", urls[0].OriginalURL)

	// Тест отмены контекста
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	logOutput := &bytes.Buffer{}
	logrus.SetOutput(logOutput)

	storage.mu.Lock()
	storage.isDirty = true
	storage.mu.Unlock()

	storage.scheduleSave(ctx)
	time.Sleep(100 * time.Millisecond)

	assert.Contains(t, logOutput.String(), "Save operation cancelled")
}
