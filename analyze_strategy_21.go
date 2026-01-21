package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// 策略21详细分析系统
type Strategy21Analyzer struct {
	db *sql.DB
}

type TradingStrategy struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Type        string    `json:"type"`
	Description string    `json:"description"`
	Config      string    `json:"config"`
	Conditions  string    `json:"conditions"`
	Status      string    `json:"status"`
	Performance string    `json:"performance"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StrategyConfig struct {
	Parameters map[string]interface{} `json:"parameters"`
	Rules      []string               `json:"rules"`
}

type StrategyConditions struct {
	Enabled               bool                   `json:"enabled"`
	MarketType            string                 `json:"market_type"`
	MinVolume             float64                `json:"min_volume"`
	MaxPositions          int                    `json:"max_positions"`
	RiskLimits            map[string]interface{} `json:"risk_limits"`
	EntryConditions       []string               `json:"entry_conditions"`
	ExitConditions        []string               `json:"exit_conditions"`
	TimeFilters           map[string]interface{} `json:"time_filters"`
	SymbolFilters         []string               `json:"symbol_filters"`
}

type StrategyPerformance struct {
	TotalTrades     int                    `json:"total_trades"`
	WinRate         float64                `json:"win_rate"`
	AvgReturn       float64                `json:"avg_return"`
	MaxDrawdown     float64                `json:"max_drawdown"`
	SharpeRatio     float64                `json:"sharpe_ratio"`
	ProfitFactor    float64                `json:"profit_factor"`
	RecoveryFactor  float64                `json:"recovery_factor"`
	CalmarRatio     float64                `json:"calmar_ratio"`
	AvgHoldTime     string                 `json:"avg_hold_time"`
	BestTrade       float64                `json:"best_trade"`
	WorstTrade      float64                `json:"worst_trade"`
	MonthlyReturns  map[string]float64     `json:"monthly_returns"`
	DailyStats      map[string]interface{} `json:"daily_stats"`
	RiskMetrics     map[string]interface{} `json:"risk_metrics"`
}

func main() {
	fmt.Println("🔍 策略21详细分析系统")
	fmt.Println("======================")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	analyzer := &Strategy21Analyzer{db: db}

	// 查询策略21的基本信息
	fmt.Println("\n📋 第一步: 获取策略基本信息")
	strategy, err := analyzer.getStrategyByID(21)
	if err != nil {
		log.Fatalf("获取策略21失败: %v", err)
	}

	analyzer.displayStrategyBasicInfo(strategy)

	// 解析策略配置
	fmt.Println("\n⚙️ 第二步: 解析策略配置")
	config, err := analyzer.parseStrategyConfig(strategy.Config)
	if err != nil {
		log.Printf("解析配置失败: %v", err)
	} else {
		analyzer.displayStrategyConfig(config)
	}

	// 解析策略条件
	fmt.Println("\n🎯 第三步: 解析策略条件")
	conditions, err := analyzer.parseStrategyConditions(strategy.Conditions)
	if err != nil {
		log.Printf("解析条件失败: %v", err)
	} else {
		analyzer.displayStrategyConditions(conditions)
	}

	// 解析策略表现
	fmt.Println("\n📊 第四步: 解析策略表现")
	performance, err := analyzer.parseStrategyPerformance(strategy.Performance)
	if err != nil {
		log.Printf("解析表现失败: %v", err)
	} else {
		analyzer.displayStrategyPerformance(performance)
	}

	// 分析策略质量
	fmt.Println("\n🔬 第五步: 策略质量评估")
	quality := analyzer.analyzeStrategyQuality(strategy, config, conditions, performance)
	analyzer.displayStrategyQualityAnalysis(quality)

	// 市场适应性分析
	fmt.Println("\n🌍 第六步: 市场适应性分析")
	marketFit := analyzer.analyzeMarketFit(strategy, conditions, performance)
	analyzer.displayMarketFitAnalysis(marketFit)

	// 改进建议
	fmt.Println("\n💡 第七步: 改进建议")
	recommendations := analyzer.generateImprovementRecommendations(strategy, quality, marketFit)
	analyzer.displayImprovementRecommendations(recommendations)

	fmt.Println("\n🎉 策略21分析完成！")
}

func (s21a *Strategy21Analyzer) getStrategyByID(id int) (*TradingStrategy, error) {
	query := `
		SELECT id, name, type, description, config, conditions, status, performance, created_at, updated_at
		FROM trading_strategies
		WHERE id = ?`

	var strategy TradingStrategy
	err := s21a.db.QueryRow(query, id).Scan(
		&strategy.ID,
		&strategy.Name,
		&strategy.Type,
		&strategy.Description,
		&strategy.Config,
		&strategy.Conditions,
		&strategy.Status,
		&strategy.Performance,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &strategy, nil
}

func (s21a *Strategy21Analyzer) parseStrategyConfig(configStr string) (*StrategyConfig, error) {
	if configStr == "" {
		return &StrategyConfig{
			Parameters: make(map[string]interface{}),
			Rules:      []string{},
		}, nil
	}

	var config StrategyConfig
	err := json.Unmarshal([]byte(configStr), &config)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

func (s21a *Strategy21Analyzer) parseStrategyConditions(conditionsStr string) (*StrategyConditions, error) {
	if conditionsStr == "" {
		return &StrategyConditions{
			RiskLimits:      make(map[string]interface{}),
			TimeFilters:     make(map[string]interface{}),
			EntryConditions: []string{},
			ExitConditions:  []string{},
			SymbolFilters:   []string{},
		}, nil
	}

	var conditions StrategyConditions
	err := json.Unmarshal([]byte(conditionsStr), &conditions)
	if err != nil {
		return nil, err
	}

	return &conditions, nil
}

func (s21a *Strategy21Analyzer) parseStrategyPerformance(performanceStr string) (*StrategyPerformance, error) {
	if performanceStr == "" {
		return &StrategyPerformance{
			MonthlyReturns: make(map[string]float64),
			DailyStats:     make(map[string]interface{}),
			RiskMetrics:    make(map[string]interface{}),
		}, nil
	}

	var performance StrategyPerformance
	err := json.Unmarshal([]byte(performanceStr), &performance)
	if err != nil {
		return nil, err
	}

	return &performance, nil
}

func (s21a *Strategy21Analyzer) displayStrategyBasicInfo(strategy *TradingStrategy) {
	fmt.Println("策略基本信息:")
	fmt.Println("─────────────")
	fmt.Printf("ID: %d\n", strategy.ID)
	fmt.Printf("名称: %s\n", strategy.Name)
	fmt.Printf("类型: %s\n", strategy.Type)
	fmt.Printf("描述: %s\n", strategy.Description)
	fmt.Printf("状态: %s\n", strategy.Status)
	fmt.Printf("创建时间: %s\n", strategy.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("更新时间: %s\n", strategy.UpdatedAt.Format("2006-01-02 15:04:05"))
}

func (s21a *Strategy21Analyzer) displayStrategyConfig(config *StrategyConfig) {
	fmt.Println("策略配置:")
	fmt.Println("─────────")

	if len(config.Parameters) > 0 {
		fmt.Println("参数设置:")
		for key, value := range config.Parameters {
			fmt.Printf("  %s: %v\n", key, value)
		}
	} else {
		fmt.Println("  无参数配置")
	}

	if len(config.Rules) > 0 {
		fmt.Println("规则设置:")
		for i, rule := range config.Rules {
			fmt.Printf("  %d. %s\n", i+1, rule)
		}
	} else {
		fmt.Println("  无规则配置")
	}
}

func (s21a *Strategy21Analyzer) displayStrategyConditions(conditions *StrategyConditions) {
	fmt.Println("策略条件:")
	fmt.Println("─────────")
	fmt.Printf("启用状态: %t\n", conditions.Enabled)
	fmt.Printf("市场类型: %s\n", conditions.MarketType)
	fmt.Printf("最小成交量: %.0f\n", conditions.MinVolume)
	fmt.Printf("最大持仓数: %d\n", conditions.MaxPositions)

	if len(conditions.RiskLimits) > 0 {
		fmt.Println("风险限制:")
		for key, value := range conditions.RiskLimits {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	if len(conditions.EntryConditions) > 0 {
		fmt.Println("入场条件:")
		for i, condition := range conditions.EntryConditions {
			fmt.Printf("  %d. %s\n", i+1, condition)
		}
	}

	if len(conditions.ExitConditions) > 0 {
		fmt.Println("出场条件:")
		for i, condition := range conditions.ExitConditions {
			fmt.Printf("  %d. %s\n", i+1, condition)
		}
	}

	if len(conditions.TimeFilters) > 0 {
		fmt.Println("时间过滤:")
		for key, value := range conditions.TimeFilters {
			fmt.Printf("  %s: %v\n", key, value)
		}
	}

	if len(conditions.SymbolFilters) > 0 {
		fmt.Println("交易对过滤:")
		for i, symbol := range conditions.SymbolFilters {
			fmt.Printf("  %d. %s\n", i+1, symbol)
		}
	}
}

func (s21a *Strategy21Analyzer) displayStrategyPerformance(performance *StrategyPerformance) {
	fmt.Println("策略表现:")
	fmt.Println("─────────")

	if performance.TotalTrades > 0 {
		fmt.Printf("总交易次数: %d\n", performance.TotalTrades)
		fmt.Printf("胜率: %.1f%%\n", performance.WinRate*100)
		fmt.Printf("平均收益率: %.2f%%\n", performance.AvgReturn*100)
		fmt.Printf("最大回撤: %.2f%%\n", performance.MaxDrawdown*100)
		fmt.Printf("夏普比率: %.3f\n", performance.SharpeRatio)
		fmt.Printf("盈利因子: %.3f\n", performance.ProfitFactor)
		fmt.Printf("恢复因子: %.3f\n", performance.RecoveryFactor)
		fmt.Printf("卡玛比率: %.3f\n", performance.CalmarRatio)
		fmt.Printf("平均持仓时间: %s\n", performance.AvgHoldTime)
		fmt.Printf("最佳交易: %.2f%%\n", performance.BestTrade*100)
		fmt.Printf("最差交易: %.2f%%\n", performance.WorstTrade*100)

		if len(performance.MonthlyReturns) > 0 {
			fmt.Println("月度收益:")
			for month, ret := range performance.MonthlyReturns {
				fmt.Printf("  %s: %.2f%%\n", month, ret*100)
			}
		}
	} else {
		fmt.Println("暂无交易记录")
	}
}

type StrategyQualityAnalysis struct {
	OverallScore       float64
	ConfigCompleteness float64
	ConditionRobustness float64
	PerformanceQuality float64
	RiskManagement     float64
	MarketAdaptability float64
	CodeQuality        float64
	Strengths          []string
	Weaknesses         []string
	Risks              []string
}

func (s21a *Strategy21Analyzer) analyzeStrategyQuality(strategy *TradingStrategy, config *StrategyConfig, conditions *StrategyConditions, performance *StrategyPerformance) *StrategyQualityAnalysis {
	analysis := &StrategyQualityAnalysis{}

	// 配置完整性评分
	configScore := 0.0
	if len(config.Parameters) > 0 {
		configScore += 0.4
	}
	if len(config.Rules) > 0 {
		configScore += 0.6
	}
	analysis.ConfigCompleteness = configScore

	// 条件健壮性评分
	conditionScore := 0.0
	if conditions.Enabled {
		conditionScore += 0.2
	}
	if conditions.MinVolume > 0 {
		conditionScore += 0.2
	}
	if len(conditions.RiskLimits) > 0 {
		conditionScore += 0.3
	}
	if len(conditions.EntryConditions) > 0 && len(conditions.ExitConditions) > 0 {
		conditionScore += 0.3
	}
	analysis.ConditionRobustness = conditionScore

	// 表现质量评分
	perfScore := 0.0
	if performance.TotalTrades > 0 {
		if performance.WinRate > 0.5 {
			perfScore += 0.3
		}
		if performance.SharpeRatio > 1.0 {
			perfScore += 0.3
		}
		if performance.MaxDrawdown < 0.2 {
			perfScore += 0.2
		}
		if performance.ProfitFactor > 1.2 {
			perfScore += 0.2
		}
	}
	analysis.PerformanceQuality = perfScore

	// 风险管理评分
	riskScore := 0.0
	if len(conditions.RiskLimits) > 0 {
		riskScore += 0.4
	}
	if performance.TotalTrades > 0 && performance.MaxDrawdown < 0.3 {
		riskScore += 0.3
	}
	if performance.RecoveryFactor > 1.0 {
		riskScore += 0.3
	}
	analysis.RiskManagement = riskScore

	// 市场适应性评分
	adaptScore := 0.0
	if strings.Contains(strategy.Type, "grid") || strings.Contains(strategy.Type, "mean_reversion") {
		adaptScore += 0.5 // 适合震荡市
	}
	if conditions.MarketType != "" {
		adaptScore += 0.3
	}
	if len(conditions.TimeFilters) > 0 {
		adaptScore += 0.2
	}
	analysis.MarketAdaptability = adaptScore

	// 总体评分
	analysis.OverallScore = (analysis.ConfigCompleteness*0.15 +
		analysis.ConditionRobustness*0.20 +
		analysis.PerformanceQuality*0.30 +
		analysis.RiskManagement*0.20 +
		analysis.MarketAdaptability*0.15)

	// 识别优势
	analysis.Strengths = s21a.identifyStrengths(analysis)

	// 识别劣势
	analysis.Weaknesses = s21a.identifyWeaknesses(analysis)

	// 识别风险
	analysis.Risks = s21a.identifyRisks(strategy, conditions, performance)

	return analysis
}

func (s21a *Strategy21Analyzer) identifyStrengths(analysis *StrategyQualityAnalysis) []string {
	strengths := []string{}

	if analysis.ConfigCompleteness > 0.8 {
		strengths = append(strengths, "配置完整，参数设置合理")
	}

	if analysis.ConditionRobustness > 0.8 {
		strengths = append(strengths, "条件设置健壮，风险控制到位")
	}

	if analysis.PerformanceQuality > 0.7 {
		strengths = append(strengths, "历史表现优秀，关键指标突出")
	}

	if analysis.RiskManagement > 0.7 {
		strengths = append(strengths, "风险管理完善，回撤控制良好")
	}

	if analysis.MarketAdaptability > 0.7 {
		strengths = append(strengths, "市场适应性强，环境匹配度高")
	}

	if len(strengths) == 0 {
		strengths = append(strengths, "策略结构完整，基础扎实")
	}

	return strengths
}

func (s21a *Strategy21Analyzer) identifyWeaknesses(analysis *StrategyQualityAnalysis) []string {
	weaknesses := []string{}

	if analysis.ConfigCompleteness < 0.5 {
		weaknesses = append(weaknesses, "配置不完整，缺少关键参数")
	}

	if analysis.ConditionRobustness < 0.5 {
		weaknesses = append(weaknesses, "条件设置薄弱，缺乏必要限制")
	}

	if analysis.PerformanceQuality < 0.4 {
		weaknesses = append(weaknesses, "历史表现一般，需要优化")
	}

	if analysis.RiskManagement < 0.5 {
		weaknesses = append(weaknesses, "风险管理不足，回撤风险较高")
	}

	if analysis.MarketAdaptability < 0.5 {
		weaknesses = append(weaknesses, "市场适应性差，环境匹配不足")
	}

	if len(weaknesses) == 0 {
		weaknesses = append(weaknesses, "需要持续监控和微调")
	}

	return weaknesses
}

func (s21a *Strategy21Analyzer) identifyRisks(strategy *TradingStrategy, conditions *StrategyConditions, performance *StrategyPerformance) []string {
	risks := []string{}

	if performance.TotalTrades < 100 {
		risks = append(risks, "交易样本不足，统计意义有限")
	}

	if performance.MaxDrawdown > 0.3 {
		risks = append(risks, "历史最大回撤较大，风险较高")
	}

	if performance.SharpeRatio < 0.5 {
		risks = append(risks, "风险调整收益不足，效率低下")
	}

	if len(conditions.RiskLimits) == 0 {
		risks = append(risks, "缺乏明确的风险限制")
	}

	if conditions.MaxPositions > 10 {
		risks = append(risks, "持仓集中度较高，风险集中")
	}

	if len(risks) == 0 {
		risks = append(risks, "整体风险可控")
	}

	return risks
}

func (s21a *Strategy21Analyzer) displayStrategyQualityAnalysis(analysis *StrategyQualityAnalysis) {
	fmt.Println("策略质量评估:")
	fmt.Println("─────────────")
	fmt.Printf("总体评分: %.1f/1.0\n", analysis.OverallScore)
	fmt.Printf("配置完整性: %.1f/1.0\n", analysis.ConfigCompleteness)
	fmt.Printf("条件健壮性: %.1f/1.0\n", analysis.ConditionRobustness)
	fmt.Printf("表现质量: %.1f/1.0\n", analysis.PerformanceQuality)
	fmt.Printf("风险管理: %.1f/1.0\n", analysis.RiskManagement)
	fmt.Printf("市场适应性: %.1f/1.0\n", analysis.MarketAdaptability)

	fmt.Println("\n优势:")
	for _, strength := range analysis.Strengths {
		fmt.Printf("  ✅ %s\n", strength)
	}

	fmt.Println("\n劣势:")
	for _, weakness := range analysis.Weaknesses {
		fmt.Printf("  ⚠️ %s\n", weakness)
	}

	fmt.Println("\n风险:")
	for _, risk := range analysis.Risks {
		fmt.Printf("  🚨 %s\n", risk)
	}
}

type MarketFitAnalysis struct {
	CurrentRegime     string
	StrategySuitability float64
	RegimeMatch       float64
	VolatilityFit     float64
	VolumeFit         float64
	TimeFit           float64
	SymbolFit         float64
	CompetitiveAdvantage float64
	Recommendations   []string
}

func (s21a *Strategy21Analyzer) analyzeMarketFit(strategy *TradingStrategy, conditions *StrategyConditions, performance *StrategyPerformance) *MarketFitAnalysis {
	analysis := &MarketFitAnalysis{}

	// 分析当前市场环境
	currentRegime := s21a.determineCurrentMarketRegime()
	analysis.CurrentRegime = currentRegime

	// 计算策略适用性
	strategySuitability := s21a.calculateStrategySuitability(strategy, currentRegime)
	analysis.StrategySuitability = strategySuitability

	// 市场环境匹配度
	regimeMatch := s21a.calculateRegimeMatch(strategy, currentRegime)
	analysis.RegimeMatch = regimeMatch

	// 波动率适应性
	volatilityFit := s21a.calculateVolatilityFit(strategy, conditions)
	analysis.VolatilityFit = volatilityFit

	// 成交量适应性
	volumeFit := s21a.calculateVolumeFit(conditions)
	analysis.VolumeFit = volumeFit

	// 时间适应性
	timeFit := s21a.calculateTimeFit(conditions)
	analysis.TimeFit = timeFit

	// 交易对适应性
	symbolFit := s21a.calculateSymbolFit(conditions)
	analysis.SymbolFit = symbolFit

	// 竞争优势
	competitiveAdvantage := s21a.calculateCompetitiveAdvantage(strategy, performance)
	analysis.CompetitiveAdvantage = competitiveAdvantage

	// 生成建议
	analysis.Recommendations = s21a.generateMarketFitRecommendations(analysis)

	return analysis
}

func (s21a *Strategy21Analyzer) determineCurrentMarketRegime() string {
	// 查询最近24小时的市场数据来判断当前市场环境
	query := `
		SELECT
			AVG(price_change_percent) as avg_change,
			STDDEV(price_change_percent) as volatility,
			COUNT(CASE WHEN price_change_percent > 5 THEN 1 END) / COUNT(*) as bull_ratio,
			COUNT(CASE WHEN price_change_percent < -5 THEN 1 END) / COUNT(*) as bear_ratio,
			COUNT(CASE WHEN ABS(price_change_percent) <= 5 THEN 1 END) / COUNT(*) as neutral_ratio
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000`

	var avgChange, volatility, bullRatio, bearRatio, neutralRatio float64
	err := s21a.db.QueryRow(query).Scan(&avgChange, &volatility, &bullRatio, &bearRatio, &neutralRatio)
	if err != nil {
		return "未知"
	}

	if neutralRatio > 0.7 && volatility > 6 {
		return "高波动震荡市"
	} else if neutralRatio > 0.6 {
		return "震荡市"
	} else if bullRatio > 0.4 {
		return "强势上涨趋势市"
	} else if bearRatio > 0.4 {
		return "强势下跌趋势市"
	} else {
		return "混合市场"
	}
}

func (s21a *Strategy21Analyzer) calculateStrategySuitability(strategy *TradingStrategy, regime string) float64 {
	// 基于策略类型和市场环境的匹配度
	baseScore := 0.5

	strategyType := strings.ToLower(strategy.Type)

	switch regime {
	case "高波动震荡市":
		if strings.Contains(strategyType, "mean_reversion") || strings.Contains(strategyType, "volatility") {
			baseScore = 0.9
		} else if strings.Contains(strategyType, "grid") {
			baseScore = 0.8
		}

	case "震荡市":
		if strings.Contains(strategyType, "grid") || strings.Contains(strategyType, "mean_reversion") {
			baseScore = 0.9
		} else if strings.Contains(strategyType, "arbitrage") {
			baseScore = 0.8
		}

	case "强势上涨趋势市", "强势下跌趋势市":
		if strings.Contains(strategyType, "trend") || strings.Contains(strategyType, "momentum") {
			baseScore = 0.8
		} else if strings.Contains(strategyType, "breakout") {
			baseScore = 0.7
		}

	default:
		baseScore = 0.6
	}

	return baseScore
}

func (s21a *Strategy21Analyzer) calculateRegimeMatch(strategy *TradingStrategy, regime string) float64 {
	return s21a.calculateStrategySuitability(strategy, regime)
}

func (s21a *Strategy21Analyzer) calculateVolatilityFit(strategy *TradingStrategy, conditions *StrategyConditions) float64 {
	// 获取当前市场波动率
	query := `
		SELECT AVG((high_price - low_price) / low_price * 100)
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000`

	var currentVolatility float64
	s21a.db.QueryRow(query).Scan(&currentVolatility)

	// 基于策略类型判断波动率适应性
	strategyType := strings.ToLower(strategy.Type)

	if strings.Contains(strategyType, "volatility") && currentVolatility > 8 {
		return 0.9
	} else if strings.Contains(strategyType, "grid") && currentVolatility < 6 {
		return 0.8
	} else if strings.Contains(strategyType, "trend") && currentVolatility > 4 {
		return 0.7
	} else {
		return 0.6
	}
}

func (s21a *Strategy21Analyzer) calculateVolumeFit(conditions *StrategyConditions) float64 {
	if conditions.MinVolume > 1000000 {
		return 0.8 // 只选择高成交量的交易对
	} else if conditions.MinVolume > 100000 {
		return 0.9 // 成交量要求适中
	} else {
		return 0.7 // 成交量要求较低
	}
}

func (s21a *Strategy21Analyzer) calculateTimeFit(conditions *StrategyConditions) float64 {
	if len(conditions.TimeFilters) > 0 {
		return 0.8 // 有时间过滤，更精确
	} else {
		return 0.6 // 无时间限制，全天运行
	}
}

func (s21a *Strategy21Analyzer) calculateSymbolFit(conditions *StrategyConditions) float64 {
	if len(conditions.SymbolFilters) > 0 {
		return 0.9 // 明确指定交易对，更专注
	} else {
		return 0.7 // 无限制，适用范围广
	}
}

func (s21a *Strategy21Analyzer) calculateCompetitiveAdvantage(strategy *TradingStrategy, performance *StrategyPerformance) float64 {
	advantage := 0.5

	if performance.SharpeRatio > 1.5 {
		advantage += 0.2
	}

	if performance.WinRate > 0.6 {
		advantage += 0.2
	}

	if performance.ProfitFactor > 1.5 {
		advantage += 0.1
	}

	return advantage
}

func (s21a *Strategy21Analyzer) generateMarketFitRecommendations(analysis *MarketFitAnalysis) []string {
	recommendations := []string{}

	if analysis.StrategySuitability < 0.7 {
		recommendations = append(recommendations, "策略与当前市场环境匹配度不高，考虑调整参数或暂停使用")
	}

	if analysis.VolatilityFit < 0.7 {
		recommendations = append(recommendations, "波动率环境不匹配，考虑调整策略对波动率的敏感度")
	}

	if analysis.VolumeFit < 0.8 {
		recommendations = append(recommendations, "成交量要求可能过于严格，考虑放宽条件以增加交易机会")
	}

	if analysis.TimeFit < 0.7 {
		recommendations = append(recommendations, "建议添加时间过滤条件，提高策略执行效率")
	}

	if analysis.CompetitiveAdvantage > 0.7 {
		recommendations = append(recommendations, "策略具有较强竞争优势，可以考虑增加资金分配")
	} else {
		recommendations = append(recommendations, "策略竞争优势一般，建议与其他策略组合使用")
	}

	if len(recommendations) == 0 {
		recommendations = append(recommendations, "策略整体适应性良好，可以继续使用")
	}

	return recommendations
}

func (s21a *Strategy21Analyzer) displayMarketFitAnalysis(analysis *MarketFitAnalysis) {
	fmt.Println("市场适应性分析:")
	fmt.Println("──────────────")
	fmt.Printf("当前市场环境: %s\n", analysis.CurrentRegime)
	fmt.Printf("策略适用性: %.1f/1.0\n", analysis.StrategySuitability)
	fmt.Printf("环境匹配度: %.1f/1.0\n", analysis.RegimeMatch)
	fmt.Printf("波动率适应性: %.1f/1.0\n", analysis.VolatilityFit)
	fmt.Printf("成交量适应性: %.1f/1.0\n", analysis.VolumeFit)
	fmt.Printf("时间适应性: %.1f/1.0\n", analysis.TimeFit)
	fmt.Printf("交易对适应性: %.1f/1.0\n", analysis.SymbolFit)
	fmt.Printf("竞争优势: %.1f/1.0\n", analysis.CompetitiveAdvantage)

	fmt.Println("\n市场适应建议:")
	for _, rec := range analysis.Recommendations {
		fmt.Printf("  💡 %s\n", rec)
	}
}

type ImprovementRecommendations struct {
	PriorityImprovements []string
	ParameterTweaks      map[string]interface{}
	RiskEnhancements     []string
	PerformanceBoosts    []string
	TechnicalUpgrades    []string
	Timeframe           string
	ResourceRequirements []string
	ExpectedOutcomes     []string
}

func (s21a *Strategy21Analyzer) generateImprovementRecommendations(strategy *TradingStrategy, quality *StrategyQualityAnalysis, marketFit *MarketFitAnalysis) *ImprovementRecommendations {
	recs := &ImprovementRecommendations{
		ParameterTweaks: make(map[string]interface{}),
	}

	// 优先改进项目
	if quality.ConfigCompleteness < 0.8 {
		recs.PriorityImprovements = append(recs.PriorityImprovements, "完善策略配置参数")
	}

	if quality.ConditionRobustness < 0.8 {
		recs.PriorityImprovements = append(recs.PriorityImprovements, "加强风险控制条件")
	}

	if quality.PerformanceQuality < 0.7 {
		recs.PriorityImprovements = append(recs.PriorityImprovements, "优化策略表现指标")
	}

	if marketFit.StrategySuitability < 0.7 {
		recs.PriorityImprovements = append(recs.PriorityImprovements, "调整市场适应性参数")
	}

	// 参数调整建议
	if quality.ConfigCompleteness < 0.5 {
		recs.ParameterTweaks["add_missing_params"] = true
		recs.ParameterTweaks["optimize_defaults"] = true
	}

	if marketFit.VolatilityFit < 0.7 {
		recs.ParameterTweaks["volatility_adjustment"] = 0.1
	}

	if marketFit.VolumeFit < 0.8 {
		recs.ParameterTweaks["min_volume_threshold"] = 50000
	}

	// 风险增强建议
	if quality.RiskManagement < 0.7 {
		recs.RiskEnhancements = append(recs.RiskEnhancements, "增加止损机制")
		recs.RiskEnhancements = append(recs.RiskEnhancements, "实施仓位限制")
		recs.RiskEnhancements = append(recs.RiskEnhancements, "添加风险敞口监控")
	}

	// 表现提升建议
	if quality.PerformanceQuality < 0.6 {
		recs.PerformanceBoosts = append(recs.PerformanceBoosts, "优化入场时机")
		recs.PerformanceBoosts = append(recs.PerformanceBoosts, "改进出场策略")
		recs.PerformanceBoosts = append(recs.PerformanceBoosts, "减少交易频率")
	}

	// 技术升级建议
	recs.TechnicalUpgrades = append(recs.TechnicalUpgrades, "增加实时监控")
	recs.TechnicalUpgrades = append(recs.TechnicalUpgrades, "完善日志记录")
	recs.TechnicalUpgrades = append(recs.TechnicalUpgrades, "添加性能分析")

	// 时间安排
	if quality.OverallScore < 0.6 {
		recs.Timeframe = "3-6个月"
	} else if quality.OverallScore < 0.8 {
		recs.Timeframe = "1-3个月"
	} else {
		recs.Timeframe = "持续优化"
	}

	// 资源需求
	recs.ResourceRequirements = append(recs.ResourceRequirements, "量化分析师: 1人")
	recs.ResourceRequirements = append(recs.ResourceRequirements, "开发工程师: 1人")
	recs.ResourceRequirements = append(recs.ResourceRequirements, "测试环境: 1套")

	// 预期结果
	if quality.OverallScore < 0.5 {
		recs.ExpectedOutcomes = append(recs.ExpectedOutcomes, "整体表现提升50%以上")
		recs.ExpectedOutcomes = append(recs.ExpectedOutcomes, "风险指标显著改善")
	} else {
		recs.ExpectedOutcomes = append(recs.ExpectedOutcomes, "整体表现提升20-30%")
		recs.ExpectedOutcomes = append(recs.ExpectedOutcomes, "稳定性进一步增强")
	}

	return recs
}

func (s21a *Strategy21Analyzer) displayImprovementRecommendations(recs *ImprovementRecommendations) {
	fmt.Println("改进建议:")
	fmt.Println("─────────")

	if len(recs.PriorityImprovements) > 0 {
		fmt.Println("优先改进项目:")
		for i, item := range recs.PriorityImprovements {
			fmt.Printf("  %d. %s\n", i+1, item)
		}
	}

	if len(recs.ParameterTweaks) > 0 {
		fmt.Println("\n参数调整建议:")
		for param, value := range recs.ParameterTweaks {
			fmt.Printf("  %s: %v\n", param, value)
		}
	}

	if len(recs.RiskEnhancements) > 0 {
		fmt.Println("\n风险增强措施:")
		for _, risk := range recs.RiskEnhancements {
			fmt.Printf("  • %s\n", risk)
		}
	}

	if len(recs.PerformanceBoosts) > 0 {
		fmt.Println("\n表现提升措施:")
		for _, boost := range recs.PerformanceBoosts {
			fmt.Printf("  • %s\n", boost)
		}
	}

	if len(recs.TechnicalUpgrades) > 0 {
		fmt.Println("\n技术升级建议:")
		for _, upgrade := range recs.TechnicalUpgrades {
			fmt.Printf("  • %s\n", upgrade)
		}
	}

	fmt.Printf("\n实施时间表: %s\n", recs.Timeframe)

	if len(recs.ResourceRequirements) > 0 {
		fmt.Println("\n资源需求:")
		for _, resource := range recs.ResourceRequirements {
			fmt.Printf("  • %s\n", resource)
		}
	}

	if len(recs.ExpectedOutcomes) > 0 {
		fmt.Println("\n预期结果:")
		for _, outcome := range recs.ExpectedOutcomes {
			fmt.Printf("  • %s\n", outcome)
		}
	}
}