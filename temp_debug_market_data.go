package main

import (
	"fmt"
	"log"

	"analysis/analysis_backend/internal/config"
	pdb "analysis/analysis_backend/internal/db"
	"analysis/analysis_backend/internal/server"
)

func main() {
	// 连接数据库
	gdb, err := db.OpenMySQL(db.Options{
		DSN:          "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:  false,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}
	defer gdb.Close()

	// 创建配置
	cfg := &config.Config{}

	// 创建服务器实例
	srv := &server.Server{
		db:  gdb,
		cfg: cfg,
	}

	fmt.Println("=== 调试MYXUSDT市场数据获取 ===")

	// 直接调用getMarketDataForSymbol
	marketData := srv.getMarketDataForSymbol("MYXUSDT")

	fmt.Printf("Symbol: %s\n", marketData.Symbol)
	fmt.Printf("GainersRank: %d\n", marketData.GainersRank)
	fmt.Printf("MarketCap: %.2f\n", marketData.MarketCap)
	fmt.Printf("HasSpot: %v\n", marketData.HasSpot)
	fmt.Printf("HasFutures: %v\n", marketData.HasFutures)

	// 直接测试checkSpotTradingSafe
	spot, spotErr := srv.checkSpotTradingSafe("MYXUSDT")
	fmt.Printf("\nDirect spot check: %v, err: %v\n", spot, spotErr)

	// 直接测试checkFuturesTradingSafe
	futures, futuresErr := srv.checkFuturesTradingSafe("MYXUSDT")
	fmt.Printf("Direct futures check: %v, err: %v\n", futures, futuresErr)

	// 测试getTradingPairsConcurrent
	spot2, futures2 := srv.getTradingPairsConcurrent("MYXUSDT")
	fmt.Printf("Concurrent check - spot: %v, futures: %v\n", spot2, futures2)
}