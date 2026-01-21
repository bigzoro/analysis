package config

import (
	"analysis/internal/server/strategy/mean_reversion"
	"fmt"

	pdb "analysis/internal/db"
)

// configManager 配置管理器实现
type configManager struct{}

// NewMRConfigManager 创建均值回归配置管理器
func NewMRConfigManager() mean_reversion.MRConfigManager {
	return &configManager{}
}

// ConvertToUnifiedConfig 将旧的StrategyConditions转换为统一配置
func (cm *configManager) ConvertToUnifiedConfig(conditions pdb.StrategyConditions) (*mean_reversion.MeanReversionConfig, error) {
	config := &mean_reversion.MeanReversionConfig{}

	// 策略启用
	config.Enabled = conditions.MeanReversionEnabled

	// 核心参数 - 统一周期和指标选择
	config.Core.Period = conditions.MRPeriod
	if config.Core.Period <= 0 {
		config.Core.Period = 20 // 默认值
	}

	// 动态选择启用的指标
	var enabledIndicators []string
	if conditions.MRBollingerBandsEnabled {
		enabledIndicators = append(enabledIndicators, "bollinger")
	}
	if conditions.MRRSIEnabled {
		enabledIndicators = append(enabledIndicators, "rsi")
	}
	if conditions.MRPriceChannelEnabled {
		enabledIndicators = append(enabledIndicators, "price_channel")
	}
	config.Core.Indicators = enabledIndicators

	// 信号模式转换
	config.Core.Mode = cm.convertSignalMode(conditions.MRSignalMode, conditions.MeanReversionSubMode)

	// 指标配置
	config.Indicators.Bollinger.Enabled = conditions.MRBollingerBandsEnabled
	config.Indicators.Bollinger.Multiplier = conditions.MRBollingerMultiplier
	if config.Indicators.Bollinger.Multiplier <= 0 {
		config.Indicators.Bollinger.Multiplier = 2.0
	}
	config.Indicators.Bollinger.Weight = 0.7 // 默认权重

	config.Indicators.RSI.Enabled = conditions.MRRSIEnabled
	config.Indicators.RSI.Overbought = conditions.MRRSIOverbought
	config.Indicators.RSI.Oversold = conditions.MRRSIOversold
	if config.Indicators.RSI.Overbought <= 0 {
		config.Indicators.RSI.Overbought = 70
	}
	if config.Indicators.RSI.Oversold <= 0 {
		config.Indicators.RSI.Oversold = 30
	}
	config.Indicators.RSI.Weight = 0.3 // 默认权重

	config.Indicators.PriceChannel.Enabled = conditions.MRPriceChannelEnabled
	config.Indicators.PriceChannel.Weight = 0.3 // 默认权重

	// 信号质量控制 - 基于模式设置默认值
	qualityDefaults := cm.getDefaultSignalQualityForMode(config.Core.Mode)
	config.SignalQuality.MinStrength = qualityDefaults.MinStrength
	config.SignalQuality.MinConfidence = qualityDefaults.MinConfidence
	config.SignalQuality.MinConsistency = qualityDefaults.MinConsistency
	config.SignalQuality.MinQuality = qualityDefaults.MinQuality

	// 风险管理参数
	config.RiskManagement.MaxPositionSize = conditions.MRMaxPositionSize
	config.RiskManagement.StopLoss = 1.0 / conditions.MRStopLossMultiplier     // 转换为比例
	config.RiskManagement.TakeProfit = conditions.MRTakeProfitMultiplier - 1.0 // 转换为比例
	config.RiskManagement.MaxHoldHours = conditions.MRMaxHoldHours

	// 设置默认值
	if config.RiskManagement.MaxPositionSize <= 0 {
		config.RiskManagement.MaxPositionSize = 0.02 // 2%
	}
	if config.RiskManagement.StopLoss <= 0 {
		config.RiskManagement.StopLoss = 0.03 // 3%
	}
	if config.RiskManagement.TakeProfit <= 0 {
		config.RiskManagement.TakeProfit = 0.06 // 6%
	}
	if config.RiskManagement.MaxHoldHours <= 0 {
		config.RiskManagement.MaxHoldHours = 24
	}

	// 增强功能
	config.Enhancements.MarketEnvironmentDetection = conditions.MarketEnvironmentDetection
	config.Enhancements.IntelligentWeights = conditions.IntelligentWeights
	config.Enhancements.AdaptiveParameters = conditions.AdaptiveParameters
	config.Enhancements.PerformanceMonitoring = conditions.PerformanceMonitoring

	// 杠杆配置
	config.EnableLeverage = conditions.EnableLeverage
	config.DefaultLeverage = conditions.DefaultLeverage
	config.MaxLeverage = conditions.MaxLeverage

	// 交易配置
	marginMode := conditions.MarginMode
	if marginMode == "" {
		marginMode = "ISOLATED" // 默认逐仓
	}
	config.MarginMode = marginMode

	return config, nil
}

