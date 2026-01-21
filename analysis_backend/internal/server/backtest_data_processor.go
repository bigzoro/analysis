package server

import (
	"math"
	"sort"
)

// DataPreprocessor 数据预处理器
type DataPreprocessor struct{}

// NewDataPreprocessor 创建数据预处理器
func NewDataPreprocessor() *DataPreprocessor {
	return &DataPreprocessor{}
}

// DetectOutliers 检测异常值
func (dp *DataPreprocessor) DetectOutliers(data []MarketData, method string) []OutlierInfo {
	outliers := make([]OutlierInfo, 0)

	switch method {
	case "iqr":
		outliers = dp.detectOutliersIQR(data)
	case "zscore":
		outliers = dp.detectOutliersZScore(data)
	case "modified_zscore":
		outliers = dp.detectOutliersModifiedZScore(data)
	default:
		outliers = dp.detectOutliersIQR(data)
	}

	return outliers
}

// detectOutliersIQR 使用IQR方法检测异常值
func (dp *DataPreprocessor) detectOutliersIQR(data []MarketData) []OutlierInfo {
	if len(data) < 4 {
		return nil
	}

	prices := make([]float64, len(data))
	for i, md := range data {
		prices[i] = md.Price
	}

	sort.Float64s(prices)

	// 计算四分位数
	q1 := prices[len(prices)/4]
	q3 := prices[3*len(prices)/4]
	iqr := q3 - q1

	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	outliers := make([]OutlierInfo, 0)
	for i, md := range data {
		if md.Price < lowerBound || md.Price > upperBound {
			outliers = append(outliers, OutlierInfo{
				Index:     i,
				Value:     md.Price,
				Method:    "IQR",
				Threshold: lowerBound,
				IsUpper:   md.Price > upperBound,
			})
		}
	}

	return outliers
}

// detectOutliersZScore 使用Z-Score方法检测异常值
func (dp *DataPreprocessor) detectOutliersZScore(data []MarketData) []OutlierInfo {
	if len(data) < 2 {
		return nil
	}

	prices := make([]float64, len(data))
	for i, md := range data {
		prices[i] = md.Price
	}

	mean, std := dp.calculateMeanStd(prices)
	if std == 0 {
		return nil
	}

	outliers := make([]OutlierInfo, 0)
	for i, md := range data {
		zScore := math.Abs((md.Price - mean) / std)
		if zScore > 3.0 { // 3倍标准差
			outliers = append(outliers, OutlierInfo{
				Index:     i,
				Value:     md.Price,
				Method:    "Z-Score",
				Threshold: 3.0,
				ZScore:    zScore,
			})
		}
	}

	return outliers
}

// detectOutliersModifiedZScore 使用修正Z-Score方法检测异常值
func (dp *DataPreprocessor) detectOutliersModifiedZScore(data []MarketData) []OutlierInfo {
	if len(data) < 3 {
		return nil
	}

	prices := make([]float64, len(data))
	for i, md := range data {
		prices[i] = md.Price
	}

	// 计算中位数
	sortedPrices := make([]float64, len(prices))
	copy(sortedPrices, prices)
	sort.Float64s(sortedPrices)

	var median float64
	n := len(sortedPrices)
	if n%2 == 0 {
		median = (sortedPrices[n/2-1] + sortedPrices[n/2]) / 2
	} else {
		median = sortedPrices[n/2]
	}

	// 计算MAD (Median Absolute Deviation)
	mads := make([]float64, len(prices))
	for i, price := range prices {
		mads[i] = math.Abs(price - median)
	}
	sort.Float64s(mads)

	var mad float64
	if n%2 == 0 {
		mad = (mads[n/2-1] + mads[n/2]) / 2
	} else {
		mad = mads[n/2]
	}

	if mad == 0 {
		return nil
	}

	outliers := make([]OutlierInfo, 0)
	for i, md := range data {
		modifiedZScore := 0.6745 * (md.Price - median) / mad
		if math.Abs(modifiedZScore) > 3.5 {
			outliers = append(outliers, OutlierInfo{
				Index:          i,
				Value:          md.Price,
				Method:         "Modified_Z-Score",
				Threshold:      3.5,
				ModifiedZScore: modifiedZScore,
			})
		}
	}

	return outliers
}

