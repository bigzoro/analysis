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

	fmt.Printf("🧪 测试修复后的数据库查询\n")
	fmt.Printf("==============================\n")

	// 测试修复后的查询
	query := `
		SELECT AVG(count) as avg_trades_count, AVG(volume) as avg_volume, MAX(count) as max_trades_count, MIN(count) as min_trades_count
		FROM binance_24h_stats
		WHERE symbol = 'DASHUSDT' AND market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY)
		GROUP BY symbol
	`

	var result struct {
		AvgTradesCount float64 `gorm:"column:avg_trades_count"`
		AvgVolume      float64 `gorm:"column:avg_volume"`
		MaxTradesCount int64   `gorm:"column:max_trades_count"`
		MinTradesCount int64   `gorm:"column:min_trades_count"`
	}

	err = db.Raw(query).Scan(&result).Error
	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询成功!\n")
		fmt.Printf("平均交易次数: %.0f\n", result.AvgTradesCount)
		fmt.Printf("平均成交量: %.2f\n", result.AvgVolume)
		fmt.Printf("最大交易次数: %d\n", result.MaxTradesCount)
		fmt.Printf("最小交易次数: %d\n", result.MinTradesCount)
	}

	// 测试另一个查询
	fmt.Printf("\n🔍 测试另一个查询:\n")
	query2 := `
		SELECT AVG(volume) as volume, AVG(count) as trades_count
		FROM binance_24h_stats
		WHERE symbol = 'BTCUSDT' AND market_type = 'spot'
		AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 DAY)
	`

	var result2 struct {
		Volume      float64 `gorm:"column:volume"`
		TradesCount float64 `gorm:"column:trades_count"`
	}

	err = db.Raw(query2).Scan(&result2).Error
	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询成功!\n")
		fmt.Printf("平均成交量: %.2f\n", result2.Volume)
		fmt.Printf("平均交易次数: %.0f\n", result2.TradesCount)
	}

	fmt.Printf("\n🎉 数据库字段修复测试完成!\n")
}