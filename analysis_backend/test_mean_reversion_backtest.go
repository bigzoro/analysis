package main

import (
	"fmt"
	"log"
	"math"
	"sort"
	"strings"
	"time"

	"analysis/internal/analysis"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

// 回测结果
type BacktestResult struct {
	Symbol              string
	TotalTrades         int
	WinningTrades       int
	LosingTrades        int
	WinRate             float64
	TotalPnL            float64
	TotalPnLPercent     float64
	MaxDrawdown         float64
	SharpeRatio         float64
	AvgTradePnL         float64
	AvgWinPnL           float64
	AvgLossPnL          float64
	LargestWin          float64
	LargestLoss         float64
	ProfitFactor        float64
	RecoveryFactor      float64
	Trades              []TradeRecord
}

// 交易记录
type TradeRecord struct {
	Symbol       string
	Side         string
	EntryTime    time.Time
	EntryPrice   float64
	ExitTime     time.Time
	ExitPrice    float64
	Quantity     float64
	PnL          float64
	PnLPercent   float64
	StopLoss     float64
	TakeProfit   float64
	Reason       string
}

// 市场数据缓存
type MarketDataCache struct {
	Klines map[string][]KlineData
	Stats  map[string]*MarketStats
}

type KlineData struct {
	Symbol    string
	OpenTime  time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
	CloseTime time.Time
}

type MarketStats struct {
	Symbol               string
	LastPrice            float64
	Volume24h            float64
	PriceChangePercent   float64
	High24h              float64
	Low24h               float64
}

func main() {
	fmt.Println("🎯 均值回归策略真实数据回测")
	fmt.Println("=====================================")

	// 连接数据库
	db, err := connectDatabase()
	if err != nil {
		log.Fatalf("❌ 连接数据库失败: %v", err)
	}

	// 初始化市场数据缓存
	cache := &MarketDataCache{
		Klines: make(map[string][]KlineData),
		Stats:  make(map[string]*MarketStats),
	}

	// 选择测试币种（扩展到25个主流币种）
	testSymbols := []string{
		// 顶级主流币种
		"BTCUSDT", "ETHUSDT", "BNBUSDT",
		// 大盘市值币种
		"ADAUSDT", "SOLUSDT", "DOTUSDT", "AVAXUSDT", "LINKUSDT", "LTCUSDT",
		// DeFi币种
		"ATOMUSDT", "ALGOUSDT", "DOGEUSDT",
		// 新兴热门币种
		"APTUSDT", "ARBUSDT", "OPUSDT", "FILUSDT", "ICPUSDT", "VETUSDT",
		// Layer 2和基础设施
		"MATICUSDT", "FTMUSDT", "NEARUSDT", "FLOWUSDT",
		// 稳定币相关和实用代币
		"CAKEUSDT", "SUSHIUSDT", "UNIUSDT",
	}

	fmt.Printf("📊 开始回测 %d 个币种的均值回归策略\n", len(testSymbols))

	// 加载市场数据
	fmt.Println("\n📥 加载市场数据...")
	err = loadMarketData(db, cache, testSymbols)
	if err != nil {
		log.Fatalf("❌ 加载市场数据失败: %v", err)
	}

	// 执行回测
	results := make(map[string]*BacktestResult)
	totalResults := &BacktestResult{Symbol: "TOTAL"}

	for _, symbol := range testSymbols {
		fmt.Printf("\n🔍 回测 %s...\n", symbol)

		result, err := backtestMeanReversionStrategy(cache, symbol)
		if err != nil {
			log.Printf("❌ 回测 %s 失败: %v", symbol, err)
			continue
		}

		results[symbol] = result

		// 汇总结果
		totalResults.TotalTrades += result.TotalTrades
		totalResults.WinningTrades += result.WinningTrades
		totalResults.LosingTrades += result.LosingTrades
		totalResults.TotalPnL += result.TotalPnL

		// 汇总盈利和亏损金额（用于计算利润因子）
		if result.AvgWinPnL > 0 && result.WinningTrades > 0 {
			totalResults.AvgWinPnL += result.AvgWinPnL * float64(result.WinningTrades)
		}
		if result.AvgLossPnL < 0 && result.LosingTrades > 0 {
			totalResults.AvgLossPnL += result.AvgLossPnL * float64(result.LosingTrades)
		}

		fmt.Printf("✅ %s 完成: %d 笔交易, 胜率 %.1f%%, PnL %.2f%%\n",
			symbol, result.TotalTrades, result.WinRate*100, result.TotalPnLPercent)
	}

	// 计算汇总统计
	if totalResults.TotalTrades > 0 {
		totalResults.WinRate = float64(totalResults.WinningTrades) / float64(totalResults.TotalTrades)
		totalResults.AvgTradePnL = totalResults.TotalPnL / float64(totalResults.TotalTrades)

		// 计算汇总利润因子
		totalWinningPnL := 0.0
		totalLosingPnL := 0.0

		for _, result := range results {
			if result.WinningTrades > 0 {
				totalWinningPnL += result.AvgWinPnL * float64(result.WinningTrades)
			}
			if result.LosingTrades > 0 {
				totalLosingPnL += math.Abs(result.AvgLossPnL) * float64(result.LosingTrades)
			}
		}

		if totalLosingPnL > 0 {
			totalResults.ProfitFactor = totalWinningPnL / totalLosingPnL
		} else if totalWinningPnL > 0 {
			totalResults.ProfitFactor = 999.0 // 只有盈利没有亏损
		} else {
			totalResults.ProfitFactor = 0.0 // 没有交易
		}

		// 计算平均盈利和亏损
		if totalResults.WinningTrades > 0 {
			totalResults.AvgWinPnL = totalWinningPnL / float64(totalResults.WinningTrades)
		}
		if totalResults.LosingTrades > 0 {
			totalResults.AvgLossPnL = -totalLosingPnL / float64(totalResults.LosingTrades) // 负数表示亏损
		}

		// 计算恢复因子
		if totalResults.MaxDrawdown > 0 {
			totalResults.RecoveryFactor = totalResults.TotalPnL / totalResults.MaxDrawdown
		} else if totalResults.TotalPnL > 0 {
			totalResults.RecoveryFactor = 999.0 // 无回撤，恢复因子无限大
		} else {
			totalResults.RecoveryFactor = 0.0 // 无盈利，无回撤
		}
	}

	// 显示详细结果
	displayResults(results, totalResults)

	// 生成交易分析报告
	generateAnalysisReport(results, totalResults)
}

func connectDatabase() (*gorm.DB, error) {
	dsn := "root:root@tcp(127.0.0.1:3306)/analysis?parseTime=true&charset=utf8mb4&loc=UTC"
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	return db, nil
}

func loadMarketData(db *gorm.DB, cache *MarketDataCache, symbols []string) error {
	// 加载K线数据（最近3个月）
	endTime := time.Now()
	startTime := endTime.AddDate(0, -3, 0)

	for _, symbol := range symbols {
		// 加载K线数据（日线数据）
		var klines []KlineData
		query := `
			SELECT
				symbol,
				open_time,
				CAST(open_price AS DECIMAL(20,8)) as open_price,
				CAST(high_price AS DECIMAL(20,8)) as high_price,
				CAST(low_price AS DECIMAL(20,8)) as low_price,
				CAST(close_price AS DECIMAL(20,8)) as close_price,
				CAST(volume AS DECIMAL(30,8)) as volume,
				open_time as close_time
			FROM market_klines
			WHERE symbol = ?
				AND kind = 'spot'
				AND ` + "`interval` = '1d'" +
			`				AND open_time >= ?
				AND open_time <= ?
			ORDER BY open_time ASC
		`

		rows, err := db.Raw(query, symbol, startTime, endTime).Rows()
		if err != nil {
			log.Printf("⚠️ 加载 %s K线数据失败: %v", symbol, err)
			continue
		}

		for rows.Next() {
			var kline KlineData
			err := rows.Scan(
				&kline.Symbol,
				&kline.OpenTime,
				&kline.Open,
				&kline.High,
				&kline.Low,
				&kline.Close,
				&kline.Volume,
				&kline.CloseTime,
			)
			if err != nil {
				continue
			}
			klines = append(klines, kline)
		}
		rows.Close()

		if len(klines) > 0 {
			cache.Klines[symbol] = klines
			fmt.Printf("✅ %s: 加载 %d 条K线数据\n", symbol, len(klines))
		} else {
			fmt.Printf("⚠️ %s: 无K线数据\n", symbol)
		}

		// 加载24小时统计数据
		var stats MarketStats
		statsQuery := `
			SELECT
				symbol,
				CAST(last_price AS DECIMAL(20,8)) as last_price,
				CAST(volume AS DECIMAL(30,8)) as volume,
				CAST(price_change_percent AS DECIMAL(10,4)) as price_change_percent,
				CAST(high_price AS DECIMAL(20,8)) as high_price,
				CAST(low_price AS DECIMAL(20,8)) as low_price
			FROM binance_24h_stats
			WHERE symbol = ? AND market_type = 'futures'
			ORDER BY close_time DESC
			LIMIT 1
		`

		err = db.Raw(statsQuery, symbol).Row().Scan(
			&stats.Symbol,
			&stats.LastPrice,
			&stats.Volume24h,
			&stats.PriceChangePercent,
			&stats.High24h,
			&stats.Low24h,
		)
		if err == nil {
			cache.Stats[symbol] = &stats
		}
	}

	return nil
}

func backtestMeanReversionStrategy(cache *MarketDataCache, symbol string) (*BacktestResult, error) {
	result := &BacktestResult{
		Symbol: symbol,
		Trades: make([]TradeRecord, 0),
	}

	klines, exists := cache.Klines[symbol]
	if !exists || len(klines) < 50 {
		return result, fmt.Errorf("数据不足")
	}

	// 均值回归策略参数（优化版本 - 提高交易频率）
	period := 20
	bbMultiplier := 2.0
	rsiPeriod := 14
	rsiOversold := 25.0      // 从30降到25，扩大买入机会
	rsiOverbought := 75.0    // 从70升到75，扩大卖出机会
	minReversionStrength := 0.15 // 从0.3降到0.15，大幅降低进入门槛
	maxHoldDays := 15        // 从30天降到15天，加快交易周转

	ti := analysis.NewTechnicalIndicators()

	// 计算技术指标
	closes := make([]float64, len(klines))
	for i, kline := range klines {
		closes[i] = kline.Close
	}

	// 计算布林带
	upper, _, lower := ti.CalculateBollingerBands(closes, period, bbMultiplier)
	rsi := ti.CalculateRSI(closes, rsiPeriod)

	if len(upper) == 0 || len(rsi) == 0 {
		return result, fmt.Errorf("技术指标计算失败")
	}

	// 模拟交易
	position := 0 // 0: 无持仓, 1: 多头, -1: 空头
	var entryPrice, stopLoss, takeProfit float64
	var entryTime time.Time
	var entryReason string

	for i := period; i < len(klines); i++ {
		currentPrice := klines[i].Close
		currentTime := klines[i].CloseTime

		// 计算布林带位置
		bbPosition := 0.0
		if i < len(upper) && upper[i] > lower[i] {
			bbPosition = (currentPrice - lower[i]) / (upper[i] - lower[i])
		}

		// 计算RSI
		currentRSI := 0.0
		if i < len(rsi) {
			currentRSI = rsi[i]
		}

		// 检查是否需要平仓
		if position != 0 {
			holdDays := currentTime.Sub(entryTime).Hours() / 24

			// 时间退出
			if holdDays >= float64(maxHoldDays) {
				exitReason := "持有时间超限"
				pnl := calculatePnL(position, entryPrice, currentPrice)
				record := TradeRecord{
					Symbol:     symbol,
					Side:       getPositionSide(position),
					EntryTime:  entryTime,
					EntryPrice: entryPrice,
					ExitTime:   currentTime,
					ExitPrice:  currentPrice,
					Quantity:   1.0,
					PnL:        pnl,
					PnLPercent: (pnl / entryPrice) * 100,
					StopLoss:   stopLoss,
					TakeProfit: takeProfit,
					Reason:     fmt.Sprintf("%s - %s", entryReason, exitReason),
				}
				result.Trades = append(result.Trades, record)

				position = 0
				continue
			}

			// 止损止盈检查
			if (position == 1 && currentPrice <= stopLoss) ||
			   (position == -1 && currentPrice >= stopLoss) {
				exitReason := "触发止损"
				pnl := calculatePnL(position, entryPrice, currentPrice)
				record := TradeRecord{
					Symbol:     symbol,
					Side:       getPositionSide(position),
					EntryTime:  entryTime,
					EntryPrice: entryPrice,
					ExitTime:   currentTime,
					ExitPrice:  currentPrice,
					Quantity:   1.0,
					PnL:        pnl,
					PnLPercent: (pnl / entryPrice) * 100,
					StopLoss:   stopLoss,
					TakeProfit: takeProfit,
					Reason:     fmt.Sprintf("%s - %s", entryReason, exitReason),
				}
				result.Trades = append(result.Trades, record)

				position = 0
				continue
			}

			// 止盈检查
			if (position == 1 && currentPrice >= takeProfit) ||
			   (position == -1 && currentPrice <= takeProfit) {
				exitReason := "触发止盈"
				pnl := calculatePnL(position, entryPrice, currentPrice)
				record := TradeRecord{
					Symbol:     symbol,
					Side:       getPositionSide(position),
					EntryTime:  entryTime,
					EntryPrice: entryPrice,
					ExitTime:   currentTime,
					ExitPrice:  currentPrice,
					Quantity:   1.0,
					PnL:        pnl,
					PnLPercent: (pnl / entryPrice) * 100,
					StopLoss:   stopLoss,
					TakeProfit: takeProfit,
					Reason:     fmt.Sprintf("%s - %s", entryReason, exitReason),
				}
				result.Trades = append(result.Trades, record)

				position = 0
				continue
			}
		}

		// 检查是否需要开仓（无持仓时）
		if position == 0 {
			// 均值回归信号：价格接近下轨且RSI超卖，做多
			if bbPosition < 0.3 && currentRSI < rsiOversold {
				strength := calculateReversionStrength(bbPosition, currentRSI, rsiOversold, 0.0, 0.2)
				if strength >= minReversionStrength {
					position = 1
					entryPrice = currentPrice
					entryTime = currentTime
					stopLoss = entryPrice * 0.92  // 8%止损
					takeProfit = entryPrice * 1.15 // 15%止盈
					entryReason = fmt.Sprintf("均值回归多头 (BB:%.2f, RSI:%.1f, 强度:%.2f)",
						bbPosition, currentRSI, strength)
				}
			}

			// 均值回归信号：价格接近上轨且RSI超买，做空
			if bbPosition > 0.7 && currentRSI > rsiOverbought {
				strength := calculateReversionStrength(bbPosition, currentRSI, 100-rsiOverbought, 0.8, 1.0)
				if strength >= minReversionStrength {
					position = -1
					entryPrice = currentPrice
					entryTime = currentTime
					stopLoss = entryPrice * 1.08  // 8%止损
					takeProfit = entryPrice * 0.85 // 15%止盈
					entryReason = fmt.Sprintf("均值回归空头 (BB:%.2f, RSI:%.1f, 强度:%.2f)",
						bbPosition, currentRSI, strength)
				}
			}
		}
	}

	// 计算统计结果
	calculateStatistics(result)

	return result, nil
}

func calculateReversionStrength(bbPosition, rsi, targetRSI, minBB, maxBB float64) float64 {
	// 布林带偏离程度
	bbDeviation := 0.0
	if bbPosition < minBB {
		bbDeviation = (minBB - bbPosition) / minBB
	} else if bbPosition > maxBB {
		bbDeviation = (bbPosition - maxBB) / (1 - maxBB)
	}

	// RSI偏离程度
	rsiDeviation := math.Abs(rsi-targetRSI) / 50.0 // 归一化到0-1

	// 综合强度
	strength := (bbDeviation + rsiDeviation) / 2.0
	return math.Min(strength, 1.0)
}

func calculatePnL(position int, entryPrice, exitPrice float64) float64 {
	// 假设每次交易使用1000元资金
	tradeAmount := 1000.0
	quantity := tradeAmount / entryPrice

	if position == 1 {
		// 多头：买入后卖出
		return (exitPrice - entryPrice) * quantity
	} else if position == -1 {
		// 空头：卖出后买入
		return (entryPrice - exitPrice) * quantity
	}
	return 0
}

func getPositionSide(position int) string {
	if position == 1 {
		return "long"
	} else if position == -1 {
		return "short"
	}
	return "unknown"
}

func calculateStatistics(result *BacktestResult) {
	if len(result.Trades) == 0 {
		return
	}

	totalPnL := 0.0
	winningPnL := 0.0
	losingPnL := 0.0
	maxDrawdown := 0.0
	peak := 0.0
	currentDrawdown := 0.0

	for _, trade := range result.Trades {
		totalPnL += trade.PnL

		if trade.PnL > 0 {
			result.WinningTrades++
			winningPnL += trade.PnL
			if trade.PnL > result.LargestWin {
				result.LargestWin = trade.PnL
			}
		} else {
			result.LosingTrades++
			losingPnL += trade.PnL
			if trade.PnL < result.LargestLoss {
				result.LargestLoss = trade.PnL
			}
		}

		// 计算最大回撤
		currentDrawdown += trade.PnL
		if currentDrawdown > 0 {
			currentDrawdown = 0
			peak = totalPnL
		} else if peak-totalPnL > maxDrawdown {
			maxDrawdown = peak - totalPnL
		}
	}

	result.TotalTrades = len(result.Trades)
	if result.TotalTrades > 0 {
		result.WinRate = float64(result.WinningTrades) / float64(result.TotalTrades)
		result.TotalPnL = totalPnL
		// TotalPnL已经是基于实际资金计算的盈利，不需要再转换
		result.TotalPnLPercent = totalPnL // 直接使用总盈利作为百分比显示
		result.MaxDrawdown = maxDrawdown

		if result.WinningTrades > 0 {
			result.AvgWinPnL = winningPnL / float64(result.WinningTrades)
		}
		if result.LosingTrades > 0 {
			result.AvgLossPnL = losingPnL / float64(result.LosingTrades)
		}

		if losingPnL != 0 {
			result.ProfitFactor = winningPnL / math.Abs(losingPnL)
		} else if winningPnL > 0 {
			result.ProfitFactor = 999.0 // 只有盈利没有亏损，设为很大值
		}

		if result.MaxDrawdown > 0 {
			result.RecoveryFactor = totalPnL / result.MaxDrawdown
		} else if totalPnL > 0 {
			result.RecoveryFactor = 999.0 // 无回撤，恢复因子无限大
		} else {
			result.RecoveryFactor = 0.0 // 无盈利，无回撤
		}
	}
}

func displayResults(results map[string]*BacktestResult, total *BacktestResult) {
	fmt.Println("\n📊 均值回归策略回测结果汇总")
	fmt.Println("=====================================")

	// 按PnL排序显示
	type symbolResult struct {
		symbol string
		result *BacktestResult
	}

	var sortedResults []symbolResult
	for symbol, result := range results {
		sortedResults = append(sortedResults, symbolResult{symbol, result})
	}

	sort.Slice(sortedResults, func(i, j int) bool {
		return sortedResults[i].result.TotalPnL > sortedResults[j].result.TotalPnL
	})

	fmt.Printf("%-12s %-8s %-8s %-8s %-10s %-10s %-10s\n",
		"币种", "交易数", "胜率", "总PnL", "最大回撤", "利润因子", "恢复因子")
	fmt.Println(strings.Repeat("-", 80))

	for _, sr := range sortedResults {
		result := sr.result
		fmt.Printf("%-12s %-8d %-7.1f%% %-8.2f %-9.2f %-9.2f %-9.2f\n",
			sr.symbol,
			result.TotalTrades,
			result.WinRate*100,
			result.TotalPnL,
			result.MaxDrawdown,
			result.ProfitFactor,
			result.RecoveryFactor,
		)
	}

	fmt.Println(strings.Repeat("-", 80))
	fmt.Printf("%-12s %-8d %-7.1f%% %-8.2f %-9.2f %-9.2f %-9.2f\n",
		"汇总",
		total.TotalTrades,
		total.WinRate*100,
		total.TotalPnL,
		total.MaxDrawdown,
		total.ProfitFactor,
		total.RecoveryFactor,
	)
}

func generateAnalysisReport(results map[string]*BacktestResult, total *BacktestResult) {
	fmt.Println("\n📋 策略分析报告")
	fmt.Println("=====================================")

	if total.TotalTrades == 0 {
		fmt.Println("❌ 无交易记录，无法生成分析报告")
		return
	}

	fmt.Printf("🎯 总体表现:\n")
	fmt.Printf("   总交易数: %d\n", total.TotalTrades)
	fmt.Printf("   胜率: %.1f%%\n", total.WinRate*100)
	fmt.Printf("   总盈亏: %.2f\n", total.TotalPnL)
	fmt.Printf("   平均每笔: %.2f\n", total.AvgTradePnL)
	fmt.Printf("   最大回撤: %.2f\n", total.MaxDrawdown)

	if total.ProfitFactor > 1.5 {
		fmt.Printf("   利润因子: %.2f ✅ (优秀)\n", total.ProfitFactor)
	} else if total.ProfitFactor > 1.2 {
		fmt.Printf("   利润因子: %.2f 👍 (良好)\n", total.ProfitFactor)
	} else {
		fmt.Printf("   利润因子: %.2f ⚠️ (需要改进)\n", total.ProfitFactor)
	}

	if total.RecoveryFactor > 2.0 {
		fmt.Printf("   恢复因子: %.2f ✅ (优秀)\n", total.RecoveryFactor)
	} else if total.RecoveryFactor > 1.0 {
		fmt.Printf("   恢复因子: %.2f 👍 (良好)\n", total.RecoveryFactor)
	} else {
		fmt.Printf("   恢复因子: %.2f ⚠️ (需要改进)\n", total.RecoveryFactor)
	}

	// 找出表现最好的币种
	var bestSymbol string
	var bestPnL float64 = -999999
	var worstSymbol string
	var worstPnL float64 = 999999

	for symbol, result := range results {
		if result.TotalPnL > bestPnL {
			bestPnL = result.TotalPnL
			bestSymbol = symbol
		}
		if result.TotalPnL < worstPnL {
			worstPnL = result.TotalPnL
			worstSymbol = symbol
		}
	}

	fmt.Printf("\n🏆 最佳表现币种: %s (PnL: %.2f)\n", bestSymbol, bestPnL)
	fmt.Printf("📉 最差表现币种: %s (PnL: %.2f)\n", worstSymbol, worstPnL)

	// 策略建议
	fmt.Printf("\n💡 策略建议:\n")
	if total.WinRate > 0.55 {
		fmt.Printf("   ✅ 胜率表现良好\n")
	} else {
		fmt.Printf("   ⚠️ 胜率偏低，建议调整入场条件\n")
	}

	if total.ProfitFactor > 1.3 {
		fmt.Printf("   ✅ 盈亏比合理\n")
	} else {
		fmt.Printf("   ⚠️ 盈亏比不佳，建议优化止损止盈设置\n")
	}

	if total.MaxDrawdown < total.TotalPnL*0.5 {
		fmt.Printf("   ✅ 回撤控制良好\n")
	} else {
		fmt.Printf("   ⚠️ 最大回撤较大，建议增加风险控制\n")
	}

	fmt.Printf("\n🎯 结论: ")
	if total.TotalPnL > 0 && total.WinRate > 0.5 && total.ProfitFactor > 1.2 {
		fmt.Printf("策略具有较好的盈利潜力，建议进一步优化和实盘测试\n")
	} else {
		fmt.Printf("策略表现一般，需要进一步优化参数和逻辑\n")
	}
}