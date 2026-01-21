package main

import (
	"fmt"
	"math"
	"strconv"
)

func adjustQuantityWithConfig(quantity string, stepSize, minNotional, maxQty float64, symbol string, currentPrice float64) string {
	// 解析数量
	qty, err := strconv.ParseFloat(quantity, 64)
	if err != nil {
		fmt.Printf("[scheduler] 解析数量失败 %s: %v", quantity, err)
		return quantity
	}

	// 计算名义价值
	notionalValue := qty * currentPrice
	fmt.Printf("原始数量: %s, 名义价值: %.4f\n", quantity, notionalValue)

	// 检查最小名义价值
	if notionalValue < minNotional {
		// 计算满足最小名义价值的最小数量
		minQty := math.Ceil(minNotional/currentPrice/stepSize) * stepSize
		qty = minQty
		fmt.Printf("%s 数量 %.8f 名义价值 %.2f 低于最小要求 %.2f，调整为 %.8f\n",
			symbol, qty, notionalValue, minNotional, minQty)
	}

	// 检查最大数量
	if qty > maxQty {
		qty = maxQty
		fmt.Printf("%s 数量 %.8f 超过最大限制 %.8f，调整为最大值", symbol, qty, maxQty)
	}

	// 根据stepSize调整精度
	adjustedQty := math.Floor(qty/stepSize) * stepSize

	// 再次检查最小名义价值（调整精度后）
	finalNotional := adjustedQty * currentPrice
	if finalNotional < minNotional && adjustedQty < maxQty {
		// 如果调整后仍然低于最小名义价值，增加到下一个step
		adjustedQty += stepSize
		fmt.Printf("%s 调整精度后名义价值 %.2f 仍低于最小要求，增加到 %.8f\n",
			symbol, finalNotional, adjustedQty)
	}

	result := strconv.FormatFloat(adjustedQty, 'f', -1, 64)
	fmt.Printf("特殊配置数量调整 %s: %s -> %s (stepSize: %s, minNotional: %.2f)\n",
		symbol, quantity, result, strconv.FormatFloat(stepSize, 'f', -1, 64), minNotional)
	return result
}

func main() {
	// 模拟RIVERUSDT的参数
	symbol := "RIVERUSDT"
	quantity := "0.010"
	currentPrice := 18.309
	stepSize := 1.0
	minNotional := 5.0
	maxQty := 1000.0

	result := adjustQuantityWithConfig(quantity, stepSize, minNotional, maxQty, symbol, currentPrice)
	fmt.Printf("最终结果: %s\n", result)
}