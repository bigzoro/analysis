package main

import "fmt"

func main() {
	fmt.Println("=== 整体止损触发条件判断逻辑详解 ===\n")

	// 策略配置
	stopLossEnabled := true
	stopLossPercent := 35.0 // 35%

	// 当前持仓情况
	symbol := "FHEUSDT"
	avgCost := 0.16466000
	currentPrice := 0.16680000
	leverage := 3.0

	fmt.Printf("策略配置:\n")
	fmt.Printf("  整体止损启用: %v\n", stopLossEnabled)
	fmt.Printf("  整体止损百分比: %.2f%%\n", stopLossPercent)
	fmt.Printf("\n当前持仓 (%s):\n", symbol)
	fmt.Printf("  平均成本: %.8f USDT\n", avgCost)
	fmt.Printf("  当前价格: %.8f USDT\n", currentPrice)
	fmt.Printf("  杠杆倍数: %.1fx\n", leverage)

	// 步骤1: 计算基础价格变动百分比
	basicProfitPercent := (avgCost - currentPrice) / avgCost
	fmt.Printf("\n步骤1: 计算基础价格变动百分比\n")
	fmt.Printf("  公式: (平均成本 - 当前价格) / 平均成本\n")
	fmt.Printf("  计算: (%.8f - %.8f) / %.8f = %.4f\n", avgCost, currentPrice, avgCost, basicProfitPercent)
	fmt.Printf("  结果: %.2f%%\n", basicProfitPercent*100)

	// 步骤2: 杠杆放大
	leveragedProfitPercent := basicProfitPercent * leverage
	fmt.Printf("\n步骤2: 杠杆放大\n")
	fmt.Printf("  公式: 基础百分比 × 杠杆倍数\n")
	fmt.Printf("  计算: %.2f%% × %.1f = %.2f%%\n", basicProfitPercent*100, leverage, leveragedProfitPercent*100)

	// 步骤3: 计算整体利润百分比（用于判断）
	overallProfitPercent := leveragedProfitPercent * 100
	fmt.Printf("\n步骤3: 计算整体利润百分比\n")
	fmt.Printf("  公式: 杠杆后百分比 × 100\n")
	fmt.Printf("  计算: %.4f × 100 = %.2f%%\n", leveragedProfitPercent, overallProfitPercent)

	// 步骤4: 判断是否触发止损
	stopLossThreshold := -stopLossPercent
	shouldTrigger := overallProfitPercent <= stopLossThreshold
	fmt.Printf("\n步骤4: 判断是否触发止损\n")
	fmt.Printf("  当前亏损: %.2f%%\n", overallProfitPercent)
	fmt.Printf("  止损阈值: %.2f%%\n", stopLossThreshold)
	fmt.Printf("  触发条件: %.2f%% <= %.2f%% ?\n", overallProfitPercent, stopLossThreshold)
	fmt.Printf("  判断结果: %v\n", shouldTrigger)

	// 详细的条件判断逻辑
	fmt.Printf("\n=== 详细的条件判断逻辑 ===\n")
	fmt.Printf("完整判断条件:\n")
	fmt.Printf("strategy.Conditions.OverallStopLossEnabled &&\n")
	fmt.Printf("strategy.Conditions.OverallStopLossPercent > 0 &&\n")
	fmt.Printf("overallProfitPercent <= -strategy.Conditions.OverallStopLossPercent\n")

	fmt.Printf("\n各部分判断:\n")
	fmt.Printf("1. 整体止损是否启用: %v → %v\n", stopLossEnabled, stopLossEnabled)
	fmt.Printf("2. 止损百分比是否>0: %.2f > 0 → %v\n", stopLossPercent, stopLossPercent > 0)
	fmt.Printf("3. 当前亏损是否超过阈值: %.2f%% <= -%.2f%% → %v\n",
		overallProfitPercent, stopLossPercent, overallProfitPercent <= -stopLossPercent)

	allConditionsMet := stopLossEnabled && stopLossPercent > 0 && overallProfitPercent <= -stopLossPercent
	fmt.Printf("\n总体判断结果: %v (所有条件都满足才会触发止损)\n", allConditionsMet)

	// 不同场景的示例
	fmt.Printf("\n=== 不同场景示例 ===\n")

	scenarios := []struct {
		name     string
		profit   float64
		expected bool
	}{
		{"轻微亏损(5%)", -5.0, false},
		{"中等亏损(20%)", -20.0, false},
		{"重度亏损(35%)", -35.0, true},
		{"严重亏损(50%)", -50.0, true},
		{"盈利(10%)", 10.0, false},
	}

	for _, scenario := range scenarios {
		trigger := scenario.profit <= -stopLossPercent
		fmt.Printf("%-15s: %.1f%% → 触发止损: %v %s\n",
			scenario.name, scenario.profit, trigger,
			map[bool]string{true: "(✓ 达到阈值)", false: "(✗ 未达到阈值)"}[trigger])
	}

	fmt.Printf("\n=== 关键要点 ===\n")
	fmt.Printf("1. 整体止损基于杠杆放大后的利润百分比进行判断\n")
	fmt.Printf("2. 使用 <= 比较，表示亏损达到或超过阈值时触发\n")
	fmt.Printf("3. 止损阈值前面有负号，表示亏损方向\n")
	fmt.Printf("4. 只有当所有条件都满足时才会触发整体平仓\n")
	fmt.Printf("5. 触发后会重置盈利加仓计数器，防止继续加仓\n")
}
