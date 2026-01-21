package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"

	pdb "analysis_backend/internal/db"
)

func main() {
	// 从环境变量或配置文件获取数据库连接信息
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "root:password@tcp(localhost:3306)/analysis?charset=utf8mb4&parseTime=True&loc=Local"
	}

	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 查询DASHUSDT的现货数据
	var spotStats pdb.Binance24hStats
	if err := db.Where("symbol = ? AND market_type = ?", "DASHUSDT", "spot").First(&spotStats).Error; err != nil {
		log.Printf("Error querying DASHUSDT spot data: %v", err)
	} else {
		fmt.Printf("DASHUSDT Spot Data:\n")
		fmt.Printf("  Price: %.8f USDT\n", spotStats.LastPrice)
		fmt.Printf("  Volume: %.4f\n", spotStats.Volume)
		fmt.Printf("  Quote Volume: %.4f USDT\n", spotStats.QuoteVolume)
		fmt.Printf("  24h Change: %.4f%%\n", spotStats.PriceChangePercent)
	}

	// 查询DASHUSDT的期货数据
	var futuresStats pdb.Binance24hStats
	if err := db.Where("symbol = ? AND market_type = ?", "DASHUSDT", "futures").First(&futuresStats).Error; err != nil {
		log.Printf("Error querying DASHUSDT futures data: %v", err)
	} else {
		fmt.Printf("\nDASHUSDT Futures Data:\n")
		fmt.Printf("  Price: %.8f USDT\n", futuresStats.LastPrice)
		fmt.Printf("  Volume: %.4f\n", futuresStats.Volume)
		fmt.Printf("  Quote Volume: %.4f USDT\n", futuresStats.QuoteVolume)
		fmt.Printf("  24h Change: %.4f%%\n", futuresStats.PriceChangePercent)
	}

	// 计算可能的保证金
	if futuresStats.LastPrice > 0 {
		// 假设一些常见的仓位大小和杠杆
		testCases := []struct {
			quantity float64
			leverage int
		}{
			{10, 5},  // 10个币，5倍杠杆
			{50, 5},  // 50个币，5倍杠杆
			{100, 5}, // 100个币，5倍杠杆
			{10, 10}, // 10个币，10倍杠杆
			{50, 10}, // 50个币，10倍杠杆
		}

		fmt.Printf("\nDASHUSDT Futures Margin Calculations:\n")
		for _, tc := range testCases {
			notionalValue := tc.quantity * futuresStats.LastPrice
			requiredMargin := notionalValue / float64(tc.leverage)
			fmt.Printf("  %.0f coins @ %dx leverage: Notional=%.2f USDT, Margin=%.2f USDT\n",
				tc.quantity, tc.leverage, notionalValue, requiredMargin)
		}
	}
}
