package main

import (
	"fmt"
	"log"
	"strconv"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
)

func main() {
	fmt.Println("=== 检查波动率阈值合理性 ===")

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

	// 3. 检查主流币种的波动率
	fmt.Println("📊 主流币种波动率统计:")
	checkMajorCoinsVolatility(gormDB)

	// 4. 分析当前阈值的合理性
	fmt.Println("\n📊 波动率阈值分析:")
	analyzeVolatilityThreshold(gormDB)

	fmt.Println("\n=== 分析完成 ===")
}

func checkMajorCoinsVolatility(gormDB *gorm.DB) {
	// 检查主流币种的波动率
	majorCoins := []string{"BTCUSDT", "ETHUSDT", "BNBUSDT", "ADAUSDT", "SOLUSDT", "DOGEUSDT", "DOTUSDT", "AVAXUSDT", "LTCUSDT", "LINKUSDT", "UNIUSDT", "XRPUSDT", "ICPUSDT", "FETUSDT", "NEARUSDT"}

	fmt.Printf("%-10s %-10s %-10s %-10s\n", "币种", "波动率%", "阈值0.05%", "状态")
	fmt.Println("--------------------------------------------")

	passedCount := 0
	for _, symbol := range majorCoins {
		prices, err := getKlinePricesForSymbol(gormDB, symbol, 200)
		if err != nil || len(prices) < 2 {
			fmt.Printf("%-10s %-10s %-10s %-10s\n", symbol, "N/A", "N/A", "数据不足")
			continue
		}

		// 计算波动率
		var changes []float64
		for i := 1; i < len(prices); i++ {
			change := (prices[i] - prices[i-1]) / prices[i-1] * 100
			if change < 0 {
				change = -change
			}
			changes = append(changes, change)
		}

		if len(changes) == 0 {
			continue
		}

		totalChange := 0.0
		for _, change := range changes {
			totalChange += change
		}
		avgVolatility := totalChange / float64(len(changes))

		threshold := 0.05 // 0.05%
		status := "❌ 过滤"
		if avgVolatility >= threshold {
			status = "✅ 通过"
			passedCount++
		}

		fmt.Printf("%-10s %-10.4f %-10.4f %-10s\n", symbol, avgVolatility, threshold, status)
	}

	fmt.Printf("\n通过阈值币种: %d/%d\n", passedCount, len(majorCoins))
}

func analyzeVolatilityThreshold(gormDB *gorm.DB) {
	fmt.Println("🎯 当前波动率阈值分析:")

	// 获取所有高交易量币种的波动率分布
	var volumeStats []struct {
		Symbol      string
		QuoteVolume float64
	}

	gormDB.Table("binance_24h_stats").
		Select("symbol, AVG(quote_volume) as quote_volume").
		Where("market_type = ? AND created_at >= ? AND quote_volume > 1000000",
			"spot", time.Now().Add(-24*time.Hour)).
		Group("symbol").
		Having("AVG(quote_volume) > 1000000").
		Order("AVG(quote_volume) DESC").
		Limit(50).
		Scan(&volumeStats)

	fmt.Printf("分析样本: %d个高交易量币种\n", len(volumeStats))

	// 计算波动率分布
	volatilityLevels := []float64{0.01, 0.05, 0.1, 0.2, 0.5, 1.0}
	levelCounts := make(map[float64]int)

	totalCoins := 0
	for _, stat := range volumeStats {
		prices, err := getKlinePricesForSymbol(gormDB, stat.Symbol, 200)
		if err != nil || len(prices) < 2 {
			continue
		}

		// 计算波动率
		var changes []float64
		for i := 1; i < len(prices); i++ {
			change := (prices[i] - prices[i-1]) / prices[i-1] * 100
			if change < 0 {
				change = -change
			}
			changes = append(changes, change)
		}

		if len(changes) == 0 {
			continue
		}

		totalChange := 0.0
		for _, change := range changes {
			totalChange += change
		}
		avgVolatility := totalChange / float64(len(changes))

		totalCoins++

		// 统计在各个阈值下的通过情况
		for _, level := range volatilityLevels {
			if avgVolatility >= level {
				levelCounts[level]++
			}
		}
	}

	fmt.Printf("实际波动率分布 (%d个币种):\n", totalCoins)
	for _, level := range volatilityLevels {
		count := levelCounts[level]
		percentage := float64(count) / float64(totalCoins) * 100
		status := ""
		if level == 0.05 {
			status = " ← 当前阈值"
		}
		fmt.Printf("  ≥ %.2f%%: %d个 (%.1f%%)%s\n", level, count, percentage, status)
	}

	fmt.Println("\n💡 阈值建议:")
	fmt.Println("• 0.01%: 过于宽松，包含太多低波动资产")
	fmt.Println("• 0.05%: 当前设置，可能过于严格")
	fmt.Println("• 0.10%: 相对合理，过滤明显低波动资产")
	fmt.Println("• 0.20%: 较为严格，适合激进策略")
	fmt.Println("• 0.50%: 很严格，只选择高波动资产")

	fmt.Println("\n🎯 优化建议:")
	if levelCounts[0.05] < 5 {
		fmt.Println("❌ 当前阈值(0.05%)过于严格，建议降低到0.02%或0.03%")
	} else if levelCounts[0.05] > 20 {
		fmt.Println("⚠️  当前阈值相对宽松，可以考虑提高到0.08%或0.10%")
	} else {
		fmt.Println("✅ 当前阈值(0.05%)基本合理")
	}
}

func getKlinePricesForSymbol(gormDB *gorm.DB, symbol string, limit int) ([]float64, error) {
	var klines []pdb.MarketKline
	err := gormDB.Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").
		Order("open_time DESC").
		Limit(limit).
		Find(&klines).Error

	if err != nil {
		return nil, err
	}

	// 反转顺序，从旧到新
	for i, j := 0, len(klines)-1; i < j; i, j = i+1, j-1 {
		klines[i], klines[j] = klines[j], klines[i]
	}

	prices := make([]float64, len(klines))
	for i, kline := range klines {
		price, err := strconv.ParseFloat(kline.ClosePrice, 64)
		if err != nil {
			return nil, fmt.Errorf("解析价格失败: %v", err)
		}
		prices[i] = price
	}

	return prices, nil
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
