package tg

import (
	"context"
	"fmt"
	"html"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"tg-rss/config"
	"tg-rss/external/db"
	"tg-rss/external/rss"
	"time"

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
				Command:     "subyt",
				Description: "Subscribe to an YouTube channel",
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
				Description: "Get the latest " + strconv.Itoa(int(config.GetMaxOldArticles())) + " articles from each subscription",
			},
			{
				Command:     "timing",
				Description: "Get the time for the last and next query",
			},
			{
				Command:     "pull",
				Description: "Get the updates now (last " + config.GetUpdatePeriod().String() + ")",
			},
		},
	})
	if err != nil {
		panic(err)
	}

	b.RegisterHandler(bot.HandlerTypeMessageText, "/start", bot.MatchTypeExact, handleStartCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/help", bot.MatchTypeExact, handleStartCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/sub ", bot.MatchTypePrefix, handleSubCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/unsub", bot.MatchTypePrefix, handleUnsubCommand)
	b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "unsub:", bot.MatchTypePrefix, handleUnsubCallback)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/list", bot.MatchTypeExact, handleListCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/latest", bot.MatchTypeExact, handleLatestCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/timing", bot.MatchTypeExact, handleTimingCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/pull", bot.MatchTypeExact, handlePullCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/subyt", bot.MatchTypePrefix, handleSubYTCommand)

	b.Start(ctx)
}

func handleSubYTCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if update.Message.Text == "/subyt" {
		SendMessageMarkdown(update.Message.Chat.ID, "Usage: /subyt <channel link>")
		return
	}

	channelURL := strings.Fields(update.Message.Text)[1]
	feedURL, err := rss.GetYouTubeRSS(channelURL)
	if err != nil {
		SendMessageMarkdown(update.Message.Chat.ID, "Error retrieving RSS feed from channel")
	} else {
		update.Message.Text = strings.Replace(update.Message.Text, channelURL, feedURL, 1)
		handleSubCommand(ctx, b, update)
	}
}

func handlePullCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	msg := rss.FormatNewsHTML(rss.GetArticlesForUser(update.Message.Chat.ID, 0))

	if msg == "" {
		msg = "No news for " + config.GetUpdatePeriod().String()
	}

	SendMessageHTML(update.Message.Chat.ID, msg)
}

func handleTimingCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	// lastRead := bot.EscapeMarkdown(rss.GetlastQuery().Format("2006/01/02 15:04:05"))
	diff := bot.EscapeMarkdown((time.Since(rss.GetlastQuery())).String())
	next := bot.EscapeMarkdown(time.Duration((time.Until(rss.GetlastQuery().Add(config.GetUpdatePeriod())))).String())
	msg := "Feeds last read " + diff + " ago, next in " + next
	SendMessageMarkdown(update.Message.Chat.ID, msg)
}

func handleLatestCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	user := update.Message.Chat.ID
	arts := rss.GetArticlesForUser(user, config.GetMaxOldArticles())

	if len(arts) > 0 {
		SendMessageHTML(update.Message.Chat.ID, rss.FormatNewsHTML(arts))
	}
}

func handleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	helpMessage := `Available commands:

• <b>/start</b> — Show this help message
• <b>/sub</b> <code>RSS_URL</code> — Subscribe to an RSS feed
• <b>/unsub</b> <code>RSS_URL</code> — Remove an RSS subscription
• <b>/list</b> — List your RSS subscriptions
• <b>/latest</b> — Get the latest articles from your subscriptions`

	SendMessageHTML(update.Message.Chat.ID, helpMessage)
}

func handleSubCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	textContent := update.Message.Text

	if len(strings.Fields(textContent)) == 1 {
		SendMessageMarkdown(update.Message.Chat.ID, "Use /sub `url`")
		return
	}

	for _, url := range strings.Fields(textContent)[1:] {

		feedTitle, err := rss.GetRSSFeedTitle(url)

		if err != nil {
			SendMessageMarkdown(update.Message.Chat.ID, "Error retrieving feed title for ``"+url+"``")
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

		msg := fmt.Sprintf(
			`Subscribed to <a href="%s">%s</a>`,
			html.EscapeString(url),
			html.EscapeString(feedTitle),
		)

		SendMessageHTML(update.Message.Chat.ID, msg)
	}
}

func handleUnsubCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	textContent := update.Message.Text

	if len(strings.Fields(textContent)) == 1 {
		handleEmptyUnsubCommand(ctx, b, update)
		return
	}

	feedID, getFeedErr := db.GetFeedID(strings.Fields(textContent)[1])

	if getFeedErr != nil {
		log.Println("Error querying the database for feed")
	}

	feed, _ := db.GetFeedByURL(strings.Fields(textContent)[1])

	if feed != nil {

		unsubErr := db.Unsubscribe(update.Message.Chat.ID, feedID)

		if unsubErr != nil {
			log.Println("Error updating the db for subsubbing")
		}

		var sb strings.Builder
		fmt.Fprintf(
			&sb,
			"Unsuscribed from <a href=\"%s\">%s</a>\n",
			html.EscapeString(feed.URL),
			html.EscapeString(feed.Title),
		)

		SendMessageHTML(update.Message.Chat.ID, sb.String())
	} else {
		SendMessageMarkdown(update.Message.Chat.ID, "Not subscribed")
	}
}

func handleListCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	feeds, _ := db.GetUserFeeds(update.Message.Chat.ID)

	if len(feeds) == 0 {
		SendMessageMarkdown(update.Message.Chat.ID, "No subscriptions found")
		return

	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>Your feeds (%d):</b>\n", len(feeds))

	for _, f := range feeds {
		fmt.Fprintf(
			&sb,
			"• <a href=\"%s\">%s</a>\n",
			html.EscapeString(f.URL),
			html.EscapeString(f.Title),
		)
	}

	SendMessageHTML(update.Message.Chat.ID, sb.String())
}

func handleEmptyUnsubCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	feeds, err := db.GetUserFeeds(update.Message.Chat.ID)
	if err != nil {
		log.Println("Error getting user feeds:", err)
		return
	}

	if len(feeds) == 0 {
		SendMessageMarkdown(update.Message.Chat.ID, "No subscriptions found")
		return
	}

	keyboard := make([][]models.InlineKeyboardButton, 0, len(feeds))

	for _, feed := range feeds {
		keyboard = append(keyboard, []models.InlineKeyboardButton{
			{
				Text:         feed.Title,
				CallbackData: fmt.Sprintf("unsub:%d", feed.ID),
			},
		})
	}

	_, err = b.SendMessage(ctx, &bot.SendMessageParams{
		ChatID: update.Message.Chat.ID,
		Text:   "Select a feed to unsubscribe:",
		ReplyMarkup: &models.InlineKeyboardMarkup{
			InlineKeyboard: keyboard,
		},
	})
	if err != nil {
		log.Println("Error sending unsubscribe buttons:", err)
	}
}

func handleUnsubCallback(ctx context.Context, b *bot.Bot, update *models.Update) {
	message := update.CallbackQuery.Message.Message
	if message == nil {
		log.Println("Callback message is inaccessible")
		return
	}

	chatID := message.Chat.ID

	feedID, err := strconv.ParseInt(
		strings.TrimPrefix(update.CallbackQuery.Data, "unsub:"),
		10,
		64,
	)
	if err != nil {
		log.Println("Invalid unsubscribe callback:", err)
		return
	}

	feeds, err := db.GetUserFeeds(chatID)
	if err != nil {
		log.Println("Error getting user feeds:", err)
		return
	}

	for _, feed := range feeds {
		if feed.ID != feedID {
			continue
		}

		// Unsubscribe
		if err := db.Unsubscribe(chatID, feed.ID); err != nil {
			log.Println("Error removing subscription:", err)
			return
		}

		// Rebuild keyboard removing the selected feed
		var keyboard [][]models.InlineKeyboardButton
		for _, f := range feeds {
			if f.ID == feedID {
				continue
			}

			keyboard = append(keyboard, []models.InlineKeyboardButton{
				{
					Text:         f.Title,
					CallbackData: "unsub:" + strconv.FormatInt(f.ID, 10),
				},
			})
		}

		_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
			ChatID:    chatID,
			MessageID: message.ID,
			ReplyMarkup: &models.InlineKeyboardMarkup{
				InlineKeyboard: keyboard,
			},
		})
		if err != nil {
			log.Println("Failed to update keyboard:", err)
		}

		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		msg := fmt.Sprintf(`Unsubscribed from <a href="%s">%s</a>`, feed.URL, feed.Title)

		SendMessageHTML(message.Chat.ID, msg)
		return
	}

	SendMessageMarkdown(chatID, "Feed not found")
}

func SendMessageMarkdown(chatID int64, msg string) {
	_, err := b.SendMessage(context.Background(), &bot.SendMessageParams{
		ChatID:    chatID,
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
