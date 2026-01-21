package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"

	_ "github.com/go-sql-driver/mysql"
)

// 综合策略分析系统
type ComprehensiveStrategyAnalyzer struct {
	db *sql.DB
}

// 市场环境分析结果
type MarketEnvironmentAnalysis struct {
	Regime         string
	Volatility     float64
	TrendStrength  float64
	BullRatio      float64
	BearRatio      float64
	NeutralRatio   float64
	AverageChange  float64
	Confidence     float64
	Description    string
}

// 策略评估结果
type StrategyEvaluation struct {
	Name            string
	Type            string
	MarketFit       float64
	RiskLevel       string
	ExpectedReturn  string
	WinRate         float64
	MaxDrawdown     float64
	TimeHorizon     string
	CapitalReq      string
	Complexity      string
	BestConditions  string
	CurrentSuitability float64
	Parameters      map[string]interface{}
	Rationale       string
}

func main() {
	fmt.Println("🤖 综合策略分析和推荐系统")
	fmt.Println("============================")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	analyzer := &ComprehensiveStrategyAnalyzer{db: db}

	// 1. 分析当前市场环境
	fmt.Println("\n📊 第一步: 市场环境深度分析")
	marketEnv, err := analyzer.analyzeMarketEnvironment()
	if err != nil {
		log.Fatal("市场环境分析失败:", err)
	}
	analyzer.displayMarketEnvironment(marketEnv)

	// 2. 评估所有常见策略
	fmt.Println("\n🎯 第二步: 策略适用性评估")
	strategies := analyzer.initializeAllStrategies()
	evaluations := analyzer.evaluateAllStrategies(strategies, marketEnv)
	analyzer.displayStrategyEvaluations(evaluations, marketEnv)

	// 3. 生成投资建议
	fmt.Println("\n💼 第三步: 投资组合建议")
	portfolio := analyzer.generatePortfolioRecommendation(evaluations, marketEnv)
	analyzer.displayPortfolioRecommendation(portfolio)

	// 4. 风险管理框架
	fmt.Println("\n⚠️ 第四步: 风险管理建议")
	riskFramework := analyzer.generateRiskManagementFramework(portfolio)
	analyzer.displayRiskManagementFramework(riskFramework)

	fmt.Println("\n🎉 策略分析完成！")
}

// 分析市场环境
func (csa *ComprehensiveStrategyAnalyzer) analyzeMarketEnvironment() (*MarketEnvironmentAnalysis, error) {
	env := &MarketEnvironmentAnalysis{}

	// 查询24小时市场数据
	query := `
		SELECT
			AVG(price_change_percent) as avg_change,
			STDDEV(price_change_percent) as change_volatility,
			COUNT(CASE WHEN price_change_percent > 5 THEN 1 END) / COUNT(*) as bull_ratio,
			COUNT(CASE WHEN price_change_percent < -5 THEN 1 END) / COUNT(*) as bear_ratio,
			COUNT(CASE WHEN ABS(price_change_percent) <= 5 THEN 1 END) / COUNT(*) as neutral_ratio,
			AVG((high_price - low_price) / low_price * 100) as avg_volatility
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000`

	var avgChange, changeVolatility, bullRatio, bearRatio, neutralRatio, avgVolatility float64
	err := csa.db.QueryRow(query).Scan(&avgChange, &changeVolatility, &bullRatio, &bearRatio, &neutralRatio, &avgVolatility)
	if err != nil {
		return nil, fmt.Errorf("市场数据查询失败: %v", err)
	}

	env.AverageChange = avgChange
	env.Volatility = avgVolatility
	env.BullRatio = bullRatio
	env.BearRatio = bearRatio
	env.NeutralRatio = neutralRatio

	// 计算趋势强度
	env.TrendStrength = bullRatio + bearRatio

	// 判断市场环境
	if avgVolatility > 8 && neutralRatio > 0.7 {
		env.Regime = "高波动震荡市"
		env.Confidence = 0.85
		env.Description = "价格剧烈波动但无明确方向，适合均值回归和波动率策略"
	} else if avgVolatility > 6 && neutralRatio > 0.6 {
		env.Regime = "震荡市"
		env.Confidence = 0.80
		env.Description = "价格在区间内震荡，适合网格交易和统计套利"
	} else if env.TrendStrength > 0.4 {
		if avgChange > 0 {
			env.Regime = "强势上涨趋势市"
			env.Confidence = 0.75
			env.Description = "明显上涨趋势，适合趋势跟随和动量策略"
		} else {
			env.Regime = "强势下跌趋势市"
			env.Confidence = 0.75
			env.Description = "明显下跌趋势，适合做空和对冲策略"
		}
	} else if avgVolatility < 4 && neutralRatio > 0.8 {
		env.Regime = "低波动整理市"
		env.Confidence = 0.70
		env.Description = "市场平静，适合网格交易和稳健策略"
	} else {
		env.Regime = "混合市场"
		env.Confidence = 0.60
		env.Description = "复杂多变的市场环境，需要灵活策略组合"
	}

	return env, nil
}

