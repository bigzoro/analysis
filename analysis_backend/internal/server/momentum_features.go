package server

import (
	"context"
	"fmt"
	"math"
)

// MomentumFeatureExtractor 动量特征提取器
type MomentumFeatureExtractor struct {
	config FeatureConfig
}

// Name 返回提取器名称
func (mfe *MomentumFeatureExtractor) Name() string {
	return "momentum"
}

// Priority 返回提取优先级
func (mfe *MomentumFeatureExtractor) Priority() int {
	return 75
}

// Extract 提取动量特征
func (mfe *MomentumFeatureExtractor) Extract(ctx context.Context, symbol string, currentData *MarketDataPoint, historyData []*MarketDataPoint) (map[string]float64, error) {
	features := make(map[string]float64)

	if len(historyData) < 10 {
		return features, fmt.Errorf("历史数据不足，至少需要10个数据点")
	}

	// 提取价格和成交量数据
	prices := make([]float64, len(historyData))
	volumes := make([]float64, len(historyData))

	for i, data := range historyData {
		prices[i] = data.Price
		volumes[i] = data.Volume24h
	}

	// 1. 价格动量特征
	mfe.extractPriceMomentumFeatures(prices, features)

	// 2. 成交量动量特征
	mfe.extractVolumeMomentumFeatures(volumes, features)

	// 3. 动量组合特征
	mfe.extractMomentumCombinationFeatures(prices, volumes, features)

	// 4. 动量稳定性分析
	mfe.extractMomentumStabilityFeatures(prices, features)

	// 5. 动量转折点检测
	mfe.extractMomentumReversalFeatures(prices, features)

	return features, nil
}

// extractPriceMomentumFeatures 提取价格动量特征
func (mfe *MomentumFeatureExtractor) extractPriceMomentumFeatures(prices []float64, features map[string]float64) {
	n := len(prices)

	if n < 5 {
		return
	}

	// 计算收益率序列
	returns := make([]float64, n-1)
	for i := 1; i < n; i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	// 不同周期的价格动量 (ROC - Rate of Change)
	periods := []int{1, 3, 5, 10, 20}
	periodNames := []string{"1d", "3d", "5d", "10d", "20d"}

	for i, period := range periods {
		if n > period {
			// ROC = (当前价格 - period前价格) / period前价格 * 100
			roc := (prices[n-1] - prices[n-1-period]) / prices[n-1-period] * 100
			features[fmt.Sprintf("price_roc_%s", periodNames[i])] = roc

			// 标准化ROC (相对于历史波动率)
			if period > 1 {
				historicalReturns := returns[n-period : n-1]
				if len(historicalReturns) > 0 {
					stdDev := calculateStandardDeviation(historicalReturns, calculateAverage(historicalReturns))
					if stdDev > 0 {
						normalizedROC := roc / (stdDev * 100) // 标准化到标准差单位
						features[fmt.Sprintf("price_roc_normalized_%s", periodNames[i])] = normalizedROC
					}
				}
			}
		}
	}

	// 动量振荡器 (Momentum Oscillator)
	if n >= 14 {
		// RSI-style momentum oscillator
		momentumOscillator := mfe.calculateMomentumOscillator(returns, 14)
		features["momentum_oscillator"] = momentumOscillator

		// 动量背离检测 (简化版)
		divergence := mfe.detectMomentumDivergence(prices, returns)
		features["momentum_divergence"] = divergence
	}

	// 动量强度 (Momentum Strength Index)
	if n >= 25 {
		msi := mfe.calculateMomentumStrengthIndex(returns, 25)
		features["momentum_strength_index"] = msi
	}
}

