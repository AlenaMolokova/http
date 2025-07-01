// Package file содержит тесты для файлового хранилища URL.
package file

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewFileStorage проверяет создание нового экземпляра хранилища и загрузку данных.
func TestNewFileStorage(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	// Пустое хранилище
	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)
	assert.NotNil(t, storage)
	assert.Equal(t, filePath, storage.filePath)
	assert.NotNil(t, storage.urls)

	// Инициализация с данными
	testData := []models.UserURL{
		{ShortURL: "abc123", OriginalURL: "https://example.com", UserID: "user1"},
		{ShortURL: "def456", OriginalURL: "https://test.com", UserID: "user2", IsDeleted: true},
	}
	jsonData, err := json.Marshal(testData)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filePath, jsonData, 0644))

	storage, err = NewFileStorage(filePath)
	require.NoError(t, err)
	assert.Len(t, storage.urls, 2)
	assert.Equal(t, "https://example.com", storage.urls["abc123"].OriginalURL)
	assert.True(t, storage.urls["def456"].IsDeleted)

	// Повреждённый JSON
	badPath := filepath.Join(tmpDir, "invalid.json")
	require.NoError(t, os.WriteFile(badPath, []byte("%%%"), 0644))
	_, err = NewFileStorage(badPath)
	assert.Error(t, err)
}

// TestFileStorage_Save проверяет корректность записи и сериализации URL.
func TestFileStorage_Save(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "nested", "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, storage.Save(ctx, "abc123", "https://example.com", "user1"))

	time.Sleep(100 * time.Millisecond)
	assert.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var urls []models.UserURL
	require.NoError(t, json.Unmarshal(data, &urls))
	assert.Len(t, urls, 1)
	assert.Equal(t, "abc123", urls[0].ShortURL)
}

// TestFileStorage_FindByOriginalURL проверяет поиск по оригинальному URL.
func TestFileStorage_FindByOriginalURL(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, storage.Save(ctx, "abc123", "https://example.com", "user1"))

	shortID, err := storage.FindByOriginalURL(ctx, "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "abc123", shortID)

	shortID, err = storage.FindByOriginalURL(ctx, "https://not-found.com")
	require.NoError(t, err)
	assert.Empty(t, shortID)

	storage.urls["abc123"] = models.UserURL{
		ShortURL:    "abc123",
		OriginalURL: "https://example.com",
		UserID:      "user1",
		IsDeleted:   true,
	}
	shortID, err = storage.FindByOriginalURL(ctx, "https://example.com")
	require.NoError(t, err)
	assert.Empty(t, shortID)
}

// TestFileStorage_SaveBatch проверяет сохранение пакета URL.
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

	require.NoError(t, storage.SaveBatch(ctx, batch, "user1"))
	assert.Len(t, storage.urls, 2)

	time.Sleep(100 * time.Millisecond)
	assert.FileExists(t, filePath)
}

// TestFileStorage_Get проверяет извлечение URL по shortID.
func TestFileStorage_Get(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()
	_ = storage.Save(ctx, "abc123", "https://example.com", "user1")
	_ = storage.Save(ctx, "def456", "https://test.com", "user1")
	storage.urls["def456"] = models.UserURL{ShortURL: "def456", OriginalURL: "https://test.com", UserID: "user1", IsDeleted: true}

	url, exists := storage.Get(ctx, "abc123")
	assert.True(t, exists)
	assert.Equal(t, "https://example.com", url)

	url, exists = storage.Get(ctx, "def456")
	assert.False(t, exists)
	assert.Empty(t, url)

	url, exists = storage.Get(ctx, "unknown")
	assert.False(t, exists)
	assert.Empty(t, url)
}

// TestFileStorage_GetURLsByUserID проверяет выборку всех URL по пользователю.
func TestFileStorage_GetURLsByUserID(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()
	_ = storage.Save(ctx, "a", "https://a.com", "user1")
	_ = storage.Save(ctx, "b", "https://b.com", "user1")
	_ = storage.Save(ctx, "c", "https://c.com", "user2")
	storage.urls["b"] = models.UserURL{ShortURL: "b", OriginalURL: "https://b.com", UserID: "user1", IsDeleted: true}

	urls, err := storage.GetURLsByUserID(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "https://a.com", urls[0].OriginalURL)

	urls, err = storage.GetURLsByUserID(ctx, "ghost")
	require.NoError(t, err)
	assert.Empty(t, urls)
}

// TestFileStorage_DeleteURLs проверяет логическое удаление URL.
func TestFileStorage_DeleteURLs(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()
	_ = storage.Save(ctx, "a", "https://a.com", "user1")
	_ = storage.Save(ctx, "b", "https://b.com", "user1")
	_ = storage.Save(ctx, "c", "https://c.com", "user2")

	err = storage.DeleteURLs(ctx, []string{"a", "c"}, "user1")
	require.NoError(t, err)

	assert.True(t, storage.urls["a"].IsDeleted)
	assert.False(t, storage.urls["b"].IsDeleted)
	assert.False(t, storage.urls["c"].IsDeleted)
}

// TestFileStorage_Ping проверяет поведение Ping().
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

// TestFileStorage_saveToFile проверяет прямое сохранение и создание директорий.
func TestFileStorage_saveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "deep", "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	storage.urls["abc"] = models.UserURL{ShortURL: "abc", OriginalURL: "https://abc.com", UserID: "user1"}

	err = storage.saveToFile()
	require.NoError(t, err)
	assert.FileExists(t, filePath)
}

// TestFileStorage_scheduleSave проверяет, что save не выполняется, если isDirty = false.
func TestFileStorage_scheduleSave(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	storage.scheduleSave()
	assert.NoFileExists(t, filePath)

	storage.isDirty = true
	storage.scheduleSave()
	assert.FileExists(t, filePath)
}

// TestFileStorage_GetStats проверяет получение статистики хранилища.
func TestFileStorage_GetStats(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, storage.Save(ctx, "abc123", "https://example.com", "user1"))
	require.NoError(t, storage.Save(ctx, "def456", "https://test.com", "user1"))
	require.NoError(t, storage.Save(ctx, "ghi789", "https://other.com", "user2"))
	storage.urls["def456"] = models.UserURL{
		ShortURL:    "def456",
		OriginalURL: "https://test.com",
		UserID:      "user1",
		IsDeleted:   true,
	}

	stats, err := storage.GetStats(ctx)
	require.NoError(t, err)
	assert.Equal(t, 2, stats.URLs, "Expected 2 non-deleted URLs")
	assert.Equal(t, 2, stats.Users, "Expected 2 unique users")
}
