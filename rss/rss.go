package rss

import (
	"encoding/xml"
	"log"
	"net/http"
	"tg-rss/config"
	"tg-rss/db"
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

var FeedParser *gofeed.Parser
var timeDelta time.Duration

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

func GetArticlesForUser(userID int64) (news []Article) {
	feedEntries, err := db.GetUserFeeds(userID)

	if err != nil {
		log.Println("Error retrieving user subscriptions")
	}

	news = make([]Article, 0)

	for _, feedEntry := range feedEntries {
		feed, err := FeedParser.ParseURL(feedEntry.URL)
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

			news = append(news, newArticle)
		}
	}
	return news
}

func filterOldArticles(newsIn []Article) (newsOut []Article) {
	for _, a := range newsIn {
		if time.Now().Add(-(config.GetUpdatePeriod())).Before(a.Timestamp) {
			newsOut = append(newsOut, a)
		}
	}
	return
}

func FormatNews(news []Article) string {

	newsString := ""

	prevFeed := ""

	for _, art := range news {
		if prevFeed != art.FeedTitle {
			newsString += "**" + art.FeedTitle + "**\n"
			prevFeed = art.FeedTitle
		}

		newsString += "- " + art.Title + " (" + art.URL + ")\n"
	}

	return newsString
}
