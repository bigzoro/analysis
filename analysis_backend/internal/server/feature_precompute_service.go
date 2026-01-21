package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// FeaturePrecomputeService 特征预计算服务 - 后台预计算和缓存特征
type FeaturePrecomputeService struct {
	featureEngineering *FeatureEngineering
	cacheManager       *FeatureCacheManager
	server             *Server
	isRunning          bool
	updateInterval     time.Duration
	symbols            []string
	ctx                context.Context
	cancel             context.CancelFunc
	wg                 sync.WaitGroup
}

// NewFeaturePrecomputeService 创建特征预计算服务
func NewFeaturePrecomputeService(featureEngineering *FeatureEngineering, server *Server) *FeaturePrecomputeService {
	ctx, cancel := context.WithCancel(context.Background())

	// 创建特征缓存管理器
	cacheManager := NewFeatureCacheManager(server)

	return &FeaturePrecomputeService{
		featureEngineering: featureEngineering,
		cacheManager:       cacheManager,
		server:             server,
		updateInterval:     30 * time.Minute, // 每30分钟更新一次
		symbols: []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
			"DOGEUSDT", "DOTUSDT", "AVAXUSDT", "LTCUSDT", "TRXUSDT",
			"LINKUSDT", "MATICUSDT", "ALGOUSDT", "VETUSDT", "ICPUSDT",
		},
		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动特征预计算服务
func (fps *FeaturePrecomputeService) Start() error {
	if fps.isRunning {
		return fmt.Errorf("特征预计算服务已在运行")
	}

	fps.isRunning = true
	log.Printf("[FeaturePrecomputeService] 启动特征预计算服务，更新间隔: %v", fps.updateInterval)

	// 启动后台更新协程
	fps.wg.Add(1)
	go fps.precomputeLoop()

	log.Printf("[FeaturePrecomputeService] 特征预计算服务启动成功")
	return nil
}

// Stop 停止特征预计算服务
func (fps *FeaturePrecomputeService) Stop() error {
	if !fps.isRunning {
		return nil
	}

	log.Printf("[FeaturePrecomputeService] 正在停止特征预计算服务...")
	fps.isRunning = false
	fps.cancel()

	// 等待所有协程结束
	done := make(chan struct{})
	go func() {
		fps.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[FeaturePrecomputeService] 特征预计算服务已停止")
	case <-time.After(30 * time.Second):
		log.Printf("[FeaturePrecomputeService] 特征预计算服务停止超时")
	}

	return nil
}

// precomputeLoop 预计算循环
func (fps *FeaturePrecomputeService) precomputeLoop() {
	defer fps.wg.Done()

	ticker := time.NewTicker(fps.updateInterval)
	defer ticker.Stop()

	// 启动时立即执行一次预计算
	fps.performFullPrecomputation()

	for {
		select {
		case <-fps.ctx.Done():
			log.Printf("[FeaturePrecomputeService] 收到停止信号，退出预计算循环")
			return
		case <-ticker.C:
			fps.performFullPrecomputation()
		}
	}
}

// performFullPrecomputation 执行完整特征预计算
func (fps *FeaturePrecomputeService) performFullPrecomputation() {
	log.Printf("[FeaturePrecomputeService] 开始执行完整特征预计算...")

	// 健康检查：确保依赖准备就绪
	if !fps.checkDependenciesReady() {
		log.Printf("[FeaturePrecomputeService] 依赖未准备就绪，跳过本次预计算")
		return
	}

	startTime := time.Now()
	successCount := 0
	totalCount := len(fps.symbols)

	// 并发预计算所有币种的特征
	semaphore := make(chan struct{}, 8) // 限制并发数为8
	results := make(chan precomputeResult, len(fps.symbols))

	for _, symbol := range fps.symbols {
		go func(sym string) {
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			result := fps.precomputeSymbolFeatures(sym)
			results <- result
		}(symbol)
	}

	// 收集结果
	for i := 0; i < len(fps.symbols); i++ {
		result := <-results
		if result.success {
			successCount++
		}
	}

	duration := time.Since(startTime)
	log.Printf("[FeaturePrecomputeService] 特征预计算完成，成功: %d/%d，总耗时: %v",
		successCount, totalCount, duration)

	// 清理过期缓存
	fps.cacheManager.CleanupExpiredFeatures()
}

// checkDependenciesReady 检查依赖是否准备就绪
func (fps *FeaturePrecomputeService) checkDependenciesReady() bool {
	// 检查服务器实例
	if fps.server == nil {
		log.Printf("[FeaturePrecomputeService] 服务器实例未准备就绪")
		return false
	}

	// 检查数据库连接
	if fps.server.db == nil {
		log.Printf("[FeaturePrecomputeService] 数据库连接未准备就绪")
		return false
	}

	// 检查回测引擎
	if fps.server.backtestEngine == nil {
		log.Printf("[FeaturePrecomputeService] BacktestEngine未准备就绪")
		return false
	}

	// 检查特征工程模块
	if fps.featureEngineering == nil {
		log.Printf("[FeaturePrecomputeService] FeatureEngineering未准备就绪")
		return false
	}

	return true
}

// precomputeSymbolFeatures 预计算单个币种的特征
func (fps *FeaturePrecomputeService) precomputeSymbolFeatures(symbol string) precomputeResult {
	log.Printf("[FeaturePrecomputeService] 开始预计算 %s 特征", symbol)

	// 计算时间范围（最近7天的数据用于特征计算）
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -7)

	// 获取历史数据
	data, err := fps.server.backtestEngine.getHistoricalData(fps.ctx, symbol, startDate, endDate)
	if err != nil {
		log.Printf("[FeaturePrecomputeService] 获取 %s 历史数据失败: %v", symbol, err)
		return precomputeResult{symbol: symbol, success: false, error: err}
	}

	if len(data) < 50 {
		log.Printf("[FeaturePrecomputeService] %s 数据点不足: %d < 50", symbol, len(data))
		return precomputeResult{symbol: symbol, success: false}
	}

	// 计算多个时间窗口的特征
	timeWindows := []int{24, 72, 168} // 1天、3天、7天
	totalFeatures := 0

	for _, hours := range timeWindows {
		windowData := fps.getDataForTimeWindow(data, hours)
		if len(windowData) < 20 {
			continue
		}

		// 预计算特征
		featureKey := fps.cacheManager.generateFeatureKey(symbol, hours, windowData)
		existingFeatures := fps.cacheManager.GetFeatures(featureKey)

		if existingFeatures == nil {
			// 计算新特征
			features, err := fps.computeFeaturesForWindow(symbol, windowData)
			if err != nil {
				log.Printf("[FeaturePrecomputeService] 计算 %s %dh 特征失败: %v", symbol, hours, err)
				continue
			}

			// 缓存特征
			fps.cacheManager.SetDetailedFeatures(featureKey, symbol, hours, len(windowData), features, FeatureQuality{Overall: 0.8})
			totalFeatures += len(features)
			log.Printf("[FeaturePrecomputeService] 缓存 %s %dh 特征: %d 个", symbol, hours, len(features))
		} else {
			totalFeatures += len(existingFeatures)
		}
	}

	log.Printf("[FeaturePrecomputeService] %s 特征预计算成功，总计: %d 个特征", symbol, totalFeatures)
	return precomputeResult{symbol: symbol, success: true, featuresCount: totalFeatures}
}

