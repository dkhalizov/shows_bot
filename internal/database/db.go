package database

import (
	"github.com/dkhalizov/shows/internal/models"
)

type Operations interface {
	Init() error

	StoreShow(show *models.Show) (string, error)
	GetShow(id string) (show *models.Show, err error)
	FollowShow(userID int, showID string) error
	UnfollowShow(userID int, showID string) error
	IsUserFollowingShow(userID int, showID string) (bool, error)
	GetUserShows(userID int) ([]models.Show, error)

	StoreUser(tgUser models.User) error
	GetAllFollowedShows() ([]string, error)
	GetUsersToNotify(episodeID, showID string) ([]int64, error)
	RecordNotification(userID int64, episodeID string) error
	IsShowFollowed(userID int64, showID string) (bool, error)

	StoreEpisode(episode *models.Episode) (string, error)
	GetNextEpisode(showID string) (*models.Episode, error)
	GetUpcomingEpisodesForUser(userID int) ([]models.Episode, error)
	GetEpisodesForShow(showID string) ([]models.Episode, error)
}
