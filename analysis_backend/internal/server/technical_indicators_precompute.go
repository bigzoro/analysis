package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	pdb "analysis/internal/db"
)

// TechnicalIndicatorsPrecomputeService 技术指标预计算服务
type TechnicalIndicatorsPrecomputeService struct {
	cacheManager   *TechnicalIndicatorsCacheManager
	server         *Server
	isRunning      bool
	updateInterval time.Duration
	symbols        []string
	timeframes     []string
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
}

// TechnicalIndicatorsCacheManager 技术指标缓存管理器
type TechnicalIndicatorsCacheManager struct {
	mu              sync.RWMutex
	indicatorsCache map[string]*CachedTechnicalIndicators
	maxSize         int
	maxAge          time.Duration
	hitCount        int64
	missCount       int64
	server          *Server // 用于数据库访问
}

// CachedTechnicalIndicators 缓存的技术指标
type CachedTechnicalIndicators struct {
	Symbol      string
	Timeframe   string
	Indicators  TechnicalIndicators
	ComputedAt  time.Time
	AccessCount int64
}

// NewTechnicalIndicatorsPrecomputeService 创建技术指标预计算服务
func NewTechnicalIndicatorsPrecomputeService(server *Server) *TechnicalIndicatorsPrecomputeService {
	ctx, cancel := context.WithCancel(context.Background())

	cacheManager := &TechnicalIndicatorsCacheManager{
		indicatorsCache: make(map[string]*CachedTechnicalIndicators),
		maxSize:         10000,         // 最大缓存10000个指标集合
		maxAge:          4 * time.Hour, // 缓存4小时
		server:          server,        // 用于数据库访问
	}

	return &TechnicalIndicatorsPrecomputeService{
		cacheManager:   cacheManager,
		server:         server,
		updateInterval: 15 * time.Minute, // 每15分钟更新一次
		symbols: []string{
			"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
			"DOGEUSDT", "DOTUSDT", "AVAXUSDT", "LTCUSDT", "TRXUSDT",
		},
		timeframes: []string{"1m", "5m", "15m", "1h", "4h", "1d"},
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动技术指标预计算服务
func (tips *TechnicalIndicatorsPrecomputeService) Start() error {
	if tips.isRunning {
		return fmt.Errorf("技术指标预计算服务已在运行")
	}

	tips.isRunning = true
	log.Printf("[TechnicalIndicatorsPrecompute] 启动技术指标预计算服务，更新间隔: %v", tips.updateInterval)

	// 启动后台更新协程
	tips.wg.Add(1)
	go tips.precomputeLoop()

	log.Printf("[TechnicalIndicatorsPrecompute] 技术指标预计算服务启动成功")
	return nil
}

// Stop 停止技术指标预计算服务
func (tips *TechnicalIndicatorsPrecomputeService) Stop() error {
	if !tips.isRunning {
		return nil
	}

	log.Printf("[TechnicalIndicatorsPrecompute] 正在停止技术指标预计算服务...")
	tips.isRunning = false
	tips.cancel()

	done := make(chan struct{})
	go func() {
		tips.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Printf("[TechnicalIndicatorsPrecompute] 技术指标预计算服务已停止")
	case <-time.After(30 * time.Second):
		log.Printf("[TechnicalIndicatorsPrecompute] 技术指标预计算服务停止超时")
	}

	return nil
}

// precomputeLoop 预计算循环
func (tips *TechnicalIndicatorsPrecomputeService) precomputeLoop() {
	defer tips.wg.Done()

	ticker := time.NewTicker(tips.updateInterval)
	defer ticker.Stop()

	// 启动时立即执行一次预计算
	tips.performFullPrecomputation()

	for {
		select {
		case <-tips.ctx.Done():
			log.Printf("[TechnicalIndicatorsPrecompute] 收到停止信号，退出预计算循环")
			return
		case <-ticker.C:
			tips.performFullPrecomputation()
		}
	}
}

// performFullPrecomputation 执行完整技术指标预计算
func (tips *TechnicalIndicatorsPrecomputeService) performFullPrecomputation() {
	log.Printf("[TechnicalIndicatorsPrecompute] 开始执行完整技术指标预计算...")

	startTime := time.Now()
	totalTasks := len(tips.symbols) * len(tips.timeframes)
	completedTasks := 0

	// 并发预计算所有币种和时间框架的指标
	semaphore := make(chan struct{}, 12) // 限制并发数为12
	results := make(chan precomputeResult, totalTasks)

	for _, symbol := range tips.symbols {
		for _, timeframe := range tips.timeframes {
			go func(sym, tf string) {
				semaphore <- struct{}{}        // 获取信号量
				defer func() { <-semaphore }() // 释放信号量

				result := tips.precomputeSymbolIndicators(sym, tf)
				results <- result
			}(symbol, timeframe)
		}
	}

	// 收集结果
	successCount := 0
	for i := 0; i < totalTasks; i++ {
		result := <-results
		if result.success {
			successCount++
		}
		completedTasks++
	}

	duration := time.Since(startTime)
	log.Printf("[TechnicalIndicatorsPrecompute] 技术指标预计算完成，成功: %d/%d，总耗时: %v",
		successCount, totalTasks, duration)

	// 清理过期缓存
	tips.cacheManager.CleanupExpiredIndicators()
}

// precomputeSymbolIndicators 预计算单个币种的技术指标
func (tips *TechnicalIndicatorsPrecomputeService) precomputeSymbolIndicators(symbol, timeframe string) precomputeResult {
	log.Printf("[TechnicalIndicatorsPrecompute] 开始预计算 %s %s 技术指标", symbol, timeframe)

	// 获取K线数据
	klineData, err := tips.getKlineData(symbol, timeframe)
	if err != nil {
		log.Printf("[TechnicalIndicatorsPrecompute] 获取 %s %s K线数据失败: %v", symbol, timeframe, err)
		return precomputeResult{symbol: symbol, success: false, error: err}
	}

	if len(klineData) < 100 {
		log.Printf("[TechnicalIndicatorsPrecompute] %s %s 数据点不足: %d < 100", symbol, timeframe, len(klineData))
		return precomputeResult{symbol: symbol, success: false}
	}

	// 检查缓存
	cacheKey := fmt.Sprintf("%s_%s", symbol, timeframe)
	existingIndicators := tips.cacheManager.GetIndicators(cacheKey)

	if existingIndicators != nil {
		// 检查是否需要更新（基于时间戳）
		if time.Since(existingIndicators.ComputedAt) < 10*time.Minute {
			log.Printf("[TechnicalIndicatorsPrecompute] %s %s 指标缓存仍然有效", symbol, timeframe)
			return precomputeResult{symbol: symbol, success: true}
		}
	}

	// 计算技术指标
	indicators, err := tips.computeTechnicalIndicators(klineData, symbol, timeframe)
	if err != nil {
		log.Printf("[TechnicalIndicatorsPrecompute] 计算 %s %s 技术指标失败: %v", symbol, timeframe, err)
		return precomputeResult{symbol: symbol, success: false, error: err}
	}

	// 缓存指标
	tips.cacheManager.SetIndicators(cacheKey, symbol, timeframe, indicators)

	log.Printf("[TechnicalIndicatorsPrecompute] %s %s 技术指标预计算成功", symbol, timeframe)
	return precomputeResult{symbol: symbol, success: true}
}

// getKlineData 获取K线数据
func (tips *TechnicalIndicatorsPrecomputeService) getKlineData(symbol, timeframe string) ([]KlineData, error) {
	// 计算数据时间范围
	endTime := time.Now()
	var startTime time.Time

	// 根据时间框架确定数据量
	switch timeframe {
	case "1m":
		startTime = endTime.Add(-24 * time.Hour) // 最近24小时
	case "5m":
		startTime = endTime.Add(-72 * time.Hour) // 最近3天
	case "15m":
		startTime = endTime.Add(-7 * 24 * time.Hour) // 最近7天
	case "1h":
		startTime = endTime.Add(-30 * 24 * time.Hour) // 最近30天
	case "4h":
		startTime = endTime.Add(-90 * 24 * time.Hour) // 最近90天
	case "1d":
		startTime = endTime.Add(-365 * 24 * time.Hour) // 最近1年
	default:
		startTime = endTime.Add(-7 * 24 * time.Hour) // 默认7天
	}

	// 从数据库获取K线数据
	marketKlines, err := pdb.GetMarketKlines(tips.server.db.DB(), symbol, "spot", timeframe, &startTime, &endTime, 1000)
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	// 转换为KlineData格式
	klineData := make([]KlineData, len(marketKlines))
	for i, mk := range marketKlines {
		// 处理可选字段的默认值
		quoteVolume := ""
		if mk.QuoteVolume != nil {
			quoteVolume = *mk.QuoteVolume
		}

		takerBuyVolume := ""
		if mk.TakerBuyVolume != nil {
			takerBuyVolume = *mk.TakerBuyVolume
		}

		takerBuyQuoteVolume := ""
		if mk.TakerBuyQuoteVolume != nil {
			takerBuyQuoteVolume = *mk.TakerBuyQuoteVolume
		}

		klineData[i] = KlineData{
			BinanceKline: BinanceKline{
				OpenTime:                 float64(mk.OpenTime.Unix() * 1000),
				Open:                     mk.OpenPrice,
				High:                     mk.HighPrice,
				Low:                      mk.LowPrice,
				Close:                    mk.ClosePrice,
				Volume:                   mk.Volume,
				CloseTime:                float64(mk.OpenTime.Add(time.Duration(getIntervalMinutes(timeframe))*time.Minute).Unix() * 1000),
				QuoteAssetVolume:         quoteVolume,
				NumberOfTrades:           0, // 默认值
				TakerBuyBaseAssetVolume:  takerBuyVolume,
				TakerBuyQuoteAssetVolume: takerBuyQuoteVolume,
			},
			Symbol:      mk.Symbol,
			Interval:    mk.Interval,
			Kind:        mk.Kind,
			Timestamp:   mk.OpenTime,
			IsValid:     true,
			DataQuality: 100,
			ProcessedAt: time.Now(),
		}
	}

	return klineData, nil
}

// getIntervalMinutes 获取时间间隔的分钟数
func getIntervalMinutes(interval string) int {
	switch interval {
	case "1m":
		return 1
	case "5m":
		return 5
	case "15m":
		return 15
	case "1h":
		return 60
	case "4h":
		return 240
	case "1d":
		return 1440
	default:
		return 60
	}
}

// computeTechnicalIndicators 计算技术指标
func (tips *TechnicalIndicatorsPrecomputeService) computeTechnicalIndicators(klineData []KlineData, symbol, timeframe string) (TechnicalIndicators, error) {
	if len(klineData) < 20 {
		return TechnicalIndicators{}, fmt.Errorf("数据点不足")
	}

	// 提取价格数据
	closes := make([]float64, len(klineData))
	highs := make([]float64, len(klineData))
	lows := make([]float64, len(klineData))
	volumes := make([]float64, len(klineData))

	for i, kline := range klineData {
		closes[i], _ = strconv.ParseFloat(kline.Close, 64)
		highs[i], _ = strconv.ParseFloat(kline.High, 64)
		lows[i], _ = strconv.ParseFloat(kline.Low, 64)
		volumes[i], _ = strconv.ParseFloat(kline.Volume, 64)
	}

	// 计算各种技术指标
	indicators := TechnicalIndicators{}

	// 趋势指标
	indicators.MA20 = calculateSMA(closes, 20)
	indicators.MA50 = calculateSMA(closes, 50)

	// 动量指标
	indicators.RSI = calculateRSI(closes, 14)
	indicators.MACD, indicators.MACDSignal, indicators.MACDHist = calculateMACD(closes, 12, 26, 9)
	indicators.Momentum10 = calculateMomentum(closes, 10)

	// 波动率指标
	indicators.BBUpper, indicators.BBMiddle, indicators.BBLower, indicators.BBPosition, indicators.BBWidth = calculateBollingerBands(closes, 20, 2.0)

	// 震荡指标
	indicators.K, indicators.D, indicators.J = calculateKDJ(highs, lows, closes, 14)

	// 成交量指标
	indicators.OBV = calculateOBV(closes, volumes)

	// 支撑阻力位
	support1, resistance1, _, _ := calculateSupportResistance(highs, lows, closes, 20)
	indicators.SupportLevel = support1
	indicators.ResistanceLevel = resistance1

	// 动量背离
	indicators.MomentumDivergence = calculateMomentumDivergence(closes, highs, lows, 14)

	// 波动率
	indicators.Volatility20 = calculateVolatility(closes, 20)

	// 信号强度和风险等级
	indicators.SignalStrength = calculateSignalStrength(indicators)
	indicators.RiskLevel = calculateRiskLevel(indicators.RSI, indicators.BBPosition, indicators.Volatility20)

	return indicators, nil
}

// calculateOverallScore 计算综合评分
func (tips *TechnicalIndicatorsPrecomputeService) calculateOverallScore(indicators TechnicalIndicators) float64 {
	score := 0.0
	weight := 0.0

	// RSI评分 (30-70为理想区间)
	rsiScore := 0.0
	if indicators.RSI >= 30 && indicators.RSI <= 70 {
		rsiScore = 1.0
	} else if indicators.RSI >= 20 && indicators.RSI <= 80 {
		rsiScore = 0.7
	} else {
		rsiScore = 0.3
	}
	score += rsiScore * 0.2
	weight += 0.2

	// 布林带位置评分 (越接近中间越好)
	bbScore := 1.0 - math.Abs(indicators.BBPosition)
	score += bbScore * 0.15
	weight += 0.15

	// 波动率评分 (适中波动率)
	volatilityScore := 0.0
	if indicators.Volatility20 >= 0.01 && indicators.Volatility20 <= 0.05 {
		volatilityScore = 1.0
	} else if indicators.Volatility20 >= 0.005 && indicators.Volatility20 <= 0.1 {
		volatilityScore = 0.7
	} else {
		volatilityScore = 0.3
	}
	score += volatilityScore * 0.15
	weight += 0.15

	// MACD信号评分
	macdScore := 0.0
	if indicators.MACDHist > 0 {
		macdScore = 0.6
	} else {
		macdScore = 0.4
	}
	score += macdScore * 0.1
	weight += 0.1

	// KDJ评分
	kdjScore := 0.0
	if indicators.J >= 20 && indicators.J <= 80 {
		kdjScore = 1.0
	} else if indicators.J >= 10 && indicators.J <= 90 {
		kdjScore = 0.7
	} else {
		kdjScore = 0.3
	}
	score += kdjScore * 0.1
	weight += 0.1

	// 信号强度
	score += indicators.SignalStrength * 0.3
	weight += 0.3

	if weight > 0 {
		return score / weight
	}
	return 0.5
}

// GetIndicators 获取缓存的技术指标
func (tips *TechnicalIndicatorsPrecomputeService) GetIndicators(symbol, timeframe string) *TechnicalIndicators {
	cacheKey := fmt.Sprintf("%s_%s", symbol, timeframe)
	cached := tips.cacheManager.GetIndicators(cacheKey)
	if cached == nil {
		return nil
	}
	return &cached.Indicators
}

// GetStatus 获取服务状态
func (tips *TechnicalIndicatorsPrecomputeService) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"is_running":       tips.isRunning,
		"update_interval":  tips.updateInterval.String(),
		"symbols_count":    len(tips.symbols),
		"timeframes_count": len(tips.timeframes),
		"symbols":          tips.symbols,
		"timeframes":       tips.timeframes,
		"cache_stats":      tips.cacheManager.GetStats(),
		"last_update":      time.Now().Format("2006-01-02 15:04:05"),
	}
}

