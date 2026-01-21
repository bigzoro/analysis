package server

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"strconv"
	"strings"

	pdb "analysis/internal/db"
)

// CoinCategory 币种类别
type CoinCategory string

const (
	CategoryDeFi     CoinCategory = "defi"
	CategoryLayer1   CoinCategory = "layer1"
	CategoryLayer2   CoinCategory = "layer2"
	CategoryMeme     CoinCategory = "meme"
	CategoryNFT      CoinCategory = "nft"
	CategoryGaming   CoinCategory = "gaming"
	CategoryStable   CoinCategory = "stable"
	CategoryExchange CoinCategory = "exchange"
	CategoryOther    CoinCategory = "other"
)

// getCoinCategory 获取币种类别（简化版，基于常见币种）
func getCoinCategory(symbol string) CoinCategory {
	symbol = strings.ToUpper(symbol)

	// Layer1
	layer1 := []string{"BTC", "ETH", "SOL", "BNB", "AVAX", "ATOM", "DOT", "ADA", "ALGO", "NEAR", "FTM", "EGLD"}
	for _, l1 := range layer1 {
		if symbol == l1 {
			return CategoryLayer1
		}
	}

	// Layer2
	layer2 := []string{"ARB", "OP", "MATIC", "IMX", "METIS", "BOBA", "LRC"}
	for _, l2 := range layer2 {
		if symbol == l2 {
			return CategoryLayer2
		}
	}

	// DeFi
	defi := []string{"UNI", "AAVE", "COMP", "MKR", "SNX", "CRV", "SUSHI", "1INCH", "CAKE", "PancakeSwap", "GMX", "GNS"}
	for _, d := range defi {
		if strings.Contains(symbol, d) {
			return CategoryDeFi
		}
	}

	// Meme
	meme := []string{"DOGE", "SHIB", "PEPE", "FLOKI", "BONK", "WIF", "BOME"}
	for _, m := range meme {
		if symbol == m {
			return CategoryMeme
		}
	}

	// Exchange
	exchange := []string{"BNB", "FTT", "HT", "OKB", "KCS"}
	for _, e := range exchange {
		if symbol == e {
			return CategoryExchange
		}
	}

	// Stable
	stable := []string{"USDT", "USDC", "BUSD", "DAI", "TUSD", "USDP", "FDUSD"}
	for _, s := range stable {
		if symbol == s {
			return CategoryStable
		}
	}

	// Gaming
	gaming := []string{"AXS", "SAND", "MANA", "ENJ", "GALA", "IMX"}
	for _, g := range gaming {
		if symbol == g {
			return CategoryGaming
		}
	}

	return CategoryOther
}

// RecommendationConfidence 推荐置信度
type RecommendationConfidence struct {
	TotalConfidence    float64  `json:"total_confidence"`    // 总置信度 0-100
	DataCompleteness   float64  `json:"data_completeness"`   // 数据完整性 0-100
	SignalStrength     float64  `json:"signal_strength"`     // 信号强度 0-100
	HistoricalAccuracy float64  `json:"historical_accuracy"` // 历史准确率 0-100
	MarketStability    float64  `json:"market_stability"`    // 市场稳定性 0-100
	Reasons            []string `json:"reasons"`             // 置信度说明
}

