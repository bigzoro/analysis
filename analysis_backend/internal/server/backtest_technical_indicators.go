package server

import (
	"log"
	"math"
)

// 注意：calculateTrendStrength函数已移至backtest_risk_manager.go中实现更完整的版本

// calculateVolatility 计算波动率
func (be *BacktestEngine) calculateVolatility(data []MarketData, period int) float64 {
	if len(data) < period+1 {
		return 0.001 // 返回最小波动率
	}

	// 计算收益率的标准差
	returns := make([]float64, 0, period)
	for i := len(data) - period; i < len(data); i++ {
		if i > 0 {
			prevPrice := data[i-1].Price
			currPrice := data[i].Price
			if prevPrice > 0 {
				returnRate := (currPrice - prevPrice) / prevPrice
				// 限制极端收益率（超过±50%可能是异常数据）
				if math.Abs(returnRate) < 0.5 {
					returns = append(returns, returnRate)
				}
			}
		}
	}

	if len(returns) < 3 { // 需要至少3个收益率来计算有意义的波动率
		return 0.001 // 返回最小波动率
	}

	// 计算标准差
	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		variance += (ret - mean) * (ret - mean)
	}
	variance /= float64(len(returns) - 1)

	volatility := math.Sqrt(variance)

	// 调试：如果波动率为0，记录详细信息
	if volatility == 0.0 && len(returns) > 0 {
		log.Printf("[DEBUG_VOLATILITY] 波动率为0，收益率数量=%d，样本收益率: %.6f, %.6f, %.6f...",
			len(returns), returns[0], returns[1], returns[2])
		// 确保波动率不为0，至少返回一个最小值
		volatility = 0.0001 // 最小波动率0.01%
	}

	// 数据验证：波动率不应该超过合理范围
	if volatility > 1.0 { // 波动率超过100%是不合理的
		log.Printf("[WARN] 异常高波动率检测: %.3f, 使用平均波动率替代", volatility)
		// 使用历史平均波动率而不是0
		return 0.02 // 2%的默认波动率
	}

	// 如果波动率太小，可能是数据问题，给一个最小值
	if volatility < 0.001 { // 小于0.1%的波动率
		log.Printf("[WARN] 波动率过小: %.6f, 可能数据质量问题，使用最小波动率", volatility)
		// 给一个小的基础波动率，避免完全为0
		return 0.005 // 0.5%的最小波动率
	}

	return volatility
}

