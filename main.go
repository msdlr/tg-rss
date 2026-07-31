package main

import (
	"log"
	"tg-rss/config"
	"tg-rss/db"
	"tg-rss/rss"
	"tg-rss/tg"
	"time"
)

func rssLoop() {
	rss.InitFeedParser()

	for {
		log.Println("Reading RSS feeds...")

		users, err := db.GetAllUsers()

		if err != nil {
			log.Println("Error fetching users:", err)
		}

		for _, user := range users {
			arts := rss.GetArticlesForUser(user.ChatID, 0)

			if len(arts) > 0 {
				tg.SendMessage(user.ChatID, rss.FormatNewsHTML(arts))
			}
		}

		time.Sleep(config.GetUpdatePeriod())
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
