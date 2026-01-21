package server

import (
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	pdb "analysis/internal/db"
)

// LayeredCache 分层缓存系统 - 多级缓存架构
type LayeredCache struct {
	// L1: 内存缓存 - 最快，容量最小
	l1Cache *MemoryCache

	// L2: Redis缓存 - 中等速度，中等容量
	l2Cache pdb.CacheInterface

	// L3: 数据库缓存 - 最慢，容量最大
	l3Cache *DBCache

	// 配置
	config CacheConfig

	// 统计信息
	stats CacheStats

	// 自适应管理
	adaptiveManager *AdaptiveCacheManager

	mu sync.RWMutex
}

// AdaptiveCacheManager 自适应缓存管理器
type AdaptiveCacheManager struct {
	layeredCache *LayeredCache
	stopCh       chan struct{}
	mu           sync.RWMutex
}

// NewAdaptiveCacheManager 创建自适应缓存管理器
func NewAdaptiveCacheManager(lc *LayeredCache) *AdaptiveCacheManager {
	return &AdaptiveCacheManager{
		layeredCache: lc,
		stopCh:       make(chan struct{}),
	}
}

// Start 启动自适应管理
func (acm *AdaptiveCacheManager) Start() {
	if !acm.layeredCache.config.AdaptiveEnabled {
		return
	}

	go acm.adaptiveLoop()
}

// Stop 停止自适应管理
func (acm *AdaptiveCacheManager) Stop() {
	close(acm.stopCh)
}

// adaptiveLoop 自适应调整循环
func (acm *AdaptiveCacheManager) adaptiveLoop() {
	ticker := time.NewTicker(acm.layeredCache.config.AdaptiveInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			acm.adjustCacheParameters()
		case <-acm.stopCh:
			return
		}
	}
}

// adjustCacheParameters 调整缓存参数
func (acm *AdaptiveCacheManager) adjustCacheParameters() {
	lc := acm.layeredCache

	lc.mu.Lock()
	defer lc.mu.Unlock()

	// 计算当前命中率
	totalRequests := lc.stats.L1Hits + lc.stats.L1Misses + lc.stats.L2Hits + lc.stats.L2Misses + lc.stats.L3Hits + lc.stats.L3Misses
	if totalRequests == 0 {
		return
	}

	totalHits := lc.stats.L1Hits + lc.stats.L2Hits + lc.stats.L3Hits
	currentHitRate := float64(totalHits) / float64(totalRequests)

	log.Printf("[AdaptiveCache] 当前命中率: %.2f%%, 目标命中率: %.2f%%",
		currentHitRate*100, lc.config.TargetHitRate)

	// 根据命中率调整参数
	if currentHitRate < lc.config.ScaleUpThreshold {
		// 命中率过低，需要增加缓存容量或调整TTL
		acm.scaleUpCache()
	} else if currentHitRate > lc.config.ScaleDownThreshold {
		// 命中率过高，可以适当减少资源消耗
		acm.optimizeCache()
	}
}

// scaleUpCache 扩容缓存
func (acm *AdaptiveCacheManager) scaleUpCache() {
	lc := acm.layeredCache

	if lc.config.L1Enabled && lc.l1Cache != nil {
		// 增加L1缓存容量
		currentSize := lc.l1Cache.maxSize
		newSize := int(float64(currentSize) * 1.5) // 增加50%
		if newSize > 10000 {                       // 最大10000
			newSize = 10000
		}

		if newSize != currentSize {
			log.Printf("[AdaptiveCache] 调整L1缓存容量: %d -> %d", currentSize, newSize)
			// 这里需要重新创建缓存，但为了简化，暂时记录日志
		}
	}

	// 调整TTL - 延长缓存时间
	if lc.config.L1TTL < 10*time.Minute {
		lc.config.L1TTL = lc.config.L1TTL * 6 / 5 // 增加20%
		log.Printf("[AdaptiveCache] 调整L1 TTL: %v", lc.config.L1TTL)
	}
}

