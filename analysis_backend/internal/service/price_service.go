package service

import (
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/netutil"
	"context"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"
)

// BinancePriceFetcher 从Binance获取价格的函数类型
type BinancePriceFetcher func(ctx context.Context, symbol string, kind string) (float64, error)

// PriceService 统一价格服务，管理所有价格获取逻辑
type PriceService struct {
	cfg        *config.Config
	db         *gorm.DB
	httpClient *http.Client

	// Binance价格获取函数（可选）
	binanceFetcher BinancePriceFetcher

	// 缓存
	mu                sync.RWMutex
	currentPriceCache map[string]priceCacheItem  // key: symbol_kind
	coinIDCache       map[string]coinIDCacheItem // key: symbol
}

type priceCacheItem struct {
	price     float64
	expiresAt time.Time
}

type coinIDCacheItem struct {
	coinID    string
	expiresAt time.Time
}

// NewPriceService 创建价格服务
func NewPriceService(cfg *config.Config, gdb *gorm.DB) *PriceService {
	return &PriceService{
		cfg:               cfg,
		db:                gdb,
		httpClient:        &http.Client{Timeout: 10 * time.Second},
		currentPriceCache: make(map[string]priceCacheItem),
		coinIDCache:       make(map[string]coinIDCacheItem),
	}
}

// SetBinanceFetcher 设置Binance价格获取函数
func (ps *PriceService) SetBinanceFetcher(fetcher BinancePriceFetcher) {
	ps.binanceFetcher = fetcher
}

// GetCurrentPrice 获取当前价格
// 优先级：1. Binance K线数据 2. 数据库市场快照
func (ps *PriceService) GetCurrentPrice(ctx context.Context, symbol string, kind string) (float64, error) {
	cacheKey := fmt.Sprintf("%s_%s", strings.ToUpper(symbol), kind)

	// 检查缓存
	ps.mu.RLock()
	if item, ok := ps.currentPriceCache[cacheKey]; ok && time.Now().Before(item.expiresAt) {
		ps.mu.RUnlock()
		return item.price, nil
	}
	ps.mu.RUnlock()

	// 尝试从Binance获取
	price, err := ps.getCurrentPriceFromBinance(ctx, symbol, kind)
	if err == nil {
		// 更新缓存（缓存1分钟）
		ps.mu.Lock()
		ps.currentPriceCache[cacheKey] = priceCacheItem{
			price:     price,
			expiresAt: time.Now().Add(1 * time.Minute),
		}
		ps.mu.Unlock()
		return price, nil
	}

	// 如果失败，尝试从数据库获取
	price, err = ps.getCurrentPriceFromDB(ctx, symbol, kind)
	if err == nil {
		// 更新缓存（缓存30秒，因为数据库数据可能不是最新的）
		ps.mu.Lock()
		ps.currentPriceCache[cacheKey] = priceCacheItem{
			price:     price,
			expiresAt: time.Now().Add(30 * time.Second),
		}
		ps.mu.Unlock()
		return price, nil
	}

	return 0, fmt.Errorf("无法获取 %s 的当前价格: %w", symbol, err)
}

// getCurrentPriceFromBinance 从Binance获取当前价格
func (ps *PriceService) getCurrentPriceFromBinance(ctx context.Context, symbol string, kind string) (float64, error) {
	if ps.binanceFetcher == nil {
		return 0, fmt.Errorf("Binance价格获取函数未设置")
	}
	return ps.binanceFetcher(ctx, symbol, kind)
}

