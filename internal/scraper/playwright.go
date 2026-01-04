package scraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"github.com/dyike/skills/models/scraper"
	"github.com/playwright-community/playwright-go"
)

// PlaywrightScraper wraps Playwright browser for scraping
type PlaywrightScraper struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

// NewPlaywrightScraper creates a new Playwright scraper
func NewPlaywrightScraper() (*PlaywrightScraper, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	return &PlaywrightScraper{
		pw:      pw,
		browser: browser,
	}, nil
}

// Close closes the browser and Playwright
func (s *PlaywrightScraper) Close() {
	if s.browser != nil {
		s.browser.Close()
	}
	if s.pw != nil {
		s.pw.Stop()
	}
}

func (s *PlaywrightScraper) newPage() (playwright.Page, error) {
	context, err := s.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(defaultHeaders["User-Agent"]),
	})
	if err != nil {
		return nil, err
	}

	page, err := context.NewPage()
	if err != nil {
		return nil, err
	}

	return page, nil
}

// ScrapeHNPlaywright scrapes Hacker News using Playwright
func (s *PlaywrightScraper) ScrapeHN(limit int) ([]scraper.NewsItem, error) {
	page, err := s.newPage()
	if err != nil {
		return nil, err
	}
	defer page.Close()

	_, err = page.Goto("https://news.ycombinator.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load HN: %w", err)
	}

	_, err = page.WaitForSelector(".athing", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(10000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find items: %w", err)
	}

	rows, err := page.QuerySelectorAll(".athing")
	if err != nil {
		return nil, err
	}

	var items []scraper.NewsItem
	scoreRegex := regexp.MustCompile(`(\d+)`)

	for i, row := range rows {
		if i >= limit {
			break
		}

		itemID, _ := row.GetAttribute("id")

		titleEl, err := row.QuerySelector(".titleline > a")
		if err != nil || titleEl == nil {
			continue
		}

		title, err := titleEl.InnerText()
		if err != nil {
			continue
		}

		itemURL, _ := titleEl.GetAttribute("href")
		if strings.HasPrefix(itemURL, "item?") {
			itemURL = "https://news.ycombinator.com/" + itemURL
		}

		// Get score
		var score *int
		scoreSelector := fmt.Sprintf("#score_%s", itemID)
		scoreEl, err := page.QuerySelector(scoreSelector)
		if err == nil && scoreEl != nil {
			scoreText, err := scoreEl.InnerText()
			if err == nil {
				if matches := scoreRegex.FindStringSubmatch(scoreText); len(matches) > 1 {
					if s, err := strconv.Atoi(matches[1]); err == nil {
						score = &s
					}
				}
			}
		}

		items = append(items, scraper.NewsItem{
			Title:  strings.TrimSpace(title),
			URL:    itemURL,
			Source: "hackernews",
			Score:  score,
		})
	}

	return items, nil
}

// ScrapePHPlaywright scrapes Product Hunt using Playwright
func (s *PlaywrightScraper) ScrapePH(limit int) ([]scraper.NewsItem, error) {
	page, err := s.newPage()
	if err != nil {
		return nil, err
	}
	defer page.Close()

	_, err = page.Goto("https://www.producthunt.com/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(60000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load PH: %w", err)
	}

	// Wait for content
	page.WaitForTimeout(2000)

	// Get HTML content and extract from embedded JSON
	html, err := page.Content()
	if err != nil {
		return nil, err
	}

	return extractPHFromHTML(html, limit)
}

