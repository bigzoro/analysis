package server

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"analysis/internal/config"
)

// DataSource 数据源接口
type DataSource interface {
	Name() string
	FetchMarketData(ctx context.Context, symbols []string) ([]MarketData, error)
	FetchNewsData(ctx context.Context, symbols []string) ([]NewsData, error)
	FetchSocialData(ctx context.Context, symbols []string) ([]SocialData, error)
	IsAvailable() bool
}

// MarketData 市场数据结构
type MarketData struct {
	Symbol      string    `json:"symbol"`
	Source      string    `json:"source"`
	Price       float64   `json:"price"`
	Volume24h   float64   `json:"volume_24h"`
	MarketCap   float64   `json:"market_cap"`
	Change24h   float64   `json:"change_24h"`
	Change7d    float64   `json:"change_7d"`
	Change30d   float64   `json:"change_30d"`
	LastUpdated time.Time `json:"last_updated"`
}

// NewsData 新闻数据结构
type NewsData struct {
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	URL         string    `json:"url"`
	Source      string    `json:"source"`
	Symbols     []string  `json:"symbols"`
	Sentiment   float64   `json:"sentiment"` // -1 到 1
	PublishedAt time.Time `json:"published_at"`
}

// SocialData 社交媒体数据结构
type SocialData struct {
	Platform   string    `json:"platform"`
	Symbol     string    `json:"symbol"`
	Mentions   int       `json:"mentions"`
	Sentiment  float64   `json:"sentiment"`
	Engagement int       `json:"engagement"`
	PostedAt   time.Time `json:"posted_at"`
}

// DataManager 数据源管理器
type DataManager struct {
	sources []DataSource
	cache   *DataCache
	config  *config.Config
}

// NewDataManager 创建数据管理器
func NewDataManager(cfg *config.Config) *DataManager {
	dm := &DataManager{
		sources: make([]DataSource, 0),
		cache:   NewDataCache(),
		config:  cfg,
	}

	// 初始化免费数据源
	dm.initFreeDataSources()

	return dm
}

// initFreeDataSources 初始化免费数据源
func (dm *DataManager) initFreeDataSources() {
	// CoinGecko - 免费API
	if dm.config.DataSources.CoinGecko.Enabled {
		if cg := NewCoinGeckoSource(); cg.IsAvailable() {
			dm.sources = append(dm.sources, cg)
			log.Printf("[INFO] CoinGecko data source initialized")
		}
	}

	// NewsAPI - 免费版 (100次/天)
	if dm.config.DataSources.NewsAPI.APIKey != "" {
		if news := NewNewsAPISourceWithKey(dm.config.DataSources.NewsAPI.APIKey); news.IsAvailable() {
			dm.sources = append(dm.sources, news)
			log.Printf("[INFO] NewsAPI data source initialized")
		}
	}

	// Reddit API - 免费
	if reddit := NewRedditSource(); reddit.IsAvailable() {
		dm.sources = append(dm.sources, reddit)
		log.Printf("[INFO] Reddit data source initialized")
	}

	// LunarCrush - 免费版 (有限额度)
	if lc := NewLunarCrushSourceWithKey(dm.config.DataSources.LunarCrush.APIKey); lc.IsAvailable() {
		dm.sources = append(dm.sources, lc)
		log.Printf("[INFO] LunarCrush data source initialized")
	}

	// 记录初始化完成
	log.Printf("[INFO] Initialized %d data sources", len(dm.sources))
}

// FetchMultiSourceData 获取多源数据
func (dm *DataManager) FetchMultiSourceData(ctx context.Context, symbols []string) (*UnifiedMarketData, error) {
	result := &UnifiedMarketData{
		SymbolData: make(map[string]*SymbolUnifiedData),
		Timestamp:  time.Now(),
	}

	// 并行获取各数据源数据
	type sourceResult struct {
		source string
		market []MarketData
		news   []NewsData
		social []SocialData
		err    error
	}

	results := make(chan sourceResult, len(dm.sources))

	for _, source := range dm.sources {
		go func(src DataSource) {
			res := sourceResult{source: src.Name()}

			if market, err := src.FetchMarketData(ctx, symbols); err == nil {
				res.market = market
			} else {
				log.Printf("[WARN] Failed to fetch market data from %s: %v", src.Name(), err)
			}

			if news, err := src.FetchNewsData(ctx, symbols); err == nil {
				res.news = news
			}

			if social, err := src.FetchSocialData(ctx, symbols); err == nil {
				res.social = social
			}

			results <- res
		}(source)
	}

	// 收集结果
	for i := 0; i < len(dm.sources); i++ {
		res := <-results

		// 融合市场数据
		for _, market := range res.market {
			dm.mergeMarketData(result, market, res.source)
		}

		// 融合新闻数据
		for _, news := range res.news {
			dm.mergeNewsData(result, news)
		}

		// 融合社交数据
		for _, social := range res.social {
			dm.mergeSocialData(result, social)
		}
	}

	// 数据质量控制
	dm.applyQualityControl(result)

	// 缓存结果
	dm.cache.StoreUnifiedData(result)

	return result, nil
}

