package main

import (
	"fmt"
	"math"
	"strconv"
)

func validateAndAdjustNotional(symbol string, qty float64, notionalPrice float64, currentQuantity string) (adjustedQuantity string, skipOrder bool, reason string) {
	finalNotional := qty * notionalPrice

	// 如果名义价值已经满足要求，直接返回
	if finalNotional >= 5.0 {
		return currentQuantity, false, ""
	}

	fmt.Printf("%s 名义价值不足 (%.4f < 5.0)，尝试调整\n", symbol, finalNotional)

	// 使用特殊配置的stepSize
	stepSize := 1.0
	minQtyForNotional := 5.0 / notionalPrice

	fmt.Printf("  目标名义价值: 5.0 USDT\n")
	fmt.Printf("  所需最小数量: %.4f\n", minQtyForNotional)
	fmt.Printf("  使用stepSize: %.1f\n", stepSize)

	// 调整数量精度
	adjustedMinQty := math.Ceil(minQtyForNotional/stepSize) * stepSize
	fmt.Printf("  调整后数量: %.4f\n", adjustedMinQty)

	// 重新计算名义价值
	newNotional := adjustedMinQty * notionalPrice
	fmt.Printf("  新名义价值: %.4f USDT\n", newNotional)

	if newNotional >= 5.0 {
		// 调整成功
		adjustedQuantity = strconv.FormatFloat(adjustedMinQty, 'f', -1, 64)
		fmt.Printf("  ✅ 调整成功: %s -> %s\n", currentQuantity, adjustedQuantity)
		return adjustedQuantity, false, ""
	} else {
		// 调整后仍不满足要求
		reason = fmt.Sprintf("即使调整后名义价值仍不足: %.4f USDT", newNotional)
		return "", true, reason
	}
}

func main() {
	fmt.Println("🧪 测试止盈单名义价值调整逻辑")
	fmt.Println("=====================================")

	// JASMYUSDT的止盈单参数
	symbol := "JASMYUSDT"
	tpPrice := 0.00861800  // 止盈价格
	currentQty := 552.0    // 当前数量
	currentQuantity := "552"

	fmt.Printf("📊 测试参数:\n")
	fmt.Printf("  交易对: %s\n", symbol)
	fmt.Printf("  止盈价格: %.8f USDT\n", tpPrice)
	fmt.Printf("  当前数量: %.1f\n", currentQty)
	fmt.Printf("  当前名义价值: %.4f USDT\n", currentQty*tpPrice)
	fmt.Printf("  是否满足5 USDT要求: %v\n", currentQty*tpPrice >= 5.0)

	fmt.Println("\n🔧 调整逻辑:")
	adjustedQuantity, skipOrder, reason := validateAndAdjustNotional(symbol, currentQty, tpPrice, currentQuantity)

	fmt.Printf("\n📋 调整结果:\n")
	if skipOrder {
		fmt.Printf("  ❌ 跳过下单: %s\n", reason)
	} else {
		fmt.Printf("  ✅ 调整成功: %s -> %s\n", currentQuantity, adjustedQuantity)

		if newQty, err := strconv.ParseFloat(adjustedQuantity, 64); err == nil {
			newNotional := newQty * tpPrice
			fmt.Printf("  📈 新名义价值: %.4f USDT\n", newNotional)
			fmt.Printf("  🎯 满足要求: %v\n", newNotional >= 5.0)
		}
	}
}