package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"

	_ "github.com/go-sql-driver/mysql"
)

// 高级策略推荐系统 - 基于市场环境智能推荐
type AdvancedStrategyRecommender struct {
	db *sql.DB
}

// 市场环境分类
type MarketEnvironment struct {
	Regime        string
	Volatility    float64
	TrendStrength float64
	Correlation   float64
	Confidence    float64
	Description   string
}

// 策略性能指标
type StrategyPerformance struct {
	Name              string
	SharpeRatio       float64
	WinRate           float64
	AvgReturn         float64
	MaxDrawdown       float64
	CalmarRatio       float64
	ProfitFactor      float64
	RecoveryFactor    float64
	ExpectedValue     float64
	RiskAdjustedScore float64
}

// 策略推荐结果
type StrategyRecommendation struct {
	Strategy       StrategyPerformance
	MarketFit      float64
	RiskScore      float64
	LiquidityFit   float64
	CompositeScore float64
	Priority       int
	Allocation     float64
	Rationale      string
	Parameters     map[string]interface{}
}

// 投资组合配置
type PortfolioConfig struct {
	MarketEnvironment    MarketEnvironment
	PrimaryStrategies    []StrategyRecommendation
	SecondaryStrategies  []StrategyRecommendation
	DiversificationScore float64
	RiskParityWeights    map[string]float64
	MaxDrawdownLimit     float64
	RebalancingFreq      string
	StopLossRules        []string
}

func main() {
	fmt.Println("🤖 高级策略推荐系统")
	fmt.Println("===================")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	recommender := &AdvancedStrategyRecommender{db: db}

	// 分析当前市场环境
	fmt.Println("\n🔍 第一步: 市场环境深度分析")
	marketEnv, err := recommender.analyzeMarketEnvironment()
	if err != nil {
		log.Fatal("市场环境分析失败:", err)
	}
	recommender.displayMarketEnvironment(marketEnv)

	// 生成策略推荐
	fmt.Println("\n🎯 第二步: 生成策略推荐")
	strategies := recommender.initializeStrategyLibrary()
	recommendations, err := recommender.generateStrategyRecommendations(strategies, marketEnv)
	if err != nil {
		log.Fatal("策略推荐生成失败:", err)
	}
	recommender.displayStrategyRecommendations(recommendations, marketEnv)

	// 构建投资组合
	fmt.Println("\n💼 第三步: 投资组合构建")
	portfolio := recommender.buildPortfolioConfig(recommendations, marketEnv)
	recommender.displayPortfolioConfig(portfolio)

	// 风险管理建议
	fmt.Println("\n⚠️ 第四步: 风险管理框架")
	riskManagement := recommender.generateRiskManagementFramework(portfolio)
	recommender.displayRiskManagementFramework(riskManagement)

	// 执行建议
	fmt.Println("\n🚀 第五步: 执行计划")
	executionPlan := recommender.generateExecutionPlan(portfolio)
	recommender.displayExecutionPlan(executionPlan)

	fmt.Println("\n🎉 策略推荐分析完成！")
}