// calculateRecommendationConfidence 计算推荐置信度
func (s *Server) calculateRecommendationConfidence(
	score RecommendationScore,
	item pdb.BinanceMarketTop,
	historicalAccuracy float64, // 该币种的历史推荐准确率
) RecommendationConfidence {
	conf := RecommendationConfidence{
		Reasons: make([]string, 0),
	}

	// 1. 数据完整性（30%）
	dataCompleteness := 100.0
	missingData := 0

	// 检查数据完整性
	if score.MarketCap == 0 {
		dataCompleteness -= 10
		missingData++
	}
	if score.Volume24h == 0 {
		dataCompleteness -= 5
		missingData++
	}
	if score.Scores.Technical == 0 {
		dataCompleteness -= 15
		missingData++
	}

	conf.DataCompleteness = math.Max(0, dataCompleteness)
	if missingData > 0 {
		conf.Reasons = append(conf.Reasons, fmt.Sprintf("缺少%d项数据源", missingData))
	}

	// 2. 信号强度（30%）
	signalStrength := 0.0

	// 涨幅信号强度
	priceChange := math.Abs(score.PriceChange24h)
	if priceChange > 10 {
		signalStrength += 20
	} else if priceChange > 5 {
		signalStrength += 15
	} else {
		signalStrength += 10
	}

	// 技术指标信号强度
	if score.Scores.Technical > 0.7 {
		signalStrength += 20
	} else if score.Scores.Technical > 0.5 {
		signalStrength += 15
	} else {
		signalStrength += 10
	}

	// 情绪信号强度
	if score.Scores.Sentiment > 0.7 {
		signalStrength += 15
	} else if score.Scores.Sentiment > 0.5 {
		signalStrength += 10
	} else {
		signalStrength += 5
	}

	conf.SignalStrength = math.Min(100, signalStrength)

	// 3. 历史准确率（20%）
	conf.HistoricalAccuracy = historicalAccuracy
	if historicalAccuracy > 0 {
		if historicalAccuracy > 70 {
			conf.Reasons = append(conf.Reasons, fmt.Sprintf("历史推荐准确率%.1f%%，表现优秀", historicalAccuracy))
		} else if historicalAccuracy < 50 {
			conf.Reasons = append(conf.Reasons, fmt.Sprintf("历史推荐准确率%.1f%%，需谨慎", historicalAccuracy))
		}
	} else {
		conf.HistoricalAccuracy = 50 // 默认值（无历史数据）
		conf.Reasons = append(conf.Reasons, "暂无历史数据")
	}

	// 4. 市场稳定性（20%）
	marketStability := 100.0

	// 基于风险评分调整稳定性
	if score.Scores.Risk > 0.7 {
		marketStability -= 30
		conf.Reasons = append(conf.Reasons, "风险评分较高，市场不稳定")
	} else if score.Scores.Risk > 0.5 {
		marketStability -= 15
	}

	// 市值稳定性
	if score.MarketCap < 10000000 {
		marketStability -= 20
		conf.Reasons = append(conf.Reasons, "市值较小，稳定性较低")
	}

	conf.MarketStability = math.Max(0, marketStability)

	// 计算总置信度
	conf.TotalConfidence = (conf.DataCompleteness*0.3 + conf.SignalStrength*0.3 +
		conf.HistoricalAccuracy*0.2 + conf.MarketStability*0.2)

	return conf
}

// ensureRecommendationDiversity 确保推荐多样性（按类别分组）
func ensureRecommendationDiversity(scores []RecommendationScore, limit int) []RecommendationScore {
	if len(scores) <= limit {
		return scores
	}

	// 按类别分组
	categoryMap := make(map[CoinCategory][]RecommendationScore)
	for _, score := range scores {
		category := getCoinCategory(score.BaseSymbol)
		categoryMap[category] = append(categoryMap[category], score)
	}

	// 每个类别最多选择的数量
	maxPerCategory := limit / len(categoryMap)
	if maxPerCategory < 1 {
		maxPerCategory = 1
	}

	// 从每个类别中选择
	diverseScores := make([]RecommendationScore, 0, limit)
	used := make(map[string]bool)

	// 按类别优先级排序（Layer1 > DeFi > Layer2 > 其他）
	categoryPriority := []CoinCategory{
		CategoryLayer1, CategoryDeFi, CategoryLayer2,
		CategoryGaming, CategoryMeme, CategoryExchange, CategoryOther,
	}

	for _, category := range categoryPriority {
		if candidates, ok := categoryMap[category]; ok {
			sort.Slice(candidates, func(i, j int) bool {
				return candidates[i].TotalScore > candidates[j].TotalScore
			})

			count := 0
			for _, candidate := range candidates {
				if count >= maxPerCategory {
					break
				}
				if !used[candidate.Symbol] {
					diverseScores = append(diverseScores, candidate)
					used[candidate.Symbol] = true
					count++
				}
			}
		}
	}

	// 如果多样性选择后数量不足，从剩余中补充
	if len(diverseScores) < limit {
		for _, score := range scores {
			if len(diverseScores) >= limit {
				break
			}
			if !used[score.Symbol] {
				diverseScores = append(diverseScores, score)
				used[score.Symbol] = true
			}
		}
	}

	// 按总分重新排序
	sort.Slice(diverseScores, func(i, j int) bool {
		return diverseScores[i].TotalScore > diverseScores[j].TotalScore
	})

	log.Printf("[Diversity] 从 %d 个候选中选择 %d 个多样性推荐（覆盖 %d 个类别）",
		len(scores), len(diverseScores), len(categoryMap))

	return diverseScores
}

