// Package storage содержит тесты для различных реализаций хранилища URL.
package storage

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNewStorage тестирует создание нового хранилища с различными конфигурациями.
// Проверяет, что возвращается правильный тип хранилища (DatabaseStorage, FileStorage или MemoryStorage)
// в зависимости от параметров databaseDSN и fileStoragePath.
func TestNewStorage(t *testing.T) {
	tests := []struct {
		name            string
		databaseDSN     string
		fileStoragePath string
		wantStorageType string
	}{
		{
			name:            "PostgreSQL storage",
			databaseDSN:     "postgres://user:password@localhost:5432/testdb",
			fileStoragePath: "",
			wantStorageType: "*database.DatabaseStorage",
		},
		{
			name:            "File storage",
			databaseDSN:     "",
			fileStoragePath: "testdata/test_urls.json",
			wantStorageType: "*file.FileStorage",
		},
		{
			name:            "Memory storage",
			databaseDSN:     "",
			fileStoragePath: "",
			wantStorageType: "*memory.MemoryStorage",
		},
	}

	tempFile := "testdata/test_urls.json"
	if err := os.MkdirAll("testdata", 0755); err != nil {
		t.Fatalf("Ошибка создания директории: %v", err)
	}
	f, err := os.Create(tempFile)
	if err != nil {
		t.Fatalf("Ошибка создания файла: %v", err)
	}
	if err := f.Close(); err != nil {
		t.Fatalf("Ошибка закрытия файла: %v", err)
	}
	defer func() {
		if err := os.RemoveAll("testdata"); err != nil {
			t.Logf("Ошибка удаления директории: %v", err)
		}
	}()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "PostgreSQL storage" {
				tt.databaseDSN = "postgres://invalid:invalid@localhost:5432/nonexistent"
				tt.wantStorageType = "*file.FileStorage"
			}

			storage, err := NewStorage(tt.databaseDSN, tt.fileStoragePath)
			require.NoError(t, err)
			assert.NotNil(t, storage)

			if tt.name == "File storage" && tt.wantStorageType == "*file.FileStorage" {
				if err := os.Chmod(tt.fileStoragePath, 0000); err != nil {
					t.Fatalf("Ошибка изменения прав файла: %v", err)
				}
				storage, err = NewStorage(tt.databaseDSN, tt.fileStoragePath)
				require.NoError(t, err)
				assert.NotNil(t, storage)
				tt.wantStorageType = "*memory.MemoryStorage"
				require.NoError(t, os.Chmod(tt.fileStoragePath, 0644), "Ошибка восстановления прав файла")
			}
		})
	}
}

// TestStorageInterfaces тестирует реализацию всех интерфейсов хранилища.
// Проверяет, что хранилище поддерживает интерфейсы URLSaver, URLBatchSaver, URLGetter,
// URLFetcher, URLDeleter и Pinger.
func TestStorageInterfaces(t *testing.T) {
	storage, err := NewStorage("", "")
	require.NoError(t, err)

	assert.NotNil(t, storage.AsURLSaver())
	assert.NotNil(t, storage.AsURLBatchSaver())
	assert.NotNil(t, storage.AsURLGetter())
	assert.NotNil(t, storage.AsURLFetcher())
	assert.NotNil(t, storage.AsURLDeleter())
	assert.NotNil(t, storage.AsPinger())
}
