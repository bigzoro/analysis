package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"gorm.io/gorm"
)

// ===== 价格缓存系统 =====
// 实现实时价格缓存和24h基准价格缓存，支持高效的价格查询和更新

// RealtimePriceCache 实时价格缓存
type RealtimePriceCache struct {
	// 价格数据缓存：symbol -> RealtimePriceData
	prices map[string]*RealtimePriceData
	mu     sync.RWMutex

	// 缓存配置
	maxEntries      int           // 最大缓存条目数
	ttl             time.Duration // 缓存过期时间
	cleanupInterval time.Duration // 清理间隔

	// 统计信息（原子操作）
	hits        int64                   // 使用原子操作
	misses      int64                   // 使用原子操作
	accessStats map[string]*AccessStats // 访问统计
	statsMu     sync.RWMutex            // 访问统计的锁
}

// AccessStats 访问统计
type AccessStats struct {
	accessCount    int64         // 访问次数
	lastAccessTime time.Time     // 最后访问时间
	avgAccessFreq  time.Duration // 平均访问频率
}

// RealtimePriceData 实时价格数据结构
type RealtimePriceData struct {
	LastPrice     float64   `json:"last_price"`               // 最新价格
	Volume24h     float64   `json:"volume_24h"`               // 24h成交量
	ChangePercent *float64  `json:"change_percent,omitempty"` // 24h涨跌幅百分比，nil表示未设置
	Source        string    `json:"source"`                   // 数据来源
	Timestamp     time.Time `json:"timestamp"`                // 更新时间戳
	ExpireTime    time.Time `json:"expire_time"`              // 过期时间
}

// BasePriceCache 基准价格缓存（24h前的价格）
type BasePriceCache struct {
	// 基准价格缓存：symbol -> basePrice
	basePrices map[string]*BasePriceData
	mu         sync.RWMutex

	// 缓存配置
	refreshInterval time.Duration // 刷新间隔
	cleanupInterval time.Duration // 清理间隔
	maxEntries      int           // 最大缓存条目数

	// 数据库引用（用于刷新基准价格）
	db interface{}

	// 运行状态
	lastRefreshTime time.Time     // 最后刷新时间
	refreshStats    *RefreshStats // 刷新统计
}

// RefreshStats 刷新统计信息
type RefreshStats struct {
	mu                  sync.RWMutex
	totalRefreshes      int64         // 总刷新次数
	successfulRefreshes int64         // 成功刷新次数
	failedRefreshes     int64         // 失败刷新次数
	lastError           error         // 最后错误
	lastErrorTime       time.Time     // 最后错误时间
	avgRefreshTime      time.Duration // 平均刷新时间
}

// BasePriceData 基准价格数据
type BasePriceData struct {
	Price      float64   `json:"price"`       // 基准价格
	Timestamp  time.Time `json:"timestamp"`   // 基准时间戳
	ExpireTime time.Time `json:"expire_time"` // 过期时间
}

// NewRealtimePriceCache 创建实时价格缓存
func NewRealtimePriceCache() *RealtimePriceCache {
	cache := &RealtimePriceCache{
		prices:          make(map[string]*RealtimePriceData),
		maxEntries:      1000,            // 默认最大1000个条目
		ttl:             5 * time.Minute, // 默认5分钟过期
		cleanupInterval: 1 * time.Minute, // 默认1分钟清理一次
		accessStats:     make(map[string]*AccessStats),
	}

	// 启动清理goroutine
	go cache.startCleanupRoutine()

	log.Printf("[RealtimePriceCache] 初始化完成 - 最大条目:%d, TTL:%v", cache.maxEntries, cache.ttl)
	return cache
}

// NewBasePriceCache 创建基准价格缓存
func NewBasePriceCache() *BasePriceCache {
	cache := &BasePriceCache{
		basePrices:      make(map[string]*BasePriceData),
		refreshInterval: 1 * time.Hour,    // 默认1小时刷新一次
		maxEntries:      1000,             // 默认最大1000个条目
		cleanupInterval: 30 * time.Minute, // 30分钟清理一次过期数据
		lastRefreshTime: time.Now(),
		refreshStats:    &RefreshStats{},
	}

	// 启动刷新和清理goroutine
	go cache.startRefreshAndCleanupRoutine()

	log.Printf("[BasePriceCache] 初始化完成 - 刷新间隔:%v, 清理间隔:%v, 最大条目:%d",
		cache.refreshInterval, cache.cleanupInterval, cache.maxEntries)
	return cache
}

