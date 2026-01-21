package main

import (
	"fmt"
)

// 模拟结构体
type StrategyConditions struct {
	MRCandidateMinOscillation float64
	MRCandidateMinLiquidity   float64
	MRCandidateMaxVolatility  float64
	MeanReversionSubMode      string
}

type MarketEnvironment struct {
	Type string
}

type EnhancedCandidateScore struct {
	Symbol                string
	OscillationScore      float64
	LiquidityScore        float64
	VolatilityScore       float64
	MomentumScore         float64
	VolumeStabilityScore  float64
	MarketCapScore        float64
	RecentPerformanceScore float64
	TotalScore            float64
}

// 候选币种优化器模拟
type CandidateOptimizer struct{}

// 计算加权综合评分
func (co *CandidateOptimizer) calculateWeightedTotalScore(score EnhancedCandidateScore, env MarketEnvironment, conditions StrategyConditions) float64 {
	// 基础权重
	baseWeights := map[string]float64{
		"oscillation":       0.25,
		"liquidity":         0.20,
		"volatility":        0.15,
		"momentum":          0.15,
		"volumeStability":   0.10,
		"marketCap":         0.10,
		"recentPerformance": 0.05,
	}

	// 根据市场环境调整权重
	switch env.Type {
	case "oscillation":
		baseWeights["oscillation"] = 0.35
		baseWeights["momentum"] = 0.10

	case "strong_trend":
		baseWeights["momentum"] = 0.25
		baseWeights["oscillation"] = 0.20

	case "high_volatility":
		baseWeights["liquidity"] = 0.25
		baseWeights["volatility"] = 0.20

	case "sideways":
		// 保持默认权重
	}

	// 计算加权总分
	totalScore :=
		score.OscillationScore*baseWeights["oscillation"] +
		score.LiquidityScore*baseWeights["liquidity"] +
		score.VolatilityScore*baseWeights["volatility"] +
		(1-score.MomentumScore)*baseWeights["momentum"] +
		score.VolumeStabilityScore*baseWeights["volumeStability"] +
		score.MarketCapScore*baseWeights["marketCap"] +
		score.RecentPerformanceScore*baseWeights["recentPerformance"]

	return totalScore
}

// 根据市场环境应用筛选策略
func (co *CandidateOptimizer) applyMarketEnvironmentFilters(candidates []EnhancedCandidateScore, env MarketEnvironment, conditions StrategyConditions) []EnhancedCandidateScore {
	var filtered []EnhancedCandidateScore

	for _, candidate := range candidates {
		shouldInclude := true

		// 基础质量筛选
		if candidate.OscillationScore < conditions.MRCandidateMinOscillation ||
		   candidate.LiquidityScore < conditions.MRCandidateMinLiquidity {
			continue
		}

		// 根据市场环境应用特殊筛选
		switch env.Type {
		case "oscillation":
			if candidate.OscillationScore < 0.7 {
				shouldInclude = false
			}
			if candidate.MomentumScore > 0.8 {
				shouldInclude = false
			}

		case "strong_trend":
			if candidate.OscillationScore < 0.8 ||
			   candidate.LiquidityScore < 0.8 ||
			   candidate.VolatilityScore < 0.6 {
				shouldInclude = false
			}
			if candidate.MomentumScore > 0.6 {
				shouldInclude = false
			}

		case "high_volatility":
			if candidate.VolatilityScore > conditions.MRCandidateMaxVolatility {
				shouldInclude = false
			}
			if candidate.LiquidityScore < 0.7 {
				shouldInclude = false
			}

		case "sideways":
			if candidate.OscillationScore < 0.5 {
				shouldInclude = false
			}
		}

		// 根据子模式调整筛选标准
		switch conditions.MeanReversionSubMode {
		case "conservative":
			if candidate.TotalScore < 0.75 ||
			   candidate.VolumeStabilityScore < 0.7 {
				shouldInclude = false
			}

		case "aggressive":
			if candidate.TotalScore < 0.5 {
				shouldInclude = false
			}
		}

		if shouldInclude {
			filtered = append(filtered, candidate)
		}
	}

	return filtered
}

func main() {
	fmt.Println("🎯 候选币种优化器测试")
	fmt.Println("=============================================")

	conditions := StrategyConditions{
		MRCandidateMinOscillation: 0.5,
		MRCandidateMinLiquidity:   0.6,
		MRCandidateMaxVolatility:  0.15,
		MeanReversionSubMode:      "conservative",
	}

	// 模拟候选币种数据
	candidates := []EnhancedCandidateScore{
		{
			Symbol:                "BTC",
			OscillationScore:      0.8,
			LiquidityScore:        0.95,
			VolatilityScore:       0.12,
			MomentumScore:         0.3,
			VolumeStabilityScore:  0.85,
			MarketCapScore:        0.9,
			RecentPerformanceScore: 0.8,
		},
		{
			Symbol:                "ETH",
			OscillationScore:      0.75,
			LiquidityScore:        0.9,
			VolatilityScore:       0.18,
			MomentumScore:         0.6,
			VolumeStabilityScore:  0.8,
			MarketCapScore:        0.85,
			RecentPerformanceScore: 0.75,
		},
		{
			Symbol:                "ADA",
			OscillationScore:      0.85,
			LiquidityScore:        0.7,
			VolatilityScore:       0.25,
			MomentumScore:         0.2,
			VolumeStabilityScore:  0.9,
			MarketCapScore:        0.6,
			RecentPerformanceScore: 0.9,
		},
		{
			Symbol:                "DOGE",
			OscillationScore:      0.4,
			LiquidityScore:        0.8,
			VolatilityScore:       0.3,
			MomentumScore:         0.8,
			VolumeStabilityScore:  0.7,
			MarketCapScore:        0.4,
			RecentPerformanceScore: 0.6,
		},
	}

	environments := []struct {
		name string
		env  MarketEnvironment
	}{
		{"震荡市", MarketEnvironment{Type: "oscillation"}},
		{"强趋势市", MarketEnvironment{Type: "strong_trend"}},
		{"高波动市", MarketEnvironment{Type: "high_volatility"}},
	}

	optimizer := &CandidateOptimizer{}

	for _, test := range environments {
		fmt.Printf("\n📊 %s 候选币种筛选:\n", test.name)

		// 计算综合评分
		for i := range candidates {
			candidates[i].TotalScore = optimizer.calculateWeightedTotalScore(candidates[i], test.env, conditions)
		}

		// 应用筛选
		filtered := optimizer.applyMarketEnvironmentFilters(candidates, test.env, conditions)

		fmt.Printf("  原始候选: %d个\n", len(candidates))
		fmt.Printf("  筛选后候选: %d个\n", len(filtered))

		for i, candidate := range filtered {
			if i >= 3 { // 只显示前3个
				break
			}
			fmt.Printf("    %d. %s (评分: %.3f, 振荡: %.2f, 动量: %.2f)\n",
				i+1, candidate.Symbol, candidate.TotalScore,
				candidate.OscillationScore, candidate.MomentumScore)
		}
	}

	fmt.Println("\n✅ 候选币种优化器测试完成")
	fmt.Println("系统能够根据市场环境智能筛选最适合均值回归的币种，")
	fmt.Println("在震荡市优先选择高振荡性币种，在趋势市严格筛选优质标的。")
}