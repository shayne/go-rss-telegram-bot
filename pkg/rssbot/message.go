package rssbot

import (
	"context"
	"fmt"
	"html"
	"log"
	"strings"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

type FeedItem struct {
	Title       string
	Link        string
	Author      string
	GUID        string
	Description string
}

func (b *Bot) extractRSSItems(feed *RSSFeed) []FeedItem {
	items := make([]FeedItem, 0, len(feed.Channel.Items))
	for _, item := range feed.Channel.Items {
		author := item.Author
		if author == "" {
			author = item.Creator
		}
		items = append(items, FeedItem{
			Title:       item.Title,
			Link:        item.Link,
			Author:      author,
			GUID:        item.GUID,
			Description: item.Description,
		})
	}
	return items
}

func (b *Bot) extractAtomItems(feed *AtomFeed) []FeedItem {
	items := make([]FeedItem, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		link := ""
		for _, l := range entry.Link {
			if l.Rel == "alternate" || l.Rel == "" {
				link = l.Href
				break
			}
		}
		content := entry.Summary
		if content == "" {
			content = entry.Content
		}
		items = append(items, FeedItem{
			Title:       entry.Title,
			Link:        link,
			Author:      entry.Author.Name,
			GUID:        entry.ID,
			Description: content,
		})
	}
	return items
}

func (b *Bot) sendFeedUpdate(ctx context.Context, sub *Subscription, item FeedItem) error {
	title := html.UnescapeString(item.Title)
	feedTitle := html.UnescapeString(sub.FeedInfo.Title)

	var messageText strings.Builder
	messageText.WriteString(fmt.Sprintf("<b><u>%s</u></b>\n\n", escapeHTML(title)))
	messageText.WriteString("via ")

	if sub.FeedInfo.Link != "" {
		messageText.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>", sub.FeedInfo.Link, escapeHTML(feedTitle)))
	} else {
		messageText.WriteString(escapeHTML(feedTitle))
	}

	if item.Author != "" {
		messageText.WriteString(fmt.Sprintf(" (author: %s)", escapeHTML(item.Author)))
	}

	messageText.WriteString("\n")
	messageText.WriteString(fmt.Sprintf("<a href=\"%s\">%s</a>", item.Link, item.Link))

	_, err := b.bot.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    sub.ChatID,
		Text:      messageText.String(),
		ParseMode: models.ParseModeHTML,
		LinkPreviewOptions: &models.LinkPreviewOptions{
			URL: &item.Link,
		},
	})

	if err != nil {
		log.Printf("Failed to send message to chat %d: %v", sub.ChatID, err)
	}

	return err
}

func escapeHTML(s string) string {
	return html.EscapeString(s)
}