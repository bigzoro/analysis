package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/netutil"

	"gorm.io/gorm"
)

// ===== 市场统计数据同步器 =====
// 同步24小时市场统计数据，包括价格、交易量、买卖盘口等完整市场信息
// 原名VolumeSyncer，实际功能是同步完整的市场统计数据而不仅是交易量

// MarketStatsSyncerConfig 市场统计同步器配置
type MarketStatsSyncerConfig struct {
	SpotSymbols    []string // 现货交易对
	FuturesSymbols []string // 期货交易对
}

// buildMarketStatsSyncerConfig 构建市场统计同步器配置
func (s *MarketStatsSyncer) buildMarketStatsSyncerConfig() MarketStatsSyncerConfig {
	config := MarketStatsSyncerConfig{}

	// 优先从数据库获取各市场的有效交易对，避免使用包含无效符号的全局配置
	if spotSymbols, err := pdb.GetUSDTTradingPairsByMarket(s.db, "spot"); err == nil {
		// 过滤掉Redis缓存中标记为无效的符号
		config.SpotSymbols = s.filterOutInvalidSymbols(spotSymbols, "spot")
		log.Printf("[MarketStatsSyncer] ✅ Loaded %d spot symbols from database (%d after filtering invalid)",
			len(spotSymbols), len(config.SpotSymbols))
	} else {
		log.Printf("[MarketStatsSyncer] ⚠️ Failed to get spot symbols: %v", err)
		// 如果数据库查询失败，尝试从配置中获取
		if len(s.config.Symbols) > 0 {
			config.SpotSymbols = s.config.Symbols
			log.Printf("[MarketStatsSyncer] 🔄 Using configured symbols as fallback for spot: %d symbols", len(config.SpotSymbols))
		}
	}

	if futuresSymbols, err := pdb.GetUSDTTradingPairsByMarket(s.db, "futures"); err == nil {
		// 过滤掉Redis缓存中标记为无效的符号
		config.FuturesSymbols = s.filterOutInvalidSymbols(futuresSymbols, "futures")
		log.Printf("[MarketStatsSyncer] ✅ Loaded %d futures symbols from database (%d after filtering invalid)",
			len(futuresSymbols), len(config.FuturesSymbols))
	} else {
		log.Printf("[MarketStatsSyncer] ⚠️ Failed to get futures symbols: %v", err)
		// 如果数据库查询失败，尝试从配置中获取
		if len(s.config.Symbols) > 0 {
			config.FuturesSymbols = s.config.Symbols
			log.Printf("[MarketStatsSyncer] 🔄 Using configured symbols as fallback for futures: %d symbols", len(config.FuturesSymbols))
		}
	}

	return config
}

// filterOutInvalidSymbols 过滤掉Redis缓存中标记为无效的符号
func (s *MarketStatsSyncer) filterOutInvalidSymbols(symbols []string, marketType string) []string {
	if len(symbols) == 0 {
		return symbols
	}

	var validSymbols []string
	for _, symbol := range symbols {
		if !s.isSymbolInvalid(symbol, marketType) {
			validSymbols = append(validSymbols, symbol)
		} else {
			log.Printf("[MarketStatsSyncer] 🗑️ Filtered out invalid symbol: %s %s", symbol, marketType)
		}
	}

	return validSymbols
}

// filterConfiguredSymbols 过滤出配置中存在的交易对
func (s *MarketStatsSyncer) filterConfiguredSymbols(configured, available []string) []string {
	configMap := make(map[string]bool)
	for _, symbol := range configured {
		configMap[symbol] = true
	}

	var result []string
	for _, symbol := range available {
		if configMap[symbol] {
			result = append(result, symbol)
		}
	}

	return result
}

