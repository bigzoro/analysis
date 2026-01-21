package main

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"

	"gorm.io/gorm"
)

// ===== 错误处理增强 =====

// ErrorHandler 错误处理器
type ErrorHandler struct {
	mu                sync.RWMutex
	consecutiveErrors int64         // 连续错误次数
	totalErrors       int64         // 总错误次数
	lastErrorTime     time.Time     // 最后错误时间
	errorHistory      []ErrorRecord // 错误历史记录
	maxHistorySize    int           // 最大历史记录数
}

// ErrorRecord 错误记录
type ErrorRecord struct {
	Timestamp time.Time
	Error     error
	Operation string
	Retryable bool
}

// RetryConfig 重试配置
type RetryConfig struct {
	MaxRetries    int           // 最大重试次数
	BaseDelay     time.Duration // 基础延迟
	MaxDelay      time.Duration // 最大延迟
	BackoffFactor float64       // 退避因子
}

// NewErrorHandler 创建错误处理器
func NewErrorHandler() *ErrorHandler {
	return &ErrorHandler{
		errorHistory:   make([]ErrorRecord, 0, 50),
		maxHistorySize: 50,
	}
}

// RecordError 记录错误
func (h *ErrorHandler) RecordError(err error, operation string, retryable bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	atomic.AddInt64(&h.consecutiveErrors, 1)
	atomic.AddInt64(&h.totalErrors, 1)
	h.lastErrorTime = time.Now()

	record := ErrorRecord{
		Timestamp: time.Now(),
		Error:     err,
		Operation: operation,
		Retryable: retryable,
	}

	h.errorHistory = append(h.errorHistory, record)
	if len(h.errorHistory) > h.maxHistorySize {
		h.errorHistory = h.errorHistory[1:]
	}

	consecutive := atomic.LoadInt64(&h.consecutiveErrors)
	total := atomic.LoadInt64(&h.totalErrors)

	// 根据错误严重程度输出不同级别的日志
	if consecutive >= 5 {
		log.Printf("[ErrorHandler] 🚨 严重错误 - 操作:%s, 连续失败:%d次, 总失败:%d次, 错误:%v",
			operation, consecutive, total, err)
	} else if consecutive >= 3 {
		log.Printf("[ErrorHandler] ⚠️ 重复错误 - 操作:%s, 连续失败:%d次, 可重试:%v, 错误:%v",
			operation, consecutive, retryable, err)
	} else {
		log.Printf("[ErrorHandler] ❌ 操作失败 - %s: %v (可重试:%v)", operation, err, retryable)
	}

	// 如果是不可重试的错误或连续失败太多，记录警告
	if !retryable || consecutive >= 10 {
		log.Printf("[ErrorHandler] 🔴 错误处理建议 - 操作:%s 需要人工干预，连续失败:%d次", operation, consecutive)
	}
}

// RecordSuccess 记录成功，重置连续错误计数
func (h *ErrorHandler) RecordSuccess() {
	atomic.StoreInt64(&h.consecutiveErrors, 0)
}

// ShouldRetry 判断是否应该重试
func (h *ErrorHandler) ShouldRetry(retryCount int, config RetryConfig) bool {
	if retryCount >= config.MaxRetries {
		return false
	}

	consecutiveErrors := atomic.LoadInt64(&h.consecutiveErrors)
	// 如果连续错误太多，停止重试
	if consecutiveErrors > 10 {
		return false
	}

	return true
}

// CalculateRetryDelay 计算重试延迟
func (h *ErrorHandler) CalculateRetryDelay(retryCount int, config RetryConfig) time.Duration {
	delay := time.Duration(float64(config.BaseDelay) * math.Pow(config.BackoffFactor, float64(retryCount)))
	if delay > config.MaxDelay {
		delay = config.MaxDelay
	}
	return delay
}

// GetErrorStats 获取错误统计
func (h *ErrorHandler) GetErrorStats() map[string]interface{} {
	h.mu.RLock()
	defer h.mu.RUnlock()

	return map[string]interface{}{
		"consecutive_errors": atomic.LoadInt64(&h.consecutiveErrors),
		"total_errors":       atomic.LoadInt64(&h.totalErrors),
		"last_error_time":    h.lastErrorTime,
		"error_history_size": len(h.errorHistory),
	}
}

// ===== 实时涨幅榜同步器 =====
// 基于WebSocket实时驱动的涨幅榜系统，实现秒级更新的市场涨幅数据
// 替代原有的定期同步gainers_history_syncer

// RealtimeGainersSyncer 实时涨幅榜同步器
// 实现 DataSyncer 接口，支持持续运行的实时同步
type RealtimeGainersSyncer struct {
	// 基础配置
	db     *gorm.DB
	cfg    *config.Config
	config *DataSyncConfig

	// 核心配置
	topSymbolsCount int           // 跟踪的热门交易对数量
	kind            string        // 市场类型："spot" 或 "futures"
	updateInterval  time.Duration // 数据更新间隔

	// WebSocket管理
	wsManager       *RealtimeWSManager
	priceUpdateChan chan PriceUpdate // 价格更新通道

	// 数据缓存系统
	priceCache        *RealtimePriceCache  // 实时价格缓存
	basePriceCache    *BasePriceCache      // 24h基准价格缓存
	currentGainers    []RealtimeGainerItem // 当前涨幅榜状态
	currentGainersMux sync.RWMutex         // 涨幅榜读写锁

	// 控制组件
	changeDetector  *ChangeDetector  // 变化检测器
	saveController  *SaveController  // 保存控制器
	snapshotManager *SnapshotManager // 快照管理器

	// 统计监控
	stats     *RealtimeStats // 运行统计
	startTime time.Time      // 启动时间

	// 错误处理增强
	errorHandler *ErrorHandler // 错误处理器
	retryConfig  RetryConfig   // 重试配置

	// 控制信号
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// RealtimeGainerItem 实时涨幅榜项目
type RealtimeGainerItem struct {
	Symbol        string    `json:"symbol"`
	Rank          int       `json:"rank"`
	CurrentPrice  float64   `json:"current_price"`
	ChangePercent float64   `json:"change_percent"`
	Volume24h     float64   `json:"volume_24h"`
	DataSource    string    `json:"data_source"`
	Timestamp     time.Time `json:"timestamp"`
}

// PriceUpdate 价格更新消息
type PriceUpdate struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Volume        float64   `json:"volume"`
	ChangePercent *float64  `json:"change_percent,omitempty"` // 24h涨跌幅百分比，nil表示未设置
	Timestamp     time.Time `json:"timestamp"`
	Source        string    `json:"source"` // "websocket" 或 "http"
}