// UpdatePrice 更新价格数据
func (c *RealtimePriceCache) UpdatePrice(update PriceUpdate) {
	// 先尝试获取读锁检查是否需要清理
	c.mu.RLock()
	needsCleanup := len(c.prices) >= c.maxEntries
	c.mu.RUnlock()

	// 如果需要清理，先清理过期条目
	if needsCleanup {
		c.mu.Lock()
		c.cleanupExpired()
		// 再次检查是否还有空间
		if len(c.prices) >= c.maxEntries {
			c.mu.Unlock()
			log.Printf("[RealtimePriceCache] 缓存已满，跳过更新: %s", update.Symbol)
			return
		}
		c.mu.Unlock()
	}

	// 创建或更新价格数据（使用写锁）
	now := time.Now()
	priceData := &RealtimePriceData{
		LastPrice:     update.Price,
		Volume24h:     update.Volume,
		ChangePercent: update.ChangePercent, // 可能是nil，表示未设置
		Source:        update.Source,
		Timestamp:     update.Timestamp,
		ExpireTime:    now.Add(c.ttl),
	}

	c.mu.Lock()
	c.prices[update.Symbol] = priceData
	c.mu.Unlock()

	// 移除频繁的价格更新日志
}

// GetPrice 获取价格数据
func (c *RealtimePriceCache) GetPrice(symbol string) (*RealtimePriceData, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	priceData, exists := c.prices[symbol]
	if !exists {
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	now := time.Now()

	// 检查是否过期
	if now.After(priceData.ExpireTime) {
		atomic.AddInt64(&c.misses, 1)
		delete(c.prices, symbol)
		c.statsMu.Lock()
		delete(c.accessStats, symbol)
		c.statsMu.Unlock()
		return nil, false
	}

	atomic.AddInt64(&c.hits, 1)

	// 更新访问统计（使用单独的锁）
	c.statsMu.Lock()
	stats, exists := c.accessStats[symbol]
	if !exists {
		stats = &AccessStats{}
		c.accessStats[symbol] = stats
	}
	stats.accessCount++
	stats.lastAccessTime = now

	// 动态调整过期时间（基于访问频率）
	c.adjustExpireTime(priceData, stats)
	c.statsMu.Unlock()

	return priceData, true
}

// GetAllPrices 获取所有价格数据
func (c *RealtimePriceCache) GetAllPrices() map[string]*RealtimePriceData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建副本，避免外部修改
	result := make(map[string]*RealtimePriceData)
	now := time.Now()

	for symbol, priceData := range c.prices {
		// 只返回未过期的数据
		if now.Before(priceData.ExpireTime) {
			result[symbol] = priceData
		}
	}

	return result
}

// GetVolume24h 获取24h成交量
func (c *RealtimePriceCache) GetVolume24h(symbol string) float64 {
	if priceData, exists := c.GetPrice(symbol); exists {
		return priceData.Volume24h
	}
	return 0
}

// GetCacheStats 获取缓存统计信息
func (c *RealtimePriceCache) GetCacheStats() map[string]interface{} {
	c.mu.RLock()
	entriesCount := len(c.prices)
	c.mu.RUnlock()

	hits := atomic.LoadInt64(&c.hits)
	misses := atomic.LoadInt64(&c.misses)

	totalRequests := hits + misses
	hitRate := float64(0)
	if totalRequests > 0 {
		hitRate = float64(hits) / float64(totalRequests)
	}

	// 计算访问频率统计
	c.statsMu.RLock()
	totalAccessCount := int64(0)
	activeSymbols := 0
	for _, stats := range c.accessStats {
		if stats.accessCount > 0 {
			totalAccessCount += stats.accessCount
			activeSymbols++
		}
	}
	accessStatsCount := len(c.accessStats)
	c.statsMu.RUnlock()

	return map[string]interface{}{
		"entries_count":      entriesCount,
		"max_entries":        c.maxEntries,
		"ttl":                c.ttl.String(),
		"hits":               hits,
		"misses":             misses,
		"hit_rate":           hitRate,
		"cleanup_interval":   c.cleanupInterval.String(),
		"total_access_count": totalAccessCount,
		"active_symbols":     activeSymbols,
		"access_stats_count": accessStatsCount,
	}
}

