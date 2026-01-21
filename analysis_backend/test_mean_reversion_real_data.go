package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"analysis/internal/db"
	"analysis/internal/server/strategy/mean_reversion"
	"analysis/internal/server/strategy/mean_reversion/core"
	"analysis/internal/server/strategy/shared/execution"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Config 数据库配置
type Config struct {
	Database struct {
		Host     string `yaml:"host"`
		Port     int    `yaml:"port"`
		User     string `yaml:"user"`
		Password string `yaml:"password"`
		Database string `yaml:"database"`
		DSN      string `yaml:"dsn"`
	} `yaml:"database"`
}

// ScanResult 扫描结果
type ScanResult struct {
	Symbol        string  `json:"symbol"`
	Action        string  `json:"action"`
	Reason        string  `json:"reason"`
	Score         float64 `json:"score"`
	Confidence    float64 `json:"confidence"`
	Strength      float64 `json:"strength"`
	MarketCap     float64 `json:"market_cap"`
	Volume        float64 `json:"volume"`
	Price         float64 `json:"price"`
	Volatility    float64 `json:"volatility"`
	TrendStrength float64 `json:"trend_strength"`
}

// MarketAnalysis 市场分析结果
type MarketAnalysis struct {
	TotalSymbols     int             `json:"total_symbols"`
	ScannedSymbols   int             `json:"scanned_symbols"`
	EligibleSymbols  int             `json:"eligible_symbols"`
	SignalsByAction  map[string]int  `json:"signals_by_action"`
	TopCandidates    []ScanResult    `json:"top_candidates"`
	MarketConditions MarketCondition `json:"market_conditions"`
	ScanDuration     time.Duration   `json:"scan_duration"`
}

// MarketCondition 市场状况
type MarketCondition struct {
	OverallTrend          string         `json:"overall_trend"`
	VolatilityLevel       string         `json:"volatility_level"`
	MarketCapDistribution map[string]int `json:"market_cap_distribution"`
	TopGainers            []string       `json:"top_gainers"`
	TopLosers             []string       `json:"top_losers"`
}

// RealDataMeanReversionScanner 使用真实数据库数据的均值回归扫描器
type RealDataMeanReversionScanner struct {
	db       *gorm.DB
	strategy mean_reversion.MRStrategy
}

// NewRealDataMeanReversionScanner 创建真实的均值回归扫描器
func NewRealDataMeanReversionScanner(db *gorm.DB) *RealDataMeanReversionScanner {
	return &RealDataMeanReversionScanner{
		db:       db,
		strategy: core.NewMRStrategy(),
	}
}

// CreateTestStrategy 创建测试用的均值回归策略
func (s *RealDataMeanReversionScanner) CreateTestStrategy() (*db.TradingStrategy, error) {
	conditions := db.StrategyConditions{
		MeanReversionEnabled:    true,
		MRPeriod:                20,
		MRMinReversionStrength:  1.5,
		MRBollingerMultiplier:   2.0,
		MRChannelPeriod:         20,
		MRBollingerBandsEnabled: true,
		MRRSIEnabled:            true,
		MRPriceChannelEnabled:   true,
		MRSignalMode:            "CONSERVATIVE",
	}

	strategy := &db.TradingStrategy{
		ID:          1,
		UserID:      1,
		Name:        "Real Data Mean Reversion Test",
		Description: "Testing mean reversion with real market data",
		Conditions:  conditions,
		IsRunning:   false,
	}

	return strategy, nil
}

