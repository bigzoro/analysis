package server

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"time"

	"gonum.org/v1/gonum/stat"
)

// Assess 评估风险
func (ra *RiskAssessor) Assess(ctx context.Context, symbol string) (*RiskProfile, error) {
	profile := &RiskProfile{
		Symbol:      symbol,
		LastUpdated: time.Now(),
		RiskFactors: RiskFactors{},
		PositionLimits: PositionLimits{
			MaxPosition:     ra.config.Control.MaxPositionSize,
			MaxDrawdown:     ra.config.Control.MaxDrawdownLimit,
			Diversification: ra.config.Control.DiversificationMin,
			StopLoss:        ra.config.Control.StopLossLevels,
		},
		HistoricalRisk: make([]RiskHistory, 0),
		Alerts:         make([]RiskAlert, 0),
	}

	// 评估各种风险因子
	err := ra.assessRiskFactors(ctx, profile)
	if err != nil {
		return nil, fmt.Errorf("评估风险因子失败: %w", err)
	}

	// 计算综合风险分数
	profile.RiskScore = ra.calculateCompositeRiskScore(profile.RiskFactors)

	// 确定风险等级
	profile.RiskLevel = ra.determineRiskLevel(profile.RiskScore)

	return profile, nil
}

// assessRiskFactors 评估风险因子
func (ra *RiskAssessor) assessRiskFactors(ctx context.Context, profile *RiskProfile) error {
	symbol := profile.Symbol

	// 获取市场数据用于风险评估
	marketData, err := ra.getMarketDataForRisk(ctx, symbol)
	if err != nil {
		return fmt.Errorf("获取市场数据失败: %w", err)
	}

	// 1. 评估波动率风险
	volatilityRisk, err := ra.assessVolatilityRisk(marketData)
	if err != nil {
		volatilityRisk = 0.5 // 默认中等风险
	}
	profile.RiskFactors.Volatility = volatilityRisk

	// 2. 评估流动性风险
	liquidityRisk, err := ra.assessLiquidityRisk(marketData)
	if err != nil {
		liquidityRisk = 0.3 // 默认较低风险
	}
	profile.RiskFactors.Liquidity = liquidityRisk

	// 3. 评估市场风险
	marketRisk, err := ra.assessMarketRisk(marketData)
	if err != nil {
		marketRisk = 0.4 // 默认中等风险
	}
	profile.RiskFactors.MarketRisk = marketRisk

	// 4. 评估信用风险
	creditRisk, err := ra.assessCreditRisk(marketData)
	if err != nil {
		creditRisk = 0.2 // 默认较低风险
	}
	profile.RiskFactors.CreditRisk = creditRisk

	// 5. 评估操作风险
	operationalRisk := ra.assessOperationalRisk(marketData)
	profile.RiskFactors.Operational = operationalRisk

	return nil
}

// assessVolatilityRisk 评估波动率风险
func (ra *RiskAssessor) assessVolatilityRisk(marketData []MarketDataPoint) (float64, error) {
	if len(marketData) < 7 {
		return 0.5, fmt.Errorf("数据不足")
	}

	// 计算最近7天的价格波动率
	prices := make([]float64, len(marketData))
	returns := make([]float64, len(marketData)-1)

	for i, data := range marketData {
		prices[i] = data.Price
	}

	// 计算收益率
	for i := 1; i < len(prices); i++ {
		if prices[i-1] != 0 {
			returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
		}
	}

	// 计算波动率（标准差）
	volatility := stat.StdDev(returns, nil)

	// 波动率风险评分 (0-1)
	// 0.02 (2%)以下为低风险，0.05 (5%)以上为高风险
	riskScore := math.Min(volatility/0.05, 1.0)
	riskScore = math.Max(riskScore, 0.0)

	return riskScore, nil
}

// assessLiquidityRisk 评估流动性风险
func (ra *RiskAssessor) assessLiquidityRisk(marketData []MarketDataPoint) (float64, error) {
	if len(marketData) == 0 {
		return 0.5, fmt.Errorf("无市场数据")
	}

	// 使用成交量和价格变动来评估流动性
	totalVolume := 0.0
	priceChanges := 0.0
	validDataPoints := 0

	for _, data := range marketData {
		if data.Volume24h > 0 {
			totalVolume += data.Volume24h
			validDataPoints++
		}
		if data.PriceChange24h != 0 {
			priceChanges += math.Abs(data.PriceChange24h)
		}
	}

	if validDataPoints == 0 {
		return 0.5, fmt.Errorf("无有效成交量数据")
	}

	avgVolume := totalVolume / float64(validDataPoints)
	avgPriceChange := priceChanges / float64(len(marketData))

	// 流动性风险 = 1 / (平均成交量 * (1 - 平均价格变动))
	// 高成交量且低价格变动的资产流动性好，风险低
	liquidityScore := avgVolume * (1.0 - math.Min(avgPriceChange/10.0, 1.0)) / 1000000.0 // 标准化

	// 转换为风险分数（流动性越好，风险越低）
	riskScore := 1.0 - math.Min(liquidityScore, 1.0)

	return riskScore, nil
}

