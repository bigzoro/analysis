package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 详细分析XNYUSDT Bracket订单问题")
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

	// 1. 检查XNYUSDT的所有Bracket订单
	fmt.Println("\n1️⃣ 检查XNYUSDT的所有Bracket订单")
	var allXNYUSDTBrackets []pdb.BracketLink
	err = gdb.GormDB().Where("symbol = ?", "XNYUSDT").Find(&allXNYUSDTBrackets).Error
	if err != nil {
		fmt.Printf("❌ 查询XNYUSDT Bracket订单失败: %v\n", err)
		return
	}

	if len(allXNYUSDTBrackets) == 0 {
		fmt.Println("❌ 没有找到任何XNYUSDT Bracket订单")
		return
	}

	fmt.Printf("找到%d个XNYUSDT Bracket订单:\n", len(allXNYUSDTBrackets))

	statusCount := make(map[string]int)
	for _, bracket := range allXNYUSDTBrackets {
		statusCount[bracket.Status]++
	}

	for status, count := range statusCount {
		fmt.Printf("   %s: %d个\n", status, count)
	}

	// 找到closed状态的Bracket订单（应该是我们修复的结果）
	var closedBracket pdb.BracketLink
	for _, bracket := range allXNYUSDTBrackets {
		if bracket.Status == "closed" {
			closedBracket = bracket
			break
		}
	}

	if closedBracket.ID == 0 {
		fmt.Println("❌ 没有找到closed状态的Bracket订单")
		return
	}

	fmt.Printf("\n📋 分析已关闭的Bracket订单: ID=%d, GroupID=%s\n", closedBracket.ID, closedBracket.GroupID)
	fmt.Printf("   开仓订单ID: %s\n", closedBracket.EntryClientID)
	fmt.Printf("   止盈订单ID: %s\n", closedBracket.TPClientID)
	fmt.Printf("   止损订单ID: %s\n", closedBracket.SLClientID)

	activeBracket := closedBracket

	fmt.Printf("📋 活跃Bracket订单: ID=%d, GroupID=%s\n", activeBracket.ID, activeBracket.GroupID)
	fmt.Printf("   开仓订单ID: %s\n", activeBracket.EntryClientID)
	fmt.Printf("   止盈订单ID: %s\n", activeBracket.TPClientID)
	fmt.Printf("   止损订单ID: %s\n", activeBracket.SLClientID)

	// 2. 检查开仓订单的详细信息
	fmt.Println("\n2️⃣ 检查开仓订单详细信息")
	var entryOrder pdb.ScheduledOrder
	err = gdb.GormDB().Where("client_order_id = ?", activeBracket.EntryClientID).First(&entryOrder).Error
	if err != nil {
		fmt.Printf("❌ 开仓订单查询失败: %v\n", err)
		fmt.Printf("   这可能是导致Bracket同步失败的原因！\n")
		return
	}

	fmt.Printf("🏠 开仓订单详情:\n")
	fmt.Printf("   订单ID: %d\n", entryOrder.ID)
	fmt.Printf("   客户端订单ID: %s\n", entryOrder.ClientOrderId)
	fmt.Printf("   订单类型: %s\n", entryOrder.OrderType)
	fmt.Printf("   状态: %s\n", entryOrder.Status)
	fmt.Printf("   数量: %s\n", entryOrder.Quantity)
	fmt.Printf("   价格: %s\n", entryOrder.Price)
	fmt.Printf("   交易所订单ID: %s\n", entryOrder.ExchangeOrderId)
	fmt.Printf("   创建时间: %s\n", entryOrder.CreatedAt.Format("2006-01-02 15:04:05"))

	// 3. 检查这个订单是否真的已执行
	fmt.Println("\n3️⃣ 订单执行状态分析")
	isExecuted := false
	if entryOrder.Status == "filled" {
		isExecuted = true
		fmt.Println("✅ 开仓订单状态为filled - 已执行")
	} else if entryOrder.Status == "executed" {
		isExecuted = true
		fmt.Println("✅ 开仓订单状态为executed - 已执行")
	} else if entryOrder.ExecutedQty != "" && entryOrder.ExecutedQty != "0" {
		isExecuted = true
		fmt.Printf("✅ 开仓订单已部分执行: %s\n", entryOrder.ExecutedQty)
	} else {
		fmt.Printf("❌ 开仓订单未执行: 状态=%s, 执行数量=%s\n", entryOrder.Status, entryOrder.ExecutedQty)
	}

	// 4. 检查条件订单状态
	fmt.Println("\n4️⃣ 检查条件订单状态")
	checkConditionalOrderDetail(gdb, activeBracket.TPClientID, "止盈")
	checkConditionalOrderDetail(gdb, activeBracket.SLClientID, "止损")

	// 5. 分析Bracket同步逻辑
	fmt.Println("\n5️⃣ Bracket同步逻辑分析")
	fmt.Printf("开仓订单已执行: %v\n", isExecuted)

	if isExecuted {
		fmt.Println("✅ 应该触发: 开仓执行分支")
		fmt.Println("   - 取消TP订单")
		fmt.Println("   - 取消SL订单")
		fmt.Println("   - 标记Bracket为closed")
		fmt.Println("   - 跳过触发检查")
	} else {
		fmt.Println("❌ 应该执行: 触发检查分支")
		fmt.Println("   - 检查TP是否触发")
		fmt.Println("   - 检查SL是否触发")
		fmt.Println("   - 如果触发则关闭Bracket")
	}

	// 6. 总结统计信息
	fmt.Println("\n6️⃣ 总结统计信息")
	fmt.Printf("XNYUSDT Bracket订单统计:\n")

	statusCount := make(map[string]int)
	for _, bracket := range allXNYUSDTBrackets {
		statusCount[bracket.Status]++
	}

	for status, count := range statusCount {
		fmt.Printf("   %s: %d个\n", status, count)
	}

	// 7. 总结问题
	fmt.Println("\n🎯 问题诊断总结")

	activeConditionalCount := 0
	if isOrderActive(gdb, activeBracket.TPClientID) {
		activeConditionalCount++
	}
	if isOrderActive(gdb, activeBracket.SLClientID) {
		activeConditionalCount++
	}

	fmt.Printf("活跃条件订单数量: %d\n", activeConditionalCount)

	if isExecuted && activeConditionalCount > 0 {
		fmt.Println("❌ 问题确认: 开仓已执行但条件订单仍活跃")
		fmt.Println("💡 可能原因:")
		fmt.Println("   1. Bracket同步逻辑未正确执行开仓分支")
		fmt.Println("   2. cancelConditionalOrderIfNeeded函数有问题")
		fmt.Println("   3. 取消API调用失败但未正确处理")
		fmt.Println("   4. 数据库状态更新失败")

		fmt.Println("\n🔧 建议修复:")
		fmt.Println("   1. 检查Bracket同步日志")
		fmt.Println("   2. 验证cancelConditionalOrderIfNeeded函数")
		fmt.Println("   3. 手动测试条件订单取消")
	} else if !isExecuted && activeConditionalCount > 0 {
		fmt.Println("ℹ️ 情况正常: 开仓未执行，条件订单等待触发")
	} else if isExecuted && activeConditionalCount == 0 {
		fmt.Println("✅ 情况正常: 开仓已执行，条件订单已清理")
	}
}