// syncMarketStats 同步指定市场的统计数据
func (s *MarketStatsSyncer) syncMarketStats(ctx context.Context, symbols []string, marketType string) (int, int) {
	if len(symbols) == 0 {
		return 0, 0
	}

	var symbolsToSync []string

	// 🔄 增量同步：只同步需要更新的交易对（如果启用）
	if s.config.EnableIncrementalSync {
		log.Printf("[MarketStatsSyncer] 🔄 Incremental sync enabled for %s market, checking for outdated symbols...", marketType)
		filteredSymbols, err := s.getSymbolsNeedingStatsSyncByMarket(symbols, marketType)
		if err != nil {
			log.Printf("[MarketStatsSyncer] ⚠️ Failed to determine symbols needing %s stats sync: %v, falling back to full sync", marketType, err)
			symbolsToSync = symbols // 回退到全量同步
		} else {
			symbolsToSync = filteredSymbols
		}
	} else {
		log.Printf("[MarketStatsSyncer] 🔄 Incremental sync disabled for %s market, performing full sync...", marketType)
		symbolsToSync = symbols // 全量同步
	}

	log.Printf("[MarketStatsSyncer] 🎯 Starting %s market stats sync for %d/%d symbols",
		marketType, len(symbolsToSync), len(symbols))

	// 如果没有需要同步的交易对，跳过同步
	if len(symbolsToSync) == 0 {
		log.Printf("[MarketStatsSyncer] ✅ All %s market symbols are up-to-date, skipping stats sync", marketType)
		return 0, 0
	}

	// API频率控制参数（从配置读取，如果没有配置则使用默认值）
	maxConcurrentRequests := s.config.WorkerPoolSize
	if maxConcurrentRequests <= 0 {
		maxConcurrentRequests = 5 // 默认最大并发请求数
	}

	// 计算请求间隔：基于API调用超时和并发数动态调整
	baseInterval := time.Duration(s.config.APICallTimeout) * time.Second
	if baseInterval <= 0 {
		baseInterval = 5 * time.Second // 默认5秒超时
	}
	requestInterval := baseInterval / time.Duration(maxConcurrentRequests)
	if requestInterval < 50*time.Millisecond {
		requestInterval = 50 * time.Millisecond // 最小间隔50ms
	}

	log.Printf("[MarketStatsSyncer] API频率控制: 最大并发=%d, 请求间隔=%v", maxConcurrentRequests, requestInterval)

	// 创建信号量控制并发
	semaphore := make(chan struct{}, maxConcurrentRequests)
	var wg sync.WaitGroup

	// 使用原子变量记录统计信息
	var updates int32 = 0
	var errors int32 = 0

	// 记录上一次请求时间，用于频率控制
	var lastRequestTime time.Time
	var timeMutex sync.Mutex

	for _, symbol := range symbolsToSync {
		wg.Add(1)

		go func(sym string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 频率控制：确保请求间隔
			timeMutex.Lock()
			elapsed := time.Since(lastRequestTime)
			if elapsed < requestInterval {
				sleepTime := requestInterval - elapsed
				time.Sleep(sleepTime)
			}
			lastRequestTime = time.Now()
			timeMutex.Unlock()

			// 同步24小时统计数据
			if err := s.sync24hStats(ctx, sym, marketType); err != nil {
				log.Printf("[MarketStatsSyncer] ❌ Failed to sync %s 24h stats for %s: %v", marketType, sym, err)
				atomic.AddInt32(&errors, 1)
			} else {
				log.Printf("[MarketStatsSyncer] ✅ Synced %s 24h stats for %s", marketType, sym)
				atomic.AddInt32(&updates, 1)
			}
		}(symbol)
	}

	// 等待所有goroutine完成
	wg.Wait()

	log.Printf("[MarketStatsSyncer] 📊 %s market stats sync completed: %d updates, %d errors",
		marketType, atomic.LoadInt32(&updates), atomic.LoadInt32(&errors))

	return int(atomic.LoadInt32(&updates)), int(atomic.LoadInt32(&errors))
}

