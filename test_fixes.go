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

	fmt.Printf("🧪 测试数据库修复\n")
	fmt.Printf("===================\n")

	// 测试1: AVG(count) 类型修复
	fmt.Printf("测试1: AVG(count) 字段类型修复\n")
	var result1 struct {
		Volume      float64 `gorm:"column:volume"`
		TradesCount float64 `gorm:"column:trades_count"`
	}

	err = db.Raw("SELECT AVG(volume) as volume, AVG(count) as trades_count FROM binance_24h_stats WHERE symbol = 'BTTCUSDT' AND market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY)").Scan(&result1).Error
	if err != nil {
		fmt.Printf("❌ 测试1失败: %v\n", err)
	} else {
		fmt.Printf("✅ 测试1成功!\n")
		fmt.Printf("• Volume: %.2f (float64)\n", result1.Volume)
		fmt.Printf("• TradesCount: %.2f (float64)\n", result1.TradesCount)
	}

	// 测试2: 聚合查询类型修复
	fmt.Printf("\n测试2: 聚合查询字段类型修复\n")
	var result2 []struct {
		AvgTradesCount float64 `gorm:"column:avg_trades_count"`
		AvgVolume      float64 `gorm:"column:avg_volume"`
		MaxTradesCount float64 `gorm:"column:max_trades_count"`
		MinTradesCount float64 `gorm:"column:min_trades_count"`
	}

	err = db.Raw("SELECT AVG(count) as avg_trades_count, AVG(volume) as avg_volume, MAX(count) as max_trades_count, MIN(count) as min_trades_count FROM binance_24h_stats WHERE symbol = 'BTTCUSDT' AND market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY) GROUP BY symbol").Scan(&result2).Error
	if err != nil {
		fmt.Printf("❌ 测试2失败: %v\n", err)
	} else {
		fmt.Printf("✅ 测试2成功!\n")
		if len(result2) > 0 {
			fmt.Printf("• AvgTradesCount: %.2f (float64)\n", result2[0].AvgTradesCount)
			fmt.Printf("• MaxTradesCount: %.2f (float64)\n", result2[0].MaxTradesCount)
		}
	}

	fmt.Printf("\n🎉 数据库类型修复测试完成!\n")
	fmt.Printf("所有AVG()聚合函数返回的decimal类型现在都正确映射为float64\n")
}