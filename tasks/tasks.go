package tasks

import (
	"log"
	"tg-rss/config"
	"tg-rss/external/db"
	"tg-rss/external/rss"
	"tg-rss/external/tg"
	"tg-rss/info"
	"time"
)

func InitDatabase() {
	db.InitDB("db/db.sqlite")
}

func StartTasks() {
	log.Printf("tg-rss version %s.%s (%s %s)\n", info.GetTag(), info.GetSubversion(), info.GetCommit(), info.GetDate())
	// Read config
	config.LoadConfig()

	// Initialize database
	InitDatabase()

	go func() {
		ticker := time.NewTicker(config.GetBackupPeriod())
		defer ticker.Stop()

		for {
			<-ticker.C
			bkPath := "db/" + time.Now().Format("0601021504") + ".sqlite"
			db.Backup("db/db.sqlite", bkPath)
		}
	}()

	// Start Telegram bot
	go tg.Start()
	time.Sleep(1 * time.Second)

	// RSS
	rss.InitFeedParser()
	go func() {
		ticker := time.NewTicker(config.GetUpdatePeriod())
		defer ticker.Stop()

		// Wait until the time is a multiple of the update period
		time.Sleep(time.Until(time.Now().Truncate(config.GetUpdatePeriod()).Add(config.GetUpdatePeriod())))

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
