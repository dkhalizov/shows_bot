package clients

import "shows/internal/models"

type ShowAPIClient interface {
	SearchShows(query string) ([]models.Show, error)

	GetShowDetails(id string) (*models.Show, error)

	GetEpisodes(showID string) ([]models.Episode, error)

	GetUpcomingEpisodes(showID string) ([]models.Episode, error)
}
