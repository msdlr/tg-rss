package db

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"tg-rss/config"
	"time"

	"github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Feed struct {
	ID      int64  `json:"id"`
	FeedURL string `json:"url"`
	WebURL  string `json:"url"`
	Title   string `json:"title"`
}

type User struct {
	ChatID   int64  `json:"chat_id"`
	Username string `json:"username"`
}

// InitDB creates or loads the database
func InitDB(dbPath string) error {
	// Ensure directory exists
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	var err error
	db, err = sql.Open("sqlite3", dbPath)
	if err != nil {
		return err
	}

	// Create tables if they don't exist
	if err := initSchema(); err != nil {
		return err
	}

	log.Printf("✅ Database ready at %s", dbPath)
	return nil
}

func initSchema() error {
	schema := `
    CREATE TABLE IF NOT EXISTS users (
        chat_id INTEGER PRIMARY KEY,
        username TEXT
    );
    
    CREATE TABLE IF NOT EXISTS feeds (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        feedurl TEXT UNIQUE NOT NULL,
        weburl TEXT UNIQUE NOT NULL,
        title TEXT
    );
    
    CREATE TABLE IF NOT EXISTS subscriptions (
        id INTEGER PRIMARY KEY AUTOINCREMENT,
        chat_id INTEGER NOT NULL,
        feed_id INTEGER NOT NULL,
        UNIQUE(chat_id, feed_id),
        FOREIGN KEY (chat_id) REFERENCES users(chat_id) ON DELETE CASCADE,
        FOREIGN KEY (feed_id) REFERENCES feeds(id) ON DELETE CASCADE
    );
    
    CREATE INDEX IF NOT EXISTS idx_subscriptions_chat_id ON subscriptions(chat_id);
    CREATE INDEX IF NOT EXISTS idx_subscriptions_feed_id ON subscriptions(feed_id);`

	_, err := db.Exec(schema)
	return err
}

// CloseDB closes the database connection
func CloseDB() error {
	if db != nil {
		return db.Close()
	}
	return nil
}

// AddUser creates or updates a user
func AddUser(chatID int64, username string) error {
	query := `
    INSERT INTO users (chat_id, username)
    VALUES (?, ?)
    ON CONFLICT(chat_id) DO UPDATE SET
        username = excluded.username`

	_, err := db.Exec(query, chatID, username)
	return err
}

// AddFeed adds a new feed if it doesn't exist, returns feed ID
func AddFeed(feedURL string, title string, weburl string) (int64, error) {
	query := `
    INSERT INTO feeds (feedurl, title, weburl)
    VALUES (?, ?, ?)
    ON CONFLICT(feedurl) DO UPDATE SET
        title = excluded.title
    RETURNING id`

	var id int64
	err := db.QueryRow(query, feedURL, title, weburl).Scan(&id)
	return id, err
}

// GetFeedID returns feed ID by URL, or 0 if not found
func GetFeedID(url string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM feeds WHERE feedurl = ?", url).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func GetFeedByURL(url string) (*Feed, error) {
	query := `SELECT id, feedurl, title, weburl FROM feeds WHERE feedurl = ?`

	var feed Feed
	err := db.QueryRow(query, url).Scan(&feed.ID, &feed.FeedURL, &feed.Title, &feed.WebURL)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &feed, nil
}

// Subscribe adds a subscription for a user to a feed
func Subscribe(chatID int64, feedID int64) error {
	query := `INSERT OR IGNORE INTO subscriptions (chat_id, feed_id) VALUES (?, ?)`
	_, err := db.Exec(query, chatID, feedID)
	return err
}

// Unsubscribe removes a subscription
func Unsubscribe(chatID int64, feedID int64) error {
	_, err := db.Exec(`DELETE FROM subscriptions WHERE chat_id = ? AND feed_id = ?`, chatID, feedID)
	db.Exec(`DELETE FROM feeds WHERE id NOT IN (SELECT DISTINCT feed_id FROM subscriptions)`)
	if err != nil {
		return err
	}
	return err
}

// GetUserFeeds returns all feeds a user is subscribed to
func GetUserFeeds(chatID int64) ([]Feed, error) {
	query := `
    SELECT f.id, f.feedurl, f.title, f.weburl
    FROM feeds f
    JOIN subscriptions s ON s.feed_id = f.id
    WHERE s.chat_id = ?`

	rows, err := db.Query(query, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var feeds []Feed
	for rows.Next() {
		var feed Feed
		if err := rows.Scan(&feed.ID, &feed.FeedURL, &feed.Title, &feed.WebURL); err != nil {
			return nil, err
		}
		feeds = append(feeds, feed)
	}
	return feeds, rows.Err()
}

// GetFeedSubscribers returns all chat IDs subscribed to a feed
func GetFeedSubscribers(feedID int64) ([]int64, error) {
	query := `
    SELECT u.chat_id
    FROM users u
    JOIN subscriptions s ON s.chat_id = u.chat_id
    WHERE s.feed_id = ? AND u.active = 1`

	rows, err := db.Query(query, feedID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var chatIDs []int64
	for rows.Next() {
		var chatID int64
		if err := rows.Scan(&chatID); err != nil {
			return nil, err
		}
		chatIDs = append(chatIDs, chatID)
	}
	return chatIDs, rows.Err()
}

func GetAllUsers() ([]User, error) {
	query := `SELECT chat_id, username FROM users`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []User
	for rows.Next() {
		var user User
		if err := rows.Scan(&user.ChatID, &user.Username); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	return users, rows.Err()
}

func GetCount(tableName string) (int, error) {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)
	err := db.QueryRow(query).Scan(&count)
	return count, err
}

func Backup(dbPath, backupPath string) error {
	info, err := os.Stat(dbPath)
	if err != nil {
		return err
	}

	minAge := config.GetBackupPeriod()

	if time.Since(info.ModTime()) > minAge {
		return nil
	}

	// Open the destination database.
	dstDB, err := sql.Open("sqlite3", backupPath)
	if err != nil {
		return err
	}
	defer dstDB.Close()

	ctx := context.Background()

	srcConn, err := db.Conn(ctx)
	if err != nil {
		return err
	}
	defer srcConn.Close()

	dstConn, err := dstDB.Conn(ctx)
	if err != nil {
		return err
	}
	defer dstConn.Close()

	return srcConn.Raw(func(src any) error {
		return dstConn.Raw(func(dst any) error {
			srcSQLite := src.(*sqlite3.SQLiteConn)
			dstSQLite := dst.(*sqlite3.SQLiteConn)

			backup, err := dstSQLite.Backup("main", srcSQLite, "main")
			if err != nil {
				return err
			}
			defer backup.Finish()

			for {
				done, err := backup.Step(-1)
				if err != nil {
					return err
				}
				if done {
					break
				}
			}

			return nil
		})
	})
}
