package main

import (
	"fmt"
	"math"
	"strconv"
)

func main() {
	// 模拟JASMYUSDT的参数
	symbol := "JASMYUSDT"
	price := 0.00902000
	currentQty := 5.0
	minNotional := 5.0

	// 当前名义价值
	currentNotional := currentQty * price
	fmt.Printf("%s 当前状态:\n", symbol)
	fmt.Printf("  数量: %.4f\n", currentQty)
	fmt.Printf("  价格: %.8f\n", price)
	fmt.Printf("  名义价值: %.4f USDT\n", currentNotional)
	fmt.Printf("  最小要求: %.1f USDT\n", minNotional)
	fmt.Printf("  需要调整: %v\n", currentNotional < minNotional)

	if currentNotional >= minNotional {
		fmt.Println("✅ 无需调整")
		return
	}

	// 使用特殊配置的stepSize=1.0进行调整
	stepSize := 1.0
	minQtyForNotional := minNotional / price
	fmt.Printf("\n调整计算:\n")
	fmt.Printf("  目标名义价值: %.1f USDT\n", minNotional)
	fmt.Printf("  所需最小数量: %.4f\n", minQtyForNotional)
	fmt.Printf("  使用stepSize: %.1f\n", stepSize)

	// 调整数量精度
	adjustedMinQty := math.Ceil(minQtyForNotional/stepSize) * stepSize
	fmt.Printf("  调整后数量: %.4f\n", adjustedMinQty)

	// 重新计算名义价值
	newNotional := adjustedMinQty * price
	fmt.Printf("  新名义价值: %.4f USDT\n", newNotional)
	fmt.Printf("  满足要求: %v\n", newNotional >= minNotional)

	// 格式化结果
	adjustedQuantity := strconv.FormatFloat(adjustedMinQty, 'f', -1, 64)
	fmt.Printf("\n最终结果:\n")
	fmt.Printf("  调整数量: %s -> %s\n", strconv.FormatFloat(currentQty, 'f', -1, 64), adjustedQuantity)
	fmt.Printf("  名义价值: %.4f -> %.4f USDT\n", currentNotional, newNotional)
}