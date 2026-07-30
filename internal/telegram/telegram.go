package telegram

import (
	"tg-rss/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var updates tgbotapi.UpdatesChannel // We query this object using range to get messages received by the bot

func Start() (upChan tgbotapi.UpdatesChannel) {
	bot, err := tgbotapi.NewBotAPI(config.Config.TelegramToken)
	if err != nil {
		panic(err)
	}

	updateConfig := tgbotapi.NewUpdate(0)

	updateConfig.Timeout = 30

	// Start polling Telegram for updates.
	upChan, chanErr := bot.GetUpdatesChan(updateConfig)

	if chanErr != nil {
		panic(err)
	}

	return upChan
}
