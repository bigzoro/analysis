package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// CoinGeckoClient CoinGecko免费API客户端
type CoinGeckoClient struct {
	baseURL    string
	httpClient *http.Client
	rateLimit  *RateLimiter
}

// CoinGeckoMarketData CoinGecko市场数据结构
type CoinGeckoMarketData struct {
	ID                    string  `json:"id"`
	Symbol                string  `json:"symbol"`
	Name                  string  `json:"name"`
	Image                 string  `json:"image"`
	CurrentPrice          float64 `json:"current_price"`
	MarketCap             float64 `json:"market_cap"`
	MarketCapRank         int     `json:"market_cap_rank"`
	FullyDilutedVal       float64 `json:"fully_diluted_valuation"`
	TotalVolume           float64 `json:"total_volume"`
	High24h               float64 `json:"high_24h"`
	Low24h                float64 `json:"low_24h"`
	PriceChange24h        float64 `json:"price_change_24h"`
	PriceChangePct24h     float64 `json:"price_change_percentage_24h"`
	MarketCapChange24h    float64 `json:"market_cap_change_24h"`
	MarketCapChangePct24h float64 `json:"market_cap_change_percentage_24h"`
	CirculatingSupply     float64 `json:"circulating_supply"`
	TotalSupply           float64 `json:"total_supply"`
	MaxSupply             float64 `json:"max_supply"`
	ATH                   float64 `json:"ath"`
	ATHChangePct          float64 `json:"ath_change_percentage"`
	ATHDate               string  `json:"ath_date"`
	ATL                   float64 `json:"atl"`
	ATLChangePct          float64 `json:"atl_change_percentage"`
	ATLDate               string  `json:"atl_date"`
	LastUpdated           string  `json:"last_updated"`
}

// CoinGeckoSearchResult 搜索结果
type CoinGeckoSearchResult struct {
	Coins []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Symbol string `json:"symbol"`
		Thumb  string `json:"thumb"`
	} `json:"coins"`
}

// CoinGeckoTrending 热门币种
type CoinGeckoTrending struct {
	Coins []struct {
		Item struct {
			ID            string  `json:"id"`
			CoinID        int     `json:"coin_id"`
			Name          string  `json:"name"`
			Symbol        string  `json:"symbol"`
			MarketCapRank int     `json:"market_cap_rank"`
			Thumb         string  `json:"thumb"`
			Small         string  `json:"small"`
			Large         string  `json:"large"`
			Slug          string  `json:"slug"`
			PriceBTC      float64 `json:"price_btc"`
			Score         int     `json:"score"`
		} `json:"item"`
	} `json:"coins"`
}

// RateLimiter 简单的速率限制器
type RateLimiter struct {
	requests  int
	resetTime time.Time
	limit     int           // 每分钟最大请求数
	window    time.Duration // 时间窗口
}

// NewRateLimiter 创建速率限制器
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests:  0,
		resetTime: time.Now().Add(window),
		limit:     limit,
		window:    window,
	}
}

// Allow 检查是否允许请求
func (rl *RateLimiter) Allow() bool {
	now := time.Now()

	// 如果超过时间窗口，重置计数器
	if now.After(rl.resetTime) {
		rl.requests = 0
		rl.resetTime = now.Add(rl.window)
	}

	// 检查是否超过限制
	if rl.requests >= rl.limit {
		return false
	}

	rl.requests++
	return true
}

// NewCoinGeckoClient 创建CoinGecko客户端
func NewCoinGeckoClient() *CoinGeckoClient {
	// CoinGecko免费版限制：每分钟30次请求
	rateLimiter := NewRateLimiter(25, time.Minute) // 留5次缓冲

	return &CoinGeckoClient{
		baseURL: "https://api.coingecko.com/api/v3",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		rateLimit: rateLimiter,
	}
}

// GetMarketData 获取市场数据
// 支持参数：
// - ids: 币种ID列表，用逗号分隔
// - vs_currency: 计价货币 (usd, btc, eth等)
// - order: 排序方式 (market_cap_desc, volume_desc等)
// - per_page: 每页数量 (1-250)
// - page: 页码
// - sparkline: 是否包含价格图表
func (cg *CoinGeckoClient) GetMarketData(ctx context.Context, params map[string]string) ([]CoinGeckoMarketData, error) {
	if !cg.rateLimit.Allow() {
		return nil, fmt.Errorf("API速率限制：已达到每分钟请求上限")
	}

	// 构建请求参数
	defaultParams := map[string]string{
		"vs_currency": "usd",
		"order":       "market_cap_desc",
		"per_page":    "100",
		"page":        "1",
		"sparkline":   "false",
	}

	// 合并参数
	for k, v := range params {
		defaultParams[k] = v
	}

	// 构建查询字符串
	var queryParts []string
	for k, v := range defaultParams {
		queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, v))
	}
	queryString := strings.Join(queryParts, "&")

	url := fmt.Sprintf("%s/coins/markets?%s", cg.baseURL, queryString)

	log.Printf("[CoinGecko] 请求市场数据: %s", url)

	var result []CoinGeckoMarketData
	err := cg.makeRequest(ctx, url, &result)
	if err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	return result, nil
}

