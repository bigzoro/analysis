package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"
)

// 市场环境分析和策略推荐系统
type MarketAnalyzer struct {
	db *sql.DB
}

// 市场状态评估
type MarketAssessment struct {
	TotalSymbols       int     `json:"total_symbols"`
	ActiveSymbols      int     `json:"active_symbols"`
	AvgVolatility      float64 `json:"avg_volatility"`
	AvgPriceChange     float64 `json:"avg_price_change"`
	BullishRatio       float64 `json:"bullish_ratio"`
	BearishRatio       float64 `json:"bearish_ratio"`
	OscillatingRatio   float64 `json:"oscillating_ratio"`
	TrendingRatio      float64 `json:"trending_ratio"`
	MarketEnvironment  string  `json:"market_environment"`
	RecommendedStrategy string `json:"recommended_strategy"`
}

// 策略表现分析
type StrategyPerformance struct {
	StrategyName      string
	WinRate          float64
	AvgProfit        float64
	MaxDrawdown      float64
	TotalTrades      int
	SuitableEnvironment string
	Score            float64 // 综合评分
}

func main() {
	fmt.Println("🎯 市场环境分析和策略推荐系统")
	fmt.Println("=====================================")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	analyzer := &MarketAnalyzer{db: db}

	// 执行全面市场分析
	fmt.Println("\n📊 执行市场环境分析...")
	assessment, err := analyzer.analyzeMarketEnvironment()
	if err != nil {
		log.Fatal("市场分析失败:", err)
	}

	// 显示市场评估结果
	analyzer.displayMarketAssessment(assessment)

	// 分析策略表现
	fmt.Println("\n📈 分析策略表现...")
	strategyPerformances := analyzer.analyzeStrategyPerformance()

	// 基于市场环境推荐策略
	recommendations := analyzer.generateStrategyRecommendations(assessment, strategyPerformances)

	// 显示详细推荐结果
	analyzer.displayStrategyRecommendations(recommendations, assessment)

	// 生成操作建议
	analyzer.generateActionPlan(assessment, recommendations)

	fmt.Println("\n🎉 分析完成！")
}

// 分析市场环境
func (ma *MarketAnalyzer) analyzeMarketEnvironment() (*MarketAssessment, error) {
	// 首先检查最近24小时是否有数据
	var recentCount int
	err := ma.db.QueryRow("SELECT COUNT(*) FROM binance_24h_stats WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)").Scan(&recentCount)
	if err != nil {
		return nil, fmt.Errorf("检查最近数据失败: %v", err)
	}

	timeRange := "24 HOUR"
	if recentCount == 0 {
		// 如果最近24小时没有数据，使用最近7天的数据
		timeRange = "7 DAY"
		log.Printf("⚠️  最近24小时无市场数据，使用最近7天数据进行分析")
	}

	query := fmt.Sprintf(`
		SELECT
			COUNT(*) as total_symbols,
			COUNT(CASE WHEN quote_volume > 1000000 THEN 1 END) as active_symbols,
			COALESCE(AVG(price_change_percent), 0) as avg_price_change,
			COALESCE(AVG((high_price - low_price) / NULLIF(low_price, 0) * 100), 0) as avg_volatility,
			COALESCE(SUM(CASE WHEN price_change_percent > 2 THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 0) as bullish_ratio,
			COALESCE(SUM(CASE WHEN price_change_percent < -2 THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 0) as bearish_ratio,
			COALESCE(SUM(CASE WHEN ABS(price_change_percent) <= 2 THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 0) as oscillating_ratio,
			COALESCE(SUM(CASE WHEN ABS(price_change_percent) > 3 THEN 1 ELSE 0 END) / NULLIF(COUNT(*), 0), 0) as trending_ratio
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL %s)
			AND market_type = 'spot'
			AND quote_volume > 100000
	`, timeRange)

	var assessment MarketAssessment
	err = ma.db.QueryRow(query).Scan(
		&assessment.TotalSymbols,
		&assessment.ActiveSymbols,
		&assessment.AvgPriceChange,
		&assessment.AvgVolatility,
		&assessment.BullishRatio,
		&assessment.BearishRatio,
		&assessment.OscillatingRatio,
		&assessment.TrendingRatio,
	)

	if err != nil {
		return nil, fmt.Errorf("查询市场数据失败: %v", err)
	}

	// 如果使用的是7天数据，调整一些指标
	if timeRange == "7 DAY" {
		assessment.AvgPriceChange *= 0.3 // 7天平均值调整为更保守的估计
		assessment.AvgVolatility *= 0.5  // 波动率也相应调整
		log.Printf("📊 使用7天数据调整: 平均价格变化 %.2f%% -> %.2f%%, 波动率 %.2f%% -> %.2f%%",
			assessment.AvgPriceChange/0.3, assessment.AvgPriceChange,
			assessment.AvgVolatility/0.5, assessment.AvgVolatility)
	}

	// 判断市场环境
	assessment.MarketEnvironment = ma.determineMarketEnvironment(&assessment)

	return &assessment, nil
}

