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
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
)

func main() {
	fmt.Println("=== ATUSDT 验证过程调试 ===")

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

	// 3. 模拟ATUSDT的完整验证过程
	fmt.Println("🔍 模拟ATUSDT完整验证过程...\n")

	symbol := "ATUSDT"
	sessionID := fmt.Sprintf("%d", time.Now().UnixMilli())

	fmt.Printf("[MA-Scan][%s][Session:%s] 开始检查均线条件\n", symbol, sessionID)

	// 1. 获取价格数据
	prices, err := getKlinePricesForSymbol(gormDB, symbol, 200)
	if err != nil {
		fmt.Printf("[MA-Scan][%s][Session:%s] 获取价格数据失败: %v\n", symbol, sessionID, err)
		return
	}

	if len(prices) < 25 {
		fmt.Printf("[MA-Scan][%s][Session:%s] 价格数据不足，至少需要25个数据点，当前%d个\n", symbol, sessionID, len(prices))
		return
	}

	fmt.Printf("[MA-Scan][%s][Session:%s] 获取到%d个价格数据点\n", symbol, sessionID, len(prices))

	// 2. 计算均线
	ti := analysis.NewTechnicalIndicators()
	shortMA := ti.CalculateMovingAverage(prices, 5, analysis.SMA)
	longMA := ti.CalculateMovingAverage(prices, 20, analysis.SMA)

	if len(shortMA) == 0 || len(longMA) == 0 {
		fmt.Printf("[MA-Scan][%s][Session:%s] 均线计算失败\n", symbol, sessionID)
		return
	}

	fmt.Printf("[MA-Scan][%s][Session:%s] 均线计算完成 - SMA5: %.6f, SMA20: %.6f\n", symbol, sessionID, shortMA[len(shortMA)-1], longMA[len(longMA)-1])

	// 3. 波动率验证
	fmt.Println("\n📊 波动率验证:")
	volatilityValid := server.ValidateVolatilityForMA(symbol, prices, 0.05) // 0.05%
	fmt.Printf("   波动率验证 (≥0.05%%): %v\n", volatilityValid)

	// 手动计算波动率
	var changes []float64
	for i := 1; i < len(prices); i++ {
		change := (prices[i] - prices[i-1]) / prices[i-1] * 100
		changes = append(changes, change)
	}

	if len(changes) > 0 {
		totalChange := 0.0
		for _, change := range changes {
			if change < 0 {
				change = -change
			}
			totalChange += change
		}
		avgVolatility := totalChange / float64(len(changes))
		fmt.Printf("   实际平均波动率: %.4f%%\n", avgVolatility)
		fmt.Printf("   波动率阈值: 0.05%%\n")
		fmt.Printf("   验证结果: %v\n", avgVolatility >= 0.05)
	}

	// 4. 趋势强度验证
	fmt.Println("\n📊 趋势强度验证:")
	trendValid := server.ValidateTrendStrength(shortMA, longMA, 0.001) // 0.1%
	fmt.Printf("   趋势强度验证 (≥0.1%%): %v\n", trendValid)

	latestShort := shortMA[len(shortMA)-1]
	latestLong := longMA[len(longMA)-1]
	trendStrength := (latestShort - latestLong) / latestLong
	if trendStrength < 0 {
		trendStrength = -trendStrength
	}
	fmt.Printf("   实际趋势强度: %.4f%%\n", trendStrength*100)
	fmt.Printf("   趋势强度阈值: 0.1%%\n")
	fmt.Printf("   验证结果: %v\n", trendStrength >= 0.001)

	// 5. 信号质量评估
	fmt.Println("\n📊 信号质量评估:")
	signalQuality := server.AssessSignalQuality(shortMA, longMA, prices)
	fmt.Printf("   信号质量评分: %.3f\n", signalQuality)
	fmt.Printf("   信号质量验证 (≥0.5): %v\n", signalQuality >= 0.5)

	// 6. 检测交叉信号
	goldenCross, deathCross := ti.DetectMACross(shortMA, longMA)
	fmt.Println("\n📊 交叉信号检测:")
	fmt.Printf("   金叉信号: %v\n", goldenCross)
	fmt.Printf("   死叉信号: %v\n", deathCross)

	// 7. 最终判断
	fmt.Println("\n🎯 综合验证结果:")
	allValid := volatilityValid && trendValid && signalQuality >= 0.5
	fmt.Printf("   波动率验证: %v\n", volatilityValid)
	fmt.Printf("   趋势强度验证: %v\n", trendValid)
	fmt.Printf("   信号质量验证: %v\n", signalQuality >= 0.5)
	fmt.Printf("   交叉信号存在: %v\n", goldenCross || deathCross)
	fmt.Printf("   总体验证结果: %v\n", allValid)

	if allValid {
		action := "buy"
		if deathCross {
			action = "sell"
		}
		fmt.Printf("\n✅ ATUSDT 符合均线策略条件!\n")
		fmt.Printf("   推荐操作: %s\n", action)
		fmt.Printf("   原因: 符合所有验证条件\n")
	} else {
		fmt.Printf("\n❌ ATUSDT 不符合均线策略条件\n")
		fmt.Printf("   原因: 未通过一项或多项验证\n")
	}

	fmt.Println("\n=== 验证过程调试完成 ===")
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
