# Telegram RSS bot
A Telegram bot written in Golang that periodically fetches RSS feeds that users subscribe to. The external libraries used are the following:

- [go-sqlite](https://github.com/mattn/go-sqlite3) to store subscriptions persistently using SQLite.
- [gofeed](https://github.com/mmcdole/gofeed) to fetch and read RSS/Atom feeds.
- [go-telegram/bot](https://github.com/go-telegram/bot) to handle and send messages to and from Telegram.

## Available Commands

- **`/start`** – Show help and usage.
- **`/sub <rss_url>`** – Subscribe to an RSS feed.
- **`/subbsky <handle>`** – Subscribe to a Bluesky.social profile (must allow posts for non-logged users).
- **`/subyt <channel_url>`** – Subscribe to the RSS feed of a YouTube channel.
- **`/unsub <rss_url>`** – Remove an RSS subscription.
- **`/list`** – List your current RSS subscriptions.
- **`/latest`** – Show the latest **N** articles from each subscribed feed.
- **`/timing`** – Display the last poll time and the next scheduled poll.
- **`/pull`** – Force an immediate feed update and return any new articles.

## Disclaimer: Use of Generative AI in Code Development
This codebase includes contributions that were partially generated or assisted by generative AI chatbots (such as OpenAI's ChatGPT, DeepSeek), but the AI was used only for specific segments (boilerplate code, routine snippets, or initial scaffolding) and not for the entirety of the project. The core architecture, logic, and validation remain human-authored and reviewed. 