// 判断市场环境
func (ma *MarketAnalyzer) determineMarketEnvironment(assessment *MarketAssessment) string {
	// 基于多个指标判断市场环境
	trendingScore := assessment.BullishRatio + assessment.BearishRatio
	oscillatingScore := assessment.OscillatingRatio
	volatilityScore := assessment.AvgVolatility

	// 趋势市场：强趋势信号 + 较高波动率
	if trendingScore > 0.4 && volatilityScore > 6 {
		if assessment.AvgPriceChange > 0 {
			return "强牛市"
		} else {
			return "强熊市"
		}
	}

	// 震荡市场：弱趋势信号 + 中等波动率
	if oscillatingScore > 0.6 && volatilityScore < 8 {
		return "震荡市"
	}

	// 横盘整理：极弱趋势信号 + 低波动率
	if trendingScore < 0.2 && volatilityScore < 4 {
		return "横盘整理"
	}

	// 其他情况
	if volatilityScore > 10 {
		return "高波动震荡"
	}

	return "混合市场"
}

// 显示市场评估结果
func (ma *MarketAnalyzer) displayMarketAssessment(assessment *MarketAssessment) {
	fmt.Println("\n📊 市场环境评估结果")
	fmt.Println("====================")

	fmt.Printf("📈 总交易对数: %d\n", assessment.TotalSymbols)
	fmt.Printf("🎯 活跃交易对: %d\n", assessment.ActiveSymbols)
	fmt.Printf("📊 平均波动率: %.2f%%\n", assessment.AvgVolatility)
	fmt.Printf("💰 平均价格变化: %.2f%%\n", assessment.AvgPriceChange)
	fmt.Printf("🐂 多头占比: %.1f%%\n", assessment.BullishRatio*100)
	fmt.Printf("🐻 空头占比: %.1f%%\n", assessment.BearishRatio*100)
	fmt.Printf("🔄 震荡占比: %.1f%%\n", assessment.OscillatingRatio*100)
	fmt.Printf("📈 趋势占比: %.1f%%\n", assessment.TrendingRatio*100)

	fmt.Printf("\n🎯 当前市场环境: %s\n", assessment.MarketEnvironment)

	// 市场环境详细说明
	switch assessment.MarketEnvironment {
	case "强牛市":
		fmt.Println("💡 市场特征: 强势上涨趋势，资金风险偏好高，适合激进策略")
	case "强熊市":
		fmt.Println("💡 市场特征: 强势下跌趋势，风险较高，适合空头策略")
	case "震荡市":
		fmt.Println("💡 市场特征: 价格在区间内震荡，适合均值回归策略")
	case "横盘整理":
		fmt.Println("💡 市场特征: 价格横盘整理，波动率低，适合稳健策略")
	case "高波动震荡":
		fmt.Println("💡 市场特征: 高波动但无明确方向，适合高频策略")
	case "混合市场":
		fmt.Println("💡 市场特征: 复杂多变，需要灵活策略组合")
	}
}

