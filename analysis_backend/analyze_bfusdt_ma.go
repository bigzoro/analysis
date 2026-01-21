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
	"os"
)

func main() {
	fmt.Println("=== BFUSDUSDT 均线分析脚本 ===")

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

	// 3. 分析BFUSDUSDT的均线
	analyzeBFUSDUTMovingAverage(db)

	fmt.Println("\n=== 分析完成 ===")
}

// 分析BFUSDUSDT的均线情况
func analyzeBFUSDUTMovingAverage(db pdb.Database) {
	symbol := "BFUSDUSDT"
	shortPeriod := 5
	longPeriod := 20

	fmt.Printf("📊 分析币种: %s\n", symbol)
	fmt.Printf("📈 短期均线: SMA(%d)\n", shortPeriod)
	fmt.Printf("📉 长期均线: SMA(%d)\n", longPeriod)

	// 1. 检查K线数据
	gormDB, _ := db.DB()
	var klineCount int64
	gormDB.Model(&pdb.MarketKline{}).Where("symbol = ? AND kind = ? AND `interval` = ?", symbol, "spot", "1h").Count(&klineCount)
	fmt.Printf("💾 K线数据条数: %d\n", klineCount)

	if klineCount < 50 {
		fmt.Printf("❌ K线数据不足，至少需要50条记录用于均线分析\n")
		return
	}

	// 2. 获取价格数据
	prices, timestamps, err := getKlinePricesForSymbol(db, symbol, 200) // 获取最近200个小时的数据
	if err != nil {
		fmt.Printf("❌ 获取价格数据失败: %v\n", err)
		return
	}

	fmt.Printf("📊 成功获取%d个价格数据点\n", len(prices))

	if len(prices) < longPeriod {
		fmt.Printf("❌ 数据点不足，需要至少%d个点，当前%d个\n", longPeriod, len(prices))
		return
	}

	// 3. 计算均线
	ti := analysis.NewTechnicalIndicators()
	shortMA := ti.CalculateMovingAverage(prices, shortPeriod, analysis.SMA)
	longMA := ti.CalculateMovingAverage(prices, longPeriod, analysis.SMA)

	fmt.Printf("✅ 均线计算完成\n")
	fmt.Printf("   短期均线数据点: %d\n", len(shortMA))
	fmt.Printf("   长期均线数据点: %d\n", len(longMA))

	if len(shortMA) == 0 || len(longMA) == 0 {
		fmt.Printf("❌ 均线计算失败\n")
		return
	}

	// 4. 检测交叉信号
	goldenCross, deathCross := ti.DetectMACross(shortMA, longMA)
	fmt.Printf("\n🎯 交叉信号检测:\n")
	fmt.Printf("   金叉信号: %v\n", goldenCross)
	fmt.Printf("   死叉信号: %v\n", deathCross)

	// 5. 显示当前均线状态
	fmt.Printf("\n📈 当前均线状态:\n")
	if len(shortMA) > 0 && len(longMA) > 0 {
		lastShort := shortMA[len(shortMA)-1]
		lastLong := longMA[len(longMA)-1]
		fmt.Printf("   最新短期均线(SMA%d): %.6f\n", shortPeriod, lastShort)
		fmt.Printf("   最新长期均线(SMA%d): %.6f\n", longPeriod, lastLong)

		if lastShort > lastLong {
			fmt.Printf("   📈 当前趋势: 短期均线在长期均线之上\n")
		} else if lastShort < lastLong {
			fmt.Printf("   📉 当前趋势: 短期均线在长期均线之下\n")
		} else {
			fmt.Printf("   ➖ 当前趋势: 短期均线与长期均线持平\n")
		}
	}

	// 6. 显示最近的价格数据
	fmt.Printf("\n💰 最近5个价格数据点:\n")
	for i := len(prices) - 5; i < len(prices); i++ {
		if i >= 0 {
			timestamp := timestamps[i].Format("01-02 15:04")
			fmt.Printf("   %s: $%.6f\n", timestamp, prices[i])
		}
	}

	// 7. 显示最近的均线交叉历史
	fmt.Printf("\n📊 最近5个交叉检测结果:\n")
	maxCheck := len(shortMA) - 1
	if maxCheck > 5 {
		maxCheck = 5
	}

	for i := len(shortMA) - maxCheck; i < len(shortMA); i++ {
		if i > 0 && i < len(shortMA) && i < len(longMA) {
			currShort := shortMA[i]
			currLong := longMA[i]
			prevShort := shortMA[i-1]
			prevLong := longMA[i-1]

			// 检测交叉
			gc := prevShort <= prevLong && currShort > currLong
			dc := prevShort >= prevLong && currShort < currLong

			timestamp := timestamps[i].Format("01-02 15:04")
			status := "➖ 无交叉"
			if gc {
				status = "📈 金叉"
			} else if dc {
				status = "📉 死叉"
			}

			fmt.Printf("   %s: SMA5=%.4f, SMA20=%.4f | %s\n",
				timestamp, currShort, currLong, status)
		}
	}

	// 8. 趋势分析
	fmt.Printf("\n📈 趋势分析:\n")
	uptrend, downtrend := ti.DetectMATrend(shortMA, longMA)
	fmt.Printf("   上升趋势: %v\n", uptrend)
	fmt.Printf("   下降趋势: %v\n", downtrend)

	if uptrend {
		fmt.Printf("   ✅ 符合上升趋势条件\n")
	} else if downtrend {
		fmt.Printf("   ✅ 符合下降趋势条件\n")
	} else {
		fmt.Printf("   ⚠️  无明确趋势\n")
	}

	// 9. 数据质量检查
	fmt.Printf("\n🔍 数据质量检查:\n")

	// 检查价格合理性
	validPrices := 0
	totalPrices := len(prices)
	for _, price := range prices {
		if price > 0 && price < 1000000 { // 假设加密货币价格不会超过100万美元
			validPrices++
		}
	}
	fmt.Printf("   有效价格: %d/%d (%.1f%%)\n", validPrices, totalPrices, float64(validPrices)/float64(totalPrices)*100)

	// 检查数据连续性
	if len(timestamps) >= 2 {
		gaps := 0
		expectedInterval := time.Hour // 1小时K线
		for i := 1; i < len(timestamps); i++ {
			actualInterval := timestamps[i].Sub(timestamps[i-1])
			if actualInterval > expectedInterval*2 { // 允许1小时的误差
				gaps++
			}
		}
		fmt.Printf("   数据连续性: %d个时间间隔异常\n", gaps)
	}

	// 10. 结论
	fmt.Printf("\n🎯 分析结论:\n")
	if goldenCross {
		fmt.Printf("   ✅ BFUSDUSDT当前触发金叉信号，可以买入\n")
	} else if deathCross {
		fmt.Printf("   ✅ BFUSDUSDT当前触发死叉信号，可以卖出\n")
	} else {
		fmt.Printf("   ⚠️  BFUSDUSDT当前无明确的均线交叉信号\n")
	}

	if uptrend {
		fmt.Printf("   📈 整体趋势向上，支持做多\n")
	} else if downtrend {
		fmt.Printf("   📉 整体趋势向下，支持做空\n")
	} else {
		fmt.Printf("   ➖ 整体趋势不明朗\n")
	}
}

func getKlinePricesForSymbol(db pdb.Database, symbol string, limit int) ([]float64, []time.Time, error) {
	gormDB, _ := db.DB()
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
