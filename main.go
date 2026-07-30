package main

import (
	"tg-rss/config"
	"tg-rss/internal/telegram"
)

func main() {
	config.LoadConfig()

	telegram.Start()
	telegram.QueryLoop()
}