// RealtimeStats 实时同步器统计信息
type RealtimeStats struct {
	mu sync.RWMutex

	// 连接统计（原子操作）
	activeWSConnections int64 // 原子操作
	totalWSReconnects   int64 // 原子操作

	// 数据处理统计（原子操作）
	priceUpdatesReceived int64 // 原子操作
	gainersCalculations  int64 // 原子操作
	savesTriggered       int64 // 原子操作

	// 性能统计
	avgCalculationTime time.Duration
	avgSaveTime        time.Duration
	cacheHitRate       float64

	// 查询性能统计
	totalQueries int64 // 原子操作
	slowQueries  int64 // 原子操作，超过100ms的查询
	avgQueryTime time.Duration

	// 错误统计（部分原子操作）
	errorsCount   int64 // 原子操作
	lastError     error
	lastErrorTime time.Time

	// 运行状态
	isRunning      bool
	lastUpdateTime time.Time
}

// NewRealtimeGainersSyncerWithKind 创建指定市场类型的实时涨幅榜同步器
func NewRealtimeGainersSyncerWithKind(db *gorm.DB, cfg *config.Config, config *DataSyncConfig, kind string) *RealtimeGainersSyncer {
	ctx, cancel := context.WithCancel(context.Background())

	syncer := &RealtimeGainersSyncer{
		db:              db,
		cfg:             cfg,
		config:          config,
		topSymbolsCount: 15,                           // 默认跟踪15个交易对
		kind:            kind,                         // 指定市场类型
		updateInterval:  5 * time.Second,              // 默认5秒更新间隔
		priceUpdateChan: make(chan PriceUpdate, 1000), // 价格更新通道，带缓冲
		stats:           &RealtimeStats{},
		ctx:             ctx,
		cancel:          cancel,
		startTime:       time.Now(),
	}

	// 初始化各个组件
	syncer.initializeComponents()

	log.Printf("[RealtimeGainersSyncer] 初始化完成 - 跟踪%d个交易对, 市场类型:%s", syncer.topSymbolsCount, syncer.kind)
	return syncer
}

// initializeComponents 初始化各个组件
func (s *RealtimeGainersSyncer) initializeComponents() {
	log.Printf("[RealtimeGainersSyncer] 🔧 开始初始化各个组件...")

	// 初始化错误处理器
	log.Printf("[RealtimeGainersSyncer]   ├── 初始化错误处理器...")
	s.errorHandler = NewErrorHandler()
	s.retryConfig = RetryConfig{
		MaxRetries:    3,
		BaseDelay:     time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
	}
	log.Printf("[RealtimeGainersSyncer]   │   └── 重试配置: 最大重试%d次, 基础延迟%v", s.retryConfig.MaxRetries, s.retryConfig.BaseDelay)

	// 初始化WebSocket管理器
	log.Printf("[RealtimeGainersSyncer]   ├── 初始化WebSocket管理器 (市场:%s)...", s.kind)
	s.wsManager = NewRealtimeWSManager(s.ctx, s.kind)

	// 初始化价格缓存
	log.Printf("[RealtimeGainersSyncer]   ├── 初始化价格缓存...")
	s.priceCache = NewRealtimePriceCache()

	// 初始化基准价格缓存
	log.Printf("[RealtimeGainersSyncer]   ├── 初始化基准价格缓存...")
	s.basePriceCache = NewBasePriceCache()
	// 设置数据库连接以启用自动刷新
	s.basePriceCache.SetDatabase(s.db)

	// 初始化智能变化检测器，只开启价格变化检测
	log.Printf("[RealtimeGainersSyncer]   ├── 初始化变化检测器...")
	changeConfig := &ChangeDetectionConfig{
		EnableRankDetection:               false,            // 关闭排名检测
		EnablePriceDetection:              false,            // 关闭价格检测
		EnablePriceChangePercentDetection: true,             // 开启涨跌幅检测
		EnableVolumeDetection:             false,            // 关闭成交量检测
		RankChangeThreshold:               3,                // 前15名中有3个排名变化算显著
		PriceChangeThreshold:              0.5,              // 价格变化0.5%算显著
		PriceChangePercentThreshold:       0.1,              // 涨跌幅变化0.1%算显著
		VolumeChangeThreshold:             5.0,              // 成交量变化5%算显著
		MinSaveInterval:                   30 * time.Second, // 最少30秒保存一次
		MaxSaveInterval:                   5 * time.Minute,  // 最多5分钟保存一次
	}
	s.changeDetector = NewChangeDetectorWithConfig(changeConfig)
	log.Printf("[RealtimeGainersSyncer]   │   └── 配置: 只检测涨跌幅变化 (阈值:%.1f%%)", changeConfig.PriceChangePercentThreshold)

	// 初始化保存控制器
	log.Printf("[RealtimeGainersSyncer]   ├── 初始化保存控制器 (市场:%s)...", s.kind)
	s.saveController = NewSaveController(s.db, s.kind)

	// 初始化快照管理器
	log.Printf("[RealtimeGainersSyncer]   ├── 初始化快照管理器 (市场:%s)...", s.kind)
	s.snapshotManager = NewSnapshotManager(s.db, s.kind)

	// 初始化统计信息
	s.stats.isRunning = false

	log.Printf("[RealtimeGainersSyncer] ✅ 所有组件初始化完成")
}

// refreshBasePricesForSymbols 为指定的交易对刷新基准价格
func (s *RealtimeGainersSyncer) refreshBasePricesForSymbols(symbols []string) {
	refreshed := 0
	for _, symbol := range symbols {
		basePrice := s.basePriceCache.queryBasePriceFromDB(s.db, symbol, s.kind)
		if basePrice > 0 {
			s.basePriceCache.UpdateBasePrice(symbol, basePrice)
			refreshed++
		}
	}
}

