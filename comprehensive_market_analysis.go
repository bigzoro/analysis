package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// 综合市场分析和策略推荐系统
type ComprehensiveMarketAnalyzer struct {
	db *sql.DB
}

// 市场分析结果
type MarketAnalysisResult struct {
	MarketOverview     MarketOverview
	VolatilityAnalysis VolatilityAnalysis
	TrendAnalysis      TrendAnalysis
	StrategyAnalysis   StrategyAnalysis
	Recommendations    []StrategyRecommendation
}

// 市场概览
type MarketOverview struct {
	TotalSymbols     int
	ActiveSymbols    int
	AverageChange    float64
	AverageVolume    float64
	TimeRange        string
	LastUpdated      time.Time
}

// 波动率分析
type VolatilityAnalysis struct {
	LowVolatilityCount     int
	MediumVolatilityCount  int
	HighVolatilityCount    int
	ExtremeVolatilityCount int
	AverageVolatility      float64
	MostVolatileCoins      []CoinVolatility
}

// 趋势分析
type TrendAnalysis struct {
	StrongBullCount    int
	ModerateBullCount  int
	NeutralCount       int
	ModerateBearCount  int
	StrongBearCount    int
	TopGainers         []CoinChange
	TopLosers          []CoinChange
	MarketSentiment    string
}

// 策略分析
type StrategyAnalysis struct {
	MarketRegime     string
	RegimeConfidence float64
	SuitableStrategies []StrategySuitability
}

// 策略推荐
type StrategyRecommendation struct {
	StrategyName     string
	SuitabilityScore float64
	RiskLevel        string
	ExpectedReturn   string
	Confidence       float64
	Reasoning        string
}

type CoinVolatility struct {
	Symbol     string
	Volatility float64
	Volume     float64
}

type CoinChange struct {
	Symbol string
	Change float64
	Volume float64
}

type StrategySuitability struct {
	Name             string
	Score            float64
	SuitableEnvs     []string
	RiskLevel        string
	BestConditions   string
}

func main() {
	fmt.Println("🎯 综合市场分析和策略推荐系统")
	fmt.Println("=====================================")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	analyzer := &ComprehensiveMarketAnalyzer{db: db}

	// 执行综合分析
	result, err := analyzer.performComprehensiveAnalysis()
	if err != nil {
		log.Fatal("分析失败:", err)
	}

	// 显示分析结果
	analyzer.displayAnalysisResults(result)

	fmt.Println("\n🎉 分析完成！")
}

// 执行综合市场分析
func (cma *ComprehensiveMarketAnalyzer) performComprehensiveAnalysis() (*MarketAnalysisResult, error) {
	result := &MarketAnalysisResult{}

	// 1. 市场概览分析
	overview, err := cma.analyzeMarketOverview()
	if err != nil {
		return nil, fmt.Errorf("市场概览分析失败: %v", err)
	}
	result.MarketOverview = *overview

	// 2. 波动率分析
	volatility, err := cma.analyzeVolatility()
	if err != nil {
		return nil, fmt.Errorf("波动率分析失败: %v", err)
	}
	result.VolatilityAnalysis = *volatility

	// 3. 趋势分析
	trend, err := cma.analyzeTrends()
	if err != nil {
		return nil, fmt.Errorf("趋势分析失败: %v", err)
	}
	result.TrendAnalysis = *trend

	// 4. 策略分析
	strategyAnalysis, err := cma.analyzeStrategySuitability(result)
	if err != nil {
		return nil, fmt.Errorf("策略分析失败: %v", err)
	}
	result.StrategyAnalysis = *strategyAnalysis

	// 5. 生成推荐
	recommendations := cma.generateRecommendations(result)
	result.Recommendations = recommendations

	return result, nil
}

