package main

import (
	pdb "analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("🔍 检查Order-Sync修复后的数据库状态")
	fmt.Println("====================================")

	// 连接数据库
	gdb, err := pdb.OpenMySQL(pdb.Options{
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

	fmt.Println("\n📊 分析修复效果:")

	// 验证修复效果
	fmt.Println("\n🔍 验证修复效果:")

	// 检查活跃条件订单数量
	var activeConditionalOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type IN (?) AND status NOT IN (?)",
		"XNYUSDT", []string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
		[]string{"cancelled", "filled", "executed"}).Find(&activeConditionalOrders).Error

	if err != nil {
		log.Printf("查询活跃条件订单失败: %v", err)
	} else {
		fmt.Printf("XNYUSDT活跃条件订单数量: %d\n", len(activeConditionalOrders))
		if len(activeConditionalOrders) == 0 {
			fmt.Println("🎉 修复成功！所有XNYUSDT条件订单都已被正确取消")
		} else {
			fmt.Println("⚠️ 仍有活跃条件订单，等待下次Order-Sync或检查日志")
			for _, order := range activeConditionalOrders {
				fmt.Printf("   - %s (%s) 状态:%s\n",
					order.ClientOrderId, order.OrderType, order.Status)
			}
		}
	}

	// 检查Bracket订单状态
	var bracketLinks []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ?", "XNYUSDT").Find(&bracketLinks).Error
	if err != nil {
		log.Printf("查询Bracket订单失败: %v", err)
	} else {
		activeCount := 0
		closedCount := 0
		orphanedCount := 0

		for _, link := range bracketLinks {
			switch link.Status {
			case "active":
				activeCount++
			case "closed":
				closedCount++
			case "orphaned":
				orphanedCount++
			}
		}

		fmt.Printf("XNYUSDT Bracket订单统计: 活跃=%d, 已关闭=%d, 孤立=%d\n",
			activeCount, closedCount, orphanedCount)

		if activeCount == 0 {
			fmt.Println("🎉 所有XNYUSDT Bracket订单都已正确关闭！")
		} else {
			fmt.Printf("⚠️ 仍有%d个活跃Bracket订单\n", activeCount)
		}
	}

	fmt.Println("\n✅ 手动Order-Sync测试完成")
}
