package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 详细检查策略ID 29的配置 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 查询所有字段
	var result map[string]interface{}
	query := "SELECT id, name, conditions, is_active, user_id, created_at, updated_at FROM trading_strategies WHERE id = 29"
	db.Raw(query).Scan(&result)

	fmt.Printf("原始查询结果:\n")
	for k, v := range result {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// 检查策略执行记录
	fmt.Println("\n=== 检查策略执行记录 ===")
	var executionResult []map[string]interface{}
	execQuery := "SELECT id, strategy_id, status, created_at, updated_at FROM strategy_executions WHERE strategy_id = 29 ORDER BY created_at DESC LIMIT 5"
	db.Raw(execQuery).Scan(&executionResult)

	fmt.Printf("策略执行记录:\n")
	for _, exec := range executionResult {
		fmt.Printf("  执行ID: %v, 状态: %v, 创建时间: %v\n", exec["id"], exec["status"], exec["created_at"])
	}

	// 检查调度记录
	fmt.Println("\n=== 检查调度记录 ===")
	var scheduleResult []map[string]interface{}
	schedQuery := "SELECT id, strategy_id, status, scheduled_at, executed_at FROM scheduled_orders WHERE strategy_id = 29 ORDER BY scheduled_at DESC LIMIT 10"
	db.Raw(schedQuery).Scan(&scheduleResult)

	fmt.Printf("调度记录:\n")
	for _, sched := range scheduleResult {
		fmt.Printf("  调度ID: %v, 状态: %v, 调度时间: %v, 执行时间: %v\n", sched["id"], sched["status"], sched["scheduled_at"], sched["executed_at"])
	}
}
