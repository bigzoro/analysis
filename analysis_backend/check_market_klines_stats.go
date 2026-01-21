package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("=== Market Klines 表数据统计 ===")

	// 1. 总记录数
	var totalCount int64
	db.Table("market_klines").Count(&totalCount)
	fmt.Printf("总记录数: %d\n", totalCount)

	// 2. 按市场类型统计
	var spotCount int64
	var futuresCount int64
	db.Table("market_klines").Where("kind = ?", "spot").Count(&spotCount)
	db.Table("market_klines").Where("kind = ?", "futures").Count(&futuresCount)
	fmt.Printf("现货记录数: %d (%.1f%%)\n", spotCount, float64(spotCount)/float64(totalCount)*100)
	fmt.Printf("期货记录数: %d (%.1f%%)\n", futuresCount, float64(futuresCount)/float64(totalCount)*100)

	// 3. 按时间间隔统计
	fmt.Println("\n=== 按时间间隔统计 ===")
	type IntervalStats struct {
		Interval string
		Count    int64
	}
	var intervalStats []IntervalStats
	db.Table("market_klines").
		Select("`interval`, count(*) as count").
		Group("`interval`").
		Order("count desc").
		Scan(&intervalStats)

	for _, stat := range intervalStats {
		fmt.Printf("%s: %d 条\n", stat.Interval, stat.Count)
	}

	// 4. 按交易对统计（前20个）
	fmt.Println("\n=== 按交易对统计（前20个）===")
	type SymbolStats struct {
		Symbol string
		Count  int64
	}
	var symbolStats []SymbolStats
	db.Table("market_klines").
		Select("symbol, count(*) as count").
		Group("symbol").
		Order("count desc").
		Limit(20).
		Scan(&symbolStats)

	for _, stat := range symbolStats {
		fmt.Printf("%s: %d 条\n", stat.Symbol, stat.Count)
	}

	// 5. 时间范围统计
	fmt.Println("\n=== 时间范围统计 ===")
	var minTime, maxTime time.Time
	db.Table("market_klines").Select("MIN(FROM_UNIXTIME(open_time/1000)) as min_time, MAX(FROM_UNIXTIME(open_time/1000)) as max_time").Row().Scan(&minTime, &maxTime)
	fmt.Printf("最早数据: %s\n", minTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("最新数据: %s\n", maxTime.Format("2006-01-02 15:04:05"))

	duration := maxTime.Sub(minTime)
	days := int(duration.Hours() / 24)
	fmt.Printf("数据时间跨度: %d 天\n", days)

	// 6. 估算存储大小
	fmt.Println("\n=== 存储大小估算 ===")
	// 假设每条记录平均大小（包括索引）
	avgRecordSize := 250 // 字节，保守估计
	estimatedSizeGB := float64(totalCount) * float64(avgRecordSize) / (1024 * 1024 * 1024)
	fmt.Printf("估算表大小: %.2f GB (基于%d字节/条记录)\n", estimatedSizeGB, avgRecordSize)

	// 7. 数据密度分析
	fmt.Println("\n=== 数据密度分析 ===")
	if days > 0 {
		dailyRecords := float64(totalCount) / float64(days)
		fmt.Printf("日均记录数: %.0f 条/天\n", dailyRecords)

		// 分析1分钟K线的数据密度
		var minuteKlinesCount int64
		db.Table("market_klines").Where("`interval` = ?", "1m").Count(&minuteKlinesCount)
		if minuteKlinesCount > 0 {
			minuteKlinesPerDay := float64(minuteKlinesCount) / float64(days)
			fmt.Printf("1分钟K线日均: %.0f 条/天 (应为活跃交易对数×1440)\n", minuteKlinesPerDay)
		}
	}

	fmt.Println("\n=== 建议 ===")
	if days > 365 {
		fmt.Printf("⚠️ 数据已存储 %d 天，建议定期清理旧数据\n", days)
		fmt.Println("💡 建议保留最近6-12个月的数据，定期运行清理任务")
	} else {
		fmt.Printf("✅ 数据存储天数 %d 天，相对合理\n", days)
	}

	fmt.Printf("💡 当前记录数: %d 条，建议监控增长趋势\n", totalCount)
}