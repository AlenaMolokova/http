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

	flag.StringVar(&cfg.ServerAddress, "a", "", "HTTP server address")
    flag.StringVar(&cfg.BaseURL, "b", "", "Base URL for shortened URLs")
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
