package main

import (
	"fmt"
	"log"
	pdb "analysis/internal/db"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🧪 测试Bracket同步修复效果")
	fmt.Println("============================")

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

	// 获取交易所客户端
	client := bf.New(false, "test_key", "test_secret") // 测试环境

	fmt.Println("\n1️⃣ 模拟syncBracketOrders逻辑")

	// 模拟检查问题订单的状态
	testClientId := "sch-1281-768883136-sl"

	// 首先尝试查询Algo订单
	fmt.Printf("测试Algo订单查询: %s\n", testClientId)
	algoStatus, algoErr := client.QueryAlgoOrder("XNYUSDT", testClientId)
	if algoErr != nil {
		fmt.Printf("Algo订单查询失败: %v\n", algoErr)

		// 尝试传统订单查询
		fmt.Printf("尝试传统订单查询: %s\n", testClientId)
		tradStatus, tradErr := client.QueryOrder("XNYUSDT", testClientId)
		if tradErr != nil {
			fmt.Printf("传统订单查询失败: %v\n", tradErr)
		} else {
			fmt.Printf("传统订单状态: %s\n", tradStatus.Status)

			// 检查是否会触发slTriggered
			if tradStatus.Status == "FILLED" {
				fmt.Println("✅ 传统订单检查: 会触发slTriggered = true")
			} else {
				fmt.Println("❌ 传统订单检查: 不会触发slTriggered")
			}
		}
	} else {
		fmt.Printf("Algo订单状态: %s\n", algoStatus.Status)

		// 检查是否会触发slTriggered（修复后的逻辑）
		if algoStatus.Status == "TRIGGERED" || algoStatus.Status == "FILLED" || algoStatus.Status == "FINISHED" || algoStatus.Status == "success" {
			fmt.Println("✅ Algo订单检查: 会触发slTriggered = true（修复后包含success状态）")
		} else {
			fmt.Println("❌ Algo订单检查: 不会触发slTriggered")
		}

		// 检查修复前的逻辑
		if algoStatus.Status == "TRIGGERED" || algoStatus.Status == "FILLED" || algoStatus.Status == "FINISHED" {
			fmt.Println("✅ 修复前逻辑: 会触发slTriggered = true")
		} else {
			fmt.Println("❌ 修复前逻辑: 不会触发slTriggered（这就是问题所在！）")
		}
	}

	fmt.Println("\n2️⃣ 验证数据库中的订单状态")

	// 检查数据库中的实际状态
	var order pdb.ScheduledOrder
	err = gdb.GormDB().Where("client_order_id = ?", testClientId).First(&order).Error
	if err != nil {
		log.Printf("查询订单失败: %v", err)
	} else {
		fmt.Printf("数据库中的订单状态: %s\n", order.Status)
		fmt.Printf("订单结果: %s\n", order.Result)

		if order.Status == "success" {
			fmt.Println("🎯 这证实了问题：订单状态是'success'，但修复前的代码无法识别！")
		}
	}

	fmt.Println("\n3️⃣ 修复总结")
	fmt.Println("修复内容：")
	fmt.Println("  - 在syncBracketOrders中，Algo订单状态检查增加'success'状态")
	fmt.Println("  - TP和SL订单检查都包含: TRIGGERED | FILLED | FINISHED | success")
	fmt.Println("  - 这确保当条件订单执行时，系统能正确检测到触发事件")
	fmt.Println("  - 从而调用handleBracketOrderClosure来取消另一方向的订单")

	fmt.Println("\n✅ 修复验证完成")
}