// HandleOutliers 处理异常值
func (dp *DataPreprocessor) HandleOutliers(data []MarketData, outliers []OutlierInfo, method string) []MarketData {
	if len(outliers) == 0 {
		return data
	}

	cleanedData := make([]MarketData, len(data))
	copy(cleanedData, data)

	switch method {
	case "remove":
		return dp.removeOutliers(data, outliers)
	case "cap":
		return dp.capOutliers(data, outliers)
	case "interpolate":
		return dp.interpolateOutliers(data, outliers)
	default:
		return dp.removeOutliers(data, outliers)
	}
}

// removeOutliers 移除异常值
func (dp *DataPreprocessor) removeOutliers(data []MarketData, outliers []OutlierInfo) []MarketData {
	// 按索引降序排序，确保移除时索引不乱
	sort.Slice(outliers, func(i, j int) bool {
		return outliers[i].Index > outliers[j].Index
	})

	cleanedData := make([]MarketData, len(data))
	copy(cleanedData, data)

	for _, outlier := range outliers {
		if outlier.Index >= 0 && outlier.Index < len(cleanedData) {
			// 移除异常值点
			cleanedData = append(cleanedData[:outlier.Index], cleanedData[outlier.Index+1:]...)
		}
	}

	return cleanedData
}

// capOutliers 限制异常值
func (dp *DataPreprocessor) capOutliers(data []MarketData, outliers []OutlierInfo) []MarketData {
	cleanedData := make([]MarketData, len(data))
	copy(cleanedData, data)

	// 计算正常范围
	prices := make([]float64, len(data))
	for i, md := range data {
		prices[i] = md.Price
	}
	sort.Float64s(prices)

	q1 := prices[len(prices)/4]
	q3 := prices[3*len(prices)/4]
	iqr := q3 - q1

	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr

	for _, outlier := range outliers {
		if outlier.Index >= 0 && outlier.Index < len(cleanedData) {
			if cleanedData[outlier.Index].Price < lowerBound {
				cleanedData[outlier.Index].Price = lowerBound
			} else if cleanedData[outlier.Index].Price > upperBound {
				cleanedData[outlier.Index].Price = upperBound
			}
		}
	}

	return cleanedData
}

// interpolateOutliers 插值异常值
func (dp *DataPreprocessor) interpolateOutliers(data []MarketData, outliers []OutlierInfo) []MarketData {
	cleanedData := make([]MarketData, len(data))
	copy(cleanedData, data)

	for _, outlier := range outliers {
		if outlier.Index <= 0 || outlier.Index >= len(cleanedData)-1 {
			continue // 无法插值
		}

		// 线性插值
		prevPrice := cleanedData[outlier.Index-1].Price
		nextPrice := cleanedData[outlier.Index+1].Price
		interpolatedPrice := (prevPrice + nextPrice) / 2

		cleanedData[outlier.Index].Price = interpolatedPrice
	}

	return cleanedData
}

// NormalizeData 数据标准化
func (dp *DataPreprocessor) NormalizeData(data []MarketData, method string) []MarketData {
	if len(data) == 0 {
		return data
	}

	normalizedData := make([]MarketData, len(data))
	copy(normalizedData, data)

	switch method {
	case "zscore":
		normalizedData = dp.normalizeZScore(data)
	case "minmax":
		normalizedData = dp.normalizeMinMax(data)
	case "robust":
		normalizedData = dp.normalizeRobust(data)
	}

	return normalizedData
}

// normalizeZScore Z-Score标准化
func (dp *DataPreprocessor) normalizeZScore(data []MarketData) []MarketData {
	prices := make([]float64, len(data))
	for i, md := range data {
		prices[i] = md.Price
	}

	mean, std := dp.calculateMeanStd(prices)
	if std == 0 {
		return data
	}

	normalizedData := make([]MarketData, len(data))
	for i, md := range data {
		normalized := (md.Price - mean) / std
		normalizedData[i] = md
		normalizedData[i].Price = normalized
	}

	return normalizedData
}

