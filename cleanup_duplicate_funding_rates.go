package main

import (
	"fmt"
	"log"
	"os"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 清理 binance_funding_rates 表重复数据 ===")

	// 从环境变量获取数据库连接信息
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	}

	fmt.Printf("连接数据库: %s\n", dsn)

	// 连接数据库
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 1. 检查重复数据
	fmt.Println("\n=== 步骤1: 检查重复数据 ===")
	var duplicates []struct {
		Symbol     string
		FundingTime int64
		Count      int64
	}

	err = db.Raw(`
		SELECT symbol, funding_time, COUNT(*) as count
		FROM binance_funding_rates
		GROUP BY symbol, funding_time
		HAVING COUNT(*) > 1
		ORDER BY count DESC, symbol, funding_time
	`).Scan(&duplicates).Error

	if err != nil {
		log.Fatalf("查询重复数据失败: %v", err)
	}

	if len(duplicates) == 0 {
		fmt.Println("✅ 没有发现重复数据")
		return
	}

	fmt.Printf("发现 %d 组重复数据:\n", len(duplicates))
	totalDuplicates := int64(0)
	for _, dup := range duplicates {
		fmt.Printf("  - %s @ %d: %d 条重复记录\n", dup.Symbol, dup.FundingTime, dup.Count)
		totalDuplicates += dup.Count - 1 // 减去保留的一条
	}
	fmt.Printf("需要删除的重复记录总数: %d\n", totalDuplicates)

	// 2. 备份重复数据（可选）
	fmt.Println("\n=== 步骤2: 备份重复数据 ===")
	backupSQL := `
		CREATE TABLE IF NOT EXISTS binance_funding_rates_duplicates_backup AS
		SELECT * FROM binance_funding_rates
		WHERE (symbol, funding_time) IN (
			SELECT symbol, funding_time
			FROM binance_funding_rates
			GROUP BY symbol, funding_time
			HAVING COUNT(*) > 1
		)
	`

	err = db.Exec(backupSQL).Error
	if err != nil {
		log.Printf("⚠️ 备份重复数据失败: %v", err)
		fmt.Println("继续执行清理...")
	} else {
		fmt.Println("✅ 重复数据已备份到 binance_funding_rates_duplicates_backup 表")
	}

	// 3. 清理重复数据
	fmt.Println("\n=== 步骤3: 清理重复数据 ===")
	fmt.Println("⚠️  这将删除重复记录，只保留每组中的最新一条...")

	// 创建临时表存储要保留的ID
	err = db.Exec(`
		CREATE TEMPORARY TABLE funding_rates_to_keep AS
		SELECT MIN(id) as keep_id
		FROM binance_funding_rates
		GROUP BY symbol, funding_time
	`).Error

	if err != nil {
		log.Fatalf("创建临时表失败: %v", err)
	}

	// 删除不在保留列表中的记录
	result := db.Exec(`
		DELETE bfr FROM binance_funding_rates bfr
		LEFT JOIN funding_rates_to_keep ftk ON bfr.id = ftk.keep_id
		WHERE ftk.keep_id IS NULL
	`)

	if result.Error != nil {
		log.Fatalf("删除重复数据失败: %v", result.Error)
	}

	deletedCount := result.RowsAffected
	fmt.Printf("✅ 成功删除 %d 条重复记录\n", deletedCount)

	// 清理临时表
	db.Exec("DROP TEMPORARY TABLE funding_rates_to_keep")

	// 4. 验证清理结果
	fmt.Println("\n=== 步骤4: 验证清理结果 ===")
	var remainingDuplicates []struct {
		Symbol     string
		FundingTime int64
		Count      int64
	}

	err = db.Raw(`
		SELECT symbol, funding_time, COUNT(*) as count
		FROM binance_funding_rates
		GROUP BY symbol, funding_time
		HAVING COUNT(*) > 1
	`).Scan(&remainingDuplicates).Error

	if err != nil {
		log.Printf("验证清理结果失败: %v", err)
	} else if len(remainingDuplicates) == 0 {
		fmt.Println("✅ 重复数据清理完成！")
	} else {
		fmt.Printf("⚠️  仍有 %d 组重复数据:\n", len(remainingDuplicates))
		for _, dup := range remainingDuplicates {
			fmt.Printf("  - %s @ %d: 仍有多条记录\n", dup.Symbol, dup.FundingTime)
		}
	}

	// 5. 创建唯一索引
	fmt.Println("\n=== 步骤5: 创建唯一索引 ===")
	err = db.Exec(`
		ALTER TABLE binance_funding_rates
		ADD UNIQUE KEY idx_funding_rates_symbol_time (symbol, funding_time)
	`).Error

	if err != nil {
		log.Printf("创建唯一索引失败: %v", err)
		fmt.Println("可能仍然存在重复数据，索引创建失败")
	} else {
		fmt.Println("✅ 唯一索引创建成功！")
	}

	// 6. 显示清理统计
	fmt.Println("\n=== 清理统计 ===")
	var stats struct {
		TotalRecords int64
		UniquePairs  int64
	}
	db.Raw("SELECT COUNT(*) as total_records FROM binance_funding_rates").Scan(&stats)
	db.Raw("SELECT COUNT(*) as unique_pairs FROM (SELECT DISTINCT symbol, funding_time FROM binance_funding_rates) t").Scan(&stats)

	fmt.Printf("总记录数: %d\n", stats.TotalRecords)
	fmt.Printf("唯一 symbol+funding_time 对数: %d\n", stats.UniquePairs)
	fmt.Printf("清理的重复记录数: %d\n", deletedCount)

	fmt.Println("\n=== 清理完成 ===")
	fmt.Println("现在可以安全地创建唯一索引了！")
}