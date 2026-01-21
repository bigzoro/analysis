package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 深入分析FHEUSDT Bracket订单取消问题")

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

	// 查找包含订单1289的Bracket
	fmt.Println("\n1️⃣ 查找包含订单1289的Bracket")
	var bracket pdb.BracketLink
	err = gdb.GormDB().Where("entry_client_id = ?", "sch-1289-entry-768884458").First(&bracket).Error
	if err != nil {
		log.Printf("查询Bracket失败: %v", err)
		return
	}

	fmt.Printf("找到Bracket订单: %s\n", bracket.GroupID)
	fmt.Printf("状态: %s\n", bracket.Status)
	fmt.Printf("开仓: %s\n", bracket.EntryClientID)
	fmt.Printf("止盈: %s\n", bracket.TPClientID)
	fmt.Printf("止损: %s\n", bracket.SLClientID)

	// 检查各个订单的状态
	fmt.Println("\n2️⃣ 检查各个订单的当前状态")

	checkOrderDetail(gdb, bracket.EntryClientID, "开仓")
	checkOrderDetail(gdb, bracket.TPClientID, "止盈")
	checkOrderDetail(gdb, bracket.SLClientID, "止损")

	// 检查操作日志
	fmt.Println("\n3️⃣ 检查操作日志")
	var logs []pdb.OperationLog
	err = gdb.GormDB().Where("entity_type = ? AND entity_id IN (?)",
		"order", []uint{1289, 1295}).Order("created_at DESC").Limit(10).Find(&logs).Error
	if err != nil {
		log.Printf("查询日志失败: %v", err)
	} else {
		fmt.Printf("找到%d条相关日志:\n", len(logs))
		for _, logEntry := range logs {
			fmt.Printf("  %s [%s] %s: %s\n",
				logEntry.CreatedAt.Format("15:04:05"),
				logEntry.Level,
				logEntry.Action,
				logEntry.Description)
		}
	}

	// 检查系统日志中的Bracket相关信息
	fmt.Println("\n4️⃣ 检查系统运行日志（最近的Bracket相关日志）")
	// 这里我们可以通过时间范围来查找相关的日志
	fmt.Println("注意：需要检查系统日志中是否有Bracket同步的相关记录")
	fmt.Println("可能的日志关键词：")
	fmt.Println("- '[Bracket-Closure]' - Bracket关闭时的日志")
	fmt.Println("- '[Order-Sync]' - 订单同步时的日志")
	fmt.Println("- 'cancelConditionalOrderIfNeeded' - 取消条件订单的日志")

	// 检查Bracket关闭的时间
	fmt.Println("\n5️⃣ 分析Bracket关闭的时间线")
	fmt.Printf("Bracket创建时间: %s\n", bracket.CreatedAt.Format("2006-01-02 15:04:05"))

	// 检查开仓和平仓的时间
	var entryOrder, closeOrder pdb.ScheduledOrder
	gdb.GormDB().Where("id = ?", 1289).First(&entryOrder)
	gdb.GormDB().Where("id = ?", 1295).First(&closeOrder)

	fmt.Printf("开仓订单时间: %s\n", entryOrder.CreatedAt.Format("2006-01-02 15:04:05"))
	fmt.Printf("平仓订单时间: %s\n", closeOrder.CreatedAt.Format("2006-01-02 15:04:05"))

	// 分析可能的取消逻辑
	fmt.Println("\n6️⃣ 分析取消逻辑")
	fmt.Println("根据代码逻辑，当检测到条件订单触发时，应该：")
	fmt.Println("1. 调用handleBracketOrderClosure")
	fmt.Println("2. 在该函数中取消另一方向的条件订单")
	fmt.Println("3. 更新Bracket状态为closed")

	fmt.Println("\n当前状态分析：")
	if bracket.Status == "closed" {
		fmt.Println("✅ Bracket状态为closed - 关闭流程已执行")
	} else {
		fmt.Println("❌ Bracket状态不是closed - 关闭流程未执行")
	}

	// 检查是否有取消记录
	var cancelledTP, cancelledSL pdb.ScheduledOrder
	hasTPCancelled := false
	hasSLCancelled := false

	if bracket.TPClientID != "" {
		gdb.GormDB().Where("client_order_id = ?", bracket.TPClientID).First(&cancelledTP)
		if cancelledTP.Status == "cancelled" {
			hasTPCancelled = true
		}
	}

	if bracket.SLClientID != "" {
		gdb.GormDB().Where("client_order_id = ?", bracket.SLClientID).First(&cancelledSL)
		if cancelledSL.Status == "cancelled" {
			hasSLCancelled = true
		}
	}

	fmt.Printf("止盈订单已取消: %v\n", hasTPCancelled)
	fmt.Printf("止损订单已取消: %v\n", hasSLCancelled)

	fmt.Println("\n7️⃣ 诊断结论")
	if bracket.Status == "closed" {
		if hasTPCancelled && hasSLCancelled {
			fmt.Println("✅ 取消逻辑工作正常：两个条件订单都被取消了")
		} else if hasTPCancelled || hasSLCancelled {
			fmt.Println("⚠️ 部分取消：只有一个方向的订单被取消")
			if hasTPCancelled {
				fmt.Println("   - 止盈订单已取消，止损订单未取消")
			} else {
				fmt.Println("   - 止损订单已取消，止盈订单未取消")
			}
		} else {
			fmt.Println("❌ 取消失败：两个条件订单都没有被取消")
			fmt.Println("   可能原因：")
			fmt.Println("   1. handleBracketOrderClosure函数没有被调用")
			fmt.Println("   2. cancelConditionalOrderIfNeeded函数执行失败")
			fmt.Println("   3. syncBracketOrders没有检测到条件订单触发")
		}
	} else {
		fmt.Println("❌ Bracket尚未关闭，取消逻辑还未执行")
	}
}

func checkOrderDetail(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("  %s订单: (空)\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("  %s订单: 查询失败 - %v\n", orderType, err)
		return
	}

	statusEmoji := ""
	switch order.Status {
	case "filled", "executed", "success":
		statusEmoji = "✅"
	case "cancelled":
		statusEmoji = "❌"
	case "new", "processing":
		statusEmoji = "⏳"
	default:
		statusEmoji = "❓"
	}

	fmt.Printf("  %s订单: %s %s (ID:%d, 时间:%s)\n",
		orderType, statusEmoji, order.Status, order.ID,
		order.CreatedAt.Format("15:04:05"))

	if order.Status == "cancelled" {
		fmt.Printf("    取消时间: %s\n", order.UpdatedAt.Format("15:04:05"))
	}

	if order.Result != "" {
		fmt.Printf("    结果: %s\n", order.Result)
	}
}