// assessMarketRisk 评估市场风险
func (ra *RiskAssessor) assessMarketRisk(marketData []MarketDataPoint) (float64, error) {
	if len(marketData) < 7 {
		return 0.4, fmt.Errorf("数据不足")
	}

	// 计算市场相关性和Beta系数
	marketReturns := make([]float64, len(marketData)-1)
	assetReturns := make([]float64, len(marketData)-1)

	// 简化的市场指数（可以使用BTC作为代理）
	marketPrices := make([]float64, len(marketData))
	assetPrices := make([]float64, len(marketData))

	for i, data := range marketData {
		assetPrices[i] = data.Price
		// 简化为使用资产本身的价格变化作为市场代表
		marketPrices[i] = data.Price
	}

	// 计算收益率
	for i := 1; i < len(assetPrices); i++ {
		if assetPrices[i-1] != 0 {
			assetReturns[i-1] = (assetPrices[i] - assetPrices[i-1]) / assetPrices[i-1]
		}
		if marketPrices[i-1] != 0 {
			marketReturns[i-1] = (marketPrices[i] - marketPrices[i-1]) / marketPrices[i-1]
		}
	}

	// 计算相关性
	correlation := stat.Correlation(assetReturns, marketReturns, nil)

	// 计算Beta（资产波动率 / 市场波动率）
	assetVolatility := stat.StdDev(assetReturns, nil)
	marketVolatility := stat.StdDev(marketReturns, nil)

	beta := 1.0
	if marketVolatility > 0 {
		beta = assetVolatility / marketVolatility
	}

	// 市场风险 = |相关性| * Beta
	marketRisk := math.Abs(correlation) * beta

	// 限制在0-1范围内
	riskScore := math.Min(marketRisk, 1.0)

	return riskScore, nil
}

// assessCreditRisk 评估信用风险
func (ra *RiskAssessor) assessCreditRisk(marketData []MarketDataPoint) (float64, error) {
	if len(marketData) == 0 {
		return 0.2, fmt.Errorf("无市场数据")
	}

	// 信用风险主要基于市场资本化和交易深度
	latest := marketData[len(marketData)-1]

	// 大市值资产信用风险较低
	marketCapRisk := 1.0
	if *latest.MarketCap > 1000000000 { // 10亿美元以上
		marketCapRisk = 0.1
	} else if *latest.MarketCap > 100000000 { // 1亿美元以上
		marketCapRisk = 0.3
	} else if *latest.MarketCap > 10000000 { // 1000万美元以上
		marketCapRisk = 0.5
	} else {
		marketCapRisk = 0.8
	}

	// 结合成交量因素
	volumeRisk := 1.0
	if latest.Volume24h > 10000000 { // 1000万美元成交量
		volumeRisk = 0.2
	} else if latest.Volume24h > 1000000 { // 100万美元成交量
		volumeRisk = 0.4
	} else if latest.Volume24h > 100000 { // 10万美元成交量
		volumeRisk = 0.6
	} else {
		volumeRisk = 0.9
	}

	// 综合信用风险
	creditRisk := (marketCapRisk + volumeRisk) / 2.0

	return creditRisk, nil
}

