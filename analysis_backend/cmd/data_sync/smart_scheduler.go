package main

import (
	"context"
	"log"
	"sync"
	"time"
)

// SmartScheduler 智能调度器 - 协调WebSocket和REST API
type SmartScheduler struct {
	// 同步器引用
	websocketSyncer *WebSocketSyncer
	klineSyncer     *KlineSyncer
	depthSyncer     *DepthSyncer
	priceSyncer     *PriceSyncer

	// 调度状态
	websocketHealthy bool
	lastWebSocketCheck time.Time
	restAPIMode       bool // 是否处于REST API模式

	// 调度配置
	checkInterval          time.Duration
	websocketGracePeriod   time.Duration // WebSocket断开后的宽限期
	restAPIBackoffFactor   float64       // REST API频率降低倍数

	// 统计信息
	stats struct {
		mu                    sync.RWMutex
		websocketUptime       time.Duration
		restAPIFallbackCount  int64
		lastModeSwitch        time.Time
		totalWebSocketDowntime time.Duration
	}

	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
}

// NewSmartScheduler 创建智能调度器
func NewSmartScheduler(
	websocketSyncer *WebSocketSyncer,
	klineSyncer *KlineSyncer,
	depthSyncer *DepthSyncer,
	priceSyncer *PriceSyncer,
) *SmartScheduler {

	ctx, cancel := context.WithCancel(context.Background())

	return &SmartScheduler{
		websocketSyncer: websocketSyncer,
		klineSyncer:     klineSyncer,
		depthSyncer:     depthSyncer,
		priceSyncer:     priceSyncer,

		websocketHealthy:      false,
		lastWebSocketCheck:     time.Now(),
		restAPIMode:           false,

		// 使用默认配置，后续可以从配置文件读取
		checkInterval:          30 * time.Second,
		websocketGracePeriod:   2 * time.Minute,
		restAPIBackoffFactor:   2.0,

		ctx:    ctx,
		cancel: cancel,
	}
}

// NewSmartSchedulerWithConfig 使用配置创建智能调度器
func NewSmartSchedulerWithConfig(
	websocketSyncer *WebSocketSyncer,
	klineSyncer *KlineSyncer,
	depthSyncer *DepthSyncer,
	priceSyncer *PriceSyncer,
	config *DataSyncConfig,
) *SmartScheduler {

	ctx, cancel := context.WithCancel(context.Background())

	return &SmartScheduler{
		websocketSyncer: websocketSyncer,
		klineSyncer:     klineSyncer,
		depthSyncer:     depthSyncer,
		priceSyncer:     priceSyncer,

		websocketHealthy:      false,
		lastWebSocketCheck:     time.Now(),
		restAPIMode:           false,

		checkInterval:          time.Duration(config.SmartScheduler.CheckInterval) * time.Second,
		websocketGracePeriod:   time.Duration(config.SmartScheduler.WebSocketGracePeriod) * time.Second,
		restAPIBackoffFactor:   config.SmartScheduler.RestAPIBackoffFactor,

		ctx:    ctx,
		cancel: cancel,
	}
}

// Start 启动智能调度器
func (s *SmartScheduler) Start() {
	log.Printf("[SmartScheduler] Starting intelligent scheduler...")

	go s.monitoringLoop()
	go s.healthCheckLoop()

	log.Printf("[SmartScheduler] Intelligent scheduler started")
}

// Stop 停止智能调度器
func (s *SmartScheduler) Stop() {
	s.cancel()
	log.Printf("[SmartScheduler] Stopped")
}

// monitoringLoop 监控循环 - 定期检查状态并调整调度
func (s *SmartScheduler) monitoringLoop() {
	ticker := time.NewTicker(s.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.performSchedulingDecision()
		}
	}
}

// healthCheckLoop 健康检查循环 - 检查WebSocket状态
func (s *SmartScheduler) healthCheckLoop() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒检查一次
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkWebSocketHealth()
		}
	}
}

