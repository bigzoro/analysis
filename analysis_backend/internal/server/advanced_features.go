package server

import (
	"math"
	"sort"
)

// ===== 高级特征计算函数 =====

// calculateSeriesNonlinearity 计算价格序列的非线性度
func (fee *FeatureEngineeringExtractor) calculateSeriesNonlinearity(prices []float64) float64 {
	if len(prices) < 10 {
		return 0.0
	}

	// 计算线性趋势残差的方差 vs 原始价格变动的方差
	n := float64(len(prices))

	// 计算线性回归
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, price := range prices {
		x := float64(i)
		sumX += x
		sumY += price
		sumXY += x * price
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// 计算残差
	residualVariance := 0.0
	priceVariance := 0.0
	meanPrice := sumY / n

	for i, actualPrice := range prices {
		x := float64(i)
		predictedPrice := slope*x + intercept
		residual := actualPrice - predictedPrice
		residualVariance += residual * residual

		priceVariance += (actualPrice - meanPrice) * (actualPrice - meanPrice)
	}

	residualVariance /= n
	priceVariance /= n

	if priceVariance == 0 {
		return 0.0
	}

	// 非线性度 = 残差方差 / 总方差
	nonlinearity := residualVariance / priceVariance

	// 标准化到0-1范围
	return math.Min(1.0, nonlinearity)
}

// calculateMeanMedianDifference 计算均值和中位数的差异（分布偏度指标）
func (fee *FeatureEngineeringExtractor) calculateMeanMedianDifference(prices []float64) float64 {
	if len(prices) < 5 {
		return 0.0
	}

	// 计算均值
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	mean := sum / float64(len(prices))

	// 计算中位数
	sortedPrices := make([]float64, len(prices))
	copy(sortedPrices, prices)
	sort.Float64s(sortedPrices)
	var median float64
	if len(sortedPrices)%2 == 0 {
		median = (sortedPrices[len(sortedPrices)/2-1] + sortedPrices[len(sortedPrices)/2]) / 2
	} else {
		median = sortedPrices[len(sortedPrices)/2]
	}

	// 计算相对差异
	if mean == 0 {
		return 0.0
	}

	diff := (mean - median) / mean

	// 标准化到-1到1范围
	return math.Max(-1.0, math.Min(1.0, diff))
}

// calculateMultiScaleVolatility 计算多时间尺度的波动率
func (fee *FeatureEngineeringExtractor) calculateMultiScaleVolatility(prices []float64) float64 {
	if len(prices) < 60 {
		return 0.0
	}

	// 计算不同周期的波动率
	shortVol := fee.calculateVolatility(prices[len(prices)-20:], 1) // 短期波动率
	mediumVol := fee.calculateVolatility(prices[len(prices)-40:], 1) // 中期波动率
	longVol := fee.calculateVolatility(prices[len(prices)-60:], 1)   // 长期波动率

	// 加权平均：短期权重更高
	volatility := 0.5*shortVol + 0.3*mediumVol + 0.2*longVol

	// 标准化（假设正常波动率范围是0.01-0.1）
	normalizedVol := math.Min(1.0, volatility/0.05)

	return normalizedVol
}

// calculateVolatility 计算价格序列的波动率
func (fee *FeatureEngineeringExtractor) calculateVolatility(prices []float64, periods int) float64 {
	if len(prices) < periods+1 {
		return 0.0
	}

	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	// 计算标准差
	sum := 0.0
	for _, ret := range returns {
		sum += ret
	}
	mean := sum / float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		variance += (ret - mean) * (ret - mean)
	}
	variance /= float64(len(returns))

	return math.Sqrt(variance)
}

// calculateRSI 计算相对强弱指数
func (fee *FeatureEngineeringExtractor) calculateRSI(prices []float64) float64 {
	if len(prices) < 14 {
		return 50.0
	}

	gains := 0.0
	losses := 0.0

	// 计算初始值
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / 14.0
	avgLoss := losses / 14.0

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))

	return rsi
}

