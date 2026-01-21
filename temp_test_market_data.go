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

	fmt.Println("=== 测试市场数据查询 ===")

	// 测试几个币种
	symbols := []string{"BTCUSDT", "ETHUSDT", "ADAUSDT", "SOLUSDT", "DOGEUSDT"}

	for _, symbol := range symbols {
		fmt.Printf("\n--- %s ---\n", symbol)

		// 检查现货
		var spotCount int64
		err = gdb.GormDB().Table("binance_exchange_info").
			Where("symbol = ? AND status = ? AND quote_asset = ?", symbol, "TRADING", "USDT").
			Count(&spotCount).Error
		if err != nil {
			fmt.Printf("现货查询失败: %v\n", err)
		} else {
			fmt.Printf("现货记录数: %d\n", spotCount)
		}

		// 检查期货
		var futuresCount int64
		err = gdb.GormDB().Table("binance_market_tops").
			Joins("JOIN binance_market_snapshots ON binance_market_tops.snapshot_id = binance_market_snapshots.id").
			Where("binance_market_snapshots.kind = ? AND binance_market_tops.symbol = ?", "futures", symbol).
			Count(&futuresCount).Error
		if err != nil {
			fmt.Printf("期货查询失败: %v\n", err)
		} else {
			fmt.Printf("期货记录数: %d\n", futuresCount)
		}

		fmt.Printf("现货: %v, 期货: %v\n", spotCount > 0, futuresCount > 0)
	}
}
