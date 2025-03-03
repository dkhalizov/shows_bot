package bot

import (
	"fmt"
	"log"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"shows/internal/models"
)

func (b *Bot) runNotificationChecker() {
	log.Println("Starting notification checker...")

	b.checkForNewEpisodes()

	for range b.notifyTicker.C {
		b.checkForNewEpisodes()
	}
}

func (b *Bot) checkForNewEpisodes() {
	log.Println("Checking for new episodes...")

	showIDs, err := b.dbManager.GetAllFollowedShows()
	if err != nil {
		log.Printf("Error querying followed shows: %v", err)
		return
	}

	for _, showID := range showIDs {
		show, err := b.dbManager.GetShow(showID)
		if err != nil {
			log.Printf("Error getting show %s: %v", showID, err)
			continue
		}

		client, ok := b.apiClients[show.Provider]
		if !ok {
			log.Printf("No API client found for provider %s", show.Provider)
			continue
		}

		episodes, err := client.GetEpisodes(show.ProviderID)
		if err != nil {
			log.Printf("Error getting episodes for show %s: %v", show.Name, err)
			continue
		}

		for _, episode := range episodes {

			episode.ShowID = show.ID

			now := time.Now()
			thirtyDaysFromNow := now.AddDate(0, 0, 30)

			if episode.AirDate.After(now) && episode.AirDate.Before(thirtyDaysFromNow) {

				episodeID, err := b.dbManager.StoreEpisode(&episode)
				if err != nil {
					log.Printf("Error storing episode: %v", err)
					continue
				}

				if episodeID != "" {

					b.notifyUsersAboutEpisode(show, &episode)
				}
			}
		}
	}
}

func (b *Bot) notifyUsersAboutEpisode(show *models.Show, episode *models.Episode) {

	userIDs, err := b.dbManager.GetUsersToNotify(episode.ID, show.ID)
	if err != nil {
		log.Printf("Error getting users to notify: %v", err)
		return
	}

	for _, userID := range userIDs {

		message := fmt.Sprintf("🔔 *New Episode Alert* 🔔\n\n*%s*\nSeason %d, Episode %d: %s\n\nAirs on %s",
			show.Name,
			episode.SeasonNumber,
			episode.EpisodeNumber,
			episode.Name,
			episode.AirDate.Format("Monday, January 2, 2006"),
		)

		if episode.Overview != "" {

			overview := episode.Overview
			if len(overview) > 150 {
				overview = overview[:147] + "..."
			}
			message += fmt.Sprintf("\n\n%s", overview)
		}

		msg := tgbotapi.NewMessage(userID, message)
		msg.ParseMode = "Markdown"
		_, err := b.api.Send(msg)

		if err != nil {
			log.Printf("Error sending notification to user %d: %v", userID, err)
			continue
		}

		err = b.dbManager.RecordNotification(userID, episode.ID)
		if err != nil {
			log.Printf("Error recording notification: %v", err)
		}
	}
}
