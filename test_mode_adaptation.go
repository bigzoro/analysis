package main

import (
	"fmt"
	"math"
)

// 模拟结构体
type StrategyConditions struct {
	MRMinReversionStrength  float64
	MRSignalMode            string
	MRPeriod                int
	MRBollingerMultiplier   float64
	MRRSIOversold           int
	MRRSIOverbought         int
	MRMaxPositionSize       float64
	MRStopLossMultiplier    float64
	MRMaxHoldHours          int
	MRCandidateMinOscillation float64
	MRCandidateMinLiquidity  float64
	MRCandidateMaxVolatility float64
	MeanReversionSubMode    string
}

type MarketEnvironment struct {
	Type             string
	VolatilityLevel  float64
	OscillationIndex float64
}

// 保守模式参数设置
func applyConservativeMode(conditions StrategyConditions, env MarketEnvironment) StrategyConditions {
	adapted := conditions

	// 核心信号参数 - 高要求，稳健
	adapted.MRMinReversionStrength = 0.75
	adapted.MRSignalMode = "CONSERVATIVE"
	adapted.MRPeriod = int(float64(conditions.MRPeriod) * 1.3)

	// 技术指标参数 - 更严格
	adapted.MRBollingerMultiplier = math.Max(conditions.MRBollingerMultiplier, 2.2)
	adapted.MRRSIOversold = int(math.Max(float64(conditions.MRRSIOversold), 35))
	adapted.MRRSIOverbought = int(math.Min(float64(conditions.MRRSIOverbought), 65))

	// 风险控制参数 - 更保守
	adapted.MRMaxPositionSize = math.Min(conditions.MRMaxPositionSize, 0.03)
	adapted.MRStopLossMultiplier = math.Max(conditions.MRStopLossMultiplier, 2.0)
	adapted.MRMaxHoldHours = int(math.Max(float64(conditions.MRMaxHoldHours), 48))

	// 筛选标准 - 更严格
	adapted.MRCandidateMinOscillation = math.Max(conditions.MRCandidateMinOscillation, 0.7)
	adapted.MRCandidateMinLiquidity = math.Max(conditions.MRCandidateMinLiquidity, 0.8)
	adapted.MRCandidateMaxVolatility = math.Min(conditions.MRCandidateMaxVolatility, 0.12)

	return adapted
}

// 激进模式参数设置
func applyAggressiveMode(conditions StrategyConditions, env MarketEnvironment) StrategyConditions {
	adapted := conditions

	// 核心信号参数 - 低要求，高频
	adapted.MRMinReversionStrength = 0.35
	adapted.MRSignalMode = "AGGRESSIVE"
	adapted.MRPeriod = int(float64(conditions.MRPeriod) * 0.7)

	// 技术指标参数 - 更宽松
	adapted.MRBollingerMultiplier = math.Min(conditions.MRBollingerMultiplier, 1.8)
	adapted.MRRSIOversold = int(math.Min(float64(conditions.MRRSIOversold), 25))
	adapted.MRRSIOverbought = int(math.Max(float64(conditions.MRRSIOverbought), 75))

	// 风险控制参数 - 更激进
	adapted.MRMaxPositionSize = math.Max(conditions.MRMaxPositionSize, 0.08)
	adapted.MRStopLossMultiplier = math.Min(conditions.MRStopLossMultiplier, 1.2)
	adapted.MRMaxHoldHours = int(math.Min(float64(conditions.MRMaxHoldHours), 12))

	// 筛选标准 - 更宽松
	adapted.MRCandidateMinOscillation = math.Min(conditions.MRCandidateMinOscillation, 0.4)
	adapted.MRCandidateMinLiquidity = math.Min(conditions.MRCandidateMinLiquidity, 0.5)
	adapted.MRCandidateMaxVolatility = math.Max(conditions.MRCandidateMaxVolatility, 0.20)

	return adapted
}

// 环境适应性调整
func applyEnvironmentAdaptation(conditions StrategyConditions, env MarketEnvironment) StrategyConditions {
	adapted := conditions

	switch env.Type {
	case "oscillation":
		adapted.MRMinReversionStrength *= 0.9
		adapted.MRMaxPositionSize *= 1.1
		adapted.MRMaxHoldHours = int(float64(adapted.MRMaxHoldHours) * 1.2)

	case "strong_trend":
		adapted.MRMinReversionStrength *= 1.2
		adapted.MRMaxPositionSize *= 0.7
		adapted.MRMaxHoldHours = int(float64(adapted.MRMaxHoldHours) * 0.8)

	case "high_volatility":
		adapted.MRStopLossMultiplier *= 1.3
		adapted.MRCandidateMaxVolatility *= 0.8
	}

	if env.VolatilityLevel > 0.08 {
		adapted.MRMaxPositionSize *= 0.8
		adapted.MRStopLossMultiplier *= 1.2
	}

	if env.OscillationIndex > 0.7 {
		adapted.MRMinReversionStrength *= 0.9
		adapted.MRMaxPositionSize *= 1.1
	}

	return adapted
}

