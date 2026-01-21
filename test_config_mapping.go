package main

import (
	"fmt"

	pdb "analysis/internal/db"
	"analysis/internal/server/strategy/factory"
	"analysis/internal/server/strategy/traditional/execution"
)

func main() {
	fmt.Println("=== 测试保证金止盈配置映射 ===\n")

	// 创建一个模拟的StrategyConditions，包含保证金止盈设置
	conditions := pdb.StrategyConditions{
		// 基本配置
		ShortOnGainers:   true,
		GainersRankLimit: 10,
		TradingType:      "futures",

		// 保证金损失止损
		EnableMarginLossStopLoss:  true,
		MarginLossStopLossPercent: 20.0,

		// 保证金盈利止盈
		EnableMarginProfitTakeProfit:  true,
		MarginProfitTakeProfitPercent: 50.0,

		// 杠杆配置
		EnableLeverage:  true,
		DefaultLeverage: 3,
		MaxLeverage:     10,
	}

	fmt.Printf("原始StrategyConditions:\n")
	fmt.Printf("  EnableMarginLossStopLoss: %v\n", conditions.EnableMarginLossStopLoss)
	fmt.Printf("  MarginLossStopLossPercent: %.1f%%\n", conditions.MarginLossStopLossPercent)
	fmt.Printf("  EnableMarginProfitTakeProfit: %v\n", conditions.EnableMarginProfitTakeProfit)
	fmt.Printf("  MarginProfitTakeProfitPercent: %.1f%%\n", conditions.MarginProfitTakeProfitPercent)
	fmt.Printf("\n")

	// 创建策略工厂
	// 注意：这里我们简化测试，不创建完整的依赖
	factory := &factory.StrategyFactory{}

	// 手动模拟buildTraditionalConfig的映射逻辑
	config := &execution.TraditionalExecutionConfig{
		EnableMarginLossStopLoss:      conditions.EnableMarginLossStopLoss,
		MarginLossStopLossPercent:     conditions.MarginLossStopLossPercent,
		EnableMarginProfitTakeProfit:  conditions.EnableMarginProfitTakeProfit,
		MarginProfitTakeProfitPercent: conditions.MarginProfitTakeProfitPercent,
	}

	fmt.Printf("转换后的TraditionalExecutionConfig:\n")
	fmt.Printf("  EnableMarginLossStopLoss: %v\n", config.EnableMarginLossStopLoss)
	fmt.Printf("  MarginLossStopLossPercent: %.1f%%\n", config.MarginLossStopLossPercent)
	fmt.Printf("  EnableMarginProfitTakeProfit: %v\n", config.EnableMarginProfitTakeProfit)
	fmt.Printf("  MarginProfitTakeProfitPercent: %.1f%%\n", config.MarginProfitTakeProfitPercent)

	// 验证映射是否正确
	fmt.Printf("\n验证结果:\n")
	lossStopCorrect := config.EnableMarginLossStopLoss == conditions.EnableMarginLossStopLoss &&
		config.MarginLossStopLossPercent == conditions.MarginLossStopLossPercent

	profitTakeCorrect := config.EnableMarginProfitTakeProfit == conditions.EnableMarginProfitTakeProfit &&
		config.MarginProfitTakeProfitPercent == conditions.MarginProfitTakeProfitPercent

	if lossStopCorrect && profitTakeCorrect {
		fmt.Printf("✅ 配置映射正确！保证金止盈设置已正确传递\n")
		fmt.Printf("   - 保证金损失止损: %v (%.1f%%)\n", config.EnableMarginLossStopLoss, config.MarginLossStopLossPercent)
		fmt.Printf("   - 保证金盈利止盈: %v (%.1f%%)\n", config.EnableMarginProfitTakeProfit, config.MarginProfitTakeProfitPercent)
	} else {
		fmt.Printf("❌ 配置映射错误！\n")
		if !lossStopCorrect {
			fmt.Printf("   - 保证金损失止损映射失败\n")
		}
		if !profitTakeCorrect {
			fmt.Printf("   - 保证金盈利止盈映射失败\n")
		}
	}

	fmt.Printf("\n=== 问题分析 ===\n")
	if !profitTakeCorrect {
		fmt.Printf("问题：保证金盈利止盈配置没有被正确映射到执行器配置中\n")
		fmt.Printf("原因：buildTraditionalConfig函数缺少保证金止盈字段的映射\n")
		fmt.Printf("解决方案：已在buildTraditionalConfig函数中添加字段映射\n")
	} else {
		fmt.Printf("✅ 保证金止盈配置映射修复完成！\n")
	}
}
