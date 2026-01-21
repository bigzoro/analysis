package server

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"
)

// TimeSeriesFeatureExtractor 时间序列特征提取器
type TimeSeriesFeatureExtractor struct {
	config FeatureConfig
}

// Name 返回提取器名称
func (tsfe *TimeSeriesFeatureExtractor) Name() string {
	return "time_series"
}

// Priority 返回提取优先级
func (tsfe *TimeSeriesFeatureExtractor) Priority() int {
	return 90 // 高优先级
}

// Extract 提取时间序列特征
func (tsfe *TimeSeriesFeatureExtractor) Extract(ctx context.Context, symbol string, currentData *MarketDataPoint, historyData []*MarketDataPoint) (map[string]float64, error) {
	features := make(map[string]float64)

	if len(historyData) < 2 {
		return features, fmt.Errorf("历史数据不足，至少需要2个数据点")
	}

	// 提取价格序列
	prices := make([]float64, len(historyData))
	volumes := make([]float64, len(historyData))
	timestamps := make([]time.Time, len(historyData))

	for i, data := range historyData {
		prices[i] = data.Price
		volumes[i] = data.Volume24h
		timestamps[i] = data.Timestamp
	}

	// 1. 价格动量特征
	tsfe.extractPriceMomentumFeatures(prices, features)

	// 2. 成交量动量特征
	tsfe.extractVolumeMomentumFeatures(volumes, features)

	// 3. 时间间隔特征
	tsfe.extractTimeIntervalFeatures(timestamps, features)

	// 4. 价格波动模式特征
	tsfe.extractPricePatternFeatures(prices, features)

	// 5. 季节性特征
	tsfe.extractSeasonalFeatures(timestamps, prices, features)

	// 6. 高级时序特征
	tsfe.extractAdvancedTimeSeriesFeatures(prices, volumes, features)

	return features, nil
}

// extractPriceMomentumFeatures 提取价格动量特征
func (tsfe *TimeSeriesFeatureExtractor) extractPriceMomentumFeatures(prices []float64, features map[string]float64) {
	n := len(prices)

	// 短期动量 (1-4小时)
	if n >= 5 {
		features["price_momentum_1h"] = (prices[n-1] - prices[n-2]) / prices[n-2] * 100
		features["price_momentum_4h"] = (prices[n-1] - prices[n-5]) / prices[n-5] * 100
	}

	// 中期动量 (12-24小时)
	if n >= 25 {
		features["price_momentum_12h"] = (prices[n-1] - prices[n-13]) / prices[n-13] * 100
		features["price_momentum_24h"] = (prices[n-1] - prices[n-25]) / prices[n-25] * 100
	}

	// 长期动量 (3-7天)
	if n >= 168 { // 7天 * 24小时
		features["price_momentum_3d"] = (prices[n-1] - prices[n-73]) / prices[n-73] * 100
		features["price_momentum_7d"] = (prices[n-1] - prices[n-168]) / prices[n-168] * 100
	}

	// 动量加速 (动量的变化率)
	if n >= 9 {
		momentum4h_1 := (prices[n-5] - prices[n-9]) / prices[n-9] * 100
		momentum4h_2 := features["price_momentum_4h"]
		features["momentum_acceleration_4h"] = momentum4h_2 - momentum4h_1
	}

	// 动量一致性 (连续动量方向)
	consecutiveUp := 0
	consecutiveDown := 0
	for i := n - 2; i >= max(0, n-10); i-- {
		if prices[i+1] > prices[i] {
			consecutiveUp++
			consecutiveDown = 0
		} else if prices[i+1] < prices[i] {
			consecutiveDown++
			consecutiveUp = 0
		} else {
			break
		}
	}
	features["momentum_consistency_up"] = float64(consecutiveUp)
	features["momentum_consistency_down"] = float64(consecutiveDown)
}

// extractVolumeMomentumFeatures 提取成交量动量特征
func (tsfe *TimeSeriesFeatureExtractor) extractVolumeMomentumFeatures(volumes []float64, features map[string]float64) {
	n := len(volumes)
	if n < 2 {
		return
	}

	// 成交量比率
	features["volume_current"] = volumes[n-1]
	if n >= 2 {
		features["volume_ratio_1h"] = volumes[n-1] / volumes[n-2]
	}
	if n >= 24 {
		avgVolume24h := calculateAverage(volumes[n-24:])
		features["volume_ratio_24h"] = volumes[n-1] / avgVolume24h
	}

	// 成交量趋势
	if n >= 10 {
		recentVolumes := volumes[max(0, n-10):]
		volumeTrend := calculateLinearTrend(recentVolumes)
		features["volume_trend_10h"] = volumeTrend
	}

	// 高成交量标识
	if n >= 24 {
		maxVolume24h := findMax(volumes[n-24:])
		features["volume_is_high"] = boolToFloat(volumes[n-1] > maxVolume24h*0.8)
	}
}

