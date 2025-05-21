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

	inaccessiblePath := filepath.Join(tmpDir, "inaccessible.json")
	err = os.WriteFile(inaccessiblePath, []byte("invalid json"), 0644)
	require.NoError(t, err)
	err = os.Chmod(inaccessiblePath, 0000)
	require.NoError(t, err)
	_, err = NewFileStorage(inaccessiblePath)
	assert.Error(t, err)
	os.Chmod(inaccessiblePath, 0644)

	invalidPath := filepath.Join(tmpDir, "invalid.json")
	err = os.WriteFile(invalidPath, []byte("invalid json"), 0644)
	require.NoError(t, err)
	_, err = NewFileStorage(invalidPath)
	assert.Error(t, err)
}

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

	time.Sleep(100 * time.Millisecond)

	assert.FileExists(t, filePath)

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

func TestFileStorage_FindByOriginalURL(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)

	ctx := context.Background()

	err = storage.Save(ctx, "abc123", "https://example.com", "user1")
	require.NoError(t, err)

	shortID, err := storage.FindByOriginalURL(ctx, "https://example.com")
	require.NoError(t, err)
	assert.Equal(t, "abc123", shortID)

	shortID, err = storage.FindByOriginalURL(ctx, "https://nonexistent.com")
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

	time.Sleep(100 * time.Millisecond)

	assert.FileExists(t, filePath)
}

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
	storage.urls["def456"] = models.UserURL{
		ShortURL:    "def456",
		OriginalURL: "https://test.com",
		UserID:      "user1",
		IsDeleted:   true,
	}

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

	storage.urls["def456"] = models.UserURL{
		ShortURL:    "def456",
		OriginalURL: "https://test.com",
		UserID:      "user1",
		IsDeleted:   true,
	}

	urls, err := storage.GetURLsByUserID(ctx, "user1")
	require.NoError(t, err)
	assert.Len(t, urls, 1)
	assert.Equal(t, "https://example.com", urls[0].OriginalURL)

	urls, err = storage.GetURLsByUserID(ctx, "nonexistent")
	require.NoError(t, err)
	assert.Empty(t, urls)
}

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

	err = storage.DeleteURLs(ctx, []string{"abc123", "ghi789"}, "user1")
	require.NoError(t, err)

	assert.True(t, storage.urls["abc123"].IsDeleted)
	assert.False(t, storage.urls["def456"].IsDeleted)
	assert.False(t, storage.urls["ghi789"].IsDeleted)

	time.Sleep(100 * time.Millisecond)

	assert.FileExists(t, filePath)
}

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

	assert.FileExists(t, filePath)

	data, err := os.ReadFile(filePath)
	require.NoError(t, err)

	var urls []models.UserURL
	err = json.Unmarshal(data, &urls)
	require.NoError(t, err)

	assert.Len(t, urls, 2)

	storage.filePath = "/nonexistent/urls.json"
	err = storage.saveToFile()
	assert.Error(t, err)
}

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
