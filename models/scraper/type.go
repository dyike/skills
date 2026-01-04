package scraper

// NewsItem represents a scraped news item
type NewsItem struct {
	Title     string `json:"title"`
	URL       string `json:"url"`
	Source    string `json:"source"`
	Score     *int   `json:"score,omitempty"`
	Comments  *int   `json:"comments,omitempty"`
	Author    string `json:"author,omitempty"`
	Tagline   string `json:"tagline,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`
}

// Selectors defines CSS selectors for newsletter scraping
type Selectors struct {
	Container string
	Title     string
	Link      string
	Date      string
}

// DefaultSelectors returns default CSS selectors for newsletter scraping
func DefaultSelectors() Selectors {
	return Selectors{
		Container: "article, .post-preview, .post, [class*='post-item'], .newsletter-item",
		Title:     "h1, h2, h3, [class*='title'], .headline",
		Link:      "a",
		Date:      "time, .date, [datetime], [class*='date']",
	}
}
