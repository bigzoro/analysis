package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 手动触发策略执行 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 检查当前策略执行状态
	var executions []map[string]interface{}
	db.Raw("SELECT id, status, created_at FROM strategy_executions WHERE strategy_id = 29 ORDER BY created_at DESC LIMIT 3").Scan(&executions)

	fmt.Printf("当前策略执行状态:\n")
	for _, exec := range executions {
		fmt.Printf("执行ID: %v, 状态: %v, 时间: %v\n", exec["id"], exec["status"], exec["created_at"])
	}

	// 创建新的策略执行记录
	fmt.Printf("\n创建新的策略执行记录...\n")

	// 只使用表中存在的字段
	newExecution := map[string]interface{}{
		"user_id":          1,
		"strategy_id":      29,
		"status":           "pending",
		"start_time":       time.Now(),
		"run_interval":     60,
		"max_runs":         1,
		"auto_stop":        true,
		"create_orders":    true,
		"run_count":        0,
		"total_orders":     0,
		"success_orders":   0,
		"failed_orders":    0,
		"total_pnl":        "0.00000000",
		"win_rate":         "0.00",
		"pnl_percentage":   "0.0000",
		"total_investment": "0.00000000",
		"current_value":    "0.00000000",
		"enable_leverage":  false,
		"execution_delay":  0,
		"created_at":       time.Now(),
		"updated_at":       time.Now(),
	}

	result := db.Table("strategy_executions").Create(&newExecution)
	if result.Error != nil {
		log.Fatalf("创建策略执行记录失败: %v", result.Error)
	}

	fmt.Printf("✅ 成功创建策略执行记录\n")
	fmt.Printf("🎯 策略调度器现在应该会自动执行策略\n")
	fmt.Printf("📊 请检查日志输出，观察网格策略是否产生交易信号\n")

	// 等待一段时间后检查执行结果
	fmt.Printf("\n⏳ 等待5秒后检查执行结果...\n")
	time.Sleep(5 * time.Second)

	var latestExecution []map[string]interface{}
	db.Raw("SELECT id, status, total_orders, success_orders, failed_orders FROM strategy_executions WHERE strategy_id = 29 ORDER BY created_at DESC LIMIT 1").Scan(&latestExecution)

	if len(latestExecution) > 0 {
		exec := latestExecution[0]
		fmt.Printf("执行结果:\n")
		fmt.Printf("执行ID: %v\n", exec["id"])
		fmt.Printf("状态: %v\n", exec["status"])
		fmt.Printf("总订单: %v\n", exec["total_orders"])
		fmt.Printf("成功订单: %v\n", exec["success_orders"])
		fmt.Printf("失败订单: %v\n", exec["failed_orders"])

		if orders, ok := exec["total_orders"].(int64); ok && orders > 0 {
			fmt.Printf("🎉 成功! 策略产生了%d个订单\n", orders)
		} else {
			fmt.Printf("⚠️ 策略仍未产生订单\n")
		}
	}

	// 检查是否有新的调度订单
	var newOrders []map[string]interface{}
	db.Raw("SELECT id, symbol, side, status, created_at FROM scheduled_orders WHERE strategy_id = 29 ORDER BY created_at DESC LIMIT 5").Scan(&newOrders)

	fmt.Printf("\n📋 最新调度订单:\n")
	for _, order := range newOrders {
		fmt.Printf("订单ID: %v, 交易对: %v, 方向: %v, 状态: %v, 时间: %v\n",
			order["id"], order["symbol"], order["side"], order["status"], order["created_at"])
	}
}