// ScrapeNewsletterPlaywright scrapes a newsletter using Playwright
func (s *PlaywrightScraper) ScrapeNewsletter(archiveURL string, selectors *scraper.Selectors, limit int) ([]scraper.NewsItem, error) {
	page, err := s.newPage()
	if err != nil {
		return nil, err
	}
	defer page.Close()

	_, err = page.Goto(archiveURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load newsletter: %w", err)
	}

	page.WaitForTimeout(2000)

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

	containers, err := page.QuerySelectorAll(sel.Container)
	if err != nil {
		return nil, err
	}

	var items []scraper.NewsItem
	seen := make(map[string]bool)

	for _, container := range containers {
		if len(items) >= limit {
			break
		}

		titleEl, err := container.QuerySelector(sel.Title)
		if err != nil || titleEl == nil {
			continue
		}

		title, err := titleEl.InnerText()
		if err != nil {
			continue
		}
		title = strings.TrimSpace(title)

		if title == "" || len(title) < 3 {
			continue
		}

		if seen[title] {
			continue
		}
		seen[title] = true

		// Find link
		var itemURL string
		linkEl, err := container.QuerySelector(sel.Link)
		if err == nil && linkEl != nil {
			if href, err := linkEl.GetAttribute("href"); err == nil && href != "" {
				itemURL = resolveURL(baseURL, href)
			}
		}

		// Date
		var timestamp string
		dateEl, err := container.QuerySelector(sel.Date)
		if err == nil && dateEl != nil {
			if dt, err := dateEl.GetAttribute("datetime"); err == nil && dt != "" {
				timestamp = dt
			} else if text, err := dateEl.InnerText(); err == nil {
				timestamp = strings.TrimSpace(text)
			}
		}

		items = append(items, scraper.NewsItem{
			Title:     title,
			URL:       itemURL,
			Source:    "newsletter",
			Timestamp: timestamp,
		})
	}

	return items, nil
}

// ScrapeHNPlaywright is a standalone function for HN scraping
func ScrapeHNPlaywright(limit int) ([]scraper.NewsItem, error) {
	scraper, err := NewPlaywrightScraper()
	if err != nil {
		return nil, err
	}
	defer scraper.Close()

	return scraper.ScrapeHN(limit)
}

// ScrapePHPlaywright is a standalone function for PH scraping
func ScrapePHPlaywright(limit int) ([]scraper.NewsItem, error) {
	scraper, err := NewPlaywrightScraper()
	if err != nil {
		return nil, err
	}
	defer scraper.Close()

	return scraper.ScrapePH(limit)
}

// ScrapeNewsletterPlaywright is a standalone function for newsletter scraping
func ScrapeNewsletterPlaywright(archiveURL string, selectors *scraper.Selectors, limit int) ([]scraper.NewsItem, error) {
	sc, err := NewPlaywrightScraper()
	if err != nil {
		return nil, err
	}
	defer sc.Close()

	return sc.ScrapeNewsletter(archiveURL, selectors, limit)
}

