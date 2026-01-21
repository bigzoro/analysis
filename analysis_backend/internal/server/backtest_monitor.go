package server

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"
)

// Monitor 回测监控器
type Monitor struct {
	startTime       time.Time
	activeBacktests sync.Map // map[string]*BacktestSession
	mutex           sync.RWMutex
	metrics         *MonitorMetrics
	alerts          []Alert
	alertChan       chan Alert
}

// BacktestSession 回测会话
type BacktestSession struct {
	ID         string          `json:"id"`
	Config     BacktestConfig  `json:"config"`
	StartTime  time.Time       `json:"start_time"`
	Status     string          `json:"status"` // "running", "completed", "failed", "cancelled"
	Progress   float64         `json:"progress"`
	Error      string          `json:"error,omitempty"`
	Result     *BacktestResult `json:"result,omitempty"`
	cancelFunc context.CancelFunc
}

// MonitorMetrics 监控指标
type MonitorMetrics struct {
	TotalBacktests     int64         `json:"total_backtests"`
	ActiveBacktests    int64         `json:"active_backtests"`
	CompletedBacktests int64         `json:"completed_backtests"`
	FailedBacktests    int64         `json:"failed_backtests"`
	AvgExecutionTime   time.Duration `json:"avg_execution_time"`
	TotalCPUTime       time.Duration `json:"total_cpu_time"`
	PeakMemoryUsage    int64         `json:"peak_memory_usage"`
	mutex              sync.RWMutex
}

// Alert 告警信息
type Alert struct {
	ID        string                 `json:"id"`
	Level     string                 `json:"level"` // "info", "warning", "error", "critical"
	Message   string                 `json:"message"`
	Source    string                 `json:"source"`
	Timestamp time.Time              `json:"timestamp"`
	SessionID string                 `json:"session_id,omitempty"`
	Details   map[string]interface{} `json:"details,omitempty"`
}

// NewMonitor 创建监控器
func NewMonitor() *Monitor {
	return &Monitor{
		startTime: time.Now(),
		metrics:   &MonitorMetrics{},
		alertChan: make(chan Alert, 100),
	}
}

// StartBacktest 开始回测会话
func (m *Monitor) StartBacktest(sessionID string, config BacktestConfig) *BacktestSession {
	session := &BacktestSession{
		ID:        sessionID,
		Config:    config,
		StartTime: time.Now(),
		Status:    "running",
		Progress:  0.0,
	}

	m.activeBacktests.Store(sessionID, session)

	m.mutex.Lock()
	m.metrics.ActiveBacktests++
	m.metrics.TotalBacktests++
	m.mutex.Unlock()

	log.Printf("[MONITOR] Started backtest session: %s for %s", sessionID, config.Symbol)

	return session
}

// UpdateProgress 更新进度
func (m *Monitor) UpdateProgress(sessionID string, progress float64) {
	if session, exists := m.getSession(sessionID); exists {
		session.Progress = progress
	}
}

// CompleteBacktest 完成回测
func (m *Monitor) CompleteBacktest(sessionID string, result *BacktestResult) {
	if session, exists := m.getSession(sessionID); exists {
		session.Status = "completed"
		session.Progress = 100.0
		session.Result = result

		m.activeBacktests.Delete(sessionID)

		m.mutex.Lock()
		m.metrics.ActiveBacktests--
		m.metrics.CompletedBacktests++
		executionTime := time.Since(session.StartTime)
		m.metrics.AvgExecutionTime = (m.metrics.AvgExecutionTime + executionTime) / 2
		m.mutex.Unlock()

		log.Printf("[MONITOR] Completed backtest session: %s in %v", sessionID, executionTime)
	}
}

// FailBacktest 回测失败
func (m *Monitor) FailBacktest(sessionID string, err error) {
	if session, exists := m.getSession(sessionID); exists {
		session.Status = "failed"
		session.Error = err.Error()

		m.activeBacktests.Delete(sessionID)

		m.mutex.Lock()
		m.metrics.ActiveBacktests--
		m.metrics.FailedBacktests++
		m.mutex.Unlock()

		log.Printf("[MONITOR] Failed backtest session: %s, error: %v", sessionID, err)

		// 发送告警
		m.sendAlert("error", fmt.Sprintf("Backtest session %s failed", sessionID), "monitor", sessionID, map[string]interface{}{
			"error": err.Error(),
		})
	}
}

// CancelBacktest 取消回测
func (m *Monitor) CancelBacktest(sessionID string) {
	if session, exists := m.getSession(sessionID); exists {
		session.Status = "cancelled"

		if session.cancelFunc != nil {
			session.cancelFunc()
		}

		m.activeBacktests.Delete(sessionID)

		m.mutex.Lock()
		m.metrics.ActiveBacktests--
		m.mutex.Unlock()

		log.Printf("[MONITOR] Cancelled backtest session: %s", sessionID)
	}
}

// GetSession 获取会话
func (m *Monitor) GetSession(sessionID string) (*BacktestSession, bool) {
	return m.getSession(sessionID)
}

func (m *Monitor) getSession(sessionID string) (*BacktestSession, bool) {
	if session, exists := m.activeBacktests.Load(sessionID); exists {
		if s, ok := session.(*BacktestSession); ok {
			return s, true
		}
	}
	return nil, false
}

