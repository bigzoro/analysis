package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ============================================================================
// 数据结构定义
// ============================================================================

// MarketData 市场数据结构
type MarketData struct {
	Symbol         string
	Price          float64
	PriceChange    float64
	Volume         float64
	Volatility     float64
	Trend          string
	RSI            float64
	MACD           float64
	BBPosition     float64
}

// MarketEnvironment 市场环境分析结果
type MarketEnvironment struct {
	Volatility        float64
	Trend             string
	Oscillation       float64
	BullishCount      int
	BearishCount      int
	SidewaysCount     int
	AvgRSI            float64
	AvgBBPosition     float64
	MarketRegime      string
	Confidence        float64
}

// StrategyRecommendation 策略推荐结果
type StrategyRecommendation struct {
	StrategyName   string
	Score          float64
	Confidence     float64
	Reason         string
	SuitableMarket string
	RiskLevel      string
	ExpectedReturn float64
}

// TechnicalIndicators 技术指标
type TechnicalIndicators struct {
	RSI        float64
	MACD       float64
	Signal     float64
	Histogram  float64
	BBUpper    float64
	BBMiddle   float64
	BBLower    float64
	BBPosition float64
	K          float64
	D          float64
	J          float64
}

// ============================================================================
// 主函数
// ============================================================================

func main() {
	fmt.Println("🎯 市场环境分析与策略推荐系统")
	fmt.Println("=====================================")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 测试数据库连接
	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	// 执行市场环境分析
	fmt.Println("\n📊 执行市场环境分析...")
	marketEnv, err := analyzeMarketEnvironment(db)
	if err != nil {
		log.Fatal("市场环境分析失败:", err)
	}

	// 显示市场环境分析结果
	displayMarketEnvironment(marketEnv)

	// 分析主要币种的技术指标
	fmt.Println("\n📈 分析主要币种技术指标...")
	technicalData, err := analyzeTechnicalIndicators(db)
	if err != nil {
		log.Printf("⚠️  技术指标分析失败: %v，使用简化分析", err)
		technicalData = []MarketData{}
	}

	// 显示技术指标分析
	displayTechnicalAnalysis(technicalData)

	// 生成策略推荐
	fmt.Println("\n🎪 生成策略推荐...")
	recommendations := generateStrategyRecommendations(marketEnv, technicalData)

	// 显示策略推荐结果
	displayStrategyRecommendations(recommendations, marketEnv)

	// 生成操作建议
	fmt.Println("\n💡 操作建议")
	fmt.Println("───────────────────────────────")
	generateActionPlan(marketEnv, recommendations)

	fmt.Println("\n🎉 分析完成！")
}

// ============================================================================
// 市场环境分析
// ============================================================================

