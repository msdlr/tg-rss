package rss

import (
	"encoding/xml"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"
	"sync"
	"tg-rss/config"
	"tg-rss/external/db"
	"tg-rss/stats"
	"time"

	"github.com/mmcdole/gofeed"
)

type Article struct {
	URL         string
	Timestamp   time.Time
	Title       string
	Description string
	FeedTitle   string
}

type UpdateMsg struct {
	User             int64
	FormattedMessage string
}

var feedParser *gofeed.Parser
var timeDelta time.Duration
var lastQuery time.Time

func SetlastQuery() {
	lastQuery = time.Now()
}

func GetlastQuery() time.Time {
	return lastQuery
}

func InitFeedParser() {
	feedParser = gofeed.NewParser()
}

func GetRSSFeedTitle(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var title string
	decoder := xml.NewDecoder(resp.Body)
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", err
		}

		switch elem := token.(type) {
		case xml.StartElement:
			if elem.Name.Local == "title" {
				if err := decoder.DecodeElement(&title, &elem); err != nil {
					return "", err
				}
				return title, nil
			}
		}
	}
}

func GetArticlesForUser(userID int64, old uint) (news []Article) {
	feedEntries, err := db.GetUserFeeds(userID)

	if err != nil {
		log.Println("Error retrieving user subscriptions")
	}

	news = make([]Article, 0)

	artChan := make(chan Article, len(feedEntries))

	var wg sync.WaitGroup
	wg.Add(len(feedEntries))

	for _, feedEntry := range feedEntries {
		go func(c chan<- Article, wg *sync.WaitGroup) {
			defer wg.Done()

			var oldPosts uint = 0
			feed, err := feedParser.ParseURL(feedEntry.URL)
			if err != nil {
				log.Println("Error fetching " + feedEntry.URL + err.Error())
			}

			for _, article := range feed.Items {
				newArticle := Article{
					URL:         article.Link,
					Timestamp:   *article.PublishedParsed,
					Title:       article.Title,
					Description: article.Description,
					FeedTitle:   feed.Title,
				}

				// The oldest timestamp possible
				lastTimestamp := time.Now().Add(-config.GetUpdatePeriod())

				if newArticle.Timestamp.Before(lastTimestamp) {
					oldPosts++
					if oldPosts > old {
						break
					}
				}

				c <- newArticle
			}

		}(artChan, &wg)
	}

	go func() {
		wg.Wait()
		close(artChan)
	}()

	for a := range artChan {
		news = append(news, a)
	}
	return

}

func FormatNewsHTML(news []Article) string {
	var b strings.Builder

	prevFeed := ""

	for _, art := range news {
		if prevFeed != art.FeedTitle {
			if prevFeed != "" {
				b.WriteByte('\n')
			}

			fmt.Fprintf(&b, "<b>%s</b>\n", html.EscapeString(art.FeedTitle))
			prevFeed = art.FeedTitle
		}

		if art.Title == "" {
			art.Title = art.URL
		}

		fmt.Fprintf(
			&b,
			"• <a href=\"%s\">%s</a>\n",
			html.EscapeString(art.URL),
			html.EscapeString(art.Title),
		)
	}

	return b.String()
}

func ReadAllFeeds() (messages []UpdateMsg) {
	users, err := db.GetAllUsers()
	if err != nil {
		log.Println("Error fetching users:", err)
	}
	SetlastQuery()
	wTimeStart := time.Now()

	var wg sync.WaitGroup
	wg.Add(len(users))
	msgChan := make(chan UpdateMsg)

	for _, user := range users {
		go func(wg *sync.WaitGroup, c chan<- UpdateMsg) {
			defer wg.Done()
			arts := GetArticlesForUser(user.ChatID, 0)

			if len(arts) > 0 {
				msg := UpdateMsg{user.ChatID, FormatNewsHTML(arts)}
				c <- msg
			}

		}(&wg, msgChan)
	}

	go func() {
		wg.Wait()
		close(msgChan)
	}()

	duration := time.Since(wTimeStart)
	stats.RecordFeedsPullDuration(duration)

	log.Println("Read all feeds in " + (duration).String())
	for msg := range msgChan {
		messages = append(messages, msg)
	}

	return
}
