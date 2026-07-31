package main

import (
	"log"
	"tg-rss/config"
	"tg-rss/db"
	"tg-rss/rss"
	"tg-rss/tg"
	"time"

	"github.com/mmcdole/gofeed"
)

func rssLoop() {
	timeDelta := time.Duration(uint64(config.Configs.UpdatePeriodMinutes)) * time.Minute
	rss.FeedParser = gofeed.NewParser()

	for {
		log.Println("Reading RSS feeds...")

		users, err := db.GetAllUsers()

		if err != nil {
			log.Println("Error fetching users:", err)
		}

		for _, user := range users {
			arts := rss.GetArticlesForUser(user.ChatID)

			if len(arts) > 0 {
				tg.SendMessage(user.ChatID, rss.FormatNews(arts))
			}
		}

		time.Sleep(timeDelta)
	}
}

func main() {
	config.LoadConfig()
	db.InitDB("db.sqlite")

	go tg.Start()
	time.Sleep(1 * time.Second)
	go rssLoop()

	select {}
}