// 分析市场概览
func (cma *ComprehensiveMarketAnalyzer) analyzeMarketOverview() (*MarketOverview, error) {
	query := `
		SELECT
			COUNT(*) as total_symbols,
			COUNT(CASE WHEN quote_volume > 1000000 THEN 1 END) as active_symbols,
			COALESCE(AVG(price_change_percent), 0) as avg_change,
			COALESCE(AVG(quote_volume), 0) as avg_volume,
			MAX(created_at) as last_updated
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000`

	var overview MarketOverview
	err := cma.db.QueryRow(query).Scan(
		&overview.TotalSymbols,
		&overview.ActiveSymbols,
		&overview.AverageChange,
		&overview.AverageVolume,
		&overview.LastUpdated,
	)

	if err != nil {
		return nil, err
	}

	overview.TimeRange = "24小时"
	return &overview, nil
}

// 分析波动率
func (cma *ComprehensiveMarketAnalyzer) analyzeVolatility() (*VolatilityAnalysis, error) {
	// 计算波动率分布
	query := `
		SELECT
			COUNT(CASE WHEN volatility < 2 THEN 1 END) as low_vol,
			COUNT(CASE WHEN volatility >= 2 AND volatility < 5 THEN 1 END) as medium_vol,
			COUNT(CASE WHEN volatility >= 5 AND volatility < 10 THEN 1 END) as high_vol,
			COUNT(CASE WHEN volatility >= 10 THEN 1 END) as extreme_vol,
			AVG(volatility) as avg_volatility
		FROM (
			SELECT (high_price - low_price) / low_price * 100 as volatility
			FROM binance_24h_stats
			WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
				AND market_type = 'spot'
				AND quote_volume > 100000
		) as vol_stats`

	var analysis VolatilityAnalysis
	err := cma.db.QueryRow(query).Scan(
		&analysis.LowVolatilityCount,
		&analysis.MediumVolatilityCount,
		&analysis.HighVolatilityCount,
		&analysis.ExtremeVolatilityCount,
		&analysis.AverageVolatility,
	)

	if err != nil {
		return nil, err
	}

	// 获取最波动性币种
	volQuery := `
		SELECT symbol, (high_price - low_price) / low_price * 100 as volatility, quote_volume
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 1000000
		ORDER BY volatility DESC
		LIMIT 10`

	rows, err := cma.db.Query(volQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var coin CoinVolatility
		if err := rows.Scan(&coin.Symbol, &coin.Volatility, &coin.Volume); err == nil {
			analysis.MostVolatileCoins = append(analysis.MostVolatileCoins, coin)
		}
	}

	return &analysis, nil
}

