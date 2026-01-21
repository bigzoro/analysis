package indicators

import (
	"analysis/internal/server/strategy/mean_reversion"
	"fmt"
	"math"

	"analysis/internal/analysis"
)

// Factory 指标工厂实现
type factory struct {
	technicalIndicators *analysis.TechnicalIndicators
}

// NewMRIndicatorFactory 创建均值回归指标工厂
func NewMRIndicatorFactory() mean_reversion.MRIndicatorFactory {
	return &factory{
		technicalIndicators: analysis.NewTechnicalIndicators(),
	}
}

// Create 创建指定类型的指标
func (f *factory) Create(name string, params map[string]interface{}) (mean_reversion.MRIndicator, error) {
	switch name {
	case "bollinger":
		return NewBollingerIndicator(f.technicalIndicators, params)
	case "rsi":
		return NewRSIIndicator(f.technicalIndicators, params)
	case "price_channel":
		return NewPriceChannelIndicator(f.technicalIndicators, params)
	default:
		return nil, fmt.Errorf("未知指标类型: %s", name)
	}
}

// GetAvailableIndicators 获取可用的指标列表
func (f *factory) GetAvailableIndicators() []string {
	return []string{"bollinger", "rsi", "price_channel"}
}

// BollingerIndicator 布林带指标实现
type BollingerIndicator struct {
	ti         *analysis.TechnicalIndicators
	multiplier float64
	period     int
}

func NewBollingerIndicator(ti *analysis.TechnicalIndicators, params map[string]interface{}) (mean_reversion.MRIndicator, error) {
	indicator := &BollingerIndicator{
		ti: ti,
	}

	// 解析参数
	if multiplier, ok := params["multiplier"].(float64); ok {
		indicator.multiplier = multiplier
	} else {
		indicator.multiplier = 2.0
	}

	if period, ok := params["period"].(int); ok {
		indicator.period = period
	} else {
		indicator.period = 20
	}

	return indicator, nil
}

func (bi *BollingerIndicator) Name() string {
	return "bollinger"
}

func (bi *BollingerIndicator) Calculate(prices []float64, params map[string]interface{}) (mean_reversion.IndicatorSignal, error) {
	// 更新参数（如果提供）
	if multiplier, ok := params["multiplier"].(float64); ok {
		bi.multiplier = multiplier
	}
	if period, ok := params["period"].(int); ok {
		bi.period = period
	}

	upper, _, lower := bi.ti.CalculateBollingerBands(prices, bi.period, bi.multiplier)
	if len(upper) == 0 || len(lower) == 0 {
		return mean_reversion.IndicatorSignal{Type: "bollinger"}, fmt.Errorf("布林带计算失败")
	}

	currentPrice := prices[len(prices)-1]
	currentUpper := upper[len(upper)-1]
	currentLower := lower[len(lower)-1]

	buySignal := currentPrice <= currentLower
	sellSignal := currentPrice >= currentUpper

	// 计算质量和置信度
	quality := bi.calculateQuality(currentPrice, currentUpper, currentLower)
	confidence := bi.calculateConfidence(prices, bi.period, currentPrice, currentUpper, currentLower)

	return mean_reversion.IndicatorSignal{
		Type:       "bollinger",
		BuySignal:  buySignal,
		SellSignal: sellSignal,
		Quality:    quality,
		Confidence: confidence,
	}, nil
}

func (bi *BollingerIndicator) GetDefaultParams() map[string]interface{} {
	return map[string]interface{}{
		"period":     20,
		"multiplier": 2.0,
	}
}

func (bi *BollingerIndicator) ValidateParams(params map[string]interface{}) error {
	if period, ok := params["period"].(int); ok && period <= 0 {
		return fmt.Errorf("period必须大于0")
	}
	if multiplier, ok := params["multiplier"].(float64); ok && multiplier <= 0 {
		return fmt.Errorf("multiplier必须大于0")
	}
	return nil
}

