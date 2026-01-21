package analysis

import (
	"fmt"
	"math"
)

// MovingAverageType 移动平均线类型
type MovingAverageType string

const (
	SMA MovingAverageType = "SMA" // 简单移动平均线
	EMA MovingAverageType = "EMA" // 指数移动平均线
	WMA MovingAverageType = "WMA" // 加权移动平均线
)

// TechnicalIndicators 技术指标计算器
type TechnicalIndicators struct{}

// NewTechnicalIndicators 创建技术指标计算器
func NewTechnicalIndicators() *TechnicalIndicators {
	return &TechnicalIndicators{}
}

// CalculateMovingAverage 计算移动平均线
func (ti *TechnicalIndicators) CalculateMovingAverage(prices []float64, period int, maType MovingAverageType) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	result := make([]float64, 0, len(prices)-period+1)

	switch maType {
	case SMA:
		result = ti.calculateSMA(prices, period)
	case EMA:
		result = ti.calculateEMA(prices, period)
	case WMA:
		result = ti.calculateWMA(prices, period)
	default:
		result = ti.calculateSMA(prices, period)
	}

	return result
}

// calculateSMA 计算简单移动平均线
func (ti *TechnicalIndicators) calculateSMA(prices []float64, period int) []float64 {
	result := make([]float64, 0, len(prices)-period+1)

	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j]
		}
		result = append(result, sum/float64(period))
	}

	return result
}

// calculateEMA 计算指数移动平均线
func (ti *TechnicalIndicators) calculateEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	result := make([]float64, 0, len(prices)-period+1)
	multiplier := 2.0 / (float64(period) + 1.0)

	// 第一个EMA值使用SMA
	firstSMA := 0.0
	for i := 0; i < period; i++ {
		firstSMA += prices[i]
	}
	firstSMA /= float64(period)
	result = append(result, firstSMA)

	// 计算后续的EMA值
	for i := period; i < len(prices); i++ {
		ema := (prices[i] * multiplier) + (result[len(result)-1] * (1 - multiplier))
		result = append(result, ema)
	}

	return result
}

// calculateWMA 计算加权移动平均线
func (ti *TechnicalIndicators) calculateWMA(prices []float64, period int) []float64 {
	result := make([]float64, 0, len(prices)-period+1)

	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		weightSum := 0.0

		for j := 0; j < period; j++ {
			weight := float64(j + 1) // 线性权重，近期价格权重更高
			sum += prices[i-period+j+1] * weight
			weightSum += weight
		}

		result = append(result, sum/weightSum)
	}

	return result
}

// DetectMACross 检测均线交叉信号
func (ti *TechnicalIndicators) DetectMACross(shortMA, longMA []float64) (goldenCross bool, deathCross bool) {
	if len(shortMA) < 2 || len(longMA) < 2 {
		return false, false
	}

	// 获取最新的两个值
	shortPrev := shortMA[len(shortMA)-2]
	shortCurr := shortMA[len(shortMA)-1]
	longPrev := longMA[len(longMA)-2]
	longCurr := longMA[len(longMA)-1]

	// 检测金叉：短期均线上穿长期均线
	if shortPrev <= longPrev && shortCurr > longCurr {
		goldenCross = true
	}

	// 检测死叉：短期均线下穿长期均线
	if shortPrev >= longPrev && shortCurr < longCurr {
		deathCross = true
	}

	return goldenCross, deathCross
}

// DetectMATrend 检测均线趋势
func (ti *TechnicalIndicators) DetectMATrend(shortMA, longMA []float64) (uptrend bool, downtrend bool) {
	if len(shortMA) == 0 || len(longMA) == 0 {
		return false, false
	}

	// 多头排列：短期均线在长期均线上方
	if shortMA[len(shortMA)-1] > longMA[len(longMA)-1] {
		uptrend = true
	} else {
		downtrend = true
	}

	return uptrend, downtrend
}