// GetCoinBySymbol 通过符号获取币种信息
func (cg *CoinGeckoClient) GetCoinBySymbol(ctx context.Context, symbol string) (*CoinGeckoMarketData, error) {
	// 先搜索币种ID
	searchResult, err := cg.SearchCoins(ctx, symbol)
	if err != nil {
		return nil, err
	}

	if len(searchResult.Coins) == 0 {
		return nil, fmt.Errorf("未找到符号为 %s 的币种", symbol)
	}

	// 获取最匹配的结果（通常是第一个）
	coinID := searchResult.Coins[0].ID

	// 获取单个币种的详细数据
	marketData, err := cg.GetMarketData(ctx, map[string]string{
		"ids": coinID,
	})
	if err != nil {
		return nil, err
	}

	if len(marketData) == 0 {
		return nil, fmt.Errorf("未找到ID为 %s 的币种数据", coinID)
	}

	return &marketData[0], nil
}

// SearchCoins 搜索币种
func (cg *CoinGeckoClient) SearchCoins(ctx context.Context, query string) (*CoinGeckoSearchResult, error) {
	if !cg.rateLimit.Allow() {
		return nil, fmt.Errorf("API速率限制：已达到每分钟请求上限")
	}

	url := fmt.Sprintf("%s/search?query=%s", cg.baseURL, query)

	var result CoinGeckoSearchResult
	err := cg.makeRequest(ctx, url, &result)
	if err != nil {
		return nil, fmt.Errorf("搜索币种失败: %w", err)
	}

	return &result, nil
}

// GetTrending 获取热门币种
func (cg *CoinGeckoClient) GetTrending(ctx context.Context) (*CoinGeckoTrending, error) {
	if !cg.rateLimit.Allow() {
		return nil, fmt.Errorf("API速率限制：已达到每分钟请求上限")
	}

	url := fmt.Sprintf("%s/search/trending", cg.baseURL)

	var result CoinGeckoTrending
	err := cg.makeRequest(ctx, url, &result)
	if err != nil {
		return nil, fmt.Errorf("获取热门币种失败: %w", err)
	}

	return &result, nil
}

// GetPriceHistory 获取价格历史数据
// 支持参数：
// - id: 币种ID
// - vs_currency: 计价货币
// - days: 时间范围 (1,7,14,30,90,180,365,max)
func (cg *CoinGeckoClient) GetPriceHistory(ctx context.Context, coinID, vsCurrency string, days int) (map[string]interface{}, error) {
	if !cg.rateLimit.Allow() {
		return nil, fmt.Errorf("API速率限制：已达到每分钟请求上限")
	}

	daysStr := strconv.Itoa(days)
	url := fmt.Sprintf("%s/coins/%s/market_chart?vs_currency=%s&days=%s",
		cg.baseURL, coinID, vsCurrency, daysStr)

	var result map[string]interface{}
	err := cg.makeRequest(ctx, url, &result)
	if err != nil {
		return nil, fmt.Errorf("获取价格历史失败: %w", err)
	}

	return result, nil
}

// GetGlobalMarketData 获取全球市场数据
func (cg *CoinGeckoClient) GetGlobalMarketData(ctx context.Context) (map[string]interface{}, error) {
	if !cg.rateLimit.Allow() {
		return nil, fmt.Errorf("API速率限制：已达到每分钟请求上限")
	}

	url := fmt.Sprintf("%s/global", cg.baseURL)

	var result map[string]interface{}
	err := cg.makeRequest(ctx, url, &result)
	if err != nil {
		return nil, fmt.Errorf("获取全球市场数据失败: %w", err)
	}

	return result, nil
}

// makeRequest 发送HTTP请求的通用方法
func (cg *CoinGeckoClient) makeRequest(ctx context.Context, url string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置User-Agent（CoinGecko推荐）
	req.Header.Set("User-Agent", "CoinAnalysis/1.0")

	resp, err := cg.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送请求失败: %w", err)
	}
	defer resp.Body.Close()

	// 检查响应状态
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API响应错误 %d: %s", resp.StatusCode, string(body))
	}

	// 解析JSON响应
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

// GetSupportedCurrencies 获取支持的货币列表
func (cg *CoinGeckoClient) GetSupportedCurrencies(ctx context.Context) ([]string, error) {
	if !cg.rateLimit.Allow() {
		return nil, fmt.Errorf("API速率限制：已达到每分钟请求上限")
	}

	url := fmt.Sprintf("%s/simple/supported_vs_currencies", cg.baseURL)

	var result []string
	err := cg.makeRequest(ctx, url, &result)
	if err != nil {
		return nil, fmt.Errorf("获取支持货币失败: %w", err)
	}

	return result, nil
}

// Ping 测试API连接
func (cg *CoinGeckoClient) Ping(ctx context.Context) error {
	if !cg.rateLimit.Allow() {
		return fmt.Errorf("API速率限制：已达到每分钟请求上限")
	}

	url := fmt.Sprintf("%s/ping", cg.baseURL)

	var result map[string]string
	err := cg.makeRequest(ctx, url, &result)
	if err != nil {
		return fmt.Errorf("API连接测试失败: %w", err)
	}

	if result["gecko_says"] != "(V3) To the Moon!" {
		return fmt.Errorf("API响应异常")
	}

	return nil
}

// ConvertToMarketDataPoint 将CoinGecko数据转换为内部格式
func (cg *CoinGeckoClient) ConvertToMarketDataPoint(cgData CoinGeckoMarketData) MarketDataPoint {
	return MarketDataPoint{
		Symbol:         strings.ToUpper(cgData.Symbol),
		BaseSymbol:     strings.ToUpper(cgData.Symbol),
		Price:          cgData.CurrentPrice,
		PriceChange24h: cgData.PriceChangePct24h,
		Volume24h:      cgData.TotalVolume,
		MarketCap:      &cgData.MarketCap,
		Timestamp:      time.Now(), // CoinGecko数据是实时的
	}
}
