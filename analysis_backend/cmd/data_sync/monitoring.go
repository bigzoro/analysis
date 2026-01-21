package main

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/mem"
)

// MonitoringSystem 监控系统
type MonitoringSystem struct {
	service *DataSyncService

	// 监控配置
	checkInterval   time.Duration
	alertThresholds AlertThresholds
	alertCooldown   time.Duration // 告警冷却时间

	// 告警状态
	alerts struct {
		mu         sync.RWMutex
		active     []Alert
		history    []Alert
		lastAlerts map[string]time.Time // alert_type -> last_alert_time
	}

	// 健康状态
	healthStatus struct {
		mu              sync.RWMutex
		overallHealth   string // "healthy", "warning", "critical"
		componentHealth map[string]string
		lastHealthCheck time.Time
	}

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// AlertThresholds 告警阈值配置
type AlertThresholds struct {
	// WebSocket相关
	WebSocketReconnectThreshold int           // 重连次数阈值
	WebSocketDowntimeThreshold  time.Duration // 允许的最大宕机时间

	// API相关
	APIFailureRateThreshold float64       // API失败率阈值
	APILatencyThreshold     time.Duration // API延迟阈值

	// 数据一致性
	DataConsistencyScoreThreshold float64       // 数据一致性得分阈值
	DataAgeThreshold              time.Duration // 数据年龄阈值

	// 系统资源
	MemoryUsageThreshold    float64 // 内存使用率阈值
	CPUUsageThreshold       float64 // CPU使用率阈值
	GoroutineCountThreshold int     // Goroutine数量阈值
}

// Alert 告警信息
type Alert struct {
	ID         string
	Type       string // "websocket", "api", "consistency", "system"
	Severity   string // "info", "warning", "error", "critical"
	Title      string
	Message    string
	Timestamp  time.Time
	Resolved   bool
	ResolvedAt *time.Time
	Component  string
	Metric     string
	Value      interface{}
	Threshold  interface{}
}

// NewMonitoringSystem 创建监控系统
func NewMonitoringSystem(service *DataSyncService) *MonitoringSystem {
	ctx, cancel := context.WithCancel(context.Background())

	return &MonitoringSystem{
		service: service,

		checkInterval: time.Duration(service.config.Monitoring.CheckInterval) * time.Second,
		alertThresholds: AlertThresholds{
			WebSocketReconnectThreshold:   service.config.Monitoring.Thresholds.WebSocketReconnectThreshold,
			WebSocketDowntimeThreshold:    time.Duration(service.config.Monitoring.Thresholds.WebSocketDowntimeThreshold) * time.Second,
			APIFailureRateThreshold:       service.config.Monitoring.Thresholds.APIFailureRateThreshold,
			APILatencyThreshold:           time.Duration(service.config.Monitoring.Thresholds.APILatencyThreshold) * time.Second,
			DataConsistencyScoreThreshold: service.config.Monitoring.Thresholds.DataConsistencyThreshold,
			DataAgeThreshold:              time.Duration(service.config.Monitoring.Thresholds.DataAgeThreshold) * time.Second,
			MemoryUsageThreshold:          service.config.Monitoring.Thresholds.MemoryUsageThreshold,
			CPUUsageThreshold:             service.config.Monitoring.Thresholds.CPUUsageThreshold,
			GoroutineCountThreshold:       service.config.Monitoring.Thresholds.GoroutineCountThreshold,
		},
		alertCooldown: time.Duration(service.config.Monitoring.AlertCooldown) * time.Second,

		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动监控系统
func (m *MonitoringSystem) Start() {
	log.Printf("[Monitoring] Starting monitoring system...")

	// 初始化告警状态
	m.alerts.lastAlerts = make(map[string]time.Time)
	m.healthStatus.componentHealth = make(map[string]string)

	go m.monitoringLoop()
	go m.healthCheckLoop()

	log.Printf("[Monitoring] Monitoring system started")
}

// Stop 停止监控系统
func (m *MonitoringSystem) Stop() {
	m.cancel()
	log.Printf("[Monitoring] Stopped")
}

// monitoringLoop 监控循环
func (m *MonitoringSystem) monitoringLoop() {
	ticker := time.NewTicker(m.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performMonitoringChecks()
		}
	}
}

// healthCheckLoop 健康检查循环
func (m *MonitoringSystem) healthCheckLoop() {
	ticker := time.NewTicker(1 * time.Minute) // 每分钟检查一次整体健康状态
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.updateOverallHealthStatus()
		}
	}
}

// performMonitoringChecks 执行监控检查
func (m *MonitoringSystem) performMonitoringChecks() {
	// 检查WebSocket状态
	m.checkWebSocketStatus()

	// 检查API性能
	m.checkAPIStatus()

	// 检查数据一致性
	m.checkDataConsistency()

	// 检查系统资源
	m.checkSystemResources()
}

