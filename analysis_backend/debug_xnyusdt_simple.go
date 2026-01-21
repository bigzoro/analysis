package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 XNYUSDT Bracket订单问题排查")

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

	// 1. 检查XNYUSDT Bracket订单状态
	fmt.Println("\n1️⃣ XNYUSDT Bracket订单状态")
	var xnyusdtBrackets []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ?", "XNYUSDT").Order("created_at DESC").Limit(5).Find(&xnyusdtBrackets).Error
	if err != nil {
		log.Printf("查询失败: %v", err)
		return
	}

	for _, bracket := range xnyusdtBrackets {
		fmt.Printf("Bracket %s - 状态:%s\n", bracket.GroupID, bracket.Status)
		fmt.Printf("  开仓:%s, TP:%s, SL:%s\n", bracket.EntryClientID, bracket.TPClientID, bracket.SLClientID)

		// 检查各订单状态
		checkOrderStatus(gdb, bracket.EntryClientID, "开仓")
		checkOrderStatus(gdb, bracket.TPClientID, "止盈")
		checkOrderStatus(gdb, bracket.SLClientID, "止损")
		fmt.Println()
	}

	// 2. 检查活跃的条件订单
	fmt.Println("\n2️⃣ 检查活跃的条件订单")
	var activeConditionalOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type IN (?) AND status IN (?)",
		"XNYUSDT",
		[]string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
		[]string{"new", "processing", "pending"}).Find(&activeConditionalOrders).Error

	if err != nil {
		log.Printf("查询活跃条件订单失败: %v", err)
	} else {
		fmt.Printf("活跃条件订单数量: %d\n", len(activeConditionalOrders))
		for _, order := range activeConditionalOrders {
			fmt.Printf("  %s %s - 状态:%s (ID:%d)\n",
				order.OrderType, order.Side, order.Status, order.ID)
		}
	}

	// 3. 检查最近的取消记录
	fmt.Println("\n3️⃣ 检查最近的订单取消记录")
	var cancelledOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND status = ? AND updated_at >= DATE_SUB(NOW(), INTERVAL 30 MINUTE)",
		"XNYUSDT", "cancelled").Order("updated_at DESC").Find(&cancelledOrders).Error

	if err != nil {
		log.Printf("查询取消订单失败: %v", err)
	} else {
		fmt.Printf("最近30分钟取消的订单: %d个\n", len(cancelledOrders))
		for _, order := range cancelledOrders {
			fmt.Printf("  %s - %s (更新时间:%s)\n",
				order.OrderType, order.ClientOrderId,
				order.UpdatedAt.Format("15:04:05"))
		}
	}
}

func checkOrderStatus(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("  %s: 空\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("  %s: 查询失败\n", orderType)
		return
	}

	status := order.Status
	if order.Status == "filled" || order.Status == "executed" {
		status += " ✅"
	} else if order.Status == "cancelled" {
		status += " ❌"
	} else if order.Status == "new" || order.Status == "processing" {
		status += " ⏳"
	}

	fmt.Printf("  %s: %s\n", orderType, status)
}