// getSymbolsNeedingStatsSyncByMarket 按市场获取需要统计同步的交易对
func (s *MarketStatsSyncer) getSymbolsNeedingStatsSyncByMarket(allSymbols []string, marketType string) ([]string, error) {
	if len(allSymbols) == 0 {
		return allSymbols, nil
	}

	// 设置统计数据过期时间（扩大检查窗口，确保数据完整性）
	maxDataAge := 2 * time.Hour // 从1小时增加到2小时
	cutoffTime := time.Now().Add(-maxDataAge)

	// 使用通道收集结果
	type checkResult struct {
		symbol    string
		needsSync bool
	}

	resultChan := make(chan checkResult, len(allSymbols))
	var wg sync.WaitGroup

	// 限制并发数量，避免数据库压力过大
	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)

	// 并发检查每个交易对
	for _, symbol := range allSymbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 检查该交易对是否需要同步
			needsSync := s.checkSymbolNeedsStatsSyncByMarket(sym, marketType, cutoffTime)
			resultChan <- checkResult{symbol: sym, needsSync: needsSync}
		}(symbol)
	}

	// 等待所有检查完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var symbolsToSync []string
	for result := range resultChan {
		if result.needsSync {
			symbolsToSync = append(symbolsToSync, result.symbol)
		}
	}

	// 如果所有数据都是最新的，至少同步几个核心交易对
	if len(symbolsToSync) == 0 && len(allSymbols) > 0 {
		coreSymbols := []string{"BTCUSDT", "ETHUSDT"}
		for _, coreSymbol := range coreSymbols {
			if s.containsString(allSymbols, coreSymbol) {
				symbolsToSync = append(symbolsToSync, coreSymbol)
			}
		}
	}

	log.Printf("[MarketStatsSyncer] 🔄 %s market incremental sync: %d/%d symbols need stats updating",
		marketType, len(symbolsToSync), len(allSymbols))

	return symbolsToSync, nil
}

// checkSymbolNeedsStatsSyncByMarket 检查单个交易对在指定市场是否需要统计同步
func (s *MarketStatsSyncer) checkSymbolNeedsStatsSyncByMarket(symbol, marketType string, cutoffTime time.Time) bool {
	var result struct {
		LastUpdateTime time.Time `json:"last_update_time"`
		RecordCount    int       `json:"record_count"`
		DataQuality    float64   `json:"data_quality"` // 数据质量评分
	}

	// 扩大检查时间窗口，确保有足够的历史数据
	checkTime := cutoffTime.Add(-24 * time.Hour) // 检查最近24小时的数据

	// 查询该交易对该市场的统计数据状态
	query := `
		SELECT
			MAX(created_at) as last_update_time,
			COUNT(*) as record_count,
			AVG(CASE WHEN volume > 0 AND last_price > 0 THEN 1.0 ELSE 0.0 END) as data_quality
		FROM binance_24h_stats
		WHERE symbol = ? AND market_type = ? AND created_at >= ?
	`

	err := s.db.Raw(query, symbol, marketType, checkTime).Scan(&result).Error
	if err != nil {
		// 查询失败，假设需要同步
		log.Printf("[MarketStatsSyncer] 查询 %s %s 统计失败: %v", symbol, marketType, err)
		return true
	}

	// 如果没有记录，需要同步
	if result.LastUpdateTime.IsZero() {
		return true
	}

	// 如果记录数太少（少于最近24小时应有的记录数），需要同步
	// 24小时统计数据应该至少有最近几条记录
	expectedMinRecords := 3 // 保守的最小记录数，至少3条
	if result.RecordCount < expectedMinRecords {
		//log.Printf("[MarketStatsSyncer] %s %s 记录数不足 (%d < %d), 需要同步",
		//	symbol, marketType, result.RecordCount, expectedMinRecords)
		return true
	}

	// 如果数据质量太差（大量无效数据），需要同步
	if result.DataQuality < 0.8 { // 数据质量低于80%
		log.Printf("[MarketStatsSyncer] %s %s 数据质量不足 (%.2f%%), 需要同步",
			symbol, marketType, result.DataQuality*100)
		return true
	}

	// 如果最新统计时间太旧，需要同步
	if result.LastUpdateTime.Before(cutoffTime) {
		log.Printf("[MarketStatsSyncer] %s %s 数据过旧 (最新: %v, 截止: %v), 需要同步",
			symbol, marketType, result.LastUpdateTime, cutoffTime)
		return true
	}

	// 数据看起来是完整的，不需要同步
	return false
}

