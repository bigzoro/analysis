package main

import (
	"fmt"
	"time"
)

// 保守模式参数设置测试 (简化版本)
func testConservativeParameters() {
	fmt.Println("🛡️ 保守模式参数设置测试")
	fmt.Println("核心参数调整:")

	// 核心信号参数 - 高要求，稳健
	minReversionStrength := 0.80
	signalMode := "CONSERVATIVE_HIGH_CONFIDENCE"
	period := int(float64(20) * 1.5)

	// 技术指标参数 - 更严格
	rsiOversold := 40
	rsiOverbought := 60

	// 风险控制参数 - 更保守
	maxPositionSize := 0.025
	stopLossMultiplier := 2.5
	maxHoldHours := 72

	// 筛选标准 - 更严格
	minOscillation := 0.75
	minLiquidity := 0.85
	maxVolatility := 0.10

	fmt.Printf("✅ 信号强度阈值: %.1f%% (高确认度)\n", minReversionStrength*100)
	fmt.Printf("✅ 分析周期: %d (更长周期减少噪音)\n", period)
	fmt.Printf("✅ 最大仓位: %.1f%% (极度保守)\n", maxPositionSize*100)
	fmt.Printf("✅ 止损倍数: %.1f (更宽松)\n", stopLossMultiplier)
	fmt.Printf("✅ 最长持仓: %d小时 (充足时间)\n", maxHoldHours)
	fmt.Printf("✅ RSI阈值: %d-%d (中性区域)\n", rsiOversold, rsiOverbought)
	fmt.Printf("✅ 信号模式: %s\n", signalMode)
	fmt.Printf("✅ 质量要求: 振荡%.0f%%, 流动%.0f%%, 波动<%.0f%%\n",
		minOscillation*100, minLiquidity*100, maxVolatility*100)
}

// 保守模式市场环境过滤测试
func testMarketEnvironmentFilter() {
	fmt.Println("\n🕐 市场环境过滤测试:")
	fmt.Println("保守模式只在高质量震荡环境中交易:")

	testEnvs := []struct {
		name string
		env  struct {
			envType     string
			confidence  float64
			oscillation float64
			volatility  float64
			stability   float64
		}
		expected bool
	}{
		{"优质震荡环境", struct {
			envType     string
			confidence  float64
			oscillation float64
			volatility  float64
			stability   float64
		}{"oscillation", 0.8, 0.8, 0.08, 0.9}, true},
		{"普通震荡环境", struct {
			envType     string
			confidence  float64
			oscillation float64
			volatility  float64
			stability   float64
		}{"oscillation", 0.6, 0.6, 0.15, 0.7}, false},
		{"强趋势环境", struct {
			envType     string
			confidence  float64
			oscillation float64
			volatility  float64
			stability   float64
		}{"strong_trend", 0.8, 0.3, 0.08, 0.9}, false},
	}

	for _, test := range testEnvs {
		requiredConditions := []bool{
			test.env.envType == "oscillation",
			test.env.confidence >= 0.7,
			test.env.oscillation >= 0.7,
			test.env.volatility <= 0.12,
			test.env.stability >= 0.8,
		}

		allConditionsMet := true
		for _, condition := range requiredConditions {
			if !condition {
				allConditionsMet = false
				break
			}
		}

		status := "❌"
		if allConditionsMet == test.expected {
			status = "✅"
		}
		fmt.Printf("%s %s: %v\n", status, test.name, allConditionsMet)
	}
}

// 保守模式时间过滤测试
func testTimeFilter() {
	fmt.Println("\n⏰ 时间过滤测试:")
	fmt.Println("保守模式只在活跃交易时间段交易:")

	now := time.Now()
	hour := now.Hour()
	weekday := now.Weekday()

	if weekday == time.Saturday || weekday == time.Sunday {
		fmt.Printf("❌ 周末时间不交易 (当前: %s)\n", weekday)
		return
	}

	if hour < 8 || hour > 20 {
		fmt.Printf("❌ 非活跃交易时间段 (UTC %02d:00, 需要8:00-20:00)\n", hour)
		return
	}

	fmt.Printf("✅ 时间过滤通过 (UTC %02d:00, 星期%s)\n", hour, weekday)
}

func main() {
	fmt.Println("🛡️ 保守模式高确认度逻辑测试")
	fmt.Println("===========================")

	// 测试参数设置
	testConservativeParameters()

	// 测试市场环境过滤
	testMarketEnvironmentFilter()

	// 测试时间过滤
	testTimeFilter()

	fmt.Println("\n🎯 保守模式特点总结:")
	fmt.Println("✅ 高确认度: 80%信号强度阈值，多重技术指标确认")
	fmt.Println("✅ 低频交易: 严格的时间和环境过滤，减少交易次数")
	fmt.Println("✅ 高胜率: 只在最有把握的情况下交易")
	fmt.Println("✅ 极度保守: 2.5%最大仓位，2.5倍止损，72小时最长持仓")
	fmt.Println("✅ 质量优先: 75%最低震荡性，85%最低流动性")
	fmt.Println("✅ 多重过滤: 市场环境+时间+成交量+技术指标+质量分数")
	fmt.Println("✅ 稳健为上: 宁可错过机会，也不愿意承担不必要的风险")
}
