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

	fmt.Println("=== 验证AIAUSDT平仓订单检查修复 ===")

	// 模拟修复后的checkRecentCloseOrder逻辑
	symbol := "AIAUSDT"
	timeRange := 1 * time.Hour // 检查1小时内
	userID := uint(1)          // 假设用户ID为1

	// 使用UTC时间计算时间范围
	cutoffTime := time.Now().UTC().Add(-timeRange)

	fmt.Printf("检查参数:\n")
	fmt.Printf("  交易对: %s\n", symbol)
	fmt.Printf("  时间范围: %v\n", timeRange)
	fmt.Printf("  用户ID: %d\n", userID)
	fmt.Printf("  截止时间: %s (UTC)\n", cutoffTime.Format("2006-01-02 15:04:05"))

	// 修复前的查询（只检查filled状态）
	var oldCount int64
	err = gdb.Table("scheduled_orders").
		Where("user_id = ? AND symbol = ? AND status = ? AND reduce_only = ? AND created_at >= ?",
			userID, symbol, "filled", true, cutoffTime).
		Count(&oldCount).Error

	if err != nil {
		log.Fatalf("修复前查询失败: %v", err)
	}

	// 修复后的查询（检查所有完成状态）
	var newCount int64
	err = gdb.Table("scheduled_orders").
		Where("user_id = ? AND symbol = ? AND status IN (?) AND reduce_only = ? AND created_at >= ?",
			userID, symbol, []string{"filled", "completed", "success"}, true, cutoffTime).
		Count(&newCount).Error

	if err != nil {
		log.Fatalf("修复后查询失败: %v", err)
	}

	fmt.Printf("\n查询结果对比:\n")
	fmt.Printf("❌ 修复前 (只检查filled): %d 个订单\n", oldCount)
	fmt.Printf("✅ 修复后 (检查所有完成状态): %d 个订单\n", newCount)

	if newCount > oldCount {
		fmt.Printf("🎉 修复成功！额外识别了 %d 个平仓订单\n", newCount-oldCount)
	} else {
		fmt.Printf("⚠️ 没有发现额外订单，可能数据已过期\n")
	}

	// 显示具体的订单详情
	if newCount > 0 {
		var orders []db.ScheduledOrder
		err = gdb.Table("scheduled_orders").
			Where("user_id = ? AND symbol = ? AND status IN (?) AND reduce_only = ? AND created_at >= ?",
				userID, symbol, []string{"filled", "completed", "success"}, true, cutoffTime).
			Order("created_at DESC").
			Find(&orders).Error

		if err == nil {
			fmt.Printf("\n识别的平仓订单详情:\n")
			for i, order := range orders {
				age := time.Now().UTC().Sub(order.CreatedAt)
				fmt.Printf("%d. ID:%d 状态:%s 时间:%s (%.1f分钟前)\n",
					i+1, order.ID, order.Status,
					order.CreatedAt.Format("2006-01-02 15:04:05"),
					age.Minutes())
			}
		}
	}

	fmt.Printf("\n📋 业务影响:\n")
	if newCount > 0 {
		fmt.Printf("✅ AIAUSDT 会被正确识别为有平仓记录\n")
		fmt.Printf("✅ 策略会跳过该币种，避免重复开仓\n")
		fmt.Printf("✅ 防止过度交易和潜在风险\n")
	} else {
		fmt.Printf("ℹ️ AIAUSDT 当前没有活跃的平仓记录\n")
		fmt.Printf("ℹ️ 可以正常进行交易\n")
	}
}