// detectAnomalies 检测异常情况
func (s *Server) detectAnomalies(
	item pdb.BinanceMarketTop,
	score RecommendationScore,
) (hasAnomaly bool, anomalies []string) {
	anomalies = make([]string, 0)

	// 临时简化实现：基于当前 RecommendationScore 结构
	// 1. 价格异常检测
	priceChange24h := math.Abs(score.PriceChange24h)
	if priceChange24h > 100 {
		hasAnomaly = true
		anomalies = append(anomalies, fmt.Sprintf("24h涨幅异常（%.2f%%），可能存在价格操纵", score.PriceChange24h))
	}

	// 2. 成交量异常检测
	if score.Volume24h > 0 && score.MarketCap > 0 {
		turnoverRate := score.Volume24h / score.MarketCap
		if turnoverRate > 5 {
			hasAnomaly = true
			anomalies = append(anomalies, fmt.Sprintf("换手率异常（%.2f倍），可能存在异常交易", turnoverRate))
		}
	}

	// 3. 市值异常检测
	if score.MarketCap < 10000 {
		hasAnomaly = true
		anomalies = append(anomalies, "市值过小，风险极高")
	}

	// 4. 技术指标异常检测
	if score.Scores.Technical > 0.9 || score.Scores.Technical < 0.1 {
		hasAnomaly = true
		anomalies = append(anomalies, fmt.Sprintf("技术指标异常（%.2f），市场可能异常", score.Scores.Technical))
	}

	// 5. 排名异常
	if item.Rank > 200 {
		hasAnomaly = true
		anomalies = append(anomalies, fmt.Sprintf("排名靠后（%d），市场认可度低", item.Rank))
	}

	return hasAnomaly, anomalies
}

// calculateMultiTimeframeScore 计算多时间维度得分
func (s *Server) calculateMultiTimeframeScore(
	ctx context.Context,
	symbol string,
	kind string,
	currentPrice float64,
) (map[string]float64, error) {
	scores := make(map[string]float64)

	// 获取K线数据
	klines1h, err := s.fetchBinanceKlines(ctx, symbol, kind, "1h", 1)
	if err == nil && len(klines1h) > 0 {
		price1h, _ := strconv.ParseFloat(klines1h[0].Close, 64)
		if price1h > 0 {
			change1h := ((currentPrice - price1h) / price1h) * 100
			scores["1h"] = change1h
		}
	}

	klines4h, err := s.fetchBinanceKlines(ctx, symbol, kind, "4h", 1)
	if err == nil && len(klines4h) > 0 {
		price4h, _ := strconv.ParseFloat(klines4h[0].Close, 64)
		if price4h > 0 {
			change4h := ((currentPrice - price4h) / price4h) * 100
			scores["4h"] = change4h
		}
	}

	// 24h数据（已有）
	scores["24h"] = 0 // 将在外部设置

	// 7天和30天数据（需要获取更多K线）
	klines7d, err := s.fetchBinanceKlines(ctx, symbol, kind, "1d", 7)
	if err == nil && len(klines7d) >= 7 {
		price7d, _ := strconv.ParseFloat(klines7d[0].Close, 64)
		if price7d > 0 {
			change7d := ((currentPrice - price7d) / price7d) * 100
			scores["7d"] = change7d
		}
	}

	klines30d, err := s.fetchBinanceKlines(ctx, symbol, kind, "1d", 30)
	if err == nil && len(klines30d) >= 30 {
		price30d, _ := strconv.ParseFloat(klines30d[0].Close, 64)
		if price30d > 0 {
			change30d := ((currentPrice - price30d) / price30d) * 100
			scores["30d"] = change30d
		}
	}

	return scores, nil
}

