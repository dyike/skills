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

// PriceChange represents price change statistics
type PriceChange struct {
	Amount      float64 `json:"amount"`
	Percent     float64 `json:"percent"`
	Description string  `json:"description"`
}

// MarketDataStats represents statistical analysis of market data
type MarketDataStats struct {
	TotalDays       int          `json:"total_days"`
	StartDate       string       `json:"start_date"`
	EndDate         string       `json:"end_date"`
	StartPrice      float64      `json:"start_price"`
	EndPrice        float64      `json:"end_price"`
	HighestPrice    float64      `json:"highest_price"`
	HighestDate     string       `json:"highest_date"`
	LowestPrice     float64      `json:"lowest_price"`
	LowestDate      string       `json:"lowest_date"`
	AveragePrice    float64      `json:"average_price"`
	AverageVolume   int64        `json:"average_volume"`
	TotalVolume     int64        `json:"total_volume"`
	PriceChange     *PriceChange `json:"price_change"`
	Volatility      float64      `json:"volatility"`
	UpDays          int          `json:"up_days"`
	DownDays        int          `json:"down_days"`
	UnchangedDays   int          `json:"unchanged_days"`
}

// MarketDataResponse represents enhanced market data with statistics
type MarketDataResponse struct {
	Symbol  string           `json:"symbol"`
	Count   int              `json:"count"`
	Data    []*MarketData    `json:"data"`
	Stats   *MarketDataStats `json:"stats"`
	Summary string           `json:"summary"`
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
