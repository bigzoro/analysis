package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("🔍 查询可用于回测的币种:")
	fmt.Println("=====================================")

	// 查询有日线数据的币种及数据条数
	var results []struct {
		Symbol string
		Count  int64
	}

	query := `
		SELECT symbol, COUNT(*) as count
		FROM market_klines
		WHERE kind = 'spot' AND ` + "`interval`" + ` = '1d'
		GROUP BY symbol
		HAVING COUNT(*) >= 200
		ORDER BY COUNT(*) DESC, symbol ASC
		LIMIT 50
	`

	err = db.Raw(query).Scan(&results).Error
	if err != nil {
		log.Fatalf("查询失败: %v", err)
	}

	fmt.Printf("%-15s %-8s %-15s\n", "币种", "数据条数", "状态")
	fmt.Println("--------------------------------------------------")

	// 主流币种列表
	majorCoins := map[string]bool{
		"BTCUSDT": true, "ETHUSDT": true, "BNBUSDT": true, "ADAUSDT": true,
		"SOLUSDT": true, "DOTUSDT": true, "AVAXUSDT": true, "LINKUSDT": true,
		"LTCUSDT": true, "MATICUSDT": true, "XRPUSDT": true, "DOGEUSDT": true,
		"TRXUSDT": true, "ETCUSDT": true, "FILUSDT": true, "ICPUSDT": true,
		"VETUSDT": true, "THETAUSDT": true, "FTTUSDT": true, "ALGOUSDT": true,
		"ATOMUSDT": true, "CAKEUSDT": true, "SUSHIUSDT": true, "COMPUSDT": true,
		"AAVEUSDT": true, "CRVUSDT": true, "YFIUSDT": true, "BALUSDT": true,
		"IMXUSDT": true, "GRTUSDT": true,
	}

	selectedSymbols := []string{}

	for _, result := range results {
		status := "✅ 可测试"
		if result.Count < 300 {
			status = "⚠️ 数据较少"
		} else if result.Count >= 600 {
			status = "⭐ 数据丰富"
		}

		isMajor := ""
		if majorCoins[result.Symbol] {
			isMajor = " (主流)"
			selectedSymbols = append(selectedSymbols, result.Symbol)
		}

		fmt.Printf("%-15s %-8d %-15s%s\n", result.Symbol, result.Count, status, isMajor)
	}

	fmt.Printf("\n🎯 推荐测试币种 (%d个):\n", len(selectedSymbols))
	for i, symbol := range selectedSymbols {
		fmt.Printf("  %d. %s\n", i+1, symbol)
		if i >= 29 { // 只显示前30个
			fmt.Printf("  ... 还有%d个币种\n", len(selectedSymbols)-30)
			break
		}
	}

	// 检查24小时统计数据
	fmt.Println("\n📊 24小时统计数据检查:")
	statsQuery := `
		SELECT COUNT(DISTINCT symbol) as total_symbols,
			   COUNT(*) as total_records
		FROM binance_24h_stats
		WHERE market_type = 'spot'
	`

	var stats struct {
		TotalSymbols int64
		TotalRecords int64
	}

	err = db.Raw(statsQuery).Scan(&stats).Error
	if err == nil {
		fmt.Printf("   现货市场币种数: %d\n", stats.TotalSymbols)
		fmt.Printf("   统计记录总数: %d\n", stats.TotalRecords)
	}
}