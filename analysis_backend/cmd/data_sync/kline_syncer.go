package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"math/rand"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"analysis/internal/analysis"
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"

	"gorm.io/gorm"
)

type KlineSyncerConfig struct {
	SpotSymbols    []string // 现货交易对
	FuturesSymbols []string // 期货交易对
}

// buildKlineSyncerConfig 构建K线同步器配置
func (s *KlineSyncer) buildKlineSyncerConfig() KlineSyncerConfig {
	config := KlineSyncerConfig{}

	// 优先从数据库获取各市场的有效交易对，避免使用包含无效符号的全局配置
	if spotSymbols, err := pdb.GetUSDTTradingPairsByMarket(s.db, "spot"); err == nil {
		// 过滤掉Redis缓存中标记为无效的符号
		config.SpotSymbols = s.filterOutInvalidSymbols(spotSymbols, "spot")
		log.Printf("[KlineSyncer] ✅ Loaded %d spot symbols from database (%d after filtering invalid)",
			len(spotSymbols), len(config.SpotSymbols))
	} else {
		log.Printf("[KlineSyncer] ⚠️ Failed to get spot symbols: %v", err)
		// 如果数据库查询失败，尝试从配置中获取
		if len(s.config.Symbols) > 0 {
			config.SpotSymbols = s.config.Symbols
			log.Printf("[KlineSyncer] 🔄 Using configured symbols as fallback for spot: %d symbols", len(config.SpotSymbols))
		}
	}

	if futuresSymbols, err := pdb.GetUSDTTradingPairsByMarket(s.db, "futures"); err == nil {
		// 过滤掉Redis缓存中标记为无效的符号
		config.FuturesSymbols = s.filterOutInvalidSymbols(futuresSymbols, "futures")
		log.Printf("[KlineSyncer] ✅ Loaded %d futures symbols from database (%d after filtering invalid)",
			len(futuresSymbols), len(config.FuturesSymbols))
	} else {
		log.Printf("[KlineSyncer] ⚠️ Failed to get futures symbols: %v", err)
		// 如果数据库查询失败，尝试从配置中获取
		if len(s.config.Symbols) > 0 {
			config.FuturesSymbols = s.config.Symbols
			log.Printf("[KlineSyncer] 🔄 Using configured symbols as fallback for futures: %d symbols", len(config.FuturesSymbols))
		}
	}

	return config
}

// filterOutInvalidSymbols 过滤掉Redis缓存中标记为无效的符号
func (s *KlineSyncer) filterOutInvalidSymbols(symbols []string, marketType string) []string {
	if len(symbols) == 0 {
		return symbols
	}

	var validSymbols []string
	for _, symbol := range symbols {
		if !s.isSymbolInvalid(symbol, marketType) {
			validSymbols = append(validSymbols, symbol)
		} else {
			log.Printf("[KlineSyncer] 🗑️ Filtered out invalid symbol: %s %s", symbol, marketType)
		}
	}

	return validSymbols
}

