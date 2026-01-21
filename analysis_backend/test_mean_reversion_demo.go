package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"sort"
	"time"
)

// MockMarketData 模拟市场数据结构
type MockMarketData struct {
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
	GeneratedAt         time.Time                `json:"generated_at"`
	TotalSymbols        int                      `json:"total_symbols"`
	AnalyzedSymbols     int                      `json:"analyzed_symbols"`
	EligibleSymbols     int                      `json:"eligible_symbols"`
	MarketConditions    MarketConditions         `json:"market_conditions"`
	TopCandidates       []MockMarketData         `json:"top_candidates"`
	Distribution        SymbolDistribution       `json:"distribution"`
	AnalysisDuration    time.Duration            `json:"analysis_duration"`
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
	mockData map[string][]float64 // 模拟的历史价格数据
}

// NewMeanReversionAnalyzer 创建分析器
func NewMeanReversionAnalyzer() *MeanReversionAnalyzer {
	return &MeanReversionAnalyzer{
		mockData: generateMockMarketData(),
	}
}

// generateMockMarketData 生成模拟的市场数据
func generateMockMarketData() map[string][]float64 {
	data := make(map[string][]float64)

	// 生成不同币种的模拟价格数据
	symbols := []string{
		"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT",
		"DOTUSDT", "AVAXUSDT", "LTCUSDT", "ETCUSDT", "LINKUSDT",
		"XRPUSDT", "DOGEUSDT", "MATICUSDT", "UNIUSDT", "ALGOUSDT",
	}

	for _, symbol := range symbols {
		prices := generatePriceSeries(symbol, 50) // 50个数据点
		data[symbol] = prices
	}

	return data
}

// generatePriceSeries 生成价格序列
func generatePriceSeries(symbol string, length int) []float64 {
	// 根据币种设置基准价格
	basePrice := 50000.0
	switch symbol {
	case "BTCUSDT":
		basePrice = 45000.0
	case "ETHUSDT":
		basePrice = 2800.0
	case "BNBUSDT":
		basePrice = 320.0
	case "ADAUSDT":
		basePrice = 0.45
	case "SOLUSDT":
		basePrice = 95.0
	case "DOTUSDT":
		basePrice = 6.8
	case "AVAXUSDT":
		basePrice = 28.0
	case "LTCUSDT":
		basePrice = 75.0
	case "ETCUSDT":
		basePrice = 18.5
	case "LINKUSDT":
		basePrice = 13.2
	case "XRPUSDT":
		basePrice = 0.52
	case "DOGEUSDT":
		basePrice = 0.085
	case "MATICUSDT":
		basePrice = 0.85
	case "UNIUSDT":
		basePrice = 6.2
	case "ALGOUSDT":
		basePrice = 0.15
	}

	prices := make([]float64, length)
	currentPrice := basePrice

	// 生成价格序列，包含一些趋势和震荡
	trend := 0.001 // 轻微上升趋势
	volatility := 0.02 // 2%的波动率

	for i := 0; i < length; i++ {
		// 添加趋势
		trendChange := currentPrice * trend

		// 添加随机波动
		randomChange := currentPrice * volatility * (0.5 - float64(i%10)/10.0)

		// 对于某些币种添加特殊模式
		if symbol == "BTCUSDT" && i > 30 {
			// BTC在后期震荡
			randomChange *= 0.5
		} else if symbol == "ETHUSDT" && i < 20 {
			// ETH前期强势
			randomChange *= 1.5
		}

		currentPrice += trendChange + randomChange

		// 确保价格不为负
		if currentPrice < 0.0001 {
			currentPrice = 0.0001
		}

		prices[i] = currentPrice
	}

	return prices
}

