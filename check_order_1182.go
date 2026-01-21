package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
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

	// 检查订单1182
	var order pdb.ScheduledOrder
	err = gdb.GormDB().Where("id = ?", 1182).First(&order).Error
	if err != nil {
		log.Printf("查询订单失败: %v", err)
		return
	}

	fmt.Printf("订单1182详情:\n")
	fmt.Printf("  ID: %d\n", order.ID)
	fmt.Printf("  OrderType: %s\n", order.OrderType)
	fmt.Printf("  BracketEnabled: %v\n", order.BracketEnabled)
	fmt.Printf("  ClientOrderId: %s\n", order.ClientOrderId)
	fmt.Printf("  Status: %s\n", order.Status)
	fmt.Printf("  ParentOrderId: %d\n", order.ParentOrderId)
	fmt.Printf("  ReduceOnly: %v\n", order.ReduceOnly)
}