// cleanupExpired 清理过期条目
func (c *RealtimePriceCache) cleanupExpired() {
	now := time.Now()
	expired := 0

	for symbol, priceData := range c.prices {
		if now.After(priceData.ExpireTime) {
			delete(c.prices, symbol)
			delete(c.accessStats, symbol)
			expired++
		}
	}

	// 清理长时间未访问的统计数据
	statsExpired := 0
	for symbol, stats := range c.accessStats {
		// 如果超过24小时未访问，清理统计数据
		if now.Sub(stats.lastAccessTime) > 24*time.Hour {
			delete(c.accessStats, symbol)
			statsExpired++
		}
	}

	if expired > 0 || statsExpired > 0 {
		log.Printf("[RealtimePriceCache] 清理完成 - 过期价格:%d个, 过期统计:%d个", expired, statsExpired)
	}
}

// adjustExpireTime 动态调整过期时间
func (c *RealtimePriceCache) adjustExpireTime(priceData *RealtimePriceData, stats *AccessStats) {
	if stats.accessCount < 5 {
		// 访问次数少，使用标准TTL
		return
	}

	// 计算访问频率（最近访问间隔）
	timeSinceLastAccess := time.Since(stats.lastAccessTime)
	if stats.avgAccessFreq == 0 {
		stats.avgAccessFreq = timeSinceLastAccess
	} else {
		// 指数移动平均
		stats.avgAccessFreq = (stats.avgAccessFreq + timeSinceLastAccess) / 2
	}

	// 根据访问频率调整TTL
	var newTTL time.Duration
	switch {
	case stats.avgAccessFreq < 30*time.Second:
		// 高频访问：延长TTL到15分钟
		newTTL = 15 * time.Minute
	case stats.avgAccessFreq < 2*time.Minute:
		// 中频访问：延长TTL到10分钟
		newTTL = 10 * time.Minute
	case stats.avgAccessFreq < 10*time.Minute:
		// 低频访问：标准TTL 5分钟
		newTTL = 5 * time.Minute
	default:
		// 极低频访问：缩短TTL到2分钟
		newTTL = 2 * time.Minute
	}

	// 更新过期时间
	newExpireTime := time.Now().Add(newTTL)
	if newExpireTime.After(priceData.ExpireTime) {
		priceData.ExpireTime = newExpireTime
	}
}

// startCleanupRoutine 启动清理例程
func (c *RealtimePriceCache) startCleanupRoutine() {
	ticker := time.NewTicker(c.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		c.mu.Lock()
		c.cleanupExpired()
		c.mu.Unlock()
	}
}

// GetBasePrice 获取基准价格
func (c *BasePriceCache) GetBasePrice(symbol string) float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	baseData, exists := c.basePrices[symbol]
	if !exists {
		return 0
	}

	// 检查是否过期
	if time.Now().After(baseData.ExpireTime) {
		delete(c.basePrices, symbol) // 删除过期数据
		return 0
	}

	return baseData.Price
}

// UpdateBasePrice 更新基准价格
func (c *BasePriceCache) UpdateBasePrice(symbol string, price float64) {
	// 先尝试获取读锁检查是否需要清理
	c.mu.RLock()
	needsCleanup := len(c.basePrices) >= c.maxEntries
	c.mu.RUnlock()

	// 如果需要清理，先清理过期条目
	if needsCleanup {
		c.mu.Lock()
		c.cleanupExpiredBasePrices()
		// 再次检查是否还有空间
		if len(c.basePrices) >= c.maxEntries {
			c.mu.Unlock()
			log.Printf("[BasePriceCache] 缓存已满，跳过更新: %s", symbol)
			return
		}
		c.mu.Unlock()
	}

	// 创建或更新基准价格数据（使用写锁）
	now := time.Now()
	baseData := &BasePriceData{
		Price:      price,
		Timestamp:  now,
		ExpireTime: now.Add(c.refreshInterval * 2), // 过期时间为刷新间隔的2倍
	}

	c.mu.Lock()
	c.basePrices[symbol] = baseData
	c.mu.Unlock()

	// 移除频繁的基准价格更新日志
}