// 初始化所有常见策略
func (csa *ComprehensiveStrategyAnalyzer) initializeAllStrategies() []StrategyEvaluation {
	strategies := []StrategyEvaluation{

		// 趋势类策略
		{
			Name:           "趋势跟随策略",
			Type:           "trend_following",
			RiskLevel:      "高",
			ExpectedReturn: "15-30%每年",
			WinRate:        0.45,
			MaxDrawdown:    25.0,
			TimeHorizon:    "中长期",
			CapitalReq:     "中等",
			Complexity:     "中等",
			BestConditions: "强趋势市场，较高波动率",
			Parameters: map[string]interface{}{
				"ma_period":     20,
				"confirmation":  2,
				"stop_loss":     0.05,
				"take_profit":   0.20,
			},
		},
		{
			Name:           "动量策略",
			Type:           "momentum",
			RiskLevel:      "高",
			ExpectedReturn: "20-40%每年",
			WinRate:        0.40,
			MaxDrawdown:    30.0,
			TimeHorizon:    "短期-中期",
			CapitalReq:     "中等",
			Complexity:     "高",
			BestConditions: "强动量信号，快速市场变动",
			Parameters: map[string]interface{}{
				"lookback":      10,
				"threshold":     0.05,
				"holding_days":  5,
			},
		},

		// 均值回归类策略
		{
			Name:           "均值回归策略",
			Type:           "mean_reversion",
			RiskLevel:      "中等",
			ExpectedReturn: "8-15%每年",
			WinRate:        0.55,
			MaxDrawdown:    15.0,
			TimeHorizon:    "短期",
			CapitalReq:     "低",
			Complexity:     "中等",
			BestConditions: "震荡市场，价格频繁偏离均值",
			Parameters: map[string]interface{}{
				"lookback":       20,
				"entry_zscore":   2.0,
				"exit_zscore":    0.5,
				"max_holding":    "4h",
			},
		},
		{
			Name:           "统计套利策略",
			Type:           "statistical_arbitrage",
			RiskLevel:      "中等",
			ExpectedReturn: "10-20%每年",
			WinRate:        0.60,
			MaxDrawdown:    12.0,
			TimeHorizon:    "短期",
			CapitalReq:     "中等",
			Complexity:     "高",
			BestConditions: "相关资产间价格偏离均值",
			Parameters: map[string]interface{}{
				"correlation":    0.8,
				"spread_threshold": 1.5,
				"hedge_ratio":    1.0,
			},
		},

		// 网格类策略
		{
			Name:           "网格交易策略",
			Type:           "grid_trading",
			RiskLevel:      "低",
			ExpectedReturn: "5-12%每年",
			WinRate:        0.70,
			MaxDrawdown:    8.0,
			TimeHorizon:    "中长期",
			CapitalReq:     "中等",
			Complexity:     "低",
			BestConditions: "震荡市场，价格区间明确",
			Parameters: map[string]interface{}{
				"grid_levels":    10,
				"grid_spacing":   0.01,
				"min_volume":     10000,
				"rebalance_freq": "1h",
			},
		},

		// 波动率类策略
		{
			Name:           "波动率策略",
			Type:           "volatility",
			RiskLevel:      "高",
			ExpectedReturn: "15-25%每年",
			WinRate:        0.50,
			MaxDrawdown:    20.0,
			TimeHorizon:    "中期",
			CapitalReq:     "高",
			Complexity:     "高",
			BestConditions: "波动率快速变化，高波动环境",
			Parameters: map[string]interface{}{
				"vol_window":     30,
				"vol_threshold":  1.5,
				"position_sizing": "volatility_adjusted",
			},
		},

		// 对冲类策略
		{
			Name:           "多空对冲策略",
			Type:           "hedge",
			RiskLevel:      "中等",
			ExpectedReturn: "6-12%每年",
			WinRate:        0.55,
			MaxDrawdown:    10.0,
			TimeHorizon:    "中长期",
			CapitalReq:     "高",
			Complexity:     "高",
			BestConditions: "多空力量相对平衡的市场",
			Parameters: map[string]interface{}{
				"long_symbols":   []string{"BTC", "ETH"},
				"short_symbols":  []string{"ALTS"},
				"rebalance_freq": "daily",
			},
		},

		// 反转类策略
		{
			Name:           "反转策略",
			Type:           "reversal",
			RiskLevel:      "高",
			ExpectedReturn: "12-20%每年",
			WinRate:        0.45,
			MaxDrawdown:    25.0,
			TimeHorizon:    "短期",
			CapitalReq:     "低",
			Complexity:     "中等",
			BestConditions: "超买超卖信号明显，震荡市",
			Parameters: map[string]interface{}{
				"rsi_overbought": 70,
				"rsi_oversold":   30,
				"confirmation":   2,
			},
		},

		// 突破类策略
		{
			Name:           "突破策略",
			Type:           "breakout",
			RiskLevel:      "中等",
			ExpectedReturn: "10-18%每年",
			WinRate:        0.50,
			MaxDrawdown:    18.0,
			TimeHorizon:    "短期-中期",
			CapitalReq:     "中等",
			Complexity:     "中等",
			BestConditions: "重要支撑阻力位突破",
			Parameters: map[string]interface{}{
				"lookback":       20,
				"breakout_pct":   0.03,
				"volume_confirm": true,
			},
		},

		// 做空策略
		{
			Name:           "做空策略",
			Type:           "short_selling",
			RiskLevel:      "极高",
			ExpectedReturn: "15-25%每年",
			WinRate:        0.40,
			MaxDrawdown:    35.0,
			TimeHorizon:    "中期",
			CapitalReq:     "高",
			Complexity:     "高",
			BestConditions: "熊市确认，风险偏好极低",
			Parameters: map[string]interface{}{
				"bear_signal":    "multiple_indicators",
				"stop_loss":      0.10,
				"position_size":  0.2,
			},
		},

		// 套利策略
		{
			Name:           "三角套利策略",
			Type:           "triangular_arbitrage",
			RiskLevel:      "低",
			ExpectedReturn: "3-8%每年",
			WinRate:        0.85,
			MaxDrawdown:    2.0,
			TimeHorizon:    "超短期",
			CapitalReq:     "高",
			Complexity:     "高",
			BestConditions: "市场效率低下，存在价格不一致",
			Parameters: map[string]interface{}{
				"min_profit":     0.001,
				"max_slippage":   0.0005,
				"execution_time": "immediate",
			},
		},
	}

	return strategies
}

