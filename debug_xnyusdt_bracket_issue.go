package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 深入分析XNYUSDT Bracket联动取消问题")
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

	// 1. 检查XNYUSDT的活跃Bracket订单
	fmt.Println("\n1️⃣ 检查XNYUSDT的活跃Bracket订单")
	var bracketLinks []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ? AND status = ?", "XNYUSDT", "active").Find(&bracketLinks).Error
	if err != nil {
		log.Printf("查询BracketLink失败: %v", err)
		return
	}

	fmt.Printf("找到 %d 个活跃的XNYUSDT Bracket订单:\n", len(bracketLinks))

	for i, link := range bracketLinks {
		fmt.Printf("\n--- Bracket订单 %d ---\n", i+1)
		fmt.Printf("ID: %d\n", link.ID)
		fmt.Printf("GroupID: %s\n", link.GroupID)
		fmt.Printf("状态: %s\n", link.Status)
		fmt.Printf("开仓订单ID: %s\n", link.EntryClientID)
		fmt.Printf("止盈订单ID: %s\n", link.TPClientID)
		fmt.Printf("止损订单ID: %s\n", link.SLClientID)
		fmt.Printf("创建时间: %s\n", link.CreatedAt.Format("2006-01-02 15:04:05"))

		// 检查每个订单的详细状态
		checkOrderDetails(gdb, link.EntryClientID, "开仓订单")
		checkOrderDetails(gdb, link.TPClientID, "止盈订单")
		checkOrderDetails(gdb, link.SLClientID, "止损订单")

		// 分析联动取消逻辑
		analyzeCancellationLogic(gdb, link)
	}

	// 2. 检查最近的XNYUSDT订单执行历史
	fmt.Println("\n2️⃣ 检查最近的XNYUSDT订单执行历史")
	var recentOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND created_at > DATE_SUB(NOW(), INTERVAL 1 HOUR)",
		"XNYUSDT").Order("created_at DESC").Find(&recentOrders).Error

	if err != nil {
		log.Printf("查询最近订单失败: %v", err)
	} else {
		fmt.Printf("最近1小时内的XNYUSDT订单 (%d个):\n", len(recentOrders))
		for _, order := range recentOrders {
			fmt.Printf("  %s | %s | %s | 数量:%s | 状态:%s | 时间:%s\n",
				order.ClientOrderId, order.OrderType, order.Side,
				order.Quantity, order.Status,
				order.CreatedAt.Format("15:04:05"))
		}
	}

	// 3. 检查是否有未正确取消的订单
	fmt.Println("\n3️⃣ 检查未正确取消的条件订单")
	var activeConditionalOrders []pdb.ScheduledOrder
	err = gdb.GormDB().Where("symbol = ? AND order_type IN (?) AND status NOT IN (?)",
		"XNYUSDT", []string{"TAKE_PROFIT_MARKET", "STOP_MARKET"},
		[]string{"cancelled", "filled", "executed"}).Find(&activeConditionalOrders).Error

	if err != nil {
		log.Printf("查询活跃条件订单失败: %v", err)
	} else {
		fmt.Printf("活跃的XNYUSDT条件订单 (%d个):\n", len(activeConditionalOrders))
		for _, order := range activeConditionalOrders {
			fmt.Printf("  ❌ %s | %s | 状态:%s | 关联订单:%s\n",
				order.ClientOrderId, order.OrderType, order.Status,
				order.ParentOrderId)
		}
	}

	// 4. 检查同步日志中可能的错误
	fmt.Println("\n4️⃣ 检查可能的问题模式")

	// 检查是否有Bracket订单的状态不一致
	fmt.Println("检查Bracket订单状态一致性...")
	for _, link := range bracketLinks {
		entryExecuted := isOrderExecuted(gdb, link.EntryClientID)
		tpExecuted := isOrderExecuted(gdb, link.TPClientID)
		slExecuted := isOrderExecuted(gdb, link.SLClientID)

		if entryExecuted && (tpExecuted || slExecuted) {
			fmt.Printf("⚠️  Bracket订单 %s 可能存在联动取消问题:\n", link.GroupID)
			fmt.Printf("   开仓: ✅ 已执行\n")
			fmt.Printf("   止盈: %s\n", executionStatus(tpExecuted))
			fmt.Printf("   止损: %s\n", executionStatus(slExecuted))

			if tpExecuted && slExecuted {
				fmt.Printf("   ❌ 问题: 止盈和止损都已执行，这不应该发生\n")
			} else if tpExecuted {
				fmt.Printf("   ⚠️  止盈已执行，止损应该被取消但可能未成功\n")
			} else if slExecuted {
				fmt.Printf("   ⚠️  止损已执行，止盈应该被取消但可能未成功\n")
			}
		}
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

	fmt.Printf("   %s %s: 状态=%s, 执行数量=%s, 平均价格=%s\n",
		orderType, clientOrderId, order.Status, order.ExecutedQty, order.AvgPrice)

	// 检查关联关系
	if order.ParentOrderId != "" {
		fmt.Printf("      父订单: %s\n", order.ParentOrderId)
	}
}

func analyzeCancellationLogic(gdb pdb.Database, link pdb.BracketLink) {
	fmt.Printf("\n   🔍 联动取消分析:\n")

	entryExecuted := isOrderExecuted(gdb, link.EntryClientID)
	tpExecuted := isOrderExecuted(gdb, link.TPClientID)
	slExecuted := isOrderExecuted(gdb, link.SLClientID)

	fmt.Printf("   开仓订单已执行: %v\n", entryExecuted)
	fmt.Printf("   止盈订单已执行: %v\n", tpExecuted)
	fmt.Printf("   止损订单已执行: %v\n", slExecuted)

	if entryExecuted {
		fmt.Printf("   ✅ 开仓已执行 -> 应该取消: TP(%s), SL(%s)\n",
			link.TPClientID, link.SLClientID)
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

	return order.Status == "filled" || order.Status == "executed" ||
		   (order.ExecutedQty != "" && order.ExecutedQty != "0")
}

func checkCancellationStatus(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("      ❌ %s订单ID为空\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("      ❌ %s订单 %s 查询失败: %v\n", orderType, clientOrderId, err)
		return
	}

	if order.Status == "cancelled" {
		fmt.Printf("      ✅ %s订单 %s 已正确取消\n", orderType, clientOrderId)
	} else {
		fmt.Printf("      ❌ %s订单 %s 未被取消 (状态: %s)\n", orderType, clientOrderId, order.Status)
	}
}

func executionStatus(executed bool) string {
	if executed {
		return "✅ 已执行"
	}
	return "❌ 未执行"
}