// GetAllBasePrices 获取所有基准价格
func (c *BasePriceCache) GetAllBasePrices() map[string]*BasePriceData {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// 创建副本，避免外部修改
	result := make(map[string]*BasePriceData)
	now := time.Now()

	for symbol, baseData := range c.basePrices {
		// 只返回未过期的数据
		if now.Before(baseData.ExpireTime) {
			result[symbol] = baseData
		}
	}

	return result
}

// RefreshBasePrices 从数据库刷新基准价格
func (c *BasePriceCache) RefreshBasePrices(db interface{}) error {
	// 开始刷新基准价格

	// 获取需要刷新的交易对列表
	symbols := c.getSymbolsToRefresh(db)
	if len(symbols) == 0 {
		log.Printf("[BasePriceCache] 没有找到需要刷新的交易对")
		return nil
	}

	// 需要刷新基准价格

	// 批量查询基准价格，避免为每个交易对单独查询
	basePrices, err := c.queryBasePricesBatch(db, symbols)
	if err != nil {
		log.Printf("[BasePriceCache] 批量查询基准价格失败: %v，将尝试逐个查询", err)
		// 降级到逐个查询
		refreshedCount := 0
		for _, symbol := range symbols {
			basePrice := c.queryBasePriceFromDB(db, symbol)
			if basePrice > 0 {
				c.UpdateBasePrice(symbol, basePrice)
				refreshedCount++
			}
		}
		// 基准价格刷新完成
		return nil
	}

	// 批量更新缓存
	refreshedCount := 0
	for symbol, basePrice := range basePrices {
		if basePrice > 0 {
			c.UpdateBasePrice(symbol, basePrice)
			refreshedCount++
		}
	}

	// 基准价格刷新完成
	return nil
}

// getSymbolsToRefresh 获取需要刷新的交易对列表
func (c *BasePriceCache) getSymbolsToRefresh(db interface{}) []string {
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		log.Printf("[BasePriceCache] 数据库类型错误，无法获取交易对列表")
		// 返回默认的主要交易对
		return []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT", "SOLUSDT"}
	}

	// 从多个来源获取活跃交易对列表：
	// 1. 当前缓存中的交易对
	// 2. 最近24小时有交易数据的交易对
	// 3. 涨幅榜中的交易对

	var symbols []string
	symbolSet := make(map[string]bool)

	// 1. 添加当前缓存中的交易对
	c.mu.RLock()
	for symbol := range c.basePrices {
		if !symbolSet[symbol] {
			symbols = append(symbols, symbol)
			symbolSet[symbol] = true
		}
	}
	c.mu.RUnlock()

	// 2. 从binance_24h_stats表获取最近活跃的交易对
	query := `
		SELECT symbol, MAX(volume) as max_volume
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 24 HOUR)
		  AND volume > 0
		  AND last_price > 0
		GROUP BY symbol
		ORDER BY max_volume DESC
		LIMIT 100
	`

	rows, err := gormDB.Raw(query).Rows()
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var symbol string
			var maxVolume float64
			if err := rows.Scan(&symbol, &maxVolume); err == nil && !symbolSet[symbol] {
				symbols = append(symbols, symbol)
				symbolSet[symbol] = true
			}
		}
	} else {
		log.Printf("[BasePriceCache] 查询活跃交易对失败: %v", err)
	}

	// 3. 添加主要交易对作为兜底
	majorSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "XRPUSDT", "SOLUSDT", "DOTUSDT"}
	for _, symbol := range majorSymbols {
		if !symbolSet[symbol] {
			symbols = append(symbols, symbol)
			symbolSet[symbol] = true
		}
	}

	log.Printf("[BasePriceCache] 获取到%d个交易对需要刷新", len(symbols))
	return symbols
}

