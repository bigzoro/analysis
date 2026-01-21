package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 排查XNYUSDT Bracket订单取消问题")
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

	// 1. 检查XNYUSDT的所有Bracket订单
	fmt.Println("\n1️⃣ 检查XNYUSDT的所有Bracket订单")
	var xnyusdtBrackets []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ?", "XNYUSDT").Order("created_at DESC").Find(&xnyusdtBrackets).Error
	if err != nil {
		log.Printf("查询XNYUSDT Bracket订单失败: %v", err)
		return
	}

	fmt.Printf("找到%d个XNYUSDT Bracket订单:\n", len(xnyusdtBrackets))

	for i, bracket := range xnyusdtBrackets {
		fmt.Printf("\n%d. Bracket订单 %s (状态: %s)\n", i+1, bracket.GroupID, bracket.Status)
		fmt.Printf("   创建时间: %s\n", bracket.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("   开仓订单: %s\n", bracket.EntryClientID)
		fmt.Printf("   止盈订单: %s\n", bracket.TPClientID)
		fmt.Printf("   止损订单: %s\n", bracket.SLClientID)

		// 检查各个订单的详细信息
		checkOrderDetails(gdb, bracket.EntryClientID, "开仓")
		checkOrderDetails(gdb, bracket.TPClientID, "止盈")
		checkOrderDetails(gdb, bracket.SLClientID, "止损")
	}

	// 2. 检查最近的XNYUSDT订单历史
	fmt.Println("\n2️⃣ 检查最近的XNYUSDT订单历史")
	var recentOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND created_at >= DATE_SUB(NOW(), INTERVAL 1 HOUR)", "XNYUSDT").
		Order("created_at DESC").Find(&recentOrders).Error
	if err != nil {
		log.Printf("查询XNYUSDT订单历史失败: %v", err)
	} else {
		fmt.Printf("最近1小时内的XNYUSDT订单: %d个\n", len(recentOrders))
		// 只显示关键的已完成或取消的订单
		completedOrders := 0
		for _, order := range recentOrders {
			if order.Status == "filled" || order.Status == "cancelled" || order.Status == "executed" {
				completedOrders++
				if completedOrders <= 5 { // 只显示前5个
					fmt.Printf("   %s %s - %s (ID:%d, ClientID:%s)\n",
						order.OrderType, order.Side, order.Status, order.ID, order.ClientOrderId)
					if order.Result != "" {
						fmt.Printf("      结果: %s\n", order.Result)
					}
				}
			}
		}
		fmt.Printf("   总计: %d个已完成/取消订单\n", completedOrders)
	}

	// 3. 检查是否有未关闭的Bracket订单
	fmt.Println("\n3️⃣ 检查活跃的Bracket订单状态")
	var activeBrackets []pdb.BracketLink
	err = gdb.GormDB().Where("status = ?", "active").Find(&activeBrackets).Error
	if err != nil {
		log.Printf("查询活跃Bracket订单失败: %v", err)
	} else {
		fmt.Printf("活跃Bracket订单数量: %d\n", len(activeBrackets))
		for _, bracket := range activeBrackets {
			fmt.Printf("   %s - %s (开仓:%s, TP:%s, SL:%s)\n",
				bracket.Symbol, bracket.GroupID,
				bracket.EntryClientID, bracket.TPClientID, bracket.SLClientID)
		}
	}
}

func checkOrderDetails(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("   %s订单: (空)\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("   %s订单: 查询失败 - %v\n", orderType, err)
		return
	}

	// 简化的关键信息显示
	statusEmoji := ""
	switch order.Status {
	case "filled", "executed":
		statusEmoji = "✅"
	case "cancelled":
		statusEmoji = "❌"
	case "pending", "processing", "new":
		statusEmoji = "⏳"
	default:
		statusEmoji = "❓"
	}

	fmt.Printf("   %s订单: %s %s (ID:%d, 时间:%s)\n",
		orderType, statusEmoji, order.Status, order.ID,
		order.CreatedAt.Format("15:04:05"))

	if order.Status == "filled" || order.Status == "executed" {
		fmt.Printf("      执行数量: %s, 平均价格: %s\n", order.ExecutedQty, order.AvgPrice)
	}

	if order.Result != "" {
		fmt.Printf("      结果: %s\n", order.Result)
	}

	// Bracket相关关键信息
	if order.BracketEnabled {
		fmt.Printf("      Bracket订单 - TP:%.2f%% SL:%.2f%%\n", order.ActualTPPercent, order.ActualSLPercent)
	}
}