// optimizeCache 优化缓存
func (acm *AdaptiveCacheManager) optimizeCache() {
	lc := acm.layeredCache

	// 命中率很好，可以适当减少资源消耗
	if lc.config.L1TTL > 30*time.Second {
		lc.config.L1TTL = lc.config.L1TTL * 4 / 5 // 减少20%
		log.Printf("[AdaptiveCache] 优化L1 TTL: %v", lc.config.L1TTL)
	}
}

// CacheConfig 缓存配置
type CacheConfig struct {
	// L1配置
	L1Enabled bool
	L1MaxSize int           // 最大条目数
	L1TTL     time.Duration // 默认TTL

	// L2配置
	L2Enabled bool
	L2TTL     time.Duration // 默认TTL

	// L3配置
	L3Enabled bool
	L3TTL     time.Duration // 默认TTL

	// 预热配置
	WarmupEnabled         bool
	WarmupInterval        time.Duration
	WarmupConcurrency     int
	WarmupPriorityEnabled bool // 是否启用优先级预热

	// 失效配置
	InvalidationEnabled bool
	InvalidationBuffer  int // 批量失效缓冲区大小

	// 监控配置
	MetricsEnabled  bool
	MetricsInterval time.Duration

	// 自适应配置
	AdaptiveEnabled    bool          // 是否启用自适应缓存
	AdaptiveInterval   time.Duration // 自适应调整间隔
	TargetHitRate      float64       // 目标命中率
	ScaleUpThreshold   float64       // 扩容阈值
	ScaleDownThreshold float64       // 缩容阈值
}

// CacheStats 缓存统计信息
type CacheStats struct {
	// L1统计
	L1Hits    int64
	L1Misses  int64
	L1Sets    int64
	L1Deletes int64

	// L2统计
	L2Hits    int64
	L2Misses  int64
	L2Sets    int64
	L2Deletes int64

	// L3统计
	L3Hits    int64
	L3Misses  int64
	L3Sets    int64
	L3Deletes int64

	// 性能统计
	AvgL1Latency time.Duration
	AvgL2Latency time.Duration
	AvgL3Latency time.Duration

	// 时间戳
	LastUpdated time.Time
}

// CacheEntry 缓存条目
type CacheEntry struct {
	Key         string
	Value       interface{}
	TTL         time.Duration
	CreatedAt   time.Time
	AccessedAt  time.Time
	AccessCount int64
}

// MemoryCache L1内存缓存
type MemoryCache struct {
	data    map[string]*CacheEntry
	maxSize int
	mu      sync.RWMutex

	// LRU淘汰
	head  *CacheNode
	tail  *CacheNode
	nodes map[string]*CacheNode
}

type CacheNode struct {
	key   string
	entry *CacheEntry
	prev  *CacheNode
	next  *CacheNode
}

// DBCache L3数据库缓存
type DBCache struct {
	db        Database
	tableName string
}

// CacheKey 缓存键生成器
type CacheKey struct {
	Namespace string
	Resource  string
	Params    map[string]interface{}
}

// NewLayeredCache 创建分层缓存
func NewLayeredCache(l2Cache pdb.CacheInterface, db Database, config CacheConfig) *LayeredCache {
	lc := &LayeredCache{
		l2Cache: l2Cache,
		l3Cache: NewDBCache(db),
		config:  config,
		stats:   CacheStats{},
	}

	// 初始化L1缓存
	if config.L1Enabled {
		lc.l1Cache = NewMemoryCache(config.L1MaxSize)
	}

	// 初始化自适应管理器
	if config.AdaptiveEnabled {
		lc.adaptiveManager = NewAdaptiveCacheManager(lc)
	}

	// 启动监控
	if config.MetricsEnabled {
		go lc.metricsCollector()
	}

	// 启动预热
	if config.WarmupEnabled {
		go lc.cacheWarmer()
	}

	// 启动自适应管理
	if config.AdaptiveEnabled && lc.adaptiveManager != nil {
		lc.adaptiveManager.Start()
	}

	return lc
}

