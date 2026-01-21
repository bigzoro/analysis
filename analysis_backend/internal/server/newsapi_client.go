package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// NewsAPIClient NewsAPI免费客户端
type NewsAPIClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	rateLimit  *RateLimiter
}

// NewsAPIArticle NewsAPI文章结构
type NewsAPIArticle struct {
	Source struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"source"`
	Author      string    `json:"author"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	URL         string    `json:"url"`
	URLToImage  string    `json:"urlToImage"`
	PublishedAt time.Time `json:"publishedAt"`
	Content     string    `json:"content"`
}

// NewsAPIResponse NewsAPI响应结构
type NewsAPIResponse struct {
	Status       string           `json:"status"`
	TotalResults int              `json:"totalResults"`
	Articles     []NewsAPIArticle `json:"articles"`
}

// AnnouncementData 公告数据结构
type AnnouncementData struct {
	Symbol      string
	Title       string
	Description string
	URL         string
	PublishedAt time.Time
	Source      string
	Relevance   float64 // 相关性评分 (0-1)
	Sentiment   string  // "positive", "negative", "neutral"
}

// NewNewsAPIClient 创建NewsAPI客户端
func NewNewsAPIClient(apiKey string) *NewsAPIClient {
	// NewsAPI免费版限制：每天100次请求
	rateLimiter := NewRateLimiter(80, 24*time.Hour) // 每天80次，留20次缓冲

	return &NewsAPIClient{
		apiKey:  apiKey,
		baseURL: "https://newsapi.org/v2",
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
		rateLimit: rateLimiter,
	}
}

// GetCryptoNews 获取加密货币新闻
func (n *NewsAPIClient) GetCryptoNews(ctx context.Context, symbol string, days int) ([]AnnouncementData, error) {
	if !n.rateLimit.Allow() {
		return nil, fmt.Errorf("API速率限制：已达到每日请求上限")
	}

	// 构建查询参数
	query := n.buildCryptoQuery(symbol)
	fromDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	params := url.Values{}
	params.Add("q", query)
	params.Add("from", fromDate)
	params.Add("sortBy", "publishedAt")
	params.Add("language", "en")
	params.Add("pageSize", "20") // 最多20条

	url := fmt.Sprintf("%s/everything?%s&apiKey=%s", n.baseURL, params.Encode(), n.apiKey)

	log.Printf("[NewsAPI] 查询%s相关新闻: %s", symbol, query)

	var response NewsAPIResponse
	err := n.makeRequest(ctx, url, &response)
	if err != nil {
		return nil, fmt.Errorf("获取新闻失败: %w", err)
	}

	if response.Status != "ok" {
		return nil, fmt.Errorf("NewsAPI响应错误: %s", response.Status)
	}

	// 转换为公告数据格式
	announcements := make([]AnnouncementData, 0, len(response.Articles))
	for _, article := range response.Articles {
		announcement := AnnouncementData{
			Symbol:      strings.ToUpper(symbol),
			Title:       article.Title,
			Description: article.Description,
			URL:         article.URL,
			PublishedAt: article.PublishedAt,
			Source:      article.Source.Name,
		}

		// 计算相关性
		announcement.Relevance = n.calculateRelevance(article.Title+" "+article.Description, symbol)

		// 分析情感
		announcement.Sentiment = n.analyzeSentiment(article.Title + " " + article.Description)

		// 只保留相关性较高的新闻
		if announcement.Relevance > 0.3 {
			announcements = append(announcements, announcement)
		}
	}

	log.Printf("[NewsAPI] 为%s找到%d条相关新闻", symbol, len(announcements))
	return announcements, nil
}

// GetTopCryptoHeadlines 获取热门加密货币头条
func (n *NewsAPIClient) GetTopCryptoHeadlines(ctx context.Context) ([]AnnouncementData, error) {
	if !n.rateLimit.Allow() {
		return nil, fmt.Errorf("API速率限制：已达到每日请求上限")
	}

	params := url.Values{}
	params.Add("q", "cryptocurrency OR bitcoin OR ethereum OR blockchain")
	params.Add("sortBy", "publishedAt")
	params.Add("language", "en")
	params.Add("pageSize", "10")

	url := fmt.Sprintf("%s/everything?%s&apiKey=%s", n.baseURL, params.Encode(), n.apiKey)

	var response NewsAPIResponse
	err := n.makeRequest(ctx, url, &response)
	if err != nil {
		return nil, fmt.Errorf("获取头条新闻失败: %w", err)
	}

	announcements := make([]AnnouncementData, 0, len(response.Articles))
	for _, article := range response.Articles {
		announcement := AnnouncementData{
			Title:       article.Title,
			Description: article.Description,
			URL:         article.URL,
			PublishedAt: article.PublishedAt,
			Source:      article.Source.Name,
			Relevance:   0.8, // 头条新闻默认较高相关性
			Sentiment:   n.analyzeSentiment(article.Title + " " + article.Description),
		}

		announcements = append(announcements, announcement)
	}

	return announcements, nil
}

