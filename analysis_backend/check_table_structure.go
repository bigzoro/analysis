package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("=== Market Klines 表结构检查 ===")

	// 获取表结构
	rows, err := db.Raw("DESCRIBE market_klines").Rows()
	if err != nil {
		log.Fatal("Failed to get table structure:", err)
	}
	defer rows.Close()

	fmt.Println("字段结构:")
	fmt.Printf("%-20s %-15s %-5s %-10s %-10s %s\n", "Field", "Type", "Null", "Key", "Default", "Extra")
	fmt.Println("--------------------------------------------------------------------------------")

	for rows.Next() {
		var field, type_, null, key, default_, extra string
		err := rows.Scan(&field, &type_, &null, &key, &default_, &extra)
		if err != nil {
			log.Printf("Scan error: %v", err)
			continue
		}
		fmt.Printf("%-20s %-15s %-5s %-10s %-10s %s\n", field, type_, null, key, default_, extra)
	}

	// 检查一些示例数据
	fmt.Println("\n=== 示例数据检查 ===")
	var sampleData []map[string]interface{}
	err = db.Table("market_klines").
		Select("symbol, kind, `interval`, open_time").
		Limit(3).
		Find(&sampleData).Error

	if err != nil {
		log.Printf("查询示例数据失败: %v", err)
	} else {
		fmt.Println("前3条记录的时间戳值:")
		for i, data := range sampleData {
			fmt.Printf("记录 %d: symbol=%s, open_time=%v\n",
				i+1, data["symbol"], data["open_time"])
		}
	}

	// 检查时间戳范围
	fmt.Println("\n=== 时间戳范围检查 ===")
	var timeStats struct {
		MinOpenTime interface{}
		MaxOpenTime interface{}
	}

	db.Table("market_klines").
		Select("MIN(open_time) as min_open_time, MAX(open_time) as max_open_time").
		Row().Scan(&timeStats.MinOpenTime, &timeStats.MaxOpenTime)

	fmt.Printf("open_time 范围: %v ~ %v\n", timeStats.MinOpenTime, timeStats.MaxOpenTime)

	// 转换为可读时间（假设是毫秒时间戳）
	if minOpen, ok := timeStats.MinOpenTime.(int64); ok {
		fmt.Printf("最小open_time可读时间: %v (秒)\n", minOpen/1000)
		fmt.Printf("最小open_time可读时间: %v (毫秒)\n", minOpen)
	}
	if maxOpen, ok := timeStats.MaxOpenTime.(int64); ok {
		fmt.Printf("最大open_time可读时间: %v (秒)\n", maxOpen/1000)
		fmt.Printf("最大open_time可读时间: %v (毫秒)\n", maxOpen)
	}

	// 转换为可读时间
	if minOpen, ok := timeStats.MinOpenTime.(int64); ok {
		fmt.Printf("最小open_time可读时间: %v\n", minOpen/1000)
	}
	if maxOpen, ok := timeStats.MaxOpenTime.(int64); ok {
		fmt.Printf("最大open_time可读时间: %v\n", maxOpen/1000)
	}
}