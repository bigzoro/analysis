package server

import (
	"fmt"
	"log"
	"math"
	"time"

	"github.com/gin-gonic/gin"
)

// PriceAlert 价格告警
type PriceAlert struct {
	ID          string     `json:"id"`
	Symbol      string     `json:"symbol"`
	AlertType   string     `json:"alert_type"` // "entry", "exit", "stop_loss", "profit_target"
	PriceLevel  float64    `json:"price_level"`
	Condition   string     `json:"condition"` // "above", "below", "cross"
	Message     string     `json:"message"`
	Priority    string     `json:"priority"` // "high", "medium", "low"
	CreatedAt   time.Time  `json:"created_at"`
	IsActive    bool       `json:"is_active"`
	TriggeredAt *time.Time `json:"triggered_at,omitempty"`
}

// PriceAlertSystem 价格告警系统
type PriceAlertSystem struct {
	Alerts       map[string][]PriceAlert `json:"alerts"`        // symbol -> alerts
	ActiveAlerts map[string]bool         `json:"active_alerts"` // alert_id -> is_active
}

// TriggeredAlert 已触发的告警
type TriggeredAlert struct {
	Alert        PriceAlert `json:"alert"`
	TriggerPrice float64    `json:"trigger_price"`
	TriggeredAt  time.Time  `json:"triggered_at"`
	Deviation    float64    `json:"deviation"` // 偏离百分比
}

// PriceMonitor 价格监控服务
type PriceMonitor struct {
	server        *Server
	alerts        map[string][]PriceAlert // symbol -> alerts
	checkInterval time.Duration
	isRunning     bool
	stopChan      chan bool
}

// NewPriceMonitor 创建价格监控服务
func NewPriceMonitor(server *Server) *PriceMonitor {
	return &PriceMonitor{
		server:        server,
		alerts:        make(map[string][]PriceAlert),
		checkInterval: 30 * time.Second, // 每30秒检查一次
		stopChan:      make(chan bool),
	}
}

// Start 启动价格监控
func (pm *PriceMonitor) Start() {
	if pm.isRunning {
		return
	}

	pm.isRunning = true
	log.Printf("[PriceMonitor] 价格监控服务已启动，检查间隔: %v", pm.checkInterval)

	go pm.monitorLoop()
}

// Stop 停止价格监控
func (pm *PriceMonitor) Stop() {
	if !pm.isRunning {
		return
	}

	pm.isRunning = false
	pm.stopChan <- true
	log.Printf("[PriceMonitor] 价格监控服务已停止")
}

// AddAlert 添加价格告警
func (pm *PriceMonitor) AddAlert(alert PriceAlert) {
	pm.alerts[alert.Symbol] = append(pm.alerts[alert.Symbol], alert)
	log.Printf("[PriceMonitor] 添加告警: %s %s %.4f (%s)",
		alert.Symbol, alert.AlertType, alert.PriceLevel, alert.Condition)
}

// RemoveAlert 移除价格告警
func (pm *PriceMonitor) RemoveAlert(symbol, alertID string) {
	alerts := pm.alerts[symbol]
	for i, alert := range alerts {
		if alert.ID == alertID {
			pm.alerts[symbol] = append(alerts[:i], alerts[i+1:]...)
			log.Printf("[PriceMonitor] 移除告警: %s", alertID)
			break
		}
	}
}

// monitorLoop 监控循环
func (pm *PriceMonitor) monitorLoop() {
	ticker := time.NewTicker(pm.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			pm.checkAllAlerts()
		case <-pm.stopChan:
			return
		}
	}
}

// checkAllAlerts 检查所有告警
func (pm *PriceMonitor) checkAllAlerts() {
	if len(pm.alerts) == 0 {
		return
	}

	// 获取当前价格
	currentPrices := pm.getCurrentPrices()

	// 检查告警
	triggeredAlerts := pm.server.checkPriceAlerts(currentPrices)

	// 发送告警通知
	if len(triggeredAlerts) > 0 {
		pm.server.sendPriceAlerts(triggeredAlerts)
	}
}

