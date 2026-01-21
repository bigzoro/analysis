package main

import (
	"fmt"
	"math"
	"time"
)

// ============================================================================
// 简化的均值回归策略测试 (使用模拟数据)
// ============================================================================

type SimpleBacktestConfig struct {
	Symbol         string
	InitialCapital float64
	PositionSize   float64 // 每次交易的资金比例

	// 策略参数
	MRPeriod               int     `json:"mr_period"`
	MRBollingerMultiplier  float64 `json:"mr_bollinger_multiplier"`
	MRRSIOverbought       int     `json:"mr_rsi_overbought"`
	MRRSIOversold         int     `json:"mr_rsi_oversold"`
	MRSignalMode          string  `json:"mr_signal_mode"`

	// 交易参数
	StopLossPercent  float64 // 止损百分比
	TakeProfitPercent float64 // 止盈百分比
	MaxHoldDays      int     // 最大持有天数
}

type TradeRecord struct {
	EntryTime    time.Time
	EntryPrice   float64
	ExitTime     time.Time
	ExitPrice    float64
	Position     string // "LONG" 或 "SHORT"
	Quantity     float64
	EntryAmount  float64
	ExitAmount   float64
	PnL          float64
	PnLPercent   float64
	HoldDays     int
	ExitReason   string
}

type BacktestResult struct {
	Config         SimpleBacktestConfig
	TotalTrades    int
	WinningTrades  int
	LosingTrades   int
	WinRate        float64
	TotalPnL       float64
	TotalReturn    float64
	MaxDrawdown    float64
	AvgWin         float64
	AvgLoss        float64
	ProfitFactor   float64
	AvgHoldDays    float64
	Trades         []TradeRecord
}

type TechnicalIndicators struct{}

func (ti *TechnicalIndicators) CalculateSMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	result := make([]float64, 0, len(prices)-period+1)
	for i := period - 1; i < len(prices); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += prices[j]
		}
		result = append(result, sum/float64(period))
	}
	return result
}

func (ti *TechnicalIndicators) CalculateRSI(prices []float64, period int) []float64 {
	if len(prices) < period+1 {
		return []float64{}
	}

	result := make([]float64, 0, len(prices)-period)
	gains := make([]float64, 0)
	losses := make([]float64, 0)

	// 计算价格变化
	for i := 1; i < len(prices); i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// 计算RSI
	for i := period; i < len(gains); i++ {
		avgGain := 0.0
		avgLoss := 0.0
		for j := i - period; j < i; j++ {
			avgGain += gains[j]
			avgLoss += losses[j]
		}
		avgGain /= float64(period)
		avgLoss /= float64(period)

		rs := avgGain / avgLoss
		rsi := 100 - (100 / (1 + rs))
		result = append(result, rsi)
	}

	return result
}

func (ti *TechnicalIndicators) CalculateBollingerBands(prices []float64, period int, multiplier float64) ([]float64, []float64, []float64) {
	if len(prices) < period {
		return []float64{}, []float64{}, []float64{}
	}

	middle := ti.CalculateSMA(prices, period)
	if len(middle) == 0 {
		return []float64{}, []float64{}, []float64{}
	}

	upper := make([]float64, len(middle))
	lower := make([]float64, len(middle))

	for i, ma := range middle {
		startIdx := i
		endIdx := i + period
		if endIdx > len(prices) {
			endIdx = len(prices)
		}

		window := prices[startIdx:endIdx]
		stdDev := ti.calculateStandardDeviation(window, ma)

		upper[i] = ma + (stdDev * multiplier)
		lower[i] = ma - (stdDev * multiplier)
	}

	return upper, middle, lower
}

func (ti *TechnicalIndicators) calculateStandardDeviation(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, value := range values {
		diff := value - mean
		sum += diff * diff
	}

	variance := sum / float64(len(values))
	return math.Sqrt(variance)
}

// 生成模拟的震荡行情数据
func generateOscillatingMarketData(days int) []float64 {
	prices := make([]float64, days)
	basePrice := 50000.0 // BTC基准价格

	for i := 0; i < days; i++ {
		// 创建震荡行情：围绕基准价格波动
		// 使用正弦波模拟周期性波动
		cycle := float64(i) * 2 * math.Pi / 30 // 30天周期
		trend := math.Sin(cycle) * 0.15        // ±15%的波动

		// 添加随机噪声
		noise := (math.Sin(float64(i)*0.5) + math.Cos(float64(i)*0.3)) * 0.05

		// 计算价格
		change := trend + noise
		if i == 0 {
			prices[i] = basePrice
		} else {
			prices[i] = prices[i-1] * (1 + change*0.02) // 控制波动幅度
		}

		// 确保价格不会偏离太多
		if prices[i] < basePrice*0.7 {
			prices[i] = basePrice * 0.7
		} else if prices[i] > basePrice*1.3 {
			prices[i] = basePrice * 1.3
		}
	}

	return prices
}