// extractTimeIntervalFeatures 提取时间间隔特征
func (tsfe *TimeSeriesFeatureExtractor) extractTimeIntervalFeatures(timestamps []time.Time, features map[string]float64) {
	if len(timestamps) < 2 {
		return
	}

	// 计算平均时间间隔
	intervals := make([]float64, len(timestamps)-1)
	for i := 0; i < len(timestamps)-1; i++ {
		intervals[i] = timestamps[i+1].Sub(timestamps[i]).Minutes()
	}

	avgInterval := calculateAverage(intervals)
	features["avg_time_interval_minutes"] = avgInterval

	// 时间间隔一致性
	intervalVariance := calculateVariance(intervals, avgInterval)
	features["time_interval_variance"] = intervalVariance

	// 检测数据更新频率
	if avgInterval < 10 { // 小于10分钟
		features["update_frequency"] = 3 // 高频
	} else if avgInterval < 60 { // 小于1小时
		features["update_frequency"] = 2 // 中频
	} else {
		features["update_frequency"] = 1 // 低频
	}
}

// extractPricePatternFeatures 提取价格模式特征
func (tsfe *TimeSeriesFeatureExtractor) extractPricePatternFeatures(prices []float64, features map[string]float64) {
	n := len(prices)
	if n < 10 {
		return
	}

	// 价格范围特征
	minPrice := findMin(prices)
	maxPrice := findMax(prices)
	currentPrice := prices[n-1]

	features["price_range_ratio"] = (maxPrice - minPrice) / minPrice * 100
	features["price_position_in_range"] = (currentPrice - minPrice) / (maxPrice - minPrice)

	// 价格分布特征
	meanPrice := calculateAverage(prices)
	stdDev := calculateStandardDeviation(prices, meanPrice)

	features["price_z_score"] = (currentPrice - meanPrice) / stdDev
	features["price_volatility_ratio"] = stdDev / meanPrice * 100

	// 价格趋势特征
	if n >= 20 {
		shortTrend := calculateLinearTrend(prices[n-10:])
		longTrend := calculateLinearTrend(prices[n-20:])

		features["price_trend_short"] = shortTrend
		features["price_trend_long"] = longTrend
		features["trend_divergence"] = shortTrend - longTrend
	}
}

// extractSeasonalFeatures 提取季节性特征
func (tsfe *TimeSeriesFeatureExtractor) extractSeasonalFeatures(timestamps []time.Time, prices []float64, features map[string]float64) {
	if len(timestamps) < 24 {
		return
	}

	// 小时模式 (0-23)
	hour := timestamps[len(timestamps)-1].Hour()
	features["hour_of_day"] = float64(hour)
	features["is_market_hours"] = boolToFloat(hour >= 9 && hour <= 16) // 假设美股时间

	// 星期模式 (0-6, 0=周日)
	weekday := int(timestamps[len(timestamps)-1].Weekday())
	features["day_of_week"] = float64(weekday)
	features["is_weekend"] = boolToFloat(weekday == 0 || weekday == 6)

	// 月度模式
	month := timestamps[len(timestamps)-1].Month()
	features["month_of_year"] = float64(month)

	// 季节性价格模式 (基于历史数据)
	if len(prices) >= 24 {
		// 计算当前小时与24小时前同小时的价格对比
		currentHour := hour
		dayAgoIndex := len(prices) - 24

		if dayAgoIndex >= 0 {
			dayAgoPrice := prices[dayAgoIndex]
			features["price_vs_day_ago_same_hour"] = (prices[len(prices)-1] - dayAgoPrice) / dayAgoPrice * 100
		}

		// 计算当前小时的历史平均价格
		hourPrices := make([]float64, 0)
		for i, ts := range timestamps {
			if ts.Hour() == currentHour {
				hourPrices = append(hourPrices, prices[i])
			}
		}

		if len(hourPrices) > 0 {
			hourAvg := calculateAverage(hourPrices)
			features["price_vs_hourly_average"] = (prices[len(prices)-1] - hourAvg) / hourAvg * 100
		}
	}
}

// 辅助函数

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateVariance(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	return sumSquares / float64(len(values)-1)
}

func calculateStandardDeviation(values []float64, mean float64) float64 {
	return math.Sqrt(calculateVariance(values, mean))
}

