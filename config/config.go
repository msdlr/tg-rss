package config

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

var updatePeriod time.Duration = time.Hour
var telegramToken string = "your-token-here"
var maxOldArticles uint = 3

func GetMaxOldArticles() uint {
	return maxOldArticles
}

func GetUpdatePeriod() time.Duration {
	return updatePeriod
}

func GetTgToken() string {
	return telegramToken
}

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	// Bot token
	telegramToken = os.Getenv("TELEGRAM_BOT_TOKEN")

	// Update period
	periodStr := os.Getenv("UPDATE_PERIOD_MINUTES")
	var period int
	if periodStr == "" {
		periodStr = "30"
	}
	period, _ = strconv.Atoi(periodStr)
	updatePeriod = time.Duration(period) * time.Minute

	// Max old messages when calling /latest
	maxOlddStr := os.Getenv("MAX_OLD_ARTICLES")
	if maxOlddStr == "" {
		maxOlddStr = strconv.Itoa(int(maxOldArticles))
	}
	old, _ := strconv.Atoi(maxOlddStr)
	maxOldArticles = uint(old)
}
