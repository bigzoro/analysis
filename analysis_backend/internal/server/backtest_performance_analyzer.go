package server

import (
	"math"
)

// calculateSummary 计算回测摘要
func (be *BacktestEngine) calculateSummary(result *BacktestResult, initialCash float64) {
	totalTrades := len(result.Trades)
	if totalTrades == 0 {
		result.Summary = BacktestSummary{
			TotalTrades:   0,
			WinningTrades: 0,
			LosingTrades:  0,
			WinRate:       0,
			TotalReturn:   0,
			MaxDrawdown:   0,
			SharpeRatio:   0,
		}
		return
	}

	winningTrades := 0
	losingTrades := 0
	totalPnL := 0.0
	totalCommission := 0.0

	// 计算交易统计
	for _, trade := range result.Trades {
		if trade.Side == "sell" && trade.PnL != 0 {
			totalPnL += trade.PnL
			totalCommission += trade.Commission

			if trade.PnL > 0 {
				winningTrades++
			} else {
				losingTrades++
			}
		}
	}

	// 计算胜率
	winRate := 0.0
	if winningTrades+losingTrades > 0 {
		winRate = float64(winningTrades) / float64(winningTrades+losingTrades)
	}

	// 计算总收益率
	totalReturn := 0.0
	if initialCash > 0 {
		totalReturn = totalPnL / initialCash
	}

	// 计算最大回撤
	maxDrawdown := be.calculateMaxDrawdown(result.DailyReturns)

	// 计算夏普比率（简化版）
	sharpeRatio := 0.0
	if len(result.DailyReturns) > 1 {
		// 计算日收益率的平均值和标准差
		dailyReturns := make([]float64, len(result.DailyReturns))
		for i, dr := range result.DailyReturns {
			if i > 0 {
				dailyReturns[i] = (dr.Value - result.DailyReturns[i-1].Value) / result.DailyReturns[i-1].Value
			}
		}

		// 移除第一个0值
		if len(dailyReturns) > 1 {
			dailyReturns = dailyReturns[1:]
		}

		meanReturn, stdDev := be.calculateMeanAndStdDev(dailyReturns)
		if stdDev > 0 {
			// 假设无风险利率为0.02 (2%)
			riskFreeRate := 0.02 / 252 // 日化无风险利率
			sharpeRatio = (meanReturn - riskFreeRate) / stdDev
		}
	}

	result.Summary = BacktestSummary{
		TotalTrades:     totalTrades,
		WinningTrades:   winningTrades,
		LosingTrades:    losingTrades,
		WinRate:         winRate,
		TotalReturn:     totalReturn,
		AnnualReturn:    totalReturn, // 这里可以进一步计算年化收益
		MaxDrawdown:     maxDrawdown,
		SharpeRatio:     sharpeRatio,
		Volatility:      0, // 这里可以计算波动率
		TotalCommission: totalCommission,
	}

	// 填充Performance字段
	result.Performance.TotalReturn = totalReturn
	result.Performance.WinRate = winRate
	result.Performance.MaxDrawdown = maxDrawdown
	result.Performance.SharpeRatio = sharpeRatio
}

