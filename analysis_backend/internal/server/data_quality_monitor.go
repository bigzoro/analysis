package server

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	pdb "analysis/internal/db"
)

// DataQualityMetrics 数据质量指标
type DataQualityMetrics struct {
	// 基础数据质量
	Completeness float64 `json:"completeness"` // 数据完整性 (0-100)
	Freshness    int64   `json:"freshness"`    // 数据新鲜度(秒)
	Accuracy     float64 `json:"accuracy"`     // 数据准确性 (0-100)
	Consistency  float64 `json:"consistency"`  // 数据一致性 (0-100)

	// 数据源状态
	DataSources map[string]*DataSourceStatus `json:"data_sources"`

	// 异常统计
	Anomalies []DataAnomaly `json:"anomalies"`

	// 时间戳
	Timestamp time.Time `json:"timestamp"`
}

// DataSourceStatus 数据源状态
type DataSourceStatus struct {
	Name         string    `json:"name"`
	Status       string    `json:"status"` // "healthy", "degraded", "failed"
	LastUpdate   time.Time `json:"last_update"`
	ErrorCount   int       `json:"error_count"`
	ResponseTime int64     `json:"response_time_ms"`
	DataQuality  float64   `json:"data_quality"`
	SampleSize   int       `json:"sample_size"`
}

