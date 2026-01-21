package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"
)

func main() {
	cfg, err := config.LoadConfig("config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	db, err := pdb.OpenMySQL(pdb.Options{
		DSN: cfg.Database.DSN,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	var count int64
	err = gdb.Model(&pdb.BinanceExchangeInfo{}).Where("quote_asset = ? AND status = ?", "USDT", "TRADING").Count(&count).Error
	if err != nil {
		log.Fatalf("Query failed: %v", err)
	}

	fmt.Printf("活跃的USDT交易对数量: %d\n", count)

	// 检查期货合约数量
	var futuresCount int64
	err = gdb.Model(&pdb.BinanceFuturesContract{}).Where("status = ?", "TRADING").Count(&futuresCount).Error
	if err != nil {
		log.Printf("Query futures failed: %v", err)
	} else {
		fmt.Printf("活跃的期货合约数量: %d\n", futuresCount)
	}
}
