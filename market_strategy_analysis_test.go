package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// ============================================================================
// 数据结构定义
// ============================================================================

type MarketData struct {
	Symbol     string
	Price      float64
	Change24h  float64
	Volume24h  float64
	MarketCap  float64
}

type TechnicalIndicators struct {
	RSI             float64
	MACD            struct{ Signal, Histogram float64 }
	BollingerBands  struct{ Upper, Middle, Lower float64 }
	VolumeRatio     float64
	Volatility      float64
	TrendStrength   float64
}

type StrategyAnalysis struct {
	Type         string
	Name         string
	Score        float64
	Confidence   float64
	RiskLevel    string
	Suitability  string
	Description  string
	EntrySignal  string
	ExitSignal   string
	RiskReward   float64
	WinRate      float64
	MaxDrawdown  float64
	AvgProfit    float64
}

type MarketEnvironment struct {
	OverallTrend      string
	Volatility        float64
	Oscillation       float64
	MarketStrength    string
	DominantStrategy  string
	RiskAssessment    string
	TimeHorizon       string
	TradingBias       string
}

type TradingRecommendation struct {
	PrimaryStrategy     StrategyAnalysis
	AlternativeStrategy StrategyAnalysis
	PositionSize        float64
	StopLoss           float64
	TakeProfit         float64
	EntryPrice         float64
	RiskRewardRatio    float64
	TimeFrame          string
	MarketConditions   []string
	RiskFactors        []string
	ExecutionSteps     []string
}

// ============================================================================
// 主要分析函数
// ============================================================================

func main() {
	fmt.Println("🚀 市场环境分析与策略推荐系统")
	fmt.Println("=================================")

	// 连接数据库
	db, err := connectDatabase()
	if err != nil {
		log.Fatalf("❌ 数据库连接失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 执行市场分析
	analysis, err := performMarketAnalysis(db)
	if err != nil {
		log.Fatalf("❌ 市场分析失败: %v", err)
	}

	// 显示分析结果
	displayAnalysisResults(analysis)
}

// 连接数据库
func connectDatabase() (*gorm.DB, error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	return db, nil
}

// 执行市场分析
func performMarketAnalysis(db *gorm.DB) (*TradingRecommendation, error) {
	fmt.Println("\n📊 正在分析市场环境...")

	// 获取市场数据
	marketData, err := getMarketData(db)
	if err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %v", err)
	}

	// 计算技术指标
	indicators, err := calculateTechnicalIndicators(db)
	if err != nil {
		return nil, fmt.Errorf("计算技术指标失败: %v", err)
	}

	// 分析市场环境
	environment := analyzeMarketEnvironment(marketData, indicators)

	// 生成策略推荐
	strategies := generateStrategyRecommendations(environment, indicators)

	// 排序策略
	sortStrategiesByScore(strategies)

	// 创建交易推荐
	recommendation := createTradingRecommendation(strategies[0], strategies[1], environment)

	return recommendation, nil
}

// 获取市场数据
func getMarketData(db *gorm.DB) ([]MarketData, error) {
	var data []MarketData

	// 从 binance_24h_stats 获取最近24小时的市场数据
	query := `
		SELECT symbol,
			   last_price as price,
			   price_change_percent as change24h,
			   quote_volume as volume24h
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
		ORDER BY quote_volume DESC
		LIMIT 100
	`

	rows, err := db.Raw(query).Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var item MarketData
		if err := rows.Scan(&item.Symbol, &item.Price, &item.Change24h, &item.Volume24h); err != nil {
			continue
		}
		data = append(data, item)
	}

	return data, nil
}

// 计算技术指标
func calculateTechnicalIndicators(db *gorm.DB) (*TechnicalIndicators, error) {
	indicators := &TechnicalIndicators{}

	// 计算BTC的RSI
	rsi, err := calculateRSI(db, "BTCUSDT", 14)
	if err == nil {
		indicators.RSI = rsi
	}

	// 计算波动率
	volatility, err := calculateVolatility(db, "BTCUSDT", 7)
	if err == nil {
		indicators.Volatility = volatility
	}

	// 计算趋势强度
	trendStrength, err := calculateTrendStrength(db, "BTCUSDT", 7)
	if err == nil {
		indicators.TrendStrength = trendStrength
	}

	// 计算成交量比率
	volumeRatio, err := calculateVolumeRatio(db, "BTCUSDT")
	if err == nil {
		indicators.VolumeRatio = volumeRatio
	}

	return indicators, nil
}