// checkWebSocketHealth 检查WebSocket健康状态
func (s *SmartScheduler) checkWebSocketHealth() {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 检查WebSocket同步器是否正在运行且健康
	isHealthy := s.websocketSyncer != nil && s.websocketSyncer.IsRunning() && s.websocketSyncer.IsHealthy()

	wasHealthy := s.websocketHealthy
	s.websocketHealthy = isHealthy
	s.lastWebSocketCheck = time.Now()

	// 状态变化处理
	if wasHealthy && !isHealthy {
		log.Printf("[SmartScheduler] ⚠️ WebSocket connection became unhealthy")
		s.stats.restAPIFallbackCount++
		s.stats.lastModeSwitch = time.Now()

		// 启动REST API模式
		s.switchToRestAPIMode()

	} else if !wasHealthy && isHealthy {
		log.Printf("[SmartScheduler] ✅ WebSocket connection restored")
		s.stats.lastModeSwitch = time.Now()

		// 延迟切换回WebSocket模式，给连接稳定时间
		time.AfterFunc(30*time.Second, func() {
			s.switchToWebSocketMode()
		})
	}

	// 更新运行时间统计
	if isHealthy {
		s.stats.websocketUptime += 10 * time.Second
	} else {
		s.stats.totalWebSocketDowntime += 10 * time.Second
	}
}

// performSchedulingDecision 执行调度决策
func (s *SmartScheduler) performSchedulingDecision() {
	s.mu.RLock()
	websocketHealthy := s.websocketHealthy
	restAPIMode := s.restAPIMode
	s.mu.RUnlock()

	// 根据当前状态调整REST API同步器的行为
	if websocketHealthy && restAPIMode {
		// WebSocket健康，但仍在REST模式 - 可能是宽限期内，等待切换
		log.Printf("[SmartScheduler] WebSocket healthy, waiting for grace period before switching back")

	} else if !websocketHealthy && !restAPIMode {
		// WebSocket不健康，但还未切换到REST模式 - 强制切换
		log.Printf("[SmartScheduler] Forcing switch to REST API mode due to unhealthy WebSocket")
		s.switchToRestAPIMode()
	}

	// 调整REST API的调用频率
	s.adjustRestAPIFrequency(websocketHealthy)
}

// switchToRestAPIMode 切换到REST API模式
func (s *SmartScheduler) switchToRestAPIMode() {
	s.mu.Lock()
	s.restAPIMode = true
	s.mu.Unlock()

	log.Printf("[SmartScheduler] 🔄 Switching to REST API mode")

	// 可以在这里增加REST API同步器的频率或启用额外的同步器
	// 目前通过调整频率来实现
}

// switchToWebSocketMode 切换到WebSocket模式
func (s *SmartScheduler) switchToWebSocketMode() {
	s.mu.Lock()
	if s.websocketHealthy {
		s.restAPIMode = false
		log.Printf("[SmartScheduler] 🔄 Switching back to WebSocket mode")
	}
	s.mu.Unlock()
}

// adjustRestAPIFrequency 根据WebSocket状态调整REST API频率
func (s *SmartScheduler) adjustRestAPIFrequency(websocketHealthy bool) {
	// 这里可以动态调整REST API同步器的调用间隔
	// 目前通过配置实现，后续可以实现运行时动态调整

	if websocketHealthy {
		// WebSocket正常时，REST API保持较低频率
		log.Printf("[SmartScheduler] WebSocket healthy - REST APIs running at reduced frequency")
	} else {
		// WebSocket异常时，REST API可以适当提高频率保证数据连续性
		log.Printf("[SmartScheduler] WebSocket unhealthy - REST APIs running at normal frequency for data continuity")
	}
}

// GetStats 获取统计信息
func (s *SmartScheduler) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	s.stats.mu.RLock()
	defer s.stats.mu.RUnlock()

	return map[string]interface{}{
		"websocket_healthy":         s.websocketHealthy,
		"rest_api_mode":             s.restAPIMode,
		"last_websocket_check":      s.lastWebSocketCheck,
		"websocket_uptime":          s.stats.websocketUptime,
		"rest_api_fallback_count":   s.stats.restAPIFallbackCount,
		"last_mode_switch":          s.stats.lastModeSwitch,
		"total_websocket_downtime":  s.stats.totalWebSocketDowntime,
		"check_interval":            s.checkInterval,
		"websocket_grace_period":    s.websocketGracePeriod,
	}
}

// IsWebSocketPreferred 是否应该优先使用WebSocket
func (s *SmartScheduler) IsWebSocketPreferred() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.websocketHealthy
}

// ShouldUseRestAPI 是否应该使用REST API
func (s *SmartScheduler) ShouldUseRestAPI() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return !s.websocketHealthy || s.restAPIMode
}
