package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

type ExchangeInfo struct {
	Symbol    string
	Filters   string
	Status    string
	UpdatedAt string
}

func main() {
	fmt.Println("=== 检查数据库中的 exchange_info 数据 ===")

	// 连接数据库
	db, err := sql.Open("sqlite3", "analysis_backend/analysis.db")
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer db.Close()

	// 检查表是否存在
	var tableExists int
	err = db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='binance_exchange_infos'").Scan(&tableExists)
	if err != nil {
		log.Fatalf("查询表失败: %v", err)
	}

	if tableExists == 0 {
		fmt.Println("❌ binance_exchange_infos 表不存在")
		return
	}

	fmt.Println("✅ binance_exchange_infos 表存在")

	// 查询记录数量
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM binance_exchange_infos").Scan(&count)
	if err != nil {
		log.Fatalf("查询记录数量失败: %v", err)
	}

	fmt.Printf("📊 表中有 %d 条记录\n", count)

	// 查询一些示例数据
	rows, err := db.Query("SELECT symbol, filters, status, updated_at FROM binance_exchange_infos LIMIT 5")
	if err != nil {
		log.Fatalf("查询数据失败: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n=== 示例数据 ===")
	for rows.Next() {
		var info ExchangeInfo
		err := rows.Scan(&info.Symbol, &info.Filters, &info.Status, &info.UpdatedAt)
		if err != nil {
			log.Printf("扫描数据失败: %v", err)
			continue
		}

		fmt.Printf("交易对: %s\n", info.Symbol)
		fmt.Printf("状态: %s\n", info.Status)
		fmt.Printf("过滤器长度: %d 字符\n", len(info.Filters))
		if len(info.Filters) > 0 && len(info.Filters) < 500 {
			fmt.Printf("过滤器内容: %s\n", info.Filters[:min(200, len(info.Filters))])
		}
		fmt.Printf("更新时间: %s\n", info.UpdatedAt)
		fmt.Println("---")
	}

	// 特别检查一些常见的交易对
	symbols := []string{"BTCUSDT", "ETHUSDT", "FILUSDT", "FHEUSDT"}
	for _, symbol := range symbols {
		var info ExchangeInfo
		err := db.QueryRow("SELECT symbol, filters, status, updated_at FROM binance_exchange_infos WHERE symbol = ?", symbol).Scan(&info.Symbol, &info.Filters, &info.Status, &info.UpdatedAt)
		if err != nil {
			fmt.Printf("❌ 查询 %s 失败: %v\n", symbol, err)
		} else {
			fmt.Printf("✅ %s 存在，过滤器长度: %d 字符\n", symbol, len(info.Filters))
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
