package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	fmt.Printf("🔍 检查 binance_24h_stats 表结构\n")
	fmt.Printf("=====================================\n")

	// 检查表结构
	rows, err := db.Raw("DESCRIBE binance_24h_stats").Rows()
	if err != nil {
		log.Fatal("查询表结构失败:", err)
	}
	defer rows.Close()

	fmt.Printf("字段列表:\n")
	for rows.Next() {
		var field, typ, null, key, def, extra string
		rows.Scan(&field, &typ, &null, &key, &def, &extra)
		fmt.Printf("• %s: %s\n", field, typ)
	}

	fmt.Printf("\n🎯 检查问题字段 trades_count:\n")

	// 检查是否存在trades_count字段
	var count int64
	err = db.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = 'analysis' AND TABLE_NAME = 'binance_24h_stats' AND COLUMN_NAME = 'trades_count'").Scan(&count).Error
	if err != nil {
		log.Fatal("检查字段失败:", err)
	}

	if count > 0 {
		fmt.Printf("✅ trades_count 字段存在\n")
	} else {
		fmt.Printf("❌ trades_count 字段不存在\n")

		// 查找可能的替代字段
		fmt.Printf("\n🔍 查找可能的替代字段:\n")
		alternativeFields := []string{"trades", "count", "trade_count", "number_of_trades", "trades_count_24h"}

		for _, field := range alternativeFields {
			db.Raw("SELECT COUNT(*) FROM information_schema.COLUMNS WHERE TABLE_SCHEMA = 'analysis' AND TABLE_NAME = 'binance_24h_stats' AND COLUMN_NAME = ?", field).Scan(&count)
			if count > 0 {
				fmt.Printf("✅ 找到替代字段: %s\n", field)
			}
		}
	}

	// 检查最近的数据样例
	fmt.Printf("\n📊 检查最近数据样例:\n")
	var result map[string]interface{}
	err = db.Raw("SELECT * FROM binance_24h_stats WHERE symbol = 'BTCUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&result).Error
	if err != nil {
		fmt.Printf("查询数据失败: %v\n", err)
	} else {
		fmt.Printf("BTCUSDT最新记录字段:\n")
		for key, value := range result {
			fmt.Printf("• %s: %v\n", key, value)
		}
	}
}