// shouldSavePeriodically 检查是否应该定期保存（每分钟一次）
func (s *RealtimeGainersSyncer) shouldSavePeriodically() bool {
	s.stats.mu.RLock()
	lastSaveTime := s.stats.lastUpdateTime
	s.stats.mu.RUnlock()

	// 如果距离上次保存超过1分钟，则保存
	return time.Since(lastSaveTime) > time.Minute
}

// internalStart 内部启动方法
func (s *RealtimeGainersSyncer) internalStart() error {
	log.Printf("[RealtimeGainersSyncer] 🚀 启动实时涨幅榜同步器 (市场:%s, 跟踪数量:%d)...",
		s.kind, s.topSymbolsCount)

	startTime := time.Now()

	// 标记为运行状态
	s.stats.mu.Lock()
	s.stats.isRunning = true
	s.startTime = time.Now()
	s.stats.mu.Unlock()

	// 在启动goroutine之前，先确保WebSocket订阅是最新的
	log.Printf("[RealtimeGainersSyncer] 📡 初始化WebSocket订阅...")
	s.updateWebSocketSubscriptions()

	// 执行一次手动同步以初始化数据
	log.Printf("[RealtimeGainersSyncer] 🔄 执行初始化数据同步...")
	if err := s.Sync(s.ctx); err != nil {
		log.Printf("[RealtimeGainersSyncer] ⚠️ 初始化同步失败，但继续启动: %v", err)
	}

	// 启动各个goroutine
	s.wg.Add(4)

	log.Printf("[RealtimeGainersSyncer] 🏃 启动后台处理协程...")

	// 1. 启动WebSocket连接管理
	go s.runWebSocketManager()
	log.Printf("[RealtimeGainersSyncer] ✅ WebSocket管理器已启动")

	// 2. 启动价格更新处理
	go s.runPriceUpdateProcessor()
	log.Printf("[RealtimeGainersSyncer] ✅ 价格更新处理器已启动")

	// 3. 启动涨幅榜计算器
	go s.runGainersCalculator()
	log.Printf("[RealtimeGainersSyncer] ✅ 涨幅榜计算器已启动")

	// 4. 启动统计监控
	go s.runStatsReporter()
	log.Printf("[RealtimeGainersSyncer] ✅ 统计报告器已启动")

	initDuration := time.Since(startTime)
	log.Printf("[RealtimeGainersSyncer] 🎉 实时涨幅榜同步器启动成功，耗时: %v", initDuration)
	return nil
}

// internalStop 内部停止方法
func (s *RealtimeGainersSyncer) internalStop() {
	log.Printf("[RealtimeGainersSyncer] 🛑 正在停止实时涨幅榜同步器...")

	stopStartTime := time.Now()

	// 发送停止信号
	log.Printf("[RealtimeGainersSyncer] 📤 发送停止信号到所有协程...")
	s.cancel()

	// 等待所有goroutine完成
	log.Printf("[RealtimeGainersSyncer] ⏳ 等待所有协程完成...")
	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[RealtimeGainersSyncer] ✅ 所有协程已正常停止")
	case <-time.After(30 * time.Second):
		log.Printf("[RealtimeGainersSyncer] ⚠️ 等待协程停止超时，继续清理资源")
	}

	// 清理资源
	log.Printf("[RealtimeGainersSyncer] 🧹 清理资源...")
	close(s.priceUpdateChan)

	// 更新统计信息
	s.stats.mu.Lock()
	s.stats.isRunning = false

	// 记录最终统计信息
	uptime := time.Since(s.startTime)

	log.Printf("[RealtimeGainersSyncer] 📊 运行统计: 运行时间=%v, 价格更新=%d, 计算次数=%d, 保存次数=%d, 错误次数=%d",
		uptime,
		atomic.LoadInt64(&s.stats.priceUpdatesReceived),
		atomic.LoadInt64(&s.stats.gainersCalculations),
		atomic.LoadInt64(&s.stats.savesTriggered),
		atomic.LoadInt64(&s.stats.errorsCount))

	s.stats.mu.Unlock()

	stopDuration := time.Since(stopStartTime)
	log.Printf("[RealtimeGainersSyncer] 🎯 实时涨幅榜同步器已完全停止，清理耗时: %v", stopDuration)
}

// runWebSocketManager 运行WebSocket连接管理器
func (s *RealtimeGainersSyncer) runWebSocketManager() {
	defer s.wg.Done()

	// WebSocket管理器启动

	ticker := time.NewTicker(5 * time.Minute) // 每5分钟检查一次交易对变化
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			// WebSocket管理器停止
			return
		case <-ticker.C:
			// 检查热门交易对是否有变化，动态调整WebSocket连接
			s.updateWebSocketSubscriptions()
		}
	}
}

// runPriceUpdateProcessor 处理价格更新
func (s *RealtimeGainersSyncer) runPriceUpdateProcessor() {
	defer s.wg.Done()

	// 价格更新处理器启动

	for {
		select {
		case <-s.ctx.Done():
			// 价格更新处理器停止
			return
		case update := <-s.priceUpdateChan:
			// 处理价格更新
			s.processPriceUpdate(update)
		}
	}
}

// runGainersCalculator 运行涨幅榜计算器
func (s *RealtimeGainersSyncer) runGainersCalculator() {
	defer s.wg.Done()

	// 涨幅榜计算器启动

	ticker := time.NewTicker(s.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			// 涨幅榜计算器停止
			return
		case <-ticker.C:
			// 定期重新计算涨幅榜
			s.recalculateGainers()
		}
	}
}

// runStatsReporter 运行统计报告器
func (s *RealtimeGainersSyncer) runStatsReporter() {
	defer s.wg.Done()

	// 统计报告器启动

	ticker := time.NewTicker(1 * time.Minute) // 每分钟报告一次统计信息
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			// 统计报告器停止
			return
		case <-ticker.C:
			s.reportStats()
		}
	}
}

