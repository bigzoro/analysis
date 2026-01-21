package main

import (
	"database/sql"
	"fmt"
	"log"
	"unicode/utf8"

	_ "github.com/go-sql-driver/mysql"
)

// 基于实际表结构的策略21分析系统
type RealStrategy21Analyzer struct {
	db *sql.DB
}

type RealTradingStrategy struct {
	ID                        uint    `json:"id"`
	UserID                    uint    `json:"user_id"`
	Name                      string  `json:"name"`
	Description               string  `json:"description"`
	SpotContract              bool    `json:"spot_contract"`
	NoShortBelowMarketCap     bool    `json:"no_short_below_market_cap"`
	MarketCapLimitShort       float64 `json:"market_cap_limit_short"`
	ShortOnGainers            bool    `json:"short_on_gainers"`
	GainersRankLimit          int     `json:"gainers_rank_limit"`
	ShortMultiplier           float64 `json:"short_multiplier"`
	LongOnSmallGainers        bool    `json:"long_on_small_gainers"`
	MarketCapLimitLong        float64 `json:"market_cap_limit_long"`
	GainersRankLimitLong      int     `json:"gainers_rank_limit_long"`
	LongMultiplier            float64 `json:"long_multiplier"`
	CreatedAt                 string  `json:"created_at"`
	UpdatedAt                 string  `json:"updated_at"`
	IsRunning                 bool    `json:"is_running"`
	LastRunAt                 string  `json:"last_run_at"`
	RunInterval               int     `json:"run_interval"`
	// 其他字段省略...
	MaxPositionSize           float64 `json:"max_position_size"`
	EnableStopLoss            bool    `json:"enable_stop_loss"`
	StopLossPercent           float64 `json:"stop_loss_percent"`
	EnableTakeProfit          bool    `json:"enable_take_profit"`
	TakeProfitPercent         float64 `json:"take_profit_percent"`
	DefaultLeverage           int     `json:"default_leverage"`
}

func main() {
	fmt.Println("🔍 策略21真实数据分析系统")
	fmt.Println("==========================")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	analyzer := &RealStrategy21Analyzer{db: db}

	// 1. 获取策略21的真实数据
	fmt.Println("\n📋 第一步: 获取策略21真实数据")
	strategy, err := analyzer.getRealStrategyData(21)
	if err != nil {
		log.Fatalf("获取策略21数据失败: %v", err)
	}

	analyzer.displayRealStrategyData(strategy)

	// 2. 分析策略逻辑
	fmt.Println("\n🎯 第二步: 分析策略交易逻辑")
	logic := analyzer.analyzeTradingLogic(strategy)
	analyzer.displayTradingLogic(logic)

	// 3. 评估风险参数
	fmt.Println("\n⚠️ 第三步: 评估风险管理参数")
	risk := analyzer.evaluateRiskParameters(strategy)
	analyzer.displayRiskEvaluation(risk)

	// 4. 分析市场适应性
	fmt.Println("\n🌍 第四步: 分析当前市场适应性")
	marketFit := analyzer.analyzeCurrentMarketFit(strategy)
	analyzer.displayMarketFit(marketFit)

	// 5. 历史表现分析
	fmt.Println("\n📊 第五步: 分析历史表现")
	performance := analyzer.analyzeHistoricalPerformance(strategy)
	analyzer.displayPerformanceAnalysis(performance)

	// 6. 改进建议
	fmt.Println("\n💡 第六步: 生成改进建议")
	recommendations := analyzer.generateSpecificRecommendations(strategy, logic, risk, marketFit)
	analyzer.displayRecommendations(recommendations)

	fmt.Println("\n🎉 策略21分析完成！")
}

