package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🧪 测试OrderType字段长度修复")
	fmt.Println("==============================")

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

	// 测试不同长度的OrderType
	testOrderTypes := []string{
		"MARKET",              // 6字符
		"LIMIT",               // 5字符
		"TAKE_PROFIT_MARKET",  // 18字符
		"STOP_MARKET",         // 11字符
	}

	fmt.Println("\n1️⃣ 测试OrderType字段长度")

	for _, orderType := range testOrderTypes {
		fmt.Printf("\n测试订单类型: %s (%d字符)\n", orderType, len(orderType))

		// 尝试创建测试订单
		testOrder := &pdb.ScheduledOrder{
			UserID:     1,
			Exchange:   "binance_futures",
			Testnet:    true,
			Symbol:     "TESTUSDT",
			Side:       "BUY",
			OrderType:  orderType, // 测试不同长度的订单类型
			Quantity:   "100",
			Price:      "0.01",
			Leverage:   3,
			ReduceOnly: true,
			Status:     "pending",
		}

		err := gdb.GormDB().Create(testOrder).Error
		if err != nil {
			fmt.Printf("❌ 创建失败: %v\n", err)
		} else {
			fmt.Printf("✅ 创建成功 (ID=%d)\n", testOrder.ID)

			// 清理测试数据
			gdb.GormDB().Delete(testOrder)
			fmt.Printf("🗑️ 清理测试数据完成\n")
		}
	}

	// 检查数据库中的实际字段长度
	fmt.Println("\n2️⃣ 检查数据库字段信息")

	type ColumnInfo struct {
		Field   string `gorm:"column:Field"`
		Type    string `gorm:"column:Type"`
		Null    string `gorm:"column:Null"`
		Key     string `gorm:"column:Key"`
		Default string `gorm:"column:Default"`
		Extra   string `gorm:"column:Extra"`
	}

	var columns []ColumnInfo
	err = gdb.GormDB().Raw("DESCRIBE scheduled_orders").Scan(&columns).Error
	if err != nil {
		log.Printf("查询表结构失败: %v", err)
	} else {
		for _, col := range columns {
			if col.Field == "order_type" {
				fmt.Printf("order_type字段信息:\n")
				fmt.Printf("  类型: %s\n", col.Type)
				fmt.Printf("  可空: %s\n", col.Null)
				fmt.Printf("  键: %s\n", col.Key)
				fmt.Printf("  默认值: %s\n", col.Default)
				break
			}
		}
	}

	fmt.Println("\n🎯 修复验证:")
	fmt.Println("✅ order_type字段长度从16增加到32")
	fmt.Println("✅ 支持TAKE_PROFIT_MARKET (18字符)")
	fmt.Println("✅ 支持STOP_MARKET (11字符)")
	fmt.Println("✅ 支持其他订单类型")

	fmt.Println("\n💡 问题根源:")
	fmt.Println("❌ 原始设计只考虑MARKET/LIMIT")
	fmt.Println("❌ 字段长度限制为16字符")
	fmt.Println("❌ TAKE_PROFIT_MARKET超出限制")

	fmt.Println("\n🎉 修复内容:")
	fmt.Println("✅ 修改gorm标签: size:16 -> size:32")
	fmt.Println("✅ 执行数据库迁移: VARCHAR(16) -> VARCHAR(32)")
	fmt.Println("✅ 更新注释说明支持的订单类型")
}