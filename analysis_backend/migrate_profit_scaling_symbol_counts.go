package main

import (
	"analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("开始添加profit_scaling_symbol_counts字段到trading_strategies表...")

	// 连接数据库
	gdb, err := db.OpenMySQL(db.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	// 检查字段是否已存在
	var result struct {
		FieldExists int
	}

	checkQuery := `
		SELECT COUNT(*) as field_exists
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = DATABASE()
		AND TABLE_NAME = 'trading_strategies'
		AND COLUMN_NAME = 'profit_scaling_symbol_counts'
	`

	dbConn, err := gdb.DB()
	if err != nil {
		log.Fatalf("获取数据库连接失败: %v", err)
	}

	if err := dbConn.Raw(checkQuery).Scan(&result).Error; err != nil {
		log.Fatalf("检查字段是否存在失败: %v", err)
	}

	if result.FieldExists > 0 {
		fmt.Println("字段 profit_scaling_symbol_counts 已存在，跳过迁移")
		return
	}

	// 添加新字段
	addColumnQuery := `
		ALTER TABLE trading_strategies
		ADD COLUMN profit_scaling_symbol_counts JSON DEFAULT ('{}')
		COMMENT '各币种的盈利加仓计数器，格式：{"BTCUSDT": 1, "ETHUSDT": 0}'
	`

	if err := dbConn.Exec(addColumnQuery).Error; err != nil {
		log.Fatalf("添加字段失败: %v", err)
	}

	fmt.Println("✅ 成功添加 profit_scaling_symbol_counts 字段")
	fmt.Println("🎉 数据库迁移完成！")
}
