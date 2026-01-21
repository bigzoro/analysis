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

	fmt.Printf("基础计算: minMarginTarget=%.1f, leverage=%d, baseTarget=%.1f\n",
		minMarginTarget, leverage, baseTarget)

	// 根据价格区间调整目标 - 更细粒度的分类
	var target float64
	if price < 0.0001 { // 极极低价币种（<0.01美分）
		target = math.Max(baseTarget, 35.0) // 适度提高目标
		fmt.Printf("价格区间: 极极低价 (<0.0001), target = max(%.1f, 35.0) = %.1f\n", baseTarget, target)
	} else if price < 0.001 { // 极低价币种（<0.1美分）
		target = math.Max(baseTarget, 30.0) // 提高到30 USDT目标
		fmt.Printf("价格区间: 极低价 (<0.001), target = max(%.1f, 30.0) = %.1f\n", baseTarget, target)
	} else if price < 0.01 { // 低价币种（<1美分）
		target = math.Max(baseTarget, 20.0) // 稍微提高目标
		fmt.Printf("价格区间: 低价 (<0.01), target = max(%.1f, 20.0) = %.1f\n", baseTarget, target)
	} else if price < 0.1 { // 中低价币种（<10美分）
		target = math.Max(baseTarget, 15.0) // 小幅提高目标
		fmt.Printf("价格区间: 中低价 (<0.1), target = max(%.1f, 15.0) = %.1f\n", baseTarget, target)
	} else if price > 100 { // 高价币种（>100 USDT）
		target = math.Max(baseTarget, 5.0) // 保持最低要求
		fmt.Printf("价格区间: 高价 (>100), target = max(%.1f, 5.0) = %.1f\n", baseTarget, target)
	} else {
		target = baseTarget // 中等价格币种使用杠杆计算的目标
		fmt.Printf("价格区间: 中等价, target = %.1f\n", target)
	}

	// 特殊币种调整
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	if strings.Contains(strings.ToLower(baseSymbol), "shib") || strings.Contains(strings.ToLower(baseSymbol), "doge") {
		target = math.Max(target, baseTarget) // meme币使用杠杆计算的目标
		fmt.Printf("特殊币种调整: meme币, target = max(%.1f, %.1f) = %.1f\n", target, baseTarget, target)
	}

	// 对于特定已知低价币种，进一步调整
	if strings.Contains(strings.ToLower(baseSymbol), "arc") {
		target = math.Max(target, baseTarget+5.0) // ARC特殊处理
		fmt.Printf("特殊币种调整: ARC, target = max(%.1f, %.1f) = %.1f\n", target, baseTarget+5.0, target)
	}

	margin := target / float64(leverage)
	fmt.Printf("最终结果: 名义价值=%.1f USDT, 杠杆=%dx, 保证金=%.2f USDT\n",
		target, leverage, margin)

	return target
}

func main() {
	fmt.Println("=== 分析BEATUSDT保证金为什么还是16点多 ===")
	fmt.Println()

	// 可能的BEATUSDT价格范围
	possiblePrices := []float64{0.00005, 0.0001, 0.0005, 0.001, 0.01}

	for _, price := range possiblePrices {
		fmt.Printf("\n🔍 如果BEATUSDT价格=%.8f:\n", price)
		calculateSmartTargetNotional(price, "BEATUSDT", 3)
	}

	fmt.Println()
	fmt.Println("💡 分析结果:")
	fmt.Println("1. 如果价格 < 0.0001: 保证金 = 35/3 ≈ 11.67u")
	fmt.Println("2. 如果价格 < 0.001: 保证金 = 30/3 = 10u")
	fmt.Println("3. 如果杠杆不是3倍，会得到不同的结果")

	fmt.Println()
	fmt.Println("🚨 可能的原因:")
	fmt.Println("1. BEATUSDT的实际价格不在预期区间")
	fmt.Println("2. 杠杆倍数不是3")
	fmt.Println("3. 代码修改还没有生效（需要重启服务）")
	fmt.Println("4. 缓存问题，使用了之前的计算结果")

	fmt.Println()
	fmt.Println("🔧 建议检查:")
	fmt.Println("1. 查看BEATUSDT的实际价格")
	fmt.Println("2. 检查订单的杠杆设置")
	fmt.Println("3. 重启相关服务使代码生效")
	fmt.Println("4. 查看最新的日志确认使用的是新逻辑")
}