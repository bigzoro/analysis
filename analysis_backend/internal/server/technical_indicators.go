package server

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strconv"
	"time"

	pdb "analysis/internal/db"
	"analysis/internal/netutil"
)

// TechnicalIndicators 技术指标
type TechnicalIndicators struct {
	// 现有指标
	RSI        float64 `json:"rsi"`         // 相对强弱指标 0-100
	MACD       float64 `json:"macd"`        // MACD值
	MACDSignal float64 `json:"macd_signal"` // MACD信号线
	MACDHist   float64 `json:"macd_hist"`   // MACD柱状图
	Trend      string  `json:"trend"`       // "up"/"down"/"sideways"

	// 新增指标：布林带
	BBUpper    float64 `json:"bb_upper"`    // 布林带上轨
	BBMiddle   float64 `json:"bb_middle"`  // 布林带中轨（SMA20）
	BBLower    float64 `json:"bb_lower"`    // 布林带下轨
	BBWidth    float64 `json:"bb_width"`    // 布林带宽度（百分比）
	BBPosition float64 `json:"bb_position"` // 价格在布林带中的位置 0-1（0=下轨，1=上轨）

	// 新增指标：KDJ
	K float64 `json:"k"` // K值 0-100
	D float64 `json:"d"` // D值 0-100
	J float64 `json:"j"` // J值 0-100

	// 新增指标：均线系统
	MA5   float64 `json:"ma5"`   // 5日均线
	MA10  float64 `json:"ma10"`  // 10日均线
	MA20  float64 `json:"ma20"`  // 20日均线
	MA60  float64 `json:"ma60"`  // 60日均线
	MA200 float64 `json:"ma200"` // 200日均线（如果有足够数据）

	// 新增指标：成交量
	OBV        float64 `json:"obv"`          // 能量潮（On-Balance Volume）
	VolumeMA5  float64 `json:"volume_ma5"`   // 5日成交量均线
	VolumeMA20 float64 `json:"volume_ma20"` // 20日成交量均线
	VolumeRatio float64 `json:"volume_ratio"` // 成交量比率（当前成交量/20日均量）

	// 新增指标：支撑位/阻力位
	SupportLevel        float64 `json:"support_level"`         // 支撑位价格
	ResistanceLevel     float64 `json:"resistance_level"`     // 阻力位价格
	DistanceToSupport   float64 `json:"distance_to_support"`   // 距离支撑位的百分比
	DistanceToResistance float64 `json:"distance_to_resistance"` // 距离阻力位的百分比
}

// BinanceKline 币安K线数据
type BinanceKline struct {
	OpenTime  int64   `json:"openTime"`
	Open      string  `json:"open"`
	High      string  `json:"high"`
	Low       string  `json:"low"`
	Close     string  `json:"close"`
	Volume    string  `json:"volume"`
	CloseTime int64   `json:"closeTime"`
}