// 分析市场环境
func (r *AdvancedStrategyRecommender) analyzeMarketEnvironment() (*MarketEnvironment, error) {
	env := &MarketEnvironment{}

	// 查询24小时市场数据
	query := `
		SELECT
			AVG(price_change_percent) as avg_change,
			STDDEV(price_change_percent) as change_volatility,
			COUNT(CASE WHEN price_change_percent > 2 THEN 1 END) / COUNT(*) as bull_ratio,
			COUNT(CASE WHEN price_change_percent < -2 THEN 1 END) / COUNT(*) as bear_ratio,
			COUNT(CASE WHEN ABS(price_change_percent) <= 2 THEN 1 END) / COUNT(*) as neutral_ratio
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 1000000`

	var avgChange, changeVolatility, bullRatio, bearRatio, neutralRatio float64
	err := r.db.QueryRow(query).Scan(&avgChange, &changeVolatility, &bullRatio, &bearRatio, &neutralRatio)
	if err != nil {
		return nil, fmt.Errorf("市场数据查询失败: %v", err)
	}

	// 计算趋势强度
	trendStrength := bullRatio + bearRatio

	// 判断市场环境
	if changeVolatility > 8 && neutralRatio > 0.6 {
		env.Regime = "高波动震荡市"
		env.Confidence = 0.85
		env.Description = "价格剧烈波动但无明确方向，适合均值回归和波动率策略"
	} else if changeVolatility > 6 && neutralRatio > 0.7 {
		env.Regime = "震荡市"
		env.Confidence = 0.80
		env.Description = "价格在区间内震荡，适合网格交易和统计套利"
	} else if trendStrength > 0.4 {
		if avgChange > 0 {
			env.Regime = "强势上涨趋势市"
			env.Confidence = 0.75
			env.Description = "明显上涨趋势，适合趋势跟随和动量策略"
		} else {
			env.Regime = "强势下跌趋势市"
			env.Confidence = 0.75
			env.Description = "明显下跌趋势，适合做空和对冲策略"
		}
	} else if changeVolatility < 4 && neutralRatio > 0.8 {
		env.Regime = "低波动整理市"
		env.Confidence = 0.70
		env.Description = "市场平静，适合网格交易和稳健策略"
	} else {
		env.Regime = "混合市场"
		env.Confidence = 0.60
		env.Description = "复杂多变的市场环境，需要灵活策略组合"
	}

	env.Volatility = changeVolatility
	env.TrendStrength = trendStrength
	env.Correlation = 0.5 // 默认中性相关性

	return env, nil
}

// 初始化策略库
func (r *AdvancedStrategyRecommender) initializeStrategyLibrary() []StrategyPerformance {
	strategies := []StrategyPerformance{
		{
			Name:              "均值回归策略",
			SharpeRatio:       1.8,
			WinRate:           0.62,
			AvgReturn:         2.3,
			MaxDrawdown:       12.5,
			CalmarRatio:       0.184,
			ProfitFactor:      1.45,
			RecoveryFactor:    0.78,
			ExpectedValue:     0.023,
			RiskAdjustedScore: 8.2,
		},
		{
			Name:              "网格交易策略",
			SharpeRatio:       2.1,
			WinRate:           0.68,
			AvgReturn:         1.8,
			MaxDrawdown:       8.2,
			CalmarRatio:       0.220,
			ProfitFactor:      1.62,
			RecoveryFactor:    1.12,
			ExpectedValue:     0.018,
			RiskAdjustedScore: 8.5,
		},
		{
			Name:              "统计套利策略",
			SharpeRatio:       1.6,
			WinRate:           0.58,
			AvgReturn:         2.8,
			MaxDrawdown:       15.8,
			CalmarRatio:       0.177,
			ProfitFactor:      1.38,
			RecoveryFactor:    0.65,
			ExpectedValue:     0.028,
			RiskAdjustedScore: 7.8,
		},
		{
			Name:              "波动率策略",
			SharpeRatio:       1.4,
			WinRate:           0.55,
			AvgReturn:         3.2,
			MaxDrawdown:       18.5,
			CalmarRatio:       0.173,
			ProfitFactor:      1.35,
			RecoveryFactor:    0.58,
			ExpectedValue:     0.032,
			RiskAdjustedScore: 7.2,
		},
		{
			Name:              "多空对冲策略",
			SharpeRatio:       1.9,
			WinRate:           0.60,
			AvgReturn:         1.9,
			MaxDrawdown:       10.2,
			CalmarRatio:       0.186,
			ProfitFactor:      1.52,
			RecoveryFactor:    0.92,
			ExpectedValue:     0.019,
			RiskAdjustedScore: 8.1,
		},
		{
			Name:              "动量策略",
			SharpeRatio:       1.2,
			WinRate:           0.52,
			AvgReturn:         4.5,
			MaxDrawdown:       22.8,
			CalmarRatio:       0.197,
			ProfitFactor:      1.28,
			RecoveryFactor:    0.42,
			ExpectedValue:     0.045,
			RiskAdjustedScore: 6.5,
		},
		{
			Name:              "趋势跟随策略",
			SharpeRatio:       1.3,
			WinRate:           0.54,
			AvgReturn:         3.8,
			MaxDrawdown:       20.5,
			CalmarRatio:       0.185,
			ProfitFactor:      1.32,
			RecoveryFactor:    0.48,
			ExpectedValue:     0.038,
			RiskAdjustedScore: 6.8,
		},
		{
			Name:              "反转策略",
			SharpeRatio:       0.9,
			WinRate:           0.48,
			AvgReturn:         2.1,
			MaxDrawdown:       25.2,
			CalmarRatio:       0.083,
			ProfitFactor:      1.18,
			RecoveryFactor:    0.32,
			ExpectedValue:     0.021,
			RiskAdjustedScore: 5.2,
		},
		{
			Name:              "突破策略",
			SharpeRatio:       1.5,
			WinRate:           0.56,
			AvgReturn:         2.9,
			MaxDrawdown:       16.8,
			CalmarRatio:       0.173,
			ProfitFactor:      1.42,
			RecoveryFactor:    0.62,
			ExpectedValue:     0.029,
			RiskAdjustedScore: 7.5,
		},
		{
			Name:              "做空策略",
			SharpeRatio:       0.8,
			WinRate:           0.45,
			AvgReturn:         -2.8,
			MaxDrawdown:       28.5,
			CalmarRatio:       -0.098,
			ProfitFactor:      0.85,
			RecoveryFactor:    0.22,
			ExpectedValue:     -0.028,
			RiskAdjustedScore: 3.5,
		},
	}

	return strategies
}

