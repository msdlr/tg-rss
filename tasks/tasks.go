package tasks

import (
	"log"
	"tg-rss/config"
	"tg-rss/external/db"
	"tg-rss/external/rss"
	"tg-rss/external/tg"
	"time"
)

func InitDatabase() {
	db.InitDB("db/db.sqlite")
}

func InitTelegramBot() {

}

func InitRSSLoop() {
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

func StartTasks() {
	config.LoadConfig()
	InitDatabase()

	go tg.Start()
	time.Sleep(1 * time.Second)
	go InitRSSLoop()
	select {}
}
