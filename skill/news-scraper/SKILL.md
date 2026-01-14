---
name: news-scraper
description: Scrape latest content from tech news sources including Hacker News, Product Hunt, Twitter/X, Xiaohongshu, and newsletters. Use when you need to fetch current headlines, trending topics, or newsletter archives. Built on github.com/crosszan/modu scraper package.
---

# News Scraper

Scrape tech news from multiple sources using cobra subcommands.

## Usage

```bash
news-scraper [command] [flags]
```

## Available Commands

| Command | Description | Requirements |
|---------|-------------|--------------|
| `hn` | Scrape Hacker News front page | None |
| `ph` | Scrape Product Hunt today's products | None |
| `twitter-trending` | Scrape trending topics from x.com | Twitter login (modu handles auth) |
| `twitter-user` | Scrape user timeline from x.com | `-u username` required |
| `newsletter` | Scrape any newsletter archive | `-u url` required |
| `xhs` | Scrape Xiaohongshu (Little Red Book) | None |

## Global Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--limit` | `-l` | 20 | Max items to scrape |
| `--format` | `-f` | text | Output format: text, markdown, json |
| `--output` | `-o` | stdout | Output file path |

## Examples

```bash
# Hacker News top 30
news-scraper hn -l 30

# Product Hunt in markdown format
news-scraper ph -f markdown

# Save to file
news-scraper hn -l 20 -o news.md

# Twitter trending topics
news-scraper twitter-trending -l 10

# Twitter user timeline
news-scraper twitter-user -u elonmusk -l 10

# Newsletter archive
news-scraper newsletter -u https://example.substack.com/archive

# Xiaohongshu
news-scraper xhs -l 5

# JSON output
news-scraper hn -f json
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
  "source": "hackernews|producthunt|twitter-trending|twitter-user|newsletter|xhs",
  "score": 123,
  "comments": 45,
  "author": "username",
  "tagline": "Product description or tweet stats",
  "timestamp": "2024-01-15"
}
```

## Twitter/X Authentication

Twitter/X scraping is handled by modu's scraper package, which uses persistent browser context for authentication.

### How it works

1. **First run**: Browser opens for manual login, session saved automatically
2. **Subsequent runs**: Uses saved session in headless mode
3. **Session expires**: Browser opens again for re-login

### Security

- No credentials stored in config
- Session cookies managed by modu package
- Delete session to force re-login

## Architecture

```
cmd/news-scraper/main.go          # Entry point (3 lines)
internal/scraper/cmd.go            # Cobra commands & logic
github.com/crosszan/modu/repos/scraper  # Core scraping (external)
```

## Troubleshooting

- **Empty results**: Check network connection, site may be blocking
- **Twitter fails**: First run opens browser for login, complete the login process
- **Rate limited**: Reduce `-l` limit value