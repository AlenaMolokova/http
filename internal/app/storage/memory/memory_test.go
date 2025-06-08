package memory

import (
	"context"
	"reflect"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/models"
)

func TestNewMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	if storage == nil {
		t.Fatal("NewMemoryStorage returned nil")
	}
	if storage.urls == nil {
		t.Error("urls map is nil")
	}
}

func TestMemoryStorage_Save(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	err := storage.Save(ctx, "abc123", "https://example.com", "user1")
	if err != nil {
		t.Fatalf("Save returned error: %v", err)
	}

	url, exists := storage.urls["abc123"]
	if !exists {
		t.Error("URL was not saved in storage")
	}
	if url.ShortURL != "abc123" {
		t.Errorf("Expected shortURL to be 'abc123', got '%s'", url.ShortURL)
	}
	if url.OriginalURL != "https://example.com" {
		t.Errorf("Expected originalURL to be 'https://example.com', got '%s'", url.OriginalURL)
	}
	if url.UserID != "user1" {
		t.Errorf("Expected userID to be 'user1', got '%s'", url.UserID)
	}
	if url.IsDeleted {
		t.Error("Expected IsDeleted to be false, got true")
	}
}

func TestMemoryStorage_FindByOriginalURL(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, "abc123", "https://example.com", "user1")
	_ = storage.Save(ctx, "def456", "https://example.org", "user1")

	shortID, err := storage.FindByOriginalURL(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("FindByOriginalURL returned error: %v", err)
	}
	if shortID != "abc123" {
		t.Errorf("Expected shortID to be 'abc123', got '%s'", shortID)
	}

	shortID, err = storage.FindByOriginalURL(ctx, "https://notfound.com")
	if err != nil {
		t.Fatalf("FindByOriginalURL returned error: %v", err)
	}
	if shortID != "" {
		t.Errorf("Expected empty shortID for non-existent URL, got '%s'", shortID)
	}

	storage.urls["def456"] = models.UserURL{
		ShortURL:    "def456",
		OriginalURL: "https://example.org",
		UserID:      "user1",
		IsDeleted:   true,
	}

	shortID, err = storage.FindByOriginalURL(ctx, "https://example.org")
	if err != nil {
		t.Fatalf("FindByOriginalURL returned error: %v", err)
	}
	if shortID != "" {
		t.Errorf("Expected empty shortID for deleted URL, got '%s'", shortID)
	}
}

func TestMemoryStorage_SaveBatch(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	batch := map[string]string{
		"abc123": "https://example.com",
		"def456": "https://example.org",
		"ghi789": "https://example.net",
	}

	err := storage.SaveBatch(ctx, batch, "user1")
	if err != nil {
		t.Fatalf("SaveBatch returned error: %v", err)
	}

	for shortID, originalURL := range batch {
		url, exists := storage.urls[shortID]
		if !exists {
			t.Errorf("URL with shortID '%s' was not saved in storage", shortID)
			continue
		}
		if url.ShortURL != shortID {
			t.Errorf("Expected shortURL to be '%s', got '%s'", shortID, url.ShortURL)
		}
		if url.OriginalURL != originalURL {
			t.Errorf("Expected originalURL to be '%s', got '%s'", originalURL, url.OriginalURL)
		}
		if url.UserID != "user1" {
			t.Errorf("Expected userID to be 'user1', got '%s'", url.UserID)
		}
		if url.IsDeleted {
			t.Error("Expected IsDeleted to be false, got true")
		}
	}
}

func TestMemoryStorage_Get(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, "abc123", "https://example.com", "user1")

	originalURL, exists := storage.Get(ctx, "abc123")
	if !exists {
		t.Fatal("Get returned exists=false for existing URL")
	}
	if originalURL != "https://example.com" {
		t.Errorf("Expected originalURL to be 'https://example.com', got '%s'", originalURL)
	}

	originalURL, exists = storage.Get(ctx, "notfound")
	if exists {
		t.Error("Get returned exists=true for non-existent URL")
	}
	if originalURL != "" {
		t.Errorf("Expected empty originalURL for non-existent URL, got '%s'", originalURL)
	}

	storage.urls["def456"] = models.UserURL{
		ShortURL:    "def456",
		OriginalURL: "https://example.org",
		UserID:      "user1",
		IsDeleted:   true,
	}

	originalURL, exists = storage.Get(ctx, "def456")
	if exists {
		t.Error("Get returned exists=true for deleted URL")
	}
	if originalURL != "" {
		t.Errorf("Expected empty originalURL for deleted URL, got '%s'", originalURL)
	}
}

