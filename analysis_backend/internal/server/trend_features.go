package server

import (
	"context"
	"fmt"
	"math"
)

// TrendFeatureExtractor 趋势特征提取器
type TrendFeatureExtractor struct {
	config FeatureConfig
}

// Name 返回提取器名称
func (tfe *TrendFeatureExtractor) Name() string {
	return "trend"
}

// Priority 返回提取优先级
func (tfe *TrendFeatureExtractor) Priority() int {
	return 80
}

// Extract 提取趋势特征
func (tfe *TrendFeatureExtractor) Extract(ctx context.Context, symbol string, currentData *MarketDataPoint, historyData []*MarketDataPoint) (map[string]float64, error) {
	features := make(map[string]float64)

	if len(historyData) < tfe.config.TrendWindow {
		return features, fmt.Errorf("历史数据不足，至少需要%d个数据点", tfe.config.TrendWindow)
	}

	// 提取价格数据
	prices := make([]float64, len(historyData))
	volumes := make([]float64, len(historyData))

	for i, data := range historyData {
		prices[i] = data.Price
		volumes[i] = data.Volume24h
	}

	// 1. 基础趋势指标
	tfe.extractBasicTrendFeatures(prices, features)

	// 2. 高级趋势指标
	tfe.extractAdvancedTrendFeatures(prices, features)

	// 3. 趋势强度分析
	tfe.extractTrendStrengthFeatures(prices, volumes, features)

	// 4. 趋势转折点检测
	tfe.extractTrendReversalFeatures(prices, features)

	// 5. 多时间尺度趋势分析
	tfe.extractMultiTimeframeTrendFeatures(prices, features)

	return features, nil
}

// extractBasicTrendFeatures 提取基础趋势特征
func (tfe *TrendFeatureExtractor) extractBasicTrendFeatures(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window {
		return
	}

	// 简单移动平均趋势
	tfe.extractMovingAverageTrends(prices, features)

	// 线性回归趋势
	tfe.extractLinearRegressionTrends(prices, features)

	// 价格位置趋势
	tfe.extractPricePositionTrends(prices, features)
}

// extractMovingAverageTrends 提取移动平均趋势
func (tfe *TrendFeatureExtractor) extractMovingAverageTrends(prices []float64, features map[string]float64) {
	n := len(prices)

	// 计算不同周期的移动平均
	periods := []int{5, 10, 20, 50}
	periodNames := []string{"5", "10", "20", "50"}

	for i, period := range periods {
		if n >= period {
			ma := calculateSimpleMovingAverage(prices[n-period:], period)
			features[fmt.Sprintf("ma_%s", periodNames[i])] = ma

			// 价格相对移动平均的位置
			currentPrice := prices[n-1]
			features[fmt.Sprintf("price_vs_ma_%s", periodNames[i])] = (currentPrice - ma) / ma * 100
		}
	}

	// 移动平均排列
	if n >= 50 {
		ma5 := calculateSimpleMovingAverage(prices[n-5:], 5)
		ma10 := calculateSimpleMovingAverage(prices[n-10:], 10)
		ma20 := calculateSimpleMovingAverage(prices[n-20:], 20)
		ma50 := calculateSimpleMovingAverage(prices[n-50:], 50)

		// 多头排列强度 (0-1)
		bullishAlignment := 0.0
		if ma5 > ma10 {
			bullishAlignment += 0.25
		}
		if ma10 > ma20 {
			bullishAlignment += 0.25
		}
		if ma20 > ma50 {
			bullishAlignment += 0.25
		}
		if ma5 > ma20 {
			bullishAlignment += 0.25
		}

		features["ma_bullish_alignment"] = bullishAlignment
		features["ma_bearish_alignment"] = 1.0 - bullishAlignment
	}
}

