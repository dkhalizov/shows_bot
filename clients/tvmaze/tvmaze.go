package tvmaze

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/dkhalizov/shows/internal/models"
)

type Client struct {
	baseURL    string
	httpClient *http.Client
	maxRetries int
}

func NewClient() *Client {
	const defaultTimeout = 10 * time.Second

	return &Client{
		baseURL: "https://api.tvmaze.com",
		httpClient: &http.Client{
			Timeout: defaultTimeout,
		},
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
	url := fmt.Sprintf("%s/search/shows?q=%s", c.baseURL, encodedQuery)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result []struct {
		Score float64 `json:"score"`
		Show  struct {
			ID      int    `json:"id"`
			Name    string `json:"name"`
			Summary string `json:"summary"`
			Image   struct {
				Medium string `json:"medium"`
			} `json:"image"`
			Premiered string `json:"premiered"`
			Status    string `json:"status"`
			Externals struct {
				IMDb string `json:"imdb"`
			}
		} `json:"show"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	shows := make([]models.Show, len(result))

	for i, item := range result {
		show := models.Show{
			Name:       item.Show.Name,
			Overview:   item.Show.Summary,
			Status:     item.Show.Status,
			Provider:   "tvmaze",
			ProviderID: strconv.Itoa(item.Show.ID),
		}

		if item.Show.Image.Medium != "" {
			show.PosterURL = item.Show.Image.Medium
		}

		if item.Show.Externals.IMDb != "" {
			show.IMDbID = item.Show.Externals.IMDb
		}

		if item.Show.Premiered != "" {
			date, err := time.Parse("2006-01-02", item.Show.Premiered)
			if err == nil {
				show.FirstAirDate = date
			}
		}

		shows[i] = show
	}

	return shows, nil
}

func (c *Client) GetShowDetails(id string) (*models.Show, error) {
	url := fmt.Sprintf("%s/shows/%s", c.baseURL, id)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Summary string `json:"summary"`
		Image   struct {
			Medium string `json:"medium"`
		} `json:"image"`
		Premiered string `json:"premiered"`
		Status    string `json:"status"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	show := &models.Show{
		Name:       result.Name,
		Overview:   result.Summary,
		Status:     result.Status,
		Provider:   "tvmaze",
		ProviderID: strconv.Itoa(result.ID),
	}

	if result.Image.Medium != "" {
		show.PosterURL = result.Image.Medium
	}

	if result.Premiered != "" {
		date, err := time.Parse("2006-01-02", result.Premiered)
		if err == nil {
			show.FirstAirDate = date
		}
	}

	return show, nil
}

func (c *Client) GetEpisodes(showID string) ([]models.Episode, error) {
	url := fmt.Sprintf("%s/shows/%s/episodes", c.baseURL, showID)

	resp, err := c.makeRequest(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result []struct {
		ID      int    `json:"id"`
		Name    string `json:"name"`
		Season  int    `json:"season"`
		Number  int    `json:"number"`
		Airdate string `json:"airdate"`
		Summary string `json:"summary"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	episodes := make([]models.Episode, len(result))

	for i, item := range result {
		episode := models.Episode{
			Name:          item.Name,
			Overview:      item.Summary,
			SeasonNumber:  item.Season,
			EpisodeNumber: item.Number,
			Provider:      "tvmaze",
			ProviderID:    strconv.Itoa(item.ID),
		}

		if item.Airdate != "" {
			date, err := time.Parse("2006-01-02", item.Airdate)
			if err == nil {
				episode.AirDate = date
			}
		}

		episodes[i] = episode
	}

	return episodes, nil
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
