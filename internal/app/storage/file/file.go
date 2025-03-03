package file

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"strconv"

	"github.com/sirupsen/logrus"
)

type URLRecord struct {
	UUID        string `json:"uuid"`
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

type FileStorage struct {
	filename string
	urls     map[string]string
	mu       sync.RWMutex
	nextID   int
}

func NewFileStorage(filename string) (*FileStorage, error) {
	storage := &FileStorage{
		filename: filename,
		urls:     make(map[string]string),
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

func (s *FileStorage) Save(shortID, originalURL string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortID] = originalURL

	record := URLRecord{
		UUID:        fmt.Sprintf("%d", s.nextID),
		ShortURL:    shortID,
		OriginalURL: originalURL,
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

	url, ok := s.urls[shortID]
	logrus.WithFields(logrus.Fields{
		"shortID": shortID,
		"url":     url,
		"found":   ok,
	}).Info("Storage lookup")
	return url, ok
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

		s.urls[record.ShortURL] = record.OriginalURL

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