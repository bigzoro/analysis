package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	pdb "analysis/internal/db"
	"analysis/internal/netutil"
)

// BinanceKline 币安K线数据
type BinanceKline struct {
	OpenTime                 float64 `json:"openTime"`                 // 开盘时间戳
	Open                     string  `json:"open"`                     // 开盘价
	High                     string  `json:"high"`                     // 最高价
	Low                      string  `json:"low"`                      // 最低价
	Close                    string  `json:"close"`                    // 收盘价
	Volume                   string  `json:"volume"`                   // 成交量
	CloseTime                float64 `json:"closeTime"`                // 收盘时间戳
	QuoteAssetVolume         string  `json:"quoteAssetVolume"`         // 成交额
	NumberOfTrades           int     `json:"numberOfTrades"`           // 成交笔数
	TakerBuyBaseAssetVolume  string  `json:"takerBuyBaseAssetVolume"`  // Taker买入基础资产成交量
	TakerBuyQuoteAssetVolume string  `json:"takerBuyQuoteAssetVolume"` // Taker买入报价资产成交额
}

// KlineData 增强的K线数据结构，包含验证和处理信息
type KlineData struct {
	BinanceKline
	Symbol      string    `json:"symbol"`      // 交易对符号
	Interval    string    `json:"interval"`    // 时间间隔
	Kind        string    `json:"kind"`        // 市场类型 (spot/futures)
	Timestamp   time.Time `json:"timestamp"`   // 数据时间戳
	IsValid     bool      `json:"isValid"`     // 数据是否有效
	DataQuality int       `json:"dataQuality"` // 数据质量评分 (0-100)
	ProcessedAt time.Time `json:"processedAt"` // 处理时间
}

// KlineValidationResult K线数据验证结果
type KlineValidationResult struct {
	IsValid          bool     `json:"isValid"`
	Errors           []string `json:"errors"`
	Warnings         []string `json:"warnings"`
	DataQualityScore int      `json:"dataQualityScore"`
	Suggestions      []string `json:"suggestions"`
}

// CalculateTechnicalIndicators 计算技术指标（默认200个数据点）
func (s *Server) CalculateTechnicalIndicators(ctx context.Context, symbol string, kind string) (*TechnicalIndicators, error) {
	return s.GetTechnicalIndicatorsWithDataPoints(ctx, symbol, kind, 200)
}

// GetTechnicalIndicatorsWithSignals 获取包含交易信号的技术指标
func (s *Server) GetTechnicalIndicatorsWithSignals(ctx context.Context, symbol string, kind string, currentPrice float64) (*TechnicalIndicators, error) {
	// 获取基础技术指标
	indicators, err := s.CalculateTechnicalIndicators(ctx, symbol, kind)
	if err != nil {
		return nil, err
	}

	// 这里可以添加交易信号生成逻辑
	// 暂时返回基础指标
	return indicators, nil
}

// GetTechnicalIndicatorsFromHistory 从历史快照数据计算技术指标（简化版）
func (s *Server) GetTechnicalIndicatorsFromHistory(ctx context.Context, symbol string, kind string) (*TechnicalIndicators, error) {
	// 获取最近的历史快照数据
	now := time.Now().UTC()
	_ = now.Add(-7 * 24 * time.Hour) // 最近7天

	// 这里应该从数据库获取历史数据
	// 暂时返回默认值
	return &TechnicalIndicators{
		RSI:   50,
		MACD:  0,
		Trend: "sideways",
	}, nil
}