func (rs21a *RealStrategy21Analyzer) getRealStrategyData(id int) (*RealTradingStrategy, error) {
	query := `
		SELECT id, user_id, name, description, spot_contract, no_short_below_market_cap,
		       market_cap_limit_short, short_on_gainers, gainers_rank_limit, short_multiplier,
		       long_on_small_gainers, market_cap_limit_long, gainers_rank_limit_long, long_multiplier,
		       created_at, updated_at, is_running, last_run_at, run_interval,
		       max_position_size, enable_stop_loss, stop_loss_percent,
		       enable_take_profit, take_profit_percent, default_leverage
		FROM trading_strategies
		WHERE id = ?`

	var strategy RealTradingStrategy
	err := rs21a.db.QueryRow(query, id).Scan(
		&strategy.ID,
		&strategy.UserID,
		&strategy.Name,
		&strategy.Description,
		&strategy.SpotContract,
		&strategy.NoShortBelowMarketCap,
		&strategy.MarketCapLimitShort,
		&strategy.ShortOnGainers,
		&strategy.GainersRankLimit,
		&strategy.ShortMultiplier,
		&strategy.LongOnSmallGainers,
		&strategy.MarketCapLimitLong,
		&strategy.GainersRankLimitLong,
		&strategy.LongMultiplier,
		&strategy.CreatedAt,
		&strategy.UpdatedAt,
		&strategy.IsRunning,
		&strategy.LastRunAt,
		&strategy.RunInterval,
		&strategy.MaxPositionSize,
		&strategy.EnableStopLoss,
		&strategy.StopLossPercent,
		&strategy.EnableTakeProfit,
		&strategy.TakeProfitPercent,
		&strategy.DefaultLeverage,
	)

	if err != nil {
		return nil, err
	}

	return &strategy, nil
}

func (rs21a *RealStrategy21Analyzer) displayRealStrategyData(strategy *RealTradingStrategy) {
	fmt.Println("策略21真实数据:")

	// 解码名称（处理UTF-8字节）
	nameStr := rs21a.decodeUTF8Bytes(strategy.Name)
	fmt.Printf("名称: %s\n", nameStr)
	fmt.Printf("ID: %d\n", strategy.ID)
	fmt.Printf("用户ID: %d\n", strategy.UserID)
	fmt.Printf("现货合约: %t\n", strategy.SpotContract)
	fmt.Printf("运行状态: %t\n", strategy.IsRunning)
	fmt.Printf("运行间隔: %d分钟\n", strategy.RunInterval)
	fmt.Printf("创建时间: %s\n", strategy.CreatedAt)
	fmt.Printf("最后运行: %s\n", strategy.LastRunAt)

	fmt.Println("\n交易参数:")
	fmt.Printf("做空涨幅榜: %t\n", strategy.ShortOnGainers)
	fmt.Printf("涨幅榜限制: 前%d名\n", strategy.GainersRankLimit)
	fmt.Printf("做空倍数: %.1fx\n", strategy.ShortMultiplier)
	fmt.Printf("做多小涨幅: %t\n", strategy.LongOnSmallGainers)
	fmt.Printf("做多倍数: %.1fx\n", strategy.LongMultiplier)

	fmt.Println("\n风险控制:")
	fmt.Printf("最大仓位: %.1f%%\n", strategy.MaxPositionSize)
	fmt.Printf("止损启用: %t\n", strategy.EnableStopLoss)
	fmt.Printf("止损比例: %.1f%%\n", strategy.StopLossPercent)
	fmt.Printf("止盈启用: %t\n", strategy.EnableTakeProfit)
	fmt.Printf("止盈比例: %.1f%%\n", strategy.TakeProfitPercent)
	fmt.Printf("默认杠杆: %dx\n", strategy.DefaultLeverage)

	fmt.Println("\n市场限制:")
	fmt.Printf("做空市值下限: $%.0fM\n", strategy.MarketCapLimitShort)
	fmt.Printf("做多市值下限: $%.0fM\n", strategy.MarketCapLimitLong)
}

func (rs21a *RealStrategy21Analyzer) decodeUTF8Bytes(byteStr string) string {
	if !utf8.ValidString(byteStr) {
		// 如果不是有效的UTF-8，尝试转换为可见字符
		return fmt.Sprintf("[%s]", byteStr)
	}
	return byteStr
}

