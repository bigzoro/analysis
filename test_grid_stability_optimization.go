package main

import (
	"fmt"
	"math"
)

// 测试新的价格稳定性评分算法
func main() {
	fmt.Println("🧪 测试网格交易价格稳定性评分优化")
	fmt.Println("=" * 60)

	// 测试不同场景的价格稳定性评分
	testCases := []struct {
		name        string
		description string
		prices      []float64
		expected    string
	}{
		{
			name:        "稳定币",
			description: "极度稳定的币种（如USDT-like）",
			prices:      []float64{1.0001, 1.0000, 1.0002, 0.9999, 1.0001, 1.0000, 1.0001},
			expected:    "评分较低（太稳定不适合网格）",
		},
		{
			name:        "适中波动",
			description: "中等波动的加密货币",
			prices:      []float64{100.0, 105.0, 102.0, 108.0, 103.0, 107.0, 105.5},
			expected:    "评分较高（适合网格交易）",
		},
		{
			name:        "高波动",
			description: "波动较大的币种",
			prices:      []float64{50.0, 75.0, 40.0, 90.0, 30.0, 110.0, 45.0},
			expected:    "评分中等（波动过大但仍可考虑）",
		},
		{
			name:        "单边上涨",
			description: "有明确上涨趋势的币种",
			prices:      []float64{100.0, 110.0, 115.0, 120.0, 125.0, 130.0, 135.0},
			expected:    "评分中等（趋势明显但波动适中）",
		},
		{
			name:        "震荡行情",
			description: "典型的震荡行情",
			prices:      []float64{100.0, 95.0, 105.0, 98.0, 102.0, 97.0, 103.0},
			expected:    "评分较高（理想的网格交易环境）",
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n📊 测试案例: %s\n", tc.name)
		fmt.Printf("   描述: %s\n", tc.description)
		fmt.Printf("   价格序列: %v\n", tc.prices)

		score := calculateTestStabilityScore(tc.prices)
		fmt.Printf("   稳定性评分: %.3f\n", score)
		fmt.Printf("   预期表现: %s\n", tc.expected)

		// 给出评分解读
		if score >= 0.8 {
			fmt.Printf("   ✅ 优秀: 非常适合网格交易\n")
		} else if score >= 0.6 {
			fmt.Printf("   ✅ 良好: 适合网格交易\n")
		} else if score >= 0.4 {
			fmt.Printf("   ⚠️ 一般: 勉强适合，可考虑\n")
		} else if score >= 0.2 {
			fmt.Printf("   ❌ 不佳: 不太适合网格交易\n")
		} else {
			fmt.Printf("   ❌ 很差: 不适合网格交易\n")
		}
	}

	fmt.Println("\n" + "="*60)
	fmt.Println("🎯 优化总结:")
	fmt.Println("   • 放宽了变异系数标准，适应加密货币特性")
	fmt.Println("   • 增加了波动一致性评估")
	fmt.Println("   • 加入了趋势稳定性分析")
	fmt.Println("   • 降低了筛选阈值，提高入选率")
	fmt.Println("   • 预计能显著增加符合条件的币种数量")
}

// 模拟新的价格稳定性评分算法
func calculateTestStabilityScore(prices []float64) float64 {
	if len(prices) < 5 {
		return 0.0
	}

	// 计算基础统计
	mean := 0.0
	for _, price := range prices {
		mean += price
	}
	mean /= float64(len(prices))

	variance := 0.0
	for _, price := range prices {
		variance += math.Pow(price-mean, 2)
	}
	variance /= float64(len(prices))
	stdDev := math.Sqrt(variance)

	cv := stdDev / mean
	if mean == 0 {
		return 0.0
	}

	// 计算价格变化
	priceChanges := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		priceChanges[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	changeMean := 0.0
	for _, change := range priceChanges {
		changeMean += change
	}
	changeMean /= float64(len(priceChanges))

	changeVariance := 0.0
	for _, change := range priceChanges {
		changeVariance += math.Pow(change-changeMean, 2)
	}
	changeVariance /= float64(len(priceChanges))
	changeStdDev := math.Sqrt(changeVariance)

	// 变异系数评分
	var cvScore float64
	if cv >= 0.02 && cv <= 0.25 {
		cvScore = 1.0
	} else if cv >= 0.01 && cv <= 0.40 {
		if cv < 0.02 {
			cvScore = 0.6 + (cv-0.01)/(0.02-0.01)*0.4
		} else {
			cvScore = 1.0 - (cv-0.25)/(0.40-0.25)*0.4
		}
	} else {
		if cv < 0.01 {
			cvScore = math.Max(0.2, cv/0.01*0.6)
		} else {
			cvScore = math.Max(0.1, 1.0-cv/0.40)
		}
	}

	// 波动一致性评分
	var consistencyScore float64
	if changeStdDev <= 0.05 {
		consistencyScore = 1.0
	} else if changeStdDev <= 0.10 {
		consistencyScore = 0.8 + (0.10-changeStdDev)/(0.10-0.05)*0.2
	} else if changeStdDev <= 0.15 {
		consistencyScore = 0.6 + (0.15-changeStdDev)/(0.15-0.10)*0.2
	} else {
		consistencyScore = math.Max(0.2, 1.0-changeStdDev/0.15)
	}

	// 趋势稳定性评分
	trendStrength := math.Abs(changeMean * 100)
	var trendScore float64
	if trendStrength <= 1.0 {
		trendScore = 1.0
	} else if trendStrength <= 2.0 {
		trendScore = 0.8 + (2.0-trendStrength)/(2.0-1.0)*0.2
	} else if trendStrength <= 5.0 {
		trendScore = 0.5 + (5.0-trendStrength)/(5.0-2.0)*0.3
	} else {
		trendScore = math.Max(0.1, 1.0-trendStrength/5.0)
	}

	// 综合评分
	overallScore := cvScore*0.4 + consistencyScore*0.3 + trendScore*0.3
	return overallScore
}