// GetTechnicalIndicatorsWithDataPoints 获取技术指标（指定数据点数量，支持数据库缓存）
func (s *Server) GetTechnicalIndicatorsWithDataPoints(ctx context.Context, symbol string, kind string, dataPoints int) (*TechnicalIndicators, error) {
	// 转换为API使用的交易对格式进行缓存操作
	apiSymbol := s.convertToBinanceSymbol(symbol, kind)

	// 1. 尝试从内存缓存获取
	cacheKey := fmt.Sprintf("technical:%s:%s:%d", apiSymbol, kind, dataPoints)
	if cachedData, err := s.cache.Get(ctx, cacheKey); err == nil && len(cachedData) > 0 {
		var cachedIndicators TechnicalIndicators
		if err := json.Unmarshal(cachedData, &cachedIndicators); err == nil {
			// 内存缓存命中，直接返回
			return &cachedIndicators, nil
		}
		// 缓存数据损坏，继续计算
	}

	// 2. 尝试从数据库缓存获取
	gdb := s.db.DB()
	if gdb != nil {
		dbCache, err := pdb.GetTechnicalIndicatorsCache(gdb, apiSymbol, kind, "1h", dataPoints)
		if err == nil && dbCache != nil {
			// 检查缓存是否新鲜（5分钟内）
			if time.Since(dbCache.CalculatedAt) <= 5*time.Minute {
				var indicators TechnicalIndicators
				if err := json.Unmarshal(dbCache.Indicators, &indicators); err == nil {
					// 数据库缓存命中，更新内存缓存
					if cacheData, err := json.Marshal(indicators); err == nil {
						s.cache.Set(ctx, cacheKey, cacheData, 5*time.Minute)
					}
					return &indicators, nil
				}
			}
		}
	}

	// 3. 缓存未命中，使用getKlinesWithCache获取数据（该函数会自动处理数据库缓存和API获取）
	log.Printf("[Technical] 缓存未命中，开始获取%s的K线数据", symbol)
	klines, err := s.getKlinesWithCache(ctx, apiSymbol, kind, "1h", dataPoints)
	if err != nil {
		log.Printf("[WARN] 获取%s K线数据失败: %v，使用默认技术指标", symbol, err)
		// 返回默认值而不是错误，让AI推荐能继续工作
		return &TechnicalIndicators{
			RSI:   50,
			MACD:  0,
			Trend: "sideways",
		}, nil
	}

	if len(klines) < 26 {
		// 数据不足，返回默认值
		log.Printf("[WARN] %s K线数据不足%d条，使用默认技术指标", symbol, len(klines))
		return &TechnicalIndicators{
			RSI:   50,
			MACD:  0,
			Trend: "sideways",
		}, nil
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
		// 数据不足，返回默认值（只计算基本指标）
		rsi := calculateRSI(closes, 14)
		macd, signal, hist := calculateMACD(closes, 12, 26, 9)
		trend := determineTrend(rsi, macd, signal)

		return &TechnicalIndicators{
			RSI:        rsi,
			MACD:       macd,
			MACDSignal: signal,
			MACDHist:   hist,
			Trend:      trend,
		}, nil
	}

	// 计算RSI（14周期）
	rsi := calculateRSI(closes, 14)

	// 计算MACD（12, 26, 9）
	macd, signal, hist := calculateMACD(closes, 12, 26, 9)

	// 计算布林带（20周期，2倍标准差）
	bbMiddle, bbUpper, bbLower, bbWidth, bbPosition := calculateBollingerBands(closes, 20, 2.0)

	// 计算KDJ指标（14周期）
	k, d, j := calculateKDJ(highs, lows, closes, 14)

	// 计算均线系统
	ma5 := calculateSMA(closes, 5)
	ma10 := calculateSMA(closes, 10)
	ma20 := calculateSMA(closes, 20)
	ma50 := calculateSMA(closes, 50)
	ma60 := calculateSMA(closes, 60)

	var ma200 float64
	if len(closes) >= 200 {
		ma200 = calculateSMA(closes, 200)
	}

	// 计算成交量指标
	obv := calculateOBV(closes, volumes)
	volumeMA5 := calculateSMA(volumes, 5)
	volumeMA20 := calculateSMA(volumes, 20)
	var volumeRatio float64
	if volumeMA20 > 0 {
		volumeRatio = volumes[len(volumes)-1] / volumeMA20
	}

	// 计算支撑位和阻力位
	support, resistance, supportStrength, resistanceStrength := calculateSupportResistance(highs, lows, closes, 20)

	// 计算动量指标
	momentum5 := calculateMomentum(closes, 5)
	momentum10 := calculateMomentum(closes, 10)
	momentum20 := calculateMomentum(closes, 20)
	momentumDivergence := calculateMomentumDivergence(closes, highs, lows, 14)

	// 计算波动率指标
	volatility5 := calculateVolatility(closes, 5)
	volatility20 := calculateVolatility(closes, 20)
	var volatilityRatio float64
	if volatility20 > 0 {
		volatilityRatio = volatility5 / volatility20
	}

	// 计算威廉指标
	williamsR := calculateWilliamsR(highs, lows, closes, 14)

	// 计算顺势指标（CCI）
	cci := calculateCCI(highs, lows, closes, 20)

	// 确定趋势
	trend := determineTrend(rsi, macd, signal)

	// 计算信号强度
	signalStrength := calculateSignalStrength(TechnicalIndicators{
		RSI: rsi, MACD: macd, MACDSignal: signal, BBPosition: bbPosition,
	})

	// 确定风险等级
	riskLevel := calculateRiskLevel(rsi, bbPosition, volatility20)

	indicators := &TechnicalIndicators{
		RSI:                rsi,
		MACD:               macd,
		MACDSignal:         signal,
		MACDHist:           hist,
		Trend:              trend,
		BBUpper:            bbUpper,
		BBMiddle:           bbMiddle,
		BBLower:            bbLower,
		BollingerUpper:     bbUpper, // 别名
		BollingerLower:     bbLower, // 别名
		BBWidth:            bbWidth,
		BBPosition:         bbPosition,
		K:                  k,
		D:                  d,
		J:                  j,
		MA5:                ma5,
		MA10:               ma10,
		MA20:               ma20,
		MA50:               ma50,
		MA60:               ma60,
		MA200:              ma200,
		OBV:                obv,
		VolumeMA5:          volumeMA5,
		VolumeMA20:         volumeMA20,
		VolumeRatio:        volumeRatio,
		SupportLevel:       support,
		ResistanceLevel:    resistance,
		SupportStrength:    supportStrength,
		ResistanceStrength: resistanceStrength,
		Momentum5:          momentum5,
		Momentum10:         momentum10,
		Momentum20:         momentum20,
		MomentumDivergence: momentumDivergence,
		Volatility5:        volatility5,
		Volatility20:       volatility20,
		VolatilityRatio:    volatilityRatio,
		WilliamsR:          williamsR,
		CCI:                cci,
		SignalStrength:     signalStrength,
		RiskLevel:          riskLevel,
	}

	// 保存到缓存
	if cacheData, err := json.Marshal(indicators); err == nil {
		s.cache.Set(ctx, cacheKey, cacheData, 5*time.Minute)
	}

	// 保存到数据库缓存
	dataFrom := time.UnixMilli(int64(klines[0].OpenTime))
	dataTo := time.UnixMilli(int64(klines[len(klines)-1].OpenTime))
	go s.saveTechnicalIndicatorsCache(ctx, symbol, kind, "1h", dataPoints, indicators, dataFrom, dataTo)

	return indicators, nil
}

