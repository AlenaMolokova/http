// Package file предоставляет реализацию хранилища сокращённых URL в файловой системе.
// Хранилище использует JSON-файл для персистентного хранения данных и поддерживает
// конкурентный доступ через механизмы синхронизации.
package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
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
		return nil, err
	}
	defer func() {
		if closeErr := file.Close(); closeErr != nil {
			logrus.WithError(closeErr).Error("Failed to close file")
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
//   - shortID: сокращенный идентификатор URL
//   - originalURL: оригинальный URL-адрес
//   - userID: идентификатор пользователя, который создал сокращение
//
// Возвращает:
//   - ошибку, если не удалось сохранить URL (в текущей реализации всегда nil)
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

	go fs.scheduleSave()
	return nil
}

// FindByOriginalURL ищет сокращенный идентификатор по оригинальному URL-адресу.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - originalURL: оригинальный URL-адрес для поиска
//
// Возвращает:
//   - сокращенный идентификатор, если URL найден
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
// Сохранение в файл происходит асинхронно.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - items: карта, где ключ - сокращенный идентификатор, значение - оригинальный URL
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - ошибку, если не удалось сохранить пакет URL-адресов (в текущей реализации всегда nil)
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

	go fs.scheduleSave()
	return nil
}

// Get возвращает оригинальный URL-адрес по сокращенному идентификатору.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращенный идентификатор URL
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

// GetURLsByUserID возвращает все неудаленные URL-адреса, созданные указанным пользователем.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - userID: идентификатор пользователя
//
// Возвращает:
//   - список структур UserURL, содержащих сокращенные и оригинальные URL-адреса
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

// DeleteURLs помечает указанные URL-адреса как удаленные.
// Фактическое удаление из файла происходит асинхронно.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortIDs: список сокращенных идентификаторов для удаления
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - ошибку, если не удалось пометить URL-адреса как удаленные (в текущей реализации всегда nil)
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

	go fs.scheduleSave()
	return nil
}

// GetStats возвращает статистику сервиса: количество сокращенных URL и пользователей.
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
func (fs *FileStorage) scheduleSave() {
	fs.flushLock.Lock()
	defer fs.flushLock.Unlock()

	fs.mu.RLock()
	dirty := fs.isDirty
	fs.mu.RUnlock()

	if !dirty {
		return
	}

	if err := fs.saveToFile(); err != nil {
		logrus.WithError(err).Error("Failed to save file storage")
	}
}

// saveToFile сохраняет текущее состояние хранилища в JSON-файл.
// Использует временный файл и безопасную замену основного.
// Убеждается, что директория для файла существует.
func (fs *FileStorage) saveToFile() error {
	tmpFile := fs.filePath + ".tmp"

	// Создаём директорию, если она не существует
	if err := os.MkdirAll(filepath.Dir(tmpFile), 0755); err != nil {
		logrus.WithError(err).Errorf("Failed to create directory for %s", tmpFile)
		return err
	}

	file, err := os.Create(tmpFile)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to create temp file %s", tmpFile)
		return err
	}

	writer := bufio.NewWriter(file)

	fs.mu.RLock()
	entries := make([]models.UserURL, 0, len(fs.urls))
	for _, url := range fs.urls {
		entries = append(entries, url)
	}
	fs.mu.RUnlock()

	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(entries); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpFile)
		logrus.WithError(err).Error("Failed to encode JSON")
		return err
	}

	if err := writer.Flush(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpFile)
		logrus.WithError(err).Error("Failed to flush writer")
		return err
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tmpFile)
		logrus.WithError(err).Error("Failed to close file")
		return err
	}

	// Проверяем существование временного файла перед переименованием
	if _, err := os.Stat(tmpFile); os.IsNotExist(err) {
		logrus.Errorf("Temporary file %s does not exist", tmpFile)
		return fmt.Errorf("temporary file %s does not exist", tmpFile)
	}

	if err := os.Rename(tmpFile, fs.filePath); err != nil {
		if removeErr := os.Remove(tmpFile); removeErr != nil && !os.IsNotExist(removeErr) {
			logrus.WithError(removeErr).Error("Failed to remove temp file after rename error")
		}
		logrus.WithError(err).Errorf("Failed to rename %s to %s", tmpFile, fs.filePath)
		return err
	}

	fs.mu.Lock()
	fs.isDirty = false
	fs.mu.Unlock()

	return nil
}
