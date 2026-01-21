package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔧 检查 binance_futures_contracts 表的索引...")

	// 加载配置
	var cfg config.Config
	config.MustLoad("config.yaml", &cfg)
	config.ApplyProxy(&cfg)

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:          cfg.Database.DSN,
		Automigrate:  false,
		MaxOpenConns: cfg.Database.MaxOpenConns,
		MaxIdleConns: cfg.Database.MaxIdleConns,
	})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}
	defer gdb.Close()

	fmt.Println("📡 数据库连接成功")

	// 查询binance_futures_contracts表的索引
	var indexes []struct {
		Table      string `json:"table"`
		NonUnique  int    `json:"non_unique"`
		KeyName    string `json:"key_name"`
		SeqInIndex int    `json:"seq_in_index"`
		ColumnName string `json:"column_name"`
	}

	sql := `
		SELECT TABLE_NAME as table_name,
			   NON_UNIQUE as non_unique,
			   INDEX_NAME as key_name,
			   SEQ_IN_INDEX as seq_in_index,
			   COLUMN_NAME as column_name
		FROM information_schema.statistics
		WHERE table_schema = DATABASE()
		AND table_name = 'binance_futures_contracts'
		ORDER BY INDEX_NAME, SEQ_IN_INDEX
	`

	if err := gdb.GormDB().Raw(sql).Scan(&indexes).Error; err != nil {
		log.Fatalf("❌ 查询索引失败: %v", err)
	}

	fmt.Printf("📊 找到 %d 个索引:\n", len(indexes))
	for _, idx := range indexes {
		uniqueStr := "YES"
		if idx.NonUnique == 1 {
			uniqueStr = "NO"
		}
		fmt.Printf("  - 索引名: %s, 列: %s, 唯一: %s\n", idx.KeyName, idx.ColumnName, uniqueStr)
	}

	// 检查是否有idx_futures_contracts_symbol索引
	hasTargetIndex := false
	for _, idx := range indexes {
		if idx.KeyName == "idx_futures_contracts_symbol" {
			hasTargetIndex = true
			break
		}
	}

	fmt.Println("\n=== 索引状态检查 ===")
	if hasTargetIndex {
		fmt.Println("✅ 找到目标索引: idx_futures_contracts_symbol")
	} else {
		fmt.Println("❌ 未找到目标索引: idx_futures_contracts_symbol")
	}

	// 检查是否有其他symbol相关的索引
	fmt.Println("\n=== Symbol相关索引检查 ===")
	symbolIndexes := 0
	for _, idx := range indexes {
		if idx.ColumnName == "symbol" {
			uniqueStr := "YES"
			if idx.NonUnique == 1 {
				uniqueStr = "NO"
			}
			fmt.Printf("  - 索引名: %s, 唯一: %s\n", idx.KeyName, uniqueStr)
			symbolIndexes++
		}
	}

	if symbolIndexes == 0 {
		fmt.Println("❌ 未找到任何symbol相关的索引")
	}

	// 检查PRIMARY KEY
	fmt.Println("\n=== 主键检查 ===")
	for _, idx := range indexes {
		if idx.KeyName == "PRIMARY" {
			fmt.Printf("  - 主键列: %s\n", idx.ColumnName)
		}
	}

	// 分析问题
	fmt.Println("\n=== 问题分析 ===")
	fmt.Println("根据代码分析，可能的问题：")
	fmt.Println("1. GORM AutoMigrate 根据结构体标签创建索引")
	fmt.Println("2. optimization.go 中的 CreateOptimizedIndexes 也尝试创建索引")
	fmt.Println("3. 如果索引名称不匹配，可能导致冲突")

	// 检查GORM可能创建的索引名称
	fmt.Println("\n=== GORM可能的索引名称 ===")
	fmt.Println("GORM为 uniqueIndex 标签通常创建的索引名:")
	fmt.Println("  - idx_binance_futures_contracts_symbol")
	fmt.Println("  - idx_binance_futures_contracts_status")
	fmt.Println("  - idx_binance_futures_contracts_updated_at")

	// 检查这些索引是否存在
	gormIndexNames := []string{
		"idx_binance_futures_contracts_symbol",
		"idx_binance_futures_contracts_status",
		"idx_binance_futures_contracts_updated_at",
	}

	fmt.Println("\n=== GORM索引存在性检查 ===")
	for _, gormIdx := range gormIndexNames {
		exists := false
		for _, idx := range indexes {
			if idx.KeyName == gormIdx {
				exists = true
				break
			}
		}
		if exists {
			fmt.Printf("✅ %s 存在\n", gormIdx)
		} else {
			fmt.Printf("❌ %s 不存在\n", gormIdx)
		}
	}

	// 建议解决方案
	fmt.Println("\n=== 建议解决方案 ===")
	fmt.Println("1. 检查 optimization.go 中的索引定义是否正确")
	fmt.Println("2. 考虑移除重复的索引定义，或统一索引名称")
	fmt.Println("3. 或者修改 CreateOptimizedIndexes 函数，增加更完善的检查逻辑")
}
