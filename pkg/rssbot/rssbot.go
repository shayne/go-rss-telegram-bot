package rssbot

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type Config struct {
	DBPath        string
	CheckInterval time.Duration
	AllowedChatIDs []string
}

type Bot struct {
	bot           *bot.Bot
	db            *Database
	config        *Config
	checkInterval time.Duration
}

func New(apiKey string, cfg *Config) (*Bot, error) {
	db, err := NewDatabase(cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database: %w", err)
	}

	opts := []bot.Option{
		bot.WithDefaultHandler(defaultHandler),
	}

	b, err := bot.New(apiKey, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create bot: %w", err)
	}

	rssBot := &Bot{
		bot:           b,
		db:            db,
		config:        cfg,
		checkInterval: cfg.CheckInterval,
	}

	rssBot.registerHandlers()

	return rssBot, nil
}

func (b *Bot) Run(ctx context.Context) error {
	log.Println("Starting bot...")

	go b.startFeedChecker(ctx)

	b.bot.Start(ctx)

	return nil
}

func (b *Bot) startFeedChecker(ctx context.Context) {
	ticker := time.NewTicker(b.checkInterval)
	defer ticker.Stop()

	log.Printf("Starting feed checker with interval %v", b.checkInterval)

	b.checkFeeds(ctx)

	for {
		select {
		case <-ctx.Done():
			log.Println("Feed checker stopping...")
			return
		case <-ticker.C:
			b.checkFeeds(ctx)
		}
	}
}

func (b *Bot) checkFeeds(ctx context.Context) {
	log.Println("Checking feeds...")

	subscriptions, err := b.db.GetAllSubscriptions()
	if err != nil {
		log.Printf("Error getting subscriptions: %v", err)
		return
	}

	for _, sub := range subscriptions {
		select {
		case <-ctx.Done():
			return
		default:
			if err := b.checkFeed(ctx, sub); err != nil {
				log.Printf("Error checking feed %s: %v", sub.FeedURL, err)
			}
		}
	}
}

func (b *Bot) checkFeed(ctx context.Context, sub *Subscription) error {
	rssFeed, atomFeed, err := b.fetchFeed(ctx, sub.FeedURL)
	if err != nil {
		b.db.RecordFeedError(sub.FeedURL, err)
		return err
	}

	b.db.ClearFeedError(sub.FeedURL)

	var items []FeedItem
	if rssFeed != nil {
		items = b.extractRSSItems(rssFeed)
	} else if atomFeed != nil {
		items = b.extractAtomItems(atomFeed)
	}

	if len(items) == 0 {
		return nil
	}

	newestItem := items[0]
	if newestItem.GUID != sub.LastItemGUID && sub.LastItemGUID != "" {
		if err := b.sendFeedUpdate(ctx, sub, newestItem); err != nil {
			log.Printf("Failed to send update for %s: %v", sub.FeedURL, err)
			return err
		}
	}

	b.db.UpdateLastChecked(sub.UserID, sub.FeedURL, newestItem.GUID)
	return nil
}

func (b *Bot) isChatAllowed(chatID string) bool {
	if len(b.config.AllowedChatIDs) == 0 {
		return true
	}

	for _, allowed := range b.config.AllowedChatIDs {
		if allowed == chatID {
			return true
		}
	}
	return false
}

func defaultHandler(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Unknown command. Use /help to see available commands.",
		})
	}
}
