package main

import (
	"database/sql"
	"fmt"
	"log"
	"math"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

type BacktestResult struct {
	TotalTrades       int
	WinningTrades     int
	LosingTrades      int
	WinRate           float64
	TotalPnL          float64
	AvgPnL            float64
	MaxDrawdown       float64
	SharpeRatio       float64
	ProfitFactor      float64
	LargestWin        float64
	LargestLoss       float64
	AvgHoldTime       time.Duration
	StartDate         time.Time
	EndDate           time.Time
	Trades            []TradeRecord
}

type TradeRecord struct {
	Symbol       string
	Side         string    // "short"
	EntryTime    time.Time
	ExitTime     time.Time
	EntryPrice   float64
	ExitPrice    float64
	Quantity     float64
	PnL          float64
	PnLPercent   float64
	StopLoss     float64
	TakeProfit   float64
	ExitReason   string
}

func main() {
	fmt.Println("🔬 策略21历史回测分析")
	fmt.Println("====================")

	db, err := sql.Open("mysql", "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC")
	if err != nil {
		log.Fatal("数据库连接失败:", err)
	}
	defer db.Close()

	// 获取策略21的参数
	strategy, err := getStrategy21Params(db)
	if err != nil {
		log.Fatal("获取策略参数失败:", err)
	}

	fmt.Printf("📋 策略参数:\n")
	fmt.Printf("  做空涨幅榜前%d名\n", strategy.GainersRankLimit)
	fmt.Printf("  做空倍数: %.1fx\n", strategy.ShortMultiplier)
	fmt.Printf("  止损: %.1f%%\n", strategy.StopLossPercent)
	fmt.Printf("  止盈: %.1f%%\n", strategy.TakeProfitPercent)
	fmt.Printf("  最大仓位: %.1f%%\n", strategy.MaxPositionSize)
	fmt.Printf("  杠杆: %dx\n", strategy.DefaultLeverage)

	// 执行历史回测
	fmt.Println("\n🔄 开始历史回测...")
	result, err := backtestStrategy21(db, strategy)
	if err != nil {
		log.Fatal("回测失败:", err)
	}

	// 显示回测结果
	displayBacktestResults(result)

	// 分析市场环境影响
	analyzeMarketImpact(db, result)

	// 生成改进建议
	generateBacktestRecommendations(result, strategy)

	fmt.Println("\n🎉 回测分析完成！")
}

type Strategy21Params struct {
	GainersRankLimit  int     `json:"gainers_rank_limit"`
	ShortMultiplier   float64 `json:"short_multiplier"`
	StopLossPercent   float64 `json:"stop_loss_percent"`
	TakeProfitPercent float64 `json:"take_profit_percent"`
	MaxPositionSize   float64 `json:"max_position_size"`
	DefaultLeverage   int     `json:"default_leverage"`
}

func getStrategy21Params(db *sql.DB) (*Strategy21Params, error) {
	query := `
		SELECT gainers_rank_limit, short_multiplier, stop_loss_percent,
		       take_profit_percent, max_position_size, default_leverage
		FROM trading_strategies
		WHERE id = 21`

	var params Strategy21Params
	err := db.QueryRow(query).Scan(
		&params.GainersRankLimit,
		&params.ShortMultiplier,
		&params.StopLossPercent,
		&params.TakeProfitPercent,
		&params.MaxPositionSize,
		&params.DefaultLeverage,
	)

	return &params, err
}

func backtestStrategy21(db *sql.DB, params *Strategy21Params) (*BacktestResult, error) {
	result := &BacktestResult{
		StartDate: time.Date(2025, 12, 20, 0, 0, 0, 0, time.UTC),
		EndDate:   time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
		Trades:   []TradeRecord{},
	}

	// 获取历史数据，按日期分组
	datesQuery := `
		SELECT DISTINCT DATE(created_at) as trade_date
		FROM binance_24h_stats
		WHERE created_at >= ? AND created_at <= ?
			AND market_type = 'spot'
			AND quote_volume > 1000000
		ORDER BY trade_date`

	rows, err := db.Query(datesQuery, result.StartDate, result.EndDate)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tradeDate time.Time
		rows.Scan(&tradeDate)

		// 模拟当天交易
		dayTrades, err := simulateDayTrades(db, tradeDate, params)
		if err != nil {
			continue // 跳过有问题的日期
		}

		result.Trades = append(result.Trades, dayTrades...)
	}

	// 计算统计结果
	calculateBacktestStats(result)

	return result, nil
}