// containsString 检查字符串切片是否包含指定字符串
func (s *MarketStatsSyncer) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type MarketStatsSyncer struct {
	db     *gorm.DB
	cfg    *config.Config
	config *DataSyncConfig

	// 无效符号缓存，避免重复请求无效的交易对
	invalidSymbols struct {
		mu      sync.RWMutex
		symbols map[string]bool // symbol_kind -> true
	}

	// Redis缓存，用于跨服务共享无效符号
	redisCache *RedisInvalidSymbolCache

	stats struct {
		mu                 sync.RWMutex
		totalSyncs         int64
		successfulSyncs    int64
		failedSyncs        int64
		lastSyncTime       time.Time
		totalVolumeUpdates int64
	}
}

func NewMarketStatsSyncer(db *gorm.DB, cfg *config.Config, config *DataSyncConfig, redisCache *RedisInvalidSymbolCache) *MarketStatsSyncer {
	return &MarketStatsSyncer{
		db:     db,
		cfg:    cfg,
		config: config,
		invalidSymbols: struct {
			mu      sync.RWMutex
			symbols map[string]bool
		}{
			symbols: make(map[string]bool),
		},
		redisCache: redisCache,
	}
}

func (s *MarketStatsSyncer) markSymbolInvalid(symbol, kind string) {
	key := fmt.Sprintf("%s_%s", symbol, kind)
	s.invalidSymbols.mu.Lock()
	s.invalidSymbols.symbols[key] = true
	s.invalidSymbols.mu.Unlock()

	// 同时写入Redis缓存
	if s.redisCache != nil {
		if err := s.redisCache.MarkInvalid(symbol, kind); err != nil {
			log.Printf("[MarketStatsSyncer] Failed to mark invalid in Redis: %v", err)
		}
	}

	log.Printf("[MarketStatsSyncer] 🛑 Marked %s %s as invalid symbol", symbol, kind)
}

func (s *MarketStatsSyncer) isSymbolInvalid(symbol, kind string) bool {
	key := fmt.Sprintf("%s_%s", symbol, kind)

	// 首先检查内存缓存
	s.invalidSymbols.mu.RLock()
	invalid := s.invalidSymbols.symbols[key]
	s.invalidSymbols.mu.RUnlock()

	if invalid {
		log.Printf("[MarketStatsSyncer] 📋 内存缓存命中，跳过无效符号: %s %s", symbol, kind)
		return true
	}

	// 如果内存缓存中没有找到，检查Redis缓存
	if s.redisCache != nil {
		if s.redisCache.IsInvalid(symbol, kind) {
			// Redis中有记录，同时更新内存缓存
			s.invalidSymbols.mu.Lock()
			s.invalidSymbols.symbols[key] = true
			s.invalidSymbols.mu.Unlock()
			log.Printf("[MarketStatsSyncer] 🔄 Redis缓存命中，从Redis恢复无效符号: %s %s", symbol, kind)
			return true
		} else {
		}
	} else {
	}

	return false
}