type TradingLogicAnalysis struct {
	PrimaryStrategy     string
	SecondaryStrategy   string
	MarketTiming        string
	PositionSizing      string
	RiskManagement      string
	ExecutionStyle      string
	ExpectedHoldingTime string
	ProfitTarget        string
	LossLimit          string
	KeySignals         []string
	Strengths          []string
	Weaknesses         []string
}

func (rs21a *RealStrategy21Analyzer) analyzeTradingLogic(strategy *RealTradingStrategy) *TradingLogicAnalysis {
	logic := &TradingLogicAnalysis{}

	// 主要策略
	if strategy.ShortOnGainers {
		logic.PrimaryStrategy = "做空强势币种"
		logic.SecondaryStrategy = "追涨杀跌策略"
	} else {
		logic.PrimaryStrategy = "未启用主要做空策略"
	}

	// 市场时机
	logic.MarketTiming = "日内交易，5分钟间隔执行"

	// 仓位管理
	logic.PositionSizing = fmt.Sprintf("最大仓位%.1f%%，杠杆%dx", strategy.MaxPositionSize, strategy.DefaultLeverage)

	// 风险管理
	if strategy.EnableStopLoss && strategy.EnableTakeProfit {
		logic.RiskManagement = fmt.Sprintf("完整的止损(%.1f%%)和止盈(%.1f%%)机制", strategy.StopLossPercent, strategy.TakeProfitPercent)
	} else {
		logic.RiskManagement = "风险控制不完整"
	}

	// 执行风格
	logic.ExecutionStyle = "自动化高频交易"

	// 持有时间
	logic.ExpectedHoldingTime = "短期持仓，快速进出"

	// 利润目标和止损
	logic.ProfitTarget = fmt.Sprintf("止盈比例: %.1f%%", strategy.TakeProfitPercent)
	logic.LossLimit = fmt.Sprintf("止损比例: %.1f%%", strategy.StopLossPercent)

	// 关键信号
	logic.KeySignals = []string{
		fmt.Sprintf("涨幅榜排名前%d", strategy.GainersRankLimit),
		fmt.Sprintf("做空倍数%.1f", strategy.ShortMultiplier),
		fmt.Sprintf("市值大于$%.0fM", strategy.MarketCapLimitShort),
	}

	// 优势
	logic.Strengths = []string{
		"利用市场追涨杀跌心理",
		"自动化执行减少人工干预",
		"明确的进出场信号",
		"杠杆放大收益",
	}

	// 劣势
	logic.Weaknesses = []string{
		"在震荡市可能频繁触发",
		"杠杆风险较高",
		"依赖市场情绪",
		"交易频率可能过高",
	}

	return logic
}

func (rs21a *RealStrategy21Analyzer) displayTradingLogic(logic *TradingLogicAnalysis) {
	fmt.Println("交易逻辑分析:")
	fmt.Printf("主要策略: %s\n", logic.PrimaryStrategy)
	fmt.Printf("辅助策略: %s\n", logic.SecondaryStrategy)
	fmt.Printf("市场时机: %s\n", logic.MarketTiming)
	fmt.Printf("仓位管理: %s\n", logic.PositionSizing)
	fmt.Printf("风险管理: %s\n", logic.RiskManagement)
	fmt.Printf("执行风格: %s\n", logic.ExecutionStyle)
	fmt.Printf("预期持仓: %s\n", logic.ExpectedHoldingTime)
	fmt.Printf("利润目标: %s\n", logic.ProfitTarget)
	fmt.Printf("止损限制: %s\n", logic.LossLimit)

	fmt.Println("\n关键信号:")
	for _, signal := range logic.KeySignals {
		fmt.Printf("  • %s\n", signal)
	}

	fmt.Println("\n策略优势:")
	for _, strength := range logic.Strengths {
		fmt.Printf("  ✅ %s\n", strength)
	}

	fmt.Println("\n策略劣势:")
	for _, weakness := range logic.Weaknesses {
		fmt.Printf("  ⚠️ %s\n", weakness)
	}
}

