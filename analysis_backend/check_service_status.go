package main

import (
	"fmt"
	"log"
	"time"

	pdb "analysis/internal/db"
)


func main() {
	fmt.Println("🔍 检查网格交易服务状态")
	fmt.Println("=====================================")

	// 直接连接数据库
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

	// 1. 检查网格策略状态
	fmt.Printf("📊 网格策略状态:\n")
	var strategies []struct {
		ID          uint    `json:"id"`
		Name        string  `json:"name"`
		IsRunning   bool    `json:"is_running"`
		RunInterval int     `json:"run_interval"`
		LastRunAt   *string `json:"last_run_at"`
	}

	err = gdb.Raw(`
		SELECT id, name, is_running, run_interval, last_run_at
		FROM trading_strategies
		WHERE grid_trading_enabled = true
	`).Scan(&strategies).Error

	if err != nil {
		log.Printf("❌ 查询策略失败: %v", err)
	} else {
		for _, strategy := range strategies {
			fmt.Printf("  策略 #%d: %s\n", strategy.ID, strategy.Name)
			fmt.Printf("    运行状态: %v\n", strategy.IsRunning)
			fmt.Printf("    执行间隔: %d 分钟\n", strategy.RunInterval)
			if strategy.LastRunAt != nil {
				fmt.Printf("    最后运行: %s\n", *strategy.LastRunAt)
			} else {
				fmt.Printf("    最后运行: 从未运行\n")
			}

			// 计算下次运行时间
			if strategy.LastRunAt != nil {
				fmt.Printf("    下次运行: %d 分钟后\n", strategy.RunInterval)
			} else {
				fmt.Printf("    下次运行: 立即 (首次运行)\n")
			}
		}
	}

	// 2. 检查最近的执行记录
	fmt.Printf("\n📋 最近的策略执行记录:\n")
	var executions []struct {
		ID         uint   `json:"id"`
		StrategyID uint   `json:"strategy_id"`
		Status     string `json:"status"`
		CreatedAt  string `json:"created_at"`
	}

	err = gdb.Raw(`
		SELECT id, strategy_id, status, created_at
		FROM strategy_executions
		WHERE strategy_id IN (SELECT id FROM trading_strategies WHERE grid_trading_enabled = true)
		ORDER BY created_at DESC
		LIMIT 3
	`).Scan(&executions).Error

	if err != nil {
		log.Printf("❌ 查询执行记录失败: %v", err)
	} else {
		for _, exec := range executions {
			fmt.Printf("  执行 #%d (策略 %d): %s - %s\n",
				exec.ID, exec.StrategyID, exec.Status, exec.CreatedAt)
		}
	}

	// 3. 检查是否有待处理的执行
	var pendingCount int64
	err = gdb.Model(&struct{}{}).Table("strategy_executions").
		Where("status = 'pending'").Count(&pendingCount).Error

	if err == nil {
		fmt.Printf("\n⏳ 待处理的执行: %d 个\n", pendingCount)
	}

	// 4. 检查调度器进程状态（通过数据库活动判断）
	fmt.Printf("\n🔄 调度器状态分析:\n")

	// 检查最近5分钟的数据库活动
	var recentActivity int64
	err = gdb.Model(&struct{}{}).Table("strategy_executions").
		Where("created_at >= DATE_SUB(NOW(), INTERVAL 5 MINUTE)").Count(&recentActivity).Error

	if err == nil && recentActivity > 0 {
		fmt.Printf("  ✅ 调度器活动: 最近5分钟有 %d 次执行\n", recentActivity)
	} else {
		fmt.Printf("  ❌ 调度器状态: 最近5分钟无活动\n")
		fmt.Printf("  🤔 可能原因: 调度器服务未运行或配置未生效\n")
	}

	// 5. 诊断结论
	fmt.Printf("\n🔍 诊断结论:\n")

	hasRunningStrategy := false
	for _, strategy := range strategies {
		if strategy.IsRunning {
			hasRunningStrategy = true
			break
		}
	}

	if !hasRunningStrategy {
		fmt.Printf("  ❌ 策略问题: 没有运行中的网格策略\n")
		fmt.Printf("  🔧 解决方案: 启用网格策略\n")
	} else if pendingCount == 0 && recentActivity == 0 {
		fmt.Printf("  ❌ 服务问题: 调度器可能未运行\n")
		fmt.Printf("  🔧 解决方案: 检查并重启调度器服务\n")
	} else {
		fmt.Printf("  ✅ 服务正常: 有运行中的策略\n")
		fmt.Printf("  📝 等待执行: 策略按%d分钟间隔运行\n", strategies[0].RunInterval)
	}

	fmt.Printf("\n💡 建议操作:\n")
	fmt.Printf("  1. 确认调度器服务正在运行\n")
	fmt.Printf("  2. 检查服务进程是否存在\n")
	fmt.Printf("  3. 如果服务未运行，重新启动它\n")
	fmt.Printf("  4. 或者手动触发策略执行进行测试\n")

	// 6. 提供手动触发建议
	fmt.Printf("\n🛠️ 手动触发策略执行:\n")
	fmt.Printf("  可以通过API直接触发策略执行:\n")
	fmt.Printf("  POST /api/strategies/%d/execute\n", strategies[0].ID)
	fmt.Printf("  或者修改策略的 run_interval 为较小值进行测试\n")

	// 7. 显示当前时间
	fmt.Printf("\n🕐 当前时间: %s\n", time.Now().Format("2006-01-02 15:04:05"))
}