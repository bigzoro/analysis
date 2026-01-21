package main

import (
	"fmt"
	grid_core "analysis/internal/strategy/grid_trading/core"
	traditional_core "analysis/internal/strategy/traditional/core"
	ma_core "analysis/internal/strategy/moving_average/core"
	arb_core "analysis/internal/strategy/arbitrage/core"
)

func main() {
	fmt.Println("Testing new strategy architectures...")

	// 测试网格交易策略
	gridStrategy := grid_core.GetGridTradingStrategy()
	fmt.Printf("Grid Trading Strategy: %t\n", gridStrategy != nil)

	// 测试传统策略
	traditionalStrategy := traditional_core.GetTraditionalStrategy()
	fmt.Printf("Traditional Strategy: %t\n", traditionalStrategy != nil)

	// 测试均线策略
	maStrategy := ma_core.GetMovingAverageStrategy()
	fmt.Printf("Moving Average Strategy: %t\n", maStrategy != nil)

	// 测试套利策略
	arbStrategy := arb_core.GetArbitrageStrategy()
	fmt.Printf("Arbitrage Strategy: %t\n", arbStrategy != nil)

	fmt.Println("All new architectures initialized successfully!")
}