// checkWebSocketStatus 检查WebSocket状态
func (m *MonitoringSystem) checkWebSocketStatus() {
	if m.service.smartScheduler == nil {
		return
	}

	schedulerStats := m.service.smartScheduler.GetStats()
	isHealthy := schedulerStats["websocket_healthy"].(bool)
	reconnectCount := schedulerStats["rest_api_fallback_count"].(int64)

	// 检查重连次数
	if reconnectCount >= int64(m.alertThresholds.WebSocketReconnectThreshold) {
		m.raiseAlert(Alert{
			Type:     "websocket",
			Severity: "warning",
			Title:    "High WebSocket Reconnect Count",
			Message: fmt.Sprintf("WebSocket has reconnected %d times, exceeding threshold of %d",
				reconnectCount, m.alertThresholds.WebSocketReconnectThreshold),
			Component: "websocket",
			Metric:    "reconnect_count",
			Value:     reconnectCount,
			Threshold: m.alertThresholds.WebSocketReconnectThreshold,
		})
	}

	// 检查健康状态
	if !isHealthy {
		m.raiseAlert(Alert{
			Type:      "websocket",
			Severity:  "error",
			Title:     "WebSocket Connection Unhealthy",
			Message:   "WebSocket connection is unhealthy, system may be relying on REST API fallback",
			Component: "websocket",
			Metric:    "health_status",
			Value:     false,
		})
	}

	m.healthStatus.mu.Lock()
	m.healthStatus.componentHealth["websocket"] = map[bool]string{true: "healthy", false: "unhealthy"}[isHealthy]
	m.healthStatus.mu.Unlock()
}

// checkAPIStatus 检查API状态
func (m *MonitoringSystem) checkAPIStatus() {
	// 检查各个同步器的API性能
	syncers := []string{"price", "kline", "depth"}

	for _, syncerName := range syncers {
		if syncer, exists := m.service.syncers[syncerName]; exists {
			stats := syncer.GetStats()

			// 检查失败率
			failureRate := 0.0
			if successRateStr, ok := stats["api_success_rate"].(string); ok {
				// 解析成功率（格式如 "95.2%"）
				var successRate float64
				fmt.Sscanf(successRateStr, "%f%%", &successRate)
				failureRate = 100.0 - successRate

				if failureRate > m.alertThresholds.APIFailureRateThreshold {
					m.raiseAlert(Alert{
						Type:     "api",
						Severity: "warning",
						Title:    fmt.Sprintf("High API Failure Rate - %s", syncerName),
						Message: fmt.Sprintf("%s syncer has %.1f%% failure rate, exceeding threshold of %.1f%%",
							syncerName, failureRate, m.alertThresholds.APIFailureRateThreshold),
						Component: syncerName,
						Metric:    "failure_rate",
						Value:     failureRate,
						Threshold: m.alertThresholds.APIFailureRateThreshold,
					})
				}
			}

			// 检查延迟
			if avgLatency, ok := stats["api_avg_latency"].(time.Duration); ok {
				if avgLatency > m.alertThresholds.APILatencyThreshold {
					m.raiseAlert(Alert{
						Type:     "api",
						Severity: "warning",
						Title:    fmt.Sprintf("High API Latency - %s", syncerName),
						Message: fmt.Sprintf("%s syncer average latency is %v, exceeding threshold of %v",
							syncerName, avgLatency, m.alertThresholds.APILatencyThreshold),
						Component: syncerName,
						Metric:    "avg_latency",
						Value:     avgLatency,
						Threshold: m.alertThresholds.APILatencyThreshold,
					})
				}
			}

			// 更新组件健康状态
			isHealthy := true
			if failureRate > 50.0 { // 如果失败率超过50%，认为不健康
				isHealthy = false
			}

			m.healthStatus.mu.Lock()
			m.healthStatus.componentHealth[syncerName] = map[bool]string{true: "healthy", false: "unhealthy"}[isHealthy]
			m.healthStatus.mu.Unlock()
		}
	}
}

