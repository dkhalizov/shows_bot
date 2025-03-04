package database

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/dkhalizov/shows/internal/config"
	"github.com/dkhalizov/shows/internal/models"
)

type Manager struct {
	db     *gorm.DB
	config config.Database
}

func NewManager(config config.Database) (*Manager, error) {
	var db *gorm.DB

	var err error

	logLevel := logger.Error
	if config.LogAllQueries {
		logLevel = logger.Info
	}

	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	}

	switch {
	case strings.HasPrefix(config.DatabaseURL, "postgres"):
		db, err = gorm.Open(postgres.Open(config.DatabaseURL), gormConfig)
	case strings.HasPrefix(config.DatabaseURL, "sqlite"):
		db, err = gorm.Open(sqlite.Open(strings.TrimPrefix(config.DatabaseURL, "sqlite://")), gormConfig)
	case strings.HasPrefix(config.DatabaseURL, "file:"):
		db, err = gorm.Open(sqlite.Open(strings.TrimPrefix(config.DatabaseURL, "file://")), gormConfig)
	default:
		db, err = gorm.Open(sqlite.Open(config.DatabaseURL), gormConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	if config.MaxConnections > 0 {
		sqlDB.SetMaxOpenConns(int(config.MaxConnections))
	}

	if config.MaxIdleConnections > 0 {
		sqlDB.SetMaxIdleConns(int(config.MaxIdleConnections))
	}

	if config.ConnectionLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnectionLifetime)
	}

	return &Manager{db: db, config: config}, nil
}

func (m *Manager) Init() error {
	if strings.HasPrefix(m.config.DatabaseURL, "postgres") {
		m.db.Exec("CREATE SCHEMA IF NOT EXISTS shows_bot")
		//set schema
		m.db.Exec("SET search_path TO shows_bot")
	}

	slog.Debug("Running database migrations...")

	err := m.db.AutoMigrate(
		&models.User{},
		&models.Show{},
		&models.Episode{},
		&models.Notification{},
		&models.UserShow{},
	)
	if err != nil {
		return fmt.Errorf("failed to migrate database: %w", err)
	}

	slog.Debug("Database migrations completed successfully")

	return nil
}

func (m *Manager) StoreUser(user models.User) error {
	return m.db.Save(&user).Error
}

func (m *Manager) StoreShow(show *models.Show) (string, error) {
	if show.IMDbID != "" {
		var existingShow models.Show

		result := m.db.Where("imdb_id = ? AND imdb_id != ''", show.IMDbID).First(&existingShow)
		if result.Error == nil {
			return existingShow.ID, nil
		}
	}

	var existingShow models.Show

	result := m.db.Where("provider = ? AND provider_id = ?", show.Provider, show.ProviderID).First(&existingShow)
	if result.Error == nil {
		if show.IMDbID != "" && (existingShow.IMDbID == "" || existingShow.IMDbID == "0") {
			existingShow.IMDbID = show.IMDbID
			m.db.Save(&existingShow)
		}

		return existingShow.ID, nil
	}

	if show.ID == "" {
		show.ID = show.GenerateID()
	}

	if err := m.db.Create(show).Error; err != nil {
		return "", err
	}

	return show.ID, nil
}

func (m *Manager) GetShow(id string) (*models.Show, error) {
	var show models.Show

	result := m.db.First(&show, "id = ?", id)
	if result.Error != nil {
		return nil, result.Error
	}

	return &show, nil
}

func (m *Manager) FollowShow(userID int, showID string) error {
	return m.db.Create(&models.UserShow{
		UserID: int64(userID),
		ShowID: showID,
	}).Error
}

func (m *Manager) UnfollowShow(userID int, showID string) error {
	return m.db.Where("user_id = ? AND show_id = ?", userID, showID).Delete(&models.UserShow{}).Error
}

func (m *Manager) IsUserFollowingShow(userID int, showID string) (bool, error) {
	var count int64
	err := m.db.Model(&models.UserShow{}).Where("user_id = ? AND show_id = ?", userID, showID).Count(&count).Error

	return count > 0, err
}

func (m *Manager) GetUserShows(userID int) ([]models.Show, error) {
	var shows []models.Show
	err := m.db.Joins("JOIN shows_bot.user_shows ON shows_bot.user_shows.show_id = shows_bot.shows.id").
		Where("shows_bot.user_shows.user_id = ?", userID).
		Order("shows_bot.shows.name").
		Find(&shows).Error

	return shows, err
}

func (m *Manager) GetAllFollowedShows() ([]string, error) {
	var showIDs []string
	err := m.db.Model(&models.UserShow{}).Distinct().Pluck("show_id", &showIDs).Error

	return showIDs, err
}

func (m *Manager) GetUsersToNotify(episodeID, showID string) ([]int64, error) {
	var userIDs []int64
	err := m.db.Model(&models.User{}).
		Joins("JOIN shows_bot.user_shows ON shows_bot.user_shows.user_id = shows_bot.users.id").
		Joins("LEFT JOIN shows_bot.notifications ON shows_bot.notifications.user_id = shows_bot.users.id AND shows_bot.notifications.episode_id = ?", episodeID).
		Where("shows_bot.user_shows.show_id = ? AND shows_bot.notifications.id IS NULL", showID).
		Pluck("shows_bot.users.id", &userIDs).Error

	return userIDs, err
}

func (m *Manager) RecordNotification(userID int64, episodeID string) error {
	return m.db.Create(&models.Notification{
		UserID:     userID,
		EpisodeID:  episodeID,
		NotifiedAt: time.Now(),
	}).Error
}

func (m *Manager) IsShowFollowed(userID int64, showID string) (bool, error) {
	var count int64
	err := m.db.Model(&models.UserShow{}).Where("user_id = ? AND show_id = ?", userID, showID).Count(&count).Error

	return count > 0, err
}

func (m *Manager) StoreEpisode(episode *models.Episode) (string, error) {
	if episode.ID == "" {
		episode.ID = fmt.Sprintf("%s_%s", episode.Provider, episode.ProviderID)
	}

	var existingEpisode models.Episode

	result := m.db.First(&existingEpisode, "id = ?", episode.ID)
	if result.Error == nil {
		return existingEpisode.ID, nil
	}

	if err := m.db.Create(episode).Error; err != nil {
		return "", err
	}

	return episode.ID, nil
}

func (m *Manager) GetNextEpisode(showID string) (*models.Episode, error) {
	var episode models.Episode
	result := m.db.Where("show_id = ? AND air_date > ?", showID, time.Now()).
		Order("air_date").
		First(&episode)

	if result.Error == gorm.ErrRecordNotFound {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &episode, nil
}

func (m *Manager) GetUpcomingEpisodesForUser(userID int) ([]models.Episode, error) {
	var episodes []models.Episode
	thirtyDaysFromNow := time.Now().AddDate(0, 0, 30)

	err := m.db.Joins("JOIN shows_bot.user_shows ON shows_bot.user_shows.show_id = shows_bot.episodes.show_id").
		Where("shows_bot.user_shows.user_id = ? AND shows_bot.episodes.air_date > ? AND shows_bot.episodes.air_date < ?",
			userID, time.Now(), thirtyDaysFromNow).
		Order("shows_bot.episodes.air_date").
		Find(&episodes).Error

	return episodes, err
}

func (m *Manager) GetEpisodesForShow(showID string) ([]models.Episode, error) {
	var episodes []models.Episode
	err := m.db.Where("show_id = ?", showID).
		Order("season_number, episode_number").
		Find(&episodes).Error

	return episodes, err
}