// 技术指标计算函数（RSI, MACD, 布林带等）
func calculateRSI(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 50
	}

	gains := make([]float64, 0, len(closes)-1)
	losses := make([]float64, 0, len(closes)-1)

	for i := 1; i < len(closes); i++ {
		change := closes[i] - closes[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// 计算初始平均值
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// 计算后续的指数移动平均
	for i := period; i < len(gains); i++ {
		avgGain = (avgGain*float64(period-1) + gains[i]) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + losses[i]) / float64(period)
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

func calculateMACD(closes []float64, fastPeriod, slowPeriod, signalPeriod int) (float64, float64, float64) {
	if len(closes) < slowPeriod {
		return 0, 0, 0
	}

	// 计算EMA
	fastEMA := calculateEMA(closes, fastPeriod)
	slowEMA := calculateEMA(closes, slowPeriod)

	// 计算MACD线
	macd := fastEMA - slowEMA

	// 计算信号线（MACD的EMA）
	macdValues := make([]float64, len(closes)-slowPeriod+1)
	for i := slowPeriod - 1; i < len(closes); i++ {
		fast := calculateEMA(closes[:i+1], fastPeriod)
		slow := calculateEMA(closes[:i+1], slowPeriod)
		macdValues[i-slowPeriod+1] = fast - slow
	}

	signal := calculateEMA(macdValues, signalPeriod)

	// 计算柱状图
	hist := macd - signal

	return macd, signal, hist
}

func calculateEMA(values []float64, period int) float64 {
	if len(values) < period {
		return 0
	}

	multiplier := 2.0 / (float64(period) + 1.0)
	ema := values[0]

	for i := 1; i < len(values); i++ {
		ema = (values[i] * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

func calculateSMA(values []float64, period int) float64 {
	if len(values) < period {
		return 0
	}

	sum := 0.0
	for i := len(values) - period; i < len(values); i++ {
		sum += values[i]
	}
	return sum / float64(period)
}

func calculateBollingerBands(closes []float64, period int, stdDev float64) (float64, float64, float64, float64, float64) {
	if len(closes) < period {
		return 0, 0, 0, 0, 0.5
	}

	// 计算SMA
	middle := calculateSMA(closes, period)

	// 计算标准差
	sum := 0.0
	for i := len(closes) - period; i < len(closes); i++ {
		sum += math.Pow(closes[i]-middle, 2)
	}
	std := math.Sqrt(sum / float64(period))

	upper := middle + (std * stdDev)
	lower := middle - (std * stdDev)

	// 计算宽度（百分比）
	width := (upper - lower) / middle

	// 计算当前位置（0-1之间，0=下轨，1=上轨）
	currentPrice := closes[len(closes)-1]
	var position float64
	if upper != lower {
		position = (currentPrice - lower) / (upper - lower)
		position = math.Max(0, math.Min(1, position)) // 限制在0-1之间
	} else {
		position = 0.5
	}

	return middle, upper, lower, width, position
}

func determineTrend(rsi, macd, signal float64) string {
	score := 0

	// RSI判断
	if rsi > 70 {
		score -= 2 // 超买，偏空
	} else if rsi < 30 {
		score += 2 // 超卖，偏多
	} else if rsi > 50 {
		score -= 1 // 中性偏空
	} else {
		score += 1 // 中性偏多
	}

	// MACD判断
	if macd > signal {
		score += 1 // MACD在信号线上方，偏多
	} else {
		score -= 1 // MACD在信号线下方，偏空
	}

	// 根据综合得分判断趋势
	if score >= 2 {
		return "up"
	} else if score <= -2 {
		return "down"
	} else {
		return "sideways"
	}
}

func calculateSignalStrength(indicators TechnicalIndicators) float64 {
	score := 0.0

	// RSI贡献
	if indicators.RSI < 30 || indicators.RSI > 70 {
		score += 20 // 极端值
	} else if indicators.RSI < 40 || indicators.RSI > 60 {
		score += 10 // 偏离中性
	}

	// MACD贡献
	if indicators.MACD > indicators.MACDSignal {
		score += 15 // 金叉
	} else {
		score -= 15 // 死叉
	}

	// 布林带位置贡献
	if indicators.BBPosition < 0.2 || indicators.BBPosition > 0.8 {
		score += 10 // 极端位置
	}

	// 确保在0-100范围内
	return math.Max(0, math.Min(100, score+50))
}

func calculateRiskLevel(rsi, bbPosition, volatility float64) string {
	riskScore := 0

	// RSI风险
	if rsi < 20 || rsi > 80 {
		riskScore += 30 // 极端超买超卖
	} else if rsi < 30 || rsi > 70 {
		riskScore += 20
	}

	// 布林带位置风险
	if bbPosition < 0.1 || bbPosition > 0.9 {
		riskScore += 25 // 极端位置
	} else if bbPosition < 0.2 || bbPosition > 0.8 {
		riskScore += 15
	}

	// 波动率风险
	if volatility > 0.05 {
		riskScore += 20 // 高波动
	} else if volatility > 0.03 {
		riskScore += 10
	}

	// 根据风险得分确定等级
	switch {
	case riskScore >= 60:
		return "critical"
	case riskScore >= 40:
		return "high"
	case riskScore >= 20:
		return "medium"
	default:
		return "low"
	}
}

// 其他技术指标计算函数
func calculateKDJ(highs, lows, closes []float64, period int) (float64, float64, float64) {
	if len(highs) < period || len(lows) < period || len(closes) < period {
		return 50, 50, 50
	}

	// 计算最高高和最低低
	high := highs[len(highs)-1]
	low := lows[len(lows)-1]
	for i := len(highs) - period; i < len(highs); i++ {
		if highs[i] > high {
			high = highs[i]
		}
		if lows[i] < low {
			low = lows[i]
		}
	}

	// 计算K值
	var k float64
	if high != low {
		k = ((closes[len(closes)-1] - low) / (high - low)) * 100
	} else {
		k = 50
	}

	// 这里简化处理，实际应该计算EMA
	d := k
	j := 3*k - 2*d

	return k, d, j
}

func calculateOBV(closes, volumes []float64) float64 {
	if len(closes) != len(volumes) || len(closes) < 2 {
		return 0
	}

	obv := 0.0
	for i := 1; i < len(closes); i++ {
		if closes[i] > closes[i-1] {
			obv += volumes[i]
		} else if closes[i] < closes[i-1] {
			obv -= volumes[i]
		}
		// 价格相等时保持不变
	}

	return obv
}

func calculateSupportResistance(highs, lows, closes []float64, period int) (float64, float64, float64, float64) {
	if len(highs) < period || len(lows) < period {
		return 0, 0, 0, 0
	}

	// 简化实现：返回最近period周期的最高和最低
	maxHigh := highs[len(highs)-period]
	minLow := lows[len(lows)-period]

	for i := len(highs) - period; i < len(highs); i++ {
		if highs[i] > maxHigh {
			maxHigh = highs[i]
		}
		if lows[i] < minLow {
			minLow = lows[i]
		}
	}

	// 计算强度（基于价格接近程度的简化计算）
	currentPrice := closes[len(closes)-1]
	resistanceStrength := math.Max(0, 100-(maxHigh-currentPrice)/currentPrice*100)
	supportStrength := math.Max(0, 100-(currentPrice-minLow)/currentPrice*100)

	return minLow, maxHigh, supportStrength, resistanceStrength
}

func calculateMomentum(closes []float64, period int) float64 {
	if len(closes) < period+1 {
		return 0
	}

	current := closes[len(closes)-1]
	past := closes[len(closes)-period-1]

	if past == 0 {
		return 0
	}

	return (current - past) / past * 100
}

func calculateMomentumDivergence(closes, highs, lows []float64, period int) float64 {
	// 简化实现：返回价格动量与成交量动量的差异
	priceMomentum := calculateMomentum(closes, period)
	// 这里应该计算成交量动量，但暂时用价格动量代替
	return priceMomentum
}

func calculateVolatility(closes []float64, period int) float64 {
	if len(closes) < period {
		return 0
	}

	// 计算收益率
	returns := make([]float64, 0, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		if closes[i-1] != 0 {
			ret := (closes[i] - closes[i-1]) / closes[i-1]
			returns = append(returns, ret)
		}
	}

	if len(returns) < period {
		return 0
	}

	// 计算标准差
	sum := 0.0
	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	for _, r := range returns {
		sum += math.Pow(r-mean, 2)
	}

	return math.Sqrt(sum / float64(len(returns)))
}

func calculateWilliamsR(highs, lows, closes []float64, period int) float64 {
	if len(highs) < period || len(lows) < period || len(closes) < period {
		return -50
	}

	// 找到周期内的最高高和最低低
	high := highs[len(highs)-period]
	low := lows[len(lows)-period]

	for i := len(highs) - period; i < len(highs); i++ {
		if highs[i] > high {
			high = highs[i]
		}
		if lows[i] < low {
			low = lows[i]
		}
	}

	if high == low {
		return -50
	}

	current := closes[len(closes)-1]
	wr := ((high - current) / (high - low)) * -100

	return math.Max(-100, math.Min(0, wr))
}

func calculateCCI(highs, lows, closes []float64, period int) float64 {
	if len(highs) < period || len(lows) < period || len(closes) < period {
		return 0
	}

	// 计算典型价格
	typicalPrices := make([]float64, len(closes))
	for i := 0; i < len(closes); i++ {
		typicalPrices[i] = (highs[i] + lows[i] + closes[i]) / 3
	}

	// 计算SMA
	sma := calculateSMA(typicalPrices, period)

	// 计算平均偏差
	sum := 0.0
	for i := len(typicalPrices) - period; i < len(typicalPrices); i++ {
		sum += math.Abs(typicalPrices[i] - sma)
	}
	meanDeviation := sum / float64(period)

	if meanDeviation == 0 {
		return 0
	}

	// 计算CCI
	currentTP := typicalPrices[len(typicalPrices)-1]
	return (currentTP - sma) / (0.015 * meanDeviation)
}

// K线数据缓存管理
func (s *Server) getKlinesWithCache(ctx context.Context, symbol, kind, interval string, limit int) ([]BinanceKline, error) {
	gdb := s.db.DB()
	if gdb == nil {
		// 数据库不可用，直接从API获取
		return s.fetchBinanceKlines(ctx, symbol, kind, interval, limit)
	}

	// 转换为API使用的交易对格式进行查询
	apiSymbol := s.convertToBinanceSymbol(symbol, kind)

	// 计算时间范围（过去的数据）
	endTime := time.Now().UTC()
	var startTime *time.Time
	switch interval {
	case "1m":
		t := endTime.Add(-time.Duration(limit) * time.Minute)
		startTime = &t
	case "5m":
		t := endTime.Add(-time.Duration(limit*5) * time.Minute)
		startTime = &t
	case "15m":
		t := endTime.Add(-time.Duration(limit*15) * time.Minute)
		startTime = &t
	case "30m":
		t := endTime.Add(-time.Duration(limit*30) * time.Minute)
		startTime = &t
	case "1h":
		t := endTime.Add(-time.Duration(limit) * time.Hour)
		startTime = &t
	case "4h":
		t := endTime.Add(-time.Duration(limit*4) * time.Hour)
		startTime = &t
	case "1d":
		t := endTime.AddDate(0, 0, -limit)
		startTime = &t
	default:
		// 默认1小时
		t := endTime.Add(-time.Duration(limit) * time.Hour)
		startTime = &t
	}

	// 从数据库查询K线数据
	dbKlines, err := pdb.GetMarketKlines(gdb, apiSymbol, kind, interval, startTime, &endTime, limit)
	if err != nil {
		log.Printf("[KlineCache] Failed to query klines from DB: %v", err)
		// 数据库查询失败，从API获取并验证
		return s.fetchAndProcessKlines(ctx, apiSymbol, kind, interval, limit)
	}

	// 检查数据量是否足够
	if len(dbKlines) < limit/2 {
		log.Printf("[KlineCache] Insufficient cached data (%d/%d), fetching from API", len(dbKlines), limit)
		return s.fetchAndProcessKlines(ctx, apiSymbol, kind, interval, limit)
	}

	// 检查最新数据是否新鲜
	maxAge := getMaxAgeForInterval(interval)
	isFresh, err := pdb.IsKlineDataFresh(gdb, apiSymbol, kind, interval, maxAge)
	if err != nil || !isFresh {
		log.Printf("[KlineCache] Cached data is stale, fetching from API")
		return s.fetchAndProcessKlines(ctx, apiSymbol, kind, interval, limit)
	}

	// 数据足够新鲜，直接使用缓存数据
	log.Printf("[KlineCache] Using cached klines: %d data points", len(dbKlines))

	// 转换为BinanceKline格式并验证
	binanceKlines := make([]BinanceKline, len(dbKlines))
	for i, kline := range dbKlines {
		binanceKlines[i] = BinanceKline{
			OpenTime:  float64(kline.OpenTime.UnixMilli()),
			Open:      kline.OpenPrice,
			High:      kline.HighPrice,
			Low:       kline.LowPrice,
			Close:     kline.ClosePrice,
			Volume:    kline.Volume,
			CloseTime: float64(kline.OpenTime.Add(getIntervalDuration(interval)).UnixMilli()),
		}
	}

	// 验证和处理数据
	validatedKlines, err := s.ValidateAndCleanKlines(binanceKlines, symbol, interval, kind)
	if err != nil {
		log.Printf("[KlineCache] Data validation failed: %v", err)
		return s.fetchAndProcessKlines(ctx, apiSymbol, kind, interval, limit)
	}

	// 转换为BinanceKline格式返回
	result := make([]BinanceKline, len(validatedKlines))
	for i, kline := range validatedKlines {
		result[i] = kline.BinanceKline
	}

	return result, nil
}

// convertToBinanceSymbol 将基础币种转换为Binance交易对格式
func (s *Server) convertToBinanceSymbol(symbol string, kind string) string {
	upperSymbol := strings.ToUpper(symbol)

	// 将基础币种转换为相应格式
	switch kind {
	case "spot":
		// 如果已经是交易对格式，直接返回，否则添加USDT
		if strings.Contains(upperSymbol, "USDT") ||
			strings.Contains(upperSymbol, "BUSD") ||
			strings.Contains(upperSymbol, "USDC") ||
			strings.Contains(upperSymbol, "BTC") && len(upperSymbol) > 3 ||
			strings.Contains(upperSymbol, "ETH") && len(upperSymbol) > 3 {
			return upperSymbol
		}
		return upperSymbol + "USDT"
	case "futures":
		// 币本位期货合约 - 将USDT格式转换为USD_PERP格式
		if strings.HasSuffix(upperSymbol, "USDT") {
			// 如果是USDT格式，转换为币本位格式
			baseSymbol := strings.TrimSuffix(upperSymbol, "USDT")
			return baseSymbol + "USD_PERP"
		} else if strings.HasSuffix(upperSymbol, "USD_PERP") {
			// 如果已经是币本位格式，直接返回
			return upperSymbol
		}
		// 如果不包含USDT且不是币本位格式，可能是基础币种，添加USD_PERP
		return upperSymbol + "USD_PERP"
	default:
		// 默认使用现货交易对
		if strings.Contains(upperSymbol, "USDT") ||
			strings.Contains(upperSymbol, "BUSD") ||
			strings.Contains(upperSymbol, "USDC") ||
			strings.Contains(upperSymbol, "BTC") && len(upperSymbol) > 3 ||
			strings.Contains(upperSymbol, "ETH") && len(upperSymbol) > 3 {
			return upperSymbol
		}
		return upperSymbol + "USDT"
	}
}

// fetchAndSaveKlines 从API获取K线数据并保存到数据库
func (s *Server) fetchAndSaveKlines(ctx context.Context, symbol, kind, interval string, limit int) ([]BinanceKline, error) {
	// 从API获取数据
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, interval, limit)
	if err != nil {
		return nil, err
	}

	// 保存到数据库（使用API格式的symbol）
	gdb := s.db.DB()
	if gdb != nil && len(klines) > 0 {
		// 转换为API使用的交易对格式进行保存
		apiSymbol := s.convertToBinanceSymbol(symbol, kind)
		dbKlines := make([]pdb.MarketKline, len(klines))
		for i, kline := range klines {
			dbKlines[i] = pdb.MarketKline{
				Symbol:     apiSymbol, // 使用转换后的交易对格式保存
				Kind:       kind,
				Interval:   interval,
				OpenTime:   time.UnixMilli(int64(kline.OpenTime)).UTC(),
				OpenPrice:  kline.Open,
				HighPrice:  kline.High,
				LowPrice:   kline.Low,
				ClosePrice: kline.Close,
				Volume:     kline.Volume,
			}
		}

		if err := pdb.SaveMarketKlines(gdb, dbKlines); err != nil {
			log.Printf("[KlineCache] Failed to save klines to DB: %v", err)
			// 保存失败不影响返回数据
		} else {
			log.Printf("[KlineCache] Saved %d klines to cache", len(dbKlines))
		}
	}

	return klines, nil
}

// fetchAndProcessKlines 从API获取K线数据，进行验证和处理
func (s *Server) fetchAndProcessKlines(ctx context.Context, symbol, kind, interval string, limit int) ([]BinanceKline, error) {
	// 从API获取原始数据
	rawKlines, err := s.fetchBinanceKlines(ctx, symbol, kind, interval, limit)
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	// 验证和清洗数据
	validatedKlines, err := s.ValidateAndCleanKlines(rawKlines, symbol, interval, kind)
	if err != nil {
		return nil, fmt.Errorf("K线数据验证失败: %w", err)
	}

	// 后处理数据
	processedKlines, err := s.ProcessKlineData(validatedKlines)
	if err != nil {
		return nil, fmt.Errorf("K线数据后处理失败: %w", err)
	}

	// 保存到数据库（异步，不影响返回）
	if gdb := s.db.DB(); gdb != nil && len(processedKlines) > 0 {
		go func() {
			dbKlines := make([]pdb.MarketKline, 0, len(processedKlines))
			for _, kline := range processedKlines {
				if kline.IsValid { // 只保存有效数据
					dbKlines = append(dbKlines, pdb.MarketKline{
						Symbol:     s.convertToBinanceSymbol(kline.Symbol, kline.Kind),
						Kind:       kline.Kind,
						Interval:   kline.Interval,
						OpenTime:   kline.Timestamp,
						OpenPrice:  kline.Open,
						HighPrice:  kline.High,
						LowPrice:   kline.Low,
						ClosePrice: kline.Close,
						Volume:     kline.Volume,
					})
				}
			}

			if len(dbKlines) > 0 {
				if err := pdb.SaveMarketKlines(gdb, dbKlines); err != nil {
					log.Printf("[KlineCache] Failed to save processed klines to DB: %v", err)
				} else {
					log.Printf("[KlineCache] Saved %d validated klines to cache", len(dbKlines))
				}
			}
		}()
	}

	// 返回处理后的数据
	result := make([]BinanceKline, len(processedKlines))
	for i, kline := range processedKlines {
		result[i] = kline.BinanceKline
	}

	log.Printf("[KlineProcessing] 返回处理后的K线数据: %s %s, %d 条", symbol, interval, len(result))
	return result, nil
}

// fetchBinanceKlines 从Binance API获取K线数据
func (s *Server) fetchBinanceKlines(ctx context.Context, symbol, kind, interval string, limit int) ([]BinanceKline, error) {
	return s.fetchBinanceKlinesWithTimeRange(ctx, symbol, kind, interval, limit, nil, nil)
}

// fetchBinanceKlinesWithTimeRange 从Binance API获取指定时间范围的K线数据
func (s *Server) fetchBinanceKlinesWithTimeRange(ctx context.Context, symbol, kind, interval string, limit int, startTime, endTime *time.Time) ([]BinanceKline, error) {
	// 将基础币种转换为Binance交易对格式
	apiSymbol := s.convertToBinanceSymbol(symbol, kind)

	var baseURL string
	switch kind {
	case "spot":
		baseURL = "https://api.binance.com/api/v3/klines"
	case "futures":
		// 币本位期货使用dapi
		baseURL = "https://dapi.binance.com/dapi/v1/klines"
	default:
		baseURL = "https://api.binance.com/api/v3/klines"
	}

	url := fmt.Sprintf("%s?symbol=%s&interval=%s&limit=%d", baseURL, apiSymbol, interval, limit)

	// 如果提供了时间范围，添加到URL参数中
	if startTime != nil {
		url += fmt.Sprintf("&startTime=%d", startTime.Unix()*1000)
	}
	if endTime != nil {
		url += fmt.Sprintf("&endTime=%d", endTime.Unix()*1000)
	}

	var raw [][]interface{}
	if err := netutil.GetJSON(ctx, url, &raw); err != nil {
		return nil, err
	}

	klines := make([]BinanceKline, 0, len(raw))
	for _, k := range raw {
		if len(k) < 6 {
			continue
		}

		// Binance API返回格式:
		// [openTime, open, high, low, close, volume, closeTime, ...]
		openTime, _ := k[0].(float64)
		open := fmt.Sprintf("%v", k[1])
		high := fmt.Sprintf("%v", k[2])
		low := fmt.Sprintf("%v", k[3])
		close := fmt.Sprintf("%v", k[4])
		volume := fmt.Sprintf("%v", k[5])
		closeTime, _ := k[6].(float64)

		klines = append(klines, BinanceKline{
			OpenTime:  openTime,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: closeTime,
		})
	}

	return klines, nil
}

// saveTechnicalIndicatorsCache 保存技术指标到数据库缓存
func (s *Server) saveTechnicalIndicatorsCache(ctx context.Context, symbol, kind, interval string, dataPoints int, indicators *TechnicalIndicators, dataFrom, dataTo time.Time) {
	gdb := s.db.DB()
	if gdb == nil {
		return // 数据库不可用，跳过缓存
	}

	indicatorsJSON, err := json.Marshal(indicators)
	if err != nil {
		log.Printf("[TechCache] Failed to marshal indicators: %v", err)
		return
	}

	sql := fmt.Sprintf(`
		INSERT INTO technical_indicators_caches (
			symbol, kind, %s, data_points, indicators,
			calculated_at, data_from, data_to, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, NOW(3), NOW(3))
		ON DUPLICATE KEY UPDATE
			indicators = VALUES(indicators),
			calculated_at = VALUES(calculated_at),
			data_from = VALUES(data_from),
			data_to = VALUES(data_to),
			updated_at = NOW(3)
	`, "`interval`")

	err = gdb.Exec(sql,
		symbol, kind, interval, dataPoints, indicatorsJSON,
		time.Now(), dataFrom, dataTo,
	).Error

	if err != nil {
		log.Printf("[TechCache] Failed to save indicators cache: %v", err)
	} else {
		log.Printf("[TechCache] Saved technical indicators cache for %s", symbol)
	}
}

// getMaxAgeForInterval 根据时间间隔返回最大年龄
func getMaxAgeForInterval(interval string) time.Duration {
	switch interval {
	case "1m":
		return 5 * time.Minute
	case "5m", "15m", "30m":
		return 30 * time.Minute
	case "1h":
		return 2 * time.Hour
	case "4h":
		return 8 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return 2 * time.Hour
	}
}

// ===== K线数据聚合函数 =====

// AggregateKlines 将K线数据聚合到更大的时间周期
func (s *Server) AggregateKlines(klines []KlineData, targetInterval string) ([]KlineData, error) {
	if len(klines) == 0 {
		return klines, nil
	}

	// 如果目标间隔与当前间隔相同，直接返回
	currentInterval := klines[0].Interval
	if currentInterval == targetInterval {
		return klines, nil
	}

	// 计算聚合倍数
	currentDuration := getIntervalDuration(currentInterval)
	targetDuration := getIntervalDuration(targetInterval)

	if targetDuration <= currentDuration {
		return nil, fmt.Errorf("目标间隔必须大于当前间隔")
	}

	multiplier := int(targetDuration / currentDuration)
	if multiplier <= 1 {
		return klines, nil
	}

	// 按目标间隔分组聚合
	aggregated := make([]KlineData, 0)
	currentGroup := make([]KlineData, 0)

	for _, kline := range klines {
		// 计算所属的目标间隔组
		klineTime := time.UnixMilli(int64(kline.OpenTime)).UTC()
		targetTimestamp := klineTime.Truncate(targetDuration)

		// 如果是新组，开始新组
		if len(currentGroup) == 0 || !time.UnixMilli(int64(currentGroup[0].OpenTime)).UTC().Truncate(targetDuration).Equal(targetTimestamp) {
			// 处理上一组
			if len(currentGroup) > 0 {
				if agg := aggregateKlineGroup(currentGroup, targetTimestamp, targetInterval, klines[0].Symbol, klines[0].Kind); agg != nil {
					aggregated = append(aggregated, *agg)
				}
			}
			currentGroup = []KlineData{kline}
		} else {
			currentGroup = append(currentGroup, kline)
		}
	}

	// 处理最后一组
	if len(currentGroup) > 0 {
		if agg := aggregateKlineGroup(currentGroup, currentGroup[0].Timestamp.Truncate(targetDuration), targetInterval, klines[0].Symbol, klines[0].Kind); agg != nil {
			aggregated = append(aggregated, *agg)
		}
	}

	log.Printf("[KlineAggregation] 聚合完成: %s %s → %s, %d → %d 条",
		klines[0].Symbol, currentInterval, targetInterval, len(klines), len(aggregated))

	return aggregated, nil
}

// aggregateKlineGroup 聚合一组K线数据
func aggregateKlineGroup(group []KlineData, timestamp time.Time, interval, symbol, kind string) *KlineData {
	if len(group) == 0 {
		return nil
	}

	// 使用第一根K线的开盘价作为聚合K线的开盘价
	openPrice := group[0].Open

	// 使用最后一根K线的收盘价作为聚合K线的收盘价
	closePrice := group[len(group)-1].Close

	// 计算最高价和最低价
	var highPrice, lowPrice float64

	// 初始化最高价和最低价
	if h, err := strconv.ParseFloat(group[0].High, 64); err == nil {
		highPrice = h
		lowPrice = h // 先用第一个值初始化
	} else {
		return nil // 如果连第一个值都解析失败，返回nil
	}

	if l, err := strconv.ParseFloat(group[0].Low, 64); err == nil {
		lowPrice = l
	}

	for _, kline := range group {
		if h, err := strconv.ParseFloat(kline.High, 64); err == nil {
			if h > highPrice {
				highPrice = h
			}
		}
		if l, err := strconv.ParseFloat(kline.Low, 64); err == nil {
			if l < lowPrice {
				lowPrice = l
			}
		}
	}

	// 累加成交量
	totalVolume := 0.0
	for _, kline := range group {
		if v, err := strconv.ParseFloat(kline.Volume, 64); err == nil {
			totalVolume += v
		}
	}

	// 计算平均数据质量
	totalQuality := 0
	validCount := 0
	for _, kline := range group {
		if kline.IsValid {
			totalQuality += kline.DataQuality
			validCount++
		}
	}

	avgQuality := 100
	if validCount > 0 {
		avgQuality = totalQuality / validCount
	}

	return &KlineData{
		BinanceKline: BinanceKline{
			OpenTime:  float64(timestamp.UnixMilli()),
			Open:      openPrice,
			High:      strconv.FormatFloat(highPrice, 'f', -1, 64),
			Low:       strconv.FormatFloat(lowPrice, 'f', -1, 64),
			Close:     closePrice,
			Volume:    strconv.FormatFloat(totalVolume, 'f', -1, 64),
			CloseTime: float64(timestamp.Add(getIntervalDuration(interval)).UnixMilli()),
		},
		Symbol:      symbol,
		Interval:    interval,
		Kind:        kind,
		Timestamp:   timestamp,
		IsValid:     validCount > 0,
		DataQuality: avgQuality,
		ProcessedAt: time.Now().UTC(),
	}
}

// ConvertKlineInterval 转换K线到指定时间间隔
func (s *Server) ConvertKlineInterval(klines []BinanceKline, fromInterval, toInterval, symbol, kind string) ([]BinanceKline, error) {
	// 先转换为KlineData进行处理
	klineData := make([]KlineData, len(klines))
	for i, kline := range klines {
		klineData[i] = KlineData{
			BinanceKline: kline,
			Symbol:       symbol,
			Interval:     fromInterval,
			Kind:         kind,
			Timestamp:    time.UnixMilli(int64(kline.OpenTime)).UTC(),
			IsValid:      true,
			DataQuality:  100,
			ProcessedAt:  time.Now().UTC(),
		}
	}

	// 聚合到目标间隔
	aggregated, err := s.AggregateKlines(klineData, toInterval)
	if err != nil {
		return nil, err
	}

	// 转换回BinanceKline格式
	result := make([]BinanceKline, len(aggregated))
	for i, kline := range aggregated {
		result[i] = kline.BinanceKline
	}

	return result, nil
}

// ===== K线数据验证和处理函数 =====

// ValidateAndCleanKlines 验证并清洗K线数据
func (s *Server) ValidateAndCleanKlines(klines []BinanceKline, symbol, interval, kind string) ([]KlineData, error) {
	if len(klines) == 0 {
		return nil, fmt.Errorf("K线数据为空")
	}

	cleanedKlines := make([]KlineData, 0, len(klines))

	for i, kline := range klines {
		klineData := KlineData{
			BinanceKline: kline,
			Symbol:       symbol,
			Interval:     interval,
			Kind:         kind,
			Timestamp:    time.UnixMilli(int64(kline.OpenTime)).UTC(),
			ProcessedAt:  time.Now().UTC(),
		}

		// 验证和清洗单个K线数据
		validationResult := s.validateSingleKline(klineData, i)

		klineData.IsValid = validationResult.IsValid
		klineData.DataQuality = validationResult.DataQualityScore

		// 即使数据无效也保留，但标记状态
		cleanedKlines = append(cleanedKlines, klineData)

		// 记录验证问题
		if len(validationResult.Errors) > 0 {
			log.Printf("[KlineValidation] %s %s: %v", symbol, interval, validationResult.Errors)
		}
		if len(validationResult.Warnings) > 0 {
			log.Printf("[KlineValidation] %s %s 警告: %v", symbol, interval, validationResult.Warnings)
		}
	}

	log.Printf("[KlineProcessing] 验证完成: %s %s, 总计 %d 条, 有效 %d 条",
		symbol, interval, len(cleanedKlines),
		len(cleanedKlines)-countInvalidKlines(cleanedKlines))

	return cleanedKlines, nil
}

// validateSingleKline 验证单个K线数据
func (s *Server) validateSingleKline(kline KlineData, index int) KlineValidationResult {
	result := KlineValidationResult{
		IsValid:          true,
		Errors:           make([]string, 0),
		Warnings:         make([]string, 0),
		DataQualityScore: 100,
		Suggestions:      make([]string, 0),
	}

	// 1. 验证价格数据
	openPrice, err := strconv.ParseFloat(kline.Open, 64)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("开盘价格式错误: %s", kline.Open))
		result.IsValid = false
		result.DataQualityScore -= 30
	}

	highPrice, err := strconv.ParseFloat(kline.High, 64)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("最高价格式错误: %s", kline.High))
		result.IsValid = false
		result.DataQualityScore -= 30
	}

	lowPrice, err := strconv.ParseFloat(kline.Low, 64)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("最低价格式错误: %s", kline.Low))
		result.IsValid = false
		result.DataQualityScore -= 30
	}

	closePrice, err := strconv.ParseFloat(kline.Close, 64)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("收盘价格式错误: %s", kline.Close))
		result.IsValid = false
		result.DataQualityScore -= 30
	}

	// 2. 验证价格逻辑关系
	if result.IsValid {
		if highPrice < openPrice || highPrice < closePrice {
			result.Warnings = append(result.Warnings, "最高价低于开盘价或收盘价")
			result.DataQualityScore -= 10
		}

		if lowPrice > openPrice || lowPrice > closePrice {
			result.Warnings = append(result.Warnings, "最低价高于开盘价或收盘价")
			result.DataQualityScore -= 10
		}

		if highPrice < lowPrice {
			result.Errors = append(result.Errors, "最高价低于最低价")
			result.IsValid = false
			result.DataQualityScore -= 50
		}
	}

	// 3. 验证成交量数据
	if volume, err := strconv.ParseFloat(kline.Volume, 64); err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("成交量格式问题: %s", kline.Volume))
		result.DataQualityScore -= 5
	} else if volume < 0 {
		result.Errors = append(result.Errors, "成交量为负数")
		result.IsValid = false
		result.DataQualityScore -= 20
	}

	// 4. 验证时间戳
	if kline.OpenTime <= 0 {
		result.Errors = append(result.Errors, "开盘时间戳无效")
		result.IsValid = false
		result.DataQualityScore -= 20
	}

	if kline.CloseTime <= kline.OpenTime {
		result.Warnings = append(result.Warnings, "收盘时间不晚于开盘时间")
		result.DataQualityScore -= 10
	}

	// 5. 验证数据连续性（与前后数据比较）
	// 这里可以添加更多连续性检查逻辑

	// 确保分数不低于0
	if result.DataQualityScore < 0 {
		result.DataQualityScore = 0
	}

	return result
}

