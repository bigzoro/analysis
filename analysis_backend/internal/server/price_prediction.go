package server

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	pdb "analysis/internal/db"
)

// PricePrediction 价格预测结果
type PricePrediction struct {
	Symbol          string    `json:"symbol"`
	CurrentPrice    float64   `json:"current_price"`
	PredictedAt     time.Time `json:"predicted_at"`
	
	// 短期预测（24小时）
	Pred24h         float64   `json:"pred_24h"`         // 预测价格
	Change24h       float64   `json:"change_24h"`       // 预测涨跌幅 %
	Confidence24h   float64   `json:"confidence_24h"`   // 置信度 0-100
	Range24h        PriceRange `json:"range_24h"`        // 价格区间
	
	// 中期预测（7天）
	Pred7d          float64   `json:"pred_7d"`          // 预测价格
	Change7d        float64   `json:"change_7d"`        // 预测涨跌幅 %
	Confidence7d    float64   `json:"confidence_7d"`     // 置信度 0-100
	Range7d         PriceRange `json:"range_7d"`         // 价格区间
	
	// 长期预测（30天）
	Pred30d         float64   `json:"pred_30d"`         // 预测价格
	Change30d       float64   `json:"change_30d"`        // 预测涨跌幅 %
	Confidence30d   float64   `json:"confidence_30d"`   // 置信度 0-100
	Range30d        PriceRange `json:"range_30d"`        // 价格区间
	
	// 预测依据
	Factors         []string  `json:"factors"`          // 预测依据说明
	Trend           string    `json:"trend"`            // "bullish"/"bearish"/"neutral"
}

// PriceRange 价格区间
type PriceRange struct {
	Min float64 `json:"min"` // 最低价
	Max float64 `json:"max"` // 最高价
	Avg float64 `json:"avg"` // 平均价
}

// GetPricePrediction 获取价格预测
func (s *Server) GetPricePrediction(ctx context.Context, symbol string, kind string) (*PricePrediction, error) {
	// 1. 获取当前价格
	currentPrice, err := s.getCurrentPrice(ctx, symbol, kind)
	if err != nil {
		return nil, fmt.Errorf("获取当前价格失败: %w", err)
	}

	// 2. 获取K线数据（用于技术分析）
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1h", 500) // 获取更多数据用于预测
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) < 60 {
		return nil, fmt.Errorf("K线数据不足，无法进行预测")
	}

	// 3. 提取价格序列
	prices := make([]float64, len(klines))
	volumes := make([]float64, len(klines))
	for i, k := range klines {
		price, _ := strconv.ParseFloat(k.Close, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)
		prices[i] = price
		volumes[i] = volume
	}

	// 4. 获取技术指标
	technical, err := s.GetTechnicalIndicators(ctx, symbol, kind)
	if err != nil {
		// 如果获取失败，使用基础预测
		technical = &TechnicalIndicators{
			RSI: 50,
			Trend: "sideways",
		}
	}

	// 5. 计算预测
	prediction := &PricePrediction{
		Symbol:       symbol,
		CurrentPrice: currentPrice,
		PredictedAt:  time.Now().UTC(),
	}

	// 6. 24小时预测
	pred24h, change24h, confidence24h, range24h := s.predictPrice(
		prices, volumes, technical, currentPrice, 24,
	)
	prediction.Pred24h = pred24h
	prediction.Change24h = change24h
	prediction.Confidence24h = confidence24h
	prediction.Range24h = range24h

	// 7. 7天预测
	pred7d, change7d, confidence7d, range7d := s.predictPrice(
		prices, volumes, technical, currentPrice, 7*24,
	)
	prediction.Pred7d = pred7d
	prediction.Change7d = change7d
	prediction.Confidence7d = confidence7d
	prediction.Range7d = range7d

	// 8. 30天预测
	pred30d, change30d, confidence30d, range30d := s.predictPrice(
		prices, volumes, technical, currentPrice, 30*24,
	)
	prediction.Pred30d = pred30d
	prediction.Change30d = change30d
	prediction.Confidence30d = confidence30d
	prediction.Range30d = range30d

	// 9. 生成预测依据和趋势
	prediction.Factors = s.generatePredictionFactors(technical, prices, volumes)
	prediction.Trend = s.determinePredictionTrend(change24h, change7d, change30d)

	return prediction, nil
}

