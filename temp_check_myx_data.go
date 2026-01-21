package main

import (
	"fmt"
	"log"

	pdb "analysis/analysis_backend/internal/db"
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
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer gdb.Close()

	fmt.Println("=== 检查MYXUSDT的详细数据 ===")

	// 检查涨幅排名
	var rankResult struct {
		Symbol    string
		ChangePct float64
		Volume    float64
		Ranking   int
	}
	query := `
		SELECT symbol, price_change_percent, volume,
			   ROW_NUMBER() OVER (ORDER BY price_change_percent DESC, volume DESC) as ranking
		FROM binance_24h_stats
		WHERE market_type = 'futures' AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
		AND symbol = 'MYXUSDT'
	`
	err = gdb.GormDB().Raw(query).Scan(&rankResult).Error
	if err != nil {
		fmt.Printf("查询排名失败: %v\n", err)
	} else {
		fmt.Printf("涨幅排名: %d, 涨幅: %.2f%%, 交易量: %.0f\n",
			rankResult.Ranking, rankResult.ChangePct, rankResult.Volume)
	}

	// 检查市值数据
	var marketData struct {
		Symbol    string
		MarketCap string
	}
	err = gdb.GormDB().Table("coincap_market_data").
		Select("symbol, market_cap_usd").
		Where("symbol = ?", "MYX").
		Scan(&marketData).Error
	if err != nil {
		fmt.Printf("查询市值失败: %v\n", err)
	} else {
		fmt.Printf("市值数据: %s = %s\n", marketData.Symbol, marketData.MarketCap)
	}

	// 检查是否有现货
	var spotCount int64
	gdb.GormDB().Table("binance_exchange_info").
		Where("symbol = ? AND status = ? AND quote_asset = ?", "MYXUSDT", "TRADING", "USDT").
		Count(&spotCount)
	fmt.Printf("现货记录数: %d\n", spotCount)

	// 检查是否有期货
	var futuresCount int64
	gdb.GormDB().Table("binance_market_tops").
		Joins("JOIN binance_market_snapshots ON binance_market_tops.snapshot_id = binance_market_snapshots.id").
		Where("binance_market_snapshots.kind = ? AND binance_market_tops.symbol = ?", "futures", "MYXUSDT").
		Count(&futuresCount)
	fmt.Printf("期货记录数: %d\n", futuresCount)

	// 分析结果
	fmt.Println("\n=== 分析结果 ===")
	if rankResult.Ranking > 0 && rankResult.Ranking <= 7 {
		fmt.Printf("✅ 涨幅排名%d，满足前7名条件\n", rankResult.Ranking)
	} else {
		fmt.Printf("❌ 涨幅排名%d，不满足前7名条件\n", rankResult.Ranking)
	}

	if spotCount > 0 {
		fmt.Println("✅ 有现货交易")
	} else {
		fmt.Println("❌ 没有现货交易")
	}

	if futuresCount > 0 {
		fmt.Println("✅ 有期货合约")
	} else {
		fmt.Println("❌ 没有期货合约")
	}
}