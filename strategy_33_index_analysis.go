package main

import "fmt"

func main() {
	fmt.Println("=== 策略ID 33执行流程索引分析 ===\n")

	fmt.Println("📊 策略33执行涉及的主要查询模式：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	queries := []struct {
		stage        string
		table        string
		condition    string
		currentIndex string
		issue        string
		recommend    string
	}{
		{
			"策略调度",
			"trading_strategies",
			"is_running = true AND (last_run_at IS NULL OR last_run_at + INTERVAL run_interval MINUTE <= NOW())",
			"无专门索引",
			"复合条件查询，缺乏索引支持",
			"添加 (is_running, last_run_at) 复合索引",
		},
		{
			"涨幅榜查询",
			"binance_24h_stats",
			"market_type = 'futures' ORDER BY price_change_percent DESC LIMIT 50",
			"idx_24h_stats_symbol_market (symbol, market_type)",
			"需要按涨幅排序，现有索引不支持",
			"添加 (market_type, price_change_percent) 索引",
		},
		{
			"24h平仓过滤",
			"scheduled_orders",
			"strategy_id = ? AND symbol = ? AND status = 'filled' AND reduce_only = true AND created_at >= ?",
			"idx_so_user_status (user_id, status)",
			"缺少strategy_id、symbol、reduce_only的索引支持",
			"添加 (strategy_id, symbol, reduce_only, created_at) 复合索引",
		},
		{
			"持仓验证",
			"scheduled_orders",
			"strategy_id = ? AND symbol = ? AND status = 'filled' AND reduce_only = false",
			"idx_so_user_status (user_id, status)",
			"缺少strategy_id、symbol的索引支持",
			"添加 (strategy_id, symbol, status, reduce_only) 复合索引",
		},
		{
			"订单到期检查",
			"scheduled_orders",
			"status = 'pending' AND trigger_time <= NOW() ORDER BY trigger_time ASC LIMIT 20",
			"idx_so_status_trigger (status, trigger_time)",
			"现有索引已覆盖，但可以考虑添加更高效的索引",
			"保持现有索引，考虑添加 (trigger_time, status) 作为辅助索引",
		},
		{
			"盈利加仓计数",
			"trading_strategies",
			"id = ?",
			"主键索引",
			"单行查询，性能良好",
			"无需额外索引",
		},
		{
			"整体止损检查",
			"scheduled_orders",
			"strategy_id = ? AND symbol = ? AND status = 'filled'",
			"缺少strategy_id、symbol的索引支持",
			"添加 (strategy_id, symbol, status) 复合索引",
		},
		{
			"价格查询",
			"price_caches",
			"symbol = ? AND kind = ?",
			"idx_price_caches_symbol_kind (symbol, kind)",
			"现有索引已覆盖",
			"无需额外索引",
		},
	}

	fmt.Printf("%-12s | %-18s | %-40s | %-30s | %-25s\n", "执行阶段", "查询表", "查询条件", "当前索引", "优化建议")
	fmt.Println("────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────────")

	for _, q := range queries {
		fmt.Printf("%-12s | %-18s | %-40s | %-30s | %-25s\n",
			q.stage, q.table, q.condition[:min(40, len(q.condition))], q.currentIndex, q.recommend)
	}

	fmt.Println("\n🎯 关键优化索引推荐：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// 分析最关键的索引缺失
	criticalIndexes := []struct {
		table     string
		indexName string
		columns   []string
		impact    string
		priority  string
	}{
		{
			"trading_strategies",
			"idx_strategies_running_last_run",
			[]string{"is_running", "last_run_at"},
			"优化策略调度查询，避免全表扫描",
			"高",
		},
		{
			"binance_24h_stats",
			"idx_24h_stats_market_change",
			[]string{"market_type", "price_change_percent"},
			"大幅提升涨幅榜查询性能",
			"高",
		},
		{
			"scheduled_orders",
			"idx_orders_strategy_symbol_reduce",
			[]string{"strategy_id", "symbol", "reduce_only", "created_at"},
			"优化24小时平仓过滤查询",
			"高",
		},
		{
			"scheduled_orders",
			"idx_orders_strategy_symbol_status",
			[]string{"strategy_id", "symbol", "status"},
			"优化持仓验证和整体止损检查",
			"高",
		},
		{
			"scheduled_orders",
			"idx_orders_trigger_time_status",
			[]string{"trigger_time", "status"},
			"进一步优化订单到期检查",
			"中",
		},
	}

	for i, idx := range criticalIndexes {
		fmt.Printf("%d. %s.%s (%v)\n", i+1, idx.table, idx.indexName, idx.columns)
		fmt.Printf("   影响：%s\n", idx.impact)
		fmt.Printf("   优先级：%s\n\n", idx.priority)
	}

	fmt.Println("📈 性能提升预期：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("• 策略调度查询：从全表扫描 → 索引扫描，提升 90%+\n")
	fmt.Printf("• 涨幅榜查询：从无索引 → 复合索引，提升 80%+\n")
	fmt.Printf("• 平仓过滤查询：从多条件扫描 → 复合索引，提升 85%+\n")
	fmt.Printf("• 持仓验证查询：从索引低效 → 复合索引，提升 75%+\n")
	fmt.Printf("• 整体响应时间：预计整体提升 60-70%\n\n")

	fmt.Println("⚠️ 实施注意事项：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	fmt.Printf("1. 索引添加顺序：\n")
	fmt.Printf("   • 先添加对查询影响最大的索引\n")
	fmt.Printf("   • 在业务低峰期执行索引创建\n")
	fmt.Printf("   • 监控索引创建过程中的性能影响\n\n")

	fmt.Printf("2. 索引维护：\n")
	fmt.Printf("   • 定期分析索引使用情况\n")
	fmt.Printf("   • 删除不再使用的索引\n")
	fmt.Printf("   • 监控索引碎片并重建\n\n")

	fmt.Printf("3. 监控指标：\n")
	fmt.Printf("   • 查询执行时间\n")
	fmt.Printf("   • 索引命中率\n")
	fmt.Printf("   • 数据库CPU使用率\n")
	fmt.Printf("   • 慢查询数量\n\n")

	fmt.Println("💡 结论：")
	fmt.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Printf("通过添加4-5个关键索引，可以显著提升策略ID 33的执行性能，\n")
	fmt.Printf("尤其是在高频交易场景下，索引优化带来的性能提升至关重要。\n")
	fmt.Printf("建议按照优先级逐步实施，并在生产环境充分测试后上线。\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
