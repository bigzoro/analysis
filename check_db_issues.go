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

	fmt.Printf("🔍 检查数据库问题\n")
	fmt.Printf("==================\n")

	// 检查第一个问题的查询
	fmt.Printf("问题1: AVG(count) 返回类型问题\n")
	fmt.Printf("查询: SELECT AVG(volume) as volume, AVG(count) as trades_count FROM binance_24h_stats WHERE symbol = 'BTTCUSDT' AND market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY)\n")

	// 测试原始查询
	var result1 struct {
		Volume      float64 `gorm:"column:volume"`
		TradesCount string  `gorm:"column:trades_count"` // 先当作字符串处理
	}

	err = db.Raw("SELECT AVG(volume) as volume, AVG(count) as trades_count FROM binance_24h_stats WHERE symbol = 'BTTCUSDT' AND market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY)").Scan(&result1).Error
	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询成功!\n")
		fmt.Printf("Volume: %.2f (类型: float64)\n", result1.Volume)
		fmt.Printf("TradesCount: %s (类型: string)\n", result1.TradesCount)
	}

	// 检查第二个问题
	fmt.Printf("\n问题2: order_book_snapshots表不存在\n")

	// 检查表是否存在
	var count int
	err = db.Raw("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = 'analysis' AND TABLE_NAME = 'order_book_snapshots'").Scan(&count).Error
	if err != nil {
		log.Fatal("检查表失败:", err)
	}

	if count > 0 {
		fmt.Printf("✅ order_book_snapshots表存在\n")
	} else {
		fmt.Printf("❌ order_book_snapshots表不存在\n")

		// 列出现有的表
		fmt.Printf("\n📋 数据库中的表列表:\n")
		var tables []string
		db.Raw("SELECT TABLE_NAME FROM information_schema.TABLES WHERE TABLE_SCHEMA = 'analysis'").Scan(&tables)
		for _, table := range tables {
			fmt.Printf("• %s\n", table)
		}
	}

	// 检查是否有替代的表
	fmt.Printf("\n🔍 查找可能的替代表:\n")
	alternativeTables := []string{"orderbook", "order_book", "depth", "market_depth"}
	for _, table := range alternativeTables {
		db.Raw("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = 'analysis' AND TABLE_NAME = ?", table).Scan(&count)
		if count > 0 {
			fmt.Printf("✅ 找到替代表: %s\n", table)
		}
	}

	fmt.Printf("\n🎯 修复建议:\n")
	fmt.Printf("1. AVG(count) 结果应该当作 float64 处理，而不是 int64\n")
	fmt.Printf("2. order_book_snapshots 表需要创建或使用替代方案\n")
}