// ValidateConfig 验证统一配置的完整性和合理性
func (cm *configManager) ValidateConfig(config *mean_reversion.MeanReversionConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	// 基础验证
	if !config.Enabled {
		return nil // 未启用，无需验证
	}

	// 周期验证
	if config.Core.Period <= 0 {
		return fmt.Errorf("计算周期必须大于0")
	}

	// 指标验证
	if len(config.Core.Indicators) == 0 {
		return fmt.Errorf("至少需要启用一个指标")
	}

	for _, indicator := range config.Core.Indicators {
		switch indicator {
		case "bollinger":
			if !config.Indicators.Bollinger.Enabled {
				return fmt.Errorf("布林带指标在列表中但未启用")
			}
		case "rsi":
			if !config.Indicators.RSI.Enabled {
				return fmt.Errorf("RSI指标在列表中但未启用")
			}
		case "price_channel":
			if !config.Indicators.PriceChannel.Enabled {
				return fmt.Errorf("价格通道指标在列表中但未启用")
			}
		default:
			return fmt.Errorf("未知指标类型: %s", indicator)
		}
	}

	// 权重验证
	totalWeight := 0.0
	if config.Indicators.Bollinger.Enabled {
		totalWeight += config.Indicators.Bollinger.Weight
	}
	if config.Indicators.RSI.Enabled {
		totalWeight += config.Indicators.RSI.Weight
	}
	if config.Indicators.PriceChannel.Enabled {
		totalWeight += config.Indicators.PriceChannel.Weight
	}

	if totalWeight <= 0 {
		return fmt.Errorf("总权重必须大于0")
	}

	// 信号质量阈值验证
	if config.SignalQuality.MinStrength < 0 || config.SignalQuality.MinStrength > 1 {
		return fmt.Errorf("最小信号强度必须在0-1之间")
	}
	if config.SignalQuality.MinConfidence < 0 || config.SignalQuality.MinConfidence > 1 {
		return fmt.Errorf("最小置信度必须在0-1之间")
	}
	if config.SignalQuality.MinConsistency < 0 || config.SignalQuality.MinConsistency > 1 {
		return fmt.Errorf("最小一致性必须在0-1之间")
	}
	if config.SignalQuality.MinQuality < 0 || config.SignalQuality.MinQuality > 1 {
		return fmt.Errorf("最小质量必须在0-1之间")
	}

	// 风险管理验证
	if config.RiskManagement.MaxPositionSize <= 0 || config.RiskManagement.MaxPositionSize > 1 {
		return fmt.Errorf("最大仓位比例必须在0-1之间")
	}
	if config.RiskManagement.StopLoss < 0 || config.RiskManagement.StopLoss > 1 {
		return fmt.Errorf("止损比例必须在0-1之间")
	}
	if config.RiskManagement.TakeProfit < 0 {
		return fmt.Errorf("止盈比例不能为负数")
	}
	if config.RiskManagement.MaxHoldHours <= 0 {
		return fmt.Errorf("最大持仓小时数必须大于0")
	}

	return nil
}

// GetDefaultConfig 获取指定模式的默认配置
func (cm *configManager) GetDefaultConfig(mode mean_reversion.MRSignalMode) *mean_reversion.MeanReversionConfig {
	config := &mean_reversion.MeanReversionConfig{
		Enabled: true,
		Core: struct {
			Mode       mean_reversion.MRSignalMode `json:"mode"`
			Period     int                         `json:"period"`
			Indicators []string                    `json:"indicators"`
		}{
			Mode:       mode,
			Period:     20,
			Indicators: []string{"bollinger", "rsi"},
		},
	}

	// 指标配置
	config.Indicators.Bollinger.Enabled = true
	config.Indicators.Bollinger.Multiplier = 2.0
	config.Indicators.Bollinger.Weight = 0.7

	config.Indicators.RSI.Enabled = true
	config.Indicators.RSI.Overbought = 70
	config.Indicators.RSI.Oversold = 30
	config.Indicators.RSI.Weight = 0.3

	// 信号质量 - 基于模式
	qualityDefaults := cm.getDefaultSignalQualityForMode(mode)
	config.SignalQuality.MinStrength = qualityDefaults.MinStrength
	config.SignalQuality.MinConfidence = qualityDefaults.MinConfidence
	config.SignalQuality.MinConsistency = qualityDefaults.MinConsistency
	config.SignalQuality.MinQuality = qualityDefaults.MinQuality

	// 风险管理
	config.RiskManagement.MaxPositionSize = 0.02
	config.RiskManagement.StopLoss = 0.03
	config.RiskManagement.TakeProfit = 0.06
	config.RiskManagement.MaxHoldHours = 24

	// 杠杆配置
	config.EnableLeverage = false // 默认不启用杠杆
	config.DefaultLeverage = 1    // 默认1倍杠杆
	config.MaxLeverage = 100      // 默认最大100倍杠杆

	// 交易配置
	config.MarginMode = "ISOLATED" // 默认逐仓

	return config
}

