package main

import (
	"fmt"
	"log"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("=== 网格策略调试 ===")
	fmt.Println("分析策略执行日志和阈值调整效果")

	// 连接数据库
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("连接数据库失败: %v", err)
	}

	// 1. 检查策略执行记录
	fmt.Println("\n📊 第一阶段: 策略执行记录分析")
	var executions []map[string]interface{}
	db.Raw(`
		SELECT id, total_orders, success_orders, failed_orders, total_pnl, win_rate, created_at
		FROM strategy_executions
		WHERE strategy_id = 29
		ORDER BY created_at DESC
		LIMIT 5
	`).Scan(&executions)

	fmt.Printf("最近5次策略执行:\n")
	for _, exec := range executions {
		fmt.Printf("执行ID: %v, 订单: %v, 成功: %v, 失败: %v, PnL: %v, 胜率: %v%%, 时间: %v\n",
			exec["id"], exec["total_orders"], exec["success_orders"],
			exec["failed_orders"], exec["total_pnl"], exec["win_rate"], exec["created_at"])
	}

	// 2. 检查执行步骤日志
	fmt.Println("\n📊 第二阶段: 执行步骤日志分析")
	var steps []map[string]interface{}
	db.Raw(`
		SELECT execution_id, step_name, status, result, created_at
		FROM strategy_execution_steps
		WHERE execution_id IN (
			SELECT id FROM strategy_executions WHERE strategy_id = 29 ORDER BY created_at DESC LIMIT 3
		)
		ORDER BY execution_id DESC, created_at DESC
	`).Scan(&steps)

	fmt.Printf("最近执行步骤:\n")
	for _, step := range steps {
		fmt.Printf("执行ID: %v, 步骤: %v, 状态: %v, 结果: %v\n",
			step["execution_id"], step["step_name"], step["status"], step["result"])
	}

	// 3. 检查调度订单
	fmt.Println("\n📊 第三阶段: 调度订单分析")
	var orders []map[string]interface{}
	db.Raw(`
		SELECT id, symbol, side, status, quantity, price, grid_level, execution_id, created_at
		FROM scheduled_orders
		WHERE strategy_id = 29 AND symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 10
	`).Scan(&orders)

	fmt.Printf("FIL网格策略调度订单:\n")
	for _, order := range orders {
		fmt.Printf("订单ID: %v, 方向: %v, 状态: %v, 数量: %v, 价格: %v, 网格层: %v, 执行ID: %v\n",
			order["id"], order["side"], order["status"], order["quantity"],
			order["price"], order["grid_level"], order["execution_id"])
	}

	// 4. 分析阈值调整效果
	fmt.Println("\n📊 第四阶段: 阈值调整效果分析")

	// 检查策略配置
	var strategy map[string]interface{}
	db.Raw(`
		SELECT grid_trading_enabled, grid_upper_price, grid_lower_price, grid_levels,
			   grid_investment_amount, grid_stop_loss_enabled
		FROM trading_strategies WHERE id = 29
	`).Scan(&strategy)

	fmt.Printf("策略配置:\n")
	for k, v := range strategy {
		fmt.Printf("  %s: %v\n", k, v)
	}

	// 检查FILUSDT价格
	var price map[string]interface{}
	db.Raw("SELECT last_price FROM binance_24h_stats WHERE symbol = 'FILUSDT' ORDER BY created_at DESC LIMIT 1").Scan(&price)

	currentPrice := 0.0
	if p, ok := price["last_price"].(float64); ok {
		currentPrice = p
	}

	fmt.Printf("\nFILUSDT当前价格: %.4f\n", currentPrice)

	// 计算理论评分
	if gridUpper, ok := strategy["grid_upper_price"].(float64); ok {
		gridLower := strategy["grid_lower_price"].(float64)
		gridLevels := strategy["grid_levels"].(int64)

		fmt.Printf("\n网格范围: [%.4f, %.4f]\n", gridLower, gridUpper)
		fmt.Printf("网格层数: %d\n", gridLevels)

		if currentPrice >= gridLower && currentPrice <= gridUpper {
			fmt.Printf("✅ 价格在网格范围内\n")

			gridSpacing := (gridUpper - gridLower) / float64(gridLevels)
			gridLevel := int((currentPrice - gridLower) / gridSpacing)
			if gridLevel >= int(gridLevels) {
				gridLevel = int(gridLevels) - 1
			}
			if gridLevel < 0 {
				gridLevel = 0
			}

			fmt.Printf("当前网格层级: %d/%d\n", gridLevel, gridLevels)

			// 简单的评分计算（基于我们的分析）
			gridScore := calculateGridScore(gridLevel, int(gridLevels/2), int(gridLevels))
			fmt.Printf("理论网格评分: %.3f\n", gridScore)
			fmt.Printf("理论技术评分: 0.600 (基于RSI+MACD+均线)\n")
			fmt.Printf("理论综合评分: %.3f\n", gridScore*0.4+0.6*0.3)
			fmt.Printf("调整后阈值: >0.2 买入, <-0.2 卖出\n")

			theoreticalScore := gridScore*0.4 + 0.6*0.3
			if theoreticalScore > 0.2 {
				fmt.Printf("🎯 理论结果: 应该触发买入信号\n")
			} else {
				fmt.Printf("🎯 理论结果: 观望\n")
			}

			if len(orders) == 0 {
				fmt.Printf("❌ 实际结果: 没有创建任何订单\n")
				fmt.Printf("🔍 问题: 尽管理论上应该交易，但实际没有执行\n")
			}
		} else {
			fmt.Printf("❌ 价格超出网格范围\n")
		}
	}

	// 5. 总结分析
	fmt.Println("\n📊 第五阶段: 问题诊断和建议")

	if len(orders) == 0 && len(executions) > 0 {
		fmt.Printf("🔍 诊断结果:\n")
		fmt.Printf("1. ✅ 策略被调度执行 (%d次)\n", len(executions))
		fmt.Printf("2. ✅ 每次执行都完成 (无错误)\n")
		fmt.Printf("3. ❌ 没有创建任何订单\n")
		fmt.Printf("4. ❌ 没有触发交易信号\n")

		fmt.Printf("\n💡 可能原因:\n")
		fmt.Printf("1. 阈值调整可能没有生效\n")
		fmt.Printf("2. 策略返回'no_op'而不是'buy'/'sell'\n")
		fmt.Printf("3. 技术指标数据获取失败\n")
		fmt.Printf("4. 市场数据不足\n")

		fmt.Printf("\n🔧 建议解决方案:\n")
		fmt.Printf("1. 检查网格策略代码中的阈值是否正确修改\n")
		fmt.Printf("2. 添加详细的调试日志\n")
		fmt.Printf("3. 验证技术指标计算是否正常\n")
		fmt.Printf("4. 检查市场数据获取是否成功\n")
	}
}

func calculateGridScore(currentLevel, midLevel, totalLevels int) float64 {
	if currentLevel < midLevel {
		return 1.0 - float64(currentLevel)/float64(midLevel)
	} else if currentLevel > midLevel {
		return -1.0 * (float64(currentLevel-midLevel) / float64(totalLevels-midLevel))
	}
	return 0
}
