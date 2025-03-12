package app

import (
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage"
)

type App struct {
	Handler *handler.Handler
}

func NewApp(cfg *config.Config) (*App, error) {
	urlStorage, err := storage.NewStorage(cfg.DatabaseDSN, cfg.FileStoragePath)
	if err != nil {
		return nil, err
	}

	urlGenerator := generator.NewGenerator(8)

	urlService := service.NewURLService(urlStorage, urlGenerator, cfg.BaseURL)

	handler := handler.NewHandler(urlService)

	return &App{
		Handler: handler,
	}, nil
}
