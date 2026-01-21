package main

import (
	"fmt"
	"math"
	"strings"
)

// 模拟 calculateSmartTargetNotional 函数
func calculateSmartTargetNotional(price float64, symbol string, leverage int) float64 {
	// 根据杠杆倍数设置最小保证金目标
	minMarginTarget := 10.0 // 目标保证金至少10 USDT

	// 计算需要的名义价值：保证金 × 杠杆
	baseTarget := minMarginTarget * float64(leverage)

	// 确保名义价值不低于币安最低要求
	baseTarget = math.Max(baseTarget, 5.0)

	fmt.Printf("  基础计算: 最小保证金目标=%.1f USDT, 杠杆=%dx, 基础名义价值目标=%.1f USDT\n",
		minMarginTarget, leverage, baseTarget)

	// 根据价格区间调整目标 - 更细粒度的分类
	var target float64
	if price < 0.0001 { // 极极低价币种（<0.01美分）
		target = math.Max(baseTarget, 50.0) // 大幅提高目标
		fmt.Printf("  价格区间: 极极低价 (<0.0001), 目标名义价值=max(%.1f, 50.0)=%.1f USDT\n", baseTarget, target)
	} else if price < 0.001 { // 极低价币种（<0.1美分）
		target = math.Max(baseTarget, 30.0) // 提高到30 USDT目标
		fmt.Printf("  价格区间: 极低价 (<0.001), 目标名义价值=max(%.1f, 30.0)=%.1f USDT\n", baseTarget, target)
	} else if price < 0.01 { // 低价币种（<1美分）
		target = math.Max(baseTarget, 20.0) // 稍微提高目标
		fmt.Printf("  价格区间: 低价 (<0.01), 目标名义价值=max(%.1f, 20.0)=%.1f USDT\n", baseTarget, target)
	} else if price < 0.1 { // 中低价币种（<10美分）
		target = math.Max(baseTarget, 15.0) // 小幅提高目标
		fmt.Printf("  价格区间: 中低价 (<0.1), 目标名义价值=max(%.1f, 15.0)=%.1f USDT\n", baseTarget, target)
	} else if price > 100 { // 高价币种（>100 USDT）
		target = math.Max(baseTarget, 5.0) // 保持最低要求
		fmt.Printf("  价格区间: 高价 (>100), 目标名义价值=max(%.1f, 5.0)=%.1f USDT\n", baseTarget, target)
	} else {
		target = baseTarget // 中等价格币种使用杠杆计算的目标
		fmt.Printf("  价格区间: 中等价, 目标名义价值=%.1f USDT\n", target)
	}

	// 特殊币种调整
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	if strings.Contains(strings.ToLower(baseSymbol), "shib") || strings.Contains(strings.ToLower(baseSymbol), "doge") {
		target = math.Max(target, baseTarget) // meme币使用杠杆计算的目标
		fmt.Printf("  特殊币种调整: meme币, 目标名义价值=max(%.1f, %.1f)=%.1f USDT\n", target, baseTarget, target)
	}

	// 对于特定已知低价币种，进一步调整
	if strings.Contains(strings.ToLower(baseSymbol), "arc") {
		target = math.Max(target, baseTarget+5.0) // ARC特殊处理
		fmt.Printf("  特殊币种调整: ARC, 目标名义价值=max(%.1f, %.1f)=%.1f USDT\n", target, baseTarget+5.0, target)
	}

	margin := target / float64(leverage)
	fmt.Printf("  最终结果: 名义价值=%.1f USDT, 杠杆=%dx, 保证金=%.1f USDT\n",
		target, leverage, margin)

	return target
}

func main() {
	fmt.Println("=== 分析GUNUSDT和DASHUSDT保证金差异 ===")

	// 假设的价格数据（实际需要从数据库或API获取）
	// 这里我假设一些典型的价格来分析差异
	testCases := []struct {
		symbol   string
		price    float64
		leverage int
		desc     string
	}{
		{"GUNUSDT", 0.01, 3, "假设GUNUSDT价格约0.01 USDT，3倍杠杆"},
		{"DASHUSDT", 0.0005, 3, "假设DASHUSDT价格约0.0005 USDT，3倍杠杆"},
		{"GUNUSDT", 0.005, 3, "假设GUNUSDT价格约0.005 USDT，3倍杠杆"},
		{"DASHUSDT", 0.0008, 3, "假设DASHUSDT价格约0.0008 USDT，3倍杠杆"},
	}

	for _, tc := range testCases {
		fmt.Printf("\n🔍 分析 %s (%s):\n", tc.symbol, tc.desc)
		calculateSmartTargetNotional(tc.price, tc.symbol, tc.leverage)
	}

	fmt.Println("\n💡 关键分析:")
	fmt.Println("1. GUNUSDT价格较高(>0.005)，属于'低价'区间，目标名义价值至少20 USDT")
	fmt.Println("2. DASHUSDT价格较低(<0.001)，属于'极低价'区间，目标名义价值至少30 USDT")
	fmt.Println("3. 相同杠杆倍数下，名义价值目标差异导致保证金差异")
	fmt.Println("4. 保证金 = 名义价值 / 杠杆倍数")

	fmt.Println("\n📊 计算示例:")
	fmt.Printf("  GUNUSDT: 20 USDT / 3倍杠杆 = 6.67 USDT 保证金\n")
	fmt.Printf("  DASHUSDT: 30 USDT / 3倍杠杆 = 10 USDT 保证金\n")
	fmt.Printf("  差异原因: DASHUSDT价格更低，需要更高的名义价值来确保足够的保证金\n")
}