func checkConditionalOrderDetail(gdb pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("   %s订单: (空)\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("   ❌ %s订单查询失败: %v\n", orderType, err)
		return
	}

	fmt.Printf("   %s订单详情:\n", orderType)
	fmt.Printf("      ID: %d\n", order.ID)
	fmt.Printf("      客户端ID: %s\n", order.ClientOrderId)
	fmt.Printf("      类型: %s\n", order.OrderType)
	fmt.Printf("      状态: %s\n", order.Status)
	fmt.Printf("      数量: %s\n", order.Quantity)
	// 对于条件订单，显示TP/SL价格
	if order.OrderType == "TAKE_PROFIT_MARKET" {
		fmt.Printf("      止盈价格: %s\n", order.TPPrice)
	} else if order.OrderType == "STOP_MARKET" {
		fmt.Printf("      止损价格: %s\n", order.SLPrice)
	}
	fmt.Printf("      执行数量: %s\n", order.ExecutedQty)
	fmt.Printf("      平均价格: %s\n", order.AvgPrice)

	if order.Status == "success" || order.Status == "new" {
		fmt.Printf("      ⚠️  状态表明订单仍活跃\n")
	} else if order.Status == "cancelled" {
		fmt.Printf("      ✅ 订单已取消\n")
	} else if order.Status == "filled" || order.Status == "executed" {
		fmt.Printf("      ✅ 订单已执行\n")
	}
}

func isOrderActive(gdb pdb.Database, clientOrderId string) bool {
	if clientOrderId == "" {
		return false
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		return false
	}

	return order.Status == "success" || order.Status == "new" || order.Status == "processing"
}