func analyzeMarketEnvironment(db *sql.DB) (*MarketEnvironment, error) {
	// 获取最近24小时的市场数据
	marketData, err := getMarketData(db, 24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	if len(marketData) == 0 {
		return nil, fmt.Errorf("没有找到市场数据")
	}

	// 计算基础统计
	totalSymbols := len(marketData)
	bullishCount := 0
	bearishCount := 0
	sidewaysCount := 0
	totalVolatility := 0.0
	totalPriceChange := 0.0
	totalRSI := 0.0
	validRSICount := 0

	for _, data := range marketData {
		totalVolatility += data.Volatility
		totalPriceChange += data.PriceChange

		// 统计趋势分布
		switch data.Trend {
		case "bullish":
			bullishCount++
		case "bearish":
			bearishCount++
		default:
			sidewaysCount++
		}

		// 统计RSI
		if data.RSI > 0 {
			totalRSI += data.RSI
			validRSICount++
		}
	}

	avgVolatility := totalVolatility / float64(totalSymbols)
	_ = totalPriceChange / float64(totalSymbols) // avgPriceChange not used in current implementation
	avgRSI := 0.0
	if validRSICount > 0 {
		avgRSI = totalRSI / float64(validRSICount)
	}

	// 计算震荡程度（价格变化的标准差）
	var priceChanges []float64
	for _, data := range marketData {
		priceChanges = append(priceChanges, data.PriceChange)
	}
	oscillation := calculateStandardDeviation(priceChanges)

	// 判断市场状态
	marketRegime, confidence := determineMarketRegime(avgVolatility, oscillation, bullishCount, bearishCount, totalSymbols)

	// 计算布林带位置平均值（如果有数据）
	avgBBPosition := 0.0
	validBBCount := 0
	for _, data := range marketData {
		if data.BBPosition != 0 {
			avgBBPosition += data.BBPosition
			validBBCount++
		}
	}
	if validBBCount > 0 {
		avgBBPosition /= float64(validBBCount)
	}

	return &MarketEnvironment{
		Volatility:     avgVolatility,
		Trend:          determineOverallTrend(bullishCount, bearishCount, totalSymbols),
		Oscillation:    oscillation,
		BullishCount:   bullishCount,
		BearishCount:   bearishCount,
		SidewaysCount:  sidewaysCount,
		AvgRSI:         avgRSI,
		AvgBBPosition:  avgBBPosition,
		MarketRegime:   marketRegime,
		Confidence:     confidence,
	}, nil
}

// 获取市场数据
func getMarketData(db *sql.DB, timeRange time.Duration) ([]MarketData, error) {
	endTime := time.Now()
	startTime := endTime.Add(-timeRange)

	query := `
		SELECT
			s.symbol,
			s.last_price as price,
			s.price_change_percent as price_change,
			s.volume,
			(s.high_price - s.low_price) / s.low_price * 100 as volatility
		FROM binance_24h_stats s
		WHERE s.created_at >= ? AND s.created_at <= ?
			AND s.quote_volume > 1000
		ORDER BY s.quote_volume DESC
		LIMIT 100
	`

	rows, err := db.Query(query, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var marketData []MarketData
	for rows.Next() {
		var data MarketData
		err := rows.Scan(&data.Symbol, &data.Price, &data.PriceChange, &data.Volume, &data.Volatility)
		if err != nil {
			continue // 跳过有问题的行
		}
		marketData = append(marketData, data)
	}

	// 为每个币种计算技术指标和趋势
	for i := range marketData {
		// 计算趋势
		marketData[i].Trend = determineSymbolTrend(marketData[i].PriceChange)

		// 尝试获取技术指标
		indicators, err := calculateSymbolIndicators(db, marketData[i].Symbol)
		if err == nil {
			marketData[i].RSI = indicators.RSI
			marketData[i].MACD = indicators.MACD
			marketData[i].BBPosition = indicators.BBPosition
		}
	}

	return marketData, nil
}

// ============================================================================
// 技术指标分析
// ============================================================================

func analyzeTechnicalIndicators(db *sql.DB) ([]MarketData, error) {
	// 获取主要币种
	majorSymbols := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT"}

	var technicalData []MarketData

	for _, symbol := range majorSymbols {
		indicators, err := calculateSymbolIndicators(db, symbol)
		if err != nil {
			log.Printf("⚠️  计算 %s 技术指标失败: %v", symbol, err)
			continue
		}

		// 获取基本价格数据
		price, priceChange, err := getSymbolPriceData(db, symbol)
		if err != nil {
			log.Printf("⚠️  获取 %s 价格数据失败: %v", symbol, err)
			continue
		}

		data := MarketData{
			Symbol:     symbol,
			Price:      price,
			PriceChange: priceChange,
			RSI:        indicators.RSI,
			MACD:       indicators.MACD,
			BBPosition: indicators.BBPosition,
			Trend:      determineSymbolTrend(priceChange),
		}

		technicalData = append(technicalData, data)
	}

	return technicalData, nil
}

// 计算单个币种的技术指标
func calculateSymbolIndicators(db *sql.DB, symbol string) (*TechnicalIndicators, error) {
	// 获取最近30天的K线数据
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -30)

	query := `
		SELECT close_price
		FROM market_klines
		WHERE symbol = ? AND open_time >= ? AND open_time <= ?
		ORDER BY open_time ASC
	`

	rows, err := db.Query(query, symbol, startTime, endTime)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var prices []float64
	for rows.Next() {
		var price float64
		if err := rows.Scan(&price); err == nil {
			prices = append(prices, price)
		}
	}

	if len(prices) < 26 {
		return nil, fmt.Errorf("数据不足")
	}

	// 计算技术指标
	rsi := calculateRSI(prices, 14)
	bbMiddle, bbUpper, bbLower, _, bbPosition := calculateBollingerBands(prices, 20, 2.0)
	macd, signal, hist := calculateMACD(prices, 12, 26, 9)
	k, d, j := calculateKDJFromPrices(prices, 14)

	return &TechnicalIndicators{
		RSI:        rsi,
		MACD:       macd,
		Signal:     signal,
		Histogram:  hist,
		BBUpper:    bbUpper,
		BBMiddle:   bbMiddle,
		BBLower:    bbLower,
		BBPosition: bbPosition,
		K:          k,
		D:          d,
		J:          j,
	}, nil
}

// 获取币种价格数据
func getSymbolPriceData(db *sql.DB, symbol string) (float64, float64, error) {
	query := `
		SELECT last_price, price_change_percent
		FROM binance_24h_stats
		WHERE symbol = ? AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		ORDER BY created_at DESC
		LIMIT 1
	`

	var price, priceChange float64
	err := db.QueryRow(query, symbol).Scan(&price, &priceChange)
	return price, priceChange, err
}

// ============================================================================
// 策略推荐算法
// ============================================================================

func generateStrategyRecommendations(env *MarketEnvironment, technicalData []MarketData) []StrategyRecommendation {
	var recommendations []StrategyRecommendation

	// 1. 均值回归策略
	meanReversion := StrategyRecommendation{
		StrategyName:   "均值回归策略",
		Score:          calculateMeanReversionScore(env),
		Confidence:     calculateMeanReversionConfidence(env, technicalData),
		Reason:         getMeanReversionReason(env),
		SuitableMarket: "高震荡市场",
		RiskLevel:      "medium",
		ExpectedReturn: 0.025,
	}
	recommendations = append(recommendations, meanReversion)

	// 2. 趋势跟踪策略
	trendFollowing := StrategyRecommendation{
		StrategyName:   "趋势跟踪策略",
		Score:          calculateTrendFollowingScore(env),
		Confidence:     calculateTrendFollowingConfidence(env, technicalData),
		Reason:         getTrendFollowingReason(env),
		SuitableMarket: "明确趋势市场",
		RiskLevel:      "medium",
		ExpectedReturn: 0.035,
	}
	recommendations = append(recommendations, trendFollowing)

	// 3. 突破策略
	breakout := StrategyRecommendation{
		StrategyName:   "突破策略",
		Score:          calculateBreakoutScore(env),
		Confidence:     calculateBreakoutConfidence(env, technicalData),
		Reason:         getBreakoutReason(env),
		SuitableMarket: "震荡整理市场",
		RiskLevel:      "high",
		ExpectedReturn: 0.045,
	}
	recommendations = append(recommendations, breakout)

	// 4. 网格交易策略
	grid := StrategyRecommendation{
		StrategyName:   "网格交易策略",
		Score:          calculateGridScore(env),
		Confidence:     calculateGridConfidence(env, technicalData),
		Reason:         getGridReason(env),
		SuitableMarket: "横盘震荡市场",
		RiskLevel:      "low",
		ExpectedReturn: 0.015,
	}
	recommendations = append(recommendations, grid)

	// 5. RSI超买超卖策略
	rsi := StrategyRecommendation{
		StrategyName:   "RSI超买超卖策略",
		Score:          calculateRSIScore(env),
		Confidence:     calculateRSIConfidence(env, technicalData),
		Reason:         getRSIReason(env),
		SuitableMarket: "震荡市场",
		RiskLevel:      "medium",
		ExpectedReturn: 0.030,
	}
	recommendations = append(recommendations, rsi)

	// 按分数排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	return recommendations
}

// ============================================================================
// 技术指标计算函数
// ============================================================================

func calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50
	}

	gains := make([]float64, 0, len(prices)-1)
	losses := make([]float64, 0, len(prices)-1)

	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)

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

