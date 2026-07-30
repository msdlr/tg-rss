package telegram

import (
	"tg-rss/config"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api"
)

var upChan tgbotapi.UpdatesChannel
var bot *tgbotapi.BotAPI

func Start() {
	var err error
	bot, err = tgbotapi.NewBotAPI(config.Configs.TelegramToken)
	if err != nil {
		panic(err)
	}

	updateConfig := tgbotapi.NewUpdate(0)

	updateConfig.Timeout = 30

	// Start polling Telegram for updates.
	c, chanErr := bot.GetUpdatesChan(updateConfig)

	upChan = c

	if chanErr != nil {
		panic(err)
	}
}

func QueryLoop() {

	for update := range upChan {
		if update.Message == nil { // ignore any non-Message updates
			continue
		}

		if !update.Message.IsCommand() { // ignore any non-command Messages
			continue
		}

		switch update.Message.Command() {
		case "start":
			handleStartCommand(*update.Message)
		}
	}

}

func handleStartCommand(inMsg tgbotapi.Message) {
	outMsg := tgbotapi.NewMessage(inMsg.Chat.ID, "")
	outMsg.Text = "/start called"
	bot.Send(outMsg)
}