func (s *MarketStatsSyncer) getSymbolsNeedingVolumeSync(allSymbols []string) ([]string, error) {
	if len(allSymbols) == 0 {
		return allSymbols, nil
	}

	// 设置市场统计数据过期时间（例如1小时）
	maxDataAge := time.Hour
	cutoffTime := time.Now().Add(-maxDataAge)

	// 使用通道收集结果
	type checkResult struct {
		symbol    string
		needsSync bool
	}

	resultChan := make(chan checkResult, len(allSymbols))
	var wg sync.WaitGroup

	// 限制并发数量，避免数据库压力过大
	maxConcurrency := 10
	semaphore := make(chan struct{}, maxConcurrency)

	// 并发检查每个交易对
	for _, symbol := range allSymbols {
		wg.Add(1)
		go func(sym string) {
			defer wg.Done()

			// 获取信号量
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// 检查该交易对是否需要同步
			needsSync := s.checkSymbolNeedsStatsSync(sym, cutoffTime)
			resultChan <- checkResult{symbol: sym, needsSync: needsSync}
		}(symbol)
	}

	// 等待所有检查完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	var symbolsToSync []string
	for result := range resultChan {
		if result.needsSync {
			symbolsToSync = append(symbolsToSync, result.symbol)
		}
	}

	// 兜底策略：确保至少同步核心交易对
	if len(symbolsToSync) == 0 && len(allSymbols) > 0 {
		coreSymbols := []string{"BTCUSDT", "ETHUSDT"}
		for _, coreSymbol := range coreSymbols {
			if s.containsString(allSymbols, coreSymbol) {
				symbolsToSync = append(symbolsToSync, coreSymbol)
			}
		}
	}

	return symbolsToSync, nil
}

func (s *MarketStatsSyncer) checkSymbolNeedsStatsSync(symbol string, cutoffTime time.Time) bool {
	var result struct {
		LastUpdate  time.Time `json:"last_update"`
		RecordCount int       `json:"record_count"`
	}

	query := `
		SELECT MAX(created_at) as last_update, COUNT(*) as record_count
		FROM binance_24h_stats
		WHERE symbol = ? AND created_at >= ?
	`

	err := s.db.Raw(query, symbol, cutoffTime).Scan(&result).Error
	if err != nil {
		// 查询失败，假设需要同步
		log.Printf("[MarketStatsSyncer] 查询 %s 失败: %v", symbol, err)
		return true
	}

	// 如果没有记录或记录数太少，需要同步
	if result.LastUpdate.IsZero() || result.RecordCount < 10 {
		return true
	}

	// 如果最新记录太旧，需要同步
	if result.LastUpdate.Before(cutoffTime) {
		return true
	}

	return false
}

func (s *MarketStatsSyncer) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[MarketStatsSyncer] Started with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[MarketStatsSyncer] Stopped")
			return
		case <-ticker.C:
			if err := s.Sync(ctx); err != nil {
				log.Printf("[MarketStatsSyncer] Sync failed: %v", err)
			}
		}
	}
}

func (s *MarketStatsSyncer) Stop() {
	log.Printf("[MarketStatsSyncer] Stop signal received")
}

func (s *MarketStatsSyncer) Name() string {
	return "MarketStatsSyncer"
}

func (s *MarketStatsSyncer) Sync(ctx context.Context) error {
	s.stats.mu.Lock()
	s.stats.totalSyncs++
	syncStartTime := time.Now()
	s.stats.lastSyncTime = syncStartTime
	s.stats.mu.Unlock()

	log.Printf("[MarketStatsSyncer] 🎯 Starting market-separated stats sync")

	// 获取现货和期货交易对配置
	syncerConfig := s.buildMarketStatsSyncerConfig()

	totalUpdates := 0
	totalErrors := 0

	// 同步现货市场统计
	if len(syncerConfig.SpotSymbols) > 0 {
		log.Printf("[MarketStatsSyncer] 📈 Starting spot market stats sync for %d symbols", len(syncerConfig.SpotSymbols))
		spotUpdates, spotErrors := s.syncMarketStats(ctx, syncerConfig.SpotSymbols, "spot")
		totalUpdates += spotUpdates
		totalErrors += spotErrors
	} else {
		log.Printf("[MarketStatsSyncer] ⚠️ No spot symbols to sync")
	}

	// 同步期货市场统计
	if len(syncerConfig.FuturesSymbols) > 0 {
		log.Printf("[MarketStatsSyncer] 📈 Starting futures market stats sync for %d symbols", len(syncerConfig.FuturesSymbols))
		futuresUpdates, futuresErrors := s.syncMarketStats(ctx, syncerConfig.FuturesSymbols, "futures")
		totalUpdates += futuresUpdates
		totalErrors += futuresErrors
	} else {
		log.Printf("[MarketStatsSyncer] ⚠️ No futures symbols to sync")
	}

	totalDuration := time.Since(syncStartTime)

	s.stats.mu.Lock()
	if totalErrors == 0 {
		s.stats.successfulSyncs++
	}
	s.stats.totalVolumeUpdates += int64(totalUpdates)
	s.stats.mu.Unlock()

	// 生成详细的同步报告
	log.Printf("[MarketStatsSyncer] 📊 Stats sync completed in %v", totalDuration)
	log.Printf("[MarketStatsSyncer] 📈 Total updates: %d", totalUpdates)
	log.Printf("[MarketStatsSyncer] 📊 Markets synced: spot(%d), futures(%d)",
		len(syncerConfig.SpotSymbols), len(syncerConfig.FuturesSymbols))

	if totalErrors > 0 {
		log.Printf("[MarketStatsSyncer] ⚠️ %d markets had errors - check logs above", totalErrors)
		return fmt.Errorf("completed with %d market errors", totalErrors)
	}

	return nil
}