func calculateBollingerBands(prices []float64, period int, stdDev float64) (float64, float64, float64, float64, float64) {
	if len(prices) < period {
		return 0, 0, 0, 0, 0.5
	}

	middle := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		middle += prices[i]
	}
	middle /= float64(period)

	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += math.Pow(prices[i]-middle, 2)
	}
	std := math.Sqrt(sum / float64(period))

	upper := middle + (std * stdDev)
	lower := middle - (std * stdDev)
	width := (upper - lower) / middle

	currentPrice := prices[len(prices)-1]
	var position float64
	if upper != lower {
		position = (currentPrice - lower) / (upper - lower)
		position = math.Max(0, math.Min(1, position))
	} else {
		position = 0.5
	}

	return middle, upper, lower, width, position
}

func calculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) (float64, float64, float64) {
	if len(prices) < slowPeriod {
		return 0, 0, 0
	}

	fastEMA := calculateEMA(prices, fastPeriod)
	slowEMA := calculateEMA(prices, slowPeriod)
	macd := fastEMA - slowEMA

	macdValues := make([]float64, len(prices)-slowPeriod+1)
	for i := slowPeriod - 1; i < len(prices); i++ {
		fast := calculateEMA(prices[:i+1], fastPeriod)
		slow := calculateEMA(prices[:i+1], slowPeriod)
		macdValues[i-slowPeriod+1] = fast - slow
	}

	signal := calculateEMA(macdValues, signalPeriod)
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

func calculateKDJFromPrices(prices []float64, period int) (float64, float64, float64) {
	if len(prices) < period {
		return 50, 50, 50
	}

	high := prices[len(prices)-period]
	low := prices[len(prices)-period]
	for i := len(prices) - period; i < len(prices); i++ {
		if prices[i] > high {
			high = prices[i]
		}
		if prices[i] < low {
			low = prices[i]
		}
	}

	current := prices[len(prices)-1]
	var k float64
	if high != low {
		k = ((current - low) / (high - low)) * 100
	} else {
		k = 50
	}

	d := k // 简化计算
	j := 3*k - 2*d

	return k, d, j
}

func calculateStandardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))

	sumSquares := 0.0
	for _, v := range values {
		sumSquares += math.Pow(v-mean, 2)
	}

	return math.Sqrt(sumSquares / float64(len(values)))
}