// extractLinearRegressionTrends 提取线性回归趋势
func (tfe *TrendFeatureExtractor) extractLinearRegressionTrends(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window {
		return
	}

	// 计算不同窗口的趋势斜率和R²
	windows := []int{window / 4, window / 2, window}
	windowNames := []string{"short", "medium", "long"}

	for i, w := range windows {
		if len(prices) >= w {
			recentPrices := prices[len(prices)-w:]

			slope, intercept, r2 := calculateLinearRegression(recentPrices)

			features[fmt.Sprintf("trend_slope_%s", windowNames[i])] = slope * 1000 // 放大便于观察
			features[fmt.Sprintf("trend_intercept_%s", windowNames[i])] = intercept
			features[fmt.Sprintf("trend_r2_%s", windowNames[i])] = r2

			// 趋势强度 (基于R²和斜率绝对值)
			trendStrength := r2 * math.Abs(slope) / calculateAverage(recentPrices) * 100
			features[fmt.Sprintf("trend_strength_%s", windowNames[i])] = trendStrength

			// 趋势方向
			if slope > 0 {
				features[fmt.Sprintf("trend_direction_%s", windowNames[i])] = 1.0 // 上涨
			} else if slope < 0 {
				features[fmt.Sprintf("trend_direction_%s", windowNames[i])] = -1.0 // 下跌
			} else {
				features[fmt.Sprintf("trend_direction_%s", windowNames[i])] = 0.0 // 横盘
			}
		}
	}
}

// extractPricePositionTrends 提取价格位置趋势
func (tfe *TrendFeatureExtractor) extractPricePositionTrends(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window {
		return
	}

	currentPrice := prices[len(prices)-1]

	// 价格在历史区间中的位置
	minPrice := findMin(prices[len(prices)-window:])
	maxPrice := findMax(prices[len(prices)-window:])
	priceRange := maxPrice - minPrice

	if priceRange > 0 {
		position := (currentPrice - minPrice) / priceRange
		features["price_position_in_range"] = position

		// 极端位置检测
		features["price_near_low"] = boolToFloat(position < 0.2)
		features["price_near_high"] = boolToFloat(position > 0.8)
		features["price_at_mid"] = boolToFloat(position >= 0.4 && position <= 0.6)
	}

	// 价格相对历史均值的位置
	meanPrice := calculateAverage(prices[len(prices)-window:])
	stdDev := calculateStandardDeviation(prices[len(prices)-window:], meanPrice)

	if stdDev > 0 {
		zScore := (currentPrice - meanPrice) / stdDev
		features["price_z_score"] = zScore

		// 价格偏离程度
		features["price_extreme_high"] = boolToFloat(zScore > 2.0)
		features["price_extreme_low"] = boolToFloat(zScore < -2.0)
		features["price_normal_range"] = boolToFloat(math.Abs(zScore) <= 1.0)
	}
}

// extractAdvancedTrendFeatures 提取高级趋势特征
func (tfe *TrendFeatureExtractor) extractAdvancedTrendFeatures(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window {
		return
	}

	// 趋势一致性分析
	tfe.extractTrendConsistency(prices, features)

	// 趋势加速分析
	tfe.extractTrendAcceleration(prices, features)

	// 趋势周期性分析
	tfe.extractTrendPeriodicity(prices, features)
}

// extractTrendConsistency 趋势一致性分析
func (tfe *TrendFeatureExtractor) extractTrendConsistency(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window*2 {
		return
	}

	// 计算多个子窗口的趋势方向
	subWindows := []int{window / 4, window / 2, window}
	directions := make([]float64, len(subWindows))

	for i, w := range subWindows {
		if len(prices) >= w {
			recentPrices := prices[len(prices)-w:]
			slope, _, _ := calculateLinearRegression(recentPrices)

			if slope > 0 {
				directions[i] = 1.0
			} else if slope < 0 {
				directions[i] = -1.0
			} else {
				directions[i] = 0.0
			}
		}
	}

	// 计算趋势一致性 (所有子趋势方向相同)
	consistency := 0.0
	if len(directions) > 1 {
		firstDirection := directions[0]
		consistentCount := 0

		for _, dir := range directions[1:] {
			if dir == firstDirection {
				consistentCount++
			}
		}

		consistency = float64(consistentCount) / float64(len(directions)-1)
	}

	features["trend_consistency"] = consistency
	features["trend_consistent_up"] = boolToFloat(consistency > 0.7 && directions[0] == 1.0)
	features["trend_consistent_down"] = boolToFloat(consistency > 0.7 && directions[0] == -1.0)
}

