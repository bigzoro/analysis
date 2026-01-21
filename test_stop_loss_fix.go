package main

import (
	"fmt"
	"math"
)

// 简化的AI止损计算测试
func calculateATRBasedStopLoss(atr float64, marketRegime string) float64 {
	// ATR倍数基于市场环境
	var atrMultiplier float64
	switch marketRegime {
	case "weak_bear":
		atrMultiplier = 2.5
	case "strong_bear":
		atrMultiplier = 3.0
	default:
		atrMultiplier = 1.5
	}

	atrBasedStopLoss := atr * atrMultiplier
	return atrBasedStopLoss
}

func calculatePerformanceBasedStopAdjustment(winRate float64, totalTrades int, maxDrawdown float64) float64 {
	if totalTrades == 0 {
		return 1.0 // 无交易历史，默认因子
	}

	// 基于胜率调整
	var winRateFactor float64
	if winRate >= 0.75 {
		winRateFactor = 1.2 // 高胜率，适度放宽
	} else if winRate >= 0.5 {
		winRateFactor = 1.0 // 中等胜率，保持不变
	} else if winRate >= 0.25 {
		winRateFactor = 0.8 // 低胜率，收紧止损
	} else {
		winRateFactor = 0.6 // 极低胜率，大幅收紧
	}

	// 基于回撤调整
	var drawdownFactor float64
	if maxDrawdown > 0.2 {
		drawdownFactor = 0.7 // 高回撤，大幅收紧
	} else if maxDrawdown > 0.1 {
		drawdownFactor = 0.85 // 中等回撤，适度收紧
	} else {
		drawdownFactor = 1.0 // 低回撤，保持不变
	}

	adjustment := winRateFactor * drawdownFactor
	return math.Max(0.5, math.Min(adjustment, 1.5)) // 限制在0.5-1.5之间
}

func calculateTimeBasedStopAdjustment(holdTime int, pnl float64) float64 {
	if holdTime <= 10 {
		return 0.8 // 短期持仓，更严格的止损
	} else if holdTime <= 50 {
		return 1.0 // 中期持仓，正常止损
	} else {
		if pnl > 0 {
			return 1.2 // 长期持仓且盈利，放宽止损
		} else {
			return 0.9 // 长期持仓但亏损，适度收紧
		}
	}
}

func calculateMLOptimizedStopLoss(atrBasedStopLoss float64, marketRegime string, winRate float64, holdTime int, pnl float64) float64 {
	// 简化的ML逻辑
	baseStopLoss := atrBasedStopLoss

	// 市场环境评分
	var regimeScore float64
	switch marketRegime {
	case "weak_bear":
		regimeScore = 0.1
	case "strong_bear":
		regimeScore = 0.05
	default:
		regimeScore = 0.5
	}

	// 胜率评分
	var winRateScore float64
	if winRate < 0.3 {
		winRateScore = 0.2
	} else if winRate < 0.5 {
		winRateScore = 0.4
	} else {
		winRateScore = 0.7
	}

	// 持仓时间评分
	var timeScore float64
	if holdTime > 100 {
		timeScore = 0.8
	} else if holdTime > 50 {
		timeScore = 0.6
	} else {
		timeScore = 0.4
	}

	combinedScore := (regimeScore + winRateScore + timeScore) / 3.0

	// 根据综合评分调整
	mlAdjustment := combinedScore * 0.8 + 0.2 // 确保最小调整因子

	// 当前盈利/亏损调整
	if pnl > 0.05 {
		mlAdjustment *= 1.1 // 大幅盈利，适度放宽止损
	} else if pnl < -0.03 {
		mlAdjustment *= 0.95 // 大幅亏损，轻微收紧止损
	}

	optimizedStopLoss := baseStopLoss * mlAdjustment

	// 限制范围
	optimizedStopLoss = math.Max(0.008, math.Min(optimizedStopLoss, 0.25))

	return optimizedStopLoss / baseStopLoss // 返回调整因子
}

func main() {
	// 测试SOLUSDT的情况
	atr := 0.01062 // 1.062%
	marketRegime := "weak_bear"
	winRate := 0.0 // 0%
	totalTrades := 2
	maxDrawdown := 0.0 // 0%
	holdTime := 470
	pnl := -0.0105 // -1.05%

	fmt.Printf("测试SOLUSDT止损计算:\n")
	fmt.Printf("ATR: %.3f%%\n", atr*100)
	fmt.Printf("市场环境: %s\n", marketRegime)
	fmt.Printf("胜率: %.1f%% (%d交易)\n", winRate*100, totalTrades)
	fmt.Printf("最大回撤: %.1f%%\n", maxDrawdown*100)
	fmt.Printf("持有时间: %d周期\n", holdTime)
	fmt.Printf("当前PNL: %.2f%%\n\n", pnl*100)

	// 计算各组件
	atrBasedStopLoss := calculateATRBasedStopLoss(atr, marketRegime)
	fmt.Printf("ATR基础止损: %.3f%%\n", atrBasedStopLoss*100)

	performanceAdjustment := calculatePerformanceBasedStopAdjustment(winRate, totalTrades, maxDrawdown)
	fmt.Printf("表现调整因子: %.2f\n", performanceAdjustment)

	timeAdjustment := calculateTimeBasedStopAdjustment(holdTime, pnl)
	fmt.Printf("时间调整因子: %.2f\n", timeAdjustment)

	// 综合调整
	adjustedStopLoss := atrBasedStopLoss * performanceAdjustment * timeAdjustment
	fmt.Printf("综合调整止损: %.3f%%\n", adjustedStopLoss*100)

	// ML优化
	mlAdjustmentFactor := calculateMLOptimizedStopLoss(atrBasedStopLoss, marketRegime, winRate, holdTime, pnl)
	mlOptimizedStopLoss := atrBasedStopLoss * mlAdjustmentFactor
	fmt.Printf("ML调整因子: %.2f\n", mlAdjustmentFactor)
	fmt.Printf("ML优化止损: %.3f%%\n", mlOptimizedStopLoss*100)

	// 最终止损
	finalStopLoss := math.Min(adjustedStopLoss, mlOptimizedStopLoss)
	fmt.Printf("最终止损阈值: %.3f%%\n", finalStopLoss*100)

	// 检查是否应该触发止损
	shouldStopLoss := pnl <= -finalStopLoss
	fmt.Printf("应该止损: %t (PNL %.2f%% <= -%.3f%%)\n", shouldStopLoss, pnl*100, finalStopLoss*100)

	fmt.Printf("\n对比旧的错误计算:\n")
	// 旧的错误计算（直接使用百分比作为因子）
	mlOptimizedStopLossOld := atrBasedStopLoss * (mlOptimizedStopLoss / atrBasedStopLoss) // 这会导致错误的平方计算
	fmt.Printf("旧ML止损(错误): %.3f%%\n", mlOptimizedStopLossOld*100)
}