// queryBasePricesBatch 批量查询基准价格
func (c *BasePriceCache) queryBasePricesBatch(db interface{}, symbols []string) (map[string]float64, error) {
	if len(symbols) == 0 {
		return make(map[string]float64), nil
	}

	gormDB, ok := db.(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("数据库类型错误")
	}

	// 使用IN查询批量获取数据，避免N+1查询问题
	// 为每个交易对找到最近的24小时前的数据
	query := `
		SELECT
			symbol,
			close_price
		FROM (
			SELECT
				symbol,
				close_price,
				ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY open_time DESC) as rn
			FROM market_klines
			WHERE symbol IN ?
				AND ` + "`interval`" + ` = '1h'
				AND open_time <= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 24 HOUR)
		) ranked
		WHERE rn = 1
	`

	// 构建IN参数列表
	args := make([]interface{}, len(symbols))
	for i, symbol := range symbols {
		args[i] = symbol
	}

	var results []struct {
		Symbol     string `json:"symbol"`
		ClosePrice string `json:"close_price"`
	}

	// 注意：GORM的Raw方法对IN查询的参数处理可能需要特殊处理
	// 我们使用字符串构建方式来确保正确性
	symbolsStr := make([]string, len(symbols))
	for i, symbol := range symbols {
		symbolsStr[i] = "'" + symbol + "'"
	}
	inClause := strings.Join(symbolsStr, ",")

	finalQuery := strings.Replace(query, "symbol IN ?", "symbol IN ("+inClause+")", 1)

	err := gormDB.Raw(finalQuery).Scan(&results).Error
	if err != nil {
		return nil, fmt.Errorf("批量查询基准价格失败: %w", err)
	}

	// 转换为map
	basePrices := make(map[string]float64)
	for _, result := range results {
		if result.ClosePrice != "" {
			if price, err := strconv.ParseFloat(result.ClosePrice, 64); err == nil && price > 0 {
				basePrices[result.Symbol] = price
			}
		}
	}

	log.Printf("[BasePriceCache] 批量查询到%d/%d个交易对的基准价格", len(basePrices), len(symbols))
	return basePrices, nil
}

// SetDatabase 设置数据库连接
func (c *BasePriceCache) SetDatabase(db interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.db = db
	log.Printf("[BasePriceCache] 数据库连接已设置")
}

// queryBasePriceFromDB 从数据库查询基准价格（24小时前）
func (c *BasePriceCache) queryBasePriceFromDB(db interface{}, symbol string, kind ...string) float64 {
	// 从market_klines表查询24小时前的收盘价作为基准价格
	// 查询最近的1小时K线数据（24小时前的数据）
	var closePriceStr string

	// 构建查询语句
	var query string
	var args []interface{}

	query = `
		SELECT close_price
		FROM market_klines
		WHERE symbol = ?
		  AND ` + "`interval`" + ` = '1h'
		  AND open_time <= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 24 HOUR)
	`

	args = []interface{}{symbol}

	// 如果提供了市场类型参数，添加kind过滤条件
	if len(kind) > 0 && kind[0] != "" {
		query += " AND kind = ?"
		args = append(args, kind[0])
	}

	query += " ORDER BY open_time DESC LIMIT 1"

	// 类型断言获取*gorm.DB实例
	gormDB, ok := db.(*gorm.DB)
	if !ok {
		log.Printf("[BasePriceCache] 数据库类型错误，无法查询基准价格: %s", symbol)
		return 0
	}

	err := gormDB.Raw(query, args...).Scan(&closePriceStr).Error
	if err != nil {
		log.Printf("[BasePriceCache] 查询基准价格失败 %s: %v", symbol, err)
		return 0
	}

	if closePriceStr == "" {
		log.Printf("[BasePriceCache] 未找到24小时前的价格数据: %s", symbol)
		return 0
	}

	// 转换为float64
	closePrice, err := strconv.ParseFloat(closePriceStr, 64)
	if err != nil {
		log.Printf("[BasePriceCache] 价格数据格式错误 %s: %s", symbol, closePriceStr)
		return 0
	}

	// 移除频繁的基准价格查询成功日志
	return closePrice
}

