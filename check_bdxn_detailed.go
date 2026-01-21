package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== BDXNUSDT 详细数据分析 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 检查exchange_info中的重复记录
	fmt.Println("🔍 检查 exchange_info 中的 BDXNUSDT 记录:")
	var exchangeInfos []struct {
		ID            uint   `json:"id"`
		Symbol        string `json:"symbol"`
		Status        string `json:"status"`
		MarketType    string `json:"market_type"`
		IsActive      bool   `json:"is_active"`
		CreatedAt     string `json:"created_at"`
		UpdatedAt     string `json:"updated_at"`
	}

	db.Raw("SELECT id, symbol, status, market_type, is_active, created_at, updated_at FROM binance_exchange_info WHERE symbol = ?", "BDXNUSDT").
		Scan(&exchangeInfos)

	for i, info := range exchangeInfos {
		fmt.Printf("  记录 %d: ID=%d, 状态=%s, 市场=%s, 活跃=%v, 创建=%s, 更新=%s\n",
			i+1, info.ID, info.Status, info.MarketType, info.IsActive, info.CreatedAt, info.UpdatedAt)
	}

	// 检查K线数据
	fmt.Println("\n🔍 检查 market_klines 数据:")
	var klineCount int64
	db.Raw("SELECT COUNT(*) FROM market_klines WHERE symbol = ?", "BDXNUSDT").Scan(&klineCount)
	fmt.Printf("  market_klines 总记录数: %d\n", klineCount)

	// 按时间间隔和市场类型统计
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
			fmt.Printf("    %s %s: %d 条\n", stat.Kind, stat.Interval, stat.Count)
		}
	}

	// 检查最近的K线数据
	fmt.Println("\n🔍 检查最近的K线数据:")
	var recentKlines []struct {
		Kind       string
		Interval   string
		OpenTime   string
		ClosePrice string
		Volume     string
	}
	db.Raw(`
		SELECT kind, ` + "`interval`" + `, open_time, close_price, volume
		FROM market_klines
		WHERE symbol = ?
		ORDER BY open_time DESC
		LIMIT 5
	`, "BDXNUSDT").Scan(&recentKlines)

	if len(recentKlines) > 0 {
		for _, kline := range recentKlines {
			fmt.Printf("    %s %s: 时间=%s, 收盘价=%s, 成交量=%s\n",
				kline.Kind, kline.Interval, kline.OpenTime, kline.ClosePrice, kline.Volume)
		}
	} else {
		fmt.Println("    无K线数据")
	}

	// 检查资金费率数据的时间范围
	fmt.Println("\n🔍 检查资金费率数据:")
	var fundingStats struct {
		Total    int64
		Earliest string
		Latest   string
	}
	db.Raw("SELECT COUNT(*) as total, MIN(funding_time) as earliest, MAX(funding_time) as latest FROM binance_funding_rates WHERE symbol = ?", "BDXNUSDT").Scan(&fundingStats)
	fmt.Printf("  资金费率记录数: %d\n", fundingStats.Total)
	if fundingStats.Total > 0 {
		fmt.Printf("  时间范围: %s 到 %s\n", fundingStats.Earliest, fundingStats.Latest)
	}

	fmt.Println("\n=== 详细分析完成 ===")
}