// 评估所有策略
func (csa *ComprehensiveStrategyAnalyzer) evaluateAllStrategies(strategies []StrategyEvaluation, marketEnv *MarketEnvironmentAnalysis) []StrategyEvaluation {
	for i := range strategies {
		strategies[i].MarketFit = csa.calculateMarketFit(strategies[i], marketEnv)
		strategies[i].CurrentSuitability = strategies[i].MarketFit
		strategies[i].Rationale = csa.generateStrategyRationale(strategies[i], marketEnv)
	}

	// 按适用性排序
	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].CurrentSuitability > strategies[j].CurrentSuitability
	})

	return strategies
}

// 计算策略的市场适应性
func (csa *ComprehensiveStrategyAnalyzer) calculateMarketFit(strategy StrategyEvaluation, marketEnv *MarketEnvironmentAnalysis) float64 {
	baseScore := 0.5

	switch marketEnv.Regime {
	case "高波动震荡市":
		switch strategy.Type {
		case "mean_reversion":
			baseScore = 1.4
		case "volatility":
			baseScore = 1.3
		case "grid_trading":
			baseScore = 1.1
		case "statistical_arbitrage":
			baseScore = 1.0
		case "reversal":
			baseScore = 0.9
		case "breakout":
			baseScore = 0.8
		case "momentum":
			baseScore = 0.7
		}

	case "震荡市":
		switch strategy.Type {
		case "mean_reversion":
			baseScore = 1.5
		case "grid_trading":
			baseScore = 1.3
		case "statistical_arbitrage":
			baseScore = 1.2
		case "reversal":
			baseScore = 1.1
		case "breakout":
			baseScore = 0.9
		case "hedge":
			baseScore = 0.8
		}

	case "强势上涨趋势市":
		switch strategy.Type {
		case "trend_following":
			baseScore = 1.5
		case "momentum":
			baseScore = 1.4
		case "breakout":
			baseScore = 1.2
		case "hedge":
			baseScore = 0.9
		case "volatility":
			baseScore = 0.8
		}

	case "强势下跌趋势市":
		switch strategy.Type {
		case "short_selling":
			baseScore = 1.6
		case "hedge":
			baseScore = 1.3
		case "trend_following":
			baseScore = 1.1
		case "volatility":
			baseScore = 0.9
		}

	case "低波动整理市":
		switch strategy.Type {
		case "grid_trading":
			baseScore = 1.4
		case "statistical_arbitrage":
			baseScore = 1.2
		case "mean_reversion":
			baseScore = 1.1
		case "triangular_arbitrage":
			baseScore = 1.0
		case "hedge":
			baseScore = 0.9
		}

	default: // 混合市场
		switch strategy.Type {
		case "hedge":
			baseScore = 1.3
		case "grid_trading":
			baseScore = 1.1
		case "trend_following":
			baseScore = 1.0
		case "mean_reversion":
			baseScore = 0.9
		}
	}

	// 基于波动率调整
	if marketEnv.Volatility > 8 && strategy.Type == "volatility" {
		baseScore *= 1.2
	}
	if marketEnv.Volatility < 3 && strategy.Type == "grid_trading" {
		baseScore *= 1.1
	}

	// 限制在合理范围内
	if baseScore > 1.5 {
		baseScore = 1.5
	} else if baseScore < 0.1 {
		baseScore = 0.1
	}

	return baseScore
}

