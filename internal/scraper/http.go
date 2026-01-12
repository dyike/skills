package scraper

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/dyike/skills/models/scraper"
)

var defaultHeaders = map[string]string{
	"User-Agent":      "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	"Accept":          "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
	"Accept-Language": "en-US,en;q=0.9",
}

// HTTPClient wraps http.Client with default headers
type HTTPClient struct {
	client  *http.Client
	headers map[string]string
}

// NewHTTPClient creates a new HTTP client with default settings
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		headers: defaultHeaders,
	}
}

func (c *HTTPClient) get(url string) (*http.Response, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	for k, v := range c.headers {
		req.Header.Set(k, v)
	}
	return c.client.Do(req)
}

func (c *HTTPClient) getDoc(url string) (*goquery.Document, error) {
	resp, err := c.get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return goquery.NewDocumentFromReader(resp.Body)
}

func (c *HTTPClient) getHTML(url string) (string, error) {
	resp, err := c.get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(body), nil
}

// ScrapeHNHTTP scrapes Hacker News using HTTP requests
func ScrapeHNHTTP(limit int) ([]scraper.NewsItem, error) {
	client := NewHTTPClient()
	doc, err := client.getDoc("https://news.ycombinator.com/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HN: %w", err)
	}

	var items []scraper.NewsItem
	scoreRegex := regexp.MustCompile(`(\d+)`)

	doc.Find("tr.athing").Each(func(i int, row *goquery.Selection) {
		if i >= limit {
			return
		}

		itemID, _ := row.Attr("id")
		titleLink := row.Find(".titleline > a").First()
		if titleLink.Length() == 0 {
			return
		}

		title := strings.TrimSpace(titleLink.Text())
		itemURL, _ := titleLink.Attr("href")
		if strings.HasPrefix(itemURL, "item?") {
			itemURL = "https://news.ycombinator.com/" + itemURL
		}

		// Get metadata from subtext row
		subtext := row.Next()
		var score, comments *int
		var author string

		if subtext.Length() > 0 {
			// Score
			scoreEl := subtext.Find(".score")
			if scoreEl.Length() > 0 {
				scoreText := scoreEl.Text()
				if matches := scoreRegex.FindStringSubmatch(scoreText); len(matches) > 1 {
					if s, err := strconv.Atoi(matches[1]); err == nil {
						score = &s
					}
				}
			}

			// Author
			authorEl := subtext.Find(".hnuser")
			if authorEl.Length() > 0 {
				author = strings.TrimSpace(authorEl.Text())
			}

			// Comments
			subtext.Find("a").Each(func(_ int, link *goquery.Selection) {
				text := link.Text()
				if strings.Contains(text, "comment") {
					if matches := scoreRegex.FindStringSubmatch(text); len(matches) > 1 {
						if c, err := strconv.Atoi(matches[1]); err == nil {
							comments = &c
						}
					}
				}
			})
		}

		items = append(items, scraper.NewsItem{
			Title:    title,
			URL:      itemURL,
			Source:   "hackernews",
			Score:    score,
			Comments: comments,
			Author:   author,
		})

		_ = itemID // suppress unused warning
	})

	return items, nil
}

// ScrapePHHTTP scrapes Product Hunt by extracting embedded JSON data
func ScrapePHHTTP(limit int) ([]scraper.NewsItem, error) {
	client := NewHTTPClient()
	client.client.Timeout = 30 * time.Second

	html, err := client.getHTML("https://www.producthunt.com/")
	if err != nil {
		return nil, fmt.Errorf("failed to fetch PH: %w", err)
	}

	return extractPHFromHTML(html, limit)
}

// extractPHFromHTML extracts Product Hunt items from HTML content
func extractPHFromHTML(html string, limit int) ([]scraper.NewsItem, error) {
	var items []scraper.NewsItem

	// Extract the embedded Apollo GraphQL data
	pattern := regexp.MustCompile(`window\[Symbol\.for\("ApolloSSRDataTransport"\)\].*?\.push\((.*?)\);?</script>`)
	matches := pattern.FindStringSubmatch(html)

	if len(matches) < 2 {
		return items, nil
	}

	jsonStr := matches[1]
	// Replace JavaScript undefined with null
	jsonStr = regexp.MustCompile(`\bundefined\b`).ReplaceAllString(jsonStr, "null")

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return items, nil
	}

	rehydrate, ok := data["rehydrate"].(map[string]interface{})
	if !ok {
		return items, nil
	}

	for _, value := range rehydrate {
		valueMap, ok := value.(map[string]interface{})
		if !ok {
			continue
		}

		feedData, ok := valueMap["data"].(map[string]interface{})
		if !ok {
			continue
		}

		homefeed, ok := feedData["homefeed"].(map[string]interface{})
		if !ok {
			continue
		}

		edges, ok := homefeed["edges"].([]interface{})
		if !ok {
			continue
		}

		for _, edge := range edges {
			edgeMap, ok := edge.(map[string]interface{})
			if !ok {
				continue
			}

			node, ok := edgeMap["node"].(map[string]interface{})
			if !ok {
				continue
			}

			posts, ok := node["items"].([]interface{})
			if !ok {
				continue
			}

			for _, post := range posts {
				if len(items) >= limit {
					break
				}

				postMap, ok := post.(map[string]interface{})
				if !ok {
					continue
				}

				name, _ := postMap["name"].(string)
				tagline, _ := postMap["tagline"].(string)
				slug, _ := postMap["slug"].(string)

				if name == "" || slug == "" {
					continue
				}

				itemURL := fmt.Sprintf("https://www.producthunt.com/posts/%s", slug)

				var score, comments *int
				if s, ok := postMap["latestScore"].(float64); ok {
					scoreInt := int(s)
					score = &scoreInt
				}
				if c, ok := postMap["commentsCount"].(float64); ok {
					commentsInt := int(c)
					comments = &commentsInt
				}

				items = append(items, scraper.NewsItem{
					Title:    name,
					URL:      itemURL,
					Source:   "producthunt",
					Score:    score,
					Comments: comments,
					Tagline:  tagline,
				})
			}

			if len(items) > 0 {
				break
			}
		}

		if len(items) > 0 {
			break
		}
	}

	return items, nil
}

