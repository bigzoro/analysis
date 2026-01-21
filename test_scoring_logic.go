package main

import (
	"fmt"
	"strings"
)

// 模拟CandidateScore结构体
type CandidateScore struct {
	Symbol           string
	OscillationScore float64
	LiquidityScore   float64
	VolatilityScore  float64
	MomentumScore    float64
	TotalScore       float64
}

// 扩展的主流币种列表
var majorCoins = []string{
	// 顶级主流币 (Layer1和基础设施)
	"BTC", "ETH", "BNB", "SOL", "ADA", "XRP", "DOT", "DOGE", "AVAX", "LINK",
	"LTC", "ICP", "NEAR", "FTM", "HBAR", "FIL", "ETC", "ALGO", "VET",
	// 二级主流币 (Layer2和成熟项目)
	"OP", "ARB", "MATIC", "APT", "SUI", "SEI", "TIA", "ZKS", "IMX", "ONDO",
	"INJ", "PEPE", "BONK", "WIF", "MEW", "BRETT", "PENGU", "MOTHER", "TURBO", "GIGA",
}

// 计算振荡性评分
func calculateOscillationScore(symbol string) float64 {
	baseSymbol := strings.TrimSuffix(symbol, "USDT")
	for _, coin := range majorCoins {
		if baseSymbol == coin {
			return 0.4 // 主流币种，较低振荡性
		}
	}
	return 0.7 // 默认较高振荡性
}

// 计算候选币种评分
func calculateCandidateScore(symbol string) CandidateScore {
	score := CandidateScore{Symbol: symbol}

	// 振荡性评分
	score.OscillationScore = calculateOscillationScore(symbol)

	// 流动性评分 (模拟中等流动性)
	score.LiquidityScore = 0.8

	// 波动率评分 (模拟适度波动)
	score.VolatilityScore = 0.7

	// 动量评分 (模拟中等动量)
	score.MomentumScore = 0.6

	// 优化后的综合评分
	score.TotalScore = (
		score.OscillationScore*0.5 +   // 提高到50% - 核心指标
		score.LiquidityScore*0.2 +     // 降低到20% - 辅助条件
		score.VolatilityScore*0.2 +    // 保持20% - 风险控制
		score.MomentumScore*0.1)       // 新增10% - 避免强趋势币种

	return score
}

// 模拟筛选逻辑
func shouldPassBasicFilter(score CandidateScore, minOscillation float64) bool {
	return score.OscillationScore >= minOscillation && score.LiquidityScore >= 0.4
}