// countInvalidKlines 统计无效K线数量
func countInvalidKlines(klines []KlineData) int {
	count := 0
	for _, kline := range klines {
		if !kline.IsValid {
			count++
		}
	}
	return count
}

// ProcessKlineData 对K线数据进行后处理和增强
func (s *Server) ProcessKlineData(klines []KlineData) ([]KlineData, error) {
	if len(klines) == 0 {
		return klines, nil
	}

	processedKlines := make([]KlineData, len(klines))
	copy(processedKlines, klines)

	// 1. 排序确保时间顺序正确
	sort.Slice(processedKlines, func(i, j int) bool {
		return processedKlines[i].OpenTime < processedKlines[j].OpenTime
	})

	// 2. 填充缺失数据（如果需要）
	processedKlines = s.fillMissingKlines(processedKlines)

	// 3. 计算额外的技术指标
	processedKlines = s.enrichKlinesWithIndicators(processedKlines)

	// 4. 数据平滑和异常值处理
	processedKlines = s.smoothKlineData(processedKlines)

	log.Printf("[KlineProcessing] 数据后处理完成: %s %s, %d 条K线",
		processedKlines[0].Symbol, processedKlines[0].Interval, len(processedKlines))

	return processedKlines, nil
}

// fillMissingKlines 填充缺失的K线数据
func (s *Server) fillMissingKlines(klines []KlineData) []KlineData {
	if len(klines) < 2 {
		return klines
	}

	interval := klines[0].Interval
	intervalDuration := getIntervalDuration(interval)

	filledKlines := make([]KlineData, 0, len(klines)*2)

	for i := 0; i < len(klines)-1; i++ {
		current := klines[i]
		next := klines[i+1]

		filledKlines = append(filledKlines, current)

		// 检查是否有缺失的K线
		expectedNextTime := current.Timestamp.Add(intervalDuration)
		if next.Timestamp.Sub(expectedNextTime) > intervalDuration {
			// 有缺失数据，创建占位符
			missingKline := KlineData{
				Symbol:      current.Symbol,
				Interval:    current.Interval,
				Kind:        current.Kind,
				Timestamp:   expectedNextTime,
				IsValid:     false,
				DataQuality: 0,
				ProcessedAt: time.Now().UTC(),
			}
			// 设置默认OHLC值（使用前一根K线的收盘价）
			closePrice := current.Close
			missingKline.BinanceKline = BinanceKline{
				OpenTime:  float64(expectedNextTime.UnixMilli()),
				Open:      closePrice,
				High:      closePrice,
				Low:       closePrice,
				Close:     closePrice,
				Volume:    "0",
				CloseTime: float64(expectedNextTime.Add(intervalDuration).UnixMilli()),
			}
			filledKlines = append(filledKlines, missingKline)
			log.Printf("[KlineProcessing] 填充缺失K线: %s at %v", current.Symbol, expectedNextTime)
		}
	}

	// 添加最后一个K线
	if len(klines) > 0 {
		filledKlines = append(filledKlines, klines[len(klines)-1])
	}

	return filledKlines
}

