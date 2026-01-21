package main

import (
	"fmt"
)

// 模拟平仓订单利润计算
func calculateCloseOrderPnL(orderSide string, entryPrice float64, parentEntryPrice float64, quantity float64) float64 {
	var pnl float64

	if orderSide == "BUY" {
		// 买入平仓（空头平仓）：利润 = 开仓价格 - 平仓价格
		pnl = (parentEntryPrice - entryPrice) * quantity
	} else {
		// 卖出平仓（多头平仓）：利润 = 平仓价格 - 开仓价格
		pnl = (entryPrice - parentEntryPrice) * quantity
	}

	return pnl
}

func main() {
	// 模拟订单173（平仓订单）
	orderSide := "BUY"       // 买入平仓（空头平仓）
	entryPrice := 4.12835    // 平仓价格
	parentEntryPrice := 4.1844 // 开仓价格
	quantity := 2.0          // 数量

	fmt.Printf("平仓订单利润计算:\n")
	fmt.Printf("  方向: %s\n", orderSide)
	fmt.Printf("  平仓价格: %.6f\n", entryPrice)
	fmt.Printf("  开仓价格: %.6f\n", parentEntryPrice)
	fmt.Printf("  数量: %.1f\n", quantity)

	pnl := calculateCloseOrderPnL(orderSide, entryPrice, parentEntryPrice, quantity)
	fmt.Printf("  计算公式: 开仓价格 - 平仓价格 = %.6f - %.6f = %.6f\n", parentEntryPrice, entryPrice, parentEntryPrice-entryPrice)
	fmt.Printf("  利润: %.6f * %.1f = %.6f USDT\n", parentEntryPrice-entryPrice, quantity, pnl)

	// 验证与开仓订单利润一致
	expectedPnl := (parentEntryPrice - entryPrice) * quantity
	fmt.Printf("  预期利润: %.6f USDT\n", expectedPnl)
	fmt.Printf("  计算正确: %t\n", pnl == expectedPnl)
}