// checkDataConsistency 检查数据一致性
func (m *MonitoringSystem) checkDataConsistency() {
	if m.service.consistencyChecker == nil {
		return
	}

	stats := m.service.consistencyChecker.GetStats()
	consistencyScore := m.service.consistencyChecker.GetConsistencyScore()

	// 检查一致性得分
	if consistencyScore < m.alertThresholds.DataConsistencyScoreThreshold {
		m.raiseAlert(Alert{
			Type:     "consistency",
			Severity: "warning",
			Title:    "Low Data Consistency Score",
			Message: fmt.Sprintf("Data consistency score is %.1f%%, below threshold of %.1f%%",
				consistencyScore, m.alertThresholds.DataConsistencyScoreThreshold),
			Component: "consistency_checker",
			Metric:    "consistency_score",
			Value:     consistencyScore,
			Threshold: m.alertThresholds.DataConsistencyScoreThreshold,
		})
	}

	// 检查最近的不一致问题
	if recentIssues, ok := stats["recent_inconsistencies"].([]map[string]interface{}); ok {
		for _, issue := range recentIssues {
			if resolved, ok := issue["resolved"].(bool); !ok || !resolved {
				if severity, ok := issue["severity"].(string); ok {
					alertSeverity := map[string]string{
						"low":      "info",
						"medium":   "warning",
						"high":     "error",
						"critical": "critical",
					}[severity]

					m.raiseAlert(Alert{
						Type:      "consistency",
						Severity:  alertSeverity,
						Title:     "Data Consistency Issue",
						Message:   issue["description"].(string),
						Component: "data_consistency",
						Metric:    "consistency_issue",
					})
				}
			}
		}
	}

	m.healthStatus.mu.Lock()
	isHealthy := consistencyScore >= 90.0 // 90%以上认为健康
	m.healthStatus.componentHealth["consistency"] = map[bool]string{true: "healthy", false: "warning"}[isHealthy]
	m.healthStatus.mu.Unlock()
}

// checkSystemResources 检查系统资源
func (m *MonitoringSystem) checkSystemResources() {
	systemHealth := "healthy"
	warnings := []string{}

	// 检查内存使用率
	if memoryStats, err := mem.VirtualMemory(); err == nil {
		memoryUsage := memoryStats.UsedPercent
		if memoryUsage > m.alertThresholds.MemoryUsageThreshold {
			warnings = append(warnings, fmt.Sprintf("High memory usage: %.1f%%", memoryUsage))
			systemHealth = "warning"

			m.raiseAlert(Alert{
				Type:     "system",
				Severity: "warning",
				Title:    "High Memory Usage",
				Message: fmt.Sprintf("Memory usage is %.1f%%, exceeding threshold of %.1f%%",
					memoryUsage, m.alertThresholds.MemoryUsageThreshold),
				Component: "system",
				Metric:    "memory_usage",
				Value:     memoryUsage,
				Threshold: m.alertThresholds.MemoryUsageThreshold,
			})
		}
	} else {
		log.Printf("[Monitoring] Failed to get memory stats: %v", err)
	}

	// 检查CPU使用率
	if cpuStats, err := cpu.Percent(time.Second, false); err == nil && len(cpuStats) > 0 {
		cpuUsage := cpuStats[0]
		// CPU使用率阈值可以根据需要调整，这里设置为80%
		if cpuUsage > 80.0 {
			warnings = append(warnings, fmt.Sprintf("High CPU usage: %.1f%%", cpuUsage))
			if systemHealth == "healthy" {
				systemHealth = "warning"
			}

			m.raiseAlert(Alert{
				Type:      "system",
				Severity:  "warning",
				Title:     "High CPU Usage",
				Message:   fmt.Sprintf("CPU usage is %.1f%%, exceeding threshold of 80%%", cpuUsage),
				Component: "system",
				Metric:    "cpu_usage",
				Value:     cpuUsage,
				Threshold: 80.0,
			})
		}
	} else {
		log.Printf("[Monitoring] Failed to get CPU stats: %v", err)
	}

	// 检查Goroutine数量
	goroutineCount := runtime.NumGoroutine()
	if goroutineCount > m.alertThresholds.GoroutineCountThreshold {
		warnings = append(warnings, fmt.Sprintf("High goroutine count: %d", goroutineCount))
		systemHealth = "warning"

		m.raiseAlert(Alert{
			Type:     "system",
			Severity: "warning",
			Title:    "High Goroutine Count",
			Message: fmt.Sprintf("Goroutine count is %d, exceeding threshold of %d",
				goroutineCount, m.alertThresholds.GoroutineCountThreshold),
			Component: "system",
			Metric:    "goroutine_count",
			Value:     goroutineCount,
			Threshold: m.alertThresholds.GoroutineCountThreshold,
		})
	}

	// 检查垃圾回收
	var gcStats runtime.MemStats
	runtime.ReadMemStats(&gcStats)

	// 检查GC暂停时间（如果平均GC暂停时间超过100ms，认为有问题）
	gcPauseTime := time.Duration(gcStats.PauseTotalNs / uint64(gcStats.NumGC))
	if gcStats.NumGC > 0 && gcPauseTime > 100*time.Millisecond {
		warnings = append(warnings, fmt.Sprintf("High GC pause time: %v", gcPauseTime))
		if systemHealth == "healthy" {
			systemHealth = "warning"
		}

		m.raiseAlert(Alert{
			Type:      "system",
			Severity:  "info",
			Title:     "High GC Pause Time",
			Message:   fmt.Sprintf("Average GC pause time is %v, which may affect performance", gcPauseTime),
			Component: "system",
			Metric:    "gc_pause_time",
			Value:     gcPauseTime,
		})
	}

	// 记录系统资源统计信息
	if len(warnings) > 0 {
		log.Printf("[Monitoring] System resource warnings: %v", warnings)
	} else {
		log.Printf("[Monitoring] System resources normal - Memory: checking, CPU: checking, Goroutines: %d", goroutineCount)
	}

	m.healthStatus.mu.Lock()
	m.healthStatus.componentHealth["system"] = systemHealth
	m.healthStatus.mu.Unlock()
}

