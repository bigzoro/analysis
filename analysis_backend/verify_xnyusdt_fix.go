package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 验证XNYUSDT Bracket联动取消修复效果")
	fmt.Println("=====================================")

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

	fmt.Println("\n📊 当前XNYUSDT Bracket订单状态:")

	// 1. 检查Bracket订单状态
	var bracketLinks []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ?", "XNYUSDT").Find(&bracketLinks).Error
	if err != nil {
		log.Printf("查询Bracket订单失败: %v", err)
	} else {
		fmt.Printf("📋 Bracket订单总数: %d\n", len(bracketLinks))
		for _, link := range bracketLinks {
			fmt.Printf("   ID:%d, GroupID:%s, 状态:%s\n", link.ID, link.GroupID, link.Status)
		}
	}

	// 2. 检查活跃条件订单
	var activeConditionalOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type IN (?) AND status NOT IN (?)",
		"XNYUSDT", []string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
		[]string{"cancelled", "filled", "executed"}).Find(&activeConditionalOrders).Error

	if err != nil {
		log.Printf("查询活跃条件订单失败: %v", err)
	} else {
		fmt.Printf("\n🎯 活跃条件订单数量: %d\n", len(activeConditionalOrders))
		if len(activeConditionalOrders) == 0 {
			fmt.Println("✅ 修复成功！所有XNYUSDT条件订单都已被正确取消")
		} else {
			fmt.Println("❌ 修复失败！仍有活跃的条件订单:")
			for _, order := range activeConditionalOrders {
				fmt.Printf("   - %s (%s) 状态:%s\n",
					order.ClientOrderId, order.OrderType, order.Status)
			}
		}
	}

	// 3. 检查开仓订单状态
	var entryOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type = ? AND status = ?",
		"XNYUSDT", "MARKET", "filled").Find(&entryOrders).Error

	if err != nil {
		log.Printf("查询开仓订单失败: %v", err)
	} else {
		fmt.Printf("\n🏠 已执行开仓订单数量: %d\n", len(entryOrders))
		for _, order := range entryOrders {
			fmt.Printf("   - %s 执行时间:%s\n",
				order.ClientOrderId, order.UpdatedAt.Format("15:04:05"))
		}
	}

	fmt.Println("\n💡 修复逻辑说明:")
	fmt.Println("   1. 开仓订单执行后，系统会自动取消对应的TP/SL订单")
	fmt.Println("   2. Bracket订单会被标记为closed状态")
	fmt.Println("   3. 条件订单状态会更新为cancelled")
	fmt.Println("   4. Order-Sync会定期执行此逻辑")

	fmt.Println("\n🎯 验证方法:")
	fmt.Println("   1. 等待下一次Order-Sync执行（每分钟一次）")
	fmt.Println("   2. 或者手动触发Order-Sync")
	fmt.Println("   3. 检查活跃条件订单数量是否为0")
	fmt.Println("   4. 检查Bracket订单状态是否为closed")

	fmt.Println("\n🎉 XNYUSDT Bracket联动取消修复验证完成！")
}