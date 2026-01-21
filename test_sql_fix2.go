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

	fmt.Printf("🧪 测试修复后的SQL查询\n")
	fmt.Printf("==========================\n")

	// 测试修复后的查询 (对应错误报告中的查询)
	fmt.Printf("测试1: SELECT close_price FROM market_klines WHERE symbol = 'WLFIUSDT' AND kind = 'spot' AND `interval` = '1d' ORDER BY open_time DESC LIMIT 30\n")

	var prices []float64
	err = db.Table("market_klines").
		Select("close_price").
		Where("symbol = ? AND kind = ? AND `interval` = ?", "WLFIUSDT", "spot", "1d").
		Order("open_time DESC").
		Limit(30).
		Pluck("close_price", &prices).Error

	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询成功! 返回%d条记录\n", len(prices))
		if len(prices) > 0 {
			fmt.Printf("最新价格: %.2f\n", prices[0])
		}
	}

	// 测试第二个查询
	fmt.Printf("\n测试2: SELECT volume FROM market_klines WHERE symbol = 'WLFIUSDT' AND kind = 'spot' AND `interval` = '1d' ORDER BY open_time DESC LIMIT 7\n")

	var volumes []float64
	err = db.Table("market_klines").
		Select("volume").
		Where("symbol = ? AND kind = ? AND `interval` = ?", "WLFIUSDT", "spot", "1d").
		Order("open_time DESC").
		Limit(7).
		Pluck("volume", &volumes).Error

	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询成功! 返回%d条记录\n", len(volumes))
		if len(volumes) > 0 {
			fmt.Printf("最新成交量: %.2f\n", volumes[0])
		}
	}

	// 测试一个小时线查询
	fmt.Printf("\n测试3: 小时线查询\n")

	var hourlyPrices []float64
	err = db.Table("market_klines").
		Select("close_price").
		Where("symbol = ? AND kind = ? AND `interval` = ?", "BTCUSDT", "spot", "1h").
		Order("open_time DESC").
		Limit(24).
		Pluck("close_price", &hourlyPrices).Error

	if err != nil {
		fmt.Printf("❌ 小时线查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 小时线查询成功! 返回%d条记录\n", len(hourlyPrices))
	}

	fmt.Printf("\n🎉 SQL语法修复测试完成!\n")
}