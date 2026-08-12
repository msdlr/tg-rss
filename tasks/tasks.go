package tasks

import (
	"log"
	"time"

	"github.com/msdlr/tg-rss/config"
	"github.com/msdlr/tg-rss/external/db"
	"github.com/msdlr/tg-rss/external/rss"
	"github.com/msdlr/tg-rss/external/tg"
	"github.com/msdlr/tg-rss/info"
)

func InitDatabase() {
	db.InitDB("db/db.sqlite")
}

func StartTasks() {
	log.Printf("tg-rss version %s (%s)\n", info.GetHead(), info.GetDate())
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
		// Wait until the time is a multiple of the update period
		time.Sleep(time.Until(time.Now().Truncate(config.GetUpdatePeriod()).Add(config.GetUpdatePeriod())))

		ticker := time.NewTicker(config.GetUpdatePeriod())
		defer ticker.Stop()

		for {
			messages := rss.ReadAllFeeds()
			for _, message := range messages {
				for _, msg := range message.FormattedMessages {
					tg.SendMessageHTML(message.User, msg)
				}
			}

			<-ticker.C
		}
	}()
	select {}
}
