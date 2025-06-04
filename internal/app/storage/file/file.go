// Package file реализует файловое хранилище для сокращённых URL.
package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
)

// FileStorage представляет хранилище URL-адресов в файловой системе.
// Данные сохраняются в JSON-формате в указанном файле и поддерживаются
// в памяти для быстрого доступа. Поддерживает конкурентный доступ через
// механизмы синхронизации.
type FileStorage struct {
	filePath  string
	urls      map[string]models.UserURL
	mu        sync.RWMutex
	isDirty   bool
	flushLock sync.Mutex
}

// NewFileStorage создаёт и инициализирует новое файловое хранилище URL-адресов.
// Если указанный файл существует, данные загружаются из него.
// Если файл не существует, создаётся пустое хранилище.
//
// Параметры:
//   - filePath: путь к файлу для хранения данных
//
// Возвращает:
//   - *FileStorage: указатель на инициализированное хранилище
//   - error: ошибка, если не удалось открыть или десериализовать файл
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		filePath: filePath,
		urls:     make(map[string]models.UserURL),
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return fs, nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logrus.WithError(closeErr).Errorf("Failed to close file %s", filePath)
		}
	}()

	decoder := json.NewDecoder(file)
	var entries []models.UserURL
	if err := decoder.Decode(&entries); err != nil {
		return nil, err
	}

	for _, entry := range entries {
		fs.urls[entry.ShortURL] = entry
	}

	return fs, nil
}

// Save сохраняет новый URL-адрес в хранилище.
// Сохранение в файл происходит асинхронно.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращённый идентификатор URL
//   - originalURL: оригинальный URL-адрес
//   - userID: идентификатор пользователя, который создал сокращение
//
// Возвращает:
//   - error: nil, так как ошибки игнорируются в текущей реализации
func (fs *FileStorage) Save(ctx context.Context, shortID, originalURL, userID string) error {
	fs.mu.Lock()
	fs.urls[shortID] = models.UserURL{
		ShortURL:    shortID,
		OriginalURL: originalURL,
		UserID:      userID,
		IsDeleted:   false,
	}
	fs.isDirty = true
	fs.mu.Unlock()

	go fs.scheduleSave(ctx)
	return nil
}

// FindByOriginalURL ищет сокращённый идентификатор по оригинальному URL-адресу.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - originalURL: оригинальный URL-адрес для поиска
//
// Возвращает:
//   - string: сокращённый идентификатор, если URL найден
//   - error: nil, так как ошибки игнорируются в текущей реализации
func (fs *FileStorage) FindByOriginalURL(ctx context.Context, originalURL string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	for shortID, url := range fs.urls {
		if url.OriginalURL == originalURL && !url.IsDeleted {
			return shortID, nil
		}
	}
	return "", nil
}

// SaveBatch сохраняет пакет URL-адресов в хранилище.
// Сохранение в файл происходит асинхронно.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - items: карта, где ключ - сокращённый идентификатор, значение - оригинальный URL
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - error: nil, так как ошибки игнорируются в текущей реализации
func (fs *FileStorage) SaveBatch(ctx context.Context, items map[string]string, userID string) error {
	fs.mu.Lock()
	for shortID, originalURL := range items {
		fs.urls[shortID] = models.UserURL{
			ShortURL:    shortID,
			OriginalURL: originalURL,
			UserID:      userID,
			IsDeleted:   false,
		}
	}
	fs.isDirty = true
	fs.mu.Unlock()

	go fs.scheduleSave(ctx)
	return nil
}

// Get возвращает оригинальный URL-адрес по сокращённому идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращённый идентификатор URL
//
// Возвращает:
//   - string: оригинальный URL-адрес, если сокращение найдено и не удалено
//   - bool: true, если сокращение найдено и не удалено; иначе false
func (fs *FileStorage) Get(ctx context.Context, shortID string) (string, bool) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	url, exists := fs.urls[shortID]
	if !exists || url.IsDeleted {
		return "", false
	}
	return url.OriginalURL, true
}

// GetURLsByUserID возвращает все неудалённые URL-адреса, созданные указанным пользователем.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - []models.UserURL: список структур UserURL, содержащих сокращённые и оригинальные URL-адреса
//   - error: nil, так как ошибки игнорируются в текущей реализации
func (fs *FileStorage) GetURLsByUserID(ctx context.Context, userID string) ([]models.UserURL, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	result := make([]models.UserURL, 0, 10)
	for _, url := range fs.urls {
		if url.UserID == userID && !url.IsDeleted {
			result = append(result, url)
		}
	}
	return result, nil
}

// DeleteURLs помечает указанные URL-адреса как удалённые.
// Фактическое обновление файла происходит асинхронно.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortIDs: список сокращённых идентификаторов для удаления
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - error: nil, так как ошибки игнорируются в текущей реализации
func (fs *FileStorage) DeleteURLs(ctx context.Context, shortIDs []string, userID string) error {
	fs.mu.Lock()
	for _, shortID := range shortIDs {
		if url, exists := fs.urls[shortID]; exists && url.UserID == userID {
			url.IsDeleted = true
			fs.urls[shortID] = url
		}
	}
	fs.isDirty = true
	fs.mu.Unlock()

	go fs.scheduleSave(ctx)
	return nil
}

// Ping проверяет доступность хранилища.
// Поскольку это файловое хранилище, метод возвращает ошибку, указывающую на неподдерживаемую операцию.
//
// Параметры:
//   - ctx: контекст выполнения операции
//
// Возвращает:
//   - error: сообщение о неподдерживаемой проверке соединения
func (fs *FileStorage) Ping(ctx context.Context) error {
	return errors.New("file storage does not support database connection check")
}

// scheduleSave инициирует сохранение данных в файл, если есть изменения.
// Сохранение выполняется с блокировкой для предотвращения одновременных записей.
//
// Параметры:
//   - ctx: контекст для отслеживания отмены операции
func (fs *FileStorage) scheduleSave(ctx context.Context) {
	fs.flushLock.Lock()
	defer fs.flushLock.Unlock()

	select {
	case <-ctx.Done():
		logrus.Warn("Save operation cancelled due to context cancellation")
		return
	default:
	}

	fs.mu.RLock()
	dirty := fs.isDirty
	fs.mu.RUnlock()

	if !dirty {
		return
	}

	if err := fs.saveToFile(); err != nil {
		logrus.WithError(err).Errorf("Failed to save file storage to %s", fs.filePath)
	}
}

// saveToFile сохраняет данные хранилища в файл в формате JSON.
// Использует временный файл для атомарной записи.
//
// Возвращает:
//   - error: ошибка при создании, записи или переименовании файла
func (fs *FileStorage) saveToFile() error {
	tmpFilePath := fs.filePath + ".tmp"
	file, err := os.Create(tmpFilePath)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")

	fs.mu.RLock()
	entries := make([]models.UserURL, 0, len(fs.urls))
	for _, url := range fs.urls {
		entries = append(entries, url)
	}
	fs.mu.RUnlock()

	if err := encoder.Encode(entries); err != nil {
		_ = writer.Flush()
		_ = file.Close()
		_ = os.Remove(tmpFilePath)
		return err
	}

	if err := writer.Flush(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpFilePath)
		return err
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tmpFilePath)
		return err
	}

	if err := os.Rename(tmpFilePath, fs.filePath); err != nil {
		_ = os.Remove(tmpFilePath)
		return err
	}

	fs.mu.Lock()
	fs.isDirty = false
	fs.mu.Unlock()

	return nil
}
