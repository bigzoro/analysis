package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"analysis/internal/analysis"
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"analysis/internal/server"
	"os"

	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 调试扫描过滤过程 ===")

	// 1. 读取配置文件
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 2. 连接数据库
	db, err := connectDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	gormDB, _ := db.DB()

	// 3. 获取策略ID 23的配置
	strategyID := uint(23)
	strategy, err := getStrategyByID(gormDB, strategyID)
	if err != nil {
		log.Fatalf("获取策略失败: %v", err)
	}

	fmt.Printf("📋 策略ID %d 配置:\n", strategyID)
	fmt.Printf("   名称: %s\n", strategy.Name)
	fmt.Printf("   信号模式: %s\n", strategy.Conditions.MASignalMode)
	fmt.Printf("   均线类型: %s\n", strategy.Conditions.MAType)
	fmt.Printf("   周期: %d/%d\n", strategy.Conditions.ShortMAPeriod, strategy.Conditions.LongMAPeriod)
	fmt.Printf("   交叉信号: %s\n", strategy.Conditions.MACrossSignal)

	// 4. 分析过滤阈值
	fmt.Println("\n🔍 过滤阈值分析:")
	thresholds := getMAValidationThresholds(strategy.Conditions.MASignalMode)
	fmt.Printf("   波动率阈值: ≥%.2f%%\n", thresholds.MinVolatility*100)
	fmt.Printf("   趋势强度阈值: ≥%.4f\n", thresholds.MinTrendStrength)
	fmt.Printf("   信号质量阈值: ≥%.1f\n", thresholds.MinSignalQuality)
	fmt.Printf("   严格模式: %v\n", thresholds.StrictMode)

	// 5. 模拟候选币种选择
	fmt.Println("\n📊 模拟候选币种选择:")
	candidates := simulateCandidateSelection(gormDB)
	fmt.Printf("   候选币种数量: %d\n", len(candidates))

	// 6. 分析为什么只有ATUSDT通过
	fmt.Println("\n🎯 分析为什么只有ATUSDT通过:")
	analyzeWhyOnlyATUSDT(gormDB, candidates, strategy, thresholds)

	fmt.Println("\n=== 调试完成 ===")
}

func simulateCandidateSelection(gormDB *gorm.DB) []string {
	// 模拟VolumeBasedSelector的选择逻辑
	var volumeStats []struct {
		Symbol      string
		QuoteVolume float64
	}

	// 查询24小时交易量前50的币种
	gormDB.Table("binance_24h_stats").
		Select("symbol, quote_volume").
		Where("market_type = ? AND created_at >= ? AND quote_volume > 1000000",
			"spot", time.Now().Add(-24*time.Hour)).
		Order("quote_volume DESC").
		Limit(50).
		Scan(&volumeStats)

	candidates := make([]string, len(volumeStats))
	for i, stat := range volumeStats {
		candidates[i] = stat.Symbol
	}

	// 过滤稳定币
	filtered := server.FilterStableCoins(candidates)
	fmt.Printf("   原始候选: %d个 → 过滤稳定币后: %d个\n", len(candidates), len(filtered))

	return filtered
}

func analyzeWhyOnlyATUSDT(gormDB *gorm.DB, candidates []string, strategy *pdb.TradingStrategy, thresholds server.MAValidationThresholds) {
	fmt.Println("   正在分析其他候选币种的过滤原因...")

	passedCount := 0
	failedReasons := make(map[string]int)

	for i, symbol := range candidates {
		if i >= 10 { // 只分析前10个，避免过多输出
			break
		}

		reason := analyzeSymbolFailure(gormDB, symbol, strategy, thresholds)
		if reason == "PASSED" {
			passedCount++
			fmt.Printf("   ✅ %s: 通过所有验证\n", symbol)
		} else {
			failedReasons[reason]++
			fmt.Printf("   ❌ %s: %s\n", symbol, reason)
		}
	}

	fmt.Printf("\n📈 分析结果:\n")
	fmt.Printf("   通过验证的币种: %d个\n", passedCount)
	fmt.Printf("   失败原因统计:\n")
	for reason, count := range failedReasons {
		fmt.Printf("     • %s: %d个币种\n", reason, count)
	}

	// 特别分析ATUSDT
	fmt.Printf("\n🎯 ATUSDT成功原因分析:\n")
	analyzeSymbolSuccess(gormDB, "ATUSDT", strategy, thresholds)
}

func analyzeSymbolFailure(gormDB *gorm.DB, symbol string, strategy *pdb.TradingStrategy, thresholds server.MAValidationThresholds) string {
	// 检查数据是否存在
	var count int64
	gormDB.Table("market_klines").Where("symbol = ? AND kind = ? AND `interval` = ?",
		symbol, "spot", "1h").Count(&count)

	if count == 0 {
		return "无K线数据"
	}

	// 获取价格数据
	prices := getPricesForSymbol(gormDB, symbol, strategy.Conditions.LongMAPeriod+10)
	if len(prices) < strategy.Conditions.LongMAPeriod {
		return fmt.Sprintf("数据不足(%d/%d)", len(prices), strategy.Conditions.LongMAPeriod)
	}

	// 波动率检查
	if !server.ValidateVolatilityForMA(symbol, prices, thresholds.MinVolatility) {
		return "波动率不足"
	}

	// 计算均线
	maType := analysis.MovingAverageType(strategy.Conditions.MAType)
	shortMA := analysis.NewTechnicalIndicators().CalculateMovingAverage(prices, strategy.Conditions.ShortMAPeriod, maType)
	longMA := analysis.NewTechnicalIndicators().CalculateMovingAverage(prices, strategy.Conditions.LongMAPeriod, maType)

	if len(shortMA) == 0 || len(longMA) == 0 {
		return "均线计算失败"
	}

	// 趋势强度检查
	if !server.ValidateTrendStrength(shortMA, longMA, thresholds.MinTrendStrength) {
		return "趋势强度不足"
	}

	// 信号质量检查
	signalQuality := server.AssessSignalQuality(shortMA, longMA, prices)
	if signalQuality < thresholds.MinSignalQuality {
		return "信号质量不足"
	}

	// 交叉信号检查
	goldenCross, deathCross := analysis.NewTechnicalIndicators().DetectMACross(shortMA, longMA)
	hasValidSignal := false

	switch strategy.Conditions.MACrossSignal {
	case "GOLDEN_CROSS":
		hasValidSignal = goldenCross
	case "DEATH_CROSS":
		hasValidSignal = deathCross
	case "BOTH":
		hasValidSignal = goldenCross || deathCross
	}

	if !hasValidSignal {
		return "无有效交叉信号"
	}

	return "PASSED"
}

