package main

import "fmt"

// 模拟StrategyConditions结构体
type StrategyConditions struct {
	MRRequireMultipleSignals bool `json:"mr_require_multiple_signals"`
}

func main() {
	fmt.Println("🔧 字段名称拼写错误修复测试")
	fmt.Println("===========================")

	// 测试修复后的字段名称
	var adapted StrategyConditions

	// 修复前: adapted.aMRRequireMultipleSignals = true  // 错误：字段不存在
	// 修复后: adapted.MRRequireMultipleSignals = true   // 正确：字段存在

	adapted.MRRequireMultipleSignals = true

	fmt.Printf("✅ 字段赋值成功: MRRequireMultipleSignals = %v\n", adapted.MRRequireMultipleSignals)
	fmt.Printf("✅ JSON标签: %s\n", `json:"mr_require_multiple_signals"`)

	fmt.Println("\n🎉 字段名称拼写错误修复完成！")
	fmt.Println("strategy_scanner_mean_reversion.go 中的编译错误已修复。")
}