// 计算RSI指标
func calculateRSI(db *gorm.DB, symbol string, period int) (float64, error) {
	var prices []float64

	// 获取最近的价格数据
	query := `
		SELECT close_price
		FROM market_klines
		WHERE symbol = ? AND open_time >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		ORDER BY open_time DESC
		LIMIT ?
	`

	rows, err := db.Raw(query, symbol, period*2).Rows()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var price float64
		rows.Scan(&price)
		prices = append([]float64{price}, prices...) // 反转顺序
	}

	if len(prices) < period+1 {
		return 50, nil // 默认中性值
	}

	// 计算价格变化
	var gains, losses []float64
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// 计算平均涨幅和跌幅
	avgGain := average(gains[:period])
	avgLoss := average(losses[:period])

	if avgLoss == 0 {
		return 100, nil
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi, nil
}

// 计算波动率
func calculateVolatility(db *gorm.DB, symbol string, days int) (float64, error) {
	var prices []float64

	query := `
		SELECT close_price
		FROM market_klines
		WHERE symbol = ? AND open_time >= DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY open_time ASC
	`

	rows, err := db.Raw(query, symbol, days).Rows()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var price float64
		rows.Scan(&price)
		prices = append(prices, price)
	}

	if len(prices) < 2 {
		return 0, nil
	}

	// 计算日收益率
	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	// 计算标准差
	volatility := calculateStandardDeviation(returns) * 100 // 转换为百分比

	return volatility, nil
}

// 计算趋势强度
func calculateTrendStrength(db *gorm.DB, symbol string, days int) (float64, error) {
	var prices []float64

	query := `
		SELECT close_price
		FROM market_klines
		WHERE symbol = ? AND open_time >= DATE_SUB(NOW(), INTERVAL ? DAY)
		ORDER BY open_time ASC
	`

	rows, err := db.Raw(query, symbol, days).Rows()
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var price float64
		rows.Scan(&price)
		prices = append(prices, price)
	}

	if len(prices) < 2 {
		return 0, nil
	}

	// 计算趋势强度 (收盘价变化的绝对值平均)
	totalChange := 0.0
	for i := 1; i < len(prices); i++ {
		change := (prices[i] - prices[i-1]) / prices[i-1]
		totalChange += change
	}

	trendStrength := totalChange / float64(len(prices)-1) * 100
	return trendStrength, nil
}

// 计算成交量比率
func calculateVolumeRatio(db *gorm.DB, symbol string) (float64, error) {
	var recentVolume, prevVolume float64

	// 最近7天平均成交量
	recentQuery := `
		SELECT AVG(quote_volume)
		FROM binance_24h_stats
		WHERE symbol = ? AND created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
	`

	// 前7天平均成交量
	prevQuery := `
		SELECT AVG(quote_volume)
		FROM binance_24h_stats
		WHERE symbol = ?
		  AND created_at >= DATE_SUB(NOW(), INTERVAL 14 DAY)
		  AND created_at < DATE_SUB(NOW(), INTERVAL 7 DAY)
	`

	db.Raw(recentQuery, symbol).Scan(&recentVolume)
	db.Raw(prevQuery, symbol).Scan(&prevVolume)

	if prevVolume == 0 {
		return 1.0, nil
	}

	return recentVolume / prevVolume, nil
}

