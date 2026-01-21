package server

import (
	"context"
	"fmt"
	"math"
	"sort"
)

// StatisticalFeatureExtractor 统计特征提取器
type StatisticalFeatureExtractor struct {
	config FeatureConfig
}

// Name 返回提取器名称
func (sfe *StatisticalFeatureExtractor) Name() string {
	return "statistical"
}

// Priority 返回提取优先级
func (sfe *StatisticalFeatureExtractor) Priority() int {
	return 65
}

// Extract 提取统计特征
func (sfe *StatisticalFeatureExtractor) Extract(ctx context.Context, symbol string, currentData *MarketDataPoint, historyData []*MarketDataPoint) (map[string]float64, error) {
	features := make(map[string]float64)

	if len(historyData) < 10 {
		return features, fmt.Errorf("历史数据不足，至少需要10个数据点")
	}

	// 提取价格序列用于统计分析
	prices := make([]float64, len(historyData))
	for i, data := range historyData {
		prices[i] = data.Price
	}

	// 1. 基本统计特征
	sfe.extractBasicStatistics(prices, features)

	// 2. 分布特征
	sfe.extractDistributionFeatures(prices, features)

	// 3. 稳定性特征
	sfe.extractStabilityFeatures(prices, features)

	// 4. 异常检测特征
	sfe.extractAnomalyFeatures(prices, features)

	// 5. 时间序列统计特征
	sfe.extractTimeSeriesStats(prices, features)

	return features, nil
}

// extractBasicStatistics 提取基本统计特征
func (sfe *StatisticalFeatureExtractor) extractBasicStatistics(prices []float64, features map[string]float64) {
	n := len(prices)
	if n == 0 {
		return
	}

	// 均值
	mean := calculateAverage(prices)
	features["price_mean"] = mean

	// 方差和标准差
	variance := calculateVariance(prices, mean)
	stdDev := math.Sqrt(variance)
	features["price_variance"] = variance
	features["price_std"] = stdDev

	// 变异系数 (标准差/均值)
	if mean != 0 {
		coefficientOfVariation := stdDev / math.Abs(mean)
		features["price_cv"] = coefficientOfVariation
	}

	// 偏度 (skewness)
	skewness := calculateSkewness(prices)
	features["price_skewness"] = skewness

	// 峰度 (kurtosis)
	kurtosis := calculateKurtosis(prices)
	features["price_kurtosis"] = kurtosis

	// 分位数
	sort.Float64s(prices)
	features["price_median"] = prices[n/2]

	if n >= 4 {
		features["price_q25"] = prices[n/4]
		features["price_q75"] = prices[3*n/4]
		features["price_iqr"] = features["price_q75"] - features["price_q25"] // 四分位距
	}

	// 范围统计
	minPrice := prices[0]
	maxPrice := prices[n-1]
	features["price_range"] = maxPrice - minPrice
	features["price_range_ratio"] = (maxPrice - minPrice) / minPrice * 100
}

// extractDistributionFeatures 提取分布特征
func (sfe *StatisticalFeatureExtractor) extractDistributionFeatures(prices []float64, features map[string]float64) {
	n := len(prices)
	if n < 10 {
		return
	}

	// 正态分布检验 (Jarque-Bera测试的简化版)
	skewness := calculateSkewness(prices)
	kurtosis := calculateKurtosis(prices)

	// JB统计量 (简化计算)
	jbStat := float64(n) / 6 * (skewness*skewness + (kurtosis-3)*(kurtosis-3)/4)
	features["jarque_bera_stat"] = jbStat

	// 正态分布概率 (简化)
	normalProbability := math.Exp(-jbStat / 2)
	features["normality_probability"] = normalProbability

	// 分布对称性
	symmetry := 1.0 - math.Abs(skewness)/2 // 偏度越小，对称性越好
	features["distribution_symmetry"] = math.Max(0, symmetry)

	// 分布峰度特征
	if kurtosis > 3 {
		features["distribution_leptokurtic"] = 1.0 // 尖峰分布
		features["distribution_platykurtic"] = 0.0
	} else if kurtosis < 3 {
		features["distribution_leptokurtic"] = 0.0
		features["distribution_platykurtic"] = 1.0 // 平峰分布
	} else {
		features["distribution_leptokurtic"] = 0.0
		features["distribution_platykurtic"] = 0.0
	}

	// 分位数统计
	mean := calculateAverage(prices)
	median := prices[n/2]

	features["mean_median_diff"] = mean - median
	features["mean_median_ratio"] = mean / median

	// 分布集中度
	q25 := prices[n/4]
	q75 := prices[3*n/4]
	concentration := (q75 - q25) / (prices[n-1] - prices[0])
	features["distribution_concentration"] = concentration
}