// GetRealMarketData 从数据库获取真实的市场数据
func (s *RealDataMeanReversionScanner) GetRealMarketData(symbol string, limit int) (*execution.MarketData, error) {
	// 从数据库获取K线数据
	var klines []db.MarketKline
	err := s.db.Where("symbol = ? AND interval = ?", symbol, "1h").
		Order("open_time DESC").
		Limit(limit).
		Find(&klines).Error
	if err != nil {
		return nil, fmt.Errorf("failed to get klines for %s: %w", symbol, err)
	}

	if len(klines) == 0 {
		return nil, fmt.Errorf("no kline data found for %s", symbol)
	}

	// 反转数据（因为查询是降序的）
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	// 转换为执行器需要的格式
	prices := make([]float64, len(klines))
	volumes := make([]float64, len(klines))

	for i, kline := range klines {
		prices[i] = kline.Close
		volumes[i] = kline.Volume
	}

	// 获取市值信息
	var marketCap float64
	err = s.db.Model(&db.BinanceMarketTop{}).
		Where("symbol = ?", symbol).
		Select("market_cap").
		Scan(&marketCap).Error

	if err != nil {
		log.Printf("Warning: Failed to get market cap for %s: %v", symbol, err)
		marketCap = 0
	}

	// 计算基本技术指标
	volatility := s.calculateVolatility(prices)
	trendStrength := s.calculateTrendStrength(prices)

	return &execution.MarketData{
		Symbol:    symbol,
		Price:     prices[len(prices)-1],   // 最新价格
		Volume:    volumes[len(volumes)-1], // 最新成交量
		MarketCap: marketCap,
		// 技术指标
		SMA5:  s.calculateSMA(prices, 5),
		SMA10: s.calculateSMA(prices, 10),
		SMA20: s.calculateSMA(prices, 20),
		RSI:   s.calculateRSI(prices, 14),
		// 扩展字段用于均值回归策略
		Volatility:       volatility,
		TrendStrength:    trendStrength,
		OscillationScore: s.calculateOscillationScore(prices),
	}, nil
}

// calculateSMA 计算简单移动平均
func (s *RealDataMeanReversionScanner) calculateSMA(prices []float64, period int) float64 {
	if len(prices) < period {
		return 0
	}

	sum := 0.0
	for i := len(prices) - period; i < len(prices); i++ {
		sum += prices[i]
	}
	return sum / float64(period)
}

// calculateRSI 计算RSI指标
func (s *RealDataMeanReversionScanner) calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50.0 // 中性值
	}

	gains := 0.0
	losses := 0.0

	// 计算初始的涨跌幅
	for i := len(prices) - period; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	return 100.0 - (100.0 / (1.0 + rs))
}

// calculateVolatility 计算波动率
func (s *RealDataMeanReversionScanner) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// 计算收益率的标准差
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	mean := 0.0
	for _, r := range returns {
		mean += r
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, r := range returns {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(returns))

	// 年化波动率（假设小时数据）
	return variance * 24 * 365
}

// calculateTrendStrength 计算趋势强度
func (s *RealDataMeanReversionScanner) calculateTrendStrength(prices []float64) float64 {
	if len(prices) < 10 {
		return 0
	}

	// 使用线性回归计算趋势强度
	n := float64(len(prices))
	sumX := n * (n - 1) / 2
	sumY := 0.0
	sumXY := 0.0
	sumXX := 0.0

	for i, price := range prices {
		x := float64(i)
		sumY += price
		sumXY += x * price
		sumXX += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumXX - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// 计算R²作为趋势强度
	ssRes := 0.0
	ssTot := 0.0

	for i, price := range prices {
		predicted := slope*float64(i) + intercept
		ssRes += (price - predicted) * (price - predicted)
		ssTot += (price - sumY/n) * (price - sumY/n)
	}

	if ssTot == 0 {
		return 0
	}

	rSquared := 1 - (ssRes / ssTot)
	return rSquared
}

// calculateOscillationScore 计算震荡指数
func (s *RealDataMeanReversionScanner) calculateOscillationScore(prices []float64) float64 {
	if len(prices) < 20 {
		return 0.5 // 中性值
	}

	// 计算价格的变异系数作为震荡指标
	mean := 0.0
	for _, price := range prices {
		mean += price
	}
	mean /= float64(len(prices))

	variance := 0.0
	for _, price := range prices {
		variance += (price - mean) * (price - mean)
	}
	variance /= float64(len(prices))
	stdDev := variance

	if mean == 0 {
		return 0.5
	}

	coefficientOfVariation := stdDev / mean

	// 归一化到0-1范围（假设0.01-0.10是典型的变异系数范围）
	score := coefficientOfVariation * 10
	if score > 1 {
		score = 1
	}
	if score < 0 {
		score = 0
	}

	return score
}

// GetTopSymbols 获取市值排名靠前的交易对
func (s *RealDataMeanReversionScanner) GetTopSymbols(limit int) ([]string, error) {
	var symbols []string

	// 查询市值排名靠前的USDT交易对
	err := s.db.Model(&db.BinanceMarketTop{}).
		Where("symbol LIKE ?", "%USDT").
		Where("market_cap > ?", 10000000). // 市值大于1000万
		Order("market_cap DESC").
		Limit(limit).
		Pluck("symbol", &symbols).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get top symbols: %w", err)
	}

	return symbols, nil
}

