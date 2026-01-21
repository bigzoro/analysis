package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 添加 updated_at 字段到数据表 ===")

	// 从环境变量获取数据库连接信息
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	}

	fmt.Printf("连接数据库: %s\n", dsn)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 添加字段到各个表
	tables := []struct {
		table string
		sql   string
	}{
		{
			table: "binance_funding_rates",
			sql:   "ALTER TABLE binance_funding_rates ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at",
		},
		{
			table: "binance_24h_stats",
			sql:   "ALTER TABLE binance_24h_stats ADD COLUMN updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP AFTER created_at",
		},
	}

	for _, table := range tables {
		fmt.Printf("\n--- 处理表: %s ---\n", table.table)

		// 检查字段是否已存在
		var count int
		checkSQL := fmt.Sprintf(`
			SELECT COUNT(*) FROM information_schema.COLUMNS
			WHERE TABLE_SCHEMA = DATABASE()
			AND TABLE_NAME = '%s'
			AND COLUMN_NAME = 'updated_at'
		`, table.table)

		err := db.Raw(checkSQL).Scan(&count).Error
		if err != nil {
			log.Printf("检查表 %s 字段失败: %v", table.table, err)
			continue
		}

		if count > 0 {
			fmt.Printf("✅ 表 %s 已存在 updated_at 字段，跳过\n", table.table)
			continue
		}

		// 添加字段
		fmt.Printf("添加 updated_at 字段到表 %s...\n", table.table)
		err = db.Exec(table.sql).Error
		if err != nil {
			log.Printf("❌ 添加字段到表 %s 失败: %v", table.table, err)
			// 尝试忽略已存在的错误
			if err.Error() != "Duplicate column name 'updated_at'" {
				continue
			}
		} else {
			fmt.Printf("✅ 成功为表 %s 添加 updated_at 字段\n", table.table)
		}

		// 为现有记录设置初始 updated_at 值
		updateSQL := fmt.Sprintf("UPDATE %s SET updated_at = created_at WHERE updated_at IS NULL", table.table)
		err = db.Exec(updateSQL).Error
		if err != nil {
			log.Printf("⚠️ 更新现有记录的 updated_at 失败: %v", err)
		} else {
			fmt.Printf("✅ 已为表 %s 的现有记录设置初始 updated_at 值\n", table.table)
		}
	}

	fmt.Println("\n=== 字段添加完成 ===")
	fmt.Println("现在数据更新时会正确维护 updated_at 字段，确保数据新鲜度！")
}