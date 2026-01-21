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
	fmt.Println("=== 测试策略扫描器重构 ===")

	// 1. 读取配置文件
	cfg, err := loadConfig("analysis_backend/config.yaml")
	if err != nil {
		log.Fatalf("加载配置文件失败: %v", err)
	}

	// 2. 连接数据库
	db, err := connectDatabase(cfg.Database)
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	fmt.Printf("✅ 数据库连接成功\n")

	// 3. 获取策略22的配置
	strategy, err := getStrategyByID(db, 22)
	if err != nil {
		log.Fatalf("获取策略22失败: %v", err)
	}

	fmt.Printf("🎯 策略名称: %s\n", strategy.Name)
	fmt.Printf("📊 策略类型: 均线策略\n")
	fmt.Printf("⚙️  均线配置: %s(%d,%d)\n",
		strategy.Conditions.MAType,
		strategy.Conditions.ShortMAPeriod,
		strategy.Conditions.LongMAPeriod)

	// 4. 测试扫描器选择逻辑
	fmt.Printf("\n🔍 测试扫描器选择逻辑:\n")

	// 测试均线策略
	fmt.Printf("均线策略扫描器: %s\n", getScannerTypeForStrategy(strategy))

	// 测试传统策略
	traditionalStrategy := *strategy
	traditionalStrategy.Conditions.MovingAverageEnabled = false
	traditionalStrategy.Conditions.ShortOnGainers = true
	fmt.Printf("传统策略扫描器: %s\n", getScannerTypeForStrategy(&traditionalStrategy))

	// 测试套利策略
	arbitrageStrategy := *strategy
	arbitrageStrategy.Conditions.MovingAverageEnabled = false
	arbitrageStrategy.Conditions.CrossExchangeArbEnabled = true
	fmt.Printf("套利策略扫描器: %s\n", getScannerTypeForStrategy(&arbitrageStrategy))

	fmt.Printf("\n✅ 扫描器重构测试完成！\n")
	fmt.Printf("📁 相关文件:\n")
	fmt.Printf("  - strategy_scanner_traditional.go (传统策略扫描)\n")
	fmt.Printf("  - strategy_scanner_moving_average.go (均线策略扫描)\n")
	fmt.Printf("  - strategy_scanner_arbitrage.go (套利策略扫描)\n")
	fmt.Printf("  - strategy_execution.go (扫描器注册表)\n")
}

// 模拟扫描器选择逻辑
func getScannerTypeForStrategy(strategy *pdb.TradingStrategy) string {
	conditions := strategy.Conditions

	// 优先检查特殊策略
	if conditions.TriangleArbEnabled {
		return "arbitrage"
	}

	// 检查均线策略
	if conditions.MovingAverageEnabled {
		return "moving_average"
	}

	// 检查传统策略
	if conditions.ShortOnGainers || conditions.LongOnSmallGainers {
		return "traditional"
	}

	// 检查其他套利策略
	if conditions.CrossExchangeArbEnabled || conditions.SpotFutureArbEnabled ||
		conditions.StatArbEnabled || conditions.FuturesSpotArbEnabled {
		return "arbitrage"
	}

	// 默认使用传统策略扫描器
	return "traditional"
}

// 其他辅助函数（复用之前的代码）
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

func getStrategyByID(db pdb.Database, strategyID int) (*pdb.TradingStrategy, error) {
	gdb := db.GormDB()

	var strategy pdb.TradingStrategy
	err := gdb.First(&strategy, strategyID).Error
	if err != nil {
		return nil, fmt.Errorf("查询策略失败: %v", err)
	}

	return &strategy, nil
}