// 分析趋势
func (cma *ComprehensiveMarketAnalyzer) analyzeTrends() (*TrendAnalysis, error) {
	query := `
		SELECT
			COUNT(CASE WHEN price_change_percent > 5 THEN 1 END) as strong_bull,
			COUNT(CASE WHEN price_change_percent > 2 AND price_change_percent <= 5 THEN 1 END) as moderate_bull,
			COUNT(CASE WHEN price_change_percent >= -2 AND price_change_percent <= 2 THEN 1 END) as neutral,
			COUNT(CASE WHEN price_change_percent < -2 AND price_change_percent >= -5 THEN 1 END) as moderate_bear,
			COUNT(CASE WHEN price_change_percent < -5 THEN 1 END) as strong_bear
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000`

	var analysis TrendAnalysis
	err := cma.db.QueryRow(query).Scan(
		&analysis.StrongBullCount,
		&analysis.ModerateBullCount,
		&analysis.NeutralCount,
		&analysis.ModerateBearCount,
		&analysis.StrongBearCount,
	)

	if err != nil {
		return nil, err
	}

	// 获取涨幅榜
	gainersQuery := `
		SELECT symbol, price_change_percent, quote_volume
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000
		ORDER BY price_change_percent DESC
		LIMIT 10`

	rows, err := cma.db.Query(gainersQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var coin CoinChange
		if err := rows.Scan(&coin.Symbol, &coin.Change, &coin.Volume); err == nil {
			analysis.TopGainers = append(analysis.TopGainers, coin)
		}
	}

	// 获取跌幅榜
	losersQuery := `
		SELECT symbol, price_change_percent, quote_volume
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000
		ORDER BY price_change_percent ASC
		LIMIT 10`

	rows, err = cma.db.Query(losersQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var coin CoinChange
		if err := rows.Scan(&coin.Symbol, &coin.Change, &coin.Volume); err == nil {
			analysis.TopLosers = append(analysis.TopLosers, coin)
		}
	}

	// 分析市场情绪
	total := analysis.StrongBullCount + analysis.ModerateBullCount + analysis.NeutralCount +
			 analysis.ModerateBearCount + analysis.StrongBearCount

	bullRatio := float64(analysis.StrongBullCount+analysis.ModerateBullCount) / float64(total)
	bearRatio := float64(analysis.StrongBearCount+analysis.ModerateBearCount) / float64(total)
	neutralRatio := float64(analysis.NeutralCount) / float64(total)

	if bullRatio > 0.6 {
		analysis.MarketSentiment = "极度乐观"
	} else if bullRatio > 0.4 {
		analysis.MarketSentiment = "乐观"
	} else if bearRatio > 0.6 {
		analysis.MarketSentiment = "极度悲观"
	} else if bearRatio > 0.4 {
		analysis.MarketSentiment = "悲观"
	} else if neutralRatio > 0.6 {
		analysis.MarketSentiment = "中性-震荡"
	} else {
		analysis.MarketSentiment = "平衡"
	}

	return &analysis, nil
}

