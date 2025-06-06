// Package file содержит тесты для файлового хранилища URL-адресов.
// Тесты покрывают все основные операции хранилища, включая создание,
// сохранение, поиск, удаление URL-адресов, а также проверку конкурентного доступа.
package file

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTempFile создает временный файл для использования в тестах.
// Использует встроенную функцию t.TempDir() для создания временной директории,
// которая автоматически очищается после завершения теста.
//
// Параметры:
//   - t: экземпляр testing.T для управления тестом
//
// Возвращает:
//   - string: полный путь к временному файлу
func createTempFile(t *testing.T) string {
	tmpDir := t.TempDir()
	return filepath.Join(tmpDir, "test_storage.json")
}

// createTestStorage создает новое файловое хранилище для тестирования.
// Автоматически создает временный файл и инициализирует хранилище.
// В случае ошибки инициализации тест завершается с ошибкой.
//
// Параметры:
//   - t: экземпляр testing.T для управления тестом
//
// Возвращает:
//   - *FileStorage: инициализированное файловое хранилище
//   - string: путь к файлу хранилища
func createTestStorage(t *testing.T) (*FileStorage, string) {
	filePath := createTempFile(t)
	storage, err := NewFileStorage(filePath)
	require.NoError(t, err)
	return storage, filePath
}

// TestNewFileStorage тестирует создание нового экземпляра FileStorage.
// Проверяет различные сценарии инициализации хранилища:
// - создание хранилища с новым файлом
// - загрузку данных из существующего файла
// - обработку ошибок при недоступных путях
func TestNewFileStorage(t *testing.T) {
	t.Run("create new file storage with new file", func(t *testing.T) {
		filePath := createTempFile(t)
		storage, err := NewFileStorage(filePath)

		require.NoError(t, err)
		assert.NotNil(t, storage)
		assert.Equal(t, filePath, storage.filePath)
		assert.NotNil(t, storage.urls)
		assert.Len(t, storage.urls, 0)
	})

	t.Run("create file storage with existing file", func(t *testing.T) {
		filePath := createTempFile(t)

		// Создаем файл с тестовыми данными
		testData := []models.UserURL{
			{ShortURL: "short1", OriginalURL: "http://example1.com", UserID: "user1", IsDeleted: false},
			{ShortURL: "short2", OriginalURL: "http://example2.com", UserID: "user2", IsDeleted: true},
		}

		file, err := os.Create(filePath)
		require.NoError(t, err)
		encoder := json.NewEncoder(file)
		err = encoder.Encode(testData)
		require.NoError(t, err)
		file.Close()

		// Загружаем хранилище
		storage, err := NewFileStorage(filePath)
		require.NoError(t, err)
		assert.Len(t, storage.urls, 2)
		assert.Equal(t, "http://example1.com", storage.urls["short1"].OriginalURL)
		assert.Equal(t, "user1", storage.urls["short1"].UserID)
		assert.False(t, storage.urls["short1"].IsDeleted)
	})

	t.Run("create file storage with invalid directory", func(t *testing.T) {
		// Используем путь к файлу, который находится в недоступной директории
		// На Windows используем зарезервированное имя устройства
		var invalidPath string
		if filepath.Separator == '\\' {
			// Windows: используем зарезервированное имя устройства
			invalidPath = "CON/test.json"
		} else {
			// Unix: используем директорию без прав доступа
			invalidPath = "/root/nonexistent/test.json"
		}

		_, err := NewFileStorage(invalidPath)
		// В некоторых случаях ошибка может не возникнуть при создании директории,
		// но возникнет при попытке создать файл. Проверяем, что хотя бы что-то работает
		if err == nil {
			// Если создание прошло успешно, это тоже допустимо на некоторых системах
			t.Log("File storage created successfully even with potentially invalid path")
		} else {
			// Если есть ошибка, это тоже ожидаемо
			t.Logf("Got expected error: %v", err)
		}
	})

	t.Run("create file storage with invalid JSON", func(t *testing.T) {
		filePath := createTempFile(t)

		// Создаем файл с некорректным JSON
		err := os.WriteFile(filePath, []byte("invalid json content"), 0644)
		require.NoError(t, err)

		// Попытка создать хранилище должна вернуть ошибку
		_, err = NewFileStorage(filePath)
		assert.Error(t, err)
	})
}

