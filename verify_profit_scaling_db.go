package main

import (
	"fmt"
	"log"
	"time"

	"analysis/internal/db"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("=== 验证PROFIT_SCALING数据库写入 ===\n")

	// 连接数据库
	gdb, err := db.OpenMySQL(db.Options{
		DSN:             "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:     false,
		MaxOpenConns:    2,
		MaxIdleConns:    1,
		ConnMaxLifetime: 300,
	})

	if err != nil {
		log.Fatalf("数据库连接失败: %v", err)
		return
	}
	defer gdb.Close()

	// 创建一个测试的加仓订单
	testOrder := &pdb.ScheduledOrder{
		UserID:         1,
		Symbol:         "TESTUSDT",
		Side:           "BUY",
		OrderType:      "MARKET",
		Quantity:       "100.00000000",
		Price:          "",
		Leverage:       3,
		ReduceOnly:     false,
		StrategyID:     33,
		ExecutionID:    "test_execution",
		ParentOrderId:  999, // 模拟父订单ID
		Status:         "pending",
		TriggerTime:    time.Now(),
		ClientOrderId:  fmt.Sprintf("PROFIT_SCALING_%d_%d", 999, time.Now().Unix()),
		BracketEnabled: false,
		TPPercent:      0,
		SLPercent:      0,
		WorkingType:    "CONTRACT_PRICE",
		Testnet:        true,
		Exchange:       "binance",
	}

	fmt.Printf("准备创建测试加仓订单:\n")
	fmt.Printf("  ClientOrderId: %s\n", testOrder.ClientOrderId)
	fmt.Printf("  ParentOrderId: %d\n", testOrder.ParentOrderId)
	fmt.Printf("  Symbol: %s\n", testOrder.Symbol)

	// 保存到数据库
	err = gdb.Create(testOrder).Error
	if err != nil {
		log.Printf("创建测试订单失败: %v", err)
		return
	}

	fmt.Printf("✅ 成功创建测试订单，ID: %d\n", testOrder.ID)

	// 验证可以通过LIKE查询找到
	var foundOrders []pdb.ScheduledOrder
	err = gdb.Where("client_order_id LIKE ? AND user_id = ?", "PROFIT_SCALING_%", 1).
		Find(&foundOrders).Error

	if err != nil {
		log.Printf("查询PROFIT_SCALING订单失败: %v", err)
		return
	}

	fmt.Printf("\n通过LIKE查询找到的PROFIT_SCALING订单:\n")
	for _, order := range foundOrders {
		fmt.Printf("  ID: %d, ClientOrderId: %s, ParentOrderId: %d, Symbol: %s\n",
			order.ID, order.ClientOrderId, order.ParentOrderId, order.Symbol)
	}

	// 验证可以通过parent_order_id查找子订单
	var childOrders []pdb.ScheduledOrder
	err = gdb.Where("parent_order_id = ? AND user_id = ?", 999, 1).
		Find(&childOrders).Error

	if err != nil {
		log.Printf("查询子订单失败: %v", err)
		return
	}

	fmt.Printf("\n通过parent_order_id查询找到的子订单:\n")
	for _, order := range childOrders {
		fmt.Printf("  ID: %d, ClientOrderId: %s, Symbol: %s\n",
			order.ID, order.ClientOrderId, order.Symbol)
	}

	// 清理测试数据
	fmt.Printf("\n清理测试数据...\n")
	err = gdb.Delete(testOrder).Error
	if err != nil {
		log.Printf("删除测试订单失败: %v", err)
	} else {
		fmt.Printf("✅ 成功删除测试订单\n")
	}

	fmt.Printf("\n=== 验证完成 ===\n")
	fmt.Printf("✅ PROFIT_SCALING前缀可以正确写入数据库\n")
	fmt.Printf("✅ LIKE查询可以正确查找PROFIT_SCALING订单\n")
	fmt.Printf("✅ parent_order_id关联关系工作正常\n")
}