// ============================================================================
// 辅助函数
// ============================================================================

func determineSymbolTrend(priceChange float64) string {
	if priceChange > 5 {
		return "bullish"
	} else if priceChange < -5 {
		return "bearish"
	}
	return "sideways"
}

func determineOverallTrend(bullish, bearish, total int) string {
	if bullish > bearish && bullish > total/3 {
		return "bullish"
	} else if bearish > bullish && bearish > total/3 {
		return "bearish"
	}
	return "sideways"
}

func determineMarketRegime(volatility, oscillation float64, bullish, bearish, total int) (string, float64) {
	score := 0.0

	// 基于波动率
	if volatility > 50 {
		score += 0.3 // 高波动
	} else if volatility > 30 {
		score += 0.2 // 中等波动
	}

	// 基于震荡程度
	if oscillation > 40 {
		score += 0.3 // 高震荡
	} else if oscillation > 25 {
		score += 0.2 // 中等震荡
	}

	// 基于趋势分布
	trendRatio := float64(max(bullish, bearish)) / float64(total)
	if trendRatio > 0.4 {
		score += 0.2 // 有明显趋势
	}

	// 判断市场状态
	var regime string
	var confidence float64

	if score >= 0.6 {
		regime = "trending"
		confidence = 0.8
	} else if score >= 0.4 {
		regime = "mixed"
		confidence = 0.6
	} else {
		regime = "ranging"
		confidence = 0.7
	}

	return regime, confidence
}

