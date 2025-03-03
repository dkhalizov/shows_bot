package bot

import (
	"fmt"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"log"
	"shows/internal/models"
)

const (
	MenuMain     = "menu_main"
	MenuMyShows  = "menu_my_shows"
	MenuUpcoming = "menu_upcoming"
	MenuSearch   = "menu_search"
	MenuHelp     = "menu_help"

	ActionFollow   = "follow"
	ActionUnfollow = "unfollow"
	ActionDetails  = "details"
	ActionBack     = "back"
)

func (b *Bot) createMainMenu() tgbotapi.InlineKeyboardMarkup {
	return tgbotapi.NewInlineKeyboardMarkup(
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🔍 Search", MenuSearch),
			tgbotapi.NewInlineKeyboardButtonData("📺 My Shows", MenuMyShows),
		),
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📅 Upcoming", MenuUpcoming),
			tgbotapi.NewInlineKeyboardButtonData("❓ Help", MenuHelp),
		),
	)
}

func (b *Bot) createBackButton(target string) [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("◀️ Back", target),
		),
	}
}

func (b *Bot) createHomeButton() [][]tgbotapi.InlineKeyboardButton {
	return [][]tgbotapi.InlineKeyboardButton{
		tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("🏠 Home", MenuMain),
		),
	}
}

func (b *Bot) createBackHomeRow(backTarget string) []tgbotapi.InlineKeyboardButton {
	return tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Back", backTarget),
		tgbotapi.NewInlineKeyboardButtonData("🏠 Home", MenuMain),
	)
}

func (b *Bot) displayMainMenu(chatID int64) {
	msg := tgbotapi.NewMessage(chatID, "📱 *Main Menu*\nSelect an option below:")
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = b.createMainMenu()
	b.api.Send(msg)
}

func (b *Bot) editMessageWithMenu(chatID int64, messageID int, text string, markup tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewEditMessageText(chatID, messageID, text)
	msg.ParseMode = "Markdown"
	msg.ReplyMarkup = &markup
	b.api.Send(msg)
}

func (b *Bot) displayUserShows(chatID int64, messageID int, userID int) {

	shows, err := b.dbManager.GetUserShows(userID)
	if err != nil {
		b.editMessageWithMenu(
			chatID,
			messageID,
			"An error occurred while fetching your shows.",
			tgbotapi.NewInlineKeyboardMarkup(b.createHomeButton()...),
		)
		return
	}

	if len(shows) == 0 {
		b.editMessageWithMenu(
			chatID,
			messageID,
			"📺 *My Shows*\n\nYou'htmlRegexp not following any shows yet. Use the Search option to find shows to follow.",
			tgbotapi.NewInlineKeyboardMarkup(b.createHomeButton()...),
		)
		return
	}

	text := "📺 *My Shows*\n\nYou are following these shows:"

	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton

	for _, show := range shows {
		text += fmt.Sprintf("\n\n• *%s*", show.Name)

		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("📋 Details", fmt.Sprintf("%s:%s", ActionDetails, show.ID)),
			tgbotapi.NewInlineKeyboardButtonData("❌ Unfollow", fmt.Sprintf("%s:%s", ActionUnfollow, show.ID)),
		))
	}

	inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 Home", MenuMain),
	))

	b.editMessageWithMenu(
		chatID,
		messageID,
		text,
		tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...),
	)
}

