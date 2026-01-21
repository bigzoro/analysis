package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 调试Bracket订单联动取消")
	fmt.Println("================================")

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

	// 1. 检查BracketLink状态
	fmt.Println("\n1️⃣ 检查BracketLink状态")
	var bracketLinks []pdb.BracketLink
	err = gdb.GormDB().Where("status IN ?", []string{"active", "partial", "closed"}).Limit(10).Find(&bracketLinks).Error
	if err != nil {
		log.Printf("查询BracketLink失败: %v", err)
		return
	}

	if len(bracketLinks) == 0 {
		fmt.Println("❌ 没有找到活跃的Bracket订单")
		return
	}

	fmt.Printf("📋 找到 %d 个Bracket订单:\n", len(bracketLinks))

	for i, link := range bracketLinks {
		fmt.Printf("\n%d. BracketLink ID: %d\n", i+1, link.ID)
		fmt.Printf("   GroupID: %s\n", link.GroupID)
		fmt.Printf("   Symbol: %s\n", link.Symbol)
		fmt.Printf("   状态: %s\n", link.Status)

		// 检查每个订单的状态
		checkOrderStatus(gdb, "开仓订单", link.EntryClientID)
		checkOrderStatus(gdb, "止盈订单", link.TPClientID)
		checkOrderStatus(gdb, "止损订单", link.SLClientID)
	}

	// 2. 检查是否有执行了的订单
	fmt.Println("\n2️⃣ 检查是否有执行了的Bracket订单")
	for _, link := range bracketLinks {
		// 检查开仓订单
		if link.EntryClientID != "" {
			if status := getOrderStatus(gdb, link.EntryClientID); status == "filled" {
				fmt.Printf("✅ 检测到开仓订单已执行: %s\n", link.EntryClientID)
				fmt.Printf("   预期: TP订单(%s)和SL订单(%s)应该被取消\n", link.TPClientID, link.SLClientID)
				checkCancellationStatus(gdb, link.TPClientID, "止盈")
				checkCancellationStatus(gdb, link.SLClientID, "止损")
			}
		}

		// 检查止盈订单
		if link.TPClientID != "" {
			if status := getOrderStatus(gdb, link.TPClientID); status == "filled" {
				fmt.Printf("✅ 检测到止盈订单已执行: %s\n", link.TPClientID)
				fmt.Printf("   预期: SL订单(%s)应该被取消\n", link.SLClientID)
				checkCancellationStatus(gdb, link.SLClientID, "止损")
			}
		}

		// 检查止损订单
		if link.SLClientID != "" {
			if status := getOrderStatus(gdb, link.SLClientID); status == "filled" {
				fmt.Printf("✅ 检测到止损订单已执行: %s\n", link.SLClientID)
				fmt.Printf("   预期: TP订单(%s)应该被取消\n", link.TPClientID)
				checkCancellationStatus(gdb, link.TPClientID, "止盈")
			}
		}
	}

	// 3. 检查联动取消逻辑是否正常
	fmt.Println("\n3️⃣ 检查联动取消逻辑")

	// 模拟订单同步逻辑
	for _, link := range bracketLinks {
		fmt.Printf("\n分析 BracketLink %d (%s):\n", link.ID, link.GroupID)

		// 检查开仓订单
		if link.EntryClientID != "" {
			status := getOrderStatus(gdb, link.EntryClientID)
			fmt.Printf("  开仓订单 %s: %s\n", link.EntryClientID, status)
			if status == "filled" {
				fmt.Printf("  🔴 问题: 开仓订单已执行，但状态仍为 '%s' (应该更新BracketLink)\n", link.Status)
			}
		}

		// 检查止盈订单
		if link.TPClientID != "" {
			status := getOrderStatus(gdb, link.TPClientID)
			fmt.Printf("  止盈订单 %s: %s\n", link.TPClientID, status)
			if link.Status == "partial" && status != "cancelled" {
				fmt.Printf("  🔴 问题: BracketLink状态为partial，但止盈订单未被取消\n")
			}
		}

		// 检查止损订单
		if link.SLClientID != "" {
			status := getOrderStatus(gdb, link.SLClientID)
			fmt.Printf("  止损订单 %s: %s\n", link.SLClientID, status)
			if link.Status == "partial" && status != "cancelled" {
				fmt.Printf("  🔴 问题: BracketLink状态为partial，但止损订单未被取消\n")
			}
		}
	}

	fmt.Println("\n🎯 调试总结:")
	fmt.Println("✅ 检查BracketLink状态")
	fmt.Println("✅ 检查订单执行状态")
	fmt.Println("✅ 检查联动取消逻辑")
	fmt.Println("✅ 识别潜在问题")

	fmt.Println("\n💡 建议:")
	fmt.Println("1. 确保订单同步服务正常运行")
	fmt.Println("2. 检查BracketLink查询是否正常")
	fmt.Println("3. 验证取消API调用是否成功")
	fmt.Println("4. 确认订单状态更新及时")
}

func checkOrderStatus(gdb *pdb.Database, orderType, clientOrderId string) {
	if clientOrderId == "" {
		fmt.Printf("   %s: (空)\n", orderType)
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("   %s %s: 查询失败 (%v)\n", orderType, clientOrderId, err)
		return
	}

	fmt.Printf("   %s %s: %s\n", orderType, clientOrderId, order.Status)
}

func getOrderStatus(gdb *pdb.Database, clientOrderId string) string {
	if clientOrderId == "" {
		return "empty"
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		return "not_found"
	}

	return order.Status
}

func checkCancellationStatus(gdb *pdb.Database, clientOrderId, orderType string) {
	if clientOrderId == "" {
		fmt.Printf("   ❌ %s订单ID为空\n", orderType)
		return
	}

	status := getOrderStatus(gdb, clientOrderId)
	if status == "cancelled" {
		fmt.Printf("   ✅ %s订单已正确取消: %s\n", orderType, clientOrderId)
	} else {
		fmt.Printf("   ❌ %s订单未被取消: %s (状态: %s)\n", orderType, clientOrderId, status)
	}
}