// ============================================================================
// 策略评分函数
// ============================================================================

func calculateMeanReversionScore(env *MarketEnvironment) float64 {
	score := 5.0

	// 高震荡市场更适合均值回归
	if env.Oscillation > 40 {
		score += 3
	} else if env.Oscillation > 25 {
		score += 2
	}

	// 中等波动率最合适
	if env.Volatility > 20 && env.Volatility < 60 {
		score += 1
	}

	// RSI中性更适合
	if env.AvgRSI > 30 && env.AvgRSI < 70 {
		score += 1
	}

	return math.Min(score, 10)
}

func calculateMeanReversionConfidence(env *MarketEnvironment, technicalData []MarketData) float64 {
	baseConfidence := 0.5

	// 基于技术指标调整置信度
	rsiSignals := 0
	bbSignals := 0

	for _, data := range technicalData {
		if data.RSI > 30 && data.RSI < 70 {
			rsiSignals++
		}
		if data.BBPosition > 0.2 && data.BBPosition < 0.8 {
			bbSignals++
		}
	}

	if len(technicalData) > 0 {
		rsiRatio := float64(rsiSignals) / float64(len(technicalData))
		bbRatio := float64(bbSignals) / float64(len(technicalData))

		baseConfidence += rsiRatio * 0.2
		baseConfidence += bbRatio * 0.2
	}

	return math.Min(baseConfidence, 0.95)
}

func calculateTrendFollowingScore(env *MarketEnvironment) float64 {
	score := 3.0

	// 明确趋势市场更适合
	if env.MarketRegime == "trending" {
		score += 4
	} else if env.MarketRegime == "mixed" {
		score += 2
	}

	// 高波动率有利于趋势跟踪
	if env.Volatility > 40 {
		score += 2
	} else if env.Volatility > 25 {
		score += 1
	}

	return math.Min(score, 10)
}

func calculateTrendFollowingConfidence(env *MarketEnvironment, technicalData []MarketData) float64 {
	baseConfidence := 0.4

	// 基于MACD信号
	macdSignals := 0
	for _, data := range technicalData {
		if data.MACD > 0 {
			macdSignals++
		}
	}

	if len(technicalData) > 0 {
		macdRatio := float64(macdSignals) / float64(len(technicalData))
		baseConfidence += macdRatio * 0.3
	}

	return math.Min(baseConfidence, 0.9)
}

func calculateBreakoutScore(env *MarketEnvironment) float64 {
	score := 4.0

	// 震荡市场适合突破
	if env.MarketRegime == "ranging" {
		score += 3
	}

	// 布林带位置极端时适合突破
	if env.AvgBBPosition < 0.2 || env.AvgBBPosition > 0.8 {
		score += 2
	}

	return math.Min(score, 10)
}

func calculateBreakoutConfidence(env *MarketEnvironment, technicalData []MarketData) float64 {
	baseConfidence := 0.45

	bbExtremeSignals := 0
	for _, data := range technicalData {
		if data.BBPosition < 0.1 || data.BBPosition > 0.9 {
			bbExtremeSignals++
		}
	}

	if len(technicalData) > 0 {
		extremeRatio := float64(bbExtremeSignals) / float64(len(technicalData))
		baseConfidence += extremeRatio * 0.4
	}

	return math.Min(baseConfidence, 0.95)
}

func calculateGridScore(env *MarketEnvironment) float64 {
	score := 6.0

	// 横盘震荡市场最适合网格
	if env.MarketRegime == "ranging" {
		score += 3
	} else if env.MarketRegime == "mixed" {
		score += 1
	} else {
		score -= 2 // 趋势市场不适合网格
	}

	// 低波动率更适合
	if env.Volatility < 30 {
		score += 1
	}

	return math.Max(score, 1)
}

func calculateGridConfidence(env *MarketEnvironment, technicalData []MarketData) float64 {
	baseConfidence := 0.6

	// 基于震荡程度
	if env.Oscillation < 30 {
		baseConfidence += 0.2
	}

	return math.Min(baseConfidence, 0.9)
}

