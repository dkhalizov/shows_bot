package models

import (
	"fmt"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"gorm.io/gorm"
)

type User struct {
	ID        int64 `gorm:"primaryKey"`
	Username  string
	FirstName string
	LastName  string
	CreatedAt time.Time `gorm:"autoCreateTime"`

	Shows []Show `gorm:"many2many:user_shows;"`
}

func FromTelegramUser(tgUser *tgbotapi.User) User {
	return User{
		ID:        int64(tgUser.ID),
		Username:  tgUser.UserName,
		FirstName: tgUser.FirstName,
		LastName:  tgUser.LastName,
	}
}

type Show struct {
	ID           string `gorm:"primaryKey"`
	Name         string `gorm:"not null"`
	Overview     string `gorm:"type:text"`
	PosterURL    string
	Status       string
	FirstAirDate time.Time
	Provider     string    `gorm:"not null;index:idx_provider_id,priority:1"`
	ProviderID   string    `gorm:"not null;index:idx_provider_id,priority:2;uniqueIndex:idx_provider_unique,priority:2"`
	IMDbID       string    `gorm:"index"`
	CreatedAt    time.Time `gorm:"autoCreateTime"`

	Episodes []Episode `gorm:"foreignKey:ShowID"`
	Users    []User    `gorm:"many2many:user_shows;"`
}

func (show *Show) BeforeCreate(tx *gorm.DB) error {
	if show.ID == "" {
		show.ID = show.GenerateID()
	}

	return nil
}

func (show Show) GenerateID() string {
	return fmt.Sprintf("%s_%s", show.Provider, show.ProviderID)
}

type Episode struct {
	ID            string `gorm:"primaryKey"`
	ShowID        string `gorm:"not null;index"`
	Name          string `gorm:"not null"`
	SeasonNumber  int    `gorm:"not null"`
	EpisodeNumber int    `gorm:"not null"`
	AirDate       time.Time
	Overview      string    `gorm:"type:text"`
	Provider      string    `gorm:"not null;index:idx_episode_provider,priority:1"`
	ProviderID    string    `gorm:"not null;index:idx_episode_provider,priority:2;uniqueIndex:idx_episode_provider_unique,priority:2"`
	CreatedAt     time.Time `gorm:"autoCreateTime"`

	Show          Show           `gorm:"foreignKey:ShowID"`
	Notifications []Notification `gorm:"foreignKey:EpisodeID"`
}

func (episode *Episode) BeforeCreate(tx *gorm.DB) error {
	if episode.ID == "" {
		episode.ID = fmt.Sprintf("%s_%s", episode.Provider, episode.ProviderID)
	}

	return nil
}

type Notification struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	UserID     int64     `gorm:"not null;uniqueIndex:idx_user_episode,priority:1"`
	EpisodeID  string    `gorm:"not null;uniqueIndex:idx_user_episode,priority:2"`
	NotifiedAt time.Time `gorm:"not null"`
	CreatedAt  time.Time `gorm:"autoCreateTime"`

	User    User    `gorm:"foreignKey:UserID"`
	Episode Episode `gorm:"foreignKey:EpisodeID"`
}

type UserShow struct {
	UserID    int64     `gorm:"primaryKey;autoCreateTime"`
	ShowID    string    `gorm:"primaryKey"`
	CreatedAt time.Time `gorm:"autoCreateTime"`

	User User `gorm:"foreignKey:UserID"`
	Show Show `gorm:"foreignKey:ShowID"`
}

func (User) TableName() string {
	return "shows_bot.users"
}

func (Show) TableName() string {
	return "shows_bot.shows"
}

func (Episode) TableName() string {
	return "shows_bot.episodes"
}

func (Notification) TableName() string {
	return "shows_bot.notifications"
}

func (UserShow) TableName() string {
	return "shows_bot.user_shows"
}
