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
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("=== 涨幅榜重复数据诊断（修改后） ===")

	// 模拟gainers_history_syncer的数据生成查询
	fmt.Println("\n=== 模拟数据生成查询 ===")
	query := `
		SELECT
			symbol,
			price_change_percent,
			volume,
			quote_volume,
			last_price,
			ROW_NUMBER() OVER (ORDER BY price_change_percent DESC, volume DESC) as ranking
		FROM binance_24h_stats
		WHERE (symbol, created_at) IN (
			SELECT symbol, MAX(created_at) as latest_time
			FROM binance_24h_stats
			WHERE market_type = 'spot' AND created_at >= DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 HOUR)
				AND volume > 0 AND last_price > 0
			GROUP BY symbol
		)
		AND market_type = 'spot'
		ORDER BY price_change_percent DESC, volume DESC
		LIMIT 10
	`

	var results []struct {
		Symbol             string  `json:"symbol"`
		PriceChangePercent float64 `json:"price_change_percent"`
		Volume             float64 `json:"volume"`
		QuoteVolume        float64 `json:"quote_volume"`
		LastPrice          float64 `json:"last_price"`
		Ranking            int     `json:"rank"`
	}

	err = db.Raw(query).Scan(&results).Error
	if err != nil {
		log.Fatal("数据生成查询失败:", err)
	}

	fmt.Printf("数据生成查询返回 %d 条记录:\n", len(results))
	seen := make(map[string]bool)
	duplicates := 0
	for i, r := range results {
		fmt.Printf("  %d. %s (涨幅:%.2f%%, 交易量:%.2f)\n",
			i+1, r.Symbol, r.PriceChangePercent, r.Volume)

		if seen[r.Symbol] {
			fmt.Printf("❌ 发现重复币种: %s\n", r.Symbol)
			duplicates++
		}
		seen[r.Symbol] = true
	}

	if duplicates == 0 {
		fmt.Println("✅ 数据生成查询无重复")
	} else {
		fmt.Printf("❌ 数据生成查询发现 %d 个重复币种\n", duplicates)
	}

	// 模拟前端查询
	fmt.Println("\n=== 模拟前端查询（修改后） ===")
	frontendQuery := `
		SELECT
			i.symbol,
			i.rank,
			i.current_price,
			i.price_change24h
		FROM realtime_gainers_items i
		WHERE i.snapshot_id = (
			SELECT s.id
			FROM realtime_gainers_snapshots s
			WHERE s.kind = 'spot'
			ORDER BY s.timestamp DESC, s.id DESC
			LIMIT 1
		)
		ORDER BY i.rank ASC
		LIMIT 10
	`

	var frontendResults []struct {
		Symbol         string  `json:"symbol"`
		Rank           int     `json:"rank"`
		CurrentPrice   float64 `json:"current_price"`
		PriceChange24h float64 `json:"price_change24h"`
	}

	err = db.Raw(frontendQuery).Scan(&frontendResults).Error
	if err != nil {
		log.Fatal("前端查询失败:", err)
	}

	fmt.Printf("前端查询返回 %d 条记录:\n", len(frontendResults))
	frontendSeen := make(map[string]bool)
	frontendDuplicates := 0
	for i, r := range frontendResults {
		fmt.Printf("  %d. %s (排名:%d, 价格:%.4f, 涨幅:%.2f%%)\n",
			i+1, r.Symbol, r.Rank, r.CurrentPrice, r.PriceChange24h)

		if frontendSeen[r.Symbol] {
			fmt.Printf("❌ 前端查询发现重复币种: %s\n", r.Symbol)
			frontendDuplicates++
		}
		frontendSeen[r.Symbol] = true
	}

	if frontendDuplicates == 0 {
		fmt.Println("✅ 前端查询无重复")
	} else {
		fmt.Printf("❌ 前端查询发现 %d 个重复币种\n", frontendDuplicates)
	}
}