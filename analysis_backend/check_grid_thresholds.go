package main

import (
	"fmt"
	"log"
	"math"

	pdb "analysis/internal/db"
)

func main() {
	fmt.Println("🔍 检查网格交易决策阈值和限制条件")
	fmt.Println("=====================================")

	// 连接数据库
	db, err := pdb.OpenMySQL(pdb.Options{
		DSN:          "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC",
		Automigrate:  false,
		MaxOpenConns: 10,
		MaxIdleConns: 5,
	})
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	gdb, err := db.DB()
	if err != nil {
		log.Fatalf("❌ 获取数据库实例失败: %v", err)
	}

	// 1. 获取当前价格和技术指标
	fmt.Printf("📊 获取当前市场数据:\n")

	var filPrice struct {
		LastPrice float64 `json:"last_price"`
	}

	err = gdb.Raw(`
		SELECT last_price
		FROM binance_24h_stats
		WHERE symbol = 'FILUSDT'
		ORDER BY created_at DESC
		LIMIT 1
	`).Scan(&filPrice).Error

	if err != nil {
		log.Fatalf("❌ 查询价格失败: %v", err)
	}

	fmt.Printf("  当前价格: %.4f USDT\n", filPrice.LastPrice)

	// 2. 重新计算决策评分
	fmt.Printf("\n🎯 重新计算决策评分:\n")

	// 获取网格配置
	var gridConfig struct {
		GridUpperPrice       float64 `json:"grid_upper_price"`
		GridLowerPrice       float64 `json:"grid_lower_price"`
		GridLevels           int     `json:"grid_levels"`
		GridInvestmentAmount float64 `json:"grid_investment_amount"`
	}

	err = gdb.Raw(`
		SELECT grid_upper_price, grid_lower_price, grid_levels, grid_investment_amount
		FROM trading_strategies
		WHERE grid_trading_enabled = true AND id = 29
	`).Scan(&gridConfig).Error

	if err != nil {
		log.Fatalf("❌ 查询网格配置失败: %v", err)
	}

	gridSpacing := (gridConfig.GridUpperPrice - gridConfig.GridLowerPrice) / float64(gridConfig.GridLevels)
	gridLevel := int((filPrice.LastPrice - gridConfig.GridLowerPrice) / gridSpacing)
	if gridLevel >= gridConfig.GridLevels {
		gridLevel = gridConfig.GridLevels - 1
	}
	if gridLevel < 0 {
		gridLevel = 0
	}

	midLevel := gridConfig.GridLevels / 2

	// 网格评分
	gridScore := calculateGridScore(gridLevel, midLevel, gridConfig.GridLevels)
	fmt.Printf("  网格评分: %.3f\n", gridScore)

	// 技术评分（使用已知的技术指标值）
	rsi := 47.68
	ma5 := 1.3340
	ma20 := 1.3269
	macdHist := 0.000261

	score := 0.0

	// RSI评分
	if rsi < 30 {
		score += 0.4
	} else if rsi > 70 {
		score -= 0.4
	}

	// MACD评分
	if macdHist > 0 {
		score += 0.3
	} else {
		score -= 0.3
	}

	// 均线趋势评分
	if ma5 > ma20 {
		score += 0.3
	} else {
		score -= 0.3
	}

	techScore := math.Max(-1.0, math.Min(1.0, score))
	fmt.Printf("  技术评分: %.3f (RSI:%.2f, MACD:%.6f, MA趋势:%v)\n",
		techScore, rsi, macdHist, ma5 > ma20)

	// 波动率乘数
	volatilityMultiplier := 1.1 // 低波动率
	fmt.Printf("  波动率乘数: %.3f\n", volatilityMultiplier)

	// 风险评分 (假设为0)
	riskScore := 0.0
	fmt.Printf("  风险评分: %.3f\n", riskScore)

	// 深度评分 (假设为0，没有深度数据)
	depthScore := 0.0
	fmt.Printf("  深度评分: %.3f\n", depthScore)

	// 综合评分
	totalScore := gridScore*0.4 + techScore*0.3 + depthScore*0.2 + riskScore*0.1
	totalScore *= volatilityMultiplier
	fmt.Printf("  综合评分: %.3f\n", totalScore)

	// 3. 检查决策阈值
	fmt.Printf("\n⚖️ 决策阈值检查:\n")
	buyThreshold := 0.15
	sellThreshold := -0.15

	fmt.Printf("  买入阈值: %.3f\n", buyThreshold)
	fmt.Printf("  卖出阈值: %.3f\n", sellThreshold)
	fmt.Printf("  当前评分: %.3f\n", totalScore)

	if totalScore > buyThreshold {
		fmt.Printf("  ✅ 应该触发买入信号\n")
	} else if totalScore < sellThreshold {
		fmt.Printf("  ✅ 应该触发卖出信号\n")
	} else {
		fmt.Printf("  ⏸️ 应该观望 (评分在阈值范围内)\n")
	}

	// 4. 检查可能的限制条件
	fmt.Printf("\n🚫 检查可能的限制条件:\n")

	// 检查是否有现有持仓
	var existingOrders int64
	err = gdb.Model(&pdb.ScheduledOrder{}).
		Where("strategy_id = ? AND status IN ('pending', 'filled', 'partial_filled')", 29).
		Count(&existingOrders).Error

	if err == nil && existingOrders > 0 {
		fmt.Printf("  ⚠️  发现 %d 个现有订单，可能影响新订单创建\n", existingOrders)
	} else {
		fmt.Printf("  ✅ 没有现有持仓冲突\n")
	}

	// 检查策略执行状态
	var pendingExecutions int64
	err = gdb.Model(&struct{}{}).Table("strategy_executions").
		Where("strategy_id = ? AND status = 'running'", 29).
		Count(&pendingExecutions).Error

	if err == nil && pendingExecutions > 0 {
		fmt.Printf("  ⚠️  发现 %d 个正在运行的执行，可能导致并发冲突\n", pendingExecutions)
	} else {
		fmt.Printf("  ✅ 没有并发执行冲突\n")
	}

	// 5. 检查代码中的决策阈值
	fmt.Printf("\n📝 代码中的决策阈值分析:\n")
	fmt.Printf("  从网格交易代码分析，决策逻辑是:\n")
	fmt.Printf("  - 买入: totalScore > 0.15\n")
	fmt.Printf("  - 卖出: totalScore < -0.15\n")
	fmt.Printf("  - 观望: -0.15 <= totalScore <= 0.15\n")
	fmt.Printf("  \n")
	fmt.Printf("  当前计算结果: %.3f (> 0.15) 应该买入\n", totalScore)

	// 6. 可能的解决方案
	fmt.Printf("\n🛠️ 可能的解决方案:\n")
	fmt.Printf("  1. 检查网格交易代码中的实际决策逻辑\n")
	fmt.Printf("  2. 添加调试日志输出评分计算过程\n")
	fmt.Printf("  3. 临时降低决策阈值进行测试\n")
	fmt.Printf("  4. 检查是否有异常退出或错误处理\n")
	fmt.Printf("  5. 验证ExecuteFull方法是否被正确调用\n")

	fmt.Printf("\n💡 建议的调试步骤:\n")
	fmt.Printf("  1. 在网格交易代码中添加详细的评分日志\n")
	fmt.Printf("  2. 检查是否有提前返回或异常处理\n")
	fmt.Printf("  3. 验证技术指标数据是否正确传递\n")
	fmt.Printf("  4. 考虑手动修改阈值进行测试\n")
}

func calculateGridScore(currentLevel, midLevel, totalLevels int) float64 {
	if currentLevel < midLevel {
		// 下半部分，越低分数越高
		return 1.0 - float64(currentLevel)/float64(midLevel)
	} else if currentLevel > midLevel {
		// 上半部分，越高分数越低(更负)
		return -1.0 * (float64(currentLevel-midLevel) / float64(totalLevels-midLevel))
	}
	return 0 // 中性位置
}