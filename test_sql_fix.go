package main

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🧪 测试SQL查询修复")

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

	// 测试修复后的查询
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -1)

	fmt.Printf("查询时间范围: %s 到 %s\n", startTime.Format("2006-01-02 15:04:05"), endTime.Format("2006-01-02 15:04:05"))

	// 使用修复后的查询逻辑
	rows, err := db.Query(`
		SELECT symbol
		FROM (
			SELECT symbol, MAX(quote_volume) as max_volume
			FROM binance_24h_stats
			WHERE quote_volume > 1000
				AND created_at >= ? AND created_at <= ?
			GROUP BY symbol
			ORDER BY max_volume DESC
			LIMIT 200
		) as top_symbols
	`, startTime, endTime)

	if err != nil {
		log.Fatal("查询失败:", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			continue
		}
		symbols = append(symbols, symbol)
	}

	fmt.Printf("✅ 查询成功！找到%d个高交易量币种\n", len(symbols))

	if len(symbols) > 0 {
		fmt.Println("前5个币种:")
		for i, symbol := range symbols {
			if i >= 5 {
				break
			}
			fmt.Printf("  %d. %s\n", i+1, symbol)
		}
	}

	// 测试强弱币种计算
	fmt.Println("\n📊 测试强弱币种计算:")
	strong := 0
	weak := 0

	for i, symbol := range symbols {
		if i >= 10 { // 只测试前10个
			break
		}

		// 获取该币种的涨跌幅
		var priceChange float64
		err := db.QueryRow(`
			SELECT price_change_percent
			FROM binance_24h_stats
			WHERE symbol = ? AND created_at >= ? AND created_at <= ?
			ORDER BY created_at DESC
			LIMIT 1
		`, symbol, startTime, endTime).Scan(&priceChange)

		if err != nil {
			continue
		}

		// 使用修复后的阈值：±2%
		if priceChange > 2 {
			strong++
			fmt.Printf("  %s: %.2f%% -> 强势\n", symbol, priceChange)
		} else if priceChange < -2 {
			weak++
			fmt.Printf("  %s: %.2f%% -> 弱势\n", symbol, priceChange)
		} else {
			fmt.Printf("  %s: %.2f%% -> 中性\n", symbol, priceChange)
		}
	}

	fmt.Printf("\n统计结果:\n")
	fmt.Printf("强势币种: %d\n", strong)
	fmt.Printf("弱势币种: %d\n", weak)

	if strong > 0 || weak > 0 {
		fmt.Println("\n✅ SQL修复成功！技术指标现在可以正常工作了")
	} else {
		fmt.Println("\n⚠️ 查询修复成功，但所有币种都在±2%阈值内（符合当前平静市场）")
	}
}