// getFuturesBookTicker 获取期货买卖盘口数据
func (s *MarketStatsSyncer) getFuturesBookTicker(ctx context.Context, symbol string) (map[string]string, error) {
	// FuturesBookTicker 期货买卖盘口数据结构
	type FuturesBookTicker struct {
		Symbol   string `json:"symbol"`
		BidPrice string `json:"bidPrice"`
		BidQty   string `json:"bidQty"`
		AskPrice string `json:"askPrice"`
		AskQty   string `json:"askQty"`
		Time     int64  `json:"time"`
	}

	url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/ticker/bookTicker?symbol=%s", symbol)

	var bookTicker FuturesBookTicker
	if err := netutil.GetJSON(ctx, url, &bookTicker); err != nil {
		errStr := err.Error()
		if strings.Contains(errStr, "Invalid symbol") || strings.Contains(errStr, "-1121") {
			s.markSymbolInvalid(symbol, "futures")
			return nil, fmt.Errorf("invalid futures symbol: %s", symbol)
		}
		return nil, fmt.Errorf("failed to get futures book ticker: %w", err)
	}

	// 返回买卖盘口数据
	return map[string]string{
		"bidPrice": bookTicker.BidPrice,
		"bidQty":   bookTicker.BidQty,
		"askPrice": bookTicker.AskPrice,
		"askQty":   bookTicker.AskQty,
	}, nil
}

