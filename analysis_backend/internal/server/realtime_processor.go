package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// RealTimeDataProcessor 实时数据处理器
type RealTimeDataProcessor struct {
	dataChannels   map[string]chan DataPoint
	processingPool *WorkerPool
	mu             sync.RWMutex

	// 内存缓存
	cache map[string]*CachedDataPoint

	// 数据质量监控
	qualityMonitor *RealTimeDataQualityMonitor

	// 实时聚合器
	aggregators map[string]*DataAggregator

	// 事件处理器
	eventHandlers map[string][]DataEventHandler
}

// CachedDataPoint 缓存的数据点
type CachedDataPoint struct {
	Point     DataPoint
	Timestamp time.Time
	TTL       time.Duration
}

// RealTimeDataQualityMonitor 实时数据质量监控器
type RealTimeDataQualityMonitor struct {
	metrics map[string]*RealTimeDataQualityMetrics
	mu      sync.RWMutex
}

type RealTimeDataQualityMetrics struct {
	TotalPoints    int64
	ValidPoints    int64
	InvalidPoints  int64
	LastUpdateTime time.Time
	AvgLatency     time.Duration
	ErrorRate      float64
}

// NewRealTimeDataQualityMonitor 创建实时数据质量监控器
func NewRealTimeDataQualityMonitor() *RealTimeDataQualityMonitor {
	return &RealTimeDataQualityMonitor{
		metrics: make(map[string]*RealTimeDataQualityMetrics),
	}
}

// ValidateDataPoint 验证数据点质量
func (dqm *RealTimeDataQualityMonitor) ValidateDataPoint(point DataPoint) error {
	// 基本验证
	if point.Symbol == "" {
		return fmt.Errorf("empty symbol")
	}
	if point.Value == 0 && point.DataType == "price" {
		return fmt.Errorf("zero price value")
	}
	if point.Timestamp.IsZero() {
		return fmt.Errorf("invalid timestamp")
	}

	return nil
}

// UpdateMetrics 更新质量指标
func (dqm *RealTimeDataQualityMonitor) UpdateMetrics(point DataPoint) {
	dqm.mu.Lock()
	defer dqm.mu.Unlock()

	key := point.Symbol + ":" + point.DataType
	if dqm.metrics[key] == nil {
		dqm.metrics[key] = &RealTimeDataQualityMetrics{}
	}

	metrics := dqm.metrics[key]
	metrics.TotalPoints++
	metrics.ValidPoints++ // 假设通过了验证
	metrics.LastUpdateTime = time.Now()

	if metrics.TotalPoints > 0 {
		metrics.ErrorRate = float64(metrics.InvalidPoints) / float64(metrics.TotalPoints)
	}
}

// GenerateReport 生成质量报告
func (dqm *RealTimeDataQualityMonitor) GenerateReport() DataQualityReport {
	dqm.mu.RLock()
	defer dqm.mu.RUnlock()

	totalPoints := int64(0)
	validPoints := int64(0)
	avgErrorRate := 0.0
	count := 0

	for _, metrics := range dqm.metrics {
		totalPoints += metrics.TotalPoints
		validPoints += metrics.ValidPoints
		avgErrorRate += metrics.ErrorRate
		count++
	}

	if count > 0 {
		avgErrorRate /= float64(count)
	}

	overallQuality := 1.0
	if totalPoints > 0 {
		overallQuality = float64(validPoints) / float64(totalPoints)
	}

	return DataQualityReport{
		TotalDataPoints: totalPoints,
		ValidDataPoints: validPoints,
		OverallQuality:  overallQuality,
		AvgErrorRate:    avgErrorRate,
		MetricsCount:    len(dqm.metrics),
		GeneratedAt:     time.Now(),
	}
}

type DataQualityReport struct {
	TotalDataPoints int64     `json:"total_data_points"`
	ValidDataPoints int64     `json:"valid_data_points"`
	OverallQuality  float64   `json:"overall_quality"`
	AvgErrorRate    float64   `json:"avg_error_rate"`
	MetricsCount    int       `json:"metrics_count"`
	GeneratedAt     time.Time `json:"generated_at"`
}

