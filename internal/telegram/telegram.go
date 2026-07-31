package telegram

import (
	"context"
	"log"
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
	b.RegisterHandler(bot.HandlerTypeMessageText, "/addSub", bot.MatchTypePrefix, handleAddCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/rmSub", bot.MatchTypeExact, handleRmCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/listSubs", bot.MatchTypeExact, handleListCommand)

	b.Start(ctx)
}

func handleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Hello, *" + bot.EscapeMarkdown(update.Message.From.FirstName) + "*",
		ParseMode: models.ParseModeMarkdown,
	})
}

func handleAddCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	textContent := update.Message.Text

	if len(strings.Fields(textContent)) == 1 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "Use /addSub `url`",
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	url := strings.Fields(textContent)[1]

	feedTitle, err := rss.GetRSSFeedTitle(url)

	if err != nil {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "Error retrieving feed title for ``" + url + "``",
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	// Add the chatID to the database
	if update.Message.Chat.Type == "private" {
		err := db.AddUser(update.Message.Chat.ID, update.Message.Chat.Username)
		if err != nil {
			log.Println("Error saving user")
			return
		}
	} else {
		err := db.AddUser(update.Message.Chat.ID, update.Message.Chat.Title)
		if err != nil {
			log.Println("Error group chat")
			return
		}
	}

	// Add the feed to the database
	feedID, feedErr := db.AddFeed(url, feedTitle)
	if feedErr != nil {
		log.Println("Error adding feed")
		return
	}

	// Cross the user and the feed
	subErr := db.Subscribe(update.Message.Chat.ID, feedID)

	if subErr != nil {
		log.Println("Error adding subscription")
		return
	}

	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Sucessfully subbed",
		ParseMode: models.ParseModeMarkdown,
	})
}

func handleRmCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Hello, *" + bot.EscapeMarkdown(update.Message.From.FirstName) + "*",
		ParseMode: models.ParseModeMarkdown,
	})
}

func handleListCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      "Hello, *" + bot.EscapeMarkdown(update.Message.From.FirstName) + "*",
		ParseMode: models.ParseModeMarkdown,
	})
}
