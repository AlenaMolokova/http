package config

import (
	"flag"
	"os"
)

type Config struct {
	ServerAddress string
	BaseURL       string
}

func InitConfig() *Config {
	cfg := &Config{}

	envServerAddress := os.Getenv("SERVER_ADDRESS")
	envBaseURL := os.Getenv("BASE_URL")

	defaultServerAddress := "localhost:8080"
	defaultBaseURL := "http://localhost:8080"

	flag.StringVar(&cfg.ServerAddress, "a", "", "Адрес запуска HTTP-сервера")
	flag.StringVar(&cfg.BaseURL, "b", "", "Базовый адрес результирующего сокращённого URL")
	flag.Parse()

	cfg.ServerAddress = firstNonEmpty(envServerAddress, cfg.ServerAddress, defaultServerAddress)
	cfg.BaseURL = firstNonEmpty(envBaseURL, cfg.BaseURL, defaultBaseURL)

	return cfg
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
