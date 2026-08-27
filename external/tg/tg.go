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
	"tg-rss/info"
	"time"

	"github.com/go-telegram/bot"
	"github.com/go-telegram/bot/models"
)

var b *bot.Bot

func Start() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	opts := []bot.Option{
		bot.WithDefaultHandler(func(ctx context.Context, b *bot.Bot, update *models.Update) {}),
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
				Command:     "help",
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
				Command:     "subtw",
				Description: "Subscribe to a public Twitter account",
			},
			{
				Command:     "subbsky",
				Description: "Subscribe to a public Bluesky account",
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
	b.RegisterHandler(bot.HandlerTypeMessageText, "/subtw", bot.MatchTypePrefix, handleSubTwitterCommand)
	b.RegisterHandler(bot.HandlerTypeMessageText, "/subbsky", bot.MatchTypePrefix, handleSubBskyCommand)

	botHandle := config.GetTelegramBotHandle()

	if botHandle != "" {
		b.RegisterHandler(bot.HandlerTypeMessageText, "/start@"+botHandle, bot.MatchTypeExact, handleStartCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/help@"+botHandle, bot.MatchTypeExact, handleStartCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/sub@"+botHandle, bot.MatchTypePrefix, handleSubCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/unsub@"+botHandle, bot.MatchTypePrefix, handleUnsubCommand)
		b.RegisterHandler(bot.HandlerTypeCallbackQueryData, "unsub:", bot.MatchTypePrefix, handleUnsubCallback)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/list@"+botHandle, bot.MatchTypeExact, handleListCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/latest@"+botHandle, bot.MatchTypeExact, handleLatestCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/timing@"+botHandle, bot.MatchTypeExact, handleTimingCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/pull@"+botHandle, bot.MatchTypeExact, handlePullCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/subyt@"+botHandle, bot.MatchTypePrefix, handleSubYTCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/subtw@"+botHandle, bot.MatchTypePrefix, handleSubTwitterCommand)
		b.RegisterHandler(bot.HandlerTypeMessageText, "/subbsky@"+botHandle, bot.MatchTypePrefix, handleSubBskyCommand)
	}

	b.Start(ctx)
}

func handleSubBskyCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if len(strings.Fields(update.Message.Text)) == 1 {
		SendMessageHTML(update.Message.Chat.ID, "Usage: /subbsky  <code>username</code>")
		return
	}

	input := strings.Fields(update.Message.Text)[1]
	feedURL, err := rss.GetBskyRSS(input)
	if err != nil {
		SendMessageMarkdown(update.Message.Chat.ID, "Error retrieving RSS feed")
	} else {
		update.Message.Text = strings.Replace(update.Message.Text, input, feedURL, 1)
		handleSubCommand(ctx, b, update)
	}
}

func handleSubTwitterCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if len(strings.Fields(update.Message.Text)) == 1 {
		SendMessageHTML(update.Message.Chat.ID, "Usage: /subtw  <code>username</code>")
		return
	}

	input := strings.Fields(update.Message.Text)[1]
	feedURL, err := rss.GetTwitterRSS(input)
	if err != nil {
		SendMessageMarkdown(update.Message.Chat.ID, "Error retrieving RSS feed")
	} else {
		update.Message.Text = strings.Replace(update.Message.Text, input, feedURL, 1)
		handleSubCommand(ctx, b, update)
	}
}

func handleSubYTCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	if len(strings.Fields(update.Message.Text)) == 1 {
		SendMessageHTML(update.Message.Chat.ID, "Usage: /subyt  <code>channel_url</code>")
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
	for _, msg := range rss.FormatNewsHTML(rss.GetArticlesForUser(update.Message.Chat.ID, 0)) {
		if msg == "" {
			msg = "No news for " + config.GetUpdatePeriod().String()
		}

		SendMessageHTML(update.Message.Chat.ID, msg)
	}
}

func handleTimingCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	lastRead := rss.GetlastQuery()
	nextRead := time.Now().Truncate(config.GetUpdatePeriod()).Add(config.GetUpdatePeriod())

	if lastRead.IsZero() {
		msg := "Feeds not read yet, will read at " + nextRead.Format("15:04")
		SendMessageMarkdown(update.Message.Chat.ID, bot.EscapeMarkdown(msg))
		return
	}

	msg := "Feeds last read at " + lastRead.Format("15:04") + ", next at " + nextRead.Format("15:04")
	SendMessageMarkdown(update.Message.Chat.ID, bot.EscapeMarkdown(msg))
}

func handleLatestCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	user := update.Message.Chat.ID
	arts := rss.GetArticlesForUser(user, config.GetMaxOldArticles())

	for _, msg := range rss.FormatNewsHTML(rss.GetArticlesForUser(update.Message.Chat.ID, 0)) {
		if len(arts) > 0 {
			SendMessageHTML(update.Message.Chat.ID, msg)
		}
	}
}

func handleStartCommand(ctx context.Context, b *bot.Bot, update *models.Update) {
	helpMessage := fmt.Sprintf(`tg-rss version %s (%s)
	
	<b>Available commands:</b>

• <b>/start</b> - Show this help message
• <b>/help</b> - Show this help message

• <b>/sub</b> <code>RSS_URL</code> - Subscribe to an RSS feed
• <b>/subyt</b> <code>CHANNEL_URL</code> - Subscribe to a YouTube channel
• <b>/subbsky</b> <code>USERNAME</code> - Subscribe to a Bluesky.social profile (posts must be visible to non-logged in)
• <b>/subtw</b> <code>USERNAME</code> - Subscribe to a Twitter public profile
• <b>/unsub</b> <code>RSS_URL</code> - Remove a subscription
• <b>/unsub</b> - Remove subscriptions (interactive)
• <b>/list</b> - List your subscriptions

• <b>/latest</b> - Show the latest %d articles from each subscription
• <b>/pull</b> - Check for updates now (last %s)
• <b>/timing</b> - Show the last and next scheduled update`, info.GetHead(), info.GetDate(), config.GetMaxOldArticles(), time.Duration(config.GetUpdatePeriod()).String())

	SendMessageHTML(update.Message.Chat.ID, helpMessage)
}

func handleSubCommand(ctx context.Context, b *bot.Bot, update *models.Update) {

	textContent := update.Message.Text

	if len(strings.Fields(textContent)) == 1 {
		SendMessageMarkdown(update.Message.Chat.ID, "Use /sub `url`")
		return
	}

	for _, url := range strings.Fields(textContent)[1:] {

		// feedTitle, err := rss.GetRSSFeedTitle(url)
		feedTitle, webURL, err := rss.GetRSSFeedInfo(url)

		if err != nil {
			SendMessageMarkdown(update.Message.Chat.ID, bot.EscapeMarkdown(url)+" is not a valid feed")
			return
		}

		url, _ = rss.SanitizeFeedURL(url)

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
		feedID, feedErr := db.AddFeed(url, feedTitle, webURL)
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
			html.EscapeString(feed.FeedURL),
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
	fmt.Fprintf(&sb, "<b>Your subscriptions (%d):</b>\n", len(feeds))

	for _, f := range feeds {
		fmt.Fprintf(
			&sb,
			"• <a href=\"%s\">%s</a> (<a href=\"%s\">feed</a>)\n",
			html.EscapeString(f.WebURL),
			html.EscapeString(f.Title),
			html.EscapeString(f.FeedURL),
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

	feedID, err := strconv.ParseInt(
		strings.TrimPrefix(update.CallbackQuery.Data, "unsub:"),
		10,
		64,
	)
	if err != nil {
		log.Println("Invalid unsubscribe callback:", err)
		return
	}

	feeds, err := db.GetUserFeeds(message.Chat.ID)
	if err != nil {
		log.Println("Error getting user feeds:", err)
		return
	}

	for _, feed := range feeds {
		if feed.ID != feedID {
			continue
		}

		// Unsubscribe
		if err := db.Unsubscribe(message.Chat.ID, feed.ID); err != nil {
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

		if len(keyboard) == 0 {
			// No feeds left: remove the keyboard by editing the message
			_, err = b.EditMessageText(ctx, &bot.EditMessageTextParams{
				ChatID:    message.Chat.ID,
				MessageID: message.ID,
				Text:      "You have no subscriptions left",
			})
		} else {
			_, err = b.EditMessageReplyMarkup(ctx, &bot.EditMessageReplyMarkupParams{
				ChatID:    message.Chat.ID,
				MessageID: message.ID,
				ReplyMarkup: &models.InlineKeyboardMarkup{
					InlineKeyboard: keyboard,
				},
			})
		}

		if err != nil {
			log.Println("Failed to update message:", err)
		}

		_, _ = b.AnswerCallbackQuery(ctx, &bot.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		msg := fmt.Sprintf(`Unsubscribed from <a href="%s">%s</a>`, feed.FeedURL, feed.Title)

		SendMessageHTML(message.Chat.ID, msg)
		return
	}

	SendMessageMarkdown(message.Chat.ID, "Feed not found")
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