// updateWebSocketSubscriptions 更新WebSocket订阅
func (s *RealtimeGainersSyncer) updateWebSocketSubscriptions() {
	// 获取当前最热门的交易对
	topSymbols := s.getTopSymbolsFromDB()

	if len(topSymbols) == 0 {
		log.Printf("[RealtimeGainersSyncer] ⚠️ 未找到热门交易对，跳过WebSocket订阅更新")
		return
	}

	// 记录订阅变化
	oldCount := atomic.LoadInt64(&s.stats.activeWSConnections)
	newCount := int64(len(topSymbols))

	if oldCount != newCount {
		log.Printf("[RealtimeGainersSyncer] 🔄 WebSocket订阅更新: %d -> %d 个交易对", oldCount, newCount)
		if len(topSymbols) <= 5 {
			log.Printf("[RealtimeGainersSyncer] 📋 新订阅交易对: %v", topSymbols)
		} else {
			log.Printf("[RealtimeGainersSyncer] 📋 新订阅交易对前5个: %v", topSymbols[:5])
		}
	}

	// 更新WebSocket管理器的订阅
	s.wsManager.UpdateSubscriptions(topSymbols, s.priceUpdateChan)

	// 更新统计信息（原子操作）
	atomic.StoreInt64(&s.stats.activeWSConnections, newCount)

	log.Printf("[RealtimeGainersSyncer] ✅ WebSocket订阅更新成功，共订阅 %d 个交易对", newCount)
}

// getTopSymbolsFromDB 从数据库获取最热门的交易对
func (s *RealtimeGainersSyncer) getTopSymbolsFromDB() []string {
	log.Printf("[RealtimeGainersSyncer] 开始获取热门交易对，市场类型: %s, 数量限制: %d", s.kind, s.topSymbolsCount)

	// 首先检查binance_24h_stats表是否有数据
	var totalCount int64
	if err := s.db.Model(&pdb.Binance24hStats{}).Count(&totalCount).Error; err != nil {
		log.Printf("[RealtimeGainersSyncer] 检查binance_24h_stats表失败: %v", err)
	} else {
		log.Printf("[RealtimeGainersSyncer] binance_24h_stats表总记录数: %d", totalCount)
	}

	// 检查指定市场类型的数据
	var marketCount int64
	if err := s.db.Model(&pdb.Binance24hStats{}).Where("market_type = ?", s.kind).Count(&marketCount).Error; err != nil {
		log.Printf("[RealtimeGainersSyncer] 检查%s市场数据失败: %v", s.kind, err)
	} else {
		log.Printf("[RealtimeGainersSyncer] %s市场记录数: %d", s.kind, marketCount)
	}

	// 检查1小时内更新的数据
	var recentCount int64
	if err := s.db.Model(&pdb.Binance24hStats{}).Where("market_type = ? AND updated_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)", s.kind).Count(&recentCount).Error; err != nil {
		log.Printf("[RealtimeGainersSyncer] 检查最近1小时数据失败: %v", err)
	} else {
		log.Printf("[RealtimeGainersSyncer] 最近1小时%s市场记录数: %d", s.kind, recentCount)
	}

	// 从binance_24h_stats表获取涨幅最大的交易对（去重）
	query := `
		SELECT symbol
		FROM (
			SELECT symbol, price_change_percent, volume,
				   ROW_NUMBER() OVER (PARTITION BY symbol ORDER BY price_change_percent DESC, volume DESC) as rn
			FROM binance_24h_stats
			WHERE market_type = ?
			  AND updated_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
			  AND volume > 0
			  AND last_price > 0
		) ranked
		WHERE rn = 1
		ORDER BY price_change_percent DESC, volume DESC
		LIMIT ?
	`

	log.Printf("[RealtimeGainersSyncer] 执行查询: market_type=%s, limit=%d", s.kind, s.topSymbolsCount)

	var symbols []string
	err := s.db.Raw(query, s.kind, s.topSymbolsCount).Scan(&symbols).Error
	if err != nil {
		log.Printf("[RealtimeGainersSyncer] 获取热门交易对失败: %v", err)
		return []string{}
	}

	// 查询到热门交易对

	// 为这些交易对刷新基准价格
	if len(symbols) > 0 {
		s.refreshBasePricesForSymbols(symbols)
	}

	return symbols
}

// executeWithRetry 带重试的执行器
func (s *RealtimeGainersSyncer) executeWithRetry(operation func() error, operationName string, retryable bool) error {
	var lastErr error
	retryCount := 0

	for {
		err := operation()
		if err == nil {
			// 成功，重置错误统计
			s.errorHandler.RecordSuccess()
			return nil
		}

		lastErr = err
		s.errorHandler.RecordError(err, operationName, retryable)

		// 检查是否应该重试
		if !retryable || !s.errorHandler.ShouldRetry(retryCount, s.retryConfig) {
			break
		}

		// 计算重试延迟
		delay := s.errorHandler.CalculateRetryDelay(retryCount, s.retryConfig)
		log.Printf("[%s] 操作失败，重试%d/%d，延迟%v: %v",
			operationName, retryCount+1, s.retryConfig.MaxRetries, delay, err)

		select {
		case <-time.After(delay):
			retryCount++
		case <-s.ctx.Done():
			return s.ctx.Err()
		}
	}

	return lastErr
}

// processPriceUpdate 处理价格更新
func (s *RealtimeGainersSyncer) processPriceUpdate(update PriceUpdate) {
	// 记录重要价格更新（每100个更新记录一次）
	updatesReceived := atomic.AddInt64(&s.stats.priceUpdatesReceived, 1)
	if updatesReceived%100 == 0 {
		log.Printf("[RealtimeGainersSyncer] 📊 已处理 %d 个价格更新，最后更新: %s@%.8f (来源:%s)",
			updatesReceived, update.Symbol, update.Price, update.Source)
	}

	// 更新价格缓存（带错误处理）
	err := s.executeWithRetry(func() error {
		s.priceCache.UpdatePrice(update)
		return nil // 价格缓存更新通常不会失败
	}, "UpdatePriceCache", true)

	if err != nil {
		log.Printf("[RealtimeGainersSyncer] ⚠️ 价格缓存更新失败，但继续处理: %v", err)
		atomic.AddInt64(&s.stats.errorsCount, 1)
		s.stats.mu.Lock()
		s.stats.lastError = err
		s.stats.lastErrorTime = time.Now()
		s.stats.mu.Unlock()
	}

	// 更新统计信息（原子操作）
	s.stats.mu.Lock()
	s.stats.lastUpdateTime = time.Now()
	s.stats.mu.Unlock()

	// 立即触发涨幅榜重新计算（而不是等待定时器）
	s.recalculateGainers()
}