// calculatePerformanceMetrics 计算绩效指标
func (be *BacktestEngine) calculatePerformanceMetrics(result *BacktestResult) {
	if len(result.DailyReturns) == 0 {
		return
	}

	// 计算日收益率序列
	dailyReturns := make([]float64, 0, len(result.DailyReturns)-1)
	for i := 1; i < len(result.DailyReturns); i++ {
		if result.DailyReturns[i-1].Value > 0 {
			dailyReturn := (result.DailyReturns[i].Value - result.DailyReturns[i-1].Value) / result.DailyReturns[i-1].Value
			dailyReturns = append(dailyReturns, dailyReturn)
			result.DailyReturns[i].Return = dailyReturn
		}
	}

	if len(dailyReturns) == 0 {
		return
	}

	// 计算基本指标
	meanReturn, volatility := be.calculateMeanAndStdDev(dailyReturns)

	// 计算夏普比率
	riskFreeRate := 0.02 / 252 // 日化无风险利率2%
	sharpeRatio := 0.0
	if volatility > 0 {
		sharpeRatio = (meanReturn - riskFreeRate) / volatility
	}

	// 计算索提诺比率（只考虑下行波动）
	downsideReturns := make([]float64, 0)
	for _, ret := range dailyReturns {
		if ret < riskFreeRate {
			downsideReturns = append(downsideReturns, ret-riskFreeRate)
		}
	}

	sortinoRatio := 0.0
	if len(downsideReturns) > 0 {
		downsideDeviation := be.calculateStdDev(downsideReturns)
		if downsideDeviation > 0 {
			sortinoRatio = (meanReturn - riskFreeRate) / downsideDeviation
		}
	}

	// 计算最大回撤
	maxDrawdown := be.calculateMaxDrawdown(result.DailyReturns)

	// 计算Calmar比率
	calmarRatio := 0.0
	if maxDrawdown > 0 {
		calmarRatio = meanReturn * 252 / maxDrawdown // 年化收益除以最大回撤
	}

	// 计算信息比率（这里简化为夏普比率）
	informationRatio := sharpeRatio

	// 计算Omega比率
	omegaRatio := be.calculateOmegaRatio(dailyReturns, riskFreeRate)

	// 计算盈亏比
	gainToPainRatio := be.calculateGainToPainRatio(dailyReturns)

	// 增强绩效指标
	recoveryFactor := 0.0
	if maxDrawdown > 0 {
		totalReturn := result.DailyReturns[len(result.DailyReturns)-1].Value - result.DailyReturns[0].Value
		if result.DailyReturns[0].Value > 0 {
			totalReturn /= result.DailyReturns[0].Value
		}
		recoveryFactor = totalReturn / maxDrawdown
	}

	profitFactor := be.calculateProfitFactor(result.Trades)
	payoffRatio := be.calculatePayoffRatio(result.Trades)
	kRatio := be.calculateKRatio(dailyReturns)
	safeFDrawdown := be.calculateSafeFDrawdown(result.DailyReturns)

	result.Performance = PerformanceMetrics{
		TotalReturn:      result.Summary.TotalReturn,
		AnnualReturn:     meanReturn * 252,            // 年化收益
		Volatility:       volatility * math.Sqrt(252), // 年化波动率
		SharpeRatio:      sharpeRatio,
		SortinoRatio:     sortinoRatio,
		MaxDrawdown:      maxDrawdown,
		CalmarRatio:      calmarRatio,
		InformationRatio: informationRatio,
		OmegaRatio:       omegaRatio,
		GainToPainRatio:  gainToPainRatio,
		RecoveryFactor:   recoveryFactor,
		ProfitFactor:     profitFactor,
		PayoffRatio:      payoffRatio,
		KRatio:           kRatio,
		SafeFDrawdown:    safeFDrawdown,
		WinRate:          result.Summary.WinRate,
	}
}

