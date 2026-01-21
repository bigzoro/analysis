package main

import "fmt"

func main() {
	// FHEUSDT 做空持仓
	avgCost := 0.16466000      // 平均持仓成本（卖出价）
	currentPrice := 0.16720000 // 当前价格（买入价）

	// 做空盈利计算：(卖出价 - 当前买入价) / 卖出价
	profitPercent := (avgCost - currentPrice) / avgCost

	fmt.Printf("平均持仓成本: %.8f\n", avgCost)
	fmt.Printf("当前价格: %.8f\n", currentPrice)
	fmt.Printf("做空盈利计算: (%.8f - %.8f) / %.8f = %.4f\n", avgCost, currentPrice, avgCost, profitPercent)
	fmt.Printf("盈利率: %.2f%%\n", profitPercent*100)

	// 如果用户理解错误，认为当前价格是avgCost
	wrongProfitPercent := (currentPrice - avgCost) / avgCost
	fmt.Printf("\n如果误以为当前价格是平均成本，计算结果: %.2f%%\n", wrongProfitPercent*100)

	// 如果用户用当前价格作为分母
	wrongProfitPercent2 := (avgCost - currentPrice) / currentPrice
	fmt.Printf("如果用当前价格作为分母: (%.8f - %.8f) / %.8f = %.4f = %.2f%%\n",
		avgCost, currentPrice, currentPrice, wrongProfitPercent2, wrongProfitPercent2*100)

	// 可能的其他计算方式
	fmt.Printf("\n=== 其他可能的计算方式 ===\n")

	// 如果用户看到的价格是标记价格 0.16652481
	markPrice := 0.16652481
	markProfitPercent := (avgCost - markPrice) / avgCost
	fmt.Printf("使用标记价格 %.8f 计算: (%.8f - %.8f) / %.8f = %.4f = %.2f%%\n",
		markPrice, avgCost, markPrice, avgCost, markProfitPercent, markProfitPercent*100)

	// 如果用户看到的价格是最新成交价格 0.1666000
	lastPrice := 0.1666000
	lastProfitPercent := (avgCost - lastPrice) / avgCost
	fmt.Printf("使用最新价格 %.7f 计算: (%.8f - %.7f) / %.8f = %.4f = %.2f%%\n",
		lastPrice, avgCost, lastPrice, avgCost, lastProfitPercent, lastProfitPercent*100)

	// 如果用户把平均成本当作当前价格来计算
	fmt.Printf("\n=== 如果用户理解错误 ===")
	fmt.Printf("\n如果认为 0.16466000 是当前价格，0.16720000 是持仓成本:")
	wrongCalc := (currentPrice - avgCost) / avgCost // 认为是做多
	fmt.Printf("错误计算: (%.8f - %.8f) / %.8f = %.4f = %.2f%%\n",
		currentPrice, avgCost, avgCost, wrongCalc, wrongCalc*100)
}
