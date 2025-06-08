package storage

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"
	"time"

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
		err := s.Save(ctx, fmt.Sprintf("shortID%d", i), "https://example.com", "user123")
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
	// Создаем уникальный файл для каждого бенчмарка
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "benchmark_urls.json")

	s, err := file.NewFileStorage(testFile)
	if err != nil {
		b.Fatalf("Ошибка создания файлового хранилища: %v", err)
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := s.Save(ctx, fmt.Sprintf("shortID%d", i), "https://example.com", "user123")
		if err != nil {
			b.Fatal(err)
		}
	}

	// Ждем завершения асинхронных записей
	b.StopTimer()
	time.Sleep(100 * time.Millisecond)
}

// BenchmarkFileStorageGet тестирует производительность получения URL из файла.
func BenchmarkFileStorageGet(b *testing.B) {
	// Создаем уникальный файл для каждого бенчмарка
	tmpDir := b.TempDir()
	testFile := filepath.Join(tmpDir, "benchmark_get_urls.json")

	s, err := file.NewFileStorage(testFile)
	if err != nil {
		b.Fatalf("Ошибка создания файлового хранилища: %v", err)
	}

	ctx := context.Background()

	// Предварительно сохраняем данные и ждем записи в файл
	if err := s.Save(ctx, "shortID", "https://example.com", "user123"); err != nil {
		b.Fatal(err)
	}

	// Ждем завершения асинхронной записи
	time.Sleep(200 * time.Millisecond)

	// Создаем новый экземпляр хранилища для чтения из файла
	// Это гарантирует, что данные действительно загружаются из файла
	s2, err := file.NewFileStorage(testFile)
	if err != nil {
		b.Fatalf("Ошибка создания файлового хранилища для чтения: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, exists := s2.Get(ctx, "shortID")
		if !exists {
			b.Fatal("URL не найден в хранилище")
		}
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
		err := s.Save(ctx, fmt.Sprintf("shortID%d", i), "https://example.com", "user123")
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
