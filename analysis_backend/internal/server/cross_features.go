package server

import (
	"context"
	"fmt"
	"math"
)

// CrossFeatureExtractor 交叉特征提取器
type CrossFeatureExtractor struct {
	config FeatureConfig
}

// Name 返回提取器名称
func (cfe *CrossFeatureExtractor) Name() string {
	return "cross"
}

// Priority 返回提取优先级
func (cfe *CrossFeatureExtractor) Priority() int {
	return 70
}

// Extract 提取交叉特征
func (cfe *CrossFeatureExtractor) Extract(ctx context.Context, symbol string, currentData *MarketDataPoint, historyData []*MarketDataPoint) (map[string]float64, error) {
	features := make(map[string]float64)

	if len(historyData) < 10 {
		return features, fmt.Errorf("历史数据不足，至少需要10个数据点")
	}

	// 首先提取基础特征用于交叉组合
	baseFeatures := cfe.extractBaseFeaturesForCross(currentData, historyData)

	// 1. 技术指标交叉特征
	cfe.extractTechnicalCrossFeatures(baseFeatures, features)

	// 2. 价格成交量交叉特征
	cfe.extractPriceVolumeCrossFeatures(baseFeatures, features)

	// 3. 时间序列交叉特征
	cfe.extractTimeSeriesCrossFeatures(baseFeatures, features)

	// 4. 统计交叉特征
	cfe.extractStatisticalCrossFeatures(baseFeatures, features)

	// 5. 动量趋势交叉特征
	cfe.extractMomentumTrendCrossFeatures(baseFeatures, features)

	// 6. DMI交叉特征
	cfe.extractDMICrossFeatures(baseFeatures, features)

	// 7. Ichimoku交叉特征
	cfe.extractIchimokuCrossFeatures(baseFeatures, features)

	return features, nil
}

// extractBaseFeaturesForCross 提取用于交叉的基础特征
func (cfe *CrossFeatureExtractor) extractBaseFeaturesForCross(currentData *MarketDataPoint, historyData []*MarketDataPoint) map[string]float64 {
	base := make(map[string]float64)

	// 价格相关
	prices := make([]float64, len(historyData))
	for i, data := range historyData {
		prices[i] = data.Price
	}
	base["current_price"] = currentData.Price
	base["price_change_24h"] = currentData.PriceChange24h
	base["price_volatility"] = calculateStandardDeviation(prices, calculateAverage(prices))

	// 成交量相关
	volumes := make([]float64, len(historyData))
	for i, data := range historyData {
		volumes[i] = data.Volume24h
	}
	base["current_volume"] = currentData.Volume24h
	base["volume_avg"] = calculateAverage(volumes)
	base["volume_trend"] = calculateLinearTrend(volumes[len(volumes)-min(10, len(volumes)):])

	// 技术指标相关
	if currentData.TechnicalData != nil {
		tech := currentData.TechnicalData
		base["rsi"] = tech.RSI
		base["macd"] = tech.MACD
		base["bb_position"] = tech.BBPosition
		base["ma5"] = tech.MA5
		base["ma20"] = tech.MA20
	}

	// DMI指标相关（简化版，使用价格波动估算）
	if len(historyData) >= 30 {
		prices := make([]float64, len(historyData))
		for i, data := range historyData {
			prices[i] = data.Price
		}
		// 简化的DMI计算：基于价格变动的趋势强度
		dmiStrength := cfe.calculateSimplifiedDMI(prices)
		base["dmi_strength"] = dmiStrength
		base["dmi_trend"] = cfe.calculatePriceTrend(prices)
	}

	// Ichimoku指标相关（简化版）
	if len(historyData) >= 30 {
		prices := make([]float64, len(historyData))
		for i, data := range historyData {
			prices[i] = data.Price
		}
		ichimokuSignals := cfe.calculateSimplifiedIchimoku(prices)
		for k, v := range ichimokuSignals {
			base[k] = v
		}
	}

	// 动量相关
	if len(prices) >= 5 {
		base["momentum_5d"] = (prices[len(prices)-1] - prices[len(prices)-6]) / prices[len(prices)-6] * 100
		base["momentum_1d"] = (prices[len(prices)-1] - prices[len(prices)-2]) / prices[len(prices)-2] * 100
	}

	// 趋势相关
	if len(prices) >= 20 {
		recentPrices := prices[len(prices)-20:]
		slope, _, r2 := calculateLinearRegression(recentPrices)
		base["trend_slope"] = slope
		base["trend_strength"] = r2
	}

	return base
}