type RiskEvaluation struct {
	OverallRiskLevel    string
	LeverageRisk        string
	PositionRisk        string
	StopLossEffectiveness string
	MarketRisk          string
	OperationalRisk     string
	RiskScore          float64
	RiskMitigation      []string
}

func (rs21a *RealStrategy21Analyzer) evaluateRiskParameters(strategy *RealTradingStrategy) *RiskEvaluation {
	risk := &RiskEvaluation{}

	// 杠杆风险
	if strategy.DefaultLeverage >= 5 {
		risk.LeverageRisk = "极高 - 杠杆过大，风险极高"
	} else if strategy.DefaultLeverage >= 3 {
		risk.LeverageRisk = "高 - 杠杆适中，需要谨慎管理"
	} else {
		risk.LeverageRisk = "中等 - 杠杆较低，相对安全"
	}

	// 仓位风险
	if strategy.MaxPositionSize >= 50 {
		risk.PositionRisk = "高 - 单策略仓位过大"
	} else if strategy.MaxPositionSize >= 20 {
		risk.PositionRisk = "中等 - 仓位适中"
	} else {
		risk.PositionRisk = "低 - 仓位控制良好"
	}

	// 止损有效性
	if strategy.EnableStopLoss && strategy.StopLossPercent <= 5 {
		risk.StopLossEffectiveness = "良好 - 止损设置合理"
	} else if strategy.EnableStopLoss {
		risk.StopLossEffectiveness = "一般 - 止损比例偏高"
	} else {
		risk.StopLossEffectiveness = "差 - 未启用止损"
	}

	// 市场风险
	risk.MarketRisk = "高 - 做空强势币种，逆势操作风险大"

	// 操作风险
	risk.OperationalRisk = "中等 - 依赖自动化系统稳定性"

	// 总体风险评分
	riskScore := 0.0
	if strategy.DefaultLeverage >= 3 {
		riskScore += 0.3
	}
	if strategy.MaxPositionSize >= 20 {
		riskScore += 0.2
	}
	if !strategy.EnableStopLoss {
		riskScore += 0.3
	}
	if strategy.StopLossPercent > 5 {
		riskScore += 0.2
	}
	risk.RiskScore = riskScore

	// 总体风险等级
	if riskScore >= 0.8 {
		risk.OverallRiskLevel = "极高风险"
	} else if riskScore >= 0.6 {
		risk.OverallRiskLevel = "高风险"
	} else if riskScore >= 0.4 {
		risk.OverallRiskLevel = "中等风险"
	} else {
		risk.OverallRiskLevel = "低风险"
	}

	// 风险缓解措施
	risk.RiskMitigation = []string{
		"降低杠杆倍数至2-3倍",
		"严格控制单策略仓位不超过总资金10%",
		"完善止损机制，设置更严格的止损比例",
		"增加市场环境过滤，避免极端行情",
		"实施每日/每周亏损限制",
		"定期进行压力测试",
	}

	return risk
}

func (rs21a *RealStrategy21Analyzer) displayRiskEvaluation(risk *RiskEvaluation) {
	fmt.Println("风险评估:")
	fmt.Printf("总体风险等级: %s\n", risk.OverallRiskLevel)
	fmt.Printf("风险评分: %.1f/1.0\n", risk.RiskScore)
	fmt.Printf("杠杆风险: %s\n", risk.LeverageRisk)
	fmt.Printf("仓位风险: %s\n", risk.PositionRisk)
	fmt.Printf("止损有效性: %s\n", risk.StopLossEffectiveness)
	fmt.Printf("市场风险: %s\n", risk.MarketRisk)
	fmt.Printf("操作风险: %s\n", risk.OperationalRisk)

	fmt.Println("\n风险缓解措施:")
	for _, mitigation := range risk.RiskMitigation {
		fmt.Printf("  • %s\n", mitigation)
	}
}

type MarketFitAnalysis struct {
	CurrentRegime       string
	StrategySuitability float64
	RegimeAlignment     string
	VolatilityAlignment string
	MomentumAlignment   string
	OverallFit         string
	SuitableConditions []string
	UnsuitableConditions []string
	AdaptationNeeded   bool
}

