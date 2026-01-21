package main

import (
	"fmt"
	"log"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 网格策略完整功能测试 ===")
	fmt.Println("测试交易时间间隔、交易次数和盈利情况")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 1. 策略配置验证
	fmt.Println("\n📋 第一阶段: 策略配置验证")
	validateStrategyConfig(db)

	// 2. 市场数据验证
	fmt.Println("\n📊 第二阶段: 市场数据验证")
	validateMarketData(db)

	// 3. 策略执行模拟
	fmt.Println("\n🔬 第三阶段: 策略执行模拟")
	simulateStrategyExecution(db)

	// 4. 订单创建测试
	fmt.Println("\n📝 第四阶段: 订单创建测试")
	testOrderCreation(db)

	// 5. 交易时间间隔分析
	fmt.Println("\n⏰ 第五阶段: 交易时间间隔分析")
	analyzeTradingIntervals(db)

	// 6. 交易次数统计
	fmt.Println("\n📈 第六阶段: 交易次数统计")
	analyzeTradingFrequency(db)

	// 7. 盈利情况分析
	fmt.Println("\n💰 第七阶段: 盈利情况分析")
	analyzeProfitability(db)

	// 8. 综合绩效评估
	fmt.Println("\n🎯 第八阶段: 综合绩效评估")
	comprehensiveAssessment(db)
}

func validateStrategyConfig(db *gorm.DB) {
	var config map[string]interface{}
	query := `
		SELECT
			grid_trading_enabled,
			grid_upper_price,
			grid_lower_price,
			grid_levels,
			grid_investment_amount,
			grid_stop_loss_enabled,
			grid_stop_loss_percent,
			use_symbol_whitelist,
			symbol_whitelist
		FROM trading_strategies
		WHERE id = 29
	`
	db.Raw(query).Scan(&config)

	fmt.Printf("策略ID 29 配置:\n")
	fmt.Printf("  网格交易启用: %v\n", config["grid_trading_enabled"])
	fmt.Printf("  网格上限价格: %v\n", config["grid_upper_price"])
	fmt.Printf("  网格下限价格: %v\n", config["grid_lower_price"])
	fmt.Printf("  网格层数: %v\n", config["grid_levels"])
	fmt.Printf("  投资金额: %v USDT\n", config["grid_investment_amount"])
	fmt.Printf("  止损启用: %v\n", config["grid_stop_loss_enabled"])
	fmt.Printf("  止损百分比: %v%%\n", config["grid_stop_loss_percent"])
	fmt.Printf("  使用白名单: %v\n", config["use_symbol_whitelist"])
	fmt.Printf("  币种白名单: %v\n", config["symbol_whitelist"])

	// 验证配置有效性
	gridEnabled := config["grid_trading_enabled"]
	if gridEnabled == nil || gridEnabled == false {
		fmt.Printf("❌ 网格交易未启用\n")
		return
	}

	upper := parseFloat(config["grid_upper_price"])
	lower := parseFloat(config["grid_lower_price"])
	levels := parseFloat(config["grid_levels"])

	if upper <= 0 || lower <= 0 || levels <= 0 {
		fmt.Printf("❌ 网格参数无效\n")
		return
	}

	if upper <= lower {
		fmt.Printf("❌ 网格范围无效: 上限(%.4f) <= 下限(%.4f)\n", upper, lower)
		return
	}

	fmt.Printf("✅ 策略配置验证通过\n")
	fmt.Printf("✅ 网格范围: [%.4f, %.4f]\n", lower, upper)
	fmt.Printf("✅ 网格层数: %.0f\n", levels)
}