// extractTechnicalCrossFeatures 提取技术指标交叉特征
func (cfe *CrossFeatureExtractor) extractTechnicalCrossFeatures(baseFeatures map[string]float64, features map[string]float64) {
	rsi, hasRSI := baseFeatures["rsi"]
	macd, hasMACD := baseFeatures["macd"]
	bbPos, hasBB := baseFeatures["bb_position"]
	ma5, hasMA5 := baseFeatures["ma5"]
	ma20, hasMA20 := baseFeatures["ma20"]

	// RSI + MACD 组合
	if hasRSI && hasMACD {
		// RSI超买超卖与MACD方向确认
		rsiSignal := 0.0
		if rsi < 30 {
			rsiSignal = -1.0 // 超卖
		} else if rsi > 70 {
			rsiSignal = 1.0 // 超买
		}

		macdSignal := 0.0
		if macd > 0 {
			macdSignal = 1.0 // 看涨
		} else {
			macdSignal = -1.0 // 看跌
		}

		// 信号一致性
		signalConsistency := 1.0
		if rsiSignal*macdSignal < 0 {
			signalConsistency = -1.0 // 信号分歧
		}

		features["rsi_macd_consistency"] = signalConsistency
		features["rsi_macd_strength"] = math.Abs(rsi-50) * math.Abs(macd) / 50
	}

	// RSI + 布林带组合
	if hasRSI && hasBB {
		// RSI与布林带位置的配合
		rsiBBCombo := 0.0
		if rsi < 30 && bbPos < -0.5 {
			rsiBBCombo = -1.0 // 双重超卖信号
		} else if rsi > 70 && bbPos > 0.5 {
			rsiBBCombo = 1.0 // 双重超买信号
		}

		features["rsi_bb_combo"] = rsiBBCombo
		features["rsi_bb_divergence"] = (rsi/50 - 1) - bbPos // RSI与BB的偏离度
	}

	// 均线系统组合
	if hasMA5 && hasMA20 {
		// 均线排列强度
		maAlignment := 0.0
		if ma5 > ma20 {
			maAlignment = 0.5 // 多头排列
		} else {
			maAlignment = -0.5 // 空头排列
		}

		// 均线斜率差异
		maSlopeDiff := (ma5 - ma20) / ma20
		features["ma_alignment"] = maAlignment
		features["ma_slope_diff"] = maSlopeDiff

		// 金叉死叉信号
		if hasRSI {
			goldenCross := boolToFloat(ma5 > ma20 && rsi > 50)
			deathCross := boolToFloat(ma5 < ma20 && rsi < 50)
			features["golden_cross_signal"] = goldenCross
			features["death_cross_signal"] = deathCross
		}
	}

	// MACD + 布林带组合
	if hasMACD && hasBB {
		// MACD与布林带位置的配合
		macdBBPower := math.Abs(macd) * (1 + math.Abs(bbPos))
		features["macd_bb_power"] = macdBBPower

		// 趋势确认强度
		trendConfirmation := macd * bbPos
		features["trend_confirmation"] = trendConfirmation
	}
}

