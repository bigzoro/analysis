package main

import (
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🎯 手动触发网格交易策略执行")
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

	// 1. 检查网格策略
	fmt.Printf("📊 检查网格策略:\n")
	var strategy struct {
		ID         uint   `json:"id"`
		Name       string `json:"name"`
		IsRunning  bool   `json:"is_running"`
		LastRunAt  *string `json:"last_run_at"`
	}

	err = gdb.Raw(`
		SELECT id, name, is_running, last_run_at
		FROM trading_strategies
		WHERE grid_trading_enabled = true AND id = 29
	`).Scan(&strategy).Error

	if err != nil {
		log.Fatalf("❌ 查询策略失败: %v", err)
	}

	fmt.Printf("  策略 #%d: %s\n", strategy.ID, strategy.Name)
	fmt.Printf("  运行状态: %v\n", strategy.IsRunning)

	if !strategy.IsRunning {
		fmt.Printf("❌ 策略未运行，无法触发执行\n")
		return
	}

	// 2. 创建策略执行记录
	fmt.Printf("\n🚀 创建策略执行记录:\n")

	// 先检查是否有正在进行的执行
	var runningCount int64
	err = gdb.Model(&struct{}{}).Table("strategy_executions").
		Where("strategy_id = ? AND status IN ('running', 'pending')", strategy.ID).
		Count(&runningCount).Error

	if err != nil {
		log.Fatalf("❌ 检查执行状态失败: %v", err)
	}

	if runningCount > 0 {
		fmt.Printf("⚠️  策略正在执行中 (%d 个进行中的执行)，请等待完成\n", runningCount)
		return
	}

	// 创建新的执行记录
	result := gdb.Exec(`
		INSERT INTO strategy_executions (
			strategy_id, user_id, status, logs, created_at, updated_at,
			total_orders, success_orders, failed_orders,
			total_pnl, win_rate, total_investment, current_value,
			create_orders, execution_delay
		) VALUES (?, 1, 'pending', '手动触发执行', NOW(), NOW(), 0, 0, 0, 0, 0, 0, 0, 1, 30)
	`, strategy.ID)

	if result.Error != nil {
		log.Fatalf("❌ 创建执行记录失败: %v", result.Error)
	}

	// 获取刚创建的执行ID
	var executionID uint
	err = gdb.Raw("SELECT LAST_INSERT_ID()").Scan(&executionID).Error
	if err != nil {
		log.Printf("⚠️  获取执行ID失败: %v", err)
	} else {
		fmt.Printf("✅ 已创建执行记录 #%d\n", executionID)
	}

	// 3. 更新策略的最后运行时间
	err = gdb.Exec(`
		UPDATE trading_strategies
		SET last_run_at = NOW()
		WHERE id = ?
	`, strategy.ID).Error

	if err != nil {
		log.Printf("⚠️  更新最后运行时间失败: %v", err)
	} else {
		fmt.Printf("✅ 已更新策略最后运行时间\n")
	}

	fmt.Printf("\n🎉 手动触发完成！\n")
	fmt.Printf("📝 接下来:\n")
	fmt.Printf("  1. 检查调度器日志，看是否开始处理执行 #%d\n", executionID)
	fmt.Printf("  2. 等待几秒钟，然后检查订单表是否有新订单\n")
	fmt.Printf("  3. 如果有订单但状态为 'pending'，说明API调用成功\n")
	fmt.Printf("  4. 如果没有订单，查看详细日志了解决策结果\n")

	fmt.Printf("\n⏱️  当前时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Printf("💡 建议等待 10-30 秒，然后运行验证脚本检查结果\n")
}