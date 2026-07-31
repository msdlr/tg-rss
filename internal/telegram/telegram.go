package telegram

import (
	"context"
	"os"
	"os/signal"
	"tg-rss/config"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

func Start() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handleStartCommand),
	}

	botfatherAPI := config.Configs.TelegramToken

	b, err := bot.New(botfatherAPI, opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handleStartCommand)

	b.Start(ctx)
}

func handleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Hello, *" + bot.EscapeMarkdown(update.Message.From.FirstName) + "*",
		ParseMode: models.ParseModeMarkdown,
	})
}