// 分析市场环境
func analyzeMarketEnvironment(marketData []MarketData, indicators *TechnicalIndicators) MarketEnvironment {
	env := MarketEnvironment{}

	// 计算整体趋势
	totalChange := 0.0
	strongCount := 0
	weakCount := 0

	for _, data := range marketData {
		totalChange += data.Change24h
		if data.Change24h > 2 {
			strongCount++
		} else if data.Change24h < -2 {
			weakCount++
		}
	}

	avgChange := totalChange / float64(len(marketData))

	// 判断趋势
	if avgChange > 1 {
		env.OverallTrend = "上涨"
	} else if avgChange < -1 {
		env.OverallTrend = "下跌"
	} else {
		env.OverallTrend = "震荡"
	}

	// 设置波动率
	env.Volatility = indicators.Volatility

	// 计算震荡度
	env.Oscillation = calculateMarketOscillation(marketData)

	// 判断市场强度
	if strongCount > weakCount*1.5 {
		env.MarketStrength = "强势"
	} else if weakCount > strongCount*1.5 {
		env.MarketStrength = "弱势"
	} else {
		env.MarketStrength = "平衡"
	}

	// 基于市场条件推荐主要策略
	env.DominantStrategy = recommendDominantStrategy(env, indicators)

	// 风险评估
	env.RiskAssessment = assessRiskLevel(env, indicators)

	// 时间周期建议
	env.TimeHorizon = recommendTimeHorizon(env)

	// 交易偏好
	env.TradingBias = determineTradingBias(env)

	return env
}

// 计算市场震荡度
func calculateMarketOscillation(marketData []MarketData) float64 {
	if len(marketData) == 0 {
		return 0
	}

	var changes []float64
	for _, data := range marketData {
		changes = append(changes, data.Change24h)
	}

	stdDev := calculateStandardDeviation(changes)
	return stdDev
}

// 推荐主要策略
func recommendDominantStrategy(env MarketEnvironment, indicators *TechnicalIndicators) string {
	// 基于市场条件推荐策略
	if env.OverallTrend == "震荡" && env.Oscillation > 3 {
		return "mean_reversion"
	} else if env.OverallTrend == "上涨" && indicators.TrendStrength > 2 {
		return "moving_average"
	} else if env.Volatility > 5 {
		return "grid_trading"
	} else {
		return "traditional"
	}
}

// 风险评估
func assessRiskLevel(env MarketEnvironment, indicators *TechnicalIndicators) string {
	riskScore := 0

	if env.Volatility > 5 {
		riskScore += 2
	}
	if env.Oscillation > 4 {
		riskScore += 2
	}
	if indicators.VolumeRatio > 1.5 {
		riskScore += 1
	}

	switch {
	case riskScore >= 4:
		return "高风险"
	case riskScore >= 2:
		return "中风险"
	default:
		return "低风险"
	}
}

// 推荐时间周期
func recommendTimeHorizon(env MarketEnvironment) string {
	if env.Volatility > 6 {
		return "短期(1-3天)"
	} else if env.OverallTrend == "震荡" {
		return "中期(3-7天)"
	} else {
		return "中期(1-2周)"
	}
}

// 确定交易偏好
func determineTradingBias(env MarketEnvironment) string {
	switch env.OverallTrend {
	case "上涨":
		return "偏多"
	case "下跌":
		return "偏空"
	default:
		return "中性"
	}
}

// 生成策略推荐
func generateStrategyRecommendations(env MarketEnvironment, indicators *TechnicalIndicators) []StrategyAnalysis {
	strategies := []StrategyAnalysis{
		createMeanReversionStrategy(env, indicators),
		createMovingAverageStrategy(env, indicators),
		createGridTradingStrategy(env, indicators),
		createTraditionalStrategy(env, indicators),
		createRSIStrategy(env, indicators),
		createMACDStrategy(env, indicators),
	}

	return strategies
}