// GetTechnicalIndicators 获取技术指标
// 从Binance API获取K线数据并计算RSI和MACD
func (s *Server) GetTechnicalIndicators(ctx context.Context, symbol string, kind string) (*TechnicalIndicators, error) {
	// 从Binance API获取K线数据
	klines, err := s.fetchBinanceKlines(ctx, symbol, kind, "1h", 200) // 获取200根1小时K线
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	if len(klines) < 26 {
		// 数据不足，返回默认值
		return &TechnicalIndicators{
			RSI:  50,
			MACD: 0,
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
	bbUpper, bbMiddle, bbLower, bbWidth, bbPosition := calculateBollingerBands(closes, 20, 2.0)

	// 计算KDJ（9周期）
	k, d, j := calculateKDJ(highs, lows, closes, 9)

	// 计算均线系统
	ma5 := calculateSMA(closes, 5)
	ma10 := calculateSMA(closes, 10)
	ma20 := calculateSMA(closes, 20)
	ma60 := calculateSMA(closes, 60)
	ma200 := 0.0
	if len(closes) >= 200 {
		ma200 = calculateSMA(closes, 200)
	}

	// 计算成交量指标
	obv := calculateOBV(closes, volumes)
	volumeMA5 := calculateSMA(volumes, 5)
	volumeMA20 := calculateSMA(volumes, 20)
	volumeRatio := 0.0
	if volumeMA20 > 0 {
		volumeRatio = volumes[len(volumes)-1] / volumeMA20
	}

	// 计算支撑位/阻力位
	support, resistance := calculateSupportResistance(highs, lows, closes)
	currentPrice := closes[len(closes)-1]
	distanceToSupport := 0.0
	distanceToResistance := 0.0
	if support > 0 {
		distanceToSupport = ((currentPrice - support) / support) * 100
	}
	if resistance > 0 {
		distanceToResistance = ((resistance - currentPrice) / currentPrice) * 100
	}

	// 使用更多指标判断趋势
	trend := determineTrendAdvanced(rsi, macd, signal, k, d, ma5, ma10, ma20, bbPosition)

	return &TechnicalIndicators{
		RSI:        rsi,
		MACD:       macd,
		MACDSignal: signal,
		MACDHist:   hist,
		Trend:      trend,
		// 布林带
		BBUpper:    bbUpper,
		BBMiddle:   bbMiddle,
		BBLower:    bbLower,
		BBWidth:    bbWidth,
		BBPosition: bbPosition,
		// KDJ
		K: k,
		D: d,
		J: j,
		// 均线
		MA5:   ma5,
		MA10:  ma10,
		MA20:  ma20,
		MA60:  ma60,
		MA200: ma200,
		// 成交量
		OBV:        obv,
		VolumeMA5:  volumeMA5,
		VolumeMA20: volumeMA20,
		VolumeRatio: volumeRatio,
		// 支撑位/阻力位
		SupportLevel:        support,
		ResistanceLevel:     resistance,
		DistanceToSupport:   distanceToSupport,
		DistanceToResistance: distanceToResistance,
	}, nil
}

// fetchBinanceKlines 从Binance API获取K线数据
func (s *Server) fetchBinanceKlines(ctx context.Context, symbol, kind, interval string, limit int) ([]BinanceKline, error) {
	var baseURL string
	switch kind {
	case "spot":
		baseURL = "https://api.binance.com/api/v3/klines"
	case "futures":
		baseURL = "https://dapi.binance.com/dapi/v1/klines"
	default:
		baseURL = "https://api.binance.com/api/v3/klines"
	}

	url := fmt.Sprintf("%s?symbol=%s&interval=%s&limit=%d", baseURL, symbol, interval, limit)

	var raw [][]interface{}
	if err := netutil.GetJSON(ctx, url, &raw); err != nil {
		return nil, err
	}

	klines := make([]BinanceKline, 0, len(raw))
	for _, k := range raw {
		if len(k) < 6 {
			continue
		}

		openTime, _ := k[0].(float64)
		open, _ := k[1].(string)
		high, _ := k[2].(string)
		low, _ := k[3].(string)
		close, _ := k[4].(string)
		volume, _ := k[5].(string)
		closeTime, _ := k[6].(float64)

		klines = append(klines, BinanceKline{
			OpenTime:  int64(openTime),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
			CloseTime: int64(closeTime),
		})
	}

	return klines, nil
}

// calculateRSI 计算RSI指标
// RSI = 100 - (100 / (1 + RS))
// RS = 平均上涨幅度 / 平均下跌幅度
func calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50 // 默认中性值
	}

	// 计算价格变化
	changes := make([]float64, 0, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		changes = append(changes, prices[i]-prices[i-1])
	}

	// 计算初始平均上涨和下跌
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		if changes[i] > 0 {
			avgGain += changes[i]
		} else {
			avgLoss += math.Abs(changes[i])
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

	// 使用平滑移动平均（Wilder's smoothing）
	for i := period; i < len(changes); i++ {
		if changes[i] > 0 {
			avgGain = (avgGain*float64(period-1) + changes[i]) / float64(period)
			avgLoss = (avgLoss * float64(period-1)) / float64(period)
		} else {
			avgGain = (avgGain * float64(period-1)) / float64(period)
			avgLoss = (avgLoss*float64(period-1) + math.Abs(changes[i])) / float64(period)
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateMACD 计算MACD指标
// MACD = EMA(12) - EMA(26)
// Signal = EMA(MACD, 9)
// Histogram = MACD - Signal
func calculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) (float64, float64, float64) {
	if len(prices) < slowPeriod+signalPeriod {
		return 0, 0, 0
	}

	// 计算快线和慢线EMA
	fastEMA := calculateEMA(prices, fastPeriod)
	slowEMA := calculateEMA(prices, slowPeriod)

	// 计算MACD线
	macdLine := make([]float64, 0, len(fastEMA))
	minLen := len(fastEMA)
	if len(slowEMA) < minLen {
		minLen = len(slowEMA)
	}

	for i := 0; i < minLen; i++ {
		macdLine = append(macdLine, fastEMA[i]-slowEMA[i])
	}

	if len(macdLine) < signalPeriod {
		return 0, 0, 0
	}

	// 计算信号线（MACD的EMA）
	signalLine := calculateEMA(macdLine, signalPeriod)

	// 取最后一个值
	macd := macdLine[len(macdLine)-1]
	signal := signalLine[len(signalLine)-1]
	hist := macd - signal

	return macd, signal, hist
}

// calculateEMA 计算指数移动平均
func calculateEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	// 计算初始SMA
	var sum float64
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	sma := sum / float64(period)

	// 计算乘数
	multiplier := 2.0 / float64(period+1)

	// 计算EMA
	ema := make([]float64, 0, len(prices)-period+1)
	ema = append(ema, sma)

	for i := period; i < len(prices); i++ {
		nextEMA := (prices[i]-ema[len(ema)-1])*multiplier + ema[len(ema)-1]
		ema = append(ema, nextEMA)
	}

	return ema
}

// calculateBollingerBands 计算布林带
// period: 周期（通常20）
// numStdDev: 标准差倍数（通常2.0）
// 返回：上轨、中轨、下轨、宽度（百分比）、价格位置（0-1）
func calculateBollingerBands(prices []float64, period int, numStdDev float64) (float64, float64, float64, float64, float64) {
	if len(prices) < period {
		return 0, 0, 0, 0, 0.5
	}

	// 计算SMA（中轨）
	sma := calculateSMA(prices, period)
	if sma == 0 {
		return 0, 0, 0, 0, 0.5
	}

	// 计算标准差
	var sumSquaredDiff float64
	recentPrices := prices[len(prices)-period:]
	for _, price := range recentPrices {
		diff := price - sma
		sumSquaredDiff += diff * diff
	}
	stdDev := math.Sqrt(sumSquaredDiff / float64(period))

	// 计算上下轨
	upper := sma + numStdDev*stdDev
	lower := sma - numStdDev*stdDev

	// 计算宽度（百分比）
	width := (upper - lower) / sma * 100

	// 计算价格位置（0=下轨，1=上轨）
	currentPrice := prices[len(prices)-1]
	position := 0.5
	if upper > lower {
		position = (currentPrice - lower) / (upper - lower)
		position = math.Max(0, math.Min(1, position)) // 限制在0-1之间
	}

	return upper, sma, lower, width, position
}

// calculateKDJ 计算KDJ指标
// period: 周期（通常9）
func calculateKDJ(highs, lows, closes []float64, period int) (float64, float64, float64) {
	if len(closes) < period {
		return 50, 50, 50 // 默认中性值
	}

	// 计算RSV（未成熟随机值）
	rsv := make([]float64, 0, len(closes)-period+1)
	
	for i := period - 1; i < len(closes); i++ {
		// 获取最近period个周期的最高价和最低价
		highest := highs[i-period+1]
		lowest := lows[i-period+1]
		for j := i - period + 2; j <= i; j++ {
			if highs[j] > highest {
				highest = highs[j]
			}
			if lows[j] < lowest {
				lowest = lows[j]
			}
		}

		// 计算RSV
		if highest == lowest {
			rsv = append(rsv, 50) // 避免除以零
		} else {
			r := ((closes[i] - lowest) / (highest - lowest)) * 100
			rsv = append(rsv, r)
		}
	}

	if len(rsv) == 0 {
		return 50, 50, 50
	}

	// 计算K值（RSV的3周期EMA）
	k := calculateEMA(rsv, 3)
	if len(k) == 0 {
		return 50, 50, 50
	}
	currentK := k[len(k)-1]

	// 计算D值（K值的3周期EMA）
	d := calculateEMA(k, 3)
	if len(d) == 0 {
		return currentK, currentK, currentK
	}
	currentD := d[len(d)-1]

	// 计算J值：J = 3K - 2D
	currentJ := 3*currentK - 2*currentD

	// 限制在0-100之间
	currentK = math.Max(0, math.Min(100, currentK))
	currentD = math.Max(0, math.Min(100, currentD))
	currentJ = math.Max(0, math.Min(100, currentJ))

	return currentK, currentD, currentJ
}

// calculateSMA 计算简单移动平均
func calculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	var sum float64
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}

	return sum / float64(period)
}

