package server

import (
	"fmt"
	"log"
	"time"
)

// checkThresholds 检查阈值并生成告警
func (rm *RiskMonitor) checkThresholds(profile *RiskProfile) []RiskAlert {
	alerts := make([]RiskAlert, 0)

	// 检查风险分数阈值
	riskScore := profile.RiskScore
	if riskScore >= rm.config.Monitoring.AlertThresholds["critical"] {
		alert := RiskAlert{
			ID:        fmt.Sprintf("critical_risk_%s_%d", profile.Symbol, time.Now().Unix()),
			Timestamp: time.Now(),
			AlertType: "risk_score",
			Severity:  "critical",
			Message:   fmt.Sprintf("风险分数过高: %.2f (阈值: %.2f)", riskScore, rm.config.Monitoring.AlertThresholds["critical"]),
			Symbol:    profile.Symbol,
			RiskScore: riskScore,
			Threshold: rm.config.Monitoring.AlertThresholds["critical"],
		}
		alerts = append(alerts, alert)
	} else if riskScore >= rm.config.Monitoring.AlertThresholds["high_risk"] {
		alert := RiskAlert{
			ID:        fmt.Sprintf("high_risk_%s_%d", profile.Symbol, time.Now().Unix()),
			Timestamp: time.Now(),
			AlertType: "risk_score",
			Severity:  "high",
			Message:   fmt.Sprintf("高风险分数: %.2f (阈值: %.2f)", riskScore, rm.config.Monitoring.AlertThresholds["high_risk"]),
			Symbol:    profile.Symbol,
			RiskScore: riskScore,
			Threshold: rm.config.Monitoring.AlertThresholds["high_risk"],
		}
		alerts = append(alerts, alert)
	}

	// 检查波动率阈值
	if profile.RiskFactors.Volatility >= rm.config.Monitoring.AlertThresholds["volatility"] {
		alert := RiskAlert{
			ID:        fmt.Sprintf("volatility_%s_%d", profile.Symbol, time.Now().Unix()),
			Timestamp: time.Now(),
			AlertType: "volatility",
			Severity:  "medium",
			Message:   fmt.Sprintf("波动率过高: %.2f (阈值: %.2f)", profile.RiskFactors.Volatility, rm.config.Monitoring.AlertThresholds["volatility"]),
			Symbol:    profile.Symbol,
			RiskScore: profile.RiskScore,
			Threshold: rm.config.Monitoring.AlertThresholds["volatility"],
		}
		alerts = append(alerts, alert)
	}

	// 检查回撤阈值
	if len(profile.HistoricalRisk) > 0 {
		recentHistory := profile.HistoricalRisk
		if len(recentHistory) > 10 {
			recentHistory = recentHistory[len(recentHistory)-10:] // 最近10条记录
		}

		maxValue := 0.0
		currentDrawdown := 0.0

		for _, history := range recentHistory {
			pnl := history.PnL
			if pnl > maxValue {
				maxValue = pnl
			}

			if maxValue > 0 {
				currentDrawdown = (maxValue - pnl) / maxValue
			}
		}

		if currentDrawdown >= rm.config.Monitoring.AlertThresholds["drawdown"] {
			alert := RiskAlert{
				ID:        fmt.Sprintf("drawdown_%s_%d", profile.Symbol, time.Now().Unix()),
				Timestamp: time.Now(),
				AlertType: "drawdown",
				Severity:  "high",
				Message:   fmt.Sprintf("回撤过高: %.2f%% (阈值: %.2f%%)", currentDrawdown*100, rm.config.Monitoring.AlertThresholds["drawdown"]*100),
				Symbol:    profile.Symbol,
				RiskScore: profile.RiskScore,
				Threshold: rm.config.Monitoring.AlertThresholds["drawdown"],
			}
			alerts = append(alerts, alert)
		}
	}

	// 检查流动性风险
	if profile.RiskFactors.Liquidity >= 0.9 {
		alert := RiskAlert{
			ID:        fmt.Sprintf("liquidity_%s_%d", profile.Symbol, time.Now().Unix()),
			Timestamp: time.Now(),
			AlertType: "liquidity",
			Severity:  "high",
			Message:   fmt.Sprintf("流动性风险极高: %.2f", profile.RiskFactors.Liquidity),
			Symbol:    profile.Symbol,
			RiskScore: profile.RiskScore,
			Threshold: 0.9,
		}
		alerts = append(alerts, alert)
	}

	// 检查市场风险
	if profile.RiskFactors.MarketRisk >= 0.8 {
		alert := RiskAlert{
			ID:        fmt.Sprintf("market_risk_%s_%d", profile.Symbol, time.Now().Unix()),
			Timestamp: time.Now(),
			AlertType: "market_risk",
			Severity:  "medium",
			Message:   fmt.Sprintf("市场风险较高: %.2f", profile.RiskFactors.MarketRisk),
			Symbol:    profile.Symbol,
			RiskScore: profile.RiskScore,
			Threshold: 0.8,
		}
		alerts = append(alerts, alert)
	}

	return alerts
}

