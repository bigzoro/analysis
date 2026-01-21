package main

import (
	"fmt"
	"log"
	"math"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 简化的均值回归策略调试
func debugMeanReversionStrategy(symbol string, prices []float64, entryThreshold float64) []TradeRecord {
	var trades []TradeRecord
	var position *TradeRecord
	capital := 10000.0

	// 成本设置
	slippage := 0.0001
	fee := 0.0002

	// 计算布林带
	period := 20
	var upper, middle, lower []float64

	for i := period - 1; i < len(prices); i++ {
		window := prices[i-period+1 : i+1]
		sum := 0.0
		for _, price := range window {
			sum += price
		}
		mean := sum / float64(period)

		// 计算标准差
		sumSquares := 0.0
		for _, price := range window {
			sumSquares += (price - mean) * (price - mean)
		}
		stdDev := math.Sqrt(sumSquares / float64(period))

		upper = append(upper, mean+2*stdDev)
		middle = append(middle, mean)
		lower = append(lower, mean-2*stdDev)
	}

	fmt.Printf("开始调试 %s 策略 (EntryThreshold=%.2f)\n", symbol, entryThreshold)
	fmt.Printf("数据点: %d, 布林带点: %d\n", len(prices), len(upper))

	signalCount := 0
	validSignalCount := 0

	for i := period; i < len(prices); i++ {
		currentPrice := prices[i]
		currentTime := time.Now().AddDate(0, 0, i-len(prices))

		bbIndex := i - period

		// 布林带信号
		bollingerSignal := 0
		if bbIndex >= 0 && bbIndex < len(lower) && bbIndex < len(upper) {
			bandwidth := upper[bbIndex] - lower[bbIndex]
			if bandwidth > 0 && middle[bbIndex] > 0 {
				lowerDeviation := (lower[bbIndex] - currentPrice) / bandwidth
				upperDeviation := (currentPrice - upper[bbIndex]) / bandwidth

				if lowerDeviation > entryThreshold {
					bollingerSignal = 1
				} else if upperDeviation > entryThreshold {
					bollingerSignal = -1
				}
			}
		}

		if bollingerSignal != 0 {
			signalCount++

			// 计算positionScore
			positionScore := 0.0
			if bbIndex >= 0 && bbIndex < len(middle) {
				bandwidth := upper[bbIndex] - lower[bbIndex]
				if bandwidth > 0 {
					positionScore = (currentPrice - middle[bbIndex]) / bandwidth
				}
			}

			// 计算RSI
			rsiValue := 50.0
			if i >= 14 {
				rsiValue = calculateRSI(prices[max(0, i-14):i+1], 14)
			}

			rsiSignal := 0.0
			if rsiValue < 30 && bollingerSignal == 1 {
				rsiSignal = 0.5
			} else if rsiValue > 70 && bollingerSignal == -1 {
				rsiSignal = 0.5
			}

			// 综合信号
			signalStrength := float64(bollingerSignal)*0.8 + rsiSignal*0.1 + positionScore*0.1

			fmt.Printf("信号 #%d: 价格=%.2f, 布林信号=%d, RSI=%.1f, RSI信号=%.1f, 位置得分=%.3f, 信号强度=%.3f\n",
				signalCount, currentPrice, bollingerSignal, rsiValue, rsiSignal, positionScore, signalStrength)

			// 开仓判断
			if position == nil && math.Abs(signalStrength) > entryThreshold {
				validSignalCount++

				fmt.Printf("  ✅ 有效开仓信号! 信号强度 %.3f > 阈值 %.3f\n", math.Abs(signalStrength), entryThreshold)

				// 计算成本
				actualEntryPrice := currentPrice * (1 + slippage)
				entryFee := actualEntryPrice * fee
				totalEntryCost := actualEntryPrice + entryFee
				availableCapital := capital * 0.05 // 5%仓位
				quantity := availableCapital / totalEntryCost

				position = &TradeRecord{
					Symbol:     symbol,
					Side:       "BUY",
					Price:      actualEntryPrice,
					Quantity:   quantity,
					Timestamp:  currentTime,
					EntryPrice: actualEntryPrice,
				}
				position.Profit = -entryFee

				fmt.Printf("  💰 开仓: 价格=%.2f, 数量=%.6f, 成本=%.2f\n", actualEntryPrice, quantity, totalEntryCost)
			} else {
				fmt.Printf("  ❌ 信号强度 %.3f <= 阈值 %.3f，跳过开仓\n", math.Abs(signalStrength), entryThreshold)
			}
		}

		// 平仓逻辑
		if position != nil {
			holdDays := currentTime.Sub(position.Timestamp).Hours() / 24

			// 动态止损
			dynamicStopLoss := 0.05
			if i >= 20 {
				recentPrices := prices[i-20 : i]
				volatility := calculateVolatility(recentPrices)
				dynamicStopLoss = math.Min(0.08, 0.05+volatility*0.5)
			}

			stopLossHit := currentPrice <= position.EntryPrice*(1-dynamicStopLoss)
			takeProfitHit := currentPrice >= position.EntryPrice*1.12 // 12%止盈
			timeout := holdDays >= 7

			if stopLossHit || takeProfitHit || timeout {
				actualExitPrice := currentPrice * (1 - slippage)
				exitFee := actualExitPrice * fee
				grossProfit := (actualExitPrice - position.EntryPrice) * position.Quantity
				actualProfit := grossProfit - exitFee

				position.ExitPrice = actualExitPrice
				position.HoldHours = currentTime.Sub(position.Timestamp).Hours()
				position.Profit = actualProfit

				trades = append(trades, *position)
				position = nil

				fmt.Printf("  📊 平仓: 价格=%.2f, 利润=%.2f\n", actualExitPrice, actualProfit)
			}
		}
	}

	fmt.Printf("\n📊 调试总结:\n")
	fmt.Printf("总信号数: %d\n", signalCount)
	fmt.Printf("有效开仓数: %d\n", validSignalCount)
	fmt.Printf("完成交易数: %d\n", len(trades))

	if len(trades) > 0 {
		totalPnL := 0.0
		winningTrades := 0
		for _, trade := range trades {
			totalPnL += trade.Profit
			if trade.Profit > 0 {
				winningTrades++
			}
		}

		winRate := float64(winningTrades) / float64(len(trades)) * 100
		fmt.Printf("胜率: %.1f%%\n", winRate)
		fmt.Printf("总盈亏: %.2f USDT\n", totalPnL)
	}

	return trades
}

func main() {
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("连接数据库失败:", err)
	}

	// 获取近期30天BTC数据
	var prices []float64
	query := `
		SELECT close_price
		FROM market_klines
		WHERE symbol = 'BTCUSDT' AND kind = 'spot' AND ` + "`interval`" + ` = '1d'
		AND open_time >= DATE_SUB(NOW(), INTERVAL 30 DAY)
		ORDER BY open_time ASC
	`
	err = db.Raw(query).Scan(&prices).Error
	if err != nil {
		log.Fatal("查询数据失败:", err)
	}

	fmt.Printf("BTCUSDT 30天数据: %d 条\n", len(prices))

	// 测试不同的EntryThreshold
	thresholds := []float64{0.5, 0.7, 0.85, 0.90, 0.95}

	for _, threshold := range thresholds {
		fmt.Printf("\n===========================================================\n")
		fmt.Printf("测试 EntryThreshold = %.2f\n", threshold)
		fmt.Printf("===========================================================\n")

		trades := debugMeanReversionStrategy("BTCUSDT", prices, threshold)

		if len(trades) > 0 {
			totalPnL := 0.0
			winningTrades := 0
			for _, trade := range trades {
				totalPnL += trade.Profit
				if trade.Profit > 0 {
					winningTrades++
				}
			}

			winRate := float64(winningTrades) / float64(len(trades)) * 100
			avgProfit := totalPnL / float64(len(trades))

			fmt.Printf("📊 结果: 交易%d笔, 胜率%.1f%%, 平均利润%.2f USDT, 总盈亏%.2f USDT\n",
				len(trades), winRate, avgProfit, totalPnL)
		} else {
			fmt.Printf("❌ 没有产生任何交易\n")
		}
	}
}

// 辅助函数
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50.0
	}

	var gains, losses []float64
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

	avgGain := 0.0
	avgLoss := 0.0
	for i := 0; i < period && i < len(gains); i++ {
		avgGain += gains[len(gains)-1-i]
		avgLoss += losses[len(losses)-1-i]
	}
	if period > 0 {
		avgGain /= float64(period)
		avgLoss /= float64(period)
	}

	if avgLoss == 0 {
		return 100.0
	}

	rs := avgGain / avgLoss
	rsi := 100.0 - (100.0 / (1.0 + rs))
	return math.Max(0.0, math.Min(100.0, rsi))
}

func calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	sumSquares := 0.0
	for _, ret := range returns {
		sumSquares += (ret - mean) * (ret - mean)
	}

	return math.Sqrt(sumSquares / float64(len(returns)))
}

type TradeRecord struct {
	Symbol     string
	Side       string
	Price      float64
	Quantity   float64
	Timestamp  time.Time
	Profit     float64
	EntryPrice float64
	ExitPrice  float64
	HoldHours  float64
}
