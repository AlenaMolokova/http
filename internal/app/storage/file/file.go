package file

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sync"
	"strconv"
	
	"github.com/AlenaMolokova/http/internal/app/models"
	"github.com/sirupsen/logrus"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
	UserID      string `json:"user_id"`
}

type FileStorage struct {
	filename string
	urls     map[string]URLRecord
	mu       sync.RWMutex
	nextID   int
}

func NewFileStorage(filename string) (*FileStorage, error) {
	storage := &FileStorage{
		filename: filename,
		urls:     make(map[string]URLRecord),
		nextID:   1,
	}

	if err := storage.loadFromFile(); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("не удалось загрузить данные из файла: %v", err)
		}
		logrus.Info("Файл хранилища не найден, будет создан новый")
	}

	return storage, nil
}

func (s *FileStorage) Save(shortID, originalURL, userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortID] = URLRecord{
		ShortURL:    shortID,
		OriginalURL: originalURL,
		UserID:      userID,
	}

	record := URLRecord{
		UUID:        fmt.Sprintf("%d", s.nextID),
		ShortURL:    shortID,
		OriginalURL: originalURL,
		UserID:      userID,
	}
	s.nextID++

	file, err := os.OpenFile(s.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("не удалось открыть файл: %v", err)
	}
	defer file.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("не удалось сериализовать запись: %v", err)
	}

	if _, err := file.Write(append(data, '\n')); err != nil {
		return fmt.Errorf("не удалось записать в файл: %v", err)
	}

	return nil
}

func (s *FileStorage) Get(shortID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	record, ok := s.urls[shortID]
	logrus.WithFields(logrus.Fields{
		"shortID": shortID,
		"url":     record.OriginalURL,
		"found":   ok,
	}).Info("Storage lookup")
	
	if !ok {
		return "", false
	}
	
	return record.OriginalURL, true
}

func (s *FileStorage) Ping() error {
	return errors.New("file storage does not support database connection check")
}

func (s *FileStorage) loadFromFile() error {
	file, err := os.Open(s.filename)
	if err != nil {
		return err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	highestID := 0

	for scanner.Scan() {
		line := scanner.Text()
		var record URLRecord
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			return fmt.Errorf("ошибка при десериализации строки: %v", err)
		}

		s.urls[record.ShortURL] = record

		id, err := strconv.Atoi(record.UUID)
		if err == nil && id > highestID {
			highestID = id
		}
		
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка при чтении файла: %v", err)
	}

	s.nextID = highestID + 1
	logrus.WithField("count", len(s.urls)).Info("Загружены URL из файла")
	return nil
}

func (s *FileStorage) SaveBatch(items map[string]string, userID string) error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    file, err := os.OpenFile(s.filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
    if err != nil {
        return fmt.Errorf("не удалось открыть файл: %v", err)
    }
    defer file.Close()
    
    for shortID, originalURL := range items {
        s.urls[shortID] = URLRecord{
            ShortURL:    shortID,
            OriginalURL: originalURL,
            UserID:      userID,
		}
        
        record := URLRecord{
            UUID:        fmt.Sprintf("%d", s.nextID),
            ShortURL:    shortID,
            OriginalURL: originalURL,
			UserID:      userID,
        }
        s.nextID++
        
        data, err := json.Marshal(record)
        if err != nil {
            return fmt.Errorf("не удалось сериализовать запись: %v", err)
        }
        
        if _, err := file.Write(append(data, '\n')); err != nil {
            return fmt.Errorf("не удалось записать в файл: %v", err)
        }
    }
    
    return nil
}

func (s *FileStorage) FindByOriginalURL(originalURL string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for shortID, record  :=range s.urls{
		if record.OriginalURL == originalURL{
			return shortID, nil
		}
	}

	return "", errors.New("url not found")
}

func (s *FileStorage) GetURLsByUserID(userID string) ([]models.UserURL, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var result []models.UserURL
	
	for _, record := range s.urls {
		if record.UserID == userID {
			result = append(result, models.UserURL{
				ShortURL:    record.ShortURL,
				OriginalURL: record.OriginalURL,
			})
		}
	}

	return result, nil
}