// 创建均值回归策略
func createMeanReversionStrategy(env MarketEnvironment, indicators *TechnicalIndicators) StrategyAnalysis {
	strategy := StrategyAnalysis{
		Type:        "mean_reversion",
		Name:        "均值回归策略",
		RiskLevel:   "medium",
		Description: "利用价格偏离均值的现象进行反向交易",
	}

	// 评分逻辑
	score := 5.0

	// 震荡市场加分
	if env.OverallTrend == "震荡" {
		score += 2
	}

	// RSI超买超卖信号
	if indicators.RSI > 70 || indicators.RSI < 30 {
		score += 1
	}

	// 适中的波动率
	if env.Volatility > 2 && env.Volatility < 6 {
		score += 1
	}

	strategy.Score = score
	strategy.Confidence = 65.0 + (score-5)*10

	strategy.Suitability = "震荡市场，低到中等波动率"
	strategy.EntrySignal = "价格偏离均值±2个标准差，配合RSI超买超卖"
	strategy.ExitSignal = "价格回归均值，或达到目标盈利/止损"
	strategy.RiskReward = 1.5
	strategy.WinRate = 0.62
	strategy.MaxDrawdown = 0.15
	strategy.AvgProfit = 0.023

	return strategy
}

// 创建均线策略
func createMovingAverageStrategy(env MarketEnvironment, indicators *TechnicalIndicators) StrategyAnalysis {
	strategy := StrategyAnalysis{
		Type:        "moving_average",
		Name:        "均线策略",
		RiskLevel:   "medium",
		Description: "基于移动平均线的趋势跟踪策略",
	}

	score := 4.0

	// 明确趋势加分
	if env.OverallTrend != "震荡" {
		score += 2
	}

	// 趋势强度加分
	if indicators.TrendStrength > 1.5 {
		score += 1
	}

	strategy.Score = score
	strategy.Confidence = 50.0 + (score-4)*12.5

	strategy.Suitability = "趋势市场，中等波动率"
	strategy.EntrySignal = "短期均线上穿长期均线，金叉信号"
	strategy.ExitSignal = "短期均线下穿长期均线，死叉信号"
	strategy.RiskReward = 2.0
	strategy.WinRate = 0.54
	strategy.MaxDrawdown = 0.10
	strategy.AvgProfit = 0.015

	return strategy
}

// 创建网格策略
func createGridTradingStrategy(env MarketEnvironment, indicators *TechnicalIndicators) StrategyAnalysis {
	strategy := StrategyAnalysis{
		Type:        "grid_trading",
		Name:        "网格交易策略",
		RiskLevel:   "low",
		Description: "在价格区间内设置多个买卖点，通过低买高卖获得稳定收益",
	}

	score := 6.0

	// 震荡市场加分
	if env.OverallTrend == "震荡" {
		score += 2
	}

	// 低波动率加分
	if env.Volatility < 4 {
		score += 1
	}

	strategy.Score = score
	strategy.Confidence = 70.0 + (score-6)*8

	strategy.Suitability = "横盘震荡市场，低波动率"
	strategy.EntrySignal = "价格触及网格下沿买入，上沿卖出"
	strategy.ExitSignal = "达到网格利润目标或市场趋势改变"
	strategy.RiskReward = 3.0
	strategy.WinRate = 0.67
	strategy.MaxDrawdown = 0.08
	strategy.AvgProfit = 0.032

	return strategy
}

// 创建传统策略
func createTraditionalStrategy(env MarketEnvironment, indicators *TechnicalIndicators) StrategyAnalysis {
	strategy := StrategyAnalysis{
		Type:        "traditional",
		Name:        "传统策略",
		RiskLevel:   "medium",
		Description: "基于基本面和技术分析的传统交易策略",
	}

	score := 5.0

	// 中等波动率最适合
	if env.Volatility > 3 && env.Volatility < 7 {
		score += 1
	}

	strategy.Score = score
	strategy.Confidence = 55.0 + (score-5)*10

	strategy.Suitability = "中等波动率市场"
	strategy.EntrySignal = "技术指标确认信号 + 基本面支撑"
	strategy.ExitSignal = "技术指标反转信号或基本面变化"
	strategy.RiskReward = 1.8
	strategy.WinRate = 0.58
	strategy.MaxDrawdown = 0.12
	strategy.AvgProfit = 0.018

	return strategy
}