// filterConfiguredSymbols 过滤出配置中存在的交易对
func (s *KlineSyncer) filterConfiguredSymbols(configured, available []string) []string {
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

// syncMarketKlines 同步指定市场的K线数据
func (s *KlineSyncer) syncMarketKlines(ctx context.Context, symbols []string, marketType string) (int, int) {
	if len(symbols) == 0 {
		return 0, 0
	}

	var symbolsToSync []string

	// 🔄 增量同步：只同步需要更新的交易对（如果启用）
	if s.config.EnableIncrementalSync {
		log.Printf("[KlineSyncer] 🔄 Incremental sync enabled for %s market, checking for outdated symbols...", marketType)
		filteredSymbols, err := s.getSymbolsNeedingKlineSyncByMarket(symbols, marketType)
		if err != nil {
			log.Printf("[KlineSyncer] ⚠️ Failed to determine symbols needing %s kline sync: %v, falling back to full sync", marketType, err)
			symbolsToSync = symbols // 回退到全量同步
		} else {
			symbolsToSync = filteredSymbols
		}
	} else {
		log.Printf("[KlineSyncer] 🔄 Incremental sync disabled for %s market, performing full sync...", marketType)
		symbolsToSync = symbols // 全量同步
	}

	log.Printf("[KlineSyncer] 🎯 Starting %s market kline sync for %d intervals and %d/%d symbols",
		marketType, len(s.config.KlineIntervals), len(symbolsToSync), len(symbols))

	// 如果没有需要同步的交易对，跳过同步
	if len(symbolsToSync) == 0 {
		log.Printf("[KlineSyncer] ✅ All %s market symbols are up-to-date, skipping sync", marketType)
		return 0, 0
	}

	// 随机化处理顺序：减少热点冲突和死锁风险
	if len(symbolsToSync) > 10 {
		log.Printf("[KlineSyncer] 🔀 随机化 %d 个交易对的处理顺序以减少死锁风险", len(symbolsToSync))
		symbolsToSync = s.shuffleSymbols(symbolsToSync)
	}

	log.Printf("[KlineSyncer] 📋 %s 市场准备同步 %d 个交易对", marketType, len(symbolsToSync))

	// 临时保存原始symbols并设置新的symbols
	originalSymbols := s.config.Symbols
	s.config.Symbols = symbolsToSync                      // 只同步需要更新的交易对
	defer func() { s.config.Symbols = originalSymbols }() // 恢复原始配置

	totalUpdates := 0
	intervalErrors := 0

	// 改为串行处理不同时间间隔，避免工作池竞争
	log.Printf("[KlineSyncer] 🚀 串行启动 %s 市场 %d 个时间间隔的同步", marketType, len(s.config.KlineIntervals))

	// 串行处理每个时间间隔
	for _, interval := range s.config.KlineIntervals {
		startTime := time.Now()
		log.Printf("[KlineSyncer] 📊 Processing %s market interval: %s", marketType, interval)

		updates, err := s.syncKlinesForMarketInterval(ctx, symbolsToSync, interval, marketType)
		duration := time.Since(startTime)

		if err != nil {
			log.Printf("[KlineSyncer] ❌ Failed to sync %s %s klines after %v: %v",
				marketType, interval, duration, err)
			intervalErrors++
		} else {
			log.Printf("[KlineSyncer] ✅ Completed %s %s interval sync: %d updates in %v",
				marketType, interval, updates, duration)
			totalUpdates += updates
		}

		// 短暂暂停，避免API压力过大
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[KlineSyncer] 📊 %s market sync completed: %d total updates, %d interval errors",
		marketType, totalUpdates, intervalErrors)

	return totalUpdates, intervalErrors
}

// getSymbolsNeedingKlineSyncByMarket 按市场获取需要同步K线的交易对
func (s *KlineSyncer) getSymbolsNeedingKlineSyncByMarket(allSymbols []string, marketType string) ([]string, error) {
	if len(allSymbols) == 0 {
		return allSymbols, nil
	}

	// 重置统计计数器
	s.stats.mu.Lock()
	s.stats.noDataSymbols = 0
	s.stats.outdatedSymbols = 0
	s.stats.mu.Unlock()

	// 设置K线数据过期时间（例如1小时）
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
			needsSync := s.checkSymbolNeedsKlineSyncByMarket(sym, marketType, cutoffTime)
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

	// 输出详细的统计信息
	s.stats.mu.RLock()
	noDataCount := s.stats.noDataSymbols
	outdatedCount := s.stats.outdatedSymbols
	s.stats.mu.RUnlock()

	log.Printf("[KlineSyncer] 🔄 %s market incremental sync: %d/%d symbols need kline updating (无数据:%d, 数据过期:%d)",
		marketType, len(symbolsToSync), len(allSymbols), noDataCount, outdatedCount)

	return symbolsToSync, nil
}

// checkSymbolNeedsKlineSyncByMarket 检查单个交易对在指定市场是否需要K线同步
func (s *KlineSyncer) checkSymbolNeedsKlineSyncByMarket(symbol, marketType string, cutoffTime time.Time) bool {
	// 检查每个配置的时间间隔是否需要同步
	for _, interval := range s.config.KlineIntervals {
		if s.checkSymbolIntervalNeedsKlineSyncByMarket(symbol, marketType, interval, cutoffTime) {
			// 只要有任何一个时间间隔需要同步，就返回true
			return true
		}
	}

	return false
}

// checkSymbolIntervalNeedsKlineSyncByMarket 检查单个交易对的特定时间间隔在指定市场是否需要K线同步
func (s *KlineSyncer) checkSymbolIntervalNeedsKlineSyncByMarket(symbol, marketType, interval string, cutoffTime time.Time) bool {
	var result struct {
		LastKlineTime time.Time `json:"last_kline_time"`
		RecordCount   int       `json:"record_count"`
	}

	// 查询该交易对该时间间隔在指定市场的最新K线时间
	// 扩大时间窗口，确保有足够的历史数据
	checkTime := cutoffTime.Add(-24 * time.Hour) // 检查最近24小时的数据
	query := `
		SELECT MAX(open_time) as last_kline_time, COUNT(*) as record_count
		FROM market_klines
		WHERE symbol = ? AND kind = ? AND ` + "`interval`" + ` = ? AND open_time >= ?
	`

	err := s.db.Raw(query, symbol, marketType, interval, checkTime).Scan(&result).Error
	if err != nil {
		// 查询失败，假设需要同步
		log.Printf("[KlineSyncer] 查询 %s %s %s 失败: %v", symbol, marketType, interval, err)
		return true
	}

	// 如果没有记录，需要同步
	if result.LastKlineTime.IsZero() {
		s.stats.mu.Lock()
		s.stats.noDataSymbols++
		s.stats.mu.Unlock()
		return true
	}

	// 如果记录数太少（少于最近24小时应有的记录数），需要同步
	// 对于1小时K线，24小时应该有至少24条记录
	// 对于1分钟K线，24小时应该有至少1440条记录
	expectedMinRecords := 10 // 保守的最小记录数
	switch interval {
	case "1m":
		expectedMinRecords = 100 // 1分钟K线至少100条记录
	case "5m":
		expectedMinRecords = 50 // 5分钟K线至少50条记录
	case "15m":
		expectedMinRecords = 30 // 15分钟K线至少30条记录
	case "1h", "4h":
		expectedMinRecords = 10 // 小时线至少10条记录
	case "1d":
		expectedMinRecords = 5 // 日线至少5条记录
	}

	if result.RecordCount < expectedMinRecords {
		s.stats.mu.Lock()
		s.stats.noDataSymbols++
		s.stats.mu.Unlock()
		log.Printf("[KlineSyncer] %s %s %s 记录数不足 (%d < %d), 需要同步",
			symbol, marketType, interval, result.RecordCount, expectedMinRecords)
		return true
	}

	// 如果最新K线时间太旧（超过1小时），需要同步
	if result.LastKlineTime.Before(cutoffTime) {
		s.stats.mu.Lock()
		s.stats.outdatedSymbols++
		s.stats.mu.Unlock()
		log.Printf("[KlineSyncer] %s %s %s 数据过旧 (最新: %v, 截止: %v), 需要同步",
			symbol, marketType, interval, result.LastKlineTime, cutoffTime)
		return true
	}

	// 数据看起来是完整的，不需要同步
	return false
}

// syncKlinesForMarketInterval 同步指定市场和间隔的K线数据
// SymbolSyncResult 单个交易对同步结果
type SymbolSyncResult struct {
	Symbol string
	Count  int
	Error  error
}

// syncKlinesForMarketInterval 并发同步指定市场和时间间隔的K线数据
func (s *KlineSyncer) syncKlinesForMarketInterval(ctx context.Context, symbols []string, interval, marketType string) (int, error) {
	if len(symbols) == 0 {
		return 0, nil
	}

	log.Printf("[KlineSyncer] 📊 Starting sync for %s market interval: %s (%d symbols)", marketType, interval, len(symbols))

	symbolCount := len(symbols)
	startTime := time.Now()

	// 配置并发参数 - 智能调整以避免死锁
	maxConcurrency := 3 // 控制并发数量，避免API过载 (K线API限制为5/秒，这里保守设置为3)
	if s.config != nil && s.config.MaxConcurrentSymbols > 0 {
		maxConcurrency = s.config.MaxConcurrentSymbols
	}

	// 最终解决方案：完全串行处理，彻底消除死锁风险
	if symbolCount > 50 {
		// 大量交易对时，使用完全串行处理
		maxConcurrency = 1
		log.Printf("[KlineSyncer] 📊 大量交易对(%d)，使用完全串行处理(并发度%d)以彻底消除死锁", symbolCount, maxConcurrency)
	} else if symbolCount > 10 {
		// 中等数量时，使用低并发度
		maxConcurrency = 2
		log.Printf("[KlineSyncer] 📊 中等交易对(%d)，设置低并发度%d", symbolCount, maxConcurrency)
	} else {
		// 小量交易对时，使用适中并发度
		maxConcurrency = min(3, symbolCount)
	}

	log.Printf("[KlineSyncer] 🚀 开始并发同步 %s 市场 %s 间隔: %d 交易对 (并发度:%d)",
		marketType, interval, symbolCount, maxConcurrency)

	// 记录开始时的goroutine数量，用于监控
	initialGoroutines := runtime.NumGoroutine()
	log.Printf("[KlineSyncer] 📊 开始时goroutine数量: %d", initialGoroutines)

	// 分批处理策略：根据并发度调整，串行时适当增大批次
	var batchSize int
	if maxConcurrency == 1 {
		// 完全串行时，可以使用稍大的批次以提高效率
		batchSize = 20
		log.Printf("[KlineSyncer] 📦 串行处理，使用较大批次策略，批次大小: %d", batchSize)
	} else {
		// 并发时使用保守的批次大小
		batchSize = maxConcurrency * 5
		log.Printf("[KlineSyncer] 📦 并发处理，使用保守批次策略，批次大小: %d", batchSize)
	}

	totalUpdates := 0
	totalErrors := 0

	// 如果交易对数量不大，直接处理
	if symbolCount <= batchSize {
		updates, errors := s.processSymbolBatch(ctx, symbols, interval, marketType, maxConcurrency)
		if errors > 0 {
			return updates, fmt.Errorf("batch processing failed with %d errors", errors)
		}
		return updates, nil
	}

	// 分批处理大量交易对
	totalBatches := int(math.Ceil(float64(symbolCount) / float64(batchSize)))
	log.Printf("[KlineSyncer] 📦 分批处理 %d 个交易对，共 %d 批次", symbolCount, totalBatches)

	for batchIndex := 0; batchIndex < totalBatches; batchIndex++ {
		startIdx := batchIndex * batchSize
		endIdx := int(math.Min(float64(startIdx+batchSize), float64(symbolCount)))
		batchSymbols := symbols[startIdx:endIdx]

		log.Printf("[KlineSyncer] 📦 处理批次 %d/%d: %d 交易对 (%d-%d)",
			batchIndex+1, totalBatches, len(batchSymbols), startIdx+1, endIdx)

		batchUpdates, batchErrors := s.processSymbolBatch(ctx, batchSymbols, interval, marketType, maxConcurrency)
		totalUpdates += batchUpdates
		totalErrors += batchErrors

		// 批次间暂停，避免数据库和API压力过大
		if batchIndex < totalBatches-1 {
			var baseDelay time.Duration
			if maxConcurrency == 1 {
				// 串行处理时，使用较短延迟以提高效率
				baseDelay = 100 * time.Millisecond
				log.Printf("[KlineSyncer] ⏱️ 串行批次间延迟 %v", baseDelay)
			} else {
				// 并发处理时，使用较长延迟确保稳定性
				baseDelay = 500 * time.Millisecond
				if symbolCount > 300 {
					baseDelay = 800 * time.Millisecond // 超大量交易对用更长延迟
				}
				log.Printf("[KlineSyncer] ⏱️ 并发批次间延迟 %v，避免数据库竞争", baseDelay)
			}
			time.Sleep(baseDelay)
		}
	}

	// 计算完成统计
	duration := time.Since(startTime)
	successRate := float64(symbolCount-totalErrors) / float64(symbolCount) * 100

	// 记录结束时的goroutine数量
	finalGoroutines := runtime.NumGoroutine()
	goroutineDiff := finalGoroutines - initialGoroutines

	log.Printf("[KlineSyncer] ✅ %s 市场 %s 间隔同步完成: %d 更新, %d 错误, %d 总计",
		marketType, interval, totalUpdates, totalErrors, symbolCount)
	log.Printf("[KlineSyncer] 📊 同步统计 - 成功率:%.1f%% | 用时:%v | 平均:%v/交易对 | 并发度:%d",
		successRate, duration.Round(time.Second),
		(duration / time.Duration(symbolCount)).Round(time.Millisecond), maxConcurrency)
	log.Printf("[KlineSyncer] 🔄 Goroutine统计 - 开始:%d, 结束:%d, 差异:%+d",
		initialGoroutines, finalGoroutines, goroutineDiff)

	if totalErrors > 0 {
		return totalUpdates, fmt.Errorf("completed with %d errors out of %d symbols", totalErrors, symbolCount)
	}

	return totalUpdates, nil
}

// reportConcurrentProgress 报告并发同步进度
func (s *KlineSyncer) reportConcurrentProgress(ctx context.Context, marketType, interval string, totalSymbols int, resultChan <-chan SymbolSyncResult, done <-chan bool) {
	ticker := time.NewTicker(10 * time.Second) // 每10秒报告一次进度
	defer ticker.Stop()

	startTime := time.Now()
	processed := 0
	updates := 0
	errors := 0

	for {
		select {
		case <-done:
			return
		case result := <-resultChan:
			processed++
			if result.Error != nil {
				errors++
			} else {
				updates += result.Count
			}
		case <-ticker.C:
			if processed > 0 {
				progress := float64(processed) / float64(totalSymbols) * 100
				elapsed := time.Since(startTime)
				estimatedTotal := time.Duration(float64(elapsed) / float64(processed) * float64(totalSymbols))
				remaining := estimatedTotal - elapsed

				log.Printf("[KlineSyncer] 📈 %s %s 并发进度: %d/%d (%.1f%%) | 已用时:%v | 预计剩余:%v | 更新:%d | 错误:%d",
					marketType, interval, processed, totalSymbols, progress,
					elapsed.Round(time.Second), remaining.Round(time.Second), updates, errors)
			}
		case <-ctx.Done():
			return
		}

		// 如果已处理完所有任务，退出
		if processed >= totalSymbols {
			break
		}
	}
}

type KlineSyncer struct {
	db     *gorm.DB
	server interface{} // 服务器实例，用于调用K线API
	cfg    *config.Config
	config *DataSyncConfig

	// 无效符号缓存，避免重复请求无效的交易对
	invalidSymbols struct {
		mu      sync.RWMutex
		symbols map[string]bool // symbol_kind -> true
	}

	// Redis缓存，用于跨服务共享无效符号
	redisCache *RedisInvalidSymbolCache

	// 简化的统计信息
	stats struct {
		mu                sync.RWMutex
		totalSyncs        int64
		successfulSyncs   int64
		failedSyncs       int64
		lastSyncTime      time.Time
		totalKlineUpdates int64

		// 增量同步统计
		noDataSymbols   int64
		outdatedSymbols int64

		// API调用统计
		totalAPICalls      int64
		successfulAPICalls int64
		totalAPILatency    time.Duration
		lastAPILatency     time.Duration
	}
}

func NewKlineSyncer(db *gorm.DB, server interface{}, cfg *config.Config, config *DataSyncConfig, redisCache *RedisInvalidSymbolCache) *KlineSyncer {
	syncer := &KlineSyncer{
		db:     db,
		server: server,
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

	log.Printf("[KlineSyncer] 初始化简化版K线同步器")

	return syncer
}

// SymbolPriority 交易对优先级

func (s *KlineSyncer) Name() string {
	return "kline"
}

// getSymbolsNeedingKlineSync 增量同步：获取需要同步K线的交易对
// 超优化版本：并发查询，大幅提升检查速度
func (s *KlineSyncer) getSymbolsNeedingKlineSync(allSymbols []string) ([]string, error) {
	if len(allSymbols) == 0 {
		return allSymbols, nil
	}

	// 重置统计计数器
	s.stats.mu.Lock()
	s.stats.noDataSymbols = 0
	s.stats.outdatedSymbols = 0
	s.stats.mu.Unlock()

	// 设置K线数据过期时间（例如1小时）
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
			needsSync := s.checkSymbolNeedsKlineSync(sym, cutoffTime)
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

	// 输出详细的统计信息
	s.stats.mu.RLock()
	noDataCount := s.stats.noDataSymbols
	outdatedCount := s.stats.outdatedSymbols
	s.stats.mu.RUnlock()

	log.Printf("[KlineSyncer] 🔄 Incremental sync: %d/%d symbols need kline updating (无数据:%d, 数据过期:%d)",
		len(symbolsToSync), len(allSymbols), noDataCount, outdatedCount)

	return symbolsToSync, nil
}

// checkSymbolNeedsKlineSync 检查单个交易对是否需要K线同步
func (s *KlineSyncer) checkSymbolNeedsKlineSync(symbol string, cutoffTime time.Time) bool {
	// 检查每个配置的时间间隔是否需要同步
	for _, interval := range s.config.KlineIntervals {
		if s.checkSymbolIntervalNeedsKlineSync(symbol, interval, cutoffTime) {
			// 只要有任何一个时间间隔需要同步，就返回true
			return true
		}
	}

	return false
}

// checkSymbolIntervalNeedsKlineSync 检查单个交易对的特定时间间隔是否需要K线同步
func (s *KlineSyncer) checkSymbolIntervalNeedsKlineSync(symbol, interval string, cutoffTime time.Time) bool {
	var result struct {
		LastKlineTime time.Time `json:"last_kline_time"`
		RecordCount   int       `json:"record_count"`
	}

	// 查询该交易对该时间间隔的最新K线时间
	query := `
		SELECT MAX(open_time) as last_kline_time, COUNT(*) as record_count
		FROM market_klines
		WHERE symbol = ? AND ` + "`interval`" + ` = ? AND open_time >= ?
	`

	err := s.db.Raw(query, symbol, interval, cutoffTime).Scan(&result).Error
	if err != nil {
		// 查询失败，假设需要同步
		log.Printf("[KlineSyncer] 查询 %s %s 失败: %v", symbol, interval, err)
		return true
	}

	// 如果没有记录或记录数太少，需要同步
	if result.LastKlineTime.IsZero() || result.RecordCount < 5 {
		s.stats.mu.Lock()
		s.stats.noDataSymbols++
		s.stats.mu.Unlock()
		return true
	}

	// 如果最新K线时间太旧，需要同步
	if result.LastKlineTime.Before(cutoffTime) {
		s.stats.mu.Lock()
		s.stats.outdatedSymbols++
		s.stats.mu.Unlock()
		return true
	}

	return false
}

// containsString 检查字符串切片是否包含指定字符串
func (s *KlineSyncer) containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isSymbolInvalid 检查交易对是否为无效符号
func (s *KlineSyncer) isSymbolInvalid(symbol, kind string) bool {
	// 首先检查Redis缓存（跨服务共享）
	if s.redisCache != nil && s.redisCache.IsInvalid(symbol, kind) {
		return true
	}

	// 然后检查本地内存缓存
	s.invalidSymbols.mu.RLock()
	defer s.invalidSymbols.mu.RUnlock()
	key := symbol + "_" + kind
	return s.invalidSymbols.symbols[key]
}

// markSymbolInvalid 将交易对标记为无效符号
func (s *KlineSyncer) markSymbolInvalid(symbol, kind string) {
	// 写入本地内存缓存
	s.invalidSymbols.mu.Lock()
	key := symbol + "_" + kind
	s.invalidSymbols.symbols[key] = true
	s.invalidSymbols.mu.Unlock()

	// 写入Redis缓存（跨服务共享）
	if s.redisCache != nil {
		if err := s.redisCache.MarkInvalid(symbol, kind); err != nil {
			log.Printf("[KlineSyncer] ⚠️ Failed to mark invalid in Redis: %v", err)
		}
	}

	log.Printf("[KlineSyncer] 🛑 Marked %s %s as invalid symbol", symbol, kind)
}

func (s *KlineSyncer) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[KlineSyncer] Started with interval: %v", interval)
	log.Printf("[KlineSyncer] Will sync intervals: %v", s.config.KlineIntervals)

	for {
		select {
		case <-ctx.Done():
			log.Printf("[KlineSyncer] Stopped")
			return
		case <-ticker.C:
			log.Printf("[KlineSyncer] 📈 Starting scheduled kline sync...")
			startTime := time.Now()

			if err := s.Sync(ctx); err != nil {
				log.Printf("[KlineSyncer] ❌ Kline sync failed: %v", err)
			} else {
				duration := time.Since(startTime)
				log.Printf("[KlineSyncer] ✅ Kline sync completed in %v", duration)
			}
		}
	}
}

func (s *KlineSyncer) Stop() {
	log.Printf("[KlineSyncer] Stop signal received")
}

func (s *KlineSyncer) Sync(ctx context.Context) error {
	s.stats.mu.Lock()
	syncStartTime := time.Now()
	s.stats.totalSyncs++
	s.stats.lastSyncTime = syncStartTime
	s.stats.mu.Unlock()

	log.Printf("[KlineSyncer] 🚀 开始K线数据同步 (第 %d 次)", s.stats.totalSyncs)

	// 获取现货和期货交易对配置
	log.Printf("[KlineSyncer] 📋 正在构建同步配置...")
	syncerConfig := s.buildKlineSyncerConfig()
	log.Printf("[KlineSyncer] ✅ 配置构建完成 - 现货:%d 期货:%d",
		len(syncerConfig.SpotSymbols), len(syncerConfig.FuturesSymbols))

	totalUpdates := 0
	totalErrors := 0

	// 同步现货市场
	if len(syncerConfig.SpotSymbols) > 0 {
		log.Printf("[KlineSyncer] 📈 Starting spot market sync for %d symbols", len(syncerConfig.SpotSymbols))
		spotUpdates, spotErrors := s.syncMarketKlines(ctx, syncerConfig.SpotSymbols, "spot")
		totalUpdates += spotUpdates
		totalErrors += spotErrors
	} else {
		log.Printf("[KlineSyncer] ⚠️ No spot symbols to sync")
	}

	// 同步期货市场
	if len(syncerConfig.FuturesSymbols) > 0 {
		log.Printf("[KlineSyncer] 📈 Starting futures market sync for %d symbols", len(syncerConfig.FuturesSymbols))
		futuresUpdates, futuresErrors := s.syncMarketKlines(ctx, syncerConfig.FuturesSymbols, "futures")
		totalUpdates += futuresUpdates
		totalErrors += futuresErrors
	} else {
		log.Printf("[KlineSyncer] ⚠️ No futures symbols to sync")
	}

	totalDuration := time.Since(syncStartTime)

	s.stats.mu.Lock()
	if totalErrors == 0 {
		s.stats.successfulSyncs++
	}
	s.stats.totalKlineUpdates += int64(totalUpdates)
	s.stats.mu.Unlock()

	// 生成详细的同步报告
	log.Printf("[KlineSyncer] ✅ K线同步完成")
	log.Printf("[KlineSyncer] 📊 总耗时: %v", totalDuration.Round(time.Second))
	log.Printf("[KlineSyncer] 📈 数据更新: %d 条", totalUpdates)
	log.Printf("[KlineSyncer] 📋 市场覆盖: 现货(%d), 期货(%d)",
		len(syncerConfig.SpotSymbols), len(syncerConfig.FuturesSymbols))

	// 计算性能指标
	if totalDuration > 0 {
		updateRate := float64(totalUpdates) / totalDuration.Seconds()
		log.Printf("[KlineSyncer] ⚡ 同步性能: %.1f 条/秒", updateRate)
	}

	if totalErrors > 0 {
		log.Printf("[KlineSyncer] ⚠️ 完成但有 %d 个市场出现错误 - 请检查上述日志", totalErrors)
		return fmt.Errorf("completed with %d market errors", totalErrors)
	}

	log.Printf("[KlineSyncer] 🎉 本次同步完全成功")
	return nil
}

func (s *KlineSyncer) syncKlinesForInterval(ctx context.Context, interval string) (int, error) {
	log.Printf("[KlineSyncer] 📊 开始串行同步时间间隔: %s", interval)

	totalUpdates := 0
	totalErrors := 0
	symbolCount := len(s.config.Symbols)

	log.Printf("[KlineSyncer] 串行处理 %d 个交易对 (%d 个市场)", symbolCount, symbolCount*2)

	// 串行处理每个交易对
	for i, symbol := range s.config.Symbols {
		// 检查上下文是否已取消
		if ctx.Err() != nil {
			log.Printf("[KlineSyncer] ⚠️ 上下文已取消，停止同步: %v", ctx.Err())
			break
		}

		log.Printf("[KlineSyncer] 处理交易对 %d/%d: %s", i+1, symbolCount, symbol)

		// 同步现货数据
		spotResult := s.syncSymbolKlines(ctx, symbol, "spot", interval)
		if spotResult.Error != nil {
			log.Printf("[KlineSyncer] ❌ 现货同步失败 %s %s %s: %v",
				spotResult.Symbol, spotResult.Kind, interval, spotResult.Error)
			totalErrors++
		} else {
			if totalUpdates < 6 { // 只显示前几个成功的详细信息
				log.Printf("[KlineSyncer] ✅ 现货同步成功: %s %s %s (%d 条数据)",
					spotResult.Symbol, spotResult.Kind, interval, spotResult.Count)
			}
			totalUpdates += spotResult.Count
		}

		// 检查上下文是否已取消
		if ctx.Err() != nil {
			log.Printf("[KlineSyncer] ⚠️ 上下文已取消，停止同步: %v", ctx.Err())
			break
		}

		// 同步期货数据
		futuresResult := s.syncSymbolKlines(ctx, symbol, "futures", interval)
		if futuresResult.Error != nil {
			log.Printf("[KlineSyncer] ❌ 期货同步失败 %s %s %s: %v",
				futuresResult.Symbol, futuresResult.Kind, interval, futuresResult.Error)
			totalErrors++
		} else {
			if totalUpdates < 6 { // 只显示前几个成功的详细信息
				log.Printf("[KlineSyncer] ✅ 期货同步成功: %s %s %s (%d 条数据)",
					futuresResult.Symbol, futuresResult.Kind, interval, futuresResult.Count)
			}
			totalUpdates += futuresResult.Count
		}
	}

	log.Printf("[KlineSyncer] 📈 时间间隔 %s 同步完成: %d 总更新, %d 错误, 处理 %d 个交易对",
		interval, totalUpdates, totalErrors, symbolCount)

	return totalUpdates, nil
}

// SymbolResult 单个交易对同步结果
type SymbolResult struct {
	Symbol string
	Kind   string
	Count  int
	Error  error
}

// syncSymbolKlines 同步单个交易对的K线数据
func (s *KlineSyncer) syncSymbolKlines(ctx context.Context, symbol, kind, interval string) SymbolResult {
	result := SymbolResult{
		Symbol: symbol,
		Kind:   kind,
	}

	// 注意：无效符号已在配置构建阶段过滤，这里不再需要检查

	// 重试机制 - 指数退避策略
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		// 获取K线数据
		klines, err := s.fetchLatestKlines(ctx, symbol, kind, interval, 100)

		if err != nil {
			// 分析错误类型并进行相应处理
			errorType := s.analyzeKlineError(err)

			switch errorType {
			case "invalid_symbol":
				// 无效符号，标记并跳过
				s.markSymbolInvalid(symbol, kind)
				result.Error = fmt.Errorf("invalid symbol: %s %s", symbol, kind)
				return result

			case "rate_limit":
				// API限流，使用智能退避策略
				if attempt < maxRetries {
					backoffDelay := time.Duration(attempt) * 5 * time.Second // 简单的固定延迟
					if backoffDelay > 30*time.Second {
						backoffDelay = 30 * time.Second
					}
					log.Printf("[KlineSyncer] API rate limited, attempt %d/%d failed for %s %s %s: %v, backing off for %v...",
						attempt, maxRetries, symbol, kind, interval, err, backoffDelay)

					// 可取消的sleep
					select {
					case <-ctx.Done():
						log.Printf("[KlineSyncer] Context cancelled during backoff for %s %s %s", symbol, kind, interval)
						result.Error = fmt.Errorf("context cancelled during rate limit backoff: %w", ctx.Err())
						return result
					case <-time.After(backoffDelay):
						// 继续重试
					}
					continue
				}

			case "network_error":
				// 网络错误，使用较短的重试间隔
				if attempt < maxRetries {
					backoffDelay := time.Duration(attempt) * 2 * time.Second // 简单的固定延迟
					if backoffDelay > 10*time.Second {
						backoffDelay = 10 * time.Second
					}
					log.Printf("[KlineSyncer] Network error, attempt %d/%d failed for %s %s %s: %v, retrying in %v...",
						attempt, maxRetries, symbol, kind, interval, err, backoffDelay)

					// 可取消的sleep
					select {
					case <-ctx.Done():
						log.Printf("[KlineSyncer] Context cancelled during backoff for %s %s %s", symbol, kind, interval)
						result.Error = fmt.Errorf("context cancelled during network error backoff: %w", ctx.Err())
						return result
					case <-time.After(backoffDelay):
						// 继续重试
					}
					continue
				}

			case "server_error":
				// 服务器错误，使用中等延迟
				if attempt < maxRetries {
					backoffDelay := time.Duration(attempt) * 3 * time.Second // 简单的固定延迟
					if backoffDelay > 15*time.Second {
						backoffDelay = 15 * time.Second
					}
					log.Printf("[KlineSyncer] Server error, attempt %d/%d failed for %s %s %s: %v, retrying in %v...",
						attempt, maxRetries, symbol, kind, interval, err, backoffDelay)

					// 可取消的sleep
					select {
					case <-ctx.Done():
						log.Printf("[KlineSyncer] Context cancelled during backoff for %s %s %s", symbol, kind, interval)
						result.Error = fmt.Errorf("context cancelled during server error backoff: %w", ctx.Err())
						return result
					case <-time.After(backoffDelay):
						// 继续重试
					}
					continue
				}

			default:
				// 未知错误，使用保守的重试策略
				if attempt < maxRetries {
					backoffDelay := time.Duration(attempt) * 1 * time.Second // 简单的固定延迟
					if backoffDelay > 5*time.Second {
						backoffDelay = 5 * time.Second
					}
					log.Printf("[KlineSyncer] Unknown error, attempt %d/%d failed for %s %s %s: %v, retrying in %v...",
						attempt, maxRetries, symbol, kind, interval, err, backoffDelay)

					// 可取消的sleep
					select {
					case <-ctx.Done():
						log.Printf("[KlineSyncer] Context cancelled during backoff for %s %s %s", symbol, kind, interval)
						result.Error = fmt.Errorf("context cancelled during unknown error backoff: %w", ctx.Err())
						return result
					case <-time.After(backoffDelay):
						// 继续重试
					}
					continue
				}
			}

			result.Error = fmt.Errorf("fetch failed after %d attempts: %w", maxRetries, err)
			return result
		}

		if len(klines) == 0 {
			log.Printf("[KlineSyncer] No kline data available for %s %s %s", symbol, kind, interval)
			result.Count = 0
			return result
		}

		// 保存K线数据（使用数据库并发控制）
		if err := s.saveKlinesWithConcurrencyControl(ctx, symbol, kind, interval, klines); err != nil {
			if attempt < maxRetries {
				log.Printf("[KlineSyncer] Save attempt %d/%d failed for %s %s %s: %v, retrying...",
					attempt, maxRetries, symbol, kind, interval, err)

				// 可取消的sleep
				retryDelay := time.Duration(attempt) * 500 * time.Millisecond
				select {
				case <-ctx.Done():
					log.Printf("[KlineSyncer] Context cancelled during save retry for %s %s %s", symbol, kind, interval)
					result.Error = fmt.Errorf("context cancelled during save retry: %w", ctx.Err())
					return result
				case <-time.After(retryDelay):
					// 继续重试
				}
				continue
			}
			result.Error = fmt.Errorf("save failed after %d attempts: %w", maxRetries, err)
			return result
		}

		// 成功

		// 成功
		result.Count = len(klines)
		if attempt > 1 {
			log.Printf("[KlineSyncer] Succeeded on attempt %d/%d for %s %s %s",
				attempt, maxRetries, symbol, kind, interval)
		}
		return result
	}

	// 不应该到达这里
	result.Error = fmt.Errorf("unexpected error in syncSymbolKlines")
	return result
}