// extractPriceVolumeCrossFeatures 提取价格成交量交叉特征
func (cfe *CrossFeatureExtractor) extractPriceVolumeCrossFeatures(baseFeatures map[string]float64, features map[string]float64) {
	priceChange, hasPriceChange := baseFeatures["price_change_24h"]
	volume, hasVolume := baseFeatures["current_volume"]
	volumeAvg, hasVolumeAvg := baseFeatures["volume_avg"]
	volumeTrend, hasVolumeTrend := baseFeatures["volume_trend"]

	// 价格变化与成交量放大
	if hasPriceChange && hasVolume && hasVolumeAvg {
		volumeRatio := volume / volumeAvg

		// 成交量确认
		volumeConfirmation := 0.0
		if priceChange > 0 && volumeRatio > 1.2 {
			volumeConfirmation = 1.0 // 上涨有成交量配合
		} else if priceChange < 0 && volumeRatio > 1.2 {
			volumeConfirmation = -1.0 // 下跌有成交量配合
		} else if math.Abs(priceChange) < 1.0 && volumeRatio < 0.8 {
			volumeConfirmation = 0.5 // 小幅波动，成交量萎缩
		}

		features["price_volume_confirmation"] = volumeConfirmation
		features["volume_price_ratio"] = volumeRatio
	}

	// 成交量趋势与价格趋势的配合
	if hasPriceChange && hasVolumeTrend {
		// 计算价格趋势方向
		priceDirection := 1.0
		if priceChange < 0 {
			priceDirection = -1.0
		}

		volumeDirection := 1.0
		if volumeTrend < 0 {
			volumeDirection = -1.0
		}

		// 趋势同步性
		trendSync := 1.0
		if priceDirection != volumeDirection {
			trendSync = -1.0 // 趋势不一致
		}

		features["price_volume_trend_sync"] = trendSync
		features["volume_trend_strength"] = math.Abs(volumeTrend)
	}

	// 异常成交量检测
	if hasVolume && hasVolumeAvg {
		volumeZScore := (volume - volumeAvg) / baseFeatures["price_volatility"] // 使用价格波动率作为成交量波动率的代理
		features["volume_z_score"] = volumeZScore
		features["volume_extreme_high"] = boolToFloat(volumeZScore > 2.0)
		features["volume_extreme_low"] = boolToFloat(volumeZScore < -2.0)
	}

	// 成交量价格动量比
	momentum1d, hasMom1d := baseFeatures["momentum_1d"]
	momentum5d, hasMom5d := baseFeatures["momentum_5d"]

	if hasMom1d && hasVolume && hasVolumeAvg {
		volumeMomentumRatio := volume / volumeAvg
		priceMomentumRatio := math.Abs(momentum1d) / 10 // 归一化

		if priceMomentumRatio > 0 {
			momentumVolumeRatio := volumeMomentumRatio / priceMomentumRatio
			features["momentum_volume_ratio"] = momentumVolumeRatio

			// 动量失衡检测
			features["momentum_volume_imbalance"] = boolToFloat(math.Abs(momentumVolumeRatio-1.0) > 0.5)
		}
	}

	if hasMom5d && hasVolumeTrend {
		// 5日动量与成交量趋势的相关性
		momentumVolumeCorr := math.Abs(momentum5d) * math.Abs(volumeTrend) / 100
		features["momentum_volume_correlation"] = momentumVolumeCorr
	}
}

// extractTimeSeriesCrossFeatures 提取时间序列交叉特征
func (cfe *CrossFeatureExtractor) extractTimeSeriesCrossFeatures(baseFeatures map[string]float64, features map[string]float64) {
	// 这里需要历史数据进行时间序列分析
	// 暂时基于基础特征进行推断

	trendSlope, hasSlope := baseFeatures["trend_slope"]
	trendStrength, hasStrength := baseFeatures["trend_strength"]
	priceVolatility, hasVol := baseFeatures["price_volatility"]
	momentum1d, hasMom1d := baseFeatures["momentum_1d"]
	momentum5d, hasMom5d := baseFeatures["momentum_5d"]

	// 趋势与波动率的交叉
	if hasSlope && hasVol {
		// 趋势质量：趋势强度相对于波动率的比率
		trendQuality := math.Abs(trendSlope) / (priceVolatility + 0.001) * 100
		features["trend_quality"] = math.Min(trendQuality, 5.0) // 限制上限
	}

	// 趋势与动量的交叉
	if hasSlope && hasMom5d {
		// 趋势动量一致性
		trendDirection := 1.0
		if trendSlope < 0 {
			trendDirection = -1.0
		}

		momentumDirection := 1.0
		if momentum5d < 0 {
			momentumDirection = -1.0
		}

		trendMomentumConsistency := 1.0
		if trendDirection != momentumDirection {
			trendMomentumConsistency = -1.0
		}

		features["trend_momentum_consistency"] = trendMomentumConsistency
	}

	// 动量加速检测
	if hasMom1d && hasMom5d {
		momentumAcceleration := momentum1d - momentum5d/5 // 简化的加速计算
		features["momentum_acceleration_cross"] = momentumAcceleration

		// 加速类型识别
		if momentumAcceleration > 1.0 {
			features["momentum_accelerating_up"] = 1.0
		} else if momentumAcceleration < -1.0 {
			features["momentum_accelerating_down"] = 1.0
		}
	}

	// 趋势强度与波动率的组合
	if hasStrength && hasVol {
		// 有效趋势：高强度，低波动
		effectiveTrend := trendStrength * (1.0 / (1.0 + priceVolatility/10))
		features["effective_trend"] = effectiveTrend
	}

	// 多时间尺度动量比较
	if hasMom1d && hasMom5d {
		momentumDivergence := math.Abs(momentum1d - momentum5d/5)
		features["momentum_time_divergence"] = momentumDivergence

		// 时间尺度一致性
		timeConsistency := 1.0 - (momentumDivergence / (math.Abs(momentum1d) + 1))
		features["momentum_time_consistency"] = math.Max(0, timeConsistency)
	}
}