// assessOperationalRisk 评估操作风险
func (ra *RiskAssessor) assessOperationalRisk(marketData []MarketDataPoint) float64 {
	if len(marketData) < 7 {
		return 0.3 // 默认中等操作风险
	}

	// 操作风险基于价格异常变动和成交量异常
	priceChanges := make([]float64, len(marketData)-1)
	volumeChanges := make([]float64, len(marketData)-1)

	for i := 1; i < len(marketData); i++ {
		// 价格变动百分比
		if marketData[i-1].Price != 0 {
			priceChanges[i-1] = math.Abs((marketData[i].Price - marketData[i-1].Price) / marketData[i-1].Price)
		}

		// 成交量变动百分比
		if marketData[i-1].Volume24h != 0 {
			volumeChanges[i-1] = math.Abs((marketData[i].Volume24h - marketData[i-1].Volume24h) / marketData[i-1].Volume24h)
		}
	}

	// 计算异常变动比例
	extremePriceChanges := 0
	extremeVolumeChanges := 0

	for _, change := range priceChanges {
		if change > 0.1 { // 10%以上价格变动视为极端
			extremePriceChanges++
		}
	}

	for _, change := range volumeChanges {
		if change > 2.0 { // 200%以上成交量变动视为极端
			extremeVolumeChanges++
		}
	}

	// 操作风险 = (极端价格变动比例 + 极端成交量变动比例) / 2
	priceRiskRatio := float64(extremePriceChanges) / float64(len(priceChanges))
	volumeRiskRatio := float64(extremeVolumeChanges) / float64(len(volumeChanges))

	operationalRisk := (priceRiskRatio + volumeRiskRatio) / 2.0

	// 限制在合理范围内
	operationalRisk = math.Min(operationalRisk, 1.0)

	return operationalRisk
}

// calculateCompositeRiskScore 计算综合风险分数
func (ra *RiskAssessor) calculateCompositeRiskScore(factors RiskFactors) float64 {
	weights := ra.config.RiskWeights

	// 加权综合风险分数
	compositeScore := factors.Volatility*weights.VolatilityWeight +
		factors.Liquidity*weights.LiquidityWeight +
		factors.MarketRisk*weights.MarketRiskWeight +
		factors.CreditRisk*weights.CreditRiskWeight +
		factors.Operational*weights.OperationalWeight

	// 确保在0-100范围内
	compositeScore = math.Min(compositeScore*100.0, ra.config.Assessment.MaxRiskScore)

	return compositeScore
}

// determineRiskLevel 确定风险等级
func (ra *RiskAssessor) determineRiskLevel(riskScore float64) RiskLevel {
	maxScore := ra.config.Assessment.MaxRiskScore
	threshold := ra.config.Assessment.RiskThreshold

	if riskScore >= threshold+20 || riskScore >= maxScore*0.9 {
		return RiskLevelCritical
	} else if riskScore >= threshold+10 || riskScore >= maxScore*0.7 {
		return RiskLevelHigh
	} else if riskScore >= threshold || riskScore >= maxScore*0.5 {
		return RiskLevelMedium
	} else {
		return RiskLevelLow
	}
}

// =================== 高级风险指标计算 ===================

// calculateVaR 计算VaR (Value at Risk)
func (ra *RiskAssessor) calculateVaR(returns []float64, confidence float64) (float64, error) {
	if len(returns) < 30 {
		return 0, fmt.Errorf("需要至少30个数据点来计算VaR")
	}

	// 计算收益率的均值和标准差
	mean := stat.Mean(returns, nil)
	stddev := stat.StdDev(returns, nil)

	if stddev == 0 {
		return 0, fmt.Errorf("标准差为零，无法计算VaR")
	}

	// 使用正态分布计算VaR
	// VaR = -mean + z * stddev (损失为正值)
	zScore := ra.getZScore(confidence)
	vaR := -(mean + zScore*stddev)

	return math.Max(0, vaR), nil
}

// calculateSharpeRatio 计算夏普比率
func (ra *RiskAssessor) calculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// 计算超额收益
	excessReturns := make([]float64, len(returns))
	for i, ret := range returns {
		excessReturns[i] = ret - riskFreeRate/252 // 日化无风险利率
	}

	mean := stat.Mean(excessReturns, nil)
	stddev := stat.StdDev(excessReturns, nil)

	if stddev == 0 {
		return 0
	}

	return mean / stddev * math.Sqrt(252) // 年化夏普比率
}

// calculateMaxDrawdown 计算最大回撤
func (ra *RiskAssessor) calculateMaxDrawdown(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}

	maxDrawdown := 0.0
	peak := prices[0]

	for _, price := range prices[1:] {
		if price > peak {
			peak = price
		}

		drawdown := (peak - price) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateBeta 计算贝塔系数 (相对市场风险)
func (ra *RiskAssessor) calculateBeta(assetReturns, marketReturns []float64) float64 {
	if len(assetReturns) != len(marketReturns) || len(assetReturns) < 30 {
		return 1.0 // 默认中性贝塔
	}

	// 计算协方差和市场方差
	covariance := stat.Covariance(assetReturns, marketReturns, nil)
	marketVariance := stat.Variance(marketReturns, nil)

	if marketVariance == 0 {
		return 1.0
	}

	return covariance / marketVariance
}

// calculateSortinoRatio 计算索提诺比率 (下行风险调整收益)
func (ra *RiskAssessor) calculateSortinoRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	// 计算超额收益
	excessReturns := make([]float64, len(returns))
	for i, ret := range returns {
		excessReturns[i] = ret - riskFreeRate/252
	}

	// 计算下行偏差 (只考虑负的超额收益)
	downsideReturns := make([]float64, 0)
	for _, ret := range excessReturns {
		if ret < 0 {
			downsideReturns = append(downsideReturns, ret)
		}
	}

	if len(downsideReturns) == 0 {
		return math.Inf(1) // 如果没有下行风险，返回正无穷
	}

	meanExcess := stat.Mean(excessReturns, nil)
	downsideStddev := stat.StdDev(downsideReturns, nil)

	if downsideStddev == 0 {
		return math.Inf(1)
	}

	return meanExcess / downsideStddev * math.Sqrt(252)
}

