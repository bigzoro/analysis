package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"strconv"
	"time"

	"analysis/internal/db"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// MarketDataAnalysisResult 市场数据分析结果
type MarketDataAnalysisResult struct {
	Symbol           string    `json:"symbol"`
	CurrentPrice     float64   `json:"current_price"`
	PriceChange24h   float64   `json:"price_change_24h"`
	Volume24h        float64   `json:"volume_24h"`
	MarketCap        float64   `json:"market_cap"`
	Volatility       float64   `json:"volatility"`
	RSI              float64   `json:"rsi"`
	SMA5             float64   `json:"sma5"`
	SMA10            float64   `json:"sma10"`
	SMA20            float64   `json:"sma20"`
	DataPoints       int       `json:"data_points"`
	LastUpdate       time.Time `json:"last_update"`

	// 均值回归分析
	MeanReversionScore   float64 `json:"mean_reversion_score"`
	ReversionProbability float64 `json:"reversion_probability"`
	OscillationLevel     string  `json:"oscillation_level"`
	TrendDirection       string  `json:"trend_direction"`
	Recommendation       string  `json:"recommendation"`
	Reason               string  `json:"reason"`
}

// MarketAnalysisReport 市场分析报告
type MarketAnalysisReport struct {
	GeneratedAt         time.Time                     `json:"generated_at"`
	TotalSymbols        int                           `json:"total_symbols"`
	AnalyzedSymbols     int                           `json:"analyzed_symbols"`
	EligibleSymbols     int                           `json:"eligible_symbols"`
	MarketConditions    MarketConditions              `json:"market_conditions"`
	TopCandidates       []MarketDataAnalysisResult   `json:"top_candidates"`
	Distribution        SymbolDistribution            `json:"distribution"`
	AnalysisDuration    time.Duration                 `json:"analysis_duration"`
}

// MarketConditions 市场状况
type MarketConditions struct {
	OverallSentiment   string  `json:"overall_sentiment"`
	AverageVolatility  float64 `json:"average_volatility"`
	MarketCapWeightedIndex float64 `json:"market_cap_weighted_index"`
	DominantTrend      string  `json:"dominant_trend"`
	ReversionOpportunities int `json:"reversion_opportunities"`
}

// SymbolDistribution 交易对分布
type SymbolDistribution struct {
	ByMarketCap      map[string]int `json:"by_market_cap"`
	ByVolatility     map[string]int `json:"by_volatility"`
	ByRecommendation map[string]int `json:"by_recommendation"`
}

// MeanReversionAnalyzer 均值回归分析器
type MeanReversionAnalyzer struct {
	db *gorm.DB
}

// NewMeanReversionAnalyzer 创建分析器
func NewMeanReversionAnalyzer(db *gorm.DB) *MeanReversionAnalyzer {
	return &MeanReversionAnalyzer{db: db}
}

// AnalyzeMarketData 分析市场数据
func (a *MeanReversionAnalyzer) AnalyzeMarketData(symbol string) (*MarketDataAnalysisResult, error) {
	result := &MarketDataAnalysisResult{
		Symbol: symbol,
	}

	// 获取24小时统计数据
	var stats db.Binance24hStats
	err := a.db.Where("symbol = ?", symbol).First(&stats).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get 24h stats for %s: %w", symbol, err)
	}

	result.CurrentPrice = stats.LastPrice
	result.PriceChange24h = stats.PriceChangePercent
	result.Volume24h = stats.Volume

	// 获取市值数据
	var marketTop db.BinanceMarketTop
	err = a.db.Where("symbol = ?", symbol).First(&marketTop).Error
	if err == nil && marketTop.MarketCapUSD != nil {
		result.MarketCap = *marketTop.MarketCapUSD
	}

	// 获取K线数据进行技术分析
	var klines []db.MarketKline
	err = a.db.Where("symbol = ? AND interval = ?", symbol, "1h").
		Order("open_time DESC").
		Limit(50). // 获取最近50小时的数据
		Find(&klines).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get klines for %s: %w", symbol, err)
	}

	if len(klines) == 0 {
		return nil, fmt.Errorf("no kline data found for %s", symbol)
	}

	result.DataPoints = len(klines)
	result.LastUpdate = klines[0].OpenTime

	// 反转数据（因为查询是降序的）
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	// 计算技术指标
	prices := make([]float64, len(klines))
	for i, kline := range klines {
		if closePrice, err := a.parsePriceString(kline.ClosePrice); err == nil {
			prices[i] = closePrice
		} else {
			prices[i] = 0
		}
	}

	result.SMA5 = a.calculateSMA(prices, 5)
	result.SMA10 = a.calculateSMA(prices, 10)
	result.SMA20 = a.calculateSMA(prices, 20)
	result.RSI = a.calculateRSI(prices, 14)
	result.Volatility = a.calculateVolatility(prices)

	// 均值回归分析
	result.MeanReversionScore = a.calculateMeanReversionScore(result)
	result.ReversionProbability = a.calculateReversionProbability(result)
	result.OscillationLevel = a.determineOscillationLevel(result.Volatility)
	result.TrendDirection = a.determineTrendDirection(result.SMA5, result.SMA20, result.CurrentPrice)

	// 生成推荐
	result.Recommendation, result.Reason = a.generateRecommendation(result)

	return result, nil
}