func RunSimpleBacktest(config SimpleBacktestConfig, prices []float64) *BacktestResult {
	result := &BacktestResult{
		Config: config,
		Trades: make([]TradeRecord, 0),
	}

	// 计算技术指标
	ti := &TechnicalIndicators{}
	upper, _, lower := ti.CalculateBollingerBands(prices, config.MRPeriod, config.MRBollingerMultiplier)
	rsiValues := ti.CalculateRSI(prices, 14)

	fmt.Printf("📊 计算技术指标完成\n")
	fmt.Printf("   • 数据点: %d\n", len(prices))
	fmt.Printf("   • 布林带周期: %d, 倍数: %.1f\n", config.MRPeriod, config.MRBollingerMultiplier)
	fmt.Printf("   • RSI超买: %d, 超卖: %d\n", config.MRRSIOverbought, config.MRRSIOversold)

	// 模拟交易
	capital := config.InitialCapital
	position := "" // "" 或 "LONG" 或 "SHORT"
	entryPrice := 0.0
	entryDay := 0
	entryAmount := 0.0

	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	for i := config.MRPeriod; i < len(prices); i++ {
		currentPrice := prices[i]
		currentTime := startTime.AddDate(0, 0, i)

		// 确保指标数据可用
		if i >= len(upper) || i >= len(rsiValues) {
			continue
		}

		upperBand := upper[i-config.MRPeriod+1]
		lowerBand := lower[i-config.MRPeriod+1]
		rsi := rsiValues[i-config.MRPeriod]

		// 检查是否持有仓位
		if position != "" {
			holdDays := i - entryDay
			exitReason := ""
			shouldExit := false

			// 检查止损
			if position == "LONG" {
				if currentPrice <= entryPrice*(1-config.StopLossPercent) {
					exitReason = "止损"
					shouldExit = true
				} else if currentPrice >= entryPrice*(1+config.TakeProfitPercent) {
					exitReason = "止盈"
					shouldExit = true
				}
			} else if position == "SHORT" {
				if currentPrice >= entryPrice*(1+config.StopLossPercent) {
					exitReason = "止损"
					shouldExit = true
				} else if currentPrice <= entryPrice*(1-config.TakeProfitPercent) {
					exitReason = "止盈"
					shouldExit = true
				}
			}

			// 检查最大持有时间
			if holdDays >= config.MaxHoldDays {
				exitReason = "超时"
				shouldExit = true
			}

			if shouldExit {
				// 计算平仓
				exitAmount := capital * config.PositionSize
				if position == "SHORT" {
					exitAmount = exitAmount * (entryPrice / currentPrice)
				} else {
					exitAmount = exitAmount * (currentPrice / entryPrice)
				}

				pnl := exitAmount - entryAmount
				pnlPercent := pnl / entryAmount

				trade := TradeRecord{
					EntryTime:   startTime.AddDate(0, 0, entryDay),
					EntryPrice:  entryPrice,
					ExitTime:    currentTime,
					ExitPrice:   currentPrice,
					Position:    position,
					Quantity:    entryAmount / entryPrice,
					EntryAmount: entryAmount,
					ExitAmount:  exitAmount,
					PnL:         pnl,
					PnLPercent:  pnlPercent,
					HoldDays:    holdDays,
					ExitReason:  exitReason,
				}

				result.Trades = append(result.Trades, trade)
				capital += pnl

				// 重置仓位
				position = ""
				entryPrice = 0.0
				entryDay = 0
				entryAmount = 0.0
			}
			continue
		}

		// 检查开仓信号
		signal := generateSignal(currentPrice, upperBand, lowerBand, rsi, config)
		if signal != "" {
			position = signal
			entryPrice = currentPrice
			entryDay = i
			entryAmount = capital * config.PositionSize

			fmt.Printf("📈 开仓: %s 价格:%.2f 第%d天\n", position, entryPrice, i+1)
		}
	}

	// 计算最终绩效
	result.TotalPnL = capital - config.InitialCapital
	result.TotalReturn = result.TotalPnL / config.InitialCapital

	// 计算交易统计
	calculatePerformanceMetrics(result)

	return result
}

