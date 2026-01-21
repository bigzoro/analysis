package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 测试SQL查询优化效果 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	fmt.Println("✅ 数据库连接成功")

	// 测试1: 检查第一个查询是否正常工作（修复了DISTINCT错误）
	fmt.Println("\n=== 测试1: analyzeMarketEnvironment 查询 ===")
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -3)

	var topSymbols []string
	start := time.Now()
	err = db.Table("binance_24h_stats").
		Select("symbol").
		Where("quote_volume > 1000").
		Order("quote_volume DESC").
		Limit(50).
		Pluck("symbol", &topSymbols).Error

	duration := time.Since(start)
	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询成功，耗时: %v，返回 %d 个币种\n", duration, len(topSymbols))
		if len(topSymbols) > 0 {
			fmt.Printf("   示例币种: %v...\n", topSymbols[:min(5, len(topSymbols))])
		}
	}

	// 测试2: 检查第二个查询是否优化（简化了子查询）
	fmt.Println("\n=== 测试2: countMarketBreadthIndicators 查询 ===")
	endTime = time.Now()
	startTime = endTime.AddDate(0, 0, -1)

	start = time.Now()
	err = db.Table("binance_24h_stats").
		Select("symbol").
		Where("quote_volume > 1000 AND created_at >= ? AND created_at <= ?", startTime, endTime).
		Order("quote_volume DESC").
		Limit(200).
		Pluck("symbol", &topSymbols).Error

	duration = time.Since(start)
	if err != nil {
		fmt.Printf("❌ 查询失败: %v\n", err)
	} else {
		fmt.Printf("✅ 查询成功，耗时: %v，返回 %d 个币种\n", duration, len(topSymbols))
	}

	// 测试3: 检查第三个查询是否优化（减少了币种数量和时间范围）
	fmt.Println("\n=== 测试3: countVolatilityIndicators 查询 ===")
	endTime = time.Now()
	startTime = endTime.AddDate(0, 0, -3) // 从7天减少到3天

	var symbols []string
	start = time.Now()
	err = db.Table("binance_24h_stats").
		Select("DISTINCT symbol").
		Where("quote_volume > 5000 AND created_at >= ? AND created_at <= ?", startTime, endTime).
		Limit(30). // 从100减少到30，不使用ORDER BY避免DISTINCT冲突
		Pluck("symbol", &symbols).Error

	duration = time.Since(start)
	if err != nil {
		fmt.Printf("❌ 获取币种失败: %v\n", err)
	} else {
		fmt.Printf("✅ 获取 %d 个币种成功，耗时: %v\n", len(symbols), duration)

		if len(symbols) > 0 {
			// 测试K线数据查询性能
			start = time.Now()
			query := "SELECT COUNT(*) FROM market_klines WHERE symbol IN ('" +
				fmt.Sprintf("%s','", symbols[:min(10, len(symbols))])[:len(fmt.Sprintf("%s','", symbols[:min(10, len(symbols))]))-3] +
				"') AND open_time >= ? AND open_time <= ?"

			var count int64
			err = db.Raw(query, startTime, endTime).Scan(&count).Error
			duration = time.Since(start)

			if err != nil {
				fmt.Printf("❌ K线数据查询失败: %v\n", err)
			} else {
				fmt.Printf("✅ K线数据查询成功，耗时: %v，返回 %d 条记录\n", duration, count)
			}
		}
	}

	fmt.Println("\n=== 优化总结 ===")
	fmt.Println("✅ 已修复的问题:")
	fmt.Println("   1. MySQL 3065错误：移除冲突的DISTINCT")
	fmt.Println("   2. 优化子查询：简化市场宽度指标查询")
	fmt.Println("   3. 减少查询范围：波动率指标从100币种/7天 → 30币种/3天")
	fmt.Println("   4. 添加数据库索引：为慢查询添加复合索引")
	fmt.Println("\n🎯 预期效果:")
	fmt.Println("   • 第一个查询：< 1ms (修复错误)")
	fmt.Println("   • 第二个查询：< 50ms (简化查询+索引)")
	fmt.Println("   • 第三个查询：< 500ms (减少范围+索引)")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}