// calculateWeightedMultiTimeframeScore 计算加权多时间维度得分
func calculateWeightedMultiTimeframeScore(timeframeScores map[string]float64) float64 {
	weights := map[string]float64{
		"1h":  0.1,
		"4h":  0.2,
		"24h": 0.4,
		"7d":  0.2,
		"30d": 0.1,
	}

	weightedSum := 0.0
	totalWeight := 0.0

	for timeframe, weight := range weights {
		if score, ok := timeframeScores[timeframe]; ok {
			weightedSum += score * weight
			totalWeight += weight
		}
	}

	if totalWeight > 0 {
		return weightedSum / totalWeight
	}

	// 如果没有多时间数据，返回24h数据
	if score24h, ok := timeframeScores["24h"]; ok {
		return score24h
	}

	return 0
}

// getHistoricalAccuracy 获取币种的历史推荐准确率
func (s *Server) getHistoricalAccuracy(ctx context.Context, symbol string) float64 {
	// 查询该币种的历史推荐表现
	perfs, err := pdb.GetPerformanceBySymbol(s.db.DB(), symbol, 10)
	if err != nil || len(perfs) == 0 {
		return 0 // 无历史数据
	}

	// 计算24h胜率
	winCount := 0
	totalCount := 0

	for _, perf := range perfs {
		if perf.Return24h != nil {
			totalCount++
			if *perf.Return24h > 0 {
				winCount++
			}
		}
	}

	if totalCount == 0 {
		return 0
	}

	accuracy := float64(winCount) / float64(totalCount) * 100
	return accuracy
}

// DynamicWeightLearner 动态权重学习器
type DynamicWeightLearner struct {
	// 历史数据窗口（天数）
	LearningWindowDays int

	// 因子表现跟踪
	FactorPerformance map[string]FactorMetrics

	// 市场环境感知器
	MarketConditionDetector *MarketConditionDetector

	// 学习算法参数
	LearningRate float64
	Momentum     float64

	// 权重约束
	MinWeight float64
	MaxWeight float64
}

// FactorMetrics 因子表现指标
type FactorMetrics struct {
	// 基础指标
	AvgReturn float64 // 平均收益率
	WinRate   float64 // 胜率
	Count     int     // 样本数量

	// 风险指标
	Volatility  float64 // 波动率
	MaxDrawdown float64 // 最大回撤

	// 时间序列指标
	TrendStrength float64 // 趋势强度
	Consistency   float64 // 一致性
}

// MarketConditionDetector 市场环境检测器
type MarketConditionDetector struct {
	// 市场波动率
	MarketVolatility float64

	// 市场趋势
	MarketTrend string // "bull", "bear", "sideways"

	// 市场情绪
	MarketSentiment string // "optimistic", "pessimistic", "neutral"
}

// NewDynamicWeightLearner 创建动态权重学习器
func NewDynamicWeightLearner() *DynamicWeightLearner {
	return &DynamicWeightLearner{
		LearningWindowDays:      30,
		FactorPerformance:       make(map[string]FactorMetrics),
		MarketConditionDetector: &MarketConditionDetector{},
		LearningRate:            0.1,
		Momentum:                0.9,
		MinWeight:               0.05,
		MaxWeight:               0.5,
	}
}

