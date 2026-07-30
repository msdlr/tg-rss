package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	TelegramToken       string
	UpdatePeriodMinutes uint
}

func LoadConfig() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found, using environment variables")
	}

	return &Config{
		TelegramToken: os.Getenv("TELEGRAM_BOT_TOKEN"),
	}
}