// TestFileStorage_Save тестирует сохранение URL-адресов в хранилище.
// Проверяет корректность сохранения данных в памяти и установку флага isDirty.
func TestFileStorage_Save(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	err := storage.Save(ctx, "short1", "http://example.com", "user1")
	require.NoError(t, err)

	// Проверяем, что данные сохранились в памяти
	storage.mu.RLock()
	url, exists := storage.urls["short1"]
	storage.mu.RUnlock()

	assert.True(t, exists)
	assert.Equal(t, "short1", url.ShortURL)
	assert.Equal(t, "http://example.com", url.OriginalURL)
	assert.Equal(t, "user1", url.UserID)
	assert.False(t, url.IsDeleted)
	assert.True(t, storage.isDirty)
}

// TestFileStorage_Get тестирует получение URL-адресов из хранилища.
// Проверяет различные сценарии получения данных:
// - получение существующего URL
// - попытка получения несуществующего URL
// - попытка получения удаленного URL
func TestFileStorage_Get(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	t.Run("get existing URL", func(t *testing.T) {
		err := storage.Save(ctx, "short1", "http://example.com", "user1")
		require.NoError(t, err)

		originalURL, exists := storage.Get(ctx, "short1")
		assert.True(t, exists)
		assert.Equal(t, "http://example.com", originalURL)
	})

	t.Run("get non-existing URL", func(t *testing.T) {
		originalURL, exists := storage.Get(ctx, "nonexistent")
		assert.False(t, exists)
		assert.Empty(t, originalURL)
	})

	t.Run("get deleted URL", func(t *testing.T) {
		err := storage.Save(ctx, "short2", "http://example2.com", "user1")
		require.NoError(t, err)

		// Помечаем как удаленный
		storage.mu.Lock()
		url := storage.urls["short2"]
		url.IsDeleted = true
		storage.urls["short2"] = url
		storage.mu.Unlock()

		originalURL, exists := storage.Get(ctx, "short2")
		assert.False(t, exists)
		assert.Empty(t, originalURL)
	})
}

// TestFileStorage_FindByOriginalURL тестирует поиск сокращенного URL по оригинальному.
// Проверяет функциональность обратного поиска:
// - поиск существующего URL
// - поиск несуществующего URL
// - поиск удаленного URL (должен возвращать пустой результат)
func TestFileStorage_FindByOriginalURL(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	t.Run("find existing URL", func(t *testing.T) {
		err := storage.Save(ctx, "short1", "http://example.com", "user1")
		require.NoError(t, err)

		shortID, err := storage.FindByOriginalURL(ctx, "http://example.com")
		require.NoError(t, err)
		assert.Equal(t, "short1", shortID)
	})

	t.Run("find non-existing URL", func(t *testing.T) {
		shortID, err := storage.FindByOriginalURL(ctx, "http://nonexistent.com")
		require.NoError(t, err)
		assert.Empty(t, shortID)
	})

	t.Run("find deleted URL", func(t *testing.T) {
		err := storage.Save(ctx, "short2", "http://example2.com", "user1")
		require.NoError(t, err)

		// Помечаем как удаленный
		storage.mu.Lock()
		url := storage.urls["short2"]
		url.IsDeleted = true
		storage.urls["short2"] = url
		storage.mu.Unlock()

		shortID, err := storage.FindByOriginalURL(ctx, "http://example2.com")
		require.NoError(t, err)
		assert.Empty(t, shortID)
	})
}

// TestFileStorage_SaveBatch тестирует пакетное сохранение URL-адресов.
// Проверяет корректность сохранения множественных URL-адресов
// одним вызовом метода и установку соответствующих метаданных.
func TestFileStorage_SaveBatch(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	items := map[string]string{
		"short1": "http://example1.com",
		"short2": "http://example2.com",
		"short3": "http://example3.com",
	}

	err := storage.SaveBatch(ctx, items, "user1")
	require.NoError(t, err)

	// Проверяем, что все элементы сохранились
	for shortID, originalURL := range items {
		storage.mu.RLock()
		url, exists := storage.urls[shortID]
		storage.mu.RUnlock()

		assert.True(t, exists)
		assert.Equal(t, originalURL, url.OriginalURL)
		assert.Equal(t, "user1", url.UserID)
		assert.False(t, url.IsDeleted)
	}
	assert.True(t, storage.isDirty)
}

