package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"
)

// RedditSource Reddit数据源
type RedditSource struct {
	httpClient *HTTPClient
	baseURL    string
	userAgent  string
}

// NewRedditSource 创建Reddit数据源
func NewRedditSource() *RedditSource {
	return &RedditSource{
		httpClient: NewHTTPClient(),
		baseURL:    "https://www.reddit.com",
		userAgent:  "CryptoAnalysisBot/1.0", // Reddit API要求设置User-Agent
	}
}

// Name 数据源名称
func (r *RedditSource) Name() string {
	return "reddit"
}

// IsAvailable 检查数据源是否可用
func (r *RedditSource) IsAvailable() bool {
	// 测试Reddit API连通性
	url := fmt.Sprintf("%s/r/cryptocurrency/hot.json?limit=1", r.baseURL)
	_, err := r.httpClient.Get(url)
	return err == nil
}

// FetchMarketData Reddit不支持市场数据
func (r *RedditSource) FetchMarketData(ctx context.Context, symbols []string) ([]MarketData, error) {
	return []MarketData{}, nil
}

// FetchNewsData Reddit不支持结构化新闻数据
func (r *RedditSource) FetchNewsData(ctx context.Context, symbols []string) ([]NewsData, error) {
	return []NewsData{}, nil
}

// FetchSocialData 获取社交数据
func (r *RedditSource) FetchSocialData(ctx context.Context, symbols []string) ([]SocialData, error) {
	result := make([]SocialData, 0)

	// 主要加密货币相关的subreddit
	subreddits := []string{
		"cryptocurrency",
		"bitcoin",
		"ethereum",
		"cryptomarkets",
		"cryptocurrencytrading",
		"binance",
		"coinbase",
	}

	for _, subreddit := range subreddits {
		socialData, err := r.fetchSubredditData(subreddit, symbols)
		if err != nil {
			continue // 跳过错误，继续下一个subreddit
		}

		result = append(result, socialData...)

		// 避免API限速
		time.Sleep(500 * time.Millisecond)
	}

	return result, nil
}

// fetchSubredditData 获取subreddit数据
func (r *RedditSource) fetchSubredditData(subreddit string, targetSymbols []string) ([]SocialData, error) {
	url := fmt.Sprintf("%s/r/%s/hot.json?limit=25", r.baseURL, subreddit)

	// 设置User-Agent头
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", r.userAgent)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Reddit API returned status %d", resp.StatusCode)
	}

	var redditResp RedditResponse
	if err := json.NewDecoder(resp.Body).Decode(&redditResp); err != nil {
		return nil, err
	}

	result := make([]SocialData, 0)

	for _, post := range redditResp.Data.Children {
		// 分析帖子标题和内容
		content := post.Data.Title + " " + post.Data.Selftext
		symbols := r.extractCryptoSymbols(content)

		if len(symbols) > 0 {
			// 计算情感得分
			sentiment := r.analyzeRedditSentiment(content)

			// 计算参与度 ( upvotes + comments )
			engagement := post.Data.Ups + post.Data.NumComments

			// 转换时间戳
			postedAt := time.Unix(int64(post.Data.CreatedUTC), 0)

			for _, symbol := range symbols {
				// 只收集目标币种的数据
				found := false
				for _, target := range targetSymbols {
					if strings.ToUpper(symbol) == strings.ToUpper(target) {
						found = true
						break
					}
				}

				if found || len(targetSymbols) == 0 { // 如果没有指定目标，收集所有
					socialData := SocialData{
						Platform:   fmt.Sprintf("reddit_%s", subreddit),
						Symbol:     strings.ToUpper(symbol),
						Mentions:   1, // 每个帖子算一次提及
						Sentiment:  sentiment,
						Engagement: engagement,
						PostedAt:   postedAt,
					}
					result = append(result, socialData)
				}
			}
		}
	}

	return result, nil
}

