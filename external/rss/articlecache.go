package rss

import "sync"

type ArticleCache struct {
	mu   sync.RWMutex
	data map[string][]Article
}

func NewArticleCache() *ArticleCache {
	return &ArticleCache{
		data: make(map[string][]Article),
	}
}

func (c *ArticleCache) Append(key string, article Article) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = append(c.data[key], article)
}

func (c *ArticleCache) Set(key string, article []Article) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.data[key] = article
}

func (c *ArticleCache) Get(key string) ([]Article, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	articles, ok := c.data[key]
	return articles, ok
}
