package server

import (
	"crypto/md5"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"time"
)

// CacheManager 缓存管理器
type CacheManager struct {
	cache     map[string]interface{}
	expires   map[string]time.Time
	maxSize   int
	mutex     sync.RWMutex
	hitCount  int64
	missCount int64
}

// BacktestDataCache 回测数据缓存系统 - 专门用于回测数据预处理和缓存
type BacktestDataCache struct {
	mu            sync.RWMutex
	processedData map[string]*ProcessedMarketData
	maxSize       int
	maxAge        time.Duration
	hitCount      int64
	missCount     int64
}

// NewCacheManager 创建缓存管理器
func NewCacheManager(maxSize int) *CacheManager {
	return &CacheManager{
		cache:   make(map[string]interface{}),
		expires: make(map[string]time.Time),
		maxSize: maxSize,
	}
}

// generateKey 生成缓存键
func (cm *CacheManager) generateKey(symbol string, startDate, endDate time.Time, dataType string) string {
	key := fmt.Sprintf("%s_%s_%s_%s", symbol, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), dataType)
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// Get 获取缓存数据
func (cm *CacheManager) Get(symbol string, startDate, endDate time.Time, dataType string) (interface{}, bool) {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	key := cm.generateKey(symbol, startDate, endDate, dataType)

	// 检查是否过期
	if expireTime, exists := cm.expires[key]; exists {
		if time.Now().After(expireTime) {
			delete(cm.cache, key)
			delete(cm.expires, key)
			cm.missCount++
			return nil, false
		}
	}

	if data, exists := cm.cache[key]; exists {
		cm.hitCount++
		return data, true
	}

	cm.missCount++
	return nil, false
}

// Set 设置缓存数据
func (cm *CacheManager) Set(symbol string, startDate, endDate time.Time, dataType string, data interface{}, ttl time.Duration) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	key := cm.generateKey(symbol, startDate, endDate, dataType)

	// 检查缓存大小限制
	if len(cm.cache) >= cm.maxSize {
		cm.evictOldest()
	}

	cm.cache[key] = data
	cm.expires[key] = time.Now().Add(ttl)
}

// Delete 删除缓存
func (cm *CacheManager) Delete(symbol string, startDate, endDate time.Time, dataType string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	key := cm.generateKey(symbol, startDate, endDate, dataType)
	delete(cm.cache, key)
	delete(cm.expires, key)
}

// Clear 清空缓存
func (cm *CacheManager) Clear() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	cm.cache = make(map[string]interface{})
	cm.expires = make(map[string]time.Time)
	cm.hitCount = 0
	cm.missCount = 0
}

// evictOldest 淘汰最旧的缓存项
func (cm *CacheManager) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, expireTime := range cm.expires {
		if oldestKey == "" || expireTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = expireTime
		}
	}

	if oldestKey != "" {
		delete(cm.cache, oldestKey)
		delete(cm.expires, oldestKey)
	}
}

// CachedBacktestResult 缓存的回测结果
type CachedBacktestResult struct {
	Result   interface{}   `json:"result"` // 支持多种结果格式
	CachedAt time.Time     `json:"cached_at"`
	TTL      time.Duration `json:"ttl"`
}

// BacktestCacheStats 回测缓存统计信息
type BacktestCacheStats struct {
	Size          int     `json:"size"`
	MaxSize       int     `json:"max_size"`
	HitCount      int64   `json:"hit_count"`
	MissCount     int64   `json:"miss_count"`
	HitRate       float64 `json:"hit_rate"`
	TotalRequests int64   `json:"total_requests"`
}

// GetStats 获取缓存统计信息
func (cm *CacheManager) GetStats() BacktestCacheStats {
	cm.mutex.RLock()
	defer cm.mutex.RUnlock()

	totalRequests := cm.hitCount + cm.missCount
	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(cm.hitCount) / float64(totalRequests)
	}

	return BacktestCacheStats{
		Size:          len(cm.cache),
		MaxSize:       cm.maxSize,
		HitCount:      cm.hitCount,
		MissCount:     cm.missCount,
		HitRate:       hitRate,
		TotalRequests: totalRequests,
	}
}

// ResultCache 结果缓存管理器
type ResultCache struct {
	*CacheManager
	ttl time.Duration
}

// NewResultCache 创建结果缓存
func NewResultCache(maxSize int, ttl time.Duration) *ResultCache {
	return &ResultCache{
		CacheManager: NewCacheManager(maxSize),
		ttl:          ttl,
	}
}

