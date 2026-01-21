package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🐛 调试强弱币种计算")

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

	// 获取最近24小时的价格变化
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)

	fmt.Printf("时间范围: %s 到 %s\n", startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	// 首先获取高交易量的币种列表
	rows, err := db.Query(`
		SELECT symbol, price_change_percent, quote_volume, created_at
		FROM binance_24h_stats
		WHERE quote_volume > 10000
			AND created_at >= ? AND created_at <= ?
		ORDER BY quote_volume DESC
		LIMIT 10
	`, startTime, endTime)

	if err != nil {
		log.Fatal("查询高交易量币种失败:", err)
	}
	defer rows.Close()

	fmt.Println("\n📊 高交易量币种的涨跌幅:")
	count := 0
	strong := 0
	weak := 0

	for rows.Next() {
		var symbol string
		var priceChange float64
		var volume float64
		var createdAt time.Time

		if err := rows.Scan(&symbol, &priceChange, &volume, &createdAt); err != nil {
			continue
		}

		fmt.Printf("  %s: 涨跌幅%.2f%%, 交易量%.0f (%s)\n",
			symbol, priceChange, volume, createdAt.Format("15:04:05"))

		if priceChange > 5 {
			strong++
			fmt.Printf("    -> 强势币种\n")
		} else if priceChange < -5 {
			weak++
			fmt.Printf("    -> 弱势币种\n")
		} else {
			fmt.Printf("    -> 中性\n")
		}

		count++
	}

	fmt.Printf("\n统计结果:\n")
	fmt.Printf("总币种数: %d\n", count)
	fmt.Printf("强势币种: %d\n", strong)
	fmt.Printf("弱势币种: %d\n", weak)

	if count == 0 {
		fmt.Println("\n❌ 问题：没有找到高交易量币种数据")

		// 检查是否有任何binance_24h_stats数据
		var totalCount int
		err = db.QueryRow("SELECT COUNT(*) FROM binance_24h_stats WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)").Scan(&totalCount)
		fmt.Printf("最近24小时binance_24h_stats总记录数: %d\n", totalCount)

		if totalCount > 0 {
			// 检查交易量分布
			fmt.Println("\n📊 检查交易量分布:")
			rows2, err := db.Query(`
				SELECT MIN(quote_volume), MAX(quote_volume), AVG(quote_volume)
				FROM binance_24h_stats
				WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			`)
			if err == nil {
				var minVol, maxVol, avgVol float64
				if rows2.Next() {
					rows2.Scan(&minVol, &maxVol, &avgVol)
					fmt.Printf("交易量范围: %.0f - %.0f, 平均: %.0f\n", minVol, maxVol, avgVol)
				}
				rows2.Close()
			}
		}
	} else {
		fmt.Printf("\n✅ 数据查询正常，但涨跌幅都小于5%%阈值\n")
		fmt.Println("💡 建议：降低强弱币种判断阈值，或使用不同的判断标准")
	}

	// 检查是否有市场大波动
	fmt.Println("\n📈 检查是否有大涨跌的币种:")
	extremeRows, err := db.Query(`
		SELECT symbol, price_change_percent, quote_volume
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
		AND ABS(price_change_percent) > 2
		ORDER BY ABS(price_change_percent) DESC
		LIMIT 5
	`)

	if err == nil {
		fmt.Println("涨跌幅超过2%的币种:")
		extremeCount := 0
		for extremeRows.Next() {
			var symbol string
			var priceChange float64
			var volume float64
			extremeRows.Scan(&symbol, &priceChange, &volume)
			fmt.Printf("  %s: %.2f%% (交易量: %.0f)\n", symbol, priceChange, volume)
			extremeCount++
		}
		if extremeCount == 0 {
			fmt.Println("  无")
		}
		extremeRows.Close()
	}
}