// calculateOBV 计算能量潮（On-Balance Volume）
func calculateOBV(prices, volumes []float64) float64 {
	if len(prices) < 2 || len(volumes) < 2 {
		return 0
	}

	obv := 0.0
	for i := 1; i < len(prices); i++ {
		if prices[i] > prices[i-1] {
			obv += volumes[i] // 上涨，加上成交量
		} else if prices[i] < prices[i-1] {
			obv -= volumes[i] // 下跌，减去成交量
		}
		// 价格不变，OBV不变
	}

	return obv
}

// calculateSupportResistance 计算支撑位和阻力位
// 使用最近N个周期的最高价和最低价
func calculateSupportResistance(highs, lows, closes []float64) (float64, float64) {
	if len(closes) < 20 {
		return 0, 0
	}

	// 使用最近20个周期
	recentHighs := highs[len(highs)-20:]
	recentLows := lows[len(lows)-20:]

	// 阻力位：最近20个周期的最高价
	resistance := recentHighs[0]
	for _, h := range recentHighs {
		if h > resistance {
			resistance = h
		}
	}

	// 支撑位：最近20个周期的最低价
	support := recentLows[0]
	for _, l := range recentLows {
		if l < support {
			support = l
		}
	}

	return support, resistance
}

// determineTrend 根据技术指标判断趋势（基础版，保持向后兼容）
func determineTrend(rsi, macd, signal float64) string {
	return determineTrendAdvanced(rsi, macd, signal, 50, 50, 0, 0, 0, 0.5)
}

