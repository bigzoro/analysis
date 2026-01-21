package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"strconv"
)

// MultiTimeframeIndicators 多时间框架技术指标
type MultiTimeframeIndicators struct {
	// 短期指标（1h, 4h）
	ShortTerm TechnicalIndicators `json:"short_term"`

	// 中期指标（1d, 3d）
	MediumTerm TechnicalIndicators `json:"medium_term"`

	// 长期指标（1w, 1M）
	LongTerm TechnicalIndicators `json:"long_term"`

	// 时间框架一致性评分
	TimeframeConsistency float64 `json:"timeframe_consistency"` // 0-100

	// 综合信号
	OverallSignal    string  `json:"overall_signal"`    // "strong_buy"/"buy"/"neutral"/"sell"/"strong_sell"
	SignalConfidence float64 `json:"signal_confidence"` // 0-100
}

// GetMultiTimeframeIndicators 获取多时间框架技术指标
func (s *Server) GetMultiTimeframeIndicators(ctx context.Context, symbol string, kind string) (*MultiTimeframeIndicators, error) {
	// 获取短期指标（1小时线，最近200小时）
	shortTerm, err := s.GetTechnicalIndicatorsWithDataPoints(ctx, symbol, kind, 200)
	if err != nil {
		return nil, fmt.Errorf("获取短期指标失败: %w", err)
	}

	// 获取中期指标（4小时线，最近100个4小时数据）
	mediumTerm, err := s.getTechnicalIndicatorsWithInterval(ctx, symbol, kind, "4h", 100)
	if err != nil {
		log.Printf("[WARN] 获取中期指标失败，使用短期指标替代: %v", err)
		mediumTerm = *shortTerm // 回退到短期指标
	}

	// 获取长期指标（日线，最近50天数据）
	longTerm, err := s.getTechnicalIndicatorsWithInterval(ctx, symbol, kind, "1d", 50)
	if err != nil {
		log.Printf("[WARN] 获取长期指标失败，使用中期指标替代: %v", err)
		longTerm = mediumTerm // 回退到中期指标
	}

	// 计算时间框架一致性
	consistency := calculateTimeframeConsistency(*shortTerm, mediumTerm, longTerm)

	// 生成综合信号
	overallSignal, confidence := generateOverallSignal(*shortTerm, mediumTerm, longTerm, consistency)

	return &MultiTimeframeIndicators{
		ShortTerm:            *shortTerm,
		MediumTerm:           mediumTerm,
		LongTerm:             longTerm,
		TimeframeConsistency: consistency,
		OverallSignal:        overallSignal,
		SignalConfidence:     confidence,
	}, nil
}

// getTechnicalIndicatorsWithInterval 获取指定时间间隔的技术指标
func (s *Server) getTechnicalIndicatorsWithInterval(ctx context.Context, symbol, kind, interval string, dataPoints int) (TechnicalIndicators, error) {
	// 转换为API使用的交易对格式
	apiSymbol := s.convertToBinanceSymbol(symbol, kind)

	// 从API获取指定间隔的K线数据
	klines, err := s.getKlinesWithCache(ctx, apiSymbol, kind, interval, dataPoints)
	if err != nil {
		return TechnicalIndicators{}, fmt.Errorf("获取%s K线数据失败: %w", interval, err)
	}

	// 计算技术指标
	return calculateTechnicalIndicators(klines)
}

// calculateTechnicalIndicators 计算技术指标
func calculateTechnicalIndicators(klines []BinanceKline) (TechnicalIndicators, error) {
	if len(klines) < 26 {
		return TechnicalIndicators{}, fmt.Errorf("K线数据不足%d条", len(klines))
	}

	// 提取价格和成交量数据
	closes := make([]float64, 0, len(klines))
	highs := make([]float64, 0, len(klines))
	lows := make([]float64, 0, len(klines))
	volumes := make([]float64, 0, len(klines))

	for _, k := range klines {
		close, _ := strconv.ParseFloat(k.Close, 64)
		high, _ := strconv.ParseFloat(k.High, 64)
		low, _ := strconv.ParseFloat(k.Low, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)

		closes = append(closes, close)
		highs = append(highs, high)
		lows = append(lows, low)
		volumes = append(volumes, volume)
	}

	if len(closes) < 60 {
		// 数据不足，只计算基本指标
		rsi := calculateRSI(closes, 14)
		macd, signal, hist := calculateMACD(closes, 12, 26, 9)
		trend := determineTrend(rsi, macd, signal)

		return TechnicalIndicators{
			RSI:        rsi,
			MACD:       macd,
			MACDSignal: signal,
			MACDHist:   hist,
			Trend:      trend,
		}, nil
	}

	// 计算完整的指标
	rsi := calculateRSI(closes, 14)
	macd, signal, hist := calculateMACD(closes, 12, 26, 9)
	bbMiddle, bbUpper, bbLower, bbWidth, bbPosition := calculateBollingerBands(closes, 20, 2.0)
	trend := determineTrend(rsi, macd, signal)

	indicators := TechnicalIndicators{
		RSI:        rsi,
		MACD:       macd,
		MACDSignal: signal,
		MACDHist:   hist,
		BBUpper:    bbUpper,
		BBMiddle:   bbMiddle,
		BBLower:    bbLower,
		BBWidth:    bbWidth,
		BBPosition: bbPosition,
		Trend:      trend,
	}

	return indicators, nil
}

