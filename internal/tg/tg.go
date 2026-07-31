package tg

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"tg-rss/config"
	"tg-rss/internal/db"
	"tg-rss/internal/rss"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var b *bot.Bot

func Start() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(handleStartCommand),
	}

	botfatherAPI := config.Configs.TelegramToken

	var err error
	b, err = bot.New(botfatherAPI, opts...)
	if nil != err {
		// panics for the sake of simplicity.
		// you should handle this error properly in your code.
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handleStartCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/addSub", bot.MatchTypePrefix, handleAddCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/rmSub", bot.MatchTypePrefix, handleRmCommand)
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

	textContent := update.Message.Text

	if len(strings.Fields(textContent)) == 1 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "Use /rmSub `url`",
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	feedID, getFeedErr := db.GetFeedID(strings.Fields(textContent)[1])

	if getFeedErr != nil {
		log.Println("Error querying the database for feed")
	}

	unsubErr := db.Unsubscribe(update.Message.Chat.ID, feedID)

	if unsubErr != nil {
		log.Println("Error updating the db for subsubbing")
	}

	feed, _ := db.GetFeedByURL(strings.Fields(textContent)[1])

	msg := "Sucessfully removed " + feed.Title + " (" + strings.Fields(textContent)[1] + ") from the list"
	strings.ReplaceAll(msg, "\n", "")

	_, sendErr := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      bot.EscapeMarkdown(msg),
		ParseMode: models.ParseModeMarkdown,
	})

	if sendErr != nil {
		log.Println("Error sending message to user")
	}
}

func handleListCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	feeds, _ := db.GetUserFeeds(update.Message.Chat.ID)

	if len(feeds) == 0 {
		b.SendMessage(ctx, &bot.SendMessageParams{
			ChatID:    update.Message.Chat.ID,
			Text:      "No subscriptions found",
			ParseMode: models.ParseModeMarkdown,
		})
		return
	}

	var msg string = "Your feeds (" + strconv.Itoa(len(feeds)) + "):\n"

	for _, f := range feeds {
		msg += "- " + f.Title + " (" + f.URL + ")\n"
	}

	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      bot.EscapeMarkdown(msg),
		ParseMode: models.ParseModeMarkdown,
	})

	if err != nil {
		log.Println("Failed to send message")
		log.Println(err)
	}
}

func SendMessage(chatID int64, msg string) {
	// Check if bot is initialized
	if b == nil {
		log.Println("Bot not initialized")
		return
	}

	// Use context.Background() with a timeout
	ctx := context.Background()

	// Send the message
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: chatID,
		Text:   msg,
	})

	if err != nil {
		log.Printf("Failed to send message to chat %d: %v", chatID, err)
		return
	}
}
