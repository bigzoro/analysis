package signals

import (
	"analysis/internal/server/strategy/mean_reversion"
	"fmt"
	"math"
)

// Processor 信号处理器实现
type processor struct{}

// NewMRSignalProcessor 创建均值回归信号处理器
func NewMRSignalProcessor() mean_reversion.MRSignalProcessor {
	return &processor{}
}

// ProcessSignals 处理多个指标信号，计算综合信号强度
func (p *processor) ProcessSignals(signals []*mean_reversion.IndicatorSignal, config *mean_reversion.MeanReversionConfig) (*mean_reversion.SignalStrength, error) {
	if len(signals) == 0 || config == nil {
		return &mean_reversion.SignalStrength{}, fmt.Errorf("信号为空或配置无效")
	}

	// 1. 计算基础加权强度
	buyStrength, sellStrength := p.calculateBaseWeightedStrength(signals)

	// 2. 计算信号一致性
	consistency := p.calculateSignalConsistency(signals)

	// 3. 计算整体质量
	quality := p.calculateOverallSignalQuality(signals)

	// 4. 计算整体置信度
	confidence := p.calculateOverallSignalConfidence(signals, consistency)

	// 5. 应用一致性调整
	buyStrength *= consistency
	sellStrength *= consistency

	// 6. 确保强度在合理范围内
	finalBuyStrength := math.Min(buyStrength, 1.0)
	finalSellStrength := math.Min(sellStrength, 1.0)

	return &mean_reversion.SignalStrength{
		BuyStrength:   finalBuyStrength,
		SellStrength:  finalSellStrength,
		Confidence:    math.Min(confidence, 1.0),
		Consistency:   consistency,
		Quality:       quality,
		ActiveSignals: len(signals),
	}, nil
}

// MakeDecision 基于信号强度进行交易决策
func (p *processor) MakeDecision(strength *mean_reversion.SignalStrength, config *mean_reversion.MeanReversionConfig) (*mean_reversion.SignalDecision, error) {
	if strength == nil || config == nil {
		return &mean_reversion.SignalDecision{
			Action: "hold",
			Reason: "信号或配置无效",
		}, nil
	}

	// 应用配置的质量过滤
	if strength.Quality < config.SignalQuality.MinQuality {
		return &mean_reversion.SignalDecision{
			Action:     "hold",
			Strength:   math.Max(strength.BuyStrength, strength.SellStrength),
			Confidence: strength.Confidence,
			Reason:     fmt.Sprintf("信号质量不足%.2f", config.SignalQuality.MinQuality),
		}, nil
	}

	if strength.Consistency < config.SignalQuality.MinConsistency {
		return &mean_reversion.SignalDecision{
			Action:     "hold",
			Strength:   math.Max(strength.BuyStrength, strength.SellStrength),
			Confidence: strength.Confidence,
			Reason:     fmt.Sprintf("信号一致性不足%.2f", config.SignalQuality.MinConsistency),
		}, nil
	}

	// 基础强度检查
	if strength.BuyStrength < config.SignalQuality.MinStrength && strength.SellStrength < config.SignalQuality.MinStrength {
		return &mean_reversion.SignalDecision{
			Action:     "hold",
			Strength:   0.0,
			Confidence: strength.Confidence,
			Reason:     "信号强度不足阈值",
		}, nil
	}

	// 置信度检查
	if strength.Confidence < config.SignalQuality.MinConfidence {
		return &mean_reversion.SignalDecision{
			Action:     "hold",
			Strength:   math.Max(strength.BuyStrength, strength.SellStrength),
			Confidence: strength.Confidence,
			Reason:     fmt.Sprintf("置信度不足%.2f", config.SignalQuality.MinConfidence),
		}, nil
	}

	// 方向比较和阈值检查
	buyValid := strength.BuyStrength >= config.SignalQuality.MinStrength
	sellValid := strength.SellStrength >= config.SignalQuality.MinStrength

	if !buyValid && !sellValid {
		return &mean_reversion.SignalDecision{
			Action:     "hold",
			Strength:   0.0,
			Confidence: strength.Confidence,
			Reason:     "无有效信号",
		}, nil
	}

	// 决定交易方向
	if buyValid && (!sellValid || strength.BuyStrength > strength.SellStrength) {
		return &mean_reversion.SignalDecision{
			Action:     "buy",
			Strength:   strength.BuyStrength,
			Confidence: strength.Confidence,
			Reason:     fmt.Sprintf("买入信号强度%.2f，一致性%.2f", strength.BuyStrength, strength.Consistency),
		}, nil
	} else if sellValid && (!buyValid || strength.SellStrength > strength.BuyStrength) {
		return &mean_reversion.SignalDecision{
			Action:     "sell",
			Strength:   strength.SellStrength,
			Confidence: strength.Confidence,
			Reason:     fmt.Sprintf("卖出信号强度%.2f，一致性%.2f", strength.SellStrength, strength.Consistency),
		}, nil
	}

	// 信号冲突的情况
	return &mean_reversion.SignalDecision{
		Action:     "hold",
		Strength:   0.0,
		Confidence: strength.Confidence,
		Reason:     "信号方向冲突",
	}, nil
}