// notifySubscribers 通知订阅者
func (rm *RiskMonitor) notifySubscribers(alert RiskAlert) {
	rm.subscribersMu.RLock()
	subscribers := make([]RiskAlertSubscriber, len(rm.subscribers))
	copy(subscribers, rm.subscribers)
	rm.subscribersMu.RUnlock()

	for _, subscriber := range subscribers {
		go func(sub RiskAlertSubscriber, a RiskAlert) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("[RiskMonitor] 通知订阅者失败: %v", r)
				}
			}()
			sub.OnRiskAlert(a)
		}(subscriber, alert)
	}
}

// GetActiveAlerts 获取活跃告警
func (rm *RiskMonitor) GetActiveAlerts() []RiskAlert {
	rm.alertsMu.RLock()
	defer rm.alertsMu.RUnlock()

	activeAlerts := make([]RiskAlert, 0)
	cutoffTime := time.Now().Add(-24 * time.Hour) // 最近24小时

	for _, alert := range rm.alerts {
		if alert.Timestamp.After(cutoffTime) {
			activeAlerts = append(activeAlerts, alert)
		}
	}

	return activeAlerts
}

// GetAlertsBySeverity 按严重程度获取告警
func (rm *RiskMonitor) GetAlertsBySeverity(severity string) []RiskAlert {
	activeAlerts := rm.GetActiveAlerts()
	filteredAlerts := make([]RiskAlert, 0)

	for _, alert := range activeAlerts {
		if alert.Severity == severity {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	return filteredAlerts
}

// GetAlertsBySymbol 按资产获取告警
func (rm *RiskMonitor) GetAlertsBySymbol(symbol string) []RiskAlert {
	activeAlerts := rm.GetActiveAlerts()
	filteredAlerts := make([]RiskAlert, 0)

	for _, alert := range activeAlerts {
		if alert.Symbol == symbol {
			filteredAlerts = append(filteredAlerts, alert)
		}
	}

	return filteredAlerts
}

// AcknowledgeAlert 确认告警
func (rm *RiskMonitor) AcknowledgeAlert(alertID string) error {
	rm.alertsMu.Lock()
	defer rm.alertsMu.Unlock()

	for i, alert := range rm.alerts {
		if alert.ID == alertID {
			// 这里可以添加确认标记或从列表中移除
			// 暂时从列表中移除
			rm.alerts = append(rm.alerts[:i], rm.alerts[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("告警ID不存在: %s", alertID)
}

// ClearOldAlerts 清除过期告警
func (rm *RiskMonitor) ClearOldAlerts(maxAge time.Duration) int {
	rm.alertsMu.Lock()
	defer rm.alertsMu.Unlock()

	cutoffTime := time.Now().Add(-maxAge)
	oldCount := 0
	newAlerts := make([]RiskAlert, 0)

	for _, alert := range rm.alerts {
		if alert.Timestamp.After(cutoffTime) {
			newAlerts = append(newAlerts, alert)
		} else {
			oldCount++
		}
	}

	rm.alerts = newAlerts
	return oldCount
}

// GenerateAlertReport 生成告警报告
func (rm *RiskMonitor) GenerateAlertReport() map[string]interface{} {
	activeAlerts := rm.GetActiveAlerts()

	report := map[string]interface{}{
		"total_alerts":      len(activeAlerts),
		"critical_alerts":   0,
		"high_alerts":       0,
		"medium_alerts":     0,
		"low_alerts":        0,
		"alerts_by_type":    make(map[string]int),
		"alerts_by_symbol":  make(map[string]int),
		"most_recent_alert": "",
		"generated_at":      time.Now(),
	}

	alertsByType := make(map[string]int)
	alertsBySymbol := make(map[string]int)

	var mostRecent *RiskAlert

	for _, alert := range activeAlerts {
		// 按严重程度统计
		switch alert.Severity {
		case "critical":
			report["critical_alerts"] = report["critical_alerts"].(int) + 1
		case "high":
			report["high_alerts"] = report["high_alerts"].(int) + 1
		case "medium":
			report["medium_alerts"] = report["medium_alerts"].(int) + 1
		case "low":
			report["low_alerts"] = report["low_alerts"].(int) + 1
		}

		// 按类型统计
		alertsByType[alert.AlertType]++

		// 按资产统计
		alertsBySymbol[alert.Symbol]++

		// 找到最近的告警
		if mostRecent == nil || alert.Timestamp.After(mostRecent.Timestamp) {
			mostRecent = &alert
		}
	}

	report["alerts_by_type"] = alertsByType
	report["alerts_by_symbol"] = alertsBySymbol

	if mostRecent != nil {
		report["most_recent_alert"] = fmt.Sprintf("%s: %s", mostRecent.Symbol, mostRecent.Message)
	}

	return report
}

// MonitorPortfolioRisk 监控投资组合风险
func (rm *RiskMonitor) MonitorPortfolioRisk(positions map[string]float64) []RiskAlert {
	alerts := make([]RiskAlert, 0)

	// 计算投资组合总风险
	totalRisk := 0.0
	totalWeight := 0.0

	riskBySymbol := make(map[string]float64)

	for symbol, weight := range positions {
		// 这里应该从风险管理系统获取实际风险分数
		// 暂时使用模拟风险分数
		riskScore := 40.0 + 30.0*weight // 简化的风险计算
		riskBySymbol[symbol] = riskScore

		totalRisk += riskScore * weight
		totalWeight += weight
	}

	if totalWeight > 0 {
		portfolioRisk := totalRisk / totalWeight

		// 检查投资组合风险阈值
		if portfolioRisk >= rm.config.Monitoring.AlertThresholds["critical"] {
			alert := RiskAlert{
				ID:        fmt.Sprintf("portfolio_critical_%d", time.Now().Unix()),
				Timestamp: time.Now(),
				AlertType: "portfolio_risk",
				Severity:  "critical",
				Message:   fmt.Sprintf("投资组合风险极高: %.2f", portfolioRisk),
				Symbol:    "PORTFOLIO",
				RiskScore: portfolioRisk,
				Threshold: rm.config.Monitoring.AlertThresholds["critical"],
			}
			alerts = append(alerts, alert)
		} else if portfolioRisk >= rm.config.Monitoring.AlertThresholds["high_risk"] {
			alert := RiskAlert{
				ID:        fmt.Sprintf("portfolio_high_%d", time.Now().Unix()),
				Timestamp: time.Now(),
				AlertType: "portfolio_risk",
				Severity:  "high",
				Message:   fmt.Sprintf("投资组合风险较高: %.2f", portfolioRisk),
				Symbol:    "PORTFOLIO",
				RiskScore: portfolioRisk,
				Threshold: rm.config.Monitoring.AlertThresholds["high_risk"],
			}
			alerts = append(alerts, alert)
		}

		// 检查集中度风险
		maxWeight := 0.0
		maxWeightSymbol := ""
		for symbol, weight := range positions {
			if weight > maxWeight {
				maxWeight = weight
				maxWeightSymbol = symbol
			}
		}

		if maxWeight > 0.3 { // 单个资产超过30%
			alert := RiskAlert{
				ID:        fmt.Sprintf("concentration_%s_%d", maxWeightSymbol, time.Now().Unix()),
				Timestamp: time.Now(),
				AlertType: "concentration",
				Severity:  "medium",
				Message:   fmt.Sprintf("资产集中度过高: %s占比%.1f%%", maxWeightSymbol, maxWeight*100),
				Symbol:    maxWeightSymbol,
				RiskScore: riskBySymbol[maxWeightSymbol],
				Threshold: 0.3,
			}
			alerts = append(alerts, alert)
		}
	}

	return alerts
}

// StartRealTimeMonitoring 启动实时监控
func (rm *RiskMonitor) StartRealTimeMonitoring() {
	if !rm.config.Monitoring.EnableRealTime {
		return
	}

	log.Printf("[RiskMonitor] 启动实时风险监控")

	// 这里可以启动WebSocket连接或其他实时数据源
	// 监听市场数据变化并实时评估风险
}

// StopRealTimeMonitoring 停止实时监控
func (rm *RiskMonitor) StopRealTimeMonitoring() {
	log.Printf("[RiskMonitor] 停止实时风险监控")
	// 清理资源
}

// SetAlertThreshold 更新告警阈值
func (rm *RiskMonitor) SetAlertThreshold(alertType string, threshold float64) {
	rm.config.Monitoring.AlertThresholds[alertType] = threshold
	log.Printf("[RiskMonitor] 更新告警阈值 - %s: %.2f", alertType, threshold)
}

// GetAlertThresholds 获取告警阈值
func (rm *RiskMonitor) GetAlertThresholds() map[string]float64 {
	thresholds := make(map[string]float64)
	for k, v := range rm.config.Monitoring.AlertThresholds {
		thresholds[k] = v
	}
	return thresholds
}

// ExportAlerts 导出告警数据
func (rm *RiskMonitor) ExportAlerts(format string) (string, error) {
	activeAlerts := rm.GetActiveAlerts()

	switch format {
	case "json":
		// 这里应该返回JSON格式的数据
		return fmt.Sprintf("JSON export of %d alerts", len(activeAlerts)), nil
	case "csv":
		// 这里应该返回CSV格式的数据
		return fmt.Sprintf("CSV export of %d alerts", len(activeAlerts)), nil
	default:
		return "", fmt.Errorf("不支持的导出格式: %s", format)
	}
}

// GetMonitoringStats 获取监控统计信息
func (rm *RiskMonitor) GetMonitoringStats() map[string]interface{} {
	activeAlerts := rm.GetActiveAlerts()

	stats := map[string]interface{}{
		"active_alerts":       len(activeAlerts),
		"monitoring_enabled":  rm.config.Monitoring.EnableRealTime,
		"monitoring_interval": rm.config.Monitoring.MonitoringInterval.String(),
		"report_interval":     rm.config.Monitoring.ReportInterval.String(),
		"alert_thresholds":    rm.GetAlertThresholds(),
		"subscribers_count":   0, // 需要实际计算
		"last_check":          time.Now(),
	}

	rm.subscribersMu.RLock()
	stats["subscribers_count"] = len(rm.subscribers)
	rm.subscribersMu.RUnlock()

	return stats
}