// convertSignalMode 信号模式转换
func (cm *configManager) convertSignalMode(oldMode, subMode string) mean_reversion.MRSignalMode {
	// 优先使用新的子模式定义
	if subMode == "conservative" {
		return mean_reversion.MRConservativeMode
	}
	if subMode == "aggressive" {
		return mean_reversion.MRAggressiveMode
	}
	if subMode == "adaptive" {
		return mean_reversion.MRAdaptiveMode
	}

	// 回退到旧的模式定义
	switch oldMode {
	case "CONSERVATIVE", "conservative":
		return mean_reversion.MRConservativeMode
	case "AGGRESSIVE", "aggressive":
		return mean_reversion.MRAggressiveMode
	case "MODERATE", "moderate":
		return mean_reversion.MRBalancedMode
	case "ADAPTIVE", "adaptive":
		return mean_reversion.MRAdaptiveMode
	default:
		return mean_reversion.MRBalancedMode // 默认平衡模式
	}
}

// getDefaultSignalQualityForMode 根据模式获取默认的信号质量参数
func (cm *configManager) getDefaultSignalQualityForMode(mode mean_reversion.MRSignalMode) struct {
	MinStrength    float64 `json:"min_strength"`
	MinConfidence  float64 `json:"min_confidence"`
	MinConsistency float64 `json:"min_consistency"`
	MinQuality     float64 `json:"min_quality"`
} {
	switch mode {
	case mean_reversion.MRConservativeMode:
		return struct {
			MinStrength    float64 `json:"min_strength"`
			MinConfidence  float64 `json:"min_confidence"`
			MinConsistency float64 `json:"min_consistency"`
			MinQuality     float64 `json:"min_quality"`
		}{
			MinStrength:    0.8, // 高强度要求
			MinConfidence:  0.7, // 高置信度要求
			MinConsistency: 0.8, // 高一致性要求
			MinQuality:     0.7, // 高质量要求
		}

	case mean_reversion.MRAggressiveMode:
		return struct {
			MinStrength    float64 `json:"min_strength"`
			MinConfidence  float64 `json:"min_confidence"`
			MinConsistency float64 `json:"min_consistency"`
			MinQuality     float64 `json:"min_quality"`
		}{
			MinStrength:    0.4, // 低强度要求
			MinConfidence:  0.4, // 低置信度要求
			MinConsistency: 0.5, // 低一致性要求
			MinQuality:     0.4, // 低质量要求
		}

	case mean_reversion.MRAdaptiveMode:
		return struct {
			MinStrength    float64 `json:"min_strength"`
			MinConfidence  float64 `json:"min_confidence"`
			MinConsistency float64 `json:"min_consistency"`
			MinQuality     float64 `json:"min_quality"`
		}{
			MinStrength:    0.3, // 较低强度要求，增加信号数量
			MinConfidence:  0.5, // 中等置信度要求
			MinConsistency: 0.4, // 较低一致性要求，适应市场变化
			MinQuality:     0.3, // 较低质量要求，提高信号覆盖率
		}

	default: // MRBalancedMode
		return struct {
			MinStrength    float64 `json:"min_strength"`
			MinConfidence  float64 `json:"min_confidence"`
			MinConsistency float64 `json:"min_consistency"`
			MinQuality     float64 `json:"min_quality"`
		}{
			MinStrength:    0.6, // 中等强度要求
			MinConfidence:  0.6, // 中等置信度要求
			MinConsistency: 0.7, // 中等一致性要求
			MinQuality:     0.6, // 中等质量要求
		}
	}
}
