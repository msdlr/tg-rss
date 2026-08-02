# Telegram RSS bot
A Telegram bot written in Golang that periodically fetches RSS feeds that users subscribe to. The external libraries used are the following:

- [go-sqlite](https://github.com/mattn/go-sqlite3) to store subscriptions persistently using SQLite.
- [gofeed](https://github.com/mmcdole/gofeed) to fetch and read RSS/Atom feeds.
- [go-telegram/bot](https://github.com/go-telegram/bot) to handle and send messages to and from Telegram.

## Available Commands

- **`/start`** – Show help and usage.
- **`/sub <rss_url>`** – Subscribe to an RSS feed.
- **`/unsub <rss_url>`** – Remove an RSS subscription.
- **`/list`** – List your current RSS subscriptions.
- **`/latest`** – Show the latest **N** articles from each subscribed feed.
- **`/timing`** – Display the last poll time and the next scheduled poll.
- **`/pull`** – Force an immediate feed update and return any new articles.
