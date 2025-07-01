// Package file предоставляет реализацию хранилища сокращённых URL в файловой системе.
// Хранилище использует JSON-файл для персистентного хранения данных и поддерживает
// конкурентный доступ через механизмы синхронизации.
package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
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
//   - указатель на FileStorage при успешной инициализации
//   - ошибку, если не удалось открыть или десериализовать файл
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
		logrus.WithError(err).Errorf("Failed to open file %s", filePath)
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
		logrus.WithError(err).Errorf("Failed to decode JSON from %s", filePath)
		return nil, err
	}

	for _, entry := range entries {
		fs.urls[entry.ShortURL] = entry
	}

	return fs, nil
}

// Save сохраняет новый URL-адрес в хранилище.
// Сохранение в файл происходит синхронно в тестах (если TEST_ENV=true) или асинхронно.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращённый идентификатор URL
//   - originalURL: оригинальный URL-адрес
//   - userID: идентификатор пользователя, который создал сокращение
//
// Возвращает:
//   - ошибку, если не удалось сохранить URL (в тестах может возвращать ошибку сохранения в файл)
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

	if os.Getenv("TEST_ENV") == "true" {
		return fs.scheduleSave()
	}
	go fs.scheduleSave()
	return nil
}

// FindByOriginalURL ищет сокращённый идентификатор по оригинальному URL-адресу.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - originalURL: оригинальный URL-адрес для поиска
//
// Возвращает:
//   - сокращённый идентификатор, если URL найден
//   - пустую строку, если URL не найден
//   - ошибку (в текущей реализации всегда nil)
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
// Сохранение в файл происходит синхронно в тестах (если TEST_ENV=true) или асинхронно.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - items: карта, где ключ — сокращённый идентификатор, значение — оригинальный URL
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - ошибку, если не удалось сохранить пакет URL-адресов (в тестах может возвращать ошибку сохранения в файл)
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

	if os.Getenv("TEST_ENV") == "true" {
		return fs.scheduleSave()
	}
	go fs.scheduleSave()
	return nil
}

// Get возвращает оригинальный URL-адрес по сокращённому идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращённый идентификатор URL
//
// Возвращает:
//   - оригинальный URL-адрес и true, если сокращение найдено и не удалено
//   - пустую строку и false, если сокращение не найдено или удалено
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
//   - список структур UserURL, содержащих сокращённые и оригинальные URL-адреса
//   - ошибку (в текущей реализации всегда nil)
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
// Сохранение в файл происходит синхронно в тестах (если TEST_ENV=true) или асинхронно.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortIDs: список сокращённых идентификаторов для удаления
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - ошибку, если не удалось пометить URL-адреса как удалённые (в тестах может возвращать ошибку сохранения в файл)
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

	if os.Getenv("TEST_ENV") == "true" {
		return fs.scheduleSave()
	}
	go fs.scheduleSave()
	return nil
}

// GetStats возвращает статистику сервиса: количество сокращённых URL и пользователей.
//
// Параметры:
//   - ctx: контекст выполнения операции
//
// Возвращает:
//   - структуру Stats с информацией о количестве URL и пользователей
//   - ошибку (в текущей реализации всегда nil)
func (fs *FileStorage) GetStats(ctx context.Context) (models.Stats, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var urlsCount int
	userIDs := make(map[string]bool)

	for _, url := range fs.urls {
		if !url.IsDeleted {
			urlsCount++
			userIDs[url.UserID] = true
		}
	}

	return models.Stats{
		URLs:  urlsCount,
		Users: len(userIDs),
	}, nil
}

// Ping проверяет доступность хранилища.
// Поскольку это файловое хранилище, метод всегда возвращает ошибку,
// указывающую на то, что проверка соединения не поддерживается.
//
// Параметры:
//   - ctx: контекст выполнения операции
//
// Возвращает:
//   - ошибку с сообщением о неподдерживаемой операции
func (fs *FileStorage) Ping(ctx context.Context) error {
	return errors.New("file storage does not support database connection check")
}

