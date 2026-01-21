package main

import (
	"fmt"
	"math"
)

func main() {
	price := 0.008996
	minNotional := 5.0
	stepSize := 1.0

	minQty := math.Ceil(minNotional/price/stepSize) * stepSize
	actualNotional := minQty * price

	fmt.Printf("JASMYUSDT 计算验证:\n")
	fmt.Printf("价格: %.6f USDT\n", price)
	fmt.Printf("最小名义价值: %.1f USDT\n", minNotional)
	fmt.Printf("stepSize: %.1f\n", stepSize)
	fmt.Printf("计算数量: %.0f\n", minQty)
	fmt.Printf("实际名义价值: %.3f USDT\n", actualNotional)
	fmt.Printf("是否满足最小要求: %v\n", actualNotional >= minNotional)
}