// calculateMaxDrawdown 计算最大回撤
func (be *BacktestEngine) calculateMaxDrawdown(dailyReturns []DailyReturn) float64 {
	if len(dailyReturns) == 0 {
		return 0
	}

	maxDrawdown := 0.0
	peak := dailyReturns[0].Value

	for _, dr := range dailyReturns {
		if dr.Value > peak {
			peak = dr.Value
		}

		drawdown := (peak - dr.Value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateMeanAndStdDev 计算均值和标准差
func (be *BacktestEngine) calculateMeanAndStdDev(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}

	// 计算均值
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(len(data))

	// 计算标准差
	variance := 0.0
	for _, v := range data {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(data) - 1)
	stdDev := math.Sqrt(variance)

	return mean, stdDev
}

// calculateStdDev 计算标准差
func (be *BacktestEngine) calculateStdDev(data []float64) float64 {
	_, stdDev := be.calculateMeanAndStdDev(data)
	return stdDev
}

// calculateOmegaRatio 计算Omega比率
func (be *BacktestEngine) calculateOmegaRatio(returns []float64, threshold float64) float64 {
	if len(returns) == 0 {
		return 1.0
	}

	aboveThreshold := 0.0
	belowThreshold := 0.0

	for _, ret := range returns {
		if ret > threshold {
			aboveThreshold += ret - threshold
		} else {
			belowThreshold += threshold - ret
		}
	}

	if belowThreshold == 0 {
		return 100.0 // 避免除零
	}

	return aboveThreshold / belowThreshold
}

// calculateGainToPainRatio 计算盈亏比
func (be *BacktestEngine) calculateGainToPainRatio(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	totalGain := 0.0
	totalLoss := 0.0

	for _, ret := range returns {
		if ret > 0 {
			totalGain += ret
		} else {
			totalLoss += math.Abs(ret)
		}
	}

	if totalLoss == 0 {
		return 100.0 // 避免除零
	}

	return totalGain / totalLoss
}

// calculateProfitFactor 计算盈利因子
func (be *BacktestEngine) calculateProfitFactor(trades []TradeRecord) float64 {
	totalProfit := 0.0
	totalLoss := 0.0

	for _, trade := range trades {
		if trade.Side == "sell" && trade.PnL != 0 {
			if trade.PnL > 0 {
				totalProfit += trade.PnL
			} else {
				totalLoss += math.Abs(trade.PnL)
			}
		}
	}

	if totalLoss == 0 {
		return 100.0 // 避免除零
	}

	return totalProfit / totalLoss
}

// calculatePayoffRatio 计算赔付比率
func (be *BacktestEngine) calculatePayoffRatio(trades []TradeRecord) float64 {
	profits := make([]float64, 0)
	losses := make([]float64, 0)

	for _, trade := range trades {
		if trade.Side == "sell" && trade.PnL != 0 {
			if trade.PnL > 0 {
				profits = append(profits, trade.PnL)
			} else {
				losses = append(losses, math.Abs(trade.PnL))
			}
		}
	}

	if len(profits) == 0 || len(losses) == 0 {
		return 0
	}

	avgProfit := 0.0
	for _, p := range profits {
		avgProfit += p
	}
	avgProfit /= float64(len(profits))

	avgLoss := 0.0
	for _, l := range losses {
		avgLoss += l
	}
	avgLoss /= float64(len(losses))

	if avgLoss == 0 {
		return 100.0
	}

	return avgProfit / avgLoss
}

// calculateKRatio 计算K比率
func (be *BacktestEngine) calculateKRatio(returns []float64) float64 {
	if len(returns) < 10 {
		return 0
	}

	// 计算价格序列的效率系数
	prices := make([]float64, len(returns)+1)
	prices[0] = 1000.0 // 初始价格

	for i, ret := range returns {
		prices[i+1] = prices[i] * (1 + ret)
	}

	// 计算K比率（简化版）
	r2 := be.calculateR2(prices)
	kRatio := r2 * math.Sqrt(float64(len(prices)))

	return kRatio
}

// calculateR2 计算R²值
func (be *BacktestEngine) calculateR2(y []float64) float64 {
	if len(y) < 2 {
		return 0
	}

	// 简单线性回归
	n := float64(len(y))
	sumX := 0.0
	sumY := 0.0
	sumXY := 0.0
	sumX2 := 0.0

	for i, val := range y {
		x := float64(i)
		sumX += x
		sumY += val
		sumXY += x * val
		sumX2 += x * x
	}

	// 斜率和截距
	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)
	intercept := (sumY - slope*sumX) / n

	// 计算R²
	ssRes := 0.0
	ssTot := 0.0
	meanY := sumY / n

	for i, val := range y {
		x := float64(i)
		predicted := slope*x + intercept
		ssRes += (val - predicted) * (val - predicted)
		ssTot += (val - meanY) * (val - meanY)
	}

	if ssTot == 0 {
		return 1.0
	}

	return 1 - (ssRes / ssTot)
}

// calculateSafeFDrawdown 计算安全F回撤
func (be *BacktestEngine) calculateSafeFDrawdown(dailyReturns []DailyReturn) float64 {
	if len(dailyReturns) < 2 {
		return 0
	}

	// F值计算（简化版）
	peak := dailyReturns[0].Value
	maxDrawdown := 0.0
	currentDrawdown := 0.0

	for _, dr := range dailyReturns {
		if dr.Value > peak {
			peak = dr.Value
			currentDrawdown = 0
		} else {
			currentDrawdown = (peak - dr.Value) / peak
			if currentDrawdown > maxDrawdown {
				maxDrawdown = currentDrawdown
			}
		}
	}

	// 安全F回撤 = 最大回撤 * (1 - 恢复因子)
	recoveryFactor := 1.0
	if maxDrawdown > 0 {
		finalValue := dailyReturns[len(dailyReturns)-1].Value
		initialValue := dailyReturns[0].Value
		if initialValue > 0 {
			totalReturn := (finalValue - initialValue) / initialValue
			recoveryFactor = totalReturn / maxDrawdown
		}
	}

	safeFDrawdown := maxDrawdown * (1 - recoveryFactor)
	if safeFDrawdown < 0 {
		safeFDrawdown = 0
	}

	return safeFDrawdown
}