// 分析策略适用性
func (cma *ComprehensiveMarketAnalyzer) analyzeStrategySuitability(result *MarketAnalysisResult) (*StrategyAnalysis, error) {
	analysis := &StrategyAnalysis{}

	// 基于市场数据判断市场环境
	volatility := result.VolatilityAnalysis.AverageVolatility
	trendStrength := float64(result.TrendAnalysis.StrongBullCount+result.TrendAnalysis.StrongBearCount) /
					float64(result.MarketOverview.TotalSymbols)
	neutralRatio := float64(result.TrendAnalysis.NeutralCount) / float64(result.MarketOverview.TotalSymbols)

	// 判断市场环境
	if neutralRatio > 0.7 && volatility > 4 {
		analysis.MarketRegime = "高波动震荡市"
		analysis.RegimeConfidence = 0.85
	} else if neutralRatio > 0.6 {
		analysis.MarketRegime = "震荡市"
		analysis.RegimeConfidence = 0.80
	} else if trendStrength > 0.3 {
		if result.MarketOverview.AverageChange > 0 {
			analysis.MarketRegime = "强势上涨趋势市"
			analysis.RegimeConfidence = 0.75
		} else {
			analysis.MarketRegime = "强势下跌趋势市"
			analysis.RegimeConfidence = 0.75
		}
	} else if volatility < 3 {
		analysis.MarketRegime = "低波动整理市"
		analysis.RegimeConfidence = 0.70
	} else {
		analysis.MarketRegime = "混合市场"
		analysis.RegimeConfidence = 0.60
	}

	// 定义各种策略的适用性
	strategies := []StrategySuitability{
		{
			Name:           "均值回归策略",
			SuitableEnvs:   []string{"震荡市", "高波动震荡市", "低波动整理市"},
			RiskLevel:      "中等",
			BestConditions: "高震荡，低趋势强度",
		},
		{
			Name:           "网格交易策略",
			SuitableEnvs:   []string{"震荡市", "低波动整理市", "混合市场"},
			RiskLevel:      "低",
			BestConditions: "中等波动，价格区间明确",
		},
		{
			Name:           "趋势跟随策略",
			SuitableEnvs:   []string{"强势上涨趋势市", "强势下跌趋势市", "混合市场"},
			RiskLevel:      "高",
			BestConditions: "强趋势信号，较高波动率",
		},
		{
			Name:           "动量策略",
			SuitableEnvs:   []string{"强势上涨趋势市", "强势下跌趋势市"},
			RiskLevel:      "高",
			BestConditions: "强动量信号，快速市场变动",
		},
		{
			Name:           "统计套利策略",
			SuitableEnvs:   []string{"震荡市", "混合市场", "低波动整理市"},
			RiskLevel:      "中等",
			BestConditions: "相关资产价格偏离均值",
		},
		{
			Name:           "反转策略",
			SuitableEnvs:   []string{"震荡市", "高波动震荡市"},
			RiskLevel:      "高",
			BestConditions: "超买超卖信号明显",
		},
		{
			Name:           "突破策略",
			SuitableEnvs:   []string{"强势上涨趋势市", "强势下跌趋势市", "高波动震荡市"},
			RiskLevel:      "中等",
			BestConditions: "重要支撑阻力位突破",
		},
		{
			Name:           "波动率策略",
			SuitableEnvs:   []string{"高波动震荡市", "混合市场"},
			RiskLevel:      "高",
			BestConditions: "波动率快速变化",
		},
		{
			Name:           "多空对冲策略",
			SuitableEnvs:   []string{"混合市场", "强势上涨趋势市", "强势下跌趋势市"},
			RiskLevel:      "中等",
			BestConditions: "多空力量相对平衡",
		},
		{
			Name:           "做空策略",
			SuitableEnvs:   []string{"强势下跌趋势市"},
			RiskLevel:      "极高",
			BestConditions: "熊市确认，风险偏好极低",
		},
	}

	// 计算每种策略的适用性评分
	for _, strategy := range strategies {
		score := 0.0
		isSuitable := false

		for _, env := range strategy.SuitableEnvs {
			if strings.Contains(analysis.MarketRegime, env) {
				isSuitable = true
				if env == analysis.MarketRegime {
					score += 1.0 // 完全匹配
				} else {
					score += 0.6 // 部分匹配
				}
			}
		}

		if isSuitable {
			strategy.Score = score * analysis.RegimeConfidence
		} else {
			strategy.Score = score * 0.3 // 不适合环境的策略给低分
		}

		analysis.SuitableStrategies = append(analysis.SuitableStrategies, strategy)
	}

	// 按评分排序
	sort.Slice(analysis.SuitableStrategies, func(i, j int) bool {
		return analysis.SuitableStrategies[i].Score > analysis.SuitableStrategies[j].Score
	})

	return analysis, nil
}

// 生成策略推荐
func (cma *ComprehensiveMarketAnalyzer) generateRecommendations(result *MarketAnalysisResult) []StrategyRecommendation {
	var recommendations []StrategyRecommendation

	strategyTemplates := map[string]StrategyRecommendation{
		"均值回归策略": {
			StrategyName:   "均值回归策略",
			RiskLevel:      "中等",
			ExpectedReturn: "2-5%每月",
			Reasoning:      "当前市场震荡特征明显，适合捕捉价格偏离机会",
		},
		"网格交易策略": {
			StrategyName:   "网格交易策略",
			RiskLevel:      "低",
			ExpectedReturn: "1-3%每月",
			Reasoning:      "价格在区间内震荡，网格策略可稳定获利",
		},
		"趋势跟随策略": {
			StrategyName:   "趋势跟随策略",
			RiskLevel:      "高",
			ExpectedReturn: "5-15%每月",
			Reasoning:      "市场有一定趋势信号，可跟随主流趋势",
		},
		"统计套利策略": {
			StrategyName:   "统计套利策略",
			RiskLevel:      "中等",
			ExpectedReturn: "2-6%每月",
			Reasoning:      "相关资产间存在价格偏离机会",
		},
		"波动率策略": {
			StrategyName:   "波动率策略",
			RiskLevel:      "高",
			ExpectedReturn: "3-10%每月",
			Reasoning:      "当前波动率较高，适合波动率相关策略",
		},
	}

	// 基于策略分析生成推荐
	for _, strategy := range result.StrategyAnalysis.SuitableStrategies {
		if template, exists := strategyTemplates[strategy.Name]; exists && strategy.Score > 0.4 {
			recommendation := template
			recommendation.SuitabilityScore = strategy.Score
			recommendation.Confidence = strategy.Score * 100

			recommendations = append(recommendations, recommendation)
		}

		if len(recommendations) >= 5 { // 最多推荐5种策略
			break
		}
	}

	// 按适用性评分排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].SuitabilityScore > recommendations[j].SuitabilityScore
	})

	return recommendations
}