// recalculateGainers 重新计算涨幅榜
func (s *RealtimeGainersSyncer) recalculateGainers() {
	startTime := time.Now()

	// 获取当前缓存的所有交易对价格
	allPrices := s.priceCache.GetAllPrices()

	if len(allPrices) == 0 {
		log.Printf("[RealtimeGainersSyncer] ⚠️ 没有缓存的价格数据，跳过涨幅榜计算")
		s.currentGainersMux.Lock()
		s.currentGainers = []RealtimeGainerItem{}
		s.currentGainersMux.Unlock()
		return
	}

	// 记录计算开始
	calculations := atomic.AddInt64(&s.stats.gainersCalculations, 1)
	if calculations%10 == 0 { // 每10次计算记录一次
		log.Printf("[RealtimeGainersSyncer] 🔄 开始第 %d 次涨幅榜计算，处理 %d 个交易对",
			calculations, len(allPrices))
	}

	// 从数据库获取最新的24h统计数据（包括现成的涨跌幅）
	statsData, err := s.getLatest24hStats()
	if err != nil {
		log.Printf("[RealtimeGainersSyncer] ⚠️ 获取24h统计数据失败，使用传统计算方法: %v", err)
		atomic.AddInt64(&s.stats.errorsCount, 1)
		s.recalculateGainersTraditional(allPrices)
		return
	}

	//if len(statsData) > 0 {
	//	log.Printf("[RealtimeGainersSyncer] ✅ 获取到 %d 条24h统计数据用于涨幅计算", len(statsData))
	//}

	// 计算涨幅榜
	var gainers []RealtimeGainerItem
	validSymbols := 0
	dataSourceStats := make(map[string]int)

	for symbol, priceData := range allPrices {
		var changePercent float64
		var volume24h float64 = priceData.Volume24h
		dataSource := priceData.Source

		// 处理涨跌幅数据
		if priceData.ChangePercent != nil {
			// 优先级1：使用WebSocket提供的实时涨跌幅（最准确）
			changePercent = *priceData.ChangePercent
			dataSource = "websocket"
		} else {
			// 优先级2：使用数据库统计数据
			if stat, exists := statsData[symbol]; exists {
				changePercent = stat.PriceChangePercent
				dataSource = "stats"
			}
		}

		// 处理成交量数据
		if volume24h == 0 {
			// 从统计数据获取成交量
			if stat, exists := statsData[symbol]; exists && stat.Volume > 0 {
				volume24h = stat.Volume
			} else {
				// 从数据库获取
				volume24h = s.getVolume24h(symbol)
			}
		}

		validSymbols++
		dataSourceStats[dataSource]++

		gainer := RealtimeGainerItem{
			Symbol:        symbol,
			CurrentPrice:  priceData.LastPrice,
			ChangePercent: changePercent,
			Volume24h:     volume24h,
			DataSource:    dataSource,
			Timestamp:     priceData.Timestamp,
		}

		gainers = append(gainers, gainer)
	}

	// 记录计算统计信息
	//if len(dataSourceStats) > 0 {
	//	log.Printf("[RealtimeGainersSyncer] 📊 涨幅榜计算完成: %d/%d 个有效交易对，数据来源分布: %v",
	//		validSymbols, len(allPrices), dataSourceStats)
	//}

	// 按涨跌幅降序排序
	sort.Slice(gainers, func(i, j int) bool {
		return gainers[i].ChangePercent > gainers[j].ChangePercent
	})

	// 限制数量并添加排名
	//originalCount := len(gainers)
	if len(gainers) > s.topSymbolsCount {
		gainers = gainers[:s.topSymbolsCount]
	}

	for i := range gainers {
		gainers[i].Rank = i + 1
	}

	s.saveAndUpdateGainers(gainers)

	calculationTime := time.Since(startTime)

	// 记录计算耗时统计
	//if calculationTime > 500*time.Millisecond {
	//	log.Printf("[RealtimeGainersSyncer] ⚠️ 涨幅榜计算耗时较长: %v (%d -> %d 交易对)",
	//		calculationTime, originalCount, len(gainers))
	//} else if calculations%50 == 0 { // 每50次计算记录一次耗时
	//	log.Printf("[RealtimeGainersSyncer] ⏱️ 涨幅榜计算耗时: %v (平均: %v)",
	//		calculationTime, s.stats.avgCalculationTime)
	//}

	// 更新统计信息（原子操作和锁保护的复杂更新）
	s.stats.mu.Lock()
	if atomic.LoadInt64(&s.stats.gainersCalculations) == 1 {
		s.stats.avgCalculationTime = calculationTime
	} else {
		// 指数移动平均
		s.stats.avgCalculationTime = (s.stats.avgCalculationTime + calculationTime) / 2
	}
	s.stats.mu.Unlock()
}

// getLatest24hStats 获取最新的24h统计数据
func (s *RealtimeGainersSyncer) getLatest24hStats() (map[string]*StatsData, error) {
	var results []StatsData
	query := `
		SELECT symbol, price_change_percent, volume, last_price
		FROM binance_24h_stats
		WHERE market_type = ?
		  AND updated_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
		  AND volume > 0
		  AND last_price > 0
	`

	err := s.db.Raw(query, s.kind).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	statsMap := make(map[string]*StatsData)
	for _, stat := range results {
		statsMap[stat.Symbol] = &stat
	}

	return statsMap, nil
}

// StatsData 24h统计数据
type StatsData struct {
	Symbol             string  `json:"symbol"`
	PriceChangePercent float64 `json:"price_change_percent"`
	Volume             float64 `json:"volume"`
	LastPrice          float64 `json:"last_price"`
}

