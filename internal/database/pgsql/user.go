package pgsql

import (
	"context"
	"github.com/dkhalizov/shows/internal/models"
)

func (m *Manager) StoreUser(tgUser models.User) error {
	query := `
		INSERT INTO shows_bot.users (id, username, first_name, last_name)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE
		SET username = $2, first_name = $3, last_name = $4
	`
	_, err := m.pool.Exec(
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

	rows, err := m.pool.Query(context.Background(), query)
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
		FROM shows_bot.users u
		JOIN shows_bot.user_shows us ON u.id = us.user_id
		LEFT JOIN shows_bot.notifications n ON u.id = n.user_id AND n.episode_id = $1
		WHERE us.show_id = $2 AND n.id IS NULL
	`

	rows, err := m.pool.Query(context.Background(), query, episodeID, showID)
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
	_, err := m.pool.Exec(context.Background(), query, userID, episodeID)

	return err
}