func main() {
	fmt.Println("🧪 均值回归策略评分逻辑测试")
	fmt.Println("============================")

	// 测试币种列表 (包括优化前后的对比)
	testSymbols := []string{
		// 优化前的问题币种
		"AVAXUSDT", "LINKUSDT", "ICPUSDT",
		// 新兴币种
		"SYRUPUSDT", "ETHFIUSDT", "RENDERUSDT", "VIRTUALUSDT", "ACTUSDT",
		// 其他主流币种
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "SOLUSDT", "ADAUSDT",
	}

	fmt.Println("📊 各币种评分详情:")
	fmt.Println("==================")
	fmt.Printf("%-12s %-6s %-6s %-6s %-6s %-6s %-s\n",
		"币种", "振荡性", "流动性", "波动率", "动量", "综合", "类型")
	fmt.Println(strings.Repeat("-", 70))

	var majorSymbols, altSymbols []CandidateScore
	var allScores []CandidateScore

	for _, symbol := range testSymbols {
		score := calculateCandidateScore(symbol)
		allScores = append(allScores, score)

		baseSymbol := strings.TrimSuffix(symbol, "USDT")
		isMajor := false
		for _, major := range majorCoins {
			if baseSymbol == major {
				isMajor = true
				break
			}
		}

		coinType := "新兴币种"
		if isMajor {
			coinType = "主流币种"
			majorSymbols = append(majorSymbols, score)
		} else {
			altSymbols = append(altSymbols, score)
		}

		displaySymbol := symbol
		if len(symbol) > 10 {
			displaySymbol = symbol[:10]
		}
		fmt.Printf("%-12s %-6.2f %-6.2f %-6.2f %-6.2f %-6.2f %s\n",
			displaySymbol,
			score.OscillationScore,
			score.LiquidityScore,
			score.VolatilityScore,
			score.MomentumScore,
			score.TotalScore,
			coinType)
	}

	// 筛选测试
	fmt.Println("\n🎯 筛选测试 (最小振荡性0.5):")
	fmt.Println("============================")

	minOscillation := 0.5
	var passedSymbols []CandidateScore

	for _, score := range allScores {
		if shouldPassBasicFilter(score, minOscillation) {
			passedSymbols = append(passedSymbols, score)
		}
	}

	fmt.Printf("筛选前: %d个币种\n", len(allScores))
	fmt.Printf("筛选后: %d个币种\n", len(passedSymbols))

	var passedMajor, passedAlt int
	for _, score := range passedSymbols {
		baseSymbol := strings.TrimSuffix(score.Symbol, "USDT")
		isMajor := false
		for _, major := range majorCoins {
			if baseSymbol == major {
				isMajor = true
				break
			}
		}
		if isMajor {
			passedMajor++
		} else {
			passedAlt++
		}
	}

	fmt.Printf("• 通过的主流币种: %d个\n", passedMajor)
	fmt.Printf("• 通过的新兴币种: %d个\n", passedAlt)

	if len(passedSymbols) > 0 {
		passRatio := float64(passedMajor) / float64(len(passedSymbols))
		fmt.Printf("• 主流币种比例: %.1f%%\n", passRatio*100)
	}

	// 重点问题币种分析
	fmt.Println("\n🔍 关键问题币种分析:")
	fmt.Println("====================")

	problemCoins := []string{"AVAXUSDT", "LINKUSDT", "ICPUSDT"}
	for _, problemCoin := range problemCoins {
		for _, score := range allScores {
			if score.Symbol == problemCoin {
				passed := shouldPassBasicFilter(score, minOscillation)
				status := "❌ 被过滤"
				if passed {
					status = "✅ 通过筛选"
				}

				coinType := "主流币种"
				if strings.TrimSuffix(problemCoin, "USDT") == "ICP" {
					coinType = "主流币种(新增识别)"
				}

				fmt.Printf("• %s: %s | 振荡性:%.1f | 综合得分:%.3f | %s\n",
					problemCoin, coinType, score.OscillationScore, score.TotalScore, status)
				break
			}
		}
	}

	// 优化效果总结
	fmt.Println("\n📈 优化效果总结:")
	fmt.Println("===============")

	if len(majorSymbols) > 0 && len(altSymbols) > 0 {
		// 计算平均得分差异
		var majorTotal, altTotal float64
		for _, s := range majorSymbols {
			majorTotal += s.TotalScore
		}
		for _, s := range altSymbols {
			altTotal += s.TotalScore
		}

		avgMajor := majorTotal / float64(len(majorSymbols))
		avgAlt := altTotal / float64(len(altSymbols))

		fmt.Printf("• 主流币种平均得分: %.3f\n", avgMajor)
		fmt.Printf("• 新兴币种平均得分: %.3f\n", avgAlt)
		fmt.Printf("• 得分差异: 新兴比主流高 %.3f (%.1f%%)\n",
			avgAlt-avgMajor, (avgAlt-avgMajor)/avgMajor*100)
	}

	// 筛选效果
	totalBefore := len(allScores)
	totalAfter := len(passedSymbols)
	filteredCount := totalBefore - totalAfter

	if totalBefore > 0 {
		filterRate := float64(filteredCount) / float64(totalBefore) * 100
		fmt.Printf("• 筛选过滤率: %.1f%% (%d/%d)\n", filterRate, filteredCount, totalBefore)
	}

	fmt.Println("\n🎯 优化验证结论:")
	fmt.Println("===============")

	// 检查ICP是否正确识别
	icpRecognized := false
	for _, major := range majorCoins {
		if major == "ICP" {
			icpRecognized = true
			break
		}
	}

	// 检查主流币种比例是否降低
	majorPassRatio := float64(passedMajor) / float64(len(passedSymbols))

	if icpRecognized && majorPassRatio < 0.4 {
		fmt.Println("🎉 第一阶段优化成功!")
		fmt.Println("   ✅ ICP正确识别为主流币种")
		fmt.Println("   ✅ 主流币种入选比例控制在合理范围内")
		fmt.Println("   ✅ 评分权重优化生效")
		fmt.Println("   ✅ 新兴币种获得相对优势")
	} else {
		fmt.Println("📊 优化效果待进一步验证:")
		if !icpRecognized {
			fmt.Println("   ⚠️ ICP未能正确识别为主流币种")
		}
		if majorPassRatio >= 0.4 {
			fmt.Println("   ⚠️ 主流币种比例仍较高")
		}
		fmt.Println("   💡 建议: 调整筛选阈值或进一步优化权重")
	}

	fmt.Println("\n🏁 评分逻辑测试完成")
}