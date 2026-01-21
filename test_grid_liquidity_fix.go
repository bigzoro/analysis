package main

import (
	"fmt"
	"math"
)

// 测试流动性评分修复
func main() {
	fmt.Println("🧪 测试网格交易流动性评分修复")
	fmt.Println("=" * 50)

	// 模拟测试数据
	testCases := []struct {
		symbol    string
		volume    float64 // 24h交易量（代币数量）
		price     float64 // 当前价格
		volumeUSD float64 // 预期交易额
		expected  float64 // 预期评分
	}{
		{"BTCUSDT", 1000.0, 45000.0, 45000000.0, 1.0},       // 4500万美元，极好流动性
		{"ETHUSDT", 5000.0, 2500.0, 12500000.0, 1.0},        // 1250万美元，极好流动性
		{"ADAUSDT", 1000000.0, 0.35, 350000.0, 0.7},         // 35万美元，中等流动性
		{"SHIBUSDT", 1000000000.0, 0.000015, 15000.0, 0.15}, // 1.5万美元，低流动性
		{"UNKNOWN", 100.0, 1.0, 100.0, 0.1},                 // 100美元，很低流动性
	}

	for _, tc := range testCases {
		// 计算实际交易额
		actualVolumeUSD := tc.volume * tc.price

		// 计算流动性评分（使用修复后的逻辑）
		var score float64
		if actualVolumeUSD >= 10000000 { // 1000万美元以上，极好流动性
			score = 1.0
		} else if actualVolumeUSD >= 5000000 { // 500万美元以上，优秀流动性
			score = 0.9 + (actualVolumeUSD-5000000)/(10000000-5000000)*0.1
		} else if actualVolumeUSD >= 1000000 { // 100万美元以上，良好流动性
			score = 0.7 + (actualVolumeUSD-1000000)/(5000000-1000000)*0.2
		} else if actualVolumeUSD >= 500000 { // 50万美元以上，中等流动性
			score = 0.5 + (actualVolumeUSD-500000)/(1000000-500000)*0.2
		} else if actualVolumeUSD >= 100000 { // 10万美元以上，基本流动性
			score = 0.3 + (actualVolumeUSD-100000)/(500000-100000)*0.2
		} else {
			score = math.Max(0.1, actualVolumeUSD/100000*0.3) // 流动性不足
		}

		fmt.Printf("📊 %s:\n", tc.symbol)
		fmt.Printf("   交易量: %.0f 个代币\n", tc.volume)
		fmt.Printf("   价格: $%.4f\n", tc.price)
		fmt.Printf("   交易额: $%.0f\n", actualVolumeUSD)
		fmt.Printf("   流动性评分: %.2f\n", score)
		fmt.Printf("   是否通过阈值(0.2): %s\n", map[bool]string{true: "✅ 通过", false: "❌ 不通过"}[score >= 0.2])
		fmt.Println()
	}

	fmt.Println("🎯 修复总结:")
	fmt.Println("   • 之前的算法错误地假设所有币种流通量都是100万")
	fmt.Println("   • 新算法基于实际交易额进行评分")
	fmt.Println("   • 降低了流动性阈值从0.4到0.2，适合网格交易特性")
	fmt.Println("   • 添加了详细的日志输出，便于调试")

	fmt.Println("\n💡 预期效果:")
	fmt.Println("   • BTC和ETH等主流币种将获得高流动性评分")
	fmt.Println("   • 中等市值币种将获得中等流动性评分")
	fmt.Println("   • 网格交易扫描器将能找到更多适合的币种")
}