// extractCryptoSymbols 从Reddit帖子中提取加密货币符号
func (r *RedditSource) extractCryptoSymbols(text string) []string {
	// 常见的加密货币符号
	cryptoSymbols := []string{
		"BTC", "ETH", "BNB", "ADA", "XRP", "SOL", "DOT", "DOGE", "AVAX",
		"MATIC", "LINK", "UNI", "ALGO", "VET", "ICP", "ETC", "XLM", "HBAR",
		"NEAR", "FLOW", "MANA", "SAND", "AXS", "CHZ", "ENJ", "APE", "LRC",
	}

	found := make([]string, 0)
	text = strings.ToUpper(text)

	for _, symbol := range cryptoSymbols {
		// 使用正则表达式匹配完整的单词
		pattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(symbol))
		matched, _ := regexp.MatchString(pattern, text)
		if matched {
			found = append(found, symbol)
		}
	}

	return found
}

// analyzeRedditSentiment 分析Reddit帖子的情感
func (r *RedditSource) analyzeRedditSentiment(text string) float64 {
	text = strings.ToLower(text)

	// Reddit特定的正面关键词
	positiveWords := []string{
		"bullish", "moon", "pump", "bull", "long", "buy", "hodl", "diamond",
		"hands", "rocket", "to the moon", "green", "up", "surge", "rally",
		"breakthrough", "partnership", "adoption", "upgrade", "milestone",
		"success", "profit", "gains", "bullish", "optimistic", "confident",
		"strong", "growing", "increasing", "positive", "great", "excellent",
		"amazing", "fantastic", "awesome", "love", "like", "good", "best",
	}

	// Reddit特定的负面关键词
	negativeWords := []string{
		"bearish", "dump", "crash", "fall", "decline", "drop", "loss", "bear",
		"short", "sell", "paper hands", "red", "down", "plunge", "collapse",
		"hack", "scam", "rug", "exploit", "vulnerable", "concern", "warning",
		"danger", "risk", "problem", "issue", "bad", "terrible", "awful",
		"hate", "dislike", "worst", "horrible", "disaster", "crash", "fail",
		"bearish", "pessimistic", "worried", "weak", "falling", "decreasing",
		"negative", "terrible", "horrible", "awful", "hate", "worst",
	}

	positiveCount := 0
	negativeCount := 0

	words := strings.Fields(text)

	for _, word := range words {
		word = strings.Trim(word, ".,!?\"'()[]{}")

		for _, positive := range positiveWords {
			if strings.Contains(word, positive) {
				positiveCount++
				break
			}
		}

		for _, negative := range negativeWords {
			if strings.Contains(word, negative) {
				negativeCount++
				break
			}
		}
	}

	total := positiveCount + negativeCount
	if total == 0 {
		return 0.5 // 中性
	}

	// 计算情感得分 (0-1)
	sentiment := float64(positiveCount) / float64(total)
	return sentiment
}

// RedditResponse Reddit API响应结构
type RedditResponse struct {
	Kind string `json:"kind"`
	Data struct {
		After     string       `json:"after"`
		Dist      int          `json:"dist"`
		Modhash   string       `json:"modhash"`
		GeoFilter string       `json:"geo_filter"`
		Children  []RedditPost `json:"children"`
	} `json:"data"`
}

// RedditPost Reddit帖子结构
type RedditPost struct {
	Kind string `json:"kind"`
	Data struct {
		Subreddit             string  `json:"subreddit"`
		Selftext              string  `json:"selftext"`
		AuthorFullname        string  `json:"author_fullname"`
		Title                 string  `json:"title"`
		SubredditNamePrefixed string  `json:"subreddit_name_prefixed"`
		Ups                   int     `json:"ups"`
		Downs                 int     `json:"downs"`
		NumComments           int     `json:"num_comments"`
		CreatedUTC            float64 `json:"created_utc"`
		Permalink             string  `json:"permalink"`
		URL                   string  `json:"url"`
		ID                    string  `json:"id"`
		Score                 int     `json:"score"`
	} `json:"data"`
}
