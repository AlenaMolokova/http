// Package main реализует HTTP-сервер для сервиса сокращения URL.
// Запускает сервер с маршрутами для обработки запросов сокращения и перенаправления URL.
package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/AlenaMolokova/http/internal/app"
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/AlenaMolokova/http/internal/app/router"
	"github.com/sirupsen/logrus"
)

// Глобальные переменные для информации о сборке.
// Значения устанавливаются на этапе сборки через флаги -ldflags.
var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// main запускает HTTP-сервер.
// Выполняет функцию run и обрабатывает ошибки, выводя их в stderr.
// Не использует os.Exit напрямую, полагаясь на естественное завершение программы.
func main() {
	printBuildInfo()

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка запуска сервера: %v\n", err)
		return // Используем return вместо os.Exit
	}
}

// printBuildInfo выводит информацию о сборке в stdout.
func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}

	date := buildDate
	if date == "" {
		date = "N/A"
	}

	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}

// run выполняет основную логику HTTP-сервера.
// Инициализирует приложение, настраивает маршруты и запускает сервер.
// Возвращает ошибку, если что-то пошло не так.
func run() error {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	cfg := config.NewConfig()
	logrus.WithField("config", cfg).Info("Конфигурация загружена")

	appInstance, err := app.NewApp(cfg)
	if err != nil {
		return fmt.Errorf("не удалось инициализировать приложение: %w", err)
	}
	logrus.Info("Приложение инициализировано")

	r := router.NewRouter(appInstance.Handler)

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: r.InitRoutes(),
	}
	logrus.WithFields(logrus.Fields{
		"address":  cfg.ServerAddress,
		"base_url": cfg.BaseURL,
	}).Info("Запуск сервера")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("не удалось запустить сервер: %w", err)
	}
	logrus.Info("Сервер работает")
	return nil
}
