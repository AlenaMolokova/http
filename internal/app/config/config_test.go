// Package config_test содержит тесты для пакета config.
package config_test

import (
	"os"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/stretchr/testify/assert"
)

// TestNewConfig проверяет, что конфигурация корректно загружается из переменных окружения и флагов.
// Важно: flag.Parse() вызывается только один раз, поэтому объединяем проверки в один тест.
func TestNewConfig(t *testing.T) {
	// Устанавливаем переменные окружения
	_ = os.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")
	_ = os.Setenv("BASE_URL", "https://short.en")
	_ = os.Setenv("FILE_STORAGE_PATH", "/tmp/test.json")
	_ = os.Setenv("DATABASE_DSN", "user:pass@/dbname")

	// Заменяем os.Args, чтобы не было флагов
	oldArgs := os.Args
	os.Args = []string{"test"}

	// Вызываем конфигурацию
	cfg := config.NewConfig()

	// Возвращаем os.Args обратно
	os.Args = oldArgs

	// Проверяем значения
	assert.Equal(t, "127.0.0.1:9090", cfg.ServerAddress)
	assert.Equal(t, "https://short.en", cfg.BaseURL)
	assert.Equal(t, "/tmp/test.json", cfg.FileStoragePath)
	assert.Equal(t, "user:pass@/dbname", cfg.DatabaseDSN)

	// Очищаем переменные окружения
	_ = os.Unsetenv("SERVER_ADDRESS")
	_ = os.Unsetenv("BASE_URL")
	_ = os.Unsetenv("FILE_STORAGE_PATH")
	_ = os.Unsetenv("DATABASE_DSN")
}