func (bi *BollingerIndicator) calculateQuality(currentPrice, upper, lower float64) float64 {
	if upper <= lower {
		return 0.0
	}

	middle := (upper + lower) / 2.0
	maxDeviation := upper - middle

	if currentPrice > middle {
		deviation := math.Abs(currentPrice - middle)
		if currentPrice >= upper {
			return 1.0
		}
		return math.Min(deviation/maxDeviation, 1.0)
	} else {
		deviation := math.Abs(currentPrice - middle)
		if currentPrice <= lower {
			return 1.0
		}
		return math.Min(deviation/maxDeviation, 1.0)
	}
}

func (bi *BollingerIndicator) calculateConfidence(prices []float64, period int, currentPrice, upper, lower float64) float64 {
	if len(prices) < period*2 {
		return 0.5
	}

	historicalSignals := 0
	totalTests := 0

	for i := period; i < len(prices)-period; i += period / 4 {
		histPrice := prices[i]
		totalTests++
		if histPrice <= lower || histPrice >= upper {
			historicalSignals++
		}
	}

	if totalTests == 0 {
		return 0.5
	}

	historicalFrequency := float64(historicalSignals) / float64(totalTests)

	if currentPrice <= lower || currentPrice >= upper {
		if historicalFrequency > 0.1 && historicalFrequency < 0.5 {
			return 0.8
		} else if historicalFrequency > 0.05 {
			return 0.6
		}
	}

	return 0.4
}

// RSIIndicator RSI指标实现
type RSIIndicator struct {
	ti         *analysis.TechnicalIndicators
	overbought int
	oversold   int
	period     int
}

func NewRSIIndicator(ti *analysis.TechnicalIndicators, params map[string]interface{}) (mean_reversion.MRIndicator, error) {
	indicator := &RSIIndicator{
		ti:     ti,
		period: 14, // RSI默认周期
	}

	if overbought, ok := params["overbought"].(int); ok {
		indicator.overbought = overbought
	} else {
		indicator.overbought = 70
	}

	if oversold, ok := params["oversold"].(int); ok {
		indicator.oversold = oversold
	} else {
		indicator.oversold = 30
	}

	return indicator, nil
}

func (ri *RSIIndicator) Name() string {
	return "rsi"
}

func (ri *RSIIndicator) Calculate(prices []float64, params map[string]interface{}) (mean_reversion.IndicatorSignal, error) {
	// 更新参数
	if overbought, ok := params["overbought"].(int); ok {
		ri.overbought = overbought
	}
	if oversold, ok := params["oversold"].(int); ok {
		ri.oversold = oversold
	}

	rsi := ri.ti.CalculateRSI(prices, ri.period)
	if len(rsi) == 0 {
		return mean_reversion.IndicatorSignal{Type: "rsi"}, fmt.Errorf("RSI计算失败")
	}

	currentRSI := rsi[len(rsi)-1]

	buySignal := currentRSI <= float64(ri.oversold)
	sellSignal := currentRSI >= float64(ri.overbought)

	quality := ri.calculateQuality(currentRSI, float64(ri.overbought), float64(ri.oversold))
	confidence := ri.calculateConfidence(rsi, float64(ri.overbought), float64(ri.oversold))

	return mean_reversion.IndicatorSignal{
		Type:       "rsi",
		BuySignal:  buySignal,
		SellSignal: sellSignal,
		Quality:    quality,
		Confidence: confidence,
	}, nil
}

func (ri *RSIIndicator) GetDefaultParams() map[string]interface{} {
	return map[string]interface{}{
		"overbought": 70,
		"oversold":   30,
		"period":     14,
	}
}

