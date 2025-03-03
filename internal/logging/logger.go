package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/dkhalizov/shows/internal/config"
)

func Init(cfg config.Logging) error {
	var level slog.Level

	switch cfg.Level {
	case "debug":
		level = slog.LevelDebug
	case "info":
		level = slog.LevelInfo
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	var handler slog.Handler

	var writer io.Writer

	if cfg.File != "" {
		if err := os.MkdirAll(filepath.Dir(cfg.File), 0o755); err != nil {
			return fmt.Errorf("mkdirall: %w", err)
		}

		logRotator := &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSize,
			MaxBackups: cfg.MaxBackups,
			MaxAge:     cfg.MaxAge,
			Compress:   cfg.Compress,
		}

		writer = io.MultiWriter(os.Stdout, logRotator)
	} else {
		writer = os.Stdout
	}

	if cfg.JSON {
		handler = slog.NewJSONHandler(writer, &slog.HandlerOptions{Level: level})
	} else {
		handler = slog.NewTextHandler(writer, &slog.HandlerOptions{Level: level})
	}

	slog.SetDefault(slog.New(handler))

	return nil
}