func calculateLinearTrend(values []float64) float64 {
	n := float64(len(values))
	if n < 2 {
		return 0
	}

	// 简单线性回归斜率
	sumX := n * (n - 1) / 2
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, y := range values {
		x := float64(i)
		sumY += y
		sumXY += x * y
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	return slope
}

func findMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func findMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

// extractAdvancedTimeSeriesFeatures 提取高级时序特征
func (tsfe *TimeSeriesFeatureExtractor) extractAdvancedTimeSeriesFeatures(prices []float64, volumes []float64, features map[string]float64) {
	n := len(prices)
	if n < 20 {
		return // 需要足够的数据
	}

	// 1. 价格序列的自相关特征
	tsfe.extractAutocorrelationFeatures(prices, features)

	// 2. 分形维数特征 (简化版)
	tsfe.extractFractalFeatures(prices, features)

	// 3. 价格分布特征
	tsfe.extractDistributionFeatures(prices, features)

	// 4. 价格-成交量协同特征
	tsfe.extractPriceVolumeSynergyFeatures(prices, volumes, features)

	// 5. 市场微观结构特征
	tsfe.extractMicrostructureFeatures(prices, volumes, features)
}

// extractAutocorrelationFeatures 提取自相关特征
func (tsfe *TimeSeriesFeatureExtractor) extractAutocorrelationFeatures(prices []float64, features map[string]float64) {
	n := len(prices)
	if n < 30 {
		return
	}

	// 计算价格序列的自相关系数 (滞后1-5期)
	for lag := 1; lag <= 5 && lag < n; lag++ {
		autocorr := tsfe.calculateAutocorrelation(prices, lag)
		features[fmt.Sprintf("price_autocorr_lag%d", lag)] = autocorr
	}

	// 自相关稳定性 (连续滞后相关系数的方差)
	if n >= 10 {
		autocorrs := make([]float64, 5)
		for i := 0; i < 5 && i+1 < n; i++ {
			autocorrs[i] = tsfe.calculateAutocorrelation(prices, i+1)
		}
		stability := tsfe.calculateVariance(autocorrs)
		features["autocorr_stability"] = stability
	}
}

// calculateAutocorrelation 计算自相关系数
func (tsfe *TimeSeriesFeatureExtractor) calculateAutocorrelation(data []float64, lag int) float64 {
	n := len(data)
	if n <= lag {
		return 0
	}

	// 计算均值
	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(n)

	// 计算自相关
	numerator := 0.0
	denom1 := 0.0
	denom2 := 0.0

	for i := lag; i < n; i++ {
		diff1 := data[i] - mean
		diff2 := data[i-lag] - mean
		numerator += diff1 * diff2
		denom1 += diff1 * diff1
		denom2 += diff2 * diff2
	}

	if denom1 > 0 && denom2 > 0 {
		return numerator / math.Sqrt(denom1*denom2)
	}
	return 0
}

// extractFractalFeatures 提取分形特征 (简化版)
func (tsfe *TimeSeriesFeatureExtractor) extractFractalFeatures(prices []float64, features map[string]float64) {
	n := len(prices)
	if n < 50 {
		return
	}

	// 基于Hurst指数的简化分形特征
	// Hurst指数 > 0.5 表示持续性，< 0.5 表示反转性
	hurst := tsfe.calculateSimplifiedHurstExponent(prices)
	features["hurst_exponent"] = hurst

	// 分形维数 (基于价格波动的复杂度)
	fractalDim := tsfe.calculateFractalDimension(prices)
	features["fractal_dimension"] = fractalDim
}

// calculateSimplifiedHurstExponent 计算简化的Hurst指数
func (tsfe *TimeSeriesFeatureExtractor) calculateSimplifiedHurstExponent(prices []float64) float64 {
	n := len(prices)
	if n < 20 {
		return 0.5
	}

	// 计算价格增量的累积和
	cumsum := make([]float64, n)
	cumsum[0] = 0
	for i := 1; i < n; i++ {
		cumsum[i] = cumsum[i-1] + (prices[i] - prices[i-1])
	}

	// 计算不同时间尺度的标准差
	scales := []int{5, 10, 20, 40}
	var logScales []float64
	var logStds []float64

	for _, scale := range scales {
		if scale >= n {
			continue
		}

		std := tsfe.calculateRescaledRangeStd(cumsum, scale)
		if std > 0 {
			logScales = append(logScales, math.Log(float64(scale)))
			logStds = append(logStds, math.Log(std))
		}
	}

	if len(logScales) < 2 {
		return 0.5
	}

	// 线性回归得到Hurst指数
	hurst := tsfe.linearRegressionSlope(logScales, logStds)
	return math.Max(0, math.Min(1, hurst)) // 限制在[0,1]范围内
}

// calculateRescaledRangeStd 计算重标度范围标准差
func (tsfe *TimeSeriesFeatureExtractor) calculateRescaledRangeStd(cumsum []float64, scale int) float64 {
	n := len(cumsum)
	if scale >= n {
		return 0
	}

	// 计算每个子区间的范围标准化标准差
	var stds []float64
	for i := 0; i <= n-scale; i += scale {
		end := i + scale
		if end > n {
			end = n
		}

		subset := cumsum[i:end]
		if len(subset) < 2 {
			continue
		}

		mean := 0.0
		for _, v := range subset {
			mean += v
		}
		mean /= float64(len(subset))

		variance := 0.0
		for _, v := range subset {
			diff := v - mean
			variance += diff * diff
		}
		variance /= float64(len(subset))

		if variance > 0 {
			stds = append(stds, math.Sqrt(variance))
		}
	}

	if len(stds) == 0 {
		return 0
	}

	// 返回平均标准差
	sum := 0.0
	for _, std := range stds {
		sum += std
	}
	return sum / float64(len(stds))
}

// extractDistributionFeatures 提取价格分布特征
func (tsfe *TimeSeriesFeatureExtractor) extractDistributionFeatures(prices []float64, features map[string]float64) {
	n := len(prices)
	if n < 20 {
		return
	}

	// 计算统计矩
	mean := tsfe.calculateMean(prices)
	std := tsfe.calculateStd(prices, mean)

	features["price_skewness"] = tsfe.calculateSkewness(prices, mean, std)
	features["price_kurtosis"] = tsfe.calculateKurtosis(prices, mean, std)

	// 价格分布不对称性
	median := tsfe.calculateMedian(prices)
	features["price_median_deviation"] = (mean - median) / std

	// 分位数特征
	q25, q75 := tsfe.calculateQuartiles(prices)
	if q75 > q25 {
		features["price_iqr"] = q75 - q25
		features["price_iqr_normalized"] = (q75 - q25) / median
	}
}

// extractPriceVolumeSynergyFeatures 提取价格-成交量协同特征
func (tsfe *TimeSeriesFeatureExtractor) extractPriceVolumeSynergyFeatures(prices []float64, volumes []float64, features map[string]float64) {
	n := len(prices)
	if n < 20 || len(volumes) != n {
		return
	}

	// 价格-成交量相关性
	corr := tsfe.calculateCorrelation(prices, volumes)
	features["price_volume_correlation"] = corr

	// 成交量加权价格动量
	if n >= 10 {
		volumeWeightedMomentum := tsfe.calculateVolumeWeightedMomentum(prices, volumes, 10)
		features["volume_weighted_momentum_10"] = volumeWeightedMomentum
	}

	// 高成交量价格变动
	highVolumeThreshold := tsfe.calculatePercentile(volumes, 0.8)
	highVolumePriceChanges := make([]float64, 0)

	for i := 1; i < n; i++ {
		if volumes[i] >= highVolumeThreshold {
			priceChange := (prices[i] - prices[i-1]) / prices[i-1]
			highVolumePriceChanges = append(highVolumePriceChanges, priceChange)
		}
	}

	if len(highVolumePriceChanges) > 0 {
		features["high_volume_price_volatility"] = tsfe.calculateStd(highVolumePriceChanges, tsfe.calculateMean(highVolumePriceChanges))
		features["high_volume_price_trend"] = tsfe.calculateMean(highVolumePriceChanges)
	}
}

// extractMicrostructureFeatures 提取市场微观结构特征
func (tsfe *TimeSeriesFeatureExtractor) extractMicrostructureFeatures(prices []float64, volumes []float64, features map[string]float64) {
	n := len(prices)
	if n < 30 || len(volumes) != n {
		return
	}

	// 价格冲击 (成交量与价格变动的关系)
	priceImpacts := make([]float64, n-1)
	for i := 1; i < n; i++ {
		priceChange := math.Abs(prices[i]-prices[i-1]) / prices[i-1]
		volume := volumes[i]
		if volume > 0 {
			priceImpacts[i-1] = priceChange / volume
		}
	}

	if len(priceImpacts) > 0 {
		features["avg_price_impact"] = tsfe.calculateMean(priceImpacts)
		features["price_impact_volatility"] = tsfe.calculateStd(priceImpacts, tsfe.calculateMean(priceImpacts))
	}

	// 成交量分布特征
	volumeMean := tsfe.calculateMean(volumes)
	volumeStd := tsfe.calculateStd(volumes, volumeMean)
	features["volume_concentration"] = volumeStd / volumeMean

	// 价格跳跃检测
	jumps := 0
	for i := 1; i < n; i++ {
		priceChange := math.Abs(prices[i]-prices[i-1]) / prices[i-1]
		if priceChange > 0.02 { // 2%的价格跳跃
			jumps++
		}
	}
	features["price_jump_frequency"] = float64(jumps) / float64(n-1)
}

// 辅助统计函数
func (tsfe *TimeSeriesFeatureExtractor) calculateMean(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	return sum / float64(len(data))
}

func (tsfe *TimeSeriesFeatureExtractor) calculateStd(data []float64, mean float64) float64 {
	if len(data) < 2 {
		return 0
	}
	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(data) - 1)
	return math.Sqrt(variance)
}