// recalculateGainersTraditional 传统涨幅榜计算方法（后备方案）
func (s *RealtimeGainersSyncer) recalculateGainersTraditional(allPrices map[string]*RealtimePriceData) {
	// 静默使用传统计算方法

	// 计算涨幅榜
	var gainers []RealtimeGainerItem
	validSymbols := 0
	noBasePrice := 0
	zeroBasePrice := 0

	for symbol, priceData := range allPrices {
		// 获取基准价格（24h前的价格）
		basePrice := s.basePriceCache.GetBasePrice(symbol)
		if basePrice <= 0 {
			if basePrice == 0 {
				zeroBasePrice++
			} else {
				noBasePrice++
			}
			continue // 没有基准价格，跳过
		}

		validSymbols++

		// 计算24h涨跌幅
		changePercent := (priceData.LastPrice - basePrice) / basePrice * 100

		// 获取24h成交量（从缓存或API获取）
		volume24h := s.getVolume24h(symbol)

		gainer := RealtimeGainerItem{
			Symbol:        symbol,
			CurrentPrice:  priceData.LastPrice,
			ChangePercent: changePercent,
			Volume24h:     volume24h,
			DataSource:    priceData.Source,
			Timestamp:     priceData.Timestamp,
		}

		gainers = append(gainers, gainer)
	}

	// 移除传统计算的详细统计日志

	// 按涨跌幅降序排序
	sort.Slice(gainers, func(i, j int) bool {
		return gainers[i].ChangePercent > gainers[j].ChangePercent
	})

	// 限制数量并添加排名
	if len(gainers) > s.topSymbolsCount {
		gainers = gainers[:s.topSymbolsCount]
	}

	for i := range gainers {
		gainers[i].Rank = i + 1
	}

	s.saveAndUpdateGainers(gainers)
}

// saveAndUpdateGainers 保存并更新涨幅榜数据
func (s *RealtimeGainersSyncer) saveAndUpdateGainers(gainers []RealtimeGainerItem) {
	// 检查是否有显著变化
	s.currentGainersMux.Lock()

	hasSignificantChanges := s.changeDetector.HasSignificantChanges(gainers)

	// 检查是否是首次运行（没有历史涨幅榜数据）
	isFirstRun := len(s.currentGainers) == 0 && len(s.changeDetector.GetLastGainers()) == 0

	// 定期保存：每分钟保存一次，或者有显著变化时保存
	shouldSave := false
	reason := ""

	if isFirstRun {
		shouldSave = true
		reason = "首次运行，强制保存涨幅榜数据"
		log.Printf("[RealtimeGainersSyncer] 🚀 首次运行，初始化涨幅榜数据")
	} else if hasSignificantChanges {
		shouldSave = true
		reason = "检测到显著变化，触发保存"
		log.Printf("[RealtimeGainersSyncer] 📈 检测到涨幅榜显著变化，准备保存")
	} else if s.shouldSavePeriodically() {
		shouldSave = true
		reason = "定期保存（每分钟一次）"
		log.Printf("[RealtimeGainersSyncer] ⏰ 定期保存时间到达")
	}

	if shouldSave {
		// 保存到数据库
		log.Printf("[RealtimeGainersSyncer] 💾 保存涨幅榜数据: %s (%d条记录)", reason, len(gainers))
		s.saveRealtimeGainers(gainers)

		// 更新当前状态
		s.currentGainers = make([]RealtimeGainerItem, len(gainers))
		copy(s.currentGainers, gainers)

		// 更新变化检测器
		s.changeDetector.UpdateLastGainers(gainers)

		// 记录保存成功的统计
		atomic.AddInt64(&s.stats.savesTriggered, 1)
		log.Printf("[RealtimeGainersSyncer] ✅ 涨幅榜数据保存成功，总保存次数: %d",
			atomic.LoadInt64(&s.stats.savesTriggered))
	} else {
		// 未达到保存条件，跳过保存
		//log.Printf("[RealtimeGainersSyncer] ⏭️ 未达到保存条件，跳过保存 (变化:%v, 定期:%v)",
		//	hasSignificantChanges, s.shouldSavePeriodically())
	}

	s.currentGainersMux.Unlock()
}

// getVolume24h 获取24h成交量
func (s *RealtimeGainersSyncer) getVolume24h(symbol string) float64 {
	// 首先尝试从缓存获取
	if volume := s.priceCache.GetVolume24h(symbol); volume > 0 {
		return volume
	}

	// 从数据库获取
	var result struct {
		Volume float64
	}
	query := `
		SELECT volume
		FROM binance_24h_stats
		WHERE symbol = ? AND market_type = ?
		  AND updated_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
		ORDER BY updated_at DESC
		LIMIT 1
	`

	err := s.db.Raw(query, symbol, s.kind).Scan(&result).Error
	if err != nil || result.Volume <= 0 {
		return 0
	}

	return result.Volume
}

// saveRealtimeGainers 保存实时涨幅榜到数据库
func (s *RealtimeGainersSyncer) saveRealtimeGainers(gainers []RealtimeGainerItem) {
	startTime := time.Now()

	// 转换为数据库格式
	items := make([]pdb.RealtimeGainersItem, 0, len(gainers))
	for _, gainer := range gainers {
		item := pdb.RealtimeGainersItem{
			Symbol:         gainer.Symbol,
			Rank:           gainer.Rank,
			CurrentPrice:   gainer.CurrentPrice,
			PriceChange24h: gainer.ChangePercent,
			Volume24h:      gainer.Volume24h,
			DataSource:     gainer.DataSource,
		}

		// 可选字段
		if gainer.ChangePercent != 0 {
			pc := gainer.ChangePercent
			item.PriceChangePercent = &pc
		}

		items = append(items, item)
	}

	// 保存到数据库（带错误处理和重试）
	err := s.executeWithRetry(func() error {
		_, dbErr := pdb.SaveRealtimeGainers(s.db, s.kind, time.Now(), items)
		return dbErr
	}, "SaveRealtimeGainers", true)

	if err != nil {
		log.Printf("[RealtimeGainersSyncer] 保存实时涨幅榜失败（已重试）: %v", err)
		atomic.AddInt64(&s.stats.errorsCount, 1)
		s.stats.mu.Lock()
		s.stats.lastError = err
		s.stats.lastErrorTime = time.Now()
		s.stats.mu.Unlock()
		return
	}

	// 更新统计信息（原子操作和锁保护的复杂更新）
	saveTime := time.Since(startTime)
	atomic.AddInt64(&s.stats.savesTriggered, 1)

	s.stats.mu.Lock()
	if atomic.LoadInt64(&s.stats.savesTriggered) == 1 {
		s.stats.avgSaveTime = saveTime
	} else {
		s.stats.avgSaveTime = (s.stats.avgSaveTime + saveTime) / 2
	}
	s.stats.mu.Unlock()

	// 移除频繁的保存完成日志
}

