package main

import (
	"context"
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"
)

func main() {
	fmt.Println("=== 测试智能候选币种选择器 ===")

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

	// 4. 测试不同的候选选择器
	testCandidateSelectors(db, strategy)
}

// 模拟Server结构
type MockServer struct {
	db pdb.Database
}

func (m *MockServer) DB() (*pdb.GormDB, error) {
	return m.db.DB()
}

func (m *MockServer) GormDB() *pdb.GormDB {
	return m.db.GormDB()
}

// 模拟其他必需的方法
func (m *MockServer) getKlinePricesForSymbol(symbol string, limit int) ([]float64, error) {
	// 简化的实现
	return []float64{100, 101, 102}, nil
}

func (m *MockServer) getMarketDataForSymbol(symbol string) interface{} {
	// 简化的实现
	return map[string]interface{}{
		"HasSpot":   true,
		"HasFutures": true,
		"MarketCap": 1000000.0,
	}
}

func testCandidateSelectors(db pdb.Database, strategy *pdb.TradingStrategy) {
	ctx := context.Background()

	// 创建模拟的server
	mockServer := &MockServer{db: db}

	// 测试不同的选择器
	fmt.Printf("由于需要完整的Server实现，这里只展示选择器架构设计:\n")

	fmt.Printf("\n🏗️  可用的选择器类型:\n")
	fmt.Printf("  1. VolumeBasedSelector - 基于交易量选择\n")
	fmt.Printf("     优点: 选择活跃币种，数据质量好\n")
	fmt.Printf("     适用: 均线策略、套利策略\n")

	fmt.Printf("  2. MarketCapBasedSelector - 基于市值选择\n")
	fmt.Printf("     优点: 选择大盘股，稳定性好\n")
	fmt.Printf("     适用: 保守策略、价值投资\n")

	fmt.Printf("  3. StrategySpecificSelector - 策略专用选择\n")
	fmt.Printf("     优点: 根据策略特点智能选择\n")
	fmt.Printf("     适用: 所有策略类型\n")

	fmt.Printf("  4. IntelligentCandidateSelector - 智能自动选择\n")
	fmt.Printf("     优点: 自动为策略选择最优选择器\n")
	fmt.Printf("     适用: 新手用户或复杂策略\n")

	// 测试智能选择器
	fmt.Printf("\n🎯 测试智能选择器:\n")
	// 暂时跳过智能选择器测试，因为需要完整的Server实现
	fmt.Printf("智能选择器测试需要完整的Server实现，暂时跳过\n")

	fmt.Printf("为均线策略推荐的选择器: %s\n", bestSelector.GetStrategyType())

	candidates, err := bestSelector.SelectCandidates(ctx, strategy, 10)
	if err != nil {
		fmt.Printf("❌ 智能选择失败: %v\n", err)
	} else {
		fmt.Printf("✅ 智能选择了%d个候选币种:\n", len(candidates))
		for i, symbol := range candidates {
			if i >= 5 {
				fmt.Printf("    ... 还有%d个\n", len(candidates)-5)
				break
			}
			fmt.Printf("    %d. %s\n", i+1, symbol)
		}
	}

	fmt.Printf("\n📊 选择器对比分析:\n")
	fmt.Printf("交易量选择器: 适合均线策略，活跃市场数据质量好\n")
	fmt.Printf("市值选择器: 适合保守策略，大盘股更稳定\n")
	fmt.Printf("策略专用选择器: 根据策略特点智能选择\n")
	fmt.Printf("智能选择器: 自动为策略选择最优选择器\n")
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

func getStrategyByID(db pdb.Database, strategyID int) (*pdb.TradingStrategy, error) {
	gdb := db.GormDB()

	var strategy pdb.TradingStrategy
	err := gdb.First(&strategy, strategyID).Error
	if err != nil {
		return nil, fmt.Errorf("查询策略失败: %v", err)
	}

	return &strategy, nil
}
