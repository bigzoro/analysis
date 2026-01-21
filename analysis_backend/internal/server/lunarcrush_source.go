package server

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// LunarCrushSource LunarCrush数据源 (免费版)
type LunarCrushSource struct {
	httpClient *HTTPClient
	baseURL    string
}

// NewLunarCrushSource 创建LunarCrush数据源
func NewLunarCrushSource() *LunarCrushSource {
	return &LunarCrushSource{
		httpClient: NewHTTPClient(),
		baseURL:    "https://api.lunarcrush.com/v2",
	}
}

// NewLunarCrushSourceWithKey 使用API Key创建LunarCrush数据源
func NewLunarCrushSourceWithKey(apiKey string) *LunarCrushSource {
	return &LunarCrushSource{
		httpClient: NewHTTPClient(),
		baseURL:    "https://api.lunarcrush.com/v2",
	}
}

// Name 数据源名称
func (lc *LunarCrushSource) Name() string {
	return "lunarcrush"
}

// IsAvailable 检查数据源是否可用
func (lc *LunarCrushSource) IsAvailable() bool {
	// LunarCrush免费版有速率限制，但基本可用
	url := fmt.Sprintf("%s/public/coins/list", lc.baseURL)
	_, err := lc.httpClient.Get(url)
	return err == nil
}

// FetchMarketData 获取市场数据
func (lc *LunarCrushSource) FetchMarketData(ctx context.Context, symbols []string) ([]MarketData, error) {
	// LunarCrush免费版主要提供社交数据，但也有一点市场数据
	result := make([]MarketData, 0)

	for _, symbol := range symbols {
		coinData, err := lc.fetchCoinData(symbol)
		if err != nil {
			continue
		}

		if coinData != nil {
			marketData := MarketData{
				Symbol:      strings.ToUpper(symbol),
				Source:      lc.Name(),
				Price:       coinData.Price,
				Volume24h:   coinData.Volume24h,
				MarketCap:   coinData.MarketCap,
				LastUpdated: time.Now(),
			}

			// LunarCrush可能没有价格变化数据，设为空
			result = append(result, marketData)
		}

		// 避免API限速
		time.Sleep(300 * time.Millisecond)
	}

	return result, nil
}

// FetchNewsData 获取新闻数据
func (lc *LunarCrushSource) FetchNewsData(ctx context.Context, symbols []string) ([]NewsData, error) {
	// LunarCrush免费版不支持新闻数据
	return []NewsData{}, nil
}

// FetchSocialData 获取社交数据
func (lc *LunarCrushSource) FetchSocialData(ctx context.Context, symbols []string) ([]SocialData, error) {
	result := make([]SocialData, 0)

	for _, symbol := range symbols {
		coinData, err := lc.fetchCoinData(symbol)
		if err != nil {
			continue
		}

		if coinData != nil {
			socialData := SocialData{
				Platform:   "lunarcrush",
				Symbol:     strings.ToUpper(symbol),
				Mentions:   coinData.SocialVolume,
				Sentiment:  coinData.Sentiment,
				Engagement: coinData.SocialEngagement,
				PostedAt:   time.Now(),
			}
			result = append(result, socialData)
		}

		// 避免API限速
		time.Sleep(300 * time.Millisecond)
	}

	return result, nil
}

// fetchCoinData 获取单个币种数据
func (lc *LunarCrushSource) fetchCoinData(symbol string) (*LunarCrushCoinData, error) {
	// 将符号转换为LunarCrush的格式
	coinId := lc.symbolToCoinId(symbol)
	if coinId == "" {
		return nil, fmt.Errorf("unsupported symbol: %s", symbol)
	}

	url := fmt.Sprintf("%s/public/coins/%s", lc.baseURL, coinId)

	data, err := lc.httpClient.Get(url)
	if err != nil {
		return nil, err
	}

	var response LunarCrushCoinResponse
	if err := json.Unmarshal(data, &response); err != nil {
		return nil, err
	}

	if len(response.Data) > 0 {
		return &response.Data[0], nil
	}

	return nil, fmt.Errorf("no data found for %s", symbol)
}

// symbolToCoinId 将币种符号转换为LunarCrush coin ID
func (lc *LunarCrushSource) symbolToCoinId(symbol string) string {
	// LunarCrush使用数字ID
	coinIdMap := map[string]string{
		"BTC":   "1",    // Bitcoin
		"ETH":   "2",    // Ethereum
		"BNB":   "1839", // Binance Coin
		"ADA":   "2010", // Cardano
		"XRP":   "52",   // Ripple
		"SOL":   "5426", // Solana
		"DOT":   "6636", // Polkadot
		"DOGE":  "74",   // Dogecoin
		"AVAX":  "5805", // Avalanche
		"MATIC": "3890", // Polygon
		"LINK":  "1975", // Chainlink
		"UNI":   "7083", // Uniswap
		"ALGO":  "4030", // Algorand
	}

	symbol = strings.ToUpper(symbol)
	if coinId, exists := coinIdMap[symbol]; exists {
		return coinId
	}

	return ""
}

// LunarCrushCoinResponse LunarCrush API响应结构
type LunarCrushCoinResponse struct {
	Data []LunarCrushCoinData `json:"data"`
}

// LunarCrushCoinData 币种数据结构
type LunarCrushCoinData struct {
	ID               int     `json:"id"`
	Symbol           string  `json:"symbol"`
	Name             string  `json:"name"`
	Price            float64 `json:"price"`
	PriceBTC         float64 `json:"price_btc"`
	MarketCap        float64 `json:"market_cap"`
	Volume24h        float64 `json:"volume_24h"`
	Volatility       float64 `json:"volatility"`
	PercentChange24h float64 `json:"percent_change_24h"`
	PercentChange7d  float64 `json:"percent_change_7d"`
	PercentChange30d float64 `json:"percent_change_30d"`
	SocialVolume     int     `json:"social_volume"`
	SocialEngagement int     `json:"social_engagement"`
	SocialSentiment  float64 `json:"social_sentiment"`
	SocialDominance  float64 `json:"social_dominance"`
	MarketDominance  float64 `json:"market_dominance"`
	Sentiment        float64 `json:"sentiment"` // 综合情感得分
	GalaxyScore      float64 `json:"galaxy_score"`
	NewsCount        int     `json:"news_count"`
}