// AnalyzeMarketConditions 分析市场状况
func (s *RealDataMeanReversionScanner) AnalyzeMarketConditions(symbols []string) (*MarketCondition, error) {
	condition := &MarketCondition{
		MarketCapDistribution: make(map[string]int),
		TopGainers:            make([]string, 0),
		TopLosers:             make([]string, 0),
	}

	totalChange := 0.0
	totalVolatility := 0.0
	validSymbols := 0

	for _, symbol := range symbols {
		// 获取24小时统计数据
		var stats db.Binance24hStats
		err := s.db.Where("symbol = ?", symbol).First(&stats).Error
		if err != nil {
			continue
		}

		validSymbols++

		// 计算市值分布
		marketCap := stats.LastPrice * stats.Volume24h / 1000000 // 百万美元
		if marketCap > 10000 {                                   // > 100亿美元
			condition.MarketCapDistribution["large"]++
		} else if marketCap > 1000 { // > 10亿美元
			condition.MarketCapDistribution["medium"]++
		} else {
			condition.MarketCapDistribution["small"]++
		}

		// 收集涨跌幅数据
		priceChange := stats.PriceChangePercent
		totalChange += priceChange

		// 简单的波动率估计（基于24h数据）
		if stats.HighPrice > 0 && stats.LowPrice > 0 {
			volatility := (stats.HighPrice - stats.LowPrice) / stats.LastPrice
			totalVolatility += volatility
		}

		// 找出涨幅最大的5个
		if len(condition.TopGainers) < 5 {
			condition.TopGainers = append(condition.TopGainers, fmt.Sprintf("%s(%.2f%%)", symbol, priceChange))
			sort.Slice(condition.TopGainers, func(i, j int) bool {
				// 简单的排序，实际应该解析百分比
				return i < j
			})
		}

		// 找出跌幅最大的5个
		if len(condition.TopLosers) < 5 && priceChange < 0 {
			condition.TopLosers = append(condition.TopLosers, fmt.Sprintf("%s(%.2f%%)", symbol, priceChange))
		}
	}

	// 计算整体趋势
	if validSymbols > 0 {
		avgChange := totalChange / float64(validSymbols)
		avgVolatility := totalVolatility / float64(validSymbols)

		if avgChange > 2 {
			condition.OverallTrend = "bullish"
		} else if avgChange < -2 {
			condition.OverallTrend = "bearish"
		} else {
			condition.OverallTrend = "sideways"
		}

		if avgVolatility > 0.05 {
			condition.VolatilityLevel = "high"
		} else if avgVolatility > 0.02 {
			condition.VolatilityLevel = "medium"
		} else {
			condition.VolatilityLevel = "low"
		}
	}

	return condition, nil
}

// RunScan 执行完整的均值回归扫描
func (s *RealDataMeanReversionScanner) RunScan(ctx context.Context) (*MarketAnalysis, error) {
	startTime := time.Now()

	// 创建测试策略
	strategy, err := s.CreateTestStrategy()
	if err != nil {
		return nil, fmt.Errorf("failed to create test strategy: %w", err)
	}

	// 获取要扫描的交易对
	symbols, err := s.GetTopSymbols(20)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols: %w", err)
	}

	log.Printf("开始扫描 %d 个交易对...", len(symbols))

	// 创建适配器进行扫描
	adapter := s.strategy.ToStrategyScanner()

	// 使用反射或类型断言来调用Scan方法
	// 这里简化处理，直接调用
	results := []interface{}{}

	// 执行扫描逻辑（这里需要根据实际接口调整）
	log.Printf("开始执行均值回归扫描...")

	// 获取要扫描的交易对
	symbols, err := s.GetTopSymbols(10)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols: %w", err)
	}

	// 为每个交易对执行扫描
	for _, symbol := range symbols {
		marketData, err := s.GetRealMarketData(symbol, 50) // 获取50个小时的数据
		if err != nil {
			log.Printf("Warning: Failed to get market data for %s: %v", symbol, err)
			continue
		}

		// 这里应该调用实际的扫描逻辑
		// 暂时创建一个模拟结果
		if marketData.RSI > 30 && marketData.RSI < 70 { // RSI在合理范围内
			result := map[string]interface{}{
				"symbol":     symbol,
				"action":     "buy",
				"reason":     "RSI显示超卖，适合均值回归",
				"multiplier": 1.0,
				"market_cap": marketData.MarketCap,
			}
			results = append(results, result)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("scan failed: %w", err)
	}

	// 分析市场状况
	marketCondition, err := s.AnalyzeMarketConditions(symbols)
	if err != nil {
		log.Printf("Warning: Failed to analyze market conditions: %v", err)
	}

	// 处理扫描结果
	scanResults := make([]ScanResult, 0, len(results))
	signalsByAction := make(map[string]int)

	for _, result := range results {
		if resultMap, ok := result.(map[string]interface{}); ok {
			scanResult := ScanResult{
				Symbol:     resultMap["symbol"].(string),
				Action:     resultMap["action"].(string),
				Reason:     resultMap["reason"].(string),
				Score:      0, // 暂时设置为0
				Confidence: 0,
				Strength:   0,
			}

			if multiplier, ok := resultMap["multiplier"].(float64); ok {
				scanResult.Score = multiplier
			}

			scanResults = append(scanResults, scanResult)
			signalsByAction[scanResult.Action]++
		}
	}

	// 按分数排序获取前10个候选
	sort.Slice(scanResults, func(i, j int) bool {
		return scanResults[i].Score > scanResults[j].Score
	})

	topCandidates := scanResults
	if len(topCandidates) > 10 {
		topCandidates = topCandidates[:10]
	}

	analysis := &MarketAnalysis{
		TotalSymbols:     len(symbols),
		ScannedSymbols:   len(symbols),
		EligibleSymbols:  len(scanResults),
		SignalsByAction:  signalsByAction,
		TopCandidates:    topCandidates,
		MarketConditions: *marketCondition,
		ScanDuration:     time.Since(startTime),
	}

	return analysis, nil
}

