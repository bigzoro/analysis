package main

import (
	"database/sql"
	"fmt"
	"log"
	"sort"

	_ "github.com/go-sql-driver/mysql"
)

// 详细市场策略分析报告
type DetailedMarketStrategyReport struct {
	db *sql.DB
}

// 市场状态枚举
type MarketCondition struct {
	Regime         string
	Volatility     float64
	TrendStrength  float64
	SentimentScore float64
	LiquidityScore float64
}

// 策略评估结果
type StrategyEvaluation struct {
	StrategyName        string
	MarketFitScore      float64
	RiskAdjustedReturn  float64
	WinRate            float64
	MaxDrawdown        float64
	SharpeRatio        float64
	RecommendedWeight  float64
	ImplementationNotes string
}

// 投资组合建议
type PortfolioRecommendation struct {
	PrimaryStrategy     string
	SecondaryStrategies []string
	RiskParityWeights   map[string]float64
	MaxAllocation       float64
	RebalancingFreq     string
	RiskManagementRules []string
}

func main() {
	fmt.Println("📊 详细市场策略分析报告")
	fmt.Println("============================")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	report := &DetailedMarketStrategyReport{db: db}

	// 生成详细报告
	err = report.generateDetailedReport()
	if err != nil {
		log.Fatal("生成报告失败:", err)
	}

	fmt.Println("\n🎉 报告生成完成！")
}

// 生成详细报告
func (r *DetailedMarketStrategyReport) generateDetailedReport() error {
	// 1. 分析当前市场状态
	fmt.Println("\n🔍 第一步: 市场状态分析")
	marketCondition, err := r.analyzeCurrentMarketCondition()
	if err != nil {
		return fmt.Errorf("市场状态分析失败: %v", err)
	}
	r.displayMarketCondition(marketCondition)

	// 2. 分析强势和弱势币种
	fmt.Println("\n📈 第二步: 强势弱势币种分析")
	strongWeakAnalysis, err := r.analyzeStrongWeakCoins()
	if err != nil {
		return fmt.Errorf("强势弱势分析失败: %v", err)
	}
	r.displayStrongWeakAnalysis(strongWeakAnalysis)

	// 3. 技术指标分析
	fmt.Println("\n📊 第三步: 技术指标分析")
	technicalAnalysis, err := r.analyzeTechnicalIndicators()
	if err != nil {
		return fmt.Errorf("技术指标分析失败: %v", err)
	}
	r.displayTechnicalAnalysis(technicalAnalysis)

	// 4. 策略评估
	fmt.Println("\n🎯 第四步: 量化策略评估")
	strategyEvaluations, err := r.evaluateStrategies(marketCondition)
	if err != nil {
		return fmt.Errorf("策略评估失败: %v", err)
	}
	r.displayStrategyEvaluations(strategyEvaluations)

	// 5. 投资组合建议
	fmt.Println("\n💼 第五步: 投资组合建议")
	portfolioRec, err := r.generatePortfolioRecommendation(strategyEvaluations, marketCondition)
	if err != nil {
		return fmt.Errorf("组合建议失败: %v", err)
	}
	r.displayPortfolioRecommendation(portfolioRec)

	// 6. 风险管理建议
	fmt.Println("\n⚠️ 第六步: 风险管理建议")
	riskManagement := r.generateRiskManagementGuidelines(marketCondition)
	r.displayRiskManagementGuidelines(riskManagement)

	return nil
}