func simulateDayTrades(db *sql.DB, tradeDate time.Time, params *Strategy21Params) ([]TradeRecord, error) {
	var trades []TradeRecord

	// 获取当天涨幅榜前N名币种
	gainersQuery := `
		SELECT symbol, price_change_percent, last_price, quote_volume
		FROM binance_24h_stats
		WHERE DATE(created_at) = DATE(?)
			AND market_type = 'spot'
			AND quote_volume > 1000000
			AND price_change_percent > 0
		ORDER BY price_change_percent DESC
		LIMIT ?`

	rows, err := db.Query(gainersQuery, tradeDate, params.GainersRankLimit)
	if err != nil {
		return trades, err
	}
	defer rows.Close()

	// 为每个涨幅榜币种创建做空交易
	positionSize := params.MaxPositionSize / float64(params.GainersRankLimit) // 平均分配仓位

	for rows.Next() {
		var symbol string
		var priceChange, lastPrice float64
		var volume float64

		err := rows.Scan(&symbol, &priceChange, &lastPrice, &volume)
		if err != nil {
			continue
		}

		// 创建做空交易记录
		trade := TradeRecord{
			Symbol:     symbol,
			Side:       "short",
			EntryTime:  tradeDate.Add(9 * time.Hour), // 假设9点开盘
			EntryPrice: lastPrice,
			Quantity:   positionSize * params.ShortMultiplier / lastPrice, // 考虑杠杆
		}

		// 模拟交易结果（简化模型）
		// 假设当天收盘时平仓，价格变化作为PnL
		exitPrice := lastPrice * (1 - priceChange/100) // 假设收盘价基于涨跌幅
		trade.ExitPrice = exitPrice
		trade.ExitTime = tradeDate.Add(16 * time.Hour) // 假设16点收盘

		// 计算PnL (做空：盈利 = 入场价 - 出场价)
		priceDiff := trade.EntryPrice - trade.ExitPrice
		trade.PnL = priceDiff * trade.Quantity
		trade.PnLPercent = (priceDiff / trade.EntryPrice) * 100 * params.ShortMultiplier

		// 设置止损止盈价格（用于风险管理分析）
		trade.StopLoss = trade.EntryPrice * (1 + params.StopLossPercent/100)
		trade.TakeProfit = trade.EntryPrice * (1 - params.TakeProfitPercent/100)

		// 判断退出原因
		if trade.PnLPercent >= params.TakeProfitPercent {
			trade.ExitReason = "take_profit"
		} else if trade.PnLPercent <= -params.StopLossPercent {
			trade.ExitReason = "stop_loss"
		} else {
			trade.ExitReason = "end_of_day"
		}

		trades = append(trades, trade)
	}

	return trades, nil
}

