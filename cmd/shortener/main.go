// Package main реализует точку входа в приложение сокращения URL.
// Поддерживает корректное завершение работы (graceful shutdown) по сигналам SIGINT, SIGTERM, SIGQUIT.
package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/AlenaMolokova/http/internal/app"
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/sirupsen/logrus"
)

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

func main() {
	printBuildInfo()
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка запуска: %v\n", err)
		os.Exit(1)
	}
}

// printBuildInfo выводит информацию о сборке приложения.
func printBuildInfo() {
	fmt.Printf("Версия: %s\nДата сборки: %s\nКоммит: %s\n",
		getValue(buildVersion),
		getValue(buildDate),
		getValue(buildCommit))
}

// getValue возвращает значение или "N/A" если значение пустое.
func getValue(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}

// run запускает сервер приложения и обрабатывает его завершение по сигналам.
// Реализует graceful shutdown: завершает активные соединения, сохраняет данные
// и корректно закрывает ресурсы при получении сигнала SIGINT, SIGTERM или SIGQUIT.
//
// Возвращает:
//   - ошибку, если запуск или остановка сервера завершились неудачно
func run() error {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	cfg := config.NewConfig()

	appInstance, err := app.NewApp(cfg)
	if err != nil {
		return fmt.Errorf("ошибка инициализации: %w", err)
	}

	// Обеспечиваем корректное закрытие приложения
	defer func() {
		if closeErr := appInstance.Close(); closeErr != nil {
			logrus.WithError(closeErr).Error("Ошибка при закрытии приложения")
		}
	}()

	server := &http.Server{
		Addr:         cfg.ServerAddress,
		Handler:      appInstance.Handler,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if cfg.EnableHTTPS {
		server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	// Канал для graceful shutdown по сигналам
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM, syscall.SIGQUIT)

	// Канал для ошибок сервера
	serverErr := make(chan error, 1)

	// Запускаем HTTP/HTTPS сервер в фоне
	go func() {
		if cfg.EnableHTTPS {
			logrus.Infof("HTTPS сервер запущен на %s", cfg.ServerAddress)
			if err := server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && err != http.ErrServerClosed {
				serverErr <- fmt.Errorf("ошибка запуска HTTPS сервера: %w", err)
			}
		} else {
			logrus.Infof("HTTP сервер запущен на %s", cfg.ServerAddress)
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErr <- fmt.Errorf("ошибка запуска HTTP сервера: %w", err)
			}
		}
	}()

	// Ожидаем сигнал завершения или ошибку сервера
	select {
	case err := <-serverErr:
		return err
	case sig := <-stop:
		logrus.Infof("Получен сигнал остановки: %v, завершаем работу...", sig)
	}

	// Контекст с таймаутом для завершения активных соединений
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Ошибка при остановке сервера")
		return err
	}

	logrus.Info("Сервер остановлен")
	return nil
}
