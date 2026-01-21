package main

import "fmt"

func main() {
	fmt.Println("=== 测试杠杆倍数在盈利加仓计算中的应用 ===")

	// 模拟FHEUSDT持仓数据
	avgCost := 0.16466000          // 平均持仓成本
	currentPrice := 0.16680000     // 当前价格
	leverage := 3.0                // 杠杆倍数
	profitScalingThreshold := 1.00 // 加仓阈值1%

	fmt.Printf("持仓数据:\n")
	fmt.Printf("  平均成本: %.8f USDT\n", avgCost)
	fmt.Printf("  当前价格: %.8f USDT\n", currentPrice)
	fmt.Printf("  杠杆倍数: %.1fx\n", leverage)
	fmt.Printf("  加仓阈值: %.2f%%\n", profitScalingThreshold)

	// 原来的计算（不考虑杠杆）
	oldProfitPercent := (avgCost - currentPrice) / avgCost
	fmt.Printf("\n原来的计算（不考虑杠杆）:\n")
	fmt.Printf("  利润百分比: %.2f%%\n", oldProfitPercent*100)
	fmt.Printf("  是否触发加仓: %.2f%% >= %.2f%% ? %v\n", oldProfitPercent*100, profitScalingThreshold, oldProfitPercent*100 >= profitScalingThreshold)

	// 修改后的计算（考虑杠杆）
	newProfitPercent := oldProfitPercent * leverage
	fmt.Printf("\n修改后的计算（考虑杠杆）:\n")
	fmt.Printf("  基础利润百分比: %.2f%%\n", oldProfitPercent*100)
	fmt.Printf("  杠杆放大后: %.2f%% × %.1f = %.2f%%\n", oldProfitPercent*100, leverage, newProfitPercent*100)
	fmt.Printf("  是否触发加仓: %.2f%% >= %.2f%% ? %v\n", newProfitPercent*100, profitScalingThreshold, newProfitPercent*100 >= profitScalingThreshold)

	// 实际盈亏计算验证
	nominalValue := 547.00000000 * avgCost // 547是持仓数量
	margin := nominalValue / leverage
	actualPnL := 547.00000000 * (avgCost - currentPrice)
	actualPnLPercent := (actualPnL / margin) * 100

	fmt.Printf("\n实际盈亏验证:\n")
	fmt.Printf("  名义价值: %.8f USDT\n", nominalValue)
	fmt.Printf("  保证金: %.8f USDT\n", margin)
	fmt.Printf("  实际盈亏: %.8f USDT\n", actualPnL)
	fmt.Printf("  实际盈亏百分比: %.2f%%\n", actualPnLPercent)
	fmt.Printf("  与杠杆计算一致: %.2f%% ≈ %.2f%% ? %v\n", actualPnLPercent, newProfitPercent*100, abs(actualPnLPercent-newProfitPercent*100) < 0.01)

	fmt.Printf("\n=== 结论 ===\n")
	fmt.Printf("修改后，FHEUSDT的盈利百分比从 %.2f%% 变为 %.2f%%，与币安显示的-4%%更接近\n", oldProfitPercent*100, newProfitPercent*100)
	fmt.Printf("这样可以正确判断是否应该触发盈利加仓\n")
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
