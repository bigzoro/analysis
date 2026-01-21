package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/netutil"
	"analysis/internal/server"

	"gorm.io/gorm"
)

// ===== 深度同步器 =====

// DepthSyncerConfig 深度同步器配置
type DepthSyncerConfig struct {
	SpotSymbols    []string // 现货交易对
	FuturesSymbols []string // 期货交易对
}

// buildDepthSyncerConfig 构建深度同步器配置
func (s *DepthSyncer) buildDepthSyncerConfig() DepthSyncerConfig {
	config := DepthSyncerConfig{}

	// 优先从数据库获取各市场的有效交易对，避免使用包含无效符号的全局配置
	if spotSymbols, err := pdb.GetUSDTTradingPairsByMarket(s.db, "spot"); err == nil {
		// 过滤掉Redis缓存中标记为无效的符号
		config.SpotSymbols = s.filterOutInvalidSymbols(spotSymbols, "spot")
		log.Printf("[DepthSyncer] ✅ Loaded %d spot symbols from database (%d after filtering invalid)",
			len(spotSymbols), len(config.SpotSymbols))
	} else {
		log.Printf("[DepthSyncer] ⚠️ Failed to get spot symbols: %v", err)
		// 如果数据库查询失败，尝试从配置中获取
		if len(s.config.Symbols) > 0 {
			config.SpotSymbols = s.config.Symbols
			log.Printf("[DepthSyncer] 🔄 Using configured symbols as fallback for spot: %d symbols", len(config.SpotSymbols))
		}
	}

	if futuresSymbols, err := pdb.GetUSDTTradingPairsByMarket(s.db, "futures"); err == nil {
		// 过滤掉Redis缓存中标记为无效的符号
		config.FuturesSymbols = s.filterOutInvalidSymbols(futuresSymbols, "futures")
		log.Printf("[DepthSyncer] ✅ Loaded %d futures symbols from database (%d after filtering invalid)",
			len(futuresSymbols), len(config.FuturesSymbols))
	} else {
		log.Printf("[DepthSyncer] ⚠️ Failed to get futures symbols: %v", err)
		// 如果数据库查询失败，尝试从配置中获取
		if len(s.config.Symbols) > 0 {
			config.FuturesSymbols = s.config.Symbols
			log.Printf("[DepthSyncer] 🔄 Using configured symbols as fallback for futures: %d symbols", len(config.FuturesSymbols))
		}
	}

	return config
}

// isSymbolInvalid 检查交易对是否为无效符号
func (s *DepthSyncer) isSymbolInvalid(symbol, kind string) bool {
	// 首先检查Redis缓存（跨服务共享）
	if s.redisCache != nil && s.redisCache.IsInvalid(symbol, kind) {
		return true
	}

	// DepthSyncer没有本地内存缓存，直接返回false
	return false
}

// filterOutInvalidSymbols 过滤掉Redis缓存中标记为无效的符号
func (s *DepthSyncer) filterOutInvalidSymbols(symbols []string, marketType string) []string {
	if len(symbols) == 0 {
		return symbols
	}

	var validSymbols []string
	for _, symbol := range symbols {
		if !s.isSymbolInvalid(symbol, marketType) {
			validSymbols = append(validSymbols, symbol)
		} else {
			log.Printf("[DepthSyncer] 🗑️ Filtered out invalid symbol: %s %s", symbol, marketType)
		}
	}

	return validSymbols
}