func (s *KlineSyncer) fetchLatestKlines(ctx context.Context, symbol, kind, interval string, limit int) ([]interface{}, error) {
	// 使用带有统计功能的Binance API客户端
	apiClient := NewBinanceAPIClientWithStats(func(success bool, latency time.Duration, apiKind string) {
		// 记录API调用统计信息
		s.stats.mu.Lock()
		s.stats.totalAPICalls++
		if success {
			s.stats.successfulAPICalls++
			s.stats.totalAPILatency += latency
			s.stats.lastAPILatency = latency
		}
		s.stats.mu.Unlock()
	})

	klines, err := apiClient.FetchKlines(ctx, symbol, kind, interval, limit)
	if err != nil {
		log.Printf("[KlineSyncer] ❌ Failed to fetch klines from API: %v", err)
		return nil, fmt.Errorf("failed to fetch klines: %w", err)
	}

	if len(klines) == 0 {
		log.Printf("[KlineSyncer] ⚠️ No kline data received for %s %s %s", symbol, kind, interval)
		return []interface{}{}, nil
	}

	// 转换为interface{}数组返回
	result := make([]interface{}, len(klines))
	for i, kline := range klines {
		result[i] = kline
	}

	return result, nil
}

// saveKlinesWithConcurrencyControl 使用并发控制保存K线数据
func (s *KlineSyncer) saveKlinesWithConcurrencyControl(ctx context.Context, symbol, kind, interval string, klines []interface{}) error {
	if len(klines) == 0 {
		log.Printf("[KlineSyncer] ℹ️ No klines to save for %s %s %s", symbol, kind, interval)
		return nil
	}

	//log.Printf("[KlineSyncer] 💾 Processing %d kline records for %s %s %s", len(klines), symbol, kind, interval)

	// 参数验证
	if symbol == "" || kind == "" || interval == "" {
		return fmt.Errorf("invalid parameters: symbol=%s, kind=%s, interval=%s", symbol, kind, interval)
	}

	// 转换为MarketKline格式
	marketKlines := make([]pdb.MarketKline, 0, len(klines))
	conversionErrors := 0

	for i, klineInterface := range klines {
		klineData, ok := klineInterface.(analysis.KlineDataAPI)
		if !ok {
			log.Printf("[KlineSyncer] ⚠️ Invalid kline data type at index %d: %T", i, klineInterface)
			conversionErrors++
			continue
		}

		// 数据验证
		if err := s.validateKlineData(&klineData); err != nil {
			log.Printf("[KlineSyncer] ⚠️ Invalid kline data at index %d: %v", i, err)
			conversionErrors++
			continue
		}

		// 转换时间戳 (毫秒转秒)
		openTime := time.Unix(klineData.OpenTime/1000, (klineData.OpenTime%1000)*1000000)

		// 验证时间戳合理性 - 对于K线数据，允许更宽松的时间范围
		now := time.Now()
		// 允许未来1小时（处理时钟偏差）和过去2年（处理历史数据）
		if openTime.After(now.Add(24*time.Hour)) || openTime.Before(now.AddDate(-2, 0, 0)) {
			log.Printf("[KlineSyncer] ⚠️ Invalid timestamp at index %d: %v (current: %v)", i, openTime, now)
			conversionErrors++
			continue
		}

		// OHLC价格关系验证已在validateKlineData中完成

		// 创建MarketKline记录
		marketKline := pdb.MarketKline{
			Symbol:     strings.ToUpper(symbol),
			Kind:       kind,
			Interval:   interval,
			OpenTime:   openTime,
			OpenPrice:  klineData.Open,
			HighPrice:  klineData.High,
			LowPrice:   klineData.Low,
			ClosePrice: klineData.Close,
			Volume:     klineData.Volume,
			// 可选字段
			QuoteVolume:         nil,
			TradeCount:          nil,
			TakerBuyVolume:      nil,
			TakerBuyQuoteVolume: nil,
		}

		marketKlines = append(marketKlines, marketKline)
	}

	if len(marketKlines) == 0 {
		log.Printf("[KlineSyncer] ❌ No valid kline records to save after validation")
		return fmt.Errorf("no valid kline records to save")
	}

	if conversionErrors > 0 {
		log.Printf("[KlineSyncer] ⚠️ Skipped %d invalid records during conversion", conversionErrors)
	}

	// 直接保存K线数据到数据库
	startTime := time.Now()
	if err := pdb.SaveMarketKlines(s.db, marketKlines); err != nil {
		log.Printf("[KlineSyncer] ❌ Failed to save klines to database: %v", err)
		return fmt.Errorf("failed to save klines to database: %w", err)
	}

	duration := time.Since(startTime)
	log.Printf("[KlineSyncer] ✅ Successfully saved %d kline records to database in %v (%.1f records/sec) (%s %s %s)",
		len(marketKlines), duration.Round(time.Millisecond), float64(len(marketKlines))/duration.Seconds(),
		symbol, kind, interval)

	return nil
}

