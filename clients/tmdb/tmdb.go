package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"shows/internal/models"
	"strconv"
	"time"
)

type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.themoviedb.org/3",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *Client) SearchShows(query string) ([]models.Show, error) {
	url := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s", c.baseURL, c.apiKey, query)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Results []struct {
			ID           int     `json:"id"`
			Name         string  `json:"name"`
			Overview     string  `json:"overview"`
			PosterPath   string  `json:"poster_path"`
			FirstAirDate string  `json:"first_air_date"`
			VoteAverage  float64 `json:"vote_average"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var shows []models.Show

	for _, item := range result.Results {
		show := models.Show{
			Name:       item.Name,
			Overview:   item.Overview,
			PosterURL:  fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", item.PosterPath),
			Status:     "",
			Provider:   "tmdb",
			ProviderID: strconv.Itoa(item.ID),
		}

		if item.FirstAirDate != "" {
			date, err := time.Parse("2006-01-02", item.FirstAirDate)
			if err == nil {
				show.FirstAirDate = date
			}
		}

		shows = append(shows, show)
	}

	return shows, nil
}

func (c *Client) GetShowDetails(id string) (*models.Show, error) {
	url := fmt.Sprintf("%s/tv/%s?api_key=%s", c.baseURL, id, c.apiKey)

	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID           int     `json:"id"`
		Name         string  `json:"name"`
		Overview     string  `json:"overview"`
		PosterPath   string  `json:"poster_path"`
		FirstAirDate string  `json:"first_air_date"`
		Status       string  `json:"status"`
		VoteAverage  float64 `json:"vote_average"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	show := &models.Show{
		Name:       result.Name,
		Overview:   result.Overview,
		PosterURL:  fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", result.PosterPath),
		Status:     result.Status,
		Provider:   "tmdb",
		ProviderID: strconv.Itoa(result.ID),
	}

	if result.FirstAirDate != "" {
		date, err := time.Parse("2006-01-02", result.FirstAirDate)
		if err == nil {
			show.FirstAirDate = date
		}
	}

	return show, nil
}

func (c *Client) GetEpisodes(showID string) ([]models.Episode, error) {

	seasonsURL := fmt.Sprintf("%s/tv/%s?api_key=%s", c.baseURL, showID, c.apiKey)

	resp, err := c.httpClient.Get(seasonsURL)
	if err != nil {
		return nil, err
	}

	var showData struct {
		Seasons []struct {
			SeasonNumber int `json:"season_number"`
		} `json:"seasons"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&showData); err != nil {
		resp.Body.Close()
		return nil, err
	}
	resp.Body.Close()

	var allEpisodes []models.Episode

	for _, season := range showData.Seasons {
		if season.SeasonNumber == 0 {

			continue
		}

		episodesURL := fmt.Sprintf("%s/tv/%s/season/%d?api_key=%s",
			c.baseURL, showID, season.SeasonNumber, c.apiKey)

		resp, err := c.httpClient.Get(episodesURL)
		if err != nil {
			continue
		}

		var seasonData struct {
			Episodes []struct {
				ID            int    `json:"id"`
				Name          string `json:"name"`
				Overview      string `json:"overview"`
				EpisodeNumber int    `json:"episode_number"`
				AirDate       string `json:"air_date"`
			} `json:"episodes"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&seasonData); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, ep := range seasonData.Episodes {
			episode := models.Episode{
				Name:          ep.Name,
				Overview:      ep.Overview,
				SeasonNumber:  season.SeasonNumber,
				EpisodeNumber: ep.EpisodeNumber,
				Provider:      "tmdb",
				ProviderID:    strconv.Itoa(ep.ID),
			}

			if ep.AirDate != "" {
				date, err := time.Parse("2006-01-02", ep.AirDate)
				if err == nil {
					episode.AirDate = date
				}
			}

			allEpisodes = append(allEpisodes, episode)
		}
	}

	return allEpisodes, nil
}

func (c *Client) GetUpcomingEpisodes(showID string) ([]models.Episode, error) {
	allEpisodes, err := c.GetEpisodes(showID)
	if err != nil {
		return nil, err
	}

	var upcomingEpisodes []models.Episode
	now := time.Now()

	for _, episode := range allEpisodes {
		if episode.AirDate.After(now) {
			upcomingEpisodes = append(upcomingEpisodes, episode)
		}
	}

	return upcomingEpisodes, nil
}
