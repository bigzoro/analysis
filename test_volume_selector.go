package main

import (
	"context"
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"

	"analysis/internal/server"
)

func main() {
	fmt.Println("=== 测试VolumeBasedSelector ===")

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

	// 4. 创建模拟的server
	mockServer := &struct {
		db interface{ DB() (*pdb.GormDB, error); GormDB() *pdb.GormDB }
	}{db: db}

	// 5. 测试VolumeBasedSelector
	fmt.Printf("\n🔍 测试VolumeBasedSelector:\n")

	selector := &server.VolumeBasedSelector{server: mockServer}
	candidates, err := selector.SelectCandidates(context.Background(), strategy, 10)

	if err != nil {
		fmt.Printf("❌ 选择器执行失败: %v\n", err)
		return
	}

	fmt.Printf("✅ 成功选择了%d个候选币种:\n", len(candidates))
	for i, symbol := range candidates {
		if i >= 5 { // 只显示前5个
			fmt.Printf("    ... 还有%d个\n", len(candidates)-5)
			break
		}
		fmt.Printf("    %d. %s\n", i+1, symbol)
	}

	fmt.Printf("\n🎉 VolumeBasedSelector测试完成！\n")
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