// 生成策略理由
func (csa *ComprehensiveStrategyAnalyzer) generateStrategyRationale(strategy StrategyEvaluation, marketEnv *MarketEnvironmentAnalysis) string {
	if strategy.CurrentSuitability < 0.7 {
		return fmt.Sprintf("在%s环境下，%s策略适应性较弱，不推荐作为主要策略", marketEnv.Regime, strategy.Name)
	}

	switch strategy.Type {
	case "mean_reversion":
		return fmt.Sprintf("在%s环境下，价格频繁偏离均值后回归，%s能有效捕捉这些机会", marketEnv.Regime, strategy.Name)
	case "grid_trading":
		return fmt.Sprintf("%s在%s环境下表现稳定，能在区间震荡中持续获利", strategy.Name, marketEnv.Regime)
	case "trend_following":
		return fmt.Sprintf("当前市场显示%s特征，%s策略能跟随主流趋势", marketEnv.Regime, strategy.Name)
	case "volatility":
		return fmt.Sprintf("波动率水平为%.1f%%，%s策略在高波动环境下表现优异", marketEnv.Volatility, strategy.Name)
	default:
		return fmt.Sprintf("%s策略在当前市场环境下具有较好的适应性", strategy.Name)
	}
}

// 生成投资组合建议
func (csa *ComprehensiveStrategyAnalyzer) generatePortfolioRecommendation(evaluations []StrategyEvaluation, marketEnv *MarketEnvironmentAnalysis) *PortfolioRecommendation {
	rec := &PortfolioRecommendation{
		MarketEnvironment: *marketEnv,
		PrimaryStrategies: []StrategyAllocation{},
		SecondaryStrategies: []StrategyAllocation{},
		RiskProfile:       csa.determineRiskProfile(marketEnv),
		TotalAllocation:   100.0,
		DiversificationScore: 0.0,
		RebalancingFrequency: csa.getRebalancingFrequency(marketEnv),
	}

	// 选择主要策略（前3名）
	for i, eval := range evaluations {
		if i < 3 && eval.CurrentSuitability > 0.8 {
			allocation := StrategyAllocation{
				Strategy:  eval,
				Weight:    csa.calculateStrategyWeight(eval, true),
				MinWeight: csa.getMinWeight(eval.RiskLevel),
				MaxWeight: csa.getMaxWeight(eval.RiskLevel),
			}
			rec.PrimaryStrategies = append(rec.PrimaryStrategies, allocation)
		} else if i < 6 && eval.CurrentSuitability > 0.6 {
			allocation := StrategyAllocation{
				Strategy:  eval,
				Weight:    csa.calculateStrategyWeight(eval, false),
				MinWeight: csa.getMinWeight(eval.RiskLevel),
				MaxWeight: csa.getMaxWeight(eval.RiskLevel),
			}
			rec.SecondaryStrategies = append(rec.SecondaryStrategies, allocation)
		}
	}

	// 计算多样化评分
	strategyCount := len(rec.PrimaryStrategies) + len(rec.SecondaryStrategies)
	rec.DiversificationScore = float64(strategyCount) / 10.0 * 100

	return rec
}