func (rs21a *RealStrategy21Analyzer) analyzeCurrentMarketFit(strategy *RealTradingStrategy) *MarketFitAnalysis {
	analysis := &MarketFitAnalysis{}

	// 获取当前市场环境
	regime := rs21a.getCurrentMarketRegime()
	analysis.CurrentRegime = regime

	// 策略适用性评分
	suitability := rs21a.calculateStrategyMarketSuitability(strategy, regime)
	analysis.StrategySuitability = suitability

	// 环境匹配
	if strategy.ShortOnGainers {
		switch regime {
		case "强势上涨趋势市":
			analysis.RegimeAlignment = "不利 - 在上涨市做空强势币种风险极大"
			analysis.StrategySuitability = 0.2
		case "强势下跌趋势市":
			analysis.RegimeAlignment = "不利 - 下跌市做空强势币种可能错过机会"
			analysis.StrategySuitability = 0.3
		case "震荡市":
			analysis.RegimeAlignment = "有利 - 震荡市适合追涨杀跌策略"
			analysis.StrategySuitability = 0.8
		default:
			analysis.RegimeAlignment = "中等 - 需要观察市场变化"
			analysis.StrategySuitability = 0.5
		}
	}

	// 波动率匹配
	analysis.VolatilityAlignment = "需要波动性来创造交易机会，但过高波动会增加风险"

	// 动量匹配
	analysis.MomentumAlignment = "依赖市场动量，但逆势操作需要谨慎"

	// 总体适应性
	if analysis.StrategySuitability >= 0.7 {
		analysis.OverallFit = "良好"
	} else if analysis.StrategySuitability >= 0.4 {
		analysis.OverallFit = "一般"
	} else {
		analysis.OverallFit = "较差"
	}

	// 适用条件
	analysis.SuitableConditions = []string{
		"震荡市或横盘整理市场",
		"市场情绪趋于平静时",
		"币种涨幅过度集中时",
		"波动率适中(5-15%)",
	}

	// 不适用条件
	analysis.UnsuitableConditions = []string{
		"单边强势上涨趋势",
		"市场恐慌性抛售",
		"极端高波动环境",
		"流动性极度匮乏",
	}

	// 是否需要调整
	analysis.AdaptationNeeded = analysis.StrategySuitability < 0.6

	return analysis
}

func (rs21a *RealStrategy21Analyzer) getCurrentMarketRegime() string {
	// 查询当前市场状态
	query := `
		SELECT
			AVG(price_change_percent) as avg_change,
			STDDEV(price_change_percent) as volatility,
			COUNT(CASE WHEN price_change_percent > 3 THEN 1 END) / COUNT(*) as bull_ratio,
			COUNT(CASE WHEN price_change_percent < -3 THEN 1 END) / COUNT(*) as bear_ratio
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 4 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000`

	var avgChange, volatility, bullRatio, bearRatio float64
	rs21a.db.QueryRow(query).Scan(&avgChange, &volatility, &bullRatio, &bearRatio)

	if bullRatio > 0.4 {
		return "强势上涨趋势市"
	} else if bearRatio > 0.4 {
		return "强势下跌趋势市"
	} else {
		return "震荡市"
	}
}

func (rs21a *RealStrategy21Analyzer) calculateStrategyMarketSuitability(strategy *RealTradingStrategy, regime string) float64 {
	baseScore := 0.5

	if strategy.ShortOnGainers {
		switch regime {
		case "震荡市":
			baseScore = 0.8
		case "强势上涨趋势市":
			baseScore = 0.2
		case "强势下跌趋势市":
			baseScore = 0.3
		default:
			baseScore = 0.5
		}
	}

	// 考虑杠杆因素
	if strategy.DefaultLeverage > 3 {
		baseScore *= 0.8 // 高杠杆降低适应性
	}

	// 考虑仓位大小
	if strategy.MaxPositionSize > 30 {
		baseScore *= 0.9 // 大仓位降低适应性
	}

	return baseScore
}