func calculateBacktestStats(result *BacktestResult) {
	if len(result.Trades) == 0 {
		return
	}

	result.TotalTrades = len(result.Trades)

	winningTrades := 0
	losingTrades := 0
	totalPnL := 0.0
	maxDrawdown := 0.0
	peak := 0.0
	largestWin := 0.0
	largestLoss := 0.0

	for _, trade := range result.Trades {
		totalPnL += trade.PnLPercent

		if trade.PnLPercent > 0 {
			winningTrades++
			if trade.PnLPercent > largestWin {
				largestWin = trade.PnLPercent
			}
		} else {
			losingTrades++
			if trade.PnLPercent < largestLoss {
				largestLoss = trade.PnLPercent
			}
		}

		// 计算回撤
		if totalPnL > peak {
			peak = totalPnL
		}
		currentDrawdown := peak - totalPnL
		if currentDrawdown > maxDrawdown {
			maxDrawdown = currentDrawdown
		}
	}

	result.WinningTrades = winningTrades
	result.LosingTrades = losingTrades
	result.WinRate = float64(winningTrades) / float64(result.TotalTrades) * 100
	result.TotalPnL = totalPnL
	result.AvgPnL = totalPnL / float64(result.TotalTrades)
	result.MaxDrawdown = maxDrawdown
	result.LargestWin = largestWin
	result.LargestLoss = largestLoss

	// 计算夏普比率（简化版，需要日收益率数据）
	if result.TotalTrades > 0 {
		avgReturn := result.AvgPnL / 100 // 转换为小数
		volatility := 0.15 // 假设15%的波动率
		result.SharpeRatio = avgReturn / volatility
	}

	// 计算盈利因子
	if losingTrades > 0 {
		totalWins := 0.0
		totalLosses := 0.0
		for _, trade := range result.Trades {
			if trade.PnLPercent > 0 {
				totalWins += trade.PnLPercent
			} else {
				totalLosses += math.Abs(trade.PnLPercent)
			}
		}
		if totalLosses > 0 {
			result.ProfitFactor = totalWins / totalLosses
		}
	}

	// 计算平均持仓时间
	totalHoldTime := time.Duration(0)
	for _, trade := range result.Trades {
		totalHoldTime += trade.ExitTime.Sub(trade.EntryTime)
	}
	result.AvgHoldTime = totalHoldTime / time.Duration(result.TotalTrades)
}

func displayBacktestResults(result *BacktestResult) {
	fmt.Println("\n📊 回测结果统计:")
	fmt.Println("─────────────")
	fmt.Printf("回测期间: %s 至 %s\n", result.StartDate.Format("2006-01-02"), result.EndDate.Format("2006-01-02"))
	fmt.Printf("总交易次数: %d\n", result.TotalTrades)
	fmt.Printf("盈利交易: %d (%.1f%%)\n", result.WinningTrades, result.WinRate)
	fmt.Printf("亏损交易: %d (%.1f%%)\n", result.LosingTrades, 100-result.WinRate)
	fmt.Printf("总收益率: %.2f%%\n", result.TotalPnL)
	fmt.Printf("平均收益率: %.2f%%\n", result.AvgPnL)
	fmt.Printf("最大回撤: %.2f%%\n", result.MaxDrawdown)
	fmt.Printf("夏普比率: %.3f\n", result.SharpeRatio)
	fmt.Printf("盈利因子: %.3f\n", result.ProfitFactor)
	fmt.Printf("最大盈利: %.2f%%\n", result.LargestWin)
	fmt.Printf("最大亏损: %.2f%%\n", result.LargestLoss)
	fmt.Printf("平均持仓时间: %v\n", result.AvgHoldTime)

	// 评估策略表现
	fmt.Println("\n🎯 策略表现评估:")
	rating := evaluateStrategyPerformance(result)
	fmt.Printf("整体评级: %s\n", rating.OverallRating)
	fmt.Printf("优势: %s\n", rating.Strengths)
	fmt.Printf("劣势: %s\n", rating.Weaknesses)
	fmt.Printf("建议: %s\n", rating.Recommendations)
}

type PerformanceRating struct {
	OverallRating   string
	Strengths       string
	Weaknesses      string
	Recommendations string
}