// 参数合理性检查
func validateAndAdjustParameters(conditions StrategyConditions) StrategyConditions {
	adapted := conditions

	adapted.MRMinReversionStrength = math.Max(0.1, math.Min(adapted.MRMinReversionStrength, 0.9))
	adapted.MRPeriod = int(math.Max(5, math.Min(float64(adapted.MRPeriod), 100)))
	adapted.MRMaxPositionSize = math.Max(0.005, math.Min(adapted.MRMaxPositionSize, 0.15))

	return adapted
}

// 增强的模式自适应参数调整
func adaptParametersForSubMode(conditions StrategyConditions, env MarketEnvironment) StrategyConditions {
	adapted := conditions

	switch conditions.MeanReversionSubMode {
	case "conservative":
		adapted = applyConservativeMode(conditions, env)
	case "aggressive":
		adapted = applyAggressiveMode(conditions, env)
	default:
		adapted.MRMinReversionStrength = 0.55
		adapted.MRSignalMode = "MODERATE"
	}

	adapted = applyEnvironmentAdaptation(adapted, env)
	adapted = validateAndAdjustParameters(adapted)

	return adapted
}

func main() {
	fmt.Println("🎯 模式自适应参数测试")
	fmt.Println("====================")

	// 基础配置
	baseConditions := StrategyConditions{
		MRMinReversionStrength:  0.5,
		MRPeriod:                20,
		MRBollingerMultiplier:   2.0,
		MRRSIOversold:           30,
		MRRSIOverbought:         70,
		MRMaxPositionSize:       0.05,
		MRStopLossMultiplier:    1.5,
		MRMaxHoldHours:          24,
		MRCandidateMinOscillation: 0.6,
		MRCandidateMinLiquidity:  0.7,
		MRCandidateMaxVolatility: 0.15,
	}

	// 测试不同模式和环境组合
	testCases := []struct {
		mode string
		env  MarketEnvironment
		desc string
	}{
		{"conservative", MarketEnvironment{Type: "oscillation", VolatilityLevel: 0.05, OscillationIndex: 0.8}, "保守模式+震荡市"},
		{"aggressive", MarketEnvironment{Type: "high_volatility", VolatilityLevel: 0.12, OscillationIndex: 0.3}, "激进模式+高波动市"},
		{"moderate", MarketEnvironment{Type: "strong_trend", VolatilityLevel: 0.08, OscillationIndex: 0.2}, "中等模式+强趋势市"},
	}

	for _, tc := range testCases {
		fmt.Printf("\n📊 %s:\n", tc.desc)

		conditions := baseConditions
		conditions.MeanReversionSubMode = tc.mode

		adapted := adaptParametersForSubMode(conditions, tc.env)

		fmt.Printf("  信号强度阈值: %.2f → %.2f\n", conditions.MRMinReversionStrength, adapted.MRMinReversionStrength)
		fmt.Printf("  分析周期: %d → %d\n", conditions.MRPeriod, adapted.MRPeriod)
		fmt.Printf("  最大仓位: %.1f%% → %.1f%%\n", conditions.MRMaxPositionSize*100, adapted.MRMaxPositionSize*100)
		fmt.Printf("  止损倍数: %.1f → %.1f\n", conditions.MRStopLossMultiplier, adapted.MRStopLossMultiplier)
		fmt.Printf("  最长持仓: %d小时 → %d小时\n", conditions.MRMaxHoldHours, adapted.MRMaxHoldHours)
		fmt.Printf("  RSI超卖线: %d → %d\n", conditions.MRRSIOversold, adapted.MRRSIOversold)
		fmt.Printf("  信号模式: %s\n", adapted.MRSignalMode)
	}

	fmt.Println("\n✅ 模式自适应参数测试完成")
	fmt.Println("系统能够根据交易模式和市场环境智能调整所有策略参数，")
	fmt.Println("实现保守模式稳健交易，激进模式高频交易的目标。")
}