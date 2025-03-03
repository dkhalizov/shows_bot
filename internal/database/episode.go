package database

import (
	"context"
	"database/sql"
	"fmt"
	"shows/internal/models"
)

func (m *Manager) StoreEpisode(episode *models.Episode) (string, error) {

	query := `
		SELECT id FROM episodes
		WHERE provider = $1 AND provider_id = $2
	`

	var existingID string
	err := m.db.QueryRow(context.Background(), query, episode.Provider, episode.ProviderID).Scan(&existingID)

	if err != nil && err != sql.ErrNoRows {
		return "", err
	}

	if err == nil {

		return existingID, nil
	}

	newID := fmt.Sprintf("%s_%s", episode.Provider, episode.ProviderID)

	insertQuery := `
		INSERT INTO episodes (id, show_id, name, season_number, episode_number, air_date, overview, provider, provider_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id
	`

	err = m.db.QueryRow(
		context.Background(),
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
	).Scan(&newID)

	if err != nil {
		return "", err
	}

	return newID, nil
}

func (m *Manager) GetNextEpisode(showID string) (*models.Episode, error) {
	query := `
		SELECT id, show_id, name, season_number, episode_number, air_date, overview, provider, provider_id
		FROM episodes
		WHERE show_id = $1 AND air_date > NOW()
		ORDER BY air_date
		LIMIT 1
	`

	var episode models.Episode
	var airDate sql.NullTime

	err := m.db.QueryRow(context.Background(), query, showID).Scan(
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
	query := `
		SELECT e.id, e.show_id, e.name, e.season_number, e.episode_number, e.air_date, e.overview, e.provider, e.provider_id
		FROM episodes e
		JOIN user_shows us ON e.show_id = us.show_id
		WHERE us.user_id = $1 AND e.air_date > NOW() AND e.air_date < NOW() + INTERVAL '30 days'
		ORDER BY e.air_date
	`

	rows, err := m.db.Query(context.Background(), query, userID)
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
		WHERE show_id = $1
		ORDER BY season_number, episode_number
	`

	rows, err := m.db.Query(context.Background(), query, showID)
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
