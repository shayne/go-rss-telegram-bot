# go-rss-telegram-bot

A Telegram bot that manages RSS/Atom feed subscriptions and sends updates to users.

## Tech Stack
- Go 1.24.4
- github.com/go-telegram/bot - Telegram bot API
- golang.org/x/net/html - HTML parsing
- tailscale.com/cmd/viewer - Generate read-only views (build-time only)
- JSON file persistence

## Project Structure
- `cmd/rssbot/` - Main bot executable
- `pkg/rssbot/` - Bot implementation
  - `database.go` - JSON persistence with viewer support
  - `handlers.go` - Telegram command handlers
  - `feed.go` - RSS/Atom feed parsing
  - `message.go` - Message formatting
  - `rssbot.go` - Core bot logic

## Commands
- `/start` - Welcome message
- `/help` - Show available commands
- `/sub <url>` - Subscribe to feed (supports root URLs)
- `/unsub <search>` - Unsubscribe with fuzzy search
- `/feeds` - List subscribed feeds

## Key Features
- Supports both RSS and Atom feeds
- HTML page parsing to find feed URLs
- Periodic feed checking (configurable interval)
- Error tracking with configurable retry attempts
- Chat ID-based access control
- Link previews for post URLs

## Development Commands
- `go mod tidy` - Clean up dependencies
- `make build` - Build the bot
- `make build-linux-amd64` - Build for Linux AMD64
- `make test` - Run tests
- `make check` - Build and run staticcheck
- `make goimports` - Format code
- `make llm` - Run all checks (check, goimports, test)

## Configuration
Environment variables:
- `TELEGRAM_API_KEY` - Required bot token

CLI flags:
- `-db` - Database file path (default: db.json)
- `-check-interval` - Feed check interval (default: 1h)
- `-allowed-chats` - Comma-separated allowed chat IDs

## Important Notes
- Empty feed titles are replaced with "Untitled Feed"
- Feed errors are tracked and persisted
- Uses viewer pattern for immutable data access
- Tests included for database and feed parsing