// reportStats 报告统计信息
func (s *RealtimeGainersSyncer) reportStats() {
	s.stats.mu.RLock()
	stats := *s.stats
	s.stats.mu.RUnlock()

	uptime := time.Since(s.startTime)

	// 计算性能指标
	var updateRate float64
	if uptime.Seconds() > 0 {
		updateRate = float64(stats.priceUpdatesReceived) / uptime.Seconds()
	}

	var calculationRate float64
	if uptime.Seconds() > 0 {
		calculationRate = float64(stats.gainersCalculations) / uptime.Seconds()
	}

	var saveRate float64
	if uptime.Seconds() > 0 {
		saveRate = float64(stats.savesTriggered) / uptime.Seconds()
	}

	errorRate := float64(stats.errorsCount) / float64(stats.priceUpdatesReceived+stats.gainersCalculations+stats.savesTriggered+1) * 100

	log.Printf("[RealtimeGainersSyncer] 📊 === 实时涨幅榜详细统计报告 ===")
	log.Printf("[RealtimeGainersSyncer] 🕒 运行时间: %v", uptime)
	log.Printf("[RealtimeGainersSyncer] 🌐 WebSocket连接: %d 个活跃连接", stats.activeWSConnections)
	log.Printf("[RealtimeGainersSyncer] 📈 价格更新接收: %d 次 (%.1f 次/秒)",
		stats.priceUpdatesReceived, updateRate)
	log.Printf("[RealtimeGainersSyncer] 🧮 涨幅榜计算: %d 次 (%.2f 次/秒)",
		stats.gainersCalculations, calculationRate)
	log.Printf("[RealtimeGainersSyncer] 💾 数据保存触发: %d 次 (%.2f 次/分钟)",
		stats.savesTriggered, saveRate*60)
	log.Printf("[RealtimeGainersSyncer] ⚡ 性能指标:")
	log.Printf("[RealtimeGainersSyncer]   ├── 平均计算时间: %v", stats.avgCalculationTime)
	log.Printf("[RealtimeGainersSyncer]   ├── 平均保存时间: %v", stats.avgSaveTime)
	log.Printf("[RealtimeGainersSyncer]   └── 缓存命中率: %.1f%%", stats.cacheHitRate*100)

	log.Printf("[RealtimeGainersSyncer] ⚠️  错误统计:")
	log.Printf("[RealtimeGainersSyncer]   ├── 错误次数: %d (错误率: %.2f%%)", stats.errorsCount, errorRate)

	if stats.lastError != nil {
		log.Printf("[RealtimeGainersSyncer]   ├── 最后错误: %v", stats.lastError)
		log.Printf("[RealtimeGainersSyncer]   └── 最后错误时间: %v", stats.lastErrorTime)
	} else {
		log.Printf("[RealtimeGainersSyncer]   └── 状态: 正常运行")
	}

	// 数据库查询性能
	if stats.totalQueries > 0 {
		slowQueryRate := float64(stats.slowQueries) / float64(stats.totalQueries) * 100
		log.Printf("[RealtimeGainersSyncer] 🗄️  数据库性能:")
		log.Printf("[RealtimeGainersSyncer]   ├── 总查询数: %d", stats.totalQueries)
		log.Printf("[RealtimeGainersSyncer]   ├── 慢查询数: %d (%.1f%%)", stats.slowQueries, slowQueryRate)
		log.Printf("[RealtimeGainersSyncer]   └── 平均查询时间: %v", stats.avgQueryTime)
	}

	// 最后更新时间检查
	if stats.lastUpdateTime.IsZero() {
		log.Printf("[RealtimeGainersSyncer] ⏰ 数据状态: 未收到任何更新")
	} else {
		timeSinceLastUpdate := time.Since(stats.lastUpdateTime)
		if timeSinceLastUpdate > 30*time.Second {
			log.Printf("[RealtimeGainersSyncer] ⏰ 数据状态: 最后更新 %v 前 (可能存在延迟)", timeSinceLastUpdate)
		} else {
			log.Printf("[RealtimeGainersSyncer] ⏰ 数据状态: 数据新鲜，最后更新 %v 前", timeSinceLastUpdate)
		}
	}

	// 运行状态评估
	healthScore := s.calculateHealthScore()
	if healthScore >= 80 {
		log.Printf("[RealtimeGainersSyncer] ✅ 系统健康评分: %.1f/100 - 运行良好", healthScore)
	} else if healthScore >= 60 {
		log.Printf("[RealtimeGainersSyncer] ⚠️  系统健康评分: %.1f/100 - 需要关注", healthScore)
	} else {
		log.Printf("[RealtimeGainersSyncer] 🚨 系统健康评分: %.1f/100 - 需要立即处理", healthScore)
	}

	log.Printf("[RealtimeGainersSyncer] 📊 === 报告结束 ===")
}

// ===== DataSyncer 接口实现 =====

// Name 返回同步器名称
func (s *RealtimeGainersSyncer) Name() string {
	return fmt.Sprintf("realtime_gainers_%s", s.kind)
}

// Start 启动同步器（DataSyncer接口）
func (s *RealtimeGainersSyncer) Start(ctx context.Context, interval time.Duration) {
	log.Printf("[RealtimeGainersSyncer] 启动实时涨幅榜同步器 (DataSyncer接口), 间隔:%v", interval)

	// 忽略interval参数，因为这是持续运行的同步器
	if err := s.internalStart(); err != nil {
		log.Printf("[RealtimeGainersSyncer] 启动失败: %v", err)
	}
}

// Stop 停止同步器（DataSyncer接口）
func (s *RealtimeGainersSyncer) Stop() {
	log.Printf("[RealtimeGainersSyncer] 停止实时涨幅榜同步器 (DataSyncer接口)")
	s.internalStop()
}