// extractTrendAcceleration 趋势加速分析
func (tfe *TrendFeatureExtractor) extractTrendAcceleration(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window*3 {
		return
	}

	// 计算三个连续窗口的趋势斜率
	w1 := window / 2
	w2 := window
	w3 := window * 3 / 2

	slopes := make([]float64, 3)

	windows := []int{w1, w2, w3}
	for i, w := range windows {
		if len(prices) >= w {
			recentPrices := prices[len(prices)-w:]
			slope, _, _ := calculateLinearRegression(recentPrices)
			slopes[i] = slope
		}
	}

	// 计算趋势加速 (斜率的变化)
	if slopes[0] != 0 {
		acceleration1 := (slopes[1] - slopes[0]) / math.Abs(slopes[0])
		features["trend_acceleration_short"] = acceleration1
	}

	if slopes[1] != 0 {
		acceleration2 := (slopes[2] - slopes[1]) / math.Abs(slopes[1])
		features["trend_acceleration_long"] = acceleration2
	}

	// 加速类型检测
	avgAcceleration := (features["trend_acceleration_short"] + features["trend_acceleration_long"]) / 2
	features["trend_accelerating_up"] = boolToFloat(avgAcceleration > 0.5)
	features["trend_accelerating_down"] = boolToFloat(avgAcceleration < -0.5)
	features["trend_decelerating"] = boolToFloat(math.Abs(avgAcceleration) < 0.1)
}

// extractTrendPeriodicity 趋势周期性分析
func (tfe *TrendFeatureExtractor) extractTrendPeriodicity(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window*4 {
		return
	}

	// 计算价格的周期性
	// 使用自相关函数检测周期性

	pricesNormalized := make([]float64, len(prices))
	mean := calculateAverage(prices)
	std := calculateStandardDeviation(prices, mean)

	for i, price := range prices {
		pricesNormalized[i] = (price - mean) / std
	}

	// 计算不同滞后的自相关
	maxLag := min(50, len(pricesNormalized)/2)
	autocorr := make([]float64, maxLag)

	for lag := 1; lag < maxLag; lag++ {
		corr := calculateCorrelation(pricesNormalized[:len(pricesNormalized)-lag],
			pricesNormalized[lag:])
		autocorr[lag] = math.Abs(corr)
	}

	// 找到最强的周期性
	maxCorr := 0.0
	bestLag := 0

	for lag, corr := range autocorr {
		if corr > maxCorr {
			maxCorr = corr
			bestLag = lag
		}
	}

	features["trend_periodicity_strength"] = maxCorr
	features["trend_periodicity_lag"] = float64(bestLag)
	features["trend_has_periodicity"] = boolToFloat(maxCorr > 0.3)
}

// extractTrendStrengthFeatures 提取趋势强度特征
func (tfe *TrendFeatureExtractor) extractTrendStrengthFeatures(prices, volumes []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window || len(volumes) < window {
		return
	}

	// 趋势强度 = 价格趋势强度 × 成交量确认强度
	priceTrendStrength := tfe.calculatePriceTrendStrength(prices)
	volumeConfirmation := tfe.calculateVolumeConfirmation(prices, volumes)

	features["trend_strength_combined"] = priceTrendStrength * volumeConfirmation

	// ADX (Average Directional Index) 简化版
	adx := tfe.calculateADXSimplified(prices)
	features["adx_trend_strength"] = adx

	// 趋势持续时间
	trendDuration := tfe.calculateTrendDuration(prices)
	features["trend_duration"] = float64(trendDuration)

	// 趋势稳定性
	trendStability := tfe.calculateTrendStability(prices)
	features["trend_stability"] = trendStability
}

// calculatePriceTrendStrength 计算价格趋势强度
func (tfe *TrendFeatureExtractor) calculatePriceTrendStrength(prices []float64) float64 {
	window := tfe.config.TrendWindow

	if len(prices) < window {
		return 0
	}

	recentPrices := prices[len(prices)-window:]

	// 使用线性回归R²作为趋势强度度量
	_, _, r2 := calculateLinearRegression(recentPrices)

	// 结合价格变化幅度
	totalChange := (recentPrices[len(recentPrices)-1] - recentPrices[0]) / recentPrices[0]
	changeMagnitude := math.Abs(totalChange)

	// 综合趋势强度
	trendStrength := r2 * changeMagnitude * 10 // 放大便于观察

	return math.Min(trendStrength, 1.0) // 限制在0-1范围内
}

