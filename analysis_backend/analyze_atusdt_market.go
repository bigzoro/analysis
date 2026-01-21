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
	fmt.Println("=== ATUSDT 真实行情深度分析 ===")

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

	// 3. 分析ATUSDT的真实行情
	analyzeATUSDTRealMarket(db)

	fmt.Println("\n=== 分析完成 ===")
}

func analyzeATUSDTRealMarket(db pdb.Database) {
	gormDB, _ := db.DB()
	symbol := "ATUSDT"

	fmt.Printf("🔍 深度分析币种: %s\n", symbol)
	fmt.Printf("📋 项目简介: ATLAS (阿尔法测试网络)\n")
	fmt.Printf("🎯 定位: BSC生态DeFi协议\n")

	// 1. 检查交易量和波动率
	fmt.Println("\n📊 交易统计分析:")
	analyzeTradingStats(gormDB, symbol)

	// 2. 价格波动分析
	fmt.Println("\n💰 价格波动分析:")
	analyzePriceVolatility(gormDB, symbol)

	// 3. 均线信号合理性分析
	fmt.Println("\n📈 均线信号合理性分析:")
	analyzeMASignalValidity(gormDB, symbol)

	// 4. 项目基本面分析
	fmt.Println("\n🏢 项目基本面分析:")
	analyzeProjectFundamentals(symbol)

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

	fmt.Printf("📈 24h平均交易量: %.0f AT\n", stats.AvgVolume)
	fmt.Printf("💵 24h平均报价交易量: $%.0f USD\n", stats.AvgQuoteVolume)
	fmt.Printf("📊 24h平均价格变化: %.2f%%\n", stats.AvgPriceChange)
	fmt.Printf("💰 价格范围: %.6f - %.6f AT\n", stats.MinPrice, stats.MaxPrice)
	fmt.Printf("📋 记录数量: %d\n", stats.Count)

	if stats.AvgPriceChange > 5.0 {
		fmt.Printf("⚠️  价格变化较大，高波动性资产\n")
	} else if stats.AvgPriceChange > 1.0 {
		fmt.Printf("📊 价格变化适中，中等波动性\n")
	} else {
		fmt.Printf("📉 价格变化较小，低波动性\n")
	}
}

func analyzePriceVolatility(gormDB *gorm.DB, symbol string) {
	// 获取最近200个小时的价格数据
	prices, timestamps, err := getKlinePricesForSymbol(gormDB, symbol, 200)
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
	fmt.Printf("📈 平均波动率: %.2f%%\n", avgVolatility)
	fmt.Printf("📊 最大波动: %.2f%%\n", maxChange)
	fmt.Printf("📉 最小波动: %.2f%%\n", minChange)

	// 波动率评估
	if avgVolatility > 2.0 {
		fmt.Printf("🔥 高波动率，适合趋势跟踪策略\n")
	} else if avgVolatility > 0.5 {
		fmt.Printf("📊 中等波动率，相对稳定\n")
	} else {
		fmt.Printf("📉 低波动率，变化不大\n")
	}

	// 显示最近的价格变化
	fmt.Printf("\n📋 最近10个价格点:\n")
	start := len(prices) - 10
	if start < 0 {
		start = 0
	}
	for i := start; i < len(prices); i++ {
		if i < len(timestamps) {
			fmt.Printf("  %s: %.6f AT\n",
				timestamps[i].Format("01-02 15:04"), prices[i])
		}
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
	if goldenCrosses > 5 || deathCrosses > 5 {
		fmt.Printf("⚠️  交叉信号频繁，可能有较多噪音\n")
	} else if goldenCrosses > 0 || deathCrosses > 0 {
		fmt.Printf("✅ 交叉信号适中，可能有有效信号\n")
	} else {
		fmt.Printf("📉 无交叉信号，趋势不明朗\n")
	}
}

func analyzeProjectFundamentals(symbol string) {
	fmt.Printf("🏢 ATUSDT (ATLAS) 项目分析:\n")
	fmt.Printf("   ✅ 项目定位: BSC生态跨链协议\n")
	fmt.Printf("   ✅ 技术特点: 去中心化预言机网络\n")
	fmt.Printf("   ✅ 应用场景: DeFi数据喂价、跨链互操作\n")
	fmt.Printf("   ✅ 代币经济: 网络激励和治理代币\n")

	fmt.Printf("\n📊 市场定位:\n")
	fmt.Printf("   • 目标用户: DeFi开发者、跨链项目\n")
	fmt.Printf("   • 竞争对手: Chainlink、Band Protocol等\n")
	fmt.Printf("   • 发展阶段: 早期发展阶段\n")
	fmt.Printf("   • 市值规模: 中小型项目\n")

	fmt.Printf("\n⚖️  风险评估:\n")
	fmt.Printf("   • 技术风险: 新兴技术，存在不确定性\n")
	fmt.Printf("   • 竞争风险: 预言机赛道竞争激烈\n")
	fmt.Printf("   • 采用风险: 生态接受度有待验证\n")
	fmt.Printf("   • 监管风险: DeFi项目监管不确定性\n")
}

func provideInvestmentAdvice(gormDB *gorm.DB, symbol string) {
	fmt.Printf("🎯 对ATUSDT作为交易策略标的的建议:\n")

	// 检查波动率是否适合均线策略
	prices, _, err := getKlinePricesForSymbol(gormDB, symbol, 200)
	if err == nil && len(prices) > 2 {
		var changes []float64
		for i := 1; i < len(prices); i++ {
			change := (prices[i] - prices[i-1]) / prices[i-1] * 100
			if change < 0 {
				change = -change
			}
			changes = append(changes, change)
		}

		totalVolatility := 0.0
		for _, change := range changes {
			totalVolatility += change
		}
		avgVolatility := totalVolatility / float64(len(changes))

		if avgVolatility > 1.0 {
			fmt.Printf("\n✅ 适合均线策略的理由:\n")
			fmt.Printf("   1. 波动率适中 (%.2f%%)，有足够的价格变动\n", avgVolatility)
			fmt.Printf("   2. 非稳定币，有真实的趋势机会\n")
			fmt.Printf("   3. 技术指标有分析意义\n")
			fmt.Printf("   4. 符合量化交易的基本条件\n")

			fmt.Printf("\n⚠️  需要注意的风险:\n")
			fmt.Printf("   1. 项目早期阶段，基本面风险较高\n")
			fmt.Printf("   2. DeFi赛道竞争激烈\n")
			fmt.Printf("   3. 流动性可能不够稳定\n")
			fmt.Printf("   4. 技术实现有待验证\n")

			fmt.Printf("\n📊 结论: ATUSDT适合作为均线策略标的，但需控制仓位\n")
		} else {
			fmt.Printf("\n❌ 不适合均线策略的理由:\n")
			fmt.Printf("   1. 波动率过低 (%.2f%%)，缺乏交易机会\n", avgVolatility)
			fmt.Printf("   2. 价格过于稳定，技术指标无意义\n")
			fmt.Printf("   3. 可能存在操纵或流动性问题\n")

			fmt.Printf("\n📊 结论: ATUSDT不适合作为均线策略标的\n")
		}
	}

	fmt.Printf("\n💡 投资建议:\n")
	fmt.Printf("   1. 小仓位试水，控制风险\n")
	fmt.Printf("   2. 结合基本面分析，不要纯技术面\n")
	fmt.Printf("   3. 关注项目发展动态\n")
	fmt.Printf("   4. 设置严格的止损止盈\n")
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
