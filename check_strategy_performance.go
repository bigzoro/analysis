package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"
)

func main() {
	fmt.Println("🔍 检查策略21的实际交易数据")
	fmt.Println("===========================")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 检查各个交易记录表的策略21数据
	tables := map[string]string{
		"strategy_executions": "策略执行记录",
		"simulated_trades":    "模拟交易记录",
		"backtest_records":    "回测记录",
		"async_backtest_records": "异步回测记录",
	}

	fmt.Println("\n📊 策略21在各表中的记录数量:")
	totalRecords := 0
	for table, desc := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE strategy_id = ?", table)
		err := db.QueryRow(query, 21).Scan(&count)
		if err != nil {
			fmt.Printf("  %s (%s): 查询失败 - %v\n", table, desc, err)
		} else {
			fmt.Printf("  %s (%s): %d 条记录\n", table, desc, count)
			totalRecords += count
		}
	}

	fmt.Printf("\n📈 总记录数: %d\n", totalRecords)

	if totalRecords == 0 {
		fmt.Println("\n❌ 策略21没有实际交易记录，无法进行真实表现分析")
		fmt.Println("建议：")
		fmt.Println("1. 检查策略是否曾经启用运行")
		fmt.Println("2. 查看策略配置是否有问题")
		fmt.Println("3. 考虑进行历史回测来评估策略效果")
		return
	}

	// 如果有记录，显示详细的交易统计
	fmt.Println("\n📋 详细交易分析:")

	// 检查strategy_executions表
	var execCount int
	db.QueryRow("SELECT COUNT(*) FROM strategy_executions WHERE strategy_id = ?", 21).Scan(&execCount)

	if execCount > 0 {
		fmt.Println("\n🎯 策略执行记录分析:")

		// 统计执行状态
		statusQuery := `
			SELECT status, COUNT(*) as count
			FROM strategy_executions
			WHERE strategy_id = ?
			GROUP BY status
			ORDER BY count DESC`

		rows, err := db.Query(statusQuery, 21)
		if err == nil {
			defer rows.Close()
			fmt.Println("  执行状态分布:")
			for rows.Next() {
				var status string
				var count int
				rows.Scan(&status, &count)
				fmt.Printf("    %s: %d 次\n", status, count)
			}
		}

		// 统计时间范围
		timeQuery := `
			SELECT MIN(created_at) as start_time, MAX(created_at) as end_time, COUNT(*) as total
			FROM strategy_executions
			WHERE strategy_id = ?`

		var startTime, endTime string
		var total int
		err = db.QueryRow(timeQuery, 21).Scan(&startTime, &endTime, &total)
		if err == nil {
			fmt.Printf("  时间范围: %s 至 %s\n", startTime, endTime)
			fmt.Printf("  总执行次数: %d\n", total)
		}
	}

	// 检查simulated_trades表
	var tradeCount int
	db.QueryRow("SELECT COUNT(*) FROM simulated_trades WHERE strategy_id = ?", 21).Scan(&tradeCount)

	if tradeCount > 0 {
		fmt.Println("\n💼 模拟交易记录分析:")

		// 统计盈亏
		pnlQuery := `
			SELECT
				COUNT(*) as total_trades,
				SUM(CASE WHEN pnl > 0 THEN 1 ELSE 0 END) as profitable_trades,
				SUM(CASE WHEN pnl < 0 THEN 1 ELSE 0 END) as losing_trades,
				AVG(pnl) as avg_pnl,
				SUM(pnl) as total_pnl,
				MAX(pnl) as best_trade,
				MIN(pnl) as worst_trade
			FROM simulated_trades
			WHERE strategy_id = ?`

		var totalTrades, profitableTrades, losingTrades int
		var avgPnl, totalPnl, bestTrade, worstTrade float64

		err := db.QueryRow(pnlQuery, 21).Scan(&totalTrades, &profitableTrades, &losingTrades,
			&avgPnl, &totalPnl, &bestTrade, &worstTrade)
		if err == nil {
			winRate := float64(profitableTrades) / float64(totalTrades) * 100

			fmt.Printf("  总交易次数: %d\n", totalTrades)
			fmt.Printf("  盈利交易: %d (%.1f%%)\n", profitableTrades, winRate)
			fmt.Printf("  亏损交易: %d (%.1f%%)\n", losingTrades, 100-winRate)
			fmt.Printf("  平均盈亏: %.4f%%\n", avgPnl)
			fmt.Printf("  总盈亏: %.4f%%\n", totalPnl)
			fmt.Printf("  最佳交易: %.4f%%\n", bestTrade)
			fmt.Printf("  最差交易: %.4f%%\n", worstTrade)

			if totalTrades > 0 {
				// 计算简化的夏普比率（需要更多数据来准确计算）
				fmt.Printf("  胜率: %.1f%%\n", winRate)

				// 估算最大回撤（简化计算）
				if worstTrade < 0 {
					fmt.Printf("  最大单笔亏损: %.2f%%\n", worstTrade)
				}
			}
		}
	}

	// 检查backtest_records表
	var backtestCount int
	db.QueryRow("SELECT COUNT(*) FROM backtest_records WHERE strategy_id = ?", 21).Scan(&backtestCount)

	if backtestCount > 0 {
		fmt.Println("\n🔄 回测记录分析:")

		backtestQuery := `
			SELECT
				COUNT(*) as total_tests,
				AVG(CASE WHEN performance_data LIKE '%win_rate%' THEN 0.5 ELSE 0 END) as avg_performance,
				MAX(created_at) as latest_test
			FROM backtest_records
			WHERE strategy_id = ?`

		var totalTests int
		var avgPerformance float64
		var latestTest string

		err := db.QueryRow(backtestQuery, 21).Scan(&totalTests, &avgPerformance, &latestTest)
		if err == nil {
			fmt.Printf("  总回测次数: %d\n", totalTests)
			fmt.Printf("  最新回测: %s\n", latestTest)
		}
	}

	// 分析市场环境对策略的影响
	fmt.Println("\n🌍 市场环境影响分析:")

	// 获取策略运行期间的市场数据
	marketQuery := `
		SELECT
			AVG(price_change_percent) as avg_change,
			STDDEV(price_change_percent) as volatility,
			COUNT(CASE WHEN price_change_percent > 5 THEN 1 END) as bull_days,
			COUNT(CASE WHEN price_change_percent < -5 THEN 1 END) as bear_days,
			COUNT(*) as total_days
		FROM binance_24h_stats
		WHERE created_at >= '2025-12-26 00:00:00'
			AND created_at <= '2025-12-27 00:00:00'
			AND market_type = 'spot'
			AND quote_volume > 100000`

	var avgChange, volatility float64
	var bullDays, bearDays, totalDays int

	err = db.QueryRow(marketQuery).Scan(&avgChange, &volatility, &bullDays, &bearDays, &totalDays)
	if err == nil && totalDays > 0 {
		bullRatio := float64(bullDays) / float64(totalDays) * 100
		bearRatio := float64(bearDays) / float64(totalDays) * 100

		fmt.Printf("  策略运行期间市场概况:\n")
		fmt.Printf("    平均涨跌幅: %.2f%%\n", avgChange)
		fmt.Printf("    市场波动率: %.2f%%\n", volatility)
		fmt.Printf("    多头行情天数: %d (%.1f%%)\n", bullDays, bullRatio)
		fmt.Printf("    空头行情天数: %d (%.1f%%)\n", bearDays, bearRatio)

		// 分析策略适合度
		if bullRatio > 60 {
			fmt.Printf("    🎯 市场环境评估: 强势上涨趋势 - 做空策略非常不利\n")
		} else if bearRatio > 60 {
			fmt.Printf("    🎯 市场环境评估: 强势下跌趋势 - 做空策略可能有利\n")
		} else {
			fmt.Printf("    🎯 市场环境评估: 震荡市场 - 做空策略相对合适\n")
		}
	}

	fmt.Println("\n🎉 数据检查完成！")
}