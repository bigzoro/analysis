package main

import "fmt"

func main() {
	fmt.Println("=== 盈利加仓次数限制修改方案：策略级别 → 币种级别 ===\n")

	fmt.Println("🎯 修改目标：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("当前：策略最大加仓次数为1 → 整个策略只能总共加仓1次\n")
	fmt.Printf("目标：策略最大加仓次数为1 → 每个币种都可以独立加仓1次\n\n")

	fmt.Println("📋 修改方案：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Println("方案1：JSON字段存储币种计数器")
	fmt.Println("─────────────────────────────────────────────────")
	fmt.Printf("优点：\n")
	fmt.Printf("  ✅ 简单实现，无需修改数据库结构\n")
	fmt.Printf("  ✅ 扩展性好，可以存储额外信息\n")
	fmt.Printf("  ✅ 原子性更新\n\n")

	fmt.Printf("实现方式：\n")
	fmt.Printf("  1. 在trading_strategies表添加字段：\n")
	fmt.Printf("     profit_scaling_symbol_counts JSON\n\n")

	fmt.Printf("  2. 数据结构：\n")
	fmt.Printf("     {\n")
	fmt.Printf("       \"BTCUSDT\": 1,\n")
	fmt.Printf("       \"ETHUSDT\": 0,\n")
	fmt.Printf("       \"ADAUSDT\": 2\n")
	fmt.Printf("     }\n\n")

	fmt.Printf("  3. 代码修改：\n")
	fmt.Printf("     // 检查币种计数器\n")
	fmt.Printf("     symbolCount := getSymbolCount(strategy.ProfitScalingSymbolCounts, symbol)\n")
	fmt.Printf("     if symbolCount >= strategy.ProfitScalingMaxCount {\n")
	fmt.Printf("         // 跳过\n")
	fmt.Printf("     }\n\n")

	fmt.Println("方案2：关系表设计")
	fmt.Println("─────────────────────────────────────────────────")
	fmt.Printf("优点：\n")
	fmt.Printf("  ✅ 查询性能好\n")
	fmt.Printf("  ✅ 支持复杂查询\n")
	fmt.Printf("  ✅ 数据一致性好\n\n")

	fmt.Printf("实现方式：\n")
	fmt.Printf("  1. 新建表：strategy_symbol_profit_scaling\n")
	fmt.Printf("     ├── strategy_id (FK)\n")
	fmt.Printf("     ├── symbol (VARCHAR)\n")
	fmt.Printf("     └── current_count (INT)\n")
	fmt.Printf("     └── updated_at (TIMESTAMP)\n\n")

	fmt.Printf("  2. 唯一约束：(strategy_id, symbol)\n\n")

	fmt.Printf("  3. 代码修改：\n")
	fmt.Printf("     // 查询币种计数器\n")
	fmt.Printf("     count := db.Where(\"strategy_id=? AND symbol=?\", strategyID, symbol).\n")
	fmt.Printf("                Select(\"current_count\").First(&count)\n\n")

	fmt.Println("🔧 推荐方案：方案1 (JSON字段)")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("理由：\n")
	fmt.Printf("  • 实现复杂度低\n")
	fmt.Printf("  • 无需数据库迁移\n")
	fmt.Printf("  • 性能足够满足需求\n")
	fmt.Printf("  • 扩展性好\n\n")

	fmt.Println("📝 具体实施步骤：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("步骤1：数据库修改\n")
	fmt.Printf("  ALTER TABLE trading_strategies \n")
	fmt.Printf("  ADD COLUMN profit_scaling_symbol_counts JSON DEFAULT ('{}');\n\n")

	fmt.Printf("步骤2：Go结构体修改\n")
	fmt.Printf("  // 在TradingStrategyConditions中添加\n")
	fmt.Printf("  ProfitScalingSymbolCounts datatypes.JSON `json:\"profit_scaling_symbol_counts\"`\n\n")

	fmt.Printf("步骤3：核心逻辑修改\n")
	fmt.Printf("  // 原代码：策略级别检查\n")
	fmt.Printf("  if strategy.Conditions.ProfitScalingCurrentCount >= strategy.Conditions.ProfitScalingMaxCount\n\n")

	fmt.Printf("  // 新代码：币种级别检查\n")
	fmt.Printf("  symbolCount := getSymbolProfitScalingCount(strategy, symbol)\n")
	fmt.Printf("  if symbolCount >= strategy.Conditions.ProfitScalingMaxCount\n\n")

	fmt.Printf("步骤4：计数器更新逻辑\n")
	fmt.Printf("  // 原代码：策略级别更新\n")
	fmt.Printf("  strategy.Conditions.ProfitScalingCurrentCount++\n\n")

	fmt.Printf("  // 新代码：币种级别更新\n")
	fmt.Printf("  updateSymbolProfitScalingCount(strategy, symbol, count+1)\n\n")

	fmt.Printf("步骤5：重置逻辑修改\n")
	fmt.Printf("  // 整体止损/止盈时：\n")
	fmt.Printf("  // 原：重置整个策略的计数器\n")
	fmt.Printf("  // 新：只重置触发止损/止盈的币种计数器\n")
	fmt.Printf("  resetSymbolProfitScalingCount(strategy, symbol)\n\n")

	fmt.Println("⚠️  兼容性考虑：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("1. 迁移现有数据：\n")
	fmt.Printf("   将现有的ProfitScalingCurrentCount迁移到新的JSON字段\n\n")

	fmt.Printf("2. 向下兼容：\n")
	fmt.Printf("   如果JSON字段为空，使用原有逻辑\n\n")

	fmt.Printf("3. 默认值处理：\n")
	fmt.Printf("   新币种默认计数器为0\n\n")

	fmt.Println("🧪 测试用例：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	testCases := []string{
		"BTCUSDT加仓1次，其他币种仍可加仓",
		"每个币种独立达到最大加仓次数",
		"整体止损只重置该币种的计数器",
		"策略重启时，所有币种计数器重置",
	}

	for i, tc := range testCases {
		fmt.Printf("%d. %s\n", i+1, tc)
	}

	fmt.Printf("\n📊 预期效果对比：\n")
	fmt.Printf("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")

	fmt.Printf("修改前 (策略级别)：\n")
	fmt.Printf("  MaxCount = 1\n")
	fmt.Printf("  BTC加仓1次 → 整个策略无法再加仓\n")
	fmt.Printf("  ETH、ADA都无法加仓\n\n")

	fmt.Printf("修改后 (币种级别)：\n")
	fmt.Printf("  MaxCount = 1\n")
	fmt.Printf("  BTC加仓1次 → BTC无法再加仓\n")
	fmt.Printf("  ETH可以加仓1次，ADA可以加仓1次\n\n")

	fmt.Println("💡 总结：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("通过将计数器从策略级别改为币种级别，\n")
	fmt.Printf("可以实现每个币种独立进行盈利加仓，\n")
	fmt.Printf("提高策略的灵活性和资金利用效率。\n")
}