func validateMarketData(db *gorm.DB) {
	// 检查FILUSDT价格数据
	var priceData map[string]interface{}
	db.Raw("SELECT last_price, volume, created_at FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&priceData)

	currentPrice := parseFloat(priceData["last_price"])
	volume := parseFloat(priceData["volume"])

	fmt.Printf("FILUSDT市场数据:\n")
	fmt.Printf("  当前价格: %.8f USDT\n", currentPrice)
	fmt.Printf("  成交量: %.2f\n", volume)
	fmt.Printf("  更新时间: %v\n", priceData["created_at"])

	if currentPrice <= 0 {
		fmt.Printf("❌ 价格数据无效\n")
		return
	}

	fmt.Printf("✅ 价格数据验证通过\n")

	// 检查技术指标数据
	var techData map[string]interface{}
	db.Raw("SELECT indicators FROM technical_indicators_caches WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&techData)

	if techData["indicators"] != nil {
		fmt.Printf("✅ 技术指标数据存在\n")
	} else {
		fmt.Printf("⚠️ 技术指标数据缺失\n")
	}
}

func simulateStrategyExecution(db *gorm.DB) {
	// 获取配置
	var config map[string]interface{}
	db.Raw("SELECT grid_upper_price, grid_lower_price, grid_levels FROM trading_strategies WHERE id = 29").Scan(&config)

	upper := parseFloat(config["grid_upper_price"])
	lower := parseFloat(config["grid_lower_price"])
	levels := parseFloat(config["grid_levels"])

	// 获取价格
	var priceData map[string]interface{}
	db.Raw("SELECT last_price FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&priceData)
	currentPrice := parseFloat(priceData["last_price"])

	fmt.Printf("策略执行模拟:\n")
	fmt.Printf("  网格范围: [%.4f, %.4f]\n", lower, upper)
	fmt.Printf("  当前价格: %.8f\n", currentPrice)

	// 检查价格是否在范围内
	inRange := currentPrice >= lower && currentPrice <= upper
	fmt.Printf("  价格在范围内: %v\n", inRange)

	if !inRange {
		fmt.Printf("❌ 价格超出网格范围，策略不会执行\n")
		return
	}

	// 计算网格位置
	gridSpacing := (upper - lower) / levels
	gridLevel := int((currentPrice - lower) / gridSpacing)
	if gridLevel >= int(levels) {
		gridLevel = int(levels) - 1
	}
	if gridLevel < 0 {
		gridLevel = 0
	}

	fmt.Printf("  网格层级: %d/%d\n", gridLevel, int(levels))
	fmt.Printf("  网格间距: %.6f\n", gridSpacing)

	// 简化的评分计算
	midLevel := int(levels) / 2
	gridScore := 0.0
	if gridLevel < midLevel {
		gridScore = 1.0 - float64(gridLevel)/float64(midLevel)
	} else if gridLevel > midLevel {
		gridScore = -1.0 * (float64(gridLevel-midLevel) / float64(int(levels)-midLevel))
	}

	techScore := 0.6 // 简化的技术评分
	totalScore := gridScore*0.4 + techScore*0.3

	fmt.Printf("  网格评分: %.3f\n", gridScore)
	fmt.Printf("  技术评分: %.3f\n", techScore)
	fmt.Printf("  综合评分: %.3f\n", totalScore)

	// 决策判断
	if totalScore > 0.2 {
		fmt.Printf("🎯 决策结果: 触发买入信号 ✅\n")
		fmt.Printf("💡 预期: 调度器将创建买入订单\n")
	} else if totalScore < -0.2 {
		fmt.Printf("🎯 决策结果: 触发卖出信号 ✅\n")
		fmt.Printf("💡 预期: 调度器将创建卖出订单\n")
	} else {
		fmt.Printf("🎯 决策结果: 观望\n")
		fmt.Printf("💡 原因: 评分%.3f未达到交易阈值\n", totalScore)
	}
}

func testOrderCreation(db *gorm.DB) {
	// 检查最近的策略执行
	var executions []map[string]interface{}
	db.Raw("SELECT id, status, total_orders, success_orders, failed_orders, created_at FROM strategy_executions WHERE strategy_id = 29 ORDER BY created_at DESC LIMIT 5").Scan(&executions)

	fmt.Printf("最近5次策略执行:\n")
	for _, exec := range executions {
		fmt.Printf("  执行ID: %v, 状态: %v, 订单: %v/%v/%v, 时间: %v\n",
			exec["id"], exec["status"], exec["total_orders"], exec["success_orders"], exec["failed_orders"], exec["created_at"])
	}

	// 检查调度订单
	var orders []map[string]interface{}
	db.Raw("SELECT id, symbol, side, status, quantity, price, created_at FROM scheduled_orders WHERE strategy_id = 29 ORDER BY created_at DESC LIMIT 10").Scan(&orders)

	fmt.Printf("\n最近10个调度订单:\n")
	for _, order := range orders {
		fmt.Printf("  订单ID: %v, 交易对: %v, 方向: %v, 状态: %v, 数量: %v, 价格: %v, 时间: %v\n",
			order["id"], order["symbol"], order["side"], order["status"], order["quantity"], order["price"], order["created_at"])
	}

	// 统计订单状态
	orderStats := make(map[string]int)
	for _, order := range orders {
		status := fmt.Sprintf("%v", order["status"])
		orderStats[status]++
	}

	fmt.Printf("\n订单状态统计:\n")
	for status, count := range orderStats {
		fmt.Printf("  %s: %d\n", status, count)
	}

	if len(orders) == 0 {
		fmt.Printf("⚠️ 暂无调度订单，可能策略还未触发\n")
	}
}

func analyzeTradingIntervals(db *gorm.DB) {
	var orders []map[string]interface{}
	db.Raw("SELECT id, created_at FROM scheduled_orders WHERE strategy_id = 29 AND symbol = 'FILUSDT' AND status = 'FILLED' ORDER BY created_at ASC").Scan(&orders)

	if len(orders) < 2 {
		fmt.Printf("成交订单不足，无法分析时间间隔 (当前成交订单: %d)\n", len(orders))
		return
	}

	fmt.Printf("交易时间间隔分析 (基于%d个已成交订单):\n", len(orders))

	totalInterval := time.Duration(0)
	minInterval := time.Hour * 24 * 365 // 1年
	maxInterval := time.Duration(0)

	for i := 1; i < len(orders); i++ {
		prevTime := parseTime(orders[i-1]["created_at"])
		currTime := parseTime(orders[i]["created_at"])
		interval := currTime.Sub(prevTime)

		fmt.Printf("  订单 %v -> %v: %v\n", orders[i-1]["id"], orders[i]["id"], interval)

		totalInterval += interval
		if interval < minInterval {
			minInterval = interval
		}
		if interval > maxInterval {
			maxInterval = interval
		}
	}

	if len(orders) > 1 {
		avgInterval := totalInterval / time.Duration(len(orders)-1)

		fmt.Printf("\n时间间隔统计:\n")
		fmt.Printf("  平均间隔: %v\n", avgInterval)
		fmt.Printf("  最小间隔: %v\n", minInterval)
		fmt.Printf("  最大间隔: %v\n", maxInterval)

		totalTime := parseTime(orders[len(orders)-1]["created_at"]).Sub(parseTime(orders[0]["created_at"]))
		fmt.Printf("  总观察时间: %v\n", totalTime)
		fmt.Printf("  平均每日交易: %.2f 次\n", float64(len(orders))/totalTime.Hours()*24)
	}
}

func analyzeTradingFrequency(db *gorm.DB) {
	// 统计不同时间段的交易次数
	now := time.Now()

	// 最近24小时
	dayAgo := now.AddDate(0, 0, -1)
	var dayOrders int64
	db.Model(&map[string]interface{}{}).Table("scheduled_orders").
		Where("strategy_id = ? AND symbol = ? AND status = ? AND created_at >= ?", 29, "FILUSDT", "FILLED", dayAgo).
		Count(&dayOrders)

	// 最近7天
	weekAgo := now.AddDate(0, 0, -7)
	var weekOrders int64
	db.Model(&map[string]interface{}{}).Table("scheduled_orders").
		Where("strategy_id = ? AND symbol = ? AND status = ? AND created_at >= ?", 29, "FILUSDT", "FILLED", weekAgo).
		Count(&weekOrders)

	// 最近30天
	monthAgo := now.AddDate(0, -1, 0)
	var monthOrders int64
	db.Model(&map[string]interface{}{}).Table("scheduled_orders").
		Where("strategy_id = ? AND symbol = ? AND status = ? AND created_at >= ?", 29, "FILUSDT", "FILLED", monthAgo).
		Count(&monthOrders)

	fmt.Printf("交易频率统计:\n")
	fmt.Printf("  最近24小时: %d 次\n", dayOrders)
	fmt.Printf("  最近7天: %d 次\n", weekOrders)
	fmt.Printf("  最近30天: %d 次\n", monthOrders)

	if weekOrders > 0 {
		fmt.Printf("  7日平均每日: %.1f 次\n", float64(weekOrders)/7.0)
	}

	if monthOrders > 0 {
		fmt.Printf("  30日平均每日: %.1f 次\n", float64(monthOrders)/30.0)
	}

	// 分析买卖比例
	var buyOrders, sellOrders int64
	db.Model(&map[string]interface{}{}).Table("scheduled_orders").
		Where("strategy_id = ? AND symbol = ? AND status = ?", 29, "FILUSDT", "FILLED").
		Where("side = ?", "BUY").Count(&buyOrders)

	db.Model(&map[string]interface{}{}).Table("scheduled_orders").
		Where("strategy_id = ? AND symbol = ? AND status = ?", 29, "FILUSDT", "FILLED").
		Where("side = ?", "SELL").Count(&sellOrders)

	fmt.Printf("\n买卖统计:\n")
	fmt.Printf("  买入订单: %d\n", buyOrders)
	fmt.Printf("  卖出订单: %d\n", sellOrders)
	fmt.Printf("  买卖比例: %.1f : %.1f\n", float64(buyOrders), float64(sellOrders))

	if buyOrders > sellOrders {
		fmt.Printf("  📈 交易偏向: 买入为主\n")
	} else if sellOrders > buyOrders {
		fmt.Printf("  📉 交易偏向: 卖出为主\n")
	} else {
		fmt.Printf("  ⚖️ 交易偏向: 均衡\n")
	}
}

func analyzeProfitability(db *gorm.DB) {
	// 分析策略执行的盈利情况
	var executions []map[string]interface{}
	db.Raw("SELECT id, total_pnl, win_rate, total_investment, current_value, created_at FROM strategy_executions WHERE strategy_id = 29").Scan(&executions)

	fmt.Printf("策略盈利分析:\n")

	totalExecutions := len(executions)
	profitableExecutions := 0
	totalPnL := 0.0
	totalInvestment := 0.0

	for _, exec := range executions {
		pnl := parseFloat(exec["total_pnl"])
		investment := parseFloat(exec["total_investment"])

		if pnl > 0 {
			profitableExecutions++
		}

		totalPnL += pnl
		totalInvestment += investment
	}

	fmt.Printf("  执行次数: %d\n", totalExecutions)
	fmt.Printf("  盈利执行: %d\n", profitableExecutions)
	if totalExecutions > 0 {
		fmt.Printf("  胜率: %.1f%%\n", float64(profitableExecutions)/float64(totalExecutions)*100)
	}
	fmt.Printf("  总PnL: %.4f USDT\n", totalPnL)
	fmt.Printf("  总投资: %.4f USDT\n", totalInvestment)

	if totalInvestment > 0 {
		fmt.Printf("  总收益率: %.2f%%\n", totalPnL/totalInvestment*100)
	}

	// 分析单个订单的盈利
	var orders []map[string]interface{}
	db.Raw("SELECT id, side, executed_qty, avg_price, created_at FROM scheduled_orders WHERE strategy_id = 29 AND status = 'FILLED' AND executed_qty IS NOT NULL").Scan(&orders)

	fmt.Printf("\n单个订单盈利分析:\n")
	fmt.Printf("  成交订单数: %d\n", len(orders))

	if len(orders) >= 2 {
		// 简化的盈亏计算（假设网格交易的买卖配对）
		buyOrders := []map[string]interface{}{}
		sellOrders := []map[string]interface{}{}

		for _, order := range orders {
			if fmt.Sprintf("%v", order["side"]) == "BUY" {
				buyOrders = append(buyOrders, order)
			} else if fmt.Sprintf("%v", order["side"]) == "SELL" {
				sellOrders = append(sellOrders, order)
			}
		}

		fmt.Printf("  买入订单: %d\n", len(buyOrders))
		fmt.Printf("  卖出订单: %d\n", len(sellOrders))

		// 计算平均买卖价差
		if len(buyOrders) > 0 && len(sellOrders) > 0 {
			avgBuyPrice := 0.0
			avgSellPrice := 0.0

			for _, order := range buyOrders {
				avgBuyPrice += parseFloat(order["avg_price"])
			}
			avgBuyPrice /= float64(len(buyOrders))

			for _, order := range sellOrders {
				avgSellPrice += parseFloat(order["avg_price"])
			}
			avgSellPrice /= float64(len(sellOrders))

			priceDiff := avgSellPrice - avgBuyPrice
			fmt.Printf("  平均买入价: %.8f\n", avgBuyPrice)
			fmt.Printf("  平均卖出价: %.8f\n", avgSellPrice)
			fmt.Printf("  平均价差: %.8f (%.4f%%)\n", priceDiff, priceDiff/avgBuyPrice*100)

			if priceDiff > 0 {
				fmt.Printf("  💰 理论盈利能力: 正向\n")
			} else {
				fmt.Printf("  ⚠️ 理论盈利能力: 负向\n")
			}
		}
	}
}

func comprehensiveAssessment(db *gorm.DB) {
	fmt.Printf("综合绩效评估报告:\n")

	// 获取关键指标
	var executions []map[string]interface{}
	db.Raw("SELECT COUNT(*) as count, SUM(total_pnl) as total_pnl FROM strategy_executions WHERE strategy_id = 29").Scan(&executions)

	var orders []map[string]interface{}
	db.Raw("SELECT COUNT(*) as total, SUM(CASE WHEN status = 'FILLED' THEN 1 ELSE 0 END) as filled FROM scheduled_orders WHERE strategy_id = 29").Scan(&orders)

	totalExecutions := parseFloat(executions[0]["count"])
	totalPnL := parseFloat(executions[0]["total_pnl"])
	totalOrders := parseFloat(orders[0]["total"])
	filledOrders := parseFloat(orders[0]["filled"])

	fmt.Printf("📊 核心指标:\n")
	fmt.Printf("  策略执行次数: %.0f\n", totalExecutions)
	fmt.Printf("  总订单数: %.0f\n", totalOrders)
	fmt.Printf("  成交订单数: %.0f\n", filledOrders)
	fmt.Printf("  总PnL: %.4f USDT\n", totalPnL)

	if totalOrders > 0 {
		fmt.Printf("  订单成交率: %.1f%%\n", filledOrders/totalOrders*100)
	}

	if totalExecutions > 0 {
		fmt.Printf("  平均每次执行订单: %.1f\n", totalOrders/totalExecutions)
	}

	// 评估策略状态
	fmt.Printf("\n🎯 策略状态评估:\n")

	if totalOrders == 0 {
		fmt.Printf("❌ 状态: 策略未产生任何订单\n")
		fmt.Printf("💡 建议: 检查策略配置和触发条件\n")
	} else if filledOrders == 0 {
		fmt.Printf("⚠️ 状态: 有订单创建但全部未成交\n")
		fmt.Printf("💡 建议: 检查订单参数和市场条件\n")
	} else if filledOrders < 10 {
		fmt.Printf("🟡 状态: 交易活动较低\n")
		fmt.Printf("💡 评估: 策略需要调整参数以提高活跃度\n")
	} else {
		fmt.Printf("✅ 状态: 策略正常运行\n")
		fmt.Printf("💡 评估: 交易活跃度良好\n")
	}

	// 盈利能力评估
	fmt.Printf("\n💰 盈利能力评估:\n")

	if totalPnL > 0 {
		fmt.Printf("✅ 总盈利: %.4f USDT\n", totalPnL)
		fmt.Printf("💡 评估: 策略具有盈利能力\n")
	} else if totalPnL < 0 {
		fmt.Printf("❌ 总亏损: %.4f USDT\n", totalPnL)
		fmt.Printf("💡 建议: 调整策略参数或停止运行\n")
	} else {
		fmt.Printf("⚪ 总PnL: 0.00 USDT\n")
		fmt.Printf("💡 评估: 策略盈亏平衡\n")
	}

	// 时间效率评估
	fmt.Printf("\n⏰ 时间效率评估:\n")

	if totalExecutions > 0 {
		now := time.Now()
		var firstExec map[string]interface{}
		db.Raw("SELECT created_at FROM strategy_executions WHERE strategy_id = 29 ORDER BY created_at ASC LIMIT 1").Scan(&firstExec)

		if firstExec["created_at"] != nil {
			startTime := parseTime(firstExec["created_at"])
			runningDays := now.Sub(startTime).Hours() / 24

			fmt.Printf("  运行天数: %.1f 天\n", runningDays)
			fmt.Printf("  日均执行: %.1f 次\n", totalExecutions/runningDays)

			if filledOrders > 0 {
				fmt.Printf("  日均成交: %.1f 单\n", filledOrders/runningDays)
				fmt.Printf("  日均PnL: %.4f USDT\n", totalPnL/runningDays)
			}
		}
	}

	fmt.Printf("\n🏆 最终结论:\n")
	if totalPnL > 0 && filledOrders >= 10 {
		fmt.Printf("🎉 网格策略运行良好，具有稳定的盈利能力\n")
	} else if totalOrders > 0 && filledOrders == 0 {
		fmt.Printf("🔧 策略需要调整订单参数以提高成交率\n")
	} else if totalOrders == 0 {
		fmt.Printf("⚙️ 策略配置或触发条件需要检查和调整\n")
	} else {
		fmt.Printf("📊 策略需要进一步优化参数\n")
	}
}

func parseFloat(val interface{}) float64 {
	if val == nil {
		return 0.0
	}
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int64:
		return float64(v)
	case int:
		return float64(v)
	default:
		return 0.0
	}
}

func parseTime(val interface{}) time.Time {
	if t, ok := val.(time.Time); ok {
		return t
	}
	return time.Now()
}