// GetIndicators 获取缓存的技术指标 (缓存管理器方法)
func (ticm *TechnicalIndicatorsCacheManager) GetIndicators(key string) *CachedTechnicalIndicators {
	ticm.mu.RLock()

	// 首先检查内存缓存
	if cached, exists := ticm.indicatorsCache[key]; exists {
		// 检查是否过期
		if time.Since(cached.ComputedAt) > ticm.maxAge {
			ticm.mu.RUnlock()
			ticm.mu.Lock()
			delete(ticm.indicatorsCache, key)
			ticm.mu.Unlock()
			ticm.missCount++
			return nil
		}

		// 更新访问计数
		cached.AccessCount++
		ticm.hitCount++
		ticm.mu.RUnlock()
		return cached
	}

	ticm.mu.RUnlock()

	// 内存缓存未命中，尝试从数据库获取
	// 解析缓存键格式: symbol_timeframe
	parts := strings.Split(key, "_")
	if len(parts) != 2 {
		ticm.missCount++
		return nil
	}

	symbol := parts[0]
	timeframe := parts[1]

	// 从数据库获取缓存
	dbCache, err := pdb.GetTechnicalIndicatorsCache(ticm.server.db.DB(), symbol, "spot", timeframe, 0)
	if err != nil {
		log.Printf("[TechnicalIndicatorsCache] 从数据库获取缓存失败 %s: %v", key, err)
		ticm.missCount++
		return nil
	}

	if dbCache == nil {
		ticm.missCount++
		return nil
	}

	// 检查数据库缓存是否过期
	if time.Since(dbCache.CalculatedAt) > ticm.maxAge {
		ticm.missCount++
		return nil
	}

	// 从数据库缓存中解析指标数据
	var indicators TechnicalIndicators
	if err := json.Unmarshal(dbCache.Indicators, &indicators); err != nil {
		log.Printf("[TechnicalIndicatorsCache] 解析数据库缓存失败 %s: %v", key, err)
		ticm.missCount++
		return nil
	}

	// 创建缓存对象并存入内存
	cached := &CachedTechnicalIndicators{
		Symbol:      symbol,
		Timeframe:   timeframe,
		Indicators:  indicators,
		ComputedAt:  dbCache.CalculatedAt,
		AccessCount: 1,
	}

	ticm.mu.Lock()
	ticm.indicatorsCache[key] = cached
	ticm.mu.Unlock()

	ticm.hitCount++
	log.Printf("[TechnicalIndicatorsCache] 从数据库加载缓存成功: %s", key)
	return cached
}

