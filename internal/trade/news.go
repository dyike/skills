package trade

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

// GoogleNewsClient handles Google News operations
type GoogleNewsClient struct {
	client *resty.Client
}

// NewGoogleNewsClient creates a new Google News client
func NewGoogleNewsClient() *GoogleNewsClient {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")

	return &GoogleNewsClient{
		client: client,
	}
}

// RSS structures for parsing
type RSS struct {
	XMLName xml.Name `xml:"rss"`
	Channel Channel  `xml:"channel"`
}

type Channel struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	Items       []Item `xml:"item"`
}

type Item struct {
	Title       string `xml:"title"`
	Link        string `xml:"link"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Source      Source `xml:"source"`
	GUID        string `xml:"guid"`
}

type Source struct {
	URL  string `xml:"url,attr"`
	Text string `xml:",chardata"`
}

// SearchGoogleNews searches Google News for articles
func (gnc *GoogleNewsClient) SearchGoogleNews(query string, language, country string, maxResults, daysBack int) ([]*NewsArticle, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if language == "" {
		language = "en"
	}
	if country == "" {
		country = "US"
	}
	if maxResults <= 0 {
		maxResults = 20
	}
	if daysBack <= 0 {
		daysBack = 7
	}

	// Try RSS method first (more reliable)
	articles, err := gnc.searchViaRSS(query, language, country, maxResults)
	if err == nil && len(articles) > 0 {
		return filterArticlesByDays(articles, daysBack), nil
	}

	// Fallback to HTML scraping
	htmlArticles, err := gnc.searchViaHTML(query, language, country, maxResults)
	if err != nil {
		return nil, err
	}
	return filterArticlesByDays(htmlArticles, daysBack), nil
}

func (gnc *GoogleNewsClient) searchViaRSS(query, language, country string, maxResults int) ([]*NewsArticle, error) {
	rssURL := gnc.buildRSSURL(query, language, country)

	resp, err := gnc.client.R().Get(rssURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch RSS feed: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP error %d when fetching RSS feed", resp.StatusCode())
	}

	var rss RSS
	if err := xml.Unmarshal(resp.Body(), &rss); err != nil {
		return nil, fmt.Errorf("failed to parse RSS XML: %w", err)
	}

	var articles []*NewsArticle
	for i, item := range rss.Channel.Items {
		if i >= maxResults {
			break
		}

		article := gnc.convertRSSItemToArticle(item, query)
		articles = append(articles, article)
	}

	return articles, nil
}

func (gnc *GoogleNewsClient) searchViaHTML(query, language, country string, maxResults int) ([]*NewsArticle, error) {
	searchURL := gnc.buildSearchURL(query, language, country, maxResults)

	resp, err := gnc.client.R().Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Google News: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP error %d when fetching Google News", resp.StatusCode())
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}

	return gnc.parseNewsHTML(doc, query), nil
}

func (gnc *GoogleNewsClient) buildRSSURL(query, language, country string) string {
	baseURL := "https://news.google.com/rss/search"
	v := url.Values{}
	v.Set("q", query)
	v.Set("hl", language)
	v.Set("gl", country)
	v.Set("ceid", fmt.Sprintf("%s:%s", country, strings.Split(language, "-")[0]))
	return fmt.Sprintf("%s?%s", baseURL, v.Encode())
}

func (gnc *GoogleNewsClient) buildSearchURL(query, language, country string, maxResults int) string {
	baseURL := "https://www.google.com/search"
	v := url.Values{}
	v.Set("q", query)
	v.Set("tbm", "nws")
	v.Set("hl", language)
	v.Set("gl", country)
	v.Set("num", fmt.Sprintf("%d", maxResults))
	return fmt.Sprintf("%s?%s", baseURL, v.Encode())
}

func (gnc *GoogleNewsClient) convertRSSItemToArticle(item Item, query string) *NewsArticle {
	pubTime, err := time.Parse(time.RFC1123Z, item.PubDate)
	if err != nil {
		pubTime, _ = time.Parse("Mon, 02 Jan 2006 15:04:05 MST", item.PubDate)
	}

	source := item.Source.Text
	if source == "" && item.Source.URL != "" {
		if u, err := url.Parse(item.Source.URL); err == nil {
			source = u.Host
		}
	}

	content := gnc.cleanHTMLContent(item.Description)

	return &NewsArticle{
		Title:       strings.TrimSpace(item.Title),
		Content:     content,
		URL:         item.Link,
		Source:      source,
		PublishedAt: pubTime,
		Keywords:    []string{query},
	}
}

func (gnc *GoogleNewsClient) parseNewsHTML(doc *goquery.Document, query string) []*NewsArticle {
	var articles []*NewsArticle

	doc.Find(".SoaBEf, .WlydOe, .g, article").Each(func(i int, s *goquery.Selection) {
		title := strings.TrimSpace(s.Find("h3, .LC20lb, [role='heading']").Text())
		if title == "" {
			return
		}

		link := s.Find("a").First()
		href, exists := link.Attr("href")
		if !exists {
			return
		}

		articleURL := gnc.cleanGoogleURL(href)
		sourceTimeText := strings.TrimSpace(s.Find(".fG8Fp, .slp, time").Text())
		source, timeText := gnc.parseSourceTime(sourceTimeText)
		publishedAt := gnc.parseTimeText(timeText)
		content := strings.TrimSpace(s.Find(".st, .s3v9rd").Text())

		articles = append(articles, &NewsArticle{
			Title:       title,
			Content:     content,
			URL:         articleURL,
			Source:      source,
			PublishedAt: publishedAt,
			Keywords:    []string{query},
		})
	})

	return articles
}

// GetFinanceNews gets finance-related news from Google News
func (gnc *GoogleNewsClient) GetFinanceNews(maxResults int) ([]*NewsArticle, error) {
	financeQueries := []string{
		"stock market",
		"financial news",
		"trading",
	}

	if maxResults <= 0 {
		maxResults = 20
	}

	var allArticles []*NewsArticle
	articlesPerQuery := maxResults / len(financeQueries)
	if articlesPerQuery < 1 {
		articlesPerQuery = 1
	}

	for _, query := range financeQueries {
		articles, err := gnc.SearchGoogleNews(query, "en", "US", articlesPerQuery, 7)
		if err != nil {
			continue
		}
		allArticles = append(allArticles, articles...)
	}

	allArticles = gnc.removeDuplicates(allArticles)
	if len(allArticles) > maxResults {
		allArticles = allArticles[:maxResults]
	}

	return allArticles, nil
}

// GetStockNews gets news for a specific stock symbol
func (gnc *GoogleNewsClient) GetStockNews(symbol string, maxResults int) ([]*NewsArticle, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("stock symbol cannot be empty")
	}

	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	if maxResults <= 0 {
		maxResults = 15
	}

	queries := []string{
		fmt.Sprintf("%s stock news", symbol),
		fmt.Sprintf("%s earnings", symbol),
		symbol,
	}

	var allArticles []*NewsArticle
	articlesPerQuery := maxResults / len(queries)
	if articlesPerQuery < 1 {
		articlesPerQuery = 1
	}

	for _, query := range queries {
		articles, err := gnc.SearchGoogleNews(query, "en", "US", articlesPerQuery, 7)
		if err != nil {
			continue
		}

		for _, article := range articles {
			if gnc.containsStockSymbol(article, symbol) {
				allArticles = append(allArticles, article)
			}
		}
	}

	allArticles = gnc.removeDuplicates(allArticles)
	if len(allArticles) > maxResults {
		allArticles = allArticles[:maxResults]
	}

	return allArticles, nil
}

func (gnc *GoogleNewsClient) cleanGoogleURL(googleURL string) string {
	if strings.Contains(googleURL, "/url?") {
		parts := strings.Split(googleURL, "url=")
		if len(parts) > 1 {
			decoded, err := url.QueryUnescape(parts[1])
			if err == nil {
				if idx := strings.Index(decoded, "&"); idx != -1 {
					decoded = decoded[:idx]
				}
				return decoded
			}
		}
	}

	if strings.HasPrefix(googleURL, "./") {
		return "https://news.google.com" + googleURL[1:]
	}

	if strings.HasPrefix(googleURL, "/") {
		return "https://news.google.com" + googleURL
	}

	return googleURL
}

func (gnc *GoogleNewsClient) parseSourceTime(text string) (source, timeText string) {
	separators := []string{" - ", " · ", " — ", " | "}
	for _, sep := range separators {
		if parts := strings.Split(text, sep); len(parts) >= 2 {
			source = strings.TrimSpace(parts[0])
			timeText = strings.TrimSpace(parts[len(parts)-1])
			return
		}
	}
	source = text
	return
}

func (gnc *GoogleNewsClient) parseTimeText(timeText string) time.Time {
	now := time.Now()
	timeText = strings.ToLower(strings.TrimSpace(timeText))
	if timeText == "" {
		return time.Time{}
	}

	patterns := map[*regexp.Regexp]func([]string) time.Duration{
		regexp.MustCompile(`(\d+)\s*minutes?\s*ago`): func(matches []string) time.Duration {
			if len(matches) > 1 {
				var mins int
				fmt.Sscanf(matches[1], "%d", &mins)
				return time.Duration(mins) * time.Minute
			}
			return 0
		},
		regexp.MustCompile(`(\d+)\s*hours?\s*ago`): func(matches []string) time.Duration {
			if len(matches) > 1 {
				var hours int
				fmt.Sscanf(matches[1], "%d", &hours)
				return time.Duration(hours) * time.Hour
			}
			return 0
		},
		regexp.MustCompile(`(\d+)\s*days?\s*ago`): func(matches []string) time.Duration {
			if len(matches) > 1 {
				var days int
				fmt.Sscanf(matches[1], "%d", &days)
				return time.Duration(days) * 24 * time.Hour
			}
			return 0
		},
	}

	for pattern, handler := range patterns {
		if matches := pattern.FindStringSubmatch(timeText); len(matches) > 0 {
			if duration := handler(matches); duration > 0 {
				return now.Add(-duration)
			}
		}
	}

	return time.Time{}
}

func (gnc *GoogleNewsClient) removeDuplicates(articles []*NewsArticle) []*NewsArticle {
	seen := make(map[string]bool)
	var unique []*NewsArticle

	for _, article := range articles {
		key := fmt.Sprintf("%s|%s", article.URL, article.Title)
		if !seen[key] {
			seen[key] = true
			unique = append(unique, article)
		}
	}

	return unique
}

func (gnc *GoogleNewsClient) containsStockSymbol(article *NewsArticle, symbol string) bool {
	text := strings.ToUpper(article.Title + " " + article.Content)

	patterns := []string{
		fmt.Sprintf(" %s ", symbol),
		fmt.Sprintf("(%s)", symbol),
		fmt.Sprintf("%s:", symbol),
		fmt.Sprintf("$%s", symbol),
	}

	for _, pattern := range patterns {
		if strings.Contains(text, strings.ToUpper(pattern)) {
			return true
		}
	}

	regex := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(symbol)))
	return regex.MatchString(text)
}

func filterArticlesByDays(articles []*NewsArticle, daysBack int) []*NewsArticle {
	if daysBack <= 0 {
		return articles
	}

	cutoff := time.Now().AddDate(0, 0, -daysBack)
	filtered := make([]*NewsArticle, 0, len(articles))

	for _, article := range articles {
		if article.PublishedAt.IsZero() || !article.PublishedAt.Before(cutoff) {
			filtered = append(filtered, article)
		}
	}

	return filtered
}

func (gnc *GoogleNewsClient) cleanHTMLContent(htmlContent string) string {
	if htmlContent == "" {
		return ""
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(htmlContent))
	if err != nil {
		return gnc.stripHTMLTags(htmlContent)
	}

	text := strings.TrimSpace(doc.Text())
	if text == "" {
		return gnc.stripHTMLTags(htmlContent)
	}

	return text
}

func (gnc *GoogleNewsClient) stripHTMLTags(content string) string {
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	content = htmlTagRegex.ReplaceAllString(content, "")

	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&amp;", "&")
	content = strings.ReplaceAll(content, "&lt;", "<")
	content = strings.ReplaceAll(content, "&gt;", ">")
	content = strings.ReplaceAll(content, "&quot;", "\"")
	content = strings.ReplaceAll(content, "&#39;", "'")

	spaceRegex := regexp.MustCompile(`\s+`)
	content = spaceRegex.ReplaceAllString(content, " ")

	return strings.TrimSpace(content)
}