// DataPoint 数据点
type DataPoint struct {
	Symbol    string                 `json:"symbol"`
	DataType  string                 `json:"data_type"` // price, volume, sentiment, etc.
	Value     float64                `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// DataAggregator 数据聚合器
type DataAggregator struct {
	symbol     string
	dataType   string
	windowSize time.Duration
	buffer     []DataPoint
	lastEmit   time.Time
	mu         sync.RWMutex
}

// DataEventHandler 数据事件处理器接口
type DataEventHandler interface {
	HandleDataPoint(point DataPoint) error
	GetName() string
}

// NewRealTimeDataProcessor 创建实时数据处理器
func NewRealTimeDataProcessor() (*RealTimeDataProcessor, error) {
	// 初始化工作池
	pool := NewWorkerPool(10)

	processor := &RealTimeDataProcessor{
		dataChannels:   make(map[string]chan DataPoint),
		cache:          make(map[string]*CachedDataPoint),
		processingPool: pool,
		qualityMonitor: NewRealTimeDataQualityMonitor(),
		aggregators:    make(map[string]*DataAggregator),
		eventHandlers:  make(map[string][]DataEventHandler),
	}

	// 启动后台任务
	go processor.startDataAggregation()
	go processor.startQualityMonitoring()
	go processor.startCacheCleanup()

	return processor, nil
}

// ProcessDataPoint 处理单个数据点
func (rtp *RealTimeDataProcessor) ProcessDataPoint(ctx context.Context, point DataPoint) error {
	// 数据质量检查
	if err := rtp.qualityMonitor.ValidateDataPoint(point); err != nil {
		log.Printf("Data quality check failed: %v", err)
		return err
	}

	// 实时缓存到内存
	rtp.cacheToMemory(point)

	// 分发到数据通道
	rtp.distributeToChannels(point)

	// 触发事件处理器
	rtp.triggerEventHandlers(point)

	// 提交到处理池进行异步处理
	rtp.processingPool.Submit(func() {
		rtp.processDataPointAsync(point)
	})

	return nil
}

// cacheToMemory 内存缓存数据
func (rtp *RealTimeDataProcessor) cacheToMemory(point DataPoint) {
	rtp.mu.Lock()
	defer rtp.mu.Unlock()

	key := getCacheKey(point.Symbol, point.DataType)

	rtp.cache[key] = &CachedDataPoint{
		Point:     point,
		Timestamp: time.Now(),
		TTL:       24 * time.Hour, // 24小时TTL
	}
}

// startCacheCleanup 启动缓存清理任务
func (rtp *RealTimeDataProcessor) startCacheCleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		rtp.mu.Lock()
		now := time.Now()

		for key, cached := range rtp.cache {
			if now.Sub(cached.Timestamp) > cached.TTL {
				delete(rtp.cache, key)
			}
		}

		rtp.mu.Unlock()
	}
}

// GetCachedData 获取缓存的数据
func (rtp *RealTimeDataProcessor) GetCachedData(symbol, dataType string) (*DataPoint, bool) {
	rtp.mu.RLock()
	defer rtp.mu.RUnlock()

	key := getCacheKey(symbol, dataType)
	if cached, exists := rtp.cache[key]; exists {
		// 检查是否过期
		if time.Since(cached.Timestamp) < cached.TTL {
			return &cached.Point, true
		}
	}
	return nil, false
}

// distributeToChannels 分发数据到通道
func (rtp *RealTimeDataProcessor) distributeToChannels(point DataPoint) {
	rtp.mu.RLock()
	defer rtp.mu.RUnlock()

	channelKey := point.Symbol + ":" + point.DataType
	if ch, exists := rtp.dataChannels[channelKey]; exists {
		select {
		case ch <- point:
		default:
			// 通道已满，丢弃旧数据
			log.Printf("Data channel full for %s, dropping data", channelKey)
		}
	}
}

// SubscribeDataChannel 订阅数据通道
func (rtp *RealTimeDataProcessor) SubscribeDataChannel(symbol, dataType string) <-chan DataPoint {
	rtp.mu.Lock()
	defer rtp.mu.Unlock()

	channelKey := symbol + ":" + dataType
	if _, exists := rtp.dataChannels[channelKey]; !exists {
		rtp.dataChannels[channelKey] = make(chan DataPoint, 100)
	}

	return rtp.dataChannels[channelKey]
}

// processDataPointAsync 异步处理数据点
func (rtp *RealTimeDataProcessor) processDataPointAsync(point DataPoint) {
	// 添加到聚合器
	rtp.addToAggregator(point)

	// 触发实时计算
	rtp.triggerRealTimeCalculations(point)

	// 更新数据质量指标
	rtp.qualityMonitor.UpdateMetrics(point)
}

// addToAggregator 添加到聚合器
func (rtp *RealTimeDataProcessor) addToAggregator(point DataPoint) {
	aggKey := point.Symbol + ":" + point.DataType

	rtp.mu.Lock()
	aggregator, exists := rtp.aggregators[aggKey]
	if !exists {
		aggregator = &DataAggregator{
			symbol:     point.Symbol,
			dataType:   point.DataType,
			windowSize: 1 * time.Minute, // 1分钟聚合窗口
			buffer:     make([]DataPoint, 0, 100),
		}
		rtp.aggregators[aggKey] = aggregator
	}
	rtp.mu.Unlock()

	aggregator.mu.Lock()
	defer aggregator.mu.Unlock()

	// 添加到缓冲区
	aggregator.buffer = append(aggregator.buffer, point)

	// 检查是否需要发射聚合数据
	now := time.Now()
	if now.Sub(aggregator.lastEmit) >= aggregator.windowSize {
		rtp.emitAggregatedData(aggregator)
		aggregator.buffer = aggregator.buffer[:0] // 清空缓冲区
		aggregator.lastEmit = now
	}
}

// emitAggregatedData 发射聚合数据
func (rtp *RealTimeDataProcessor) emitAggregatedData(aggregator *DataAggregator) {
	if len(aggregator.buffer) == 0 {
		return
	}

	// 计算聚合统计
	sum := 0.0
	min := aggregator.buffer[0].Value
	max := aggregator.buffer[0].Value

	for _, point := range aggregator.buffer {
		sum += point.Value
		if point.Value < min {
			min = point.Value
		}
		if point.Value > max {
			max = point.Value
		}
	}

	avg := sum / float64(len(aggregator.buffer))

	// 创建聚合数据点
	aggregatedPoint := DataPoint{
		Symbol:    aggregator.symbol,
		DataType:  aggregator.dataType + "_aggregated",
		Value:     avg,
		Timestamp: time.Now(),
		Source:    "aggregator",
		Metadata: map[string]interface{}{
			"count": len(aggregator.buffer),
			"sum":   sum,
			"avg":   avg,
			"min":   min,
			"max":   max,
		},
	}

	// 发送聚合数据
	rtp.distributeToChannels(aggregatedPoint)
}

// startDataAggregation 启动数据聚合任务
func (rtp *RealTimeDataProcessor) startDataAggregation() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		rtp.mu.RLock()
		aggregators := make([]*DataAggregator, 0, len(rtp.aggregators))
		for _, agg := range rtp.aggregators {
			aggregators = append(aggregators, agg)
		}
		rtp.mu.RUnlock()

		// 检查每个聚合器是否需要发射数据
		now := time.Now()
		for _, aggregator := range aggregators {
			aggregator.mu.RLock()
			shouldEmit := len(aggregator.buffer) > 0 &&
				now.Sub(aggregator.lastEmit) >= aggregator.windowSize
			aggregator.mu.RUnlock()

			if shouldEmit {
				aggregator.mu.Lock()
				rtp.emitAggregatedData(aggregator)
				aggregator.buffer = aggregator.buffer[:0]
				aggregator.lastEmit = now
				aggregator.mu.Unlock()
			}
		}
	}
}

// startQualityMonitoring 启动质量监控
func (rtp *RealTimeDataProcessor) startQualityMonitoring() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		report := rtp.qualityMonitor.GenerateReport()
		log.Printf("Data Quality Report: %+v", report)

		// 可以发送到监控系统或告警
		if report.OverallQuality < 0.8 {
			log.Printf("WARNING: Data quality dropped below 80%%")
		}
	}
}

// RegisterEventHandler 注册事件处理器
func (rtp *RealTimeDataProcessor) RegisterEventHandler(dataType string, handler DataEventHandler) {
	rtp.mu.Lock()
	defer rtp.mu.Unlock()

	rtp.eventHandlers[dataType] = append(rtp.eventHandlers[dataType], handler)
}

// triggerEventHandlers 触发事件处理器
func (rtp *RealTimeDataProcessor) triggerEventHandlers(point DataPoint) {
	rtp.mu.RLock()
	handlers := rtp.eventHandlers[point.DataType]
	rtp.mu.RUnlock()

	for _, handler := range handlers {
		go func(h DataEventHandler) {
			if err := h.HandleDataPoint(point); err != nil {
				log.Printf("Event handler %s failed: %v", h.GetName(), err)
			}
		}(handler)
	}
}

// triggerRealTimeCalculations 触发实时计算
func (rtp *RealTimeDataProcessor) triggerRealTimeCalculations(point DataPoint) {
	// 这里可以触发各种实时计算任务
	// 例如：实时指标计算、异常检测、预测更新等

	switch point.DataType {
	case "price":
		rtp.updatePriceBasedCalculations(point)
	case "volume":
		rtp.updateVolumeBasedCalculations(point)
	case "sentiment":
		rtp.updateSentimentBasedCalculations(point)
	}
}

// updatePriceBasedCalculations 更新基于价格的计算
func (rtp *RealTimeDataProcessor) updatePriceBasedCalculations(point DataPoint) {
	// 计算实时收益率、波动率等
	// 这里可以集成到现有的推荐系统
}

// updateVolumeBasedCalculations 更新基于交易量的计算
func (rtp *RealTimeDataProcessor) updateVolumeBasedCalculations(point DataPoint) {
	// 计算流动性指标、交易活跃度等
}

// updateSentimentBasedCalculations 更新基于情绪的计算
func (rtp *RealTimeDataProcessor) updateSentimentBasedCalculations(point DataPoint) {
	// 更新实时情绪指标
}

// Close 关闭处理器
func (rtp *RealTimeDataProcessor) Close() error {
	rtp.processingPool.Shutdown(30 * time.Second)

	rtp.mu.Lock()
	defer rtp.mu.Unlock()

	// 关闭所有数据通道
	for _, ch := range rtp.dataChannels {
		close(ch)
	}

	return nil
}

// 辅助函数
func getCacheKey(symbol, dataType string) string {
	return fmt.Sprintf("realtime:%s:%s", symbol, dataType)
}
