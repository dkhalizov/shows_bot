package config

import (
	"errors"
	"os"
)

type Config struct {
	TelegramToken string
	DatabaseURL   string
	APIKeys       map[string]string
}

func Load() (Config, error) {
	telegramToken := os.Getenv("TELEGRAM_TOKEN")
	if telegramToken == "" {
		return Config{}, errors.New("TELEGRAM_TOKEN environment variable is required")
	}

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return Config{}, errors.New("DATABASE_URL environment variable is required")
	}

	apiKeys := make(map[string]string)

	tmdbKey := os.Getenv("TMDB_API_KEY")
	if tmdbKey != "" {
		apiKeys["tmdb"] = tmdbKey
	}

	return Config{
		TelegramToken: telegramToken,
		DatabaseURL:   databaseURL,
		APIKeys:       apiKeys,
	}, nil
}