// getCurrentPrices 获取当前价格
func (pm *PriceMonitor) getCurrentPrices() map[string]float64 {
	prices := make(map[string]float64)

	// 从所有有告警的symbol中获取价格
	for symbol := range pm.alerts {
		// 这里应该从价格服务获取最新价格
		price := pm.getMockPrice(symbol)
		if price > 0 {
			prices[symbol] = price
		}
	}

	return prices
}

// getMockPrice 获取模拟价格（实际应该从价格服务获取）
func (pm *PriceMonitor) getMockPrice(symbol string) float64 {
	// 暂时返回模拟价格
	switch symbol {
	case "BTC":
		return 95000 + float64(time.Now().Unix()%1000-500) // 模拟价格波动
	case "ETH":
		return 3800 + float64(time.Now().Unix()%200-100)
	case "ADA":
		return 0.85 + float64(time.Now().Unix()%10-5)*0.01
	default:
		return 1.0
	}
}

// GetStats 获取监控统计信息
func (pm *PriceMonitor) GetStats() gin.H {
	totalAlerts := 0
	activeAlerts := 0

	for _, alerts := range pm.alerts {
		totalAlerts += len(alerts)
		for _, alert := range alerts {
			if alert.IsActive {
				activeAlerts++
			}
		}
	}

	return gin.H{
		"is_running":        pm.isRunning,
		"check_interval":    pm.checkInterval.String(),
		"total_alerts":      totalAlerts,
		"active_alerts":     activeAlerts,
		"monitored_symbols": len(pm.alerts),
	}
}

// generatePriceAlerts 生成价格告警
func (s *Server) generatePriceAlerts(executionPlan *ExecutionPlan) []PriceAlert {
	var alerts []PriceAlert
	now := time.Now()

	// 为建仓计划生成告警
	for _, entry := range executionPlan.EntryPlan {
		alert := PriceAlert{
			ID:         fmt.Sprintf("entry_%s_%d_%d", executionPlan.Symbol, entry.StageNumber, now.Unix()),
			Symbol:     executionPlan.Symbol,
			AlertType:  "entry",
			PriceLevel: entry.PriceRange.Avg,
			Condition:  "below", // 价格跌到区间内时提醒
			Message:    fmt.Sprintf("建仓机会：第%d批 (%.1f%%仓位) - 价格区间 %.4f-%.4f", entry.StageNumber, entry.Percentage*100, entry.PriceRange.Min, entry.PriceRange.Max),
			Priority:   entry.Priority,
			CreatedAt:  now,
			IsActive:   true,
		}
		alerts = append(alerts, alert)
	}

	// 为出场计划生成告警
	for _, exit := range executionPlan.ExitPlan {
		alert := PriceAlert{
			ID:         fmt.Sprintf("exit_%s_%d_%d", executionPlan.Symbol, exit.StageNumber, now.Unix()),
			Symbol:     executionPlan.Symbol,
			AlertType:  "exit",
			PriceLevel: exit.PriceRange.Avg,
			Condition:  "above", // 多头策略价格上涨到目标时提醒
			Message:    fmt.Sprintf("出场机会：第%d批 (%.1f%%仓位) - 利润目标 %.1f%%", exit.StageNumber, exit.Percentage*100, exit.ProfitTarget*100),
			Priority:   "high",
			CreatedAt:  now,
			IsActive:   true,
		}

		// 根据策略类型调整条件
		if executionPlan.StrategyType == "SHORT" {
			alert.Condition = "below" // 空头策略价格下跌到目标时提醒
		}

		alerts = append(alerts, alert)
	}

	// 生成风险告警
	riskAlerts := s.generatePriceRiskAlerts(executionPlan)
	alerts = append(alerts, riskAlerts...)

	return alerts
}