// Sync 执行一次性同步（DataSyncer接口）
// 对于实时同步器，这个方法用于初始化数据，不建立WebSocket连接
func (s *RealtimeGainersSyncer) Sync(ctx context.Context) error {
	log.Printf("[RealtimeGainersSyncer] 🔄 开始执行手动同步...")

	syncStartTime := time.Now()

	// 获取当前热门交易对（用于初始化数据）
	log.Printf("[RealtimeGainersSyncer] 📋 获取热门交易对用于初始化...")
	topSymbols := s.getTopSymbolsFromDB()
	if len(topSymbols) == 0 {
		log.Printf("[RealtimeGainersSyncer] ❌ 手动同步失败：没有找到热门交易对")
		return fmt.Errorf("没有找到热门交易对")
	}

	log.Printf("[RealtimeGainersSyncer] ✅ 找到 %d 个热门交易对，开始初始化数据", len(topSymbols))

	// 注意：不在Sync阶段建立WebSocket连接，避免与Start()冲突
	// WebSocket连接由Start()方法统一管理

	// 执行一次涨幅榜计算（用于初始化数据）
	log.Printf("[RealtimeGainersSyncer] 🧮 执行初始化涨幅榜计算...")
	s.recalculateGainers()

	syncDuration := time.Since(syncStartTime)
	log.Printf("[RealtimeGainersSyncer] ✅ 手动同步完成，耗时: %v (WebSocket连接由Start()管理)", syncDuration)

	return nil
}

// GetStats 获取统计信息（DataSyncer接口）
func (s *RealtimeGainersSyncer) GetStats() map[string]interface{} {
	return s.getStats()
}

// getStats 获取统计信息（内部方法）
func (s *RealtimeGainersSyncer) getStats() map[string]interface{} {
	// 使用原子操作读取计数器，避免锁竞争
	activeWSConnections := atomic.LoadInt64(&s.stats.activeWSConnections)
	totalWSReconnects := atomic.LoadInt64(&s.stats.totalWSReconnects)
	priceUpdatesReceived := atomic.LoadInt64(&s.stats.priceUpdatesReceived)
	gainersCalculations := atomic.LoadInt64(&s.stats.gainersCalculations)
	savesTriggered := atomic.LoadInt64(&s.stats.savesTriggered)
	errorsCount := atomic.LoadInt64(&s.stats.errorsCount)
	totalQueries := atomic.LoadInt64(&s.stats.totalQueries)
	slowQueries := atomic.LoadInt64(&s.stats.slowQueries)

	s.stats.mu.RLock()
	isRunning := s.stats.isRunning
	startTime := s.startTime
	avgCalculationTime := s.stats.avgCalculationTime
	avgSaveTime := s.stats.avgSaveTime
	cacheHitRate := s.stats.cacheHitRate
	avgQueryTime := s.stats.avgQueryTime
	lastError := s.stats.lastError
	lastErrorTime := s.stats.lastErrorTime
	lastUpdateTime := s.stats.lastUpdateTime
	s.stats.mu.RUnlock()

	return map[string]interface{}{
		"is_running":             isRunning,
		"start_time":             startTime,
		"uptime":                 time.Since(startTime).String(),
		"active_ws_connections":  activeWSConnections,
		"total_ws_reconnects":    totalWSReconnects,
		"price_updates_received": priceUpdatesReceived,
		"gainers_calculations":   gainersCalculations,
		"saves_triggered":        savesTriggered,
		"avg_calculation_time":   avgCalculationTime.String(),
		"avg_save_time":          avgSaveTime.String(),
		"cache_hit_rate":         cacheHitRate,
		"errors_count":           errorsCount,
		"total_queries":          totalQueries,
		"slow_queries":           slowQueries,
		"avg_query_time":         avgQueryTime.String(),
		"last_error":             fmt.Sprintf("%v", lastError),
		"last_error_time":        lastErrorTime,
		"last_update_time":       lastUpdateTime,
	}
}

// calculateHealthScore 计算系统健康评分 (0-100)
func (s *RealtimeGainersSyncer) calculateHealthScore() float64 {
	score := 100.0

	// 检查运行状态
	if !s.stats.isRunning {
		return 0.0 // 未运行
	}

	// 检查数据新鲜度 (30分)
	timeSinceLastUpdate := time.Since(s.stats.lastUpdateTime)
	if timeSinceLastUpdate > 60*time.Second {
		score -= 30
	} else if timeSinceLastUpdate > 30*time.Second {
		score -= 15
	}

	// 检查错误率 (25分)
	totalOperations := s.stats.priceUpdatesReceived + s.stats.gainersCalculations + s.stats.savesTriggered
	if totalOperations > 0 {
		errorRate := float64(s.stats.errorsCount) / float64(totalOperations)
		if errorRate > 0.1 { // 错误率超过10%
			score -= 25
		} else if errorRate > 0.05 { // 错误率超过5%
			score -= 12.5
		} else if errorRate > 0.01 { // 错误率超过1%
			score -= 5
		}
	}

	// 检查连接状态 (15分)
	if s.stats.activeWSConnections == 0 {
		score -= 15
	} else if s.stats.activeWSConnections < 5 {
		score -= 7.5
	}

	// 检查性能 (15分)
	if s.stats.avgCalculationTime > 2*time.Second {
		score -= 15
	} else if s.stats.avgCalculationTime > 1*time.Second {
		score -= 7.5
	}

	// 检查缓存命中率 (10分)
	if s.stats.cacheHitRate < 0.5 { // 缓存命中率低于50%
		score -= 10
	} else if s.stats.cacheHitRate < 0.7 { // 缓存命中率低于70%
		score -= 5
	}

	// 检查慢查询比例 (10分)
	if s.stats.totalQueries > 0 {
		slowQueryRate := float64(s.stats.slowQueries) / float64(s.stats.totalQueries)
		if slowQueryRate > 0.2 { // 慢查询比例超过20%
			score -= 10
		} else if slowQueryRate > 0.1 { // 慢查询比例超过10%
			score -= 5
		}
	}

	// 确保分数在合理范围内
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// GetInternalStats 获取内部统计信息
func (s *RealtimeGainersSyncer) GetInternalStats() map[string]interface{} {
	stats := s.getStats()
	stats["health_score"] = s.calculateHealthScore()
	stats["uptime_seconds"] = time.Since(s.startTime).Seconds()
	return stats
}