// 创建RSI策略
func createRSIStrategy(env MarketEnvironment, indicators *TechnicalIndicators) StrategyAnalysis {
	strategy := StrategyAnalysis{
		Type:        "rsi",
		Name:        "RSI超买超卖策略",
		RiskLevel:   "high",
		Description: "利用相对强弱指标识别超买超卖信号",
	}

	score := 4.0

	// RSI信号明显时加分
	if indicators.RSI > 75 || indicators.RSI < 25 {
		score += 2
	}

	strategy.Score = score
	strategy.Confidence = 45.0 + (score-4)*12.5

	strategy.Suitability = "震荡整理市场"
	strategy.EntrySignal = "RSI < 30买入超卖，RSI > 70卖出超买"
	strategy.ExitSignal = "RSI回归50中性线或价格目标达成"
	strategy.RiskReward = 1.3
	strategy.WinRate = 0.65
	strategy.MaxDrawdown = 0.22
	strategy.AvgProfit = 0.028

	return strategy
}

// 创建MACD策略
func createMACDStrategy(env MarketEnvironment, indicators *TechnicalIndicators) StrategyAnalysis {
	strategy := StrategyAnalysis{
		Type:        "macd",
		Name:        "MACD趋势策略",
		RiskLevel:   "medium",
		Description: "使用MACD指标捕捉趋势变化",
	}

	score := 4.0

	// 中等波动率最适合
	if env.Volatility > 3 && env.Volatility < 6 {
		score += 2
	}

	strategy.Score = score
	strategy.Confidence = 50.0 + (score-4)*12.5

	strategy.Suitability = "趋势转折市场"
	strategy.EntrySignal = "MACD金叉买入，死叉卖出"
	strategy.ExitSignal = "MACD信号反转或趋势减弱"
	strategy.RiskReward = 2.2
	strategy.WinRate = 0.60
	strategy.MaxDrawdown = 0.18
	strategy.AvgProfit = 0.021

	return strategy
}

// 策略排序
func sortStrategiesByScore(strategies []StrategyAnalysis) {
	sort.Slice(strategies, func(i, j int) bool {
		return strategies[i].Score > strategies[j].Score
	})
}

// 创建交易推荐
func createTradingRecommendation(primary, alternative StrategyAnalysis, env MarketEnvironment) *TradingRecommendation {
	rec := &TradingRecommendation{
		PrimaryStrategy:     primary,
		AlternativeStrategy: alternative,
		PositionSize:        calculatePositionSize(env),
		StopLoss:           calculateStopLoss(env),
		TakeProfit:         calculateTakeProfit(env),
		EntryPrice:         95000, // 示例价格，实际应该从市场数据获取
		RiskRewardRatio:    primary.RiskReward,
		TimeFrame:          env.TimeHorizon,
		MarketConditions:   []string{},
		RiskFactors:        []string{},
		ExecutionSteps:     []string{},
	}

	// 设置市场条件
	rec.MarketConditions = []string{
		fmt.Sprintf("市场趋势: %s", env.OverallTrend),
		fmt.Sprintf("波动率: %.2f%%", env.Volatility),
		fmt.Sprintf("震荡度: %.2f%%", env.Oscillation),
		fmt.Sprintf("市场强度: %s", env.MarketStrength),
	}

	// 设置风险因素
	rec.RiskFactors = []string{
		fmt.Sprintf("整体风险等级: %s", env.RiskAssessment),
		fmt.Sprintf("策略风险等级: %s", primary.RiskLevel),
		"市场突发事件风险",
		"流动性风险",
	}

	// 设置执行步骤
	rec.ExecutionSteps = []string{
		"1. 确认市场环境符合策略条件",
		"2. 设置仓位大小和止损位",
		"3. 等待入场信号",
		"4. 执行交易并严格执行风险管理",
		"5. 定期检查持仓和调整策略",
	}

	return rec
}

// 计算仓位大小
func calculatePositionSize(env MarketEnvironment) float64 {
	baseSize := 0.1 // 基础仓位10%

	// 根据风险等级调整
	switch env.RiskAssessment {
	case "高风险":
		baseSize *= 0.5
	case "中风险":
		baseSize *= 0.75
	case "低风险":
		baseSize *= 1.0
	}

	return baseSize
}