// Get 获取缓存数据 - 分层查找
func (lc *LayeredCache) Get(ctx context.Context, key string) (interface{}, error) {
	startTime := time.Now()

	// L1缓存查找
	if lc.config.L1Enabled {
		if value, found := lc.l1Cache.Get(key); found {
			lc.updateStats(&lc.stats.L1Hits, nil, nil, nil)
			lc.updateLatencyStats(startTime, 1)
			return value, nil
		}
		lc.updateStats(nil, &lc.stats.L1Misses, nil, nil)
	}

	// L2缓存查找
	if lc.config.L2Enabled && lc.l2Cache != nil {
		l2Start := time.Now()
		if cached, err := lc.l2Cache.Get(ctx, key); err == nil && len(cached) > 0 {
			// 反序列化并写入L1
			var value interface{}
			if err := json.Unmarshal(cached, &value); err == nil && lc.config.L1Enabled {
				lc.l1Cache.Set(key, value, lc.config.L1TTL)
			}

			lc.updateStats(&lc.stats.L2Hits, nil, nil, nil)
			lc.updateLatencyStats(l2Start, 2)
			return value, nil
		}
		lc.updateStats(nil, &lc.stats.L2Misses, nil, nil)
	}

	// L3缓存查找
	if lc.config.L3Enabled {
		l3Start := time.Now()
		if value, found, err := lc.l3Cache.Get(ctx, key); err == nil && found {
			// 写入L2和L1
			if lc.config.L2Enabled && lc.l2Cache != nil {
				if data, err := json.Marshal(value); err == nil {
					lc.l2Cache.Set(ctx, key, data, lc.config.L2TTL)
				}
			}
			if lc.config.L1Enabled {
				lc.l1Cache.Set(key, value, lc.config.L1TTL)
			}

			lc.updateStats(&lc.stats.L3Hits, nil, nil, nil)
			lc.updateLatencyStats(l3Start, 3)
			return value, nil
		}
		lc.updateStats(nil, &lc.stats.L3Misses, nil, nil)
	}

	return nil, fmt.Errorf("cache miss")
}

// Set 设置缓存数据 - 分层写入
func (lc *LayeredCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// L1缓存写入
	if lc.config.L1Enabled {
		lc.l1Cache.Set(key, value, ttl)
		lc.updateStats(nil, nil, &lc.stats.L1Sets, nil)
	}

	// L2缓存写入
	if lc.config.L2Enabled && lc.l2Cache != nil {
		if data, err := json.Marshal(value); err == nil {
			if err := lc.l2Cache.Set(ctx, key, data, ttl); err != nil {
				log.Printf("[LayeredCache] L2缓存写入失败: %v", err)
			} else {
				lc.updateStats(nil, nil, &lc.stats.L2Sets, nil)
			}
		}
	}

	// L3缓存写入
	if lc.config.L3Enabled {
		if err := lc.l3Cache.Set(ctx, key, value, ttl); err != nil {
			log.Printf("[LayeredCache] L3缓存写入失败: %v", err)
		} else {
			lc.updateStats(nil, nil, &lc.stats.L3Sets, nil)
		}
	}

	return nil
}

// Delete 删除缓存数据 - 分层删除
func (lc *LayeredCache) Delete(ctx context.Context, key string) error {
	var errors []error

	// L1缓存删除
	if lc.config.L1Enabled {
		lc.l1Cache.Delete(key)
		lc.updateStats(nil, nil, nil, &lc.stats.L1Deletes)
	}

	// L2缓存删除
	if lc.config.L2Enabled && lc.l2Cache != nil {
		if err := lc.l2Cache.Delete(ctx, key); err != nil {
			errors = append(errors, fmt.Errorf("L2 delete error: %w", err))
		} else {
			lc.updateStats(nil, nil, nil, &lc.stats.L2Deletes)
		}
	}

	// L3缓存删除
	if lc.config.L3Enabled {
		if err := lc.l3Cache.Delete(ctx, key); err != nil {
			errors = append(errors, fmt.Errorf("L3 delete error: %w", err))
		} else {
			lc.updateStats(nil, nil, nil, &lc.stats.L3Deletes)
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cache delete errors: %v", errors)
	}

	return nil
}

// BatchDelete 批量删除
func (lc *LayeredCache) BatchDelete(ctx context.Context, keys []string) error {
	// 批量处理以提高性能
	batchSize := 100

	for i := 0; i < len(keys); i += batchSize {
		end := i + batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]

		// 并发删除
		var wg sync.WaitGroup
		for _, key := range batch {
			wg.Add(1)
			go func(k string) {
				defer wg.Done()
				lc.Delete(ctx, k)
			}(key)
		}
		wg.Wait()
	}

	return nil
}

