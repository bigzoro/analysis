package validation

import (
	"analysis/internal/server/strategy/moving_average"
)

// Validator 验证器实现
type Validator struct {
	signalProcessor moving_average.SignalProcessor
}

// NewValidator 创建验证器
func NewValidator(signalProcessor moving_average.SignalProcessor) moving_average.Validator {
	return &Validator{
		signalProcessor: signalProcessor,
	}
}

// ValidatePriceRange 验证价格范围
func (v *Validator) ValidatePriceRange(price float64, config *moving_average.MovingAverageConfig) bool {
	return price >= config.MinPriceThreshold && price <= config.MaxPriceThreshold
}

// ValidateVolume 验证交易量
func (v *Validator) ValidateVolume(volume float64, config *moving_average.MovingAverageConfig) bool {
	return volume >= config.MinVolumeThreshold
}

// ValidateCrossStrength 验证交叉强度
func (v *Validator) ValidateCrossStrength(strength float64, config *moving_average.MovingAverageConfig) bool {
	return strength >= config.MinCrossStrength
}

// CalculateOverallScore 计算整体评分
func (v *Validator) CalculateOverallScore(result *moving_average.ValidationResult, config *moving_average.MovingAverageConfig) float64 {
	if result.Signal == nil {
		return 0.0
	}

	score := result.Score

	// 额外因素调整
	// 这里可以添加更多评分逻辑，如：
	// - 市场趋势
	// - 波动率
	// - 资金流向等

	// 确保在0-1范围内
	if score > 1.0 {
		score = 1.0
	} else if score < 0.0 {
		score = 0.0
	}

	return score
}