// ScrapeTwitterTrending scrapes Twitter/X trending topics directly from x.com using Playwright
func (s *PlaywrightScraper) ScrapeTwitterTrending(nitterInstance string, limit int) ([]scraper.NewsItem, error) {
	page, err := s.newPage()
	if err != nil {
		return nil, err
	}
	defer page.Close()

	// Go directly to X.com explore/trending
	_, err = page.Goto("https://x.com/explore/tabs/trending", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(60000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load X explore page: %w", err)
	}

	// Wait for content to load
	page.WaitForTimeout(3000)

	var items []scraper.NewsItem

	// Extract trending topics using JavaScript evaluation
	result, err := page.Evaluate(`() => {
		const trends = [];
		// Find all trend items - X uses various selectors
		const cells = document.querySelectorAll('[data-testid="trend"], [data-testid="cellInnerDiv"]');
		
		cells.forEach((cell, index) => {
			if (trends.length >= `+fmt.Sprintf("%d", limit)+`) return;
			
			// Try to find trend name
			const spans = cell.querySelectorAll('span');
			let trendName = '';
			let tweetCount = '';
			let category = '';
			
			spans.forEach(span => {
				const text = span.textContent.trim();
				// Skip common non-trend text
				if (text.startsWith('#') || (text.length > 2 && text.length < 100 && !text.includes('·') && !text.toLowerCase().includes('trending'))) {
					if (!trendName && text.length > 1) {
						trendName = text;
					}
				}
				if (text.includes('posts') || text.includes('tweets') || text.match(/\d+[KMB]?\s*(posts|tweets)/i)) {
					tweetCount = text;
				}
				if (text.includes('Trending in') || text.includes('trending')) {
					category = text;
				}
			});
			
			if (trendName && trendName.length > 1) {
				trends.push({
					name: trendName,
					count: tweetCount,
					category: category
				});
			}
		});
		
		return trends;
	}`, nil)

	if err != nil {
		return nil, fmt.Errorf("failed to extract trends: %w", err)
	}

	// Parse the result
	if trends, ok := result.([]interface{}); ok {
		for i, t := range trends {
			if i >= limit {
				break
			}
			if trend, ok := t.(map[string]interface{}); ok {
				name, _ := trend["name"].(string)
				count, _ := trend["count"].(string)
				category, _ := trend["category"].(string)

				if name == "" {
					continue
				}

				var tagline string
				if count != "" {
					tagline = count
				}
				if category != "" {
					if tagline != "" {
						tagline = category + " • " + tagline
					} else {
						tagline = category
					}
				}

				items = append(items, scraper.NewsItem{
					Title:   name,
					URL:     fmt.Sprintf("https://x.com/search?q=%s", url.QueryEscape(name)),
					Source:  "twitter-trending",
					Tagline: tagline,
				})
			}
		}
	}

	return items, nil
}

// ScrapeTwitterUser scrapes a user's timeline directly from x.com using Playwright
func (s *PlaywrightScraper) ScrapeTwitterUser(nitterInstance string, username string, limit int) ([]scraper.NewsItem, error) {
	page, err := s.newPage()
	if err != nil {
		return nil, err
	}
	defer page.Close()

	// Go directly to X.com user page
	userURL := fmt.Sprintf("https://x.com/%s", username)
	_, err = page.Goto(userURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(60000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load user timeline: %w", err)
	}

	// Wait for tweets to load
	page.WaitForTimeout(3000)

	var items []scraper.NewsItem

	// Extract tweets using JavaScript evaluation
	result, err := page.Evaluate(`(args) => {
		const tweets = [];
		const username = args.username;
		const limit = args.limit;
		
		// Find all tweet articles
		const articles = document.querySelectorAll('article[data-testid="tweet"]');
		
		articles.forEach((article, index) => {
			if (tweets.length >= limit) return;
			
			// Extract tweet text
			const tweetTextEl = article.querySelector('[data-testid="tweetText"]');
			const tweetText = tweetTextEl ? tweetTextEl.textContent.trim() : '';
			
			if (!tweetText) return;
			
			// Extract author info
			const userLinks = article.querySelectorAll('a[href^="/"]');
			let author = '';
			let displayName = '';
			userLinks.forEach(link => {
				const href = link.getAttribute('href');
				if (href && href.startsWith('/') && !href.includes('/status/') && href.split('/').length === 2) {
					const spans = link.querySelectorAll('span');
					spans.forEach(span => {
						const text = span.textContent.trim();
						if (text.startsWith('@')) {
							author = text.replace('@', '');
						} else if (text && !displayName) {
							displayName = text;
						}
					});
				}
			});
			
			// Extract tweet URL
			let tweetURL = '';
			const timeLink = article.querySelector('a[href*="/status/"] time');
			if (timeLink) {
				const parentLink = timeLink.closest('a');
				if (parentLink) {
					tweetURL = 'https://x.com' + parentLink.getAttribute('href');
				}
			}
			
			// Extract time
			const timeEl = article.querySelector('time');
			const timestamp = timeEl ? timeEl.getAttribute('datetime') : '';
			
			// Extract stats
			const statsGroup = article.querySelector('[role="group"]');
			let likes = '';
			let retweets = '';
			let replies = '';
			
			if (statsGroup) {
				const buttons = statsGroup.querySelectorAll('button');
				buttons.forEach(btn => {
					const ariaLabel = btn.getAttribute('aria-label') || '';
					if (ariaLabel.includes('like') || ariaLabel.includes('Like')) {
						const match = ariaLabel.match(/(\d+)/);
						if (match) likes = match[1];
					}
					if (ariaLabel.includes('repost') || ariaLabel.includes('Repost') || ariaLabel.includes('retweet')) {
						const match = ariaLabel.match(/(\d+)/);
						if (match) retweets = match[1];
					}
					if (ariaLabel.includes('repl') || ariaLabel.includes('Repl')) {
						const match = ariaLabel.match(/(\d+)/);
						if (match) replies = match[1];
					}
				});
			}
			
			tweets.push({
				text: tweetText,
				author: author || username,
				displayName: displayName,
				url: tweetURL,
				timestamp: timestamp,
				likes: likes,
				retweets: retweets,
				replies: replies
			});
		});
		
		return tweets;
	}`, map[string]interface{}{"username": username, "limit": limit})

	if err != nil {
		return nil, fmt.Errorf("failed to extract tweets: %w", err)
	}

	// Parse the result
	if tweets, ok := result.([]interface{}); ok {
		for i, t := range tweets {
			if i >= limit {
				break
			}
			if tweet, ok := t.(map[string]interface{}); ok {
				text, _ := tweet["text"].(string)
				author, _ := tweet["author"].(string)
				tweetURL, _ := tweet["url"].(string)
				timestamp, _ := tweet["timestamp"].(string)
				likes, _ := tweet["likes"].(string)
				retweets, _ := tweet["retweets"].(string)

				if text == "" {
					continue
				}

				// Truncate title if needed
				title := text
				if len(title) > 100 {
					title = title[:97] + "..."
				}

				// Build tagline
				var taglineParts []string
				if likes != "" && likes != "0" {
					taglineParts = append(taglineParts, "❤️ "+likes)
				}
				if retweets != "" && retweets != "0" {
					taglineParts = append(taglineParts, "🔄 "+retweets)
				}

				item := scraper.NewsItem{
					Title:     title,
					URL:       tweetURL,
					Source:    "twitter-user",
					Author:    author,
					Timestamp: timestamp,
				}
				if len(taglineParts) > 0 {
					item.Tagline = strings.Join(taglineParts, " • ")
				}

				items = append(items, item)
			}
		}
	}

	return items, nil
}

// ScrapeTwitterTrendingPlaywright is a standalone function for Twitter trending
func ScrapeTwitterTrendingPlaywright(nitterInstance string, limit int) ([]scraper.NewsItem, error) {
	sc, err := NewPlaywrightScraper()
	if err != nil {
		return nil, err
	}
	defer sc.Close()

	return sc.ScrapeTwitterTrending(nitterInstance, limit)
}

// ScrapeTwitterUserPlaywright is a standalone function for Twitter user timeline
func ScrapeTwitterUserPlaywright(nitterInstance string, username string, limit int) ([]scraper.NewsItem, error) {
	sc, err := NewPlaywrightScraper()
	if err != nil {
		return nil, err
	}
	defer sc.Close()

	return sc.ScrapeTwitterUser(nitterInstance, username, limit)
}

// cleanTwitterTextPW removes extra whitespace from tweet text
func cleanTwitterTextPW(text string) string {
	re := regexp.MustCompile(`\s+`)
	text = re.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}

// intPtr helper to create int pointer
func intPtr(i int) *int {
	return &i
}

// Helper to extract JSON data from Apollo SSR
func extractApolloData(html string) (map[string]interface{}, error) {
	pattern := regexp.MustCompile(`window\[Symbol\.for\("ApolloSSRDataTransport"\)\].*?\.push\((.*?)\);?</script>`)
	matches := pattern.FindStringSubmatch(html)

	if len(matches) < 2 {
		return nil, fmt.Errorf("no Apollo data found")
	}

	jsonStr := matches[1]
	jsonStr = regexp.MustCompile(`\bundefined\b`).ReplaceAllString(jsonStr, "null")

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		return nil, err
	}

	return data, nil
}
