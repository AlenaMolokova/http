// Package file реализует файловое хранилище для сокращённых URL.
package file

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
)

// FileStorage представляет хранилище URL-адресов в файловой системе.
// Данные сохраняются в JSON-формате в указанном файле и поддерживаются
// в памяти для быстрого доступа. Поддерживает конкурентный доступ через
// механизмы синхронизации.
type FileStorage struct {
	filePath   string
	urls       map[string]models.UserURL
	mu         sync.RWMutex
	isDirty    bool
	flushLock  sync.Mutex
	saveTimer  *time.Timer
	timerMutex sync.Mutex
}

// NewFileStorage создаёт и инициализирует новое файловое хранилище URL-адресов.
// Если указанный файл существует, данные загружаются из него.
// Если файл не существует, создаётся пустое хранилище и файл с необходимыми директориями.
//
// Параметры:
//   - filePath: путь к файлу для хранения данных
//
// Возвращает:
//   - *FileStorage: указатель на инициализированное хранилище
//   - error: ошибка, если не удалось открыть, создать или десериализовать файл
func NewFileStorage(filePath string) (*FileStorage, error) {
	fs := &FileStorage{
		filePath: filePath,
		urls:     make(map[string]models.UserURL),
	}

	// Создаём директорию для файла, если она не существует
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		logrus.WithError(err).Errorf("Failed to create directory for %s", filePath)
		return nil, err
	}

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		// Создаём файл с правильными правами доступа
		file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			logrus.WithError(err).Errorf("Failed to create file %s", filePath)
			return nil, err
		}
		if err := file.Close(); err != nil {
			logrus.WithError(err).Errorf("Failed to close file %s during creation", filePath)
		}
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
	if err := decoder.Decode(&entries); err != nil && err != io.EOF {
		logrus.WithError(err).Errorf("Failed to decode JSON from %s", filePath)
		return nil, err
	}

	for _, entry := range entries {
		fs.urls[entry.ShortURL] = entry
	}

	return fs, nil
}

// Save сохраняет новый URL-адрес в хранилище.
// Сохранение в файл происходит с небольшой задержкой для группировки операций.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortID: сокращённый идентификатор URL
//   - originalURL: оригинальный URL-адрес
//   - userID: идентификатор пользователя, который создал сокращение
//
// Возвращает:
//   - error: ошибка, если сохранение не удалось
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

	fs.scheduleSaveWithDelay(ctx)
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
//   - error: ошибка, если поиск не удался
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
// Сохранение в файл происходит с небольшой задержкой для группировки операций.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - items: карта, где ключ - сокращённый идентификатор, значение - оригинальный URL
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - error: ошибка, если сохранение не удалось
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

	fs.scheduleSaveWithDelay(ctx)
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
//   - error: ошибка, если получение не удалось
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
// Фактическое обновление файла происходит с небольшой задержкой.
//
// Параметры:
//   - ctx: контекст выполнения операции
//   - shortIDs: список сокращённых идентификаторов для удаления
//   - userID: идентификатор пользователя, которому принадлежат URL-адреса
//
// Возвращает:
//   - error: ошибка, если удаление не удалось
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

	fs.scheduleSaveWithDelay(ctx)
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

// scheduleSaveWithDelay планирует сохранение данных с задержкой для группировки операций.
// Использует таймер для отложенного выполнения сохранения.
//
// Параметры:
//   - ctx: контекст для отслеживания отмены операции
func (fs *FileStorage) scheduleSaveWithDelay(ctx context.Context) {
	fs.timerMutex.Lock()
	defer fs.timerMutex.Unlock()

	// Отменяем предыдущий таймер, если он существует
	if fs.saveTimer != nil {
		fs.saveTimer.Stop()
	}

	// Создаем новый таймер на 100ms
	fs.saveTimer = time.AfterFunc(100*time.Millisecond, func() {
		fs.performSave(ctx)
	})
}

// performSave выполняет фактическое сохранение данных в файл.
//
// Параметры:
//   - ctx: контекст для отслеживания отмены операции
func (fs *FileStorage) performSave(ctx context.Context) {
	fs.flushLock.Lock()
	defer fs.flushLock.Unlock()

	fs.mu.RLock()
	dirty := fs.isDirty
	fs.mu.RUnlock()

	if !dirty {
		return
	}

	// Проверяем, не был ли контекст отменен
	select {
	case <-ctx.Done():
		logrus.Warn("Save operation cancelled due to context cancellation, attempting save with timeout")
		// Создаем новый контекст с коротким таймаутом для завершения сохранения
		saveCtx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		if err := fs.saveToFile(); err != nil {
			logrus.WithError(err).Errorf("Failed to save file storage to %s during shutdown", fs.filePath)
			return
		}
		logrus.Infof("Successfully saved file storage to %s during shutdown", fs.filePath)

		select {
		case <-saveCtx.Done():
			logrus.Warn("Save operation timed out during shutdown")
		default:
		}
	default:
		if err := fs.saveToFile(); err != nil {
			logrus.WithError(err).Errorf("Failed to save file storage to %s", fs.filePath)
		}
	}
}

// saveToFile сохраняет данные хранилища в файл в формате JSON.
// Использует временный файл для атомарной записи.
//
// Возвращает:
//   - error: ошибка при создании, записи или переименовании файла
func (fs *FileStorage) saveToFile() error {
	// Создаём директорию для временного файла
	if err := os.MkdirAll(filepath.Dir(fs.filePath), 0755); err != nil {
		logrus.WithError(err).Errorf("Failed to create directory for %s", fs.filePath)
		return err
	}

	tmpFilePath := fs.filePath + ".tmp"
	file, err := os.OpenFile(tmpFilePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		logrus.WithError(err).Errorf("Failed to create temporary file %s", tmpFilePath)
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
		logrus.WithError(err).Errorf("Failed to encode JSON to %s", tmpFilePath)
		return err
	}

	if err := writer.Flush(); err != nil {
		_ = file.Close()
		_ = os.Remove(tmpFilePath)
		logrus.WithError(err).Errorf("Failed to flush writer for %s", tmpFilePath)
		return err
	}

	if err := file.Close(); err != nil {
		_ = os.Remove(tmpFilePath)
		logrus.WithError(err).Errorf("Failed to close file %s", tmpFilePath)
		return err
	}

	if err := os.Rename(tmpFilePath, fs.filePath); err != nil {
		_ = os.Remove(tmpFilePath)
		logrus.WithError(err).Errorf("Failed to rename %s to %s", tmpFilePath, fs.filePath)
		return err
	}

	fs.mu.Lock()
	fs.isDirty = false
	fs.mu.Unlock()

	logrus.Infof("Successfully saved file storage to %s", fs.filePath)
	return nil
}