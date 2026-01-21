package main

import (
	"fmt"
	"math"
)

// 模拟结构体
type StrategyConditions struct {
	MRMaxDailyLoss         float64
	MRMaxPositionSize      float64
	MRStopLossMultiplier   float64
	MRTakeProfitMultiplier float64
	MRMaxHoldHours         int
	MeanReversionSubMode   string
}

type MarketEnvironment struct {
	Type             string
	Confidence       float64
	TrendStrength    float64
	VolatilityLevel  float64
	OscillationIndex float64
}

type DynamicRiskConfig struct {
	MaxDailyLoss         float64
	MaxPositionSize      float64
	StopLossMultiplier   float64
	TakeProfitMultiplier float64
	MaxHoldHours         int
	CurrentDailyLoss     float64
	MarketVolatility     float64
	PortfolioRiskLevel   float64
}

// 动态风险管理函数
func calculateDynamicRiskManagement(conditions StrategyConditions, env MarketEnvironment, currentDailyLoss float64) DynamicRiskConfig {
	baseConfig := DynamicRiskConfig{
		MaxDailyLoss:         conditions.MRMaxDailyLoss,
		MaxPositionSize:      conditions.MRMaxPositionSize,
		StopLossMultiplier:   conditions.MRStopLossMultiplier,
		TakeProfitMultiplier: conditions.MRTakeProfitMultiplier,
		MaxHoldHours:         conditions.MRMaxHoldHours,
		CurrentDailyLoss:     currentDailyLoss,
		MarketVolatility:     env.VolatilityLevel,
		PortfolioRiskLevel:   0.5,
	}

	// 默认值
	if baseConfig.MaxDailyLoss <= 0 {
		baseConfig.MaxDailyLoss = 0.03
	}
	if baseConfig.MaxPositionSize <= 0 {
		baseConfig.MaxPositionSize = 0.05
	}
	if baseConfig.StopLossMultiplier <= 0 {
		baseConfig.StopLossMultiplier = 1.5
	}
	if baseConfig.TakeProfitMultiplier <= 0 {
		baseConfig.TakeProfitMultiplier = 2.0
	}
	if baseConfig.MaxHoldHours <= 0 {
		baseConfig.MaxHoldHours = 24
	}

	// 根据市场环境调整
	switch env.Type {
	case "high_volatility":
		baseConfig.MaxDailyLoss *= 0.7
		baseConfig.MaxPositionSize *= 0.8
		baseConfig.StopLossMultiplier *= 0.9
		baseConfig.MaxHoldHours = int(float64(baseConfig.MaxHoldHours) * 0.8)

	case "strong_trend":
		baseConfig.MaxPositionSize *= 0.9
		baseConfig.StopLossMultiplier *= 1.2
		baseConfig.TakeProfitMultiplier *= 1.1

	case "oscillation":
		baseConfig.MaxPositionSize *= 1.1
		baseConfig.StopLossMultiplier *= 0.9
		baseConfig.MaxHoldHours = int(float64(baseConfig.MaxHoldHours) * 1.2)

	case "sideways":
		baseConfig.MaxPositionSize *= 0.9
		baseConfig.MaxHoldHours = int(float64(baseConfig.MaxHoldHours) * 1.3)
	}

	// 根据当前亏损调整
	if currentDailyLoss > 0 {
		remainingLossBudget := baseConfig.MaxDailyLoss - currentDailyLoss
		if remainingLossBudget > 0 {
			riskMultiplier := remainingLossBudget / baseConfig.MaxDailyLoss
			baseConfig.MaxPositionSize *= math.Max(0.3, riskMultiplier)
			baseConfig.StopLossMultiplier *= 0.9
		} else {
			baseConfig.MaxPositionSize = 0
		}
	}

	// 根据波动率调整
	if env.VolatilityLevel > 0.08 {
		baseConfig.MaxPositionSize *= 0.8
		baseConfig.StopLossMultiplier *= 0.9
	} else if env.VolatilityLevel < 0.03 {
		baseConfig.MaxPositionSize *= 1.1
		baseConfig.StopLossMultiplier *= 1.1
	}

	// 根据子模式调整
	switch conditions.MeanReversionSubMode {
	case "conservative":
		baseConfig.MaxPositionSize *= 0.8
		baseConfig.StopLossMultiplier *= 0.9
		baseConfig.MaxHoldHours = int(float64(baseConfig.MaxHoldHours) * 1.2)

	case "aggressive":
		baseConfig.MaxPositionSize *= 1.2
		baseConfig.StopLossMultiplier *= 1.1
		baseConfig.MaxHoldHours = int(float64(baseConfig.MaxHoldHours) * 0.8)
	}

	// 确保参数合理
	baseConfig.MaxDailyLoss = math.Max(0.005, math.Min(baseConfig.MaxDailyLoss, 0.1))
	baseConfig.MaxPositionSize = math.Max(0.005, math.Min(baseConfig.MaxPositionSize, 0.2))
	baseConfig.StopLossMultiplier = math.Max(1.1, math.Min(baseConfig.StopLossMultiplier, 3.0))
	baseConfig.TakeProfitMultiplier = math.Max(1.5, math.Min(baseConfig.TakeProfitMultiplier, 5.0))
	baseConfig.MaxHoldHours = int(math.Max(1, math.Min(float64(baseConfig.MaxHoldHours), 168)))

	return baseConfig
}