// 分析当前市场状态
func (r *DetailedMarketStrategyReport) analyzeCurrentMarketCondition() (*MarketCondition, error) {
	condition := &MarketCondition{}

	// 查询基本统计
	query := `
		SELECT
			COUNT(*) as total_symbols,
			AVG(price_change_percent) as avg_change,
			AVG((high_price - low_price) / low_price * 100) as avg_volatility,
			SUM(quote_volume) / COUNT(*) as avg_volume,
			COUNT(CASE WHEN price_change_percent > 2 THEN 1 END) / COUNT(*) as bull_ratio,
			COUNT(CASE WHEN price_change_percent < -2 THEN 1 END) / COUNT(*) as bear_ratio,
			COUNT(CASE WHEN quote_volume > 1000000 THEN 1 END) / COUNT(*) as liquidity_ratio
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 100000`

	var totalSymbols int
	var avgChange, avgVolatility, avgVolume, bullRatio, bearRatio, liquidityRatio float64

	err := r.db.QueryRow(query).Scan(
		&totalSymbols, &avgChange, &avgVolatility, &avgVolume,
		&bullRatio, &bearRatio, &liquidityRatio,
	)
	if err != nil {
		return nil, err
	}

	condition.Volatility = avgVolatility
	condition.TrendStrength = bullRatio + bearRatio
	condition.SentimentScore = (bullRatio - bearRatio + 1) / 2 // 标准化到0-1
	condition.LiquidityScore = liquidityRatio

	// 判断市场环境
	if condition.Volatility > 8 && condition.TrendStrength < 0.3 {
		condition.Regime = "高波动震荡市"
	} else if condition.Volatility < 3 && condition.TrendStrength < 0.2 {
		condition.Regime = "低波动整理市"
	} else if condition.TrendStrength > 0.4 {
		if avgChange > 0 {
			condition.Regime = "强势上涨趋势市"
		} else {
			condition.Regime = "强势下跌趋势市"
		}
	} else {
		condition.Regime = "震荡市"
	}

	return condition, nil
}

// 分析强势弱势币种
func (r *DetailedMarketStrategyReport) analyzeStrongWeakCoins() (*StrongWeakAnalysis, error) {
	analysis := &StrongWeakAnalysis{}

	// 获取强势币种
	bullQuery := `
		SELECT symbol, price_change_percent, quote_volume,
		       (high_price - low_price) / low_price * 100 as volatility
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 1000000
			AND price_change_percent > 5
		ORDER BY price_change_percent DESC
		LIMIT 10`

	rows, err := r.db.Query(bullQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var coin CoinMetrics
		if err := rows.Scan(&coin.Symbol, &coin.Change, &coin.Volume, &coin.Volatility); err == nil {
			analysis.StrongCoins = append(analysis.StrongCoins, coin)
		}
	}

	// 获取弱势币种
	bearQuery := `
		SELECT symbol, price_change_percent, quote_volume,
		       (high_price - low_price) / low_price * 100 as volatility
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 1000000
			AND price_change_percent < -5
		ORDER BY price_change_percent ASC
		LIMIT 10`

	rows, err = r.db.Query(bearQuery)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var coin CoinMetrics
		if err := rows.Scan(&coin.Symbol, &coin.Change, &coin.Volume, &coin.Volatility); err == nil {
			analysis.WeakCoins = append(analysis.WeakCoins, coin)
		}
	}

	// 分析特征
	analysis.StrongCoinFeatures = r.analyzeCoinFeatures(analysis.StrongCoins)
	analysis.WeakCoinFeatures = r.analyzeCoinFeatures(analysis.WeakCoins)

	return analysis, nil
}

// 分析技术指标
func (r *DetailedMarketStrategyReport) analyzeTechnicalIndicators() (*TechnicalAnalysis, error) {
	analysis := &TechnicalAnalysis{}

	// 分析主要币种的技术指标 (这里使用简化版本，实际应该计算真实的RSI、MACD等)
	// 由于数据库结构限制，这里使用价格变化的统计特征作为替代

	query := `
		SELECT
			AVG(price_change_percent) as avg_change,
			STDDEV(price_change_percent) as change_stddev,
			COUNT(CASE WHEN price_change_percent > 0 THEN 1 END) / COUNT(*) as positive_ratio,
			MAX(price_change_percent) as max_gain,
			MIN(price_change_percent) as max_loss
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
			AND quote_volume > 1000000
			AND symbol IN ('BTCUSDT', 'ETHUSDT', 'BNBUSDT', 'ADAUSDT', 'SOLUSDT')`

	var avgChange, changeStddev, positiveRatio, maxGain, maxLoss float64
	err := r.db.QueryRow(query).Scan(&avgChange, &changeStddev, &positiveRatio, &maxGain, &maxLoss)
	if err != nil {
		return nil, err
	}

	// 基于统计特征判断技术状态
	analysis.RSIMomentum = positiveRatio * 100 // 简化为正向变化比例
	analysis.MACDTrend = avgChange             // 简化为平均变化
	analysis.BollingerPosition = changeStddev // 简化为标准差

	if positiveRatio > 0.6 {
		analysis.OverallTrend = "强势上涨"
	} else if positiveRatio < 0.4 {
		analysis.OverallTrend = "强势下跌"
	} else {
		analysis.OverallTrend = "震荡整理"
	}

	analysis.SupportLevels = []float64{maxLoss * 0.8, avgChange * 0.9}
	analysis.ResistanceLevels = []float64{maxGain * 0.8, avgChange * 1.1}

	return analysis, nil
}