// SetIndicators 缓存技术指标
func (ticm *TechnicalIndicatorsCacheManager) SetIndicators(key, symbol, timeframe string, indicators TechnicalIndicators) {
	ticm.mu.Lock()
	defer ticm.mu.Unlock()

	now := time.Now()
	cached := &CachedTechnicalIndicators{
		Symbol:      symbol,
		Timeframe:   timeframe,
		Indicators:  indicators,
		ComputedAt:  now,
		AccessCount: 0,
	}

	ticm.indicatorsCache[key] = cached

	// 清理过期数据
	ticm.cleanupExpiredIndicators()

	// 异步写入数据库
	go func() {
		if err := ticm.saveToDatabase(symbol, timeframe, indicators, now); err != nil {
			log.Printf("[TechnicalIndicatorsCache] 保存到数据库失败 %s_%s: %v", symbol, timeframe, err)
		}
	}()
}

// saveToDatabase 将技术指标保存到数据库
func (ticm *TechnicalIndicatorsCacheManager) saveToDatabase(symbol, timeframe string, indicators TechnicalIndicators, computedAt time.Time) error {
	// 将指标序列化为JSON
	indicatorsJSON, err := json.Marshal(indicators)
	if err != nil {
		return fmt.Errorf("序列化技术指标失败: %w", err)
	}

	// 创建数据库缓存对象
	dbCache := &pdb.TechnicalIndicatorsCache{
		Symbol:       symbol,
		Kind:         "spot", // 预计算服务默认使用现货数据
		Interval:     timeframe,
		DataPoints:   0, // 预计算服务不使用固定的数据点数
		Indicators:   indicatorsJSON,
		CalculatedAt: computedAt,
		DataFrom:     computedAt.Add(-24 * time.Hour), // 简化处理，假设数据范围为24小时
		DataTo:       computedAt,
	}

	// 保存到数据库
	if err := pdb.SaveTechnicalIndicatorsCache(ticm.server.db.DB(), dbCache); err != nil {
		return fmt.Errorf("保存技术指标缓存到数据库失败: %w", err)
	}

	log.Printf("[TechnicalIndicatorsCache] 成功保存到数据库: %s_%s", symbol, timeframe)
	return nil
}