// extractStatisticalCrossFeatures 提取统计交叉特征
func (cfe *CrossFeatureExtractor) extractStatisticalCrossFeatures(baseFeatures map[string]float64, features map[string]float64) {
	// 基于现有特征计算统计交叉特征

	// 特征稳定性评估
	featureStability := cfe.calculateFeatureStability(baseFeatures)
	features["feature_stability"] = featureStability

	// 特征多样性评估
	featureDiversity := cfe.calculateFeatureDiversity(baseFeatures)
	features["feature_diversity"] = featureDiversity

	// 特征一致性评估
	featureConsistency := cfe.calculateFeatureConsistency(baseFeatures)
	features["feature_consistency"] = featureConsistency

	// 异常特征检测
	outlierScore := cfe.detectFeatureOutliers(baseFeatures)
	features["feature_outlier_score"] = outlierScore

	// 特征强度综合评估
	featureStrength := cfe.calculateFeatureStrength(baseFeatures)
	features["feature_strength"] = featureStrength
}

// extractMomentumTrendCrossFeatures 提取动量趋势交叉特征
func (cfe *CrossFeatureExtractor) extractMomentumTrendCrossFeatures(baseFeatures map[string]float64, features map[string]float64) {
	trendSlope, hasSlope := baseFeatures["trend_slope"]
	trendStrength, hasStrength := baseFeatures["trend_strength"]
	momentum1d, hasMom1d := baseFeatures["momentum_1d"]
	momentum5d, hasMom5d := baseFeatures["momentum_5d"]

	// 动量趋势背离检测
	if hasSlope && hasMom5d {
		// 计算背离程度
		trendDirection := trendSlope / math.Abs(trendSlope+0.001) // 归一化方向
		momentumDirection := momentum5d / math.Abs(momentum5d+0.001)

		divergence := trendDirection * momentumDirection * -1 // 相反方向为背离
		features["momentum_trend_divergence"] = divergence

		// 背离强度
		divergenceStrength := math.Abs(trendSlope) * math.Abs(momentum5d) / 100
		features["divergence_strength"] = math.Min(divergenceStrength, 1.0)
	}

	// 动量趋势同步性
	if hasStrength && hasMom5d {
		// 高趋势强度下的动量确认
		momentumConfirmation := trendStrength * math.Abs(momentum5d) / 10
		features["momentum_trend_confirmation"] = math.Min(momentumConfirmation, 1.0)
	}

	// 动量趋势加速
	if hasSlope && hasMom1d && hasMom5d {
		// 计算趋势加速与动量加速的关系
		trendAcceleration := math.Abs(trendSlope) // 简化为趋势强度
		momentumAcceleration := math.Abs(momentum1d - momentum5d/5)

		accelerationSync := 1.0
		if trendAcceleration > 0.001 && momentumAcceleration < trendAcceleration*0.1 {
			accelerationSync = 0.5 // 动量加速不足
		}

		features["acceleration_synchronization"] = accelerationSync
	}

	// 趋势动量强度比
	if hasSlope && hasMom5d {
		strengthRatio := math.Abs(momentum5d) / (math.Abs(trendSlope)*100 + 0.001)
		features["momentum_trend_strength_ratio"] = math.Min(strengthRatio, 2.0)

		// 强度平衡评估
		balanceScore := 1.0 - math.Abs(strengthRatio-1.0)
		features["momentum_trend_balance"] = math.Max(0, balanceScore)
	}
}

// calculateFeatureStability 计算特征稳定性
func (cfe *CrossFeatureExtractor) calculateFeatureStability(features map[string]float64) float64 {
	if len(features) == 0 {
		return 0.0
	}

	// 计算特征值的变异系数（标准差/均值）
	values := make([]float64, 0, len(features))
	for _, v := range features {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			values = append(values, v)
		}
	}

	if len(values) < 2 {
		return 1.0
	}

	mean := calculateAverage(values)
	std := calculateStandardDeviation(values, mean)

	if mean == 0 {
		return 0.0
	}

	coefficientOfVariation := std / math.Abs(mean)
	stability := 1.0 / (1.0 + coefficientOfVariation) // 变异系数越小，稳定性越高

	return stability
}

