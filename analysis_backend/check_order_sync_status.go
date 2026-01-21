package main

import (
	"fmt"
	"log"

	db "analysis/internal/db"
)

func main() {
	fmt.Println("=== 检查订单同步状态 ===")

	// 连接数据库
	database, err := db.OpenMySQL(db.Options{
		DSN:         "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate: false,
	})
	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
	}
	defer database.Close()

	gdb, err := database.DB()
	if err != nil {
		log.Fatalf("获取数据库实例失败: %v", err)
	}

	// 检查用户提供的订单状态
	orderClientIds := []string{
		"sch-1532-768961283-sl", // HANAUSDT 止损订单
		"sch-1534-768961284-tp", // ARCUSDT 止盈订单
		"sch-1531-768961289-sl", // NAORISUSDT 止损订单
	}

	fmt.Println("\n📊 检查订单状态:")
	for _, clientId := range orderClientIds {
		var order db.ScheduledOrder
		err := gdb.Where("client_order_id = ?", clientId).First(&order).Error
		if err != nil {
			fmt.Printf("❌ 订单 %s 未找到: %v\n", clientId, err)
			continue
		}

		fmt.Printf("\n订单 %s:\n", clientId)
		fmt.Printf("  ID: %d\n", order.ID)
		fmt.Printf("  状态: %s\n", order.Status)
		fmt.Printf("  类型: %s\n", order.OrderType)
		fmt.Printf("  交易所: %s\n", order.Exchange)
		fmt.Printf("  结果: %s\n", order.Result)

		// 检查这个订单是否会被syncAllOrderStatus查询到
		wouldBeSynced := false
		if order.Status == "success" || order.Status == "processing" {
			if order.ClientOrderId != "" && order.Exchange == "binance_futures" {
				if order.OrderType == "TAKE_PROFIT_MARKET" || order.OrderType == "STOP_MARKET" {
					wouldBeSynced = true
				}
			}
		}

		if wouldBeSynced {
			fmt.Printf("  ❌ 会被syncAllOrderStatus查询 (条件订单 + 活跃状态)\n")
		} else {
			fmt.Printf("  ✅ 不会被syncAllOrderStatus查询\n")
		}
	}

	// 检查有多少条件订单仍在活跃状态
	fmt.Println("\n📈 检查活跃条件订单统计:")
	var activeConditionalOrders []db.ScheduledOrder
	err = gdb.Where("status IN (?) AND order_type IN (?) AND exchange = ? AND client_order_id != ''",
		[]string{"success", "processing"},
		[]string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
		"binance_futures").Find(&activeConditionalOrders).Error

	if err != nil {
		log.Printf("查询活跃条件订单失败: %v", err)
	} else {
		fmt.Printf("活跃条件订单数量: %d\n", len(activeConditionalOrders))
		for _, order := range activeConditionalOrders {
			fmt.Printf("  - %s (%s): %s\n", order.ClientOrderId, order.Symbol, order.Status)
		}
	}

	fmt.Println("\n🎯 问题诊断:")
	fmt.Println("如果订单状态仍然是'success'或'processing'，")
	fmt.Println("syncAllOrderStatus会继续查询这些FINISHED状态的订单，")
	fmt.Println("造成不必要的API调用和资源浪费。")

	fmt.Println("\n💡 解决方案:")
	fmt.Println("1. 确保handleBracketOrderClosure正确更新TP/SL订单状态为'filled'")
	fmt.Println("2. 添加状态修复脚本，将已完成订单的状态更新")
	fmt.Println("3. 优化syncAllOrderStatus，跳过已知完成的订单")
}