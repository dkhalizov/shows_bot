package bot

import (
	"context"
	"fmt"
	"log"
	"shows/clients/tmdb"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
	"github.com/jackc/pgx/v4/pgxpool"

	"shows/clients"
	"shows/clients/tvmaze"
	"shows/internal/config"
	"shows/internal/database"
)

type Bot struct {
	api           *tgbotapi.BotAPI
	db            *pgxpool.Pool
	apiClients    map[string]clients.ShowAPIClient
	notifyTicker  *time.Ticker
	checkInterval time.Duration
	dbManager     *database.Manager
}

func New(config config.Config) (*Bot, error) {

	bot, err := tgbotapi.NewBotAPI(config.TelegramToken)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Telegram API: %w", err)
	}

	db, err := pgxpool.Connect(context.Background(), config.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	dbManager := database.NewManager(db)

	apiClients := make(map[string]clients.ShowAPIClient)

	if tmdbKey, ok := config.APIKeys["tmdb"]; ok {
		apiClients["tmdb"] = tmdb.NewClient(tmdbKey)
	}

	apiClients["tvmaze"] = tvmaze.NewClient()

	return &Bot{
		api:           bot,
		db:            db,
		dbManager:     dbManager,
		apiClients:    apiClients,
		notifyTicker:  time.NewTicker(6 * time.Hour),
		checkInterval: 6 * time.Hour,
	}, nil
}

func (b *Bot) Start() error {
	log.Println("Starting TV Shows notification bot...")

	if err := b.dbManager.InitDatabase(); err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	go b.runNotificationChecker()

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

	if err := b.dbManager.StoreUser(update.Message.From); err != nil {
		log.Printf("Error storing user: %v", err)
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
		log.Printf("Error sending message: %v", err)
	}
}