// GetAllSessions 获取所有会话
func (m *Monitor) GetAllSessions() []*BacktestSession {
	sessions := make([]*BacktestSession, 0)

	m.activeBacktests.Range(func(key, value interface{}) bool {
		if session, ok := value.(*BacktestSession); ok {
			sessions = append(sessions, session)
		}
		return true
	})

	return sessions
}

// GetMetrics 获取监控指标
func (m *Monitor) GetMetrics() *MonitorMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// 复制指标以避免并发问题
	metrics := *m.metrics
	return &metrics
}

// sendAlert 发送告警
func (m *Monitor) sendAlert(level, message, source, sessionID string, details map[string]interface{}) {
	alert := Alert{
		ID:        generateID(),
		Level:     level,
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
		SessionID: sessionID,
		Details:   details,
	}

	m.alerts = append(m.alerts, alert)

	// 非阻塞发送到告警通道
	select {
	case m.alertChan <- alert:
	default:
		log.Printf("[MONITOR] Alert channel full, dropping alert: %s", message)
	}

	log.Printf("[ALERT] [%s] %s", level, message)
}

// GetAlerts 获取告警
func (m *Monitor) GetAlerts(limit int) []Alert {
	if limit <= 0 || limit > len(m.alerts) {
		limit = len(m.alerts)
	}

	// 返回最新的告警
	start := len(m.alerts) - limit
	if start < 0 {
		start = 0
	}

	return m.alerts[start:]
}

// ClearAlerts 清除告警
func (m *Monitor) ClearAlerts() {
	m.alerts = make([]Alert, 0)
}

// HealthCheck 健康检查
func (m *Monitor) HealthCheck() *HealthStatus {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := &HealthStatus{
		Status:          "healthy",
		Uptime:          time.Since(m.startTime),
		ActiveSessions:  m.getActiveSessionCount(),
		TotalSessions:   m.metrics.TotalBacktests,
		MemoryUsage:     memStats.Alloc,
		Goroutines:      runtime.NumGoroutine(),
		LastHealthCheck: time.Now(),
	}

	// 检查是否有太多活动会话
	if status.ActiveSessions > 10 {
		status.Status = "warning"
		status.Message = "High number of active backtest sessions"
	}

	// 检查内存使用
	if memStats.Alloc > 1024*1024*1024 { // 1GB
		status.Status = "warning"
		status.Message = "High memory usage detected"
	}

	return status
}

func (m *Monitor) getActiveSessionCount() int64 {
	count := int64(0)
	m.activeBacktests.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// HealthStatus 健康状态
type HealthStatus struct {
	Status          string        `json:"status"`
	Message         string        `json:"message,omitempty"`
	Uptime          time.Duration `json:"uptime"`
	ActiveSessions  int64         `json:"active_sessions"`
	TotalSessions   int64         `json:"total_sessions"`
	MemoryUsage     uint64        `json:"memory_usage"`
	Goroutines      int           `json:"goroutines"`
	LastHealthCheck time.Time     `json:"last_health_check"`
}

// PerformanceMonitor 性能监控器
type PerformanceMonitor struct {
	metrics map[string]*PerformanceMetric
	mutex   sync.RWMutex
}

// PerformanceMetric 性能指标
type PerformanceMetric struct {
	Name       string
	Count      int64
	TotalTime  time.Duration
	AvgTime    time.Duration
	MinTime    time.Duration
	MaxTime    time.Duration
	LastUpdate time.Time
}

// NewPerformanceMonitor 创建性能监控器
func NewPerformanceMonitor() *PerformanceMonitor {
	return &PerformanceMonitor{
		metrics: make(map[string]*PerformanceMetric),
	}
}

// StartTimer 开始计时
func (pm *PerformanceMonitor) StartTimer(name string) func() {
	start := time.Now()
	return func() {
		duration := time.Since(start)
		pm.recordMetric(name, duration)
	}
}

// RecordMetric 记录指标
func (pm *PerformanceMonitor) RecordMetric(name string, duration time.Duration) {
	pm.recordMetric(name, duration)
}

func (pm *PerformanceMonitor) recordMetric(name string, duration time.Duration) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	if pm.metrics[name] == nil {
		pm.metrics[name] = &PerformanceMetric{
			Name:    name,
			MinTime: time.Hour, // 初始化为较大值
		}
	}

	metric := pm.metrics[name]
	metric.Count++
	metric.TotalTime += duration
	metric.AvgTime = metric.TotalTime / time.Duration(metric.Count)
	metric.LastUpdate = time.Now()

	if duration < metric.MinTime {
		metric.MinTime = duration
	}
	if duration > metric.MaxTime {
		metric.MaxTime = duration
	}
}

// GetMetrics 获取所有性能指标
func (pm *PerformanceMonitor) GetMetrics() map[string]*PerformanceMetric {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	// 复制指标
	metrics := make(map[string]*PerformanceMetric)
	for k, v := range pm.metrics {
		metric := *v // 复制值
		metrics[k] = &metric
	}

	return metrics
}

// Reset 重置所有指标
func (pm *PerformanceMonitor) Reset() {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.metrics = make(map[string]*PerformanceMetric)
}

// generateID 生成唯一ID
func generateID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}
