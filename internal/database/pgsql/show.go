package pgsql

import (
	"context"
	"database/sql"
	"github.com/dkhalizov/shows/internal/models"
)

func (m *Manager) StoreShow(show *models.Show) (string, error) {
	var existingID string

	if show.IMDbID != "" {
		query := `SELECT id FROM shows_bot.shows WHERE imdb_id = $1 AND imdb_id != ''`

		err := m.pool.QueryRow(context.Background(), query, show.IMDbID).Scan(&existingID)
		if err == nil {
			return existingID, nil
		}
	}

	newID := show.GenerateID()

	query := `
        SELECT id FROM shows_bot.shows
        WHERE provider = $1 AND provider_id = $2
    `

	err := m.pool.QueryRow(context.Background(), query, show.Provider, show.ProviderID).Scan(&existingID)
	if err == nil {
		if show.IMDbID != "" {
			updateQuery := `
                UPDATE shows_bot.shows 
                SET imdb_id = $1 
                WHERE id = $2 AND (imdb_id IS NULL OR imdb_id = '')
            `
			_, _ = m.pool.Exec(context.Background(), updateQuery, show.IMDbID, existingID)
		}

		return existingID, nil
	}

	insertQuery := `
        INSERT INTO shows_bot.shows (id, name, overview, poster_url, status, first_air_date, provider, provider_id, imdb_id)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
        RETURNING id
    `

	err = m.pool.QueryRow(
		context.Background(),
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
	).Scan(&newID)
	if err != nil {
		return "", err
	}

	return newID, nil
}

func (m *Manager) GetShow(showID string) (*models.Show, error) {
	query := `
		SELECT id, name, overview, poster_url, status, first_air_date, provider, provider_id, imdb_id
		FROM shows_bot.shows
		WHERE id = $1
	`

	var show models.Show

	var firstAirDate sql.NullTime

	err := m.pool.QueryRow(context.Background(), query, showID).Scan(
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
		INSERT INTO shows_bot.user_shows (user_id, show_id)
		VALUES ($1, $2)
		ON CONFLICT (user_id, show_id) DO NOTHING
	`
	_, err := m.pool.Exec(context.Background(), query, userID, showID)

	return err
}

func (m *Manager) UnfollowShow(userID int, showID string) error {
	query := `
		DELETE FROM shows_bot.user_shows
		WHERE user_id = $1 AND show_id = $2
	`
	_, err := m.pool.Exec(context.Background(), query, userID, showID)

	return err
}

func (m *Manager) IsUserFollowingShow(userID int, showID string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM shows_bot.user_shows
			WHERE user_id = $1 AND show_id = $2
		)
	`

	var following bool
	err := m.pool.QueryRow(context.Background(), query, userID, showID).Scan(&following)

	return following, err
}

func (m *Manager) GetUserShows(userID int) ([]models.Show, error) {
	query := `
		SELECT s.id, s.name, s.overview, s.poster_url, s.status, s.first_air_date, s.provider, s.provider_id, s.imdb_id
		FROM shows_bot.shows s
		JOIN shows_bot.user_shows us ON s.id = us.show_id
		WHERE us.user_id = $1
		ORDER BY s.name
	`

	rows, err := m.pool.Query(context.Background(), query, userID)
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

func (m *Manager) IsShowFollowed(userID int64, showID string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT 1 FROM shows_bot.user_shows WHERE user_id = $1 AND show_id = $2
		)
	`

	var followed bool
	if err := m.pool.QueryRow(context.Background(), query, userID, showID).Scan(&followed); err != nil {
		return false, err
	}

	return followed, nil
}
