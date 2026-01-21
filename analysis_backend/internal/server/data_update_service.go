package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// DataUpdateService 数据更新服务 - 后台定期更新和预处理数据
type DataUpdateService struct {
	cache          *BacktestDataCache
	preprocessor   *DataPreprocessor
	server         *Server
	isRunning      bool
	updateInterval time.Duration
	symbols        []string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// NewDataUpdateService 创建数据更新服务
func NewDataUpdateService(cache *BacktestDataCache, preprocessor *DataPreprocessor, server *Server) *DataUpdateService {
	ctx, cancel := context.WithCancel(context.Background())

	return &DataUpdateService{
		cache:          cache,
		preprocessor:   preprocessor,
		server:         server,
		updateInterval: 1 * time.Hour, // 每小时更新一次
		symbols: []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
			"DOGEUSDT", "DOTUSDT", "AVAXUSDT", "LTCUSDT", "TRXUSDT",
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动数据更新服务
func (dus *DataUpdateService) Start() error {
	if dus.isRunning {
		return fmt.Errorf("数据更新服务已在运行")
	}

	dus.isRunning = true
	log.Printf("[DataUpdateService] 启动数据更新服务，更新间隔: %v", dus.updateInterval)

	// 启动后台更新协程
	dus.wg.Add(1)
	go dus.updateLoop()

	log.Printf("[DataUpdateService] 数据更新服务启动成功")
	return nil
}

// Stop 停止数据更新服务
func (dus *DataUpdateService) Stop() error {
	if !dus.isRunning {
		return nil
	}

	log.Printf("[DataUpdateService] 正在停止数据更新服务...")
	dus.isRunning = false
	dus.cancel()

	// 等待所有协程结束
	done := make(chan struct{})
	go func() {
		dus.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[DataUpdateService] 数据更新服务已停止")
	case <-time.After(30 * time.Second):
		log.Printf("[DataUpdateService] 数据更新服务停止超时")
	}

	return nil
}

// updateLoop 更新循环
func (dus *DataUpdateService) updateLoop() {
	defer dus.wg.Done()

	ticker := time.NewTicker(dus.updateInterval)
	defer ticker.Stop()

	// 启动时立即执行一次更新
	dus.performFullUpdate()

	for {
		select {
		case <-dus.ctx.Done():
			log.Printf("[DataUpdateService] 收到停止信号，退出更新循环")
			return
		case <-ticker.C:
			dus.performFullUpdate()
		}
	}
}

// performFullUpdate 执行完整的数据更新
func (dus *DataUpdateService) performFullUpdate() {
	log.Printf("[DataUpdateService] 开始执行完整数据更新...")

	// 健康检查：确保依赖准备就绪
	if !dus.checkDependenciesReady() {
		log.Printf("[DataUpdateService] 依赖未准备就绪，跳过本次更新")
		return
	}

	startTime := time.Now()
	successCount := 0
	totalCount := len(dus.symbols)

	// 并发更新所有币种的数据
	semaphore := make(chan struct{}, 5) // 限制并发数为5
	results := make(chan updateResult, len(dus.symbols))

	for _, symbol := range dus.symbols {
		go func(sym string) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result := dus.updateSymbolData(sym)
			results <- result
		}(symbol)
	}

	// 收集结果
	for i := 0; i < len(dus.symbols); i++ {
		result := <-results
		if result.success {
			successCount++
		}
	}

	duration := time.Since(startTime)
	log.Printf("[DataUpdateService] 数据更新完成，成功: %d/%d，总耗时: %v",
		successCount, totalCount, duration)

	// 清理过期缓存
	dus.cleanupExpiredCache()
}

// checkDependenciesReady 检查依赖是否准备就绪
func (dus *DataUpdateService) checkDependenciesReady() bool {
	// 检查数据库连接
	if dus.server == nil || dus.server.db == nil {
		log.Printf("[DataUpdateService] 数据库连接未准备就绪")
		return false
	}

	// 检查回测引擎
	if dus.server.backtestEngine == nil {
		log.Printf("[DataUpdateService] BacktestEngine未准备就绪")
		return false
	}

	return true
}

// updateSymbolData 更新单个币种的数据
func (dus *DataUpdateService) updateSymbolData(symbol string) updateResult {
	log.Printf("[DataUpdateService] 开始更新 %s 数据", symbol)

	// 计算时间范围（最近30天的数据用于回测）
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)

	// 获取原始数据
	rawData, err := dus.fetchMarketData(symbol, startDate, endDate)
	if err != nil {
		log.Printf("[DataUpdateService] 获取 %s 原始数据失败: %v", symbol, err)
		return updateResult{symbol: symbol, success: false, error: err}
	}

	if len(rawData) == 0 {
		log.Printf("[DataUpdateService] %s 没有获取到数据", symbol)
		return updateResult{symbol: symbol, success: false}
	}

	// 缓存预处理后的数据
	dus.cache.SetMarketData(symbol, startDate, endDate, rawData)

	log.Printf("[DataUpdateService] %s 数据更新成功，%d 个数据点", symbol, len(rawData))
	return updateResult{symbol: symbol, success: true, dataPoints: len(rawData)}
}

// fetchMarketData 获取市场数据
func (dus *DataUpdateService) fetchMarketData(symbol string, startDate, endDate time.Time) ([]MarketData, error) {
	// 通过backtestEngine获取历史数据
	data, err := dus.server.backtestEngine.getHistoricalData(dus.ctx, symbol, startDate, endDate)
	if err != nil {
		log.Printf("[DataUpdateService] 获取 %s 数据失败: %v", symbol, err)
		return nil, err
	}

	return data, nil
}

// cleanupExpiredCache 清理过期缓存
func (dus *DataUpdateService) cleanupExpiredCache() {
	// 这里可以添加更复杂的缓存清理逻辑
	// 目前DataCache已经有自动清理机制
	log.Printf("[DataUpdateService] 缓存清理检查完成")
}

// GetStatus 获取服务状态
func (dus *DataUpdateService) GetStatus() map[string]interface{} {
	dus.cache.mu.RLock()
	defer dus.cache.mu.RUnlock()

	cacheStats := map[string]interface{}{
		"cache_size":       len(dus.cache.processedData),
		"max_cache_size":   dus.cache.maxSize,
		"cache_max_age":    dus.cache.maxAge.String(),
		"cache_hit_count":  dus.cache.hitCount,
		"cache_miss_count": dus.cache.missCount,
	}

	hitRate := float64(0)
	if dus.cache.hitCount+dus.cache.missCount > 0 {
		hitRate = float64(dus.cache.hitCount) / float64(dus.cache.hitCount+dus.cache.missCount)
	}

	return map[string]interface{}{
		"is_running":      dus.isRunning,
		"update_interval": dus.updateInterval.String(),
		"symbols_count":   len(dus.symbols),
		"symbols":         dus.symbols,
		"cache_stats":     cacheStats,
		"cache_hit_rate":  fmt.Sprintf("%.2f%%", hitRate*100),
		"last_update":     time.Now().Format("2006-01-02 15:04:05"),
	}
}

// updateResult 更新结果
type updateResult struct {
	symbol     string
	success    bool
	dataPoints int
	error      error
}
