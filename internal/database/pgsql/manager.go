package pgsql

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

func configurePool(poolConfig *pgxpool.Config, dbConfig config.Database) {
	if !dbConfig.EnablePreparedStmts {
		poolConfig.ConnConfig.PreferSimpleProtocol = true
	}
}

func (m *Manager) Init() error {
	ctx := context.Background()

	if err := m.pool.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	return nil
}

func ConfigureConnectionPool(dbURL string, config config.Database) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	configurePool(poolConfig, config)

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

func MakeManager(config config.Database) (*Manager, error) {
	dbConfig, err := pgxpool.ParseConfig(config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse database URL: %w", err)
	}

	dbConfig.MaxConns = config.MaxConnections
	dbConfig.MaxConnIdleTime = config.ConnectionLifetime
	dbConfig.MaxConnLifetime = config.ConnectionLifetime * 2

	if !config.EnablePreparedStmts {
		dbConfig.ConnConfig.PreferSimpleProtocol = true
	}

	pool, err := ConfigureConnectionPool(dbConfig.ConnString(), config)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool to database: %w", err)
	}

	dbManager := &Manager{
		pool:   pool,
		config: config,
	}

	return dbManager, nil
}