// calculateSMA 计算简单移动平均
func (a *MeanReversionAnalyzer) calculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	sum := 0.0
	count := 0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
		count++
	}

	if count == 0 {
		return 0
	}
	return sum / float64(count)
}

// calculateRSI 计算RSI指标
func (a *MeanReversionAnalyzer) calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50.0 // 中性值
	}

	gains := 0.0
	losses := 0.0

	// 计算价格变化
	for i := len(prices) - period; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	return 100.0 - (100.0 / (1.0 + rs))
}

// calculateVolatility 计算波动率
func (a *MeanReversionAnalyzer) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// 计算收益率的标准差
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))

	return math.Sqrt(variance) // 标准差作为波动率度量
}

// calculateMeanReversionScore 计算均值回归得分
func (a *MeanReversionAnalyzer) calculateMeanReversionScore(result *MarketDataAnalysisResult) float64 {
	score := 0.0

	// RSI偏离度评分 (0-100分)
	rsi := result.RSI
	if rsi < 30 {
		// 超卖，均值回归机会高
		score += 100 - rsi*2 // RSI=0时得100分，RSI=30时得40分
	} else if rsi > 70 {
		// 超买，均值回归机会高
		score += (rsi - 70) * 2 // RSI=70时得0分，RSI=100时得60分
	} else {
		// 中性区间，均值回归机会中等
		distanceFromCenter := math.Abs(rsi - 50)
		score += 50 - distanceFromCenter // RSI=50时得50分，RSI=30或70时得30分
	}

	// 波动率评分 (0-30分)
	// 适中的波动率更有利于均值回归
	volatility := result.Volatility
	if volatility > 0.01 && volatility < 0.05 { // 1%-5%的波动率最合适
		score += 30
	} else if volatility > 0.005 && volatility < 0.08 {
		score += 20
	} else if volatility > 0.002 && volatility < 0.1 {
		score += 10
	}

	// 价格位置评分 (0-20分)
	// 价格相对均线的偏离度
	if result.SMA20 > 0 {
		deviation := math.Abs(result.CurrentPrice-result.SMA20) / result.SMA20
		if deviation < 0.02 { // 偏离2%以内
			score += 20
		} else if deviation < 0.05 { // 偏离5%以内
			score += 15
		} else if deviation < 0.1 { // 偏离10%以内
			score += 10
		} else {
			score += 5
		}
	}

	return math.Min(score, 150) // 最高150分
}

// calculateReversionProbability 计算均值回归概率
func (a *MeanReversionAnalyzer) calculateReversionProbability(result *MarketDataAnalysisResult) float64 {
	// 基于历史数据的简单概率估算
	probability := 0.5 // 基准概率50%

	// RSI因素
	if result.RSI < 30 || result.RSI > 70 {
		probability += 0.2 // 极端RSI增加20%的概率
	}

	// 波动率因素
	if result.Volatility > 0.02 && result.Volatility < 0.06 {
		probability += 0.1 // 适中波动率增加10%的概率
	}

	// 价格偏离因素
	if result.SMA20 > 0 {
		deviation := math.Abs(result.CurrentPrice-result.SMA20) / result.SMA20
		if deviation > 0.03 {
			probability += 0.1 // 较大偏离增加10%的概率
		}
	}

	return math.Min(probability, 0.9) // 最高90%的概率
}

// parsePriceString 将字符串价格转换为float64
func (a *MeanReversionAnalyzer) parsePriceString(priceStr string) (float64, error) {
	return strconv.ParseFloat(priceStr, 64)
}

// determineOscillationLevel 确定震荡水平
func (a *MeanReversionAnalyzer) determineOscillationLevel(volatility float64) string {
	if volatility > 0.06 {
		return "high"
	} else if volatility > 0.03 {
		return "medium"
	} else {
		return "low"
	}
}

// determineTrendDirection 确定趋势方向
func (a *MeanReversionAnalyzer) determineTrendDirection(sma5, sma20, currentPrice float64) string {
	if sma5 > sma20 && currentPrice > sma5 {
		return "uptrend"
	} else if sma5 < sma20 && currentPrice < sma5 {
		return "downtrend"
	} else {
		return "sideways"
	}
}

