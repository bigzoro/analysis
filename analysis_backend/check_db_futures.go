package main

import (
	"fmt"
	"log"

	"analysis/internal/config"
	pdb "analysis/internal/db"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 读取配置
	cfg, err := config.Load("./config.yaml")
	if err != nil {
		log.Fatalf("加载配置失败: %v", err)
	}

	// 连接数据库
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		cfg.Database.User, cfg.Database.Password, cfg.Database.Host,
		cfg.Database.Port, cfg.Database.DBName)

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("=== 检查futures市场数据 ===")

	// 检查binance_exchange_info表中的futures数据
	var futuresExchangeCount int64
	db.Model(&pdb.BinanceExchangeInfo{}).Where("market_type = ?", "futures").Count(&futuresExchangeCount)
	fmt.Printf("binance_exchange_info表futures记录数: %d\n", futuresExchangeCount)

	// 检查USDT期货交易对
	futuresSymbols, err := pdb.GetUSDTTradingPairsByMarket(db, "futures")
	if err != nil {
		fmt.Printf("获取futures USDT交易对失败: %v\n", err)
	} else {
		fmt.Printf("futures USDT交易对数量: %d\n", len(futuresSymbols))
		if len(futuresSymbols) > 0 {
			fmt.Printf("前5个futures符号: %v\n", futuresSymbols[:min(5, len(futuresSymbols))])
		}
	}

	// 检查binance_24h_stats表中的futures数据
	var futuresStatsCount int64
	db.Model(&pdb.Binance24hStats{}).Where("market_type = ?", "futures").Count(&futuresStatsCount)
	fmt.Printf("binance_24h_stats表futures记录数: %d\n", futuresStatsCount)

	// 显示一些期货交易对样本
	if futuresStatsCount > 0 {
		var samples []pdb.Binance24hStats
		db.Model(&pdb.Binance24hStats{}).
			Where("market_type = ?", "futures").
			Order("volume DESC").
			Limit(5).
			Find(&samples)

		fmt.Println("期货交易对样本:")
		for _, s := range samples {
			fmt.Printf("  %s: 价格=%.4f, 涨幅=%.2f%%, 成交量=%.0f\n",
				s.Symbol, s.LastPrice, s.PriceChangePercent, s.Volume)
		}
	}

	// 检查realtime_gainers_snapshots表中的futures数据
	var futuresSnapshotCount int64
	db.Model(&pdb.RealtimeGainersSnapshot{}).Where("kind = ?", "futures").Count(&futuresSnapshotCount)
	fmt.Printf("realtime_gainers_snapshots表futures记录数: %d\n", futuresSnapshotCount)

	// 检查realtime_gainers_items表中的futures数据
	var futuresItemsCount int64
	db.Table("realtime_gainers_items").
		Joins("JOIN realtime_gainers_snapshots s ON realtime_gainers_items.snapshot_id = s.id").
		Where("s.kind = ?", "futures").
		Count(&futuresItemsCount)
	fmt.Printf("realtime_gainers_items表futures记录数: %d\n", futuresItemsCount)

	// 检查market_klines表中的futures数据
	var futuresKlinesCount int64
	db.Table("market_klines").Where("kind = ?", "futures").Count(&futuresKlinesCount)
	fmt.Printf("market_klines表futures记录数: %d\n", futuresKlinesCount)

	if futuresKlinesCount > 0 {
		// 检查最近的期货K线数据
		var recentKlines []map[string]interface{}
		db.Table("market_klines").
			Select("symbol, kind, `interval`, open_time, close_price").
			Where("kind = ?", "futures").
			Order("open_time DESC").
			Limit(3).
			Find(&recentKlines)

		fmt.Println("最近的期货K线数据样本:")
		for _, k := range recentKlines {
			fmt.Printf("  %s (%s) %s: open_time=%v, close_price=%v\n",
				k["symbol"], k["kind"], k["interval"], k["open_time"], k["close_price"])
		}
	}

	fmt.Println("=== 检查完成 ===")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
