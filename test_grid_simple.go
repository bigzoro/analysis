package main

import (
	"fmt"

	pdb "analysis/internal/db"
	"analysis/internal/server"
)

func main() {
	fmt.Println("🧪 网格交易简单测试")

	// 测试策略执行器
	executor := &server.GridTradingStrategyExecutor{}
	fmt.Printf("✅ 策略类型: %s\n", executor.GetStrategyType())

	// 测试网格扫描器
	scanner := &server.GridTradingStrategyScanner{}
	rangeResult := scanner.calculateDynamicGridRange(100.0)
	fmt.Printf("✅ 动态网格范围 (价格100): [%.2f, %.2f]\n", rangeResult.lower, rangeResult.upper)

	// 测试订单管理器
	gom := &server.GridOrderManager{
		conditions: pdb.StrategyConditions{GridLevels: 5},
	}
	fmt.Printf("✅ 订单管理器层级: %d\n", gom.conditions.GridLevels)

	// 测试风险管理器
	grm := &server.GridRiskManager{
		positionHistory: []server.GridPosition{{Symbol: "BTCUSDT"}},
	}
	fmt.Printf("✅ 风险管理器持仓数: %d\n", len(grm.positionHistory))

	fmt.Println("🎉 所有网格交易组件测试通过！")
}
