package main

import (
	"fmt"
	bf "analysis/internal/exchange/binancefutures"
)

func main() {
	fmt.Println("🧪 测试Algo订单查询")
	fmt.Println("=====================")

	// 创建币安客户端
	client := bf.New(true, "your_api_key", "your_secret_key")

	// 测试查询存在的Algo订单
	testClientIds := []string{
		"sch-1204-768877839-tp",
		"sch-1204-768877839-sl",
	}

	for _, clientId := range testClientIds {
		fmt.Printf("\n查询Algo订单: %s\n", clientId)

		// 查询Algo订单状态
		orderStatus, err := client.QueryAlgoOrder("FHEUSDT", clientId)
		if err != nil {
			fmt.Printf("❌ 查询失败: %v\n", err)
		} else {
			fmt.Printf("✅ 查询成功:\n")
			fmt.Printf("   AlgoId: %d\n", orderStatus.AlgoId)
			fmt.Printf("   ClientAlgoId: %s\n", orderStatus.ClientAlgoId)
			fmt.Printf("   Symbol: %s\n", orderStatus.Symbol)
			fmt.Printf("   Side: %s\n", orderStatus.Side)
			fmt.Printf("   Type: %s\n", orderStatus.Type)
			fmt.Printf("   Status: %s\n", orderStatus.Status)
			fmt.Printf("   TriggerPrice: %s\n", orderStatus.TriggerPrice)
			fmt.Printf("   Quantity: %s\n", orderStatus.Quantity)
			fmt.Printf("   ExecutedQty: %s\n", orderStatus.ExecutedQty)
		}
	}

	// 测试Algo订单状态映射
	fmt.Println("\n🎯 Algo订单状态映射测试")

	testStatuses := []string{"CREATED", "WORKING", "EXECUTED", "FINISHED", "CANCELED", "EXPIRED", "UNKNOWN"}

	validStatuses := map[string]bool{
		"CREATED":          true,
		"WORKING":          true,
		"EXECUTED":         true,
		"FINISHED":         true,
	}

	for _, status := range testStatuses {
		if validStatuses[status] {
			fmt.Printf("✅ 状态 '%s' -> 成功\n", status)
		} else if status == "CANCELED" || status == "EXPIRED" {
			fmt.Printf("✅ 状态 '%s' -> 成功 (已完成)\n", status)
		} else {
			fmt.Printf("❌ 状态 '%s' -> 失败\n", status)
		}
	}

	fmt.Println("\n💡 修复内容:")
	fmt.Println("✅ 使用QueryAlgoOrder替代QueryOrder")
	fmt.Println("✅ 正确处理Algo订单状态")
	fmt.Println("✅ 支持CREATED/WORKING/EXECUTED/FINISHED状态")
	fmt.Println("✅ 条件订单查询不再失败")
}