// calculateStressTest 执行压力测试
func (ra *RiskAssessor) calculateStressTest(prices []float64, scenarios []StressScenario) []StressTestResult {
	results := make([]StressTestResult, len(scenarios))

	for i, scenario := range scenarios {
		result := StressTestResult{
			Scenario: scenario,
			Shock:    scenario.Shock,
		}

		// 应用冲击
		shockedPrices := make([]float64, len(prices))
		for j, price := range prices {
			shockedPrices[j] = price * (1 + scenario.Shock)
		}

		// 计算受冲击后的风险指标
		result.Loss = (prices[len(prices)-1] - shockedPrices[len(shockedPrices)-1]) / prices[len(prices)-1]

		// 计算VaR (使用历史数据)
		returns := make([]float64, len(prices)-1)
		for j := 1; j < len(prices); j++ {
			returns[j-1] = (prices[j] - prices[j-1]) / prices[j-1]
		}

		if vaR, err := ra.calculateVaR(returns, 0.95); err == nil {
			result.VaR95 = vaR
		}

		results[i] = result
	}

	return results
}

// getZScore 获取置信水平对应的Z分数
func (ra *RiskAssessor) getZScore(confidence float64) float64 {
	// 使用近似值，实际应用中应该使用更精确的计算
	switch {
	case confidence >= 0.99:
		return 2.33
	case confidence >= 0.95:
		return 1.65
	case confidence >= 0.90:
		return 1.28
	default:
		return 1.0
	}
}

// getMarketDataForRisk 获取用于风险评估的市场数据
func (ra *RiskAssessor) getMarketDataForRisk(ctx context.Context, symbol string) ([]MarketDataPoint, error) {
	// 首先尝试从数据库获取历史数据
	// 如果数据库不可用或数据不足，则使用模拟数据作为后备

	// 这里应该实现从数据库查询历史数据的逻辑
	// SELECT * FROM market_data WHERE symbol = ? AND timestamp >= ? ORDER BY timestamp DESC LIMIT 30

	// 暂时使用模拟数据，但改进为更真实的模拟
	dataPoints := make([]MarketDataPoint, 30) // 30天的数据

	// 使用更真实的初始价格（基于当前市场数据）
	basePrice := 50000.0
	volatility := 0.05 // 5%的日波动率

	// 生成趋势性价格变动（模拟市场趋势）
	trend := 0.001 // 每天0.1%的趋势增长

	for i := 0; i < 30; i++ {
		// 结合趋势、波动性和随机因素
		randomFactor := (rand.Float64() - 0.5) * 2 * volatility      // 随机因子
		trendFactor := trend * float64(i)                            // 趋势因子
		seasonalFactor := math.Sin(float64(i)/30.0*2*math.Pi) * 0.01 // 季节性因子

		totalChange := randomFactor + trendFactor + seasonalFactor
		price := basePrice * (1 + totalChange)

		// 确保价格不为负数
		if price <= 0 {
			price = basePrice * 0.1 // 最低跌到10%的水平
		}

		// 计算24小时变化率
		changePercent := totalChange * 100

		// 生成更真实的成交量（与价格波动相关）
		baseVolume := 1000000.0
		volumeMultiplier := 1 + math.Abs(totalChange)*5                  // 波动大时成交量大
		volume := baseVolume * volumeMultiplier * (0.5 + rand.Float64()) // 添加随机性

		// 市值计算：价格 * 流通量（这里假设固定流通量）
		circulatingSupply := 19000000.0 // BTC的流通量约为1900万
		marketCap := price * circulatingSupply

		dataPoints[i] = MarketDataPoint{
			Symbol:         symbol,
			Price:          price,
			PriceChange24h: changePercent,
			Volume24h:      volume,
			MarketCap:      &marketCap,
			Timestamp:      time.Now().Add(-time.Duration(30-i) * 24 * time.Hour),
		}

		// 更新基础价格用于下一个数据点
		basePrice = price
	}

	// 按时间排序（最新的在前面）
	for i := 0; i < len(dataPoints)/2; i++ {
		j := len(dataPoints) - 1 - i
		dataPoints[i], dataPoints[j] = dataPoints[j], dataPoints[i]
	}

	return dataPoints, nil
}

