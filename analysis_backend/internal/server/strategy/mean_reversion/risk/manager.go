package risk

import (
	"analysis/internal/server/strategy/mean_reversion"
	"math"
)

// Manager 风险管理器实现
type manager struct{}

// NewMRRiskManager 创建均值回归风险管理器
func NewMRRiskManager() mean_reversion.MRRiskManager {
	return &manager{}
}

// CalculatePositionSize 计算仓位大小
func (rm *manager) CalculatePositionSize(entryPrice float64, stopLoss float64, totalCapital float64, config *mean_reversion.MeanReversionConfig) float64 {
	if entryPrice <= 0 || stopLoss <= 0 || totalCapital <= 0 || config == nil {
		return 0
	}

	// 基础风险金额 = 总资金 * 最大仓位比例
	riskAmount := totalCapital * config.RiskManagement.MaxPositionSize

	// 损失金额 = 入场价格 * 止损比例
	lossAmount := entryPrice * stopLoss

	if lossAmount <= 0 {
		return 0
	}

	// 仓位大小 = 风险金额 / 损失金额
	positionSize := riskAmount / lossAmount

	// 限制在合理范围内
	maxPositionSize := totalCapital * config.RiskManagement.MaxPositionSize
	positionValue := positionSize * entryPrice

	if positionValue > maxPositionSize {
		positionSize = maxPositionSize / entryPrice
	}

	return positionSize
}

// CalculateStopLoss 计算止损价格
func (rm *manager) CalculateStopLoss(entryPrice float64, direction string, config *mean_reversion.MeanReversionConfig) float64 {
	if entryPrice <= 0 || config == nil {
		return entryPrice
	}

	stopLossRatio := config.RiskManagement.StopLoss

	switch direction {
	case "buy":
		return entryPrice * (1 - stopLossRatio)
	case "sell":
		return entryPrice * (1 + stopLossRatio)
	default:
		return entryPrice
	}
}

// CalculateTakeProfit 计算止盈价格
func (rm *manager) CalculateTakeProfit(entryPrice float64, stopLoss float64, direction string, config *mean_reversion.MeanReversionConfig) float64 {
	if entryPrice <= 0 || config == nil {
		return entryPrice
	}

	takeProfitRatio := config.RiskManagement.TakeProfit

	switch direction {
	case "buy":
		return entryPrice * (1 + takeProfitRatio)
	case "sell":
		return entryPrice * (1 - takeProfitRatio)
	default:
		return entryPrice
	}
}

// ValidateRiskLimits 验证风险限制
func (rm *manager) ValidateRiskLimits(positionSize float64, dailyLoss float64, config *mean_reversion.MeanReversionConfig) bool {
	if config == nil {
		return false
	}

	// 检查仓位大小限制
	maxPositionSize := config.RiskManagement.MaxPositionSize
	if positionSize > maxPositionSize {
		return false
	}

	// 检查每日损失限制（暂时简化）
	if dailyLoss < 0 {
		return false
	}

	return true
}

// ============================================================================
// 高级风险管理功能
// ============================================================================

// DynamicRiskManager 动态风险管理器
type DynamicRiskManager struct {
	baseManager mean_reversion.MRRiskManager
}

// NewDynamicRiskManager 创建动态风险管理器
func NewDynamicRiskManager() *DynamicRiskManager {
	return &DynamicRiskManager{
		baseManager: NewMRRiskManager(),
	}
}

// CalculateAdaptiveStopLoss 基于波动率计算自适应止损
func (drm *DynamicRiskManager) CalculateAdaptiveStopLoss(entryPrice float64, direction string, volatility float64, config *mean_reversion.MeanReversionConfig) float64 {
	if config == nil {
		return entryPrice
	}

	// 基础止损比例
	baseStopLoss := config.RiskManagement.StopLoss

	// 根据波动率调整
	var adjustedRatio float64
	switch {
	case volatility < 0.02: // 低波动
		adjustedRatio = baseStopLoss * 0.8 // 减少止损
	case volatility > 0.08: // 高波动
		adjustedRatio = baseStopLoss * 1.5 // 增加止损
	default: // 中等波动
		adjustedRatio = baseStopLoss
	}

	// 限制在合理范围内
	adjustedRatio = math.Max(0.005, math.Min(adjustedRatio, 0.10)) // 0.5% - 10%

	switch direction {
	case "buy":
		return entryPrice * (1 - adjustedRatio)
	case "sell":
		return entryPrice * (1 + adjustedRatio)
	default:
		return entryPrice
	}
}

// CalculatePositionSizeWithVolatility 基于波动率调整仓位大小
func (drm *DynamicRiskManager) CalculatePositionSizeWithVolatility(entryPrice float64, stopLoss float64, totalCapital float64, volatility float64, config *mean_reversion.MeanReversionConfig) float64 {
	if config == nil {
		return 0
	}

	// 基础仓位计算
	basePositionSize := drm.baseManager.CalculatePositionSize(entryPrice, stopLoss, totalCapital, config)

	// 根据波动率调整
	var volatilityMultiplier float64
	switch {
	case volatility < 0.02: // 低波动，可以增加仓位
		volatilityMultiplier = 1.2
	case volatility > 0.08: // 高波动，减少仓位
		volatilityMultiplier = 0.7
	default:
		volatilityMultiplier = 1.0
	}

	adjustedPositionSize := basePositionSize * volatilityMultiplier

	// 确保不超过最大仓位限制
	maxPositionSize := totalCapital * config.RiskManagement.MaxPositionSize
	positionValue := adjustedPositionSize * entryPrice

	if positionValue > maxPositionSize {
		adjustedPositionSize = maxPositionSize / entryPrice
	}

	return adjustedPositionSize
}

// CalculateRiskRewardRatio 计算风险收益比
func (drm *DynamicRiskManager) CalculateRiskRewardRatio(entryPrice, stopLoss, takeProfit float64, direction string) float64 {
	if entryPrice <= 0 {
		return 0
	}

	var risk, reward float64

	switch direction {
	case "buy":
		risk = entryPrice - stopLoss
		reward = takeProfit - entryPrice
	case "sell":
		risk = stopLoss - entryPrice
		reward = entryPrice - takeProfit
	default:
		return 0
	}

	if risk <= 0 {
		return 0
	}

	return reward / risk
}

// ValidateRiskRewardRatio 验证风险收益比
func (drm *DynamicRiskManager) ValidateRiskRewardRatio(ratio float64, minRatio float64) bool {
	return ratio >= minRatio
}
