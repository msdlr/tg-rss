package main

import (
	"log"
	"tg-rss/config"
	"tg-rss/external/db"
	"tg-rss/external/rss"
	"tg-rss/external/tg"
	"time"
)

func rssLoop() {
	rss.InitFeedParser()

	for {
		rss.SetlastQuery()
		wTimeStart := time.Now()

		users, err := db.GetAllUsers()

		if err != nil {
			log.Println("Error fetching users:", err)
		}

		for _, user := range users {
			arts := rss.GetArticlesForUser(user.ChatID, 0)

			if len(arts) > 0 {
				tg.SendMessageHTML(user.ChatID, rss.FormatNewsHTML(arts))
			}
		}
		wTimeEnd := time.Now()

		log.Println("Read all feeds in " + (wTimeEnd.Sub(wTimeStart)).String())

		time.Sleep(config.GetUpdatePeriod())
	}
}

func main() {
	config.LoadConfig()
	db.InitDB("db/db.sqlite")

	go tg.Start()
	time.Sleep(1 * time.Second)
	go rssLoop()

	select {}
}