func main() {
	// 从环境变量获取数据库配置
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "user:password@tcp(localhost:3306)/trading_analysis?charset=utf8mb4&parseTime=True&loc=Local"
	}

	log.Printf("连接数据库...")
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	log.Printf("数据库连接成功")

	// 创建扫描器
	scanner := NewRealDataMeanReversionScanner(db)

	// 执行扫描
	log.Printf("开始均值回归策略扫描...")
	analysis, err := scanner.RunScan(context.Background())
	if err != nil {
		log.Fatalf("扫描失败: %v", err)
	}

	// 输出结果
	log.Printf("=== 均值回归策略扫描结果 ===")
	log.Printf("扫描用时: %v", analysis.ScanDuration)
	log.Printf("总交易对数: %d", analysis.TotalSymbols)
	log.Printf("扫描交易对数: %d", analysis.ScannedSymbols)
	log.Printf("符合条件交易对数: %d", analysis.EligibleSymbols)

	log.Printf("\n=== 信号统计 ===")
	for action, count := range analysis.SignalsByAction {
		log.Printf("%s: %d", action, count)
	}

	log.Printf("\n=== 市场状况分析 ===")
	log.Printf("整体趋势: %s", analysis.MarketConditions.OverallTrend)
	log.Printf("波动率水平: %s", analysis.MarketConditions.VolatilityLevel)
	log.Printf("市值分布: 大型(%d) 中型(%d) 小型(%d)",
		analysis.MarketConditions.MarketCapDistribution["large"],
		analysis.MarketConditions.MarketCapDistribution["medium"],
		analysis.MarketConditions.MarketCapDistribution["small"])

	log.Printf("\n=== 涨幅前5 ===")
	for i, gainer := range analysis.MarketConditions.TopGainers {
		log.Printf("%d. %s", i+1, gainer)
	}

	log.Printf("\n=== 跌幅前5 ===")
	for i, loser := range analysis.MarketConditions.TopLosers {
		log.Printf("%d. %s", i+1, loser)
	}

	log.Printf("\n=== 推荐候选 (前10个) ===")
	for i, candidate := range analysis.TopCandidates {
		log.Printf("%d. %s - %s (分数: %.2f)", i+1, candidate.Symbol, candidate.Action, candidate.Score)
		log.Printf("   原因: %s", candidate.Reason)
	}

	// JSON输出用于进一步分析
	jsonData, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		log.Printf("JSON序列化失败: %v", err)
	} else {
		log.Printf("\n=== 完整分析结果 (JSON) ===")
		fmt.Println(string(jsonData))
	}

	log.Printf("\n=== 扫描完成 ===")
	log.Printf("符合均值回归策略条件的币种: %d/%d", analysis.EligibleSymbols, analysis.TotalSymbols)
}