// generatePriceRiskAlerts 生成价格风险告警
func (s *Server) generatePriceRiskAlerts(executionPlan *ExecutionPlan) []PriceAlert {
	var alerts []PriceAlert
	now := time.Now()

	// 止损告警
	stopLossPrice := executionPlan.CurrentPrice * 0.95 // 默认止损5%
	if executionPlan.StrategyType == "SHORT" {
		stopLossPrice = executionPlan.CurrentPrice * 1.05
	}

	stopLossAlert := PriceAlert{
		ID:         fmt.Sprintf("stop_loss_%s_%d", executionPlan.Symbol, now.Unix()),
		Symbol:     executionPlan.Symbol,
		AlertType:  "stop_loss",
		PriceLevel: stopLossPrice,
		Condition:  "cross",
		Message:    fmt.Sprintf("⚠️ 止损提醒：价格已触及 %.4f，建议立即止损", stopLossPrice),
		Priority:   "high",
		CreatedAt:  now,
		IsActive:   true,
	}
	alerts = append(alerts, stopLossAlert)

	// 追踪止损告警（如果启用）
	if executionPlan.RiskControls.TrailingStop {
		trailingStopPrice := executionPlan.CurrentPrice * (1.0 - executionPlan.RiskControls.TrailingStopPercent)
		if executionPlan.StrategyType == "SHORT" {
			trailingStopPrice = executionPlan.CurrentPrice * (1.0 + executionPlan.RiskControls.TrailingStopPercent)
		}

		trailingAlert := PriceAlert{
			ID:         fmt.Sprintf("trailing_stop_%s_%d", executionPlan.Symbol, now.Unix()),
			Symbol:     executionPlan.Symbol,
			AlertType:  "stop_loss",
			PriceLevel: trailingStopPrice,
			Condition:  "cross",
			Message:    fmt.Sprintf("🔄 追踪止损：价格已触及 %.4f，建议调整止损位", trailingStopPrice),
			Priority:   "medium",
			CreatedAt:  now,
			IsActive:   true,
		}
		alerts = append(alerts, trailingAlert)
	}

	// 重大价格变动告警
	majorMoveUp := executionPlan.CurrentPrice * 1.1 // 上涨10%
	majorMoveAlert := PriceAlert{
		ID:         fmt.Sprintf("major_move_%s_%d", executionPlan.Symbol, now.Unix()),
		Symbol:     executionPlan.Symbol,
		AlertType:  "risk_warning",
		PriceLevel: majorMoveUp, // 主要监控上涨突破
		Condition:  "above",
		Message:    fmt.Sprintf("🚨 重大价格变动：%s价格突破10%%，请重新评估风险", executionPlan.Symbol),
		Priority:   "high",
		CreatedAt:  now,
		IsActive:   true,
	}
	alerts = append(alerts, majorMoveAlert)

	return alerts
}

// checkPriceAlerts 检查价格告警
func (s *Server) checkPriceAlerts(currentPrices map[string]float64) []TriggeredAlert {
	var triggeredAlerts []TriggeredAlert

	// 这里应该是从数据库或缓存中获取活跃的告警
	// 暂时模拟检查逻辑
	for symbol, currentPrice := range currentPrices {
		// 模拟一些告警检查
		alerts := s.getMockAlertsForSymbol(symbol)
		for _, alert := range alerts {
			if s.isAlertTriggered(alert, currentPrice) {
				triggered := TriggeredAlert{
					Alert:        alert,
					TriggerPrice: currentPrice,
					TriggeredAt:  time.Now(),
					Deviation:    (currentPrice - alert.PriceLevel) / alert.PriceLevel,
				}
				triggeredAlerts = append(triggeredAlerts, triggered)

				// 标记告警为已触发
				alert.IsActive = false
				alert.TriggeredAt = &triggered.TriggeredAt
			}
		}
	}

	return triggeredAlerts
}

// isAlertTriggered 检查告警是否触发
func (s *Server) isAlertTriggered(alert PriceAlert, currentPrice float64) bool {
	if !alert.IsActive {
		return false
	}

	switch alert.Condition {
	case "above":
		return currentPrice >= alert.PriceLevel
	case "below":
		return currentPrice <= alert.PriceLevel
	case "cross":
		// 这里需要历史价格来判断是否穿越，暂时简化为接近
		return math.Abs(currentPrice-alert.PriceLevel)/alert.PriceLevel < 0.005 // 0.5%内
	default:
		return false
	}
}

