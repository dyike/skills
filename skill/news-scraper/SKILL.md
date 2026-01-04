---
name: news-scraper
description: Scrape latest content from tech news sources including Hacker News, Product Hunt, Twitter/X (via Nitter), Substack newsletters, and TLDR newsletters. Use when Claude needs to fetch current headlines, trending topics, or newsletter archives without API access. Supports both HTTP-based and browser-based (Playwright) scraping methods.
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
| Twitter Trending | `twitter-trending` | Trending topics on X/Twitter | Requires `-use-browser` |
| Twitter User | `twitter-user` | User timeline tweets | Requires `-use-browser` and `-twitter-user` |
| Newsletter | `newsletter` | Any newsletter archive | Requires `-newsletter-url` |
| Substack | `substack` | Substack publications | Requires `-substack-name` |
| TLDR | `tldr` | TLDR newsletters | Categories: tech, ai, webdev, crypto, devops, founders |

## Examples

```bash
# Hacker News top 30
news-scraper -source hn -limit 30

# Multiple sources to markdown
news-scraper -source hn -source ph -format markdown -o news.md

# Twitter trending topics (requires browser)
news-scraper -source twitter-trending -limit 10 -use-browser

# Twitter user timeline (requires browser)
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
  "source": "hackernews|producthunt|twitter-trending|twitter-user|newsletter|tldr-*",
  "score": 123,
  "comments": 45,
  "author": "username",
  "tagline": "Product description or tweet stats",
  "timestamp": "2024-01-15"
}
```

## Troubleshooting

- **Product Hunt returns empty**: Use `-use-browser` flag
- **Twitter scraping fails**: Ensure `-use-browser` flag is set; X.com requires JavaScript
- **Rate limited**: Reduce `-limit` value