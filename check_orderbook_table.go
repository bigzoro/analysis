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

	fmt.Printf("🔍 检查order book相关表结构\n")
	fmt.Printf("===============================\n")

	// 检查binance_order_book_depth表结构
	fmt.Printf("binance_order_book_depth表结构:\n")
	rows, err := db.Raw("DESCRIBE binance_order_book_depth").Rows()
	if err != nil {
		fmt.Printf("❌ 查询表结构失败: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var field, typ, null, key, def, extra string
			rows.Scan(&field, &typ, &null, &key, &def, &extra)
			fmt.Printf("• %s: %s\n", field, typ)
		}
	}

	// 检查最近的数据样例
	fmt.Printf("\n📊 binance_order_book_depth最新数据:\n")
	var result map[string]interface{}
	err = db.Raw("SELECT * FROM binance_order_book_depth WHERE symbol = 'BTTCUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&result).Error
	if err != nil {
		fmt.Printf("❌ 查询数据失败: %v\n", err)
	} else {
		fmt.Printf("BTTCUSDT最新记录:\n")
		for key, value := range result {
			fmt.Printf("• %s: %v\n", key, value)
		}
	}

	// 测试修复后的查询逻辑
	fmt.Printf("\n🧪 测试修复后的查询:\n")

	// 模拟我们需要的聚合查询
	query := `
		SELECT
			SUM(bids_0_quantity + bids_1_quantity + bids_2_quantity + bids_3_quantity + bids_4_quantity) as bid_volume,
			SUM(asks_0_quantity + asks_1_quantity + asks_2_quantity + asks_3_quantity + asks_4_quantity) as ask_volume,
			AVG(bids_0_price) as bid_price,
			AVG(asks_0_price) as ask_price
		FROM binance_order_book_depth
		WHERE symbol = 'BTTCUSDT'
		AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
	`

	var stats struct {
		BidVolume float64 `gorm:"column:bid_volume"`
		AskVolume float64 `gorm:"column:ask_volume"`
		BidPrice  float64 `gorm:"column:bid_price"`
		AskPrice  float64 `gorm:"column:ask_price"`
	}

	err = db.Raw(query).Scan(&stats).Error
	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询成功!\n")
		fmt.Printf("• Bid Volume: %.2f\n", stats.BidVolume)
		fmt.Printf("• Ask Volume: %.2f\n", stats.AskVolume)
		fmt.Printf("• Bid Price: %.4f\n", stats.BidPrice)
		fmt.Printf("• Ask Price: %.4f\n", stats.AskPrice)
	}

	fmt.Printf("\n🎯 修复方案:\n")
	fmt.Printf("1. 将order_book_snapshots替换为binance_order_book_depth\n")
	fmt.Printf("2. 调整查询字段和聚合逻辑\n")
	fmt.Printf("3. 使用bids_0_price, asks_0_price等字段\n")
}