// filterConfiguredSymbols 过滤出配置中存在的交易对
func (s *DepthSyncer) filterConfiguredSymbols(configured, available []string) []string {
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

// syncMarketDepth 同步指定市场的深度数据
func (s *DepthSyncer) syncMarketDepth(ctx context.Context, symbols []string, marketType string) (int, int) {
	if len(symbols) == 0 {
		return 0, 0
	}

	var symbolsToSync []string

	// 🔄 增量同步：只同步需要更新的交易对（如果启用）
	if s.config.EnableIncrementalSync {
		log.Printf("[DepthSyncer] 🔄 Incremental sync enabled for %s market, checking for outdated symbols...", marketType)
		filteredSymbols, err := s.getSymbolsNeedingDepthSyncByMarket(symbols, marketType)
		if err != nil {
			log.Printf("[DepthSyncer] ⚠️ Failed to determine symbols needing %s depth sync: %v, falling back to full sync", marketType, err)
			symbolsToSync = symbols // 回退到全量同步
		} else {
			symbolsToSync = filteredSymbols
		}
	} else {
		log.Printf("[DepthSyncer] 🔄 Incremental sync disabled for %s market, performing full sync...", marketType)
		symbolsToSync = symbols // 全量同步
	}

	log.Printf("[DepthSyncer] 🎯 Starting %s market depth sync for %d/%d symbols",
		marketType, len(symbolsToSync), len(symbols))

	// 如果没有需要同步的交易对，跳过同步
	if len(symbolsToSync) == 0 {
		log.Printf("[DepthSyncer] ✅ All %s market symbols are up-to-date, skipping depth sync", marketType)
		return 0, 0
	}

	// 临时保存原始symbols并设置新的symbols
	originalSymbols := s.config.Symbols
	s.config.Symbols = symbolsToSync                      // 只同步需要更新的交易对
	defer func() { s.config.Symbols = originalSymbols }() // 恢复原始配置

	updates := 0
	errors := 0

	for i, symbol := range symbolsToSync {
		// 获取订单簿深度
		if err := s.syncOrderBookDepth(ctx, symbol, marketType); err != nil {
			log.Printf("[DepthSyncer] ❌ Failed to sync %s depth for %s: %v", marketType, symbol, err)
			errors++
		} else {
			log.Printf("[DepthSyncer] ✅ Synced %s depth for %s", marketType, symbol)
			updates++
		}

		// 添加小延迟避免API限流，每处理10个交易对后增加延迟
		if (i+1)%10 == 0 && i < len(symbolsToSync)-1 {
			time.Sleep(200 * time.Millisecond)
			log.Printf("[DepthSyncer] Added delay after processing %d %s market symbols to prevent API rate limiting", i+1, marketType)
		}
	}

	log.Printf("[DepthSyncer] 📊 %s market depth sync completed: %d updates, %d errors",
		marketType, updates, errors)

	return updates, errors
}

// getSymbolsNeedingDepthSyncByMarket 按市场获取需要深度同步的交易对
func (s *DepthSyncer) getSymbolsNeedingDepthSyncByMarket(allSymbols []string, marketType string) ([]string, error) {
	if len(allSymbols) == 0 {
		return allSymbols, nil
	}

	// 设置深度数据过期时间（例如10分钟）
	maxDataAge := 10 * time.Minute
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
			needsSync := s.checkSymbolNeedsDepthSyncByMarket(sym, marketType, cutoffTime)
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

	log.Printf("[DepthSyncer] 🔄 %s market incremental sync: %d/%d symbols need depth updating",
		marketType, len(symbolsToSync), len(allSymbols))

	return symbolsToSync, nil
}

// checkSymbolNeedsDepthSyncByMarket 检查单个交易对在指定市场是否需要深度同步
func (s *DepthSyncer) checkSymbolNeedsDepthSyncByMarket(symbol, marketType string, cutoffTime time.Time) bool {
	var result struct {
		LastUpdateTime time.Time `json:"last_update_time"`
		RecordCount    int       `json:"record_count"`
	}

	// 查询该交易对该市场的最新深度时间
	query := `
		SELECT MAX(created_at) as last_update_time, COUNT(*) as record_count
		FROM binance_order_book_depth
		WHERE symbol = ? AND market_type = ? AND created_at >= ?
	`

	err := s.db.Raw(query, symbol, marketType, cutoffTime).Scan(&result).Error
	if err != nil {
		// 查询失败，假设需要同步
		log.Printf("[DepthSyncer] 查询 %s %s 深度失败: %v", symbol, marketType, err)
		return true
	}

	// 如果没有记录或记录数太少，需要同步
	if result.LastUpdateTime.IsZero() || result.RecordCount < 3 {
		return true
	}

	// 如果最新深度时间太旧，需要同步
	if result.LastUpdateTime.Before(cutoffTime) {
		return true
	}

	return false
}

// containsString 检查字符串切片是否包含指定字符串
func (s *DepthSyncer) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

type DepthSyncer struct {
	db     *gorm.DB
	cfg    *config.Config
	config *DataSyncConfig

	// Redis缓存，用于跨服务共享无效符号
	redisCache *RedisInvalidSymbolCache

	stats struct {
		mu                 sync.RWMutex
		totalSyncs         int64
		successfulSyncs    int64
		failedSyncs        int64
		lastSyncTime       time.Time
		totalDepthUpdates  int64
		totalAPICalls      int64
		successfulAPICalls int64
		totalLatency       time.Duration
	}
}

func NewDepthSyncer(db *gorm.DB, cfg *config.Config, config *DataSyncConfig, redisCache *RedisInvalidSymbolCache) *DepthSyncer {
	return &DepthSyncer{
		db:         db,
		cfg:        cfg,
		config:     config,
		redisCache: redisCache,
	}
}

func (s *DepthSyncer) Name() string {
	return "depth"
}

// getSymbolsNeedingDepthSync 增量同步：获取需要同步市场深度的交易对
// 超优化版本：并发查询，市场深度变化快，需要快速检查
func (s *DepthSyncer) getSymbolsNeedingDepthSync(allSymbols []string) ([]string, error) {
	if len(allSymbols) == 0 {
		return allSymbols, nil
	}

	// 设置市场深度数据过期时间（30秒，深度数据变化很快）
	maxDataAge := 30 * time.Second
	cutoffTime := time.Now().Add(-maxDataAge)

	// 使用通道收集结果
	type checkResult struct {
		symbol    string
		needsSync bool
	}

	resultChan := make(chan checkResult, len(allSymbols))
	var wg sync.WaitGroup

	// 限制并发数量，避免数据库压力过大
	maxConcurrency := 20 // 深度检查并发可以更高，因为查询很简单
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
			needsSync := s.checkSymbolNeedsDepthSync(sym, cutoffTime)
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

	log.Printf("[DepthSyncer] 🔄 Incremental sync: %d/%d symbols need depth updating",
		len(symbolsToSync), len(allSymbols))

	return symbolsToSync, nil
}

// checkSymbolNeedsDepthSync 检查单个交易对是否需要深度同步
func (s *DepthSyncer) checkSymbolNeedsDepthSync(symbol string, cutoffTime time.Time) bool {
	var result struct {
		LastUpdate time.Time `json:"last_update"`
	}

	// 查询该交易对的最新深度更新时间
	query := `
		SELECT MAX(created_at) as last_update
		FROM binance_order_book_depth
		WHERE symbol = ?
	`

	err := s.db.Raw(query, symbol).Scan(&result).Error
	if err != nil {
		// 查询失败，假设需要同步
		return true
	}

	// 如果没有记录，需要同步
	if result.LastUpdate.IsZero() {
		return true
	}

	// 如果最新记录太旧，需要同步
	if result.LastUpdate.Before(cutoffTime) {
		return true
	}

	return false
}

func (s *DepthSyncer) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[DepthSyncer] Started with interval: %v", interval)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[DepthSyncer] Stopped")
			return
		case <-ticker.C:
			if err := s.Sync(ctx); err != nil {
				log.Printf("[DepthSyncer] Sync failed: %v", err)
			}
		}
	}
}