func (b *Bot) displayShowDetails(chatID int64, messageID int, showID string, userID int) {

	show, err := b.dbManager.GetShow(showID)
	if err != nil {
		b.editMessageWithMenu(
			chatID,
			messageID,
			"An error occurred while fetching show details.",
			tgbotapi.NewInlineKeyboardMarkup(b.createHomeButton()...),
		)
		return
	}

	following, err := b.dbManager.IsUserFollowingShow(userID, show.ID)
	if err != nil {
		log.Printf("Error checking if user is following show: %v", err)
	}

	nextEpisode, err := b.dbManager.GetNextEpisode(show.ID)
	if err != nil {
		log.Printf("Error getting next episode: %v", err)
	}

	details := fmt.Sprintf("🎬 *%s*\n\n", show.Name)

	if show.Overview != "" {

		overview := show.Overview
		if len(overview) > 150 {
			overview = overview[:147] + "..."
		}
		details += fmt.Sprintf("%s\n\n", overview)
	}

	details += fmt.Sprintf("Status: %s\n", show.Status)
	details += fmt.Sprintf("First aired: %s\n", show.FirstAirDate.Format("January 2, 2006"))

	if nextEpisode != nil {
		details += fmt.Sprintf("\n📺 Next episode: S%02dE%02d - %s\n",
			nextEpisode.SeasonNumber,
			nextEpisode.EpisodeNumber,
			nextEpisode.Name,
		)
		details += fmt.Sprintf("Air date: %s", nextEpisode.AirDate.Format("January 2, 2006"))
	}

	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton

	if following {

		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("❌ Unfollow", fmt.Sprintf("%s:%s", ActionUnfollow, show.ID)),
		))
	} else {

		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData("✅ Follow", fmt.Sprintf("%s:%s", ActionFollow, show.ID)),
		))
	}

	inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("◀️ Back", fmt.Sprintf("%s:my_shows", ActionBack)),
		tgbotapi.NewInlineKeyboardButtonData("🏠 Home", MenuMain),
	))

	b.editMessageWithMenu(
		chatID,
		messageID,
		details,
		tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...),
	)
}

func (b *Bot) displayUpcomingEpisodes(chatID int64, messageID int, userID int) {

	episodes, err := b.dbManager.GetUpcomingEpisodesForUser(userID)
	if err != nil {
		b.editMessageWithMenu(
			chatID,
			messageID,
			"An error occurred while fetching upcoming episodes.",
			tgbotapi.NewInlineKeyboardMarkup(b.createHomeButton()...),
		)
		return
	}

	if len(episodes) == 0 {
		b.editMessageWithMenu(
			chatID,
			messageID,
			"📅 *Upcoming Episodes*\n\nNo upcoming episodes for your followed shows.",
			tgbotapi.NewInlineKeyboardMarkup(b.createHomeButton()...),
		)
		return
	}

	text := "📅 *Upcoming Episodes*\n"

	episodesByShow := make(map[string][]models.Episode)
	showNames := make(map[string]string)
	showIDs := make(map[string]string)

	for _, episode := range episodes {
		episodesByShow[episode.ShowID] = append(episodesByShow[episode.ShowID], episode)

		if _, ok := showNames[episode.ShowID]; !ok {
			show, err := b.dbManager.GetShow(episode.ShowID)
			if err != nil {
				log.Printf("Error getting show: %v", err)
				continue
			}
			showNames[episode.ShowID] = show.Name
			showIDs[show.Name] = show.ID
		}
	}

	for showID, showEpisodes := range episodesByShow {
		showName := showNames[showID]
		text += fmt.Sprintf("\n\n*%s*", showName)

		for _, episode := range showEpisodes {
			text += fmt.Sprintf("\n- S%02dE%02d: %s - %s",
				episode.SeasonNumber,
				episode.EpisodeNumber,
				episode.Name,
				episode.AirDate.Format("Jan 2, 2006"),
			)
		}
	}

	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton

	for showName, showID := range showIDs {
		shortName := showName
		if len(shortName) > 20 {
			shortName = shortName[:17] + "..."
		}

		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(
			tgbotapi.NewInlineKeyboardButtonData(
				fmt.Sprintf("📋 %s", shortName),
				fmt.Sprintf("%s:%s", ActionDetails, showID),
			),
		))
	}

	inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 Home", MenuMain),
	))

	b.editMessageWithMenu(
		chatID,
		messageID,
		text,
		tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...),
	)
}
