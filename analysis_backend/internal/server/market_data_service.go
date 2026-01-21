package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"
	"analysis/internal/netutil"
)

// ============================================================================
// 市场数据服务 - 负责市场数据获取、缓存和处理
// ============================================================================

// MarketDataService 市场数据服务结构体
type MarketDataService struct {
	server *Server
}

// NewMarketDataService 创建市场数据服务
func NewMarketDataService(server *Server) *MarketDataService {
	return &MarketDataService{
		server: server,
	}
}

// ============================================================================
// 市场数据获取接口方法
// ============================================================================

// getMarketDataForSymbol 获取单个币种的市场数据（优化版本：直接从 binance_24h_stats 查询）
func (mds *MarketDataService) getMarketDataForSymbol(symbol string) StrategyMarketData {
	// 使用优化版本
	return mds.getMarketDataForSymbolOptimized(symbol)
}

// getMarketDataForSymbolOptimized 优化版本：直接从 binance_24h_stats 查询
func (mds *MarketDataService) getMarketDataForSymbolOptimized(symbol string) StrategyMarketData {
	// 检查缓存（改进的缓存键设计）
	cacheKey := mds.makeMarketDataCacheKey(symbol, "futures")
	if cached, exists := mds.getCachedMarketData(cacheKey); exists {
		return cached
	}

	data := StrategyMarketData{
		Symbol:      symbol,
		MarketCap:   0,     // 默认市值设为0，避免返回虚假数据
		GainersRank: 999,   // 默认排名很低
		HasSpot:     false, // 默认没有现货
		HasFutures:  false, // 默认没有合约
	}

	// 直接从 binance_24h_stats 查询排名信息（使用快速版本以提高性能）
	rank, volume, err := mds.getGainerRankFrom24hStatsFast(symbol, "futures")
	if err == nil {
		data.GainersRank = rank
		// 使用成交量估算市值（更好的算法）
		if volume > 1000 { // 成交量大于1000单位
			// 获取当前价格用于估算
			if currentPrice, priceErr := mds.server.getCurrentPrice(context.Background(), symbol, "spot"); priceErr == nil && currentPrice > 0 {
				// 市值估算：24h成交量 * 当前价格 * 流通因子
				// 使用更保守的因子，避免高估
				circulationFactor := 0.05 // 假设24h成交量占总流通量的5%
				estimatedCap := volume * currentPrice * circulationFactor
				if estimatedCap > 1000000 { // 至少100万美元
					data.MarketCap = estimatedCap
				}
			}
		}
	}

	// 并行查询交易对信息（带错误处理和超时保护）
	data.HasSpot, data.HasFutures = mds.getTradingPairsConcurrent(symbol)

	// 根据数据重要性设置不同的缓存时间
	cacheTTL := mds.getMarketDataCacheTTL(data)
	mds.setCachedMarketData(cacheKey, data, cacheTTL)

	// 只在重要币种或异常情况下记录详细信息
	mds.logMarketDataInfo(symbol, data)

	return data
}

// ============================================================================
// 缓存相关辅助方法
// ============================================================================

// 生成市场数据的缓存键
func (mds *MarketDataService) makeMarketDataCacheKey(symbol, marketType string) string {
	// 格式: market_data:{market_type}:{symbol}
	// 例如: market_data:futures:BTCUSDT
	return fmt.Sprintf("market_data:%s:%s", marketType, symbol)
}

// 根据市场数据特征确定缓存时间
func (mds *MarketDataService) getMarketDataCacheTTL(data StrategyMarketData) time.Duration {
	// 排名越高的币种，数据变化越快，缓存时间越短
	if data.GainersRank <= 10 {
		return 30 * time.Second // 前10名，30秒缓存
	} else if data.GainersRank <= 50 {
		return 1 * time.Minute // 前50名，1分钟缓存
	} else if data.GainersRank <= 200 {
		return 2 * time.Minute // 前200名，2分钟缓存
	} else {
		return 5 * time.Minute // 其他币种，5分钟缓存
	}
}