func (s *DepthSyncer) Stop() {
	log.Printf("[DepthSyncer] Stop signal received")
}

func (s *DepthSyncer) Sync(ctx context.Context) error {
	s.stats.mu.Lock()
	s.stats.totalSyncs++
	syncStartTime := time.Now()
	s.stats.lastSyncTime = syncStartTime
	s.stats.mu.Unlock()

	log.Printf("[DepthSyncer] 🎯 Starting market-separated depth sync")

	// 获取现货和期货交易对配置
	syncerConfig := s.buildDepthSyncerConfig()

	totalUpdates := 0
	totalErrors := 0

	// 同步现货市场深度
	if len(syncerConfig.SpotSymbols) > 0 {
		log.Printf("[DepthSyncer] 📈 Starting spot market depth sync for %d symbols", len(syncerConfig.SpotSymbols))
		spotUpdates, spotErrors := s.syncMarketDepth(ctx, syncerConfig.SpotSymbols, "spot")
		totalUpdates += spotUpdates
		totalErrors += spotErrors
	} else {
		log.Printf("[DepthSyncer] ⚠️ No spot symbols to sync")
	}

	// 同步期货市场深度
	if len(syncerConfig.FuturesSymbols) > 0 {
		log.Printf("[DepthSyncer] 📈 Starting futures market depth sync for %d symbols", len(syncerConfig.FuturesSymbols))
		futuresUpdates, futuresErrors := s.syncMarketDepth(ctx, syncerConfig.FuturesSymbols, "futures")
		totalUpdates += futuresUpdates
		totalErrors += futuresErrors
	} else {
		log.Printf("[DepthSyncer] ⚠️ No futures symbols to sync")
	}

	totalDuration := time.Since(syncStartTime)

	s.stats.mu.Lock()
	if totalErrors == 0 {
		s.stats.successfulSyncs++
	}
	s.stats.totalDepthUpdates += int64(totalUpdates)
	s.stats.mu.Unlock()

	// 生成详细的同步报告
	log.Printf("[DepthSyncer] 📊 Depth sync completed in %v", totalDuration)
	log.Printf("[DepthSyncer] 📈 Total updates: %d", totalUpdates)
	log.Printf("[DepthSyncer] 📊 Markets synced: spot(%d), futures(%d)",
		len(syncerConfig.SpotSymbols), len(syncerConfig.FuturesSymbols))

	if totalErrors > 0 {
		log.Printf("[DepthSyncer] ⚠️ %d markets had errors - check logs above", totalErrors)
		return fmt.Errorf("completed with %d market errors", totalErrors)
	}

	return nil
}

