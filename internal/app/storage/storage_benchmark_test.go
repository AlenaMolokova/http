package storage

import (
	"context"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/storage/database"
	"github.com/AlenaMolokova/http/internal/app/storage/file"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
)

// BenchmarkMemoryStorageSave тестирует производительность сохранения URL в память.
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

// BenchmarkMemoryStorageGet тестирует производительность получения URL из памяти.
func BenchmarkMemoryStorageGet(b *testing.B) {
	s := memory.NewMemoryStorage()
	ctx := context.Background()
	if err := s.Save(ctx, "shortID", "https://example.com", "user123"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(ctx, "shortID")
	}
}

// BenchmarkFileStorageSave тестирует производительность сохранения URL в файл.
func BenchmarkFileStorageSave(b *testing.B) {
	s, err := file.NewFileStorage("test_urls.json")
	if err != nil {
		b.Fatalf("Ошибка создания файлового хранилища: %v", err)
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

// BenchmarkFileStorageGet тестирует производительность получения URL из файла.
func BenchmarkFileStorageGet(b *testing.B) {
	s, err := file.NewFileStorage("test_urls.json")
	if err != nil {
		b.Fatalf("Ошибка создания файлового хранилища: %v", err)
	}
	ctx := context.Background()
	if err := s.Save(ctx, "shortID", "https://example.com", "user123"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(ctx, "shortID")
	}
}

// BenchmarkDatabaseStorageSave тестирует производительность сохранения URL в базу данных.
func BenchmarkDatabaseStorageSave(b *testing.B) {
	s, err := database.NewPostgresStorage("postgres://user:password@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		b.Skip("База данных недоступна")
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

// BenchmarkDatabaseStorageGet тестирует производительность получения URL из базы данных.
func BenchmarkDatabaseStorageGet(b *testing.B) {
	s, err := database.NewPostgresStorage("postgres://user:password@localhost:5432/dbname?sslmode=disable")
	if err != nil {
		b.Skip("База данных недоступна")
	}
	ctx := context.Background()
	if err := s.Save(ctx, "shortID", "https://example.com", "user123"); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		s.Get(ctx, "shortID")
	}
}
