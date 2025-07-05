# go-rss-telegram-bot

A Telegram bot for managing RSS/Atom feed subscriptions.

🤖 100% written by Claude Code

## Installation

```bash
go install github.com/shayne/go-rss-telegram-bot/cmd/rssbot@latest
```

## Usage

```bash
export TELEGRAM_API_KEY=your_bot_token
rssbot -db=feeds.json -check-interval=30m -allowed-chats=123456789
```

## Commands

- `/sub <url>` - Subscribe to a feed
- `/unsub <search>` - Unsubscribe from a feed
- `/feeds` - List your feeds
- `/help` - Show help

## Configuration

| Flag | Default | Description |
|------|---------|-------------|
| `-db` | `db.json` | Database file path |
| `-check-interval` | `1h` | Feed check interval |
| `-allowed-chats` | (empty) | Comma-separated chat IDs |

## Building

```bash
make build
```

## License

[MIT](LICENSE) © 2025 Shayne Sweeney