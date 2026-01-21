package signals

import (
	"analysis/internal/server/strategy/moving_average"
)

// Processor 信号处理器实现
type Processor struct {
	// 这里可以注入依赖
}

// NewProcessor 创建信号处理器
func NewProcessor() moving_average.SignalProcessor {
	return &Processor{}
}

// ProcessGoldenCross 处理金叉信号
func (p *Processor) ProcessGoldenCross(signal *moving_average.CrossSignal, config *moving_average.MovingAverageConfig) *moving_average.ValidationResult {
	result := &moving_average.ValidationResult{
		Symbol:  signal.Symbol,
		Signal:  signal,
		Action:  "buy",
		IsValid: false,
	}

	// 检查交叉强度
	if !p.validateCrossStrength(signal, config) {
		result.Reason = "交叉强度不足"
		return result
	}

	// 计算置信度
	result.Score = p.CalculateSignalConfidence(signal, config)

	// 检查是否需要交易量确认
	if config.RequireVolumeConfirmation {
		result.Reason = "金叉信号有效，等待交易量确认"
	} else {
		result.IsValid = result.Score >= 0.6 // 金叉信号阈值
		result.Reason = "金叉信号确认"
	}

	return result
}

// ProcessDeathCross 处理死叉信号
func (p *Processor) ProcessDeathCross(signal *moving_average.CrossSignal, config *moving_average.MovingAverageConfig) *moving_average.ValidationResult {
	result := &moving_average.ValidationResult{
		Symbol:  signal.Symbol,
		Signal:  signal,
		Action:  "sell",
		IsValid: false,
	}

	// 检查交叉强度
	if !p.validateCrossStrength(signal, config) {
		result.Reason = "交叉强度不足"
		return result
	}

	// 计算置信度
	result.Score = p.CalculateSignalConfidence(signal, config)

	// 检查是否需要交易量确认
	if config.RequireVolumeConfirmation {
		result.Reason = "死叉信号有效，等待交易量确认"
	} else {
		result.IsValid = result.Score >= 0.6 // 死叉信号阈值
		result.Reason = "死叉信号确认"
	}

	return result
}

// ConfirmWithVolume 使用交易量确认信号
func (p *Processor) ConfirmWithVolume(signal *moving_average.CrossSignal, volumes []float64, config *moving_average.MovingAverageConfig) bool {
	if len(volumes) == 0 {
		return false
	}

	// 简单的交易量确认：检查交叉点附近的交易量是否高于平均水平
	signalIndex := len(volumes) - 1 // 假设信号是针对最新的数据

	if signalIndex >= len(volumes) {
		return false
	}

	// 计算最近N周期的平均交易量
	confirmationPeriod := config.ConfirmationPeriod
	if confirmationPeriod <= 0 {
		confirmationPeriod = 3
	}

	startIndex := signalIndex - confirmationPeriod + 1
	if startIndex < 0 {
		startIndex = 0
	}

	sum := 0.0
	count := 0
	for i := startIndex; i <= signalIndex && i < len(volumes); i++ {
		if volumes[i] > 0 {
			sum += volumes[i]
			count++
		}
	}

	if count == 0 {
		return false
	}

	avgVolume := sum / float64(count)
	signalVolume := volumes[signalIndex]

	// 如果信号期间的交易量高于平均水平30%，则确认
	return signalVolume > avgVolume*1.3
}

// CalculateSignalConfidence 计算信号置信度
func (p *Processor) CalculateSignalConfidence(signal *moving_average.CrossSignal, config *moving_average.MovingAverageConfig) float64 {
	confidence := 0.0

	// 交叉强度贡献 (权重40%)
	strengthScore := signal.CrossStrength
	confidence += strengthScore * 0.4

	// 交易量确认贡献 (权重30%)
	volumeConfirmed := signal.VolumeConfirmed
	volumeScore := 0.0
	if volumeConfirmed {
		volumeScore = 1.0
	}
	confidence += volumeScore * 0.3

	// 信号类型贡献 (权重20%)
	// 金叉通常比死叉更可靠
	typeScore := 0.5
	if signal.SignalType == "golden_cross" {
		typeScore = 1.0
	} else if signal.SignalType == "death_cross" {
		typeScore = 0.7
	}
	confidence += typeScore * 0.2

	// 基础置信度 (权重10%)
	confidence += 0.8 * 0.1 // 均线策略的基础置信度

	// 确保在0-1范围内
	if confidence > 1.0 {
		confidence = 1.0
	} else if confidence < 0.0 {
		confidence = 0.0
	}

	return confidence
}

// validateCrossStrength 验证交叉强度
func (p *Processor) validateCrossStrength(signal *moving_average.CrossSignal, config *moving_average.MovingAverageConfig) bool {
	return signal.CrossStrength >= config.MinCrossStrength
}