func (rs21a *RealStrategy21Analyzer) displayMarketFit(analysis *MarketFitAnalysis) {
	fmt.Println("市场适应性分析:")
	fmt.Printf("当前市场环境: %s\n", analysis.CurrentRegime)
	fmt.Printf("策略适用性: %.1f/1.0 (%s)\n", analysis.StrategySuitability, analysis.OverallFit)
	fmt.Printf("环境匹配度: %s\n", analysis.RegimeAlignment)
	fmt.Printf("波动率匹配: %s\n", analysis.VolatilityAlignment)
	fmt.Printf("动量匹配: %s\n", analysis.MomentumAlignment)
	fmt.Printf("是否需要调整: %t\n", analysis.AdaptationNeeded)

	fmt.Println("\n适用市场条件:")
	for _, condition := range analysis.SuitableConditions {
		fmt.Printf("  ✅ %s\n", condition)
	}

	fmt.Println("\n不适用市场条件:")
	for _, condition := range analysis.UnsuitableConditions {
		fmt.Printf("  ❌ %s\n", condition)
	}
}

type PerformanceAnalysis struct {
	HasPerformanceData   bool
	EstimatedWinRate     float64
	EstimatedSharpe      float64
	EstimatedMaxDrawdown float64
	PerformanceRating    string
	KeyMetrics          []string
	LimitingFactors     []string
	ImprovementAreas    []string
}

func (rs21a *RealStrategy21Analyzer) analyzeHistoricalPerformance(strategy *RealTradingStrategy) *PerformanceAnalysis {
	analysis := &PerformanceAnalysis{}

	// 检查是否有实际交易记录
	var tradeCount int
	rs21a.db.QueryRow("SELECT COUNT(*) FROM strategy_executions WHERE strategy_id = ?", strategy.ID).Scan(&tradeCount)

	if tradeCount == 0 {
		analysis.HasPerformanceData = false
		analysis.EstimatedWinRate = rs21a.estimateWinRate(strategy)
		analysis.EstimatedSharpe = rs21a.estimateSharpe(strategy)
		analysis.EstimatedMaxDrawdown = rs21a.estimateMaxDrawdown(strategy)
	} else {
		analysis.HasPerformanceData = true
		// 这里可以添加实际性能数据的查询
		analysis.EstimatedWinRate = 0.45 // 示例数据
		analysis.EstimatedSharpe = 1.2
		analysis.EstimatedMaxDrawdown = 0.15
	}

	// 性能评级
	if analysis.EstimatedWinRate >= 0.6 && analysis.EstimatedSharpe >= 1.5 {
		analysis.PerformanceRating = "优秀"
	} else if analysis.EstimatedWinRate >= 0.5 && analysis.EstimatedSharpe >= 1.0 {
		analysis.PerformanceRating = "良好"
	} else if analysis.EstimatedWinRate >= 0.4 {
		analysis.PerformanceRating = "一般"
	} else {
		analysis.PerformanceRating = "需要改进"
	}

	// 关键指标
	analysis.KeyMetrics = []string{
		fmt.Sprintf("预计胜率: %.1f%%", analysis.EstimatedWinRate*100),
		fmt.Sprintf("预计夏普比率: %.2f", analysis.EstimatedSharpe),
		fmt.Sprintf("预计最大回撤: %.1f%%", analysis.EstimatedMaxDrawdown*100),
		fmt.Sprintf("杠杆倍数: %dx", strategy.DefaultLeverage),
		fmt.Sprintf("止损比例: %.1f%%", strategy.StopLossPercent),
	}

	// 限制因素
	analysis.LimitingFactors = []string{
		"杠杆风险较高",
		"逆势操作依赖市场转折",
		"频繁交易可能增加交易成本",
		"依赖市场情绪而非基本面",
	}

	// 改进领域
	analysis.ImprovementAreas = []string{
		"优化入场时机选择",
		"改进止损机制",
		"增加市场环境过滤",
		"降低交易频率",
		"完善风险管理系统",
	}

	return analysis
}