// GetBacktestResult 获取缓存的回测结果
func (rc *ResultCache) GetBacktestResult(config BacktestConfig) (interface{}, bool) {
	key := rc.generateBacktestKey(config)
	data, exists := rc.Get(config.Symbol, config.StartDate, config.EndDate, "backtest_"+key)
	if !exists {
		return nil, false
	}

	if cachedResult, ok := data.(*CachedBacktestResult); ok {
		// 检查是否在TTL内
		if time.Since(cachedResult.CachedAt) < cachedResult.TTL {
			return cachedResult.Result, true
		}
		// TTL过期，删除缓存
		rc.Delete(config.Symbol, config.StartDate, config.EndDate, "backtest_"+key)
	}

	return nil, false
}

// SetBacktestResult 缓存回测结果
func (rc *ResultCache) SetBacktestResult(config BacktestConfig, result *BacktestResult) {
	key := rc.generateBacktestKey(config)
	cachedResult := &CachedBacktestResult{
		Result:   result,
		CachedAt: time.Now(),
		TTL:      rc.ttl,
	}

	rc.Set(config.Symbol, config.StartDate, config.EndDate, "backtest_"+key, cachedResult, rc.ttl)
}

// generateBacktestKey 生成回测缓存键
func (rc *ResultCache) generateBacktestKey(config BacktestConfig) string {
	// 基于配置参数生成唯一键
	key := fmt.Sprintf("strategy:%s_maxpos:%.2f_stoploss:%.2f_takeprofit:%.2f_commission:%.4f",
		config.Strategy, config.MaxPosition, config.StopLoss, config.TakeProfit, config.Commission)
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("%x", hash)
}

// WarmUpCache 预热缓存
func (rc *ResultCache) WarmUpCache(configs []BacktestConfig, backtestFunc func(BacktestConfig) (*BacktestResult, error)) {
	for _, config := range configs {
		// 检查是否已缓存
		if _, exists := rc.GetBacktestResult(config); !exists {
			// 执行回测并缓存
			if result, err := backtestFunc(config); err == nil {
				rc.SetBacktestResult(config, result)
			}
		}
	}
}

// ProcessedMarketData 预处理后的市场数据
type ProcessedMarketData struct {
	RawData       []MarketData         `json:"raw_data"`
	ProcessedData []MarketData         `json:"processed_data"`
	Indicators    map[string][]float64 `json:"indicators"`
	Quality       BacktestDataQuality  `json:"quality"`
	ProcessedAt   time.Time            `json:"processed_at"`
	Version       int                  `json:"version"`
}

// BacktestDataQuality 回测数据质量指标
type BacktestDataQuality struct {
	Completeness float64 `json:"completeness"` // 数据完整性 0-1
	Consistency  float64 `json:"consistency"`  // 数据一致性 0-1
	Accuracy     float64 `json:"accuracy"`     // 数据准确性 0-1
	Overall      float64 `json:"overall"`      // 整体质量评分 0-1
}

// NewBacktestDataCache 创建回测数据缓存实例
func NewBacktestDataCache() *BacktestDataCache {
	return &BacktestDataCache{
		processedData: make(map[string]*ProcessedMarketData),
		maxSize:       1000,           // 最大缓存1000个条目
		maxAge:        24 * time.Hour, // 缓存24小时
	}
}

// GetMarketData 获取缓存的市场数据
func (dc *BacktestDataCache) GetMarketData(symbol string, startDate, endDate time.Time) ([]MarketData, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	key := dc.generateMarketDataKey(symbol, startDate, endDate)

	if processedData, exists := dc.processedData[key]; exists {
		// 检查是否过期
		if time.Since(processedData.ProcessedAt) > dc.maxAge {
			delete(dc.processedData, key)
			dc.missCount++
			return nil, false
		}

		dc.hitCount++
		return processedData.ProcessedData, true
	}

	dc.missCount++
	return nil, false
}

// SetMarketData 缓存市场数据
func (dc *BacktestDataCache) SetMarketData(symbol string, startDate, endDate time.Time, data []MarketData) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	key := dc.generateMarketDataKey(symbol, startDate, endDate)

	// 预处理数据
	processedData := dc.preprocessAndCacheData(symbol, startDate, endDate, data)

	dc.processedData[key] = processedData

	// 清理过期数据
	dc.cleanupExpiredData()
}

