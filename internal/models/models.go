package models

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"time"
)

type User *tgbotapi.User
type Show struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Overview     string    `json:"overview"`
	PosterURL    string    `json:"poster_url"`
	Status       string    `json:"status"`
	FirstAirDate time.Time `json:"first_air_date"`
	NextEpisode  *Episode  `json:"next_episode,omitempty"`
	Provider     string    `json:"provider"`
	ProviderID   string    `json:"provider_id"`
	IMDbID       string    `json:"imdb_id"`
}

func (show Show) GenerateID() string {
	return fmt.Sprintf("%s_%s", show.Provider, show.ProviderID)
}

type Episode struct {
	ID            string    `json:"id"`
	ShowID        string    `json:"show_id"`
	Name          string    `json:"name"`
	SeasonNumber  int       `json:"season_number"`
	EpisodeNumber int       `json:"episode_number"`
	AirDate       time.Time `json:"air_date"`
	Overview      string    `json:"overview"`
	Provider      string    `json:"provider"`
	ProviderID    string    `json:"provider_id"`
}