// 确定风险偏好
func (csa *ComprehensiveStrategyAnalyzer) determineRiskProfile(marketEnv *MarketEnvironmentAnalysis) string {
	switch marketEnv.Regime {
	case "高波动震荡市":
		return "中等风险偏好"
	case "强势上涨趋势市":
		return "积极风险偏好"
	case "强势下跌趋势市":
		return "保守风险偏好"
	case "低波动整理市":
		return "保守风险偏好"
	default:
		return "平衡风险偏好"
	}
}

// 获取再平衡频率
func (csa *ComprehensiveStrategyAnalyzer) getRebalancingFrequency(marketEnv *MarketEnvironmentAnalysis) string {
	switch marketEnv.Regime {
	case "高波动震荡市", "强势上涨趋势市", "强势下跌趋势市":
		return "每日"
	case "震荡市", "混合市场":
		return "每周"
	default:
		return "每月"
	}
}

// 计算策略权重
func (csa *ComprehensiveStrategyAnalyzer) calculateStrategyWeight(eval StrategyEvaluation, isPrimary bool) float64 {
	baseWeight := eval.CurrentSuitability * 10

	if isPrimary {
		baseWeight *= 1.5
	} else {
		baseWeight *= 0.8
	}

	// 根据风险等级调整
	switch eval.RiskLevel {
	case "低":
		baseWeight *= 1.2
	case "极高":
		baseWeight *= 0.7
	}

	return baseWeight
}

// 获取最小权重
func (csa *ComprehensiveStrategyAnalyzer) getMinWeight(riskLevel string) float64 {
	switch riskLevel {
	case "低":
		return 5.0
	case "中等":
		return 3.0
	case "高":
		return 2.0
	case "极高":
		return 1.0
	default:
		return 2.0
	}
}

// 获取最大权重
func (csa *ComprehensiveStrategyAnalyzer) getMaxWeight(riskLevel string) float64 {
	switch riskLevel {
	case "低":
		return 25.0
	case "中等":
		return 20.0
	case "高":
		return 15.0
	case "极高":
		return 10.0
	default:
		return 15.0
	}
}

// 生成风险管理框架
func (csa *ComprehensiveStrategyAnalyzer) generateRiskManagementFramework(portfolio *PortfolioRecommendation) *RiskManagementFramework {
	framework := &RiskManagementFramework{
		MaxDrawdownLimit:     csa.getMaxDrawdownLimit(portfolio.MarketEnvironment.Regime),
		DailyLossLimit:       csa.getDailyLossLimit(portfolio.MarketEnvironment.Regime),
		StrategyLimits:       make(map[string]float64),
		StopLossRules:        csa.getStopLossRules(portfolio.MarketEnvironment.Regime),
		RiskMonitoringFreq:   "实时",
		StressTestScenarios:   csa.getStressTestScenarios(),
		PositionSizingMethod: csa.getPositionSizingMethod(portfolio.RiskProfile),
	}

	// 设置策略限制
	for _, alloc := range portfolio.PrimaryStrategies {
		framework.StrategyLimits[alloc.Strategy.Name] = alloc.MaxWeight
	}
	for _, alloc := range portfolio.SecondaryStrategies {
		framework.StrategyLimits[alloc.Strategy.Name] = alloc.MaxWeight
	}

	return framework
}