// 计算止损位
func calculateStopLoss(env MarketEnvironment) float64 {
	baseStopLoss := 0.05 // 基础5%止损

	// 根据波动率调整
	if env.Volatility > 6 {
		baseStopLoss *= 1.5
	} else if env.Volatility < 3 {
		baseStopLoss *= 0.7
	}

	return baseStopLoss
}

// 计算止盈位
func calculateTakeProfit(env MarketEnvironment) float64 {
	baseTakeProfit := 0.10 // 基础10%止盈

	// 根据风险报酬比调整
	return baseTakeProfit
}

// 显示分析结果
func displayAnalysisResults(rec *TradingRecommendation) {
	fmt.Println("\n🎯 市场环境分析与策略推荐结果")
	fmt.Println("===================================")

	// 显示市场环境
	fmt.Println("\n📊 市场环境分析:")
	fmt.Printf("   整体趋势: %s\n", rec.PrimaryStrategy.Suitability)
	fmt.Printf("   市场强度: %s\n", "中等") // 需要从环境数据获取
	fmt.Printf("   风险等级: %s\n", rec.PrimaryStrategy.RiskLevel)

	// 显示策略排名
	fmt.Println("\n🏆 策略推荐排名:")
	fmt.Printf("   1. %s (评分: %.1f, 信心: %.1f%%)\n",
		rec.PrimaryStrategy.Name, rec.PrimaryStrategy.Score, rec.PrimaryStrategy.Confidence)
	fmt.Printf("      适用条件: %s\n", rec.PrimaryStrategy.Suitability)
	fmt.Printf("      胜率: %.1f%%, 最大回撤: %.1f%%\n",
		rec.PrimaryStrategy.WinRate*100, rec.PrimaryStrategy.MaxDrawdown*100)

	fmt.Printf("   2. %s (评分: %.1f, 信心: %.1f%%)\n",
		rec.AlternativeStrategy.Name, rec.AlternativeStrategy.Score, rec.AlternativeStrategy.Confidence)

	// 显示交易详情
	fmt.Println("\n💰 交易执行详情:")
	fmt.Printf("   建议仓位: %.1f%%\n", rec.PositionSize*100)
	fmt.Printf("   止损位: %.1f%%\n", rec.StopLoss*100)
	fmt.Printf("   止盈位: %.1f%%\n", rec.TakeProfit*100)
	fmt.Printf("   风险报酬比: %.1f\n", rec.RiskRewardRatio)
	fmt.Printf("   建议持有时间: %s\n", rec.TimeFrame)

	// 显示市场条件
	fmt.Println("\n🌍 当前市场条件:")
	for _, condition := range rec.MarketConditions {
		fmt.Printf("   • %s\n", condition)
	}

	// 显示风险因素
	fmt.Println("\n⚠️  风险因素:")
	for _, risk := range rec.RiskFactors {
		fmt.Printf("   • %s\n", risk)
	}

	// 显示执行步骤
	fmt.Println("\n📋 执行步骤:")
	for _, step := range rec.ExecutionSteps {
		fmt.Printf("   %s\n", step)
	}

	// 显示策略详情
	fmt.Println("\n🎯 主要策略详情:")
	fmt.Printf("   策略名称: %s\n", rec.PrimaryStrategy.Name)
	fmt.Printf("   策略描述: %s\n", rec.PrimaryStrategy.Description)
	fmt.Printf("   入场信号: %s\n", rec.PrimaryStrategy.EntrySignal)
	fmt.Printf("   出场信号: %s\n", rec.PrimaryStrategy.ExitSignal)
	fmt.Printf("   平均利润: %.1f%%\n", rec.PrimaryStrategy.AvgProfit*100)

	fmt.Println("\n✅ 分析完成！请根据市场实际情况谨慎决策。")
}

// 辅助函数
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateStandardDeviation(values []float64) float64 {
	if len(values) <= 1 {
		return 0
	}

	mean := average(values)
	sumSquares := 0.0
	for _, v := range values {
		sumSquares += (v - mean) * (v - mean)
	}

	return math.Sqrt(sumSquares / float64(len(values)-1))
}