// getCurrentPriceFromDB 从数据库获取当前价格
func (ps *PriceService) getCurrentPriceFromDB(ctx context.Context, symbol string, kind string) (float64, error) {
	if ps.db == nil {
		return 0, fmt.Errorf("数据库未初始化")
	}

	now := time.Now().UTC()
	startTime := now.Add(-2 * time.Hour)
	snaps, tops, err := pdb.ListBinanceMarket(ps.db, kind, startTime, now)
	if err != nil {
		return 0, fmt.Errorf("查询市场数据失败: %w", err)
	}

	if len(snaps) == 0 {
		return 0, fmt.Errorf("未找到市场快照")
	}

	// 获取最新的快照
	latestSnap := snaps[len(snaps)-1]
	if items, ok := tops[latestSnap.ID]; ok {
		for _, item := range items {
			if item.Symbol == symbol {
				price, err := strconv.ParseFloat(item.LastPrice, 64)
				if err != nil {
					return 0, fmt.Errorf("解析价格失败: %w", err)
				}
				return price, nil
			}
		}
	}

	return 0, fmt.Errorf("未找到 %s 的价格数据", symbol)
}

// BatchGetCurrentPrices 批量获取当前价格
func (ps *PriceService) BatchGetCurrentPrices(ctx context.Context, symbols []string, kind string) (map[string]float64, error) {
	result := make(map[string]float64)

	// 先检查内存缓存
	ps.mu.RLock()
	for _, symbol := range symbols {
		cacheKey := fmt.Sprintf("%s_%s", strings.ToUpper(symbol), kind)
		if item, ok := ps.currentPriceCache[cacheKey]; ok && time.Now().Before(item.expiresAt) {
			result[symbol] = item.price
		}
	}
	ps.mu.RUnlock()

	// 获取未缓存的symbols
	uncached := make([]string, 0)
	for _, symbol := range symbols {
		if _, ok := result[symbol]; !ok {
			uncached = append(uncached, symbol)
		}
	}

	if len(uncached) == 0 {
		return result, nil
	}

	// 批量从数据库查询缓存，避免逐个查询产生大量日志
	if ps.db != nil {
		var caches []pdb.PriceCache
		upperSymbols := make([]string, len(uncached))
		for i, symbol := range uncached {
			upperSymbols[i] = strings.ToUpper(symbol)
		}

		// 批量查询数据库缓存
		err := ps.db.Where("symbol IN (?) AND kind = ? AND last_updated > ?",
			upperSymbols, kind, time.Now().Add(-30*time.Second)).Find(&caches).Error

		if err == nil {
			// 将数据库缓存结果加入内存缓存和结果
			ps.mu.Lock()
			for _, cache := range caches {
				if price, err := strconv.ParseFloat(cache.Price, 64); err == nil {
					cacheKey := fmt.Sprintf("%s_%s", cache.Symbol, cache.Kind)
					ps.currentPriceCache[cacheKey] = priceCacheItem{
						price:     price,
						expiresAt: cache.LastUpdated.Add(30 * time.Second),
					}
					result[cache.Symbol] = price
				}
			}
			ps.mu.Unlock()

			// 更新未缓存列表
			stillUncached := make([]string, 0)
			for _, symbol := range uncached {
				if _, ok := result[symbol]; !ok {
					stillUncached = append(stillUncached, symbol)
				}
			}
			uncached = stillUncached
		}
	}

	// 对于仍然没有缓存的，从API获取
	for _, symbol := range uncached {
		price, err := ps.GetCurrentPrice(ctx, symbol, kind)
		if err == nil {
			result[symbol] = price
		} else {
			// 获取失败时明确设置为0，避免返回零值
			result[symbol] = 0
			log.Printf("[WARN] 获取 %s 价格失败: %v", symbol, err)
		}
	}

	return result, nil
}