func (s *KlineSyncer) saveKlines(symbol, kind, interval string, klines []interface{}) error {
	if len(klines) == 0 {
		log.Printf("[KlineSyncer] ℹ️ No klines to save for %s %s %s", symbol, kind, interval)
		return nil
	}

	//log.Printf("[KlineSyncer] 💾 Processing %d kline records for %s %s %s", len(klines), symbol, kind, interval)

	// 参数验证
	if symbol == "" || kind == "" || interval == "" {
		return fmt.Errorf("invalid parameters: symbol=%s, kind=%s, interval=%s", symbol, kind, interval)
	}

	// 转换为MarketKline格式
	marketKlines := make([]pdb.MarketKline, 0, len(klines))
	conversionErrors := 0

	for i, klineInterface := range klines {
		klineData, ok := klineInterface.(analysis.KlineDataAPI)
		if !ok {
			log.Printf("[KlineSyncer] ⚠️ Invalid kline data type at index %d: %T", i, klineInterface)
			conversionErrors++
			continue
		}

		// 数据验证
		if err := s.validateKlineData(&klineData); err != nil {
			log.Printf("[KlineSyncer] ⚠️ Invalid kline data at index %d: %v", i, err)
			conversionErrors++
			continue
		}

		// 转换时间戳 (毫秒转秒)
		openTime := time.Unix(klineData.OpenTime/1000, (klineData.OpenTime%1000)*1000000)

		// 验证时间戳合理性 - 对于K线数据，允许更宽松的时间范围
		now := time.Now()
		// 允许未来1小时（处理时钟偏差）和过去2年（处理历史数据）
		if openTime.After(now.Add(24*time.Hour)) || openTime.Before(now.AddDate(-2, 0, 0)) {
			log.Printf("[KlineSyncer] ⚠️ Invalid timestamp at index %d: %v (current: %v)", i, openTime, now)
			conversionErrors++
			continue
		}

		// OHLC价格关系验证已在validateKlineData中完成

		// 创建MarketKline记录
		marketKline := pdb.MarketKline{
			Symbol:     strings.ToUpper(symbol),
			Kind:       kind,
			Interval:   interval,
			OpenTime:   openTime,
			OpenPrice:  klineData.Open,
			HighPrice:  klineData.High,
			LowPrice:   klineData.Low,
			ClosePrice: klineData.Close,
			Volume:     klineData.Volume,
			// 可选字段
			QuoteVolume:         nil,
			TradeCount:          nil,
			TakerBuyVolume:      nil,
			TakerBuyQuoteVolume: nil,
		}

		marketKlines = append(marketKlines, marketKline)
	}

	if len(marketKlines) == 0 {
		log.Printf("[KlineSyncer] ❌ No valid kline records to save after validation")
		return fmt.Errorf("no valid kline records to save")
	}

	if conversionErrors > 0 {
		log.Printf("[KlineSyncer] ⚠️ Skipped %d invalid records during conversion", conversionErrors)
	}

	//startTime := time.Now()
	if err := pdb.SaveMarketKlines(s.db, marketKlines); err != nil {
		log.Printf("[KlineSyncer] ❌ Failed to save klines to database: %v", err)
		return fmt.Errorf("failed to save klines to database: %w", err)
	}

	//duration := time.Since(startTime)
	//log.Printf("[KlineSyncer] ✅ Successfully saved %d kline records to database in %v (%.1f records/sec)",
	//	len(marketKlines), duration, float64(len(marketKlines))/duration.Seconds())

	return nil
}

