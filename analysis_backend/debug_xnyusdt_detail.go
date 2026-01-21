package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 XNYUSDT Bracket订单详细排查")

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

	// 检查问题订单
	fmt.Println("\n🎯 检查问题订单: sch-1281-768883136-sl")
	var slOrder pdb.ScheduledOrder
	err = gdb.GormDB().Where("client_order_id = ?", "sch-1281-768883136-sl").First(&slOrder).Error
	if err != nil {
		log.Printf("查询止损订单失败: %v", err)
	} else {
		fmt.Printf("止损订单详情:\n")
		fmt.Printf("  ID: %d\n", slOrder.ID)
		fmt.Printf("  状态: %s\n", slOrder.Status)
		fmt.Printf("  类型: %s\n", slOrder.OrderType)
		fmt.Printf("  交易所订单ID: %s\n", slOrder.ExchangeOrderId)
		fmt.Printf("  创建时间: %s\n", slOrder.CreatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  更新时间: %s\n", slOrder.UpdatedAt.Format("2006-01-02 15:04:05"))
		fmt.Printf("  结果: %s\n", slOrder.Result)

		// 检查BracketLink
		var bracket pdb.BracketLink
		err = gdb.GormDB().Where("sl_client_id = ?", "sch-1281-768883136-sl").First(&bracket).Error
		if err != nil {
			log.Printf("查询BracketLink失败: %v", err)
		} else {
			fmt.Printf("Bracket状态: %s\n", bracket.Status)
			fmt.Printf("TP订单状态: ")
			checkOrderStatusSimple(gdb, bracket.TPClientID)
			fmt.Printf("Entry订单状态: ")
			checkOrderStatusSimple(gdb, bracket.EntryClientID)
		}
	}

	// 检查日志记录
	fmt.Println("\n📋 检查操作日志")
	var logs []pdb.OperationLog
	err = gdb.GormDB().Where("entity_type = ? AND entity_id = ? AND action = ?",
		"order", slOrder.ID, "sync").Order("created_at DESC").Limit(5).Find(&logs).Error
	if err != nil {
		log.Printf("查询日志失败: %v", err)
	} else {
		fmt.Printf("找到%d条相关日志:\n", len(logs))
		for _, logEntry := range logs {
			fmt.Printf("  %s: %s\n", logEntry.CreatedAt.Format("15:04:05"), logEntry.Description)
		}
	}
}

func checkOrderStatusSimple(gdb pdb.Database, clientOrderId string) {
	if clientOrderId == "" {
		fmt.Printf("空\n")
		return
	}

	var order pdb.ScheduledOrder
	err := gdb.GormDB().Where("client_order_id = ?", clientOrderId).First(&order).Error
	if err != nil {
		fmt.Printf("查询失败\n")
		return
	}

	fmt.Printf("%s\n", order.Status)
}