// generateRecommendation 生成推荐
func (a *MeanReversionAnalyzer) generateRecommendation(result *MarketDataAnalysisResult) (string, string) {
	score := result.MeanReversionScore

	if score > 120 {
		return "strong_buy", "均值回归机会极佳，建议积极建仓"
	} else if score > 90 {
		return "buy", "均值回归机会良好，可适量建仓"
	} else if score > 60 {
		return "weak_buy", "均值回归机会一般，谨慎建仓"
	} else if score > 30 {
		return "hold", "等待更好的均值回归机会"
	} else {
		return "avoid", "当前不适合均值回归策略"
	}
}

// RunAnalysis 执行完整分析
func (a *MeanReversionAnalyzer) RunAnalysis() (*MarketAnalysisReport, error) {
	startTime := time.Now()

	report := &MarketAnalysisReport{
		GeneratedAt:      startTime,
		TopCandidates:    []MarketDataAnalysisResult{},
		Distribution:     SymbolDistribution{
			ByMarketCap:      make(map[string]int),
			ByVolatility:     make(map[string]int),
			ByRecommendation: make(map[string]int),
		},
	}

	// 获取要分析的交易对
	var symbols []string
	err := a.db.Model(&db.BinanceMarketTop{}).
		Where("symbol LIKE ?", "%USDT").
		Where("market_cap > ?", 50000000). // 市值大于5000万美金
		Order("market_cap DESC").
		Limit(50). // 分析前50个交易对
		Pluck("symbol", &symbols).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get symbols: %w", err)
	}

	report.TotalSymbols = len(symbols)
	log.Printf("开始分析 %d 个交易对...", len(symbols))

	// 分析每个交易对
	analyzedCount := 0
	eligibleCount := 0
	totalVolatility := 0.0
	totalMarketCap := 0.0
	marketCapWeightedIndex := 0.0

	for _, symbol := range symbols {
		result, err := a.AnalyzeMarketData(symbol)
		if err != nil {
			log.Printf("Warning: Failed to analyze %s: %v", symbol, err)
			continue
		}

		analyzedCount++

		// 统计分布
		a.updateDistribution(&report.Distribution, result)

		// 计算加权指数
		if result.MarketCap > 0 {
			weight := result.MarketCap
			totalMarketCap += weight
			marketCapWeightedIndex += result.MeanReversionScore * weight
			totalVolatility += result.Volatility * weight
		}

		// 筛选优质候选
		if result.MeanReversionScore > 70 { // 分数大于70的认为是优质候选
			report.TopCandidates = append(report.TopCandidates, *result)
			eligibleCount++
		}
	}

	report.AnalyzedSymbols = analyzedCount
	report.EligibleSymbols = eligibleCount

	// 计算市场状况
	if totalMarketCap > 0 {
		report.MarketConditions.MarketCapWeightedIndex = marketCapWeightedIndex / totalMarketCap
		report.MarketConditions.AverageVolatility = totalVolatility / totalMarketCap
	}

	// 确定整体市场情绪
	avgScore := report.MarketConditions.MarketCapWeightedIndex
	if avgScore > 100 {
		report.MarketConditions.OverallSentiment = "bullish_reversion"
	} else if avgScore > 70 {
		report.MarketConditions.OverallSentiment = "neutral_reversion"
	} else {
		report.MarketConditions.OverallSentiment = "bearish_reversion"
	}

	// 确定主导趋势
	uptrend := 0
	downtrend := 0
	sideways := 0
	for _, candidate := range report.TopCandidates {
		switch candidate.TrendDirection {
		case "uptrend":
			uptrend++
		case "downtrend":
			downtrend++
		default:
			sideways++
		}
	}

	if uptrend > downtrend && uptrend > sideways {
		report.MarketConditions.DominantTrend = "uptrend"
	} else if downtrend > uptrend && downtrend > sideways {
		report.MarketConditions.DominantTrend = "downtrend"
	} else {
		report.MarketConditions.DominantTrend = "sideways"
	}

	report.MarketConditions.ReversionOpportunities = eligibleCount

	// 按得分排序候选
	sort.Slice(report.TopCandidates, func(i, j int) bool {
		return report.TopCandidates[i].MeanReversionScore > report.TopCandidates[j].MeanReversionScore
	})

	// 限制候选数量
	if len(report.TopCandidates) > 20 {
		report.TopCandidates = report.TopCandidates[:20]
	}

	report.AnalysisDuration = time.Since(startTime)

	return report, nil
}

