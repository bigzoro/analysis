package main

import (
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 检查最新的策略执行详情")
	fmt.Println("=====================================")

	// 连接数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:          "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:  false,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取数据库实例失败: %v", err)
	}

	// 查询最新的执行记录
	var executions []map[string]interface{}
	err = gdb.Raw(`
		SELECT id, strategy_id, status, logs, created_at, updated_at,
			   total_orders, success_orders, failed_orders,
			   total_pnl, win_rate, current_step
		FROM strategy_executions
		WHERE strategy_id = 29
		ORDER BY created_at DESC
		LIMIT 3
	`).Scan(&executions).Error

	if err != nil {
		log.Fatalf("❌ 查询执行记录失败: %v", err)
	}

	fmt.Printf("📋 最新的执行记录:\n")
	for i, exec := range executions {
		fmt.Printf("\n执行 #%d:\n", i+1)
		for k, v := range exec {
			if k == "logs" && v != nil {
				fmt.Printf("  %-12s: %v\n", k, v)
			} else {
				fmt.Printf("  %-12s: %v\n", k, v)
			}
		}

		// 特别分析日志内容
		if logs, ok := exec["logs"].(string); ok && logs != "" {
			fmt.Printf("  日志详情:\n")
			// 这里可以进一步解析日志内容
			if len(logs) > 200 {
				fmt.Printf("    %s...\n", logs[:200])
			} else {
				fmt.Printf("    %s\n", logs)
			}
		}
	}

	// 检查是否有正在运行的执行
	var runningCount int64
	err = gdb.Model(&struct{}{}).Table("strategy_executions").
		Where("strategy_id = 29 AND status = 'running'").Count(&runningCount).Error

	if err == nil {
		fmt.Printf("\n🔄 运行状态: ")
		if runningCount > 0 {
			fmt.Printf("有 %d 个执行正在运行\n", runningCount)
		} else {
			fmt.Printf("没有正在运行的执行\n")
		}
	}

	fmt.Printf("\n🎯 分析建议:\n")
	fmt.Printf("  1. 检查服务日志中是否有 'GridStrategy' 开头的调试信息\n")
	fmt.Printf("  2. 查看是否有 '触发买入信号' 或 '触发卖出信号' 的日志\n")
	fmt.Printf("  3. 确认调度器是否正确调用了网格交易策略\n")
	fmt.Printf("  4. 检查是否有代码异常导致提前返回\n")
}