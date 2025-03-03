package models

import (
	"time"
)

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

type User struct {
	ID        int64  `json:"id"`
	Username  string `json:"username"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}
