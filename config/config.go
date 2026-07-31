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

	telegramToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	period, err := strconv.Atoi(os.Getenv("UPDATE_PERIOD_MINUTES"))
	if err != nil {
		period = 30
	}
	updatePeriod = time.Duration(period) * time.Minute
}