// updateOverallHealthStatus 更新整体健康状态
func (m *MonitoringSystem) updateOverallHealthStatus() {
	m.healthStatus.mu.Lock()
	defer m.healthStatus.mu.Unlock()

	m.healthStatus.lastHealthCheck = time.Now()

	// 计算整体健康状态
	healthCounts := map[string]int{"healthy": 0, "warning": 0, "unhealthy": 0}

	for _, status := range m.healthStatus.componentHealth {
		switch status {
		case "healthy":
			healthCounts["healthy"]++
		case "warning":
			healthCounts["warning"]++
		case "unhealthy":
			healthCounts["unhealthy"]++
		}
	}

	// 确定整体健康状态
	if healthCounts["unhealthy"] > 0 {
		m.healthStatus.overallHealth = "critical"
	} else if healthCounts["warning"] > 0 {
		m.healthStatus.overallHealth = "warning"
	} else {
		m.healthStatus.overallHealth = "healthy"
	}

	log.Printf("[Monitoring] Health check: %s (%d healthy, %d warning, %d unhealthy)",
		m.healthStatus.overallHealth,
		healthCounts["healthy"], healthCounts["warning"], healthCounts["unhealthy"])
}

// raiseAlert 触发告警
func (m *MonitoringSystem) raiseAlert(alert Alert) {
	alert.ID = fmt.Sprintf("%s_%s_%d", alert.Type, alert.Component, time.Now().Unix())
	alert.Timestamp = time.Now()

	// 检查告警冷却时间
	alertKey := fmt.Sprintf("%s_%s_%s", alert.Type, alert.Component, alert.Metric)
	if lastAlert, exists := m.alerts.lastAlerts[alertKey]; exists {
		if time.Since(lastAlert) < m.alertCooldown {
			// 在冷却期内，跳过告警
			return
		}
	}

	// 记录告警
	m.alerts.mu.Lock()
	m.alerts.active = append(m.alerts.active, alert)
	m.alerts.history = append(m.alerts.history, alert)
	m.alerts.lastAlerts[alertKey] = alert.Timestamp

	// 限制历史记录数量
	if len(m.alerts.history) > 100 {
		m.alerts.history = m.alerts.history[len(m.alerts.history)-100:]
	}
	m.alerts.mu.Unlock()

	// 记录告警日志
	log.Printf("[Monitoring] 🚨 ALERT [%s] %s: %s", alert.Severity, alert.Title, alert.Message)
}

// resolveAlert 解决告警
func (m *MonitoringSystem) resolveAlert(alertID string) {
	m.alerts.mu.Lock()
	defer m.alerts.mu.Unlock()

	for i, alert := range m.alerts.active {
		if alert.ID == alertID {
			now := time.Now()
			alert.Resolved = true
			alert.ResolvedAt = &now
			m.alerts.active = append(m.alerts.active[:i], m.alerts.active[i+1:]...)
			break
		}
	}
}

// GetAlerts 获取告警信息
func (m *MonitoringSystem) GetAlerts() map[string]interface{} {
	m.alerts.mu.RLock()
	defer m.alerts.mu.RUnlock()

	activeAlerts := make([]map[string]interface{}, 0, len(m.alerts.active))
	for _, alert := range m.alerts.active {
		activeAlerts = append(activeAlerts, map[string]interface{}{
			"id":        alert.ID,
			"type":      alert.Type,
			"severity":  alert.Severity,
			"title":     alert.Title,
			"message":   alert.Message,
			"timestamp": alert.Timestamp,
			"component": alert.Component,
			"metric":    alert.Metric,
			"value":     alert.Value,
			"threshold": alert.Threshold,
		})
	}

	return map[string]interface{}{
		"active_count":  len(m.alerts.active),
		"active_alerts": activeAlerts,
		"total_history": len(m.alerts.history),
	}
}

// GetHealthStatus 获取健康状态
func (m *MonitoringSystem) GetHealthStatus() map[string]interface{} {
	m.healthStatus.mu.RLock()
	defer m.healthStatus.mu.RUnlock()

	return map[string]interface{}{
		"overall_health":   m.healthStatus.overallHealth,
		"component_health": m.healthStatus.componentHealth,
		"last_check":       m.healthStatus.lastHealthCheck,
	}
}