// 分析策略表现
func (ma *MarketAnalyzer) analyzeStrategyPerformance() []StrategyPerformance {
	// 基于市场环境和历史数据分析策略表现
	// 这里是模拟的策略表现数据，实际应该从数据库中获取

	strategies := []StrategyPerformance{
		{
			StrategyName:         "均值回归策略",
			WinRate:             0.68,
			AvgProfit:           2.3,
			MaxDrawdown:         8.5,
			TotalTrades:         245,
			SuitableEnvironment: "震荡市,横盘整理",
			Score:               8.2,
		},
		{
			StrategyName:         "均线策略",
			WinRate:             0.62,
			AvgProfit:           1.8,
			MaxDrawdown:         12.3,
			TotalTrades:         189,
			SuitableEnvironment: "强牛市,强熊市",
			Score:               6.8,
		},
		{
			StrategyName:         "做空策略",
			WinRate:             0.71,
			AvgProfit:           3.1,
			MaxDrawdown:         15.2,
			TotalTrades:         67,
			SuitableEnvironment: "强熊市,高波动震荡",
			Score:               7.1,
		},
		{
			StrategyName:         "高级均线策略",
			WinRate:             0.59,
			AvgProfit:           1.5,
			MaxDrawdown:         9.8,
			TotalTrades:         312,
			SuitableEnvironment: "混合市场",
			Score:               7.3,
		},
	}

	// 按评分排序
	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Score > strategies[j].Score
	})

	return strategies
}

// 生成策略推荐
func (ma *MarketAnalyzer) generateStrategyRecommendations(assessment *MarketAssessment, performances []StrategyPerformance) []StrategyRecommendation {
	var recommendations []StrategyRecommendation

	for _, perf := range performances {
		// 计算环境匹配度
		environmentMatch := ma.calculateEnvironmentMatch(assessment.MarketEnvironment, perf.SuitableEnvironment)

		// 计算综合评分
		compositeScore := perf.Score * environmentMatch

		recommendations = append(recommendations, StrategyRecommendation{
			Strategy:        perf,
			EnvironmentMatch: environmentMatch,
			CompositeScore:   compositeScore,
			Priority:         ma.calculatePriority(compositeScore),
		})
	}

	// 按综合评分排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].CompositeScore > recommendations[j].CompositeScore
	})

	return recommendations
}

// 计算环境匹配度
func (ma *MarketAnalyzer) calculateEnvironmentMatch(currentEnv, suitableEnvs string) float64 {
	// 简单的字符串匹配，实际可以更复杂
	if strings.Contains(suitableEnvs, currentEnv) {
		return 1.0 // 完全匹配
	}

	// 部分匹配逻辑
	switch currentEnv {
	case "震荡市":
		if strings.Contains(suitableEnvs, "横盘整理") {
			return 0.8
		}
	case "横盘整理":
		if strings.Contains(suitableEnvs, "震荡市") {
			return 0.8
		}
	case "强牛市", "强熊市":
		if strings.Contains(suitableEnvs, "混合市场") {
			return 0.6
		}
	}

	return 0.3 // 默认匹配度
}

// 计算优先级
func (ma *MarketAnalyzer) calculatePriority(score float64) string {
	switch {
	case score >= 8.0:
		return "⭐⭐⭐⭐⭐ 极力推荐"
	case score >= 7.0:
		return "⭐⭐⭐⭐ 强烈推荐"
	case score >= 6.0:
		return "⭐⭐⭐ 推荐"
	case score >= 5.0:
		return "⭐⭐ 谨慎推荐"
	default:
		return "⭐ 不推荐"
	}
}

