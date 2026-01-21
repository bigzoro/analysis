package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 从环境变量获取数据库连接信息
	dsn := os.Getenv("DATABASE_DSN")
	if dsn == "" {
		dsn = "user:password@tcp(localhost:3306)/analysis?charset=utf8mb4&parseTime=True&loc=Local"
	}

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("=== 期货数据完整性检查 ===")

	// 检查最近的期货数据
	var count int64
	err = db.Table("binance_24h_stats").
		Where("market_type = ? AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)", "futures").
		Count(&count).Error

	if err != nil {
		log.Fatal("查询失败:", err)
	}

	fmt.Printf("最近1小时内的期货记录数: %d\n", count)

	if count == 0 {
		fmt.Println("⚠️ 没有找到最近的期货数据，请检查数据同步服务是否正常运行")
		return
	}

	// 检查买卖盘口数据是否完整
	type StatsResult struct {
		Symbol     string
		BidPrice   float64
		BidQty     float64
		AskPrice   float64
		AskQty     float64
		LastQty    float64
		FirstId    int64
		LastId     int64
		CreatedAt  string
	}

	var results []StatsResult
	err = db.Table("binance_24h_stats").
		Select("symbol, bid_price, bid_qty, ask_price, ask_qty, last_qty, first_id, last_id, created_at").
		Where("market_type = ? AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)", "futures").
		Order("created_at DESC").
		Limit(5).
		Scan(&results).Error

	if err != nil {
		log.Fatal("查询详细数据失败:", err)
	}

	fmt.Println("\n=== 最近5条期货记录的买卖盘口数据 ===")
	fmt.Printf("%-12s %-12s %-12s %-12s %-12s %-10s %-10s %s\n",
		"Symbol", "BidPrice", "BidQty", "AskPrice", "AskQty", "LastQty", "FirstId", "LastId")

	for _, r := range results {
		fmt.Printf("%-12s %-12.2f %-12.4f %-12.2f %-12.4f %-10.6f %-10d %-10d\n",
			r.Symbol, r.BidPrice, r.BidQty, r.AskPrice, r.AskQty, r.LastQty, r.FirstId, r.LastId)
	}

	// 检查数据完整性
	completeCount := 0
	incompleteCount := 0

	for _, r := range results {
		if r.BidPrice > 0 && r.BidQty > 0 && r.AskPrice > 0 && r.AskQty > 0 && r.LastQty > 0 && r.FirstId > 0 && r.LastId > 0 {
			completeCount++
		} else {
			incompleteCount++
		}
	}

	fmt.Printf("\n=== 数据完整性统计 ===\n")
	fmt.Printf("完整记录数: %d\n", completeCount)
	fmt.Printf("不完整记录数: %d\n", incompleteCount)
	fmt.Printf("完整率: %.1f%%\n", float64(completeCount)/float64(len(results))*100)

	if incompleteCount == 0 {
		fmt.Println("✅ 期货数据完整性检查通过！")
	} else {
		fmt.Println("❌ 发现不完整的数据记录")
	}
}