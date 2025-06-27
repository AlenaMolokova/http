// Package app содержит точку входа для инициализации приложения и его компонентов.
package app

import (
	"context"
	"net/http"
	"time"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/router"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage"
	"github.com/sirupsen/logrus"
)

// App представляет собой основную структуру приложения,
// содержащую обработчик URL, сервисный слой и хранилище.
type App struct {
	Handler http.Handler
	Service *service.Service
	Storage *storage.Storage
}

// GenerateTestLoad генерирует тестовую нагрузку, создавая указанное
// количество сокращенных URL для тестового пользователя.
//
// Параметры:
//   - count: количество URL для генерации
//
// Метод также выполняет операции чтения, чтобы проверить
// корректность сохранения и получения данных.
func (a *App) GenerateTestLoad(count int) {
	ctx := context.Background()
	userID := "test-user"

	logrus.Info("Generating test load: ", count, " URLs")

	// Создаем временный генератор для создания тестовых URL
	testGenerator := generator.New(4)

	for i := 0; i < count; i++ {
		originalURL := "https://example.com/" + time.Now().String() + "/" + testGenerator.Generate()
		_, err := a.Service.ShortenURL(ctx, originalURL, userID)
		if err != nil {
			logrus.WithError(err).Warn("Failed to shorten URL during test load")
		}

		if i%100 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	urls, err := a.Service.GetURLsByUserID(ctx, userID)
	if err != nil {
		logrus.WithError(err).Warn("Failed to get user URLs during test load")
	} else {
		logrus.Info("Retrieved ", len(urls), " URLs for test user")
	}

	if len(urls) > 0 {
		for i := 0; i < min(10, len(urls)); i++ {
			shortID := urls[i].ShortURL
			if len(shortID) > 8 {
				shortID = shortID[len(shortID)-8:]
				_, found := a.Service.Get(ctx, shortID)
				if !found {
					logrus.Warn("URL not found during test load: ", shortID)
				}
			}
		}
	}
}

// Close завершает работу приложения и закрывает все ресурсы.
// Обеспечивает корректное сохранение данных при завершении работы.
//
// Возвращает:
//   - ошибку, если не удалось корректно закрыть ресурсы
func (a *App) Close() error {
	if a.Storage != nil {
		return a.Storage.Close()
	}
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// NewApp создает и инициализирует новый экземпляр приложения.
//
// Параметры:
//   - cfg: конфигурация приложения
//
// Возвращает:
//   - указатель на новый экземпляр App
//   - ошибку, если произошла проблема при инициализации
//
// Функция настраивает все необходимые компоненты приложения,
// включая хранилище URL, генератор коротких идентификаторов,
// сервисный слой и обработчики запросов с роутером.
func NewApp(cfg *config.Config) (*App, error) {
	// Инициализируем хранилище
	urlStorage, err := storage.NewStorage(cfg.DatabaseDSN, cfg.FileStoragePath)
	if err != nil {
		return nil, err
	}

	// Создаем генератор коротких идентификаторов
	// Используем конкретную реализацию, которая будет передана в сервис как интерфейс
	urlGenerator := generator.New(8)

	// Инициализируем сервисный слой
	// Сервис получает конкретную реализацию, но работает через свой интерфейс IDGenerator
	urlService := service.NewService(
		urlStorage.AsURLSaver(),
		urlStorage.AsURLBatchSaver(),
		urlStorage.AsURLGetter(),
		urlStorage.AsURLFetcher(),
		urlStorage.AsURLDeleter(),
		urlStorage.AsPinger(),
		urlGenerator, // *generator.SimpleGenerator автоматически реализует service.IDGenerator
		cfg.BaseURL,
	)

	// Создаем роутер
	urlRouter := router.NewRouter(
		urlService, // URLShortener
		urlService, // BatchURLShortener
		urlService, // URLGetter
		urlService, // URLFetcher
		urlService, // URLDeleter
		urlService, // Pinger
		cfg.BaseURL,
	)

	return &App{
		Handler: urlRouter,
		Service: urlService,
		Storage: urlStorage,
	}, nil
}
