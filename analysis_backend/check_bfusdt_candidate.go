package main

import (
	"fmt"
	"log"
	"time"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"gorm.io/gorm"
	"os"
)

func main() {
	fmt.Println("=== 检查BFUSDUSDT是否在候选名单中 ===")

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

	// 3. 获取策略22
	var strategy pdb.TradingStrategy
	err = gormDB.First(&strategy, 22).Error
	if err != nil {
		log.Fatalf("获取策略22失败: %v", err)
	}

	fmt.Printf("策略ID: %d, 名称: %s\n", strategy.ID, strategy.Name)
	fmt.Printf("均线策略启用: %v\n", strategy.Conditions.MovingAverageEnabled)

	// 4. 模拟VolumeBasedSelector的逻辑
	fmt.Println("\n=== 模拟VolumeBasedSelector选择逻辑 ===")

	// 查询最近24小时的交易统计
	var volumeStats []struct {
		Symbol      string
		Volume      float64
		QuoteVolume float64
		Count       int64
	}

	err = gormDB.Table("binance_24h_stats").
		Select("symbol, AVG(volume) as volume, AVG(quote_volume) as quote_volume, COUNT(*) as count").
		Where("market_type = ? AND created_at >= ?", "spot", time.Now().Add(-24*time.Hour)).
		Group("symbol").
		Having("COUNT(*) >= 1").
		Order("AVG(quote_volume) DESC").
		Limit(55). // 多取一些备用
		Scan(&volumeStats).Error

	if err != nil {
		log.Printf("查询交易量数据失败: %v", err)
		return
	}

	// 筛选出有足够交易量的币种
	var candidates []string
	for _, stat := range volumeStats {
		if stat.QuoteVolume > 1000000 { // 24h交易量超过100万美元
			candidates = append(candidates, stat.Symbol)
			if len(candidates) >= 50 {
				break
			}
		}
	}

	fmt.Printf("筛选出%d个候选币种（交易量>100万美元）\n", len(candidates))

	// 检查BFUSDUSDT是否在候选名单中
	found := false
	rank := -1
	for i, symbol := range candidates {
		if symbol == "BFUSDUSDT" {
			found = true
			rank = i + 1
			break
		}
	}

	fmt.Printf("\nBFUSDUSDT检查结果:\n")
	if found {
		fmt.Printf("✅ BFUSDUSDT在候选名单中，排名 #%d\n", rank)
		fmt.Printf("这意味着它应该会被MovingAverageStrategyScanner检查\n")
	} else {
		fmt.Printf("❌ BFUSDUSDT不在候选名单中\n")
		fmt.Printf("这解释了为什么日志中没有显示BFUSDUSDT的检查信息\n")

		// 检查BFUSDUSDT的交易量数据
		fmt.Printf("\n=== 检查BFUSDUSDT的交易量数据 ===\n")
		checkBFUSDUTVolumeData(gormDB)
	}

	// 显示前10个候选币种
	fmt.Printf("\n前10个候选币种:\n")
	for i, symbol := range candidates {
		if i >= 10 {
			break
		}
		fmt.Printf("%d. %s\n", i+1, symbol)
	}

	fmt.Println("\n=== 检查完成 ===")
}

func checkBFUSDUTVolumeData(gormDB *gorm.DB) {
	// 查询BFUSDUSDT最近24小时的交易量统计
	var volumeStats struct {
		Symbol      string
		QuoteVolume float64
		Volume      float64
		Count       int64
	}

	err := gormDB.Table("binance_24h_stats").
		Select("symbol, AVG(quote_volume) as quote_volume, AVG(volume) as volume, COUNT(*) as count").
		Where("symbol = ? AND market_type = ? AND created_at >= ?", "BFUSDUSDT", "spot", time.Now().Add(-24*time.Hour)).
		Group("symbol").
		Scan(&volumeStats).Error

	if err != nil {
		fmt.Printf("查询BFUSDUSDT交易量数据失败: %v\n", err)
		return
	}

	fmt.Printf("BFUSDUSDT 24h统计:\n")
	fmt.Printf("  平均交易量: %.0f\n", volumeStats.Volume)
	fmt.Printf("  平均报价交易量: %.0f USD\n", volumeStats.QuoteVolume)
	fmt.Printf("  记录数: %d\n", volumeStats.Count)

	if volumeStats.QuoteVolume >= 1000000 {
		fmt.Printf("✅ 符合VolumeBasedSelector条件 (>=100万美元)\n")
	} else {
		fmt.Printf("❌ 不符合VolumeBasedSelector条件 (<100万美元)\n")
		fmt.Printf("   需要至少100万美元的24h交易量才能入选\n")
	}

	// 比较排名
	var rankInfo struct {
		Rank int
	}
	gormDB.Raw(`
		SELECT COUNT(*) + 1 as rank
		FROM (
			SELECT symbol, AVG(quote_volume) as avg_quote_volume
			FROM binance_24h_stats
			WHERE market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 24 HOUR)
			GROUP BY symbol
			HAVING AVG(quote_volume) > ?
		) as ranked_symbols
	`, volumeStats.QuoteVolume).Scan(&rankInfo)

	fmt.Printf("BFUSDUSDT的交易量排名: #%d (1为最高)\n", rankInfo.Rank)
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
