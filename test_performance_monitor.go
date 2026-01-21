package main

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

// 模拟结构体
type PerformanceMetrics struct {
	TotalSignals      int       `json:"total_signals"`
	SuccessfulSignals int       `json:"successful_signals"`
	FailedSignals     int       `json:"failed_signals"`
	WinRate           float64   `json:"win_rate"`
	TotalPnL          float64   `json:"total_pnl"`
	AvgProfit         float64   `json:"avg_profit"`
	AvgLoss           float64   `json:"avg_loss"`
	ProfitFactor      float64   `json:"profit_factor"`
	MaxDrawdown       float64   `json:"max_drawdown"`
	AvgHoldTime       float64   `json:"avg_hold_time"`
	MaxHoldTime       float64   `json:"max_hold_time"`
	SignalsPerDay     float64   `json:"signals_per_day"`
	LastUpdated       time.Time `json:"last_updated"`
}

type ModePerformance struct {
	Signals     int     `json:"signals"`
	WinRate     float64 `json:"win_rate"`
	AvgPnL      float64 `json:"avg_pnl"`
	SharpeRatio float64 `json:"sharpe_ratio"`
	MaxDrawdown float64 `json:"max_drawdown"`
}

type SignalRecord struct {
	Symbol          string    `json:"symbol"`
	Action          string    `json:"action"`
	EntryPrice      float64   `json:"entry_price"`
	StopLossPrice   float64   `json:"stop_loss_price"`
	TakeProfitPrice float64   `json:"take_profit_price"`
	Strength        float64   `json:"strength"`
	Confidence      float64   `json:"confidence"`
	MarketEnv       string    `json:"market_env"`
	SubMode         string    `json:"sub_mode"`
	Timestamp       time.Time `json:"timestamp"`
	Status          string    `json:"status"`
	ExitPrice       float64   `json:"exit_price,omitempty"`
	PnL             float64   `json:"pnl,omitempty"`
	HoldTime        float64   `json:"hold_time,omitempty"`
}

type PerformanceMonitor struct {
	signalHistory []SignalRecord
	metrics       map[string]*PerformanceMetrics
	mutex         sync.RWMutex
}

// 更新性能指标
func (pm *PerformanceMonitor) updateMetrics(modeKey string) {
	if pm.metrics[modeKey] == nil {
		pm.metrics[modeKey] = &PerformanceMetrics{
			LastUpdated: time.Now(),
		}
	}

	metrics := pm.metrics[modeKey]
	metrics.LastUpdated = time.Now()

	var modeSignals []SignalRecord
	for _, signal := range pm.signalHistory {
		if signal.Status != "active" {
			modeSignals = append(modeSignals, signal)
		}
	}

	if len(modeSignals) == 0 {
		return
	}

	metrics.TotalSignals = len(modeSignals)
	closedSignals := 0
	profitableSignals := 0
	totalPnL := 0.0
	totalProfits := 0.0
	totalLosses := 0.0
	totalHoldTime := 0.0
	maxHoldTime := 0.0

	for _, signal := range modeSignals {
		if signal.Status != "active" {
			closedSignals++
			totalPnL += signal.PnL
			totalHoldTime += signal.HoldTime

			if signal.HoldTime > maxHoldTime {
				maxHoldTime = signal.HoldTime
			}

			if signal.PnL > 0 {
				profitableSignals++
				totalProfits += signal.PnL
			} else {
				totalLosses += math.Abs(signal.PnL)
			}
		}
	}

	if closedSignals > 0 {
		metrics.SuccessfulSignals = profitableSignals
		metrics.FailedSignals = closedSignals - profitableSignals
		metrics.WinRate = float64(profitableSignals) / float64(closedSignals)
		metrics.TotalPnL = totalPnL
		metrics.AvgHoldTime = totalHoldTime / float64(closedSignals)
		metrics.MaxHoldTime = maxHoldTime

		if profitableSignals > 0 {
			metrics.AvgProfit = totalProfits / float64(profitableSignals)
		}
		if failedSignals := closedSignals - profitableSignals; failedSignals > 0 {
			metrics.AvgLoss = totalLosses / float64(failedSignals)
		}
		if metrics.AvgLoss != 0 {
			metrics.ProfitFactor = totalProfits / totalLosses
		}
	}

	// 计算日信号频率
	recentSignals := pm.getRecentSignals(7 * 24)
	if len(recentSignals) > 0 {
		metrics.SignalsPerDay = float64(len(recentSignals)) / 7.0
	}

	// 计算最大回撤
	if len(modeSignals) > 10 {
		metrics.MaxDrawdown = pm.calculateMaxDrawdown(modeSignals)
	}
}

// 获取最近N小时的信号
func (pm *PerformanceMonitor) getRecentSignals(hours int) []SignalRecord {
	cutoff := time.Now().Add(-time.Duration(hours) * time.Hour)
	var recent []SignalRecord

	for _, signal := range pm.signalHistory {
		if signal.Timestamp.After(cutoff) {
			recent = append(recent, signal)
		}
	}

	return recent
}

