package main

import "fmt"

func main() {
	fmt.Println("=== FHEUSDT杠杆交易盈亏计算分析 ===")

	// 当前数据
	avgCost := 0.16466000    // 平均持仓成本
	markPrice := 0.16680000  // 标记价格
	quantity := 547.00000000 // 持仓数量
	leverage := 3.0          // 杠杆倍数

	fmt.Printf("交易参数:\n")
	fmt.Printf("  平均持仓成本: %.8f USDT\n", avgCost)
	fmt.Printf("  当前标记价格: %.8f USDT\n", markPrice)
	fmt.Printf("  持仓数量: %.8f\n", quantity)
	fmt.Printf("  杠杆倍数: %.1fx\n", leverage)
	fmt.Printf("  持仓方向: 做空\n")

	// 计算名义价值
	nominalValue := quantity * avgCost
	margin := nominalValue / leverage

	fmt.Printf("\n持仓价值计算:\n")
	fmt.Printf("  名义价值: %.8f × %.8f = %.8f USDT\n", quantity, avgCost, nominalValue)
	fmt.Printf("  保证金: %.8f ÷ %.1f = %.8f USDT\n", nominalValue, leverage, margin)

	// 计算未实现盈亏
	priceDiff := avgCost - markPrice // 做空盈利：成本价 > 市场价
	pnlPerContract := priceDiff      // 每张合约的盈亏
	totalPnL := quantity * pnlPerContract

	fmt.Printf("\n盈亏计算:\n")
	fmt.Printf("  价格差异: %.8f - %.8f = %.8f USDT/张\n", avgCost, markPrice, priceDiff)
	fmt.Printf("  总未实现盈亏: %.8f × %.8f = %.8f USDT\n", quantity, pnlPerContract, totalPnL)

	// 计算盈亏百分比（基于保证金）
	pnlPercentage := (totalPnL / margin) * 100

	fmt.Printf("\n盈亏百分比:\n")
	fmt.Printf("  基于保证金: (%.8f ÷ %.8f) × 100 = %.2f%%\n", totalPnL, margin, pnlPercentage)
	fmt.Printf("  基于名义价值: (%.8f ÷ %.8f) × 100 = %.2f%%\n", totalPnL, nominalValue, (totalPnL/nominalValue)*100)

	// 币安的实际计算方式分析
	fmt.Printf("\n=== 币安实际计算方式分析 ===\n")

	// 情况1：基于名义价值的百分比（不考虑杠杆）
	pnlPercentNominal := ((avgCost - markPrice) / avgCost) * 100
	fmt.Printf("1. 基于名义价值百分比: %.2f%%\n", pnlPercentNominal)

	// 情况2：基于保证金的百分比（杠杆放大）
	pnlPercentMargin := ((avgCost - markPrice) / avgCost) * leverage * 100
	fmt.Printf("2. 基于保证金百分比（杠杆放大）: %.2f%%\n", pnlPercentMargin)

	// 情况3：实际盈亏除以保证金
	actualPnlPercent := (totalPnL / margin) * 100
	fmt.Printf("3. 实际盈亏/保证金: %.2f%%\n", actualPnlPercent)

	// 情况4：币安可能使用的计算方式
	// 做空：(开仓价 - 标记价) / 开仓价 * 杠杆
	binanceStyle := ((avgCost - markPrice) / avgCost) * leverage * 100
	fmt.Printf("4. 币安风格计算: (开仓价-标记价)/开仓价 × 杠杆 = %.2f%%\n", binanceStyle)

	fmt.Printf("\n=== 结论 ===\n")
	fmt.Printf("如果币安显示-4%%，那么程序应该显示: %.2f%%\n", pnlPercentMargin)
	fmt.Printf("程序当前显示-1.54%%，确实没有考虑杠杆放大\n")

	// 修正建议
	fmt.Printf("\n修正建议:\n")
	fmt.Printf("盈利加仓判断应该使用: %.2f%% (杠杆放大后的百分比)\n", pnlPercentMargin)
	fmt.Printf("而不是当前的: %.2f%% (基础价格变动)\n", pnlPercentNominal)
}
