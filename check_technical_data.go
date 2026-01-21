package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🔍 检查技术指标数据")

	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatal("数据库连接测试失败:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	// 检查BTCUSDT数据
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM market_klines WHERE symbol = 'BTCUSDT'").Scan(&count)
	if err != nil {
		log.Fatal("查询BTCUSDT总记录数失败:", err)
	}
	fmt.Printf("BTCUSDT总记录数: %d\n", count)

	var count30 int
	err = db.QueryRow("SELECT COUNT(*) FROM market_klines WHERE symbol = 'BTCUSDT' AND open_time >= DATE_SUB(NOW(), INTERVAL 30 DAY)").Scan(&count30)
	if err != nil {
		log.Fatal("查询BTCUSDT最近30天记录数失败:", err)
	}
	fmt.Printf("BTCUSDT最近30天记录数: %d\n", count30)

	// 检查binance_24h_stats数据
	var statsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM binance_24h_stats").Scan(&statsCount)
	if err != nil {
		log.Fatal("查询binance_24h_stats记录数失败:", err)
	}
	fmt.Printf("binance_24h_stats总记录数: %d\n", statsCount)

	// 检查最近24小时的数据
	var recentStatsCount int
	err = db.QueryRow("SELECT COUNT(*) FROM binance_24h_stats WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)").Scan(&recentStatsCount)
	if err != nil {
		log.Fatal("查询最近24小时binance_24h_stats记录数失败:", err)
	}
	fmt.Printf("最近24小时binance_24h_stats记录数: %d\n", recentStatsCount)

	// 检查是否有高交易量的币种
	var highVolumeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM binance_24h_stats WHERE quote_volume > 10000 AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)").Scan(&highVolumeCount)
	if err != nil {
		log.Fatal("查询高交易量币种失败:", err)
	}
	fmt.Printf("最近24小时高交易量币种数 (>10000): %d\n", highVolumeCount)

	// 检查最近的BTCUSDT数据
	fmt.Println("\n📊 最近的BTCUSDT数据:")
	rows, err := db.Query(`
		SELECT open_time, close_price
		FROM market_klines
		WHERE symbol = 'BTCUSDT'
		ORDER BY open_time DESC
		LIMIT 5
	`)
	if err != nil {
		log.Fatal("查询BTCUSDT最近数据失败:", err)
	}
	defer rows.Close()

	for rows.Next() {
		var openTime time.Time
		var closePrice float64
		if err := rows.Scan(&openTime, &closePrice); err != nil {
			continue
		}
		fmt.Printf("  %s: %.2f\n", openTime.Format("2006-01-02 15:04:05"), closePrice)
	}

	// 检查最近的binance_24h_stats数据
	fmt.Println("\n📊 最近的binance_24h_stats数据:")
	rows2, err := db.Query(`
		SELECT symbol, price_change_percent, quote_volume, created_at
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
		ORDER BY quote_volume DESC
		LIMIT 3
	`)
	if err != nil {
		log.Printf("查询binance_24h_stats最近数据失败: %v", err)
	} else {
		defer rows2.Close()
		count := 0
		for rows2.Next() {
			var symbol string
			var priceChange float64
			var volume float64
			var createdAt time.Time
			if err := rows2.Scan(&symbol, &priceChange, &volume, &createdAt); err != nil {
				continue
			}
			fmt.Printf("  %s: 涨跌幅%.2f%%, 交易量%.0f (%s)\n", symbol, priceChange, volume, createdAt.Format("15:04:05"))
			count++
		}
		if count == 0 {
			fmt.Println("  无数据")
		}
	}

	fmt.Println("\n🎯 问题诊断:")
	if count30 < 14 {
		fmt.Printf("❌ BTCUSDT最近30天数据不足 (%d < 14)，无法计算RSI\n", count30)
	} else {
		fmt.Printf("✅ BTCUSDT最近30天数据充足 (%d >= 14)\n", count30)
	}

	if recentStatsCount == 0 {
		fmt.Println("❌ 最近24小时无binance_24h_stats数据，无法计算强弱币种")
	} else {
		fmt.Printf("✅ 最近24小时有binance_24h_stats数据 (%d条)\n", recentStatsCount)
	}

	if highVolumeCount == 0 {
		fmt.Println("❌ 无高交易量币种数据")
	} else {
		fmt.Printf("✅ 有高交易量币种数据 (%d个)\n", highVolumeCount)
	}
}