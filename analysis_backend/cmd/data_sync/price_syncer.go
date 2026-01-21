package main

import (
	"context"
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

// ===== 价格同步器 =====

// PriceSyncerConfig 价格同步器配置
type PriceSyncerConfig struct {
	SpotSymbols    []string // 现货交易对
	FuturesSymbols []string // 期货交易对
}

// buildPriceSyncerConfig 构建价格同步器配置
func (s *PriceSyncer) buildPriceSyncerConfig() PriceSyncerConfig {
	config := PriceSyncerConfig{}

	// 优先从数据库获取各市场的有效交易对，避免使用包含无效符号的全局配置
	if spotSymbols, err := pdb.GetUSDTTradingPairsByMarket(s.db, "spot"); err == nil {
		// 过滤掉Redis缓存中标记为无效的符号
		config.SpotSymbols = s.filterOutInvalidSymbols(spotSymbols, "spot")
		log.Printf("[PriceSyncer] ✅ Loaded %d spot symbols from database (%d after filtering invalid)",
			len(spotSymbols), len(config.SpotSymbols))
	} else {
		log.Printf("[PriceSyncer] ⚠️ Failed to get spot symbols: %v", err)
		// 如果数据库查询失败，尝试从配置中获取
		if len(s.config.Symbols) > 0 {
			config.SpotSymbols = s.config.Symbols
			log.Printf("[PriceSyncer] 🔄 Using configured symbols as fallback for spot: %d symbols", len(config.SpotSymbols))
		}
	}

	if futuresSymbols, err := pdb.GetUSDTTradingPairsByMarket(s.db, "futures"); err == nil {
		// 过滤掉Redis缓存中标记为无效的符号
		config.FuturesSymbols = s.filterOutInvalidSymbols(futuresSymbols, "futures")
		log.Printf("[PriceSyncer] ✅ Loaded %d futures symbols from database (%d after filtering invalid)",
			len(futuresSymbols), len(config.FuturesSymbols))
	} else {
		log.Printf("[PriceSyncer] ⚠️ Failed to get futures symbols: %v", err)
		// 如果数据库查询失败，尝试从配置中获取
		if len(s.config.Symbols) > 0 {
			config.FuturesSymbols = s.config.Symbols
			log.Printf("[PriceSyncer] 🔄 Using configured symbols as fallback for futures: %d symbols", len(config.FuturesSymbols))
		}
	}

	return config
}

// filterOutInvalidSymbols 过滤掉Redis缓存中标记为无效的符号
func (s *PriceSyncer) filterOutInvalidSymbols(symbols []string, marketType string) []string {
	if len(symbols) == 0 {
		return symbols
	}

	var validSymbols []string
	for _, symbol := range symbols {
		if !s.isSymbolInvalid(symbol, marketType) {
			validSymbols = append(validSymbols, symbol)
		} else {
			//log.Printf("[PriceSyncer] 🗑️ Filtered out invalid symbol: %s %s", symbol, marketType)
		}
	}

	return validSymbols
}

// filterConfiguredSymbols 过滤出配置中存在的交易对
func (s *PriceSyncer) filterConfiguredSymbols(configured, available []string) []string {
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

type PriceSyncer struct {
	db     *gorm.DB
	cfg    *config.Config
	config *DataSyncConfig

	// WebSocket同步器引用，用于获取实时价格数据
	websocketSyncer *WebSocketSyncer

	// 无效符号缓存，避免重复请求无效的交易对
	invalidSymbols struct {
		mu      sync.RWMutex
		symbols map[string]bool // symbol_kind -> true
	}

	// Redis缓存，用于跨服务共享无效符号
	redisCache *RedisInvalidSymbolCache

	stats struct {
		mu                sync.RWMutex
		totalSyncs        int64
		successfulSyncs   int64
		failedSyncs       int64
		lastSyncTime      time.Time
		totalPriceUpdates int64
		websocketHits     int64 // 从WebSocket缓存命中的次数
		restAPICalls      int64 // REST API调用的次数
	}
}

func NewPriceSyncer(db *gorm.DB, cfg *config.Config, config *DataSyncConfig, redisCache *RedisInvalidSymbolCache) *PriceSyncer {
	return &PriceSyncer{
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

// SetWebSocketSyncer 设置WebSocket同步器引用
func (s *PriceSyncer) SetWebSocketSyncer(ws *WebSocketSyncer) {
	s.websocketSyncer = ws
}

func (s *PriceSyncer) Name() string {
	return "price"
}

// getSymbolsNeedingSync 增量同步：获取需要同步的交易对
// 只返回数据过期或不存在的交易对，避免重复同步
func (s *PriceSyncer) getSymbolsNeedingSync(allSymbols []string) ([]string, error) {
	if len(allSymbols) == 0 {
		return allSymbols, nil
	}

	// 设置数据过期时间（例如5分钟）
	maxDataAge := 5 * time.Minute
	cutoffTime := time.Now().Add(-maxDataAge)

	var symbolsToSync []string

	// 批量查询所有交易对的最新价格更新时间
	query := `
		SELECT symbol, MAX(last_updated) as last_update, kind as market_type
		FROM price_caches
		WHERE symbol IN ?
		GROUP BY symbol, kind
	`

	// 构建IN查询的参数
	args := make([]interface{}, len(allSymbols))
	for i, symbol := range allSymbols {
		args[i] = symbol
	}

	var results []struct {
		Symbol     string    `json:"symbol"`
		LastUpdate time.Time `json:"last_update"`
		MarketType string    `json:"market_type"`
	}

	err := s.db.Raw(query, args).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("查询交易对更新时间失败: %w", err)
	}

	// 创建已存在交易对的映射
	existingSymbols := make(map[string]time.Time)
	for _, result := range results {
		key := result.Symbol + "_" + result.MarketType
		existingSymbols[key] = result.LastUpdate
	}

	// 确定需要同步的交易对
	for _, symbol := range allSymbols {
		needsSpotSync := false
		needsFuturesSync := false

		// 检查现货数据
		spotKey := symbol + "_spot"
		if lastUpdate, exists := existingSymbols[spotKey]; !exists {
			// 数据不存在，需要同步
			needsSpotSync = true
		} else if lastUpdate.Before(cutoffTime) {
			// 数据过期，需要同步
			needsSpotSync = true
		}

		// 检查期货数据
		futuresKey := symbol + "_futures"
		if lastUpdate, exists := existingSymbols[futuresKey]; !exists {
			// 数据不存在，需要同步
			needsFuturesSync = true
		} else if lastUpdate.Before(cutoffTime) {
			// 数据过期，需要同步
			needsFuturesSync = true
		}

		// 如果任一市场需要同步，则加入同步列表
		if needsSpotSync || needsFuturesSync {
			symbolsToSync = append(symbolsToSync, symbol)
		}
	}

	// 如果所有数据都是最新的，返回空列表（表示无需同步）
	// 但是为了确保服务正常运行，至少同步几个核心交易对
	if len(symbolsToSync) == 0 && len(allSymbols) > 0 {
		coreSymbols := []string{"BTCUSDT", "ETHUSDT"}
		for _, coreSymbol := range coreSymbols {
			if containsString(allSymbols, coreSymbol) {
				symbolsToSync = append(symbolsToSync, coreSymbol)
			}
		}
	}

	log.Printf("[PriceSyncer] 🔄 Incremental sync: %d/%d symbols need updating",
		len(symbolsToSync), len(allSymbols))

	return symbolsToSync, nil
}

// containsString 检查字符串切片是否包含指定字符串
func containsString(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isSymbolInvalid 检查交易对是否为无效符号
func (s *PriceSyncer) isSymbolInvalid(symbol, kind string) bool {
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
func (s *PriceSyncer) markSymbolInvalid(symbol, kind string) {
	// 写入本地内存缓存
	s.invalidSymbols.mu.Lock()
	key := symbol + "_" + kind
	s.invalidSymbols.symbols[key] = true
	s.invalidSymbols.mu.Unlock()

	// 写入Redis缓存（跨服务共享）
	if s.redisCache != nil {
		if err := s.redisCache.MarkInvalid(symbol, kind); err != nil {
			log.Printf("[PriceSyncer] ⚠️ Failed to mark invalid in Redis: %v", err)
		}
	}

	log.Printf("[PriceSyncer] 🛑 Marked %s %s as invalid symbol", symbol, kind)
}

func (s *PriceSyncer) Start(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	log.Printf("[PriceSyncer] Started with interval: %v", interval)
	nextSync := time.Now().Add(interval)
	log.Printf("[PriceSyncer] Next sync scheduled at: %s", nextSync.Format("15:04:05"))

	for {
		select {
		case <-ctx.Done():
			log.Printf("[PriceSyncer] Stopped")
			return
		case <-ticker.C:
			log.Printf("[PriceSyncer] 🔄 Starting scheduled price sync...")
			startTime := time.Now()

			if err := s.Sync(ctx); err != nil {
				log.Printf("[PriceSyncer] ❌ Sync failed: %v", err)
				s.stats.mu.Lock()
				s.stats.failedSyncs++
				s.stats.mu.Unlock()
			} else {
				duration := time.Since(startTime)
				log.Printf("[PriceSyncer] ✅ Sync completed in %v", duration)

				s.stats.mu.Lock()
				s.stats.successfulSyncs++
				s.stats.mu.Unlock()
			}

			nextSync = time.Now().Add(interval)
			log.Printf("[PriceSyncer] Next sync at: %s", nextSync.Format("15:04:05"))
		}
	}
}

func (s *PriceSyncer) Stop() {
	log.Printf("[PriceSyncer] Stop signal received")
}

func (s *PriceSyncer) Sync(ctx context.Context) error {
	s.stats.mu.Lock()
	s.stats.totalSyncs++
	syncStartTime := time.Now()
	s.stats.lastSyncTime = syncStartTime
	s.stats.mu.Unlock()

	log.Printf("[PriceSyncer] 🎯 Starting market-separated price sync")

	// 获取现货和期货交易对配置
	syncerConfig := s.buildPriceSyncerConfig()

	totalUpdates := 0
	totalErrors := 0

	// 同步现货价格
	if len(syncerConfig.SpotSymbols) > 0 {
		log.Printf("[PriceSyncer] 📈 Starting spot market price sync for %d symbols", len(syncerConfig.SpotSymbols))
		spotUpdates, spotErrors := s.syncSpotPricesForSymbols(ctx, syncerConfig.SpotSymbols)
		totalUpdates += spotUpdates
		totalErrors += spotErrors
	} else {
		log.Printf("[PriceSyncer] ⚠️ No spot symbols to sync")
	}

	// 同步期货价格
	if len(syncerConfig.FuturesSymbols) > 0 {
		log.Printf("[PriceSyncer] 📈 Starting futures market price sync for %d symbols", len(syncerConfig.FuturesSymbols))
		futuresUpdates, futuresErrors := s.syncFuturesPricesForSymbols(ctx, syncerConfig.FuturesSymbols)
		totalUpdates += futuresUpdates
		totalErrors += futuresErrors
	} else {
		log.Printf("[PriceSyncer] ⚠️ No futures symbols to sync")
	}

	totalDuration := time.Since(syncStartTime)

	s.stats.mu.Lock()
	if totalErrors == 0 {
		s.stats.successfulSyncs++
	}
	s.stats.totalPriceUpdates += int64(totalUpdates)
	s.stats.mu.Unlock()

	// 生成详细的同步报告
	log.Printf("[PriceSyncer] 📊 Price sync completed in %v", totalDuration)
	log.Printf("[PriceSyncer] 📈 Total updates: %d", totalUpdates)
	log.Printf("[PriceSyncer] 📊 Markets synced: spot(%d), futures(%d)",
		len(syncerConfig.SpotSymbols), len(syncerConfig.FuturesSymbols))

	if totalErrors > 0 {
		log.Printf("[PriceSyncer] ⚠️ %d markets had errors - check logs above", totalErrors)
		return fmt.Errorf("completed with %d market errors", totalErrors)
	}

	return nil
}

func (s *PriceSyncer) syncSpotPrices(ctx context.Context, symbols []string) (int, error) {
	if len(symbols) == 0 {
		return 0, nil
	}

	updates := 0
	errors := 0
	websocketHits := 0
	restAPICalls := 0

	log.Printf("[PriceSyncer] 🌐 Syncing spot prices for %d symbols (WebSocket priority)...", len(symbols))

	// 设置最大数据年龄（例如5分钟内的数据认为有效）
	maxDataAge := 5 * time.Minute
	if s.config.Timeouts.DataAgeMax > 0 {
		maxDataAge = time.Duration(s.config.Timeouts.DataAgeMax) * time.Second
	}

	// 检查WebSocket状态（只在开始时打印一次）
	if s.websocketSyncer != nil {
		isRunning := s.websocketSyncer.IsRunning()
		isHealthy := s.websocketSyncer.IsHealthy()
		healthStatus := s.websocketSyncer.GetHealthStatus()

		log.Printf("[PriceSyncer] 📊 WebSocket status: running=%v, healthy=%v, spot_conns=%v, futures_conns=%v, last_msg=%v",
			isRunning, isHealthy,
			healthStatus["spot_connections"],
			healthStatus["futures_connections"],
			healthStatus["time_since_last_message"])
	} else {
		log.Printf("[PriceSyncer] ⚠️ WebSocket syncer not available, will use REST API only")
	}

	for _, symbol := range symbols {
		// 注意：无效符号已在配置构建阶段过滤，这里不再需要检查

		var price string
		var lastUpdated time.Time
		var fromWebSocket bool

		// 优先尝试从WebSocket缓存获取数据
		if s.websocketSyncer != nil && s.websocketSyncer.IsRunning() && s.websocketSyncer.IsHealthy() {
			if wsPrice, wsTime, exists := s.websocketSyncer.GetLatestPrice(symbol, "spot"); exists && time.Since(wsTime) <= maxDataAge {
				price = wsPrice
				lastUpdated = wsTime
				fromWebSocket = true
				websocketHits++
			}
		}

		// 如果WebSocket数据不可用，回退到REST API
		if !fromWebSocket {
			restAPICalls++

			// 调用Binance现货价格API
			url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol)
			type PriceResponse struct {
				Symbol string `json:"symbol"`
				Price  string `json:"price"`
			}

			startTime := time.Now()
			var resp PriceResponse
			if err := netutil.GetJSON(ctx, url, &resp); err != nil {
				// 检查是否为无效符号错误
				errStr := err.Error()
				if strings.Contains(errStr, "Invalid symbol") || strings.Contains(errStr, "-1121") {
					s.markSymbolInvalid(symbol, "spot")
				} else {
					log.Printf("[PriceSyncer] ❌ Failed to get spot price for %s: %v", symbol, err)
					errors++
				}
				continue
			}

			price = resp.Price
			lastUpdated = time.Now()
			latency := time.Since(startTime)

			// 验证价格数据
			if price == "" || price == "0" {
				log.Printf("[PriceSyncer] ⚠️ Invalid spot price for %s: %s", symbol, price)
				errors++
				continue
			}

			log.Printf("[PriceSyncer] ✅ Spot price fetched via REST: %s = %s (latency: %v)", symbol, price, latency)
		} else {
			log.Printf("[PriceSyncer] ✅ Spot price from WebSocket: %s = %s (age: %v)", symbol, price, time.Since(lastUpdated))
		}

		// 保存到价格缓存
		cache := &pdb.PriceCache{
			Symbol:         symbol,
			Kind:           "spot",
			Price:          price,
			PriceChange24h: nil, // 不设置24小时价格变化
			LastUpdated:    lastUpdated,
		}

		if err := pdb.SavePriceCache(s.db, cache); err != nil {
			log.Printf("[PriceSyncer] ❌ Failed to save spot price cache for %s: %v", symbol, err)
			errors++
			continue
		}

		updates++
	}

	// 更新统计信息
	s.stats.mu.Lock()
	s.stats.websocketHits += int64(websocketHits)
	s.stats.restAPICalls += int64(restAPICalls)
	s.stats.mu.Unlock()

	log.Printf("[PriceSyncer] 📊 Spot price sync summary: %d successful, %d errors, %d WebSocket hits, %d REST API calls",
		updates, errors, websocketHits, restAPICalls)
	return updates, nil
}

func (s *PriceSyncer) syncFuturesPrices(ctx context.Context, symbols []string) (int, error) {
	if len(symbols) == 0 {
		return 0, nil
	}

	updates := 0
	errors := 0
	websocketHits := 0
	restAPICalls := 0

	log.Printf("[PriceSyncer] 🚀 Syncing futures prices for %d symbols (WebSocket priority)...", len(symbols))

	// 设置最大数据年龄（例如5分钟内的数据认为有效）
	maxDataAge := 5 * time.Minute
	if s.config.Timeouts.DataAgeMax > 0 {
		maxDataAge = time.Duration(s.config.Timeouts.DataAgeMax) * time.Second
	}

	// 检查WebSocket状态（只在开始时打印一次）
	if s.websocketSyncer != nil {
		isRunning := s.websocketSyncer.IsRunning()
		isHealthy := s.websocketSyncer.IsHealthy()
		healthStatus := s.websocketSyncer.GetHealthStatus()

		log.Printf("[PriceSyncer] 📊 WebSocket status: running=%v, healthy=%v, spot_conns=%v, futures_conns=%v, last_msg=%v",
			isRunning, isHealthy,
			healthStatus["spot_connections"],
			healthStatus["futures_connections"],
			healthStatus["time_since_last_message"])
	} else {
		log.Printf("[PriceSyncer] ⚠️ WebSocket syncer not available, will use REST API only")
	}

	for _, symbol := range symbols {
		// 注意：无效符号已在配置构建阶段过滤，这里不再需要检查

		var price string
		var lastUpdated time.Time
		var fromWebSocket bool

		// 优先尝试从WebSocket缓存获取数据
		if s.websocketSyncer != nil && s.websocketSyncer.IsRunning() && s.websocketSyncer.IsHealthy() {
			if wsPrice, wsTime, exists := s.websocketSyncer.GetLatestPrice(symbol, "futures"); exists && time.Since(wsTime) <= maxDataAge {
				price = wsPrice
				lastUpdated = wsTime
				fromWebSocket = true
				websocketHits++
			}
		}

		// 如果WebSocket数据不可用，回退到REST API
		if !fromWebSocket {
			restAPICalls++

			// 调用Binance期货价格API
			url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/ticker/price?symbol=%s", symbol)
			type PriceResponse struct {
				Symbol string `json:"symbol"`
				Price  string `json:"price"`
			}

			startTime := time.Now()
			var resp PriceResponse
			if err := netutil.GetJSON(ctx, url, &resp); err != nil {
				// 检查是否为无效符号错误
				errStr := err.Error()
				if strings.Contains(errStr, "Invalid symbol") || strings.Contains(errStr, "-1121") {
					s.markSymbolInvalid(symbol, "futures")
				} else {
					log.Printf("[PriceSyncer] ❌ Failed to get futures price for %s: %v", symbol, err)
					errors++
				}
				continue
			}

			price = resp.Price
			lastUpdated = time.Now()
			latency := time.Since(startTime)

			// 验证价格数据
			if price == "" || price == "0" {
				log.Printf("[PriceSyncer] ⚠️ Invalid futures price for %s: %s", symbol, price)
				errors++
				continue
			}

			log.Printf("[PriceSyncer] ✅ Futures price fetched via REST: %s = %s (latency: %v)", symbol, price, latency)
		} else {
			log.Printf("[PriceSyncer] ✅ Futures price from WebSocket: %s = %s (age: %v)", symbol, price, time.Since(lastUpdated))
		}

		// 保存到价格缓存
		cache := &pdb.PriceCache{
			Symbol:         symbol,
			Kind:           "futures",
			Price:          price,
			PriceChange24h: nil, // 不设置24小时价格变化
			LastUpdated:    lastUpdated,
		}

		if err := pdb.SavePriceCache(s.db, cache); err != nil {
			log.Printf("[PriceSyncer] ❌ Failed to save futures price cache for %s: %v", symbol, err)
			errors++
			continue
		}

		updates++
	}

	// 更新统计信息
	s.stats.mu.Lock()
	s.stats.websocketHits += int64(websocketHits)
	s.stats.restAPICalls += int64(restAPICalls)
	s.stats.mu.Unlock()

	log.Printf("[PriceSyncer] 📊 Futures price sync summary: %d successful, %d errors, %d WebSocket hits, %d REST API calls",
		updates, errors, websocketHits, restAPICalls)
	return updates, nil
}

func (s *PriceSyncer) GetStats() map[string]interface{} {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	// 计算命中率
	totalDataRequests := s.stats.websocketHits + s.stats.restAPICalls
	websocketHitRate := float64(0)
	if totalDataRequests > 0 {
		websocketHitRate = float64(s.stats.websocketHits) / float64(totalDataRequests) * 100
	}

	return map[string]interface{}{
		"total_syncs":         s.stats.totalSyncs,
		"successful_syncs":    s.stats.successfulSyncs,
		"failed_syncs":        s.stats.failedSyncs,
		"last_sync_time":      s.stats.lastSyncTime,
		"total_updates":       s.stats.totalPriceUpdates,
		"websocket_hits":      s.stats.websocketHits,
		"rest_api_calls":      s.stats.restAPICalls,
		"websocket_hit_rate":  fmt.Sprintf("%.1f%%", websocketHitRate),
		"websocket_available": s.websocketSyncer != nil && s.websocketSyncer.IsRunning(),
	}
}

// GetAPIStats 获取API统计信息
func (s *PriceSyncer) GetAPIStats() *server.APIStats {
	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	totalCalls := s.stats.websocketHits + s.stats.restAPICalls
	successRate := "0%"
	if totalCalls > 0 {
		rate := float64(s.stats.websocketHits+s.stats.restAPICalls) / float64(totalCalls) * 100
		successRate = fmt.Sprintf("%.1f%%", rate)
	}

	hitRate := "0%"
	if s.stats.websocketHits+s.stats.restAPICalls > 0 {
		rate := float64(s.stats.websocketHits) / float64(s.stats.websocketHits+s.stats.restAPICalls) * 100
		hitRate = fmt.Sprintf("%.1f%%", rate)
	}

	return &server.APIStats{
		TotalCalls:       totalCalls,
		APICallsTotal:    totalCalls,
		APISuccessRate:   successRate,
		TotalSyncs:       s.stats.totalPriceUpdates,
		SuccessfulSyncs:  s.stats.totalPriceUpdates,
		WebSocketHits:    s.stats.websocketHits,
		RestAPICalls:     s.stats.restAPICalls,
		WebSocketHitRate: hitRate,
	}
}

// syncSpotPricesForSymbols 同步指定现货交易对的价格数据
func (s *PriceSyncer) syncSpotPricesForSymbols(ctx context.Context, symbols []string) (int, int) {
	if len(symbols) == 0 {
		return 0, 0
	}

	updates := 0
	errors := 0
	websocketHits := 0
	restAPICalls := 0

	log.Printf("[PriceSyncer] 🌐 Syncing spot prices for %d symbols (WebSocket priority)...", len(symbols))

	// 设置最大数据年龄（例如5分钟内的数据认为有效）
	maxDataAge := 5 * time.Minute
	if s.config.Timeouts.DataAgeMax > 0 {
		maxDataAge = time.Duration(s.config.Timeouts.DataAgeMax) * time.Second
	}

	// 检查WebSocket状态
	if s.websocketSyncer != nil {
		isRunning := s.websocketSyncer.IsRunning()
		isHealthy := s.websocketSyncer.IsHealthy()
		healthStatus := s.websocketSyncer.GetHealthStatus()

		log.Printf("[PriceSyncer] 📊 WebSocket status: running=%v, healthy=%v, spot_conns=%v",
			isRunning, isHealthy, healthStatus["spot_connections"])
	} else {
		log.Printf("[PriceSyncer] ⚠️ WebSocket syncer not available, will use REST API only")
	}

	for _, symbol := range symbols {
		// 注意：无效符号已在配置构建阶段过滤，这里不再需要检查

		var price string
		var lastUpdated time.Time
		var fromWebSocket bool

		// 优先尝试从WebSocket缓存获取数据
		if s.websocketSyncer != nil && s.websocketSyncer.IsRunning() && s.websocketSyncer.IsHealthy() {
			if wsPrice, wsTime, exists := s.websocketSyncer.GetLatestPrice(symbol, "spot"); exists && time.Since(wsTime) <= maxDataAge {
				price = wsPrice
				lastUpdated = wsTime
				fromWebSocket = true
				websocketHits++
			}
		}

		// 如果WebSocket数据不可用，回退到REST API
		if !fromWebSocket {
			restAPICalls++

			// 调用Binance现货价格API
			url := fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%s", symbol)
			type PriceResponse struct {
				Symbol string `json:"symbol"`
				Price  string `json:"price"`
			}

			startTime := time.Now()
			var resp PriceResponse
			if err := netutil.GetJSON(ctx, url, &resp); err != nil {
				// 检查是否为无效符号错误
				errStr := err.Error()
				if strings.Contains(errStr, "Invalid symbol") || strings.Contains(errStr, "-1121") {
					s.markSymbolInvalid(symbol, "spot")
				} else {
					log.Printf("[PriceSyncer] ❌ Failed to get spot price for %s: %v", symbol, err)
					errors++
				}
				continue
			}

			price = resp.Price
			lastUpdated = time.Now()
			latency := time.Since(startTime)

			// 验证价格数据
			if price == "" || price == "0" {
				log.Printf("[PriceSyncer] ⚠️ Invalid spot price for %s: %s", symbol, price)
				errors++
				continue
			}

			log.Printf("[PriceSyncer] ✅ Spot price fetched via REST: %s = %s (latency: %v)", symbol, price, latency)
		} else {
			log.Printf("[PriceSyncer] ✅ Spot price from WebSocket: %s = %s (age: %v)", symbol, price, time.Since(lastUpdated))
		}

		// 保存到价格缓存
		cache := &pdb.PriceCache{
			Symbol:         symbol,
			Kind:           "spot",
			Price:          price,
			PriceChange24h: nil, // 不设置24小时价格变化
			LastUpdated:    lastUpdated,
		}

		if err := pdb.SavePriceCache(s.db, cache); err != nil {
			log.Printf("[PriceSyncer] ❌ Failed to save spot price cache for %s: %v", symbol, err)
			errors++
		} else {
			updates++
		}
	}

	log.Printf("[PriceSyncer] 📊 Spot price sync: %d updates, %d errors, %d WebSocket hits, %d REST calls",
		updates, errors, websocketHits, restAPICalls)

	return updates, errors
}

// syncFuturesPricesForSymbols 同步指定期货交易对的价格数据
func (s *PriceSyncer) syncFuturesPricesForSymbols(ctx context.Context, symbols []string) (int, int) {
	if len(symbols) == 0 {
		return 0, 0
	}

	updates := 0
	errors := 0
	websocketHits := 0
	restAPICalls := 0

	log.Printf("[PriceSyncer] 🌐 Syncing futures prices for %d symbols (WebSocket priority)...", len(symbols))

	// 设置最大数据年龄
	maxDataAge := 5 * time.Minute
	if s.config.Timeouts.DataAgeMax > 0 {
		maxDataAge = time.Duration(s.config.Timeouts.DataAgeMax) * time.Second
	}

	// 检查WebSocket状态
	if s.websocketSyncer != nil {
		isRunning := s.websocketSyncer.IsRunning()
		isHealthy := s.websocketSyncer.IsHealthy()
		healthStatus := s.websocketSyncer.GetHealthStatus()

		log.Printf("[PriceSyncer] 📊 WebSocket status: running=%v, healthy=%v, futures_conns=%v",
			isRunning, isHealthy, healthStatus["futures_connections"])
	} else {
		log.Printf("[PriceSyncer] ⚠️ WebSocket syncer not available, will use REST API only")
	}

	for _, symbol := range symbols {
		// 注意：无效符号已在配置构建阶段过滤，这里不再需要检查

		var price string
		var lastUpdated time.Time
		var fromWebSocket bool

		// 优先尝试从WebSocket缓存获取数据
		if s.websocketSyncer != nil && s.websocketSyncer.IsRunning() && s.websocketSyncer.IsHealthy() {
			if wsPrice, wsTime, exists := s.websocketSyncer.GetLatestPrice(symbol, "futures"); exists && time.Since(wsTime) <= maxDataAge {
				price = wsPrice
				lastUpdated = wsTime
				fromWebSocket = true
				websocketHits++
			}
		}

		// 如果WebSocket数据不可用，回退到REST API
		if !fromWebSocket {
			restAPICalls++

			// 调用Binance期货价格API
			url := fmt.Sprintf("https://fapi.binance.com/fapi/v1/ticker/price?symbol=%s", symbol)
			type PriceResponse struct {
				Symbol string `json:"symbol"`
				Price  string `json:"price"`
			}

			startTime := time.Now()
			var resp PriceResponse
			if err := netutil.GetJSON(ctx, url, &resp); err != nil {
				// 检查是否为无效符号错误
				errStr := err.Error()
				if strings.Contains(errStr, "Invalid symbol") || strings.Contains(errStr, "-1121") {
					s.markSymbolInvalid(symbol, "futures")
				} else {
					log.Printf("[PriceSyncer] ❌ Failed to get futures price for %s: %v", symbol, err)
					errors++
				}
				continue
			}

			price = resp.Price
			lastUpdated = time.Now()
			latency := time.Since(startTime)

			// 验证价格数据
			if price == "" || price == "0" {
				log.Printf("[PriceSyncer] ⚠️ Invalid futures price for %s: %s", symbol, price)
				errors++
				continue
			}

			log.Printf("[PriceSyncer] ✅ Futures price fetched via REST: %s = %s (latency: %v)", symbol, price, latency)
		} else {
			log.Printf("[PriceSyncer] ✅ Futures price from WebSocket: %s = %s (age: %v)", symbol, price, time.Since(lastUpdated))
		}

		// 保存到价格缓存
		cache := &pdb.PriceCache{
			Symbol:         symbol,
			Kind:           "futures",
			Price:          price,
			PriceChange24h: nil, // 不设置24小时价格变化
			LastUpdated:    lastUpdated,
		}

		if err := pdb.SavePriceCache(s.db, cache); err != nil {
			log.Printf("[PriceSyncer] ❌ Failed to save futures price cache for %s: %v", symbol, err)
			errors++
		} else {
			updates++
		}
	}

	log.Printf("[PriceSyncer] 📊 Futures price sync: %d updates, %d errors, %d WebSocket hits, %d REST calls",
		updates, errors, websocketHits, restAPICalls)

	return updates, errors
}
