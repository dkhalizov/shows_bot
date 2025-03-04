package main

import (
	"log"
	"log/slog"

	"github.com/dkhalizov/shows/internal/bot"
	"github.com/dkhalizov/shows/internal/config"
	"github.com/dkhalizov/shows/internal/logging"
)

func main() {
	cfg := config.Load()
	if err := logging.Init(cfg.Logging); err != nil {
		log.Printf("Failed to initialize logging: %v", err)
	}

	slog.Debug("Loaded", "config", cfg)

	b, err := bot.New(cfg)
	if err != nil {
		log.Fatal("Failed to create bot:", err)
	}

	if err = b.Start(); err != nil {
		log.Fatal("Failed to start bot:", err)
	}
}
