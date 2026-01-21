package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 检查策略ID 29的网格配置 ===")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 查询策略29的网格配置
	var result map[string]interface{}
	query := `
		SELECT
			id, name, is_running,
			grid_trading_enabled,
			grid_upper_price,
			grid_lower_price,
			grid_levels,
			grid_investment_amount,
			grid_stop_loss_enabled,
			grid_stop_loss_percent,
			use_symbol_whitelist,
			symbol_whitelist,
			created_at, updated_at
		FROM trading_strategies
		WHERE id = 29
	`
	db.Raw(query).Scan(&result)

	fmt.Printf("策略基本信息:\n")
	fmt.Printf("  ID: %v\n", result["id"])
	fmt.Printf("  名称: %v\n", result["name"])
	fmt.Printf("  是否运行中: %v\n", result["is_running"])

	fmt.Printf("\n网格配置:\n")
	fmt.Printf("  网格交易启用: %v\n", result["grid_trading_enabled"])
	fmt.Printf("  网格上限价格: %v\n", result["grid_upper_price"])
	fmt.Printf("  网格下限价格: %v\n", result["grid_lower_price"])
	fmt.Printf("  网格层数: %v\n", result["grid_levels"])
	fmt.Printf("  投资金额: %v\n", result["grid_investment_amount"])
	fmt.Printf("  止损启用: %v\n", result["grid_stop_loss_enabled"])
	fmt.Printf("  止损百分比: %v%%\n", result["grid_stop_loss_percent"])
	fmt.Printf("  使用白名单: %v\n", result["use_symbol_whitelist"])
	fmt.Printf("  币种白名单: %v\n", result["symbol_whitelist"])

	fmt.Printf("\n时间信息:\n")
	fmt.Printf("  创建时间: %v\n", result["created_at"])
	fmt.Printf("  更新时间: %v\n", result["updated_at"])

	// 检查是否有调度记录
	fmt.Println("\n=== 检查调度记录 ===")
	var scheduleResult []map[string]interface{}
	schedQuery := `
		SELECT id, strategy_id, status, trigger_time, created_at
		FROM scheduled_orders
		WHERE strategy_id = 29
		ORDER BY created_at DESC LIMIT 5
	`
	db.Raw(schedQuery).Scan(&scheduleResult)

	fmt.Printf("调度记录 (最近5条):\n")
	for _, sched := range scheduleResult {
		fmt.Printf("  调度ID: %v, 状态: %v, 触发时间: %v, 创建时间: %v\n",
			sched["id"], sched["status"], sched["trigger_time"], sched["created_at"])
	}
}
