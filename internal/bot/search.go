package bot

import (
	"fmt"
	"log"
	"regexp"
	"sort"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"

	"github.com/dkhalizov/shows/internal/models"
)

var htmlRegexp = regexp.MustCompile(`<[^>]*>`)

type SearchSession struct {
	Query    string
	Results  []models.Show
	Page     int
	PageSize int
}

func (b *Bot) enhanceSearchResults(chatID int64, query string, results []models.Show) {
	if len(results) == 0 {
		b.sendMessageWithMarkup(chatID, fmt.Sprintf("No shows found for: %s", query), b.createMainMenu())
		return
	}

	text := fmt.Sprintf("🔍 *Search Results for \"%s\"*\n", query)
	text += fmt.Sprintf("Found %d shows. Select for more options:", len(results))

	var inlineKeyboard [][]tgbotapi.InlineKeyboardButton

	for _, show := range results {
		status := ""

		if show.Status != "" {
			switch show.Status {
			case "Running":
			case "Continuing":
				status = "📺 Running"
			case "Ended":
				status = "🏁 Ended"
			default:
				status = show.Status
			}
		}

		yearInfo := ""
		if !show.FirstAirDate.IsZero() {
			yearInfo = fmt.Sprintf(" (%s)", show.FirstAirDate.Format("2006"))
		}

		text += fmt.Sprintf("\n\n• *%s*%s - %s",
			show.Name,
			yearInfo,
			status,
		)

		if show.Overview != "" {
			overview := stripHTMLTags(show.Overview)
			if len(overview) > 100 {
				overview = overview[:97] + "..."
			}

			text += fmt.Sprintf("\n  %s", overview)
		}

		detailsButton := tgbotapi.NewInlineKeyboardButtonData(
			"📋 Details",
			fmt.Sprintf("%s:%s", ActionDetails, show.ID),
		)

		followed, err := b.dbManager.IsShowFollowed(chatID, show.ID)
		if err != nil {
			log.Printf("Error checking if show is followed: %v", err)

			continue
		}

		var followButton tgbotapi.InlineKeyboardButton
		if followed {
			followButton = tgbotapi.NewInlineKeyboardButtonData(
				"❌ Unfollow",
				fmt.Sprintf("%s:%s", ActionUnfollow, show.ID),
			)
		} else {
			followButton = tgbotapi.NewInlineKeyboardButtonData(
				"✅ Follow",
				fmt.Sprintf("%s:%s", ActionFollow, show.ID),
			)
		}

		inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(detailsButton, followButton))
	}

	inlineKeyboard = append(inlineKeyboard, tgbotapi.NewInlineKeyboardRow(
		tgbotapi.NewInlineKeyboardButtonData("🏠 Home", MenuMain),
		tgbotapi.NewInlineKeyboardButtonData("🔍 New Search", MenuSearch),
	))

	escapedText := escapeMarkdown(text)

	b.sendMessageWithMarkup(chatID, escapedText, tgbotapi.NewInlineKeyboardMarkup(inlineKeyboard...))
}

func stripHTMLTags(s string) string {
	return htmlRegexp.ReplaceAllString(s, " ")
}

func escapeMarkdown(text string) string {
	specialChars := []string{"_", "*", "[", "]", "(", ")", "~", "`", ">", "#", "+", "-", "=", "|", "{", "}", ".", "!"}
	for _, char := range specialChars {
		text = strings.ReplaceAll(text, char, "\\"+char)
	}

	return text
}

func (b *Bot) searchShows(chatID int64, query string) {
	allResults := make([]models.Show, 0)

	for providerName, client := range b.apiClients {
		results, err := client.SearchShows(query)
		if err != nil {
			log.Printf("Error searching shows with %s: %v", providerName, err)

			continue
		}

		allResults = append(allResults, results...)
	}

	if len(allResults) == 0 {
		b.sendMessage(chatID, fmt.Sprintf("No shows found for query: %s", query))

		return
	}

	showsByIMDb := make(map[string][]models.Show)
	showsWithoutIMDb := make([]models.Show, 0)

	for _, show := range allResults {
		if show.IMDbID != "" {
			showsByIMDb[show.IMDbID] = append(showsByIMDb[show.IMDbID], show)
		} else {
			showsWithoutIMDb = append(showsWithoutIMDb, show)
		}
	}

	mergedResults := make([]models.Show, 0)

	for _, shows := range showsByIMDb {
		sort.Slice(shows, func(i, j int) bool {
			iScore := 0
			jScore := 0

			if shows[i].Overview != "" {
				iScore += 3
			}

			if shows[j].Overview != "" {
				jScore += 3
			}

			if shows[i].PosterURL != "" {
				iScore += 2
			}

			if shows[j].PosterURL != "" {
				jScore += 2
			}

			if !shows[i].FirstAirDate.IsZero() {
				iScore++
			}

			if !shows[j].FirstAirDate.IsZero() {
				jScore++
			}

			return iScore > jScore
		})

		bestShow := shows[0]

		showID, err := b.dbManager.StoreShow(&bestShow)
		if err != nil {
			log.Printf("Error storing show: %v", err)

			continue
		}

		bestShow.ID = showID
		mergedResults = append(mergedResults, bestShow)
	}

	for _, show := range showsWithoutIMDb {
		showID, err := b.dbManager.StoreShow(&show)
		if err != nil {
			log.Printf("Error storing show: %v", err)

			continue
		}

		show.ID = showID
		mergedResults = append(mergedResults, show)
	}

	maxResults := b.config.Bot.MaxResults
	if len(mergedResults) > maxResults {
		mergedResults = mergedResults[:maxResults]
	}

	b.enhanceSearchResults(chatID, query, mergedResults)
}