// predictPrice 预测价格
// hours: 预测时间（小时）
func (s *Server) predictPrice(
	prices []float64,
	volumes []float64,
	technical *TechnicalIndicators,
	currentPrice float64,
	hours int,
) (predictedPrice, changePercent, confidence float64, priceRange PriceRange) {
	// 使用多种方法预测，然后加权平均

	// 方法1：基于技术指标的趋势预测
	techPred, techConf := s.predictByTechnical(technical, currentPrice, hours)

	// 方法2：基于移动平均的趋势预测
	maPred, maConf := s.predictByMovingAverage(prices, currentPrice, hours)

	// 方法3：基于历史波动率的统计预测
	statPred, statConf := s.predictByStatistics(prices, currentPrice, hours)

	// 方法4：基于成交量的动量预测
	volumePred, volumeConf := s.predictByVolume(prices, volumes, currentPrice, hours)

	// 加权平均（置信度作为权重）
	totalWeight := techConf + maConf + statConf + volumeConf
	if totalWeight == 0 {
		totalWeight = 1
	}

	predictedPrice = (techPred*techConf + maPred*maConf + statPred*statConf + volumePred*volumeConf) / totalWeight
	confidence = (techConf + maConf + statConf + volumeConf) / 4

	// 计算涨跌幅
	changePercent = ((predictedPrice - currentPrice) / currentPrice) * 100

	// 计算价格区间（基于波动率）
	volatility := s.calculateVolatility(prices)
	rangeWidth := predictedPrice * volatility * math.Sqrt(float64(hours)/24) * 0.5 // 0.5倍标准差
	priceRange = PriceRange{
		Min: math.Max(0, predictedPrice-rangeWidth),
		Max: predictedPrice + rangeWidth,
		Avg: predictedPrice,
	}

	return predictedPrice, changePercent, confidence, priceRange
}

// predictByTechnical 基于技术指标预测
func (s *Server) predictByTechnical(tech *TechnicalIndicators, currentPrice float64, hours int) (float64, float64) {
	if tech == nil {
		return currentPrice, 0.3
	}

	// 基于趋势判断
	trendFactor := 1.0
	confidence := 0.5

	switch tech.Trend {
	case "up":
		trendFactor = 1.02 // 上涨趋势，预测上涨2%
		confidence = 0.6
	case "down":
		trendFactor = 0.98 // 下跌趋势，预测下跌2%
		confidence = 0.6
	default:
		trendFactor = 1.0
		confidence = 0.4
	}

	// RSI调整
	if tech.RSI > 70 {
		trendFactor *= 0.98 // 超买，可能回调
		confidence += 0.1
	} else if tech.RSI < 30 {
		trendFactor *= 1.02 // 超卖，可能反弹
		confidence += 0.1
	}

	// MACD调整
	if tech.MACD > tech.MACDSignal && tech.MACDHist > 0 {
		trendFactor *= 1.01 // 金叉
		confidence += 0.05
	} else if tech.MACD < tech.MACDSignal && tech.MACDHist < 0 {
		trendFactor *= 0.99 // 死叉
		confidence += 0.05
	}

	// 均线调整
	if tech.MA5 > tech.MA10 && tech.MA10 > tech.MA20 {
		trendFactor *= 1.01 // 多头排列
		confidence += 0.05
	} else if tech.MA5 < tech.MA10 && tech.MA10 < tech.MA20 {
		trendFactor *= 0.99 // 空头排列
		confidence += 0.05
	}

	// 时间衰减（预测时间越长，趋势影响越小）
	timeDecay := math.Pow(0.95, float64(hours)/24)
	trendFactor = 1.0 + (trendFactor-1.0)*timeDecay

	predictedPrice := currentPrice * trendFactor
	confidence = math.Min(1.0, confidence)

	return predictedPrice, confidence
}

