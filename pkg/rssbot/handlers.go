package rssbot

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func (b *Bot) registerHandlers() {
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, b.wrapHandler(b.handleStart))
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, b.wrapHandler(b.handleHelp))
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/sub", bot.MatchTypePrefix, b.wrapHandler(b.handleSubscribe))
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/unsub", bot.MatchTypePrefix, b.wrapHandler(b.handleUnsubscribe))
	b.bot.RegisterHandler(bot.HandlerTypeMessageText, "/feeds", bot.MatchTypeExact, b.wrapHandler(b.handleListFeeds))
}

func (b *Bot) wrapHandler(handler func(context.Context, *bot.Bot, *models.Update)) func(context.Context, *bot.Bot, *models.Update) {
	return func(ctx context.Context, tgbot *bot.Bot, update *models.Update) {
		if update.Message == nil || update.Message.From == nil {
			return
		}

		chatID := fmt.Sprintf("%d", update.Message.Chat.ID)
		if !b.isChatAllowed(chatID) {
			tgbot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "Sorry, this is a private bot. Access is restricted to authorized users only.",
			})
			return
		}

		handler(ctx, tgbot, update)
	}
}

func (b *Bot) handleStart(ctx context.Context, tgbot *bot.Bot, update *models.Update) {
	text := "Welcome to RSS Bot! 🤖\n\n" +
		"I can help you subscribe to RSS feeds and notify you when new posts are published.\n\n" +
		"Use /help to see available commands."

	tgbot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}

func (b *Bot) handleHelp(ctx context.Context, tgbot *bot.Bot, update *models.Update) {
	text := "Available commands:\n\n" +
		"/start - Welcome message\n" +
		"/help - Show this help message\n" +
		"/sub <url> - Subscribe to an RSS feed\n" +
		"/unsub <search> - Unsubscribe from a feed\n" +
		"/feeds - List your subscribed feeds"

	tgbot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}

func (b *Bot) handleSubscribe(ctx context.Context, tgbot *bot.Bot, update *models.Update) {
	parts := strings.SplitN(update.Message.Text, " ", 2)
	if len(parts) < 2 {
		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please provide a URL. Usage: /sub <url>",
		})
		return
	}

	urlStr := strings.TrimSpace(parts[1])
	parsedURL, err := url.Parse(urlStr)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") {
		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please provide a valid HTTP or HTTPS URL.",
		})
		return
	}

	tgbot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Looking for RSS feed...",
	})

	feedURL, feedInfo, err := b.findAndParseFeed(ctx, urlStr)
	if err != nil {
		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("Failed to find RSS feed: %v", err),
		})
		return
	}

	sub := &Subscription{
		UserID:   update.Message.From.ID,
		ChatID:   update.Message.Chat.ID,
		FeedURL:  feedURL,
		FeedInfo: *feedInfo,
	}

	if err := b.db.AddSubscription(sub); err != nil {
		if strings.Contains(err.Error(), "already subscribed") {
			tgbot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   "You are already subscribed to this feed.",
			})
		} else {
			tgbot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   fmt.Sprintf("Failed to subscribe: %v", err),
			})
		}
		return
	}

	tgbot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   fmt.Sprintf("✅ Subscribed to: %s", feedInfo.Title),
	})
}

func (b *Bot) handleUnsubscribe(ctx context.Context, tgbot *bot.Bot, update *models.Update) {
	parts := strings.SplitN(update.Message.Text, " ", 2)
	if len(parts) < 2 {
		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Please provide a search term. Usage: /unsub <search>",
		})
		return
	}

	search := strings.TrimSpace(parts[1])
	subscriptions, err := b.db.GetUserSubscriptions(update.Message.From.ID)
	if err != nil {
		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to get your subscriptions.",
		})
		return
	}

	var matches []*Subscription
	searchLower := strings.ToLower(search)
	for _, sub := range subscriptions {
		if strings.Contains(strings.ToLower(sub.FeedInfo.Title), searchLower) ||
			strings.Contains(strings.ToLower(sub.FeedURL), searchLower) {
			matches = append(matches, sub)
		}
	}

	if len(matches) == 0 {
		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "No matching feeds found.",
		})
		return
	}

	if len(matches) == 1 {
		if err := b.db.RemoveSubscription(update.Message.From.ID, matches[0].FeedURL); err != nil {
			tgbot.SendMessage(ctx, &bot.SendMessageParams{
				ChatID: update.Message.Chat.ID,
				Text:   fmt.Sprintf("Failed to unsubscribe: %v", err),
			})
			return
		}

		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   fmt.Sprintf("✅ Unsubscribed from: %s", matches[0].FeedInfo.Title),
		})
		return
	}

	text := "Multiple feeds match your search:\n\n"
	for i, sub := range matches {
		text += fmt.Sprintf("%d. %s\n", i+1, sub.FeedInfo.Title)
	}
	text += "\nPlease be more specific."

	tgbot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}

func (b *Bot) handleListFeeds(ctx context.Context, tgbot *bot.Bot, update *models.Update) {
	subscriptions, err := b.db.GetUserSubscriptions(update.Message.From.ID)
	if err != nil {
		log.Printf("Error getting user subscriptions: %v", err)
		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "Failed to get your subscriptions.",
		})
		return
	}

	if len(subscriptions) == 0 {
		tgbot.SendMessage(ctx, &bot.SendMessageParams{
			ChatID: update.Message.Chat.ID,
			Text:   "You have no active subscriptions. Use /sub <url> to subscribe to a feed.",
		})
		return
	}

	text := "Your subscribed feeds:\n\n"
	for i, sub := range subscriptions {
		title := sub.FeedInfo.Title
		if len(title) > 50 {
			title = title[:47] + "..."
		}
		text += fmt.Sprintf("%d. %s\n", i+1, title)
	}

	tgbot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   text,
	})
}