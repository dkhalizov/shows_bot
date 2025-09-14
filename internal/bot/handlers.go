package bot

import (
	"fmt"
	"log"
	"log/slog"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/dkhalizov/shows/internal/models"
)

func (b *Bot) handleCommand(message *tgbotapi.Message) {
	switch message.Command() {
	case "start":
		b.handleStartCommand(message)
	case "help":
		b.handleHelpCommand(message)
	case "search":
		b.handleSearchCommand(message)
	case "list":
		b.handleListCommand(message)
	case "upcoming":
		b.handleUpcomingCommand(message)
	default:
		b.sendMessage(message.Chat.ID, "Unknown command. Type /help for available commands.")
	}
}

func (b *Bot) handleStartCommand(message *tgbotapi.Message) {
	welcomeMsg := `Welcome to the TV Shows Notification Bot!

I'll help you stay updated on your favorite shows. Use the menu below to navigate:

• Search for TV shows to follow
• View your followed shows
• Check upcoming episodes
• Get help

You can also type a show name directly to search for it.`

	b.sendMessageWithMarkup(message.Chat.ID, welcomeMsg, b.createMainMenu())
}

func (b *Bot) handleHelpCommand(message *tgbotapi.Message) {
	helpMsg := `*TV Shows Notification Bot Help*

• Use the Search button to find shows
• My Shows displays what you'htmlRegexp following
• Upcoming shows new episodes for your shows
• You can also just type a show name to search for it

When you follow a show, you'll receive notifications about new episodes.`

	b.sendMessageWithMarkup(message.Chat.ID, helpMsg, b.createMainMenu())
}

func (b *Bot) handleTextMessage(message *tgbotapi.Message) {
	b.searchShows(message.Chat.ID, message.Text)
}

func (b *Bot) handleSearchCommand(message *tgbotapi.Message) {
	query := message.CommandArguments()
	if query == "" {
		b.sendMessage(message.Chat.ID, "Please provide a show name to search for. Example: /search Breaking Bad")

		return
	}

	b.searchShows(message.Chat.ID, query)
}

func (b *Bot) handleListCommand(message *tgbotapi.Message) {
	userID := message.From.ID

	shows, err := b.dbManager.GetUserShows(userID)
	if err != nil {
		slog.Error("failed getting shows", "err", err)
		b.sendMessage(message.Chat.ID, "An error occurred while fetching your shows.")

		return
	}

	if len(shows) == 0 {
		b.sendMessage(message.Chat.ID, "You're not following any shows yet. Use /search to find shows to follow.")

		return
	}

	text := "📺 My Shows\n\nYou are following these shows:"

	for _, show := range shows {
		text += fmt.Sprintf("\n\n• %s", show.Name)
	}

	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton

	for _, show := range shows {
		row := []tgbotapi.InlineKeyboardButton{
			tgbotapi.NewInlineKeyboardButtonData(
				"📋 Details",
				fmt.Sprintf("details:%s", show.ID),
			),
			tgbotapi.NewInlineKeyboardButtonData(
				"❌ Unfollow",
				fmt.Sprintf("unfollow:%s", show.ID),
			),
		}
		inlineKeyboard = append(inlineKeyboard, row)
	}

	b.sendMessageWithMarkup(message.Chat.ID, text, tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...))
}

func (b *Bot) handleUpcomingCommand(message *tgbotapi.Message) {
	userID := message.From.ID

	episodes, err := b.dbManager.GetUpcomingEpisodesForUser(userID)
	if err != nil {
		log.Printf("Error getting upcoming episodes: %v", err)
		b.sendMessage(message.Chat.ID, "An error occurred while fetching upcoming episodes.")

		return
	}

	if len(episodes) == 0 {
		b.sendMessage(message.Chat.ID, "No upcoming episodes for your followed shows.")

		return
	}

	msg := "Upcoming episodes for your followed shows:\n"

	episodesByShow := make(map[string][]models.Episode)
	showNames := make(map[string]string)

	for _, episode := range episodes {
		episodesByShow[episode.ShowID] = append(episodesByShow[episode.ShowID], episode)

		if _, ok := showNames[episode.ShowID]; !ok {
			show, err := b.dbManager.GetShow(episode.ShowID)
			if err != nil {
				log.Printf("Error getting show: %v", err)

				continue
			}

			showNames[episode.ShowID] = show.Name
		}
	}

	for showID, showEpisodes := range episodesByShow {
		showName := showNames[showID]
		msg += fmt.Sprintf("\n\n*%s*", showName)

		for _, episode := range showEpisodes {
			msg += fmt.Sprintf("\n- S%02dE%02d: %s - %s",
				episode.SeasonNumber,
				episode.EpisodeNumber,
				episode.Name,
				episode.AirDate.Format("Jan 2, 2006"),
			)
		}
	}

	b.sendMessage(message.Chat.ID, msg)
}

