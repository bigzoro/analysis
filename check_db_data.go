package main

import (
	"fmt"
	"log"
	"database/sql"
	_ "github.com/go-sql-driver/mysql"
)

func main() {
	// 连接数据库
	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 检查数据表
	fmt.Println("📊 数据库数据检查")
	fmt.Println("==================")

	// 检查binance_24h_stats表
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM binance_24h_stats").Scan(&count)
	if err != nil {
		fmt.Printf("❌ 查询binance_24h_stats失败: %v\n", err)
	} else {
		fmt.Printf("✅ binance_24h_stats表总记录数: %d\n", count)
	}

	// 检查最近24小时的数据
	err = db.QueryRow("SELECT COUNT(*) FROM binance_24h_stats WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)").Scan(&count)
	if err != nil {
		fmt.Printf("❌ 查询最近24小时数据失败: %v\n", err)
	} else {
		fmt.Printf("✅ 最近24小时数据条数: %d\n", count)
	}

	// 检查策略表
	err = db.QueryRow("SELECT COUNT(*) FROM trading_strategies").Scan(&count)
	if err != nil {
		fmt.Printf("❌ 查询trading_strategies失败: %v\n", err)
	} else {
		fmt.Printf("✅ trading_strategies表记录数: %d\n", count)
	}

	// 检查scheduled_orders表
	err = db.QueryRow("SELECT COUNT(*) FROM scheduled_orders").Scan(&count)
	if err != nil {
		fmt.Printf("❌ 查询scheduled_orders失败: %v\n", err)
	} else {
		fmt.Printf("✅ scheduled_orders表记录数: %d\n", count)
	}

	// 检查price_caches表
	err = db.QueryRow("SELECT COUNT(*) FROM price_caches").Scan(&count)
	if err != nil {
		fmt.Printf("❌ 查询price_caches失败: %v\n", err)
	} else {
		fmt.Printf("✅ price_caches表总记录数: %d\n", count)
	}

	// 检查price_caches表按kind分组统计
	rows2, err := db.Query("SELECT kind, COUNT(*) as count FROM price_caches GROUP BY kind")
	if err != nil {
		fmt.Printf("❌ 查询price_caches分组统计失败: %v\n", err)
	} else {
		fmt.Println("📊 price_caches表按类型统计:")
		for rows2.Next() {
			var kind string
			var count int
			rows2.Scan(&kind, &count)
			fmt.Printf("   %s: %d 条记录\n", kind, count)
		}
		rows2.Close()
	}

	// 检查最新的5条price_caches记录
	rows3, err := db.Query("SELECT symbol, kind, price, last_updated FROM price_caches ORDER BY updated_at DESC LIMIT 5")
	if err != nil {
		fmt.Printf("❌ 查询最新price_caches记录失败: %v\n", err)
	} else {
		fmt.Println("🔄 最新的5条price_caches记录:")
		for rows3.Next() {
			var symbol, kind, price string
			var lastUpdated string
			rows3.Scan(&symbol, &kind, &price, &lastUpdated)
			fmt.Printf("   %s (%s): %s (更新时间: %s)\n", symbol, kind, price, lastUpdated)
		}
		rows3.Close()
	}

	// 检查2025-12-31这一天的记录
	fmt.Println("\n📅 检查2025-12-31这一天的记录:")
	rows4, err := db.Query("SELECT COUNT(*) FROM price_caches WHERE DATE(last_updated) = '2025-12-31'")
	if err != nil {
		fmt.Printf("❌ 查询2025-12-31记录失败: %v\n", err)
	} else {
		var count int
		if rows4.Next() {
			rows4.Scan(&count)
			fmt.Printf("✅ 2025-12-31的记录数: %d\n", count)
		}
		rows4.Close()
	}

	// 检查当前时间和最近的同步记录详情
	fmt.Println("\n⏰ 检查最近的同步时间详情:")
	rows7, err := db.Query("SELECT symbol, kind, price, last_updated, updated_at FROM price_caches ORDER BY updated_at DESC LIMIT 10")
	if err != nil {
		fmt.Printf("❌ 查询最近同步详情失败: %v\n", err)
	} else {
		fmt.Println("🔍 最近10条记录的时间详情:")
		for rows7.Next() {
			var symbol, kind, price string
			var lastUpdated, updatedAt string
			rows7.Scan(&symbol, &kind, &price, &lastUpdated, &updatedAt)
			fmt.Printf("   %s (%s): %s | last_updated: %s | updated_at: %s\n", symbol, kind, price, lastUpdated, updatedAt)
		}
		rows7.Close()
	}

	// 检查系统当前时间
	var currentTime string
	err = db.QueryRow("SELECT NOW()").Scan(&currentTime)
	if err != nil {
		fmt.Printf("❌ 获取当前时间失败: %v\n", err)
	} else {
		fmt.Printf("🕐 数据库当前时间: %s\n", currentTime)
	}

	// 模拟可能的错误查询场景
	fmt.Println("\n⚠️  可能的查询错误场景检查:")

	// 1. 如果用last_updated查询北京时间2025-12-31
	lastUpdatedCount := 0
	err = db.QueryRow("SELECT COUNT(*) FROM price_caches WHERE DATE(last_updated) = '2025-12-31'").Scan(&lastUpdatedCount)
	if err == nil {
		fmt.Printf("   📅 用last_updated字段查询2025-12-31: %d 条\n", lastUpdatedCount)
	}

	// 2. 如果用updated_at查询北京时间2025-12-31
	updatedAtCount := 0
	err = db.QueryRow("SELECT COUNT(*) FROM price_caches WHERE DATE(updated_at) = '2025-12-31'").Scan(&updatedAtCount)
	if err == nil {
		fmt.Printf("   📅 用updated_at字段查询2025-12-31: %d 条\n", updatedAtCount)
	}

	// 3. 查询特定交易对的记录数
	specificSymbolCount := 0
	err = db.QueryRow("SELECT COUNT(*) FROM price_caches WHERE symbol = 'BTCUSDT'").Scan(&specificSymbolCount)
	if err == nil {
		fmt.Printf("   🪙 BTCUSDT交易对的总记录数: %d 条\n", specificSymbolCount)
	}

	// 4. 查询今天是否有BTCUSDT的记录
	todayBTCCount := 0
	err = db.QueryRow("SELECT COUNT(*) FROM price_caches WHERE symbol = 'BTCUSDT' AND DATE(updated_at) = CURDATE()").Scan(&todayBTCCount)
	if err == nil {
		fmt.Printf("   📅 今天BTCUSDT的记录数: %d 条\n", todayBTCCount)
	}

	// 检查最近的同步记录（按last_updated分组）
	rows5, err := db.Query("SELECT DATE(last_updated) as sync_date, COUNT(*) as count FROM price_caches GROUP BY DATE(last_updated) ORDER BY sync_date DESC LIMIT 5")
	if err != nil {
		fmt.Printf("❌ 查询同步日期统计失败: %v\n", err)
	} else {
		fmt.Println("📊 按同步日期统计最近5天:")
		for rows5.Next() {
			var syncDate string
			var count int
			rows5.Scan(&syncDate, &count)
			fmt.Printf("   %s: %d 条记录\n", syncDate, count)
		}
		rows5.Close()
	}

	// 检查updated_at字段的分布
	rows6, err := db.Query("SELECT DATE(updated_at) as update_date, COUNT(*) as count FROM price_caches GROUP BY DATE(updated_at) ORDER BY update_date DESC LIMIT 5")
	if err != nil {
		fmt.Printf("❌ 查询更新日期统计失败: %v\n", err)
	} else {
		fmt.Println("📊 按更新日期统计最近5天:")
		for rows6.Next() {
			var updateDate string
			var count int
			rows6.Scan(&updateDate, &count)
			fmt.Printf("   %s: %d 条记录\n", updateDate, count)
		}
		rows6.Close()
	}

	// 检查最近的市场数据样本
	fmt.Println("\n📈 最近市场数据样本:")
	rows, err := db.Query(`
		SELECT symbol, price_change_percent, quote_volume, created_at
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)
			AND market_type = 'spot'
		ORDER BY created_at DESC
		LIMIT 5
	`)
	if err != nil {
		fmt.Printf("❌ 查询市场数据样本失败: %v\n", err)
	} else {
		defer rows.Close()
		for rows.Next() {
			var symbol string
			var priceChange sql.NullFloat64
			var volume sql.NullFloat64
			var createdAt string

			err := rows.Scan(&symbol, &priceChange, &volume, &createdAt)
			if err != nil {
				fmt.Printf("❌ 扫描数据失败: %v\n", err)
				continue
			}

			priceStr := "NULL"
			if priceChange.Valid {
				priceStr = fmt.Sprintf("%.2f%%", priceChange.Float64)
			}

			volumeStr := "NULL"
			if volume.Valid {
				volumeStr = fmt.Sprintf("%.0f", volume.Float64)
			}

			fmt.Printf("   %s: 涨幅=%s, 成交量=%s, 时间=%s\n", symbol, priceStr, volumeStr, createdAt)
		}
	}

	// 分析市场环境
	fmt.Println("\n🎯 市场环境分析:")
	var totalSymbols, activeSymbols int
	var avgPriceChange, avgVolatility sql.NullFloat64

	err = db.QueryRow(`
		SELECT
			COUNT(*) as total_symbols,
			COUNT(CASE WHEN quote_volume > 1000000 THEN 1 END) as active_symbols,
			AVG(CASE WHEN price_change_percent IS NOT NULL THEN price_change_percent ELSE 0 END) as avg_price_change,
			AVG(CASE WHEN high_price > low_price AND low_price > 0 THEN (high_price - low_price) / low_price * 100 ELSE 0 END) as avg_volatility
		FROM binance_24h_stats
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
			AND market_type = 'spot'
	`).Scan(&totalSymbols, &activeSymbols, &avgPriceChange, &avgVolatility)

	if err != nil {
		fmt.Printf("❌ 市场环境分析失败: %v\n", err)
	} else {
		fmt.Printf("📊 总交易对数: %d\n", totalSymbols)
		fmt.Printf("🎯 活跃交易对: %d\n", activeSymbols)
		fmt.Printf("📈 平均波动率: %.2f%%\n", avgVolatility.Float64)
		fmt.Printf("💰 平均价格变化: %.2f%%\n", avgPriceChange.Float64)

		// 判断市场环境
		marketEnv := "未知"
		if avgVolatility.Float64 < 4 {
			marketEnv = "横盘整理"
		} else if avgVolatility.Float64 < 8 {
			marketEnv = "震荡市"
		} else {
			marketEnv = "高波动市"
		}

		fmt.Printf("🎪 市场环境: %s\n", marketEnv)
	}

	// 策略推荐
	fmt.Println("\n🎯 策略推荐:")
	fmt.Printf("基于市场数据分析，当前环境适合以下策略:\n")

	if avgVolatility.Float64 < 6 {
		fmt.Println("   ⭐ 均值回归策略 - 适合震荡和横盘市场")
		fmt.Println("   📊 高级均线策略 - 适合温和趋势环境")
	} else {
		fmt.Println("   🚀 均线策略 - 适合趋势明显的市场")
		fmt.Println("   🐻 做空策略 - 适合高波动环境")
	}

	fmt.Println("\n✅ 数据检查完成")
}