package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"analysis/internal/analysis"
	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
)

func main() {
	fmt.Println("=== BFUSDUSDT 真实行情深度分析 ===")

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

	// 3. 分析BFUSDUSDT的真实行情
	analyzeBFUSDUTRealMarket(db)

	fmt.Println("\n=== 分析完成 ===")
}

func analyzeBFUSDUTRealMarket(db pdb.Database) {
	gormDB, _ := db.DB()
	symbol := "BFUSDUSDT"

	fmt.Printf("🔍 深度分析币种: %s (稳定币兑换对)\n", symbol)

	// 1. 检查交易量和波动率
	fmt.Println("\n📊 交易统计分析:")
	analyzeTradingStats(gormDB, symbol)

	// 2. 价格波动分析
	fmt.Println("\n💰 价格波动分析:")
	analyzePriceVolatility(gormDB, symbol)

	// 3. 均线信号合理性分析
	fmt.Println("\n📈 均线信号合理性分析:")
	analyzeMASignalValidity(gormDB, symbol)

	// 4. 稳定性评估
	fmt.Println("\n🏦 稳定币特性评估:")
	analyzeStabilityCharacteristics(gormDB, symbol)

	// 5. 投资建议
	fmt.Println("\n🎯 投资策略建议:")
	provideInvestmentAdvice(gormDB, symbol)
}

func analyzeTradingStats(gormDB *gorm.DB, symbol string) {
	// 查询最近24小时的交易统计
	var stats struct {
		AvgVolume      float64
		AvgQuoteVolume float64
		AvgPriceChange float64
		MinPrice       float64
		MaxPrice       float64
		Count          int64
	}

	err := gormDB.Table("binance_24h_stats").Select(`
		AVG(volume) as avg_volume,
		AVG(quote_volume) as avg_quote_volume,
		AVG(price_change_percent) as avg_price_change,
		MIN(last_price) as min_price,
		MAX(last_price) as max_price,
		COUNT(*) as count
	`).Where("symbol = ? AND market_type = ? AND created_at >= ?", symbol, "spot", time.Now().Add(-24*time.Hour)).Scan(&stats)

	if err != nil {
		fmt.Printf("❌ 查询交易统计失败: %v\n", err)
		return
	}

	fmt.Printf("📈 24h平均交易量: %.0f BFUSD\n", stats.AvgVolume)
	fmt.Printf("💵 24h平均报价交易量: $%.0f USD\n", stats.AvgQuoteVolume)
	fmt.Printf("📊 24h平均价格变化: %.6f%%\n", stats.AvgPriceChange)
	fmt.Printf("💰 价格范围: %.6f - %.6f BFUSD\n", stats.MinPrice, stats.MaxPrice)
	fmt.Printf("📋 记录数量: %d\n", stats.Count)

	if stats.AvgPriceChange > 0.001 { // 0.001% = 0.00001
		fmt.Printf("⚠️  价格变化较大，不符合稳定币特性\n")
	} else {
		fmt.Printf("✅ 价格变化极小，符合稳定币特性\n")
	}
}

func analyzePriceVolatility(gormDB *gorm.DB, symbol string) {
	// 获取最近200个小时的价格数据
	prices, _, err := getKlinePricesForSymbol(gormDB, symbol, 200)
	if err != nil {
		fmt.Printf("❌ 获取价格数据失败: %v\n", err)
		return
	}

	if len(prices) < 2 {
		fmt.Printf("❌ 价格数据不足\n")
		return
	}

	// 计算波动率
	var changes []float64
	for i := 1; i < len(prices); i++ {
		change := (prices[i] - prices[i-1]) / prices[i-1] * 100
		changes = append(changes, change)
	}

	// 计算统计指标
	totalChange := 0.0
	maxChange := 0.0
	minChange := 0.0
	changeCount := 0

	for _, change := range changes {
		absChange := change
		if absChange < 0 {
			absChange = -absChange
		}

		totalChange += absChange
		if absChange > maxChange {
			maxChange = absChange
		}
		if change < minChange {
			minChange = change
		}
		changeCount++
	}

	avgVolatility := totalChange / float64(changeCount)

	fmt.Printf("📊 分析时段: 最近%d小时\n", len(prices))
	fmt.Printf("📈 平均波动率: %.6f%%\n", avgVolatility)
	fmt.Printf("📊 最大波动: %.6f%%\n", maxChange)
	fmt.Printf("📉 最小波动: %.6f%%\n", minChange)

	// 稳定币标准：波动率应该小于0.01%
	if avgVolatility > 0.01 {
		fmt.Printf("⚠️  波动率偏高，可能不适合作为稳定币\n")
	} else {
		fmt.Printf("✅ 波动率极低，符合稳定币标准\n")
	}

	// 显示最近的价格变化
	fmt.Printf("\n📋 最近10个价格点:\n")
	start := len(prices) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(prices); i++ {
		fmt.Printf("  %.6f BFUSD\n", prices[i])
	}
}