func (tsfe *TimeSeriesFeatureExtractor) calculateVariance(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}
	mean := tsfe.calculateMean(data)
	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}
	return variance / float64(len(data)-1)
}

func (tsfe *TimeSeriesFeatureExtractor) calculateSkewness(data []float64, mean, std float64) float64 {
	if len(data) < 3 || std == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		diff := v - mean
		sum += diff * diff * diff
	}
	n := float64(len(data))
	return (sum / n) / (std * std * std)
}

func (tsfe *TimeSeriesFeatureExtractor) calculateKurtosis(data []float64, mean, std float64) float64 {
	if len(data) < 4 || std == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		diff := v - mean
		sum += diff * diff * diff * diff
	}
	n := float64(len(data))
	return (sum/n)/(std*std*std*std) - 3 // 减去3得到超额峰度
}

func (tsfe *TimeSeriesFeatureExtractor) calculateMedian(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func (tsfe *TimeSeriesFeatureExtractor) calculateQuartiles(data []float64) (float64, float64) {
	if len(data) < 4 {
		return 0, 0
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	n := len(sorted)
	q25Idx := n / 4
	q75Idx := 3 * n / 4
	return sorted[q25Idx], sorted[q75Idx]
}

func (tsfe *TimeSeriesFeatureExtractor) calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}
	n := len(x)
	meanX := tsfe.calculateMean(x)
	meanY := tsfe.calculateMean(y)

	numerator := 0.0
	sumX2 := 0.0
	sumY2 := 0.0

	for i := 0; i < n; i++ {
		diffX := x[i] - meanX
		diffY := y[i] - meanY
		numerator += diffX * diffY
		sumX2 += diffX * diffX
		sumY2 += diffY * diffY
	}

	denom := math.Sqrt(sumX2 * sumY2)
	if denom == 0 {
		return 0
	}
	return numerator / denom
}