// cleanupExpiredBasePrices 清理过期的基准价格
func (c *BasePriceCache) cleanupExpiredBasePrices() {
	now := time.Now()
	expired := 0

	for symbol, baseData := range c.basePrices {
		if now.After(baseData.ExpireTime) {
			delete(c.basePrices, symbol)
			expired++
		}
	}

	if expired > 0 {
		log.Printf("[BasePriceCache] 清理过期基准价格: %d个", expired)
	}
}

// startRefreshAndCleanupRoutine 启动刷新和清理例程
func (c *BasePriceCache) startRefreshAndCleanupRoutine() {
	refreshTicker := time.NewTicker(c.refreshInterval)
	cleanupTicker := time.NewTicker(c.cleanupInterval)

	defer refreshTicker.Stop()
	defer cleanupTicker.Stop()

	for {
		select {
		case <-refreshTicker.C:
			c.performRefresh()
		case <-cleanupTicker.C:
			c.performCleanup()
		}
	}
}

// performRefresh 执行刷新操作
func (c *BasePriceCache) performRefresh() {
	startTime := time.Now()
	log.Printf("[BasePriceCache] 开始定时刷新基准价格...")

	c.refreshStats.mu.Lock()
	c.refreshStats.totalRefreshes++
	c.refreshStats.mu.Unlock()

	// 检查数据库连接
	if c.db == nil {
		log.Printf("[BasePriceCache] 数据库连接未设置，跳过刷新")
		c.updateRefreshStats(false, time.Since(startTime), fmt.Errorf("数据库连接未设置"))
		return
	}

	// 执行刷新
	err := c.RefreshBasePrices(c.db)
	if err != nil {
		log.Printf("[BasePriceCache] 刷新失败: %v", err)
		c.updateRefreshStats(false, time.Since(startTime), err)
		return
	}

	refreshTime := time.Since(startTime)
	log.Printf("[BasePriceCache] 刷新完成，耗时: %v", refreshTime)

	c.mu.Lock()
	c.lastRefreshTime = time.Now()
	c.mu.Unlock()

	c.updateRefreshStats(true, refreshTime, nil)
}

// performCleanup 执行清理操作
func (c *BasePriceCache) performCleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.cleanupExpiredBasePrices()
}

// updateRefreshStats 更新刷新统计
func (c *BasePriceCache) updateRefreshStats(success bool, duration time.Duration, err error) {
	c.refreshStats.mu.Lock()
	defer c.refreshStats.mu.Unlock()

	if success {
		c.refreshStats.successfulRefreshes++
	} else {
		c.refreshStats.failedRefreshes++
		c.refreshStats.lastError = err
		c.refreshStats.lastErrorTime = time.Now()
	}

	// 计算平均刷新时间（指数移动平均）
	if c.refreshStats.totalRefreshes == 1 {
		c.refreshStats.avgRefreshTime = duration
	} else {
		c.refreshStats.avgRefreshTime = (c.refreshStats.avgRefreshTime + duration) / 2
	}
}

// GetBasePriceCacheStats 获取基准价格缓存统计信息
func (c *BasePriceCache) GetBasePriceCacheStats() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()

	c.refreshStats.mu.RLock()
	defer c.refreshStats.mu.RUnlock()

	return map[string]interface{}{
		"entries_count":        len(c.basePrices),
		"max_entries":          c.maxEntries,
		"refresh_interval":     c.refreshInterval.String(),
		"cleanup_interval":     c.cleanupInterval.String(),
		"last_refresh_time":    c.lastRefreshTime,
		"total_refreshes":      c.refreshStats.totalRefreshes,
		"successful_refreshes": c.refreshStats.successfulRefreshes,
		"failed_refreshes":     c.refreshStats.failedRefreshes,
		"avg_refresh_time":     c.refreshStats.avgRefreshTime.String(),
		"last_error":           fmt.Sprintf("%v", c.refreshStats.lastError),
		"last_error_time":      c.refreshStats.lastErrorTime,
	}
}