// calculateBaseWeightedStrength 计算基础加权强度
func (p *processor) calculateBaseWeightedStrength(signals []*mean_reversion.IndicatorSignal) (float64, float64) {
	buyStrength := 0.0
	sellStrength := 0.0
	actualWeight := 0.0

	for _, signal := range signals {
		// 使用动态权重：基础权重 * 质量 * 置信度
		dynamicWeight := signal.BaseWeight * signal.Quality * signal.Confidence
		actualWeight += dynamicWeight

		if signal.BuySignal {
			buyStrength += dynamicWeight
		}
		if signal.SellSignal {
			sellStrength += dynamicWeight
		}
	}

	// 归一化
	if actualWeight > 0 {
		buyStrength /= actualWeight
		sellStrength /= actualWeight
	}

	return buyStrength, sellStrength
}

// calculateSignalConsistency 计算信号一致性
func (p *processor) calculateSignalConsistency(signals []*mean_reversion.IndicatorSignal) float64 {
	if len(signals) <= 1 {
		return 1.0 // 单个信号完全一致
	}

	buyCount := 0
	sellCount := 0
	totalSignals := 0

	for _, signal := range signals {
		if signal.BuySignal {
			buyCount++
			totalSignals++
		}
		if signal.SellSignal {
			sellCount++
			totalSignals++
		}
	}

	if totalSignals == 0 {
		return 0.0
	}

	// 一致性 = 主要方向信号数 / 总信号数
	consistency := math.Max(float64(buyCount), float64(sellCount)) / float64(totalSignals)
	return consistency
}

// calculateOverallSignalQuality 计算整体信号质量
func (p *processor) calculateOverallSignalQuality(signals []*mean_reversion.IndicatorSignal) float64 {
	if len(signals) == 0 {
		return 0.0
	}

	totalQuality := 0.0
	totalWeight := 0.0

	for _, signal := range signals {
		weight := signal.BaseWeight
		totalQuality += signal.Quality * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return totalQuality / totalWeight
}

// calculateOverallSignalConfidence 计算整体信号置信度
func (p *processor) calculateOverallSignalConfidence(signals []*mean_reversion.IndicatorSignal, consistency float64) float64 {
	if len(signals) == 0 {
		return 0.0
	}

	totalConfidence := 0.0
	totalWeight := 0.0

	for _, signal := range signals {
		weight := signal.BaseWeight
		totalConfidence += signal.Confidence * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	baseConfidence := totalConfidence / totalWeight

	// 一致性对置信度的影响：一致性越高，置信度越高
	consistencyBonus := consistency * 0.2 // 最多增加20%的置信度

	return math.Min(baseConfidence+consistencyBonus, 1.0)
}