// DataAnomaly 数据异常
type DataAnomaly struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"` // "low", "medium", "high", "critical"
	Description string    `json:"description"`
	Symbol      string    `json:"symbol,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
}

// DataQualityMonitor 数据质量监控器
type DataQualityMonitor struct {
	db                 Database
	metrics            *DataQualityMetrics
	mu                 sync.RWMutex
	monitoringInterval time.Duration
	alertThresholds    AlertThresholds
	alertCallbacks     []AlertCallback
	isRunning          bool
	stopChan           chan struct{}
}

// AlertThresholds 告警阈值
type AlertThresholds struct {
	MaxFreshnessSeconds    int64   `json:"max_freshness_seconds"`
	MinCompletenessPercent float64 `json:"min_completeness_percent"`
	MaxErrorRatePercent    float64 `json:"max_error_rate_percent"`
	MinAccuracyPercent     float64 `json:"min_accuracy_percent"`
}

// AlertCallback 告警回调
type AlertCallback func(anomaly DataAnomaly)

// NewDataQualityMonitor 创建数据质量监控器
func NewDataQualityMonitor(db Database, thresholds AlertThresholds) *DataQualityMonitor {
	return &DataQualityMonitor{
		db:                 db,
		metrics:            &DataQualityMetrics{},
		monitoringInterval: 5 * time.Minute, // 每5分钟检查一次
		alertThresholds:    thresholds,
		alertCallbacks:     make([]AlertCallback, 0),
		stopChan:           make(chan struct{}),
	}
}

// StartMonitoring 启动监控
func (dqm *DataQualityMonitor) StartMonitoring() {
	dqm.isRunning = true
	log.Printf("[DataQualityMonitor] 启动数据质量监控，间隔: %v", dqm.monitoringInterval)

	// 首次运行
	go dqm.collectMetrics()

	// 定时运行
	ticker := time.NewTicker(dqm.monitoringInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			go dqm.collectMetrics()
		case <-dqm.stopChan:
			log.Printf("[DataQualityMonitor] 停止数据质量监控")
			dqm.isRunning = false
			return
		}
	}
}

// StopMonitoring 停止监控
func (dqm *DataQualityMonitor) StopMonitoring() {
	if dqm.isRunning {
		close(dqm.stopChan)
	}
}

// AddAlertCallback 添加告警回调
func (dqm *DataQualityMonitor) AddAlertCallback(callback AlertCallback) {
	dqm.mu.Lock()
	defer dqm.mu.Unlock()
	dqm.alertCallbacks = append(dqm.alertCallbacks, callback)
}

// GetMetrics 获取当前指标
func (dqm *DataQualityMonitor) GetMetrics() *DataQualityMetrics {
	dqm.mu.RLock()
	defer dqm.mu.RUnlock()
	return dqm.metrics
}

// collectMetrics 收集数据质量指标
func (dqm *DataQualityMonitor) collectMetrics() {
	log.Printf("[DataQualityMonitor] 开始收集数据质量指标")

	ctx := context.Background()
	newMetrics := &DataQualityMetrics{
		DataSources: make(map[string]*DataSourceStatus),
		Anomalies:   make([]DataAnomaly, 0),
		Timestamp:   time.Now(),
	}

	// 检查各数据源状态
	dqm.checkMarketDataQuality(ctx, newMetrics)
	dqm.checkTechnicalDataQuality(ctx, newMetrics)
	dqm.checkSentimentDataQuality(ctx, newMetrics)
	dqm.checkFlowDataQuality(ctx, newMetrics)
	dqm.checkAnnouncementDataQuality(ctx, newMetrics)

	// 计算总体指标
	dqm.calculateOverallMetrics(newMetrics)

	// 检查告警阈值
	dqm.checkAlertThresholds(newMetrics)

	// 更新指标
	dqm.mu.Lock()
	dqm.metrics = newMetrics
	dqm.mu.Unlock()

	log.Printf("[DataQualityMonitor] 数据质量指标更新完成 - 完整性: %.1f%%, 新鲜度: %ds",
		newMetrics.Completeness, newMetrics.Freshness)
}

// checkMarketDataQuality 检查市场数据质量
func (dqm *DataQualityMonitor) checkMarketDataQuality(ctx context.Context, metrics *DataQualityMetrics) {
	status := &DataSourceStatus{
		Name:       "market_data",
		Status:     "healthy",
		LastUpdate: time.Now(),
	}

	// 检查最近的市场数据
	var count int64
	err := dqm.db.DB().Model(&pdb.BinanceMarketTop{}).Count(&count).Error
	if err != nil {
		status.Status = "failed"
		status.ErrorCount++
		metrics.Anomalies = append(metrics.Anomalies, DataAnomaly{
			Type:        "database_error",
			Severity:    "high",
			Description: fmt.Sprintf("市场数据查询失败: %v", err),
			Timestamp:   time.Now(),
		})
	} else {
		status.SampleSize = int(count)

		// 检查数据新鲜度
		var latestRecord pdb.BinanceMarketTop
		err := dqm.db.DB().Order("created_at DESC").First(&latestRecord).Error
		if err == nil {
			freshness := time.Since(latestRecord.CreatedAt).Seconds()
			if freshness > 3600 { // 1小时
				status.Status = "degraded"
				metrics.Anomalies = append(metrics.Anomalies, DataAnomaly{
					Type:        "stale_data",
					Severity:    "medium",
					Description: fmt.Sprintf("市场数据过期: %.0f秒", freshness),
					Timestamp:   time.Now(),
				})
			}
		}
	}

	metrics.DataSources["market_data"] = status
}

// checkTechnicalDataQuality 检查技术指标数据质量
func (dqm *DataQualityMonitor) checkTechnicalDataQuality(ctx context.Context, metrics *DataQualityMetrics) {
	status := &DataSourceStatus{
		Name:       "technical_data",
		Status:     "healthy",
		LastUpdate: time.Now(),
	}

	// 检查技术指标缓存
	var count int64
	err := dqm.db.DB().Model(&pdb.TechnicalIndicatorsCache{}).Count(&count).Error
	if err != nil {
		status.Status = "failed"
		status.ErrorCount++
	} else {
		status.SampleSize = int(count)
		if count == 0 {
			status.Status = "degraded"
			metrics.Anomalies = append(metrics.Anomalies, DataAnomaly{
				Type:        "missing_data",
				Severity:    "medium",
				Description: "技术指标数据为空",
				Timestamp:   time.Now(),
			})
		}
	}

	metrics.DataSources["technical_data"] = status
}

// checkSentimentDataQuality 检查情绪数据质量
func (dqm *DataQualityMonitor) checkSentimentDataQuality(ctx context.Context, metrics *DataQualityMetrics) {
	status := &DataSourceStatus{
		Name:       "sentiment_data",
		Status:     "healthy",
		LastUpdate: time.Now(),
	}

	// 检查Twitter数据
	var count int64
	err := dqm.db.DB().Model(&pdb.TwitterPost{}).Count(&count).Error
	if err != nil {
		status.Status = "failed"
		status.ErrorCount++
	} else {
		status.SampleSize = int(count)

		// 检查最近24小时的数据
		since := time.Now().Add(-24 * time.Hour)
		var recentCount int64
		dqm.db.DB().Model(&pdb.TwitterPost{}).Where("tweet_time >= ?", since).Count(&recentCount)

		if recentCount < 10 { // 24小时内少于10条推文
			status.Status = "degraded"
			metrics.Anomalies = append(metrics.Anomalies, DataAnomaly{
				Type:        "insufficient_data",
				Severity:    "low",
				Description: fmt.Sprintf("24小时内推文数量不足: %d", recentCount),
				Timestamp:   time.Now(),
			})
		}
	}

	metrics.DataSources["sentiment_data"] = status
}

// checkFlowDataQuality 检查资金流数据质量
func (dqm *DataQualityMonitor) checkFlowDataQuality(ctx context.Context, metrics *DataQualityMetrics) {
	status := &DataSourceStatus{
		Name:       "flow_data",
		Status:     "degraded", // 当前实现返回默认值
		LastUpdate: time.Now(),
		SampleSize: 0,
	}

	// 当前资金流数据实现不完整，标记为降级状态
	metrics.Anomalies = append(metrics.Anomalies, DataAnomaly{
		Type:        "missing_implementation",
		Severity:    "high",
		Description: "资金流数据功能未实现，返回默认值",
		Timestamp:   time.Now(),
	})

	metrics.DataSources["flow_data"] = status
}

// checkAnnouncementDataQuality 检查公告数据质量
func (dqm *DataQualityMonitor) checkAnnouncementDataQuality(ctx context.Context, metrics *DataQualityMetrics) {
	status := &DataSourceStatus{
		Name:       "announcement_data",
		Status:     "degraded", // 当前实现返回默认值
		LastUpdate: time.Now(),
		SampleSize: 0,
	}

	// 当前公告数据实现不完整，标记为降级状态
	metrics.Anomalies = append(metrics.Anomalies, DataAnomaly{
		Type:        "missing_implementation",
		Severity:    "high",
		Description: "公告数据功能未实现，返回默认值",
		Timestamp:   time.Now(),
	})

	metrics.DataSources["announcement_data"] = status
}

// calculateOverallMetrics 计算总体指标
func (dqm *DataQualityMonitor) calculateOverallMetrics(metrics *DataQualityMetrics) {
	totalSources := len(metrics.DataSources)
	healthySources := 0
	totalQuality := 0.0

	for _, source := range metrics.DataSources {
		if source.Status == "healthy" {
			healthySources++
			totalQuality += 100.0
		} else if source.Status == "degraded" {
			totalQuality += 50.0
		}
		// failed状态为0
	}

	// 计算完整性（数据源可用性）
	metrics.Completeness = float64(healthySources) / float64(totalSources) * 100

	// 计算一致性（所有数据源质量平均值）
	if totalSources > 0 {
		metrics.Consistency = totalQuality / float64(totalSources)
	}

	// 设置默认新鲜度（1小时）
	metrics.Freshness = 3600

	// 设置默认准确性
	metrics.Accuracy = metrics.Consistency
}

// checkAlertThresholds 检查告警阈值
func (dqm *DataQualityMonitor) checkAlertThresholds(metrics *DataQualityMetrics) {
	// 检查数据新鲜度
	if metrics.Freshness > dqm.alertThresholds.MaxFreshnessSeconds {
		dqm.triggerAlert(DataAnomaly{
			Type:        "data_freshness",
			Severity:    "medium",
			Description: fmt.Sprintf("数据新鲜度超标: %ds > %ds", metrics.Freshness, dqm.alertThresholds.MaxFreshnessSeconds),
			Timestamp:   time.Now(),
		})
	}

	// 检查数据完整性
	if metrics.Completeness < dqm.alertThresholds.MinCompletenessPercent {
		severity := "medium"
		if metrics.Completeness < 50 {
			severity = "high"
		}

		dqm.triggerAlert(DataAnomaly{
			Type:        "data_completeness",
			Severity:    severity,
			Description: fmt.Sprintf("数据完整性不足: %.1f%% < %.1f%%", metrics.Completeness, dqm.alertThresholds.MinCompletenessPercent),
			Timestamp:   time.Now(),
		})
	}
}

// triggerAlert 触发告警
func (dqm *DataQualityMonitor) triggerAlert(anomaly DataAnomaly) {
	log.Printf("[DataQualityMonitor] 触发告警 - 类型: %s, 严重程度: %s, 描述: %s",
		anomaly.Type, anomaly.Severity, anomaly.Description)

	// 添加到异常列表
	dqm.mu.Lock()
	dqm.metrics.Anomalies = append(dqm.metrics.Anomalies, anomaly)
	dqm.mu.Unlock()

	// 调用告警回调
	for _, callback := range dqm.alertCallbacks {
		go callback(anomaly)
	}
}

// GetHealthReport 获取健康报告
func (dqm *DataQualityMonitor) GetHealthReport() map[string]interface{} {
	metrics := dqm.GetMetrics()

	report := map[string]interface{}{
		"overall_health":  dqm.calculateOverallHealth(metrics),
		"metrics":         metrics,
		"recommendations": dqm.generateRecommendations(metrics),
		"timestamp":       time.Now(),
	}

	return report
}

// calculateOverallHealth 计算总体健康状态
func (dqm *DataQualityMonitor) calculateOverallHealth(metrics *DataQualityMetrics) string {
	if metrics.Completeness >= 80 && len(metrics.Anomalies) == 0 {
		return "healthy"
	} else if metrics.Completeness >= 60 {
		return "degraded"
	} else {
		return "unhealthy"
	}
}

// generateRecommendations 生成改进建议
func (dqm *DataQualityMonitor) generateRecommendations(metrics *DataQualityMetrics) []string {
	recommendations := make([]string, 0)

	if metrics.Completeness < 80 {
		recommendations = append(recommendations, "提升数据源可用性，实现多数据源备份")
	}

	if len(metrics.Anomalies) > 0 {
		recommendations = append(recommendations, "处理数据异常，完善错误处理机制")
	}

	for name, source := range metrics.DataSources {
		if source.Status == "failed" {
			recommendations = append(recommendations,
				fmt.Sprintf("修复%s数据源连接问题", name))
		} else if source.Status == "degraded" {
			recommendations = append(recommendations,
				fmt.Sprintf("优化%s数据质量", name))
		}
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "数据质量良好，继续监控")
	}

	return recommendations
}
