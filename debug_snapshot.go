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

	// 查看 snapshot_id = 380 的详细信息
	fmt.Println("=== Snapshot 380 详情 ===")
	rows, err := db.Query(`
		SELECT id, bucket, fetched_at, kind
		FROM binance_market_snapshots
		WHERE id = 380
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows.Close()

	if rows.Next() {
		var id int
		var bucket, fetchedAt, kind string
		rows.Scan(&id, &bucket, &fetchedAt, &kind)
		fmt.Printf("ID: %d, Bucket: %s, FetchedAt: %s, Kind: %s\n", id, bucket, fetchedAt, kind)
	} else {
		fmt.Println("未找到 snapshot_id = 380")
		return
	}

	// 查看该 snapshot 下的所有币种数据，按涨幅排序
	fmt.Println("\n=== Snapshot 380 的涨幅榜（按涨幅降序）===")
	rows2, err := db.Query(`
		SELECT symbol, pct_change, rank
		FROM binance_market_top
		WHERE snapshot_id = 380
		ORDER BY pct_change DESC
		LIMIT 10
	`)
	if err != nil {
		log.Fatal(err)
	}
	defer rows2.Close()

	for rows2.Next() {
		var symbol string
		var pctChange float64
		var rank int
		rows2.Scan(&symbol, &pctChange, &rank)
		fmt.Printf("涨幅第%d: %s (%.2f%%)\n", rank, symbol, pctChange)
	}
}