// GetIndicatorData 获取缓存的指标数据
func (dc *BacktestDataCache) GetIndicatorData(symbol string, startDate, endDate time.Time, indicator string) (map[string][]float64, bool) {
	dc.mu.RLock()
	defer dc.mu.RUnlock()

	key := dc.generateIndicatorKey(symbol, startDate, endDate, indicator)

	if processedData, exists := dc.processedData[key]; exists {
		if time.Since(processedData.ProcessedAt) > dc.maxAge {
			delete(dc.processedData, key)
			dc.missCount++
			return nil, false
		}

		dc.hitCount++
		return processedData.Indicators, true
	}

	dc.missCount++
	return nil, false
}

// SetIndicatorData 缓存指标数据
func (dc *BacktestDataCache) SetIndicatorData(symbol string, startDate, endDate time.Time, indicator string, data map[string][]float64) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	key := dc.generateIndicatorKey(symbol, startDate, endDate, indicator)

	processedData := &ProcessedMarketData{
		Indicators:  data,
		ProcessedAt: time.Now(),
		Version:     1,
	}

	dc.processedData[key] = processedData
	dc.cleanupExpiredData()
}

// preprocessAndCacheData 预处理并缓存数据
func (dc *BacktestDataCache) preprocessAndCacheData(symbol string, startDate, endDate time.Time, rawData []MarketData) *ProcessedMarketData {
	// 创建数据预处理器
	preprocessor := NewDataPreprocessor()

	// 1. 检测和处理异常值
	outliers := preprocessor.DetectOutliers(rawData, "iqr")
	processedData := rawData
	if len(outliers) > 0 {
		log.Printf("[DataCache] 检测到 %d 个异常值，使用 capping 方法处理", len(outliers))
		processedData = preprocessor.HandleOutliers(rawData, outliers, "cap")
	}

	// 2. 填充缺失数据
	processedData = preprocessor.FillMissingData(processedData, "forward_fill")

	// 3. 计算基础指标
	indicators := dc.calculateBasicIndicators(processedData)

	// 4. 评估数据质量
	quality := dc.assessDataQuality(rawData, processedData, outliers)

	return &ProcessedMarketData{
		RawData:       rawData,
		ProcessedData: processedData,
		Indicators:    indicators,
		Quality:       quality,
		ProcessedAt:   time.Now(),
		Version:       1,
	}
}

// calculateBasicIndicators 计算基础技术指标
func (dc *BacktestDataCache) calculateBasicIndicators(data []MarketData) map[string][]float64 {
	indicators := make(map[string][]float64)

	if len(data) < 50 {
		return indicators // 数据不足，无法计算指标
	}

	// 提取价格序列
	prices := make([]float64, len(data))
	volumes := make([]float64, len(data))

	for i, md := range data {
		prices[i] = md.Price
		volumes[i] = md.Volume24h
	}

	// 计算SMA
	indicators["sma_20"] = dc.calculateSMA(prices, 20)
	indicators["sma_50"] = dc.calculateSMA(prices, 50)

	// 计算EMA
	indicators["ema_12"] = dc.calculateEMA(prices, 12)
	indicators["ema_26"] = dc.calculateEMA(prices, 26)

	// 计算RSI
	indicators["rsi_14"] = dc.calculateRSI(prices, 14)

	// 计算MACD
	macd, signal, histogram := dc.calculateMACD(prices)
	indicators["macd"] = macd
	indicators["macd_signal"] = signal
	indicators["macd_histogram"] = histogram

	// 计算布林带
	middle, upper, lower := dc.calculateBollingerBands(prices, 20, 2.0)
	indicators["bb_middle"] = middle
	indicators["bb_upper"] = upper
	indicators["bb_lower"] = lower

	return indicators
}

// assessDataQuality 评估数据质量
func (dc *BacktestDataCache) assessDataQuality(rawData, processedData []MarketData, outliers []OutlierInfo) BacktestDataQuality {
	quality := BacktestDataQuality{}

	// 1. 完整性 - 检查缺失数据比例
	totalPoints := len(rawData)
	missingCount := 0
	for _, md := range rawData {
		if md.Price == 0 || md.Volume24h == 0 {
			missingCount++
		}
	}
	quality.Completeness = 1.0 - float64(missingCount)/float64(totalPoints)

	// 2. 一致性 - 检查数据波动合理性
	if len(rawData) > 1 {
		priceChanges := make([]float64, len(rawData)-1)
		for i := 1; i < len(rawData); i++ {
			if rawData[i-1].Price > 0 {
				priceChanges[i-1] = math.Abs(rawData[i].Price-rawData[i-1].Price) / rawData[i-1].Price
			}
		}
		// 计算价格变化的标准差
		_, std := dc.calculateMeanStd(priceChanges)
		// 如果标准差太大（>50%），认为数据不一致
		if std > 0.5 {
			quality.Consistency = 0.5
		} else {
			quality.Consistency = 1.0 - (std / 0.5)
		}
	} else {
		quality.Consistency = 1.0
	}

	// 3. 准确性 - 基于异常值比例
	outlierRatio := float64(len(outliers)) / float64(len(rawData))
	quality.Accuracy = 1.0 - outlierRatio

	// 4. 整体质量评分
	quality.Overall = (quality.Completeness*0.3 + quality.Consistency*0.3 + quality.Accuracy*0.4)

	return quality
}