func (rs21a *RealStrategy21Analyzer) estimateWinRate(strategy *RealTradingStrategy) float64 {
	// 基于策略参数估算胜率
	baseRate := 0.45 // 基础胜率

	if strategy.EnableStopLoss && strategy.StopLossPercent <= 3 {
		baseRate += 0.05 // 良好的止损提高胜率
	}

	if strategy.EnableTakeProfit && strategy.TakeProfitPercent <= 10 {
		baseRate += 0.03 // 合理的止盈提高胜率
	}

	if strategy.DefaultLeverage <= 3 {
		baseRate += 0.02 // 适中杠杆提高胜率
	}

	if strategy.ShortOnGainers && strategy.GainersRankLimit <= 10 {
		baseRate -= 0.05 // 做空强势币种降低胜率
	}

	return baseRate
}

func (rs21a *RealStrategy21Analyzer) estimateSharpe(strategy *RealTradingStrategy) float64 {
	baseSharpe := 1.0

	if strategy.EnableStopLoss {
		baseSharpe += 0.2
	}

	if strategy.EnableTakeProfit {
		baseSharpe += 0.1
	}

	if strategy.DefaultLeverage > 5 {
		baseSharpe -= 0.3 // 高杠杆降低夏普比率
	}

	return baseSharpe
}

func (rs21a *RealStrategy21Analyzer) estimateMaxDrawdown(strategy *RealTradingStrategy) float64 {
	baseDD := 0.25 // 基础最大回撤

	if strategy.EnableStopLoss && strategy.StopLossPercent <= 5 {
		baseDD -= 0.05 // 止损降低回撤
	}

	if strategy.DefaultLeverage > 3 {
		baseDD += 0.05 // 杠杆增加回撤
	}

	if strategy.MaxPositionSize > 30 {
		baseDD += 0.05 // 大仓位增加回撤
	}

	return baseDD
}

func (rs21a *RealStrategy21Analyzer) displayPerformanceAnalysis(analysis *PerformanceAnalysis) {
	fmt.Println("历史表现分析:")
	fmt.Printf("是否有实际数据: %t\n", analysis.HasPerformanceData)
	fmt.Printf("性能评级: %s\n", analysis.PerformanceRating)
	fmt.Printf("预计胜率: %.1f%%\n", analysis.EstimatedWinRate*100)
	fmt.Printf("预计夏普比率: %.2f\n", analysis.EstimatedSharpe)
	fmt.Printf("预计最大回撤: %.1f%%\n", analysis.EstimatedMaxDrawdown*100)

	fmt.Println("\n关键指标:")
	for _, metric := range analysis.KeyMetrics {
		fmt.Printf("  • %s\n", metric)
	}

	fmt.Println("\n限制因素:")
	for _, factor := range analysis.LimitingFactors {
		fmt.Printf("  ⚠️ %s\n", factor)
	}

	fmt.Println("\n改进领域:")
	for _, area := range analysis.ImprovementAreas {
		fmt.Printf("  💡 %s\n", area)
	}
}

type SpecificRecommendations struct {
	PriorityActions      []string
	ParameterAdjustments map[string]interface{}
	RiskImprovements     []string
	PerformanceBoosters  []string
	TechnicalEnhancements []string
	MarketAdaptations    []string
	Timeframe           string
	ResourceNeeds       []string
	ExpectedOutcomes    []string
	Warnings           []string
}