// CalculateRSI 计算RSI指标
func (ti *TechnicalIndicators) CalculateRSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return []float64{}
	}

	result := make([]float64, 0, len(prices)-period)

	// 计算价格变化
	changes := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		changes[i-1] = prices[i] - prices[i-1]
	}

	for i := period - 1; i < len(changes); i++ {
		gains := make([]float64, 0)
		losses := make([]float64, 0)

		for j := i - period + 1; j <= i; j++ {
			if changes[j] > 0 {
				gains = append(gains, changes[j])
			} else {
				losses = append(losses, math.Abs(changes[j]))
			}
		}

		avgGain := ti.average(gains)
		avgLoss := ti.average(losses)

		if avgLoss == 0 {
			result = append(result, 100.0)
		} else {
			rs := avgGain / avgLoss
			rsi := 100.0 - (100.0 / (1.0 + rs))
			result = append(result, rsi)
		}
	}

	return result
}

// CalculateMACD 计算MACD指标
func (ti *TechnicalIndicators) CalculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) ([]float64, []float64, []float64) {
	if len(prices) < slowPeriod {
		return []float64{}, []float64{}, []float64{}
	}

	// 计算快线和慢线
	fastEMA := ti.calculateEMA(prices, fastPeriod)
	slowEMA := ti.calculateEMA(prices, slowPeriod)

	// 计算MACD线
	minLen := int(math.Min(float64(len(fastEMA)), float64(len(slowEMA))))
	macdLine := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		macdLine[i] = fastEMA[i] - slowEMA[i]
	}

	// 计算信号线
	signalLine := ti.calculateEMA(macdLine, signalPeriod)

	// 计算柱状图
	minLen = int(math.Min(float64(len(macdLine)), float64(len(signalLine))))
	histogram := make([]float64, minLen)
	for i := 0; i < minLen; i++ {
		histogram[i] = macdLine[i] - signalLine[i]
	}

	return macdLine, signalLine, histogram
}

// average 计算数组平均值
func (ti *TechnicalIndicators) average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// GetKlineDataForSymbol 获取币种的K线数据用于技术指标计算
func (ti *TechnicalIndicators) GetKlineDataForSymbol(symbol string, days int) ([]float64, error) {
	// 这里需要实现从数据库获取K线数据的逻辑
	// 暂时返回空实现，后续完善
	return []float64{}, nil
}

// ValidateMAParameters 验证均线参数
func (ti *TechnicalIndicators) ValidateMAParameters(shortPeriod, longPeriod int) error {
	if shortPeriod >= longPeriod {
		return fmt.Errorf("短期均线周期(%d)必须小于长期均线周期(%d)", shortPeriod, longPeriod)
	}
	if shortPeriod < 2 || longPeriod < 2 {
		return fmt.Errorf("均线周期必须大于等于2")
	}
	if longPeriod > 200 {
		return fmt.Errorf("长期均线周期不能超过200")
	}
	return nil
}

// CalculateBollingerBands 计算布林带
// 返回：上轨、中轨、下轨
func (ti *TechnicalIndicators) CalculateBollingerBands(prices []float64, period int, multiplier float64) ([]float64, []float64, []float64) {
	if len(prices) < period || period < 2 {
		return []float64{}, []float64{}, []float64{}
	}

	// 计算SMA作为中轨
	middle := ti.calculateSMA(prices, period)
	if len(middle) == 0 {
		return []float64{}, []float64{}, []float64{}
	}

	upper := make([]float64, len(middle))
	lower := make([]float64, len(middle))

	for i, ma := range middle {
		// 计算对应价格窗口的标准差
		startIdx := i
		endIdx := i + period
		if endIdx > len(prices) {
			endIdx = len(prices)
		}

		window := prices[startIdx:endIdx]
		stdDev := ti.calculateStandardDeviation(window, ma)

		upper[i] = ma + (stdDev * multiplier)
		lower[i] = ma - (stdDev * multiplier)
	}

	return upper, middle, lower
}

// calculateStandardDeviation 计算标准差
func (ti *TechnicalIndicators) calculateStandardDeviation(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, value := range values {
		diff := value - mean
		sum += diff * diff
	}

	variance := sum / float64(len(values))
	return math.Sqrt(variance)
}
