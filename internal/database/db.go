package database

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jackc/pgx/v4/pgxpool"
)

type Manager struct {
	db *pgxpool.Pool
}

func NewManager(db *pgxpool.Pool) *Manager {
	return &Manager{
		db: db,
	}
}

func (m *Manager) InitDatabase() error {
	queries := []string{
		`CREATE SCHEMA IF NOT EXISTS shows_bot;
		SET search_path TO shows_bot;`,
		`CREATE TABLE IF NOT EXISTS users (
			id BIGINT PRIMARY KEY,
			username TEXT,
			first_name TEXT,
			last_name TEXT,
			created_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS shows (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			overview TEXT,
			poster_url TEXT,
			status TEXT,
			imdb_id TEXT,
			first_air_date TIMESTAMP,
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (provider, provider_id)
		)`,
		`CREATE TABLE IF NOT EXISTS episodes (
			id TEXT PRIMARY KEY,
			show_id TEXT NOT NULL REFERENCES shows_bot.shows(id) ON DELETE CASCADE,
			name TEXT NOT NULL,
			season_number INTEGER NOT NULL,
			episode_number INTEGER NOT NULL,
			air_date TIMESTAMP,
			overview TEXT,
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (provider, provider_id)
		)`,
		`CREATE TABLE IF NOT EXISTS user_shows (
			user_id BIGINT REFERENCES shows_bot.users(id) ON DELETE CASCADE,
			show_id TEXT REFERENCES shows_bot.shows(id) ON DELETE CASCADE,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			PRIMARY KEY (user_id, show_id)
		)`,
		`CREATE TABLE IF NOT EXISTS notifications (
			id SERIAL PRIMARY KEY,
			user_id BIGINT NOT NULL REFERENCES shows_bot.users(id) ON DELETE CASCADE,
			episode_id TEXT NOT NULL REFERENCES shows_bot.episodes(id) ON DELETE CASCADE,
			notified_at TIMESTAMP,
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			UNIQUE (user_id, episode_id)
		)`,
	}

	for _, query := range queries {
		_, err := m.db.Exec(context.Background(), query)
		if err != nil {
			return fmt.Errorf("failed to execute query %s: %w", query, err)
		}
	}

	return nil
}

func (m *Manager) StoreUser(tgUser *tgbotapi.User) error {
	query := `
		INSERT INTO shows_bot.users (id, username, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET username = $2, first_name = $3, last_name = $4
	`
	_, err := m.db.Exec(
		context.Background(),
		query,
		tgUser.ID,
		tgUser.UserName,
		tgUser.FirstName,
		tgUser.LastName,
	)
	return err
}

func (m *Manager) GetAllFollowedShows() ([]string, error) {
	query := `
		SELECT DISTINCT show_id FROM shows_bot.user_shows
	`

	rows, err := m.db.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var showIDs []string

	for rows.Next() {
		var showID string
		if err := rows.Scan(&showID); err != nil {
			return nil, err
		}
		showIDs = append(showIDs, showID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return showIDs, nil
}

func (m *Manager) GetUsersToNotify(episodeID string, showID string) ([]int64, error) {
	query := `
		SELECT u.id
		FROM shows_bot.users u
		JOIN shows_bot.user_shows us ON u.id = us.user_id
		LEFT JOIN shows_bot.notifications n ON u.id = n.user_id AND n.episode_id = $1
		WHERE us.show_id = $2 AND n.id IS NULL
	`

	rows, err := m.db.Query(context.Background(), query, episodeID, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var userIDs []int64

	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return userIDs, nil
}

func (m *Manager) RecordNotification(userID int64, episodeID string) error {
	query := `
		INSERT INTO shows_bot.notifications (user_id, episode_id, notified_at) 
		VALUES ($1, $2, NOW())
		ON CONFLICT (user_id, episode_id) DO NOTHING
	`
	_, err := m.db.Exec(context.Background(), query, userID, episodeID)
	return err
}

func (m *Manager) IsShowFollowed(userID int64, showID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM shows_bot.user_shows WHERE user_id = $1 AND show_id = $2
		)
	`

	var followed bool
	if err := m.db.QueryRow(context.Background(), query, userID, showID).Scan(&followed); err != nil {
		return false, err
	}

	return followed, nil
}
