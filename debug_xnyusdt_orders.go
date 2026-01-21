package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 调试XNYUSDT订单联动取消问题")
	fmt.Println("===============================")

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

	// 1. 查找XNYUSDT的Bracket订单
	fmt.Println("\n1️⃣ 查找XNYUSDT的Bracket订单")
	var bracketLinks []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ?", "XNYUSDT").Find(&bracketLinks).Error
	if err != nil {
		log.Printf("查询BracketLink失败: %v", err)
		return
	}

	if len(bracketLinks) == 0 {
		fmt.Println("❌ 没有找到XNYUSDT的Bracket订单")
		return
	}

	fmt.Printf("📋 找到 %d 个XNYUSDT的Bracket订单:\n", len(bracketLinks))

	for i, link := range bracketLinks {
		fmt.Printf("\n%d. BracketLink ID: %d\n", i+1, link.ID)
		fmt.Printf("   GroupID: %s\n", link.GroupID)
		fmt.Printf("   状态: %s\n", link.Status)

		// 检查开仓订单
		checkOrderDetails(gdb, link.EntryClientID, "开仓订单")

		// 检查止盈订单
		checkOrderDetails(gdb, link.TPClientID, "止盈订单")

		// 检查止损订单
		checkOrderDetails(gdb, link.SLClientID, "止损订单")

		// 分析联动取消逻辑
		analyzeCancellationLogic(gdb, link)
	}
}

func checkOrderDetails(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("   %s: (空)\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("   %s %s: ❌ 查询失败 (%v)\n", orderType, clientOrderId, err)
		return
	}

	fmt.Printf("   %s %s: 状态=%s, 类型=%s, 执行数量=%s\n",
		orderType, clientOrderId, order.Status, order.OrderType, order.ExecutedQty)
}

func analyzeCancellationLogic(gdb pdb.Database, link pdb.BracketLink) {
	fmt.Printf("\n   🔍 联动取消分析:\n")

	// 检查是否有任何订单已执行
	entryExecuted := isOrderExecuted(gdb, link.EntryClientID)
	tpExecuted := isOrderExecuted(gdb, link.TPClientID)
	slExecuted := isOrderExecuted(gdb, link.SLClientID)

	fmt.Printf("   开仓订单已执行: %v\n", entryExecuted)
	fmt.Printf("   止盈订单已执行: %v\n", tpExecuted)
	fmt.Printf("   止损订单已执行: %v\n", slExecuted)

	// 分析应该取消的订单
	if entryExecuted {
		fmt.Printf("   ✅ 开仓已执行 -> 应该取消: TP(%s), SL(%s)\n", link.TPClientID, link.SLClientID)
		checkCancellationStatus(gdb, link.TPClientID, "止盈")
		checkCancellationStatus(gdb, link.SLClientID, "止损")
	} else if tpExecuted {
		fmt.Printf("   ✅ 止盈已执行 -> 应该取消: SL(%s)\n", link.SLClientID)
		checkCancellationStatus(gdb, link.SLClientID, "止损")
	} else if slExecuted {
		fmt.Printf("   ✅ 止损已执行 -> 应该取消: TP(%s)\n", link.TPClientID)
		checkCancellationStatus(gdb, link.TPClientID, "止盈")
	} else {
		fmt.Printf("   ⏳ 所有订单都未执行，等待中...\n")
	}
}

func isOrderExecuted(gdb pdb.Database, clientOrderId string) bool {
	if clientOrderId == "" {
		return false
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		return false
	}

	// 普通订单：检查状态为filled
	if order.OrderType == "MARKET" || order.OrderType == "LIMIT" {
		return order.Status == "filled" || (order.ExecutedQty != "" && order.ExecutedQty != "0")
	}

	// 条件订单：检查状态为filled或executed
	if order.OrderType == "TAKE_PROFIT_MARKET" || order.OrderType == "STOP_MARKET" {
		return order.Status == "filled" || order.Status == "executed"
	}

	return false
}

func checkCancellationStatus(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("   ❌ %s订单ID为空\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("   ❌ %s订单 %s 查询失败: %v\n", orderType, clientOrderId, err)
		return
	}

	if order.Status == "cancelled" {
		fmt.Printf("   ✅ %s订单 %s 已正确取消\n", orderType, clientOrderId)
	} else {
		fmt.Printf("   ❌ %s订单 %s 未被取消 (状态: %s)\n", orderType, clientOrderId, order.Status)
	}
}