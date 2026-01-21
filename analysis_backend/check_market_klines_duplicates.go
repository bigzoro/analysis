package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	// 连接数据库 - 不使用parseTime避免数据类型转换问题
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	fmt.Println("=== Market Klines 表重复数据检查 ===")

	// 1. 检查是否有重复记录（违反唯一索引）
	fmt.Println("\n--- 检查重复记录 ---")

	var duplicateGroups int64
	err = db.Table("market_klines").
		Select("count(*)").
		Group("symbol, kind, `interval`, open_time").
		Having("count(*) > 1").
		Count(&duplicateGroups).Error

	if err != nil {
		log.Printf("查询重复记录失败: %v", err)
	} else if duplicateGroups > 0 {
		fmt.Printf("⚠️ 发现 %d 组重复记录!\n", duplicateGroups)

		// 获取最严重的重复情况
		var worstDuplicates []struct {
			Symbol   string
			Kind     string
			Interval string
			Count    int64
		}

		err = db.Table("market_klines").
			Select("symbol, kind, `interval`, count(*) as count").
			Group("symbol, kind, `interval`, open_time").
			Having("count(*) > 1").
			Order("count desc").
			Limit(10).
			Scan(&worstDuplicates).Error

		if err == nil {
			fmt.Println("最严重的重复情况:")
			for _, d := range worstDuplicates {
				fmt.Printf("  %s %s %s: %d 条重复\n", d.Symbol, d.Kind, d.Interval, d.Count)
			}
		}
	} else {
		fmt.Println("✅ 未发现重复记录")
	}

	// 2. 检查数据质量
	fmt.Println("\n--- 数据质量检查 ---")

	var totalCount int64
	db.Table("market_klines").Count(&totalCount)
	fmt.Printf("总记录数: %d\n", totalCount)

	// 检查价格数据质量
	var validPriceCount int64
	db.Table("market_klines").
		Where("open_price > 0 AND high_price > 0 AND low_price > 0 AND close_price > 0").
		Count(&validPriceCount)

	fmt.Printf("有效价格记录: %d (%.2f%%)\n", validPriceCount,
		float64(validPriceCount)/float64(totalCount)*100)

	// 检查交易量数据质量
	var validVolumeCount int64
	db.Table("market_klines").
		Where("volume >= 0").
		Count(&validVolumeCount)

	fmt.Printf("有效交易量记录: %d (%.2f%%)\n", validVolumeCount,
		float64(validVolumeCount)/float64(totalCount)*100)

	// 3. 检查时间范围
	fmt.Println("\n--- 时间范围检查 ---")

	var timeStats struct {
		MinTime string
		MaxTime string
	}

	db.Table("market_klines").
		Select("MIN(FROM_UNIXTIME(open_time/1000)) as min_time, MAX(FROM_UNIXTIME(open_time/1000)) as max_time").
		Row().Scan(&timeStats.MinTime, &timeStats.MaxTime)

	fmt.Printf("最早数据时间: %s\n", timeStats.MinTime)
	fmt.Printf("最新数据时间: %s\n", timeStats.MaxTime)

	// 4. 检查未来时间戳
	var futureCount int64
	currentTime := int64(1736371200000) // 2025-01-09 00:00:00 UTC in milliseconds
	db.Table("market_klines").Where("open_time > ?", currentTime).Count(&futureCount)
	fmt.Printf("未来时间戳记录: %d\n", futureCount)

	// 5. 检查数据分布
	fmt.Println("\n--- 数据分布检查 ---")

	// 按时间间隔统计
	type IntervalStat struct {
		Interval string
		Count    int64
	}
	var intervalStats []IntervalStat
	db.Table("market_klines").
		Select("`interval`, count(*) as count").
		Group("`interval`").
		Order("count desc").
		Scan(&intervalStats)

	fmt.Println("按时间间隔分布:")
	for _, stat := range intervalStats {
		fmt.Printf("  %s: %d 条\n", stat.Interval, stat.Count)
	}

	// 6. 简单的数据一致性检查
	fmt.Println("\n--- 数据一致性检查 ---")

	// 检查是否有OHLC关系不正确的数据 (high >= low, close在范围内等)
	var inconsistentCount int64
	db.Table("market_klines").
		Where("NOT (high_price >= low_price AND high_price >= open_price AND high_price >= close_price AND low_price <= open_price AND low_price <= close_price)").
		Count(&inconsistentCount)

	fmt.Printf("OHLC关系异常记录: %d\n", inconsistentCount)

	fmt.Println("\n=== 检查完成 ===")
}