// validateKlineData 验证K线数据的有效性
func (s *KlineSyncer) validateKlineData(kline *analysis.KlineDataAPI) error {
	if kline == nil {
		return fmt.Errorf("kline data is nil")
	}

	// 检查必要字段
	if kline.OpenTime <= 0 {
		return fmt.Errorf("invalid openTime: %d", kline.OpenTime)
	}

	// 验证价格数据
	prices := []string{kline.Open, kline.High, kline.Low, kline.Close}
	for i, price := range prices {
		if price == "" {
			fieldNames := []string{"open", "high", "low", "close"}
			return fmt.Errorf("empty %s price", fieldNames[i])
		}

		// 验证价格格式
		if _, err := strconv.ParseFloat(price, 64); err != nil {
			fieldNames := []string{"open", "high", "low", "close"}
			return fmt.Errorf("invalid %s price format: %s", fieldNames[i], price)
		}
	}

	// 验证成交量
	if kline.Volume != "" {
		if _, err := strconv.ParseFloat(kline.Volume, 64); err != nil {
			return fmt.Errorf("invalid volume format: %s", kline.Volume)
		}
	}

	// 验证价格逻辑关系 (高 >= 低, 最高价 >= 开盘价等)
	high, _ := strconv.ParseFloat(kline.High, 64)
	low, _ := strconv.ParseFloat(kline.Low, 64)
	open, _ := strconv.ParseFloat(kline.Open, 64)
	close, _ := strconv.ParseFloat(kline.Close, 64)

	if high < low {
		return fmt.Errorf("high price %.8f < low price %.8f", high, low)
	}

	if open < low || open > high {
		return fmt.Errorf("open price %.8f not within [low, high] range [%.8f, %.8f]", open, low, high)
	}

	if close < low || close > high {
		return fmt.Errorf("close price %.8f not within [low, high] range [%.8f, %.8f]", close, low, high)
	}

	return nil
}

