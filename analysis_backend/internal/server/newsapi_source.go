package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// NewsAPISource NewsAPI数据源
type NewsAPISource struct {
	httpClient *HTTPClient
	apiKey     string // 需要从newsapi.org获取免费API Key
	baseURL    string
}

// NewNewsAPISource 创建NewsAPI数据源
func NewNewsAPISource() *NewsAPISource {
	return &NewsAPISource{
		httpClient: NewHTTPClient(),
		baseURL:    "https://newsapi.org/v2",
		apiKey:     "", // 需要配置API Key
	}
}

// NewNewsAPISourceWithKey 使用API Key创建NewsAPI数据源
func NewNewsAPISourceWithKey(apiKey string) *NewsAPISource {
	return &NewsAPISource{
		httpClient: NewHTTPClient(),
		baseURL:    "https://newsapi.org/v2",
		apiKey:     apiKey,
	}
}

// Name 数据源名称
func (na *NewsAPISource) Name() string {
	return "newsapi"
}

// IsAvailable 检查数据源是否可用
func (na *NewsAPISource) IsAvailable() bool {
	// 检查是否有API Key
	if na.apiKey == "" {
		return false
	}

	// 测试API连通性 (免费版每天100次请求)
	url := fmt.Sprintf("%s/top-headlines?country=us&apiKey=%s", na.baseURL, na.apiKey)
	_, err := na.httpClient.Get(url)
	return err == nil
}

// FetchMarketData NewsAPI不支持市场数据
func (na *NewsAPISource) FetchMarketData(ctx context.Context, symbols []string) ([]MarketData, error) {
	return []MarketData{}, nil
}

// FetchNewsData 获取新闻数据
func (na *NewsAPISource) FetchNewsData(ctx context.Context, symbols []string) ([]NewsData, error) {
	if na.apiKey == "" {
		return nil, fmt.Errorf("NewsAPI key not configured")
	}

	result := make([]NewsData, 0)

	// 为每个币种搜索相关新闻
	for _, symbol := range symbols {
		news, err := na.searchCryptoNews(symbol)
		if err != nil {
			continue // 跳过错误，继续下一个币种
		}

		result = append(result, news...)

		// 避免API限速
		time.Sleep(200 * time.Millisecond)
	}

	return result, nil
}

// FetchSocialData NewsAPI不支持社交数据
func (na *NewsAPISource) FetchSocialData(ctx context.Context, symbols []string) ([]SocialData, error) {
	return []SocialData{}, nil
}

// searchCryptoNews 搜索加密货币相关新闻
func (na *NewsAPISource) searchCryptoNews(symbol string) ([]NewsData, error) {
	// 构建搜索查询
	query := na.buildCryptoQuery(symbol)

	url := fmt.Sprintf("%s/everything?q=%s&language=en&sortBy=publishedAt&pageSize=10&apiKey=%s",
		na.baseURL, query, na.apiKey)

	data, err := na.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch news for %s: %w", symbol, err)
	}

	var response NewsAPIResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, fmt.Errorf("failed to parse NewsAPI response: %w", err)
	}

	result := make([]NewsData, 0, len(response.Articles))

	for _, article := range response.Articles {
		// 提取文章中的币种符号
		symbols := na.extractCryptoSymbols(article.Title + " " + article.Description)

		if len(symbols) > 0 {
			// 分析情感 (简单的关键词分析)
			sentiment := na.analyzeSentiment(article.Title + " " + article.Description)

			newsData := NewsData{
				Title:       article.Title,
				Content:     article.Description,
				URL:         article.URL,
				Source:      article.Source.Name,
				Symbols:     symbols,
				Sentiment:   sentiment,
				PublishedAt: article.PublishedAt,
			}

			result = append(result, newsData)
		}
	}

	return result, nil
}

// buildCryptoQuery 构建加密货币搜索查询
func (na *NewsAPISource) buildCryptoQuery(symbol string) string {
	// 为不同币种构建合适的搜索关键词
	cryptoQueries := map[string]string{
		"BTC":   "bitcoin OR BTC",
		"ETH":   "ethereum OR ETH",
		"BNB":   "binance OR BNB",
		"ADA":   "cardano OR ADA",
		"XRP":   "ripple OR XRP",
		"SOL":   "solana OR SOL",
		"DOT":   "polkadot OR DOT",
		"DOGE":  "dogecoin OR DOGE",
		"AVAX":  "avalanche OR AVAX",
		"MATIC": "polygon OR MATIC",
		"LINK":  "chainlink OR LINK",
		"UNI":   "uniswap OR UNI",
		"ALGO":  "algorand OR ALGO",
	}

	symbol = strings.ToUpper(symbol)
	if query, exists := cryptoQueries[symbol]; exists {
		return fmt.Sprintf("(%s) AND (cryptocurrency OR crypto OR blockchain)", query)
	}

	// 默认查询
	return fmt.Sprintf("%s AND (cryptocurrency OR crypto OR blockchain)", symbol)
}

// extractCryptoSymbols 从文本中提取加密货币符号
func (na *NewsAPISource) extractCryptoSymbols(text string) []string {
	// 常见的加密货币符号正则表达式
	cryptoSymbols := []string{
		"BTC", "ETH", "BNB", "ADA", "XRP", "SOL", "DOT", "DOGE", "AVAX",
		"MATIC", "LINK", "UNI", "ALGO", "VET", "ICP", "ETC", "XLM", "HBAR",
		"NEAR", "FLOW", "MANA", "SAND", "AXS", "CHZ", "ENJ", "APE", "LRC",
	}

	found := make([]string, 0)
	text = strings.ToUpper(text)

	for _, symbol := range cryptoSymbols {
		if strings.Contains(text, symbol) {
			found = append(found, symbol)
		}
	}

	return found
}

// analyzeSentiment 简单的情感分析
func (na *NewsAPISource) analyzeSentiment(text string) float64 {
	text = strings.ToLower(text)

	// 正面关键词
	positiveWords := []string{
		"bullish", "surge", "rally", "breakthrough", "partnership", "adoption",
		"upgrade", "milestone", "growth", "success", "positive", "gains",
		"moon", "pump", "bull", "rise", "up", "high", "increase", "profit",
	}

	// 负面关键词
	negativeWords := []string{
		"bearish", "crash", "dump", "fall", "decline", "drop", "loss",
		"bear", "sell", "short", "down", "low", "decrease", "hack",
		"scam", "rug", "exploit", "vulnerability", "concern", "warning",
	}

	positiveCount := 0
	negativeCount := 0

	words := strings.Fields(text)

	for _, word := range words {
		word = strings.Trim(word, ".,!?\"'")

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