// 计算最大回撤
func (pm *PerformanceMonitor) calculateMaxDrawdown(signals []SignalRecord) float64 {
	if len(signals) < 2 {
		return 0
	}

	sortedSignals := make([]SignalRecord, len(signals))
	copy(sortedSignals, signals)
	sort.Slice(sortedSignals, func(i, j int) bool {
		return sortedSignals[i].Timestamp.Before(sortedSignals[j].Timestamp)
	})

	maxDrawdown := 0.0
	peak := 0.0
	current := 0.0

	for _, signal := range sortedSignals {
		current += signal.PnL

		if current > peak {
			peak = current
		}

		if peak > 0 {
			drawdown := (peak - current) / peak
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
	}

	return maxDrawdown
}

func main() {
	fmt.Println("📈 性能监控面板测试")
	fmt.Println("===================")

	pm := &PerformanceMonitor{
		signalHistory: make([]SignalRecord, 0),
		metrics:       make(map[string]*PerformanceMetrics),
	}

	// 模拟一些历史信号数据
	baseTime := time.Now().Add(-24 * time.Hour)
	testSignals := []SignalRecord{
		{Symbol: "BTC", Action: "buy", MarketEnv: "oscillation", SubMode: "conservative", Timestamp: baseTime, Status: "closed_profit", PnL: 0.05, HoldTime: 12},
		{Symbol: "ETH", Action: "sell", MarketEnv: "oscillation", SubMode: "conservative", Timestamp: baseTime.Add(2 * time.Hour), Status: "closed_loss", PnL: -0.02, HoldTime: 8},
		{Symbol: "ADA", Action: "buy", MarketEnv: "strong_trend", SubMode: "aggressive", Timestamp: baseTime.Add(4 * time.Hour), Status: "closed_profit", PnL: 0.08, HoldTime: 6},
		{Symbol: "DOT", Action: "buy", MarketEnv: "high_volatility", SubMode: "conservative", Timestamp: baseTime.Add(6 * time.Hour), Status: "closed_profit", PnL: 0.03, HoldTime: 10},
		{Symbol: "LINK", Action: "sell", MarketEnv: "oscillation", SubMode: "aggressive", Timestamp: baseTime.Add(8 * time.Hour), Status: "closed_loss", PnL: -0.015, HoldTime: 4},
		{Symbol: "UNI", Action: "buy", MarketEnv: "strong_trend", SubMode: "conservative", Timestamp: baseTime.Add(10 * time.Hour), Status: "closed_profit", PnL: 0.06, HoldTime: 14},
		{Symbol: "AAVE", Action: "buy", MarketEnv: "high_volatility", SubMode: "aggressive", Timestamp: baseTime.Add(12 * time.Hour), Status: "closed_loss", PnL: -0.04, HoldTime: 3},
		{Symbol: "SUSHI", Action: "sell", MarketEnv: "oscillation", SubMode: "conservative", Timestamp: baseTime.Add(14 * time.Hour), Status: "closed_profit", PnL: 0.025, HoldTime: 9},
		{Symbol: "COMP", Action: "buy", MarketEnv: "strong_trend", SubMode: "aggressive", Timestamp: baseTime.Add(16 * time.Hour), Status: "closed_profit", PnL: 0.07, HoldTime: 7},
		{Symbol: "MKR", Action: "sell", MarketEnv: "high_volatility", SubMode: "conservative", Timestamp: baseTime.Add(18 * time.Hour), Status: "closed_loss", PnL: -0.03, HoldTime: 5},
	}

	pm.signalHistory = testSignals

	// 更新指标
	pm.updateMetrics("conservative")
	pm.updateMetrics("aggressive")

	// 显示结果
	modes := []string{"conservative", "aggressive"}
	for _, mode := range modes {
		if metrics, exists := pm.metrics[mode]; exists {
			fmt.Printf("\n🎯 %s模式性能报告:\n", mode)
			fmt.Printf("  总信号数: %d\n", metrics.TotalSignals)
			fmt.Printf("  胜率: %.1f%%\n", metrics.WinRate*100)
			fmt.Printf("  总盈亏: %.2f%%\n", metrics.TotalPnL*100)
			fmt.Printf("  平均利润: %.2f%%\n", metrics.AvgProfit*100)
			fmt.Printf("  平均亏损: %.2f%%\n", metrics.AvgLoss*100)
			fmt.Printf("  盈利因子: %.2f\n", metrics.ProfitFactor)
			fmt.Printf("  最大回撤: %.1f%%\n", metrics.MaxDrawdown*100)
			fmt.Printf("  平均持仓时间: %.1f小时\n", metrics.AvgHoldTime)
			fmt.Printf("  日均信号数: %.1f\n", metrics.SignalsPerDay)
		}
	}

	fmt.Println("\n✅ 性能监控面板测试完成")
	fmt.Println("系统能够实时跟踪和对比不同模式的交易表现，")
	fmt.Println("为策略优化提供数据支持。")
}
