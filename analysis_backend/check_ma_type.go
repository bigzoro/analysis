package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	fmt.Println("=== 检查策略ID 23的均线类型 ===")

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

	// 3. 查询策略ID 23
	strategy, err := getStrategyByID(gormDB, 23)
	if err != nil {
		log.Fatalf("查询策略失败: %v", err)
	}

	// 4. 显示均线类型
	fmt.Printf("📊 策略ID 23均线类型配置:\n")
	fmt.Printf("   策略名称: %s\n", strategy.Name)
	fmt.Printf("   均线类型 (ma_type): %s\n", strategy.Conditions.MAType)
	fmt.Printf("   短期均线周期: %d\n", strategy.Conditions.ShortMAPeriod)
	fmt.Printf("   长期均线周期: %d\n", strategy.Conditions.LongMAPeriod)

	// 5. 解释均线类型
	explainMAType(strategy.Conditions.MAType)

	fmt.Println("\n=== 查询完成 ===")
}

func getStrategyByID(gormDB *gorm.DB, id uint) (*pdb.TradingStrategy, error) {
	var strategy pdb.TradingStrategy
	err := gormDB.Preload("Conditions").Where("id = ?", id).First(&strategy).Error
	if err != nil {
		return nil, fmt.Errorf("策略ID %d不存在: %v", id, err)
	}
	return &strategy, nil
}

func explainMAType(maType string) {
	fmt.Printf("\n🔍 均线类型说明:\n")

	switch maType {
	case "SMA":
		fmt.Println("   📈 SMA (Simple Moving Average) - 简单移动平均线")
		fmt.Println("   ✨ 特点:")
		fmt.Println("     • 计算简单直接")
		fmt.Println("     • 对所有价格点平等对待")
		fmt.Println("     • 对价格变化反应相对平滑")
		fmt.Println("     • 适合长期趋势跟踪")
		fmt.Println("   🎯 适用场景:")
		fmt.Println("     • 趋势明显的稳定市场")
		fmt.Println("     • 需要平滑信号的保守策略")
		fmt.Println("     • 避免短期噪音干扰")

	case "EMA":
		fmt.Println("   📈 EMA (Exponential Moving Average) - 指数移动平均线")
		fmt.Println("   ✨ 特点:")
		fmt.Println("     • 对近期价格赋予更高权重")
		fmt.Println("     • 对价格变化反应更灵敏")
		fmt.Println("     • 更早发现趋势变化")
		fmt.Println("     • 适合捕捉短期机会")
		fmt.Println("   🎯 适用场景:")
		fmt.Println("     • 波动较大的活跃市场")
		fmt.Println("     • 需要快速反应的激进策略")
		fmt.Println("     • 追求更高胜率的交易")

	default:
		fmt.Printf("   ⚠️ 未知均线类型: %s\n", maType)
		fmt.Println("   使用默认类型: SMA")
	}

	fmt.Printf("\n💡 当前配置的均线周期: %d日短期线 vs %d日长期线\n",
		strategy.Conditions.ShortMAPeriod, strategy.Conditions.LongMAPeriod)
}

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
