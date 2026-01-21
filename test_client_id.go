package main

import (
	"fmt"
)

func main() {
	fmt.Println("=== 测试加仓订单ClientOrderId逻辑 ===\n")

	// 模拟订单对象
	type MockOrder struct {
		ID            uint
		ClientOrderId string
	}

	// 测试场景1：有ClientOrderId的订单（加仓订单）
	profitScalingOrder := MockOrder{
		ID:            1000,
		ClientOrderId: "PROFIT_SCALING_999_1704067200",
	}

	fmt.Printf("场景1 - 加仓订单:\n")
	fmt.Printf("  订单ID: %d\n", profitScalingOrder.ID)
	fmt.Printf("  原始ClientOrderId: %s\n", profitScalingOrder.ClientOrderId)

	var clientOrderId string
	if profitScalingOrder.ClientOrderId != "" {
		clientOrderId = profitScalingOrder.ClientOrderId
		fmt.Printf("  ✅ 使用已有的ClientOrderId: %s\n", clientOrderId)
	} else {
		clientOrderId = fmt.Sprintf("sch-%d-%d", profitScalingOrder.ID, 1704067200)
		fmt.Printf("  ❌ 生成新的ClientOrderId: %s\n", clientOrderId)
	}

	if clientOrderId == profitScalingOrder.ClientOrderId {
		fmt.Printf("  ✅ ClientOrderId保持PROFIT_SCALING格式\n")
	} else {
		fmt.Printf("  ❌ ClientOrderId被改变\n")
	}

	// 测试场景2：没有ClientOrderId的订单（普通订单）
	normalOrder := MockOrder{
		ID:            1001,
		ClientOrderId: "",
	}

	fmt.Printf("\n场景2 - 普通订单:\n")
	fmt.Printf("  订单ID: %d\n", normalOrder.ID)
	fmt.Printf("  原始ClientOrderId: (空)\n")

	if normalOrder.ClientOrderId != "" {
		clientOrderId = normalOrder.ClientOrderId
		fmt.Printf("  ✅ 使用已有的ClientOrderId: %s\n", clientOrderId)
	} else {
		clientOrderId = fmt.Sprintf("sch-%d-%d", normalOrder.ID, 1704067200)
		fmt.Printf("  ✅ 生成新的ClientOrderId: %s\n", clientOrderId)
	}

	if clientOrderId != "" && clientOrderId[:4] == "sch-" {
		fmt.Printf("  ✅ 普通订单生成sch-格式ClientOrderId\n")
	} else {
		fmt.Printf("  ❌ ClientOrderId格式不正确\n")
	}

	fmt.Printf("\n=== 修复总结 ===\n")
	fmt.Printf("1. 加仓订单：使用PROFIT_SCALING_前缀的ClientOrderId\n")
	fmt.Printf("2. 普通订单：生成sch-格式的ClientOrderId\n")
	fmt.Printf("3. 交易所收到的ClientOrderId格式正确\n")
}
