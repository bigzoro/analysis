package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== BDXNUSDT现货同步问题深度分析 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 1. 检查BDXNUSDT在exchange_info中的所有记录
	fmt.Println("🔍 步骤1: 检查BDXNUSDT的exchange_info记录")
	var allRecords []struct {
		ID            uint
		Symbol        string
		MarketType    string
		Status        string
		IsActive      bool
		CreatedAt     string
		UpdatedAt     string
		DeactivatedAt *string
	}

	db.Raw(`
		SELECT id, symbol, market_type, status, is_active,
			   created_at, updated_at, deactivated_at
		FROM binance_exchange_info
		WHERE symbol = ?
		ORDER BY market_type, created_at
	`, "BDXNUSDT").Scan(&allRecords)

	for i, record := range allRecords {
		deactivatedStr := "NULL"
		if record.DeactivatedAt != nil {
			deactivatedStr = *record.DeactivatedAt
		}

		fmt.Printf("  记录%d: ID=%d, 市场=%s, 状态=%s, 活跃=%v\n",
			i+1, record.ID, record.MarketType, record.Status, record.IsActive)
		fmt.Printf("         创建时间=%s, 更新时间=%s, 下架时间=%s\n",
			record.CreatedAt, record.UpdatedAt, deactivatedStr)
	}

	// 2. 模拟GetUSDTTradingPairsByMarket查询
	fmt.Println("\n🔍 步骤2: 模拟K线同步器查询逻辑")

	var spotSymbols []string
	var futuresSymbols []string

	// 现货查询
	db.Raw(`
		SELECT symbol FROM binance_exchange_info
		WHERE quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?
		ORDER BY symbol
	`, "USDT", "TRADING", "spot", true).Scan(&spotSymbols)

	// 期货查询
	db.Raw(`
		SELECT symbol FROM binance_exchange_info
		WHERE quote_asset = ? AND status = ? AND market_type = ? AND is_active = ?
		ORDER BY symbol
	`, "USDT", "TRADING", "futures", true).Scan(&futuresSymbols)

	fmt.Printf("  现货活跃交易对数量: %d\n", len(spotSymbols))
	fmt.Printf("  期货活跃交易对数量: %d\n", len(futuresSymbols))

	// 检查BDXNUSDT是否在查询结果中
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

	fmt.Printf("  BDXNUSDT在现货活跃列表: %v ❌ (应该为false)\n", bdxnInSpot)
	fmt.Printf("  BDXNUSDT在期货活跃列表: %v ✅ (应该为true)\n", bdxnInFutures)

	// 3. 检查Redis缓存清理逻辑的问题
	fmt.Println("\n🔍 步骤3: 分析Redis缓存清理逻辑")

	// 获取所有非活跃的现货交易对
	var inactiveSpot []struct {
		Symbol        string
		DeactivatedAt string
	}

	db.Raw(`
		SELECT symbol, deactivated_at FROM binance_exchange_info
		WHERE market_type = 'spot' AND is_active = false
		ORDER BY deactivated_at DESC
		LIMIT 10
	`).Scan(&inactiveSpot)

	fmt.Printf("  最近10个下架的现货交易对:\n")
	for _, item := range inactiveSpot {
		if item.Symbol == "BDXNUSDT" {
			fmt.Printf("    ✅ BDXNUSDT 下架时间: %s\n", item.DeactivatedAt)
		}
	}

	// 4. 检查是否存在竞态条件
	fmt.Println("\n🔍 步骤4: 检查数据同步时间线")

	var timeline []struct {
		Symbol     string
		MarketType string
		IsActive   bool
		UpdatedAt  string
	}

	db.Raw(`
		SELECT symbol, market_type, is_active, updated_at
		FROM binance_exchange_info
		WHERE symbol = ?
		ORDER BY updated_at DESC
		LIMIT 5
	`, "BDXNUSDT").Scan(&timeline)

	fmt.Printf("  BDXNUSDT更新时间线 (最近5次):\n")
	for _, t := range timeline {
		fmt.Printf("    %s %s: 活跃=%v, 更新=%s\n",
			t.Symbol, t.MarketType, t.IsActive, t.UpdatedAt)
	}

	// 5. 分析问题根因
	fmt.Println("\n💡 问题根因分析:")

	if bdxnInSpot {
		fmt.Println("  ❌ 严重问题: BDXNUSDT现货记录仍然标记为活跃!")
		fmt.Println("  ❌ 这会导致K线同步器尝试同步已下架的现货BDXNUSDT")
		fmt.Println("  🔧 解决方案: 将现货BDXNUSDT的is_active设置为false")
	} else {
		fmt.Println("  ✅ 数据库状态正确: BDXNUSDT现货记录已正确标记为非活跃")
		fmt.Println("  ❓ 问题可能在于: 缓存清理时机或并发问题")
		fmt.Println("  🔍 需要检查数据同步服务的执行顺序")
	}

	// 6. 提供修复建议
	fmt.Println("\n🛠️ 修复建议:")

	if bdxnInSpot {
		fmt.Println("  1. 执行SQL修复现货记录状态:")
		fmt.Println("     UPDATE binance_exchange_info")
		fmt.Println("     SET is_active = false, deactivated_at = NOW()")
		fmt.Println("     WHERE symbol = 'BDXNUSDT' AND market_type = 'spot';")
	}

	fmt.Println("  2. 重启数据同步服务，确保exchange_info同步在缓存清理之前")
	fmt.Println("  3. 清理Redis缓存中的BDXNUSDT spot条目")

	fmt.Println("\n=== 分析完成 ===")
}