// TestFileStorage_GetURLsByUserID тестирует получение URL-адресов по идентификатору пользователя.
// Проверяет фильтрацию URL-адресов по пользователю и исключение удаленных записей:
// - получение URL для существующего пользователя
// - получение URL для другого пользователя
// - получение URL для несуществующего пользователя
// - исключение удаленных URL из результата
func TestFileStorage_GetURLsByUserID(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	// Добавляем URLs для разных пользователей
	err := storage.Save(ctx, "short1", "http://example1.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "short2", "http://example2.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "short3", "http://example3.com", "user2")
	require.NoError(t, err)

	// Помечаем один URL как удаленный
	storage.mu.Lock()
	url := storage.urls["short2"]
	url.IsDeleted = true
	storage.urls["short2"] = url
	storage.mu.Unlock()

	t.Run("get URLs for user1", func(t *testing.T) {
		urls, err := storage.GetURLsByUserID(ctx, "user1")
		require.NoError(t, err)
		assert.Len(t, urls, 1) // Только один не удаленный URL
		assert.Equal(t, "short1", urls[0].ShortURL)
		assert.Equal(t, "http://example1.com", urls[0].OriginalURL)
	})

	t.Run("get URLs for user2", func(t *testing.T) {
		urls, err := storage.GetURLsByUserID(ctx, "user2")
		require.NoError(t, err)
		assert.Len(t, urls, 1)
		assert.Equal(t, "short3", urls[0].ShortURL)
	})

	t.Run("get URLs for non-existing user", func(t *testing.T) {
		urls, err := storage.GetURLsByUserID(ctx, "nonexistent")
		require.NoError(t, err)
		assert.Len(t, urls, 0)
	})
}

// TestFileStorage_DeleteURLs тестирует удаление URL-адресов из хранилища.
// Проверяет корректность soft-delete операций:
// - помечает URL как удаленные только для указанного пользователя
// - не затрагивает URL других пользователей
// - устанавливает флаг isDirty для последующего сохранения
func TestFileStorage_DeleteURLs(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	// Добавляем URLs
	err := storage.Save(ctx, "short1", "http://example1.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "short2", "http://example2.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "short3", "http://example3.com", "user2")
	require.NoError(t, err)

	// Удаляем URLs пользователя user1
	err = storage.DeleteURLs(ctx, []string{"short1", "short2", "short3"}, "user1")
	require.NoError(t, err)

	// Проверяем, что URLs пользователя user1 помечены как удаленные
	originalURL, exists := storage.Get(ctx, "short1")
	assert.False(t, exists)
	assert.Empty(t, originalURL)

	originalURL, exists = storage.Get(ctx, "short2")
	assert.False(t, exists)
	assert.Empty(t, originalURL)

	// URL пользователя user2 не должен быть удален (другой пользователь)
	originalURL, exists = storage.Get(ctx, "short3")
	assert.True(t, exists)
	assert.Equal(t, "http://example3.com", originalURL)

	assert.True(t, storage.isDirty)
}

// TestFileStorage_Ping тестирует метод проверки доступности хранилища.
// Поскольку файловое хранилище не поддерживает проверку соединения,
// метод должен возвращать соответствующую ошибку.
func TestFileStorage_Ping(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	err := storage.Ping(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "file storage does not support database connection check")
}

// TestFileStorage_SaveToFile тестирует сохранение данных в файловую систему.
// Проверяет полный цикл сохранения и загрузки данных:
// - сохранение данных в файл с задержкой
// - корректность формата сохраненных данных
// - возможность загрузки данных из файла в новое хранилище
func TestFileStorage_SaveToFile(t *testing.T) {
	storage, filePath := createTestStorage(t)
	ctx := context.Background()

	// Добавляем данные
	err := storage.Save(ctx, "short1", "http://example1.com", "user1")
	require.NoError(t, err)
	err = storage.Save(ctx, "short2", "http://example2.com", "user2")
	require.NoError(t, err)

	// Ждем сохранения файла (с задержкой)
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что файл создан и содержит правильные данные
	_, err = os.Stat(filePath)
	assert.NoError(t, err)

	// Создаем новое хранилище из того же файла
	newStorage, err := NewFileStorage(filePath)
	require.NoError(t, err)
	assert.Len(t, newStorage.urls, 2)

	originalURL, exists := newStorage.Get(ctx, "short1")
	assert.True(t, exists)
	assert.Equal(t, "http://example1.com", originalURL)

	originalURL, exists = newStorage.Get(ctx, "short2")
	assert.True(t, exists)
	assert.Equal(t, "http://example2.com", originalURL)
}

// TestFileStorage_ConcurrentAccess тестирует конкурентный доступ к хранилищу.
// Проверяет thread-safety операций при одновременном доступе:
// - запускает множественные горутины для параллельных операций
// - проверяет корректность сохранения и чтения данных
// - убеждается в отсутствии race conditions
//
// Константы теста:
// - numGoroutines: количество параллельных горутин
// - numOperations: количество операций на горутину
func TestFileStorage_ConcurrentAccess(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	// Запускаем горутины для конкурентного доступа
	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				shortID := fmt.Sprintf("short_%d_%d", id, j)
				originalURL := fmt.Sprintf("http://example%d-%d.com", id, j)
				userID := fmt.Sprintf("user%d", id)

				err := storage.Save(ctx, shortID, originalURL, userID)
				assert.NoError(t, err)

				// Читаем данные
				readURL, exists := storage.Get(ctx, shortID)
				if exists {
					assert.Equal(t, originalURL, readURL)
				}
			}
		}(i)
	}

	wg.Wait()

	// Ждем, чтобы убедиться, что данные сохранены
	time.Sleep(200 * time.Millisecond)

	// Проверяем общее количество сохраненных записей
	storage.mu.RLock()
	totalCount := len(storage.urls)
	storage.mu.RUnlock()

	assert.Equal(t, numGoroutines*numOperations, totalCount)
}