// buildCryptoQuery 构建加密货币查询
func (n *NewsAPIClient) buildCryptoQuery(symbol string) string {
	symbol = strings.ToUpper(symbol)

	// 为主要币种构建更精确的查询
	switch symbol {
	case "BTC", "BITCOIN":
		return "bitcoin OR BTC OR cryptocurrency"
	case "ETH", "ETHEREUM":
		return "ethereum OR ETH OR cryptocurrency"
	case "BNB":
		return "binance OR BNB OR cryptocurrency"
	case "ADA":
		return "cardano OR ADA OR cryptocurrency"
	case "SOL":
		return "solana OR SOL OR cryptocurrency"
	case "DOT":
		return "polkadot OR DOT OR cryptocurrency"
	case "LINK":
		return "chainlink OR LINK OR cryptocurrency"
	case "UNI":
		return "uniswap OR UNI OR DeFi"
	case "AAVE":
		return "aave OR DeFi OR lending"
	default:
		// 对于其他币种，使用通用查询
		return fmt.Sprintf("%s OR %s OR cryptocurrency OR blockchain",
			symbol, strings.ToLower(symbol))
	}
}

// calculateRelevance 计算相关性
func (n *NewsAPIClient) calculateRelevance(text, symbol string) float64 {
	text = strings.ToLower(text)
	symbolLower := strings.ToLower(symbol)

	// 直接提到币种名称
	if strings.Contains(text, symbolLower) {
		return 0.9
	}

	// 包含相关关键词
	cryptoKeywords := []string{"crypto", "cryptocurrency", "blockchain", "bitcoin", "ethereum", "defi", "nft"}
	for _, keyword := range cryptoKeywords {
		if strings.Contains(text, keyword) {
			return 0.7
		}
	}

	// 包含价格或市场相关词汇
	marketKeywords := []string{"price", "market", "trading", "exchange", "coin", "token"}
	for _, keyword := range marketKeywords {
		if strings.Contains(text, keyword) {
			return 0.5
		}
	}

	return 0.1 // 默认低相关性
}

// analyzeSentiment 简单情感分析
func (n *NewsAPIClient) analyzeSentiment(text string) string {
	text = strings.ToLower(text)

	positiveWords := []string{"surge", "rally", "bull", "bullish", "rise", "up", "gain", "growth", "success", "breakthrough", "adoption", "partnership"}
	negativeWords := []string{"crash", "dump", "bear", "bearish", "fall", "down", "loss", "decline", "hack", "scam", "ban", "regulation"}

	positiveCount := 0
	negativeCount := 0

	words := strings.Fields(text)
	for _, word := range words {
		word = strings.Trim(word, ".,!?\"'")

		for _, pos := range positiveWords {
			if strings.Contains(word, pos) {
				positiveCount++
				break
			}
		}

		for _, neg := range negativeWords {
			if strings.Contains(word, neg) {
				negativeCount++
				break
			}
		}
	}

	if positiveCount > negativeCount {
		return "positive"
	} else if negativeCount > positiveCount {
		return "negative"
	} else {
		return "neutral"
	}
}

// ConvertToAnnouncementScore 转换为公告评分
func (n *NewsAPIClient) ConvertToAnnouncementScore(announcements []AnnouncementData) AnnouncementScore {
	if len(announcements) == 0 {
		return AnnouncementScore{
			TotalScore: 0.0,
			Importance: "low",
		}
	}

	// 计算综合评分
	totalRelevance := 0.0
	positiveCount := 0
	negativeCount := 0
	recentCount := 0 // 最近7天内的新闻

	now := time.Now()
	weekAgo := now.AddDate(0, 0, -7)

	for _, ann := range announcements {
		totalRelevance += ann.Relevance

		if ann.Sentiment == "positive" {
			positiveCount++
		} else if ann.Sentiment == "negative" {
			negativeCount++
		}

		if ann.PublishedAt.After(weekAgo) {
			recentCount++
		}
	}

	avgRelevance := totalRelevance / float64(len(announcements))

	// 基础评分：相关性权重0.6，情感权重0.3，新鲜度权重0.1
	score := avgRelevance * 0.6

	// 情感影响
	if positiveCount > negativeCount {
		score += 0.2
	} else if negativeCount > positiveCount {
		score -= 0.2
	}

	// 新鲜度加成
	if recentCount > 0 {
		score += 0.1
	}

	// 确定重要性级别
	var importance string
	switch {
	case score > 0.8:
		importance = "high"
	case score > 0.5:
		importance = "medium"
	default:
		importance = "low"
	}

	return AnnouncementScore{
		TotalScore: score,
		Importance: importance,
	}
}

// makeRequest 发送HTTP请求
func (n *NewsAPIClient) makeRequest(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	req.Header.Set("User-Agent", "CryptoAnalysis/1.0")

	resp, err := n.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API响应错误 %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	err = json.Unmarshal(body, result)
	if err != nil {
		return fmt.Errorf("解析JSON失败: %w", err)
	}

	return nil
}

// GetRemainingRequests 获取剩余请求次数（估算）
func (n *NewsAPIClient) GetRemainingRequests() int {
	// 简单的估算，实际应该从响应头获取
	return 100 - n.rateLimit.requests
}