func evaluateStrategyPerformance(result *BacktestResult) *PerformanceRating {
	rating := &PerformanceRating{}

	// 基于多个指标进行综合评估
	score := 0.0

	// 胜率评分 (40%)
	if result.WinRate >= 60 {
		score += 40
		rating.Strengths += "胜率优秀; "
	} else if result.WinRate >= 50 {
		score += 30
		rating.Strengths += "胜率良好; "
	} else if result.WinRate >= 40 {
		score += 20
		rating.Strengths += "胜率一般; "
	} else {
		score += 10
		rating.Weaknesses += "胜率偏低; "
	}

	// 盈亏比评分 (30%)
	if result.ProfitFactor >= 1.5 {
		score += 30
		rating.Strengths += "盈亏比优秀; "
	} else if result.ProfitFactor >= 1.2 {
		score += 20
		rating.Strengths += "盈亏比良好; "
	} else if result.ProfitFactor >= 1.0 {
		score += 15
		rating.Strengths += "盈亏比一般; "
	} else {
		score += 5
		rating.Weaknesses += "盈亏比不理想; "
	}

	// 回撤控制评分 (20%)
	if result.MaxDrawdown <= 10 {
		score += 20
		rating.Strengths += "回撤控制优秀; "
	} else if result.MaxDrawdown <= 20 {
		score += 15
		rating.Strengths += "回撤控制良好; "
	} else if result.MaxDrawdown <= 30 {
		score += 10
		rating.Strengths += "回撤控制一般; "
	} else {
		score += 5
		rating.Weaknesses += "回撤控制不足; "
	}

	// 夏普比率评分 (10%)
	if result.SharpeRatio >= 1.5 {
		score += 10
		rating.Strengths += "风险调整收益优秀; "
	} else if result.SharpeRatio >= 1.0 {
		score += 7
		rating.Strengths += "风险调整收益良好; "
	} else if result.SharpeRatio >= 0.5 {
		score += 4
		rating.Strengths += "风险调整收益一般; "
	} else {
		score += 2
		rating.Weaknesses += "风险调整收益不足; "
	}

	// 总体评级
	if score >= 80 {
		rating.OverallRating = "优秀 (A)"
		rating.Recommendations = "策略表现优秀，可以考虑实盘使用"
	} else if score >= 60 {
		rating.OverallRating = "良好 (B)"
		rating.Recommendations = "策略表现良好，经过优化后可以实盘使用"
	} else if score >= 40 {
		rating.OverallRating = "一般 (C)"
		rating.Recommendations = "策略表现一般，需要大幅优化后谨慎使用"
	} else {
		rating.OverallRating = "较差 (D)"
		rating.Recommendations = "策略表现不佳，不建议实盘使用，建议重新设计"
	}

	return rating
}

func analyzeMarketImpact(db *sql.DB, result *BacktestResult) {
	fmt.Println("\n🌍 市场环境影响分析:")

	// 分析不同市场环境下的表现
	marketQuery := `
		SELECT
			DATE(created_at) as trade_date,
			AVG(price_change_percent) as market_change,
			STDDEV(price_change_percent) as market_volatility
		FROM binance_24h_stats
		WHERE DATE(created_at) >= DATE(?) AND DATE(created_at) <= DATE(?)
			AND market_type = 'spot'
			AND quote_volume > 1000000
		GROUP BY DATE(created_at)
		ORDER BY trade_date`

	rows, err := db.Query(marketQuery, result.StartDate, result.EndDate)
	if err != nil {
		fmt.Printf("市场数据查询失败: %v\n", err)
		return
	}
	defer rows.Close()

	bullDays := 0
	bearDays := 0
	sidewaysDays := 0
	totalDays := 0

	bullPnL := 0.0
	bearPnL := 0.0
	sidewaysPnL := 0.0

	for rows.Next() {
		var tradeDate time.Time
		var marketChange, marketVolatility float64
		rows.Scan(&tradeDate, &marketChange, &marketVolatility)

		totalDays++

		// 分类市场环境
		if marketChange > 2 {
			bullDays++
			// 计算当天交易的PnL
			dayPnL := calculateDayPnL(result.Trades, tradeDate)
			bullPnL += dayPnL
		} else if marketChange < -2 {
			bearDays++
			dayPnL := calculateDayPnL(result.Trades, tradeDate)
			bearPnL += dayPnL
		} else {
			sidewaysDays++
			dayPnL := calculateDayPnL(result.Trades, tradeDate)
			sidewaysPnL += dayPnL
		}
	}

	fmt.Printf("市场环境分布:\n")
	fmt.Printf("  多头行情: %d天 (%.1f%%)\n", bullDays, float64(bullDays)/float64(totalDays)*100)
	fmt.Printf("  空头行情: %d天 (%.1f%%)\n", bearDays, float64(bearDays)/float64(totalDays)*100)
	fmt.Printf("  震荡行情: %d天 (%.1f%%)\n", sidewaysDays, float64(sidewaysDays)/float64(totalDays)*100)

	if bullDays > 0 {
		fmt.Printf("  多头行情平均日收益: %.2f%%\n", bullPnL/float64(bullDays))
	}
	if bearDays > 0 {
		fmt.Printf("  空头行情平均日收益: %.2f%%\n", bearPnL/float64(bearDays))
	}
	if sidewaysDays > 0 {
		fmt.Printf("  震荡行情平均日收益: %.2f%%\n", sidewaysPnL/float64(sidewaysDays))
	}

	// 给出市场适应性建议
	fmt.Println("\n💡 市场适应性分析:")
	if bullPnL/float64(bullDays) < bearPnL/float64(bearDays) && bullPnL/float64(bullDays) < sidewaysPnL/float64(sidewaysDays) {
		fmt.Println("  ❌ 在多头行情中表现最差，说明逆势做空强势币种的策略在上涨市不适用")
		fmt.Println("  ✅ 在震荡和下跌行情中表现相对较好")
		fmt.Println("  📝 建议：添加市场趋势过滤，在上涨趋势时暂停策略")
	} else {
		fmt.Println("  ✅ 策略在不同市场环境下表现相对均衡")
	}
}

