package main

import (
	"analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("=== 分析策略ID 33的执行历史和性能数据 ===")

	// 连接数据库
	gdb, err := db.OpenMySQL(db.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer gdb.Close()

	// 1. 获取策略执行记录
	fmt.Printf("\n📊 策略执行统计:\n")
	var execStats struct {
		TotalExecutions   int     `json:"total_executions"`
		RunningExecutions int     `json:"running_executions"`
		CompletedCount    int     `json:"completed_count"`
		FailedCount       int     `json:"failed_count"`
		TotalOrders       int     `json:"total_orders"`
		SuccessOrders     int     `json:"success_orders"`
		TotalPnL          float64 `json:"total_pnl"`
		TotalInvestment   float64 `json:"total_investment"`
		CurrentValue      float64 `json:"current_value"`
		WinRate           float64 `json:"win_rate"`
	}

	execQuery := `
		SELECT
			COUNT(*) as total_executions,
			SUM(CASE WHEN status = 'running' THEN 1 ELSE 0 END) as running_executions,
			SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END) as completed_count,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) as failed_count,
			COALESCE(SUM(total_orders), 0) as total_orders,
			COALESCE(SUM(success_orders), 0) as success_orders,
			COALESCE(SUM(total_pnl), 0) as total_pnl,
			COALESCE(SUM(total_investment), 0) as total_investment,
			COALESCE(SUM(current_value), 0) as current_value,
			CASE WHEN SUM(total_orders) > 0 THEN (SUM(success_orders) * 100.0 / SUM(total_orders)) ELSE 0 END as win_rate
		FROM strategy_executions
		WHERE strategy_id = 33
	`

	gdb.GormDB().Raw(execQuery).Scan(&execStats)

	fmt.Printf("  总执行次数: %d\n", execStats.TotalExecutions)
	fmt.Printf("  运行中执行: %d\n", execStats.RunningExecutions)
	fmt.Printf("  已完成执行: %d\n", execStats.CompletedCount)
	fmt.Printf("  失败执行: %d\n", execStats.FailedCount)
	fmt.Printf("  总订单数: %d\n", execStats.TotalOrders)
	fmt.Printf("  成功订单数: %d\n", execStats.SuccessOrders)
	fmt.Printf("  胜率: %.2f%%\n", execStats.WinRate)
	fmt.Printf("  总盈亏: %.4f USDT\n", execStats.TotalPnL)
	fmt.Printf("  总投资: %.4f USDT\n", execStats.TotalInvestment)
	fmt.Printf("  当前价值: %.4f USDT\n", execStats.CurrentValue)

	if execStats.TotalInvestment > 0 {
		roi := (execStats.TotalPnL / execStats.TotalInvestment) * 100
		fmt.Printf("  投资回报率: %.2f%%\n", roi)
	}

	// 2. 最近的执行记录
	fmt.Printf("\n📝 最近5次执行记录:\n")
	var recentExecutions []struct {
		ID          uint    `json:"id"`
		Status      string  `json:"status"`
		StartTime   string  `json:"start_time"`
		EndTime     string  `json:"end_time"`
		Duration    int     `json:"duration"`
		TotalOrders int     `json:"total_orders"`
		TotalPnL    float64 `json:"total_pnl"`
		ErrorMsg    string  `json:"error_message"`
	}

	recentQuery := `
		SELECT id, status, DATE_FORMAT(start_time, '%Y-%m-%d %H:%i:%s') as start_time,
			   DATE_FORMAT(end_time, '%Y-%m-%d %H:%i:%s') as end_time,
			   duration, total_orders, total_pnl, error_message
		FROM strategy_executions
		WHERE strategy_id = 33
		ORDER BY created_at DESC LIMIT 5
	`

	gdb.GormDB().Raw(recentQuery).Scan(&recentExecutions)

	for i, exec := range recentExecutions {
		fmt.Printf("  %d. 执行ID: %d\n", i+1, exec.ID)
		fmt.Printf("     状态: %s\n", exec.Status)
		fmt.Printf("     开始时间: %s\n", exec.StartTime)
		if exec.EndTime != "" {
			fmt.Printf("     结束时间: %s\n", exec.EndTime)
			fmt.Printf("     执行时长: %d 秒\n", exec.Duration)
		}
		fmt.Printf("     订单数: %d\n", exec.TotalOrders)
		fmt.Printf("     盈亏: %.4f USDT\n", exec.TotalPnL)
		if exec.ErrorMsg != "" {
			fmt.Printf("     错误: %s\n", exec.ErrorMsg)
		}
		fmt.Println()
	}

	// 3. 订单统计
	fmt.Printf("\n💰 订单统计:\n")
	var orderStats struct {
		TotalOrders     int     `json:"total_orders"`
		FilledOrders    int     `json:"filled_orders"`
		CancelledOrders int     `json:"cancelled_orders"`
		BuyOrders       int     `json:"buy_orders"`
		SellOrders      int     `json:"sell_orders"`
		TotalVolume     float64 `json:"total_volume"`
		SuccessRate     float64 `json:"success_rate"`
	}

	orderQuery := `
		SELECT
			COUNT(*) as total_orders,
			SUM(CASE WHEN status = 'filled' THEN 1 ELSE 0 END) as filled_orders,
			SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled_orders,
			SUM(CASE WHEN side = 'BUY' THEN 1 ELSE 0 END) as buy_orders,
			SUM(CASE WHEN side = 'SELL' THEN 1 ELSE 0 END) as sell_orders,
			COALESCE(SUM(CASE WHEN status = 'filled' THEN quantity * price ELSE 0 END), 0) as total_volume,
			CASE WHEN COUNT(*) > 0 THEN (SUM(CASE WHEN status = 'filled' THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) ELSE 0 END as success_rate
		FROM orders
		WHERE strategy_id = 33
	`

	gdb.GormDB().Raw(orderQuery).Scan(&orderStats)

	fmt.Printf("  总订单数: %d\n", orderStats.TotalOrders)
	fmt.Printf("  已成交订单: %d\n", orderStats.FilledOrders)
	fmt.Printf("  已取消订单: %d\n", orderStats.CancelledOrders)
	fmt.Printf("  买入订单: %d\n", orderStats.BuyOrders)
	fmt.Printf("  卖出订单: %d\n", orderStats.SellOrders)
	fmt.Printf("  成交率: %.2f%%\n", orderStats.SuccessRate)
	fmt.Printf("  总交易量: %.4f USDT\n", orderStats.TotalVolume)

	// 4. 调度订单统计
	fmt.Printf("\n⏰ 调度订单统计:\n")
	var scheduleStats struct {
		TotalScheduled     int `json:"total_scheduled"`
		ExecutedOrders     int `json:"executed_orders"`
		PendingOrders      int `json:"pending_orders"`
		CancelledScheduled int `json:"cancelled_scheduled"`
	}

	scheduleQuery := `
		SELECT
			COUNT(*) as total_scheduled,
			SUM(CASE WHEN status = 'executed' THEN 1 ELSE 0 END) as executed_orders,
			SUM(CASE WHEN status = 'pending' THEN 1 ELSE 0 END) as pending_orders,
			SUM(CASE WHEN status = 'cancelled' THEN 1 ELSE 0 END) as cancelled_scheduled
		FROM scheduled_orders
		WHERE strategy_id = 33
	`

	gdb.GormDB().Raw(scheduleQuery).Scan(&scheduleStats)

	fmt.Printf("  总调度订单: %d\n", scheduleStats.TotalScheduled)
	fmt.Printf("  已执行调度: %d\n", scheduleStats.ExecutedOrders)
	fmt.Printf("  待执行调度: %d\n", scheduleStats.PendingOrders)
	fmt.Printf("  已取消调度: %d\n", scheduleStats.CancelledScheduled)

	// 5. 分析策略表现
	fmt.Printf("\n📈 策略表现分析:\n")

	// 按交易对统计
	var symbolStats []struct {
		Symbol       string  `json:"symbol"`
		OrderCount   int     `json:"order_count"`
		SuccessCount int     `json:"success_count"`
		TotalPnL     float64 `json:"total_pnl"`
		AvgPnL       float64 `json:"avg_pnl"`
		SuccessRate  float64 `json:"success_rate"`
	}

	symbolQuery := `
		SELECT
			o.symbol,
			COUNT(*) as order_count,
			SUM(CASE WHEN o.status = 'filled' THEN 1 ELSE 0 END) as success_count,
			COALESCE(SUM(CASE WHEN o.status = 'filled' THEN o.pnl ELSE 0 END), 0) as total_pnl,
			CASE WHEN COUNT(*) > 0 THEN COALESCE(SUM(CASE WHEN o.status = 'filled' THEN o.pnl ELSE 0 END), 0) / COUNT(*) ELSE 0 END as avg_pnl,
			CASE WHEN COUNT(*) > 0 THEN (SUM(CASE WHEN o.status = 'filled' THEN 1 ELSE 0 END) * 100.0 / COUNT(*)) ELSE 0 END as success_rate
		FROM orders o
		WHERE o.strategy_id = 33
		GROUP BY o.symbol
		ORDER BY total_pnl DESC
		LIMIT 10
	`

	gdb.GormDB().Raw(symbolQuery).Scan(&symbolStats)

	fmt.Printf("  按交易对表现排名:\n")
	for i, stat := range symbolStats {
		fmt.Printf("    %d. %s: 订单%d个, 成功率%.1f%%, 总盈亏%.2fU, 平均盈亏%.2fU\n",
			i+1, stat.Symbol, stat.OrderCount, stat.SuccessRate, stat.TotalPnL, stat.AvgPnL)
	}

	// 6. 总结分析
	fmt.Printf("\n🎯 策略总结分析:\n")

	if execStats.TotalExecutions > 0 {
		fmt.Printf("✅ 策略已执行 %d 次\n", execStats.TotalExecutions)

		if execStats.WinRate >= 50 {
			fmt.Printf("✅ 胜率 %.1f%% 表现良好\n", execStats.WinRate)
		} else {
			fmt.Printf("⚠️ 胜率 %.1f%% 需要优化\n", execStats.WinRate)
		}

		if execStats.TotalPnL > 0 {
			fmt.Printf("✅ 总盈利 %.2f USDT\n", execStats.TotalPnL)
		} else {
			fmt.Printf("❌ 总亏损 %.2f USDT\n", execStats.TotalPnL)
		}

		if orderStats.SuccessRate >= 70 {
			fmt.Printf("✅ 订单成交率 %.1f%% 很好\n", orderStats.SuccessRate)
		} else {
			fmt.Printf("⚠️ 订单成交率 %.1f%% 需要关注\n", orderStats.SuccessRate)
		}
	} else {
		fmt.Printf("📝 策略尚未执行\n")
	}
}
