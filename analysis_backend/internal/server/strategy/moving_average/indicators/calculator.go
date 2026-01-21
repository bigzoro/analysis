package indicators

import (
	"analysis/internal/server/strategy/moving_average"
	"math"
)

// Calculator 技术指标计算器实现
type Calculator struct{}

// NewCalculator 创建指标计算器
func NewCalculator() moving_average.IndicatorCalculator {
	return &Calculator{}
}

// CalculateSMA 计算简单移动平均线
func (c *Calculator) CalculateSMA(prices []float64, period int) []float64 {
	if len(prices) < period || period <= 0 {
		return []float64{}
	}

	sma := make([]float64, len(prices)-period+1)

	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j]
		}
		sma[i-period+1] = sum / float64(period)
	}

	return sma
}

// CalculateEMA 计算指数移动平均线
func (c *Calculator) CalculateEMA(prices []float64, period int) []float64 {
	if len(prices) < period || period <= 0 {
		return []float64{}
	}

	ema := make([]float64, len(prices))
	multiplier := 2.0 / (float64(period) + 1.0)

	// 第一个EMA值使用SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	ema[period-1] = sum / float64(period)

	// 计算后续的EMA值
	for i := period; i < len(prices); i++ {
		ema[i] = (prices[i] * multiplier) + (ema[i-1] * (1 - multiplier))
	}

	// 返回有效部分（从第period个元素开始）
	return ema[period-1:]
}

// DetectCross 检测交叉点
func (c *Calculator) DetectCross(shortMA, longMA []float64) (crosses []moving_average.CrossInfo) {
	if len(shortMA) != len(longMA) || len(shortMA) < 2 {
		return
	}

	for i := 1; i < len(shortMA); i++ {
		prevShort := shortMA[i-1]
		prevLong := longMA[i-1]
		currShort := shortMA[i]
		currLong := longMA[i]

		// 检测金叉：短期均线上穿长期均线
		if prevShort <= prevLong && currShort > currLong {
			strength := c.CalculateCrossStrength(shortMA, longMA, i)
			crosses = append(crosses, moving_average.CrossInfo{
				Index:    i,
				Type:     "golden",
				Price:    currShort, // 使用交叉点的短期均线值作为参考价格
				Strength: strength,
			})
		}

		// 检测死叉：短期均线下穿长期均线
		if prevShort >= prevLong && currShort < currLong {
			strength := c.CalculateCrossStrength(shortMA, longMA, i)
			crosses = append(crosses, moving_average.CrossInfo{
				Index:    i,
				Type:     "death",
				Price:    currShort, // 使用交叉点的短期均线值作为参考价格
				Strength: strength,
			})
		}
	}

	return
}

// CalculateCrossStrength 计算交叉强度
func (c *Calculator) CalculateCrossStrength(shortMA, longMA []float64, crossIndex int) float64 {
	if crossIndex >= len(shortMA) || crossIndex >= len(longMA) {
		return 0
	}

	shortValue := shortMA[crossIndex]
	longValue := longMA[crossIndex]

	if longValue == 0 {
		return 0
	}

	// 交叉强度 = |短期均线 - 长期均线| / 长期均线
	strength := math.Abs(shortValue-longValue) / longValue

	// 归一化到0-1范围（强度过高时 capping）
	if strength > 1.0 {
		strength = 1.0
	}

	return strength
}
