package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔧 测试Bracket订单修复效果")
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

	// 检查活跃的Bracket订单
	var activeBrackets []pdb.BracketLink
	err = gdb.GormDB().Where("status = ?", "active").Find(&activeBrackets).Error
	if err != nil {
		log.Printf("查询活跃Bracket订单失败: %v", err)
		return
	}

	fmt.Printf("📊 当前活跃Bracket订单数量: %d\n", len(activeBrackets))

	for i, bracket := range activeBrackets {
		fmt.Printf("\n%d. Bracket订单 %s:\n", i+1, bracket.GroupID)
		fmt.Printf("   交易对: %s\n", bracket.Symbol)
		fmt.Printf("   开仓订单ID: %s\n", bracket.EntryClientID)
		fmt.Printf("   止盈订单ID: %s\n", bracket.TPClientID)
		fmt.Printf("   止损订单ID: %s\n", bracket.SLClientID)

		// 检查各个订单的状态
		checkOrderStatus(gdb, bracket.EntryClientID, "开仓")
		checkOrderStatus(gdb, bracket.TPClientID, "止盈")
		checkOrderStatus(gdb, bracket.SLClientID, "止损")
	}

	if len(activeBrackets) == 0 {
		fmt.Println("✅ 没有活跃的Bracket订单")
		fmt.Println("\n📝 修复说明:")
		fmt.Println("   修复后的逻辑将在以下场景中生效:")
		fmt.Println("   1. 当止盈触发时，自动取消止损订单")
		fmt.Println("   2. 当止损触发时，自动取消止盈订单")
		fmt.Println("   3. 避免同一仓位被双重平仓")
		fmt.Println("   4. 释放被占用的保证金")
	}
}

func checkOrderStatus(gdb pdb.Database, clientOrderId, orderType string) {
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

	fmt.Printf("   %s订单: %s (ID: %d)\n", orderType, order.Status, order.ID)

	// 检查是否可能是条件订单
	if order.OrderType == "TAKE_PROFIT_MARKET" || order.OrderType == "STOP_MARKET" {
		fmt.Printf("      类型: %s (条件订单)\n", order.OrderType)
		if order.Price != "" {
			fmt.Printf("      触发价格: %s\n", order.Price)
		}
	}
}