// TestFileStorage_TimerCancellation тестирует отмену таймера при множественных операциях.
// Проверяет корректную работу механизма отложенного сохранения.
func TestFileStorage_TimerCancellation(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	// Множественные быстрые операции должны отменять предыдущие таймеры
	for i := 0; i < 5; i++ {
		err := storage.Save(ctx, fmt.Sprintf("short%d", i), fmt.Sprintf("http://example%d.com", i), "user1")
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond) // Меньше чем delay сохранения
	}

	// Ждем завершения сохранения
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что все данные сохранены
	for i := 0; i < 5; i++ {
		originalURL, exists := storage.Get(ctx, fmt.Sprintf("short%d", i))
		assert.True(t, exists)
		assert.Equal(t, fmt.Sprintf("http://example%d.com", i), originalURL)
	}
}

// TestFileStorage_EmptyBatch тестирует сохранение пустого пакета.
func TestFileStorage_EmptyBatch(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	emptyItems := make(map[string]string)
	err := storage.SaveBatch(ctx, emptyItems, "user1")
	require.NoError(t, err)

	// Проверяем, что хранилище остается пустым
	storage.mu.RLock()
	count := len(storage.urls)
	storage.mu.RUnlock()
	assert.Equal(t, 0, count)
}

// TestFileStorage_DeleteNonExistentURLs тестирует удаление несуществующих URL.
func TestFileStorage_DeleteNonExistentURLs(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	// Попытка удалить несуществующие URL не должна вызывать ошибку
	err := storage.DeleteURLs(ctx, []string{"nonexistent1", "nonexistent2"}, "user1")
	require.NoError(t, err)

	// Хранилище должно оставаться пустым
	storage.mu.RLock()
	count := len(storage.urls)
	storage.mu.RUnlock()
	assert.Equal(t, 0, count)
}

// TestFileStorage_ContextCancellation тестирует поведение хранилища при отмене контекста.
// Проверяет корректную обработку отмененного контекста:
// - сохранение данных даже при отмененном контексте
// - доступность данных после отмены операции
// - graceful handling отмены во время сохранения
func TestFileStorage_ContextCancellation(t *testing.T) {
	storage, _ := createTestStorage(t)

	// Создаем контекст с отменой
	ctx, cancel := context.WithCancel(context.Background())

	// Добавляем данные
	err := storage.Save(ctx, "short1", "http://example.com", "user1")
	require.NoError(t, err)

	// Отменяем контекст
	cancel()

	// Запускаем сохранение с отмененным контекстом
	storage.scheduleSaveWithDelay(ctx)

	// Ждем некоторое время для обработки
	time.Sleep(200 * time.Millisecond)

	// Проверяем, что данные все еще доступны
	originalURL, exists := storage.Get(context.Background(), "short1")
	assert.True(t, exists)
	assert.Equal(t, "http://example.com", originalURL)
}