// enrichKlinesWithIndicators 为K线数据添加基础技术指标
func (s *Server) enrichKlinesWithIndicators(klines []KlineData) []KlineData {
	// 这里可以添加简单的技术指标计算
	// 如移动平均线、价格变化率等
	// 目前保持原样，将来扩展
	return klines
}

// smoothKlineData 数据平滑处理，处理异常值
func (s *Server) smoothKlineData(klines []KlineData) []KlineData {
	// 简单的异常值检测和处理
	// 可以根据价格波动幅度、成交量异常等规则进行平滑处理
	return klines
}

// getIntervalDuration 获取时间间隔的持续时间
func getIntervalDuration(interval string) time.Duration {
	switch interval {
	case "1m":
		return time.Minute
	case "5m":
		return 5 * time.Minute
	case "15m":
		return 15 * time.Minute
	case "30m":
		return 30 * time.Minute
	case "1h":
		return time.Hour
	case "4h":
		return 4 * time.Hour
	case "1d":
		return 24 * time.Hour
	default:
		return time.Hour
	}
}

// DMIResult DMI指标结果
type DMIResult struct {
	DIP float64 // Plus Directional Indicator
	DIM float64 // Minus Directional Indicator
	DX  float64 // Directional Movement Index
	ADX float64 // Average Directional Index
}