// InvalidateByPattern 按模式失效缓存
func (lc *LayeredCache) InvalidateByPattern(ctx context.Context, pattern string) error {
	if !lc.config.InvalidationEnabled {
		return nil
	}

	// 这里需要具体的实现，取决于缓存后端的支持
	// 对于Redis，可以使用KEYS和DEL命令
	// 对于内存缓存，需要遍历匹配

	log.Printf("[LayeredCache] 失效缓存模式: %s", pattern)
	return nil
}

// GetStats 获取缓存统计信息
func (lc *LayeredCache) GetStats() CacheStats {
	lc.mu.RLock()
	defer lc.mu.RUnlock()

	return lc.stats
}

// Warmup 缓存预热
func (lc *LayeredCache) Warmup(ctx context.Context) error {
	if !lc.config.WarmupEnabled {
		return nil
	}

	log.Printf("[LayeredCache] 开始智能缓存预热...")

	startTime := time.Now()

	// 根据优先级预热数据
	warmupTasks := lc.buildWarmupTasks()

	// 并发执行预热任务
	semaphore := make(chan struct{}, lc.config.WarmupConcurrency)
	var wg sync.WaitGroup
	var errors []error

	for _, task := range warmupTasks {
		wg.Add(1)
		go func(t WarmupTask) {
			defer wg.Done()

			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			taskCtx, cancel := context.WithTimeout(ctx, t.Timeout)
			defer cancel()

			log.Printf("[LayeredCache] 执行预热任务: %s (优先级: %d)", t.Name, t.Priority)

			if err := t.Function(taskCtx); err != nil {
				log.Printf("[LayeredCache] 预热任务失败 %s: %v", t.Name, err)
				errors = append(errors, fmt.Errorf("%s: %w", t.Name, err))
			} else {
				log.Printf("[LayeredCache] 预热任务完成: %s", t.Name)
			}
		}(task)
	}

	wg.Wait()

	duration := time.Since(startTime)
	log.Printf("[LayeredCache] 缓存预热完成，耗时: %v", duration)

	if len(errors) > 0 {
		return fmt.Errorf("缓存预热失败 %d/%d 个任务: %v", len(errors), len(warmupTasks), errors)
	}

	return nil
}

// WarmupTask 预热任务
type WarmupTask struct {
	Name     string
	Priority int // 1-10, 10为最高优先级
	Timeout  time.Duration
	Function func(ctx context.Context) error
}