// extractVolumeMomentumFeatures 提取成交量动量特征
func (mfe *MomentumFeatureExtractor) extractVolumeMomentumFeatures(volumes []float64, features map[string]float64) {
	n := len(volumes)

	if n < 5 {
		return
	}

	// 成交量变化率
	periods := []int{1, 3, 5, 10}
	periodNames := []string{"1d", "3d", "5d", "10d"}

	for i, period := range periods {
		if n > period {
			volumeROC := (volumes[n-1] - volumes[n-1-period]) / volumes[n-1-period] * 100
			features[fmt.Sprintf("volume_roc_%s", periodNames[i])] = volumeROC
		}
	}

	// 成交量相对强度指数 (Volume RSI)
	if n >= 14 {
		volumeRSI := mfe.calculateVolumeRSI(volumes, 14)
		features["volume_rsi"] = volumeRSI

		// 成交量趋势
		volumeTrend := calculateLinearTrend(volumes[max(0, n-20):])
		features["volume_trend"] = volumeTrend

		// 成交量动量一致性
		volumeMomentumConsistency := mfe.calculateVolumeMomentumConsistency(volumes)
		features["volume_momentum_consistency"] = volumeMomentumConsistency
	}

	// 异常成交量检测
	if n >= 20 {
		volumeOutlier := mfe.detectVolumeOutlier(volumes)
		features["volume_outlier"] = boolToFloat(volumeOutlier)

		// 成交量价格确认 (Volume-Price Confirmation)
		vpc := mfe.calculateVolumePriceConfirmation(volumes, 20)
		features["volume_price_confirmation"] = vpc
	}
}

// extractMomentumCombinationFeatures 提取动量组合特征
func (mfe *MomentumFeatureExtractor) extractMomentumCombinationFeatures(prices, volumes []float64, features map[string]float64) {
	n := len(prices)

	if n < 10 {
		return
	}

	// 计算价格和成交量的联合动量
	priceROC5 := (prices[n-1] - prices[n-6]) / prices[n-6] * 100
	volumeROC5 := (volumes[n-1] - volumes[n-6]) / volumes[n-6] * 100

	// 动量同步性 (价格和成交量动量方向是否一致)
	priceDirection := 1.0
	if priceROC5 < 0 {
		priceDirection = -1.0
	}

	volumeDirection := 1.0
	if volumeROC5 < 0 {
		volumeDirection = -1.0
	}

	synchronization := 0.0
	if priceDirection == volumeDirection {
		synchronization = 1.0
	}
	features["momentum_synchronization"] = synchronization

	// 动量强度组合
	combinedMomentumStrength := math.Abs(priceROC5) * math.Abs(volumeROC5) / 1000
	features["combined_momentum_strength"] = combinedMomentumStrength

	// 动量比率 (成交量动量相对价格动量)
	if priceROC5 != 0 {
		momentumRatio := volumeROC5 / priceROC5
		features["momentum_ratio"] = momentumRatio

		// 动量失衡检测
		features["momentum_imbalance"] = boolToFloat(math.Abs(momentumRatio) > 2.0)
	}

	// 动量加速检测
	if n >= 15 {
		// 计算近期和远期的动量变化
		recentMomentum := (prices[n-1] - prices[n-6]) / prices[n-6]
		olderMomentum := (prices[n-6] - prices[n-11]) / prices[n-11]

		momentumAcceleration := recentMomentum - olderMomentum
		features["momentum_acceleration"] = momentumAcceleration * 100

		// 加速类型
		features["momentum_accelerating"] = boolToFloat(momentumAcceleration > 0.01)
		features["momentum_decelerating"] = boolToFloat(momentumAcceleration < -0.01)
	}
}