// CalculateDMI 计算DMI指标
func CalculateDMI(highs, lows, closes []float64, period int) DMIResult {
	if len(highs) < period+1 || len(lows) < period+1 || len(closes) < period+1 {
		return DMIResult{}
	}

	n := len(highs)
	dxValues := make([]float64, 0, n-period)

	// 计算DM+和DM-
	for i := 1; i < n; i++ {
		highDiff := highs[i] - highs[i-1]
		lowDiff := lows[i-1] - lows[i]

		var dmPlus, dmMinus float64

		if highDiff > lowDiff && highDiff > 0 {
			dmPlus = highDiff
		}
		if lowDiff > highDiff && lowDiff > 0 {
			dmMinus = lowDiff
		}

		// 计算TR (True Range)
		tr := math.Max(highs[i]-lows[i],
			math.Max(math.Abs(highs[i]-closes[i-1]), math.Abs(lows[i]-closes[i-1])))

		if tr > 0 {
			dip := (dmPlus / tr) * 100
			dim := (dmMinus / tr) * 100

			// 计算DX
			dx := math.Abs(dip-dim) / (dip + dim) * 100
			dxValues = append(dxValues, dx)
		}
	}

	// 计算ADX (DX的EMA)
	adx := 0.0
	if len(dxValues) >= period {
		// 简化的ADX计算
		sum := 0.0
		for i := len(dxValues) - period; i < len(dxValues); i++ {
			sum += dxValues[i]
		}
		adx = sum / float64(period)
	}

	// 计算最新的DIP和DIM
	var dip, dim, dx float64
	if len(dxValues) > 0 {
		// 使用最新的值
		dx = dxValues[len(dxValues)-1]

		// 重新计算最新的DIP和DIM
		i := n - 1
		highDiff := highs[i] - highs[i-1]
		lowDiff := lows[i-1] - lows[i]

		var dmPlus, dmMinus float64
		if highDiff > lowDiff && highDiff > 0 {
			dmPlus = highDiff
		}
		if lowDiff > highDiff && lowDiff > 0 {
			dmMinus = lowDiff
		}

		tr := math.Max(highs[i]-lows[i],
			math.Max(math.Abs(highs[i]-closes[i-1]), math.Abs(lows[i]-closes[i-1])))

		if tr > 0 {
			dip = (dmPlus / tr) * 100
			dim = (dmMinus / tr) * 100
		}
	}

	return DMIResult{
		DIP: dip,
		DIM: dim,
		DX:  dx,
		ADX: adx,
	}
}