func calculateDynamicStopLoss(entryPrice float64, direction string, config DynamicRiskConfig, env MarketEnvironment) float64 {
	baseVolatility := math.Max(env.VolatilityLevel, 0.02)
	baseStopDistance := entryPrice * baseVolatility * config.StopLossMultiplier

	var stopLossPrice float64
	if direction == "long" {
		stopLossPrice = entryPrice - baseStopDistance
	} else {
		stopLossPrice = entryPrice + baseStopDistance
	}

	// 市场环境调整
	switch env.Type {
	case "high_volatility":
		adjustment := 1.2
		if direction == "long" {
			stopLossPrice = entryPrice - (baseStopDistance * adjustment)
		} else {
			stopLossPrice = entryPrice + (baseStopDistance * adjustment)
		}
	case "strong_trend":
		adjustment := 1.5
		if direction == "long" {
			stopLossPrice = entryPrice - (baseStopDistance * adjustment)
		} else {
			stopLossPrice = entryPrice + (baseStopDistance * adjustment)
		}
	}

	// 最小止损距离
	minStopDistance := entryPrice * 0.005
	if direction == "long" {
		stopLossPrice = math.Min(stopLossPrice, entryPrice-minStopDistance)
	} else {
		stopLossPrice = math.Max(stopLossPrice, entryPrice+minStopDistance)
	}

	return stopLossPrice
}

func main() {
	fmt.Println("🛡️ 动态风险管理框架测试")
	fmt.Println("========================================")

	conditions := StrategyConditions{
		MRMaxDailyLoss:         0.03,
		MRMaxPositionSize:      0.05,
		MRStopLossMultiplier:   1.5,
		MRTakeProfitMultiplier: 2.0,
		MRMaxHoldHours:         24,
		MeanReversionSubMode:   "conservative",
	}

	environments := []struct {
		name string
		env  MarketEnvironment
	}{
		{"震荡市", MarketEnvironment{Type: "oscillation", VolatilityLevel: 0.03}},
		{"高波动市", MarketEnvironment{Type: "high_volatility", VolatilityLevel: 0.12}},
		{"强趋势市", MarketEnvironment{Type: "strong_trend", VolatilityLevel: 0.05}},
	}

	for _, test := range environments {
		fmt.Printf("\n📊 %s 风险配置:\n", test.name)
		config := calculateDynamicRiskManagement(conditions, test.env, 0.0)
		fmt.Printf("  每日最大亏损: %.1f%%\n", config.MaxDailyLoss*100)
		fmt.Printf("  最大仓位比例: %.1f%%\n", config.MaxPositionSize*100)
		fmt.Printf("  止损倍数: %.1f倍\n", config.StopLossMultiplier)
		fmt.Printf("  止盈倍数: %.1f倍\n", config.TakeProfitMultiplier)
		fmt.Printf("  最大持仓时间: %d小时\n", config.MaxHoldHours)

		// 测试止损价格计算
		entryPrice := 50000.0
		stopLossLong := calculateDynamicStopLoss(entryPrice, "long", config, test.env)
		stopLossShort := calculateDynamicStopLoss(entryPrice, "short", config, test.env)
		fmt.Printf("  多头止损价格: %.2f (距离: %.1f%%)\n", stopLossLong, (entryPrice-stopLossLong)/entryPrice*100)
		fmt.Printf("  空头止损价格: %.2f (距离: %.1f%%)\n", stopLossShort, (stopLossShort-entryPrice)/entryPrice*100)
	}

	fmt.Println("\n✅ 动态风险管理框架测试完成")
	fmt.Println("系统能够根据市场环境和交易模式动态调整风险参数，")
	fmt.Println("在高波动市降低仓位，在震荡市增加持仓时间。")
}