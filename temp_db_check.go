package main

import (
	pdb "analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
		DSN:          "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:  false,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	fmt.Println("✅ 数据库连接成功")

	// 检查涨幅榜数据
	var count int64
	err = gdb.DB().Table("binance_24h_stats").Count(&count).Error
	if err != nil {
		fmt.Printf("❌ 检查binance_24h_stats表失败: %v\n", err)
	} else {
		fmt.Printf("📊 binance_24h_stats表有 %d 条记录\n", count)
	}

	// 检查涨幅榜快照
	err = gdb.DB().Table("realtime_gainers_items").Count(&count).Error
	if err != nil {
		fmt.Printf("❌ 检查realtime_gainers_items表失败: %v\n", err)
	} else {
		fmt.Printf("📈 realtime_gainers_items表有 %d 条记录\n", count)
	}

	// 检查CoinCap数据
	err = gdb.DB().Table("coin_cap_market_data").Count(&count).Error
	if err != nil {
		fmt.Printf("❌ 检查coin_cap_market_data表失败: %v\n", err)
	} else {
		fmt.Printf("💰 coin_cap_market_data表有 %d 条记录\n", count)
	}
}