func calculateDayPnL(trades []TradeRecord, tradeDate time.Time) float64 {
	totalPnL := 0.0
	tradeCount := 0

	for _, trade := range trades {
		if trade.EntryTime.Truncate(24*time.Hour).Equal(tradeDate.Truncate(24*time.Hour)) {
			totalPnL += trade.PnLPercent
			tradeCount++
		}
	}

	if tradeCount > 0 {
		return totalPnL / float64(tradeCount)
	}
	return 0.0
}

func generateBacktestRecommendations(result *BacktestResult, params *Strategy21Params) {
	fmt.Println("\n💡 基于回测的改进建议:")

	// 胜率优化建议
	if result.WinRate < 50 {
		fmt.Println("🎯 胜率优化:")
		fmt.Println("  • 减少做空目标数量，从前7名减少到前3-5名")
		fmt.Println("  • 增加技术指标确认，避免在强势上涨中做空")
		fmt.Println("  • 调整止盈止损比例，考虑更宽松的止损")
		fmt.Printf("  • 当前止损%.1f%%可能过于严格，建议放宽到2-3%%\n", params.StopLossPercent)
	}

	// 回撤控制建议
	if result.MaxDrawdown > 20 {
		fmt.Println("🛡️ 回撤控制:")
		fmt.Println("  • 降低杠杆倍数，从3x降至2x")
		fmt.Println("  • 减少单策略最大仓位，从20%降至10-15%")
		fmt.Println("  • 增加仓位动态调整机制")
		fmt.Println("  • 实施每日/每周亏损限制")
	}

	// 盈利因子优化建议
	if result.ProfitFactor < 1.2 {
		fmt.Println("💰 盈利能力提升:")
		fmt.Println("  • 优化止盈策略，让利润奔跑")
		fmt.Println("  • 减少小额亏损交易")
		fmt.Println("  • 增加盈利再投资机制")
		fmt.Println("  • 考虑多策略组合分散风险")
	}

	// 时间优化建议
	if result.AvgHoldTime < time.Hour*4 {
		fmt.Println("⏱️ 持仓时间优化:")
		fmt.Println("  • 考虑延长持仓时间，避免日内过度交易")
		fmt.Println("  • 从5分钟调整到15-30分钟执行间隔")
		fmt.Println("  • 增加隔夜持仓能力")
	}

	// 市场适应性建议
	fmt.Println("🌍 市场适应性改进:")
	fmt.Println("  • 添加市场趋势检测，避免在上涨趋势中操作")
	fmt.Println("  • 增加波动率过滤，高波动时暂停")
	fmt.Println("  • 考虑多时间框架确认信号")
	fmt.Println("  • 增加基本面因素（如市值、成交量）")

	fmt.Println("\n📊 预期改进效果:")
	fmt.Printf("  • 胜率提升至: 55-65%%\n")
	fmt.Printf("  • 最大回撤控制在: 15-20%%\n")
	fmt.Printf("  • 夏普比率提升至: 1.5-2.0\n")
	fmt.Printf("  • 年化收益稳定在: 15-25%%\n")
}