// extractStabilityFeatures 提取稳定性特征
func (sfe *StatisticalFeatureExtractor) extractStabilityFeatures(prices []float64, features map[string]float64) {
	window := sfe.config.VolatilityWindow
	if len(prices) < window*2 {
		return
	}

	// 滚动统计特征
	rollingMeans := make([]float64, 0)
	rollingStdDevs := make([]float64, 0)

	for i := window; i <= len(prices); i++ {
		windowData := prices[i-window : i]
		mean := calculateAverage(windowData)
		stdDev := calculateStandardDeviation(windowData, mean)

		rollingMeans = append(rollingMeans, mean)
		rollingStdDevs = append(rollingStdDevs, stdDev)
	}

	// 均值稳定性
	if len(rollingMeans) >= 2 {
		meanStability := 1.0 - (calculateStandardDeviation(rollingMeans, calculateAverage(rollingMeans)) / (calculateAverage(rollingMeans) + 0.001))
		features["mean_stability"] = math.Max(0, meanStability)
	}

	// 波动率稳定性
	if len(rollingStdDevs) >= 2 {
		volatilityStability := 1.0 - (calculateStandardDeviation(rollingStdDevs, calculateAverage(rollingStdDevs)) / (calculateAverage(rollingStdDevs) + 0.001))
		features["volatility_stability"] = math.Max(0, volatilityStability)
	}

	// 统计特征的时序稳定性
	statStability := sfe.calculateStatisticalStability(prices, window)
	features["statistical_stability"] = statStability

	// 分布稳定性 (多个窗口的分布相似性)
	distributionStability := sfe.calculateDistributionStability(prices, window)
	features["distribution_stability"] = distributionStability
}

// extractAnomalyFeatures 提取异常检测特征
func (sfe *StatisticalFeatureExtractor) extractAnomalyFeatures(prices []float64, features map[string]float64) {
	n := len(prices)
	if n < 10 {
		return
	}

	// 计算Z分数检测异常值
	mean := calculateAverage(prices)
	stdDev := calculateStandardDeviation(prices, mean)

	if stdDev > 0 {
		currentPrice := prices[n-1]
		zScore := (currentPrice - mean) / stdDev
		features["price_z_score"] = zScore

		// 异常程度分类
		features["price_outlier_mild"] = boolToFloat(math.Abs(zScore) > 1.5)
		features["price_outlier_moderate"] = boolToFloat(math.Abs(zScore) > 2.0)
		features["price_outlier_extreme"] = boolToFloat(math.Abs(zScore) > 3.0)
	}

	// 基于IQR的异常检测
	q25 := prices[n/4]
	q75 := prices[3*n/4]
	iqr := q75 - q25

	if iqr > 0 {
		currentPrice := prices[n-1]

		// Tukey方法
		lowerFence := q25 - 1.5*iqr
		upperFence := q75 + 1.5*iqr

		features["price_below_lower_fence"] = boolToFloat(currentPrice < lowerFence)
		features["price_above_upper_fence"] = boolToFloat(currentPrice > upperFence)

		// 相对于IQR的位置
		if currentPrice >= q25 && currentPrice <= q75 {
			iqrPosition := (currentPrice - q25) / iqr
			features["price_iqr_position"] = iqrPosition
		}
	}

	// 异常点比例
	anomalyRatio := sfe.calculateAnomalyRatio(prices)
	features["anomaly_ratio"] = anomalyRatio

	// 异常聚集度
	anomalyClustering := sfe.calculateAnomalyClustering(prices)
	features["anomaly_clustering"] = anomalyClustering
}

