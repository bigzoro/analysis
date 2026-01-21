package main

import (
	"fmt"
	"log"

	"analysis/internal/db"
)

func main() {
	fmt.Println("=== 检查Bracket订单状态 ===")

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

	// 检查用户提供的订单对应的BracketLink状态
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
		fmt.Printf("  结果: %s\n", order.Result)

		// 查找对应的BracketLink
		var bracket db.BracketLink
		err = gdb.Where("sl_client_id = ? OR tp_client_id = ?", clientId, clientId).First(&bracket).Error
		if err != nil {
			fmt.Printf("  ❌ BracketLink未找到: %v\n", err)
		} else {
			fmt.Printf("  BracketLink ID: %d\n", bracket.ID)
			fmt.Printf("  Bracket状态: %s\n", bracket.Status)
			fmt.Printf("  GroupID: %s\n", bracket.GroupID)
			fmt.Printf("  Symbol: %s\n", bracket.Symbol)
			fmt.Printf("  TP订单: %s\n", bracket.TPClientID)
			fmt.Printf("  SL订单: %s\n", bracket.SLClientID)
		}
	}

	// 检查活跃的Bracket订单数量
	var activeBrackets []db.BracketLink
	err = gdb.Where("status = ?", "active").Find(&activeBrackets).Error
	if err != nil {
		log.Printf("查询活跃Bracket订单失败: %v", err)
	} else {
		fmt.Printf("\n📈 当前活跃Bracket订单数量: %d\n", len(activeBrackets))
		for _, bracket := range activeBrackets {
			fmt.Printf("  - %s (%s): TP=%s, SL=%s\n",
				bracket.GroupID, bracket.Symbol, bracket.TPClientID, bracket.SLClientID)
		}
	}

	// 检查已关闭的Bracket订单数量
	var closedBrackets []db.BracketLink
	err = gdb.Where("status = ?", "closed").Find(&closedBrackets).Error
	if err != nil {
		log.Printf("查询已关闭Bracket订单失败: %v", err)
	} else {
		fmt.Printf("\n📉 当前已关闭Bracket订单数量: %d\n", len(closedBrackets))
	}

	fmt.Println("\n🎯 问题分析:")
	fmt.Println("如果订单已执行完成但BracketLink状态仍为'active'，")
	fmt.Println("系统会继续查询这些订单，造成API资源浪费。")

	fmt.Println("\n💡 解决方案:")
	fmt.Println("1. 检查handleBracketOrderClosure是否正确执行")
	fmt.Println("2. 添加状态修复脚本，将已完成订单的BracketLink设为closed")
	fmt.Println("3. 优化syncBracketOrders逻辑，避免重复查询")
}