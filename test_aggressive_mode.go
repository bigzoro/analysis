package main

import "fmt"

// 模拟StrategyConditions结构体
type StrategyConditions struct {
	MRMinReversionStrength float64 `json:"mr_min_reversion_strength"`
	MRSignalMode          string  `json:"mr_signal_mode"`
	MRPeriod              int     `json:"mr_period"`
	MRBollingerMultiplier float64 `json:"mr_bollinger_multiplier"`
	MRRSIOversold         int     `json:"mr_rsi_oversold"`
	MRRSIOverbought       int     `json:"mr_rsi_overbought"`
	MRMaxPositionSize     float64 `json:"mr_max_position_size"`
	MRStopLossMultiplier  float64 `json:"mr_stop_loss_multiplier"`
	MRMaxHoldHours        int     `json:"mr_max_hold_hours"`
	MRCandidateMinOscillation float64 `json:"mr_candidate_min_oscillation"`
	MRCandidateMinLiquidity   float64 `json:"mr_candidate_min_liquidity"`
	MRCandidateMaxVolatility  float64 `json:"mr_candidate_max_volatility"`
	MRRequireMultipleSignals         bool `json:"mr_require_multiple_signals"`
	MRRequireVolumeConfirmation      bool `json:"mr_require_volume_confirmation"`
	MRRequireTimeFilter              bool `json:"mr_require_time_filter"`
	MRRequireMarketEnvironmentFilter bool `json:"mr_require_market_environment_filter"`
}

// 模拟MarketEnvironment
type MarketEnvironment struct {
	Type string
}

func applyAggressiveMode(conditions StrategyConditions, env MarketEnvironment) StrategyConditions {
	adapted := conditions

	// ========== 核心信号参数 - 低确认度，高频 ==========
	adapted.MRMinReversionStrength = 0.25                      // 更低的信号强度要求 (25%)
	adapted.MRSignalMode = "AGGRESSIVE_HIGH_FREQUENCY"         // 高频激进模式
	adapted.MRPeriod = int(float64(conditions.MRPeriod) * 0.6) // 更短周期，更敏感 (60%原周期)

	// ========== 技术指标参数 - 单重快速 ==========
	adapted.MRBollingerMultiplier = 1.5  // 更窄的布林带 (1.5倍)
	adapted.MRRSIOversold = 20           // 更低的超卖线 (20)
	adapted.MRRSIOverbought = 80         // 更高的超买线 (80)

	// ========== 风险控制参数 - 激进高风险 ==========
	adapted.MRMaxPositionSize = 0.12     // 最大12%仓位 (更高风险)
	adapted.MRStopLossMultiplier = 1.0   // 更紧的止损 (1.0倍)
	adapted.MRMaxHoldHours = 6           // 最长6小时持仓 (快速进出)

	// ========== 筛选标准 - 极度宽松 ==========
	adapted.MRCandidateMinOscillation = 0.25 // 25%最小震荡 (很低)
	adapted.MRCandidateMinLiquidity = 0.35   // 35%最小流动性 (很低)
	adapted.MRCandidateMaxVolatility = 0.35  // 35%最大波动率 (很高容忍)

	// ========== 激进模式特殊参数 ==========
	adapted.MRRequireMultipleSignals = false        // 不需要多重信号确认 (单重即可)
	adapted.MRRequireVolumeConfirmation = false     // 不需要成交量确认 (快速交易)
	adapted.MRRequireTimeFilter = false             // 不需要时间过滤 (全时段交易)
	adapted.MRRequireMarketEnvironmentFilter = true // 仍需要市场环境过滤 (避免极端情况)

	return adapted
}

func main() {
	fmt.Println("🔥 激进模式参数设置测试")
	fmt.Println("========================")

	// 原始参数
	original := StrategyConditions{
		MRMinReversionStrength: 0.5,
		MRSignalMode:          "MODERATE",
		MRPeriod:              20,
		MRBollingerMultiplier: 2.0,
		MRRSIOversold:         30,
		MRRSIOverbought:       70,
		MRMaxPositionSize:     0.05,
		MRStopLossMultiplier:  1.5,
		MRMaxHoldHours:        24,
		MRCandidateMinOscillation: 0.6,
		MRCandidateMinLiquidity:   0.7,
		MRCandidateMaxVolatility:  0.15,
	}

	env := MarketEnvironment{Type: "oscillation"}

	// 应用激进模式
	adapted := applyAggressiveMode(original, env)

	fmt.Printf("📊 参数对比:\n")
	fmt.Printf("信号强度: %.2f → %.2f (降低要求)\n", original.MRMinReversionStrength, adapted.MRMinReversionStrength)
	fmt.Printf("信号模式: %s → %s\n", original.MRSignalMode, adapted.MRSignalMode)
	fmt.Printf("计算周期: %d → %d (缩短60%%)\n", original.MRPeriod, adapted.MRPeriod)
	fmt.Printf("布林倍数: %.1f → %.1f (更窄)\n", original.MRBollingerMultiplier, adapted.MRBollingerMultiplier)
	fmt.Printf("RSI超卖线: %d → %d (更低)\n", original.MRRSIOversold, adapted.MRRSIOversold)
	fmt.Printf("RSI超买线: %d → %d (更高)\n", original.MRRSIOverbought, adapted.MRRSIOverbought)

	fmt.Printf("\n💰 风险参数:\n")
	fmt.Printf("最大仓位: %.1f%% → %.1f%% (提高风险)\n", original.MRMaxPositionSize*100, adapted.MRMaxPositionSize*100)
	fmt.Printf("止损倍数: %.1f → %.1f (更紧)\n", original.MRStopLossMultiplier, adapted.MRStopLossMultiplier)
	fmt.Printf("最长持仓: %d小时 → %d小时 (快速进出)\n", original.MRMaxHoldHours, adapted.MRMaxHoldHours)

	fmt.Printf("\n🎯 筛选标准:\n")
	fmt.Printf("最小震荡: %.1f%% → %.1f%% (大幅降低)\n", original.MRCandidateMinOscillation*100, adapted.MRCandidateMinOscillation*100)
	fmt.Printf("最小流动性: %.1f%% → %.1f%% (大幅降低)\n", original.MRCandidateMinLiquidity*100, adapted.MRCandidateMinLiquidity*100)
	fmt.Printf("最大波动: %.1f%% → %.1f%% (大幅提高容忍)\n", original.MRCandidateMaxVolatility*100, adapted.MRCandidateMaxVolatility*100)

	fmt.Printf("\n🚫 过滤要求:\n")
	fmt.Printf("多重信号: %v → %v (不需要)\n", original.MRRequireMultipleSignals, adapted.MRRequireMultipleSignals)
	fmt.Printf("成交量确认: %v → %v (不需要)\n", original.MRRequireVolumeConfirmation, adapted.MRRequireVolumeConfirmation)
	fmt.Printf("时间过滤: %v → %v (不需要)\n", original.MRRequireTimeFilter, adapted.MRRequireTimeFilter)
	fmt.Printf("环境过滤: %v → %v (仍需要)\n", original.MRRequireMarketEnvironmentFilter, adapted.MRRequireMarketEnvironmentFilter)

	fmt.Println("\n✅ 激进模式参数设置完成！")
	fmt.Println("特点：低确认度、高频交易、高风险高收益、宽松筛选")
}