// LearnFromHistoricalData 从历史数据学习权重
func (dwl *DynamicWeightLearner) LearnFromHistoricalData(ctx context.Context, db interface{}) error {
	// 这里暂时使用模拟数据，实际实现需要连接数据库
	// 模拟因子表现数据
	dwl.FactorPerformance["market"] = FactorMetrics{
		AvgReturn:     5.2,
		WinRate:       0.58,
		Count:         150,
		Volatility:    0.15,
		MaxDrawdown:   -12.3,
		TrendStrength: 0.75,
		Consistency:   0.82,
	}

	dwl.FactorPerformance["flow"] = FactorMetrics{
		AvgReturn:     7.8,
		WinRate:       0.62,
		Count:         145,
		Volatility:    0.18,
		MaxDrawdown:   -15.7,
		TrendStrength: 0.68,
		Consistency:   0.75,
	}

	dwl.FactorPerformance["heat"] = FactorMetrics{
		AvgReturn:     3.1,
		WinRate:       0.52,
		Count:         140,
		Volatility:    0.12,
		MaxDrawdown:   -8.9,
		TrendStrength: 0.45,
		Consistency:   0.91,
	}

	dwl.FactorPerformance["event"] = FactorMetrics{
		AvgReturn:     8.5,
		WinRate:       0.65,
		Count:         85,
		Volatility:    0.22,
		MaxDrawdown:   -18.2,
		TrendStrength: 0.82,
		Consistency:   0.68,
	}

	dwl.FactorPerformance["sentiment"] = FactorMetrics{
		AvgReturn:     4.7,
		WinRate:       0.55,
		Count:         120,
		Volatility:    0.14,
		MaxDrawdown:   -11.5,
		TrendStrength: 0.55,
		Consistency:   0.78,
	}

	return nil
}

// analyzeFactorContributions 分析各因子的贡献
func (dwl *DynamicWeightLearner) analyzeFactorContributions(rec pdb.CoinRecommendation, perf *pdb.RecommendationPerformance) map[string]FactorContribution {
	contributions := make(map[string]FactorContribution)

	// 市场表现因子
	if perf.Return24h != nil {
		contributions["market"] = FactorContribution{
			Return: *perf.Return24h,
			Score:  rec.MarketScore,
		}
	}

	// 资金流因子
	if perf.Return24h != nil {
		contributions["flow"] = FactorContribution{
			Return: *perf.Return24h,
			Score:  rec.FlowScore,
		}
	}

	// 热度因子
	if perf.Return24h != nil {
		contributions["heat"] = FactorContribution{
			Return: *perf.Return24h,
			Score:  rec.HeatScore,
		}
	}

	// 事件因子
	if perf.Return24h != nil {
		contributions["event"] = FactorContribution{
			Return: *perf.Return24h,
			Score:  rec.EventScore,
		}
	}

	// 情绪因子
	if perf.Return24h != nil {
		contributions["sentiment"] = FactorContribution{
			Return: *perf.Return24h,
			Score:  rec.SentimentScore,
		}
	}

	return contributions
}

// FactorContribution 因子贡献
type FactorContribution struct {
	Return float64 // 该因子下的收益率
	Score  float64 // 该因子的评分
}

// RecommendationResult 推荐结果
type RecommendationResult struct {
	Return       float64 // 总收益率
	Win          bool    // 是否盈利
	FactorScore  float64 // 因子评分
	MarketReturn float64 // 市场基准收益
}

// calculateFactorMetrics 计算因子表现指标
func (dwl *DynamicWeightLearner) calculateFactorMetrics(results []RecommendationResult) FactorMetrics {
	if len(results) == 0 {
		return FactorMetrics{}
	}

	metrics := FactorMetrics{
		Count: len(results),
	}

	// 计算平均收益率
	totalReturn := 0.0
	winCount := 0
	returns := make([]float64, len(results))

	for i, result := range results {
		totalReturn += result.Return
		returns[i] = result.Return
		if result.Win {
			winCount++
		}
	}

	metrics.AvgReturn = totalReturn / float64(len(results))
	metrics.WinRate = float64(winCount) / float64(len(results))

	// 计算波动率
	metrics.Volatility = dwl.calculateVolatility(returns)

	// 计算最大回撤
	metrics.MaxDrawdown = dwl.calculateMaxDrawdown(returns)

	// 计算趋势强度（使用线性回归斜率）
	metrics.TrendStrength = dwl.calculateTrendStrength(returns)

	// 计算一致性（收益率标准差的倒数）
	if metrics.Volatility > 0 {
		metrics.Consistency = 1.0 / metrics.Volatility
	}

	return metrics
}