// ScrapeNewsletterHTTP scrapes a newsletter archive with configurable selectors
func ScrapeNewsletterHTTP(archiveURL string, selectors *scraper.Selectors, limit int) ([]scraper.NewsItem, error) {
	client := NewHTTPClient()
	doc, err := client.getDoc(archiveURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch newsletter: %w", err)
	}

	sel := scraper.DefaultSelectors()
	if selectors != nil {
		if selectors.Container != "" {
			sel.Container = selectors.Container
		}
		if selectors.Title != "" {
			sel.Title = selectors.Title
		}
		if selectors.Link != "" {
			sel.Link = selectors.Link
		}
		if selectors.Date != "" {
			sel.Date = selectors.Date
		}
	}

	parsedURL, err := url.Parse(archiveURL)
	if err != nil {
		return nil, err
	}
	baseURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	var items []scraper.NewsItem
	seen := make(map[string]bool)

	doc.Find(sel.Container).Each(func(i int, container *goquery.Selection) {
		if len(items) >= limit {
			return
		}

		titleEl := container.Find(sel.Title).First()
		if titleEl.Length() == 0 {
			return
		}

		title := strings.TrimSpace(titleEl.Text())
		if title == "" || len(title) < 3 {
			return
		}

		// Dedupe
		if seen[title] {
			return
		}
		seen[title] = true

		// Find link
		var itemURL string
		linkEl := container.Find(sel.Link).First()
		if linkEl.Length() > 0 {
			if href, exists := linkEl.Attr("href"); exists {
				itemURL = resolveURL(baseURL, href)
			}
		}

		// Date
		var timestamp string
		dateEl := container.Find(sel.Date).First()
		if dateEl.Length() > 0 {
			if dt, exists := dateEl.Attr("datetime"); exists {
				timestamp = dt
			} else {
				timestamp = strings.TrimSpace(dateEl.Text())
			}
		}

		items = append(items, scraper.NewsItem{
			Title:     title,
			URL:       itemURL,
			Source:    "newsletter",
			Timestamp: timestamp,
		})
	})

	return items, nil
}

// ScrapeSubstackHTTP scrapes a Substack publication's archive
func ScrapeSubstackHTTP(publication string, limit int) ([]scraper.NewsItem, error) {
	archiveURL := fmt.Sprintf("https://%s.substack.com/archive", publication)
	return ScrapeNewsletterHTTP(archiveURL, nil, limit)
}

// ScrapeTLDRHTTP scrapes TLDR newsletter latest issue
func ScrapeTLDRHTTP(category string, limit int) ([]scraper.NewsItem, error) {
	client := NewHTTPClient()

	// Get archive page
	archiveURL := fmt.Sprintf("https://tldr.tech/%s/archives", category)
	doc, err := client.getDoc(archiveURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch TLDR archives: %w", err)
	}

	// Find latest issue link
	latestLink := doc.Find("a[href*='/archives/']").First()
	if latestLink.Length() == 0 {
		return nil, fmt.Errorf("no archive links found")
	}

	latestHref, exists := latestLink.Attr("href")
	if !exists {
		return nil, fmt.Errorf("no href in archive link")
	}

	latestURL := resolveURL("https://tldr.tech", latestHref)

	// Fetch latest issue
	doc, err = client.getDoc(latestURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch latest issue: %w", err)
	}

	var items []scraper.NewsItem
	source := fmt.Sprintf("tldr-%s", category)

	// Try articles first
	doc.Find("article, .article-link, [class*='article']").Each(func(i int, article *goquery.Selection) {
		if len(items) >= limit {
			return
		}

		titleEl := article.Find("h3, h4, .title, strong").First()
		if titleEl.Length() == 0 {
			return
		}

		title := strings.TrimSpace(titleEl.Text())
		if title == "" || len(title) <= 5 {
			return
		}

		var itemURL string
		link := article.Find("a[href^='http']").First()
		if link.Length() > 0 {
			itemURL, _ = link.Attr("href")
		}

		items = append(items, scraper.NewsItem{
			Title:  title,
			URL:    itemURL,
			Source: source,
		})
	})

	// Fallback to h3/h4 elements
	if len(items) == 0 {
		doc.Find("h3, h4").Each(func(i int, heading *goquery.Selection) {
			if len(items) >= limit {
				return
			}

			title := strings.TrimSpace(heading.Text())
			if title == "" || len(title) <= 5 {
				return
			}

			var itemURL string
			parent := heading.Parent()
			if parent.Is("a") {
				itemURL, _ = parent.Attr("href")
			} else {
				link := heading.Next()
				if link.Is("a") {
					itemURL, _ = link.Attr("href")
				}
			}

			items = append(items, scraper.NewsItem{
				Title:  title,
				URL:    itemURL,
				Source: source,
			})
		})
	}

	return items, nil
}

// resolveURL resolves a relative URL against a base URL
func resolveURL(baseURL, href string) string {
	if strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") {
		return href
	}

	base, err := url.Parse(baseURL)
	if err != nil {
		return href
	}

	ref, err := url.Parse(href)
	if err != nil {
		return href
	}

	return base.ResolveReference(ref).String()
}