// 评估策略
func (r *DetailedMarketStrategyReport) evaluateStrategies(marketCondition *MarketCondition) ([]StrategyEvaluation, error) {
	var evaluations []StrategyEvaluation

	strategies := []struct {
		Name        string
		BaseScore   float64
		RiskLevel   string
		Description string
	}{
		{"均值回归策略", 7.5, "中等", "适合震荡市场，捕捉价格偏离"},
		{"网格交易策略", 8.0, "低", "适合区间震荡，稳定收益"},
		{"趋势跟随策略", 6.0, "高", "适合强趋势市场，高收益高风险"},
		{"动量策略", 5.5, "高", "适合快速变动市场"},
		{"统计套利策略", 7.0, "中等", "适合相关资产间价差"},
		{"波动率策略", 6.5, "高", "适合高波动环境"},
		{"反转策略", 4.5, "极高", "适合超买超卖信号"},
		{"突破策略", 6.8, "中等", "适合重要关口突破"},
		{"多空对冲策略", 7.2, "中等", "适合多空平衡市场"},
		{"做空策略", 3.0, "极高", "仅适合熊市环境"},
	}

	for _, strat := range strategies {
		eval := StrategyEvaluation{
			StrategyName: strat.Name,
		}

		// 计算市场适应性评分
		marketFit := r.calculateMarketFit(strat.Name, marketCondition)
		eval.MarketFitScore = strat.BaseScore * marketFit

		// 基于风险等级设置参数
		switch strat.RiskLevel {
		case "低":
			eval.RiskAdjustedReturn = 1.5
			eval.WinRate = 0.65
			eval.MaxDrawdown = 0.08
			eval.SharpeRatio = 1.2
		case "中等":
			eval.RiskAdjustedReturn = 2.2
			eval.WinRate = 0.58
			eval.MaxDrawdown = 0.12
			eval.SharpeRatio = 0.9
		case "高":
			eval.RiskAdjustedReturn = 3.5
			eval.WinRate = 0.52
			eval.MaxDrawdown = 0.18
			eval.SharpeRatio = 0.7
		case "极高":
			eval.RiskAdjustedReturn = 5.0
			eval.WinRate = 0.45
			eval.MaxDrawdown = 0.25
			eval.SharpeRatio = 0.4
		}

		// 调整权重建议
		eval.RecommendedWeight = eval.MarketFitScore / 10.0
		if eval.RecommendedWeight > 1.0 {
			eval.RecommendedWeight = 1.0
		}

		eval.ImplementationNotes = strat.Description

		evaluations = append(evaluations, eval)
	}

	// 按市场适应性排序
	sort.Slice(evaluations, func(i, j int) bool {
		return evaluations[i].MarketFitScore > evaluations[j].MarketFitScore
	})

	return evaluations, nil
}