// normalizeMinMax Min-Max标准化
func (dp *DataPreprocessor) normalizeMinMax(data []MarketData) []MarketData {
	if len(data) == 0 {
		return data
	}

	minPrice := data[0].Price
	maxPrice := data[0].Price

	for _, md := range data {
		if md.Price < minPrice {
			minPrice = md.Price
		}
		if md.Price > maxPrice {
			maxPrice = md.Price
		}
	}

	if maxPrice == minPrice {
		return data
	}

	normalizedData := make([]MarketData, len(data))
	for i, md := range data {
		normalized := (md.Price - minPrice) / (maxPrice - minPrice)
		normalizedData[i] = md
		normalizedData[i].Price = normalized
	}

	return normalizedData
}

// normalizeRobust 鲁棒标准化
func (dp *DataPreprocessor) normalizeRobust(data []MarketData) []MarketData {
	prices := make([]float64, len(data))
	for i, md := range data {
		prices[i] = md.Price
	}

	sort.Float64s(prices)
	q1 := prices[len(prices)/4]
	q3 := prices[3*len(prices)/4]
	iqr := q3 - q1

	if iqr == 0 {
		return data
	}

	median := prices[len(prices)/2]

	normalizedData := make([]MarketData, len(data))
	for i, md := range data {
		normalized := (md.Price - median) / iqr
		normalizedData[i] = md
		normalizedData[i].Price = normalized
	}

	return normalizedData
}

// FillMissingData 填充缺失数据
func (dp *DataPreprocessor) FillMissingData(data []MarketData, method string) []MarketData {
	if len(data) == 0 {
		return data
	}

	filledData := make([]MarketData, 0, len(data))

	switch method {
	case "forward_fill":
		filledData = dp.forwardFill(data)
	case "backward_fill":
		filledData = dp.backwardFill(data)
	case "interpolate":
		filledData = dp.interpolateMissing(data)
	default:
		filledData = dp.forwardFill(data)
	}

	return filledData
}

// forwardFill 前向填充
func (dp *DataPreprocessor) forwardFill(data []MarketData) []MarketData {
	filledData := make([]MarketData, len(data))
	copy(filledData, data)

	lastValidPrice := filledData[0].Price
	for i := 1; i < len(filledData); i++ {
		if filledData[i].Price == 0 { // 假设0表示缺失
			filledData[i].Price = lastValidPrice
		} else {
			lastValidPrice = filledData[i].Price
		}
	}

	return filledData
}

// backwardFill 后向填充
func (dp *DataPreprocessor) backwardFill(data []MarketData) []MarketData {
	filledData := make([]MarketData, len(data))
	copy(filledData, data)

	for i := len(filledData) - 2; i >= 0; i-- {
		if filledData[i].Price == 0 {
			filledData[i].Price = filledData[i+1].Price
		}
	}

	return filledData
}

// interpolateMissing 插值填充
func (dp *DataPreprocessor) interpolateMissing(data []MarketData) []MarketData {
	filledData := make([]MarketData, len(data))
	copy(filledData, data)

	// 简单线性插值
	validPoints := make([]int, 0)
	for i, md := range filledData {
		if md.Price != 0 {
			validPoints = append(validPoints, i)
		}
	}

	if len(validPoints) < 2 {
		return filledData
	}

	for i := 0; i < len(validPoints)-1; i++ {
		start := validPoints[i]
		end := validPoints[i+1]
		startPrice := filledData[start].Price
		endPrice := filledData[end].Price

		for j := start + 1; j < end; j++ {
			ratio := float64(j-start) / float64(end-start)
			filledData[j].Price = startPrice + (endPrice-startPrice)*ratio
		}
	}

	return filledData
}

// calculateMeanStd 计算均值和标准差
func (dp *DataPreprocessor) calculateMeanStd(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(len(data))

	variance := 0.0
	for _, v := range data {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(data) - 1)
	std := math.Sqrt(variance)

	return mean, std
}

// OutlierInfo 异常值信息
type OutlierInfo struct {
	Index          int     `json:"index"`
	Value          float64 `json:"value"`
	Method         string  `json:"method"`
	Threshold      float64 `json:"threshold"`
	IsUpper        bool    `json:"is_upper,omitempty"`
	ZScore         float64 `json:"z_score,omitempty"`
	ModifiedZScore float64 `json:"modified_z_score,omitempty"`
}
