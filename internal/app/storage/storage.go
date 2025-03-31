package storage

import (
	"github.com/AlenaMolokova/http/internal/app/service" 
	"github.com/AlenaMolokova/http/internal/app/storage/database"
	"github.com/AlenaMolokova/http/internal/app/storage/file"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/sirupsen/logrus"
)

type Storage struct {
	Saver      service.URLSaver
	BatchSaver service.URLBatchSaver
	Getter     service.URLGetter
	Fetcher    service.URLFetcher
	Deleter    service.URLDeleter
	Pinger     service.Pinger
}

func NewStorage(databaseDSN, fileStoragePath string) (*Storage, error) {
	var impl interface{}
	if databaseDSN != "" {
		dbStorage, err := database.NewPostgresStorage(databaseDSN)
		if err == nil {
			logrus.Info("Используется хранилище PostgreSQL")
			impl = dbStorage
		} else {
			logrus.WithError(err).Warn("Не удалось использовать PostgreSQL")
		}
	}

	if impl == nil && fileStoragePath != "" {
		fileStorage, err := file.NewFileStorage(fileStoragePath)
		if err == nil {
			logrus.WithField("file", fileStoragePath).Info("Используется файловое хранилище")
			impl = fileStorage
		} else {
			logrus.WithError(err).Warn("Не удалось использовать файловое хранилище")
		}
	}

	if impl == nil {
		logrus.Info("Используется хранилище в памяти")
		impl = memory.NewMemoryStorage()
	}

	return &Storage{
		Saver:      impl.(service.URLSaver),
		BatchSaver: impl.(service.URLBatchSaver),
		Getter:     impl.(service.URLGetter),
		Fetcher:    impl.(service.URLFetcher),
		Deleter:    impl.(service.URLDeleter),
		Pinger:     impl.(service.Pinger),
	}, nil
}