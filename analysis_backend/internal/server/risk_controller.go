package server

import (
	"fmt"
	"math"
)

// MakeDecision 做出风险决策
func (rc *RiskController) MakeDecision(profile *RiskProfile, requestedPosition float64) *RiskDecision {
	decision := &RiskDecision{
		Symbol:          profile.Symbol,
		CanTrade:        true,
		MaxPosition:     requestedPosition,
		RiskScore:       profile.RiskScore,
		RiskLevel:       profile.RiskLevel,
		Recommendations: make([]string, 0),
		Warnings:        make([]string, 0),
	}

	// 应用风险控制规则
	rc.applyRiskRules(decision, profile, requestedPosition)

	// 生成建议和警告
	rc.generateRecommendations(decision, profile)

	return decision
}

// =================== 高级风险控制策略 ===================

// ApplyAdvancedRiskControls 应用高级风险控制
func (rc *RiskController) ApplyAdvancedRiskControls(decision *RiskDecision, profile *RiskProfile, portfolio *PortfolioRisk, marketState MarketState) {
	// 1. 组合风险控制
	rc.applyPortfolioRiskControls(decision, profile, portfolio)

	// 2. 市场状态适应性控制
	rc.applyMarketAdaptiveControls(decision, profile, marketState)

	// 3. 动态VaR控制
	rc.applyDynamicVaRControls(decision, profile)

	// 4. 压力测试验证
	rc.applyStressTestValidation(decision, profile)
}

// applyPortfolioRiskControls 组合风险控制
func (rc *RiskController) applyPortfolioRiskControls(decision *RiskDecision, profile *RiskProfile, portfolio *PortfolioRisk) {
	if portfolio == nil {
		return
	}

	// 检查组合多样化
	diversificationScore := portfolio.Diversification
	if diversificationScore < 0.6 {
		decision.MaxPosition *= 0.8 // 降低仓位20%
		decision.Warnings = append(decision.Warnings, "组合多样化不足，降低仓位")
	}

	// 检查相关性风险
	symbol := profile.Symbol
	if correlations, exists := portfolio.Correlation[symbol]; exists {
		highCorrCount := 0
		for _, corr := range correlations {
			if math.Abs(corr) > 0.7 {
				highCorrCount++
			}
		}

		if highCorrCount > 2 {
			decision.MaxPosition *= 0.7 // 降低仓位30%
			decision.Warnings = append(decision.Warnings, "与组合中多个资产高度相关，降低仓位")
		}
	}

	// 检查风险贡献
	if contribution, exists := portfolio.RiskContribution[symbol]; exists && contribution > 0.15 {
		decision.MaxPosition *= 0.85 // 降低仓位15%
		decision.Recommendations = append(decision.Recommendations, "该资产风险贡献过高，建议降低仓位")
	}
}

// applyMarketAdaptiveControls 市场状态适应性控制
func (rc *RiskController) applyMarketAdaptiveControls(decision *RiskDecision, profile *RiskProfile, marketState MarketState) {
	// 根据市场状态调整风险控制
	switch marketState.State {
	case "bull":
		// 牛市：可以适当放宽风险控制
		decision.MaxPosition *= 1.2
		if decision.MaxPosition > rc.config.Control.MaxPositionSize {
			decision.MaxPosition = rc.config.Control.MaxPositionSize
		}
		decision.Recommendations = append(decision.Recommendations, "牛市环境下，可以适当增加仓位")

	case "bear":
		// 熊市：严格控制风险
		decision.MaxPosition *= 0.5
		decision.Warnings = append(decision.Warnings, "熊市环境下，严格控制仓位")

	case "sideways":
		// 震荡市：正常风险控制
		// 不做调整
	}

	// 根据波动率调整
	if marketState.Volatility > 0.05 { // 高波动
		volatilityMultiplier := 1.0 - (marketState.Volatility-0.05)*2
		decision.MaxPosition *= math.Max(volatilityMultiplier, 0.3)
		decision.Warnings = append(decision.Warnings, "市场波动率较高，降低仓位")
	}
}

// applyDynamicVaRControls 动态VaR控制
func (rc *RiskController) applyDynamicVaRControls(decision *RiskDecision, profile *RiskProfile) {
	// 这里可以实现动态VaR控制逻辑
	// 基于历史VaR和当前市场条件调整仓位

	// 示例：如果VaR超过阈值，降低仓位
	if profile.VaR95 > 0.1 { // 95% VaR超过10%
		vaRMultiplier := 0.1 / profile.VaR95
		decision.MaxPosition *= math.Max(vaRMultiplier, 0.4)
		decision.Warnings = append(decision.Warnings, "VaR风险较高，自动调整仓位")
	}
}

