package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type Logging struct {
	Level      string `yaml:"level"`
	File       string `yaml:"file"`
	MaxSize    int    `yaml:"max_size"`    // megabytes
	MaxBackups int    `yaml:"max_backups"` // number of backups
	MaxAge     int    `yaml:"max_age"`     // days
	Compress   bool   `yaml:"compress"`    // compress old log files
	JSON       bool   `yaml:"json_format"` // output logs in JSON format
}

type Server struct {
	Port            int           `yaml:"port"`
	ReadTimeout     time.Duration `yaml:"read_timeout"`
	WriteTimeout    time.Duration `yaml:"write_timeout"`
	ShutdownTimeout time.Duration `yaml:"shutdown_timeout"`
}

type Bot struct {
	NotificationEnabled          bool          `yaml:"notification_enabled"`
	CheckInterval                time.Duration `yaml:"check_interval"`
	MaxResults                   int           `yaml:"max_results"`
	MaxFollowedShows             int           `yaml:"max_followed_shows"`
	EpisodeNotificationThreshold time.Duration `yaml:"episode_notification_threshold"`
}

type Database struct {
	Type                string        `yaml:"type"`
	DatabaseURL         string        `yaml:"database_url"`
	MaxConnections      int32         `yaml:"max_connections"`
	MaxIdleConnections  int32         `yaml:"max_idle_connections"`
	ConnectionLifetime  time.Duration `yaml:"connection_lifetime"`
	StatementCacheSize  int           `yaml:"statement_cache_size"`
	EnablePreparedStmts bool          `yaml:"enable_prepared_statements"`
	LogAllQueries       bool          `yaml:"log_all_queries"`
}

type APIClients struct {
	TMDB struct {
		BaseURL     string        `yaml:"base_url"`
		Timeout     time.Duration `yaml:"timeout"`
		MaxRetries  int           `yaml:"max_retries"`
		RateLimit   int           `yaml:"rate_limit"`
		UsePosterV2 bool          `yaml:"use_poster_v2"`
	} `yaml:"tmdb"`
	TVMaze struct {
		BaseURL    string        `yaml:"base_url"`
		Timeout    time.Duration `yaml:"timeout"`
		MaxRetries int           `yaml:"max_retries"`
		RateLimit  int           `yaml:"rate_limit"`
	} `yaml:"tvmaze"`
}

type Development struct {
	Enabled   bool `yaml:"enabled"`
	MockAPIs  bool `yaml:"mock_apis"`
	DebugMode bool `yaml:"debug_mode"`
}

type Config struct {
	TelegramToken string            `yaml:"telegram_token"`
	APIKeys       map[string]string `yaml:"api_keys"`

	Bot Bot `yaml:"bot"`

	Logging  Logging  `yaml:"logging"`
	Database Database `yaml:"database"`

	APIClients APIClients `yaml:"api_clients"`

	Development Development `yaml:"development"`
}

func DefaultConfig() Config {
	cfg := Config{
		APIKeys: make(map[string]string),
	}

	cfg.Bot.NotificationEnabled = true
	cfg.Bot.CheckInterval = 6 * time.Hour
	cfg.Bot.MaxResults = 5
	cfg.Bot.MaxFollowedShows = 100
	cfg.Bot.EpisodeNotificationThreshold = 7 * 24 * time.Hour

	cfg.Logging.Level = "info"
	cfg.Logging.MaxSize = 100
	cfg.Logging.MaxBackups = 3
	cfg.Logging.MaxAge = 28
	cfg.Logging.Compress = true

	cfg.Database.MaxConnections = 10
	cfg.Database.MaxIdleConnections = 5
	cfg.Database.ConnectionLifetime = 5 * time.Minute
	cfg.Database.StatementCacheSize = 100

	cfg.APIClients.TMDB.BaseURL = "https://api.themoviedb.org/3"
	cfg.APIClients.TMDB.Timeout = 10 * time.Second
	cfg.APIClients.TMDB.MaxRetries = 3
	cfg.APIClients.TMDB.RateLimit = 40

	cfg.APIClients.TVMaze.BaseURL = "https://api.tvmaze.com"
	cfg.APIClients.TVMaze.Timeout = 10 * time.Second
	cfg.APIClients.TVMaze.MaxRetries = 3
	cfg.APIClients.TVMaze.RateLimit = 20

	return cfg
}

func Load() Config {
	cfg, err := load()
	if err != nil {
		slog.Error("Failed to load config", "err", err)

		return DefaultConfig()
	}

	return cfg
}

func load() (Config, error) {
	configFile := os.Getenv("CONFIG_FILE")
	if configFile == "" {
		possibleLocations := []string{
			"config.yml",
			"config.yaml",
			"./config/config.yml",
			"./config/config.yaml",
			"/etc/shows-bot/config.yml",
		}

		for _, loc := range possibleLocations {
			if _, err := os.Stat(loc); err == nil {
				configFile = loc

				break
			}
		}
	}

	var cfg Config
	if configFile != "" {
		if err := cfg.loadFromFile(configFile); err != nil {
			return cfg, fmt.Errorf("failed to load config file %s: %w", configFile, err)
		}
	}

	cfg.loadFromEnv()

	if err := cfg.validate(); err != nil {
		return cfg, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

func (c *Config) loadFromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(c); err != nil {
		return err
	}

	return nil
}

func (c *Config) loadFromEnv() {
	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		c.TelegramToken = token
	}

	if dbURL := os.Getenv("DATABASE_URL"); dbURL != "" {
		c.Database.DatabaseURL = dbURL
	}

	if tmdbKey := os.Getenv("TMDB_API_KEY"); tmdbKey != "" {
		c.APIKeys["tmdb"] = tmdbKey
	}

	if interval := os.Getenv("BOT_CHECK_INTERVAL"); interval != "" {
		if duration, err := time.ParseDuration(interval); err == nil {
			c.Bot.CheckInterval = duration
		}
	}

	if notifyEnabled := os.Getenv("NOTIFICATIONS_ENABLED"); notifyEnabled != "" {
		c.Bot.NotificationEnabled = notifyEnabled == "true" || notifyEnabled == "1" || notifyEnabled == "yes"
	}

	if logLevel := os.Getenv("LOG_LEVEL"); logLevel != "" {
		c.Logging.Level = logLevel
	}

	if logFile := os.Getenv("LOG_FILE"); logFile != "" {
		c.Logging.File = logFile
	}

	if devMode := os.Getenv("DEV_MODE"); devMode != "" {
		c.Development.Enabled = devMode == "true" || devMode == "1" || devMode == "yes"
	}
}

func (c *Config) validate() error {
	if c.TelegramToken == "" {
		return errors.New("telegram token is required")
	}

	if c.Database.DatabaseURL == "" {
		return errors.New("database URL is required")
	}

	return nil
}

func (c *Config) SaveToFile(filePath string) error {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return err
	}

	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := yaml.NewEncoder(file)
	defer encoder.Close()

	return encoder.Encode(c)
}
