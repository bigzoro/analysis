package main

import (
	"fmt"
	"log"
	"strings"

	pdb "analysis/internal/db"
	"gopkg.in/yaml.v3"
	"os"
)

type Config struct {
	Database struct {
		DSN          string `yaml:"dsn"`
		Automigrate  bool   `yaml:"automigrate"`
		MaxOpenConns int    `yaml:"max_open_conns"`
		MaxIdleConns int    `yaml:"max_idle_conns"`
	} `yaml:"database"`
}

func main() {
	// 加载配置
	cfg, err := loadConfig("config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  false,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("获取数据库实例失败: %v", err)
	}

	// 查询所有表名
	var tables []map[string]interface{}
	gdb.Raw("SHOW TABLES").Scan(&tables)

	fmt.Println("🔍 数据库中的表:")
	fmt.Println("=====================================")

	indicatorTables := make([]string, 0)
	for _, tableMap := range tables {
		for _, tableName := range tableMap {
			tableStr := fmt.Sprintf("%v", tableName)
			if strings.Contains(strings.ToLower(tableStr), "indicator") ||
			   strings.Contains(strings.ToLower(tableStr), "technical") {
				indicatorTables = append(indicatorTables, tableStr)
			}
		}
	}

	if len(indicatorTables) == 0 {
		fmt.Println("❌ 未找到技术指标相关的表")
	} else {
		fmt.Printf("📊 找到 %d 个技术指标相关表:\n", len(indicatorTables))
		for _, table := range indicatorTables {
			fmt.Printf("  - %s\n", table)
		}
	}

	// 检查FILUSDT的技术指标数据
	fmt.Println("\n📈 检查FILUSDT技术指标数据:")
	for _, table := range indicatorTables {
		var count int64
		err := gdb.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE symbol = 'FILUSDT'", table)).Scan(&count).Error
		if err == nil {
			fmt.Printf("  %s: %d 条记录\n", table, count)

			// 检查表结构
			var columns []map[string]interface{}
			gdb.Raw(fmt.Sprintf("DESCRIBE %s", table)).Scan(&columns)

			fmt.Printf("    表结构:\n")
			for _, col := range columns {
				field := fmt.Sprintf("%v", col["Field"])
				fieldType := fmt.Sprintf("%v", col["Type"])
				fmt.Printf("      %s: %s\n", field, fieldType)
			}

			// 如果有FILUSDT数据，显示一条记录
			if count > 0 {
				var record map[string]interface{}
				gdb.Raw(fmt.Sprintf("SELECT * FROM %s WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1", table)).Scan(&record)
				fmt.Printf("    最新记录: %+v\n", record)
			}
		} else {
			fmt.Printf("  %s: 查询失败 - %v\n", table, err)
		}
	}
}

func loadConfig(configPath string) (*Config, error) {
	file, err := os.Open(configPath)
	if err != nil {
		return nil, fmt.Errorf("打开配置文件失败: %v", err)
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %v", err)
	}

	return &cfg, nil
}