// applyStressTestValidation 压力测试验证
func (rc *RiskController) applyStressTestValidation(decision *RiskDecision, profile *RiskProfile) {
	if len(profile.StressTestResults) == 0 {
		return
	}

	// 检查压力测试结果
	maxLoss := 0.0
	for _, result := range profile.StressTestResults {
		if math.Abs(result.Loss) > maxLoss {
			maxLoss = math.Abs(result.Loss)
		}
	}

	// 如果最大损失超过15%，降低仓位
	if maxLoss > 0.15 {
		stressMultiplier := 0.15 / maxLoss
		decision.MaxPosition *= math.Max(stressMultiplier, 0.5)
		decision.Warnings = append(decision.Warnings, "压力测试显示潜在损失较大，降低仓位")
	}
}

// CalculatePortfolioRisk 计算投资组合风险
func (rc *RiskController) CalculatePortfolioRisk(positions map[string]float64, returns map[string][]float64) (*PortfolioRisk, error) {
	if len(positions) == 0 {
		return &PortfolioRisk{}, fmt.Errorf("空的投资组合")
	}

	portfolio := &PortfolioRisk{
		AssetWeights:     positions,
		RiskContribution: make(map[string]float64),
		Correlation:      make(map[string]map[string]float64),
	}

	// 计算总价值
	totalValue := 0.0
	for _, weight := range positions {
		// 这里应该使用实际的价格数据计算价值
		// 暂时使用权重作为近似值
		totalValue += weight * 1000 // 假设基准价值为1000
	}
	portfolio.TotalValue = totalValue

	// 计算相关性矩阵
	symbols := make([]string, 0, len(positions))
	for symbol := range positions {
		symbols = append(symbols, symbol)
	}

	for _, symbol1 := range symbols {
		portfolio.Correlation[symbol1] = make(map[string]float64)
		for _, symbol2 := range symbols {
			if returns1, ok1 := returns[symbol1]; ok1 {
				if returns2, ok2 := returns[symbol2]; ok2 {
					corr := rc.calculateCorrelation(returns1, returns2)
					portfolio.Correlation[symbol1][symbol2] = corr
				}
			}
		}
	}

	// 计算多样化得分
	portfolio.Diversification = rc.calculateDiversificationScore(portfolio.Correlation)

	// 计算风险贡献
	portfolio = rc.calculateRiskContributions(portfolio, returns)

	// 计算总风险
	totalRisk := 0.0
	for _, contribution := range portfolio.RiskContribution {
		totalRisk += contribution * contribution // 方差形式
	}
	portfolio.TotalRisk = math.Sqrt(totalRisk)

	return portfolio, nil
}

// calculateCorrelation 计算相关系数
func (rc *RiskController) calculateCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	n := float64(len(x))
	sumX, sumY := 0.0, 0.0
	sumXY, sumX2, sumY2 := 0.0, 0.0, 0.0

	for i := 0; i < len(x); i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
		sumY2 += y[i] * y[i]
	}

	numerator := n*sumXY - sumX*sumY
	denominator := math.Sqrt((n*sumX2 - sumX*sumX) * (n*sumY2 - sumY*sumY))

	if denominator == 0 {
		return 0
	}

	return numerator / denominator
}

// calculateDiversificationScore 计算多样化得分
func (rc *RiskController) calculateDiversificationScore(correlation map[string]map[string]float64) float64 {
	if len(correlation) <= 1 {
		return 0.0 // 单个资产没有多样化
	}

	totalCorr := 0.0
	count := 0

	for symbol1, corrs := range correlation {
		for symbol2, corr := range corrs {
			if symbol1 != symbol2 {
				totalCorr += math.Abs(corr)
				count++
			}
		}
	}

	if count == 0 {
		return 1.0
	}

	avgCorr := totalCorr / float64(count)
	// 多样化得分 = 1 - 平均相关系数
	return math.Max(0, 1-avgCorr)
}

// calculateRiskContributions 计算风险贡献
func (rc *RiskController) calculateRiskContributions(portfolio *PortfolioRisk, returns map[string][]float64) *PortfolioRisk {
	// 简化的风险贡献计算
	// 实际实现应该使用更复杂的数学模型

	for symbol, weight := range portfolio.AssetWeights {
		if ret, exists := returns[symbol]; exists && len(ret) > 0 {
			// 使用波动率作为风险度量
			volatility := rc.calculateVolatility(ret)
			portfolio.RiskContribution[symbol] = weight * volatility
		}
	}

	return portfolio
}

