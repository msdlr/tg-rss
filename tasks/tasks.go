package tasks

import (
	"tg-rss/config"
	"tg-rss/external/db"
	"tg-rss/external/rss"
	"tg-rss/external/tg"
	"time"
)

func InitDatabase() {
	db.InitDB("db/db.sqlite")
}

func StartTasks() {
	// Read database
	config.LoadConfig()
	InitDatabase()

	// Start Telegram bot
	go tg.Start()
	time.Sleep(1 * time.Second)

	// RSS
	rss.InitFeedParser()
	go func() {
		ticker := time.NewTicker(config.GetUpdatePeriod())
		defer ticker.Stop()

		for {
			messages := rss.ReadAllFeeds()
			for _, message := range messages {
				tg.SendMessageHTML(message.User, message.FormattedMessage)
			}

			<-ticker.C
		}
	}()
	select {}
}
