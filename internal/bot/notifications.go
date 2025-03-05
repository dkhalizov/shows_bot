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

		if err = b.notifyUsersAboutShowEpisodes(show); err != nil {
			slog.Error("Error notifying users about the show", "showID", showID, "err", err)
		}
	}
}

func (b *Bot) notifyUsersAboutShowEpisodes(show *models.Show) error {
	client, ok := b.apiClients[show.Provider]
	if !ok {
		return fmt.Errorf("apiClient for show %s :  %s not found", show.ID, show.Provider)
	}

	episodes, err := client.GetEpisodes(show.ProviderID)
	if err != nil {
		return fmt.Errorf("could not get episodes for show %s :  %w", show.ID, err)
	}

	for _, episode := range episodes {
		episode.ShowID = show.ID
		now := time.Now()
		isFutureEpisode := episode.AirDate.After(now)

		if isFutureEpisode {
			episodeID, err := b.dbManager.StoreEpisode(&episode)
			if err != nil {
				slog.Error("Error storing episode", "episodeID", episodeID, "err", err)
			}
		}
		// notify users if the episode is within the notification threshold
		fromNow := now.Add(b.config.Bot.EpisodeNotificationThreshold)
		shouldNotify := isFutureEpisode && episode.AirDate.Before(fromNow)

		if shouldNotify {
			if err = b.notifyUsersAboutEpisode(show, &episode); err != nil {
				slog.Error("Error notifying users about episode", "episodeID", episode.ID, "err", err)

				continue
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

	slog.Debug("Got users to notify", "userIDs", userIDs)

	for _, userID := range userIDs {
		slog.Debug("Notifying user about episode", "userID", userID, "episodeID", episode.ID)
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

		b.sendMessage(userID, message)

		err = b.dbManager.RecordNotification(userID, episode.ID)
		if err != nil {
			return fmt.Errorf("could not record notification for user %d: %w", userID, err)
		}
	}

	return nil
}