// 获取最大回撤限制
func (csa *ComprehensiveStrategyAnalyzer) getMaxDrawdownLimit(regime string) float64 {
	switch regime {
	case "高波动震荡市", "强势上涨趋势市", "强势下跌趋势市":
		return 0.15
	case "震荡市", "混合市场":
		return 0.20
	default:
		return 0.18
	}
}

// 获取每日亏损限制
func (csa *ComprehensiveStrategyAnalyzer) getDailyLossLimit(regime string) float64 {
	switch regime {
	case "高波动震荡市":
		return 0.03
	case "强势上涨趋势市", "强势下跌趋势市":
		return 0.02
	default:
		return 0.025
	}
}

// 获取止损规则
func (csa *ComprehensiveStrategyAnalyzer) getStopLossRules(regime string) []string {
	baseRules := []string{
		"单策略回撤超过5%时减仓20%",
		"组合回撤超过10%时暂停新开仓位",
		"连续3次亏损自动减仓50%",
	}

	switch regime {
	case "高波动震荡市":
		baseRules = append(baseRules, "波动率>8%时自动减仓30%")
	case "强势下跌趋势市":
		baseRules = append(baseRules, "市场极端事件触发时全部清仓")
	case "低波动整理市":
		baseRules = append(baseRules, "突破历史低点时暂停交易")
	}

	return baseRules
}

// 获取压力测试场景
func (csa *ComprehensiveStrategyAnalyzer) getStressTestScenarios() []string {
	return []string{
		"价格波动率突然增加200%",
		"主要币种价格下跌30%",
		"市场出现极端事件",
		"流动性突然枯竭",
		"相关性系数急剧变化",
		"交易量锐减50%",
		"网络中断1小时",
		"交易所临时下架主要币种",
	}
}

// 获取仓位管理方法
func (csa *ComprehensiveStrategyAnalyzer) getPositionSizingMethod(riskProfile string) string {
	switch riskProfile {
	case "保守风险偏好":
		return "固定百分比仓位管理"
	case "中等风险偏好":
		return "凯利公式的保守版本"
	case "积极风险偏好":
		return "波动率调整仓位管理"
	default:
		return "等权重风险平价"
	}
}

// 显示函数
func (csa *ComprehensiveStrategyAnalyzer) displayMarketEnvironment(env *MarketEnvironmentAnalysis) {
	fmt.Printf("市场环境: %s\n", env.Regime)
	fmt.Printf("平均波动率: %.2f%%\n", env.Volatility)
	fmt.Printf("平均涨跌幅: %.2f%%\n", env.AverageChange)
	fmt.Printf("趋势强度: %.1f%%\n", env.TrendStrength*100)
	fmt.Printf("多头占比: %.1f%%\n", env.BullRatio*100)
	fmt.Printf("空头占比: %.1f%%\n", env.BearRatio*100)
	fmt.Printf("中性占比: %.1f%%\n", env.NeutralRatio*100)
	fmt.Printf("判断置信度: %.1f%%\n", env.Confidence*100)
	fmt.Printf("环境描述: %s\n", env.Description)
}