// calculateVolumeConfirmation 计算成交量确认强度
func (tfe *TrendFeatureExtractor) calculateVolumeConfirmation(prices, volumes []float64) float64 {
	window := tfe.config.TrendWindow

	if len(prices) < window || len(volumes) < window {
		return 0.5 // 中性值
	}

	recentPrices := prices[len(prices)-window:]
	recentVolumes := volumes[len(volumes)-window:]

	// 计算价格趋势方向
	priceSlope, _, _ := calculateLinearRegression(recentPrices)
	priceDirection := 1.0
	if priceSlope < 0 {
		priceDirection = -1.0
	}

	// 计算成交量趋势方向
	volumeSlope, _, _ := calculateLinearRegression(recentVolumes)

	// 成交量确认：价格上涨时成交量应该增加，价格下跌时成交量应该减少
	expectedVolumeDirection := priceDirection
	actualVolumeDirection := 1.0
	if volumeSlope < 0 {
		actualVolumeDirection = -1.0
	}

	// 计算确认度
	confirmation := 1.0
	if expectedVolumeDirection != actualVolumeDirection {
		confirmation = 0.3 // 不确认
	} else {
		confirmation = 0.8 // 确认
	}

	// 考虑成交量变化幅度
	volumeChange := (recentVolumes[len(recentVolumes)-1] - recentVolumes[0]) / recentVolumes[0]
	volumeMagnitude := math.Abs(volumeChange)

	if volumeMagnitude > 0.5 { // 成交量变化显著
		confirmation *= 1.2
	} else if volumeMagnitude < 0.1 { // 成交量变化很小
		confirmation *= 0.8
	}

	return math.Min(confirmation, 1.0)
}

// calculateADXSimplified 简化的ADX计算
func (tfe *TrendFeatureExtractor) calculateADXSimplified(prices []float64) float64 {
	window := 14 // ADX标准周期

	if len(prices) < window+1 {
		return 25.0 // 中性值
	}

	// 计算DM+/DM-
	dmp := make([]float64, len(prices)-1)
	dmm := make([]float64, len(prices)-1)

	for i := 1; i < len(prices); i++ {
		upMove := prices[i] - prices[i-1]
		downMove := prices[i-1] - prices[i]

		if upMove > downMove && upMove > 0 {
			dmp[i-1] = upMove
		}
		if downMove > upMove && downMove > 0 {
			dmm[i-1] = downMove
		}
	}

	// 计算ADX (简化版)
	if len(dmp) >= window && len(dmm) >= window {
		recentDMP := dmp[len(dmp)-window:]
		recentDMM := dmm[len(dmm)-window:]

		avgDMP := calculateAverage(recentDMP)
		avgDMM := calculateAverage(recentDMM)

		if avgDMP+avgDMM > 0 {
			dx := math.Abs(avgDMP-avgDMM) / (avgDMP + avgDMM) * 100
			return dx
		}
	}

	return 25.0
}

// calculateTrendDuration 计算趋势持续时间
func (tfe *TrendFeatureExtractor) calculateTrendDuration(prices []float64) int {
	if len(prices) < 10 {
		return 0
	}

	// 计算价格变化方向
	directions := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		if prices[i] > prices[i-1] {
			directions[i-1] = 1.0
		} else if prices[i] < prices[i-1] {
			directions[i-1] = -1.0
		} else {
			directions[i-1] = 0.0
		}
	}

	// 找到当前趋势的持续时间
	currentDirection := directions[len(directions)-1]
	duration := 0

	for i := len(directions) - 1; i >= 0; i-- {
		if directions[i] == currentDirection || directions[i] == 0 {
			duration++
		} else {
			break
		}
	}

	return duration
}

// calculateTrendStability 计算趋势稳定性
func (tfe *TrendFeatureExtractor) calculateTrendStability(prices []float64) float64 {
	window := tfe.config.TrendWindow

	if len(prices) < window {
		return 0.5
	}

	recentPrices := prices[len(prices)-window:]

	// 计算价格变化的标准差
	returns := make([]float64, len(recentPrices)-1)
	for i := 1; i < len(recentPrices); i++ {
		returns[i-1] = (recentPrices[i] - recentPrices[i-1]) / recentPrices[i-1]
	}

	if len(returns) == 0 {
		return 0.5
	}

	// 稳定性 = 1 / (1 + 波动率)
	volatility := calculateStandardDeviation(returns, calculateAverage(returns))
	stability := 1.0 / (1.0 + volatility)

	return stability
}

