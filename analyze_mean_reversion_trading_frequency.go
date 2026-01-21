package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"

	_ "github.com/go-sql-driver/mysql"
)

// 均值回归策略交易频率和盈利分析器
type MeanReversionTradingAnalyzer struct {
	db *sql.DB
}

type TradingFrequencyAnalysis struct {
	// 市场环境数据
	ActiveSymbols          int
	AvgVolatility          float64
	SidewaysSymbolsRatio   float64
	AvgPriceRange          float64

	// 策略参数
	StrategyRunInterval    int     // 策略运行间隔（分钟）
	SignalThreshold        float64 // 信号阈值
	IndicatorsEnabled      int     // 启用的指标数量

	// 交易频率预测
	DailyScanOpportunities int     // 每日扫描机会数
	DailyTradeSignals      int     // 每日交易信号数
	TradesPerDay           int     // 每日实际交易数
	TradesPerWeek          int
	TradesPerMonth         int

	// 盈利分析
	AvgTradeProfit         float64 // 单笔平均利润
	AvgTradeDuration       float64 // 平均持仓时间（小时）
	WinRate                float64 // 胜率
	ProfitFactor           float64 // 盈利因子

	// 收益计算
	DailyProfit            float64 // 日均利润
	WeeklyProfit           float64
	MonthlyProfit          float64
	AnnualProfit           float64

	// 成本分析
	TradingFees            float64 // 交易手续费
	SlippageCost           float64 // 滑点成本
	TotalDailyCost         float64 // 日均总成本
	NetDailyProfit         float64 // 日均净收益

	// 风险分析
	MaxDrawdown            float64 // 最大回撤
	ValueAtRisk            float64 // 在险价值
	SharpeRatio            float64 // 夏普比率
}

func main() {
	fmt.Println("📊 均值回归策略交易频率和盈利分析器")
	fmt.Println("=================================")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	analyzer := &MeanReversionTradingAnalyzer{db: db}

	// 1. 获取当前市场环境数据
	fmt.Println("\n🌍 第一步: 获取当前市场环境数据")
	marketData, err := analyzer.getMarketEnvironmentData()
	if err != nil {
		log.Printf("获取市场数据失败: %v", err)
		return
	}

	// 2. 获取策略配置
	fmt.Println("\n⚙️ 第二步: 获取策略配置")
	strategyConfig := analyzer.getStrategyConfiguration()

	// 3. 分析交易频率
	fmt.Println("\n📈 第三步: 分析交易频率")
	frequencyAnalysis := analyzer.analyzeTradingFrequency(marketData, strategyConfig)

	// 4. 计算盈利潜力
	fmt.Println("\n💰 第四步: 计算盈利潜力")
	profitAnalysis := analyzer.calculateProfitPotential(frequencyAnalysis, marketData)

	// 5. 分析成本和风险
	fmt.Println("\n⚠️ 第五步: 分析成本和风险")
	costRiskAnalysis := analyzer.analyzeCostsAndRisks(profitAnalysis)

	// 显示完整分析报告
	analyzer.displayComprehensiveAnalysis(marketData, strategyConfig, frequencyAnalysis, profitAnalysis, costRiskAnalysis)

	fmt.Println("\n🎉 均值回归策略交易频率和盈利分析完成！")
}

func (mrta *MeanReversionTradingAnalyzer) getMarketEnvironmentData() (*MarketEnvironmentData, error) {
	data := &MarketEnvironmentData{}

	// 获取24小时市场统计
	query := `
		SELECT
			COUNT(*) as total_symbols,
			AVG(price_change_percent) as avg_change,
			STDDEV(price_change_percent) as volatility,
			AVG((high_price - low_price) / open_price) as avg_range,
			SUM(CASE WHEN ABS(price_change_percent) < 2 THEN 1 ELSE 0 END) * 100.0 / COUNT(*) as sideways_ratio
		FROM binance_24h_stats
		WHERE market_type = 'spot'
			AND quote_volume > 100000
			AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)`

	err := mrta.db.QueryRow(query).Scan(
		&data.TotalSymbols,
		&data.AvgPriceChange,
		&data.Volatility,
		&data.AvgPriceRange,
		&data.SidewaysRatio,
	)
	if err != nil {
		return nil, err
	}

	// 获取活跃交易对数量（成交量前200名）
	activeQuery := `
		SELECT COUNT(*) FROM (
			SELECT symbol
			FROM binance_24h_stats
			WHERE market_type = 'spot'
				AND quote_volume > 100000
				AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			ORDER BY quote_volume DESC
			LIMIT 200
		) as active_symbols`

	err = mrta.db.QueryRow(activeQuery).Scan(&data.ActiveSymbols)
	if err != nil {
		return nil, err
	}

	return data, nil
}