// 缓存相关的辅助方法
func (mds *MarketDataService) getCachedMarketData(key string) (StrategyMarketData, bool) {
	if mds.server.cache == nil {
		return StrategyMarketData{}, false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cacheData, err := mds.server.cache.Get(ctx, key)
	if err != nil {
		return StrategyMarketData{}, false
	}

	var data StrategyMarketData
	if err := json.Unmarshal(cacheData, &data); err != nil {
		log.Printf("[Cache] 反序列化失败 %s: %v", key, err)
		return StrategyMarketData{}, false
	}

	return data, true
}

func (mds *MarketDataService) setCachedMarketData(key string, data StrategyMarketData, ttl time.Duration) error {
	if mds.server.cache == nil {
		return fmt.Errorf("cache service not available")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	cacheData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("序列化失败: %w", err)
	}

	if err := mds.server.cache.Set(ctx, key, cacheData, ttl); err != nil {
		log.Printf("[Cache] 设置缓存失败(非致命): %s, %v", key, err)
		return err
	}

	return nil
}

// ============================================================================
// 排名查询方法
// ============================================================================

// getGainerRankFrom24hStatsFast 快速版本：使用近似排名计算（性能优先）
func (mds *MarketDataService) getGainerRankFrom24hStatsFast(symbol, marketType string) (int, float64, error) {
	// 1. 参数校验
	if symbol == "" || marketType == "" {
		return 999, 0, fmt.Errorf("无效参数: symbol=%s, marketType=%s", symbol, marketType)
	}

	// 2. 检查是否有任何市场数据
	var dataCount int64
	checkErr := mds.server.db.DB().Table("binance_24h_stats").
		Where("market_type = ? AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)", marketType).
		Count(&dataCount).Error

	if checkErr != nil {
		log.Printf("[MarketData] 检查市场数据失败 %s: %v", marketType, checkErr)
		return 999, 0, fmt.Errorf("检查市场数据失败: %w", checkErr)
	}

	if dataCount == 0 {
		log.Printf("[MarketData] 市场 %s 最近1小时没有数据", marketType)
		return 999, 0, nil
	}

	// 3. 获取目标币种的数据
	var targetStats struct {
		PriceChangePercent float64
		Volume             float64
	}

	err := mds.server.db.DB().Table("binance_24h_stats").
		Select("price_change_percent, volume").
		Where("symbol = ? AND market_type = ? AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)", symbol, marketType).
		Order("created_at DESC").
		Limit(1).
		Scan(&targetStats).Error

	if err != nil {
		// 目标币种没有数据，使用默认排名
		log.Printf("[MarketData] 币种 %s 在市场 %s 中最近1小时没有数据", symbol, marketType)
		return 999, 0, nil
	}

	// 4. 计算排名：有多少币种的涨幅高于目标币种
	var higherCount int64
	err = mds.server.db.DB().Raw(`
		SELECT COUNT(DISTINCT symbol) FROM (
			SELECT symbol,
				   FIRST_VALUE(price_change_percent) OVER (PARTITION BY symbol ORDER BY created_at DESC) as latest_change,
				   FIRST_VALUE(volume) OVER (PARTITION BY symbol ORDER BY created_at DESC) as latest_volume
			FROM binance_24h_stats
			WHERE market_type = ? AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
		) as latest_data
		WHERE latest_change > ? OR (latest_change = ? AND latest_volume > ?)
	`, marketType, targetStats.PriceChangePercent, targetStats.PriceChangePercent, targetStats.Volume).
		Scan(&higherCount).Error

	if err != nil {
		log.Printf("[MarketData] 计算排名失败 %s: %v", symbol, err)
		return 999, targetStats.Volume, fmt.Errorf("计算排名失败: %w", err)
	}

	// 5. 计算最终排名
	rank := int(higherCount) + 1

	// 6. 限制排名范围
	if rank > 1000 {
		rank = 999 // 使用999表示排名很低
	} else if rank < 1 {
		rank = 1 // 最小排名为1
	}

	return rank, targetStats.Volume, nil
}

// ============================================================================
// 其他市场数据相关方法
// ============================================================================

// getAllUSDTTradingPairs 动态获取所有可用的USDT交易对
func (mds *MarketDataService) getAllUSDTTradingPairs(ctx context.Context) ([]string, error) {
	log.Printf("[TradingPairs] 开始获取所有USDT交易对...")

	// 首先检查缓存
	if mds.server.tradingPairsCache != nil {
		if symbols, ok := mds.server.tradingPairsCache.Get(); ok {
			log.Printf("[TradingPairs] 从缓存获取到%d个USDT交易对", len(symbols))
			return symbols, nil
		}
	}

	// 优先从数据库获取（避免频繁API调用）
	if symbols, err := pdb.GetUSDTTradingPairs(mds.server.db.DB()); err == nil && len(symbols) > 0 {
		log.Printf("[TradingPairs] 从数据库获取到%d个USDT交易对", len(symbols))
		// 更新缓存
		if mds.server.tradingPairsCache != nil {
			mds.server.tradingPairsCache.Set(symbols)
		}
		return symbols, nil
	}

	log.Printf("[TradingPairs] 数据库中无交易对数据，回退到API获取...")

	// 从Binance期货API获取交易对信息
	url := "https://fapi.binance.com/fapi/v1/exchangeInfo"

	var response struct {
		Symbols []struct {
			Symbol     string `json:"symbol"`
			Status     string `json:"status"`
			QuoteAsset string `json:"quoteAsset"`
		} `json:"symbols"`
	}

	// 设置超时时间
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	if err := netutil.GetJSON(ctx, url, &response); err != nil {
		return nil, fmt.Errorf("获取交易对信息失败: %w", err)
	}

	var usdtSymbols []string
	for _, symbol := range response.Symbols {
		// 只选择状态为TRADING且以USDT为计价货币的交易对
		if symbol.Status == "TRADING" && symbol.QuoteAsset == "USDT" {
			usdtSymbols = append(usdtSymbols, symbol.Symbol)
		}
	}

	log.Printf("[TradingPairs] 从Binance API获取到%d个USDT交易对", len(usdtSymbols))

	// 更新缓存
	if mds.server.tradingPairsCache != nil {
		mds.server.tradingPairsCache.Set(usdtSymbols)
	}

	return usdtSymbols, nil
}

// getCurrentFundingRate 获取当前资金费率
func (mds *MarketDataService) getCurrentFundingRate(symbol string) float64 {
	// 从数据库查询最新的资金费率
	var rate pdb.BinanceFundingRate
	err := mds.server.db.DB().Where("symbol = ?", symbol).Order("funding_time DESC").First(&rate).Error
	if err != nil {
		// 如果查询失败，返回0（表示无法评估资金费率影响）
		return 0
	}
	return rate.FundingRate
}

// 智能日志记录：只在重要情况下记录详细信息
func (mds *MarketDataService) logMarketDataInfo(symbol string, data StrategyMarketData) {
	// 始终记录前50名币种或异常情况
	shouldLog := data.GainersRank <= 50 ||
		data.GainersRank == 999 ||
		data.MarketCap > 1000000000 || // 市值超过10亿
		(!data.HasSpot && !data.HasFutures) // 没有任何交易对

	if shouldLog {
		log.Printf("[MarketData] %s: rank=%d, cap=%.0f万, spot=%v, futures=%v",
			symbol, data.GainersRank, data.MarketCap/10000, data.HasSpot, data.HasFutures)
	}

	// 记录异常情况
	if data.GainersRank == 999 {
		log.Printf("[MarketData:Warning] %s 不在涨幅榜前1000名", symbol)
	}
}

// ============================================================================
// 并发查询和错误处理方法
// ============================================================================

// 并发查询交易对信息（带错误处理和超时保护）
func (mds *MarketDataService) getTradingPairsConcurrent(symbol string) (hasSpot, hasFutures bool) {
	type queryResult struct {
		hasSpot    bool
		hasFutures bool
		err        error
	}

	resultChan := make(chan queryResult, 1)

	// 启动goroutine执行查询
	go func() {
		defer func() {
			if r := recover(); r != nil {
				resultChan <- queryResult{
					hasSpot:    false,
					hasFutures: false,
					err:        fmt.Errorf("concurrent query panic: %v", r),
				}
			}
		}()

		spot, spotErr := mds.checkSpotTradingSafe(symbol)
		futures, futuresErr := mds.checkFuturesTradingSafe(symbol)

		var err error
		if spotErr != nil || futuresErr != nil {
			err = fmt.Errorf("spot_check_err: %v, futures_check_err: %v", spotErr, futuresErr)
		}

		resultChan <- queryResult{
			hasSpot:    spot,
			hasFutures: futures,
			err:        err,
		}
	}()

	// 设置3秒超时
	timeout := 3 * time.Second
	select {
	case result := <-resultChan:
		if result.err != nil {
			log.Printf("[MarketData] 并发查询失败 %s: %v", symbol, result.err)
			// 失败时返回保守的默认值（false）
			return false, false
		}
		return result.hasSpot, result.hasFutures

	case <-time.After(timeout):
		log.Printf("[MarketData] 并发查询超时 %s", symbol)
		return false, false
	}
}

// 安全的交易对检查方法（带错误处理）
func (mds *MarketDataService) checkSpotTradingSafe(symbol string) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MarketData] checkSpotTradingSafe panic %s: %v", symbol, r)
		}
	}()

	var count int64
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 使用带超时的查询，只检查活跃的交易对
	db := mds.server.db.DB().WithContext(ctx)
	err := db.Table("binance_exchange_info").
		Where("symbol = ? AND status = ? AND quote_asset = ? AND is_active = ?",
			symbol, "TRADING", "USDT", true).
		Count(&count).Error

	log.Printf("[DEBUG] checkSpotTradingSafe %s: count=%d, err=%v", symbol, count, err)

	if err != nil {
		return false, fmt.Errorf("spot trading check failed: %w", err)
	}

	result := count > 0
	log.Printf("[DEBUG] checkSpotTradingSafe %s: result=%v", symbol, result)
	return result, nil
}

func (mds *MarketDataService) checkFuturesTradingSafe(symbol string) (bool, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[MarketData] checkFuturesTradingSafe panic %s: %v", symbol, r)
		}
	}()

	var count int64
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// 使用带超时的查询
	db := mds.server.db.DB().WithContext(ctx)
	err := db.Table("binance_market_tops").
		Joins("JOIN binance_market_snapshots ON binance_market_tops.snapshot_id = binance_market_snapshots.id").
		Where("binance_market_snapshots.kind = ? AND binance_market_tops.symbol = ?", "futures", symbol).
		Count(&count).Error

	log.Printf("[DEBUG] checkFuturesTradingSafe %s: count=%d, err=%v", symbol, count, err)

	if err != nil {
		return false, err
	}

	result := count > 0
	log.Printf("[DEBUG] checkFuturesTradingSafe %s: result=%v", symbol, result)
	return result, nil
}
