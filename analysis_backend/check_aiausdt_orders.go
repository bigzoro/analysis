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

	fmt.Println("=== AIAUSDT 平仓订单分析 ===")

	// 检查AIAUSDT的所有订单
	var allOrders []db.ScheduledOrder
	if err := gdb.Where("symbol = ?", "AIAUSDT").
		Order("created_at DESC").
		Find(&allOrders).Error; err != nil {
		log.Fatalf("Failed to query AIAUSDT orders: %v", err)
	}

	fmt.Printf("AIAUSDT 总订单数: %d\n", len(allOrders))

	// 分析订单状态分布
	statusCount := make(map[string]int)
	for _, order := range allOrders {
		statusCount[order.Status]++
	}

	fmt.Println("\n订单状态分布:")
	for status, count := range statusCount {
		fmt.Printf("  %s: %d\n", status, count)
	}

	// 专门检查平仓订单 (reduce_only = true)
	var closeOrders []db.ScheduledOrder
	if err := gdb.Where("symbol = ? AND reduce_only = ?", "AIAUSDT", true).
		Order("created_at DESC").
		Limit(10).
		Find(&closeOrders).Error; err != nil {
		log.Fatalf("Failed to query AIAUSDT close orders: %v", err)
	}

	fmt.Printf("\nAIAUSDT 最近10条平仓订单:\n")
	for i, order := range closeOrders {
		fmt.Printf("%d. ID:%d 状态:%s 创建时间:%s\n",
			i+1, order.ID, order.Status,
			order.CreatedAt.Format("2006-01-02 15:04:05"))
	}

	// 检查最近24小时的所有订单（包括不同状态的平仓订单）
	now := time.Now().UTC()
	past24h := now.Add(-24 * time.Hour)

	var recentOrders []db.ScheduledOrder
	if err := gdb.Where("symbol = ? AND created_at >= ?", "AIAUSDT", past24h).
		Order("created_at DESC").
		Find(&recentOrders).Error; err != nil {
		log.Fatalf("Failed to query recent AIAUSDT orders: %v", err)
	}

	fmt.Printf("\n最近24小时内AIAUSDT的所有订单数: %d\n", len(recentOrders))
	for i, order := range recentOrders {
		age := now.Sub(order.CreatedAt)
		fmt.Printf("%d. ID:%d 状态:%s 平仓:%v 创建时间:%s (%.1f小时前)\n",
			i+1, order.ID, order.Status, order.ReduceOnly,
			order.CreatedAt.Format("2006-01-02 15:04:05"),
			age.Hours())
	}

	// 检查最近1小时的所有订单
	past1h := now.Add(-1 * time.Hour)
	var veryRecentOrders []db.ScheduledOrder
	if err := gdb.Where("symbol = ? AND created_at >= ?", "AIAUSDT", past1h).
		Order("created_at DESC").
		Find(&veryRecentOrders).Error; err != nil {
		log.Fatalf("Failed to query very recent AIAUSDT orders: %v", err)
	}

	fmt.Printf("\n最近1小时内AIAUSDT的所有订单数: %d\n", len(veryRecentOrders))
	for i, order := range veryRecentOrders {
		age := now.Sub(order.CreatedAt)
		fmt.Printf("%d. ID:%d 状态:%s 平仓:%v 创建时间:%s (%.1f分钟前)\n",
			i+1, order.ID, order.Status, order.ReduceOnly,
			order.CreatedAt.Format("2006-01-02 15:04:05"),
			age.Minutes())
	}

	// 检查可能被认为是平仓订单的其他状态
	fmt.Printf("\n=== 检查不同状态的平仓订单 ===\n")
	statuses := []string{"filled", "completed", "success"}
	for _, status := range statuses {
		var orders []db.ScheduledOrder
		if err := gdb.Where("symbol = ? AND reduce_only = ? AND status = ? AND created_at >= ?",
			"AIAUSDT", true, status, past1h).
			Order("created_at DESC").
			Find(&orders).Error; err != nil {
			log.Printf("Failed to query %s orders: %v", status, err)
			continue
		}

		if len(orders) > 0 {
			fmt.Printf("状态'%s'的平仓订单数 (1小时内): %d\n", status, len(orders))
			for i, order := range orders {
				age := now.Sub(order.CreatedAt)
				fmt.Printf("  %d. ID:%d 创建时间:%s (%.1f分钟前)\n",
					i+1, order.ID,
					order.CreatedAt.Format("2006-01-02 15:04:05"),
					age.Minutes())
			}
		}
	}

	// 检查时间计算逻辑
	fmt.Printf("\n=== 时间计算验证 ===\n")
	fmt.Printf("当前UTC时间: %s\n", now.Format("2006-01-02 15:04:05"))
	fmt.Printf("24小时前UTC时间: %s\n", past24h.Format("2006-01-02 15:04:05"))
	fmt.Printf("1小时前UTC时间: %s\n", past1h.Format("2006-01-02 15:04:05"))

	// 检查数据库中AIAUSDT最新订单的时间
	if len(allOrders) > 0 {
		latestOrder := allOrders[0]
		latestAge := now.Sub(latestOrder.CreatedAt)
		fmt.Printf("\nAIAUSDT最新订单:\n")
		fmt.Printf("  ID: %d\n", latestOrder.ID)
		fmt.Printf("  状态: %s\n", latestOrder.Status)
		fmt.Printf("  平仓: %v\n", latestOrder.ReduceOnly)
		fmt.Printf("  创建时间: %s\n", latestOrder.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  年龄: %.1f分钟\n", latestAge.Minutes())
	}

	// 检查策略执行记录
	var executions []db.StrategyExecution
	if err := gdb.Where("strategy_id IN (SELECT id FROM trading_strategies WHERE name LIKE ?)", "%AIAUSDT%").
		Order("created_at DESC").
		Limit(5).
		Find(&executions).Error; err != nil {
		log.Printf("Failed to query AIAUSDT strategy executions: %v", err)
	}

	if len(executions) > 0 {
		fmt.Printf("\nAIAUSDT相关的策略执行记录:\n")
		for i, exec := range executions {
			fmt.Printf("%d. 执行ID:%d 状态:%s 创建时间:%s\n",
				i+1, exec.ID, exec.Status,
				exec.CreatedAt.Format("2006-01-02 15:04:05"))
		}
	}
}