package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 检查market_klines表结构
	fmt.Println("检查market_klines表结构:")
	rows, err := db.Raw("DESCRIBE market_klines").Rows()
	if err != nil {
		log.Fatalf("查询表结构失败: %v", err)
	}
	defer rows.Close()

	fmt.Printf("%-20s %-15s %-10s %-10s %s\n", "Field", "Type", "Null", "Key", "Default")
	fmt.Println("--------------------------------------------------------------------------------")
	for rows.Next() {
		var field, fieldType, null, key, defaultValue string
		rows.Scan(&field, &fieldType, &null, &key, &defaultValue, nil)
		fmt.Printf("%-20s %-15s %-10s %-10s %s\n", field, fieldType, null, key, defaultValue)
	}

	// 检查是否有数据
	var count int64
	db.Raw("SELECT COUNT(*) FROM market_klines").Scan(&count)
	fmt.Printf("\nmarket_klines表总记录数: %d\n", count)

	// 如果有数据，显示一条记录
	if count > 0 {
		var record map[string]interface{}
		db.Raw("SELECT * FROM market_klines LIMIT 1").Scan(&record)
		fmt.Printf("\n示例记录: %+v\n", record)
	}

	// 检查binance_24h_stats表结构
	fmt.Println("\n检查binance_24h_stats表结构:")
	rows2, err := db.Raw("DESCRIBE binance_24h_stats").Rows()
	if err != nil {
		log.Fatalf("查询binance_24h_stats表结构失败: %v", err)
	}
	defer rows2.Close()

	fmt.Printf("%-20s %-15s %-10s %-10s %s\n", "Field", "Type", "Null", "Key", "Default")
	fmt.Println("--------------------------------------------------------------------------------")
	for rows2.Next() {
		var field, fieldType, null, key, defaultValue string
		rows2.Scan(&field, &fieldType, &null, &key, &defaultValue, nil)
		fmt.Printf("%-20s %-15s %-10s %-10s %s\n", field, fieldType, null, key, defaultValue)
	}

	// 检查binance_24h_stats是否有数据
	var count2 int64
	db.Raw("SELECT COUNT(*) FROM binance_24h_stats").Scan(&count2)
	fmt.Printf("\nbinance_24h_stats表总记录数: %d\n", count2)

	// 如果有数据，显示一条记录
	if count2 > 0 {
		var record map[string]interface{}
		db.Raw("SELECT * FROM binance_24h_stats LIMIT 1").Scan(&record)
		fmt.Printf("\n示例记录: %+v\n", record)
	}
}