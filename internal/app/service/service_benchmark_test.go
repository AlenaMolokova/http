package service

import (
	"context"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
)

func BenchmarkShortenURL(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	ctx := context.Background()
	originalURL := "https://example.com"
	userID := "user123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.ShortenURL(ctx, originalURL, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkShortenBatch(b *testing.B) {
	storage := memory.NewMemoryStorage()
	generator := generator.NewGenerator(8)
	s := NewService(storage, storage, storage, storage, storage, storage, generator, "http://localhost:8080")
	ctx := context.Background()
	batch := []models.BatchShortenRequest{
		{CorrelationID: "1", OriginalURL: "https://example.com/1"},
		{CorrelationID: "2", OriginalURL: "https://example.com/2"},
	}
	userID := "user123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.ShortenBatch(ctx, batch, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