// calculateFeatureDiversity 计算特征多样性
func (cfe *CrossFeatureExtractor) calculateFeatureDiversity(features map[string]float64) float64 {
	if len(features) < 2 {
		return 0.0
	}

	// 计算特征值的熵（多样性度量）
	values := make([]float64, 0, len(features))
	for _, v := range features {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			values = append(values, math.Abs(v)+1) // 确保正值
		}
	}

	if len(values) < 2 {
		return 0.0
	}

	// 归一化值到0-1范围
	minVal := findMin(values)
	maxVal := findMax(values)
	range_ := maxVal - minVal

	if range_ == 0 {
		return 0.0
	}

	normalized := make([]float64, len(values))
	for i, v := range values {
		normalized[i] = (v - minVal) / range_
	}

	// 计算熵
	entropy := 0.0
	bins := 10
	hist := make([]int, bins)

	for _, v := range normalized {
		bin := int(v * float64(bins-1))
		if bin >= bins {
			bin = bins - 1
		}
		hist[bin]++
	}

	for _, count := range hist {
		if count > 0 {
			p := float64(count) / float64(len(values))
			entropy -= p * math.Log2(p)
		}
	}

	maxEntropy := math.Log2(float64(bins))
	diversity := entropy / maxEntropy

	return diversity
}

// calculateFeatureConsistency 计算特征一致性
func (cfe *CrossFeatureExtractor) calculateFeatureConsistency(features map[string]float64) float64 {
	if len(features) < 2 {
		return 1.0
	}

	// 计算特征之间的相关性平均值
	featureNames := make([]string, 0, len(features))
	values := make([]float64, 0, len(features))

	for name, value := range features {
		if !math.IsNaN(value) && !math.IsInf(value, 0) {
			featureNames = append(featureNames, name)
			values = append(values, value)
		}
	}

	if len(values) < 2 {
		return 1.0
	}

	// 计算所有特征对之间的相关性
	totalCorr := 0.0
	pairCount := 0

	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			corr := calculateCorrelation([]float64{values[i]}, []float64{values[j]})
			totalCorr += math.Abs(corr)
			pairCount++
		}
	}

	if pairCount == 0 {
		return 1.0
	}

	avgCorrelation := totalCorr / float64(pairCount)
	consistency := avgCorrelation // 高相关性表示一致性好

	return consistency
}

// detectFeatureOutliers 检测特征异常值
func (cfe *CrossFeatureExtractor) detectFeatureOutliers(features map[string]float64) float64 {
	if len(features) < 3 {
		return 0.0
	}

	values := make([]float64, 0, len(features))
	for _, v := range features {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			values = append(values, v)
		}
	}

	if len(values) < 3 {
		return 0.0
	}

	// 计算Z分数，检测异常值
	mean := calculateAverage(values)
	std := calculateStandardDeviation(values, mean)

	if std == 0 {
		return 0.0
	}

	outlierCount := 0
	for _, v := range values {
		zScore := math.Abs((v - mean) / std)
		if zScore > 2.5 { // 2.5倍标准差作为异常阈值
			outlierCount++
		}
	}

	outlierRatio := float64(outlierCount) / float64(len(values))
	return outlierRatio
}

// calculateFeatureStrength 计算特征强度
func (cfe *CrossFeatureExtractor) calculateFeatureStrength(features map[string]float64) float64 {
	if len(features) == 0 {
		return 0.0
	}

	// 计算特征的平均绝对值（强度度量）
	totalStrength := 0.0
	validCount := 0

	for _, v := range features {
		if !math.IsNaN(v) && !math.IsInf(v, 0) {
			totalStrength += math.Abs(v)
			validCount++
		}
	}

	if validCount == 0 {
		return 0.0
	}

	avgStrength := totalStrength / float64(validCount)

	// 归一化强度（0-1）
	normalizedStrength := math.Min(avgStrength, 1.0)

	return normalizedStrength
}

// =================== DMI交叉特征 ===================