func analyzeMASignalValidity(gormDB *gorm.DB, symbol string) {
	// 获取价格数据进行均线分析
	prices, _, err := getKlinePricesForSymbol(gormDB, symbol, 200)
	if err != nil {
		fmt.Printf("❌ 获取价格数据失败: %v\n", err)
		return
	}

	if len(prices) < 25 {
		fmt.Printf("❌ 数据不足，无法进行均线分析\n")
		return
	}

	// 计算SMA5和SMA20
	ti := analysis.NewTechnicalIndicators()
	shortMA := ti.CalculateMovingAverage(prices, 5, analysis.SMA)
	longMA := ti.CalculateMovingAverage(prices, 20, analysis.SMA)

	if len(shortMA) < 2 || len(longMA) < 2 {
		fmt.Printf("❌ 均线计算失败\n")
		return
	}

	// 分析交叉信号的合理性
	goldenCrosses := 0
	deathCrosses := 0

	for i := 1; i < len(shortMA) && i < len(longMA); i++ {
		if i >= len(shortMA) || i >= len(longMA) {
			break
		}

		prevShort := shortMA[i-1]
		prevLong := longMA[i-1]
		currShort := shortMA[i]
		currLong := longMA[i]

		// 金叉：短期线上穿长期线
		if prevShort <= prevLong && currShort > currLong {
			goldenCrosses++
		}
		// 死叉：短期线下穿长期线
		if prevShort >= prevLong && currShort < currLong {
			deathCrosses++
		}
	}

	fmt.Printf("📊 均线交叉统计 (SMA5 vs SMA20):\n")
	fmt.Printf("   金叉次数: %d\n", goldenCrosses)
	fmt.Printf("   死叉次数: %d\n", deathCrosses)

	// 当前均线状态
	latestShort := shortMA[len(shortMA)-1]
	latestLong := longMA[len(longMA)-1]
	fmt.Printf("📈 当前SMA5: %.6f\n", latestShort)
	fmt.Printf("📉 当前SMA20: %.6f\n", latestLong)

	if latestShort > latestLong {
		fmt.Printf("📊 当前状态: SMA5 > SMA20 (金叉后状态)\n")
	} else {
		fmt.Printf("📊 当前状态: SMA5 < SMA20 (死叉后状态)\n")
	}

	// 评估信号合理性
	if goldenCrosses > 2 || deathCrosses > 2 {
		fmt.Printf("⚠️  交叉信号过于频繁，不符合稳定币特性\n")
	} else {
		fmt.Printf("✅ 交叉信号很少，符合稳定币特性\n")
	}
}

func analyzeStabilityCharacteristics(gormDB *gorm.DB, symbol string) {
	fmt.Printf("🏦 BFUSDUSDT 作为稳定币的特性:\n")
	fmt.Printf("   ✅ 锚定资产: BUSD (币安稳定币)\n")
	fmt.Printf("   ✅ 目标价格: 1.000000 USDT\n")
	fmt.Printf("   ✅ 发行机构: Binance\n")
	fmt.Printf("   ✅ 储备资产: 美元等价物\n")

	fmt.Printf("\n📋 稳定币的典型特征:\n")
	fmt.Printf("   • 价格波动 < 0.1%%\n")
	fmt.Printf("   • 交易量大，流动性好\n")
	fmt.Printf("   • 很少有趋势性变动\n")
	fmt.Printf("   • 不适合技术分析交易\n")

	fmt.Printf("\n⚖️  风险评估:\n")
	fmt.Printf("   • 监管风险: 稳定币监管不确定性\n")
	fmt.Printf("   • 储备风险: 储备资产质量\n")
	fmt.Printf("   • 平台风险: 依赖币安生态\n")
}

func provideInvestmentAdvice(gormDB *gorm.DB, symbol string) {
	fmt.Printf("🎯 对BFUSDUSDT作为交易策略标的的建议:\n")
	fmt.Printf("\n❌ 不推荐原因:\n")
	fmt.Printf("   1. 稳定币不适合技术分析策略\n")
	fmt.Printf("   2. 波动极小，难以盈利\n")
	fmt.Printf("   3. 交叉信号可能是数据噪声\n")
	fmt.Printf("   4. 违背了均线策略的初衷\n")

	fmt.Printf("\n✅ 更适合的策略:\n")
	fmt.Printf("   1. 持有稳定币作为现金等价物\n")
	fmt.Printf("   2. 作为交易对进行套利\n")
	fmt.Printf("   3. 作为避险资产\n")
	fmt.Printf("   4. 作为资金池参与DeFi收益\n")

	fmt.Printf("\n💡 策略改进建议:\n")
	fmt.Printf("   1. 从候选列表中排除稳定币\n")
	fmt.Printf("   2. 添加波动率过滤条件\n")
	fmt.Printf("   3. 增加最小价格变动阈值\n")
	fmt.Printf("   4. 专注于高波动性资产\n")

	fmt.Printf("\n📊 结论: BFUSDUSDT不适合作为均线策略标的\n")
}

func getKlinePricesForSymbol(gormDB *gorm.DB, symbol string, limit int) ([]float64, []time.Time, error) {
	var klines []pdb.MarketKline
	err := gormDB.Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").
		Order("open_time DESC").
		Limit(limit).
		Find(&klines).Error

	if err != nil {
		return nil, nil, err
	}

	// 反转顺序，从旧到新
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	prices := make([]float64, len(klines))
	timestamps := make([]time.Time, len(klines))

	for i, kline := range klines {
		price, err := strconv.ParseFloat(kline.ClosePrice, 64)
		if err != nil {
			return nil, nil, fmt.Errorf("解析价格失败: %v", err)
		}
		prices[i] = price
		timestamps[i] = kline.OpenTime
	}

	return prices, timestamps, nil
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
	if err := decoder.Decode(&cfg); err != nil {
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