// updateDistribution 更新分布统计
func (a *MeanReversionAnalyzer) updateDistribution(dist *SymbolDistribution, result *MarketDataAnalysisResult) {
	// 按市值分布
	if result.MarketCap > 10000000000 { // > 1000亿
		dist.ByMarketCap["large"]++
	} else if result.MarketCap > 1000000000 { // > 100亿
		dist.ByMarketCap["medium"]++
	} else {
		dist.ByMarketCap["small"]++
	}

	// 按波动率分布
	switch result.OscillationLevel {
	case "high":
		dist.ByVolatility["high"]++
	case "medium":
		dist.ByVolatility["medium"]++
	default:
		dist.ByVolatility["low"]++
	}

	// 按推荐分布
	dist.ByRecommendation[result.Recommendation]++
}

func main() {
	// 从环境变量获取数据库配置
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "user:password@tcp(localhost:3306)/trading_analysis?charset=utf8mb4&parseTime=True&loc=Local"
	}

	log.Printf("连接数据库...")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	log.Printf("数据库连接成功")

	// 创建分析器
	analyzer := NewMeanReversionAnalyzer(db)

	// 执行分析
	log.Printf("开始均值回归市场分析...")
	report, err := analyzer.RunAnalysis()
	if err != nil {
		log.Fatalf("分析失败: %v", err)
	}

	// 输出结果
	log.Printf("=== 均值回归策略市场分析报告 ===")
	log.Printf("分析时间: %v", report.GeneratedAt.Format("2006-01-02 15:04:05"))
	log.Printf("分析用时: %v", report.AnalysisDuration)
	log.Printf("总交易对数: %d", report.TotalSymbols)
	log.Printf("成功分析数: %d", report.AnalyzedSymbols)
	log.Printf("符合条件数: %d", report.EligibleSymbols)

	log.Printf("\n=== 市场状况分析 ===")
	log.Printf("整体情绪: %s", report.MarketConditions.OverallSentiment)
	log.Printf("主导趋势: %s", report.MarketConditions.DominantTrend)
	log.Printf("平均波动率: %.2f%%", report.MarketConditions.AverageVolatility*100)
	log.Printf("市值加权指数: %.1f", report.MarketConditions.MarketCapWeightedIndex)
	log.Printf("均值回归机会: %d", report.MarketConditions.ReversionOpportunities)

	log.Printf("\n=== 市值分布 ===")
	for category, count := range report.Distribution.ByMarketCap {
		log.Printf("%s: %d", category, count)
	}

	log.Printf("\n=== 波动率分布 ===")
	for level, count := range report.Distribution.ByVolatility {
		log.Printf("%s: %d", level, count)
	}

	log.Printf("\n=== 推荐分布 ===")
	for recommendation, count := range report.Distribution.ByRecommendation {
		log.Printf("%s: %d", recommendation, count)
	}

	log.Printf("\n=== 顶级候选交易对 (前10个) ===")
	for i, candidate := range report.TopCandidates {
		if i >= 10 {
			break
		}
		log.Printf("%d. %s - 得分: %.1f, 推荐: %s",
			i+1, candidate.Symbol, candidate.MeanReversionScore, candidate.Recommendation)
		log.Printf("   价格: $%.4f (24h: %.2f%%), 市值: $%.0f",
			candidate.CurrentPrice, candidate.PriceChange24h, candidate.MarketCap)
		log.Printf("   RSI: %.1f, 波动率: %.2f%%, 趋势: %s",
			candidate.RSI, candidate.Volatility*100, candidate.TrendDirection)
		log.Printf("   均值回归概率: %.1f%%, 原因: %s",
			candidate.ReversionProbability*100, candidate.Reason)
		log.Printf("")
	}

	// 保存详细报告到JSON文件
	jsonData, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Printf("JSON序列化失败: %v", err)
	} else {
		filename := fmt.Sprintf("mean_reversion_analysis_%s.json",
			report.GeneratedAt.Format("20060102_150405"))
		err = os.WriteFile(filename, jsonData, 0644)
		if err != nil {
			log.Printf("保存报告失败: %v", err)
		} else {
			log.Printf("详细报告已保存到: %s", filename)
		}
	}

	log.Printf("\n=== 分析总结 ===")
	log.Printf("在当前市场环境下，发现了 %d 个具有均值回归机会的交易对", report.EligibleSymbols)
	log.Printf("整体市场%s均值回归机会，%s趋势主导，平均波动率为%.2f%%",
		report.MarketConditions.OverallSentiment,
		report.MarketConditions.DominantTrend,
		report.MarketConditions.AverageVolatility*100)

	if len(report.TopCandidates) > 0 {
		topCandidate := report.TopCandidates[0]
		log.Printf("最佳机会: %s (得分: %.1f), 建议: %s",
			topCandidate.Symbol, topCandidate.MeanReversionScore, topCandidate.Recommendation)
	}
}