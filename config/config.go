package config

import (
	"log"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

var Configs Config

type Config struct {
	TelegramToken       string
	UpdatePeriodMinutes uint8
}

func LoadConfig() {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	Configs.TelegramToken = os.Getenv("TELEGRAM_BOT_TOKEN")
	period, err := strconv.Atoi(os.Getenv("UPDATE_PERIOD_MINUTES"))
	if err != nil {
		// Handle error - environment variable not set or invalid
		period = 5 // default value
	} else {
		Configs.UpdatePeriodMinutes = uint8(period)
	}
}
