package main

import (
	"fmt"
)

// 模拟calculateRealizedPnL函数
func calculateRealizedPnL(orderSide string, entryPrice float64) float64 {
	// 模拟平仓订单数据（订单173）
	closePrice := 4.12835
	closeQty := 2.0

	var closePnL float64
	if orderSide == "BUY" {
		// 多头开仓，对应的平仓是SELL
		closePnL = (closePrice - entryPrice) * closeQty
	} else {
		// 空头开仓，对应的平仓是BUY
		closePnL = (entryPrice - closePrice) * closeQty
	}

	return closePnL
}

func main() {
	// 模拟订单172的数据
	orderSide := "SELL"  // 空头开仓
	entryPrice := 4.1844 // 开仓价格
	closeOrderIds := "173" // 关联的平仓订单

	fmt.Printf("开仓订单信息:\n")
	fmt.Printf("  方向: %s\n", orderSide)
	fmt.Printf("  开仓价格: %.6f\n", entryPrice)
	fmt.Printf("  关联平仓订单: %s\n", closeOrderIds)

	// 计算已实现利润
	realizedPnL := calculateRealizedPnL(orderSide, entryPrice)
	fmt.Printf("  已实现利润: %.6f USDT\n", realizedPnL)

	// 模拟当前价格（假设价格没变）
	currentPrice := 4.12835
	quantity := 2.0

	// 判断持仓状态（模拟已平仓）
	actualPositionStatus := "closed" // 已平仓

	// 计算未实现利润
	var unrealizedPnL float64
	if actualPositionStatus == "closed" {
		unrealizedPnL = 0 // 已平仓，未实现利润为0
	} else {
		if orderSide == "BUY" {
			unrealizedPnL = quantity * (currentPrice - entryPrice)
		} else {
			unrealizedPnL = quantity * (entryPrice - currentPrice)
		}
	}

	fmt.Printf("  持仓状态: %s\n", actualPositionStatus)
	fmt.Printf("  未实现利润: %.6f USDT\n", unrealizedPnL)
	fmt.Printf("  总利润: %.6f USDT\n", realizedPnL + unrealizedPnL)

	// 验证计算结果
	expectedPnL := (4.1844 - 4.12835) * 2
	fmt.Printf("  预期利润: %.6f USDT\n", expectedPnL)
	fmt.Printf("  详细比较:\n")
	fmt.Printf("    realizedPnL ≈ expectedPnL: %t (|%.6f - %.6f| < 0.000001)\n", abs(realizedPnL-expectedPnL) < 0.000001, realizedPnL, expectedPnL)
	fmt.Printf("    unrealizedPnL == 0: %t (%.6f == 0)\n", unrealizedPnL == 0, unrealizedPnL)
	fmt.Printf("  计算正确: %t\n", (abs(realizedPnL-expectedPnL) < 0.000001) && (unrealizedPnL == 0))
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
