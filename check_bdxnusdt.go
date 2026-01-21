package main

import (
	"database/sql"
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== BDXNUSDT 交易对分析 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 查询BDXNUSDT的基本信息
	var symbol, status, marketType, baseAsset, quoteAsset string
	var isActive bool
	var deactivatedAt, lastSeenActive, createdAt, updatedAt sql.NullTime

	query := `
		SELECT symbol, status, market_type, base_asset, quote_asset,
			   is_active, deactivated_at, last_seen_active,
			   created_at, updated_at
		FROM binance_exchange_info
		WHERE symbol = ?
	`

	err = db.Raw(query, "BDXNUSDT").Row().Scan(
		&symbol, &status, &marketType, &baseAsset, &quoteAsset,
		&isActive, &deactivatedAt, &lastSeenActive, &createdAt, &updatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("❌ BDXNUSDT 不在数据库中")
			return
		}
		log.Fatalf("查询失败: %v", err)
	}

	fmt.Printf("📊 基本信息:\n")
	fmt.Printf("  交易对: %s\n", symbol)
	fmt.Printf("  状态: %s\n", status)
	fmt.Printf("  市场类型: %s\n", marketType)
	fmt.Printf("  基础资产: %s\n", baseAsset)
	fmt.Printf("  计价资产: %s\n", quoteAsset)
	fmt.Printf("  活跃状态: %v\n", isActive)

	if deactivatedAt.Valid {
		fmt.Printf("  下架时间: %v\n", deactivatedAt.Time.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("  下架时间: 未下架\n")
	}

	if lastSeenActive.Valid {
		fmt.Printf("  最后活跃时间: %v\n", lastSeenActive.Time.Format("2006-01-02 15:04:05"))
	} else {
		fmt.Printf("  最后活跃时间: 无记录\n")
	}

	fmt.Printf("  创建时间: %v\n", createdAt.Time.Format("2006-01-02 15:04:05"))
	fmt.Printf("  更新时间: %v\n", updatedAt.Time.Format("2006-01-02 15:04:05"))

	// 查询整体统计
	var total, active, inactive, spotActive, futuresActive int64

	db.Raw("SELECT COUNT(*) FROM binance_exchange_info").Scan(&total)
	db.Raw("SELECT COUNT(*) FROM binance_exchange_info WHERE is_active = 1").Scan(&active)
	db.Raw("SELECT COUNT(*) FROM binance_exchange_info WHERE is_active = 0").Scan(&inactive)
	db.Raw("SELECT COUNT(*) FROM binance_exchange_info WHERE market_type = 'spot' AND is_active = 1").Scan(&spotActive)
	db.Raw("SELECT COUNT(*) FROM binance_exchange_info WHERE market_type = 'futures' AND is_active = 1").Scan(&futuresActive)

	fmt.Printf("\n📈 整体统计:\n")
	fmt.Printf("  总交易对数: %d\n", total)
	fmt.Printf("  活跃交易对数: %d\n", active)
	fmt.Printf("  非活跃交易对数: %d\n", inactive)
	fmt.Printf("  现货活跃: %d\n", spotActive)
	fmt.Printf("  期货活跃: %d\n", futuresActive)

	// 检查是否在活跃交易对列表中
	var activeSymbols []string
	db.Raw("SELECT symbol FROM binance_exchange_info WHERE quote_asset = 'USDT' AND status = 'TRADING' AND is_active = 1 ORDER BY symbol").
		Scan(&activeSymbols)

	fmt.Printf("\n🎯 活跃状态检查:\n")
	isInActiveList := false
	for _, s := range activeSymbols {
		if s == "BDXNUSDT" {
			isInActiveList = true
			break
		}
	}

	if isInActiveList {
		fmt.Printf("  ✅ BDXNUSDT 在活跃交易对列表中\n")
	} else {
		fmt.Printf("  ❌ BDXNUSDT 不在活跃交易对列表中\n")
	}
	fmt.Printf("  当前活跃USDT交易对总数: %d\n", len(activeSymbols))

	// 检查BDXNUSDT在各个表中的记录数
	fmt.Printf("\n📊 BDXNUSDT 数据记录统计:\n")

	// exchange_info表
	var exchangeInfoCount int64
	db.Raw("SELECT COUNT(*) FROM binance_exchange_info WHERE symbol = ?", "BDXNUSDT").Scan(&exchangeInfoCount)
	fmt.Printf("  binance_exchange_info: %d 条记录\n", exchangeInfoCount)

	// market_klines表 - 按时间间隔统计
	var klineStats []struct {
		Interval string
		Kind     string
		Count    int64
	}
	db.Raw(`
		SELECT `+"`interval`"+` as interval, kind, COUNT(*) as count
		FROM market_klines
		WHERE symbol = ?
		GROUP BY `+"`interval`"+`, kind
		ORDER BY kind, `+"`interval`"+`
	`, "BDXNUSDT").Scan(&klineStats)

	fmt.Printf("  market_klines:\n")
	if len(klineStats) > 0 {
		for _, stat := range klineStats {
			fmt.Printf("    %s %s: %d 条记录\n", stat.Kind, stat.Interval, stat.Count)
		}
	} else {
		fmt.Printf("    无K线数据\n")
	}

	// 24小时统计数据
	var stats24hCount int64
	db.Raw("SELECT COUNT(*) FROM binance_24h_stats WHERE symbol = ?", "BDXNUSDT").Scan(&stats24hCount)
	fmt.Printf("  binance_24h_stats: %d 条记录\n", stats24hCount)

	// 24小时统计历史数据
	var statsHistoryCount int64
	db.Raw("SELECT COUNT(*) FROM binance_24h_stats_history WHERE symbol = ?", "BDXNUSDT").Scan(&statsHistoryCount)
	fmt.Printf("  binance_24h_stats_history: %d 条记录\n", statsHistoryCount)

	// 资金费率数据（如果是期货）
	if marketType == "futures" {
		var fundingRateCount int64
		db.Raw("SELECT COUNT(*) FROM binance_funding_rates WHERE symbol = ?", "BDXNUSDT").Scan(&fundingRateCount)
		fmt.Printf("  binance_funding_rates: %d 条记录\n", fundingRateCount)
	}

	// 订单簿深度数据
	var depthCount int64
	db.Raw("SELECT COUNT(*) FROM binance_order_book_depth WHERE symbol = ?", "BDXNUSDT").Scan(&depthCount)
	fmt.Printf("  binance_order_book_depth: %d 条记录\n", depthCount)

	// 交易数据
	var tradeCount int64
	db.Raw("SELECT COUNT(*) FROM binance_trades WHERE symbol = ?", "BDXNUSDT").Scan(&tradeCount)
	fmt.Printf("  binance_trades: %d 条记录\n", tradeCount)

	// 计算K线数据总数
	var totalKlines int64
	for _, stat := range klineStats {
		totalKlines += stat.Count
	}

	fmt.Printf("\n📈 数据汇总:\n")
	fmt.Printf("  K线数据总量: %d 条\n", totalKlines)
	fmt.Printf("  数据表总数: %d 个表有数据\n",
		func() int {
			count := 1 // exchange_info总是有的
			if totalKlines > 0 {
				count++
			}
			if stats24hCount > 0 {
				count++
			}
			if statsHistoryCount > 0 {
				count++
			}
			if marketType == "futures" {
				var fundingRateCount int64
				db.Raw("SELECT COUNT(*) FROM binance_funding_rates WHERE symbol = ?", "BDXNUSDT").Scan(&fundingRateCount)
				if fundingRateCount > 0 {
					count++
				}
			}
			if depthCount > 0 {
				count++
			}
			if tradeCount > 0 {
				count++
			}
			return count
		}())

	fmt.Println("\n=== 分析完成 ===")
}