func (b *Bot) handleCallbackQuery(callbackQuery *tgbotapi.CallbackQuery) {
	data := callbackQuery.Data
	userID := callbackQuery.From.ID
	chatID := callbackQuery.Message.Chat.ID

	slog.Debug("handleCallbackQuery", "data", data, "userID", userID)

	switch data {
	case MenuMain:
		b.editMessageWithMenu(
			chatID,
			callbackQuery.Message.MessageID,
			"📱 *Main Menu*\nSelect an option below:",
			b.createMainMenu(),
		)
		b.answerCallback(callbackQuery.ID, "")

		return

	case MenuMyShows:
		b.displayUserShows(chatID, callbackQuery.Message.MessageID, userID)
		b.answerCallback(callbackQuery.ID, "")

		return

	case MenuUpcoming:
		b.displayUpcomingEpisodes(chatID, callbackQuery.Message.MessageID, userID)
		b.answerCallback(callbackQuery.ID, "")

		return

	case MenuSearch:
		b.editMessageWithMenu(
			chatID,
			callbackQuery.Message.MessageID,
			"🔍 *Search for Shows*\n\nType the name of a show to search for it.",
			tgbotapi.NewInlineKeyboardMarkup(b.createBackHomeRow(MenuMain)),
		)
		b.answerCallback(callbackQuery.ID, "")

		return

	case MenuHelp:
		helpText := `*TV Shows Notification Bot Help*

• Use the Search button to find shows
• My Shows displays what you'htmlRegexp following
• Upcoming shows new episodes for your shows
• You can also just type a show name to search for it

When you follow a show, you'll receive notifications about new episodes.`

		b.editMessageWithMenu(
			chatID,
			callbackQuery.Message.MessageID,
			helpText,
			tgbotapi.NewInlineKeyboardMarkup(b.createBackHomeRow(MenuMain)),
		)
		b.answerCallback(callbackQuery.ID, "")

		return
	}

	parts := strings.SplitN(data, ":", 2)
	if len(parts) != 2 {
		log.Printf("Invalid callback data: %s", data)
		b.answerCallback(callbackQuery.ID, "Invalid action")

		return
	}

	action := parts[0]
	param := parts[1]

	var responseText string

	switch action {
	case ActionFollow:
		show, err := b.dbManager.GetShow(param)
		if err != nil {
			slog.Error("Error getting show", "err", err)
			b.answerCallback(callbackQuery.ID, "An error occurred while following the show.")

			return
		}

		err = b.dbManager.FollowShow(userID, param)
		if err != nil {
			log.Printf("Error following show: %v", err)

			responseText = "An error occurred while following the show."
		} else {
			responseText = fmt.Sprintf("You are now following %s", show.Name)

			b.displayShowDetails(chatID, callbackQuery.Message.MessageID, param, userID)
		}

		b.answerCallback(callbackQuery.ID, responseText)

		go func() {
			err = b.storeAllEpisodes(show)
			if err != nil {
				slog.Error("Error storing episodes", "err", err)
			}

			if err = b.notifyUsersAboutShowEpisodes(show); err != nil {
				slog.Error("failed to notify show episodes", "err", err)
			}
		}()

	case ActionUnfollow:
		show, err := b.dbManager.GetShow(param)
		if err != nil {
			log.Printf("Error getting show: %v", err)
			b.answerCallback(callbackQuery.ID, "An error occurred while unfollowing the show.")

			return
		}

		err = b.dbManager.UnfollowShow(userID, param)
		if err != nil {
			log.Printf("Error unfollowing show: %v", err)

			responseText = "An error occurred while unfollowing the show."
		} else {
			responseText = fmt.Sprintf("You have unfollowed %s", show.Name)

			b.displayShowDetails(chatID, callbackQuery.Message.MessageID, param, userID)
		}

		b.answerCallback(callbackQuery.ID, responseText)

	case ActionDetails:
		b.displayShowDetails(chatID, callbackQuery.Message.MessageID, param, userID)
		b.answerCallback(callbackQuery.ID, "")

	case ActionEpisodes:
		b.displayShowEpisodes(chatID, callbackQuery.Message.MessageID, param)
		b.answerCallback(callbackQuery.ID, "")
	case ActionBack:
		switch param {
		case "search_results":
			responseText = "Returning to search results..."
			b.answerCallback(callbackQuery.ID, responseText)

		case "my_shows":
			b.displayUserShows(chatID, callbackQuery.Message.MessageID, userID)
			b.answerCallback(callbackQuery.ID, "")

		default:
			b.editMessageWithMenu(
				chatID,
				callbackQuery.Message.MessageID,
				"📱 *Main Menu*\nSelect an option below:",
				b.createMainMenu(),
			)
			b.answerCallback(callbackQuery.ID, "")
		}

	default:
		log.Printf("Unknown callback action: %s", action)
		b.answerCallback(callbackQuery.ID, "Invalid action")
	}
}

func (b *Bot) storeAllEpisodes(show *models.Show) error {
	episodes, err := b.apiClients[show.Provider].GetEpisodes(show.ProviderID)
	if err != nil {
		return fmt.Errorf("error getting episodes %w", err)
	}

	for _, episode := range episodes {
		episode.Show = *show
		_, err = b.dbManager.StoreEpisode(&episode)
		if err != nil {
			return fmt.Errorf("error storing episode %w", err)
		}
	}

	return nil
}

func (b *Bot) answerCallback(id, text string) {
	if _, err := b.api.AnswerCallbackQuery(tgbotapi.NewCallback(id, text)); err != nil {
		slog.Error("Error answering callback query", "err", err)
	}
}
