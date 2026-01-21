package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== BDXNUSDT 市场状态分析 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 分析BDXNUSDT在不同市场中的状态
	fmt.Println("🔍 分析 BDXNUSDT 在不同市场的活跃状态:")

	var results []struct {
		MarketType string
		IsActive   bool
		Status     string
		Symbol     string
	}

	db.Raw(`
		SELECT market_type, is_active, status, symbol
		FROM binance_exchange_info
		WHERE symbol = ?
		ORDER BY market_type
	`, "BDXNUSDT").Scan(&results)

	for _, result := range results {
		status := "❌ 非活跃"
		if result.IsActive {
			status = "✅ 活跃"
		}
		fmt.Printf("  %s市场: %s (状态: %s)\n", result.MarketType, status, result.Status)
	}

	// 检查K线同步器会获取哪些交易对
	fmt.Println("\n🔍 检查K线同步器获取的交易对列表:")

	// 模拟GetUSDTTradingPairsByMarket查询
	var spotSymbols []string
	var futuresSymbols []string

	db.Raw(`
		SELECT symbol FROM binance_exchange_info
		WHERE quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?
		ORDER BY symbol
	`, "USDT", "TRADING", "spot", true).Scan(&spotSymbols)

	db.Raw(`
		SELECT symbol FROM binance_exchange_info
		WHERE quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?
		ORDER BY symbol
	`, "USDT", "TRADING", "futures", true).Scan(&futuresSymbols)

	fmt.Printf("  现货市场活跃交易对数量: %d\n", len(spotSymbols))
	fmt.Printf("  期货市场活跃交易对数量: %d\n", len(futuresSymbols))

	// 检查BDXNUSDT是否在这些列表中
	bdxnInSpot := false
	bdxnInFutures := false

	for _, symbol := range spotSymbols {
		if symbol == "BDXNUSDT" {
			bdxnInSpot = true
			break
		}
	}

	for _, symbol := range futuresSymbols {
		if symbol == "BDXNUSDT" {
			bdxnInFutures = true
			break
		}
	}

	fmt.Printf("  BDXNUSDT在现货活跃列表中: %v\n", bdxnInSpot)
	fmt.Printf("  BDXNUSDT在期货活跃列表中: %v\n", bdxnInFutures)

	// 检查K线数据分布
	fmt.Println("\n🔍 检查BDXNUSDT的K线数据分布:")

	var klineStats []struct {
		Kind     string
		Interval string
		Count    int64
	}

	db.Raw(`
		SELECT kind, ` + "`interval`" + `, COUNT(*) as count
		FROM market_klines
		WHERE symbol = ?
		GROUP BY kind, ` + "`interval`" + `
		ORDER BY kind, ` + "`interval`" + `
	`, "BDXNUSDT").Scan(&klineStats)

	if len(klineStats) > 0 {
		for _, stat := range klineStats {
			fmt.Printf("  %s %s: %d 条记录\n", stat.Kind, stat.Interval, stat.Count)
		}
	} else {
		fmt.Println("  无K线数据")
	}

	// 分析结论
	fmt.Println("\n💡 分析结论:")

	if bdxnInFutures && !bdxnInSpot {
		fmt.Println("  ✅ BDXNUSDT 正确地只在期货市场活跃")
		fmt.Println("  ✅ K线同步器会同步BDXNUSDT的期货K线数据")
		fmt.Println("  ℹ️  如果您不希望同步BDXNUSDT，请检查其活跃状态设置")
	} else if bdxnInSpot && !bdxnInFutures {
		fmt.Println("  ✅ BDXNUSDT 只在现货市场活跃")
		fmt.Println("  ✅ K线同步器会同步BDXNUSDT的现货K线数据")
	} else if bdxnInSpot && bdxnInFutures {
		fmt.Println("  ⚠️  BDXNUSDT 在两个市场都活跃")
		fmt.Println("  ⚠️  K线同步器会同时同步现货和期货K线数据")
	} else {
		fmt.Println("  ❌ BDXNUSDT 在两个市场都不活跃")
		fmt.Println("  ❌ K线同步器不会同步BDXNUSDT的数据")
	}

	fmt.Println("\n=== 分析完成 ===")
}