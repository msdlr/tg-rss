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

	for _, feedEntry := range feedEntries {
		var oldPosts uint = 0
		feed, err := feedParser.ParseURL(feedEntry.URL)
		if err != nil {
			return make([]Article, 0)
		}

		for _, article := range feed.Items {
			newArticle := Article{
				URL:         article.Link,
				Timestamp:   *article.PublishedParsed,
				Title:       article.Title,
				Description: article.Description,
				FeedTitle:   feed.Title,
			}

			lastTimestamp := time.Now().Add(-config.GetUpdatePeriod())

			if newArticle.Timestamp.Before(lastTimestamp) {
				oldPosts++
				if oldPosts > old {
					break
				}
			}

			news = append(news, newArticle)
		}
	}
	return news
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
