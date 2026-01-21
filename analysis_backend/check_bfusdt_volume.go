package main

import (
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"
)

func main() {
	// 初始化数据库连接
	db, err := pdb.GetDB()
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}

	var stats []struct {
		Symbol      string
		QuoteVolume float64
	}

	// 查询最近24小时交易量最大的币种
	err = db.Table("binance_24h_stats").
		Select("symbol, AVG(quote_volume) as quote_volume").
		Where("market_type = ? AND created_at >= ?", "spot", time.Now().Add(-24*time.Hour)).
		Group("symbol").
		Having("AVG(quote_volume) > 1000000").
		Order("AVG(quote_volume) DESC").
		Limit(55). // 多取一些，看看BFUSDUSDT的排名
		Scan(&stats).Error

	if err != nil {
		log.Fatal("查询失败:", err)
	}

	fmt.Println("=== 交易量最大的币种排名 ===")
	for i, stat := range stats {
		fmt.Printf("%d. %s: %.0f USD\n", i+1, stat.Symbol, stat.QuoteVolume)
		if stat.Symbol == "BFUSDUSDT" {
			fmt.Printf("🎯 BFUSDUSDT 排名: #%d\n", i+1)
		}
	}

	// 单独查询BFUSDUSDT
	var bfusdtStats struct {
		Symbol      string
		QuoteVolume float64
		Count       int64
	}

	err = db.Table("binance_24h_stats").
		Select("symbol, AVG(quote_volume) as quote_volume, COUNT(*) as count").
		Where("symbol = ? AND market_type = ? AND created_at >= ?", "BFUSDUSDT", "spot", time.Now().Add(-24*time.Hour)).
		Group("symbol").
		Scan(&bfusdtStats).Error

	if err != nil {
		fmt.Printf("查询BFUSDUSDT失败: %v\n", err)
	} else {
		fmt.Printf("\n=== BFUSDUSDT详情 ===\n")
		fmt.Printf("Symbol: %s\n", bfusdtStats.Symbol)
		fmt.Printf("Avg Quote Volume: %.0f USD\n", bfusdtStats.QuoteVolume)
		fmt.Printf("Records Count: %d\n", bfusdtStats.Count)
		if bfusdtStats.QuoteVolume > 1000000 {
			fmt.Printf("✅ 符合VolumeBasedSelector条件 (>100万美元)\n")
		} else {
			fmt.Printf("❌ 不符合VolumeBasedSelector条件 (<=100万美元)\n")
		}
	}
}
