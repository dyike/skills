package trade

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

// RedditClient handles Reddit API operations
type RedditClient struct {
	client *resty.Client
}

// NewRedditClient creates a new Reddit client
func NewRedditClient() *RedditClient {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("User-Agent", "TradeSkill/1.0")

	return &RedditClient{
		client: client,
	}
}

// RedditResponse represents the API response structure
type RedditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		After    string        `json:"after"`
		Before   string        `json:"before"`
		Children []RedditChild `json:"children"`
	} `json:"data"`
}

// RedditChild represents a Reddit post wrapper
type RedditChild struct {
	Kind string         `json:"kind"`
	Data RedditPostData `json:"data"`
}

// RedditPostData represents Reddit post data from API
type RedditPostData struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Selftext    string  `json:"selftext"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
	Subreddit   string  `json:"subreddit"`
	Author      string  `json:"author"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
	CreatedUTC  float64 `json:"created_utc"`
	Stickied    bool    `json:"stickied"`
	Locked      bool    `json:"locked"`
	IsSelf      bool    `json:"is_self"`
}

// GetSubredditPosts retrieves posts from a specific subreddit
func (rc *RedditClient) GetSubredditPosts(subreddit, sort string, limit int) ([]*RedditPost, error) {
	if strings.TrimSpace(subreddit) == "" {
		return nil, fmt.Errorf("subreddit cannot be empty")
	}

	if sort == "" {
		sort = "hot"
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	redditURL := fmt.Sprintf("https://www.reddit.com/r/%s/%s.json?limit=%d", subreddit, sort, limit)

	resp, err := rc.client.R().Get(redditURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Reddit posts: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP error %d when fetching Reddit posts", resp.StatusCode())
	}

	var redditResp RedditResponse
	if err := json.Unmarshal(resp.Body(), &redditResp); err != nil {
		return nil, fmt.Errorf("failed to parse Reddit JSON: %w", err)
	}

	return rc.convertToRedditPosts(redditResp.Data.Children), nil
}

// SearchReddit searches Reddit for posts matching a query
func (rc *RedditClient) SearchReddit(query, subreddit, sort, timePeriod string, limit int) ([]*RedditPost, error) {
	if strings.TrimSpace(query) == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	if sort == "" {
		sort = "relevance"
	}
	if timePeriod == "" {
		timePeriod = "week"
	}
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	searchURL := "https://www.reddit.com/search.json"
	values := url.Values{}
	values.Set("q", query)
	values.Set("sort", sort)
	values.Set("t", timePeriod)
	values.Set("limit", fmt.Sprintf("%d", limit))

	if subreddit != "" {
		values.Set("q", fmt.Sprintf("%s subreddit:%s", query, subreddit))
	}

	fullURL := fmt.Sprintf("%s?%s", searchURL, values.Encode())

	resp, err := rc.client.R().Get(fullURL)
	if err != nil {
		return nil, fmt.Errorf("failed to search Reddit: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("HTTP error %d when searching Reddit", resp.StatusCode())
	}

	var redditResp RedditResponse
	if err := json.Unmarshal(resp.Body(), &redditResp); err != nil {
		return nil, fmt.Errorf("failed to parse Reddit JSON: %w", err)
	}

	return rc.convertToRedditPosts(redditResp.Data.Children), nil
}

// GetStockMentions searches for mentions of a specific stock symbol
func (rc *RedditClient) GetStockMentions(symbol string) ([]*RedditPost, error) {
	if strings.TrimSpace(symbol) == "" {
		return nil, fmt.Errorf("stock symbol cannot be empty")
	}

	symbol = strings.ToUpper(strings.TrimSpace(symbol))
	financeSubreddits := "wallstreetbets+stocks+investing+SecurityAnalysis+StockMarket"

	queries := []string{
		fmt.Sprintf("$%s", symbol),
		fmt.Sprintf("%s stock", symbol),
		symbol,
	}

	var allResults []*RedditPost
	seen := make(map[string]bool)

	for _, query := range queries {
		posts, err := rc.SearchReddit(query, financeSubreddits, "relevance", "week", 25)
		if err != nil {
			continue
		}

		for _, post := range posts {
			if !seen[post.ID] && rc.containsStockSymbol(post, symbol) {
				seen[post.ID] = true
				allResults = append(allResults, post)
			}
		}
	}

	return allResults, nil
}

// GetPopularFinancePosts gets posts from popular finance-related subreddits
func (rc *RedditClient) GetPopularFinancePosts(limit int) ([]*RedditPost, error) {
	financeSubreddits := []string{
		"wallstreetbets", "investing", "stocks", "SecurityAnalysis",
		"ValueInvesting", "options", "StockMarket",
	}

	if limit <= 0 {
		limit = 50
	}

	var allPosts []*RedditPost
	postsPerSub := limit / len(financeSubreddits)
	if postsPerSub < 1 {
		postsPerSub = 1
	}

	for _, subreddit := range financeSubreddits {
		posts, err := rc.GetSubredditPosts(subreddit, "hot", postsPerSub)
		if err != nil {
			continue
		}
		allPosts = append(allPosts, posts...)
	}

	// Sort by score and limit results
	if len(allPosts) > limit {
		for i := 0; i < len(allPosts)-1; i++ {
			for j := i + 1; j < len(allPosts); j++ {
				if allPosts[i].Score < allPosts[j].Score {
					allPosts[i], allPosts[j] = allPosts[j], allPosts[i]
				}
			}
		}
		allPosts = allPosts[:limit]
	}

	return allPosts, nil
}

func (rc *RedditClient) convertToRedditPosts(children []RedditChild) []*RedditPost {
	var posts []*RedditPost

	for _, child := range children {
		if child.Kind != "t3" {
			continue
		}

		data := child.Data
		createdAt := time.Unix(int64(data.CreatedUTC), 0)

		fullURL := data.URL
		if data.IsSelf {
			fullURL = fmt.Sprintf("https://www.reddit.com%s", data.Permalink)
		}

		post := &RedditPost{
			ID:         data.ID,
			Title:      data.Title,
			Content:    data.Selftext,
			URL:        fullURL,
			Subreddit:  data.Subreddit,
			Author:     data.Author,
			Score:      data.Score,
			Comments:   data.NumComments,
			CreatedAt:  createdAt,
			IsStickied: data.Stickied,
			IsLocked:   data.Locked,
		}

		posts = append(posts, post)
	}

	return posts
}

func (rc *RedditClient) containsStockSymbol(post *RedditPost, symbol string) bool {
	text := strings.ToUpper(post.Title + " " + post.Content)

	patterns := []string{
		fmt.Sprintf("$%s", symbol),
		fmt.Sprintf(" %s ", symbol),
		fmt.Sprintf("(%s)", symbol),
		fmt.Sprintf("%s STOCK", symbol),
		fmt.Sprintf("%s SHARES", symbol),
	}

	for _, pattern := range patterns {
		if strings.Contains(text, strings.ToUpper(pattern)) {
			return true
		}
	}

	regex := regexp.MustCompile(fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(symbol)))
	return regex.MatchString(text)
}