// 生成策略推荐
func (r *AdvancedStrategyRecommender) generateStrategyRecommendations(strategies []StrategyPerformance, marketEnv *MarketEnvironment) ([]StrategyRecommendation, error) {
	var recommendations []StrategyRecommendation

	for _, strategy := range strategies {
		rec := StrategyRecommendation{
			Strategy:   strategy,
			Parameters: make(map[string]interface{}),
		}

		// 计算市场适应性
		rec.MarketFit = r.calculateMarketFit(strategy.Name, marketEnv)

		// 计算风险评分 (基于最大回撤和夏普比率)
		rec.RiskScore = (strategy.SharpeRatio * 0.6) + ((1 - strategy.MaxDrawdown/30) * 0.4)

		// 计算流动性适应性 (基于胜率和利润因子)
		rec.LiquidityFit = (strategy.WinRate * 0.5) + (strategy.ProfitFactor * 0.3) + (strategy.RecoveryFactor * 0.2)

		// 计算综合评分
		rec.CompositeScore = (rec.MarketFit * 0.4) + (rec.RiskScore * 0.35) + (rec.LiquidityFit * 0.25)

		// 设置参数
		rec.Parameters = r.getStrategyParameters(strategy.Name, marketEnv)

		// 生成推荐理由
		rec.Rationale = r.generateRationale(strategy.Name, marketEnv, rec.MarketFit)

		recommendations = append(recommendations, rec)
	}

	// 按综合评分排序
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].CompositeScore > recommendations[j].CompositeScore
	})

	// 分配优先级和权重
	totalScore := 0.0
	for _, rec := range recommendations {
		totalScore += rec.CompositeScore
	}

	for i := range recommendations {
		if i < 3 {
			recommendations[i].Priority = i + 1
		} else {
			recommendations[i].Priority = 0 // 不推荐
		}
		recommendations[i].Allocation = recommendations[i].CompositeScore / totalScore
	}

	return recommendations, nil
}