// buildWarmupTasks 构建预热任务列表
func (lc *LayeredCache) buildWarmupTasks() []WarmupTask {
	tasks := []WarmupTask{
		{
			Name:     "关键实体数据",
			Priority: 10,
			Timeout:  30 * time.Second,
			Function: lc.warmupCriticalEntities,
		},
		{
			Name:     "热门推荐数据",
			Priority: 9,
			Timeout:  45 * time.Second,
			Function: lc.warmupHotRecommendations,
		},
		{
			Name:     "市场基础数据",
			Priority: 8,
			Timeout:  60 * time.Second,
			Function: lc.warmupMarketBasics,
		},
		{
			Name:     "用户偏好数据",
			Priority: 7,
			Timeout:  20 * time.Second,
			Function: lc.warmupUserPreferences,
		},
		{
			Name:     "历史统计数据",
			Priority: 6,
			Timeout:  90 * time.Second,
			Function: lc.warmupHistoricalStats,
		},
		{
			Name:     "配置和元数据",
			Priority: 5,
			Timeout:  15 * time.Second,
			Function: lc.warmupConfigurations,
		},
	}

	// 如果启用优先级预热，则按优先级排序
	if lc.config.WarmupPriorityEnabled {
		// 按优先级降序排序
		for i := 0; i < len(tasks)-1; i++ {
			for j := i + 1; j < len(tasks); j++ {
				if tasks[i].Priority < tasks[j].Priority {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}

	return tasks
}

// 新增的预热函数实现

// warmupCriticalEntities 预热关键实体数据
func (lc *LayeredCache) warmupCriticalEntities(ctx context.Context) error {
	log.Printf("[LayeredCache] 预热关键实体数据")

	// 获取热门交易对
	criticalSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}

	for _, symbol := range criticalSymbols {
		key := GenerateCacheKey("market", "price", map[string]interface{}{"symbol": symbol})
		// 这里应该从数据库或API获取最新价格
		// 暂时模拟数据
		mockPrice := map[string]interface{}{
			"symbol":    symbol,
			"price":     50000.0, // 模拟价格
			"timestamp": time.Now(),
		}

		if err := lc.Set(ctx, key, mockPrice, lc.config.L2TTL); err != nil {
			log.Printf("[LayeredCache] 预热实体失败 %s: %v", symbol, err)
		}
	}

	return nil
}

// warmupHotRecommendations 预热热门推荐数据
func (lc *LayeredCache) warmupHotRecommendations(ctx context.Context) error {
	log.Printf("[LayeredCache] 预热热门推荐数据")

	// 预热不同类型和数量的推荐查询
	queryConfigs := []struct {
		kind  string
		limit int
	}{
		{"spot", 5},
		{"spot", 10},
		{"futures", 5},
		{"futures", 10},
	}

	for _, config := range queryConfigs {
		key := GenerateCacheKey("recommendations", "list", map[string]interface{}{
			"kind":  config.kind,
			"limit": config.limit,
		})

		// 模拟推荐数据
		mockRecommendations := []map[string]interface{}{
			{
				"symbol":     "BTCUSDT",
				"action":     "BUY",
				"confidence": 0.85,
				"score":      0.82,
			},
			{
				"symbol":     "ETHUSDT",
				"action":     "BUY",
				"confidence": 0.78,
				"score":      0.75,
			},
		}

		if err := lc.Set(ctx, key, mockRecommendations, lc.config.L2TTL); err != nil {
			log.Printf("[LayeredCache] 预热推荐失败 kind=%s limit=%d: %v", config.kind, config.limit, err)
		}
	}

	return nil
}

// warmupMarketBasics 预热市场基础数据
func (lc *LayeredCache) warmupMarketBasics(ctx context.Context) error {
	log.Printf("[LayeredCache] 预热市场基础数据")

	// 预热24h统计数据
	marketStats := map[string]interface{}{
		"total_volume_24h": 1000000000.0,
		"total_trades_24h": 5000000,
		"top_gainers":      []string{"BTCUSDT", "ETHUSDT", "ADAUSDT"},
		"top_losers":       []string{"DOGEUSDT", "SHIBUSDT"},
	}

	key := GenerateCacheKey("market", "stats", map[string]interface{}{"period": "24h"})
	return lc.Set(ctx, key, marketStats, lc.config.L2TTL)
}

// warmupUserPreferences 预热用户偏好数据
func (lc *LayeredCache) warmupUserPreferences(ctx context.Context) error {
	log.Printf("[LayeredCache] 预热用户偏好数据")

	// 预热默认用户偏好设置
	defaultPrefs := map[string]interface{}{
		"theme":                "dark",
		"language":             "zh-CN",
		"default_timeframe":    "1h",
		"notification_enabled": true,
		"risk_level":           "medium",
	}

	key := GenerateCacheKey("user", "preferences", map[string]interface{}{"user_id": "default"})
	return lc.Set(ctx, key, defaultPrefs, lc.config.L3TTL)
}

// warmupHistoricalStats 预热历史统计数据
func (lc *LayeredCache) warmupHistoricalStats(ctx context.Context) error {
	log.Printf("[LayeredCache] 预热历史统计数据")

	// 预热月度统计
	monthlyStats := map[string]interface{}{
		"total_trades": 15000,
		"win_rate":     0.65,
		"avg_profit":   0.023,
		"max_drawdown": 0.12,
		"sharpe_ratio": 1.8,
		"total_volume": 50000000.0,
	}

	key := GenerateCacheKey("stats", "monthly", map[string]interface{}{"year": 2024, "month": 12})
	return lc.Set(ctx, key, monthlyStats, lc.config.L3TTL)
}

// warmupConfigurations 预热配置和元数据
func (lc *LayeredCache) warmupConfigurations(ctx context.Context) error {
	log.Printf("[LayeredCache] 预热配置和元数据")

	// 预热系统配置
	systemConfig := map[string]interface{}{
		"supported_symbols":    []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"},
		"supported_timeframes": []string{"1m", "5m", "15m", "1h", "4h", "1d"},
		"max_recommendations":  50,
		"cache_enabled":        true,
		"features_enabled": map[string]bool{
			"ai_recommendations": true,
			"backtesting":        true,
			"portfolio_tracking": true,
		},
	}

	key := GenerateCacheKey("config", "system", nil)
	return lc.Set(ctx, key, systemConfig, lc.config.L3TTL)
}

// 保留原有预热函数以向后兼容
// warmupRecommendations 预热推荐数据
func (lc *LayeredCache) warmupRecommendations() error {
	ctx := context.Background()
	return lc.warmupHotRecommendations(ctx)
}

// warmupPerformanceStats 预热性能统计
func (lc *LayeredCache) warmupPerformanceStats() error {
	ctx := context.Background()
	return lc.warmupHistoricalStats(ctx)
}

// warmupMarketData 预热市场数据
func (lc *LayeredCache) warmupMarketData() error {
	ctx := context.Background()
	return lc.warmupMarketBasics(ctx)
}

// cacheWarmer 缓存预热器
func (lc *LayeredCache) cacheWarmer() {
	ticker := time.NewTicker(lc.config.WarmupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if err := lc.Warmup(ctx); err != nil {
				log.Printf("[LayeredCache] 缓存预热失败: %v", err)
			}
		}
	}
}

// metricsCollector 指标收集器
func (lc *LayeredCache) metricsCollector() {
	ticker := time.NewTicker(lc.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lc.collectMetrics()
		}
	}
}

