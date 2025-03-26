package bot

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/dkhalizov/shows/internal/models"
)

func (b *Bot) runNotificationChecker() {
	b.checkForNewEpisodes()

	for range b.notifyTicker.C {
		b.checkForNewEpisodes()
	}
}

func (b *Bot) checkForNewEpisodes() {
	showIDs, err := b.dbManager.GetAllFollowedShows()
	if err != nil {
		slog.Error("Error querying followed shows", "err", err)

		return
	}

	slog.Debug("Checking for new episodes...", "showIDs", showIDs)

	for _, showID := range showIDs {
		show, err := b.dbManager.GetShow(showID)
		if err != nil {
			slog.Error("Error querying show", "showID", showID, "err", err)

			continue
		}

		if err = b.refreshShowEpisodes(show); err != nil {
			slog.Error("Error refreshing episodes for show", "showID", showID, "err", err)
		}

		if err = b.notifyUsersAboutShowEpisodes(show); err != nil {
			slog.Error("Error notifying users about the show", "showID", showID, "err", err)
		}
	}
}

// refreshShowEpisodes fetches only upcoming episode data from the API and updates the database
// This is separated from notification logic to ensure we always have updated episode data
func (b *Bot) refreshShowEpisodes(show *models.Show) error {
	client, ok := b.apiClients[show.Provider]
	if !ok {
		return fmt.Errorf("apiClient for show %s : %s not found", show.ID, show.Provider)
	}

	// Only get upcoming episodes from the API
	episodes, err := client.GetUpcomingEpisodes(show.ProviderID)
	if err != nil {
		slog.Error("Failed to get upcoming episodes, falling back to stored episodes",
			"showID", show.ID,
			"showName", show.Name,
			"err", err)
		return nil // Don't fail the entire notification process if API fails
	}

	if len(episodes) == 0 {
		slog.Debug("No upcoming episodes found for show", "showID", show.ID, "showName", show.Name)
		return nil
	}

	slog.Debug("Refreshing upcoming episodes",
		"showName", show.Name,
		"count", len(episodes))

	// Store only upcoming episodes
	for _, episode := range episodes {
		episode.ShowID = show.ID
		_, err := b.dbManager.StoreEpisode(&episode)
		if err != nil {
			slog.Error("Error storing episode", "episode", episode.Name, "err", err)
		}
	}

	return nil
}

func (b *Bot) notifyUsersAboutShowEpisodes(show *models.Show) error {
	episodes, err := b.dbManager.GetEpisodesForShow(show.ID)
	if err != nil {
		return fmt.Errorf("could not get episodes for show %s : %w", show.ID, err)
	}

	now := time.Now()
	notificationThreshold := now.Add(b.config.Bot.EpisodeNotificationThreshold)

	slog.Debug("Checking episodes for notification",
		"show", show.Name,
		"episodeCount", len(episodes),
		"threshold", b.config.Bot.EpisodeNotificationThreshold)

	for _, episode := range episodes {
		// Only notify for episodes that:
		// 1. Are in the future
		// 2. Are within the notification threshold (e.g., in the next 24 hours)
		// 3. Haven't been notified yet
		isFutureEpisode := episode.AirDate.After(now)
		isWithinThreshold := episode.AirDate.Before(notificationThreshold)

		if isFutureEpisode && isWithinThreshold {
			if err = b.notifyUsersAboutEpisode(show, &episode); err != nil {
				slog.Error("Error notifying users about episode",
					"episodeID", episode.ID,
					"showName", show.Name,
					"episodeName", episode.Name,
					"err", err)
			}
		}
	}

	return nil
}

func (b *Bot) notifyUsersAboutEpisode(show *models.Show, episode *models.Episode) error {
	userIDs, err := b.dbManager.GetUsersToNotify(episode.ID, show.ID)
	if err != nil {
		return fmt.Errorf("could not get users to notify: %w", err)
	}

	if len(userIDs) == 0 {
		slog.Debug("No users to notify for episode",
			"showName", show.Name,
			"episodeName", episode.Name,
			"episodeID", episode.ID)
		return nil
	}

	slog.Info("Notifying users about upcoming episode",
		"userCount", len(userIDs),
		"show", show.Name,
		"episode", episode.Name,
		"airDate", episode.AirDate)

	for _, userID := range userIDs {
		message := fmt.Sprintf("🔔 *New Episode Alert* 🔔\n\n*%s*\nSeason %d, Episode %d: %s\n\nAirs on %s",
			show.Name,
			episode.SeasonNumber,
			episode.EpisodeNumber,
			episode.Name,
			episode.AirDate.Format("Monday, January 2, 2006"),
		)

		if episode.Overview != "" {
			overview := stripHTMLTags(episode.Overview)
			if len(overview) > 150 {
				overview = overview[:147] + "..."
			}

			message += fmt.Sprintf("\n\n%s", overview)
		}

		// Add information about how soon the episode will air
		timeUntilAiring := episode.AirDate.Sub(time.Now())
		if timeUntilAiring < 24*time.Hour {
			message += fmt.Sprintf("\n\n⏰ This episode airs in less than 24 hours!")
		} else {
			daysUntil := int(timeUntilAiring.Hours() / 24)
			message += fmt.Sprintf("\n\n⏰ This episode airs in %d days", daysUntil)
		}

		b.sendMessage(userID, message)

		err = b.dbManager.RecordNotification(userID, episode.ID)
		if err != nil {
			return fmt.Errorf("could not record notification for user %d: %w", userID, err)
		}
	}

	return nil
}