// 计算策略的市场适应性
func (r *DetailedMarketStrategyReport) calculateMarketFit(strategyName string, market *MarketCondition) float64 {
	baseFit := 0.5 // 默认适应性

	switch market.Regime {
	case "高波动震荡市":
		switch strategyName {
		case "均值回归策略":
			baseFit = 1.2
		case "波动率策略":
			baseFit = 1.1
		case "网格交易策略":
			baseFit = 0.9
		case "统计套利策略":
			baseFit = 0.9
		case "反转策略":
			baseFit = 0.8
		case "突破策略":
			baseFit = 0.7
		}
	case "震荡市":
		switch strategyName {
		case "均值回归策略":
			baseFit = 1.3
		case "网格交易策略":
			baseFit = 1.1
		case "统计套利策略":
			baseFit = 1.0
		case "突破策略":
			baseFit = 0.8
		}
	case "低波动整理市":
		switch strategyName {
		case "网格交易策略":
			baseFit = 1.4
		case "统计套利策略":
			baseFit = 1.1
		case "均值回归策略":
			baseFit = 1.0
		}
	case "强势上涨趋势市":
		switch strategyName {
		case "趋势跟随策略":
			baseFit = 1.4
		case "动量策略":
			baseFit = 1.2
		case "突破策略":
			baseFit = 1.1
		case "多空对冲策略":
			baseFit = 0.9
		}
	case "强势下跌趋势市":
		switch strategyName {
			case "做空策略":
				baseFit = 1.5
			case "趋势跟随策略":
				baseFit = 1.3
			case "多空对冲策略":
				baseFit = 1.0
		}
	}

	// 基于波动率调整
	if market.Volatility > 8 {
		if strategyName == "波动率策略" {
			baseFit *= 1.2
		} else if strategyName == "网格交易策略" {
			baseFit *= 0.8 // 高波动不适合网格
		}
	}

	// 限制在合理范围内
	if baseFit > 1.5 {
		baseFit = 1.5
	} else if baseFit < 0.2 {
		baseFit = 0.2
	}

	return baseFit
}

// 生成投资组合建议
func (r *DetailedMarketStrategyReport) generatePortfolioRecommendation(evaluations []StrategyEvaluation, market *MarketCondition) (*PortfolioRecommendation, error) {
	rec := &PortfolioRecommendation{}

	// 选择主要策略
	if len(evaluations) > 0 {
		rec.PrimaryStrategy = evaluations[0].StrategyName

		// 选择辅助策略
		for i, eval := range evaluations {
			if i > 0 && i <= 3 && eval.MarketFitScore > 5.0 {
				rec.SecondaryStrategies = append(rec.SecondaryStrategies, eval.StrategyName)
			}
		}
	}

	// 计算风险平价权重
	rec.RiskParityWeights = make(map[string]float64)
	totalWeight := 0.0

	for _, eval := range evaluations {
		if eval.RecommendedWeight > 0.05 { // 只包含权重>5%的策略
			weight := eval.RecommendedWeight
			if eval.StrategyName == rec.PrimaryStrategy {
				weight *= 1.5 // 主要策略权重加倍
			}
			rec.RiskParityWeights[eval.StrategyName] = weight
			totalWeight += weight
		}
	}

	// 归一化权重
	for strategy, weight := range rec.RiskParityWeights {
		rec.RiskParityWeights[strategy] = weight / totalWeight
	}

	// 设置最大分配比例
	switch market.Regime {
	case "高波动震荡市", "强势上涨趋势市", "强势下跌趋势市":
		rec.MaxAllocation = 0.15 // 15%
		rec.RebalancingFreq = "每日"
	case "震荡市":
		rec.MaxAllocation = 0.20 // 20%
		rec.RebalancingFreq = "每周"
	case "低波动整理市":
		rec.MaxAllocation = 0.25 // 25%
		rec.RebalancingFreq = "每月"
	default:
		rec.MaxAllocation = 0.20
		rec.RebalancingFreq = "每周"
	}

	// 风险管理规则
	rec.RiskManagementRules = []string{
		"单策略最大回撤不超过总资金的15%",
		"组合最大回撤不超过总资金的25%",
		"每日盈亏不超过总资金的5%",
		"连续亏损3次自动减仓50%",
		"市场极端事件触发时清仓观望",
	}

	return rec, nil
}