// 计算策略的市场适应性
func (r *AdvancedStrategyRecommender) calculateMarketFit(strategyName string, marketEnv *MarketEnvironment) float64 {
	baseFit := 0.5

	switch marketEnv.Regime {
	case "高波动震荡市":
		switch strategyName {
		case "均值回归策略":
			baseFit = 1.3
		case "波动率策略":
			baseFit = 1.2
		case "网格交易策略":
			baseFit = 1.1
		case "统计套利策略":
			baseFit = 1.0
		case "多空对冲策略":
			baseFit = 0.9
		case "突破策略":
			baseFit = 0.8
		}
	case "震荡市":
		switch strategyName {
		case "均值回归策略":
			baseFit = 1.4
		case "网格交易策略":
			baseFit = 1.2
		case "统计套利策略":
			baseFit = 1.1
		case "多空对冲策略":
			baseFit = 1.0
		case "突破策略":
			baseFit = 0.9
		}
	case "强势上涨趋势市":
		switch strategyName {
		case "趋势跟随策略":
			baseFit = 1.4
		case "动量策略":
			baseFit = 1.3
		case "突破策略":
			baseFit = 1.1
		case "多空对冲策略":
			baseFit = 0.9
		}
	case "强势下跌趋势市":
		switch strategyName {
		case "做空策略":
			baseFit = 1.5
		case "多空对冲策略":
			baseFit = 1.3
		case "趋势跟随策略":
			baseFit = 1.1
		}
	case "低波动整理市":
		switch strategyName {
		case "网格交易策略":
			baseFit = 1.4
		case "统计套利策略":
			baseFit = 1.2
		case "均值回归策略":
			baseFit = 1.1
		case "多空对冲策略":
			baseFit = 1.0
		}
	}

	// 基于波动率调整
	if marketEnv.Volatility > 8 && strategyName == "波动率策略" {
		baseFit *= 1.2
	}

	// 限制在合理范围内
	if baseFit > 1.5 {
		baseFit = 1.5
	} else if baseFit < 0.1 {
		baseFit = 0.1
	}

	return baseFit
}

// 获取策略参数
func (r *AdvancedStrategyRecommender) getStrategyParameters(strategyName string, marketEnv *MarketEnvironment) map[string]interface{} {
	params := make(map[string]interface{})

	switch strategyName {
	case "均值回归策略":
		params["lookback_period"] = 20
		params["entry_threshold"] = 2.0
		params["exit_threshold"] = 0.5
		params["max_holding_time"] = "4h"
		if marketEnv.Volatility > 6 {
			params["entry_threshold"] = 2.5
		}

	case "网格交易策略":
		params["grid_levels"] = 10
		params["grid_spacing"] = 0.01
		params["min_volume"] = 10000
		params["rebalance_freq"] = "1h"
		if marketEnv.Volatility > 6 {
			params["grid_levels"] = 8
			params["grid_spacing"] = 0.015
		}

	case "统计套利策略":
		params["correlation_threshold"] = 0.8
		params["spread_threshold"] = 1.5
		params["hedge_ratio"] = 1.0
		params["max_holding_time"] = "2h"

	case "波动率策略":
		params["volatility_window"] = 30
		params["volatility_threshold"] = 1.5
		params["position_sizing"] = "volatility_adjusted"
		params["rebalance_freq"] = "6h"
	}

	return params
}

// 生成推荐理由
func (r *AdvancedStrategyRecommender) generateRationale(strategyName string, marketEnv *MarketEnvironment, marketFit float64) string {
	if marketFit < 0.7 {
		return fmt.Sprintf("在%s环境下，%s策略适应性较弱，不推荐作为主要策略", marketEnv.Regime, strategyName)
	}

	switch strategyName {
	case "均值回归策略":
		return fmt.Sprintf("在%s环境下，价格频繁偏离均值后回归，%s策略能有效捕捉这些机会", marketEnv.Regime, strategyName)
	case "网格交易策略":
		return fmt.Sprintf("%s策略在%s环境下表现稳定，能在区间震荡中持续获利", strategyName, marketEnv.Regime)
	case "统计套利策略":
		return fmt.Sprintf("利用相关资产间的价差机会，%s环境有利于发现套利机会", marketEnv.Regime)
	default:
		return fmt.Sprintf("%s策略在当前市场环境下具有较好的适应性", strategyName)
	}
}

