---
name: news-scraper
description: Scrape latest content from tech news sources including Hacker News, Product Hunt, Twitter/X, Substack newsletters, and TLDR newsletters. Use when Claude needs to fetch current headlines, trending topics, or newsletter archives without API access. Supports both HTTP-based and browser-based (Playwright) scraping methods.
---

# News Scraper

Scrape tech news from multiple sources.

## Usage

```bash
news-scraper -source hn -source ph -limit 20 -format markdown
```

## Supported Sources

| Source | Flag | Description | Requirements |
|--------|------|-------------|--------------|
| Hacker News | `hn` | Front page stories with scores | HTTP works |
| Product Hunt | `ph` | Today's products with upvotes | HTTP limited, Playwright better |
| Twitter Trending | `twitter-trending` | Trending topics from x.com | Requires `-use-browser` (scrapes directly from x.com) |
| Twitter User | `twitter-user` | User timeline from x.com | Requires `-use-browser` and `-twitter-user` (scrapes directly from x.com) |
| Newsletter | `newsletter` | Any newsletter archive | Requires `-newsletter-url` |
| Substack | `substack` | Substack publications | Requires `-substack-name` |
| TLDR | `tldr` | TLDR newsletters | Categories: tech, ai, webdev, crypto, devops, founders |

## Examples

```bash
# Hacker News top 30
news-scraper -source hn -limit 30

# Multiple sources to markdown
news-scraper -source hn -source ph -format markdown -o news.md

# Twitter trending topics from x.com (requires browser)
news-scraper -source twitter-trending -limit 10 -use-browser

# Twitter user timeline from x.com (requires browser)
news-scraper -source twitter-user -twitter-user elonmusk -limit 10 -use-browser

# Substack newsletter
news-scraper -source substack -substack-name stratechery -limit 10

# Custom newsletter archive
news-scraper -source newsletter -newsletter-url https://example.com/archive

# TLDR AI newsletter
news-scraper -source tldr -tldr-category ai

# Use Playwright for JS-heavy sites
news-scraper -source ph -use-browser
```

## Output Formats

- `text` (default): Plain text, numbered list
- `markdown`: Markdown with links and sections
- `json`: JSON array of NewsItem objects

## NewsItem Schema

```json
{
  "title": "Article Title",
  "url": "https://...",
  "source": "hackernews|producthunt|twitter-trending|twitter-user|newsletter|substack|tldr-*",
  "score": 123,
  "comments": 45,
  "author": "username",
  "tagline": "Product description or tweet stats",
  "timestamp": "2024-01-15"
}
```

## Twitter/X Authentication

Twitter/X requires authentication to view trending topics and user timelines. The scraper supports **manual browser login** with session persistence.

### How it works

1. **First run**:
   - Browser window will open automatically (visible, not headless)
   - You manually log in to Twitter/X (supports 2FA, email verification, etc.)
   - After successful login, cookies are saved to `~/.news-scraper/twitter-cookies.json`
   - Scraper continues automatically

2. **Subsequent runs**:
   - Scraper reuses saved cookies in headless mode
   - No browser window needed
   - Much faster execution

3. **Session expires**:
   - Browser window opens again for re-login
   - Process repeats automatically

### Example

```bash
# First run - browser will open for manual login
news-scraper -source twitter-trending -limit 10 -use-browser
# -> Browser opens, you log in manually, window closes automatically

# Subsequent runs - uses saved session (headless)
news-scraper -source twitter-user -twitter-user elonmusk -limit 20 -use-browser
# -> Runs in background, no browser window

# Multiple sources work too
news-scraper -source twitter-trending -source twitter-user -twitter-user elonmusk -use-browser
```

### Manual Login Process

When the browser opens:
1. Complete the Twitter/X login form
2. Handle 2FA if enabled
3. Complete any email/phone verification
4. Wait for redirect to home timeline
5. Browser will close automatically
6. Scraper continues and saves your session

### Security Notes

- **No credentials stored** - you log in manually each time session expires
- Cookies saved locally in `~/.news-scraper/twitter-cookies.json`
- Never commit this file to version control
- Session typically lasts for weeks/months
- Delete cookie file to force fresh login: `rm ~/.news-scraper/twitter-cookies.json`

## Troubleshooting

- **Product Hunt returns empty**: Use `-use-browser` flag
- **Twitter scraping fails**:
  - Twitter sources require `-use-browser` flag (scrapes directly from x.com which requires JavaScript)
  - Check if saved cookies exist at `~/.news-scraper/twitter-cookies.json`
  - Try deleting the cookie file to force a fresh login: `rm ~/.news-scraper/twitter-cookies.json`
  - On first run, wait for browser window to open
- **Browser doesn't open for Twitter login**:
  - Make sure you have Playwright browsers installed
  - Check that you're using the `-use-browser` flag
  - Browser will only open if no saved session exists
- **Login hangs or times out**:
  - You have 5 minutes to complete login
  - Make sure you complete all verification steps
  - Browser will auto-close after successful login (when home timeline loads)
- **Rate limited**: Reduce `-limit` value or add delays between requests