package main

import (
	"fmt"
	"math"
	"strings"
)

func calculateSmartTargetNotional(price float64, symbol string, leverage int) float64 {
	// 根据杠杆倍数设置最小保证金目标
	minMarginTarget := 10.0 // 目标保证金至少10 USDT

	// 计算需要的名义价值：保证金 × 杠杆
	baseTarget := minMarginTarget * float64(leverage)

	// 确保名义价值不低于币安最低要求
	baseTarget = math.Max(baseTarget, 5.0)

	// 根据价格区间调整目标 - 更细粒度的分类
	var target float64
	if price < 0.0001 { // 极极低价币种（<0.01美分）
		target = math.Max(baseTarget, 50.0) // 大幅提高目标
	} else if price < 0.001 { // 极低价币种（<0.1美分）
		target = math.Max(baseTarget, 30.0) // 提高到30 USDT目标
	} else if price < 0.01 { // 低价币种（<1美分）
		target = math.Max(baseTarget, 20.0) // 稍微提高目标
	} else if price < 0.1 { // 中低价币种（<10美分）
		target = math.Max(baseTarget, 15.0) // 小幅提高目标
	} else if price > 100 { // 高价币种（>100 USDT）
		target = math.Max(baseTarget, 5.0) // 保持最低要求
	} else {
		target = baseTarget // 中等价格币种使用杠杆计算的目标
	}

	// 特殊币种调整
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	if strings.Contains(strings.ToLower(baseSymbol), "shib") || strings.Contains(strings.ToLower(baseSymbol), "doge") {
		target = math.Max(target, baseTarget) // meme币使用杠杆计算的目标
	}

	// 对于特定已知低价币种，进一步调整
	if strings.Contains(strings.ToLower(baseSymbol), "arc") {
		target = math.Max(target, baseTarget+5.0) // ARC特殊处理
	}

	margin := target / float64(leverage)
	fmt.Printf("%s: 价格=%.8f, 杠杆=%dx, 名义价值目标=%.1f USDT, 保证金=%.2f USDT\n",
		symbol, price, leverage, target, margin)

	return target
}

func main() {
	fmt.Println("=== 调试为什么保证金都是17u左右 ===")
	fmt.Println()

	fmt.Println("📊 测试不同价格区间的保证金计算:")
	fmt.Println("假设杠杆都是3倍")
	fmt.Println()

	testCases := []struct {
		symbol string
		price  float64
		desc   string
	}{
		{"DASHUSDT", 0.00005, "极极低价 (<0.0001)"},
		{"GUNUSDT", 0.0005, "极低价 (<0.001)"},
		{"BTCUSDT", 50000, "高价 (>100)"},
		{"ETHUSDT", 3000, "中等价"},
	}

	for _, tc := range testCases {
		fmt.Printf("🔍 %s (%s):\n", tc.symbol, tc.desc)
		calculateSmartTargetNotional(tc.price, tc.symbol, 3)
		fmt.Println()
	}

	fmt.Println("💡 分析结果:")
	fmt.Println("1. 极极低价币种 (<0.0001): 名义价值目标 = max(30, 50) = 50, 保证金 = 50/3 ≈ 16.67u")
	fmt.Println("2. 极低价币种 (<0.001): 名义价值目标 = max(30, 30) = 30, 保证金 = 30/3 = 10u")
	fmt.Println("3. 高价币种 (>100): 名义价值目标 = max(30, 5) = 30, 保证金 = 30/3 = 10u")
	fmt.Println("4. 中等价币种: 名义价值目标 = 30, 保证金 = 30/3 = 10u")
	fmt.Println()
	fmt.Println("🚨 如果保证金都是17u左右，说明当前交易的币种价格都 < 0.0001 (极极低价区间)")
	fmt.Println("🚨 或者杠杆倍数不是3，而是其他值导致 50/x ≈ 17")
	fmt.Println()
	fmt.Println("🔧 可能的解决方案:")
	fmt.Println("1. 检查当前交易币种的实际价格")
	fmt.Println("2. 检查杠杆倍数设置")
	fmt.Println("3. 调整名义价值目标参数")
}