// 构建投资组合配置
func (r *AdvancedStrategyRecommender) buildPortfolioConfig(recommendations []StrategyRecommendation, marketEnv *MarketEnvironment) *PortfolioConfig {
	config := &PortfolioConfig{
		MarketEnvironment: *marketEnv,
		RiskParityWeights: make(map[string]float64),
	}

	// 选择主要策略 (前2名)
	for i, rec := range recommendations {
		if i < 2 {
			config.PrimaryStrategies = append(config.PrimaryStrategies, rec)
		} else if i < 5 {
			config.SecondaryStrategies = append(config.SecondaryStrategies, rec)
		}
	}

	// 计算多样化评分
	strategyCount := len(config.PrimaryStrategies) + len(config.SecondaryStrategies)
	config.DiversificationScore = float64(strategyCount) / 10.0 * 100

	// 设置风险平价权重
	totalWeight := 0.0
	for _, rec := range config.PrimaryStrategies {
		weight := rec.CompositeScore * 1.5 // 主要策略权重更高
		config.RiskParityWeights[rec.Strategy.Name] = weight
		totalWeight += weight
	}

	for _, rec := range config.SecondaryStrategies {
		weight := rec.CompositeScore * 0.8 // 辅助策略权重较低
		config.RiskParityWeights[rec.Strategy.Name] = weight
		totalWeight += weight
	}

	// 归一化权重
	for strategy, weight := range config.RiskParityWeights {
		config.RiskParityWeights[strategy] = weight / totalWeight
	}

	// 根据市场环境设置参数
	switch marketEnv.Regime {
	case "高波动震荡市":
		config.MaxDrawdownLimit = 0.15
		config.RebalancingFreq = "每日"
	case "震荡市":
		config.MaxDrawdownLimit = 0.20
		config.RebalancingFreq = "每周"
	case "强势上涨趋势市", "强势下跌趋势市":
		config.MaxDrawdownLimit = 0.12
		config.RebalancingFreq = "每日"
	default:
		config.MaxDrawdownLimit = 0.18
		config.RebalancingFreq = "每周"
	}

	config.StopLossRules = []string{
		"单策略回撤超过5%时减仓20%",
		"组合回撤超过10%时暂停新开仓位",
		"连续3次亏损自动减仓50%",
		"市场极端事件触发时全部清仓",
	}

	return config
}

// 生成风险管理框架
func (r *AdvancedStrategyRecommender) generateRiskManagementFramework(portfolio *PortfolioConfig) *RiskManagementFramework {
	framework := &RiskManagementFramework{
		PortfolioMaxDrawdown: portfolio.MaxDrawdownLimit,
		StrategyMaxDrawdown:  portfolio.MaxDrawdownLimit * 0.6,
		DailyLossLimit:       portfolio.MaxDrawdownLimit * 0.1,
		PositionSizingRules:  []string{},
		RiskMonitoringFreq:   "实时",
		StressTestScenarios:  []string{},
	}

	// 基于市场环境设置仓位管理规则
	switch portfolio.MarketEnvironment.Regime {
	case "高波动震荡市":
		framework.PositionSizingRules = append(framework.PositionSizingRules,
			"单策略最大仓位不超过总资金的15%",
			"波动率>8%时自动减仓30%",
			"使用凯利公式的保守版本")
	case "震荡市":
		framework.PositionSizingRules = append(framework.PositionSizingRules,
			"单策略最大仓位不超过总资金的20%",
			"根据胜率和赔率动态调整仓位",
			"实施等权重风险平价")
	default:
		framework.PositionSizingRules = append(framework.PositionSizingRules,
			"单策略最大仓位不超过总资金的25%",
			"使用固定分数仓位管理",
			"定期进行风险再平衡")
	}

	framework.StressTestScenarios = []string{
		"价格波动率突然增加200%",
		"主要币种价格下跌30%",
		"市场出现极端事件",
		"流动性突然枯竭",
		"相关性系数急剧变化",
	}

	return framework
}

