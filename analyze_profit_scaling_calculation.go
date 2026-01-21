package main

import "fmt"

func main() {
	fmt.Println("=== FHEUSDT策略33盈利加仓计算分析 ===")

	// 策略配置
	leverage := 3.0
	profitScalingThreshold := 1.00 // 1%

	// 当前持仓数据
	avgCost := 0.16466000      // 平均持仓成本
	currentPrice := 0.16720000 // 当前价格
	quantity := 547.00000000   // 持仓数量

	fmt.Printf("策略配置:\n")
	fmt.Printf("  杠杆倍数: %.1fx\n", leverage)
	fmt.Printf("  盈利加仓触发阈值: %.2f%%\n", profitScalingThreshold)
	fmt.Printf("\n当前持仓:\n")
	fmt.Printf("  平均持仓成本: %.8f USDT\n", avgCost)
	fmt.Printf("  当前价格: %.8f USDT\n", currentPrice)
	fmt.Printf("  持仓数量: %.8f\n", quantity)
	fmt.Printf("  持仓方向: 做空\n")

	// 1. 基础价格变动计算（策略监控日志使用）
	priceChangePercent := (avgCost - currentPrice) / avgCost
	fmt.Printf("\n1. 基础价格变动计算（策略日志）:\n")
	fmt.Printf("   公式: (平均成本 - 当前价格) / 平均成本\n")
	fmt.Printf("   计算: (%.8f - %.8f) / %.8f = %.4f = %.2f%%\n",
		avgCost, currentPrice, avgCost, priceChangePercent, priceChangePercent*100)

	// 2. 杠杆放大后的实际盈亏计算
	actualPnL := quantity * (avgCost - currentPrice) * leverage
	nominalValue := quantity * avgCost
	actualPnLPercent := (actualPnL / (nominalValue / leverage)) * 100

	fmt.Printf("\n2. 杠杆放大后的实际盈亏（订单详情页面显示）:\n")
	fmt.Printf("   名义价值: %.8f × %.8f = %.8f USDT\n", quantity, avgCost, nominalValue)
	fmt.Printf("   保证金: %.8f / %.1f = %.8f USDT\n", nominalValue, leverage, nominalValue/leverage)
	fmt.Printf("   未实现盈亏: %.8f × (%.8f - %.8f) × %.1f = %.8f USDT\n",
		quantity, avgCost, currentPrice, leverage, actualPnL)
	fmt.Printf("   盈亏百分比: (%.8f / %.8f) × 100 = %.2f%%\n",
		actualPnL, nominalValue/leverage, actualPnLPercent)

	// 3. 盈利加仓触发条件判断
	fmt.Printf("\n3. 盈利加仓触发条件:\n")
	fmt.Printf("   当前计算的盈利: %.2f%%\n", priceChangePercent*100)
	fmt.Printf("   触发阈值: %.2f%%\n", profitScalingThreshold)
	fmt.Printf("   是否触发加仓: %.2f%% >= %.2f%% ? %v\n",
		priceChangePercent*100, profitScalingThreshold, priceChangePercent*100 >= profitScalingThreshold)

	// 4. 如果使用杠杆放大后的百分比判断
	fmt.Printf("\n4. 如果使用杠杆后的盈亏百分比判断:\n")
	fmt.Printf("   杠杆后盈利: %.2f%%\n", actualPnLPercent)
	fmt.Printf("   是否触发加仓: %.2f%% >= %.2f%% ? %v\n",
		actualPnLPercent, profitScalingThreshold, actualPnLPercent >= profitScalingThreshold)

	// 5. 正确的盈利加仓计算方式建议
	fmt.Printf("\n5. 建议的正确计算方式:\n")
	fmt.Printf("   盈利加仓应该基于基础价格变动，而不是杠杆放大后的百分比\n")
	fmt.Printf("   原因: 杠杆已经体现在了持仓大小和名义价值上\n")
	fmt.Printf("   正确触发条件: %.2f%% >= %.2f%% = %v\n",
		priceChangePercent*100, profitScalingThreshold, priceChangePercent*100 >= profitScalingThreshold)

	fmt.Printf("\n=== 结论 ===\n")
	fmt.Printf("当前系统使用基础价格变动百分比 (%.2f%%) 来判断是否触发盈利加仓，这是正确的\n",
		priceChangePercent*100)
	fmt.Printf("用户看到的-4%%是订单详情页面的杠杆放大后盈亏显示，与加仓逻辑无关\n")
	fmt.Printf("FHEUSDT当前没有达到1%%的加仓阈值，所以不会触发盈利加仓\n")
}
