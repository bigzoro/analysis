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
	fmt.Println("=== 测试OrderScheduler初始化状态 ===")

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

	// 4. 检查OrderScheduler是否已初始化
	if server.GetOrderScheduler() != nil {
		fmt.Printf("✅ OrderScheduler已正确初始化\n")

		// 测试OrderScheduler的基本功能
		orderScheduler := server.GetOrderScheduler()
		fmt.Printf("✅ OrderScheduler引用有效\n")

	} else {
		fmt.Printf("❌ OrderScheduler未初始化\n")
	}

	fmt.Printf("\n🎉 OrderScheduler初始化测试完成！\n")

	// 显示配置信息
	fmt.Printf("📊 配置信息:\n")
	fmt.Printf("   EnableDataAnalysis: %v\n", cfg.Services.EnableDataAnalysis)
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