// 生成执行计划
func (r *AdvancedStrategyRecommender) generateExecutionPlan(portfolio *PortfolioConfig) *ExecutionPlan {
	plan := &ExecutionPlan{
		Phase1Actions:     []string{},
		Phase2Actions:     []string{},
		Phase3Actions:     []string{},
		MonitoringKPIs:    []string{},
		ReviewFrequency:   "每周",
		ScalingConditions: []string{},
	}

	// 第一阶段：准备和测试
	plan.Phase1Actions = append(plan.Phase1Actions,
		"开展小规模策略回测验证",
		"设置风险管理系统和监控",
		"准备资金和交易权限",
		"建立策略执行日志系统",
		fmt.Sprintf("按%s频率进行再平衡", portfolio.RebalancingFreq))

	// 第二阶段：逐步执行
	plan.Phase2Actions = append(plan.Phase2Actions,
		"从小仓位开始执行主要策略",
		"监控策略表现和市场环境变化",
		"根据表现调整策略参数",
		"逐步增加辅助策略权重",
		"建立应急响应机制")

	// 第三阶段：全量执行
	plan.Phase3Actions = append(plan.Phase3Actions,
		"达到目标权重分配",
		"实施完全自动化执行",
		"定期进行策略优化",
		"监控整体投资组合表现",
		"根据市场变化动态调整")

	// 关键绩效指标
	plan.MonitoringKPIs = []string{
		"组合夏普比率 > 1.5",
		"最大回撤 < 15%",
		"月化收益 > 2%",
		"胜率 > 55%",
		"策略相关性 < 0.7",
	}

	// 扩容条件
	plan.ScalingConditions = []string{
		"策略连续3个月表现良好",
		"风险指标控制在目标范围内",
		"市场环境保持稳定",
		"资金管理能力得到验证",
	}

	return plan
}

// 显示函数
func (r *AdvancedStrategyRecommender) displayMarketEnvironment(env *MarketEnvironment) {
	fmt.Printf("市场环境: %s\n", env.Regime)
	fmt.Printf("波动率水平: %.2f%%\n", env.Volatility)
	fmt.Printf("趋势强度: %.1f%%\n", env.TrendStrength*100)
	fmt.Printf("价格-成交量相关性: %.3f\n", env.Correlation)
	fmt.Printf("判断置信度: %.1f%%\n", env.Confidence*100)
	fmt.Printf("环境描述: %s\n", env.Description)
}

func (r *AdvancedStrategyRecommender) displayStrategyRecommendations(recommendations []StrategyRecommendation, marketEnv *MarketEnvironment) {
	fmt.Printf("基于%s环境的策略推荐:\n\n", marketEnv.Regime)

	fmt.Println("┌─────────────────────┬──────────┬──────────┬──────────┬──────────┬──────────┐")
	fmt.Println("│ 策略名称           │ 市场适应 │ 风险评分 │ 流动性  │ 综合评分 │ 权重分配 │")
	fmt.Println("├─────────────────────┼──────────┼──────────┼──────────┼──────────┼──────────┤")

	for _, rec := range recommendations {
		if rec.Priority > 0 {
			fmt.Printf("│ %-18s │ %8.1f │ %8.1f │ %8.1f │ %8.1f │ %7.1f%% │\n",
				rec.Strategy.Name,
				rec.MarketFit,
				rec.RiskScore,
				rec.LiquidityFit,
				rec.CompositeScore,
				rec.Allocation*100)
		}
	}
	fmt.Println("└─────────────────────┴──────────┴──────────┴──────────┴──────────┴──────────┘")

	fmt.Println("\n推荐策略详情:")
	for _, rec := range recommendations {
		if rec.Priority > 0 && rec.Priority <= 3 {
			fmt.Printf("\n%d. %s (优先级: %d)\n", rec.Priority, rec.Strategy.Name, rec.Priority)
			fmt.Printf("   理由: %s\n", rec.Rationale)
			fmt.Printf("   预期表现: 胜率%.0f%%, 年化收益%.1f%%, 最大回撤%.1f%%\n",
				rec.Strategy.WinRate*100, rec.Strategy.AvgReturn*12, rec.Strategy.MaxDrawdown)
			if len(rec.Parameters) > 0 {
				fmt.Printf("   关键参数: ")
				for k, v := range rec.Parameters {
					fmt.Printf("%s=%v ", k, v)
				}
				fmt.Println()
			}
		}
	}
}