func (s *MarketStatsSyncer) sync24hStats(ctx context.Context, symbol, kind string) error {
	// 检查是否为已知的无效符号
	if s.isSymbolInvalid(symbol, kind) {
		return fmt.Errorf("symbol marked as invalid, skipping")
	}

	var url string
	if kind == "spot" {
		url = fmt.Sprintf("https://api.binance.com/api/v3/ticker/24hr?symbol=%s", symbol)
	} else {
		url = fmt.Sprintf("https://fapi.binance.com/fapi/v1/ticker/24hr?symbol=%s", symbol)
	}

	type Ticker24h struct {
		Symbol             string `json:"symbol"`
		PriceChange        string `json:"priceChange"`
		PriceChangePercent string `json:"priceChangePercent"`
		WeightedAvgPrice   string `json:"weightedAvgPrice"`
		PrevClosePrice     string `json:"prevClosePrice"`
		LastPrice          string `json:"lastPrice"`
		LastQty            string `json:"lastQty"` // 最后交易数量
		BidPrice           string `json:"bidPrice"`
		BidQty             string `json:"bidQty"` // 买一档数量
		AskPrice           string `json:"askPrice"`
		AskQty             string `json:"askQty"` // 卖一档数量
		OpenPrice          string `json:"openPrice"`
		HighPrice          string `json:"highPrice"`
		LowPrice           string `json:"lowPrice"`
		Volume             string `json:"volume"`
		QuoteVolume        string `json:"quoteVolume"`
		OpenTime           int64  `json:"openTime"`
		CloseTime          int64  `json:"closeTime"`
		FirstId            int64  `json:"firstId"` // 第一笔交易ID
		LastId             int64  `json:"lastId"`  // 最后一笔交易ID
		Count              int64  `json:"count"`
	}

	var ticker Ticker24h
	if err := netutil.GetJSON(ctx, url, &ticker); err != nil {
		// 检查是否为无效符号错误
		errStr := err.Error()
		if strings.Contains(errStr, "Invalid symbol") || strings.Contains(errStr, "-1121") {
			s.markSymbolInvalid(symbol, kind)
			return fmt.Errorf("invalid symbol: %s %s", symbol, kind)
		}
		return fmt.Errorf("failed to get 24h ticker: %w", err)
	}

	// 对于期货市场，额外获取买卖盘口数据
	if kind == "futures" {
		bookTicker, err := s.getFuturesBookTicker(ctx, symbol)
		if err != nil {
			log.Printf("[MarketStatsSyncer] Failed to get futures book ticker for %s: %v, using default values", symbol, err)
		} else if bookTicker != nil {
			// 将买卖盘口数据合并到24小时统计中
			ticker.BidPrice = bookTicker["bidPrice"]
			ticker.BidQty = bookTicker["bidQty"]
			ticker.AskPrice = bookTicker["askPrice"]
			ticker.AskQty = bookTicker["askQty"]
			log.Printf("[MarketStatsSyncer] Merged futures book ticker data for %s", symbol)
		}
	}

	// 创建24小时统计数据对象
	statsData := pdb.Binance24hStats{
		Symbol:             ticker.Symbol,
		MarketType:         kind,
		PriceChange:        parseFloat(ticker.PriceChange),
		PriceChangePercent: parseFloat(ticker.PriceChangePercent),
		WeightedAvgPrice:   parseFloat(ticker.WeightedAvgPrice),
		PrevClosePrice:     parseFloat(ticker.PrevClosePrice),
		LastPrice:          parseFloat(ticker.LastPrice),
		LastQty:            parseFloat(ticker.LastQty), // 最后交易数量
		BidPrice:           parseFloat(ticker.BidPrice),
		BidQty:             parseFloat(ticker.BidQty), // 买一档数量
		AskPrice:           parseFloat(ticker.AskPrice),
		AskQty:             parseFloat(ticker.AskQty), // 卖一档数量
		OpenPrice:          parseFloat(ticker.OpenPrice),
		HighPrice:          parseFloat(ticker.HighPrice),
		LowPrice:           parseFloat(ticker.LowPrice),
		Volume:             parseFloat(ticker.Volume),
		QuoteVolume:        parseFloat(ticker.QuoteVolume),
		OpenTime:           ticker.OpenTime,
		CloseTime:          ticker.CloseTime,
		FirstId:            ticker.FirstId, // 第一笔交易ID
		LastId:             ticker.LastId,  // 最后一笔交易ID
		Count:              ticker.Count,
	}

	// 创建历史统计数据对象
	historyStats := s.createHistoryStatsFromRealtime(statsData)

	// 并发保存到实时表和历史表
	if err := s.saveStatsDualTable(statsData, historyStats); err != nil {
		return fmt.Errorf("failed to save dual table stats: %w", err)
	}

	log.Printf("[MarketStatsSyncer] Saved %s %s dual table stats - Volume: %.2f, Quote Volume: %.2f, Price Change: %.2f%%, Bid: %.2f(%.4f), Ask: %.2f(%.4f)",
		symbol, kind, statsData.Volume, statsData.QuoteVolume, statsData.PriceChangePercent,
		statsData.BidPrice, statsData.BidQty, statsData.AskPrice, statsData.AskQty)

	return nil
}