// extractDMICrossFeatures 提取DMI交叉特征
func (cfe *CrossFeatureExtractor) extractDMICrossFeatures(baseFeatures map[string]float64, features map[string]float64) {
	dip, hasDIP := baseFeatures["dmi_dip"]
	dim, hasDIM := baseFeatures["dmi_dim"]
	adx, hasADX := baseFeatures["dmi_adx"]
	rsi, hasRSI := baseFeatures["rsi"]
	macd, hasMACD := baseFeatures["macd"]

	if !hasDIP || !hasDIM || !hasADX {
		return // 没有DMI数据，跳过
	}

	// 1. DMI强度特征
	dmiStrength := math.Abs(dip - dim)
	features["dmi_strength_normalized"] = dmiStrength / 100.0 // 归一化到0-1

	// 2. DMI趋势确认
	if dip > dim {
		features["dmi_trend_up"] = 1.0
		features["dmi_trend_down"] = 0.0
	} else {
		features["dmi_trend_up"] = 0.0
		features["dmi_trend_down"] = 1.0
	}

	// 3. ADX趋势强度分类
	if adx < 20 {
		features["adx_weak_trend"] = 1.0
		features["adx_strong_trend"] = 0.0
	} else if adx > 25 {
		features["adx_weak_trend"] = 0.0
		features["adx_strong_trend"] = 1.0
	} else {
		features["adx_weak_trend"] = 0.0
		features["adx_strong_trend"] = 0.0
	}

	// 4. DMI与RSI的交叉确认
	if hasRSI {
		rsiDMIConfirm := 0.0
		if (dip > dim && rsi > 50) || (dim > dip && rsi < 50) {
			rsiDMIConfirm = 1.0 // RSI与DMI方向一致
		}
		features["rsi_dmi_consistency"] = rsiDMIConfirm
	}

	// 5. DMI与MACD的交叉确认
	if hasMACD {
		macdDMIConfirm := 0.0
		if (dip > dim && macd > 0) || (dim > dip && macd < 0) {
			macdDMIConfirm = 1.0 // MACD与DMI方向一致
		}
		features["macd_dmi_consistency"] = macdDMIConfirm
	}

	// 6. DMI动量评分
	dmiMomentum := (dip + dim) / 2.0
	features["dmi_momentum_score"] = dmiMomentum / 100.0 // 归一化
}

// =================== Ichimoku交叉特征 ===================

// extractIchimokuCrossFeatures 提取Ichimoku交叉特征
func (cfe *CrossFeatureExtractor) extractIchimokuCrossFeatures(baseFeatures map[string]float64, features map[string]float64) {
	tenkan, hasTenkan := baseFeatures["ichimoku_tenkan"]
	kijun, hasKijun := baseFeatures["ichimoku_kijun"]
	spanA, hasSpanA := baseFeatures["ichimoku_span_a"]
	spanB, hasSpanB := baseFeatures["ichimoku_span_b"]
	currentPrice := baseFeatures["current_price"]

	if !hasTenkan || !hasKijun || !hasSpanA || !hasSpanB {
		return // 没有完整的Ichimoku数据
	}

	// 1. TK交叉信号强度
	tkDiff := tenkan - kijun
	features["ichimoku_tk_spread"] = tkDiff / currentPrice // 相对价格的价差

	// 2. TK交叉方向
	if tenkan > kijun {
		features["ichimoku_tk_bullish"] = 1.0
		features["ichimoku_tk_bearish"] = 0.0
	} else {
		features["ichimoku_tk_bullish"] = 0.0
		features["ichimoku_tk_bearish"] = 1.0
	}

	// 3. 云层厚度
	cloudThickness := math.Abs(spanA - spanB)
	features["ichimoku_cloud_thickness"] = cloudThickness / currentPrice // 相对厚度

	// 4. 云层颜色和位置
	cloudTop := math.Max(spanA, spanB)
	cloudBottom := math.Min(spanA, spanB)

	if currentPrice > cloudTop {
		features["ichimoku_cloud_position"] = 1.0 // 在云上
	} else if currentPrice < cloudBottom {
		features["ichimoku_cloud_position"] = -1.0 // 在云下
	} else {
		features["ichimoku_cloud_position"] = 0.0 // 在云中
	}

	// 5. 价格与基准线的相对位置
	kijunDistance := (currentPrice - kijun) / kijun
	features["ichimoku_price_vs_kijun"] = kijunDistance

	// 6. 价格与转换线的相对位置
	tenkanDistance := (currentPrice - tenkan) / tenkan
	features["ichimoku_price_vs_tenkan"] = tenkanDistance

	// 7. 云层支撑阻力强度
	if spanA > spanB {
		features["ichimoku_cloud_green"] = 1.0 // 绿色云（上涨）
		features["ichimoku_cloud_red"] = 0.0
	} else {
		features["ichimoku_cloud_green"] = 0.0
		features["ichimoku_cloud_red"] = 1.0 // 红色云（下跌）
	}

	// 8. 多重时间框架确认
	// TK在云上且价格在云上 = 强多头信号
	tkAboveCloud := tenkan > cloudTop && kijun > cloudTop
	priceAboveCloud := currentPrice > cloudTop

	if tkAboveCloud && priceAboveCloud {
		features["ichimoku_bullish_alignment"] = 1.0
	} else if tenkan < cloudBottom && kijun < cloudBottom && currentPrice < cloudBottom {
		features["ichimoku_bearish_alignment"] = 1.0
	} else {
		features["ichimoku_bullish_alignment"] = 0.0
		features["ichimoku_bearish_alignment"] = 0.0
	}
}