// AnalyzeMarketData 分析市场数据
func (a *MeanReversionAnalyzer) AnalyzeMarketData(symbol string) (*MockMarketData, error) {
	prices, exists := a.mockData[symbol]
	if !exists {
		return nil, fmt.Errorf("no data found for symbol %s", symbol)
	}

	result := &MockMarketData{
		Symbol:     symbol,
		DataPoints: len(prices),
		LastUpdate: time.Now(),
	}

	// 当前价格是最新价格
	result.CurrentPrice = prices[len(prices)-1]

	// 计算24小时涨跌幅（模拟）
	if len(prices) >= 24 {
		oldPrice := prices[len(prices)-24]
		result.PriceChange24h = (result.CurrentPrice - oldPrice) / oldPrice * 100
	}

	// 计算技术指标
	result.SMA5 = a.calculateSMA(prices, 5)
	result.SMA10 = a.calculateSMA(prices, 10)
	result.SMA20 = a.calculateSMA(prices, 20)
	result.RSI = a.calculateRSI(prices, 14)
	result.Volatility = a.calculateVolatility(prices)

	// 模拟市值数据（基于价格和模拟的流通量）
	circulatingSupply := 1000000.0 // 模拟流通量
	switch symbol {
	case "BTCUSDT":
		circulatingSupply = 19500000
	case "ETHUSDT":
		circulatingSupply = 120000000
	case "BNBUSDT":
		circulatingSupply = 165000000
	case "ADAUSDT":
		circulatingSupply = 35000000000
	case "SOLUSDT":
		circulatingSupply = 500000000
	}
	result.MarketCap = result.CurrentPrice * circulatingSupply
	result.Volume24h = result.MarketCap * 0.01 // 假设成交量是市值的1%

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
func (a *MeanReversionAnalyzer) calculateMeanReversionScore(result *MockMarketData) float64 {
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
func (a *MeanReversionAnalyzer) calculateReversionProbability(result *MockMarketData) float64 {
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
func (a *MeanReversionAnalyzer) generateRecommendation(result *MockMarketData) (string, string) {
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
		TopCandidates:    []MockMarketData{},
		Distribution:     SymbolDistribution{
			ByMarketCap:      make(map[string]int),
			ByVolatility:     make(map[string]int),
			ByRecommendation: make(map[string]int),
		},
	}

	symbols := make([]string, 0, len(a.mockData))
	for symbol := range a.mockData {
		symbols = append(symbols, symbol)
	}

	report.TotalSymbols = len(symbols)
	log.Printf("开始分析 %d 个交易对的均值回归机会...", len(symbols))

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
func (a *MeanReversionAnalyzer) updateDistribution(dist *SymbolDistribution, result *MockMarketData) {
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
	log.Printf("开始均值回归策略市场分析（模拟数据演示）...")

	// 创建分析器
	analyzer := NewMeanReversionAnalyzer()

	// 执行分析
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
	log.Printf("符合条件数: %d (%.1f%%)", report.EligibleSymbols,
		float64(report.EligibleSymbols)/float64(report.AnalyzedSymbols)*100)

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
		filename := fmt.Sprintf("mean_reversion_demo_analysis_%s.json",
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
		log.Printf("最佳机会: %s (得分: %.1f), 推荐: %s",
			topCandidate.Symbol, topCandidate.MeanReversionScore, topCandidate.Recommendation)
		log.Printf("该币种当前价格为$%.4f，24小时涨跌幅为%.2f%%，RSI为%.1f，波动率为%.2f%%",
			topCandidate.CurrentPrice, topCandidate.PriceChange24h,
			topCandidate.RSI, topCandidate.Volatility*100)
		log.Printf("基于技术分析，该币种具有%.1f%%的均值回归概率，建议%s",
			topCandidate.ReversionProbability*100, topCandidate.Reason)
	}

	log.Printf("\n=== 市场环境解读 ===")
	log.Printf("当前市场环境适合均值回归策略的币种主要集中在市值%s的品种上",
		report.MarketConditions.OverallSentiment)
	log.Printf("波动率水平为%.1f%%，表明市场%s震荡，适合均值回归策略",
		report.MarketConditions.AverageVolatility*100,
		func() string {
			if report.MarketConditions.AverageVolatility > 0.05 {
				return "较为剧烈"
			} else if report.MarketConditions.AverageVolatility > 0.03 {
				return "适度"
			} else {
				return "相对平稳"
			}
		}())

	if report.MarketConditions.DominantTrend == "sideways" {
		log.Printf("当前市场以横盘震荡为主，均值回归策略效果会更好")
	} else {
		log.Printf("当前市场存在%s趋势，均值回归策略需要谨慎使用", report.MarketConditions.DominantTrend)
	}
}