// calculateVolatility 计算波动率
func (rc *RiskController) calculateVolatility(returns []float64) float64 {
	if len(returns) < 2 {
		return 0
	}

	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		diff := ret - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)

	return math.Sqrt(variance)
}

// OptimizePortfolio 投资组合优化
func (rc *RiskController) OptimizePortfolio(targetReturn float64, returns map[string][]float64, constraints map[string]float64) (map[string]float64, error) {
	// 简化的投资组合优化实现
	// 使用现代投资组合理论 (MPT)

	symbols := make([]string, 0, len(returns))
	for symbol := range returns {
		symbols = append(symbols, symbol)
	}

	// 计算预期收益和协方差矩阵
	expectedReturns := make(map[string]float64)
	covarianceMatrix := make(map[string]map[string]float64)

	for _, symbol := range symbols {
		if ret, exists := returns[symbol]; exists {
			// 计算预期收益（使用历史平均）
			sum := 0.0
			for _, r := range ret {
				sum += r
			}
			expectedReturns[symbol] = sum / float64(len(ret))

			// 初始化协方差矩阵行
			covarianceMatrix[symbol] = make(map[string]float64)
		}
	}

	// 计算协方差矩阵
	for _, symbol1 := range symbols {
		for _, symbol2 := range symbols {
			if ret1, ok1 := returns[symbol1]; ok1 {
				if ret2, ok2 := returns[symbol2]; ok2 {
					cov := rc.calculateCovariance(ret1, ret2)
					covarianceMatrix[symbol1][symbol2] = cov
				}
			}
		}
	}

	// 使用均值-方差优化找到最优权重
	// 这里使用简化的等权重分配作为示例
	optimalWeights := make(map[string]float64)
	weight := 1.0 / float64(len(symbols))

	for _, symbol := range symbols {
		optimalWeights[symbol] = weight
	}

	return optimalWeights, nil
}

// calculateCovariance 计算协方差
func (rc *RiskController) calculateCovariance(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	n := float64(len(x))
	meanX, meanY := 0.0, 0.0

	for i := 0; i < len(x); i++ {
		meanX += x[i]
		meanY += y[i]
	}
	meanX /= n
	meanY /= n

	cov := 0.0
	for i := 0; i < len(x); i++ {
		cov += (x[i] - meanX) * (y[i] - meanY)
	}
	cov /= (n - 1)

	return cov
}

// applyRiskRules 应用风险控制规则
func (rc *RiskController) applyRiskRules(decision *RiskDecision, profile *RiskProfile, requestedPosition float64) {
	riskLevel := profile.RiskLevel
	riskScore := profile.RiskScore

	// 1. 基于风险等级的仓位限制
	switch riskLevel {
	case RiskLevelCritical:
		decision.CanTrade = false
		decision.MaxPosition = 0
		decision.Warnings = append(decision.Warnings, "风险等级为关键，不允许交易")
	case RiskLevelHigh:
		decision.MaxPosition = math.Min(requestedPosition, rc.config.Control.MaxPositionSize*0.3) // 30%限制
		if requestedPosition > decision.MaxPosition {
			decision.Warnings = append(decision.Warnings, fmt.Sprintf("高风险资产，仓位限制为%.2f", decision.MaxPosition))
		}
	case RiskLevelMedium:
		decision.MaxPosition = math.Min(requestedPosition, rc.config.Control.MaxPositionSize*0.6) // 60%限制
		if requestedPosition > decision.MaxPosition {
			decision.Warnings = append(decision.Warnings, fmt.Sprintf("中等风险资产，仓位限制为%.2f", decision.MaxPosition))
		}
	case RiskLevelLow:
		decision.MaxPosition = math.Min(requestedPosition, rc.config.Control.MaxPositionSize) // 100%限制
	}

	// 2. 基于风险分数的额外限制
	if riskScore > rc.config.Assessment.RiskThreshold+10 {
		reductionFactor := 1.0 - (riskScore-rc.config.Assessment.RiskThreshold)/rc.config.Assessment.MaxRiskScore
		decision.MaxPosition *= math.Max(reductionFactor, 0.1) // 最少10%仓位
		decision.Warnings = append(decision.Warnings, fmt.Sprintf("风险分数过高，仓位减少%.0f%%", (1.0-reductionFactor)*100))
	}

	// 3. 检查历史回撤
	if rc.checkHistoricalDrawdown(profile) {
		decision.CanTrade = false
		decision.MaxPosition = 0
		decision.Warnings = append(decision.Warnings, "历史回撤超过限制，暂停交易")
	}

	// 4. 波动率检查
	if profile.RiskFactors.Volatility > 0.7 {
		decision.MaxPosition *= 0.5
		decision.Warnings = append(decision.Warnings, "高波动率，仓位减半")
	}

	// 5. 流动性检查
	if profile.RiskFactors.Liquidity > 0.8 {
		decision.CanTrade = false
		decision.MaxPosition = 0
		decision.Warnings = append(decision.Warnings, "流动性风险过高，不允许交易")
	}

	// 6. 分散度检查
	if !rc.checkDiversification(profile) {
		decision.MaxPosition *= 0.7
		decision.Warnings = append(decision.Warnings, "分散度不足，仓位减少30%")
	}
}

