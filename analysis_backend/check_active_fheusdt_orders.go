package main

import (
	pdb "analysis/internal/db"
	"fmt"
	"log"
)

func main() {
	fmt.Println("🔍 检查FHEUSDT活跃的条件订单")

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

	// 1. 检查数据库中活跃的条件订单
	fmt.Println("\n1️⃣ 检查数据库中FHEUSDT的活跃条件订单")
	var activeOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type IN (?) AND status IN (?)",
		"FHEUSDT",
		[]string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
		[]string{"new", "processing", "pending", "success"}).Find(&activeOrders).Error

	if err != nil {
		log.Printf("查询活跃条件订单失败: %v", err)
	} else {
		fmt.Printf("数据库中活跃的FHEUSDT条件订单数量: %d\n", len(activeOrders))
		for i, order := range activeOrders {
			fmt.Printf("  %d. %s %s - %s (ID:%d, ClientID:%s)\n",
				i+1, order.OrderType, order.Side, order.Status, order.ID, order.ClientOrderId)
			fmt.Printf("     创建时间: %s\n", order.CreatedAt.Format("15:04:05"))
			if order.Status == "success" {
				fmt.Printf("     ⚠️  这个订单在数据库中显示为success状态！\n")
			}
		}
	}

	// 2. 检查所有FHEUSDT的条件订单（包括已取消的）
	fmt.Println("\n2️⃣ 检查所有FHEUSDT的条件订单")
	var allConditionalOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type IN (?)",
		"FHEUSDT",
		[]string{"TAKE_PROFIT_MARKET", "STOP_MARKET"}).Order("created_at DESC").Limit(20).Find(&allConditionalOrders).Error

	if err != nil {
		log.Printf("查询所有条件订单失败: %v", err)
	} else {
		fmt.Printf("最近20个FHEUSDT条件订单:\n")
		for i, order := range allConditionalOrders {
			statusEmoji := ""
			switch order.Status {
			case "filled", "executed", "success":
				statusEmoji = "✅"
			case "cancelled":
				statusEmoji = "❌"
			case "new", "processing", "pending":
				statusEmoji = "⏳"
			default:
				statusEmoji = "❓"
			}

			fmt.Printf("  %d. %s %s - %s %s (ID:%d)\n",
				i+1, order.OrderType, order.Side, statusEmoji, order.Status, order.ID)
			fmt.Printf("     ClientID: %s\n", order.ClientOrderId)
			fmt.Printf("     创建时间: %s\n", order.CreatedAt.Format("15:04:05"))
			if order.Status == "cancelled" {
				fmt.Printf("     取消时间: %s\n", order.UpdatedAt.Format("15:04:05"))
			}
			if order.Result != "" {
				fmt.Printf("     结果: %s\n", order.Result)
			}
		}
	}

	// 3. 检查Bracket订单状态
	fmt.Println("\n3️⃣ 检查活跃的Bracket订单")
	var activeBrackets []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ? AND status = ?", "FHEUSDT", "active").Find(&activeBrackets).Error
	if err != nil {
		log.Printf("查询活跃Bracket失败: %v", err)
	} else {
		fmt.Printf("活跃的FHEUSDT Bracket数量: %d\n", len(activeBrackets))
		for _, bracket := range activeBrackets {
			fmt.Printf("  Bracket: %s\n", bracket.GroupID)
			fmt.Printf("    止盈: %s\n", bracket.TPClientID)
			fmt.Printf("    止损: %s\n", bracket.SLClientID)

			// 检查这些订单的状态
			checkBracketOrderStatus(gdb, bracket.TPClientID, "止盈")
			checkBracketOrderStatus(gdb, bracket.SLClientID, "止损")
		}
	}

	// 4. 分析问题
	fmt.Println("\n4️⃣ 问题分析")
	fmt.Println("如果币安网站上还有FHEUSDT的条件委托存在，可能的原因：")

	if len(activeOrders) > 0 {
		fmt.Println("❌ 数据库中仍有活跃的条件订单未被取消")
		for _, order := range activeOrders {
			fmt.Printf("   - 订单ID:%d, ClientID:%s, 状态:%s\n", order.ID, order.ClientOrderId, order.Status)
		}
	} else {
		fmt.Println("✅ 数据库中没有活跃的条件订单")
		fmt.Println("💡 可能的原因：")
		fmt.Println("   1. 取消API调用失败，但数据库状态已更新")
		fmt.Println("   2. 系统日志中可能有取消失败的记录")
		fmt.Println("   3. 币安网站上的订单状态没有及时同步")
		fmt.Println("   4. 存在其他非Bracket相关的条件订单")
	}

	fmt.Println("\n5️⃣ 建议检查项")
	fmt.Println("🔍 请检查以下内容：")
	fmt.Println("   1. 系统运行日志中是否有 'cancelConditionalOrderIfNeeded' 的错误信息")
	fmt.Println("   2. 币安API是否返回了取消失败的响应")
	fmt.Println("   3. 是否有网络或API限流问题")
	fmt.Println("   4. Bracket同步任务是否正常运行")
}

func checkBracketOrderStatus(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("    %s: 空\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("    %s: 查询失败\n", orderType)
		return
	}

	statusEmoji := ""
	switch order.Status {
	case "filled", "executed", "success":
		statusEmoji = "✅"
	case "cancelled":
		statusEmoji = "❌"
	case "new", "processing", "pending":
		statusEmoji = "⏳"
	default:
		statusEmoji = "❓"
	}

	fmt.Printf("    %s: %s %s\n", orderType, statusEmoji, order.Status)
}
