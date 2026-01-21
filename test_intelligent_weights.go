package main

import (
	"fmt"
	"math"
)

// 模拟IntelligentWeights结构体
type IntelligentWeights struct {
	BollingerBands float64 // 布林带权重
	RSI            float64 // RSI权重
	PriceChannel   float64 // 价格通道权重
	TimeDecay      float64 // 时间衰减权重
}

// 模拟MarketEnvironment结构体
type MarketEnvironment struct {
	Type             string  // "oscillation", "strong_trend", "high_volatility", "mixed"
	Confidence       float64 // 0-1
	TrendStrength    float64
	VolatilityLevel  float64
	OscillationIndex float64
}

// 模拟StrategyConditions结构体
type StrategyConditions struct {
	MRWeightBollingerBands float64
	MRWeightRSI            float64
	MRWeightPriceChannel   float64
	MRWeightTimeDecay      float64
}

// 智能权重计算 - 根据市场环境动态调整指标权重
func calculateIntelligentWeights(conditions StrategyConditions, env MarketEnvironment) IntelligentWeights {
	baseWeights := IntelligentWeights{
		BollingerBands: conditions.MRWeightBollingerBands,
		RSI:            conditions.MRWeightRSI,
		PriceChannel:   conditions.MRWeightPriceChannel,
		TimeDecay:      conditions.MRWeightTimeDecay,
	}

	// 根据市场环境调整权重
	switch env.Type {
	case "oscillation":
		// 震荡市：均值回归最佳环境，所有指标权重均衡
		baseWeights.BollingerBands *= 1.0
		baseWeights.RSI *= 1.0
		baseWeights.PriceChannel *= 1.0
		baseWeights.TimeDecay *= 0.8 // 时间衰减稍低

	case "strong_trend", "bull_trend", "bear_trend":
		// 强趋势市：均值回归困难，降低权重
		baseWeights.BollingerBands *= 0.7
		baseWeights.RSI *= 0.6
		baseWeights.PriceChannel *= 0.8
		baseWeights.TimeDecay *= 1.2 // 增强时间衰减

	case "high_volatility":
		// 高波动市：增加布林带权重，降低RSI权重
		baseWeights.BollingerBands *= 1.3
		baseWeights.RSI *= 0.7
		baseWeights.PriceChannel *= 0.9
		baseWeights.TimeDecay *= 1.1

	case "sideways":
		// 横盘整理：适合均值回归，适当增加权重
		baseWeights.BollingerBands *= 1.1
		baseWeights.RSI *= 1.1
		baseWeights.PriceChannel *= 1.1
		baseWeights.TimeDecay *= 0.9

	default:
		// 未知环境：使用基础权重
	}

	// 根据趋势强度调整
	trendAbs := math.Abs(env.TrendStrength)
	if trendAbs > 0.7 {
		// 强趋势：降低均值回归指标权重
		baseWeights.BollingerBands *= 0.8
		baseWeights.RSI *= 0.8
		baseWeights.PriceChannel *= 0.9
	} else if trendAbs < 0.3 {
		// 弱趋势：增加均值回归指标权重
		baseWeights.BollingerBands *= 1.1
		baseWeights.RSI *= 1.1
		baseWeights.PriceChannel *= 1.1
	}

	// 根据波动率调整
	if env.VolatilityLevel > 0.08 {
		// 高波动：增强布林带，降低RSI
		baseWeights.BollingerBands *= 1.2
		baseWeights.RSI *= 0.8
	} else if env.VolatilityLevel < 0.03 {
		// 低波动：增强RSI，降低布林带
		baseWeights.BollingerBands *= 0.9
		baseWeights.RSI *= 1.1
	}

	// 根据震荡指数调整
	if env.OscillationIndex > 0.7 {
		// 高震荡：所有指标权重增加
		baseWeights.BollingerBands *= 1.1
		baseWeights.RSI *= 1.1
		baseWeights.PriceChannel *= 1.1
	} else if env.OscillationIndex < 0.3 {
		// 低震荡：降低权重
		baseWeights.BollingerBands *= 0.9
		baseWeights.RSI *= 0.9
		baseWeights.PriceChannel *= 0.9
	}

	// 权重归一化
	totalWeight := baseWeights.BollingerBands + baseWeights.RSI + baseWeights.PriceChannel + baseWeights.TimeDecay
	if totalWeight > 0 {
		baseWeights.BollingerBands /= totalWeight
		baseWeights.RSI /= totalWeight
		baseWeights.PriceChannel /= totalWeight
		baseWeights.TimeDecay /= totalWeight
	}

	return baseWeights
}

func main() {
	fmt.Println("🧠 智能信号权重系统测试")
	fmt.Println("==================================================")

	// 基础配置
	conditions := StrategyConditions{
		MRWeightBollingerBands: 0.4,
		MRWeightRSI:            0.3,
		MRWeightPriceChannel:   0.2,
		MRWeightTimeDecay:      0.1,
	}

	// 测试不同市场环境
	environments := []struct {
		name string
		env  MarketEnvironment
	}{
		{
			name: "震荡市",
			env: MarketEnvironment{
				Type:             "oscillation",
				TrendStrength:    0.2,
				VolatilityLevel:  0.05,
				OscillationIndex: 0.8,
			},
		},
		{
			name: "强趋势市",
			env: MarketEnvironment{
				Type:             "strong_trend",
				TrendStrength:    0.8,
				VolatilityLevel:  0.03,
				OscillationIndex: 0.2,
			},
		},
		{
			name: "高波动市",
			env: MarketEnvironment{
				Type:             "high_volatility",
				TrendStrength:    0.1,
				VolatilityLevel:  0.12,
				OscillationIndex: 0.6,
			},
		},
	}

	for _, test := range environments {
		fmt.Printf("\n📊 %s 权重调整:\n", test.name)
		weights := calculateIntelligentWeights(conditions, test.env)
		fmt.Printf("  布林带: %.3f (%.1f%%)\n", weights.BollingerBands, weights.BollingerBands*100)
		fmt.Printf("  RSI:     %.3f (%.1f%%)\n", weights.RSI, weights.RSI*100)
		fmt.Printf("  价格通道: %.3f (%.1f%%)\n", weights.PriceChannel, weights.PriceChannel*100)
		fmt.Printf("  时间衰减: %.3f (%.1f%%)\n", weights.TimeDecay, weights.TimeDecay*100)
	}

	fmt.Println("\n✅ 智能权重系统测试完成")
	fmt.Println("系统能够根据市场环境动态调整各指标权重，")
	fmt.Println("在震荡市增加均值回归指标权重，在趋势市降低权重。")
}