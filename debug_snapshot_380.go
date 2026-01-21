package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	db, err := sql.Open("sqlite3", "./analysis.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	fmt.Println("=== 检查 snapshot_id = 380 的数据 ===")

	// 查看 snapshot 信息
	rows, err := db.Query(`
		SELECT id, bucket, fetched_at, kind
		FROM binance_market_snapshots
		WHERE id = 380
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	if !rows.Next() {
		fmt.Println("未找到 snapshot_id = 380")
		return
	}

	var id int
	var bucket, fetchedAt, kind string
	rows.Scan(&id, &bucket, &fetchedAt, &kind)
	fmt.Printf("Snapshot: ID=%d, Bucket=%s, Kind=%s\n", id, bucket, kind)

	// 查看涨幅最高的前5名（按pct_change降序）
	fmt.Println("\n=== 涨幅最高的前5名（按pct_change DESC）===")
	rows2, err := db.Query(`
		SELECT symbol, pct_change, rank, volume
		FROM binance_market_top
		WHERE snapshot_id = 380
		ORDER BY pct_change DESC
		LIMIT 5
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var symbol string
		var pctChange float64
		var rank int
		var volume string
		rows2.Scan(&symbol, &pctChange, &rank, &volume)
		fmt.Printf("涨幅第%d名: %s (%.2f%%, 成交量排名: %d, 成交量: %s)\n",
			rank, symbol, pctChange, rank, volume)
	}

	// 查看成交量最高的前5名（按rank ASC，即成交量降序）
	fmt.Println("\n=== 成交量最高的前5名（按rank ASC）===")
	rows3, err := db.Query(`
		SELECT symbol, pct_change, rank, volume
		FROM binance_market_top
		WHERE snapshot_id = 380
		ORDER BY rank ASC
		LIMIT 5
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows3.Close()

	for rows3.Next() {
		var symbol string
		var pctChange float64
		var rank int
		var volume string
		rows3.Scan(&symbol, &pctChange, &rank, &volume)
		fmt.Printf("成交量第%d名: %s (%.2f%%涨幅, 成交量: %s)\n",
			rank, symbol, pctChange, volume)
	}
}



