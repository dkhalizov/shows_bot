package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/dkhalizov/shows/internal/config"
	"github.com/dkhalizov/shows/internal/models"
)

type Manager struct {
	db     *sql.DB
	config config.Database
}

func (m *Manager) Init() error {
	if err := m.db.Ping(); err != nil {
		return fmt.Errorf("failed to ping database: %w", err)
	}

	_, err := m.db.Exec(`
		CREATE TABLE IF NOT EXISTS shows (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			overview TEXT,
			poster_url TEXT,
			status TEXT,
			first_air_date TIMESTAMP,
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			imdb_id TEXT
		);

		CREATE TABLE IF NOT EXISTS episodes (
			id TEXT PRIMARY KEY,
			show_id TEXT NOT NULL,
			name TEXT NOT NULL,
			season_number INTEGER,
			episode_number INTEGER,
			air_date TIMESTAMP,
			overview TEXT,
			provider TEXT NOT NULL,
			provider_id TEXT NOT NULL,
			FOREIGN KEY (show_id) REFERENCES shows(id)
		);

		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			username TEXT,
			first_name TEXT,
			last_name TEXT
		);

		CREATE TABLE IF NOT EXISTS user_shows (
			user_id INTEGER,
			show_id TEXT,
			PRIMARY KEY (user_id, show_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (show_id) REFERENCES shows(id)
		);

		CREATE TABLE IF NOT EXISTS notifications (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER,
			episode_id TEXT,
			notified_at TIMESTAMP NOT NULL,
			UNIQUE(user_id, episode_id),
			FOREIGN KEY (user_id) REFERENCES users(id),
			FOREIGN KEY (episode_id) REFERENCES episodes(id)
		);

		CREATE INDEX IF NOT EXISTS idx_episodes_show_id ON episodes(show_id);
		CREATE INDEX IF NOT EXISTS idx_episodes_air_date ON episodes(air_date);
		CREATE INDEX IF NOT EXISTS idx_shows_imdb_id ON shows(imdb_id);
		CREATE INDEX IF NOT EXISTS idx_shows_provider_id ON shows(provider, provider_id);
	`)

	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

func (m *Manager) StoreShow(show *models.Show) (string, error) {
	var existingID string

	if show.IMDbID != "" {
		query := `SELECT id FROM shows WHERE imdb_id = ? AND imdb_id != ''`

		err := m.db.QueryRow(query, show.IMDbID).Scan(&existingID)
		if err == nil {
			return existingID, nil
		}
	}

	newID := show.GenerateID()

	query := `SELECT id FROM shows WHERE provider = ? AND provider_id = ?`

	err := m.db.QueryRow(query, show.Provider, show.ProviderID).Scan(&existingID)
	if err == nil {
		if show.IMDbID != "" {
			updateQuery := `UPDATE shows SET imdb_id = ? WHERE id = ? AND (imdb_id IS NULL OR imdb_id = '')`
			_, _ = m.db.Exec(updateQuery, show.IMDbID, existingID)
		}

		return existingID, nil
	}

	insertQuery := `
        INSERT INTO shows (id, name, overview, poster_url, status, first_air_date, provider, provider_id, imdb_id)
        VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	_, err = m.db.Exec(
		insertQuery,
		newID,
		show.Name,
		show.Overview,
		show.PosterURL,
		show.Status,
		show.FirstAirDate,
		show.Provider,
		show.ProviderID,
		show.IMDbID,
	)
	if err != nil {
		return "", err
	}

	return newID, nil
}

func (m *Manager) GetShow(showID string) (*models.Show, error) {
	query := `
		SELECT id, name, overview, poster_url, status, first_air_date, provider, provider_id, imdb_id
		FROM shows
		WHERE id = ?
	`

	var show models.Show
	var firstAirDate sql.NullTime

	err := m.db.QueryRow(query, showID).Scan(
		&show.ID,
		&show.Name,
		&show.Overview,
		&show.PosterURL,
		&show.Status,
		&firstAirDate,
		&show.Provider,
		&show.ProviderID,
		&show.IMDbID,
	)
	if err != nil {
		return nil, err
	}

	if firstAirDate.Valid {
		show.FirstAirDate = firstAirDate.Time
	}

	return &show, nil
}

func (m *Manager) FollowShow(userID int, showID string) error {
	query := `
		INSERT OR IGNORE INTO user_shows (user_id, show_id)
		VALUES (?, ?)
	`
	_, err := m.db.Exec(query, userID, showID)

	return err
}

func (m *Manager) UnfollowShow(userID int, showID string) error {
	query := `
		DELETE FROM user_shows
		WHERE user_id = ? AND show_id = ?
	`
	_, err := m.db.Exec(query, userID, showID)

	return err
}

func (m *Manager) IsUserFollowingShow(userID int, showID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM user_shows
			WHERE user_id = ? AND show_id = ?
		)
	`

	var following bool
	err := m.db.QueryRow(query, userID, showID).Scan(&following)

	return following, err
}

func (m *Manager) GetUserShows(userID int) ([]models.Show, error) {
	query := `
		SELECT s.id, s.name, s.overview, s.poster_url, s.status, s.first_air_date, s.provider, s.provider_id, s.imdb_id
		FROM shows s
		JOIN user_shows us ON s.id = us.show_id
		WHERE us.user_id = ?
		ORDER BY s.name
	`

	rows, err := m.db.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shows []models.Show

	for rows.Next() {
		var show models.Show
		var firstAirDate sql.NullTime

		err := rows.Scan(
			&show.ID,
			&show.Name,
			&show.Overview,
			&show.PosterURL,
			&show.Status,
			&firstAirDate,
			&show.Provider,
			&show.ProviderID,
			&show.IMDbID,
		)
		if err != nil {
			return nil, err
		}

		if firstAirDate.Valid {
			show.FirstAirDate = firstAirDate.Time
		}

		shows = append(shows, show)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return shows, nil
}

func (m *Manager) StoreUser(tgUser models.User) error {
	query := `
		INSERT INTO users (id, username, first_name, last_name)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE
		SET username = ?, first_name = ?, last_name = ?
	`
	_, err := m.db.Exec(
		query,
		tgUser.ID,
		tgUser.UserName,
		tgUser.FirstName,
		tgUser.LastName,
		tgUser.UserName,
		tgUser.FirstName,
		tgUser.LastName,
	)

	return err
}

func (m *Manager) GetAllFollowedShows() ([]string, error) {
	query := `
		SELECT DISTINCT show_id FROM user_shows
	`

	rows, err := m.db.Query(query)
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

func (m *Manager) GetUsersToNotify(episodeID, showID string) ([]int64, error) {
	query := `
		SELECT u.id
		FROM users u
		JOIN user_shows us ON u.id = us.user_id
		LEFT JOIN notifications n ON u.id = n.user_id AND n.episode_id = ?
		WHERE us.show_id = ? AND n.id IS NULL
	`

	rows, err := m.db.Query(query, episodeID, showID)
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
		INSERT OR IGNORE INTO notifications (user_id, episode_id, notified_at) 
		VALUES (?, ?, ?)
	`
	_, err := m.db.Exec(query, userID, episodeID, time.Now())

	return err
}

// IsShowFollowed checks if a show is followed by a user
func (m *Manager) IsShowFollowed(userID int64, showID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM user_shows WHERE user_id = ? AND show_id = ?
		)
	`

	var followed bool
	if err := m.db.QueryRow(query, userID, showID).Scan(&followed); err != nil {
		return false, err
	}

	return followed, nil
}

func (m *Manager) StoreEpisode(episode *models.Episode) (string, error) {
	newID := fmt.Sprintf("%s_%s", episode.Provider, episode.ProviderID)
	episode.ID = newID

	insertQuery := `
       INSERT OR IGNORE INTO episodes (id, show_id, name, season_number, episode_number, air_date, overview, provider, provider_id)
       VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
    `

	_, err := m.db.Exec(
		insertQuery,
		newID,
		episode.ShowID,
		episode.Name,
		episode.SeasonNumber,
		episode.EpisodeNumber,
		episode.AirDate,
		episode.Overview,
		episode.Provider,
		episode.ProviderID,
	)
	if err != nil {
		return "", err
	}

	return newID, nil
}

func (m *Manager) GetNextEpisode(showID string) (*models.Episode, error) {
	query := `
		SELECT id, show_id, name, season_number, episode_number, air_date, overview, provider, provider_id
		FROM episodes
		WHERE show_id = ? AND air_date > datetime('now')
		ORDER BY air_date
		LIMIT 1
	`

	var episode models.Episode
	var airDate sql.NullTime

	err := m.db.QueryRow(query, showID).Scan(
		&episode.ID,
		&episode.ShowID,
		&episode.Name,
		&episode.SeasonNumber,
		&episode.EpisodeNumber,
		&airDate,
		&episode.Overview,
		&episode.Provider,
		&episode.ProviderID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if airDate.Valid {
		episode.AirDate = airDate.Time
	}

	return &episode, nil
}

func (m *Manager) GetUpcomingEpisodesForUser(userID int) ([]models.Episode, error) {
	thirtyDaysFromNow := time.Now().AddDate(0, 0, 30).Format("2006-01-02 15:04:05")

	query := `
		SELECT e.id, e.show_id, e.name, e.season_number, e.episode_number, e.air_date, e.overview, e.provider, e.provider_id
		FROM episodes e
		JOIN user_shows us ON e.show_id = us.show_id
		WHERE us.user_id = ? AND e.air_date > datetime('now') AND e.air_date < ?
		ORDER BY e.air_date
	`

	rows, err := m.db.Query(query, userID, thirtyDaysFromNow)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []models.Episode

	for rows.Next() {
		var episode models.Episode
		var airDate sql.NullTime

		err := rows.Scan(
			&episode.ID,
			&episode.ShowID,
			&episode.Name,
			&episode.SeasonNumber,
			&episode.EpisodeNumber,
			&airDate,
			&episode.Overview,
			&episode.Provider,
			&episode.ProviderID,
		)
		if err != nil {
			return nil, err
		}

		if airDate.Valid {
			episode.AirDate = airDate.Time
		}

		episodes = append(episodes, episode)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return episodes, nil
}

func (m *Manager) GetEpisodesForShow(showID string) ([]models.Episode, error) {
	query := `
		SELECT id, show_id, name, season_number, episode_number, air_date, overview, provider, provider_id
		FROM episodes
		WHERE show_id = ?
		ORDER BY season_number, episode_number
	`

	rows, err := m.db.Query(query, showID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var episodes []models.Episode

	for rows.Next() {
		var episode models.Episode
		var airDate sql.NullTime

		err := rows.Scan(
			&episode.ID,
			&episode.ShowID,
			&episode.Name,
			&episode.SeasonNumber,
			&episode.EpisodeNumber,
			&airDate,
			&episode.Overview,
			&episode.Provider,
			&episode.ProviderID,
		)
		if err != nil {
			return nil, err
		}

		if airDate.Valid {
			episode.AirDate = airDate.Time
		}

		episodes = append(episodes, episode)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return episodes, nil
}

func MakeManager(config config.Database) (*Manager, error) {
	// Extract filename from the SQLite database URL
	// Expected format: sqlite:///path/to/database.db or file:///path/to/database.db
	dbURL := config.DatabaseURL
	var dbPath string

	if strings.HasPrefix(dbURL, "sqlite://") {
		dbPath = strings.TrimPrefix(dbURL, "sqlite://")
	} else if strings.HasPrefix(dbURL, "file://") {
		dbPath = strings.TrimPrefix(dbURL, "file://")
	} else {
		// If no prefix, assume it's a direct path
		dbPath = dbURL
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open SQLite database: %w", err)
	}

	if config.MaxConnections > 0 {
		db.SetMaxOpenConns(int(config.MaxConnections))
	}

	if config.ConnectionLifetime > 0 {
		db.SetConnMaxLifetime(config.ConnectionLifetime)
		db.SetConnMaxIdleTime(config.ConnectionLifetime / 2)
	}

	dbManager := &Manager{
		db:     db,
		config: config,
	}

	return dbManager, nil
}