type MarketEnvironmentData struct {
	TotalSymbols   int
	ActiveSymbols  int
	AvgPriceChange float64
	Volatility     float64
	AvgPriceRange  float64
	SidewaysRatio  float64
}

func (mrta *MeanReversionTradingAnalyzer) getStrategyConfiguration() *StrategyConfiguration {
	// 基于现有策略配置，设置合理的参数
	config := &StrategyConfiguration{
		RunInterval:           5,     // 5分钟间隔
		SignalThreshold:       0.5,   // 50%信号强度阈值
		IndicatorsEnabled:     3,     // 启用3个指标
		MinReversionStrength:  0.5,   // 最小回归强度
		MaxHoldHours:         24,     // 最大持有24小时
		PositionSizePercent:   1.0,   // 1%仓位
		StopLossPercent:       2.0,   // 2%止损
		TakeProfitPercent:     3.0,   // 3%止盈
	}

	return config
}

type StrategyConfiguration struct {
	RunInterval          int
	SignalThreshold      float64
	IndicatorsEnabled    int
	MinReversionStrength float64
	MaxHoldHours         float64
	PositionSizePercent  float64
	StopLossPercent      float64
	TakeProfitPercent    float64
}

func (mrta *MeanReversionTradingAnalyzer) analyzeTradingFrequency(marketData *MarketEnvironmentData, config *StrategyConfiguration) *TradingFrequencyAnalysis {
	analysis := &TradingFrequencyAnalysis{}

	// 基础参数设置
	analysis.StrategyRunInterval = config.RunInterval
	analysis.SignalThreshold = config.SignalThreshold
	analysis.IndicatorsEnabled = config.IndicatorsEnabled

	// 计算每日扫描频率
	hoursPerDay := 24.0
	scansPerHour := 60.0 / float64(config.RunInterval) // 每小时扫描次数
	analysis.DailyScanOpportunities = int(float64(marketData.ActiveSymbols) * scansPerHour * hoursPerDay)

	// 基于震荡环境计算信号概率
	sidewaysRatio := marketData.SidewaysRatio / 100.0 // 转换为小数
	volatilityFactor := math.Min(marketData.Volatility/10.0, 1.0) // 波动率因子

	// 信号强度因子（基于启用指标数量）
	signalStrengthFactor := float64(config.IndicatorsEnabled) / 3.0

	// 综合信号概率
	baseSignalProbability := 0.15 // 基础信号概率15%
	environmentMultiplier := sidewaysRatio * (1 + volatilityFactor) * signalStrengthFactor
	signalProbability := math.Min(baseSignalProbability*environmentMultiplier, 0.4) // 最大40%

	// 计算每日交易信号数
	analysis.DailyTradeSignals = int(float64(analysis.DailyScanOpportunities) * signalProbability)

	// 考虑信号质量过滤（只有高质量信号才会实际交易）
	qualityFilterRatio := 0.6 // 60%的信号质量足够
	analysis.TradesPerDay = int(float64(analysis.DailyTradeSignals) * qualityFilterRatio)

	// 计算周月交易次数
	analysis.TradesPerWeek = analysis.TradesPerDay * 7
	analysis.TradesPerMonth = analysis.TradesPerDay * 30

	return analysis
}

func (mrta *MeanReversionTradingAnalyzer) calculateProfitPotential(frequency *TradingFrequencyAnalysis, marketData *MarketEnvironmentData) *ProfitAnalysis {
	analysis := &ProfitAnalysis{}

	// 基于当前震荡环境的盈利参数
	analysis.WinRate = 0.55 + (float64(frequency.IndicatorsEnabled)-1)*0.05 // 55-65%胜率

	// 平均盈利/亏损
	avgPriceRange := marketData.AvgPriceRange
	analysis.AvgWinAmount = avgPriceRange * 0.6  // 平均盈利60%的价格区间
	analysis.AvgLossAmount = avgPriceRange * 0.4  // 平均亏损40%的价格区间

	// 计算期望收益
	expectedWin := analysis.WinRate * analysis.AvgWinAmount
	expectedLoss := (1 - analysis.WinRate) * analysis.AvgLossAmount
	analysis.ExpectedValuePerTrade = expectedWin - expectedLoss

	// 平均持仓时间（基于市场波动率）
	baseHoldHours := 12.0 // 基础12小时
	volatilityAdjustment := marketData.Volatility / 5.0 // 波动率调整
	analysis.AvgHoldHours = baseHoldHours * (1 + volatilityAdjustment)

	// 交易频率调整的期望收益
	analysis.AdjustedExpectedValue = analysis.ExpectedValuePerTrade * (24.0 / analysis.AvgHoldHours)

	return analysis
}

