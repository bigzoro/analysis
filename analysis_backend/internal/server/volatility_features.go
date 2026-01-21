package server

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// VolatilityFeatureExtractor 波动率特征提取器
type VolatilityFeatureExtractor struct {
	config FeatureConfig
}

// Name 返回提取器名称
func (vfe *VolatilityFeatureExtractor) Name() string {
	return "volatility"
}

// Priority 返回提取优先级
func (vfe *VolatilityFeatureExtractor) Priority() int {
	return 85
}

// Extract 提取波动率特征
func (vfe *VolatilityFeatureExtractor) Extract(ctx context.Context, symbol string, currentData *MarketDataPoint, historyData []*MarketDataPoint) (map[string]float64, error) {
	features := make(map[string]float64)

	if len(historyData) < vfe.config.VolatilityWindow {
		return features, fmt.Errorf("历史数据不足，至少需要%d个数据点", vfe.config.VolatilityWindow)
	}

	// 提取价格数据
	prices := make([]float64, len(historyData))
	returns := make([]float64, len(historyData)-1)

	for i, data := range historyData {
		prices[i] = data.Price
		if i > 0 {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	// 1. 基础波动率指标
	vfe.extractBasicVolatilityFeatures(returns, features)

	// 2. 高级波动率指标
	vfe.extractAdvancedVolatilityFeatures(returns, features)

	// 3. 波动率趋势分析
	vfe.extractVolatilityTrendFeatures(returns, features)

	// 4. 波动率风险指标
	vfe.extractVolatilityRiskFeatures(returns, features)

	// 5. 波动率模式识别
	vfe.extractVolatilityPatternFeatures(returns, features)

	return features, nil
}

// extractBasicVolatilityFeatures 提取基础波动率特征
func (vfe *VolatilityFeatureExtractor) extractBasicVolatilityFeatures(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window {
		return
	}

	// 计算不同窗口期的波动率
	windows := []int{window, window * 2, window * 4}
	windowNames := []string{"short", "medium", "long"}

	for i, w := range windows {
		if len(returns) >= w {
			recentReturns := returns[len(returns)-w:]

			// 标准差波动率
			volatility := calculateStandardDeviation(recentReturns, calculateAverage(recentReturns))
			features[fmt.Sprintf("volatility_%s_std", windowNames[i])] = volatility * 100 // 转换为百分比

			// 绝对波动率 (平均绝对收益率)
			absVolatility := 0.0
			for _, ret := range recentReturns {
				absVolatility += math.Abs(ret)
			}
			absVolatility /= float64(len(recentReturns))
			features[fmt.Sprintf("volatility_%s_abs", windowNames[i])] = absVolatility * 100

			// 偏度 (skewness)
			skewness := calculateSkewness(recentReturns)
			features[fmt.Sprintf("volatility_%s_skew", windowNames[i])] = skewness

			// 峰度 (kurtosis)
			kurtosis := calculateKurtosis(recentReturns)
			features[fmt.Sprintf("volatility_%s_kurt", windowNames[i])] = kurtosis
		}
	}

	// 当前波动率水平
	if len(returns) >= window {
		currentVolatility := calculateStandardDeviation(returns[len(returns)-window:], calculateAverage(returns[len(returns)-window:]))
		features["volatility_current_level"] = currentVolatility * 100
	}
}

// extractAdvancedVolatilityFeatures 提取高级波动率特征
func (vfe *VolatilityFeatureExtractor) extractAdvancedVolatilityFeatures(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window*2 {
		return
	}

	// 波动率聚类分析
	vfe.extractVolatilityClusters(returns, features)

	// 波动率自相关
	vfe.extractVolatilityAutocorrelation(returns, features)

	// 波动率波动 (volatility of volatility)
	vfe.extractVolatilityOfVolatility(returns, features)

	// 异常波动检测
	vfe.extractVolatilityOutliers(returns, features)
}

// extractVolatilityClusters 波动率聚类分析
func (vfe *VolatilityFeatureExtractor) extractVolatilityClusters(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window*4 {
		return
	}

	// 计算滚动波动率
	volatilities := make([]float64, 0)
	for i := window; i <= len(returns); i++ {
		segment := returns[i-window : i]
		vol := calculateStandardDeviation(segment, calculateAverage(segment))
		volatilities = append(volatilities, vol)
	}

	// 波动率分位数
	sort.Float64s(volatilities)
	features["volatility_q25"] = volatilities[len(volatilities)/4] * 100
	features["volatility_q50"] = volatilities[len(volatilities)/2] * 100
	features["volatility_q75"] = volatilities[3*len(volatilities)/4] * 100

	// 当前波动率在历史中的位置
	currentVol := volatilities[len(volatilities)-1]
	features["volatility_percentile"] = calculatePercentile(volatilities, currentVol)
}

// extractVolatilityAutocorrelation 波动率自相关分析
func (vfe *VolatilityFeatureExtractor) extractVolatilityAutocorrelation(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window*3 {
		return
	}

	// 计算波动率序列
	volatilities := make([]float64, 0)
	for i := window; i <= len(returns); i++ {
		segment := returns[i-window : i]
		vol := calculateStandardDeviation(segment, calculateAverage(segment))
		volatilities = append(volatilities, vol)
	}

	// 计算自相关系数 (lags 1-3)
	for lag := 1; lag <= 3; lag++ {
		if len(volatilities) > lag {
			corr := calculateCorrelation(volatilities[:len(volatilities)-lag], volatilities[lag:])
			features[fmt.Sprintf("volatility_autocorr_lag%d", lag)] = corr
		}
	}
}

// extractVolatilityOfVolatility 波动率的波动
func (vfe *VolatilityFeatureExtractor) extractVolatilityOfVolatility(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window*4 {
		return
	}

	// 计算波动率序列
	volatilities := make([]float64, 0)
	for i := window; i <= len(returns); i++ {
		segment := returns[i-window : i]
		vol := calculateStandardDeviation(segment, calculateAverage(segment))
		volatilities = append(volatilities, vol)
	}

	// 计算波动率的波动率
	if len(volatilities) >= window {
		recentVols := volatilities[len(volatilities)-window:]
		volOfVol := calculateStandardDeviation(recentVols, calculateAverage(recentVols))
		features["volatility_of_volatility"] = volOfVol * 100

		// 波动率稳定性
		avgVol := calculateAverage(volatilities)
		features["volatility_stability"] = 1.0 / (1.0 + volOfVol/avgVol)
	}
}

// extractVolatilityOutliers 异常波动检测
func (vfe *VolatilityFeatureExtractor) extractVolatilityOutliers(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window*2 {
		return
	}

	// 计算滚动波动率
	volatilities := make([]float64, 0)
	for i := window; i <= len(returns); i++ {
		segment := returns[i-window : i]
		vol := calculateStandardDeviation(segment, calculateAverage(segment))
		volatilities = append(volatilities, vol)
	}

	// 检测异常高的波动率
	if len(volatilities) >= 10 {
		avgVol := calculateAverage(volatilities)
		stdVol := calculateStandardDeviation(volatilities, avgVol)

		currentVol := volatilities[len(volatilities)-1]
		zScore := (currentVol - avgVol) / stdVol

		features["volatility_z_score"] = zScore
		features["volatility_is_extreme"] = boolToFloat(math.Abs(zScore) > 2.0)
		features["volatility_is_high"] = boolToFloat(zScore > 1.0)
		features["volatility_is_low"] = boolToFloat(zScore < -1.0)
	}
}

// extractVolatilityTrendFeatures 提取波动率趋势特征
func (vfe *VolatilityFeatureExtractor) extractVolatilityTrendFeatures(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window*3 {
		return
	}

	// 计算波动率趋势
	volatilities := make([]float64, 0)
	for i := window; i <= len(returns); i++ {
		segment := returns[i-window : i]
		vol := calculateStandardDeviation(segment, calculateAverage(segment))
		volatilities = append(volatilities, vol)
	}

	if len(volatilities) >= 10 {
		// 波动率趋势
		trend := calculateLinearTrend(volatilities[len(volatilities)-10:])
		features["volatility_trend_short"] = trend * 1000 // 放大以便观察

		// 波动率变化率
		if len(volatilities) >= 20 {
			recentTrend := calculateLinearTrend(volatilities[len(volatilities)-10:])
			olderTrend := calculateLinearTrend(volatilities[len(volatilities)-20 : len(volatilities)-10])
			features["volatility_trend_change"] = (recentTrend - olderTrend) * 1000
		}
	}
}

// extractVolatilityRiskFeatures 提取波动率风险特征
func (vfe *VolatilityFeatureExtractor) extractVolatilityRiskFeatures(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window {
		return
	}

	// Value at Risk (VaR) 估算
	recentReturns := returns[len(returns)-window:]
	sort.Float64s(recentReturns)

	// 95% VaR
	varIndex := int(float64(len(recentReturns)) * 0.05)
	if varIndex < len(recentReturns) {
		features["value_at_risk_95"] = -recentReturns[varIndex] * 100 // 负号表示损失
	}

	// 99% VaR
	varIndex99 := int(float64(len(recentReturns)) * 0.01)
	if varIndex99 < len(recentReturns) {
		features["value_at_risk_99"] = -recentReturns[varIndex99] * 100
	}

	// Expected Shortfall (CVaR)
	if varIndex < len(recentReturns) {
		tailLosses := recentReturns[:varIndex+1]
		avgTailLoss := 0.0
		for _, loss := range tailLosses {
			avgTailLoss += loss
		}
		avgTailLoss /= float64(len(tailLosses))
		features["expected_shortfall_95"] = -avgTailLoss * 100
	}

	// 最大回撤风险
	if len(returns) >= window {
		maxDrawdown := calculateMaxDrawdown(returns[len(returns)-window:])
		features["max_drawdown_risk"] = maxDrawdown * 100
	}
}

// extractVolatilityPatternFeatures 提取波动率模式特征
func (vfe *VolatilityFeatureExtractor) extractVolatilityPatternFeatures(returns []float64, features map[string]float64) {
	window := vfe.config.VolatilityWindow

	if len(returns) < window*2 {
		return
	}

	// 计算波动率序列
	volatilities := make([]float64, 0)
	for i := window; i <= len(returns); i++ {
		segment := returns[i-window : i]
		vol := calculateStandardDeviation(segment, calculateAverage(segment))
		volatilities = append(volatilities, vol)
	}

	if len(volatilities) >= 10 {
		// 波动率周期性检测
		features["volatility_has_pattern"] = vfe.detectVolatilityPattern(volatilities)

		// 波动率聚集性 (volatility clustering)
		highVolPeriods := 0
		threshold := calculateAverage(volatilities) + calculateStandardDeviation(volatilities, calculateAverage(volatilities))

		for _, vol := range volatilities[len(volatilities)-10:] {
			if vol > threshold {
				highVolPeriods++
			}
		}

		features["volatility_clustering_ratio"] = float64(highVolPeriods) / 10.0
	}
}

// detectVolatilityPattern 检测波动率模式
func (vfe *VolatilityFeatureExtractor) detectVolatilityPattern(volatilities []float64) float64 {
	if len(volatilities) < 20 {
		return 0.0
	}

	// 简单的模式检测：检查是否存在周期性
	autocorr1 := calculateCorrelation(volatilities[:len(volatilities)-1], volatilities[1:])
	autocorr2 := calculateCorrelation(volatilities[:len(volatilities)-2], volatilities[2:])

	// 如果自相关系数显著，则认为存在模式
	patternStrength := math.Max(math.Abs(autocorr1), math.Abs(autocorr2))

	if patternStrength > 0.3 {
		return patternStrength
	}

	return 0.0
}

// 辅助函数

func calculateSkewness(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	mean := calculateAverage(values)
	std := calculateStandardDeviation(values, mean)

	sum := 0.0
	for _, v := range values {
		diff := (v - mean) / std
		sum += diff * diff * diff
	}

	return sum / float64(len(values))
}

func calculateKurtosis(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	mean := calculateAverage(values)
	std := calculateStandardDeviation(values, mean)

	sum := 0.0
	for _, v := range values {
		diff := (v - mean) / std
		sum += diff * diff * diff * diff
	}

	return (sum / float64(len(values))) - 3 // 减去正态分布的峰度
}

func calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) == 0 {
		return 0
	}

	meanX := calculateAverage(x)
	meanY := calculateAverage(y)

	numerator := 0.0
	sumXX := 0.0
	sumYY := 0.0

	for i := 0; i < len(x); i++ {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		sumXX += dx * dx
		sumYY += dy * dy
	}

	if sumXX == 0 || sumYY == 0 {
		return 0
	}

	return numerator / math.Sqrt(sumXX*sumYY)
}

func calculatePercentile(values []float64, value float64) float64 {
	if len(values) == 0 {
		return 0
	}

	count := 0
	for _, v := range values {
		if v <= value {
			count++
		}
	}

	return float64(count) / float64(len(values)) * 100
}

func calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 计算累积收益率
	cumulative := make([]float64, len(returns)+1)
	cumulative[0] = 1.0

	for i, ret := range returns {
		cumulative[i+1] = cumulative[i] * (1 + ret)
	}

	// 找到最大回撤
	maxDrawdown := 0.0
	peak := cumulative[0]

	for _, cum := range cumulative[1:] {
		if cum > peak {
			peak = cum
		} else {
			drawdown := (peak - cum) / peak
			if drawdown > maxDrawdown {
				maxDrawdown = drawdown
			}
		}
	}

	return maxDrawdown
}
