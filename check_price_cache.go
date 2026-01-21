package main

import (
	"fmt"
	"log"
	"time"

	"analysis/internal/db"
)

func main() {
	// 连接数据库
	database, err := db.OpenMySQL(db.Options{
		DSN:         "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate: false,
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer database.Close()

	gdb, err := database.DB()
	if err != nil {
		log.Fatalf("Failed to get database instance: %v", err)
	}

	// 检查HANAUSDT的具体缓存情况
	fmt.Printf("=== HANAUSDT Price Cache Check ===\n")

	var hanaRecords []db.PriceCache
	if err := gdb.Model(&db.PriceCache{}).
		Where("symbol = ?", "HANAUSDT").
		Order("updated_at DESC").
		Find(&hanaRecords).Error; err != nil {
		log.Fatalf("Failed to query HANAUSDT records: %v", err)
	}

	if len(hanaRecords) == 0 {
		fmt.Printf("❌ HANAUSDT 没有任何缓存记录\n")
	} else {
		fmt.Printf("✅ HANAUSDT 缓存记录数量: %d\n", len(hanaRecords))
		for i, record := range hanaRecords {
			fmt.Printf("  %d. %s (%s): %s (last_updated: %s, updated_at: %s)\n",
				i+1, record.Symbol, record.Kind, record.Price,
				record.LastUpdated.Format("2006-01-02 15:04:05"),
				record.UpdatedAt.Format("2006-01-02 15:04:05"))
		}
	}

	// 检查30秒内是否有有效的缓存（调度器使用的逻辑）
	now := time.Now()
	fmt.Printf("\n=== 30秒内有效缓存检查 ===\n")
	fmt.Printf("当前时间: %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("30秒前时间: %s\n", now.Add(-30*time.Second).Format("2006-01-02 15:04:05"))

	for _, record := range hanaRecords {
		age := now.Sub(record.LastUpdated)
		isFresh := age <= 30*time.Second
		status := "❌ 过期"
		if isFresh {
			status = "✅ 有效"
		}
		fmt.Printf("  %s (%s): 缓存年龄 %v %s\n",
			record.Symbol, record.Kind, age, status)
	}

	// 查询price_caches表统计信息
	var totalCount int64
	if err := gdb.Model(&db.PriceCache{}).Count(&totalCount).Error; err != nil {
		log.Fatalf("Failed to count price_caches: %v", err)
	}

	// 查询不同kind的数量
	type KindCount struct {
		Kind  string
		Count int64
	}
	var kindCounts []KindCount
	if err := gdb.Model(&db.PriceCache{}).
		Select("kind, count(*) as count").
		Group("kind").
		Scan(&kindCounts).Error; err != nil {
		log.Fatalf("Failed to count by kind: %v", err)
	}

	// 查询最新的几条记录
	var latestRecords []db.PriceCache
	if err := gdb.Model(&db.PriceCache{}).
		Order("updated_at DESC").
		Limit(10).
		Find(&latestRecords).Error; err != nil {
		log.Fatalf("Failed to get latest records: %v", err)
	}

	fmt.Printf("\n=== Price Cache Statistics ===\n")
	fmt.Printf("Total records: %d\n", totalCount)
	fmt.Printf("Records by kind:\n")
	for _, kc := range kindCounts {
		fmt.Printf("  %s: %d\n", kc.Kind, kc.Count)
	}

	fmt.Printf("\nLatest 10 records:\n")
	for _, record := range latestRecords {
		fmt.Printf("  %s (%s): %s (last_updated: %s)\n",
			record.Symbol, record.Kind, record.Price, record.LastUpdated.Format("2006-01-02 15:04:05"))
	}

	// 检查futures类型的最近记录
	var futuresRecords []db.PriceCache
	if err := gdb.Model(&db.PriceCache{}).
		Where("kind = ?", "futures").
		Order("last_updated DESC").
		Limit(5).
		Find(&futuresRecords).Error; err != nil {
		log.Fatalf("Failed to query futures records: %v", err)
	}

	fmt.Printf("\nRecent futures cache records:\n")
	for _, record := range futuresRecords {
		age := now.Sub(record.LastUpdated)
		fmt.Printf("  %s: %s (age: %v)\n",
			record.Symbol, record.LastUpdated.Format("2006-01-02 15:04:05"), age)
	}
}