// extractTimeSeriesStats 提取时间序列统计特征
func (sfe *StatisticalFeatureExtractor) extractTimeSeriesStats(prices []float64, features map[string]float64) {
	n := len(prices)
	if n < 20 {
		return
	}

	// 自相关特征
	autocorr1 := calculateCorrelation(prices[:n-1], prices[1:])
	autocorr5 := calculateCorrelation(prices[:n-5], prices[5:])
	autocorr10 := calculateCorrelation(prices[:n-10], prices[10:])

	features["autocorrelation_1"] = autocorr1
	features["autocorrelation_5"] = autocorr5
	features["autocorrelation_10"] = autocorr10

	// 自相关衰减
	autocorrDecay := 1.0
	if autocorr1 > 0 {
		autocorrDecay = autocorr10 / autocorr1
		if autocorrDecay < 0 {
			autocorrDecay = 0
		}
	}
	features["autocorrelation_decay"] = autocorrDecay

	// 周期性检测 (基于FFT的简化版)
	periodicity := sfe.detectPeriodicity(prices)
	features["series_periodicity"] = periodicity

	// 趋势站稳性 (detrended fluctuation analysis的简化版)
	trendStationarity := sfe.calculateTrendStationarity(prices)
	features["trend_stationarity"] = trendStationarity

	// 非线性特征
	nonlinearity := sfe.calculateNonlinearity(prices)
	features["series_nonlinearity"] = nonlinearity

	// 混沌程度 (基于lyapunov指数的简化计算)
	chaosMeasure := sfe.calculateChaosMeasure(prices)
	features["chaos_measure"] = chaosMeasure
}

// calculateStatisticalStability 计算统计稳定性
func (sfe *StatisticalFeatureExtractor) calculateStatisticalStability(prices []float64, window int) float64 {
	if len(prices) < window*3 {
		return 0.5
	}

	// 计算多个窗口的统计特征
	means := make([]float64, 0)
	stdDevs := make([]float64, 0)
	skewnesses := make([]float64, 0)

	for i := window; i <= len(prices); i += window / 2 {
		windowData := prices[max(0, i-window):i]
		if len(windowData) >= 5 {
			mean := calculateAverage(windowData)
			stdDev := calculateStandardDeviation(windowData, mean)
			skewness := calculateSkewness(windowData)

			means = append(means, mean)
			stdDevs = append(stdDevs, stdDev)
			skewnesses = append(skewnesses, skewness)
		}
	}

	// 计算统计特征的变异性
	stabilityScore := 0.0
	count := 0

	if len(means) >= 2 {
		meanStability := 1.0 - calculateStandardDeviation(means, calculateAverage(means))/(calculateAverage(means)+0.001)
		stabilityScore += meanStability
		count++
	}

	if len(stdDevs) >= 2 {
		stdStability := 1.0 - calculateStandardDeviation(stdDevs, calculateAverage(stdDevs))/(calculateAverage(stdDevs)+0.001)
		stabilityScore += stdStability
		count++
	}

	if len(skewnesses) >= 2 {
		skewStability := 1.0 - calculateStandardDeviation(skewnesses, calculateAverage(skewnesses))/(calculateAverage(skewnesses)+0.001)
		stabilityScore += skewStability
		count++
	}

	if count == 0 {
		return 0.5
	}

	return stabilityScore / float64(count)
}

