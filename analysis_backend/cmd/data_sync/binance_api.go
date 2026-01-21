package main

import (
	"context"
	"fmt"
	"strings"
	"time"

	"analysis/internal/analysis"
	"analysis/internal/netutil"
)

// BinanceAPIClient 统一的Binance API客户端
type BinanceAPIClient struct {
	baseURLs map[string]string
	timeout  time.Duration // API调用超时时间
	// 统计回调函数，用于记录API调用统计信息
	onAPICall func(success bool, latency time.Duration, kind string)
}

// NewBinanceAPIClient 创建Binance API客户端
func NewBinanceAPIClient() *BinanceAPIClient {
	return &BinanceAPIClient{
		baseURLs: map[string]string{
			"spot":    "https://api.binance.com/api/v3",
			"futures": "https://fapi.binance.com/fapi/v1",
		},
		timeout:   10 * time.Second, // 默认10秒超时
		onAPICall: nil,              // 默认不记录统计信息
	}
}

// NewBinanceAPIClientWithStats 创建带有统计功能的Binance API客户端
func NewBinanceAPIClientWithStats(onAPICall func(success bool, latency time.Duration, kind string)) *BinanceAPIClient {
	return &BinanceAPIClient{
		baseURLs: map[string]string{
			"spot":    "https://api.binance.com/api/v3",
			"futures": "https://fapi.binance.com/fapi/v1",
		},
		timeout:   10 * time.Second, // 默认10秒超时
		onAPICall: onAPICall,
	}
}

// NewBinanceAPIClientWithConfig 使用配置创建Binance API客户端
func NewBinanceAPIClientWithConfig(config *DataSyncConfig, onAPICall func(success bool, latency time.Duration, kind string)) *BinanceAPIClient {
	return &BinanceAPIClient{
		baseURLs: map[string]string{
			"spot":    "https://api.binance.com/api/v3",
			"futures": "https://fapi.binance.com/fapi/v1",
		},
		timeout:   time.Duration(config.Timeouts.APICallTimeout) * time.Second,
		onAPICall: onAPICall,
	}
}

// FetchKlines 获取K线数据
func (c *BinanceAPIClient) FetchKlines(ctx context.Context, symbol, kind, interval string, limit int) ([]analysis.KlineDataAPI, error) {
	// 参数验证
	if symbol == "" || kind == "" || interval == "" {
		return nil, fmt.Errorf("invalid parameters: symbol=%s, kind=%s, interval=%s", symbol, kind, interval)
	}
	if limit <= 0 || limit > 1000 {
		limit = 100 // 默认限制100条
	}

	// 获取基础URL
	baseURL, exists := c.baseURLs[kind]
	if !exists {
		return nil, fmt.Errorf("unsupported market kind: %s", kind)
	}

	// 构建完整的API URL
	url := fmt.Sprintf("%s/klines?symbol=%s&interval=%s&limit=%d",
		baseURL, strings.ToUpper(symbol), interval, limit)

	// 等待获取API调用令牌（速率限制）
	if err := c.getRateLimiter(kind).WaitForToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to acquire rate limit token for %s: %w", kind, err)
	}

	// 设置超时时间
	apiCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 记录API调用开始时间
	startTime := time.Now()

	// 调用Binance API
	// Binance API返回的是[][]interface{}格式
	var rawKlines [][]interface{}
	err := netutil.GetJSON(apiCtx, url, &rawKlines)

	// 记录API调用统计信息
	latency := time.Since(startTime)
	success := err == nil
	if c.onAPICall != nil {
		c.onAPICall(success, latency, kind)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to fetch klines from Binance %s API: %w", kind, err)
	}

	// 转换为内部格式
	klines := make([]analysis.KlineDataAPI, 0, len(rawKlines))
	for _, raw := range rawKlines {
		if len(raw) < 12 {
			continue // 跳过不完整的K线数据
		}

		kline := analysis.KlineDataAPI{
			OpenTime: int64(raw[0].(float64)),
			Open:     raw[1].(string),
			High:     raw[2].(string),
			Low:      raw[3].(string),
			Close:    raw[4].(string),
			Volume:   raw[5].(string),
		}

		// KlineDataAPI结构只包含基本字段，不包含可选字段
		// 如果需要更多字段，可以考虑扩展结构或使用不同的类型

		klines = append(klines, kline)
	}

	return klines, nil
}