// Close завершает работу файлового хранилища и сохраняет все данные в файл.
//
// Возвращает:
//   - ошибку, если не удалось сохранить данные в файл
func (fs *FileStorage) Close() error {
	fs.flushLock.Lock()
	defer fs.flushLock.Unlock()

	fs.mu.RLock()
	dirty := fs.isDirty
	fs.mu.RUnlock()

	if dirty {
		return fs.saveToFile()
	}
	return nil
}

// scheduleSave планирует сохранение данных в файл, если данные изменены.
//
// Возвращает:
//   - ошибку, если не удалось сохранить данные в файл
func (fs *FileStorage) scheduleSave() error {
	fs.flushLock.Lock()
	defer fs.flushLock.Unlock()

	fs.mu.RLock()
	dirty := fs.isDirty
	fs.mu.RUnlock()

	if !dirty {
		return nil
	}

	if err := fs.saveToFile(); err != nil {
		logrus.WithError(err).Error("Failed to save file storage")
		return err
	}
	return nil
}

// saveToFile сохраняет текущее состояние хранилища в JSON-файл.
// Использует временный файл и безопасную замену основного.
// Убеждается, что директория для файла существует.
//
// Возвращает:
//   - ошибку, если не удалось сохранить данные в файл
func (fs *FileStorage) saveToFile() error {
	tmpFile := fs.filePath + ".tmp"

	// Создаём директорию, если она не существует
	logrus.Infof("Creating directory for temp file: %s", filepath.Dir(tmpFile))
	if err := os.MkdirAll(filepath.Dir(tmpFile), 0755); err != nil {
		logrus.WithError(err).Errorf("Failed to create directory for %s", tmpFile)
		return err
	}

	logrus.Infof("Creating temp file: %s", tmpFile)
	file, err := os.Create(tmpFile)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to create temp file %s", tmpFile)
		return err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil && !errors.Is(closeErr, os.ErrClosed) {
			logrus.WithError(closeErr).Errorf("Failed to close temp file %s", tmpFile)
		}
	}()

	writer := bufio.NewWriter(file)

	fs.mu.RLock()
	entries := make([]models.UserURL, 0, len(fs.urls))
	for _, url := range fs.urls {
		entries = append(entries, url)
	}
	fs.mu.RUnlock()

	logrus.Info("Encoding JSON to temp file")
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entries); err != nil {
		logrus.WithError(err).Error("Failed to encode JSON")
		if removeErr := os.Remove(tmpFile); removeErr != nil && !os.IsNotExist(removeErr) {
			logrus.WithError(removeErr).Errorf("Failed to remove temp file %s after encode error", tmpFile)
		}
		return err
	}

	logrus.Info("Flushing writer")
	if err := writer.Flush(); err != nil {
		logrus.WithError(err).Error("Failed to flush writer")
		if removeErr := os.Remove(tmpFile); removeErr != nil && !os.IsNotExist(removeErr) {
			logrus.WithError(removeErr).Errorf("Failed to remove temp file %s after flush error", tmpFile)
		}
		return err
	}

	logrus.Info("Closing temp file")
	if err := file.Close(); err != nil {
		logrus.WithError(err).Error("Failed to close file")
		if removeErr := os.Remove(tmpFile); removeErr != nil && !os.IsNotExist(removeErr) {
			logrus.WithError(removeErr).Errorf("Failed to remove temp file %s after close error", tmpFile)
		}
		return err
	}

	logrus.Infof("Renaming %s to %s", tmpFile, fs.filePath)
	if err := os.Rename(tmpFile, fs.filePath); err != nil {
		logrus.WithError(err).Errorf("Failed to rename %s to %s", tmpFile, fs.filePath)
		if removeErr := os.Remove(tmpFile); removeErr != nil && !os.IsNotExist(removeErr) {
			logrus.WithError(removeErr).Errorf("Failed to remove temp file %s after rename error", tmpFile)
		}
		return err
	}

	fs.mu.Lock()
	fs.isDirty = false
	fs.mu.Unlock()

	logrus.Info("Successfully saved to file")
	return nil
}