// createHistoryStatsFromRealtime 从实时统计数据创建历史统计数据
func (s *MarketStatsSyncer) createHistoryStatsFromRealtime(realtimeStats pdb.Binance24hStats) pdb.Binance24hStatsHistory {
	// 计算当前时间窗口
	windowStart, windowEnd := s.calculateCurrentTimeWindow()

	return pdb.Binance24hStatsHistory{
		Symbol:         realtimeStats.Symbol,
		MarketType:     realtimeStats.MarketType,
		WindowStart:    windowStart,
		WindowEnd:      windowEnd,
		WindowDuration: 3600, // 1小时
		// 复制所有统计数据字段
		PriceChange:        realtimeStats.PriceChange,
		PriceChangePercent: realtimeStats.PriceChangePercent,
		WeightedAvgPrice:   realtimeStats.WeightedAvgPrice,
		PrevClosePrice:     realtimeStats.PrevClosePrice,
		LastPrice:          realtimeStats.LastPrice,
		LastQty:            realtimeStats.LastQty,
		BidPrice:           realtimeStats.BidPrice,
		BidQty:             realtimeStats.BidQty,
		AskPrice:           realtimeStats.AskPrice,
		AskQty:             realtimeStats.AskQty,
		OpenPrice:          realtimeStats.OpenPrice,
		HighPrice:          realtimeStats.HighPrice,
		LowPrice:           realtimeStats.LowPrice,
		Volume:             realtimeStats.Volume,
		QuoteVolume:        realtimeStats.QuoteVolume,
		OpenTime:           realtimeStats.OpenTime,
		CloseTime:          realtimeStats.CloseTime,
		FirstId:            realtimeStats.FirstId,
		LastId:             realtimeStats.LastId,
		Count:              realtimeStats.Count,
	}
}

// calculateCurrentTimeWindow 计算当前时间窗口
func (s *MarketStatsSyncer) calculateCurrentTimeWindow() (windowStart, windowEnd time.Time) {
	now := time.Now().UTC()

	// 1小时时间窗口对齐到整点
	windowStart = time.Date(
		now.Year(), now.Month(), now.Day(),
		now.Hour(), 0, 0, 0, time.UTC,
	)
	windowEnd = windowStart.Add(time.Hour)

	return windowStart, windowEnd
}

// saveStatsDualTable 双表保存逻辑
func (s *MarketStatsSyncer) saveStatsDualTable(realtimeStats pdb.Binance24hStats, historyStats pdb.Binance24hStatsHistory) error {
	// 使用goroutine并发保存到两张表，提高性能
	var wg sync.WaitGroup
	var realtimeErr, historyErr error
	var mu sync.Mutex

	// 保存到实时表
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pdb.Save24hStats(s.db, []pdb.Binance24hStats{realtimeStats}); err != nil {
			mu.Lock()
			realtimeErr = fmt.Errorf("failed to save realtime stats: %w", err)
			mu.Unlock()
		}
	}()

	// 保存到历史表
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := pdb.Save24hStatsHistory(s.db, []pdb.Binance24hStatsHistory{historyStats}); err != nil {
			mu.Lock()
			historyErr = fmt.Errorf("failed to save history stats: %w", err)
			mu.Unlock()
		}
	}()

	// 等待所有goroutine完成
	wg.Wait()

	// 收集错误信息
	var errors []string
	if realtimeErr != nil {
		errors = append(errors, realtimeErr.Error())
	}
	if historyErr != nil {
		errors = append(errors, historyErr.Error())
	}

	if len(errors) > 0 {
		return fmt.Errorf("dual table save failed: %s", strings.Join(errors, "; "))
	}

	return nil
}

func (s *MarketStatsSyncer) GetStats() map[string]interface{} {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	return map[string]interface{}{
		"total_syncs":      s.stats.totalSyncs,
		"successful_syncs": s.stats.successfulSyncs,
		"failed_syncs":     s.stats.failedSyncs,
		"last_sync_time":   s.stats.lastSyncTime,
		"total_updates":    s.stats.totalVolumeUpdates,
	}
}
