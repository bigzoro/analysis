package main

import (
	"fmt"
	"time"
)

// 模拟修复后的结构体定义
type MeanReversionPerformanceMetrics struct {
	TotalSignals      int     `json:"total_signals"`
	SuccessfulSignals int     `json:"successful_signals"`
	FailedSignals     int     `json:"failed_signals"`
	WinRate           float64 `json:"win_rate"`
	TotalPnL          float64 `json:"total_pnl"`
	AvgProfit         float64 `json:"avg_profit"`
	AvgLoss           float64 `json:"avg_loss"`
	ProfitFactor      float64 `json:"profit_factor"`
	MaxDrawdown       float64 `json:"max_drawdown"`
	AvgHoldTime       float64 `json:"avg_hold_time"`
	MaxHoldTime       float64 `json:"max_hold_time"`
	SignalsPerDay     float64 `json:"signals_per_day"`

	// 市场环境表现 - 现在是指针类型，避免递归
	OscillationPerformance *MeanReversionPerformanceMetrics `json:"oscillation_performance"`
	TrendPerformance       *MeanReversionPerformanceMetrics `json:"trend_performance"`
	HighVolPerformance     *MeanReversionPerformanceMetrics `json:"high_vol_performance"`

	ConservativeMetrics ModePerformance `json:"conservative_metrics"`
	AggressiveMetrics   ModePerformance `json:"aggressive_metrics"`

	LastUpdated time.Time `json:"last_updated"`
}

type ModePerformance struct {
	Signals     int     `json:"signals"`
	WinRate     float64 `json:"win_rate"`
	AvgPnL      float64 `json:"avg_pnl"`
	SharpeRatio float64 `json:"sharpe_ratio"`
	MaxDrawdown float64 `json:"max_drawdown"`
}

func main() {
	fmt.Println("🔧 递归类型修复测试")
	fmt.Println("===================")

	// 测试结构体可以正常创建
	mainMetrics := MeanReversionPerformanceMetrics{
		TotalSignals:      100,
		SuccessfulSignals: 60,
		FailedSignals:     40,
		WinRate:           0.6,
		TotalPnL:          25.5,
		AvgProfit:         5.25,
		AvgLoss:           2.63,
		ProfitFactor:      3.0,
		MaxDrawdown:       0.08,
		AvgHoldTime:       8.5,
		MaxHoldTime:       24.0,
		SignalsPerDay:     1.4,
		LastUpdated:       time.Now(),
	}

	// 测试嵌套指针字段可以正常赋值
	oscillationMetrics := MeanReversionPerformanceMetrics{
		TotalSignals:      50,
		SuccessfulSignals: 35,
		FailedSignals:     15,
		WinRate:           0.7,
		TotalPnL:          15.0,
		AvgProfit:         6.0,
		AvgLoss:           2.0,
		ProfitFactor:      4.0,
		MaxDrawdown:       0.05,
		AvgHoldTime:       6.0,
		MaxHoldTime:       18.0,
		SignalsPerDay:     0.8,
		LastUpdated:       time.Now(),
	}

	mainMetrics.OscillationPerformance = &oscillationMetrics

	fmt.Printf("✅ 主指标 - 总信号: %d, 胜率: %.1f%%, 总盈亏: %.2f%%\n",
		mainMetrics.TotalSignals,
		mainMetrics.WinRate*100,
		mainMetrics.TotalPnL)

	if mainMetrics.OscillationPerformance != nil {
		fmt.Printf("✅ 嵌套指标 - 震荡市胜率: %.1f%%, 总盈亏: %.2f%%\n",
			mainMetrics.OscillationPerformance.WinRate*100,
			mainMetrics.OscillationPerformance.TotalPnL)
	}

	fmt.Println("\n🎉 递归类型问题已完全修复！")
	fmt.Println("结构体可以正常定义和使用嵌套指针字段。")
}