func (ri *RSIIndicator) ValidateParams(params map[string]interface{}) error {
	if overbought, ok := params["overbought"].(int); ok && (overbought <= 0 || overbought >= 100) {
		return fmt.Errorf("overbought必须在1-99之间")
	}
	if oversold, ok := params["oversold"].(int); ok && (oversold <= 0 || oversold >= 100) {
		return fmt.Errorf("oversold必须在1-99之间")
	}
	oversold, okOversold := params["oversold"].(int)
	overbought, okOverbought := params["overbought"].(int)
	if okOversold && okOverbought && oversold >= overbought {
		return fmt.Errorf("oversold必须小于overbought")
	}
	return nil
}

func (ri *RSIIndicator) calculateQuality(currentRSI, overbought, oversold float64) float64 {
	if currentRSI <= oversold {
		deviation := oversold - currentRSI
		maxDeviation := oversold - 50.0
		return math.Min(deviation/maxDeviation, 1.0)
	} else if currentRSI >= overbought {
		deviation := currentRSI - overbought
		maxDeviation := 50.0 - overbought
		return math.Min(deviation/maxDeviation, 1.0)
	}
	return 0.0
}

func (ri *RSIIndicator) calculateConfidence(rsi []float64, overbought, oversold float64) float64 {
	if len(rsi) < 20 {
		return 0.5
	}

	signalCount := 0
	totalBars := len(rsi)

	for _, rsiValue := range rsi {
		if rsiValue <= oversold || rsiValue >= overbought {
			signalCount++
		}
	}

	signalFrequency := float64(signalCount) / float64(totalBars)

	if signalFrequency >= 0.05 && signalFrequency <= 0.20 {
		return 0.8
	} else if signalFrequency >= 0.02 && signalFrequency <= 0.30 {
		return 0.6
	}

	return 0.3
}

// PriceChannelIndicator 价格通道指标实现
type PriceChannelIndicator struct {
	ti     *analysis.TechnicalIndicators
	period int
}

func NewPriceChannelIndicator(ti *analysis.TechnicalIndicators, params map[string]interface{}) (mean_reversion.MRIndicator, error) {
	indicator := &PriceChannelIndicator{
		ti: ti,
	}

	if period, ok := params["period"].(int); ok {
		indicator.period = period
	} else {
		indicator.period = 20
	}

	return indicator, nil
}

func (pci *PriceChannelIndicator) Name() string {
	return "price_channel"
}

func (pci *PriceChannelIndicator) Calculate(prices []float64, params map[string]interface{}) (mean_reversion.IndicatorSignal, error) {
	if period, ok := params["period"].(int); ok {
		pci.period = period
	}

	if len(prices) < pci.period {
		return mean_reversion.IndicatorSignal{Type: "price_channel"}, fmt.Errorf("价格数据不足")
	}

	// 计算价格通道（最高价和最低价的移动平均）
	channelHigh, channelLow := pci.calculatePriceChannel(prices, pci.period)

	currentPrice := prices[len(prices)-1]

	buySignal := currentPrice <= channelLow
	sellSignal := currentPrice >= channelHigh

	return mean_reversion.IndicatorSignal{
		Type:       "price_channel",
		BuySignal:  buySignal,
		SellSignal: sellSignal,
		Quality:    0.5, // 简化的质量评估
		Confidence: 0.5, // 简化的置信度评估
	}, nil
}

func (pci *PriceChannelIndicator) GetDefaultParams() map[string]interface{} {
	return map[string]interface{}{
		"period": 20,
	}
}

func (pci *PriceChannelIndicator) ValidateParams(params map[string]interface{}) error {
	if period, ok := params["period"].(int); ok && period <= 0 {
		return fmt.Errorf("period必须大于0")
	}
	return nil
}

func (pci *PriceChannelIndicator) calculatePriceChannel(prices []float64, period int) (float64, float64) {
	if len(prices) < period {
		return 0, 0
	}

	// 计算最近period个周期的最高价和最低价
	high := prices[len(prices)-period]
	low := prices[len(prices)-period]

	for i := len(prices) - period + 1; i < len(prices); i++ {
		if prices[i] > high {
			high = prices[i]
		}
		if prices[i] < low {
			low = prices[i]
		}
	}

	return high, low
}