func (tsfe *TimeSeriesFeatureExtractor) calculateVolumeWeightedMomentum(prices, volumes []float64, window int) float64 {
	n := len(prices)
	if n < window+1 || len(volumes) != n {
		return 0
	}

	totalWeight := 0.0
	weightedChange := 0.0

	for i := n - window; i < n; i++ {
		if i > 0 {
			priceChange := (prices[i] - prices[i-1]) / prices[i-1]
			weight := volumes[i]
			weightedChange += priceChange * weight
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		return 0
	}
	return weightedChange / totalWeight
}

func (tsfe *TimeSeriesFeatureExtractor) calculatePercentile(data []float64, p float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)-1) * p)
	return sorted[idx]
}

func (tsfe *TimeSeriesFeatureExtractor) linearRegressionSlope(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}
	n := float64(len(x))

	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	numerator := n*sumXY - sumX*sumY
	denominator := n*sumX2 - sumX*sumX

	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func (tsfe *TimeSeriesFeatureExtractor) calculateFractalDimension(prices []float64) float64 {
	n := len(prices)
	if n < 10 {
		return 1.0
	}

	// 简化的分形维数计算：基于价格波动的复杂度
	totalVariation := 0.0
	for i := 1; i < n; i++ {
		totalVariation += math.Abs(prices[i] - prices[i-1])
	}

	avgPrice := tsfe.calculateMean(prices)
	if avgPrice == 0 {
		return 1.0
	}

	normalizedVariation := totalVariation / (avgPrice * float64(n-1))

	// 将变异系数映射到分形维数 (1.0-1.5)
	// 低变异 -> 接近1.0, 高变异 -> 接近1.5
	return 1.0 + math.Min(0.5, normalizedVariation*2)
}
