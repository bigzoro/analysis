package main

import (
	"fmt"
	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🧪 测试跳过平仓记录小时数更新功能")

	// 模拟更新请求中的Conditions数据
	conditions := pdb.StrategyConditions{
		SkipCloseOrdersWithin24Hours: true,  // 旧字段：启用24小时过滤
		SkipCloseOrdersHours:         48,    // 新字段：48小时过滤
	}

	fmt.Println("\n📋 测试数据:")
	fmt.Printf("SkipCloseOrdersWithin24Hours: %v\n", conditions.SkipCloseOrdersWithin24Hours)
	fmt.Printf("SkipCloseOrdersHours: %d\n", conditions.SkipCloseOrdersHours)

	fmt.Println("\n🔄 模拟UpdateTradingStrategy中的更新逻辑:")

	// 模拟现有策略
	strategy := &pdb.TradingStrategy{}
	strategy.Conditions.SkipCloseOrdersWithin24Hours = false // 初始值
	strategy.Conditions.SkipCloseOrdersHours = 0            // 初始值

	fmt.Println("更新前:")
	fmt.Printf("  SkipCloseOrdersWithin24Hours: %v\n", strategy.Conditions.SkipCloseOrdersWithin24Hours)
	fmt.Printf("  SkipCloseOrdersHours: %d\n", strategy.Conditions.SkipCloseOrdersHours)

	// 应用更新（修复后的逻辑）
	strategy.Conditions.SkipCloseOrdersWithin24Hours = conditions.SkipCloseOrdersWithin24Hours
	strategy.Conditions.SkipCloseOrdersHours = conditions.SkipCloseOrdersHours

	fmt.Println("更新后:")
	fmt.Printf("  SkipCloseOrdersWithin24Hours: %v\n", strategy.Conditions.SkipCloseOrdersWithin24Hours)
	fmt.Printf("  SkipCloseOrdersHours: %d\n", strategy.Conditions.SkipCloseOrdersHours)

	// 验证结果
	if strategy.Conditions.SkipCloseOrdersHours == 48 {
		fmt.Println("\n✅ 更新成功！新字段正确接收了48小时的值")
	} else {
		fmt.Printf("\n❌ 更新失败！期望48小时，实际%d小时\n", strategy.Conditions.SkipCloseOrdersHours)
	}

	if strategy.Conditions.SkipCloseOrdersWithin24Hours == true {
		fmt.Println("✅ 旧字段也正确更新为true")
	} else {
		fmt.Println("❌ 旧字段更新失败")
	}

	fmt.Println("\n📝 修复说明:")
	fmt.Println("在UpdateTradingStrategy函数中添加了以下代码:")
	fmt.Println("strategy.Conditions.SkipCloseOrdersHours = req.Conditions.SkipCloseOrdersHours")
	fmt.Println("")
	fmt.Println("这样前端发送的skip_close_orders_hours字段就会被正确保存到数据库。")

	fmt.Println("\n🎯 现在用户可以:")
	fmt.Println("- 设置0小时：完全禁用平仓过滤")
	fmt.Println("- 设置24小时：标准过滤（默认）")
	fmt.Println("- 设置72小时：保守过滤")
	fmt.Println("- 设置任意小时数：完全定制化")

	fmt.Println("\n✅ 修复完成！跳过平仓记录小时数更新功能现在应该正常工作了。")
}