// cleanupExpiredIndicators 清理过期指标
func (ticm *TechnicalIndicatorsCacheManager) cleanupExpiredIndicators() {
	now := time.Now()
	expiredKeys := make([]string, 0)

	for key, cached := range ticm.indicatorsCache {
		if now.Sub(cached.ComputedAt) > ticm.maxAge {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(ticm.indicatorsCache, key)
	}

	// 如果缓存过大，清理最少访问的条目
	if len(ticm.indicatorsCache) > ticm.maxSize {
		ticm.evictLeastAccessed()
	}
}

// evictLeastAccessed 清除最少访问的条目
func (ticm *TechnicalIndicatorsCacheManager) evictLeastAccessed() {
	if len(ticm.indicatorsCache) <= ticm.maxSize {
		return
	}

	// 收集所有条目
	type cacheEntry struct {
		key         string
		accessCount int64
		computedAt  time.Time
	}

	entries := make([]cacheEntry, 0, len(ticm.indicatorsCache))
	for key, cached := range ticm.indicatorsCache {
		entries = append(entries, cacheEntry{
			key:         key,
			accessCount: cached.AccessCount,
			computedAt:  cached.ComputedAt,
		})
	}

	// 按访问次数和计算时间排序（先按访问次数升序，再按时间升序）
	for i := 0; i < len(entries)-1; i++ {
		for j := i + 1; j < len(entries); j++ {
			// 如果访问次数相同，比较计算时间（越早计算的优先清除）
			shouldSwap := false
			if entries[i].accessCount > entries[j].accessCount {
				shouldSwap = true
			} else if entries[i].accessCount == entries[j].accessCount {
				if entries[i].computedAt.Before(entries[j].computedAt) {
					shouldSwap = true
				}
			}

			if shouldSwap {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}

	// 清除最少访问的20%条目
	removeCount := len(entries) / 5
	if removeCount < 1 {
		removeCount = 1
	}

	for i := 0; i < removeCount && i < len(entries); i++ {
		delete(ticm.indicatorsCache, entries[i].key)
	}

	log.Printf("[TechnicalIndicatorsCache] 清除 %d 个最少访问的技术指标缓存条目", removeCount)
}

// GetStats 获取缓存统计信息
func (ticm *TechnicalIndicatorsCacheManager) GetStats() map[string]interface{} {
	ticm.mu.RLock()
	defer ticm.mu.RUnlock()

	hitRate := 0.0
	totalRequests := ticm.hitCount + ticm.missCount
	if totalRequests > 0 {
		hitRate = float64(ticm.hitCount) / float64(totalRequests)
	}

	return map[string]interface{}{
		"cache_size":     len(ticm.indicatorsCache),
		"max_cache_size": ticm.maxSize,
		"cache_max_age":  ticm.maxAge.String(),
		"hit_count":      ticm.hitCount,
		"miss_count":     ticm.missCount,
		"hit_rate":       fmt.Sprintf("%.2f%%", hitRate*100),
	}
}

// CleanupExpiredIndicators 清理过期指标（公开方法）
func (ticm *TechnicalIndicatorsCacheManager) CleanupExpiredIndicators() {
	ticm.mu.Lock()
	defer ticm.mu.Unlock()

	ticm.cleanupExpiredIndicators()
	log.Printf("[TechnicalIndicatorsCache] 完成过期技术指标清理")
}
