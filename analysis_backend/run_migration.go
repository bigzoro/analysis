package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🔄 执行数据库迁移: 添加保证金损失止损字段")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer db.Close()

	// 执行迁移SQL
	migrationSQL := `
		-- 添加保证金损失止损字段到trading_strategies表
		ALTER TABLE trading_strategies
		    ADD COLUMN enable_margin_loss_stop_loss TINYINT(1) DEFAULT 0 COMMENT '启用保证金损失止损',
		    ADD COLUMN margin_loss_stop_loss_percent DECIMAL(5,2) DEFAULT 30.00 COMMENT '保证金损失止损百分比';
	`

	_, err = db.Exec(migrationSQL)
	if err != nil {
		log.Printf("❌ 数据库迁移失败: %v", err)

		// 检查是否已经存在这些字段
		checkSQL := `
			SELECT COLUMN_NAME
			FROM INFORMATION_SCHEMA.COLUMNS
			WHERE TABLE_SCHEMA = 'analysis'
			AND TABLE_NAME = 'trading_strategies'
			AND COLUMN_NAME IN ('enable_margin_loss_stop_loss', 'margin_loss_stop_loss_percent');
		`

		rows, err := db.Query(checkSQL)
		if err != nil {
			log.Fatalf("检查字段失败: %v", err)
		}
		defer rows.Close()

		var existingColumns []string
		for rows.Next() {
			var columnName string
			rows.Scan(&columnName)
			existingColumns = append(existingColumns, columnName)
		}

		if len(existingColumns) > 0 {
			fmt.Printf("ℹ️ 字段已存在: %v\n", existingColumns)
			fmt.Println("✅ 数据库结构已是最新状态")
		} else {
			log.Fatalf("字段不存在且迁移失败")
		}
	} else {
		fmt.Println("✅ 数据库迁移成功!")
		fmt.Println("   添加了字段: enable_margin_loss_stop_loss, margin_loss_stop_loss_percent")
	}

	fmt.Println("🎉 迁移完成!")
}