// mergeMarketData 融合市场数据
func (dm *DataManager) mergeMarketData(result *UnifiedMarketData, data MarketData, source string) {
	if result.SymbolData[data.Symbol] == nil {
		result.SymbolData[data.Symbol] = &SymbolUnifiedData{
			Symbol:     data.Symbol,
			Sources:    make(map[string]MarketData),
			News:       make([]NewsData, 0),
			SocialData: make([]SocialData, 0),
		}
	}

	symbolData := result.SymbolData[data.Symbol]
	symbolData.Sources[source] = data

	// 计算综合价格（加权平均）
	dm.calculateUnifiedPrice(symbolData)
}

// mergeNewsData 融合新闻数据
func (dm *DataManager) mergeNewsData(result *UnifiedMarketData, data NewsData) {
	for _, symbol := range data.Symbols {
		if symbolData, exists := result.SymbolData[symbol]; exists {
			symbolData.News = append(symbolData.News, data)
		}
	}
}

// mergeSocialData 融合社交数据
func (dm *DataManager) mergeSocialData(result *UnifiedMarketData, data SocialData) {
	if symbolData, exists := result.SymbolData[data.Symbol]; exists {
		symbolData.SocialData = append(symbolData.SocialData, data)
	}
}

// calculateUnifiedPrice 计算综合价格
func (dm *DataManager) calculateUnifiedPrice(symbolData *SymbolUnifiedData) {
	totalWeight := 0.0
	totalPrice := 0.0

	// 不同数据源的权重
	sourceWeights := map[string]float64{
		"coingecko":     0.4,  // 高质量数据源
		"binance":       0.35, // 主要交易所
		"coinmarketcap": 0.25, // 市场数据
	}

	for source, data := range symbolData.Sources {
		weight := sourceWeights[source]
		if weight == 0 {
			weight = 0.1 // 默认权重
		}

		totalPrice += data.Price * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		symbolData.UnifiedPrice = totalPrice / totalWeight
		symbolData.Confidence = totalWeight // 置信度基于数据源权重总和
	}
}

// applyQualityControl 数据质量控制
func (dm *DataManager) applyQualityControl(data *UnifiedMarketData) {
	for symbol, symbolData := range data.SymbolData {
		// 价格异常检测
		if len(symbolData.Sources) > 1 {
			prices := make([]float64, 0, len(symbolData.Sources))
			for _, sourceData := range symbolData.Sources {
				prices = append(prices, sourceData.Price)
			}

			// 简单的异常检测：如果价格差异超过20%，标记为异常
			if dm.detectPriceAnomaly(prices) {
				log.Printf("[WARN] Price anomaly detected for %s", symbol)
				symbolData.HasAnomaly = true
				symbolData.AnomalyReason = "价格数据差异过大"
			}
		}

		// 数据时效性检查
		now := time.Now()
		for source, sourceData := range symbolData.Sources {
			if now.Sub(sourceData.LastUpdated) > time.Hour {
				log.Printf("[WARN] Stale data from %s for %s", source, symbol)
			}
		}
	}
}

// detectPriceAnomaly 检测价格异常
func (dm *DataManager) detectPriceAnomaly(prices []float64) bool {
	if len(prices) < 2 {
		return false
	}

	minPrice, maxPrice := prices[0], prices[0]
	for _, price := range prices {
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
	}

	// 如果最大价格是最小价格的1.2倍以上，认为异常
	return maxPrice/minPrice > 1.2
}

// GetCachedData 获取缓存数据
func (dm *DataManager) GetCachedData(symbol string) (*SymbolUnifiedData, bool) {
	return dm.cache.GetSymbolData(symbol)
}

// UnifiedMarketData 统一市场数据结构
type UnifiedMarketData struct {
	SymbolData map[string]*SymbolUnifiedData `json:"symbol_data"`
	Timestamp  time.Time                     `json:"timestamp"`
}

// SymbolUnifiedData 单个币种的统一数据
type SymbolUnifiedData struct {
	Symbol        string                `json:"symbol"`
	UnifiedPrice  float64               `json:"unified_price"`
	Confidence    float64               `json:"confidence"` // 0-1 置信度
	Sources       map[string]MarketData `json:"sources"`
	News          []NewsData            `json:"news"`
	SocialData    []SocialData          `json:"social_data"`
	HasAnomaly    bool                  `json:"has_anomaly"`
	AnomalyReason string                `json:"anomaly_reason,omitempty"`
}

// HTTPClient HTTP客户端工具
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient 创建HTTP客户端
func NewHTTPClient() *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// Get 执行GET请求
func (hc *HTTPClient) Get(url string) ([]byte, error) {
	resp, err := hc.client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}

// Post 执行POST请求
func (hc *HTTPClient) Post(url string, data []byte) ([]byte, error) {
	resp, err := hc.client.Post(url, "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	return io.ReadAll(resp.Body)
}
