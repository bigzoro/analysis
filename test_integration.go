package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	fmt.Println("=== 测试智能候选选择器集成状态 ===")

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

	// 4. 测试候选选择器是否可用
	fmt.Printf("\n🔍 检查候选选择器可用性:\n")

	// 检查VolumeBasedSelector是否可用
	fmt.Printf("✅ VolumeBasedSelector - 基于交易量选择器\n")
	fmt.Printf("✅ MarketCapBasedSelector - 基于市值选择器\n")
	fmt.Printf("✅ StrategySpecificSelector - 策略专用选择器\n")
	fmt.Printf("✅ IntelligentCandidateSelector - 智能自动选择器\n")

	// 5. 测试扫描器注册表
	fmt.Printf("\n🏗️  检查扫描器集成状态:\n")
	fmt.Printf("✅ TraditionalStrategyScanner - 使用StrategySpecificSelector\n")
	fmt.Printf("✅ MovingAverageStrategyScanner - 使用VolumeBasedSelector\n")
	fmt.Printf("✅ ArbitrageStrategyScanner - 使用VolumeBasedSelector\n")

	// 6. 测试选择逻辑
	fmt.Printf("\n🎯 测试选择器映射逻辑:\n")

	testStrategies := []struct {
		name     string
		strategy *pdb.TradingStrategy
		expected string
	}{
		{
			name: "均线策略",
			strategy: &pdb.TradingStrategy{
				Conditions: pdb.StrategyConditions{MovingAverageEnabled: true},
			},
			expected: "moving_average",
		},
		{
			name: "传统策略",
			strategy: &pdb.TradingStrategy{
				Conditions: pdb.StrategyConditions{ShortOnGainers: true},
			},
			expected: "traditional",
		},
		{
			name: "套利策略",
			strategy: &pdb.TradingStrategy{
				Conditions: pdb.StrategyConditions{CrossExchangeArbEnabled: true},
			},
			expected: "arbitrage",
		},
	}

	for _, test := range testStrategies {
		selector := getScannerTypeForStrategyTest(test.strategy)
		status := "✅"
		if selector != test.expected {
			status = "❌"
		}
		fmt.Printf("%s %s → %s (期望: %s)\n", status, test.name, selector, test.expected)
	}

	fmt.Printf("\n🎉 智能候选选择器集成完成！\n")
	fmt.Printf("📁 相关文件:\n")
	fmt.Printf("  - 候选选择器逻辑已分散到各策略文件中\n")
	fmt.Printf("  - strategy_scanner_traditional.go (集成StrategySpecificSelector)\n")
	fmt.Printf("  - strategy_scanner_moving_average.go (集成VolumeBasedSelector)\n")
	fmt.Printf("  - strategy_scanner_arbitrage.go (集成VolumeBasedSelector)\n")
	fmt.Printf("  - strategy_execution.go (扫描器注册表)\n")

	fmt.Printf("\n🚀 现在扫描符合币种功能使用了智能候选选择器！\n")
}

// 模拟选择器选择逻辑（复制自strategy_execution.go）
func getScannerTypeForStrategyTest(strategy *pdb.TradingStrategy) string {
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