// IchimokuResult 一目均衡表结果
type IchimokuResult struct {
	TenkanSen   float64 // 转换线 (Tenkan-sen)
	KijunSen    float64 // 基准线 (Kijun-sen)
	SenkouSpanA float64 // 先行带A (Senkou Span A)
	SenkouSpanB float64 // 先行带B (Senkou Span B)
	ChikouSpan  float64 // 延迟线 (Chikou Span)
}

// CalculateIchimoku 计算一目均衡表
func CalculateIchimoku(highs, lows, closes []float64) IchimokuResult {
	if len(highs) < 52 || len(lows) < 52 || len(closes) < 52 {
		return IchimokuResult{}
	}

	// 转换线 (Tenkan-sen): (9日最高+9日最低)/2
	tenkanHigh := math.Inf(-1)
	tenkanLow := math.Inf(1)
	for i := len(highs) - 9; i < len(highs); i++ {
		if highs[i] > tenkanHigh {
			tenkanHigh = highs[i]
		}
		if lows[i] < tenkanLow {
			tenkanLow = lows[i]
		}
	}
	tenkanSen := (tenkanHigh + tenkanLow) / 2

	// 基准线 (Kijun-sen): (26日最高+26日最低)/2
	kijunHigh := math.Inf(-1)
	kijunLow := math.Inf(1)
	for i := len(highs) - 26; i < len(highs); i++ {
		if highs[i] > kijunHigh {
			kijunHigh = highs[i]
		}
		if lows[i] < kijunLow {
			kijunLow = lows[i]
		}
	}
	kijunSen := (kijunHigh + kijunLow) / 2

	// 先行带A (Senkou Span A): (转换线+基准线)/2，投影到未来26日
	senkouSpanA := (tenkanSen + kijunSen) / 2

	// 先行带B (Senkou Span B): (52日最高+52日最低)/2，投影到未来26日
	spanBHigh := math.Inf(-1)
	spanBLow := math.Inf(1)
	for i := len(highs) - 52; i < len(highs); i++ {
		if highs[i] > spanBHigh {
			spanBHigh = highs[i]
		}
		if lows[i] < spanBLow {
			spanBLow = lows[i]
		}
	}
	senkouSpanB := (spanBHigh + spanBLow) / 2

	// 延迟线 (Chikou Span): 当前收盘价，投影到过去26日
	chikouSpan := closes[len(closes)-1]

	return IchimokuResult{
		TenkanSen:   tenkanSen,
		KijunSen:    kijunSen,
		SenkouSpanA: senkouSpanA,
		SenkouSpanB: senkouSpanB,
		ChikouSpan:  chikouSpan,
	}
}
