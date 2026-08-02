package tasks

import (
	"tg-rss/config"
	"tg-rss/external/db"
	"tg-rss/external/rss"
	"tg-rss/external/tg"
	"tg-rss/stats"
	"time"
)

func InitDatabase() {
	db.InitDB("db/db.sqlite")
}

func StartTasks() {
	stats.SetStartUpTime(time.Now())
	// Read database
	config.LoadConfig()
	InitDatabase()

	// Start Telegram bot
	go tg.Start()
	time.Sleep(1 * time.Second)

	// RSS
	rss.InitFeedParser()
	timesLooped := 0
	go func() {
		ticker := time.NewTicker(config.GetUpdatePeriod())
		defer ticker.Stop()

		for {
			if timesLooped != 0 {
				messages := rss.ReadAllFeeds()
				for _, message := range messages {
					tg.SendMessageHTML(message.User, message.FormattedMessage)
				}
			}
			timesLooped++

			<-ticker.C
		}
	}()
	select {}
}
