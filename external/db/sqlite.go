package db

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type Feed struct {
	ID    int64  `json:"id"`
	URL   string `json:"url"`
	Title string `json:"title"`
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
        url TEXT UNIQUE NOT NULL,
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
func AddFeed(url, title string) (int64, error) {
	query := `
    INSERT INTO feeds (url, title)
    VALUES (?, ?)
    ON CONFLICT(url) DO UPDATE SET
        title = excluded.title
    RETURNING id`

	var id int64
	err := db.QueryRow(query, url, title).Scan(&id)
	return id, err
}

// GetFeedID returns feed ID by URL, or 0 if not found
func GetFeedID(url string) (int64, error) {
	var id int64
	err := db.QueryRow("SELECT id FROM feeds WHERE url = ?", url).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return id, err
}

func GetFeedByURL(url string) (*Feed, error) {
	query := `SELECT id, url, title FROM feeds WHERE url = ?`

	var feed Feed
	err := db.QueryRow(query, url).Scan(&feed.ID, &feed.URL, &feed.Title)
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
	return err
}

// GetUserFeeds returns all feeds a user is subscribed to
func GetUserFeeds(chatID int64) ([]Feed, error) {
	query := `
    SELECT f.id, f.url, f.title
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
		if err := rows.Scan(&feed.ID, &feed.URL, &feed.Title); err != nil {
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