// FetchDepth 获取深度数据
func (c *BinanceAPIClient) FetchDepth(ctx context.Context, symbol, kind string, limit int) (map[string]interface{}, error) {
	// 参数验证
	if symbol == "" || kind == "" {
		return nil, fmt.Errorf("invalid parameters: symbol=%s, kind=%s", symbol, kind)
	}
	if limit <= 0 || limit > 5000 {
		limit = 100 // 默认限制100档
	}

	// 获取基础URL
	baseURL, exists := c.baseURLs[kind]
	if !exists {
		return nil, fmt.Errorf("unsupported market kind: %s", kind)
	}

	// 构建完整的API URL
	url := fmt.Sprintf("%s/depth?symbol=%s&limit=%d", baseURL, strings.ToUpper(symbol), limit)

	// 等待获取API调用令牌（速率限制）
	if err := c.getRateLimiter(kind).WaitForToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to acquire rate limit token for %s: %w", kind, err)
	}

	// 设置超时时间
	apiCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 调用Binance API
	var depth map[string]interface{}
	if err := netutil.GetJSON(apiCtx, url, &depth); err != nil {
		return nil, fmt.Errorf("failed to fetch depth from Binance %s API: %w", kind, err)
	}

	return depth, nil
}

// FetchPrice 获取价格数据
func (c *BinanceAPIClient) FetchPrice(ctx context.Context, symbol, kind string) (map[string]string, error) {
	// 参数验证
	if symbol == "" || kind == "" {
		return nil, fmt.Errorf("invalid parameters: symbol=%s, kind=%s", symbol, kind)
	}

	// 获取基础URL
	baseURL, exists := c.baseURLs[kind]
	if !exists {
		return nil, fmt.Errorf("unsupported market kind: %s", kind)
	}

	// 构建完整的API URL
	url := fmt.Sprintf("%s/ticker/price?symbol=%s", baseURL, strings.ToUpper(symbol))

	// 等待获取API调用令牌（速率限制）
	if err := c.getRateLimiter(kind).WaitForToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to acquire rate limit token for %s: %w", kind, err)
	}

	// 设置超时时间
	apiCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// 调用Binance API
	var price map[string]interface{}
	if err := netutil.GetJSON(apiCtx, url, &price); err != nil {
		return nil, fmt.Errorf("failed to fetch price from Binance %s API: %w", kind, err)
	}

	// 转换为字符串格式
	result := make(map[string]string)
	for k, v := range price {
		if str, ok := v.(string); ok {
			result[k] = str
		}
	}

	return result, nil
}

// getRateLimiter 根据市场类型获取相应的速率限制器
func (c *BinanceAPIClient) getRateLimiter(kind string) *APIRateLimiter {
	switch kind {
	case "spot":
		return GlobalAPIRateLimiter // 或使用专门的价格限制器
	case "futures":
		return GlobalAPIRateLimiter // 或使用专门的期货限制器
	default:
		return GlobalAPIRateLimiter
	}
}

// GetSupportedKinds 获取支持的市场类型
func (c *BinanceAPIClient) GetSupportedKinds() []string {
	kinds := make([]string, 0, len(c.baseURLs))
	for kind := range c.baseURLs {
		kinds = append(kinds, kind)
	}
	return kinds
}

// AddMarketKind 添加新的市场类型
func (c *BinanceAPIClient) AddMarketKind(kind, baseURL string) {
	c.baseURLs[kind] = baseURL
}
