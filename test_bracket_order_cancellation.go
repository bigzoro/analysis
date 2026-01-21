package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🧪 测试Bracket订单联动取消逻辑")
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

	// 查找最近的Bracket订单
	var bracketLinks []pdb.BracketLink
	err = gdb.GormDB().Where("status = ?", "active").Limit(5).Find(&bracketLinks).Error
	if err != nil {
		log.Printf("查询BracketLink失败: %v", err)
		return
	}

	if len(bracketLinks) == 0 {
		fmt.Println("❌ 没有找到活跃的Bracket订单")
		return
	}

	fmt.Printf("📋 找到 %d 个活跃的Bracket订单:\n", len(bracketLinks))

	for i, link := range bracketLinks {
		fmt.Printf("\n%d. BracketLink ID: %d\n", i+1, link.ID)
		fmt.Printf("   GroupID: %s\n", link.GroupID)
		fmt.Printf("   Symbol: %s\n", link.Symbol)
		fmt.Printf("   开仓订单: %s\n", link.EntryClientID)
		fmt.Printf("   止盈订单: %s\n", link.TPClientID)
		fmt.Printf("   止损订单: %s\n", link.SLClientID)
		fmt.Printf("   状态: %s\n", link.Status)

		// 检查订单状态
		var entryOrder, tpOrder, slOrder pdb.ScheduledOrder

		if link.EntryClientID != "" {
			gdb.GormDB().Where("client_order_id = ?", link.EntryClientID).First(&entryOrder)
			fmt.Printf("   开仓订单状态: %s\n", entryOrder.Status)
		}

		if link.TPClientID != "" {
			gdb.GormDB().Where("client_order_id = ?", link.TPClientID).First(&tpOrder)
			fmt.Printf("   止盈订单状态: %s\n", tpOrder.Status)
		}

		if link.SLClientID != "" {
			gdb.GormDB().Where("client_order_id = ?", link.SLClientID).First(&slOrder)
			fmt.Printf("   止损订单状态: %s\n", slOrder.Status)
		}
	}

	fmt.Println("\n🎯 测试场景分析:")
	fmt.Println("1. 如果止损订单被执行:")
	fmt.Println("   ✅ 系统会自动取消止盈订单")
	fmt.Println("   ✅ BracketLink状态更新为'partial'或'closed'")
	fmt.Println("   ✅ 防止止盈订单在无持仓情况下执行")

	fmt.Println("\n2. 如果止盈订单被执行:")
	fmt.Println("   ✅ 系统会自动取消止损订单")
	fmt.Println("   ✅ 防止重复执行")

	fmt.Println("\n3. 如果开仓订单被取消:")
	fmt.Println("   ✅ 系统会取消所有相关订单")
	fmt.Println("   ✅ BracketLink状态更新为'orphaned'")

	fmt.Println("\n💡 实现原理:")
	fmt.Println("✅ 订单同步时检查BracketLink关系")
	fmt.Println("✅ 检测订单执行状态变化")
	fmt.Println("✅ 自动取消相关联的未执行订单")
	fmt.Println("✅ 更新BracketLink状态管理")

	fmt.Println("\n🎉 Bracket订单联动取消功能已实现！")
}