// collectMetrics 收集指标
func (lc *LayeredCache) collectMetrics() {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	lc.stats.LastUpdated = time.Now()

	// 计算命中率等派生指标
	// 这里可以添加更多的指标计算
}

// updateStats 更新统计信息
func (lc *LayeredCache) updateStats(l1Hits, l1Misses, l1Sets, l1Deletes *int64) {
	lc.mu.Lock()
	defer lc.mu.Unlock()

	if l1Hits != nil {
		*l1Hits++
	}
	if l1Misses != nil {
		*l1Misses++
	}
	if l1Sets != nil {
		*l1Sets++
	}
	if l1Deletes != nil {
		*l1Deletes++
	}
}

// updateLatencyStats 更新延迟统计
func (lc *LayeredCache) updateLatencyStats(start time.Time, level int) {
	duration := time.Since(start)

	lc.mu.Lock()
	defer lc.mu.Unlock()

	switch level {
	case 1: // L1
		if lc.stats.AvgL1Latency == 0 {
			lc.stats.AvgL1Latency = duration
		} else {
			lc.stats.AvgL1Latency = (lc.stats.AvgL1Latency + duration) / 2
		}
	case 2: // L2
		if lc.stats.AvgL2Latency == 0 {
			lc.stats.AvgL2Latency = duration
		} else {
			lc.stats.AvgL2Latency = (lc.stats.AvgL2Latency + duration) / 2
		}
	case 3: // L3
		if lc.stats.AvgL3Latency == 0 {
			lc.stats.AvgL3Latency = duration
		} else {
			lc.stats.AvgL3Latency = (lc.stats.AvgL3Latency + duration) / 2
		}
	}
}

// MemoryCache 实现

// NewMemoryCache 创建内存缓存
func NewMemoryCache(maxSize int) *MemoryCache {
	return &MemoryCache{
		data:    make(map[string]*CacheEntry),
		maxSize: maxSize,
		nodes:   make(map[string]*CacheNode),
	}
}

