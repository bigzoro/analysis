package main

import (
	"fmt"
	"strconv"
)

func main() {
	fmt.Println("=== 调试保证金计算问题 ===")

	// 模拟用户的输入
	perOrderAmount := 100.0  // 用户输入的每一单金额
	leverage := 3.0          // 杠杆倍数

	fmt.Printf("用户输入参数:\n")
	fmt.Printf("  每一单金额: %.2f USDT\n", perOrderAmount)
	fmt.Printf("  杠杆倍数: %.0f\n", leverage)

	// 理论计算（创建订单时的逻辑）
	theoreticalMargin := perOrderAmount
	theoreticalNotional := theoreticalMargin * leverage

	fmt.Printf("\n理论计算（创建订单时）:\n")
	fmt.Printf("  保证金: %.2f USDT\n", theoreticalMargin)
	fmt.Printf("  名义价值: %.2f USDT\n", theoreticalNotional)

	// 模拟实际的交易数据（从数据库查询到的）
	// 以BTRUSDT为例
	symbol := "BTRUSDT"
	originalQuantity := "1"      // 创建时的数量
	adjustedQuantity := "66"     // 执行后的调整数量
	price := 0.004512           // 假设的执行价格

	fmt.Printf("\n实际执行数据 (%s):\n", symbol)
	fmt.Printf("  原始数量: %s\n", originalQuantity)
	fmt.Printf("  调整后数量: %s\n", adjustedQuantity)
	fmt.Printf("  执行价格: %.6f\n", price)

	// 计算实际的名义价值和保证金
	adjustedQtyFloat, _ := strconv.ParseFloat(adjustedQuantity, 64)
	actualNotional := adjustedQtyFloat * price
	actualMargin := actualNotional / leverage

	fmt.Printf("\n实际保证金计算:\n")
	fmt.Printf("  名义价值: %.4f USDT (%.0f * %.6f)\n", actualNotional, adjustedQtyFloat, price)
	fmt.Printf("  保证金: %.4f USDT (%.4f / %.0f)\n", actualMargin, actualNotional, leverage)

	// 分析问题
	fmt.Printf("\n问题分析:\n")
	fmt.Printf("  用户期望保证金: %.2f USDT\n", perOrderAmount)
	fmt.Printf("  实际保证金: %.4f USDT\n", actualMargin)
	fmt.Printf("  差异: %.4f USDT\n", actualMargin - perOrderAmount)

	if actualMargin < perOrderAmount * 0.1 { // 如果实际保证金不到期望的10%
		fmt.Printf("  结论: ❌ 实际保证金远小于用户期望，存在严重计算错误\n")
	} else if actualMargin < perOrderAmount * 0.5 { // 如果实际保证金不到期望的50%
		fmt.Printf("  结论: ⚠️  实际保证金明显小于用户期望，存在计算偏差\n")
	} else {
		fmt.Printf("  结论: ✅ 实际保证金接近用户期望，计算正常\n")
	}

	// 解释原因
	fmt.Printf("\n原因解释:\n")
	fmt.Printf("1. 创建订单时，系统根据名义价值计算初始数量\n")
	fmt.Printf("2. 但初始数量可能太小，不满足交易所的最低名义价值要求\n")
	fmt.Printf("3. 执行时系统自动调整数量以满足名义价值要求\n")
	fmt.Printf("4. 调整后的数量远大于初始数量，导致保证金偏离用户期望\n")

	// 计算正确的初始数量
	correctInitialQty := theoreticalNotional / price
	fmt.Printf("\n正确的初始数量计算:\n")
	fmt.Printf("  理论名义价值: %.2f USDT\n", theoreticalNotional)
	fmt.Printf("  当前价格: %.6f USDT\n", price)
	fmt.Printf("  正确初始数量: %.2f (%.2f / %.6f)\n", correctInitialQty, theoreticalNotional, price)

	fmt.Printf("\n解决方案建议:\n")
	fmt.Printf("1. 改进初始数量计算，确保满足名义价值要求\n")
	fmt.Printf("2. 或者在创建订单时就进行名义价值验证和调整\n")
	fmt.Printf("3. 提供更准确的用户反馈，避免期望与实际的差距\n")

	fmt.Printf("\n当前状态总结:\n")
	fmt.Printf("• 用户输入100 USDT期望保证金\n")
	fmt.Printf("• 系统计算出保证金约%.2f USDT\n", actualMargin)
	fmt.Printf("• 这就是为什么实际开仓保证金是1.67左右的原因\n", actualMargin)
}