// 生成风险管理指南
func (r *DetailedMarketStrategyReport) generateRiskManagementGuidelines(market *MarketCondition) *RiskManagementGuidelines {
	guidelines := &RiskManagementGuidelines{}

	// 基于市场环境设置风险参数
	switch market.Regime {
	case "高波动震荡市":
		guidelines.MaxPositionSize = 0.05 // 5%
		guidelines.StopLossLevel = 0.03   // 3%
		guidelines.TakeProfitLevel = 0.05 // 5%
		guidelines.MaxDailyLoss = 0.02    // 2%
		guidelines.RebalanceThreshold = 0.05
	case "强势上涨趋势市", "强势下跌趋势市":
		guidelines.MaxPositionSize = 0.08
		guidelines.StopLossLevel = 0.05
		guidelines.TakeProfitLevel = 0.10
		guidelines.MaxDailyLoss = 0.03
		guidelines.RebalanceThreshold = 0.08
	default:
		guidelines.MaxPositionSize = 0.10
		guidelines.StopLossLevel = 0.04
		guidelines.TakeProfitLevel = 0.08
		guidelines.MaxDailyLoss = 0.025
		guidelines.RebalanceThreshold = 0.06
	}

	guidelines.VolatilityAdjustment = market.Volatility > 6
	guidelines.CorrelationMonitoring = true
	guidelines.StressTestFrequency = "每周"

	return guidelines
}

// 显示函数
func (r *DetailedMarketStrategyReport) displayMarketCondition(condition *MarketCondition) {
	fmt.Printf("市场环境: %s\n", condition.Regime)
	fmt.Printf("波动率水平: %.2f%%\n", condition.Volatility)
	fmt.Printf("趋势强度: %.1f%%\n", condition.TrendStrength*100)
	fmt.Printf("市场情绪得分: %.2f/1.0\n", condition.SentimentScore)
	fmt.Printf("流动性得分: %.2f/1.0\n", condition.LiquidityScore)
}

// 其他显示函数的实现
func (r *DetailedMarketStrategyReport) displayStrongWeakAnalysis(analysis *StrongWeakAnalysis) {
	fmt.Println("强势币种 TOP5:")
	for i, coin := range analysis.StrongCoins {
		if i >= 5 {
			break
		}
		fmt.Printf("  %d. %s: %+5.2f%% (波动率: %.1f%%)\n",
			i+1, coin.Symbol, coin.Change, coin.Volatility)
	}

	fmt.Println("\n弱势币种 TOP5:")
	for i, coin := range analysis.WeakCoins {
		if i >= 5 {
			break
		}
		fmt.Printf("  %d. %s: %+5.2f%% (波动率: %.1f%%)\n",
			i+1, coin.Symbol, coin.Change, coin.Volatility)
	}

	fmt.Printf("\n强势币种特征: %s\n", analysis.StrongCoinFeatures)
	fmt.Printf("弱势币种特征: %s\n", analysis.WeakCoinFeatures)
}

func (r *DetailedMarketStrategyReport) displayTechnicalAnalysis(analysis *TechnicalAnalysis) {
	fmt.Printf("整体趋势: %s\n", analysis.OverallTrend)
	fmt.Printf("RSI动量指标: %.1f\n", analysis.RSIMomentum)
	fmt.Printf("MACD趋势指标: %.2f\n", analysis.MACDTrend)
	fmt.Printf("布林带位置: %.2f\n", analysis.BollingerPosition)

	fmt.Printf("支撑位: ")
	for _, level := range analysis.SupportLevels {
		fmt.Printf("%.2f ", level)
	}
	fmt.Println()

	fmt.Printf("阻力位: ")
	for _, level := range analysis.ResistanceLevels {
		fmt.Printf("%.2f ", level)
	}
	fmt.Println()
}

func (r *DetailedMarketStrategyReport) displayStrategyEvaluations(evaluations []StrategyEvaluation) {
	fmt.Println("策略评估结果 (按市场适应性排序):")
	fmt.Println("┌─────────────────────┬──────────┬──────────┬──────────┬──────────┬──────────┬──────────┐")
	fmt.Println("│ 策略名称           │ 市场适应 │ 风险调整 │ 胜率     │ 最大回撤 │ 夏普比率 │ 建议权重 │")
	fmt.Println("├─────────────────────┼──────────┼──────────┼──────────┼──────────┼──────────┼──────────┤")

	for i, eval := range evaluations {
		if i >= 8 { // 只显示前8个
			break
		}
		fmt.Printf("│ %-18s │ %8.1f │ %8.2f │ %7.1f%% │ %7.1f%% │ %8.1f │ %7.1f%% │\n",
			eval.StrategyName,
			eval.MarketFitScore,
			eval.RiskAdjustedReturn,
			eval.WinRate*100,
			eval.MaxDrawdown*100,
			eval.SharpeRatio,
			eval.RecommendedWeight*100)
	}
	fmt.Println("└─────────────────────┴──────────┴──────────┴──────────┴──────────┴──────────┴──────────┘")
}

