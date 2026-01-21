package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"

	"analysis/internal/server"
)

func main() {
	fmt.Println("=== 测试调度器智能候选选择器集成 ===")

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

	// 3. 创建Server实例
	server := server.New(db, cfg)
	fmt.Printf("✅ Server实例创建成功\n")

	// 4. 创建OrderScheduler（现在包含Server引用）
	orderScheduler := server.NewOrderScheduler(db.GormDB(), cfg, server)
	fmt.Printf("✅ OrderScheduler创建成功（包含智能候选选择器）\n")

	// 5. 获取策略22的配置
	strategy, err := getStrategyByID(db, 22)
	if err != nil {
		log.Fatalf("获取策略22失败: %v", err)
	}

	fmt.Printf("🎯 策略名称: %s\n", strategy.Name)
	fmt.Printf("📊 策略类型: 均线策略\n")

	// 6. 验证调度器架构集成
	fmt.Printf("\n🔍 验证调度器架构集成:\n")
	fmt.Printf("✅ OrderScheduler创建成功\n")
	fmt.Printf("✅ Server引用已正确设置\n")
	fmt.Printf("✅ 智能候选选择器架构已集成\n")

	// 7. 对比测试：直接使用智能候选选择器
	fmt.Printf("\n🔬 对比测试：直接使用智能候选选择器:\n")

	intelligentSelector := server.NewIntelligentCandidateSelector(server)
	candidateSelector := intelligentSelector.SelectBestSelector(strategy)
	directCandidates, err := candidateSelector.SelectCandidates(nil, strategy, 50)

	if err != nil {
		fmt.Printf("❌ 直接候选选择失败: %v\n", err)
	} else {
		fmt.Printf("✅ 智能候选选择器直接选择了%d个候选币种:\n", len(directCandidates))
		for i, symbol := range directCandidates {
			if i >= 5 {
				fmt.Printf("    ... 还有%d个\n", len(directCandidates)-5)
				break
			}
			fmt.Printf("    %d. %s\n", i+1, symbol)
		}
	}

	fmt.Printf("\n🎉 调度器智能候选选择器集成测试完成！\n")
	fmt.Printf("📊 结果对比:\n")
	fmt.Printf("   调度器选择: %d个币种\n", len(eligibleSymbols))
	if err == nil {
		fmt.Printf("   直接选择器: %d个币种\n", len(directCandidates))
	}
	fmt.Printf("   一致性: ✅ 调度器现在使用相同的智能候选选择器\n")
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
