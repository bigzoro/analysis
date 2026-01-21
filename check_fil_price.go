package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 检查FILUSDT当前价格和网格范围 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 查询最新的FILUSDT价格
	var priceResult map[string]interface{}
	priceQuery := `
		SELECT symbol, price, volume, timestamp
		FROM binance_24h_stats
		WHERE symbol = 'FILUSDT'
		ORDER BY timestamp DESC
		LIMIT 1
	`
	db.Raw(priceQuery).Scan(&priceResult)

	fmt.Printf("FILUSDT最新价格信息:\n")
	fmt.Printf("  交易对: %v\n", priceResult["symbol"])
	fmt.Printf("  价格: %v\n", priceResult["price"])
	fmt.Printf("  成交量: %v\n", priceResult["volume"])
	fmt.Printf("  时间戳: %v\n", priceResult["timestamp"])

	// 网格配置
	gridUpper := 1.4919874999999998
	gridLower := 1.1700125000000001

	fmt.Printf("\n网格配置:\n")
	fmt.Printf("  网格上限: %.8f\n", gridUpper)
	fmt.Printf("  网格下限: %.8f\n", gridLower)
	fmt.Printf("  网格范围: [%.4f, %.4f]\n", gridLower, gridUpper)

	if price, ok := priceResult["price"].(float64); ok {
		fmt.Printf("\n价格分析:\n")
		fmt.Printf("  当前价格: %.8f\n", price)

		if price >= gridLower && price <= gridUpper {
			fmt.Printf("  ✅ 价格在网格范围内\n")
			gridLevel := int((price - gridLower) / ((gridUpper - gridLower) / 20))
			fmt.Printf("  当前网格层级: %d/20\n", gridLevel)
		} else if price > gridUpper {
			fmt.Printf("  ❌ 价格高于网格上限 (%.4f > %.4f)\n", price, gridUpper)
		} else {
			fmt.Printf("  ❌ 价格低于网格下限 (%.4f < %.4f)\n", price, gridLower)
		}
	} else {
		fmt.Printf("  无法获取价格数据\n")
	}

	// 检查技术指标数据
	fmt.Println("\n=== 检查技术指标数据 ===")
	var techResult map[string]interface{}
	techQuery := `
		SELECT symbol, rsi, macd, histogram, ma5, ma20, bb_upper, bb_lower, bb_width, volatility, trend
		FROM technical_indicators
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 1
	`
	db.Raw(techQuery).Scan(&techResult)

	fmt.Printf("FILUSDT技术指标:\n")
	if len(techResult) > 0 {
		for k, v := range techResult {
			fmt.Printf("  %s: %v\n", k, v)
		}
	} else {
		fmt.Printf("  没有找到技术指标数据\n")
	}
}