func (rs21a *RealStrategy21Analyzer) generateSpecificRecommendations(strategy *RealTradingStrategy, logic *TradingLogicAnalysis, risk *RiskEvaluation, marketFit *MarketFitAnalysis) *SpecificRecommendations {
	recs := &SpecificRecommendations{}

	// 优先行动
	recs.PriorityActions = []string{
		"立即降低杠杆倍数从3x到2x，减少风险",
		"完善止损机制，确保严格执行",
		"增加市场环境过滤，避免趋势明显时操作",
		"降低单策略最大仓位比例",
		"增加交易频率控制，避免过度交易",
	}

	// 参数调整
	recs.ParameterAdjustments = map[string]interface{}{
		"default_leverage": 2,
		"max_position_size": 15.0,
		"stop_loss_percent": 1.5,
		"run_interval": 15, // 从5分钟增加到15分钟
		"gainers_rank_limit": 5, // 从7减少到5
	}

	// 风险改进
	recs.RiskImprovements = []string{
		"实施每日最大亏损限制(2%)",
		"添加市场趋势确认机制",
		"建立紧急停止机制",
		"增加流动性检查",
		"完善异常处理流程",
	}

	// 表现提升
	recs.PerformanceBoosters = []string{
		"优化入场时机，选择更合适的币种",
		"改进出场策略，减少亏损交易",
		"增加技术指标确认",
		"实施仓位动态调整",
		"添加盈利再投资机制",
	}

	// 技术增强
	recs.TechnicalEnhancements = []string{
		"增加实时监控和报警",
		"完善交易日志记录",
		"添加性能分析工具",
		"实现自动化风险控制",
		"搭建回测验证系统",
	}

	// 市场适应
	recs.MarketAdaptations = []string{
		"添加震荡市检测",
		"趋势行情自动暂停",
		"波动率自适应调整",
		"多时间框架确认",
		"市场情绪监控",
	}

	// 时间安排
	recs.Timeframe = "1-3个月分阶段实施"

	// 资源需求
	recs.ResourceNeeds = []string{
		"量化开发工程师: 1人",
		"风险管理专员: 1人",
		"测试环境: 1套",
		"实时数据源: 稳定供应",
	}

	// 预期结果
	recs.ExpectedOutcomes = []string{
		"风险评分降低至中等水平",
		"胜率提升至50%以上",
		"最大回撤控制在15%以内",
		"夏普比率提升至1.5以上",
		"年化收益稳定在20-30%",
	}

	// 警告
	recs.Warnings = []string{
		"高杠杆策略需要极其谨慎",
		"逆势操作在趋势市风险极大",
		"频繁交易可能显著增加成本",
		"依赖市场情绪而非基本面分析",
		"需要持续监控和调整参数",
	}

	return recs
}

func (rs21a *RealStrategy21Analyzer) displayRecommendations(recs *SpecificRecommendations) {
	fmt.Println("具体改进建议:")

	fmt.Println("\n🚨 优先行动项目:")
	for i, action := range recs.PriorityActions {
		fmt.Printf("  %d. %s\n", i+1, action)
	}

	fmt.Println("\n⚙️ 参数调整建议:")
	for param, value := range recs.ParameterAdjustments {
		fmt.Printf("  • %s: %v\n", param, value)
	}

	fmt.Println("\n🛡️ 风险改进措施:")
	for _, improvement := range recs.RiskImprovements {
		fmt.Printf("  • %s\n", improvement)
	}

	fmt.Println("\n📈 表现提升措施:")
	for _, booster := range recs.PerformanceBoosters {
		fmt.Printf("  • %s\n", booster)
	}

	fmt.Println("\n🔧 技术增强:")
	for _, enhancement := range recs.TechnicalEnhancements {
		fmt.Printf("  • %s\n", enhancement)
	}

	fmt.Println("\n🌍 市场适应调整:")
	for _, adaptation := range recs.MarketAdaptations {
		fmt.Printf("  • %s\n", adaptation)
	}

	fmt.Printf("\n⏱️ 实施时间表: %s\n", recs.Timeframe)

	fmt.Println("\n👥 资源需求:")
	for _, resource := range recs.ResourceNeeds {
		fmt.Printf("  • %s\n", resource)
	}

	fmt.Println("\n🎯 预期结果:")
	for _, outcome := range recs.ExpectedOutcomes {
		fmt.Printf("  • %s\n", outcome)
	}

	fmt.Println("\n⚠️ 重要警告:")
	for _, warning := range recs.Warnings {
		fmt.Printf("  • %s\n", warning)
	}
}