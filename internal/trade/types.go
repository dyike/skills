package trade

import (
	"time"
)

// MarketData represents stock price data
type MarketData struct {
	Symbol string  `json:"symbol"`
	Date   string  `json:"date"`
	Open   float64 `json:"open"`
	High   float64 `json:"high"`
	Low    float64 `json:"low"`
	Close  float64 `json:"close"`
	Volume int64   `json:"volume"`
}

// IndicatorValue represents a single indicator data point
type IndicatorValue struct {
	Date  string  `json:"date"`
	Value float64 `json:"value"`
}

// StockIndicators represents technical indicators for a stock
type StockIndicators struct {
	Symbol     string                      `json:"symbol"`
	StartDate  string                      `json:"start_date"`
	EndDate    string                      `json:"end_date"`
	Indicators map[string][]IndicatorValue `json:"indicators"`
	Summary    string                      `json:"summary"`
}

// RedditPost represents a Reddit post
type RedditPost struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content,omitempty"`
	URL        string    `json:"url"`
	Subreddit  string    `json:"subreddit"`
	Author     string    `json:"author"`
	Score      int       `json:"score"`
	Comments   int       `json:"comments"`
	CreatedAt  time.Time `json:"created_at"`
	IsStickied bool      `json:"is_stickied,omitempty"`
	IsLocked   bool      `json:"is_locked,omitempty"`
}

// RedditOutput represents the output from Reddit tools
type RedditOutput struct {
	Posts  []*RedditPost `json:"posts"`
	Result string        `json:"result"`
}

// NewsArticle represents a news article
type NewsArticle struct {
	Title       string    `json:"title"`
	Content     string    `json:"content,omitempty"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	PublishedAt time.Time `json:"published_at"`
	Keywords    []string  `json:"keywords,omitempty"`
}

// NewsOutput represents the output from news tools
type NewsOutput struct {
	Articles []*NewsArticle `json:"articles"`
	Result   string         `json:"result"`
}
