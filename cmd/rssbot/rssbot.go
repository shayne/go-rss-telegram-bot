package main

import (
	"context"
	"flag"
	"log"
	"os"
	"os/signal"
	"strings"
	"time"
	"github.com/shayne/go-rss-telegram-bot/pkg/rssbot"
)

func main() {
	var (
		dbPath       = flag.String("db", "db.json", "Path to the database JSON file")
		checkInterval = flag.Duration("check-interval", time.Hour, "Interval between RSS feed checks")
		allowedChats = flag.String("allowed-chats", "", "Comma-separated list of allowed Telegram chat IDs")
	)
	flag.Parse()

	apiKey := os.Getenv("TELEGRAM_API_KEY")
	if apiKey == "" {
		log.Fatal("TELEGRAM_API_KEY environment variable is required")
	}

	allowList := []string{}
	if *allowedChats != "" {
		allowList = strings.Split(*allowedChats, ",")
		for i := range allowList {
			allowList[i] = strings.TrimSpace(allowList[i])
		}
	}

	log.Printf("Starting RSS bot with database at %s", *dbPath)
	log.Printf("Check interval: %v", *checkInterval)
	if len(allowList) > 0 {
		log.Printf("Allowed chat IDs: %v", allowList)
	} else {
		log.Printf("No chat restrictions (allow list is empty)")
	}

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	cfg := &rssbot.Config{
		DBPath:        *dbPath,
		CheckInterval: *checkInterval,
		AllowedChatIDs: allowList,
	}

	rssBot, err := rssbot.New(apiKey, cfg)
	if err != nil {
		log.Fatalf("Failed to create bot: %v", err)
	}

	if err := rssBot.Run(ctx); err != nil {
		log.Fatalf("Bot error: %v", err)
	}

	log.Println("Bot stopped gracefully")
}
