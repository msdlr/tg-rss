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
	"unicode/utf8"

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
	User              int64
	FormattedMessages []string
}

var feedParser *gofeed.Parser
var lastQuery time.Time
var cache ArticleCache

func SetlastQuery() {
	lastQuery = time.Now()
}

func GetlastQuery() time.Time {
	return lastQuery
}

func InitCache() {
	cache = *(NewArticleCache())
	FetchAllFeeds(0)
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
				// Timeout
				log.Println("Skipping source " + feedEntry.FeedURL + ": " + err.Error())
				return
			}

			for _, article := range feed.Items {
				newArticle := Article{
					URL:         article.Link,
					Timestamp:   *article.PublishedParsed,
					Title:       article.Title,
					Description: article.Description,
					FeedTitle:   feed.Title,
				}

				// Detect links and remove them from the title
				newArticle.Title = func(s string) string {
					result := s

					for _, word := range strings.Fields(s) {
						if strings.HasPrefix(word, "https://") {
							result = strings.Replace(result, word, "[link]", 1)
						}
					}

					return result
				}(newArticle.Title)

				// But if the title was empty, use the URL
				if newArticle.Title == "" {
					newArticle.Title = strings.ReplaceAll(newArticle.URL, "https://", "")
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

func FormatNewsHTML(news []Article) []string {
	var messages []string
	var current strings.Builder
	prevFeed := ""

	for _, art := range news {
		if art.FeedTitle != prevFeed {
			block := fmt.Sprintf(
				"<b>🆕 %s</b>\n",
				html.EscapeString(art.FeedTitle),
			)
			prevFeed = art.FeedTitle

			if utf8.RuneCountInString(current.String())+
				utf8.RuneCountInString(block) > 4096 {
				messages = append(messages, current.String())
				current.Reset()
			}

			current.WriteString(block)
		}

		if art.Title == "" {
			art.Title = art.URL
		}

		line := fmt.Sprintf(
			"• <a href=\"%s\">%s</a>\n",
			html.EscapeString(art.URL),
			html.EscapeString(art.Title),
		)

		if utf8.RuneCountInString(current.String())+
			utf8.RuneCountInString(line) > 4096 {
			messages = append(messages, current.String())
			current.Reset()
		}

		current.WriteString(line)
	}

	if current.Len() > 0 {
		messages = append(messages, current.String())
	}

	return messages
}

func FetchAllFeeds(old uint) {
	feeds, err := db.GetAllFeeds()
	if err != nil {
		log.Println("Error gettings feeds from database:", err)
		return
	}

	var wg sync.WaitGroup
	wg.Add(len(feeds))

	SetlastQuery()
	wTimeStart := time.Now()

	for _, f := range feeds {
		go func(f db.Feed) {
			defer wg.Done()

			arts := []Article{}
			var oldPosts uint = 0
			feed, err := feedParser.ParseURL(f.FeedURL)
			if err != nil {
				log.Println("Skipping source " + f.FeedURL + ": " + err.Error())
				return
			}

			for _, article := range feed.Items {
				newArticle := Article{
					URL:         article.Link,
					Timestamp:   *article.PublishedParsed,
					Title:       article.Title,
					Description: article.Description,
					FeedTitle:   feed.Title,
				}

				newArticle.Title = func(s string) string {
					result := s
					for _, word := range strings.Fields(s) {
						if strings.HasPrefix(word, "https://") {
							result = strings.Replace(result, word, "[link]", 1)
						}
					}
					return result
				}(newArticle.Title)

				if newArticle.Title == "" {
					newArticle.Title = strings.ReplaceAll(newArticle.URL, "https://", "")
				}

				lastTimestamp := time.Now().Add(-config.GetUpdatePeriod())

				if newArticle.Timestamp.Before(lastTimestamp) {
					oldPosts++
					if oldPosts > old {
						break
					} else {
						arts = append(arts, newArticle)
					}
				}
			}

			if len(arts) > 0 {
				cache.Set(f.FeedURL, arts)
			}
		}(f)
	}
	wg.Wait()
	log.Println("Read all feeds in " + (time.Since(wTimeStart)).String())
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

func GetBskyRSS(input string) (feedURL string, err error) {
	// Get the user
	username := input

	// Input is the URL
	u, err := url.Parse(input)
	if err == nil {
		username = u.Path
		username = strings.ReplaceAll(username, "/", "")
	}

	rssURL := "https://bsky.app/profile/" + username + ".bsky.social/rss"

	// Check if nitter returns an error (Not found/private acc)
	_, _, err2 := GetRSSFeedInfo(rssURL)
	if err2 != nil {
		return "", err2

	} else {
		return rssURL, nil
	}
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