// calculateDistributionStability 计算分布稳定性
func (sfe *StatisticalFeatureExtractor) calculateDistributionStability(prices []float64, window int) float64 {
	if len(prices) < window*3 {
		return 0.5
	}

	// 计算多个窗口的分布特征
	distributions := make([][]float64, 0)

	for i := window; i <= len(prices); i += window / 2 {
		windowData := prices[max(0, i-window):i]
		if len(windowData) >= 5 {
			// 计算百分位数作为分布特征
			sort.Float64s(windowData)
			percentiles := []float64{
				windowData[len(windowData)/4],   // Q25
				windowData[len(windowData)/2],   // Q50
				windowData[3*len(windowData)/4], // Q75
			}
			distributions = append(distributions, percentiles)
		}
	}

	if len(distributions) < 2 {
		return 0.5
	}

	// 计算分布之间的一致性
	totalSimilarity := 0.0
	pairCount := 0

	for i := 0; i < len(distributions)-1; i++ {
		for j := i + 1; j < len(distributions); j++ {
			similarity := 0.0
			for k := 0; k < len(distributions[i]); k++ {
				if distributions[j][k] != 0 {
					diff := math.Abs(distributions[i][k] - distributions[j][k])
					relDiff := diff / math.Abs(distributions[j][k])
					similarity += 1.0 / (1.0 + relDiff)
				}
			}
			similarity /= float64(len(distributions[i]))
			totalSimilarity += similarity
			pairCount++
		}
	}

	if pairCount == 0 {
		return 0.5
	}

	averageSimilarity := totalSimilarity / float64(pairCount)
	return averageSimilarity
}

// calculateAnomalyRatio 计算异常值比例
func (sfe *StatisticalFeatureExtractor) calculateAnomalyRatio(prices []float64) float64 {
	n := len(prices)
	if n < 10 {
		return 0.0
	}

	mean := calculateAverage(prices)
	stdDev := calculateStandardDeviation(prices, mean)

	if stdDev == 0 {
		return 0.0
	}

	anomalyCount := 0
	for _, price := range prices {
		zScore := math.Abs((price - mean) / stdDev)
		if zScore > 2.5 {
			anomalyCount++
		}
	}

	return float64(anomalyCount) / float64(n)
}

// calculateAnomalyClustering 计算异常聚集度
func (sfe *StatisticalFeatureExtractor) calculateAnomalyClustering(prices []float64) float64 {
	n := len(prices)
	if n < 10 {
		return 0.0
	}

	mean := calculateAverage(prices)
	stdDev := calculateStandardDeviation(prices, mean)

	if stdDev == 0 {
		return 0.0
	}

	// 识别异常点
	anomalies := make([]bool, n)
	anomalyCount := 0

	for i, price := range prices {
		zScore := math.Abs((price - mean) / stdDev)
		if zScore > 2.0 {
			anomalies[i] = true
			anomalyCount++
		}
	}

	if anomalyCount < 2 {
		return 0.0
	}

	// 计算异常点之间的平均距离
	distances := make([]int, 0)
	lastAnomalyIndex := -1

	for i, isAnomaly := range anomalies {
		if isAnomaly {
			if lastAnomalyIndex >= 0 {
				distances = append(distances, i-lastAnomalyIndex)
			}
			lastAnomalyIndex = i
		}
	}

	if len(distances) == 0 {
		return 1.0 // 所有异常点都相邻
	}

	avgDistance := calculateAverage(intSliceToFloatSlice(distances))
	clustering := 1.0 / (1.0 + avgDistance/10.0) // 距离越小，聚集度越高

	return clustering
}

// detectPeriodicity 检测周期性 (简化版)
func (sfe *StatisticalFeatureExtractor) detectPeriodicity(prices []float64) float64 {
	n := len(prices)
	if n < 20 {
		return 0.0
	}

	// 计算不同滞后的自相关
	maxLag := min(10, n/3)
	autocorrs := make([]float64, maxLag)

	for lag := 1; lag < maxLag; lag++ {
		autocorrs[lag] = math.Abs(calculateCorrelation(prices[:n-lag], prices[lag:]))
	}

	// 找到最强的自相关
	maxAutocorr := 0.0
	for _, ac := range autocorrs {
		if ac > maxAutocorr {
			maxAutocorr = ac
		}
	}

	// 周期性强度
	periodicity := maxAutocorr * (1.0 - 1.0/float64(maxLag)) // 考虑多个候选周期

	return math.Min(periodicity, 1.0)
}

