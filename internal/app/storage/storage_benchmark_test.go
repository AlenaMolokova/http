package storage

import (
	"context"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/storage/database"
	"github.com/AlenaMolokova/http/internal/app/storage/file"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
)

func BenchmarkMemoryStorageSave(b *testing.B) {
	s := memory.NewMemoryStorage()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := s.Save(ctx, "shortID"+string(rune(i)), "https://example.com", "user123")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkMemoryStorageGet(b *testing.B) {
	s := memory.NewMemoryStorage()
	ctx := context.Background()
	s.Save(ctx, "shortID", "https://example.com", "user123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(ctx, "shortID")
	}
}

func BenchmarkFileStorageSave(b *testing.B) {
	s, err := file.NewFileStorage("test_urls.json")
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := s.Save(ctx, "shortID"+string(rune(i)), "https://example.com", "user123")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFileStorageGet(b *testing.B) {
	s, err := file.NewFileStorage("test_urls.json")
	if err != nil {
		b.Fatal(err)
	}
	ctx := context.Background()
	s.Save(ctx, "shortID", "https://example.com", "user123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(ctx, "shortID")
	}
}

func BenchmarkDatabaseStorageSave(b *testing.B) {
	s, err := database.NewPostgresStorage("postgres://user:password@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		b.Skip("Database not available")
	}
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := s.Save(ctx, "shortID"+string(rune(i)), "https://example.com", "user123")
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDatabaseStorageGet(b *testing.B) {
	s, err := database.NewPostgresStorage("postgres://user:password@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		b.Skip("Database not available")
	}
	ctx := context.Background()
	s.Save(ctx, "shortID", "https://example.com", "user123")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(ctx, "shortID")
	}
}
