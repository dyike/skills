package scraper

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/dyike/skills/models/scraper"
	"github.com/playwright-community/playwright-go"
)

// PlaywrightScraper wraps Playwright browser for scraping
type PlaywrightScraper struct {
	pw              *playwright.Playwright
	browser         playwright.Browser
	twitterUsername string
	twitterPassword string
	cookieFile      string
}

// NewPlaywrightScraper creates a new Playwright scraper
func NewPlaywrightScraper() (*PlaywrightScraper, error) {
	return NewPlaywrightScraperWithAuth("", "")
}

// NewPlaywrightScraperWithAuth creates a new Playwright scraper with Twitter authentication
func NewPlaywrightScraperWithAuth(twitterUsername, twitterPassword string) (*PlaywrightScraper, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}

	// Get cookie file path
	homeDir, _ := os.UserHomeDir()
	cookieFile := filepath.Join(homeDir, ".news-scraper", "twitter-cookies.json")

	// Check if we have saved cookies - if not, launch browser in visible mode for login
	headless := true
	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		headless = false
		fmt.Fprintln(os.Stderr, "No saved session found - will open visible browser for login")
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(headless),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-web-security",
		},
	})
	if err != nil {
		pw.Stop()
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	return &PlaywrightScraper{
		pw:              pw,
		browser:         browser,
		twitterUsername: twitterUsername,
		twitterPassword: twitterPassword,
		cookieFile:      cookieFile,
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
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
		Locale:         playwright.String("en-US"),
		TimezoneId:     playwright.String("America/New_York"),
		JavaScriptEnabled: playwright.Bool(true),
	})
	if err != nil {
		return nil, err
	}

	page, err := context.NewPage()
	if err != nil {
		return nil, err
	}

	// Add script to override navigator properties to avoid detection
	err = page.AddInitScript(playwright.Script{
		Content: playwright.String(`
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined
			});
			Object.defineProperty(navigator, 'platform', {
				get: () => 'MacIntel'
			});
			Object.defineProperty(navigator, 'plugins', {
				get: () => [1, 2, 3, 4, 5]
			});
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en']
			});
		`),
	})
	if err != nil {
		return nil, err
	}

	return page, nil
}

// newTwitterPage creates a new page with Twitter authentication
func (s *PlaywrightScraper) newTwitterPage() (playwright.Page, playwright.BrowserContext, error) {
	context, err := s.browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String(defaultHeaders["User-Agent"]),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
		Locale:            playwright.String("en-US"),
		TimezoneId:        playwright.String("America/New_York"),
		JavaScriptEnabled: playwright.Bool(true),
	})
	if err != nil {
		return nil, nil, err
	}

	// Ensure Twitter authentication
	if err := s.ensureTwitterAuth(context); err != nil {
		context.Close()
		return nil, nil, err
	}

	page, err := context.NewPage()
	if err != nil {
		context.Close()
		return nil, nil, err
	}

	// Add script to override navigator properties to avoid detection
	err = page.AddInitScript(playwright.Script{
		Content: playwright.String(`
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined
			});
			Object.defineProperty(navigator, 'platform', {
				get: () => 'MacIntel'
			});
			Object.defineProperty(navigator, 'plugins', {
				get: () => [1, 2, 3, 4, 5]
			});
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en']
			});
		`),
	})
	if err != nil {
		page.Close()
		context.Close()
		return nil, nil, err
	}

	return page, context, nil
}