// 显示分析结果
func (cma *ComprehensiveMarketAnalyzer) displayAnalysisResults(result *MarketAnalysisResult) {
	fmt.Println("\n📊 市场概览")
	fmt.Println("====================")
	fmt.Printf("总交易对数: %d\n", result.MarketOverview.TotalSymbols)
	fmt.Printf("活跃交易对: %d\n", result.MarketOverview.ActiveSymbols)
	fmt.Printf("平均涨跌幅: %.2f%%\n", result.MarketOverview.AverageChange)
	fmt.Printf("平均交易量: %.0f\n", result.MarketOverview.AverageVolume)
	fmt.Printf("时间范围: %s\n", result.MarketOverview.TimeRange)
	fmt.Printf("最后更新: %s\n", result.MarketOverview.LastUpdated.Format("2006-01-02 15:04:05"))

	fmt.Println("\n🌊 波动率分析")
	fmt.Println("====================")
	fmt.Printf("平均波动率: %.2f%%\n", result.VolatilityAnalysis.AverageVolatility)
	fmt.Printf("低波动率币种 (<2%%): %d\n", result.VolatilityAnalysis.LowVolatilityCount)
	fmt.Printf("中等波动率币种 (2-5%%): %d\n", result.VolatilityAnalysis.MediumVolatilityCount)
	fmt.Printf("高波动率币种 (5-10%%): %d\n", result.VolatilityAnalysis.HighVolatilityCount)
	fmt.Printf("极高波动率币种 (>10%%): %d\n", result.VolatilityAnalysis.ExtremeVolatilityCount)

	if len(result.VolatilityAnalysis.MostVolatileCoins) > 0 {
		fmt.Println("\n最波动性币种 TOP5:")
		for i, coin := range result.VolatilityAnalysis.MostVolatileCoins {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s: %.1f%% (交易量: %.0f)\n",
				i+1, coin.Symbol, coin.Volatility, coin.Volume)
		}
	}

	fmt.Println("\n📈 趋势分析")
	fmt.Println("====================")
	fmt.Printf("强势上涨 (>5%%): %d\n", result.TrendAnalysis.StrongBullCount)
	fmt.Printf("温和上涨 (2-5%%): %d\n", result.TrendAnalysis.ModerateBullCount)
	fmt.Printf("横盘震荡 (-2%%到2%%): %d\n", result.TrendAnalysis.NeutralCount)
	fmt.Printf("温和下跌 (-5%%到-2%%): %d\n", result.TrendAnalysis.ModerateBearCount)
	fmt.Printf("强势下跌 (<-5%%): %d\n", result.TrendAnalysis.StrongBearCount)
	fmt.Printf("市场情绪: %s\n", result.TrendAnalysis.MarketSentiment)

	if len(result.TrendAnalysis.TopGainers) > 0 {
		fmt.Println("\n涨幅榜 TOP5:")
		for i, coin := range result.TrendAnalysis.TopGainers {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s: %+6.2f%% (交易量: %.0f)\n",
				i+1, coin.Symbol, coin.Change, coin.Volume)
		}
	}

	if len(result.TrendAnalysis.TopLosers) > 0 {
		fmt.Println("\n跌幅榜 TOP5:")
		for i, coin := range result.TrendAnalysis.TopLosers {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s: %+6.2f%% (交易量: %.0f)\n",
				i+1, coin.Symbol, coin.Change, coin.Volume)
		}
	}

	fmt.Println("\n🎯 策略分析")
	fmt.Println("====================")
	fmt.Printf("当前市场环境: %s\n", result.StrategyAnalysis.MarketRegime)
	fmt.Printf("环境判断置信度: %.1f%%\n", result.StrategyAnalysis.RegimeConfidence*100)

	fmt.Println("\n🏆 策略推荐")
	fmt.Println("====================")
	for i, rec := range result.Recommendations {
		fmt.Printf("\n%d. %s\n", i+1, rec.StrategyName)
		fmt.Printf("   适用性评分: %.1f/1.0\n", rec.SuitabilityScore)
		fmt.Printf("   风险等级: %s\n", rec.RiskLevel)
		fmt.Printf("   预期收益: %s\n", rec.ExpectedReturn)
		fmt.Printf("   置信度: %.1f%%\n", rec.Confidence)
		fmt.Printf("   推荐理由: %s\n", rec.Reasoning)
	}

	fmt.Println("\n💡 投资建议")
	fmt.Println("====================")
	fmt.Printf("• 当前市场环境: %s，建议重点关注%s类策略\n", result.StrategyAnalysis.MarketRegime, cma.getStrategyCategory(result.StrategyAnalysis.MarketRegime))
	fmt.Printf("• 波动率水平: %.1f%%，%s\n", result.VolatilityAnalysis.AverageVolatility, cma.getVolatilityAdvice(result.VolatilityAnalysis.AverageVolatility))
	fmt.Printf("• 市场情绪: %s，%s\n", result.TrendAnalysis.MarketSentiment, cma.getSentimentAdvice(result.TrendAnalysis.MarketSentiment))
	fmt.Printf("• 风险控制: 建议单策略仓位不超过总资金的%d%%\n", cma.getPositionLimit(result.StrategyAnalysis.MarketRegime))
}