func (r *AdvancedStrategyRecommender) displayPortfolioConfig(config *PortfolioConfig) {
	fmt.Printf("投资组合配置 (针对%s环境)\n", config.MarketEnvironment.Regime)
	fmt.Printf("多样化评分: %.1f%%\n", config.DiversificationScore)
	fmt.Printf("最大回撤限制: %.0f%%\n", config.MaxDrawdownLimit*100)
	fmt.Printf("再平衡频率: %s\n\n", config.RebalancingFreq)

	fmt.Println("权重分配:")
	for strategy, weight := range config.RiskParityWeights {
		if weight > 0.01 {
			fmt.Printf("  %s: %.1f%%\n", strategy, weight*100)
		}
	}

	fmt.Println("\n止损规则:")
	for _, rule := range config.StopLossRules {
		fmt.Printf("  • %s\n", rule)
	}
}

func (r *AdvancedStrategyRecommender) displayRiskManagementFramework(framework *RiskManagementFramework) {
	fmt.Printf("风险管理框架\n")
	fmt.Printf("组合最大回撤: %.0f%%\n", framework.PortfolioMaxDrawdown*100)
	fmt.Printf("策略最大回撤: %.0f%%\n", framework.StrategyMaxDrawdown*100)
	fmt.Printf("每日亏损限制: %.0f%%\n", framework.DailyLossLimit*100)
	fmt.Printf("风险监控频率: %s\n\n", framework.RiskMonitoringFreq)

	fmt.Println("仓位管理规则:")
	for _, rule := range framework.PositionSizingRules {
		fmt.Printf("  • %s\n", rule)
	}

	fmt.Println("\n压力测试场景:")
	for _, scenario := range framework.StressTestScenarios {
		fmt.Printf("  • %s\n", scenario)
	}
}

func (r *AdvancedStrategyRecommender) displayExecutionPlan(plan *ExecutionPlan) {
	fmt.Println("执行计划分三个阶段:")

	fmt.Println("\n第一阶段 - 准备测试:")
	for _, action := range plan.Phase1Actions {
		fmt.Printf("  • %s\n", action)
	}

	fmt.Println("\n第二阶段 - 逐步执行:")
	for _, action := range plan.Phase2Actions {
		fmt.Printf("  • %s\n", action)
	}

	fmt.Println("\n第三阶段 - 全量执行:")
	for _, action := range plan.Phase3Actions {
		fmt.Printf("  • %s\n", action)
	}

	fmt.Println("\n关键绩效指标:")
	for _, kpi := range plan.MonitoringKPIs {
		fmt.Printf("  • %s\n", kpi)
	}

	fmt.Println("\n扩容条件:")
	for _, condition := range plan.ScalingConditions {
		fmt.Printf("  • %s\n", condition)
	}
}

// 数据结构定义
type RiskManagementFramework struct {
	PortfolioMaxDrawdown float64
	StrategyMaxDrawdown  float64
	DailyLossLimit       float64
	PositionSizingRules  []string
	RiskMonitoringFreq   string
	StressTestScenarios  []string
}

type ExecutionPlan struct {
	Phase1Actions     []string
	Phase2Actions     []string
	Phase3Actions     []string
	MonitoringKPIs    []string
	ReviewFrequency   string
	ScalingConditions []string
}