// extractMomentumStabilityFeatures 提取动量稳定性特征
func (mfe *MomentumFeatureExtractor) extractMomentumStabilityFeatures(prices []float64, features map[string]float64) {
	n := len(prices)

	if n < 20 {
		return
	}

	// 计算动量序列
	momenta := make([]float64, n-1)
	for i := 1; i < n; i++ {
		momenta[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	// 动量稳定性 (动量标准差的倒数)
	momentumStd := calculateStandardDeviation(momenta, calculateAverage(momenta))
	momentumStability := 1.0 / (1.0 + momentumStd)
	features["momentum_stability"] = momentumStability

	// 动量变化率的标准差
	momentumChanges := make([]float64, len(momenta)-1)
	for i := 1; i < len(momenta); i++ {
		momentumChanges[i-1] = momenta[i] - momenta[i-1]
	}

	if len(momentumChanges) > 0 {
		changeStd := calculateStandardDeviation(momentumChanges, calculateAverage(momentumChanges))
		features["momentum_change_volatility"] = changeStd

		// 动量平滑度
		smoothness := 1.0 / (1.0 + changeStd)
		features["momentum_smoothness"] = smoothness
	}

	// 动量持久性 (连续同向动量的平均长度)
	persistence := mfe.calculateMomentumPersistence(momenta)
	features["momentum_persistence"] = persistence

	// 动量可预测性
	predictability := mfe.calculateMomentumPredictability(momenta)
	features["momentum_predictability"] = predictability
}

// extractMomentumReversalFeatures 提取动量转折点特征
func (mfe *MomentumFeatureExtractor) extractMomentumReversalFeatures(prices []float64, features map[string]float64) {
	n := len(prices)

	if n < 20 {
		return
	}

	// 计算动量序列
	momenta := make([]float64, n-1)
	for i := 1; i < n; i++ {
		momenta[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	// 动量转折检测
	reversalSignal := mfe.detectMomentumReversal(momenta)
	features["momentum_reversal_signal"] = reversalSignal

	// 动量衰减检测
	momentumDecay := mfe.detectMomentumDecay(momenta)
	features["momentum_decay"] = momentumDecay

	// 动量背离 (Momentum Divergence)
	divergence := mfe.detectMomentumDivergence(prices, momenta)
	features["momentum_divergence"] = divergence

	// 动量支撑/阻力
	support, resistance := mfe.calculateMomentumSupportResistance(momenta)
	features["momentum_support_level"] = support
	features["momentum_resistance_level"] = resistance

	currentMomentum := momenta[len(momenta)-1]
	features["momentum_near_support"] = boolToFloat(currentMomentum < support*1.1)
	features["momentum_near_resistance"] = boolToFloat(currentMomentum > resistance*0.9)
}

// calculateMomentumOscillator 计算动量振荡器 (类似RSI的动量版本)
func (mfe *MomentumFeatureExtractor) calculateMomentumOscillator(returns []float64, period int) float64 {
	if len(returns) < period {
		return 50.0
	}

	// 计算动量序列
	momenta := make([]float64, len(returns))
	for i, ret := range returns {
		momenta[i] = ret * 100 // 转换为百分比
	}

	// 计算正向和负向动量
	positive := make([]float64, len(momenta))
	negative := make([]float64, len(momenta))

	for i, mom := range momenta {
		if mom > 0 {
			positive[i] = mom
		} else {
			negative[i] = -mom
		}
	}

	// 计算RSI风格的动量振荡器
	recentPositive := positive[len(positive)-period:]
	recentNegative := negative[len(negative)-period:]

	avgPositive := calculateAverage(recentPositive)
	avgNegative := calculateAverage(recentNegative)

	if avgPositive+avgNegative == 0 {
		return 50.0
	}

	rs := avgPositive / avgNegative
	momentumOscillator := 100 - (100 / (1 + rs))

	return momentumOscillator
}

// detectMomentumDivergence 检测动量背离
func (mfe *MomentumFeatureExtractor) detectMomentumDivergence(prices, returns []float64) float64 {
	if len(prices) < 20 || len(returns) < 20 {
		return 0.0
	}

	// 简化的背离检测：价格创新高但动量在下降
	priceHigh := findMax(prices[len(prices)-20:])
	_ = len(prices) - 20 + findMaxIndex(prices[len(prices)-20:]) // priceHighIndex

	momentumRecent := returns[len(returns)-10:]
	momentumTrend := calculateLinearTrend(momentumRecent)

	// 检查价格是否在近期创高，但动量是否在下降
	priceAtHigh := prices[len(prices)-1] >= priceHigh*0.98
	momentumDeclining := momentumTrend < -0.001

	if priceAtHigh && momentumDeclining {
		// 计算背离强度
		divergenceStrength := math.Min(1.0, math.Abs(momentumTrend)*1000)
		return divergenceStrength
	}

	return 0.0
}

// calculateMomentumStrengthIndex 计算动量强度指数
func (mfe *MomentumFeatureExtractor) calculateMomentumStrengthIndex(returns []float64, period int) float64 {
	if len(returns) < period*2 {
		return 0.5
	}

	// 计算动量强度：动量相对于其历史波动率的强度
	recentReturns := returns[len(returns)-period:]
	historicalReturns := returns[len(returns)-period*2 : len(returns)-period]

	recentVolatility := calculateStandardDeviation(recentReturns, calculateAverage(recentReturns))
	historicalVolatility := calculateStandardDeviation(historicalReturns, calculateAverage(historicalReturns))

	if historicalVolatility == 0 {
		return 0.5
	}

	// 当前动量相对于历史波动率的强度
	strengthRatio := recentVolatility / historicalVolatility

	// 归一化到0-1范围
	strengthIndex := 1.0 / (1.0 + math.Exp(-strengthRatio+1)) // Sigmoid函数

	return strengthIndex
}

// calculateVolumeRSI 计算成交量RSI
func (mfe *MomentumFeatureExtractor) calculateVolumeRSI(volumes []float64, period int) float64 {
	if len(volumes) < period+1 {
		return 50.0
	}

	// 计算成交量变化
	volumeChanges := make([]float64, len(volumes)-1)
	for i := 1; i < len(volumes); i++ {
		volumeChanges[i-1] = (volumes[i] - volumes[i-1]) / volumes[i-1]
	}

	// 计算正向和负向成交量变化
	positive := make([]float64, len(volumeChanges))
	negative := make([]float64, len(volumeChanges))

	for i, change := range volumeChanges {
		if change > 0 {
			positive[i] = change
		} else {
			negative[i] = -change
		}
	}

	// 计算RSI
	recentPositive := positive[len(positive)-period:]
	recentNegative := negative[len(negative)-period:]

	avgPositive := calculateAverage(recentPositive)
	avgNegative := calculateAverage(recentNegative)

	if avgPositive+avgNegative == 0 {
		return 50.0
	}

	rs := avgPositive / avgNegative
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateVolumeMomentumConsistency 计算成交量动量一致性
func (mfe *MomentumFeatureExtractor) calculateVolumeMomentumConsistency(volumes []float64) float64 {
	if len(volumes) < 10 {
		return 0.5
	}

	// 计算成交量变化方向的一致性
	changes := make([]float64, len(volumes)-1)
	for i := 1; i < len(volumes); i++ {
		if volumes[i] > volumes[i-1] {
			changes[i-1] = 1.0
		} else if volumes[i] < volumes[i-1] {
			changes[i-1] = -1.0
		} else {
			changes[i-1] = 0.0
		}
	}

	// 计算一致性分数
	consecutiveSame := 0
	maxConsecutive := 0
	currentDirection := changes[0]

	for _, change := range changes {
		if change == currentDirection {
			consecutiveSame++
			if consecutiveSame > maxConsecutive {
				maxConsecutive = consecutiveSame
			}
		} else if change != 0 {
			currentDirection = change
			consecutiveSame = 1
		}
	}

	consistency := float64(maxConsecutive) / float64(len(changes))
	return math.Min(consistency, 1.0)
}

// detectVolumeOutlier 检测异常成交量
func (mfe *MomentumFeatureExtractor) detectVolumeOutlier(volumes []float64) bool {
	if len(volumes) < 20 {
		return false
	}

	currentVolume := volumes[len(volumes)-1]
	recentVolumes := volumes[len(volumes)-20:]

	mean := calculateAverage(recentVolumes)
	std := calculateStandardDeviation(recentVolumes, mean)

	if std == 0 {
		return false
	}

	zScore := (currentVolume - mean) / std
	return math.Abs(zScore) > 3.0 // 3倍标准差作为异常阈值
}

// calculateVolumePriceConfirmation 计算成交量价格确认
func (mfe *MomentumFeatureExtractor) calculateVolumePriceConfirmation(volumes []float64, window int) float64 {
	if len(volumes) < window {
		return 0.5
	}

	// 这里应该结合价格数据计算，但暂时只基于成交量趋势
	recentVolumes := volumes[len(volumes)-window:]
	volumeTrend := calculateLinearTrend(recentVolumes)

	// 标准化趋势强度
	avgVolume := calculateAverage(recentVolumes)
	if avgVolume == 0 {
		return 0.5
	}

	trendStrength := volumeTrend / avgVolume
	confirmation := 0.5 + math.Min(math.Max(trendStrength, -0.5), 0.5) // 归一化到0-1

	return confirmation
}

// calculateMomentumPersistence 计算动量持久性
func (mfe *MomentumFeatureExtractor) calculateMomentumPersistence(momenta []float64) float64 {
	if len(momenta) < 10 {
		return 0.0
	}

	// 计算连续同向动量的平均长度
	consecutiveLengths := make([]int, 0)
	currentLength := 1
	currentDirection := 1.0

	if momenta[0] < 0 {
		currentDirection = -1.0
	}

	for i := 1; i < len(momenta); i++ {
		direction := 1.0
		if momenta[i] < 0 {
			direction = -1.0
		}

		if direction == currentDirection {
			currentLength++
		} else {
			if currentLength > 1 {
				consecutiveLengths = append(consecutiveLengths, currentLength)
			}
			currentDirection = direction
			currentLength = 1
		}
	}

	if len(consecutiveLengths) == 0 {
		return 0.0
	}

	avgLength := calculateAverage(intSliceToFloatSlice(consecutiveLengths))
	return math.Min(avgLength/10.0, 1.0) // 归一化
}

// calculateMomentumPredictability 计算动量可预测性
func (mfe *MomentumFeatureExtractor) calculateMomentumPredictability(momenta []float64) float64 {
	if len(momenta) < 20 {
		return 0.0
	}

	// 使用自相关系数衡量可预测性
	autocorr := calculateCorrelation(momenta[:len(momenta)-1], momenta[1:])

	// 取绝对值，因为负相关也表示某种可预测性
	predictability := math.Abs(autocorr)

	return predictability
}

// detectMomentumReversal 检测动量转折
func (mfe *MomentumFeatureExtractor) detectMomentumReversal(momenta []float64) float64 {
	if len(momenta) < 10 {
		return 0.0
	}

	// 计算动量变化的二阶导数（加速/减速）
	changes := make([]float64, len(momenta)-1)
	for i := 1; i < len(momenta); i++ {
		changes[i-1] = momenta[i] - momenta[i-1]
	}

	// 检测转折点：动量方向改变
	recentChanges := changes[max(0, len(changes)-5):]
	directionChanges := 0

	for i := 1; i < len(recentChanges); i++ {
		if (recentChanges[i] > 0 && recentChanges[i-1] < 0) ||
			(recentChanges[i] < 0 && recentChanges[i-1] > 0) {
			directionChanges++
		}
	}

	reversalStrength := float64(directionChanges) / 4.0 // 归一化到0-1
	return math.Min(reversalStrength, 1.0)
}

// detectMomentumDecay 检测动量衰减
func (mfe *MomentumFeatureExtractor) detectMomentumDecay(momenta []float64) float64 {
	if len(momenta) < 10 {
		return 0.0
	}

	// 计算动量的趋势
	recentMomenta := momenta[len(momenta)-10:]
	trend := calculateLinearTrend(recentMomenta)

	// 计算动量衰减：正动量在减弱，或负动量在加强（变为更负）
	currentMomentum := momenta[len(momenta)-1]
	avgMomentum := calculateAverage(recentMomenta)

	// 衰减信号：当前动量偏离平均动量，且趋势与之相反
	deviation := currentMomentum - avgMomentum
	trendOpposesDeviation := (deviation > 0 && trend < 0) || (deviation < 0 && trend > 0)

	if trendOpposesDeviation {
		decayStrength := math.Min(math.Abs(trend)*100, 1.0)
		return decayStrength
	}

	return 0.0
}

// calculateMomentumSupportResistance 计算动量支撑/阻力
func (mfe *MomentumFeatureExtractor) calculateMomentumSupportResistance(momenta []float64) (float64, float64) {
	if len(momenta) < 20 {
		return 0.0, 0.0
	}

	// 使用历史动量数据计算支撑和阻力水平
	recentMomenta := momenta[len(momenta)-20:]

	support := findMin(recentMomenta)
	resistance := findMax(recentMomenta)

	return support, resistance
}

// 辅助函数

func intSliceToFloatSlice(ints []int) []float64 {
	floats := make([]float64, len(ints))
	for i, v := range ints {
		floats[i] = float64(v)
	}
	return floats
}

func findMaxIndex(values []float64) int {
	if len(values) == 0 {
		return -1
	}

	maxIndex := 0
	maxValue := values[0]

	for i, v := range values[1:] {
		if v > maxValue {
			maxValue = v
			maxIndex = i + 1
		}
	}

	return maxIndex
}