func TestMemoryStorage_GetURLsByUserID(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, "abc123", "https://example.com", "user1")
	_ = storage.Save(ctx, "def456", "https://example.org", "user1")
	_ = storage.Save(ctx, "ghi789", "https://example.net", "user2")

	storage.urls["def456"] = models.UserURL{
		ShortURL:    "def456",
		OriginalURL: "https://example.org",
		UserID:      "user1",
		IsDeleted:   true,
	}

	urls, err := storage.GetURLsByUserID(ctx, "user1")
	if err != nil {
		t.Fatalf("GetURLsByUserID returned error: %v", err)
	}

	if len(urls) != 1 {
		t.Errorf("Expected 1 URL for user1, got %d", len(urls))
	}

	if urls[0].ShortURL != "abc123" {
		t.Errorf("Expected shortURL to be 'abc123', got '%s'", urls[0].ShortURL)
	}

	urls, err = storage.GetURLsByUserID(ctx, "user2")
	if err != nil {
		t.Fatalf("GetURLsByUserID returned error: %v", err)
	}

	if len(urls) != 1 {
		t.Errorf("Expected 1 URL for user2, got %d", len(urls))
	}

	if urls[0].ShortURL != "ghi789" {
		t.Errorf("Expected shortURL to be 'ghi789', got '%s'", urls[0].ShortURL)
	}

	urls, err = storage.GetURLsByUserID(ctx, "user3")
	if err != nil {
		t.Fatalf("GetURLsByUserID returned error: %v", err)
	}

	if len(urls) != 0 {
		t.Errorf("Expected 0 URLs for non-existent user, got %d", len(urls))
	}
}

func TestMemoryStorage_DeleteURLs(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, "abc123", "https://example.com", "user1")
	_ = storage.Save(ctx, "def456", "https://example.org", "user1")
	_ = storage.Save(ctx, "ghi789", "https://example.net", "user2")

	err := storage.DeleteURLs(ctx, []string{"abc123", "notfound"}, "user1")
	if err != nil {
		t.Fatalf("DeleteURLs returned error: %v", err)
	}

	if url, exists := storage.urls["abc123"]; !exists || !url.IsDeleted {
		t.Error("URL was not marked as deleted")
	}

	if url, exists := storage.urls["def456"]; !exists || url.IsDeleted {
		t.Error("URL def456 was incorrectly affected by delete operation")
	}

	err = storage.DeleteURLs(ctx, []string{"def456"}, "user2")
	if err != nil {
		t.Fatalf("DeleteURLs returned error: %v", err)
	}

	if url, exists := storage.urls["def456"]; !exists || url.IsDeleted {
		t.Error("URL def456 was incorrectly deleted by another user")
	}
}

func TestMemoryStorage_Ping(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	err := storage.Ping(ctx)
	if err == nil {
		t.Error("Ping did not return expected error")
	}
}

func TestEmptyMemoryStorage(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_, exists := storage.Get(ctx, "notfound")
	if exists {
		t.Error("Get returned exists=true for empty storage")
	}

	shortID, err := storage.FindByOriginalURL(ctx, "https://example.com")
	if err != nil {
		t.Fatalf("FindByOriginalURL returned error: %v", err)
	}
	if shortID != "" {
		t.Errorf("Expected empty shortID for empty storage, got '%s'", shortID)
	}

	urls, err := storage.GetURLsByUserID(ctx, "user1")
	if err != nil {
		t.Fatalf("GetURLsByUserID returned error: %v", err)
	}
	if len(urls) != 0 {
		t.Errorf("Expected 0 URLs for empty storage, got %d", len(urls))
	}
}

func TestMemoryStorage_Overwrite(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, "abc123", "https://example.com", "user1")

	err := storage.Save(ctx, "abc123", "https://new-example.com", "user2")
	if err != nil {
		t.Fatalf("Save returned error on overwrite: %v", err)
	}

	url, exists := storage.urls["abc123"]
	if !exists {
		t.Error("URL was not found after overwrite")
	}
	if url.OriginalURL != "https://new-example.com" {
		t.Errorf("Expected originalURL to be updated to 'https://new-example.com', got '%s'", url.OriginalURL)
	}
	if url.UserID != "user2" {
		t.Errorf("Expected userID to be updated to 'user2', got '%s'", url.UserID)
	}
}

func TestMemoryStorage_GetURLsByUserIDStructure(t *testing.T) {
	storage := NewMemoryStorage()
	ctx := context.Background()

	_ = storage.Save(ctx, "abc123", "https://example.com", "user1")

	urls, err := storage.GetURLsByUserID(ctx, "user1")
	if err != nil {
		t.Fatalf("GetURLsByUserID returned error: %v", err)
	}
	if len(urls) != 1 {
		t.Fatalf("Expected 1 URL, got %d", len(urls))
	}

	expected := models.UserURL{
		ShortURL:    "abc123",
		OriginalURL: "https://example.com",
		UserID:      "user1",
		IsDeleted:   false,
	}

	if !reflect.DeepEqual(urls[0], expected) {
		t.Errorf("Expected %+v, got %+v", expected, urls[0])
	}
}