// predictByMovingAverage 基于移动平均预测
func (s *Server) predictByMovingAverage(prices []float64, currentPrice float64, hours int) (float64, float64) {
	if len(prices) < 20 {
		return currentPrice, 0.3
	}

	// 计算短期和长期均线
	shortMA := calculateSMA(prices, 5)
	longMA := calculateSMA(prices, 20)

	if shortMA == 0 || longMA == 0 {
		return currentPrice, 0.3
	}

	// 均线斜率
	recentPrices := prices[len(prices)-5:]
	oldPrices := prices[len(prices)-10 : len(prices)-5]
	shortSlope := (recentPrices[len(recentPrices)-1] - oldPrices[0]) / oldPrices[0]
	longSlope := (prices[len(prices)-1] - prices[len(prices)-20]) / prices[len(prices)-20]

	// 预测价格 = 当前价格 * (1 + 斜率 * 时间因子)
	timeFactor := float64(hours) / 24.0
	slope := (shortSlope*0.6 + longSlope*0.4) // 短期权重更高
	predictedPrice := currentPrice * (1 + slope*timeFactor*0.5) // 0.5是衰减因子

	// 置信度：均线越接近，置信度越高
	maDiff := math.Abs(shortMA - longMA) / currentPrice
	confidence := 0.5 - maDiff*10 // 差异越小，置信度越高
	confidence = math.Max(0.3, math.Min(0.7, confidence))

	return predictedPrice, confidence
}

// predictByStatistics 基于统计方法预测
func (s *Server) predictByStatistics(prices []float64, currentPrice float64, hours int) (float64, float64) {
	if len(prices) < 30 {
		return currentPrice, 0.3
	}

	// 计算历史收益率
	returns := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	// 计算平均收益率和波动率
	var meanReturn float64
	for _, r := range returns {
		meanReturn += r
	}
	meanReturn /= float64(len(returns))

	volatility := s.calculateVolatility(prices)

	// 预测价格 = 当前价格 * (1 + 平均收益率 * 时间)
	timeFactor := float64(hours) / 24.0
	predictedPrice := currentPrice * math.Pow(1+meanReturn, timeFactor)

	// 置信度：波动率越低，置信度越高
	confidence := 0.5 - volatility*5
	confidence = math.Max(0.2, math.Min(0.6, confidence))

	return predictedPrice, confidence
}

// predictByVolume 基于成交量预测
func (s *Server) predictByVolume(prices []float64, volumes []float64, currentPrice float64, hours int) (float64, float64) {
	if len(prices) < 20 || len(volumes) < 20 {
		return currentPrice, 0.3
	}

	// 计算成交量均线
	volumeMA := calculateSMA(volumes, 20)
	if volumeMA == 0 {
		return currentPrice, 0.3
	}

	// 当前成交量比率
	currentVolume := volumes[len(volumes)-1]
	volumeRatio := currentVolume / volumeMA

	// 价格动量（最近5个周期的平均涨跌幅）
	recentReturns := make([]float64, 0, 5)
	for i := len(prices) - 5; i < len(prices); i++ {
		if i > 0 {
			ret := (prices[i] - prices[i-1]) / prices[i-1]
			recentReturns = append(recentReturns, ret)
		}
	}

	var momentum float64
	for _, r := range recentReturns {
		momentum += r
	}
	momentum /= float64(len(recentReturns))

	// 预测：成交量放大 + 正动量 = 上涨
	// 成交量放大 + 负动量 = 下跌
	timeFactor := float64(hours) / 24.0
	volumeFactor := 1.0
	if volumeRatio > 1.2 {
		volumeFactor = 1.01 // 成交量放大，看涨
	} else if volumeRatio < 0.8 {
		volumeFactor = 0.99 // 成交量萎缩，看跌
	}

	predictedPrice := currentPrice * math.Pow(1+momentum*volumeFactor, timeFactor*0.3)

	// 置信度：成交量比率越极端，置信度越高
	confidence := 0.3 + math.Min(0.3, math.Abs(volumeRatio-1.0)*0.5)

	return predictedPrice, confidence
}