func analyzeSymbolSuccess(gormDB *gorm.DB, symbol string, strategy *pdb.TradingStrategy, thresholds server.MAValidationThresholds) {
	prices := getPricesForSymbol(gormDB, symbol, strategy.Conditions.LongMAPeriod+10)

	// 计算各项指标
	avgVolatility := calculateAvgVolatility(prices)
	fmt.Printf("   • 波动率: %.2f%% (阈值: %.2f%%) ✅\n", avgVolatility*100, thresholds.MinVolatility*100)

	// 计算均线
	maType := analysis.MovingAverageType(strategy.Conditions.MAType)
	shortMA := analysis.NewTechnicalIndicators().CalculateMovingAverage(prices, strategy.Conditions.ShortMAPeriod, maType)
	longMA := analysis.NewTechnicalIndicators().CalculateMovingAverage(prices, strategy.Conditions.LongMAPeriod, maType)

	if len(shortMA) > 0 && len(longMA) > 0 {
		latestShort := shortMA[len(shortMA)-1]
		latestLong := longMA[len(longMA)-1]
		trendStrength := (latestShort - latestLong) / latestLong
		if trendStrength < 0 {
			trendStrength = -trendStrength
		}
		fmt.Printf("   • 趋势强度: %.4f (阈值: %.4f) ✅\n", trendStrength, thresholds.MinTrendStrength)

		signalQuality := server.AssessSignalQuality(shortMA, longMA, prices)
		fmt.Printf("   • 信号质量: %.3f (阈值: %.1f) ✅\n", signalQuality, thresholds.MinSignalQuality)

		goldenCross, deathCross := analysis.NewTechnicalIndicators().DetectMACross(shortMA, longMA)
		fmt.Printf("   • 交叉信号: 金叉=%v, 死叉=%v ✅\n", goldenCross, deathCross)
	}

	fmt.Printf("   • 数据点数: %d ✅\n", len(prices))
}

func getPricesForSymbol(gormDB *gorm.DB, symbol string, limit int) []float64 {
	var klines []pdb.MarketKline
	gormDB.Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").
		Order("open_time DESC").
		Limit(limit).
		Find(&klines)

	// 反转顺序，从旧到新
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	prices := make([]float64, len(klines))
	for i, kline := range klines {
		price, _ := strconv.ParseFloat(kline.ClosePrice, 64)
		prices[i] = price
	}

	return prices
}

func calculateAvgVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0.0
	}

	var changes []float64
	for i := 1; i < len(prices); i++ {
		change := (prices[i] - prices[i-1]) / prices[i-1] * 100
		if change < 0 {
			change = -change
		}
		changes = append(changes, change)
	}

	if len(changes) == 0 {
		return 0.0
	}

	totalChange := 0.0
	for _, change := range changes {
		totalChange += change
	}
	return totalChange / float64(len(changes))
}

func getStrategyByID(gormDB *gorm.DB, id uint) (*pdb.TradingStrategy, error) {
	var strategy pdb.TradingStrategy
	err := gormDB.Preload("Conditions").Where("id = ?", id).First(&strategy).Error
	if err != nil {
		return nil, fmt.Errorf("策略ID %d不存在: %v", id, err)
	}
	return &strategy, nil
}

func getMAValidationThresholds(signalMode string) server.MAValidationThresholds {
	switch signalMode {
	case "QUALITY_FIRST":
		return server.MAValidationThresholds{
			MinVolatility:    0.08,
			MinTrendStrength: 0.002,
			MinSignalQuality: 0.7,
			StrictMode:       true,
		}
	case "QUANTITY_FIRST":
		return server.MAValidationThresholds{
			MinVolatility:    0.03,
			MinTrendStrength: 0.0005,
			MinSignalQuality: 0.4,
			StrictMode:       false,
		}
	default:
		return server.MAValidationThresholds{
			MinVolatility:    0.05,
			MinTrendStrength: 0.001,
			MinSignalQuality: 0.5,
			StrictMode:       false,
		}
	}
}

// 辅助函数
func loadConfig(configPath string) (*config.Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	var cfg config.Config
	decoder := yaml.NewDecoder(file)
	err := decoder.Decode(&cfg)
	if err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &cfg, nil
}

func connectDatabase(dbConfig struct {
	DSN          string `yaml:"dsn"`
	Automigrate  bool   `yaml:"automigrate"`
	MaxOpenConns int    `yaml:"max_open_conns"`
	MaxIdleConns int    `yaml:"max_idle_conns"`
}) (pdb.Database, error) {
	options := pdb.Options{
		DSN:          dbConfig.DSN,
		Automigrate:  false,
		MaxOpenConns: dbConfig.MaxOpenConns,
		MaxIdleConns: dbConfig.MaxIdleConns,
	}

	return pdb.OpenMySQL(options)
}
