// Package main реализует профилировщик для анализа производительности приложения.
// Поддерживает создание профилей кучи, CPU и выделений памяти, а также генерацию тестовой нагрузки.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"

	"github.com/AlenaMolokova/http/internal/app"
	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/sirupsen/logrus"
)

// benchmarkLoad генерирует тестовую нагрузку для приложения.
// Выполняет указанное количество операций сокращения URL для анализа производительности.
// Параметры:
//   - appInstance: экземпляр приложения
//   - count: количество URL для генерации
func benchmarkLoad(appInstance *app.App, count int) {
	appInstance.GenerateTestLoad(count)
}

// main запускает профилировщик приложения.
// Выполняет функцию run и обрабатывает ошибки, выводя их в stderr.
// Не использует os.Exit напрямую, полагаясь на естественное завершение программы.
func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "Ошибка профилирования: %v\n", err)
		return // Используем return вместо os.Exit
	}
}

// run выполняет основную логику профилировщика.
// Настраивает профилирование (CPU, куча, выделения), инициализирует приложение и генерирует тестовую нагрузку.
// Возвращает ошибку, если что-то пошло не так.
func run() error {
	var memProfileName string
	var cpuProfileName string
	var testLoad int
	var profileMode string
	var allocsProfileName string

	flag.StringVar(&allocsProfileName, "allocsprofile", "", "имя файла для профиля выделений (опционально)")
	flag.StringVar(&memProfileName, "profile", "base.pprof", "имя файла профиля кучи (по умолчанию: base.pprof)")
	flag.StringVar(&cpuProfileName, "cpuprofile", "", "имя файла профиля CPU (опционально)")
	flag.IntVar(&testLoad, "load", 1000, "количество URL для тестовой нагрузки")
	flag.StringVar(&profileMode, "mode", "base", "режим профилирования: 'base' или 'result'")
	flag.Parse()

	logrus.SetFormatter(&logrus.JSONFormatter{})
	logrus.SetLevel(logrus.InfoLevel)

	if err := os.MkdirAll("profiles", os.ModePerm); err != nil {
		return fmt.Errorf("не удалось создать директорию profiles: %w", err)
	}

	if profileMode == "result" {
		memProfileName = "result.pprof"
	} else {
		memProfileName = "base.pprof"
	}

	memProfilePath := filepath.Join("profiles", memProfileName)

	logrus.WithFields(logrus.Fields{
		"mode":      profileMode,
		"profile":   memProfileName,
		"test_load": testLoad,
		"heap_path": memProfilePath,
		"cpu_path":  cpuProfileName,
	}).Info("Запуск профилирования")

	var cpuProfileFile *os.File
	if cpuProfileName != "" {
		cpuPath := filepath.Join("profiles", cpuProfileName)
		f, err := os.Create(cpuPath)
		if err != nil {
			return fmt.Errorf("не удалось создать профиль CPU: %w", err)
		}
		defer func() {
			if closeErr := f.Close(); closeErr != nil {
				logrus.WithError(closeErr).Error("Не удалось закрыть файл профиля CPU")
			}
		}()
		if err := pprof.StartCPUProfile(f); err != nil {
			return fmt.Errorf("не удалось запустить профиль CPU: %w", err)
		}
		logrus.WithField("file", cpuPath).Info("Профилирование CPU начато")
		cpuProfileFile = f
	}

	cfg := config.NewConfig()
	logrus.WithField("config", cfg).Info("Конфигурация загружена")

	appInstance, err := app.NewApp(cfg)
	if err != nil {
		if cpuProfileFile != nil {
			pprof.StopCPUProfile()
		}
		return fmt.Errorf("не удалось инициализировать приложение: %w", err)
	}
	logrus.Info("Приложение инициализировано")

	if testLoad > 0 {
		logrus.Infof("Генерация тестовой нагрузки: %d URL", testLoad)
		benchmarkLoad(appInstance, testLoad)
	}

	if cpuProfileFile != nil {
		pprof.StopCPUProfile()
		logrus.WithField("file", cpuProfileName).Info("Профилирование CPU остановлено")
	}

	f, err := os.Create(memProfilePath)
	if err != nil {
		return fmt.Errorf("не удалось создать профиль памяти: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			logrus.WithError(closeErr).Error("Не удалось закрыть файл профиля памяти")
		}
	}()

	runtime.GC()
	if err := pprof.WriteHeapProfile(f); err != nil {
		return fmt.Errorf("не удалось записать профиль памяти: %w", err)
	}
	logrus.Infof("Профиль кучи записан в %s", memProfilePath)

	if allocsProfileName != "" {
		allocsPath := filepath.Join("profiles", allocsProfileName)
		allocsFile, err := os.Create(allocsPath)
		if err != nil {
			return fmt.Errorf("не удалось создать профиль выделений: %w", err)
		}
		defer func() {
			if closeErr := allocsFile.Close(); closeErr != nil {
				logrus.WithError(closeErr).Error("Не удалось закрыть файл профиля выделений")
			}
		}()

		if err := pprof.Lookup("allocs").WriteTo(allocsFile, 0); err != nil {
			return fmt.Errorf("не удалось записать профиль выделений: %w", err)
		}
		logrus.Infof("Профиль выделений записан в %s", allocsPath)
	}

	if profileMode == "base" {
		logrus.Info("==========================================")
		logrus.Info("ЭТАП 1 ЗАВЕРШЕН: Создан базовый профиль")
		logrus.Info("Для анализа: go tool pprof -http=:8080 profiles/base.pprof")
		logrus.Info("Затем выполните: go run cmd/profiler/main.go -mode=result")
		logrus.Info("==========================================")
	} else {
		logrus.Info("==========================================")
		logrus.Info("ЭТАП 3 ЗАВЕРШЕН: Создан результирующий профиль")
		logrus.Info("Сравните профили: go tool pprof -http=:8080 -diff_base=profiles/base.pprof profiles/result.pprof")
		logrus.Info("==========================================")
	}

	return nil
}
