// Package config_test содержит тесты для пакета config.
package config_test

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/AlenaMolokova/http/internal/app/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// resetFlags сбрасывает состояние флагов для изоляции тестов.
func resetFlags() {
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)
}

// createTempJSONConfig создает временный JSON-файл конфигурации.
func createTempJSONConfig(t *testing.T, cfg interface{}) string {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "config.json")

	data, err := json.Marshal(cfg)
	require.NoError(t, err)

	err = os.WriteFile(configPath, data, 0644)
	require.NoError(t, err)

	return configPath
}

// clearEnvVars очищает все переменные окружения конфигурации.
func clearEnvVars() {
	envVars := []string{
		"SERVER_ADDRESS", "BASE_URL", "FILE_STORAGE_PATH",
		"DATABASE_DSN", "ENABLE_HTTPS", "CERT_FILE", "KEY_FILE", "CONFIG",
	}
	for _, env := range envVars {
		os.Unsetenv(env)
	}
}

// TestNewConfig_DefaultValues проверяет загрузку значений по умолчанию.
func TestNewConfig_DefaultValues(t *testing.T) {
	resetFlags()
	clearEnvVars()

	// Устанавливаем пустые аргументы
	oldArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = oldArgs }()

	cfg := config.NewConfig()

	assert.Equal(t, "localhost:8080", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
	assert.Equal(t, "urls.json", cfg.FileStoragePath)
	assert.Equal(t, "", cfg.DatabaseDSN)
	assert.False(t, cfg.EnableHTTPS)
	assert.Equal(t, "server.crt", cfg.CertFile)
	assert.Equal(t, "server.key", cfg.KeyFile)
}

// TestNewConfig_EnvironmentVariables проверяет загрузку из переменных окружения.
func TestNewConfig_EnvironmentVariables(t *testing.T) {
	resetFlags()
	clearEnvVars()

	// Устанавливаем переменные окружения
	os.Setenv("SERVER_ADDRESS", "127.0.0.1:9090")
	os.Setenv("BASE_URL", "https://short.en")
	os.Setenv("FILE_STORAGE_PATH", "/tmp/test.json")
	os.Setenv("DATABASE_DSN", "user:pass@/dbname")
	os.Setenv("ENABLE_HTTPS", "true")

	defer clearEnvVars()

	oldArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = oldArgs }()

	cfg := config.NewConfig()

	assert.Equal(t, "127.0.0.1:9090", cfg.ServerAddress)
	assert.Equal(t, "https://short.en", cfg.BaseURL)
	assert.Equal(t, "/tmp/test.json", cfg.FileStoragePath)
	assert.Equal(t, "user:pass@/dbname", cfg.DatabaseDSN)
	assert.True(t, cfg.EnableHTTPS)
}