// determineTrendAdvanced 使用更多指标判断趋势
func determineTrendAdvanced(rsi, macd, signal, k, d, ma5, ma10, ma20, bbPosition float64) string {
	bullishSignals := 0
	bearishSignals := 0

	// RSI信号
	if rsi > 50 && rsi < 70 {
		bullishSignals++
	} else if rsi < 50 && rsi > 30 {
		bearishSignals++
	} else if rsi > 70 {
		bearishSignals++ // 超买
	} else if rsi < 30 {
		bullishSignals++ // 超卖可能反弹
	}

	// MACD信号
	if macd > signal {
		bullishSignals++
	} else if macd < signal {
		bearishSignals++
	}

	// KDJ信号（如果有效）
	if k > 0 && d > 0 {
		if k > d && k < 80 {
			bullishSignals++
		} else if k < d && k > 20 {
			bearishSignals++
		} else if k > 80 {
			bearishSignals++ // 超买
		} else if k < 20 {
			bullishSignals++ // 超卖
		}
	}

	// 均线信号（多头排列/空头排列）
	if ma5 > 0 && ma10 > 0 && ma20 > 0 {
		if ma5 > ma10 && ma10 > ma20 {
			bullishSignals += 2 // 多头排列，强烈看涨
		} else if ma5 < ma10 && ma10 < ma20 {
			bearishSignals += 2 // 空头排列，强烈看跌
		}
	}

	// 布林带信号
	if bbPosition > 0 {
		if bbPosition < 0.2 {
			bullishSignals++ // 接近下轨，可能反弹
		} else if bbPosition > 0.8 {
			bearishSignals++ // 接近上轨，可能回调
		}
	}

	// 判断趋势
	if bullishSignals > bearishSignals+1 {
		return "up"
	} else if bearishSignals > bullishSignals+1 {
		return "down"
	}
	return "sideways"
}

// GetTechnicalIndicatorsFromHistory 从历史快照数据计算技术指标（简化版）
// 当无法获取K线数据时，使用历史快照数据计算
func (s *Server) GetTechnicalIndicatorsFromHistory(ctx context.Context, symbol string, kind string) (*TechnicalIndicators, error) {
	// 获取最近的历史快照数据
	now := time.Now().UTC()
	startTime := now.Add(-7 * 24 * time.Hour) // 最近7天

	snaps, tops, err := pdb.ListBinanceMarket(s.db.DB(), kind, startTime, now)
	if err != nil {
		return nil, fmt.Errorf("获取历史数据失败: %w", err)
	}

	if len(snaps) < 14 {
		// 数据不足，返回默认值
		return &TechnicalIndicators{
			RSI:  50,
			MACD: 0,
			Trend: "sideways",
		}, nil
	}

	// 提取价格序列（按时间排序）
	type pricePoint struct {
		Time  time.Time
		Price float64
	}
	pricePoints := make([]pricePoint, 0)

	for _, snap := range snaps {
		if items, ok := tops[snap.ID]; ok {
			for _, item := range items {
				if item.Symbol == symbol {
					price, err := strconv.ParseFloat(item.LastPrice, 64)
					if err == nil {
						pricePoints = append(pricePoints, pricePoint{
							Time:  snap.Bucket,
							Price: price,
						})
					}
					break
				}
			}
		}
	}

	if len(pricePoints) < 14 {
		return &TechnicalIndicators{
			RSI:  50,
			MACD: 0,
			Trend: "sideways",
		}, nil
	}

	// 按时间排序
	sort.Slice(pricePoints, func(i, j int) bool {
		return pricePoints[i].Time.Before(pricePoints[j].Time)
	})

	// 提取价格数组
	prices := make([]float64, 0, len(pricePoints))
	for _, pp := range pricePoints {
		prices = append(prices, pp.Price)
	}

	// 计算RSI
	rsi := calculateRSI(prices, 14)

	// 计算MACD
	macd, signal, hist := calculateMACD(prices, 12, 26, 9)

	// 判断趋势
	trend := determineTrend(rsi, macd, signal)

	return &TechnicalIndicators{
		RSI:        rsi,
		MACD:       macd,
		MACDSignal: signal,
		MACDHist:   hist,
		Trend:      trend,
	}, nil
}