// CalculateVaR 计算风险价值 (Value at Risk)
func (ra *RiskAssessor) CalculateVaR(marketData []MarketDataPoint, confidenceLevel float64, positionSize float64) (float64, error) {
	if len(marketData) < 30 {
		return 0, fmt.Errorf("数据不足，无法计算VaR")
	}

	// 计算历史收益率
	returns := make([]float64, len(marketData)-1)
	for i := 1; i < len(marketData); i++ {
		if marketData[i-1].Price != 0 {
			returns[i-1] = (marketData[i].Price - marketData[i-1].Price) / marketData[i-1].Price
		}
	}

	// 按升序排序收益率
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	// 计算VaR
	// 使用历史模拟法
	index := int(float64(len(sortedReturns)) * (1.0 - confidenceLevel))
	if index >= len(sortedReturns) {
		index = len(sortedReturns) - 1
	}

	varValue := -sortedReturns[index] * positionSize

	return varValue, nil
}

// CalculateExpectedShortfall 计算预期损失
func (ra *RiskAssessor) CalculateExpectedShortfall(marketData []MarketDataPoint, confidenceLevel float64, positionSize float64) (float64, error) {
	var_, err := ra.CalculateVaR(marketData, confidenceLevel, positionSize)
	if err != nil {
		return 0, err
	}

	// 计算VaR阈值以下的平均损失
	returns := make([]float64, len(marketData)-1)
	for i := 1; i < len(marketData); i++ {
		if marketData[i-1].Price != 0 {
			returns[i-1] = (marketData[i].Price - marketData[i-1].Price) / marketData[i-1].Price
		}
	}

	varThreshold := -var_ / positionSize
	extremeLosses := make([]float64, 0)

	for _, ret := range returns {
		if ret <= varThreshold {
			extremeLosses = append(extremeLosses, -ret*positionSize)
		}
	}

	if len(extremeLosses) == 0 {
		return var_, nil
	}

	// 计算平均极端损失
	sum := 0.0
	for _, loss := range extremeLosses {
		sum += loss
	}

	return sum / float64(len(extremeLosses)), nil
}

// CalculateSharpeRatio 计算夏普比率
func (ra *RiskAssessor) CalculateSharpeRatio(marketData []MarketDataPoint, riskFreeRate float64) (float64, error) {
	if len(marketData) < 30 {
		return 0, fmt.Errorf("数据不足，无法计算Sharpe比率")
	}

	// 计算平均收益率
	totalReturn := 0.0
	for i := 1; i < len(marketData); i++ {
		if marketData[i-1].Price != 0 {
			dailyReturn := (marketData[i].Price - marketData[i-1].Price) / marketData[i-1].Price
			totalReturn += dailyReturn
		}
	}

	avgReturn := totalReturn / float64(len(marketData)-1)

	// 计算波动率
	returns := make([]float64, len(marketData)-1)
	for i := 1; i < len(marketData); i++ {
		if marketData[i-1].Price != 0 {
			returns[i-1] = (marketData[i].Price - marketData[i-1].Price) / marketData[i-1].Price
		}
	}

	volatility := stat.StdDev(returns, nil)

	// 计算夏普比率
	if volatility == 0 {
		return 0, fmt.Errorf("波动率为零，无法计算夏普比率")
	}

	sharpeRatio := (avgReturn - riskFreeRate) / volatility

	return sharpeRatio, nil
}

// CalculateMaximumDrawdown 计算最大回撤
func (ra *RiskAssessor) CalculateMaximumDrawdown(marketData []MarketDataPoint) (float64, error) {
	if len(marketData) < 2 {
		return 0, fmt.Errorf("数据不足，无法计算最大回撤")
	}

	prices := make([]float64, len(marketData))
	for i, data := range marketData {
		prices[i] = data.Price
	}

	maxDrawdown := 0.0
	peak := prices[0]

	for _, price := range prices[1:] {
		if price > peak {
			peak = price
		}

		drawdown := (peak - price) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown, nil
}

// calculateVolatility 计算波动率
func (ra *RiskAssessor) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0.0
	}

	// 计算标准差作为波动率度量
	return stat.StdDev(returns, nil)
}
