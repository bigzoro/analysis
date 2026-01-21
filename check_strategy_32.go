package main

import (
	"fmt"
)

// StrategyConditions 模拟策略条件结构体
type StrategyConditions struct {
	MRStopLossMultiplier   float64
	MRTakeProfitMultiplier float64
	MRMaxPositionSize      float64
	MRMaxHoldHours         int
}

func main() {
	fmt.Println("=== 均值回归策略配置转换分析 ===")

	// 测试不同的配置场景
	testCases := []struct {
		name       string
		conditions StrategyConditions
	}{
		{
			name: "正常配置",
			conditions: StrategyConditions{
				MRStopLossMultiplier:   1.5,  // 1.5倍标准差作为止损
				MRTakeProfitMultiplier: 1.08, // 8%止盈
				MRMaxPositionSize:      0.05, // 5%仓位
				MRMaxHoldHours:         48,   // 48小时最大持仓
			},
		},
		{
			name: "问题配置场景1",
			conditions: StrategyConditions{
				MRStopLossMultiplier:   1.0, // 边界值
				MRTakeProfitMultiplier: 1.0, // 边界值
				MRMaxPositionSize:      0.0, // 无效值
				MRMaxHoldHours:         0,   // 无效值
			},
		},
		{
			name: "问题配置场景2",
			conditions: StrategyConditions{
				MRStopLossMultiplier:   0.5, // 小于1，会导致StopLoss > 1
				MRTakeProfitMultiplier: 0.9, // 小于1，会导致TakeProfit < 0
				MRMaxPositionSize:      1.5, // 大于1，无效
				MRMaxHoldHours:         -1,  // 负数，无效
			},
		},
	}

	for _, tc := range testCases {
		fmt.Printf("\n--- %s ---\n", tc.name)
		analyzeConversion(tc.conditions)
	}
}

func analyzeConversion(conditions StrategyConditions) {
	fmt.Printf("原始配置:\n")
	fmt.Printf("  MRStopLossMultiplier: %.3f\n", conditions.MRStopLossMultiplier)
	fmt.Printf("  MRTakeProfitMultiplier: %.3f\n", conditions.MRTakeProfitMultiplier)
	fmt.Printf("  MRMaxPositionSize: %.3f\n", conditions.MRMaxPositionSize)
	fmt.Printf("  MRMaxHoldHours: %d\n", conditions.MRMaxHoldHours)

	// 应用转换逻辑
	maxPositionSize := conditions.MRMaxPositionSize
	stopLoss := 1.0 / conditions.MRStopLossMultiplier
	takeProfit := conditions.MRTakeProfitMultiplier - 1.0
	maxHoldHours := conditions.MRMaxHoldHours

	fmt.Printf("\n转换后的配置:\n")
	fmt.Printf("  MaxPositionSize: %.3f (直接赋值)\n", maxPositionSize)
	fmt.Printf("  StopLoss: %.3f (1.0 / %.3f)\n", stopLoss, conditions.MRStopLossMultiplier)
	fmt.Printf("  TakeProfit: %.3f (%.3f - 1.0)\n", takeProfit, conditions.MRTakeProfitMultiplier)
	fmt.Printf("  MaxHoldHours: %d (直接赋值)\n", maxHoldHours)

	// 应用默认值逻辑
	if maxPositionSize <= 0 {
		maxPositionSize = 0.02
		fmt.Printf("  → MaxPositionSize 使用默认值: %.3f\n", maxPositionSize)
	}
	if stopLoss <= 0 || stopLoss > 0.5 {
		stopLoss = 0.03
		fmt.Printf("  → StopLoss 使用默认值: %.3f (原值%.3f超出范围)\n", stopLoss, 1.0/conditions.MRStopLossMultiplier)
	}
	if takeProfit <= 0 {
		takeProfit = 0.06
		fmt.Printf("  → TakeProfit 使用默认值: %.3f (原值%.3f无效)\n", takeProfit, conditions.MRTakeProfitMultiplier-1.0)
	}
	if maxHoldHours <= 0 {
		maxHoldHours = 24
		fmt.Printf("  → MaxHoldHours 使用默认值: %d\n", maxHoldHours)
	}

	fmt.Printf("\n最终配置:\n")
	fmt.Printf("  MaxPositionSize: %.3f\n", maxPositionSize)
	fmt.Printf("  StopLoss: %.3f\n", stopLoss)
	fmt.Printf("  TakeProfit: %.3f\n", takeProfit)
	fmt.Printf("  MaxHoldHours: %d\n", maxHoldHours)

	// 问题分析
	fmt.Printf("\n问题诊断:\n")

	if conditions.MRStopLossMultiplier <= 1.0 {
		fmt.Printf("  ❌ MRStopLossMultiplier (%.3f) <= 1.0\n", conditions.MRStopLossMultiplier)
		fmt.Printf("     问题: 会导致StopLoss >= 1.0 (100%%止损)\n")
		fmt.Printf("     建议: 设置为1.5-3.0之间，表示几倍标准差作为止损距离\n")
	}

	if conditions.MRTakeProfitMultiplier <= 1.0 {
		fmt.Printf("  ❌ MRTakeProfitMultiplier (%.3f) <= 1.0\n", conditions.MRTakeProfitMultiplier)
		fmt.Printf("     问题: 会导致TakeProfit <= 0 (无效止盈)\n")
		fmt.Printf("     建议: 设置为1.05-1.20之间，表示几%%的止盈比例\n")
	}

	if conditions.MRMaxPositionSize > 1.0 {
		fmt.Printf("  ❌ MRMaxPositionSize (%.3f) > 1.0\n", conditions.MRMaxPositionSize)
		fmt.Printf("     问题: 仓位比例不能超过100%%\n")
		fmt.Printf("     建议: 设置为0.01-0.10之间，表示1%%-10%%仓位\n")
	}

	if conditions.MRMaxHoldHours < 0 {
		fmt.Printf("  ❌ MRMaxHoldHours (%d) < 0\n", conditions.MRMaxHoldHours)
		fmt.Printf("     问题: 持仓时间不能为负数\n")
		fmt.Printf("     建议: 设置为24-168之间，表示24小时到1周\n")
	}

	// 验证最终结果
	valid := true
	if stopLoss <= 0 || stopLoss > 0.5 {
		fmt.Printf("  ❌ 最终StopLoss (%.3f) 仍然无效\n", stopLoss)
		valid = false
	}
	if takeProfit < 0 {
		fmt.Printf("  ❌ 最终TakeProfit (%.3f) 仍然无效\n", takeProfit)
		valid = false
	}
	if maxPositionSize <= 0 || maxPositionSize > 1 {
		fmt.Printf("  ❌ 最终MaxPositionSize (%.3f) 仍然无效\n", maxPositionSize)
		valid = false
	}
	if maxHoldHours <= 0 {
		fmt.Printf("  ❌ 最终MaxHoldHours (%d) 仍然无效\n", maxHoldHours)
		valid = false
	}

	if valid {
		fmt.Printf("  ✅ 配置转换成功，所有参数有效\n")
	} else {
		fmt.Printf("  ❌ 配置转换失败，需要修复参数\n")
	}
}