// GetHistoricalPrice 获取历史时间点的价格（从CoinGecko）
func (ps *PriceService) GetHistoricalPrice(ctx context.Context, symbol string, targetTime time.Time) (float64, error) {
	if ps.cfg == nil || !ps.cfg.Pricing.Enable {
		return 0, fmt.Errorf("价格服务未启用")
	}

	// 获取币种ID
	coinID, err := ps.GetCoinGeckoID(ctx, symbol)
	if err != nil {
		return 0, fmt.Errorf("获取币种ID失败: %w", err)
	}

	// 计算需要获取的天数
	now := time.Now().UTC()
	days := int(now.Sub(targetTime).Hours()/24) + 1
	if days < 1 {
		days = 1
	}
	if days > 365 {
		days = 365 // CoinGecko限制
	}

	// 构建CoinGecko API URL
	baseURL := ps.getCoinGeckoBaseURL()
	var url string
	if days <= 1 {
		url = fmt.Sprintf("%s/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d&interval=hourly", baseURL, coinID, days)
	} else {
		url = fmt.Sprintf("%s/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d", baseURL, coinID, days)
	}

	// 获取价格数据
	var resp struct {
		Prices [][]float64 `json:"prices"`
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = netutil.GetJSON(ctxWithTimeout, url, &resp)
	if err != nil {
		return 0, fmt.Errorf("获取历史价格数据失败: %w", err)
	}

	// 查找目标时间点的价格
	price := ps.findPriceAtTime(targetTime, resp.Prices)
	if price == nil {
		return 0, fmt.Errorf("未找到时间点 %s 的价格", targetTime.Format(time.RFC3339))
	}

	return *price, nil
}

// GetHistoricalPrices 批量获取历史价格
func (ps *PriceService) GetHistoricalPrices(ctx context.Context, symbol string, targetTimes []time.Time) (map[time.Time]float64, error) {
	if len(targetTimes) == 0 {
		return make(map[time.Time]float64), nil
	}

	if ps.cfg == nil || !ps.cfg.Pricing.Enable {
		return nil, fmt.Errorf("价格服务未启用")
	}

	// 获取币种ID
	coinID, err := ps.GetCoinGeckoID(ctx, symbol)
	if err != nil {
		return nil, fmt.Errorf("获取币种ID失败: %w", err)
	}

	// 计算需要获取的天数（取最远的时间点）
	now := time.Now().UTC()
	maxDays := 1
	for _, t := range targetTimes {
		days := int(now.Sub(t).Hours()/24) + 1
		if days > maxDays {
			maxDays = days
		}
	}
	if maxDays > 365 {
		maxDays = 365
	}

	// 构建CoinGecko API URL
	baseURL := ps.getCoinGeckoBaseURL()
	var url string
	if maxDays <= 1 {
		url = fmt.Sprintf("%s/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d&interval=hourly", baseURL, coinID, maxDays)
	} else {
		url = fmt.Sprintf("%s/api/v3/coins/%s/market_chart?vs_currency=usd&days=%d", baseURL, coinID, maxDays)
	}

	// 获取价格数据
	var resp struct {
		Prices [][]float64 `json:"prices"`
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	err = netutil.GetJSON(ctxWithTimeout, url, &resp)
	if err != nil {
		return nil, fmt.Errorf("获取历史价格数据失败: %w", err)
	}

	// 查找所有目标时间点的价格
	result := make(map[time.Time]float64)
	for _, targetTime := range targetTimes {
		price := ps.findPriceAtTime(targetTime, resp.Prices)
		if price != nil {
			result[targetTime] = *price
		}
	}

	return result, nil
}

// GetCoinGeckoID 获取CoinGecko币种ID（带缓存）
func (ps *PriceService) GetCoinGeckoID(ctx context.Context, symbol string) (string, error) {
	symbolUpper := strings.ToUpper(strings.TrimSpace(symbol))
	if symbolUpper == "" {
		return "", fmt.Errorf("币种符号不能为空")
	}

	// 检查缓存
	ps.mu.RLock()
	if item, ok := ps.coinIDCache[symbolUpper]; ok && time.Now().Before(item.expiresAt) {
		ps.mu.RUnlock()
		return item.coinID, nil
	}
	ps.mu.RUnlock()

	// 先检查配置映射
	if ps.cfg != nil && ps.cfg.Pricing.Enable {
		if coinID := ps.cfg.Pricing.Map[symbolUpper]; coinID != "" {
			// 更新缓存（缓存24小时）
			ps.mu.Lock()
			ps.coinIDCache[symbolUpper] = coinIDCacheItem{
				coinID:    coinID,
				expiresAt: time.Now().Add(24 * time.Hour),
			}
			ps.mu.Unlock()
			return coinID, nil
		}
	}

	// 从CoinGecko搜索API查找
	coinID, err := ps.findCoinGeckoIDFromAPI(ctx, symbolUpper)
	if err != nil {
		return "", err
	}

	// 更新缓存（缓存24小时）
	ps.mu.Lock()
	ps.coinIDCache[symbolUpper] = coinIDCacheItem{
		coinID:    coinID,
		expiresAt: time.Now().Add(24 * time.Hour),
	}
	ps.mu.Unlock()

	return coinID, nil
}

// findCoinGeckoIDFromAPI 从CoinGecko API搜索币种ID
func (ps *PriceService) findCoinGeckoIDFromAPI(ctx context.Context, symbol string) (string, error) {
	if ps.cfg == nil || !ps.cfg.Pricing.Enable {
		return "", fmt.Errorf("价格服务未启用")
	}

	baseURL := ps.getCoinGeckoBaseURL()
	url := fmt.Sprintf("%s/api/v3/search?query=%s", baseURL, symbol)

	var resp struct {
		Coins []struct {
			ID     string `json:"id"`
			Symbol string `json:"symbol"`
			Name   string `json:"name"`
		} `json:"coins"`
	}

	ctxWithTimeout, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := netutil.GetJSON(ctxWithTimeout, url, &resp)
	if err != nil {
		return "", fmt.Errorf("搜索币种失败: %w", err)
	}

	if len(resp.Coins) == 0 {
		return "", fmt.Errorf("未找到币种 %s", symbol)
	}

	// 优先精确匹配symbol（不区分大小写）
	symbolUpper := strings.ToUpper(symbol)
	for _, coin := range resp.Coins {
		if strings.ToUpper(coin.Symbol) == symbolUpper {
			return coin.ID, nil
		}
	}

	// 如果没有精确匹配，返回第一个结果
	return resp.Coins[0].ID, nil
}

// getCoinGeckoBaseURL 获取CoinGecko基础URL
func (ps *PriceService) getCoinGeckoBaseURL() string {
	if ps.cfg == nil || !ps.cfg.Pricing.Enable {
		return ""
	}

	endpoint := ps.cfg.Pricing.CoinGeckoEndpoint
	baseURL := endpoint
	if strings.Contains(endpoint, "/api/v3") {
		parts := strings.Split(endpoint, "/api/v3")
		if len(parts) > 0 {
			baseURL = strings.TrimSuffix(parts[0], "/")
		}
	}
	return baseURL
}

// findPriceAtTime 在价格数据中查找指定时间点的价格
func (ps *PriceService) findPriceAtTime(targetTime time.Time, prices [][]float64) *float64 {
	if len(prices) == 0 {
		return nil
	}

	targetUnix := float64(targetTime.Unix()) * 1000 // CoinGecko使用毫秒时间戳

	// 如果目标时间在未来，返回nil
	if targetUnix > prices[len(prices)-1][0] {
		return nil
	}

	// 二分查找最接近的时间点
	bestIdx := -1
	minDiff := float64(1 << 62)
	for i, p := range prices {
		diff := p[0] - targetUnix
		if diff >= 0 && diff < minDiff {
			minDiff = diff
			bestIdx = i
		}
	}

	if bestIdx >= 0 && minDiff < 24*3600*1000 { // 允许1天内的误差
		price := prices[bestIdx][1]
		return &price
	}

	// 如果找不到精确匹配，使用最接近的时间点（允许12小时误差）
	for i, p := range prices {
		diff := p[0] - targetUnix
		if diff >= -12*3600*1000 && diff <= 12*3600*1000 {
			price := p[1]
			return &price
		}
		if diff > 0 && i > 0 {
			// 已经过了目标时间，使用前一个点
			price := prices[i-1][1]
			return &price
		}
	}

	return nil
}

// ClearCache 清空缓存
func (ps *PriceService) ClearCache() {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	ps.currentPriceCache = make(map[string]priceCacheItem)
	ps.coinIDCache = make(map[string]coinIDCacheItem)
}
