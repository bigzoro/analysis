package main

import (
	"fmt"
	"math"
)

// 完整的AI止损测试，模拟实际逻辑
func testAIStopLoss() {
	fmt.Println("=== AI止损逻辑完整测试 ===\n")

	// 测试参数 - 基于日志数据
	atr := 0.01062
	marketRegime := "weak_bear"
	winRate := 0.0
	totalTrades := 2
	maxDrawdown := 0.0
	holdTime := 470
	pnl := -0.0105

	fmt.Printf("输入参数:\n")
	fmt.Printf("  ATR: %.3f%%\n", atr*100)
	fmt.Printf("  市场环境: %s\n", marketRegime)
	fmt.Printf("  胜率: %.1f%% (%d交易)\n", winRate*100, totalTrades)
	fmt.Printf("  最大回撤: %.1f%%\n", maxDrawdown*100)
	fmt.Printf("  持有时间: %d周期\n", holdTime)
	fmt.Printf("  当前PNL: %.2f%%\n\n", pnl*100)

	// 1. ATR基础止损计算 - 使用实际代码逻辑
	var atrMultiplier float64
	switch marketRegime {
	case "strong_bear":
		atrMultiplier = 1.2
	case "weak_bear":
		atrMultiplier = 1.0
	case "sideways":
		atrMultiplier = 0.8
	case "weak_bull", "strong_bull":
		atrMultiplier = 0.6
	default:
		atrMultiplier = 1.0
	}
	atrBasedStopLoss := atr * atrMultiplier
	fmt.Printf("1. ATR基础止损: %.3f%% (ATR %.3f%% × 倍数 %.1f)\n", atrBasedStopLoss*100, atr*100, atrMultiplier)

	// 2. 表现调整
	var performanceAdjustment float64
	if totalTrades == 0 {
		performanceAdjustment = 1.0
	} else {
		winRateFactor := 0.6 // 胜率0%，极低
		drawdownFactor := 1.0 // 回撤0%，正常
		performanceAdjustment = math.Max(0.5, math.Min(winRateFactor*drawdownFactor, 1.5))
	}
	fmt.Printf("2. 表现调整因子: %.2f (胜率低，收紧止损)\n", performanceAdjustment)

	// 3. 时间调整
	var timeAdjustment float64
	if holdTime <= 10 {
		timeAdjustment = 0.8
	} else if holdTime <= 50 {
		timeAdjustment = 1.0
	} else {
		if pnl > 0 {
			timeAdjustment = 1.2
		} else {
			timeAdjustment = 0.9
		}
	}
	fmt.Printf("3. 时间调整因子: %.2f (长期持仓且亏损，适度收紧)\n", timeAdjustment)

	// 4. 综合调整
	adjustedStopLoss := atrBasedStopLoss * performanceAdjustment * timeAdjustment
	fmt.Printf("4. 综合调整止损: %.3f%%\n", adjustedStopLoss*100)

	// 5. ML优化
	regimeScore := 0.1 // weak_bear
	winRateScore := 0.2 // winRate < 0.3
	timeScore := 0.8 // holdTime > 100
	combinedScore := (regimeScore + winRateScore + timeScore) / 3.0
	mlAdjustment := combinedScore*0.8 + 0.2

	if pnl > 0.05 {
		mlAdjustment *= 1.1
	} else if pnl < -0.03 {
		mlAdjustment *= 0.95
	}

	mlOptimizedStopLoss := atrBasedStopLoss * mlAdjustment
	mlOptimizedStopLoss = math.Max(0.008, math.Min(mlOptimizedStopLoss, 0.25))
	mlAdjustmentFactor := mlOptimizedStopLoss / atrBasedStopLoss

	fmt.Printf("5. ML优化:\n")
	fmt.Printf("   市场评分: %.2f, 胜率评分: %.2f, 时间评分: %.2f\n", regimeScore, winRateScore, timeScore)
	fmt.Printf("   综合评分: %.2f, 调整因子: %.2f\n", combinedScore, mlAdjustment)
	fmt.Printf("   ML优化止损: %.3f%%, 调整因子: %.2f\n", mlOptimizedStopLoss*100, mlAdjustmentFactor)

	// 6. 最终止损
	finalStopLoss := math.Min(adjustedStopLoss, mlOptimizedStopLoss)
	fmt.Printf("6. 最终止损阈值: %.3f%%\n", finalStopLoss*100)

	// 7. 实际使用的动态止损
	dynamicStopLoss := -finalStopLoss
	fmt.Printf("7. 动态止损值: %.3f%%\n", dynamicStopLoss*100)

	// 8. 止损判断
	shouldStopLoss := pnl <= -math.Abs(dynamicStopLoss)
	fmt.Printf("8. 止损判断: %t\n", shouldStopLoss)
	fmt.Printf("   条件: %.2f%% <= %.2f%% (%t)\n", pnl*100, -math.Abs(dynamicStopLoss)*100, pnl <= -math.Abs(dynamicStopLoss))

	fmt.Printf("\n=== 关键发现 ===\n")
	fmt.Printf("SOLUSDT在持有%d周期后仍未止损，说明之前的止损设置过宽松\n", holdTime)
	fmt.Printf("新的AI止损应该在早期就触发止损，避免过度亏损\n")

	// 9. 验证修复效果
	fmt.Printf("\n=== 修复验证 ===\n")
	fmt.Printf("假设在第50周期时，亏损-0.5%%:\n")
	earlyPnl := -0.005
	earlyHoldTime := 50
	earlyTimeAdjustment := 1.0
	earlyAdjustedStopLoss := atrBasedStopLoss * performanceAdjustment * earlyTimeAdjustment
	earlyFinalStopLoss := math.Min(earlyAdjustedStopLoss, mlOptimizedStopLoss)
	earlyShouldStopLoss := earlyPnl <= -earlyFinalStopLoss
	fmt.Printf("  第%d周期，亏损%.1f%%，止损阈值%.3f%%，应该止损: %t\n",
		earlyHoldTime, earlyPnl*100, earlyFinalStopLoss*100, earlyShouldStopLoss)
}

func main() {
	testAIStopLoss()
}