func calculateRSIScore(env *MarketEnvironment) float64 {
	score := 4.0

	// 震荡市场适合RSI策略
	if env.Oscillation > 30 {
		score += 3
	}

	// RSI极端值多时适合
	rsiExtreme := 0
	if env.AvgRSI < 25 || env.AvgRSI > 75 {
		rsiExtreme++
	}

	for _, data := range []MarketData{} { // 简化处理
		if data.RSI < 25 || data.RSI > 75 {
			rsiExtreme++
		}
	}

	if rsiExtreme > 0 {
		score += 2
	}

	return math.Min(score, 10)
}

func calculateRSIConfidence(env *MarketEnvironment, technicalData []MarketData) float64 {
	baseConfidence := 0.5

	rsiExtremeSignals := 0
	for _, data := range technicalData {
		if data.RSI < 25 || data.RSI > 75 {
			rsiExtremeSignals++
		}
	}

	if len(technicalData) > 0 {
		extremeRatio := float64(rsiExtremeSignals) / float64(len(technicalData))
		baseConfidence += extremeRatio * 0.3
	}

	return math.Min(baseConfidence, 0.9)
}

// ============================================================================
// 原因说明函数
// ============================================================================

func getMeanReversionReason(env *MarketEnvironment) string {
	if env.Oscillation > 40 {
		return "当前市场震荡明显，价格围绕均线波动，均值回归策略最适合捕捉反弹机会"
	} else if env.Volatility > 20 && env.Volatility < 60 {
		return "市场波动适中，均值回归策略可以在价格偏离均线时获利"
	} else {
		return "适合中等波动和震荡环境，当前市场条件相对合适"
	}
}

func getTrendFollowingReason(env *MarketEnvironment) string {
	if env.MarketRegime == "trending" {
		return "市场显示明确趋势，趋势跟踪策略可以顺应市场方向获得较好收益"
	} else {
		return "需要明确的市场趋势，当前市场缺乏方向性，效果可能有限"
	}
}

func getBreakoutReason(env *MarketEnvironment) string {
	if env.MarketRegime == "ranging" {
		return "市场处于震荡整理阶段，突破策略适合在价格突破关键价位时入场"
	} else {
		return "适合震荡市场中的突破机会，当前市场条件一般"
	}
}

func getGridReason(env *MarketEnvironment) string {
	if env.MarketRegime == "ranging" {
		return "市场横盘震荡，网格策略可以在价格区间内多次交易获利"
	} else {
		return "最适合横盘震荡市场，当前市场有一定趋势，收益可能受限"
	}
}

func getRSIReason(env *MarketEnvironment) string {
	if env.Oscillation > 30 {
		return "市场震荡明显，RSI指标可以有效识别超买超卖信号"
	} else {
		return "利用相对强弱指标识别超买超卖区域，适合震荡环境"
	}
}

// ============================================================================
// 显示函数
// ============================================================================

func displayMarketEnvironment(env *MarketEnvironment) {
	fmt.Println("\n📊 市场环境分析结果")
	fmt.Println("───────────────────────────────")
	fmt.Printf("市场状态: %s (置信度: %.1f%%)\n", env.MarketRegime, env.Confidence*100)
	fmt.Printf("整体趋势: %s\n", env.Trend)
	fmt.Printf("平均波动率: %.2f%%\n", env.Volatility)
	fmt.Printf("震荡程度: %.2f%%\n", env.Oscillation)
	fmt.Printf("强势上涨币种: %d\n", env.BullishCount)
	fmt.Printf("强势下跌币种: %d\n", env.BearishCount)
	fmt.Printf("横盘震荡币种: %d\n", env.SidewaysCount)

	if env.AvgRSI > 0 {
		fmt.Printf("平均RSI: %.1f", env.AvgRSI)
		if env.AvgRSI < 30 {
			fmt.Printf(" (超卖)\n")
		} else if env.AvgRSI > 70 {
			fmt.Printf(" (超买)\n")
		} else {
			fmt.Printf(" (中性)\n")
		}
	}

	if env.AvgBBPosition != 0 {
		fmt.Printf("平均布林带位置: %.1f", env.AvgBBPosition)
		if env.AvgBBPosition < 0.2 {
			fmt.Printf(" (下轨附近)\n")
		} else if env.AvgBBPosition > 0.8 {
			fmt.Printf(" (上轨附近)\n")
		} else {
			fmt.Printf(" (中轨附近)\n")
		}
	}
}