// TestNewConfig_JSONConfiguration проверяет загрузку из JSON-файла.
func TestNewConfig_JSONConfiguration(t *testing.T) {
	resetFlags()
	clearEnvVars()

	// Создаем временный JSON-файл
	jsonConfig := map[string]interface{}{
		"server_address":    "localhost:7070",
		"base_url":          "http://localhost:7070",
		"file_storage_path": "test_urls.json",
		"database_dsn":      "test:test@/test",
		"enable_https":      true,
	}

	configPath := createTempJSONConfig(t, jsonConfig)

	oldArgs := os.Args
	os.Args = []string{"test", "-c", configPath}
	defer func() { os.Args = oldArgs }()

	cfg := config.NewConfig()

	assert.Equal(t, "localhost:7070", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:7070", cfg.BaseURL)
	assert.Equal(t, "test_urls.json", cfg.FileStoragePath)
	assert.Equal(t, "test:test@/test", cfg.DatabaseDSN)
	assert.True(t, cfg.EnableHTTPS)
}

// TestNewConfig_JSONConfigurationViaEnv проверяет загрузку JSON через переменную CONFIG.
func TestNewConfig_JSONConfigurationViaEnv(t *testing.T) {
	resetFlags()
	clearEnvVars()

	jsonConfig := map[string]interface{}{
		"server_address": "localhost:6060",
		"base_url":       "http://localhost:6060",
	}

	configPath := createTempJSONConfig(t, jsonConfig)
	os.Setenv("CONFIG", configPath)
	defer clearEnvVars()

	oldArgs := os.Args
	os.Args = []string{"test"}
	defer func() { os.Args = oldArgs }()

	cfg := config.NewConfig()

	assert.Equal(t, "localhost:6060", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:6060", cfg.BaseURL)
}

// TestNewConfig_PriorityOrder проверяет правильность приоритета конфигураций.
func TestNewConfig_PriorityOrder(t *testing.T) {
	resetFlags()
	clearEnvVars()

	// 1. Создаем JSON с одними значениями
	jsonConfig := map[string]interface{}{
		"server_address": "localhost:7070",
		"base_url":       "http://json.config",
	}
	configPath := createTempJSONConfig(t, jsonConfig)

	// 2. Устанавливаем переменные окружения с другими значениями
	os.Setenv("SERVER_ADDRESS", "localhost:8080")
	os.Setenv("BASE_URL", "http://env.config")
	defer clearEnvVars()

	// 3. Передаем флаги с третьими значениями
	oldArgs := os.Args
	os.Args = []string{"test", "-c", configPath, "-a", "localhost:9090", "-b", "http://flag.config"}
	defer func() { os.Args = oldArgs }()

	cfg := config.NewConfig()

	// Флаги должны иметь наивысший приоритет
	assert.Equal(t, "localhost:9090", cfg.ServerAddress)
	assert.Equal(t, "http://flag.config", cfg.BaseURL)
}

// TestNewConfig_PartialOverride проверяет частичное переопределение конфигурации.
func TestNewConfig_PartialOverride(t *testing.T) {
	resetFlags()
	clearEnvVars()

	// JSON определяет только часть параметров
	jsonConfig := map[string]interface{}{
		"server_address": "localhost:7070",
		"enable_https":   true,
	}
	configPath := createTempJSONConfig(t, jsonConfig)

	// Переменная окружения определяет другой параметр
	os.Setenv("BASE_URL", "http://env.example.com")
	defer clearEnvVars()

	oldArgs := os.Args
	os.Args = []string{"test", "-c", configPath}
	defer func() { os.Args = oldArgs }()

	cfg := config.NewConfig()

	// Проверяем комбинированный результат
	assert.Equal(t, "localhost:7070", cfg.ServerAddress)   // из JSON
	assert.Equal(t, "http://env.example.com", cfg.BaseURL) // из env
	assert.True(t, cfg.EnableHTTPS)                        // из JSON
	assert.Equal(t, "urls.json", cfg.FileStoragePath)      // значение по умолчанию
}

// TestNewConfig_NonExistentJSONFile проверяет обработку несуществующего JSON-файла.
func TestNewConfig_NonExistentJSONFile(t *testing.T) {
	resetFlags()
	clearEnvVars()

	oldArgs := os.Args
	os.Args = []string{"test", "-c", "nonexistent.json"}
	defer func() { os.Args = oldArgs }()

	// Не должно паниковать при отсутствии файла
	cfg := config.NewConfig()

	// Должны применяться значения по умолчанию
	assert.Equal(t, "localhost:8080", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
}

// TestNewConfig_EmptyJSONFile проверяет обработку пустого JSON-файла.
func TestNewConfig_EmptyJSONFile(t *testing.T) {
	resetFlags()
	clearEnvVars()

	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "empty.json")

	// Создаем пустой файл
	err := os.WriteFile(configPath, []byte(""), 0644)
	require.NoError(t, err)

	oldArgs := os.Args
	os.Args = []string{"test", "-c", configPath}
	defer func() { os.Args = oldArgs }()

	cfg := config.NewConfig()

	// Должны применяться значения по умолчанию
	assert.Equal(t, "localhost:8080", cfg.ServerAddress)
	assert.Equal(t, "http://localhost:8080", cfg.BaseURL)
}