// getDataForTimeWindow 获取指定时间窗口的数据
func (fps *FeaturePrecomputeService) getDataForTimeWindow(data []MarketData, hours int) []MarketData {
	if len(data) == 0 {
		return nil
	}

	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)
	result := make([]MarketData, 0)

	for _, md := range data {
		if md.LastUpdated.After(cutoffTime) {
			result = append(result, md)
		}
	}

	return result
}

// computeFeaturesForWindow 为时间窗口计算特征
func (fps *FeaturePrecomputeService) computeFeaturesForWindow(symbol string, data []MarketData) (map[string]float64, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("no data provided")
	}

	// 转换为MarketDataPoint格式
	dataPoints := make([]*MarketDataPoint, len(data))
	for i, md := range data {
		dataPoints[i] = &MarketDataPoint{
			Symbol:         md.Symbol,
			Price:          md.Price,
			Volume24h:      md.Volume24h,
			Timestamp:      md.LastUpdated,
			PriceChange24h: md.Change24h,
		}
	}

	// 使用特征工程系统计算特征
	ctx := context.Background()

	featureSet, err := fps.featureEngineering.ExtractFeatures(ctx, symbol)
	if err != nil {
		// 如果ExtractFeatures失败，尝试直接使用特征提取器
		log.Printf("[FeaturePrecomputeService] ExtractFeatures失败，尝试直接计算: %v", err)

		features := make(map[string]float64)

		// 计算基础技术指标
		if len(dataPoints) >= 50 {
			prices := make([]float64, len(dataPoints))
			volumes := make([]float64, len(dataPoints))

			for i, dp := range dataPoints {
				prices[i] = dp.Price
				volumes[i] = dp.Volume24h
			}

			// 计算SMA
			sma20 := fps.calculateSMA(prices, 20)
			sma50 := fps.calculateSMA(prices, 50)

			if len(sma20) > 0 {
				features["sma_20"] = sma20[len(sma20)-1]
			}
			if len(sma50) > 0 {
				features["sma_50"] = sma50[len(sma50)-1]
			}

			// 计算RSI
			rsi := fps.calculateRSI(prices, 14)
			if len(rsi) > 0 {
				features["rsi_14"] = rsi[len(rsi)-1]
			}

			// 计算波动率
			if len(prices) >= 20 {
				volatility := fps.calculateVolatility(prices, 20)
				features["volatility_20"] = volatility
			}
		}

		return features, nil
	}

	return featureSet.Features, nil
}

// 基础技术指标计算方法
func (fps *FeaturePrecomputeService) calculateSMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	sma := make([]float64, len(prices))
	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j]
		}
		sma[i] = sum / float64(period)
	}

	return sma
}

func (fps *FeaturePrecomputeService) calculateRSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return []float64{}
	}

	rsi := make([]float64, len(prices))

	for i := period; i < len(prices); i++ {
		gains := 0.0
		losses := 0.0

		for j := i - period + 1; j <= i; j++ {
			change := prices[j] - prices[j-1]
			if change > 0 {
				gains += change
			} else {
				losses -= change
			}
		}

		if losses == 0 {
			rsi[i] = 100.0
		} else {
			rs := gains / losses
			rsi[i] = 100.0 - (100.0 / (1.0 + rs))
		}
	}

	return rsi
}

func (fps *FeaturePrecomputeService) calculateVolatility(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	// 计算收益率
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	// 计算标准差作为波动率度量
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}

// GetStatus 获取服务状态
func (fps *FeaturePrecomputeService) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"is_running":      fps.isRunning,
		"update_interval": fps.updateInterval.String(),
		"symbols_count":   len(fps.symbols),
		"symbols":         fps.symbols,
		"cache_stats":     fps.cacheManager.GetStats(),
		"last_update":     time.Now().Format("2006-01-02 15:04:05"),
	}
}

// precomputeResult 预计算结果
type precomputeResult struct {
	symbol        string
	success       bool
	featuresCount int
	error         error
}