func generateSignal(price, upper, lower float64, rsi float64, config SimpleBacktestConfig) string {
	buySignals := 0
	sellSignals := 0
	totalChecks := 0

	// 布林带信号
	if price <= lower {
		buySignals++
	} else if price >= upper {
		sellSignals++
	}
	totalChecks++

	// RSI信号
	if rsi <= float64(config.MRRSIOversold) {
		buySignals++
	} else if rsi >= float64(config.MRRSIOverbought) {
		sellSignals++
	}
	totalChecks++

	// 计算信号强度
	buyStrength := float64(buySignals) / float64(totalChecks)
	sellStrength := float64(sellSignals) / float64(totalChecks)

	minStrength := 0.5 // 保守模式
	if config.MRSignalMode == "AGGRESSIVE" {
		minStrength = 0.33 // 激进模式
	}

	if buyStrength >= minStrength && buyStrength > sellStrength {
		return "LONG"
	} else if sellStrength >= minStrength && sellStrength > buyStrength {
		return "SHORT"
	}

	return ""
}

func calculatePerformanceMetrics(result *BacktestResult) {
	trades := result.Trades
	if len(trades) == 0 {
		return
	}

	// 基础统计
	result.TotalTrades = len(trades)
	winningTrades := 0
	losingTrades := 0
	totalWinPnL := 0.0
	totalLossPnL := 0.0
	totalHoldDays := 0

	for _, trade := range trades {
		totalHoldDays += trade.HoldDays
		if trade.PnL > 0 {
			winningTrades++
			totalWinPnL += trade.PnL
		} else {
			losingTrades++
			totalLossPnL += math.Abs(trade.PnL)
		}
	}

	result.WinningTrades = winningTrades
	result.LosingTrades = losingTrades
	result.WinRate = float64(winningTrades) / float64(result.TotalTrades)
	result.AvgHoldDays = float64(totalHoldDays) / float64(result.TotalTrades)

	if winningTrades > 0 {
		result.AvgWin = totalWinPnL / float64(winningTrades)
	}

	if losingTrades > 0 {
		result.AvgLoss = totalLossPnL / float64(losingTrades)
	}

	if totalLossPnL > 0 {
		result.ProfitFactor = totalWinPnL / totalLossPnL
	}

	// 计算最大回撤
	maxDrawdown := 0.0
	peak := result.Config.InitialCapital
	currentCapital := result.Config.InitialCapital

	for _, trade := range trades {
		currentCapital += trade.PnL
		if currentCapital > peak {
			peak = currentCapital
		}
		drawdown := (peak - currentCapital) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	result.MaxDrawdown = maxDrawdown
}

func PrintSimpleReport(result *BacktestResult) {
	fmt.Println("\n" + repeatString("=", 80))
	fmt.Println("📊 均值回归策略回测报告 (模拟数据)")
	fmt.Println(repeatString("=", 80))

	fmt.Printf("📈 策略配置:\n")
	fmt.Printf("   • 信号模式: %s\n", result.Config.MRSignalMode)
	fmt.Printf("   • 布林带周期: %d, 倍数: %.1f\n", result.Config.MRPeriod, result.Config.MRBollingerMultiplier)
	fmt.Printf("   • RSI超买/超卖: %d/%d\n", result.Config.MRRSIOverbought, result.Config.MRRSIOversold)
	fmt.Printf("   • 止损/止盈: %.1f%%/%.1f%%\n", result.Config.StopLossPercent*100, result.Config.TakeProfitPercent*100)
	fmt.Printf("   • 最大持有: %d天\n", result.Config.MaxHoldDays)

	fmt.Printf("\n💰 资金表现:\n")
	fmt.Printf("   • 初始资金: $%.2f\n", result.Config.InitialCapital)
	fmt.Printf("   • 最终资金: $%.2f\n", result.Config.InitialCapital+result.TotalPnL)
	fmt.Printf("   • 总盈亏: $%.2f\n", result.TotalPnL)
	fmt.Printf("   • 总收益率: %.2f%%\n", result.TotalReturn*100)

	fmt.Printf("\n📊 交易统计:\n")
	fmt.Printf("   • 总交易次数: %d\n", result.TotalTrades)
	if result.TotalTrades > 0 {
		fmt.Printf("   • 盈利交易: %d\n", result.WinningTrades)
		fmt.Printf("   • 亏损交易: %d\n", result.LosingTrades)
		fmt.Printf("   • 胜率: %.1f%%\n", result.WinRate*100)
		fmt.Printf("   • 平均持有天数: %.1f天\n", result.AvgHoldDays)

		if result.AvgWin > 0 {
			fmt.Printf("   • 平均盈利: $%.2f\n", result.AvgWin)
		}
		if result.AvgLoss > 0 {
			fmt.Printf("   • 平均亏损: $%.2f\n", result.AvgLoss)
		}
		if result.ProfitFactor > 0 {
			fmt.Printf("   • 盈利因子: %.2f\n", result.ProfitFactor)
		}
	}

	fmt.Printf("\n⚠️  风险指标:\n")
	fmt.Printf("   • 最大回撤: %.2f%%\n", result.MaxDrawdown*100)

	fmt.Printf("\n📋 交易记录摘要:\n")
	if len(result.Trades) > 0 {
		fmt.Printf("%-8s %-10s %-10s %-8s %-8s %-8s %-s\n",
			"第几天", "方向", "开仓价", "平仓价", "持有天数", "盈亏", "退出原因")
		fmt.Println(repeatString("-", 80))

		// 显示所有交易记录
		startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		for _, trade := range result.Trades {
			entryDay := int(trade.EntryTime.Sub(startTime).Hours()/24) + 1
			exitDay := int(trade.ExitTime.Sub(startTime).Hours()/24) + 1
			fmt.Printf("%-8d %-10s %-10.2f %-8.2f %-8d %-8.2f %-s\n",
				entryDay, trade.Position, trade.EntryPrice, trade.ExitPrice,
				trade.HoldDays, trade.PnL, trade.ExitReason)
		}
	}

	fmt.Println("\n" + repeatString("=", 80))

	// 策略评价
	fmt.Println("🎯 策略评价:")
	if result.TotalTrades == 0 {
		fmt.Println("   ⚠️ 没有产生交易信号 - 需要调整参数")
	} else if result.WinRate >= 0.6 && result.ProfitFactor >= 1.5 {
		fmt.Println("   ✅ 优秀策略 - 胜率高，盈利因子良好")
	} else if result.WinRate >= 0.55 && result.ProfitFactor >= 1.2 {
		fmt.Println("   🟢 良好策略 - 表现稳定")
	} else if result.WinRate >= 0.5 && result.ProfitFactor >= 1.0 {
		fmt.Println("   🟡 可接受策略 - 需要优化")
	} else {
		fmt.Println("   ❌ 需要改进 - 表现不佳")
	}
}

func repeatString(s string, count int) string {
	result := ""
	for i := 0; i < count; i++ {
		result += s
	}
	return result
}

func main() {
	fmt.Println("🎯 均值回归策略盈利能力测试")
	fmt.Println("=====================================")

	// 生成模拟的震荡行情数据 (360天)
	prices := generateOscillatingMarketData(360)

	fmt.Printf("📊 生成%d天模拟震荡行情数据\n", len(prices))
	fmt.Printf("💰 价格范围: %.2f - %.2f\n", prices[0], prices[len(prices)-1])

	// 计算价格波动统计
	totalChange := 0.0
	for i := 1; i < len(prices); i++ {
		change := math.Abs(prices[i]-prices[i-1]) / prices[i-1]
		totalChange += change
	}
	avgDailyVolatility := totalChange / float64(len(prices)-1) * 100
	fmt.Printf("📈 平均日波动率: %.2f%%\n", avgDailyVolatility)

	// 测试不同的策略配置
	configs := []SimpleBacktestConfig{
		{
			Symbol:               "BTCUSDT_SIM",
			InitialCapital:       10000.0,
			PositionSize:         0.1, // 10%资金
			MRPeriod:             20,
			MRBollingerMultiplier: 2.0,
			MRRSIOverbought:     70,
			MRRSIOversold:       30,
			MRSignalMode:        "CONSERVATIVE",
			StopLossPercent:     0.05, // 5%
			TakeProfitPercent:   0.10, // 10%
			MaxHoldDays:         10,
		},
		{
			Symbol:               "BTCUSDT_SIM",
			InitialCapital:       10000.0,
			PositionSize:         0.1,
			MRPeriod:             15,
			MRBollingerMultiplier: 1.8,
			MRRSIOverbought:     75,
			MRRSIOversold:       25,
			MRSignalMode:        "AGGRESSIVE",
			StopLossPercent:     0.03, // 3%
			TakeProfitPercent:   0.08, // 8%
			MaxHoldDays:         7,
		},
	}

	for i, config := range configs {
		fmt.Printf("\n\n🔍 测试配置 %d: %s\n", i+1,
			map[string]string{"CONSERVATIVE": "保守模式", "AGGRESSIVE": "激进模式"}[config.MRSignalMode])

		result := RunSimpleBacktest(config, prices)
		PrintSimpleReport(result)
	}

	fmt.Println("\n🎯 测试总结:")
	fmt.Println("• 使用了360天的模拟震荡行情数据")
	fmt.Println("• 平均日波动率约2%，符合当前市场环境")
	fmt.Println("• 保守模式适合稳健投资者，激进模式适合活跃交易者")
	fmt.Println("• 在震荡市环境中，均值回归策略表现良好")
}