// CalculateAdaptiveWeights 计算自适应权重
func (dwl *DynamicWeightLearner) CalculateAdaptiveWeights() map[string]float64 {
	weights := make(map[string]float64)
	totalScore := 0.0

	// 基于表现指标计算权重
	for factor, metrics := range dwl.FactorPerformance {
		if metrics.Count < 5 {
			// 样本不足，使用默认权重
			weights[factor] = 0.2
			totalScore += 0.2
			continue
		}

		// 综合评分：收益率 × 胜率 × 一致性 × 趋势强度
		score := metrics.AvgReturn * metrics.WinRate * metrics.Consistency * metrics.TrendStrength

		// 归一化处理（避免负数）
		score = math.Max(0, score)

		// 应用市场环境调整
		score = dwl.adjustForMarketConditions(factor, score, metrics)

		weights[factor] = score
		totalScore += score
	}

	// 归一化权重
	if totalScore > 0 {
		for factor := range weights {
			weights[factor] /= totalScore
			// 应用权重约束
			weights[factor] = math.Max(dwl.MinWeight, math.Min(dwl.MaxWeight, weights[factor]))
		}
	}

	// 再次归一化确保总和为1
	totalNormalized := 0.0
	for _, w := range weights {
		totalNormalized += w
	}
	if totalNormalized > 0 {
		for factor := range weights {
			weights[factor] /= totalNormalized
		}
	}

	return weights
}

// adjustForMarketConditions 根据市场环境调整权重
func (dwl *DynamicWeightLearner) adjustForMarketConditions(factor string, baseScore float64, metrics FactorMetrics) float64 {
	market := dwl.MarketConditionDetector

	// 根据市场趋势调整
	switch market.MarketTrend {
	case "bull":
		// 牛市中，资金流和情绪因子更重要
		if factor == "flow" || factor == "sentiment" {
			baseScore *= 1.2
		}
	case "bear":
		// 熊市中，事件和市场表现因子更重要
		if factor == "event" || factor == "market" {
			baseScore *= 1.2
		}
	}

	// 根据市场波动率调整
	if market.MarketVolatility > 0.5 {
		// 高波动环境下，技术指标更可靠
		if factor == "heat" || factor == "sentiment" {
			baseScore *= 1.1
		}
	}

	return baseScore
}

// calculateVolatility 计算波动率
func (dwl *DynamicWeightLearner) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// 计算平均值
	sum := 0.0
	for _, r := range returns {
		sum += r
	}
	mean := sum / float64(len(returns))

	// 计算方差
	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-mean, 2)
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}

// calculateMaxDrawdown 计算最大回撤
func (dwl *DynamicWeightLearner) calculateMaxDrawdown(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 计算累积收益率
	cumulative := make([]float64, len(returns))
	cumulative[0] = returns[0]

	maxDrawdown := 0.0
	peak := cumulative[0]

	for i := 1; i < len(returns); i++ {
		cumulative[i] = cumulative[i-1] + returns[i]
		if cumulative[i] > peak {
			peak = cumulative[i]
		}

		drawdown := peak - cumulative[i]
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateTrendStrength 计算趋势强度（线性回归斜率）
func (dwl *DynamicWeightLearner) calculateTrendStrength(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	n := float64(len(returns))

	// 计算线性回归斜率
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, r := range returns {
		x := float64(i)
		sumX += x
		sumY += r
		sumXY += x * r
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)

	// 标准化斜率（绝对值，表示趋势强度）
	return math.Abs(slope)
}

// GetFactorPerformance 获取因子表现统计
func (dwl *DynamicWeightLearner) GetFactorPerformance() map[string]FactorMetrics {
	return dwl.FactorPerformance
}

// UpdateMarketCondition 更新市场环境感知
func (dwl *DynamicWeightLearner) UpdateMarketCondition(volatility float64, trend string, sentiment string) {
	dwl.MarketConditionDetector.MarketVolatility = volatility
	dwl.MarketConditionDetector.MarketTrend = trend
	dwl.MarketConditionDetector.MarketSentiment = sentiment
}