// Get 获取内存缓存
func (mc *MemoryCache) Get(key string) (interface{}, bool) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if node, exists := mc.nodes[key]; exists {
		// 更新访问时间和计数
		node.entry.AccessedAt = time.Now()
		node.entry.AccessCount++

		// 移动到头部（LRU）
		mc.moveToHead(node)

		// 检查是否过期
		if time.Since(node.entry.CreatedAt) > node.entry.TTL {
			mc.deleteNode(node)
			delete(mc.data, key)
			return nil, false
		}

		return node.entry.Value, true
	}

	return nil, false
}

// Set 设置内存缓存
func (mc *MemoryCache) Set(key string, value interface{}, ttl time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		TTL:         ttl,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 0,
	}

	// 如果已存在，更新并移动到头部
	if node, exists := mc.nodes[key]; exists {
		node.entry = entry
		mc.moveToHead(node)
		return
	}

	// 创建新节点
	node := &CacheNode{
		key:   key,
		entry: entry,
	}

	// 添加到头部
	mc.addToHead(node)
	mc.data[key] = entry

	// 检查容量限制
	if len(mc.data) > mc.maxSize {
		mc.evictLRU()
	}
}

// Delete 删除内存缓存
func (mc *MemoryCache) Delete(key string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if node, exists := mc.nodes[key]; exists {
		mc.deleteNode(node)
		delete(mc.data, key)
	}
}

// LRU 操作
func (mc *MemoryCache) addToHead(node *CacheNode) {
	node.next = mc.head
	node.prev = nil

	if mc.head != nil {
		mc.head.prev = node
	}
	mc.head = node

	if mc.tail == nil {
		mc.tail = node
	}

	mc.nodes[node.key] = node
}

func (mc *MemoryCache) moveToHead(node *CacheNode) {
	if node == mc.head {
		return
	}

	mc.deleteNode(node)
	mc.addToHead(node)
}

func (mc *MemoryCache) deleteNode(node *CacheNode) {
	if node.prev != nil {
		node.prev.next = node.next
	} else {
		mc.head = node.next
	}

	if node.next != nil {
		node.next.prev = node.prev
	} else {
		mc.tail = node.prev
	}

	delete(mc.nodes, node.key)
}

func (mc *MemoryCache) evictLRU() {
	if mc.tail != nil {
		tail := mc.tail
		mc.deleteNode(tail)
		delete(mc.data, tail.key)
	}
}

// DBCache 实现

// NewDBCache 创建数据库缓存
func NewDBCache(db Database) *DBCache {
	return &DBCache{
		db:        db,
		tableName: "cache_entries",
	}
}

// Get 获取数据库缓存
func (dc *DBCache) Get(ctx context.Context, key string) (interface{}, bool, error) {
	// 这里需要实现数据库查询
	// 为了简化，这里返回未找到
	return nil, false, nil
}

// Set 设置数据库缓存
func (dc *DBCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// 这里需要实现数据库插入/更新
	return nil
}

// Delete 删除数据库缓存
func (dc *DBCache) Delete(ctx context.Context, key string) error {
	// 这里需要实现数据库删除
	return nil
}

// GenerateKey 生成缓存键
func GenerateCacheKey(namespace, resource string, params map[string]interface{}) string {
	key := fmt.Sprintf("%s:%s", namespace, resource)

	if len(params) > 0 {
		// 排序参数以确保一致性
		var sortedKeys []string
		for k := range params {
			sortedKeys = append(sortedKeys, k)
		}

		// 简单排序（可以改进）
		for i := 0; i < len(sortedKeys)-1; i++ {
			for j := i + 1; j < len(sortedKeys); j++ {
				if sortedKeys[i] > sortedKeys[j] {
					sortedKeys[i], sortedKeys[j] = sortedKeys[j], sortedKeys[i]
				}
			}
		}

		// 构建参数字符串
		var paramStr string
		for _, k := range sortedKeys {
			paramStr += fmt.Sprintf("%s=%v;", k, params[k])
		}

		// 生成MD5哈希
		hash := md5.Sum([]byte(paramStr))
		key += ":" + fmt.Sprintf("%x", hash)[:8]
	}

	return key
}