// extractTrendReversalFeatures 提取趋势转折点特征
func (tfe *TrendFeatureExtractor) extractTrendReversalFeatures(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window*2 {
		return
	}

	// 检测双顶/双底模式
	tfe.detectDoubleTopBottom(prices, features)

	// 检测头肩顶/底模式 (简化版)
	tfe.detectHeadShoulders(prices, features)

	// 检测支撑/阻力突破
	tfe.detectSupportResistanceBreakout(prices, features)
}

// detectDoubleTopBottom 检测双顶/双底模式
func (tfe *TrendFeatureExtractor) detectDoubleTopBottom(prices []float64, features map[string]float64) {
	if len(prices) < 20 {
		return
	}

	// 寻找局部最高点和最低点
	peaks, troughs := findPeaksAndTroughs(prices)

	// 检查双顶模式 (两个相近的峰值)
	if len(peaks) >= 2 {
		recentPeaks := peaks[max(0, len(peaks)-3):]
		if len(recentPeaks) >= 2 {
			peak1 := recentPeaks[len(recentPeaks)-2]
			peak2 := recentPeaks[len(recentPeaks)-1]

			// 检查峰值高度相似度
			heightDiff := math.Abs(peak1.Value-peak2.Value) / ((peak1.Value + peak2.Value) / 2)
			timeDiff := peak2.Index - peak1.Index

			if heightDiff < 0.05 && timeDiff > 5 && timeDiff < 50 { // 高度差异<5%, 时间间隔合理
				features["double_top_pattern"] = 1.0
				features["pattern_strength"] = (1.0 - heightDiff) * 0.8
			}
		}
	}

	// 检查双底模式
	if len(troughs) >= 2 {
		recentTroughs := troughs[max(0, len(troughs)-3):]
		if len(recentTroughs) >= 2 {
			trough1 := recentTroughs[len(recentTroughs)-2]
			trough2 := recentTroughs[len(recentTroughs)-1]

			heightDiff := math.Abs(trough1.Value-trough2.Value) / ((trough1.Value + trough2.Value) / 2)
			timeDiff := trough2.Index - trough1.Index

			if heightDiff < 0.05 && timeDiff > 5 && timeDiff < 50 {
				features["double_bottom_pattern"] = 1.0
				features["pattern_strength"] = (1.0 - heightDiff) * 0.8
			}
		}
	}
}

// detectHeadShoulders 检测头肩顶/底模式 (简化版)
func (tfe *TrendFeatureExtractor) detectHeadShoulders(prices []float64, features map[string]float64) {
	if len(prices) < 30 {
		return
	}

	peaks, _ := findPeaksAndTroughs(prices)

	if len(peaks) >= 3 {
		// 检查是否有左肩、头部、右肩的模式
		recentPeaks := peaks[max(0, len(peaks)-4):]

		if len(recentPeaks) >= 3 {
			leftShoulder := recentPeaks[len(recentPeaks)-3]
			head := recentPeaks[len(recentPeaks)-2]
			rightShoulder := recentPeaks[len(recentPeaks)-1]

			// 头肩顶模式: 头部最高，两肩相近且低于头部
			if head.Value > leftShoulder.Value && head.Value > rightShoulder.Value {
				shoulderAvg := (leftShoulder.Value + rightShoulder.Value) / 2
				if shoulderAvg < head.Value*0.95 { // 肩膀低于头部5%
					features["head_shoulders_top"] = 1.0
				}
			}
		}
	}
}

// detectSupportResistanceBreakout 检测支撑/阻力突破
func (tfe *TrendFeatureExtractor) detectSupportResistanceBreakout(prices []float64, features map[string]float64) {
	window := tfe.config.TrendWindow

	if len(prices) < window {
		return
	}

	currentPrice := prices[len(prices)-1]
	recentPrices := prices[max(0, len(prices)-window):]

	// 计算最近的高点和低点
	recentMax := findMax(recentPrices)
	recentMin := findMin(recentPrices)

	// 检查突破
	resistanceBreakout := currentPrice > recentMax*0.98 // 接近阻力位
	supportBreakout := currentPrice < recentMin*1.02    // 接近支撑位

	features["near_resistance"] = boolToFloat(resistanceBreakout)
	features["near_support"] = boolToFloat(supportBreakout)

	// 突破强度
	if resistanceBreakout {
		breakoutStrength := (currentPrice - recentMax) / recentMax * 100
		features["resistance_breakout_strength"] = breakoutStrength
	}

	if supportBreakout {
		breakoutStrength := (recentMin - currentPrice) / recentMin * 100
		features["support_breakout_strength"] = breakoutStrength
	}
}

