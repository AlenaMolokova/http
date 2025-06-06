package main

import (
	"context"
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
	}
}

func printBuildInfo() {
	fmt.Printf("Версия: %s\nДата сборки: %s\nКоммит: %s\n",
		getValue(buildVersion),
		getValue(buildDate),
		getValue(buildCommit))
}

func getValue(v string) string {
	if v == "" {
		return "N/A"
	}
	return v
}

func run() error {
	logrus.SetFormatter(&logrus.JSONFormatter{})
	cfg := config.NewConfig()

	appInstance, err := app.NewApp(cfg)
	if err != nil {
		return fmt.Errorf("ошибка инициализации: %w", err)
	}

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: appInstance.Handler,
	}

	// Канал для graceful shutdown
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Запускаем сервер в отдельной горутине
	go func() {
		logrus.Infof("Сервер запущен на %s", cfg.ServerAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logrus.WithError(err).Fatal("Ошибка запуска сервера")
		}
	}()

	// Ожидаем сигнал остановки
	<-stop
	logrus.Info("Получен сигнал остановки, завершаем работу...")

	// Создаем контекст с таймаутом для graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Останавливаем HTTP сервер
	if err := server.Shutdown(ctx); err != nil {
		logrus.WithError(err).Error("Ошибка при остановке сервера")
	}

	// Даем время на завершение асинхронных операций сохранения
	time.Sleep(2 * time.Second)

	logrus.Info("Сервер остановлен")
	return nil
}
