package database

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	"github.com/jackc/pgx/v4/pgxpool"

	"github.com/dkhalizov/shows/internal/config"
)

type Manager struct {
	pool   *pgxpool.Pool
	config config.Database
}

func NewManagerWithConfig(pool *pgxpool.Pool, config config.Database) *Manager {
	return &Manager{
		pool:   pool,
		config: config,
	}
}

func ConfigurePool(poolConfig *pgxpool.Config, dbConfig config.Database) {
	if !dbConfig.EnablePreparedStmts {
		poolConfig.ConnConfig.PreferSimpleProtocol = true
	}
}

func (m *Manager) InitDatabase() error {
	ctx := context.Background()

	if err := m.pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	if m.config.EnableAutomigrations {
		if err := m.runMigrations(); err != nil {
			return fmt.Errorf("failed to run migrations: %w", err)
		}
	}

	return nil
}

func (m *Manager) runMigrations() error {
	migrationDir := m.config.MigrationDirectory
	if migrationDir == "" {
		migrationDir = "migrations"
	}

	slog.Info("Running database migrations from directory", "dir", migrationDir)

	files, err := filepath.Glob(filepath.Join(migrationDir, "*.sql"))
	if err != nil {
		return fmt.Errorf("failed to find migration files: %w", err)
	}
	// TODO: Implement the rest of the function

	slog.Info("Found migration files", "count", len(files))

	return nil
}

func ConfigureConnectionPool(dbURL string, config config.Database) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	ConfigurePool(poolConfig, config)

	if config.MaxConnections > 0 {
		poolConfig.MaxConns = config.MaxConnections
	}

	if config.ConnectionLifetime > 0 {
		poolConfig.MaxConnLifetime = config.ConnectionLifetime
		poolConfig.MaxConnIdleTime = config.ConnectionLifetime / 2
	}

	pool, err := pgxpool.ConnectConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return pool, nil
}