// calculateSimplifiedDMI 简化的DMI计算（基于价格变动趋势）
func (cfe *CrossFeatureExtractor) calculateSimplifiedDMI(prices []float64) float64 {
	if len(prices) < 14 {
		return 0.5 // 中性
	}

	// 计算价格变化的方向性
	upMoves := 0
	downMoves := 0

	for i := 1; i < len(prices); i++ {
		if prices[i] > prices[i-1] {
			upMoves++
		} else if prices[i] < prices[i-1] {
			downMoves++
		}
	}

	totalMoves := upMoves + downMoves
	if totalMoves == 0 {
		return 0.5
	}

	// 返回趋势强度（0-1之间）
	trendStrength := math.Abs(float64(upMoves-downMoves)) / float64(totalMoves)
	return trendStrength
}

// calculatePriceTrend 计算价格趋势方向
func (cfe *CrossFeatureExtractor) calculatePriceTrend(prices []float64) float64 {
	if len(prices) < 10 {
		return 0.0 // 中性
	}

	// 计算长期趋势（最近20个点）
	recentPrices := prices
	if len(prices) > 20 {
		recentPrices = prices[len(prices)-20:]
	}

	// 简单的趋势计算：比较开始和结束的价格
	startPrice := recentPrices[0]
	endPrice := recentPrices[len(recentPrices)-1]

	if startPrice == 0 {
		return 0.0
	}

	trend := (endPrice - startPrice) / startPrice

	// 归一化到-1到1之间
	return math.Max(-1.0, math.Min(1.0, trend*10)) // 放大趋势信号
}

// calculateSimplifiedIchimoku 简化的Ichimoku计算
func (cfe *CrossFeatureExtractor) calculateSimplifiedIchimoku(prices []float64) map[string]float64 {
	result := make(map[string]float64)

	if len(prices) < 20 {
		return result
	}

	// 简化的转换线：9日平均
	tenkanPeriod := 9
	if len(prices) >= tenkanPeriod {
		tenkanSum := 0.0
		for i := len(prices) - tenkanPeriod; i < len(prices); i++ {
			tenkanSum += prices[i]
		}
		result["ichimoku_tenkan"] = tenkanSum / float64(tenkanPeriod)
	}

	// 简化的基准线：26日平均
	kijunPeriod := 26
	if len(prices) >= kijunPeriod {
		kijunSum := 0.0
		for i := len(prices) - kijunPeriod; i < len(prices); i++ {
			kijunSum += prices[i]
		}
		result["ichimoku_kijun"] = kijunSum / float64(kijunPeriod)
	}

	// TK交叉信号
	tenkan, hasTenkan := result["ichimoku_tenkan"]
	kijun, hasKijun := result["ichimoku_kijun"]
	currentPrice := prices[len(prices)-1]

	if hasTenkan && hasKijun {
		// TK交叉方向
		if tenkan > kijun {
			result["ichimoku_tk_bullish"] = 1.0
			result["ichimoku_tk_bearish"] = 0.0
		} else {
			result["ichimoku_tk_bullish"] = 0.0
			result["ichimoku_tk_bearish"] = 1.0
		}

		// 价格相对位置
		result["ichimoku_price_vs_tenkan"] = (currentPrice - tenkan) / tenkan
		result["ichimoku_price_vs_kijun"] = (currentPrice - kijun) / kijun
	}

	return result
}