// 辅助函数
func (dc *BacktestDataCache) generateMarketDataKey(symbol string, startDate, endDate time.Time) string {
	return fmt.Sprintf("market_data:%s:%s:%s",
		symbol,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"))
}

func (dc *BacktestDataCache) generateIndicatorKey(symbol string, startDate, endDate time.Time, indicator string) string {
	return fmt.Sprintf("indicator:%s:%s:%s:%s",
		symbol,
		startDate.Format("2006-01-02"),
		endDate.Format("2006-01-02"),
		indicator)
}

func (dc *BacktestDataCache) cleanupExpiredData() {
	now := time.Now()
	for key, data := range dc.processedData {
		if now.Sub(data.ProcessedAt) > dc.maxAge {
			delete(dc.processedData, key)
		}
	}

	// 限制缓存大小
	if len(dc.processedData) > dc.maxSize {
		// 简单的LRU策略：删除最旧的数据
		type cacheEntry struct {
			key  string
			time time.Time
		}

		entries := make([]cacheEntry, 0, len(dc.processedData))
		for key, data := range dc.processedData {
			entries = append(entries, cacheEntry{key: key, time: data.ProcessedAt})
		}

		// 按时间排序
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].time.Before(entries[j].time)
		})

		// 删除最旧的50%数据
		removeCount := len(entries) / 2
		for i := 0; i < removeCount; i++ {
			delete(dc.processedData, entries[i].key)
		}
	}
}

// 技术指标计算函数
func (dc *BacktestDataCache) calculateSMA(prices []float64, period int) []float64 {
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

func (dc *BacktestDataCache) calculateEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	ema := make([]float64, len(prices))
	multiplier := 2.0 / (float64(period) + 1.0)

	// 第一个EMA值使用SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)

	// 计算后续EMA值
	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i]-ema[i-1])*multiplier + ema[i-1]
	}

	return ema
}

func (dc *BacktestDataCache) calculateRSI(prices []float64, period int) []float64 {
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

func (dc *BacktestDataCache) calculateMACD(prices []float64) ([]float64, []float64, []float64) {
	ema12 := dc.calculateEMA(prices, 12)
	ema26 := dc.calculateEMA(prices, 26)

	if len(ema12) < len(ema26) || len(ema26) == 0 {
		return []float64{}, []float64{}, []float64{}
	}

	// MACD线
	macd := make([]float64, len(ema26))
	for i := 0; i < len(ema26); i++ {
		macd[i] = ema12[i+len(ema12)-len(ema26)] - ema26[i]
	}

	// 信号线
	signal := dc.calculateEMA(macd, 9)

	// 直方图
	histogram := make([]float64, len(signal))
	for i := 0; i < len(signal); i++ {
		histogram[i] = macd[i+len(macd)-len(signal)] - signal[i]
	}

	return macd, signal, histogram
}

func (dc *BacktestDataCache) calculateBollingerBands(prices []float64, period int, stdDev float64) ([]float64, []float64, []float64) {
	if len(prices) < period {
		return []float64{}, []float64{}, []float64{}
	}

	middle := dc.calculateSMA(prices, period)
	upper := make([]float64, len(middle))
	lower := make([]float64, len(middle))

	for i := period - 1; i < len(prices); i++ {
		// 计算标准差
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			diff := prices[j] - middle[i]
			sum += diff * diff
		}
		std := math.Sqrt(sum / float64(period))

		upper[i] = middle[i] + stdDev*std
		lower[i] = middle[i] - stdDev*std
	}

	return middle, upper, lower
}

func (dc *BacktestDataCache) calculateMeanStd(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}

	// 计算均值
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(len(data))

	// 计算标准差
	sumSq := 0.0
	for _, v := range data {
		diff := v - mean
		sumSq += diff * diff
	}
	std := math.Sqrt(sumSq / float64(len(data)))

	return mean, std
}