func (s *DepthSyncer) syncOrderBookDepth(ctx context.Context, symbol, kind string) error {
	// 检查是否为无效符号
	if s.redisCache != nil && s.redisCache.IsInvalid(symbol, kind) {
		return fmt.Errorf("symbol marked as invalid, skipping")
	}

	// 等待获取API调用令牌（速率限制）
	// 使用深度专用速率限制器
	if err := DepthAPIRateLimiter.WaitForToken(ctx); err != nil {
		return fmt.Errorf("failed to acquire depth rate limit token: %w", err)
	}

	var url string
	if kind == "spot" {
		url = fmt.Sprintf("https://api.binance.com/api/v3/depth?symbol=%s&limit=20", symbol)
	} else {
		url = fmt.Sprintf("https://fapi.binance.com/fapi/v1/depth?symbol=%s&limit=20", symbol)
	}

	type OrderBook struct {
		LastUpdateId int64      `json:"lastUpdateId"`
		Bids         [][]string `json:"bids"` // [price, quantity]
		Asks         [][]string `json:"asks"` // [price, quantity]
	}

	var book OrderBook
	if err := netutil.GetJSON(ctx, url, &book); err != nil {
		// 检查是否为无效符号错误
		errStr := err.Error()
		if strings.Contains(errStr, "Invalid symbol") || strings.Contains(errStr, "-1121") {
			// 标记为无效符号
			if s.redisCache != nil {
				if markErr := s.redisCache.MarkInvalid(symbol, kind); markErr != nil {
					log.Printf("[DepthSyncer] ⚠️ Failed to mark invalid in Redis: %v", markErr)
				}
			}
			log.Printf("[DepthSyncer] 🛑 Marked %s %s as invalid symbol", symbol, kind)
			return fmt.Errorf("invalid symbol: %s %s", symbol, kind)
		}
		return fmt.Errorf("failed to get order book: %w", err)
	}

	// 将数据转换为JSON字符串
	bidsJSON, _ := json.Marshal(book.Bids)
	asksJSON, _ := json.Marshal(book.Asks)

	// 创建深度数据对象
	depthData := pdb.BinanceOrderBookDepth{
		Symbol:       symbol,
		MarketType:   kind,
		LastUpdateID: book.LastUpdateId,
		Bids:         string(bidsJSON),
		Asks:         string(asksJSON),
		SnapshotTime: time.Now().UnixMilli(), // 使用毫秒时间戳
	}

	// 保存到数据库
	if err := pdb.SaveOrderBookDepth(s.db, []pdb.BinanceOrderBookDepth{depthData}); err != nil {
		return fmt.Errorf("failed to save order book depth: %w", err)
	}

	// 计算买卖价差用于日志
	if len(book.Bids) > 0 && len(book.Asks) > 0 {
		bestBid := book.Bids[0][0]
		bestAsk := book.Asks[0][0]
		spread := fmt.Sprintf("%.4f", (parseFloat(bestAsk)-parseFloat(bestBid))/parseFloat(bestBid)*100)

		log.Printf("[DepthSyncer] Saved %s %s depth - ID: %d, Bids: %d, Asks: %d, Spread: %s%%",
			symbol, kind, book.LastUpdateId, len(book.Bids), len(book.Asks), spread)
	}

	return nil
}

func (s *DepthSyncer) GetStats() map[string]interface{} {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	return map[string]interface{}{
		"total_syncs":      s.stats.totalSyncs,
		"successful_syncs": s.stats.successfulSyncs,
		"failed_syncs":     s.stats.failedSyncs,
		"last_sync_time":   s.stats.lastSyncTime,
		"total_updates":    s.stats.totalDepthUpdates,
	}
}

// GetAPIStats 获取API统计信息
func (s *DepthSyncer) GetAPIStats() *server.APIStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	successRate := "0%"
	if s.stats.totalAPICalls > 0 {
		rate := float64(s.stats.successfulAPICalls) / float64(s.stats.totalAPICalls) * 100
		successRate = fmt.Sprintf("%.1f%%", rate)
	}

	avgLatency := ""
	if s.stats.totalAPICalls > 0 && s.stats.totalLatency > 0 {
		avg := s.stats.totalLatency / time.Duration(s.stats.totalAPICalls)
		avgLatency = avg.String()
	}

	return &server.APIStats{
		TotalCalls:      s.stats.totalAPICalls,
		APICallsTotal:   s.stats.totalAPICalls,
		APISuccessRate:  successRate,
		APIAvgLatency:   &avgLatency,
		TotalSyncs:      s.stats.totalSyncs,
		SuccessfulSyncs: s.stats.successfulSyncs,
		FailedSyncs:     s.stats.failedSyncs,
		LastSyncTime:    &s.stats.lastSyncTime,
		TotalUpdates:    s.stats.totalDepthUpdates,
	}
}