// calculateTrendStationarity 计算趋势平稳性
func (sfe *StatisticalFeatureExtractor) calculateTrendStationarity(prices []float64) float64 {
	n := len(prices)
	if n < 20 {
		return 0.5
	}

	// 计算累积和 (积分)
	cumsum := make([]float64, n+1)
	for i := 1; i <= n; i++ {
		cumsum[i] = cumsum[i-1] + prices[i-1]
	}

	// 去趋势
	detrended := make([]float64, n)
	for i := 0; i < n; i++ {
		// 简单的线性去趋势
		expected := cumsum[n] * float64(i+1) / float64(n)
		detrended[i] = cumsum[i+1] - expected
	}

	// 计算去趋势数据的方差
	detrendedMean := calculateAverage(detrended)
	detrendedVariance := calculateVariance(detrended, detrendedMean)

	// 计算原始数据的方差
	originalMean := calculateAverage(prices)
	originalVariance := calculateVariance(prices, originalMean)

	if originalVariance == 0 {
		return 1.0
	}

	// 平稳性 = 1 - (去趋势方差 / 原始方差)
	stationarity := 1.0 - detrendedVariance/originalVariance

	return math.Max(0, math.Min(1, stationarity))
}

// calculateNonlinearity 计算非线性程度
func (sfe *StatisticalFeatureExtractor) calculateNonlinearity(prices []float64) float64 {
	n := len(prices)
	if n < 10 {
		return 0.0
	}

	// 使用延迟嵌入重建吸引子
	embedding := make([][]float64, n-2)
	for i := 0; i < n-2; i++ {
		embedding[i] = []float64{prices[i], prices[i+1], prices[i+2]}
	}

	// 计算嵌入空间中的非线性特征
	// 这里使用简化的非线性检测：检查预测误差

	predictionErrors := make([]float64, 0)
	for i := 3; i < n; i++ {
		// 简单的线性预测
		predicted := prices[i-1] + (prices[i-1] - prices[i-2])
		actualError := math.Abs(predicted - prices[i])
		relativeError := actualError / (math.Abs(prices[i]) + 0.001)
		predictionErrors = append(predictionErrors, relativeError)
	}

	if len(predictionErrors) == 0 {
		return 0.0
	}

	avgPredictionError := calculateAverage(predictionErrors)

	// 非线性程度与预测误差相关
	nonlinearity := math.Min(avgPredictionError*10, 1.0)

	return nonlinearity
}

// calculateChaosMeasure 计算混沌度量 (简化的Lyapunov指数)
func (sfe *StatisticalFeatureExtractor) calculateChaosMeasure(prices []float64) float64 {
	n := len(prices)
	if n < 20 {
		return 0.0
	}

	// 计算相邻轨迹的发散程度
	divergences := make([]float64, 0)

	for lag := 1; lag <= 5; lag++ {
		divergence := 0.0
		count := 0

		for i := lag; i < n-lag; i++ {
			// 计算短期发散
			shortTermDiff := math.Abs(prices[i] - prices[i-lag])
			longTermDiff := math.Abs(prices[i+lag] - prices[i])

			if shortTermDiff > 0 {
				div := longTermDiff / shortTermDiff
				divergence += div
				count++
			}
		}

		if count > 0 {
			divergences = append(divergences, divergence/float64(count))
		}
	}

	if len(divergences) == 0 {
		return 0.0
	}

	// 平均发散程度
	avgDivergence := calculateAverage(divergences)

	// 归一化到0-1范围 (经验值)
	chaosMeasure := math.Min(avgDivergence/2.0, 1.0)

	return chaosMeasure
}