// calculateMACD 计算MACD指标
func (fee *FeatureEngineeringExtractor) calculateMACD(prices []float64) (float64, float64, float64) {
	if len(prices) < 26 {
		return 0.0, 0.0, 0.0
	}

	// 计算EMA12
	ema12 := fee.calculateEMA(prices, 12)

	// 计算EMA26
	ema26 := fee.calculateEMA(prices, 26)

	// MACD线
	macd := ema12 - ema26

	// 信号线 (MACD的9日EMA)
	macdHistory := make([]float64, len(prices)-25) // 从第26天开始有MACD值
	for i := 25; i < len(prices); i++ {
		ema12_i := fee.calculateEMA(prices[:i+1], 12)
		ema26_i := fee.calculateEMA(prices[:i+1], 26)
		macdHistory[i-25] = ema12_i - ema26_i
	}

	signal := fee.calculateEMA(macdHistory, 9)

	// 柱状图
	histogram := macd - signal

	return macd, signal, histogram
}

// calculateEMA 计算指数移动平均
func (fee *FeatureEngineeringExtractor) calculateEMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return prices[len(prices)-1]
	}

	multiplier := 2.0 / (float64(period) + 1.0)
	ema := prices[0]

	for i := 1; i < len(prices); i++ {
		ema = (prices[i] - ema) * multiplier + ema
	}

	return ema
}

// calculateBollingerPosition 计算布林带位置
func (fee *FeatureEngineeringExtractor) calculateBollingerPosition(prices []float64, currentPrice float64) float64 {
	if len(prices) < 20 {
		return 0.5
	}

	// 计算20日简单移动平均
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	sma := sum / float64(len(prices))

	// 计算标准差
	variance := 0.0
	for _, price := range prices {
		variance += (price - sma) * (price - sma)
	}
	variance /= float64(len(prices))
	stdDev := math.Sqrt(variance)

	upperBand := sma + 2*stdDev
	lowerBand := sma - 2*stdDev

	// 计算当前位置 (0-1之间)
	if upperBand == lowerBand {
		return 0.5
	}

	position := (currentPrice - lowerBand) / (upperBand - lowerBand)
	return math.Max(0.0, math.Min(1.0, position))
}

// calculateMarketSentiment 计算市场情绪指标
func (fee *FeatureEngineeringExtractor) calculateMarketSentiment(prices []float64, currentPrice float64) float64 {
	if len(prices) < 30 {
		return 0.5
	}

	// 1. 动量分数
	recentPrices := prices[len(prices)-10:]
	momentumScore := 0.0
	for i := 1; i < len(recentPrices); i++ {
		if recentPrices[i] > recentPrices[i-1] {
			momentumScore += 1.0
		} else {
			momentumScore -= 1.0
		}
	}
	momentumScore /= float64(len(recentPrices)-1) // -1到1之间

	// 2. 波动率分数
	volatility := fee.calculateVolatility(prices[len(prices)-20:], 1)
	volatilityScore := math.Min(1.0, volatility/0.03) // 标准化波动率

	// 3. 价格位置分数 (相对于历史区间)
	minPrice, maxPrice := fee.findMinMaxPrices(prices)
	var positionScore float64
	if maxPrice > minPrice {
		positionScore = (currentPrice - minPrice) / (maxPrice - minPrice)
	} else {
		positionScore = 0.5
	}

	// 综合情绪分数：动量权重0.4，波动率权重0.3，位置权重0.3
	sentiment := 0.4*(momentumScore+1.0)/2.0 + 0.3*(1.0-volatilityScore) + 0.3*positionScore

	return math.Max(0.0, math.Min(1.0, sentiment))
}

// calculateVolumePriceTrend 计算成交量价格趋势
func (fee *FeatureEngineeringExtractor) calculateVolumePriceTrend(prices []float64, currentData *MarketDataPoint) float64 {
	if len(prices) < 20 {
		return 0.0
	}

	vpt := 0.0
	baseVolume := 100000.0 // 基准成交量

	for i := 1; i < len(prices); i++ {
		priceChange := (prices[i] - prices[i-1]) / prices[i-1]
		volumeRatio := currentData.Volume24h / baseVolume

		vpt += priceChange * math.Min(volumeRatio, 5.0) // 限制成交量影响
	}

	return vpt / float64(len(prices)-1) // 标准化
}

// calculatePriceChannelPosition 计算价格通道位置
func (fee *FeatureEngineeringExtractor) calculatePriceChannelPosition(prices []float64, currentPrice float64) float64 {
	if len(prices) < 20 {
		return 0.5
	}

	// 计算20日最高价和最低价通道
	minPrice, maxPrice := fee.findMinMaxPrices(prices)

	if maxPrice == minPrice {
		return 0.5
	}

	position := (currentPrice - minPrice) / (maxPrice - minPrice)
	return math.Max(0.0, math.Min(1.0, position))
}