// generateRecommendations 生成建议和警告
func (rc *RiskController) generateRecommendations(decision *RiskDecision, profile *RiskProfile) {
	// 基于风险因子的建议
	if profile.RiskFactors.Volatility > 0.6 {
		decision.Recommendations = append(decision.Recommendations, "考虑降低仓位以控制波动风险")
	}

	if profile.RiskFactors.Liquidity > 0.5 {
		decision.Recommendations = append(decision.Recommendations, "关注流动性风险，准备应急退出策略")
	}

	if profile.RiskFactors.MarketRisk > 0.7 {
		decision.Recommendations = append(decision.Recommendations, "市场风险较高，考虑对冲策略")
	}

	if profile.RiskFactors.CreditRisk > 0.6 {
		decision.Recommendations = append(decision.Recommendations, "信用风险较高，谨慎投资")
	}

	// 基于风险等级的建议
	switch profile.RiskLevel {
	case RiskLevelHigh:
		decision.Recommendations = append(decision.Recommendations,
			"定期监控仓位",
			"设置止损点",
			"考虑部分平仓")
	case RiskLevelMedium:
		decision.Recommendations = append(decision.Recommendations,
			"适度监控",
			"关注市场变化")
	case RiskLevelLow:
		decision.Recommendations = append(decision.Recommendations,
			"可以正常交易",
			"关注长期表现")
	}

	// 基于仓位限制的建议
	if decision.MaxPosition < 1.0 { // 如果最大仓位小于100%，给出建议
		decision.Recommendations = append(decision.Recommendations,
			fmt.Sprintf("建议最大仓位控制在%.4f", decision.MaxPosition))
	}
}

// checkHistoricalDrawdown 检查历史回撤
func (rc *RiskController) checkHistoricalDrawdown(profile *RiskProfile) bool {
	if len(profile.HistoricalRisk) < 10 {
		return false // 数据不足，不做判断
	}

	// 计算最近的回撤
	maxValue := 0.0
	currentDrawdown := 0.0

	for _, history := range profile.HistoricalRisk {
		pnl := history.PnL
		if pnl > maxValue {
			maxValue = pnl
		}

		if maxValue > 0 {
			currentDrawdown = (maxValue - pnl) / maxValue
			if currentDrawdown > rc.config.Control.MaxDrawdownLimit {
				return true // 超过回撤限制
			}
		}
	}

	return false
}

// checkDiversification 检查分散度
func (rc *RiskController) checkDiversification(profile *RiskProfile) bool {
	// 这里应该检查投资组合的分散度
	// 暂时使用简化的检查：基于风险等级的多样性

	// 如果只有一个资产且风险较高，则分散度不足
	if profile.RiskLevel == RiskLevelHigh || profile.RiskLevel == RiskLevelCritical {
		return false
	}

	return true
}

// ApplyStopLoss 应用止损策略
func (rc *RiskController) ApplyStopLoss(currentPrice, entryPrice, positionSize float64) (bool, float64) {
	if len(rc.config.Control.StopLossLevels) == 0 {
		return false, 0
	}

	// 计算亏损百分比
	lossPercentage := (entryPrice - currentPrice) / entryPrice

	// 检查是否触发止损
	for _, stopLevel := range rc.config.Control.StopLossLevels {
		if lossPercentage >= stopLevel {
			// 计算应该卖出的数量（逐步减仓）
			sellRatio := math.Min(lossPercentage/stopLevel, 1.0)
			sellAmount := positionSize * sellRatio

			return true, sellAmount
		}
	}

	return false, 0
}