// saveCookies saves cookies to file
func (s *PlaywrightScraper) saveCookies(context playwright.BrowserContext) error {
	cookies, err := context.Cookies()
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(s.cookieFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// Save cookies to file
	data, err := json.MarshalIndent(cookies, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(s.cookieFile, data, 0644)
}

// loadCookies loads cookies from file into context
func (s *PlaywrightScraper) loadCookies(context playwright.BrowserContext) error {
	data, err := os.ReadFile(s.cookieFile)
	if err != nil {
		return err
	}

	var cookies []playwright.Cookie
	if err := json.Unmarshal(data, &cookies); err != nil {
		return err
	}

	// Convert []playwright.Cookie to []playwright.OptionalCookie
	optionalCookies := make([]playwright.OptionalCookie, len(cookies))
	for i, cookie := range cookies {
		optionalCookies[i] = playwright.OptionalCookie{
			Name:     cookie.Name,
			Value:    cookie.Value,
			Domain:   &cookie.Domain,
			Path:     &cookie.Path,
			Expires:  &cookie.Expires,
			HttpOnly: &cookie.HttpOnly,
			Secure:   &cookie.Secure,
			SameSite: cookie.SameSite,
		}
	}

	return context.AddCookies(optionalCookies)
}

// loginToTwitter performs Twitter login - opens browser for manual login
func (s *PlaywrightScraper) loginToTwitter(page playwright.Page) error {
	fmt.Fprintln(os.Stderr, "\n==========================================================")
	fmt.Fprintln(os.Stderr, "Please log in to Twitter/X in the browser window")
	fmt.Fprintln(os.Stderr, "The browser will open shortly...")
	fmt.Fprintln(os.Stderr, "After logging in, the scraper will continue automatically")
	fmt.Fprintln(os.Stderr, "==========================================================\n")

	// Go to login page
	_, err := page.Goto("https://x.com/i/flow/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		return fmt.Errorf("failed to load login page: %w", err)
	}

	// Wait for user to complete login by checking for navigation bar
	fmt.Fprintln(os.Stderr, "Waiting for you to complete login...")
	_, err = page.WaitForSelector("nav[role='navigation']", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(300000), // 5 minutes timeout
	})
	if err != nil {
		return fmt.Errorf("login timeout or failed - could not find navigation: %w", err)
	}

	fmt.Fprintln(os.Stderr, "✓ Successfully logged in to Twitter/X")
	fmt.Fprintln(os.Stderr, "✓ Session saved for future use\n")
	return nil
}

// ensureTwitterAuth ensures we have valid Twitter authentication
func (s *PlaywrightScraper) ensureTwitterAuth(context playwright.BrowserContext) error {
	// Try to load existing cookies first
	if _, err := os.Stat(s.cookieFile); err == nil {
		if err := s.loadCookies(context); err == nil {
			fmt.Fprintln(os.Stderr, "✓ Loaded saved Twitter session")
			return nil
		}
		fmt.Fprintln(os.Stderr, "⚠ Failed to load saved session, will need to login")
	}

	// No valid cookies, need to login manually
	fmt.Fprintln(os.Stderr, "No saved session found, opening browser for login...")

	// Create a page for login
	page, err := context.NewPage()
	if err != nil {
		return err
	}
	defer page.Close()

	// Perform manual login
	if err := s.loginToTwitter(page); err != nil {
		return err
	}

	// Save cookies for future use
	if err := s.saveCookies(context); err != nil {
		fmt.Fprintf(os.Stderr, "⚠ Warning: failed to save cookies: %v\n", err)
	} else {
		fmt.Fprintln(os.Stderr, "✓ Session cookies saved to", s.cookieFile)
	}

	return nil
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
func (s *PlaywrightScraper) ScrapeTwitterTrending(limit int) ([]scraper.NewsItem, error) {
	page, context, err := s.newTwitterPage()
	if err != nil {
		return nil, err
	}
	defer context.Close()
	defer page.Close()

	// Go directly to X.com explore/trending
	_, err = page.Goto("https://x.com/explore/tabs/trending", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(90000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load X explore page: %w", err)
	}

	// Wait for content to load
	page.WaitForTimeout(5000)

	// Scroll to load more trends
	for i := 0; i < 2; i++ {
		page.Evaluate("window.scrollBy(0, 800)")
		page.WaitForTimeout(1000)
	}

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
func (s *PlaywrightScraper) ScrapeTwitterUser(username string, limit int) ([]scraper.NewsItem, error) {
	page, context, err := s.newTwitterPage()
	if err != nil {
		return nil, err
	}
	defer context.Close()
	defer page.Close()

	// Go directly to X.com user page
	userURL := fmt.Sprintf("https://x.com/%s", username)
	_, err = page.Goto(userURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(90000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to load user timeline: %w", err)
	}

	// Wait for tweets to load - wait for the tweet article elements
	page.WaitForTimeout(5000)

	// Try to wait for tweet elements, but don't fail if they don't appear
	page.WaitForSelector("article[data-testid=\"tweet\"]", playwright.PageWaitForSelectorOptions{
		State:   playwright.WaitForSelectorStateAttached,
		Timeout: playwright.Float(15000),
	})

	// Scroll to load more tweets
	for i := 0; i < 3; i++ {
		page.Evaluate("window.scrollBy(0, 1000)")
		page.WaitForTimeout(1000)
	}

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
func ScrapeTwitterTrendingPlaywright(limit int) ([]scraper.NewsItem, error) {
	// Get credentials from environment variables
	username := os.Getenv("TWITTER_USERNAME")
	password := os.Getenv("TWITTER_PASSWORD")

	sc, err := NewPlaywrightScraperWithAuth(username, password)
	if err != nil {
		return nil, err
	}
	defer sc.Close()

	return sc.ScrapeTwitterTrending(limit)
}

// ScrapeTwitterUserPlaywright is a standalone function for Twitter user timeline
func ScrapeTwitterUserPlaywright(username string, limit int) ([]scraper.NewsItem, error) {
	// Get credentials from environment variables
	twitterUsername := os.Getenv("TWITTER_USERNAME")
	twitterPassword := os.Getenv("TWITTER_PASSWORD")

	sc, err := NewPlaywrightScraperWithAuth(twitterUsername, twitterPassword)
	if err != nil {
		return nil, err
	}
	defer sc.Close()

	return sc.ScrapeTwitterUser(username, limit)
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