// calculateTimeframeConsistency 计算时间框架一致性
func calculateTimeframeConsistency(short, medium, long TechnicalIndicators) float64 {
	consistencyScore := 0.0

	// 检查趋势一致性 (权重40%)
	trends := []string{short.Trend, medium.Trend, long.Trend}
	consistentTrends := countConsistentTrends(trends)
	trendConsistency := float64(consistentTrends) / float64(len(trends)) * 100.0 // 0-100
	consistencyScore += trendConsistency * 0.4

	// 检查RSI一致性（相对强弱指标，权重30%）
	rsiConsistency := calculateIndicatorConsistency([]float64{short.RSI, medium.RSI, long.RSI})
	consistencyScore += rsiConsistency * 0.3

	// 检查MACD一致性 (权重30%)
	macdConsistency := calculateIndicatorConsistency([]float64{short.MACD, medium.MACD, long.MACD})
	consistencyScore += macdConsistency * 0.3

	// 确保结果在0-100范围内
	return math.Max(0.0, math.Min(100.0, consistencyScore))
}

// countConsistentTrends 计算趋势一致性
func countConsistentTrends(trends []string) int {
	if len(trends) == 0 {
		return 0
	}

	// 使用多数投票原则
	trendCount := make(map[string]int)
	for _, trend := range trends {
		trendCount[trend]++
	}

	maxCount := 0
	for _, count := range trendCount {
		if count > maxCount {
			maxCount = count
		}
	}

	return maxCount
}

// calculateIndicatorConsistency 计算指标一致性
func calculateIndicatorConsistency(values []float64) float64 {
	if len(values) < 2 {
		return 100.0
	}

	// 计算标准差
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		variance += math.Pow(v-mean, 2)
	}
	variance /= float64(len(values))
	stdDev := math.Sqrt(variance)

	// 一致性 = 100 - (标准差/均值)*100，但不超过100
	if mean == 0 {
		return 100.0
	}

	consistency := 100.0 - (stdDev/mean)*100.0
	return math.Max(0.0, math.Min(100.0, consistency))
}

// generateOverallSignal 生成综合信号
func generateOverallSignal(short, medium, long TechnicalIndicators, consistency float64) (string, float64) {
	// 计算各时间框架的信号得分
	shortSignal := calculateSignalScore(short)
	mediumSignal := calculateSignalScore(medium)
	longSignal := calculateSignalScore(long)

	// 加权平均（短期权重最高）
	weights := []float64{0.5, 0.3, 0.2} // 短期50%, 中期30%, 长期20%
	totalWeight := 0.0
	weightedSignal := 0.0

	signals := []float64{shortSignal, mediumSignal, longSignal}
	for i, signal := range signals {
		weightedSignal += signal * weights[i]
		totalWeight += weights[i]
	}
	weightedSignal /= totalWeight

	// 一致性调整
	consistencyFactor := consistency / 100.0
	finalSignal := weightedSignal * (0.7 + 0.3*consistencyFactor) // 一致性影响30%的权重

	// 计算置信度 (0-100范围)
	signalStrength := math.Min(100.0, math.Abs(finalSignal))
	confidence := math.Min(consistency, signalStrength)

	// 根据最终信号值确定信号类型
	var signalType string
	switch {
	case finalSignal >= 70:
		signalType = "strong_buy"
	case finalSignal >= 30:
		signalType = "buy"
	case finalSignal >= -30:
		signalType = "neutral"
	case finalSignal >= -70:
		signalType = "sell"
	default:
		signalType = "strong_sell"
	}

	return signalType, confidence
}

// calculateSignalScore 计算单个信号得分
func calculateSignalScore(indicators TechnicalIndicators) float64 {
	score := 0.0

	// RSI贡献 (0-30分)
	rsiScore := 0.0
	switch {
	case indicators.RSI < 30:
		rsiScore = 30 // 超卖
	case indicators.RSI > 70:
		rsiScore = -30 // 超买
	default:
		rsiScore = 0 // 中性
	}
	score += rsiScore

	// MACD贡献 (0-40分)
	macdScore := 0.0
	if indicators.MACD > indicators.MACDSignal && indicators.MACDHist > 0 {
		macdScore = 20 // 金叉向上
	} else if indicators.MACD < indicators.MACDSignal && indicators.MACDHist < 0 {
		macdScore = -20 // 死叉向下
	}
	score += macdScore

	// 布林带位置贡献 (0-30分)
	bbScore := 0.0
	if indicators.BBPosition < 0.2 {
		bbScore = 15 // 接近下轨，买入信号
	} else if indicators.BBPosition > 0.8 {
		bbScore = -15 // 接近上轨，卖出信号
	}
	score += bbScore

	// 确保得分在-100到100之间
	return math.Max(-100.0, math.Min(100.0, score))
}