// CalculatePositionSize 计算建议仓位大小
func (rc *RiskController) CalculatePositionSize(profile *RiskProfile, totalCapital, availableCapital float64) float64 {
	// 基于风险等级的仓位计算
	var basePosition float64

	switch profile.RiskLevel {
	case RiskLevelLow:
		basePosition = totalCapital * 0.1 // 10% 仓位
	case RiskLevelMedium:
		basePosition = totalCapital * 0.05 // 5% 仓位
	case RiskLevelHigh:
		basePosition = totalCapital * 0.02 // 2% 仓位
	case RiskLevelCritical:
		basePosition = 0 // 不允许交易
	}

	// 基于风险分数的调整
	riskAdjustment := 1.0 - (profile.RiskScore / rc.config.Assessment.MaxRiskScore)
	basePosition *= riskAdjustment

	// 基于波动率的调整
	volatilityAdjustment := 1.0 - profile.RiskFactors.Volatility
	basePosition *= volatilityAdjustment

	// 确保不超过可用资本和最大仓位限制
	maxPosition := math.Min(availableCapital, rc.config.Control.MaxPositionSize*totalCapital)
	basePosition = math.Min(basePosition, maxPosition)

	return math.Max(basePosition, 0)
}

// ValidateTrade 验证交易是否符合风险控制
func (rc *RiskController) ValidateTrade(profile *RiskProfile, tradeSize, currentPosition float64) (bool, string) {
	// 检查风险等级
	if profile.RiskLevel == RiskLevelCritical {
		return false, "关键风险等级，禁止交易"
	}

	// 检查仓位限制
	totalPosition := currentPosition + tradeSize
	if totalPosition > profile.PositionLimits.MaxPosition {
		return false, fmt.Sprintf("超过最大仓位限制 %.4f", profile.PositionLimits.MaxPosition)
	}

	// 检查风险分数
	if profile.RiskScore > rc.config.Assessment.RiskThreshold {
		return false, fmt.Sprintf("风险分数 %.2f 超过阈值 %.2f", profile.RiskScore, rc.config.Assessment.RiskThreshold)
	}

	// 检查流动性风险
	if profile.RiskFactors.Liquidity > 0.8 {
		return false, "流动性风险过高"
	}

	// 检查波动率风险
	if profile.RiskFactors.Volatility > 0.9 {
		return false, "波动率风险极高"
	}

	return true, ""
}

// GetRiskLimits 获取风险限制
func (rc *RiskController) GetRiskLimits(symbol string) (*PositionLimits, error) {
	// 这里应该从配置或数据库获取特定资产的风险限制
	// 暂时返回默认限制

	limits := &PositionLimits{
		MaxPosition:     rc.config.Control.MaxPositionSize,
		MaxDrawdown:     rc.config.Control.MaxDrawdownLimit,
		Diversification: rc.config.Control.DiversificationMin,
		StopLoss:        rc.config.Control.StopLossLevels,
	}

	return limits, nil
}

// UpdateRiskLimits 更新风险限制
func (rc *RiskController) UpdateRiskLimits(symbol string, limits PositionLimits) error {
	// 这里应该将更新后的限制保存到配置或数据库
	// 暂时只记录日志

	fmt.Printf("[RiskController] 更新风险限制 - %s: MaxPosition=%.4f, MaxDrawdown=%.4f\n",
		symbol, limits.MaxPosition, limits.MaxDrawdown)

	return nil
}

// getRiskAssessor 获取风险评估器（这里需要从RiskManagement中获取）
func (rc *RiskController) getRiskAssessor() (*RiskAssessor, error) {
	// 这里应该从RiskManagement实例中获取风险评估器
	// 暂时返回nil，表示没有可用的风险评估器
	return nil, fmt.Errorf("风险评估器未初始化")
}

// GenerateRiskReport 生成风险控制报告
func (rc *RiskController) GenerateRiskReport() map[string]interface{} {
	report := map[string]interface{}{
		"max_position_size":   rc.config.Control.MaxPositionSize,
		"max_drawdown_limit":  rc.config.Control.MaxDrawdownLimit,
		"diversification_min": rc.config.Control.DiversificationMin,
		"stop_loss_levels":    rc.config.Control.StopLossLevels,
		"risk_threshold":      rc.config.Assessment.RiskThreshold,
		"monitoring_enabled":  rc.config.Monitoring.EnableRealTime,
		"alert_thresholds":    rc.config.Monitoring.AlertThresholds,
		"last_updated":        "2025-11-28",
	}

	return report
}