// extractMultiTimeframeTrendFeatures 多时间尺度趋势分析
func (tfe *TrendFeatureExtractor) extractMultiTimeframeTrendFeatures(prices []float64, features map[string]float64) {
	if len(prices) < 100 {
		return
	}

	// 计算不同时间尺度的趋势
	timeframes := []int{20, 50, 100}        // 短期、中期、长期
	_ = []string{"short", "medium", "long"} // timeframeNames

	trends := make([]float64, len(timeframes))

	for i, tf := range timeframes {
		if len(prices) >= tf {
			recentPrices := prices[len(prices)-tf:]
			slope, _, _ := calculateLinearRegression(recentPrices)
			trends[i] = slope
		}
	}

	// 多时间尺度趋势一致性
	consistency := 0.0
	if len(trends) > 1 {
		baseDirection := 1.0
		if trends[0] < 0 {
			baseDirection = -1.0
		}

		consistentCount := 0
		for _, trend := range trends[1:] {
			direction := 1.0
			if trend < 0 {
				direction = -1.0
			}

			if direction == baseDirection {
				consistentCount++
			}
		}

		consistency = float64(consistentCount) / float64(len(trends)-1)
	}

	features["multi_timeframe_consistency"] = consistency

	// 趋势分化检测 (不同时间尺度的趋势相反)
	divergence := 0.0
	if len(trends) >= 2 {
		shortDirection := 1.0
		if trends[0] < 0 {
			shortDirection = -1.0
		}

		longDirection := 1.0
		if trends[len(trends)-1] < 0 {
			longDirection = -1.0
		}

		if shortDirection != longDirection {
			divergence = 1.0
		}
	}

	features["trend_divergence"] = divergence
}

// 辅助函数

func calculateSimpleMovingAverage(values []float64, period int) float64 {
	if len(values) < period {
		return calculateAverage(values)
	}

	sum := 0.0
	for i := len(values) - period; i < len(values); i++ {
		sum += values[i]
	}

	return sum / float64(period)
}

func calculateLinearRegression(values []float64) (slope, intercept, r2 float64) {
	n := float64(len(values))
	if n < 2 {
		return 0, 0, 0
	}

	// 计算必要统计量
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0
	sumYY := 0.0

	for i, y := range values {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumXX += x * x
		sumYY += y * y
	}

	// 计算斜率和截距
	slope = (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept = (sumY - slope*sumX) / n

	// 计算R²
	yMean := sumY / n
	ssRes := 0.0
	ssTot := 0.0

	for i, y := range values {
		x := float64(i)
		yPred := slope*x + intercept
		ssRes += (y - yPred) * (y - yPred)
		ssTot += (y - yMean) * (y - yMean)
	}

	if ssTot != 0 {
		r2 = 1 - (ssRes / ssTot)
	}

	return slope, intercept, r2
}

// findPeaksAndTroughs 寻找峰值和谷值
func findPeaksAndTroughs(prices []float64) (peaks, troughs []PeakTrough) {
	peaks = make([]PeakTrough, 0)
	troughs = make([]PeakTrough, 0)

	if len(prices) < 5 {
		return peaks, troughs
	}

	for i := 2; i < len(prices)-2; i++ {
		// 检查峰值 (局部最大值)
		if prices[i] > prices[i-1] && prices[i] > prices[i-2] &&
			prices[i] > prices[i+1] && prices[i] > prices[i+2] {
			peaks = append(peaks, PeakTrough{Index: i, Value: prices[i]})
		}

		// 检查谷值 (局部最小值)
		if prices[i] < prices[i-1] && prices[i] < prices[i-2] &&
			prices[i] < prices[i+1] && prices[i] < prices[i+2] {
			troughs = append(troughs, PeakTrough{Index: i, Value: prices[i]})
		}
	}

	return peaks, troughs
}

type PeakTrough struct {
	Index int
	Value float64
}