func (csa *ComprehensiveStrategyAnalyzer) displayStrategyEvaluations(evaluations []StrategyEvaluation, marketEnv *MarketEnvironmentAnalysis) {
	fmt.Printf("基于%s环境的策略评估结果:\n\n", marketEnv.Regime)

	fmt.Println("┌─────────────────────┬──────────┬──────────┬──────────┬──────────┬──────────┐")
	fmt.Println("│ 策略名称           │ 市场适应 │ 风险等级 │ 预期收益 │ 胜率     │ 最大回撤 │")
	fmt.Println("├─────────────────────┼──────────┼──────────┼──────────┼──────────┼──────────┤")

	for _, eval := range evaluations {
		if eval.CurrentSuitability > 0.6 {
			fmt.Printf("│ %-18s │ %8.1f │ %-8s │ %-8s │ %6.1f%% │ %6.1f%% │\n",
				eval.Name,
				eval.CurrentSuitability,
				eval.RiskLevel,
				eval.ExpectedReturn,
				eval.WinRate*100,
				eval.MaxDrawdown)
		}
	}
	fmt.Println("└─────────────────────┴──────────┴──────────┴──────────┴──────────┴──────────┘")

	fmt.Println("\n📋 详细策略分析:")
	for i, eval := range evaluations {
		if i >= 8 { // 只显示前8个
			break
		}
		fmt.Printf("\n%d. %s (适用性: %.1f/1.0)\n", i+1, eval.Name, eval.CurrentSuitability)
		fmt.Printf("   类型: %s | 风险: %s | 时间周期: %s\n", eval.Type, eval.RiskLevel, eval.TimeHorizon)
		fmt.Printf("   预期收益: %s | 胜率: %.0f%% | 最大回撤: %.0f%%\n", eval.ExpectedReturn, eval.WinRate*100, eval.MaxDrawdown)
		fmt.Printf("   推荐理由: %s\n", eval.Rationale)
		if len(eval.Parameters) > 0 {
			fmt.Printf("   关键参数: ")
			for k, v := range eval.Parameters {
				fmt.Printf("%s=%v ", k, v)
			}
			fmt.Println()
		}
	}
}

// 数据结构定义
type PortfolioRecommendation struct {
	MarketEnvironment    MarketEnvironmentAnalysis
	PrimaryStrategies    []StrategyAllocation
	SecondaryStrategies  []StrategyAllocation
	RiskProfile          string
	TotalAllocation      float64
	DiversificationScore float64
	RebalancingFrequency string
}

type StrategyAllocation struct {
	Strategy  StrategyEvaluation
	Weight    float64
	MinWeight float64
	MaxWeight float64
}

type RiskManagementFramework struct {
	MaxDrawdownLimit     float64
	DailyLossLimit       float64
	StrategyLimits       map[string]float64
	StopLossRules        []string
	RiskMonitoringFreq   string
	StressTestScenarios   []string
	PositionSizingMethod string
}

func (csa *ComprehensiveStrategyAnalyzer) displayPortfolioRecommendation(portfolio *PortfolioRecommendation) {
	fmt.Printf("投资组合建议 (针对%s环境)\n", portfolio.MarketEnvironment.Regime)
	fmt.Printf("风险偏好: %s\n", portfolio.RiskProfile)
	fmt.Printf("多样化评分: %.1f%%\n", portfolio.DiversificationScore)
	fmt.Printf("再平衡频率: %s\n\n", portfolio.RebalancingFrequency)

	fmt.Println("主要策略配置:")
	for _, alloc := range portfolio.PrimaryStrategies {
		fmt.Printf("  %s: %.1f%% (%.1f%%-%.1f%%)\n",
			alloc.Strategy.Name, alloc.Weight, alloc.MinWeight, alloc.MaxWeight)
	}

	if len(portfolio.SecondaryStrategies) > 0 {
		fmt.Println("\n辅助策略配置:")
		for _, alloc := range portfolio.SecondaryStrategies {
			fmt.Printf("  %s: %.1f%% (%.1f%%-%.1f%%)\n",
				alloc.Strategy.Name, alloc.Weight, alloc.MinWeight, alloc.MaxWeight)
		}
	}
}

func (csa *ComprehensiveStrategyAnalyzer) displayRiskManagementFramework(framework *RiskManagementFramework) {
	fmt.Printf("风险管理框架\n")
	fmt.Printf("组合最大回撤: %.0f%%\n", framework.MaxDrawdownLimit*100)
	fmt.Printf("每日亏损限制: %.0f%%\n", framework.DailyLossLimit*100)
	fmt.Printf("风险监控频率: %s\n", framework.RiskMonitoringFreq)
	fmt.Printf("仓位管理方法: %s\n\n", framework.PositionSizingMethod)

	fmt.Println("策略权重限制:")
	for strategy, limit := range framework.StrategyLimits {
		fmt.Printf("  %s: 最大%.0f%%\n", strategy, limit)
	}

	fmt.Println("\n止损规则:")
	for _, rule := range framework.StopLossRules {
		fmt.Printf("  • %s\n", rule)
	}

	fmt.Println("\n压力测试场景:")
	for _, scenario := range framework.StressTestScenarios {
		fmt.Printf("  • %s\n", scenario)
	}
}