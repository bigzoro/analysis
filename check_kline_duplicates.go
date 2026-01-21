package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== K线数据重复检查 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}

	sqlDB, _ := db.DB()
	defer sqlDB.Close()

	// 1. 检查K线数据的总量
	var totalKlines int64
	db.Raw("SELECT COUNT(*) FROM market_klines").Scan(&totalKlines)
	fmt.Printf("📊 K线数据总量: %d 条\n", totalKlines)

	// 2. 检查是否有重复的K线记录（基于symbol, kind, interval, open_time）
	fmt.Println("\n🔍 检查K线数据重复情况:")

	// 检查重复记录数
	var duplicateCount int64
	db.Raw(`
		SELECT COUNT(*) FROM (
			SELECT symbol, kind, ` + "`interval`" + `, open_time, COUNT(*) as cnt
			FROM market_klines
			GROUP BY symbol, kind, ` + "`interval`" + `, open_time
			HAVING COUNT(*) > 1
		) as duplicates
	`).Scan(&duplicateCount)

	fmt.Printf("  重复记录组数: %d 个\n", duplicateCount)

	if duplicateCount > 0 {
		// 显示具体的重复记录示例
		var duplicates []struct {
			Symbol   string
			Kind     string
			Interval string
			OpenTime string
			Count    int64
		}

		db.Raw(`
			SELECT symbol, kind, ` + "`interval`" + `, open_time, COUNT(*) as count
			FROM market_klines
			GROUP BY symbol, kind, ` + "`interval`" + `, open_time
			HAVING COUNT(*) > 1
			ORDER BY COUNT(*) DESC
			LIMIT 10
		`).Scan(&duplicates)

		fmt.Printf("  重复记录Top 10:\n")
		for i, dup := range duplicates {
			fmt.Printf("    %d. %s %s %s %s: %d 条重复\n",
				i+1, dup.Symbol, dup.Kind, dup.Interval, dup.OpenTime, dup.Count)
		}

		// 计算重复数据的总量
		var totalDuplicateRecords int64
		db.Raw(`
			SELECT SUM(cnt - 1) FROM (
				SELECT COUNT(*) as cnt
				FROM market_klines
				GROUP BY symbol, kind, ` + "`interval`" + `, open_time
				HAVING COUNT(*) > 1
			) as dup_counts
		`).Scan(&totalDuplicateRecords)

		fmt.Printf("  重复数据总量: %d 条（可以清理）\n", totalDuplicateRecords)
		fmt.Printf("  重复数据占比: %.2f%%\n", float64(totalDuplicateRecords)/float64(totalKlines)*100)
	} else {
		fmt.Printf("  ✅ 无重复记录\n")
	}

	// 3. 检查K线数据分布
	fmt.Println("\n📈 K线数据分布分析:")

	var distributions []struct {
		Kind     string
		Interval string
		Count    int64
	}

	db.Raw(`
		SELECT kind, ` + "`interval`" + `, COUNT(*) as count
		FROM market_klines
		GROUP BY kind, ` + "`interval`" + `
		ORDER BY kind, ` + "`interval`" + `
	`).Scan(&distributions)

	for _, dist := range distributions {
		fmt.Printf("  %s %s: %d 条\n", dist.Kind, dist.Interval, dist.Count)
	}

	// 4. 检查最新同步的数据
	fmt.Println("\n⏰ 检查最新同步的数据:")

	var latestKlines []struct {
		Symbol   string
		Kind     string
		Interval string
		OpenTime string
		UpdatedAt string
	}

	db.Raw(`
		SELECT symbol, kind, ` + "`interval`" + `, open_time, updated_at
		FROM market_klines
		ORDER BY updated_at DESC
		LIMIT 5
	`).Scan(&latestKlines)

	for i, kline := range latestKlines {
		fmt.Printf("  %d. %s %s %s %s (更新: %s)\n",
			i+1, kline.Symbol, kline.Kind, kline.Interval, kline.OpenTime, kline.UpdatedAt)
	}

	// 5. 检查数据一致性
	fmt.Println("\n🔍 检查数据一致性:")

	// 检查是否有异常的OHLC关系（低价 > 高价等）
	var invalidOHLC int64
	db.Raw(`
		SELECT COUNT(*) FROM market_klines
		WHERE CAST(low_price AS DECIMAL(32,8)) > CAST(high_price AS DECIMAL(32,8))
		   OR CAST(open_price AS DECIMAL(32,8)) < 0
		   OR CAST(close_price AS DECIMAL(32,8)) < 0
	`).Scan(&invalidOHLC)

	fmt.Printf("  OHLC数据异常记录: %d 条\n", invalidOHLC)

	// 检查时间戳合理性
	var futureTimestamps int64
	db.Raw(`
		SELECT COUNT(*) FROM market_klines
		WHERE open_time > NOW() + INTERVAL 1 HOUR
	`).Scan(&futureTimestamps)

	fmt.Printf("  未来时间戳记录: %d 条\n", futureTimestamps)

	var oldTimestamps int64
	db.Raw(`
		SELECT COUNT(*) FROM market_klines
		WHERE open_time < NOW() - INTERVAL 2 YEAR
	`).Scan(&oldTimestamps)

	fmt.Printf("  超过2年历史记录: %d 条\n", oldTimestamps)

	// 6. 分析重复数据的来源
	if duplicateCount > 0 {
		fmt.Println("\n💡 重复数据分析:")

		// 检查重复记录的创建时间差异
		var timeDiffs []struct {
			Symbol     string
			Kind       string
			Interval   string
			OpenTime   string
			TimeSpan   string
			RecordCount int64
		}

		db.Raw(`
			SELECT
				symbol, kind, ` + "`interval`" + `, open_time,
				TIMEDIFF(MAX(updated_at), MIN(updated_at)) as time_span,
				COUNT(*) as record_count
			FROM market_klines
			GROUP BY symbol, kind, ` + "`interval`" + `, open_time
			HAVING COUNT(*) > 1
			ORDER BY COUNT(*) DESC
			LIMIT 5
		`).Scan(&timeDiffs)

		fmt.Printf("  重复记录创建时间差异分析:\n")
		for i, diff := range timeDiffs {
			fmt.Printf("    %d. %s %s %s: %d条记录，时间跨度: %s\n",
				i+1, diff.Symbol, diff.Kind, diff.Interval, diff.RecordCount, diff.TimeSpan)
		}

		fmt.Println("\n🔧 建议解决方案:")
		fmt.Println("  1. 运行数据清理脚本来删除重复记录")
		fmt.Println("  2. 检查K线同步器的去重逻辑是否正常工作")
		fmt.Println("  3. 考虑添加唯一约束来防止未来重复")
		fmt.Println("  4. 优化同步策略，避免重复请求相同数据")
	}

	fmt.Println("\n=== 分析完成 ===")
}