// calculateVolumeTrend 计算成交量趋势
func (be *BacktestEngine) calculateVolumeTrend(data []MarketData, period int) float64 {
	if len(data) < period {
		return 0
	}

	// 计算成交量的移动平均线斜率
	volumes := make([]float64, period)
	for i := len(data) - period; i < len(data); i++ {
		volumes[i-(len(data)-period)] = data[i].Volume24h
	}

	// 计算线性回归斜率
	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	n := float64(period)

	for i := 0; i < period; i++ {
		x := float64(i)
		y := volumes[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	slope := (n*sumXY - sumX*sumY) / (n*sumX2 - sumX*sumX)

	// 返回标准化斜率
	if len(volumes) > 0 {
		avgVolume := 0.0
		for _, v := range volumes {
			avgVolume += v
		}
		avgVolume /= float64(len(volumes))
		if avgVolume > 0 {
			return slope / avgVolume
		}
	}

	return slope
}

// calculateRSI 计算RSI指标
func (be *BacktestEngine) calculateRSI(data []MarketData, period int) float64 {
	if len(data) < period+1 {
		return 50 // 默认中性值
	}

	// 检查数据质量：如果所有价格都相同，返回50
	allSame := true
	firstPrice := data[0].Price
	for _, md := range data {
		if md.Price != firstPrice {
			allSame = false
			break
		}
	}
	if allSame {
		return 50 // 数据无变化，返回中性RSI
	}

	gains := make([]float64, 0, period)
	losses := make([]float64, 0, period)

	// 计算价格变化
	for i := len(data) - period - 1; i < len(data)-1; i++ {
		change := data[i+1].Price - data[i].Price
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// 计算平均涨幅和跌幅
	avgGain := 0.0
	avgLoss := 0.0

	for i := 0; i < len(gains); i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}

	if len(gains) > 0 {
		avgGain /= float64(len(gains))
		avgLoss /= float64(len(losses))
	}

	// 计算RSI
	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// calculateMACDSignal 计算MACD信号
func (be *BacktestEngine) calculateMACDSignal(data []MarketData) float64 {
	if len(data) < 26 {
		return 0
	}

	// 计算EMA12, EMA26, MACD线
	ema12 := be.calculateEMAFromData(data, 12)
	ema26 := be.calculateEMAFromData(data, 26)

	macdLine := ema12 - ema26

	// 计算信号线 (MACD线的9日EMA)
	// 需要至少26+9=35个数据点来计算信号线
	if len(data) < 35 {
		return 0
	}

	// 计算MACD线的时间序列（最后26个周期）
	macdValues := make([]float64, 26)
	for i := 0; i < 26; i++ {
		idx := len(data) - 26 + i
		if idx >= 12 { // 确保有足够数据计算EMA12
			ema12AtI := be.calculateEMAFromData(data[idx-12+1:idx+1], 12)
			ema26AtI := be.calculateEMAFromData(data[idx-26+1:idx+1], 26)
			macdValues[i] = ema12AtI - ema26AtI
		}
	}

	signalLine := be.calculateEMA(macdValues, 9)

	macdSignal := macdLine - signalLine

	// 数据验证：MACD信号不应该超过合理范围
	if math.Abs(macdSignal) > 10000 { // MACD信号超过10000是不合理的
		log.Printf("[WARN] 异常MACD信号检测: %.3f (MACD线: %.3f, 信号线: %.3f), 使用0替代", macdSignal, macdLine, signalLine)
		return 0
	}

	return macdSignal
}

// calculateEMAFromData 从MarketData计算EMA
func (be *BacktestEngine) calculateEMAFromData(data []MarketData, period int) float64 {
	if len(data) < period {
		return 0
	}

	prices := make([]float64, len(data))
	for i, md := range data {
		prices[i] = md.Price
	}

	return be.calculateEMA(prices[len(prices)-period:], period)
}

// calculateEMA 计算指数移动平均
func (be *BacktestEngine) calculateEMA(values []float64, period int) float64 {
	if len(values) == 0 {
		return 0
	}

	multiplier := 2.0 / (float64(period) + 1.0)
	ema := values[0]

	for i := 1; i < len(values); i++ {
		ema = (values[i] * multiplier) + (ema * (1 - multiplier))
	}

	return ema
}

// 注意：calculateSignalConsistency函数已移至backtest_strategy_executor.go中实现更完整的版本

// calculateAdvancedSignalConsistency 高级信号一致性检查
func (be *BacktestEngine) calculateAdvancedSignalConsistency(state map[string]float64, hasPosition bool) float64 {
	bullishSignals := 0
	bearishSignals := 0
	totalSignals := 0

	// 检查各种技术指标
	signals := []string{"trend_5", "trend_20", "trend_50", "rsi_14", "momentum_10", "macd_signal"}

	for _, signalName := range signals {
		if value, exists := state[signalName]; exists && value != 0 {
			totalSignals++
			if value > 0 {
				bullishSignals++
			} else {
				bearishSignals++
			}
		}
	}

	if totalSignals == 0 {
		return 0.5 // 默认中性
	}

	// 计算一致性
	consistency := float64(bullishSignals) / float64(totalSignals)

	// 根据持仓状态调整一致性阈值
	if hasPosition {
		// 有持仓时，需要更强的看涨一致性才能继续持有
		if consistency > 0.6 {
			return consistency
		} else {
			return consistency * 0.8 // 降低一致性评分
		}
	} else {
		// 无持仓时，需要更强的看涨一致性才能买入
		if consistency > 0.7 {
			return consistency
		} else {
			return consistency * 0.9 // 适度降低一致性评分
		}
	}
}

// buildDeepState 构建深度学习状态特征
func (be *BacktestEngine) buildDeepState(data []MarketData, currentData MarketData) map[string]float64 {
	state := make(map[string]float64)

	if len(data) < 50 {
		return state
	}

	// 价格和趋势指标
	state["price"] = currentData.Price
	trend5 := be.analyzeTrendStrength(data, len(data)-1, 5)
	trend20 := be.analyzeTrendStrength(data, len(data)-1, 20)
	trend50 := be.analyzeTrendStrength(data, len(data)-1, 50)
	state["trend_5"] = trend5.Strength
	state["trend_20"] = trend20.Strength
	state["trend_50"] = trend50.Strength

	// 波动率指标
	state["volatility_20"] = be.calculateVolatility(data[len(data)-20:], 20)

	// 成交量指标
	state["volume_trend"] = be.calculateVolumeTrend(data[len(data)-20:], 20)

	// 动量指标
	state["rsi_14"] = be.calculateRSI(data[len(data)-20:], 14)
	state["macd_signal"] = be.calculateMACDSignal(data[len(data)-30:])

	// 动量指标 (10日)
	if len(data) >= 10 {
		recentPrices := make([]float64, 10)
		for i := 0; i < 10; i++ {
			recentPrices[i] = data[len(data)-10+i].Price
		}
		state["momentum_10"] = be.calculateMomentum(recentPrices)
	} else {
		state["momentum_10"] = 0.0
	}

	// 市场结构指标
	state["support_level"] = be.calculateSupportLevel(data[len(data)-20:])
	state["resistance_level"] = be.calculateResistanceLevel(data[len(data)-20:])
	state["bollinger_position"] = be.calculateBollingerPosition(data[len(data)-20:])

	// 随机指标
	state["stoch_k"] = be.calculateStochasticK(data[len(data)-20:])

	// 威廉指标
	state["williams_r"] = be.calculateWilliamsR(data[len(data)-20:])

	// 市场阶段判断
	state["market_phase"] = be.calculateMarketPhase(state)

	// 新增：时间序列动量特征
	if len(data) >= 10 {
		currentPrice := currentData.Price

		// 价格动量序列 - 捕捉短期加速/减速
		state["price_momentum_3"] = be.calculatePriceChange(data[len(data)-3:], currentPrice)
		state["price_momentum_5"] = be.calculatePriceChange(data[len(data)-5:], currentPrice)
		state["price_acceleration"] = state["price_momentum_3"] - state["price_momentum_5"]

		// 价格动量强度 - 相对波动率归一化
		if vol, exists := state["volatility_20"]; exists && vol > 0 {
			state["price_momentum_normalized"] = state["price_momentum_5"] / vol
		}
	}

	// 新增：成交量趋势特征
	if len(data) >= 20 {
		// 成交量相对强弱
		state["volume_rsi"] = be.calculateVolumeRSI(data[len(data)-20:])

		// 成交量价格趋势 (Volume Price Trend)
		state["volume_price_trend"] = be.calculateVolumePriceTrend(data[len(data)-20:])

		// 成交量波动率
		state["volume_volatility"] = be.calculateVolumeVolatility(data[len(data)-20:])
	}

	// 新增：市场微观结构特征
	if len(data) >= 30 {
		// 价格跳跃特征 - 捕捉异常波动
		state["price_jump_ratio"] = be.calculatePriceJumpRatio(data[len(data)-30:])

		// 注意：趋势一致性计算已移至backtest_strategy_executor.go中实现
	}

	return state
}

// calculateMomentum 计算动量
func (be *BacktestEngine) calculateMomentum(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	// 计算价格变化率
	current := prices[len(prices)-1]
	previous := prices[0]

	if previous == 0 {
		return 0
	}

	return (current - previous) / previous
}

// calculateSupportLevel 计算支撑位
func (be *BacktestEngine) calculateSupportLevel(data []MarketData) float64 {
	if len(data) < 10 {
		return 0
	}

	currentPrice := data[len(data)-1].Price
	if currentPrice == 0 {
		return 0
	}

	// 1. 计算多个时间周期的支撑位
	supportLevels := []float64{}

	// 短期支撑（最近5日）
	if len(data) >= 5 {
		shortTermMin := data[len(data)-1].Price
		for i := len(data) - 5; i < len(data); i++ {
			if data[i].Price < shortTermMin {
				shortTermMin = data[i].Price
			}
		}
		supportLevels = append(supportLevels, shortTermMin)
	}

	// 中期支撑（最近10日）
	if len(data) >= 10 {
		midTermMin := data[len(data)-1].Price
		for i := len(data) - 10; i < len(data); i++ {
			if data[i].Price < midTermMin {
				midTermMin = data[i].Price
			}
		}
		supportLevels = append(supportLevels, midTermMin)
	}

	// 长期支撑（最近20日）
	longTermMin := data[0].Price
	for _, md := range data {
		if md.Price < longTermMin {
			longTermMin = md.Price
		}
	}
	supportLevels = append(supportLevels, longTermMin)

	// 2. 找到最具影响力的支撑位
	// 优先选择离当前价格最近且被多次触及的支撑位
	bestSupport := longTermMin // 默认选择长期支撑
	minDistance := math.Abs(currentPrice - bestSupport)

	for _, support := range supportLevels {
		distance := math.Abs(currentPrice - support)
		if distance < minDistance && support < currentPrice { // 只考虑下方支撑
			minDistance = distance
			bestSupport = support
		}
	}

	// 3. 计算支撑强度（离当前价格的距离，越近越重要）
	supportStrength := 0.0
	if bestSupport > 0 && bestSupport < currentPrice {
		distanceRatio := (currentPrice - bestSupport) / currentPrice
		if distanceRatio < 0.02 { // 2%以内，强支撑
			supportStrength = 0.15
		} else if distanceRatio < 0.05 { // 5%以内，中等支撑
			supportStrength = 0.08
		} else if distanceRatio < 0.10 { // 10%以内，弱支撑
			supportStrength = 0.03
		}
		// 超过10%的支撑位影响力很弱
	}

	return supportStrength
}

// calculateResistanceLevel 计算阻力位 - 优化版本
func (be *BacktestEngine) calculateResistanceLevel(data []MarketData) float64 {
	if len(data) < 10 {
		return 0
	}

	currentPrice := data[len(data)-1].Price
	if currentPrice == 0 {
		return 0
	}

	// 1. 计算多个时间周期的阻力位
	resistanceLevels := []float64{}

	// 短期阻力（最近5日）
	if len(data) >= 5 {
		shortTermMax := data[len(data)-1].Price
		for i := len(data) - 5; i < len(data); i++ {
			if data[i].Price > shortTermMax {
				shortTermMax = data[i].Price
			}
		}
		resistanceLevels = append(resistanceLevels, shortTermMax)
	}

	// 中期阻力（最近10日）
	if len(data) >= 10 {
		midTermMax := data[len(data)-1].Price
		for i := len(data) - 10; i < len(data); i++ {
			if data[i].Price > midTermMax {
				midTermMax = data[i].Price
			}
		}
		resistanceLevels = append(resistanceLevels, midTermMax)
	}

	// 长期阻力（最近20日）
	longTermMax := data[0].Price
	for _, md := range data {
		if md.Price > longTermMax {
			longTermMax = md.Price
		}
	}
	resistanceLevels = append(resistanceLevels, longTermMax)

	// 2. 找到最具影响力的阻力位
	bestResistance := longTermMax // 默认选择长期阻力
	minDistance := math.Abs(bestResistance - currentPrice)

	for _, resistance := range resistanceLevels {
		distance := math.Abs(currentPrice - resistance)
		if distance < minDistance && resistance > currentPrice { // 只考虑上方阻力
			minDistance = distance
			bestResistance = resistance
		}
	}

	// 3. 计算阻力强度（离当前价格的距离，越近越重要）
	resistanceStrength := 0.0
	if bestResistance > currentPrice {
		distanceRatio := (bestResistance - currentPrice) / currentPrice
		if distanceRatio < 0.02 { // 2%以内，强阻力
			resistanceStrength = 0.15
		} else if distanceRatio < 0.05 { // 5%以内，中等阻力
			resistanceStrength = 0.08
		} else if distanceRatio < 0.10 { // 10%以内，弱阻力
			resistanceStrength = 0.03
		}
		// 超过10%的阻力位影响力很弱
	}

	return resistanceStrength
}

// calculateBollingerPosition 计算布林带位置
func (be *BacktestEngine) calculateBollingerPosition(data []MarketData) float64 {
	if len(data) < 20 {
		return 0
	}

	// 计算20日简单移动平均
	sum := 0.0
	for _, md := range data {
		sum += md.Price
	}
	sma := sum / float64(len(data))

	// 计算标准差
	variance := 0.0
	for _, md := range data {
		variance += (md.Price - sma) * (md.Price - sma)
	}
	stdDev := math.Sqrt(variance / float64(len(data)))

	if stdDev == 0 {
		return 0
	}

	currentPrice := data[len(data)-1].Price
	return (currentPrice - sma) / (2 * stdDev) // 标准化到[-1, 1]范围
}

// calculateStochasticK 计算随机指标K值
func (be *BacktestEngine) calculateStochasticK(data []MarketData) float64 {
	if len(data) < 14 {
		return 50
	}

	// 计算14日内的最高价和最低价
	high := data[0].Price
	low := data[0].Price

	for _, md := range data {
		if md.Price > high {
			high = md.Price
		}
		if md.Price < low {
			low = md.Price
		}
	}

	currentPrice := data[len(data)-1].Price
	if high == low {
		return 50
	}

	return ((currentPrice - low) / (high - low)) * 100
}

// calculateWilliamsR 计算威廉指标
func (be *BacktestEngine) calculateWilliamsR(data []MarketData) float64 {
	if len(data) < 14 {
		return -50
	}

	// 计算14日内的最高价和最低价
	high := data[0].Price
	low := data[0].Price

	for _, md := range data {
		if md.Price > high {
			high = md.Price
		}
		if md.Price < low {
			low = md.Price
		}
	}

	currentPrice := data[len(data)-1].Price
	if high == low {
		return -50
	}

	return ((high - currentPrice) / (high - low)) * -100
}

// calculateMarketPhase 计算市场阶段
func (be *BacktestEngine) calculateMarketPhase(state map[string]float64) float64 {
	// 基于多个指标判断市场阶段
	trendScore := 0.0
	volatilityScore := 0.0

	// 趋势强度
	if trend, exists := state["trend_20"]; exists {
		trendScore = trend * 10 // 放大趋势信号
	}

	// 波动率
	if vol, exists := state["volatility_20"]; exists {
		volatilityScore = vol * 100 // 转换为百分比
	}

	// RSI判断超买超卖
	rsiScore := 0.0
	if rsi, exists := state["rsi_14"]; exists {
		if rsi > 70 {
			rsiScore = 0.5 // 超买
		} else if rsi < 30 {
			rsiScore = -0.5 // 超卖
		}
	}

	// 综合判断市场阶段
	phaseScore := trendScore + rsiScore

	// 根据波动率调整
	if volatilityScore > 3 {
		phaseScore *= 0.8 // 高波动时降低趋势权重
	}

	return phaseScore
}

// calculatePriceChange 计算价格变化率
func (be *BacktestEngine) calculatePriceChange(recentData []MarketData, currentPrice float64) float64 {
	if len(recentData) < 2 {
		return 0
	}

	oldestPrice := recentData[0].Price
	if oldestPrice == 0 {
		return 0
	}

	return (currentPrice - oldestPrice) / oldestPrice
}

// calculateVolumeRSI 计算成交量RSI
func (be *BacktestEngine) calculateVolumeRSI(data []MarketData) float64 {
	if len(data) < 14 {
		return 50
	}

	volumes := make([]float64, len(data))
	for i, md := range data {
		volumes[i] = md.Volume24h
	}

	return be.calculateRSIFromValues(volumes, 14)
}

// calculateRSIFromValues 从数值数组计算RSI
func (be *BacktestEngine) calculateRSIFromValues(values []float64, period int) float64 {
	if len(values) < period+1 {
		return 50
	}

	gains := 0.0
	losses := 0.0

	// 计算初始平均值
	for i := 1; i <= period; i++ {
		change := values[i] - values[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// 计算后续值
	for i := period + 1; i < len(values); i++ {
		change := values[i] - values[i-1]
		if change > 0 {
			avgGain = (avgGain*13 + change) / 14
			avgLoss = (avgLoss*13 + 0) / 14
		} else {
			avgGain = (avgGain*13 + 0) / 14
			avgLoss = (avgLoss*13 - change) / 14
		}
	}

	if avgLoss == 0 {
		return 100
	}

	rs := avgGain / avgLoss
	return 100 - (100 / (1 + rs))
}

// calculateVolumePriceTrend 计算成交量价格趋势(VPT)，标准化为-1到1之间
func (be *BacktestEngine) calculateVolumePriceTrend(data []MarketData) float64 {
	if len(data) < 5 {
		return 0
	}

	// 计算VPT
	rawVpt := 0.0
	priceChanges := make([]float64, 0, len(data)-1)

	for i := 1; i < len(data); i++ {
		priceChange := (data[i].Price - data[i-1].Price) / data[i-1].Price
		priceChanges = append(priceChanges, priceChange)
		rawVpt += priceChange * data[i].Volume24h
	}

	// 计算平均VPT
	avgVpt := rawVpt / float64(len(data)-1)

	// 标准化：计算VPT的标准差
	sumSquares := 0.0
	for _, change := range priceChanges {
		sumSquares += change * change
	}

	if len(priceChanges) == 0 {
		return 0
	}

	// 使用价格变化的标准差作为标准化因子
	priceStdDev := math.Sqrt(sumSquares / float64(len(priceChanges)))

	if priceStdDev == 0 {
		return 0
	}

	// 返回标准化的VPT，限制在-1到1之间
	normalizedVpt := avgVpt / priceStdDev
	if normalizedVpt > 1.0 {
		return 1.0
	} else if normalizedVpt < -1.0 {
		return -1.0
	}

	return normalizedVpt
}

// calculateVolumeVolatility 计算成交量波动率（相对波动系数，0-1之间）
func (be *BacktestEngine) calculateVolumeVolatility(data []MarketData) float64 {
	if len(data) < 10 {
		return 0
	}

	volumes := make([]float64, len(data))
	sum := 0.0

	for i, md := range data {
		volumes[i] = md.Volume24h
		sum += md.Volume24h
	}

	mean := sum / float64(len(data))
	if mean == 0 {
		return 0
	}

	// 计算方差
	variance := 0.0
	for _, volume := range volumes {
		variance += (volume - mean) * (volume - mean)
	}

	// 计算相对波动系数（标准差/均值），限制在0-1之间
	volatility := math.Sqrt(variance / float64(len(data)))
	relativeVolatility := volatility / mean

	// 限制在合理范围内，避免极端值
	if relativeVolatility > 1.0 {
		relativeVolatility = 1.0
	} else if relativeVolatility < 0 {
		relativeVolatility = 0
	}

	return relativeVolatility
}

// calculatePriceJumpRatio 计算价格跳跃比率
func (be *BacktestEngine) calculatePriceJumpRatio(data []MarketData) float64 {
	if len(data) < 30 {
		return 0
	}

	// 计算平均价格变化
	totalChange := 0.0
	for i := 1; i < len(data); i++ {
		change := math.Abs((data[i].Price - data[i-1].Price) / data[i-1].Price)
		totalChange += change
	}
	avgChange := totalChange / float64(len(data)-1)

	// 计算标准差
	variance := 0.0
	for i := 1; i < len(data); i++ {
		change := math.Abs((data[i].Price - data[i-1].Price) / data[i-1].Price)
		variance += (change - avgChange) * (change - avgChange)
	}
	stdDev := math.Sqrt(variance / float64(len(data)-1))

	// 计算跳跃比率 (最近一次变化 / 平均变化)
	if len(data) >= 2 {
		recentChange := math.Abs((data[len(data)-1].Price - data[len(data)-2].Price) / data[len(data)-2].Price)
		if avgChange > 0 {
			return (recentChange - avgChange) / stdDev // 标准化跳跃
		}
	}

	return 0
}

// 注意：calculateTrendConsistency函数已移至backtest_strategy_executor.go中实现更完整的版本
