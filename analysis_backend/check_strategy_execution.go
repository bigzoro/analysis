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

	fmt.Println("=== 策略执行配置检查 ===")

	// 查询最近的策略执行记录
	var executions []db.StrategyExecution
	if err := gdb.Preload("Strategy").
		Order("created_at DESC").
		Limit(10).
		Find(&executions).Error; err != nil {
		log.Fatalf("Failed to query strategy executions: %v", err)
	}

	if len(executions) == 0 {
		fmt.Println("❌ 没有找到任何策略执行记录")
		return
	}

	fmt.Printf("找到 %d 条策略执行记录:\n", len(executions))
	for i, exec := range executions {
		fmt.Printf("\n%d. 执行记录 ID: %d\n", i+1, exec.ID)
		fmt.Printf("   策略ID: %d\n", exec.StrategyID)
		fmt.Printf("   用户ID: %d\n", exec.UserID)
		fmt.Printf("   状态: %s\n", exec.Status)
		fmt.Printf("   创建时间: %s\n", exec.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("   启动时间: %s\n", formatTime(&exec.StartTime))
		fmt.Printf("   结束时间: %s\n", formatTime(exec.EndTime))
		fmt.Printf("   每一单金额: %.2f USDT\n", exec.PerOrderAmount)
		fmt.Printf("   创建订单: %v\n", exec.CreateOrders)
		fmt.Printf("   执行延迟: %d秒\n", exec.ExecutionDelay)
		fmt.Printf("   运行次数: %d/%d\n", exec.RunCount, exec.MaxRuns)

		if exec.Strategy.Name != "" {
			fmt.Printf("   策略名称: %s\n", exec.Strategy.Name)
		} else {
			fmt.Printf("   策略名称: (未加载)\n")
		}
	}

	// 检查是否有pending状态的执行记录
	var pendingCount int64
	if err := gdb.Model(&db.StrategyExecution{}).Where("status = ?", "pending").Count(&pendingCount).Error; err != nil {
		log.Fatalf("Failed to count pending executions: %v", err)
	}

	fmt.Printf("\n=== 状态统计 ===\n")
	fmt.Printf("Pending状态执行记录数量: %d\n", pendingCount)

	// 按状态分组统计
	type StatusCount struct {
		Status string
		Count  int64
	}
	var statusCounts []StatusCount
	if err := gdb.Model(&db.StrategyExecution{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&statusCounts).Error; err != nil {
		log.Fatalf("Failed to count by status: %v", err)
	}

	fmt.Println("各状态执行记录数量:")
	for _, sc := range statusCounts {
		fmt.Printf("  %s: %d\n", sc.Status, sc.Count)
	}

	// 检查执行ID 981的具体信息
	fmt.Println("\n=== 执行ID 981检查 ===")
	var exec981 db.StrategyExecution
	if err := gdb.Preload("Strategy").Where("id = ?", 981).First(&exec981).Error; err != nil {
		fmt.Printf("❌ 找不到执行ID 981: %v\n", err)
	} else {
		fmt.Printf("✅ 找到执行ID 981:\n")
		fmt.Printf("   策略ID: %d\n", exec981.StrategyID)
		fmt.Printf("   用户ID: %d\n", exec981.UserID)
		fmt.Printf("   状态: %s\n", exec981.Status)
		fmt.Printf("   创建时间: %s\n", exec981.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("   每一单金额: %.2f USDT\n", exec981.PerOrderAmount)
		fmt.Printf("   创建订单: %v\n", exec981.CreateOrders)
		fmt.Printf("   执行延迟: %d秒\n", exec981.ExecutionDelay)
		if exec981.Strategy.Name != "" {
			fmt.Printf("   策略名称: %s\n", exec981.Strategy.Name)
		}
	}

	// 检查HANAUSDT相关的策略执行
	fmt.Println("\n=== HANAUSDT相关检查 ===")
	var hanaOrders []db.ScheduledOrder
	if err := gdb.Where("symbol = ? AND strategy_id IS NOT NULL", "HANAUSDT").
		Order("created_at DESC").
		Limit(5).
		Find(&hanaOrders).Error; err != nil {
		log.Fatalf("Failed to query HANAUSDT orders: %v", err)
	}

	if len(hanaOrders) == 0 {
		fmt.Println("❌ 没有找到HANAUSDT的策略订单")
	} else {
		fmt.Printf("找到 %d 条HANAUSDT策略订单:\n", len(hanaOrders))
		for i, order := range hanaOrders {
			fmt.Printf("  %d. 订单ID: %d, 状态: %s, 创建时间: %s\n",
				i+1, order.ID, order.Status, order.CreatedAt.Format("2006-01-02 15:04:05"))
			if order.ExecutionID != nil {
				fmt.Printf("      执行ID: %d\n", *order.ExecutionID)

				// 检查这个执行ID的配置
				var execConfig db.StrategyExecution
				if err := gdb.Where("id = ?", *order.ExecutionID).First(&execConfig).Error; err == nil {
					fmt.Printf("      执行配置 - 金额: %.2f, 状态: %s\n", execConfig.PerOrderAmount, execConfig.Status)
				}
			} else {
				fmt.Printf("      执行ID: (null)\n")
			}
			if order.StrategyID != nil {
				fmt.Printf("      策略ID: %d\n", *order.StrategyID)
			} else {
				fmt.Printf("      策略ID: (null)\n")
			}
		}
	}
}

func formatTime(t *time.Time) string {
	if t == nil || (*t).IsZero() {
		return "(未设置)"
	}
	return (*t).Format("2006-01-02 15:04:05")
}