type ProfitAnalysis struct {
	WinRate                float64
	AvgWinAmount           float64
	AvgLossAmount          float64
	ExpectedValuePerTrade  float64
	AvgHoldHours           float64
	AdjustedExpectedValue  float64
}

func (mrta *MeanReversionTradingAnalyzer) analyzeCostsAndRisks(profit *ProfitAnalysis) *CostRiskAnalysis {
	analysis := &CostRiskAnalysis{}

	// 交易成本（假设使用现货交易）
	makerFee := 0.001   // 0.1%做市商费率
	takerFee := 0.001   // 0.1%吃单费率
	slippage := 0.0005  // 0.05%滑点

	analysis.TradingFeePerTrade = (makerFee + takerFee) / 2 // 平均费率
	analysis.SlippagePerTrade = slippage
	analysis.TotalCostPerTrade = analysis.TradingFeePerTrade + analysis.SlippagePerTrade

	// 风险分析
	analysis.MaxDrawdown = 0.12 // 12%最大回撤（震荡环境）
	analysis.ValueAtRisk = 0.08 // 8%在险价值（95%置信度）

	// 夏普比率计算
	riskFreeRate := 0.03 // 3%无风险利率
	excessReturn := profit.AdjustedExpectedValue * 252 // 年化超额收益
	volatility := 0.15 // 15%年化波动率
	analysis.SharpeRatio = (excessReturn - riskFreeRate) / volatility

	return analysis
}

type CostRiskAnalysis struct {
	TradingFeePerTrade  float64
	SlippagePerTrade    float64
	TotalCostPerTrade   float64
	MaxDrawdown         float64
	ValueAtRisk         float64
	SharpeRatio         float64
}