// getMockAlertsForSymbol 获取模拟告警（实际应该从数据库获取）
func (s *Server) getMockAlertsForSymbol(symbol string) []PriceAlert {
	// 这里应该从数据库查询该symbol的活跃告警
	// 暂时返回空切片，实际实现需要数据库查询
	return []PriceAlert{}
}

// sendPriceAlerts 发送价格告警通知
func (s *Server) sendPriceAlerts(triggeredAlerts []TriggeredAlert) {
	for _, triggered := range triggeredAlerts {
		log.Printf("[PRICE_ALERT] %s %s: %s (价格: %.4f, 目标: %.4f)",
			triggered.Alert.Symbol,
			triggered.Alert.AlertType,
			triggered.Alert.Message,
			triggered.TriggerPrice,
			triggered.Alert.PriceLevel)

		// 这里可以集成推送通知、邮件、短信等
		// s.sendPushNotification(triggered)
		// s.sendEmailAlert(triggered)
	}
}

// CreatePriceAlert 创建价格告警API
func (s *Server) CreatePriceAlert(c *gin.Context) {
	var req struct {
		Symbol     string  `json:"symbol" binding:"required"`
		AlertType  string  `json:"alert_type" binding:"required"`
		PriceLevel float64 `json:"price_level" binding:"required"`
		Condition  string  `json:"condition" binding:"required"`
		Message    string  `json:"message"`
		Priority   string  `json:"priority"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		sendRecommendationError(c, 400, "无效的请求参数", "INVALID_REQUEST", err.Error())
		return
	}

	alert := PriceAlert{
		ID:         fmt.Sprintf("user_%s_%d", req.Symbol, time.Now().Unix()),
		Symbol:     req.Symbol,
		AlertType:  req.AlertType,
		PriceLevel: req.PriceLevel,
		Condition:  req.Condition,
		Message:    req.Message,
		Priority:   req.Priority,
		CreatedAt:  time.Now(),
		IsActive:   true,
	}

	// 这里应该保存到数据库
	// s.savePriceAlertToDB(alert)

	c.JSON(200, gin.H{
		"success": true,
		"alert":   alert,
	})
}

// GetPriceAlerts 获取价格告警API
func (s *Server) GetPriceAlerts(c *gin.Context) {
	_ = c.Query("symbol")
	_ = c.Query("alert_type")

	// 这里应该从数据库查询告警
	// alerts := s.getPriceAlertsFromDB(symbol, alertType)

	// 暂时返回空结果
	c.JSON(200, gin.H{
		"success": true,
		"alerts":  []PriceAlert{},
	})
}

// DeletePriceAlert 删除价格告警API
func (s *Server) DeletePriceAlert(c *gin.Context) {
	_ = c.Param("id")

	// 这里应该从数据库删除告警
	// s.deletePriceAlertFromDB(alertID)

	c.JSON(200, gin.H{
		"success": true,
		"message": "告警已删除",
	})
}

// GetPriceMonitorStats 获取价格监控统计
func (s *Server) GetPriceMonitorStats(c *gin.Context) {
	if s.priceMonitor == nil {
		c.JSON(500, gin.H{"error": "价格监控服务未初始化"})
		return
	}

	stats := s.priceMonitor.GetStats()
	c.JSON(200, gin.H{
		"success": true,
		"stats":   stats,
	})
}

// StartPriceMonitor 启动价格监控
func (s *Server) StartPriceMonitor(c *gin.Context) {
	if s.priceMonitor == nil {
		c.JSON(500, gin.H{"error": "价格监控服务未初始化"})
		return
	}

	s.priceMonitor.Start()
	c.JSON(200, gin.H{
		"success": true,
		"message": "价格监控服务已启动",
	})
}

// StopPriceMonitor 停止价格监控
func (s *Server) StopPriceMonitor(c *gin.Context) {
	if s.priceMonitor == nil {
		c.JSON(500, gin.H{"error": "价格监控服务未初始化"})
		return
	}

	s.priceMonitor.Stop()
	c.JSON(200, gin.H{
		"success": true,
		"message": "价格监控服务已停止",
	})
}