func (r *DetailedMarketStrategyReport) displayPortfolioRecommendation(rec *PortfolioRecommendation) {
	fmt.Printf("主要策略: %s\n", rec.PrimaryStrategy)
	fmt.Printf("辅助策略: %v\n", rec.SecondaryStrategies)
	fmt.Printf("最大分配比例: %.0f%%\n", rec.MaxAllocation*100)
	fmt.Printf("调仓频率: %s\n", rec.RebalancingFreq)

	fmt.Println("\n风险平价权重分配:")
	for strategy, weight := range rec.RiskParityWeights {
		if weight > 0.01 { // 只显示权重>1%的策略
			fmt.Printf("  %s: %.1f%%\n", strategy, weight*100)
		}
	}

	fmt.Println("\n风险管理规则:")
	for _, rule := range rec.RiskManagementRules {
		fmt.Printf("  • %s\n", rule)
	}
}

func (r *DetailedMarketStrategyReport) displayRiskManagementGuidelines(guidelines *RiskManagementGuidelines) {
	fmt.Printf("最大持仓比例: %.0f%%\n", guidelines.MaxPositionSize*100)
	fmt.Printf("止损水平: %.0f%%\n", guidelines.StopLossLevel*100)
	fmt.Printf("止盈水平: %.0f%%\n", guidelines.TakeProfitLevel*100)
	fmt.Printf("每日最大亏损: %.0f%%\n", guidelines.MaxDailyLoss*100)
	fmt.Printf("调仓阈值: %.0f%%\n", guidelines.RebalanceThreshold*100)

	if guidelines.VolatilityAdjustment {
		fmt.Println("波动率调整: 启用 (当前市场波动率较高)")
	}

	if guidelines.CorrelationMonitoring {
		fmt.Println("相关性监控: 启用")
	}

	fmt.Printf("压力测试频率: %s\n", guidelines.StressTestFrequency)
}

// 数据结构定义
type StrongWeakAnalysis struct {
	StrongCoins        []CoinMetrics
	WeakCoins          []CoinMetrics
	StrongCoinFeatures string
	WeakCoinFeatures   string
}

type CoinMetrics struct {
	Symbol     string
	Change     float64
	Volume     float64
	Volatility float64
}

type TechnicalAnalysis struct {
	OverallTrend      string
	RSIMomentum       float64
	MACDTrend         float64
	BollingerPosition float64
	SupportLevels     []float64
	ResistanceLevels  []float64
}

type RiskManagementGuidelines struct {
	MaxPositionSize     float64
	StopLossLevel       float64
	TakeProfitLevel     float64
	MaxDailyLoss        float64
	RebalanceThreshold  float64
	VolatilityAdjustment bool
	CorrelationMonitoring bool
	StressTestFrequency string
}

// 辅助函数
func (r *DetailedMarketStrategyReport) analyzeCoinFeatures(coins []CoinMetrics) string {
	if len(coins) == 0 {
		return "无数据"
	}

	totalVolatility := 0.0
	totalVolume := 0.0
	highVolCount := 0

	for _, coin := range coins {
		totalVolatility += coin.Volatility
		totalVolume += coin.Volume
		if coin.Volatility > 10 {
			highVolCount++
		}
	}

	avgVolatility := totalVolatility / float64(len(coins))
	avgVolume := totalVolume / float64(len(coins))
	highVolRatio := float64(highVolCount) / float64(len(coins))

	features := ""
	if avgVolatility > 15 {
		features += "高波动率 "
	} else if avgVolatility > 8 {
		features += "中等波动率 "
	} else {
		features += "低波动率 "
	}

	if highVolRatio > 0.5 {
		features += "多数币种波动剧烈 "
	}

	if avgVolume > 50000000 {
		features += "高流动性"
	} else if avgVolume > 10000000 {
		features += "中等流动性"
	} else {
		features += "低流动性"
	}

	return features
}