// 显示策略推荐结果
func (ma *MarketAnalyzer) displayStrategyRecommendations(recommendations []StrategyRecommendation, assessment *MarketAssessment) {
	fmt.Println("\n🎯 策略推荐结果")
	fmt.Println("===============")

	fmt.Printf("基于当前市场环境 (%s) 的分析结果:\n\n", assessment.MarketEnvironment)

	for i, rec := range recommendations {
		if i >= 3 { // 只显示前3个推荐
			break
		}

		fmt.Printf("%d. %s\n", i+1, rec.Strategy.StrategyName)
		fmt.Printf("   %s\n", rec.Priority)
		fmt.Printf("   胜率: %.1f%% | 平均收益: %.1f%% | 最大回撤: %.1f%%\n", rec.Strategy.WinRate*100, rec.Strategy.AvgProfit, rec.Strategy.MaxDrawdown)
		fmt.Printf("   环境匹配度: %.0f%% | 综合评分: %.1f\n", rec.EnvironmentMatch*100, rec.CompositeScore)
		fmt.Printf("   适用环境: %s\n\n", rec.Strategy.SuitableEnvironment)
	}
}

// 生成操作计划
func (ma *MarketAnalyzer) generateActionPlan(assessment *MarketAssessment, recommendations []StrategyRecommendation) {
	fmt.Println("\n📋 操作执行计划")
	fmt.Println("===============")

	if len(recommendations) == 0 {
		fmt.Println("⚠️  没有找到合适的策略推荐")
		return
	}

	topRecommendation := recommendations[0]

	fmt.Printf("🎯 首要推荐策略: %s\n", topRecommendation.Strategy.StrategyName)
	fmt.Printf("📊 当前市场环境: %s\n", assessment.MarketEnvironment)
	fmt.Printf("💯 综合评分: %.1f\n\n", topRecommendation.CompositeScore)

	// 基于市场环境给出具体建议
	switch assessment.MarketEnvironment {
	case "震荡市", "横盘整理":
		fmt.Println("🎪 策略调整建议:")
		fmt.Println("   • 启用均值回归策略，捕捉价格偏离机会")
		fmt.Println("   • 设置较宽的止盈止损区间")
		fmt.Println("   • 降低单次仓位比例")
		fmt.Println("   • 关注成交量确认信号")

	case "强牛市":
		fmt.Println("🚀 策略调整建议:")
		fmt.Println("   • 启用均线策略，跟随上涨趋势")
		fmt.Println("   • 适当增加杠杆使用")
		fmt.Println("   • 调整止盈目标为更高水平")
		fmt.Println("   • 关注强势币种的补涨机会")

	case "强熊市":
		fmt.Println("🐻 策略调整建议:")
		fmt.Println("   • 启用做空策略，利用下跌机会")
		fmt.Println("   • 严格控制风险敞口")
		fmt.Println("   • 关注超跌反弹机会")
		fmt.Println("   • 适当降低止损幅度")

	case "高波动震荡":
		fmt.Println("⚡ 策略调整建议:")
		fmt.Println("   • 结合均值回归和突破策略")
		fmt.Println("   • 使用更严格的信号过滤")
		fmt.Println("   • 实施更频繁的风险控制")
		fmt.Println("   • 关注高波动币种的机会")

	default:
		fmt.Println("🔄 策略调整建议:")
		fmt.Println("   • 保持灵活策略组合")
		fmt.Println("   • 定期评估策略表现")
		fmt.Println("   • 根据市场变化调整参数")
		fmt.Println("   • 关注系统性风险")
	}

	fmt.Println("\n⚙️  技术参数建议:")
	fmt.Println("   • 波动率阈值: 根据当前市场调整")
	fmt.Println("   • 信号质量要求: 震荡市放宽，趋势市收紧")
	fmt.Println("   • 仓位管理: 控制在总资金的20-50%")
	fmt.Println("   • 止损设置: 根据波动率动态调整")

	fmt.Println("\n⏰ 监控建议:")
	fmt.Println("   • 每日检查市场环境变化")
	fmt.Println("   • 每周评估策略表现指标")
	fmt.Println("   • 每月进行策略参数优化")
	fmt.Println("   • 及时响应重大市场事件")
}

// 策略推荐结构
type StrategyRecommendation struct {
	Strategy         StrategyPerformance
	EnvironmentMatch float64 // 环境匹配度 (0-1)
	CompositeScore   float64 // 综合评分
	Priority         string  // 优先级描述
}