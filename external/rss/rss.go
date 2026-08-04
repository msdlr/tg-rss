package rss

import (
	"fmt"
	"html"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"sync"
	"tg-rss/config"
	"tg-rss/external/db"
	"time"

	"github.com/mmcdole/gofeed"
	gh "golang.org/x/net/html"
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

func GetRSSFeedInfo(feedURL string) (title string, website string, err error) {
	feed, err := feedParser.ParseURL(feedURL)
	if err != nil {
		return "", "", err
	}

	return feed.Title, feed.Link, nil
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
			feed, err := feedParser.ParseURL(feedEntry.FeedURL)
			if err != nil {
				log.Println("Error fetching " + feedEntry.FeedURL + err.Error())
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

	log.Println("Read all feeds in " + (time.Since(wTimeStart)).String())
	for msg := range msgChan {
		messages = append(messages, msg)
	}

	return
}

func GetYouTubeRSS(url string) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	doc, err := gh.Parse(resp.Body)
	if err != nil {
		return "", err
	}

	var channelID string

	var walk func(*gh.Node)
	walk = func(n *gh.Node) {
		if n.Type == gh.ElementNode && n.Data == "link" {
			var rel, href string
			for _, a := range n.Attr {
				switch a.Key {
				case "rel":
					rel = a.Val
				case "href":
					href = a.Val
				}
			}

			if rel == "canonical" && strings.Contains(href, "/channel/") {
				channelID = href[strings.LastIndex(href, "/")+1:]
				return
			}
		}

		for c := n.FirstChild; c != nil && channelID == ""; c = c.NextSibling {
			walk(c)
		}
	}

	walk(doc)

	if channelID == "" {
		return "", fmt.Errorf("channel ID not found")
	}

	return "https://www.youtube.com/feeds/videos.xml?channel_id=" + channelID, nil
}

func SanitizeFeedURL(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}

	u.Scheme = strings.ToLower(u.Scheme)
	u.Host = strings.ToLower(u.Host)
	u.Fragment = ""

	// Remove default ports.
	switch {
	case u.Scheme == "http" && strings.HasSuffix(u.Host, ":80"):
		u.Host = strings.TrimSuffix(u.Host, ":80")
	case u.Scheme == "https" && strings.HasSuffix(u.Host, ":443"):
		u.Host = strings.TrimSuffix(u.Host, ":443")
	}

	// Clean path and remove trailing slashes.
	p := path.Clean(u.Path)
	if p != "/" {
		p = strings.TrimRight(p, "/")
	}
	u.Path = p

	// Sort query parameters.
	q := u.Query()
	u.RawQuery = q.Encode()

	return u.String(), nil
}