func (s *KlineSyncer) GetStats() map[string]interface{} {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	var avgLatency time.Duration
	var successRate float64

	if s.stats.totalAPICalls > 0 {
		avgLatency = s.stats.totalAPILatency / time.Duration(s.stats.totalAPICalls)
		successRate = float64(s.stats.successfulAPICalls) / float64(s.stats.totalAPICalls) * 100
	}

	return map[string]interface{}{
		"total_syncs":      s.stats.totalSyncs,
		"successful_syncs": s.stats.successfulSyncs,
		"failed_syncs":     s.stats.failedSyncs,
		"last_sync_time":   s.stats.lastSyncTime,
		"total_updates":    s.stats.totalKlineUpdates,
		// 增量同步统计
		"no_data_symbols":  s.stats.noDataSymbols,
		"outdated_symbols": s.stats.outdatedSymbols,
		// API性能指标
		"api_calls_total":   s.stats.totalAPICalls,
		"api_calls_success": s.stats.successfulAPICalls,
		"api_success_rate":  fmt.Sprintf("%.1f%%", successRate),
		"api_avg_latency":   avgLatency.String(),
		"api_last_latency":  s.stats.lastAPILatency.String(),
	}
}

// analyzeKlineError 分析K线API错误的类型
func (s *KlineSyncer) analyzeKlineError(err error) string {
	if err == nil {
		return "none"
	}

	errStr := strings.ToLower(err.Error())

	// 无效符号错误
	if strings.Contains(errStr, "invalid symbol") || strings.Contains(errStr, "-1121") ||
		strings.Contains(errStr, "symbol not found") {
		return "invalid_symbol"
	}

	// API限流错误
	if strings.Contains(errStr, "way too many requests") || strings.Contains(errStr, "-1003") ||
		strings.Contains(errStr, "rate limit") || strings.Contains(errStr, "too many requests") {
		return "rate_limit"
	}

	// 网络相关错误
	if strings.Contains(errStr, "timeout") || strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "network") || strings.Contains(errStr, "dial tcp") ||
		strings.Contains(errStr, "no such host") {
		return "network_error"
	}

	// 服务器错误
	if strings.Contains(errStr, "internal server error") || strings.Contains(errStr, "-1000") ||
		strings.Contains(errStr, "service unavailable") || strings.Contains(errStr, "-1001") ||
		strings.Contains(errStr, "server error") {
		return "server_error"
	}

	// 参数错误
	if strings.Contains(errStr, "invalid parameter") || strings.Contains(errStr, "-1100") ||
		strings.Contains(errStr, "bad request") {
		return "parameter_error"
	}

	return "unknown"
}