// 获取策略类别建议
func (cma *ComprehensiveMarketAnalyzer) getStrategyCategory(regime string) string {
	switch regime {
	case "震荡市", "高波动震荡市":
		return "均值回归和网格交易"
	case "强势上涨趋势市", "强势下跌趋势市":
		return "趋势跟随和动量"
	case "低波动整理市":
		return "网格交易和统计套利"
	default:
		return "多元化策略组合"
	}
}

// 获取波动率建议
func (cma *ComprehensiveMarketAnalyzer) getVolatilityAdvice(volatility float64) string {
	if volatility > 8 {
		return "波动率较高，建议降低杠杆倍数"
	} else if volatility > 5 {
		return "波动率适中，策略参数可正常设置"
	} else {
		return "波动率较低，可适当放宽止损条件"
	}
}

// 获取情绪建议
func (cma *ComprehensiveMarketAnalyzer) getSentimentAdvice(sentiment string) string {
	switch sentiment {
	case "极度乐观":
		return "市场过热，注意风险控制，适当减仓"
	case "乐观":
		return "市场向好，可适度增加仓位"
	case "极度悲观":
		return "市场恐慌，可关注抄底机会"
	case "悲观":
		return "市场谨慎，建议轻仓操作"
	case "中性-震荡":
		return "市场平静，适合稳健策略"
	default:
		return "市场平衡，可正常操作"
	}
}

// 获取仓位限制建议
func (cma *ComprehensiveMarketAnalyzer) getPositionLimit(regime string) int {
	switch regime {
	case "高波动震荡市", "强势上涨趋势市", "强势下跌趋势市":
		return 15
	case "震荡市", "混合市场":
		return 20
	case "低波动整理市":
		return 25
	default:
		return 20
	}
}