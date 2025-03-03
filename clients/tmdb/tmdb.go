package tmdb

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dkhalizov/shows/internal/models"
)

type Client struct {
	apiKey      string
	baseURL     string
	httpClient  *http.Client
	usePosterV2 bool
	maxRetries  int
}

func NewClient(apiKey string) *Client {
	const defaultTimeout = 10 * time.Second

	const retries = 3

	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.themoviedb.org/3",
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
		maxRetries:  retries,
		usePosterV2: false,
	}
}

func (c *Client) SetBaseURL(baseURL string) {
	if baseURL != "" {
		c.baseURL = baseURL
	}
}

func (c *Client) SetTimeout(timeout time.Duration) {
	if timeout > 0 {
		c.httpClient.Timeout = timeout
	}
}

func (c *Client) SetMaxRetries(maxRetries int) {
	if maxRetries > 0 {
		c.maxRetries = maxRetries
	}
}

func (c *Client) EnablePosterV2(enabled bool) {
	c.usePosterV2 = enabled
}

func (c *Client) getPosterURL(posterPath string) string {
	if c.usePosterV2 {
		return fmt.Sprintf("https://image.tmdb.org/t/p/w780%s", posterPath)
	}

	return fmt.Sprintf("https://image.tmdb.org/t/p/w500%s", posterPath)
}

func (c *Client) makeRequest(url string) (*http.Response, error) {
	var resp *http.Response

	var err error

	for i := 0; i <= c.maxRetries; i++ {
		resp, err = c.httpClient.Get(url)
		if err == nil && resp.StatusCode < 500 {
			return resp, nil
		}

		if resp != nil {
			resp.Body.Close()
		}

		if i < c.maxRetries {
			// nolint:gosec
			backoff := time.Duration(1<<uint(i)) * time.Second
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts: %w", c.maxRetries+1, err)
}

func (c *Client) SearchShows(query string) ([]models.Show, error) {
	encodedQuery := url.QueryEscape(query)
	url := fmt.Sprintf("%s/search/tv?api_key=%s&query=%s", c.baseURL, c.apiKey, encodedQuery)

	resp, err := c.makeRequest(url)
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

	shows := make([]models.Show, len(result.Results))

	for i, item := range result.Results {
		show := models.Show{
			Name:       item.Name,
			Overview:   item.Overview,
			PosterURL:  c.getPosterURL(item.PosterPath),
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

		shows[i] = show
	}

	for i, item := range result.Results {
		if shows[i].IMDbID == "" {
			details, err := c.GetShowDetails(strconv.Itoa(item.ID))
			if err == nil && details.IMDbID != "" {
				shows[i].IMDbID = details.IMDbID
			}
		}
	}

	return shows, nil
}

func (c *Client) GetShowDetails(id string) (*models.Show, error) {
	url := fmt.Sprintf("%s/tv/%s?append_to_response=external_ids&api_key=%s", c.baseURL, id, c.apiKey)

	resp, err := c.makeRequest(url)
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
		ExternalIDs  struct {
			IMDb string `json:"imdb_id"`
		} `json:"external_ids"`
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
		IMDbID:     result.ExternalIDs.IMDb,
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

	resp, err := c.makeRequest(seasonsURL)
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