// GetAPIStats 获取API统计信息
func (s *KlineSyncer) GetAPIStats() *server.APIStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	successRate := "0%"
	if s.stats.totalAPICalls > 0 {
		rate := float64(s.stats.successfulAPICalls) / float64(s.stats.totalAPICalls) * 100
		successRate = fmt.Sprintf("%.1f%%", rate)
	}

	avgLatency := ""
	if s.stats.totalAPICalls > 0 && s.stats.totalAPILatency > 0 {
		avg := s.stats.totalAPILatency / time.Duration(s.stats.totalAPICalls)
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
		TotalUpdates:    s.stats.totalKlineUpdates,
	}
}

// processSymbolBatch 处理一批交易对的K线同步
func (s *KlineSyncer) processSymbolBatch(ctx context.Context, symbols []string, interval, marketType string, maxConcurrency int) (int, int) {
	if len(symbols) == 0 {
		return 0, 0
	}

	symbolCount := len(symbols)

	// 创建结果通道和信号量 - 优化缓冲区大小避免阻塞
	resultChan := make(chan SymbolSyncResult, symbolCount*2) // 增加缓冲区大小
	semaphore := make(chan struct{}, maxConcurrency)
	var wg sync.WaitGroup

	// 启动进度报告goroutine
	progressDone := make(chan bool)
	go s.reportConcurrentProgress(ctx, marketType, interval, symbolCount, resultChan, progressDone)

	// 并发处理每个交易对
	for i, symbol := range symbols {
		wg.Add(1)
		go func(index int, sym string) {
			defer wg.Done()

			// 获取信号量（控制并发）
			select {
			case semaphore <- struct{}{}:
				defer func() { <-semaphore }()
			case <-ctx.Done():
				resultChan <- SymbolSyncResult{Symbol: sym, Count: 0, Error: ctx.Err()}
				return
			}

			symbolStartTime := time.Now()

			// 创建带超时的上下文
			symbolCtx, cancel := context.WithTimeout(ctx, 15*time.Minute)
			defer cancel()

			// 同步指定市场的K线数据
			result := s.syncSymbolKlines(symbolCtx, sym, marketType, interval)

			// 发送结果到通道
			resultChan <- SymbolSyncResult{
				Symbol: sym,
				Count:  result.Count,
				Error:  result.Error,
			}

			// 记录处理时间（仅前几个）
			if index < 3 {
				symbolDuration := time.Since(symbolStartTime)
				if result.Error == nil {
					log.Printf("[KlineSyncer] ✅ 同步成功 %s %s %s: %d 条数据 (%v)",
						sym, marketType, interval, result.Count, symbolDuration.Round(time.Millisecond))
				}
			}
		}(i, symbol)
	}

	// 等待所有goroutine完成
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// 收集结果
	totalUpdates := 0
	totalErrors := 0
	processed := 0

	for result := range resultChan {
		processed++

		if result.Error != nil {
			totalErrors++
			// 只在少量错误时详细输出
			if totalErrors <= 5 {
				log.Printf("[KlineSyncer] ❌ 同步失败 %s %s %s: %v",
					result.Symbol, marketType, interval, result.Error)
			}
		} else {
			totalUpdates += result.Count
		}

		// 检查是否完成所有处理
		if processed >= symbolCount {
			break
		}
	}

	// 停止进度报告
	progressDone <- true

	// 计算完成统计
	log.Printf("[KlineSyncer] ✅ 批次完成 %s 市场 %s 间隔: %d 更新, %d 错误, %d 总计",
		marketType, interval, totalUpdates, totalErrors, symbolCount)

	return totalUpdates, totalErrors
}

// shuffleSymbols 随机化交易对顺序，减少热点冲突
func (s *KlineSyncer) shuffleSymbols(symbols []string) []string {
	if len(symbols) <= 1 {
		return symbols
	}

	// 创建副本避免修改原切片
	shuffled := make([]string, len(symbols))
	copy(shuffled, symbols)

	// 使用Fisher-Yates洗牌算法
	for i := len(shuffled) - 1; i > 0; i-- {
		j := int(math.Floor(rand.Float64() * float64(i+1)))
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled
}