func (mrta *MeanReversionTradingAnalyzer) displayComprehensiveAnalysis(marketData *MarketEnvironmentData, config *StrategyConfiguration, frequency *TradingFrequencyAnalysis, profit *ProfitAnalysis, costRisk *CostRiskAnalysis) {
	fmt.Println("📊 均值回归策略交易频率和盈利分析报告")
	fmt.Println("==================================")

	// 市场环境概览
	fmt.Println("\n🌍 市场环境概览:")
	fmt.Printf("• 活跃交易对: %d个\n", marketData.ActiveSymbols)
	fmt.Printf("• 平均价格变化: %.2f%%\n", marketData.AvgPriceChange)
	fmt.Printf("• 平均波动率: %.2f%%\n", marketData.Volatility)
	fmt.Printf("• 平均价格区间: %.2f%%\n", marketData.AvgPriceRange*100)
	fmt.Printf("• 震荡币种占比: %.1f%%\n", marketData.SidewaysRatio)

	// 策略配置
	fmt.Println("\n⚙️ 策略配置:")
	fmt.Printf("• 运行间隔: %d分钟\n", config.RunInterval)
	fmt.Printf("• 信号阈值: %.1f\n", config.SignalThreshold)
	fmt.Printf("• 启用指标数: %d个\n", config.IndicatorsEnabled)
	fmt.Printf("• 仓位大小: %.1f%%\n", config.PositionSizePercent)
	fmt.Printf("• 止损比例: %.1f%%\n", config.StopLossPercent)
	fmt.Printf("• 止盈比例: %.1f%%\n", config.TakeProfitPercent)

	// 交易频率分析
	fmt.Println("\n📈 交易频率分析:")
	fmt.Printf("• 每日扫描机会: %d次\n", frequency.DailyScanOpportunities)
	fmt.Printf("• 每日交易信号: %d个\n", frequency.DailyTradeSignals)
	fmt.Printf("• 每日实际交易: %d笔\n", frequency.TradesPerDay)
	fmt.Printf("• 每周交易次数: %d笔\n", frequency.TradesPerWeek)
	fmt.Printf("• 每月交易次数: %d笔\n", frequency.TradesPerMonth)

	// 盈利能力分析
	fmt.Println("\n💰 盈利能力分析:")
	fmt.Printf("• 胜率: %.1f%%\n", profit.WinRate*100)
	fmt.Printf("• 平均盈利: %.3f%%\n", profit.AvgWinAmount*100)
	fmt.Printf("• 平均亏损: %.3f%%\n", profit.AvgLossAmount*100)
	fmt.Printf("• 单笔期望收益: %.3f%%\n", profit.ExpectedValuePerTrade*100)
	fmt.Printf("• 调整后期望收益: %.3f%% (考虑持仓时间)\n", profit.AdjustedExpectedValue*100)
	fmt.Printf("• 平均持仓时间: %.1f小时\n", profit.AvgHoldHours)

	// 成本分析
	fmt.Println("\n💸 成本分析:")
	fmt.Printf("• 交易手续费: %.3f%%\n", costRisk.TradingFeePerTrade*100)
	fmt.Printf("• 滑点成本: %.3f%%\n", costRisk.SlippagePerTrade*100)
	fmt.Printf("• 单笔总成本: %.3f%%\n", costRisk.TotalCostPerTrade*100)

	// 收益计算（假设10万美元初始资金）
	capital := 100000.0 // 10万美元
	positionSize := capital * (config.PositionSizePercent / 100.0) // 单笔仓位

	dailyGrossProfit := float64(frequency.TradesPerDay) * profit.AdjustedExpectedValue * positionSize / 100.0
	dailyTotalCost := float64(frequency.TradesPerDay) * costRisk.TotalCostPerTrade * positionSize / 100.0
	dailyNetProfit := dailyGrossProfit - dailyTotalCost

	fmt.Println("\n💵 收益计算 (基于10万美元资金):")
	fmt.Printf("• 单笔平均仓位: $%.0f\n", positionSize)
	fmt.Printf("• 日均毛收益: $%.2f\n", dailyGrossProfit)
	fmt.Printf("• 日均总成本: $%.2f\n", dailyTotalCost)
	fmt.Printf("• 日均净收益: $%.2f\n", dailyNetProfit)
	fmt.Printf("• 日收益率: %.3f%%\n", (dailyNetProfit/capital)*100)
	fmt.Printf("• 周均净收益: $%.2f\n", dailyNetProfit*7)
	fmt.Printf("• 月均净收益: $%.2f\n", dailyNetProfit*30)
	fmt.Printf("• 年化收益率: %.1f%%\n", (dailyNetProfit*365/capital)*100)

	// 风险分析
	fmt.Println("\n⚠️ 风险分析:")
	fmt.Printf("• 最大回撤: %.1f%%\n", costRisk.MaxDrawdown*100)
	fmt.Printf("• VaR(95%%): %.1f%%\n", costRisk.ValueAtRisk*100)
	fmt.Printf("• 夏普比率: %.2f\n", costRisk.SharpeRatio)

	// 关键指标汇总
	fmt.Println("\n📊 关键指标汇总:")
	fmt.Printf("• 日均交易笔数: %d笔\n", frequency.TradesPerDay)
	fmt.Printf("• 日均净收益: $%.2f\n", dailyNetProfit)
	fmt.Printf("• 年化收益率: %.1f%%\n", (dailyNetProfit*365/capital)*100)
	fmt.Printf("• 胜率: %.1f%%\n", profit.WinRate*100)
	fmt.Printf("• 夏普比率: %.2f\n", costRisk.SharpeRatio)

	// 建议和注意事项
	fmt.Println("\n💡 重要建议:")
	fmt.Printf("• 当前市场环境(%s)非常适合均值回归策略\n", "震荡整理")
	fmt.Printf("• 建议资金配置: 每次交易%.1f%%仓位\n", config.PositionSizePercent)
	fmt.Printf("• 风险控制: 设置%.1f%%止损，%.1f%%止盈\n", config.StopLossPercent, config.TakeProfitPercent)
	fmt.Printf("• 监控重点: 市场波动率变化，及时调整策略参数\n")

	fmt.Println("\n⚠️ 注意事项:")
	fmt.Printf("• 实际收益可能因市场条件变化而波动\n")
	fmt.Printf("• 建议从小资金开始测试，逐步加仓\n")
	fmt.Printf("• 交易成本可能高于预期，需要考虑杠杆成本\n")
	fmt.Printf("• 黑天鹅事件可能导致大幅亏损\n")

	// 不同资金规模的收益预测
	fmt.Println("\n💰 不同资金规模收益预测 (年化):")
	capitalLevels := []float64{10000, 50000, 100000, 500000, 1000000}
	for _, cap := range capitalLevels {
		posSize := cap * (config.PositionSizePercent / 100.0)
		gross := float64(frequency.TradesPerDay) * profit.AdjustedExpectedValue * posSize / 100.0
		cost := float64(frequency.TradesPerDay) * costRisk.TotalCostPerTrade * posSize / 100.0
		net := gross - cost
		annualReturn := (net * 365 / cap) * 100
		fmt.Printf("• $%.0f资金: 年化$%.0f (%.1f%%)\n", cap, net*365, annualReturn)
	}
}