// calculateVolatility 计算波动率（标准差）
func (s *Server) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.1 // 默认波动率
	}

	// 计算收益率
	returns := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	// 计算平均收益率
	var meanReturn float64
	for _, r := range returns {
		meanReturn += r
	}
	meanReturn /= float64(len(returns))

	// 计算标准差
	var variance float64
	for _, r := range returns {
		variance += (r - meanReturn) * (r - meanReturn)
	}
	variance /= float64(len(returns))
	volatility := math.Sqrt(variance)

	return volatility
}

// generatePredictionFactors 生成预测依据
func (s *Server) generatePredictionFactors(tech *TechnicalIndicators, prices []float64, volumes []float64) []string {
	factors := make([]string, 0)

	if tech == nil {
		return factors
	}

	// 技术指标因素
	if tech.Trend == "up" {
		factors = append(factors, "技术指标显示上涨趋势")
	} else if tech.Trend == "down" {
		factors = append(factors, "技术指标显示下跌趋势")
	}

	if tech.RSI > 70 {
		factors = append(factors, "RSI超买，可能回调")
	} else if tech.RSI < 30 {
		factors = append(factors, "RSI超卖，可能反弹")
	}

	if tech.MACD > tech.MACDSignal && tech.MACDHist > 0 {
		factors = append(factors, "MACD金叉，技术面看涨")
	}

	if tech.MA5 > tech.MA10 && tech.MA10 > tech.MA20 {
		factors = append(factors, "均线多头排列")
	}

	// 成交量因素
	if len(volumes) >= 20 {
		volumeMA := calculateSMA(volumes, 20)
		if volumeMA > 0 {
			currentVolume := volumes[len(volumes)-1]
			volumeRatio := currentVolume / volumeMA
			if volumeRatio > 1.5 {
				factors = append(factors, "成交量显著放大")
			} else if volumeRatio < 0.5 {
				factors = append(factors, "成交量萎缩")
			}
		}
	}

	// 波动率因素
	volatility := s.calculateVolatility(prices)
	if volatility > 0.05 {
		factors = append(factors, "波动率较高，价格波动可能较大")
	} else if volatility < 0.02 {
		factors = append(factors, "波动率较低，价格相对稳定")
	}

	return factors
}

// determinePredictionTrend 确定预测趋势
func (s *Server) determinePredictionTrend(change24h, change7d, change30d float64) string {
	// 计算加权平均涨跌幅
	avgChange := (change24h*0.5 + change7d*0.3 + change30d*0.2)

	if avgChange > 5 {
		return "bullish"
	} else if avgChange < -5 {
		return "bearish"
	}
	return "neutral"
}

// getCurrentPrice 获取当前价格
func (s *Server) getCurrentPrice(ctx context.Context, symbol string, kind string) (float64, error) {
	// 尝试从Binance获取最新价格
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1m", 1)
	if err == nil && len(klines) > 0 {
		price, err := strconv.ParseFloat(klines[0].Close, 64)
		if err == nil {
			return price, nil
		}
	}

	// 如果失败，尝试从市场快照获取
	now := time.Now().UTC()
	startTime := now.Add(-2 * time.Hour)
	snaps, tops, err := pdb.ListBinanceMarket(s.db.DB(), kind, startTime, now)
	if err == nil && len(snaps) > 0 {
		// 获取最新的快照
		latestSnap := snaps[len(snaps)-1]
		if items, ok := tops[latestSnap.ID]; ok {
			for _, item := range items {
				if item.Symbol == symbol {
					price, err := strconv.ParseFloat(item.LastPrice, 64)
					if err == nil {
						return price, nil
					}
				}
			}
		}
	}

	return 0, fmt.Errorf("无法获取当前价格")
}

