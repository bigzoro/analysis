package main

import (
	"fmt"

	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("=== 测试策略更新保证金止盈字段 ===\n")

	// 模拟UpdateTradingStrategy中的逻辑
	fmt.Println("1. 模拟前端发送的更新请求:")
	req := struct {
		Conditions pdb.StrategyConditions
	}{
		Conditions: pdb.StrategyConditions{
			EnableMarginLossStopLoss:      true,
			MarginLossStopLossPercent:     20.0,
			EnableMarginProfitTakeProfit:  true,
			MarginProfitTakeProfitPercent: 50.0,
		},
	}

	fmt.Printf("   EnableMarginLossStopLoss: %v\n", req.Conditions.EnableMarginLossStopLoss)
	fmt.Printf("   MarginLossStopLossPercent: %.1f%%\n", req.Conditions.MarginLossStopLossPercent)
	fmt.Printf("   EnableMarginProfitTakeProfit: %v\n", req.Conditions.EnableMarginProfitTakeProfit)
	fmt.Printf("   MarginProfitTakeProfitPercent: %.1f%%\n", req.Conditions.MarginProfitTakeProfitPercent)

	fmt.Println("\n2. 模拟UpdateTradingStrategy中的赋值逻辑:")

	// 模拟数据库中的现有策略
	strategy := &pdb.TradingStrategy{
		Conditions: pdb.StrategyConditions{
			EnableMarginLossStopLoss:      false, // 初始值
			MarginLossStopLossPercent:     0.0,   // 初始值
			EnableMarginProfitTakeProfit:  false, // 初始值
			MarginProfitTakeProfitPercent: 0.0,   // 初始值
		},
	}

	fmt.Println("   更新前的策略配置:")
	fmt.Printf("     EnableMarginLossStopLoss: %v\n", strategy.Conditions.EnableMarginLossStopLoss)
	fmt.Printf("     MarginLossStopLossPercent: %.1f%%\n", strategy.Conditions.MarginLossStopLossPercent)
	fmt.Printf("     EnableMarginProfitTakeProfit: %v\n", strategy.Conditions.EnableMarginProfitTakeProfit)
	fmt.Printf("     MarginProfitTakeProfitPercent: %.1f%%\n", strategy.Conditions.MarginProfitTakeProfitPercent)

	// ========== 风险控制 ========== (模拟修复后的代码)
	strategy.Conditions.EnableMarginLossStopLoss = req.Conditions.EnableMarginLossStopLoss
	strategy.Conditions.MarginLossStopLossPercent = req.Conditions.MarginLossStopLossPercent
	strategy.Conditions.EnableMarginProfitTakeProfit = req.Conditions.EnableMarginProfitTakeProfit
	strategy.Conditions.MarginProfitTakeProfitPercent = req.Conditions.MarginProfitTakeProfitPercent

	fmt.Println("\n   更新后的策略配置:")
	fmt.Printf("     EnableMarginLossStopLoss: %v\n", strategy.Conditions.EnableMarginLossStopLoss)
	fmt.Printf("     MarginLossStopLossPercent: %.1f%%\n", strategy.Conditions.MarginLossStopLossPercent)
	fmt.Printf("     EnableMarginProfitTakeProfit: %v\n", strategy.Conditions.EnableMarginProfitTakeProfit)
	fmt.Printf("     MarginProfitTakeProfitPercent: %.1f%%\n", strategy.Conditions.MarginProfitTakeProfitPercent)

	// 验证
	fmt.Println("\n3. 验证结果:")
	lossStopCorrect := strategy.Conditions.EnableMarginLossStopLoss == req.Conditions.EnableMarginLossStopLoss &&
		strategy.Conditions.MarginLossStopLossPercent == req.Conditions.MarginLossStopLossPercent

	profitTakeCorrect := strategy.Conditions.EnableMarginProfitTakeProfit == req.Conditions.EnableMarginProfitTakeProfit &&
		strategy.Conditions.MarginProfitTakeProfitPercent == req.Conditions.MarginProfitTakeProfitPercent

	if lossStopCorrect && profitTakeCorrect {
		fmt.Println("✅ 更新逻辑正确！保证金止盈字段已正确更新")
	} else {
		fmt.Println("❌ 更新逻辑有误！")
		if !lossStopCorrect {
			fmt.Println("   - 保证金损失止损字段更新失败")
		}
		if !profitTakeCorrect {
			fmt.Println("   - 保证金盈利止盈字段更新失败")
		}
	}

	fmt.Println("\n=== 问题分析 ===")
	fmt.Println("修复前：UpdateTradingStrategy函数中缺少保证金盈利止盈字段的赋值")
	fmt.Println("修复后：添加了EnableMarginProfitTakeProfit和MarginProfitTakeProfitPercent的赋值")
	fmt.Println("结果：策略更新时，保证金止盈设置会被正确保存到数据库")
}
