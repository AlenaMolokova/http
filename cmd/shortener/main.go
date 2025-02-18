package main

import (
	"net/http"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/generator"
	"github.com/AlenaMolokova/http/internal/app/handler"
	"github.com/AlenaMolokova/http/internal/app/middleware"
	"github.com/AlenaMolokova/http/internal/app/router"
	"github.com/AlenaMolokova/http/internal/app/service"
	"github.com/AlenaMolokova/http/internal/app/storage/memory"
	"github.com/sirupsen/logrus"
)

func main() {

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	cfg := config.NewConfig()

	urlStorage := memory.NewMemoryStorage()
	urlGenerator := generator.NewGenerator(8)
	urlService := service.NewURLService(urlStorage, urlGenerator, cfg.BaseURL)
	urlHandler := handler.NewHandler(urlService)
	urlRouter := router.NewRouter(urlHandler)

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler:  urlRouter.InitRoutes(),
	}
	logrus.WithFields(logrus.Fields{
		"address":  cfg.ServerAddress,
		"base_url": cfg.BaseURL,
	}).Info("Starting server")

	if err := server.ListenAndServe(); err != nil {
		logrus.Fatal(err)
	}

}