func displayTechnicalAnalysis(technicalData []MarketData) {
	if len(technicalData) == 0 {
		fmt.Println("⚠️  没有技术指标数据")
		return
	}

	fmt.Println("\n📈 主要币种技术指标分析")
	fmt.Println("───────────────────────────────")

	for _, data := range technicalData {
		fmt.Printf("\n%s (%.2f%%):\n", data.Symbol, data.PriceChange)
		fmt.Printf("  趋势: %s\n", data.Trend)

		if data.RSI > 0 {
			fmt.Printf("  RSI: %.1f", data.RSI)
			if data.RSI < 30 {
				fmt.Printf(" (超卖🔴)\n")
			} else if data.RSI > 70 {
				fmt.Printf(" (超买🟢)\n")
			} else {
				fmt.Printf(" (中性🟡)\n")
			}
		}

		if data.BBPosition != 0 {
			fmt.Printf("  布林带位置: %.1f", data.BBPosition)
			if data.BBPosition < 0.2 {
				fmt.Printf(" (下轨附近 - 可能反弹)\n")
			} else if data.BBPosition > 0.8 {
				fmt.Printf(" (上轨附近 - 可能回落)\n")
			} else {
				fmt.Printf(" (中轨附近 - 震荡)\n")
			}
		}
	}
}

func displayStrategyRecommendations(recommendations []StrategyRecommendation, env *MarketEnvironment) {
	fmt.Println("\n🎪 策略推荐 (按匹配度排序)")
	fmt.Println("───────────────────────────────")

	for i, rec := range recommendations {
		if i >= 3 { // 只显示前3个
			break
		}

		fmt.Printf("\n%d. %s (评分: %.1f/10, 置信度: %.1f%%)\n",
			i+1, rec.StrategyName, rec.Score, rec.Confidence*100)
		fmt.Printf("   适用市场: %s\n", rec.SuitableMarket)
		fmt.Printf("   风险等级: %s\n", rec.RiskLevel)
		fmt.Printf("   预期收益: %.1f%%\n", rec.ExpectedReturn*100)
		fmt.Printf("   推荐原因: %s\n", rec.Reason)
	}
}

func generateActionPlan(env *MarketEnvironment, recommendations []StrategyRecommendation) {
	if len(recommendations) == 0 {
		fmt.Println("⚠️  没有策略推荐")
		return
	}

	topStrategy := recommendations[0]

	fmt.Printf("🎯 最佳策略: %s\n", topStrategy.StrategyName)
	fmt.Printf("📊 市场状态: %s\n", env.MarketRegime)

	switch env.MarketRegime {
	case "ranging":
		fmt.Println("💡 当前市场震荡为主，建议:")
		fmt.Println("   • 控制仓位，避免过度集中")
		fmt.Println("   • 设置止损止盈，保护本金")
		fmt.Println("   • 关注支撑阻力位")
	case "trending":
		fmt.Println("💡 当前市场有明确趋势，建议:")
		fmt.Println("   • 顺势而为，跟随市场方向")
		fmt.Println("   • 适当加大仓位")
		fmt.Println("   • 滚动止盈，锁定利润")
	case "mixed":
		fmt.Println("💡 当前市场趋势不明确，建议:")
		fmt.Println("   • 谨慎操作，等待更明确信号")
		fmt.Println("   • 分散投资，降低风险")
		fmt.Println("   • 关注大盘走势")
	}

	fmt.Printf("\n⚠️  风险提醒: %s策略风险等级为%s，请根据自身风险承受能力操作\n",
		topStrategy.StrategyName, topStrategy.RiskLevel)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}