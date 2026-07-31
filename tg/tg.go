package tg

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"tg-rss/config"
	"tg-rss/db"
	"tg-rss/rss"

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

	var err error
	b, err = bot.New(config.GetTgToken(), opts...)
	if err != nil {
		panic(err)
	}

	// Register commands shown in the Telegram client.
	_, err = b.SetMyCommands(ctx, &bot.SetMyCommandsParams{
		Commands: []models.BotCommand{
			{
				Command:     "start",
				Description: "Show help and usage",
			},
			{
				Command:     "sub",
				Description: "Subscribe to an RSS feed",
			},
			{
				Command:     "unsub",
				Description: "Remove an RSS subscription",
			},
			{
				Command:     "list",
				Description: "List your RSS subscriptions",
			},
			{
				Command:     "latest",
				Description: "Get the latest articles from your subscriptions",
			},
		},
	})
	if err != nil {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handleStartCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/sub", bot.MatchTypePrefix, handleAddCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/unsub", bot.MatchTypePrefix, handleRmCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/list", bot.MatchTypeExact, handleListCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/latest", bot.MatchTypeExact, handleLatestCommand)

	b.Start(ctx)
}

func handleLatestCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	user := update.Message.Chat.ID
	arts := rss.GetArticlesForUser(user, 3)

	if len(arts) > 0 {
		SendMessageHTML(update.Message.Chat.ID, rss.FormatNewsHTML(arts))
	}
}

func handleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	SendMessageMarkdown(ctx, b, update, "Hello, *"+bot.EscapeMarkdown(update.Message.From.FirstName)+"*")
}

func handleAddCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	textContent := update.Message.Text

	if len(strings.Fields(textContent)) == 1 {
		SendMessageMarkdown(ctx, b, update, "Use /sub `url`")
		return
	}

	url := strings.Fields(textContent)[1]

	feedTitle, err := rss.GetRSSFeedTitle(url)

	if err != nil {
		SendMessageMarkdown(ctx, b, update, "Error retrieving feed title for ``"+url+"``")
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

	SendMessageMarkdown(ctx, b, update, "Subscribed")
}

func handleRmCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	textContent := update.Message.Text

	if len(strings.Fields(textContent)) == 1 {
		SendMessageMarkdown(ctx, b, update, "Use /unsub `url`")
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

	SendMessageMarkdown(ctx, b, update, msg)
}

func handleListCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	feeds, _ := db.GetUserFeeds(update.Message.Chat.ID)

	if len(feeds) == 0 {
		SendMessageMarkdown(ctx, b, update, "No subscriptions found")
		return

	}

	var msg string = "Your feeds (" + strconv.Itoa(len(feeds)) + "):\n"

	for _, f := range feeds {
		msg += "- " + f.Title + " (" + f.URL + ")\n"
	}

	SendMessageMarkdown(ctx, b, update, msg)
}

func SendMessageMarkdown(ctx context.Context, b *bot.Bot, update *models.Update, msg string) {
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    update.Message.Chat.ID,
		Text:      msg,
		ParseMode: models.ParseModeMarkdown,
	})

	if err != nil {
		log.Println("Failed to send message")
		log.Println(err)
	}
}

func SendMessageHTML(chatID int64, msg string) {
	// Check if bot is initialized
	if b == nil {
		log.Println("Bot not initialized")
		return
	}

	// Use context.Background() with a timeout
	ctx := context.Background()

	// Send the message
	_, err := b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID:    chatID,
		Text:      msg,
		ParseMode: models.ParseModeHTML,
	})

	if err != nil {
		log.Printf("Failed to send message to chat %d: %v", chatID, err)
		return
	}
}
