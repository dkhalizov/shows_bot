package bot

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/dkhalizov/shows/internal/models"

	"github.com/dkhalizov/shows/clients"
	"github.com/dkhalizov/shows/clients/tmdb"
	"github.com/dkhalizov/shows/clients/tvmaze"
	"github.com/dkhalizov/shows/internal/config"
	"github.com/dkhalizov/shows/internal/database"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

type Bot struct {
	api           *tgbotapi.BotAPI
	apiClients    map[string]clients.ShowAPIClient
	notifyTicker  *time.Ticker
	checkInterval time.Duration
	dbManager     Operations
	config        config.Config
}

func New(config config.Config) (*Bot, error) {
	cli := makeHttpClient(config)

	bot, err := tgbotapi.NewBotAPIWithClient(config.TelegramToken, cli)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram API: %w", err)
	}

	bot.Debug = config.Development.DebugMode

	dbManager, err := database.NewManager(config.Database)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database manager: %w", err)
	}

	apiClients := makeAPIClients(config)

	return &Bot{
		api:           bot,
		dbManager:     dbManager,
		apiClients:    apiClients,
		notifyTicker:  time.NewTicker(config.Bot.CheckInterval),
		checkInterval: config.Bot.CheckInterval,
		config:        config,
	}, nil
}

func makeAPIClients(config config.Config) map[string]clients.ShowAPIClient {
	apiClients := make(map[string]clients.ShowAPIClient)

	if tmdbKey, ok := config.APIKeys["tmdb"]; ok {
		tmdbClient := tmdb.NewClient(tmdbKey)

		tmdbClient.SetBaseURL(config.APIClients.TMDB.BaseURL)
		tmdbClient.SetTimeout(config.APIClients.TMDB.Timeout)

		apiClients["tmdb"] = tmdbClient
	}

	tvmazeClient := tvmaze.NewClient()
	tvmazeClient.SetBaseURL(config.APIClients.TVMaze.BaseURL)
	tvmazeClient.SetTimeout(config.APIClients.TVMaze.Timeout)

	apiClients["tvmaze"] = tvmazeClient

	return apiClients
}

func (b *Bot) Start() error {
	slog.Info("Starting Telegram Bot")

	if err := b.dbManager.Init(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	if b.config.Bot.NotificationEnabled {
		slog.Debug("Starting notification checker...")

		go b.runNotificationChecker()
	} else {
		slog.Debug("Notifications are disabled in config")
	}

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates, err := b.api.GetUpdatesChan(u)
	if err != nil {
		return fmt.Errorf("failed to get updates channel: %w", err)
	}

	for update := range updates {
		go b.processUpdate(update)
	}

	return nil
}

func (b *Bot) processUpdate(update tgbotapi.Update) {
	if update.CallbackQuery != nil {
		b.handleCallbackQuery(update.CallbackQuery)

		return
	}

	if update.Message == nil {
		return
	}

	if update.Message.Text == "" {
		return
	}

	if err := b.dbManager.StoreUser(models.FromTelegramUser(update.Message.From)); err != nil {
		slog.Error("failed storing user", "err", err)
	}

	if update.Message.IsCommand() {
		b.handleCommand(update.Message)

		return
	}

	b.handleTextMessage(update.Message)
}

func (b *Bot) sendMessage(chatID int64, text string) {
	msg := tgbotapi.NewMessage(chatID, text)

	_, err := b.api.Send(msg)
	if err != nil {
		slog.Error("Error sending message", "err", err)
	}
}

func (b *Bot) sendMessageWithMarkup(chatID int64, text string, ikm tgbotapi.InlineKeyboardMarkup) {
	msg := tgbotapi.NewMessage(chatID, text)
	msg.ReplyMarkup = ikm

	_, err := b.api.Send(msg)
	if err != nil {
		slog.Error("Error sending message", "err", err)
	}
}

func makeHttpClient(config config.Config) *http.Client {
	return http.DefaultClient
}
