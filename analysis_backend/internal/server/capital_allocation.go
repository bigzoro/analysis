package server

import (
	"math"
	"sort"
	"time"
)

// ===== 动态资本分配系统 =====

// CapitalAllocator 资本分配器
type CapitalAllocator struct {
	totalCapital    float64
	riskTolerance   float64 // 风险承受度 (0.0-1.0)
	maxDrawdown     float64 // 最大允许回撤
	minAllocation   float64 // 最小单仓位分配
	maxAllocation   float64 // 最大单仓位分配

	// 风险度量
	currentDrawdown   float64
	volatilityBudget  float64
	correlationBudget float64

	// 分配历史
	allocationHistory []AllocationRecord
}

// AllocationRecord 分配记录
type AllocationRecord struct {
	Timestamp    time.Time
	Symbol       string
	Allocation   float64 // 分配金额
	Weight       float64 // 权重 (0.0-1.0)
	RiskScore    float64 // 风险评分
	ExpectedPnL  float64 // 预期损益
	ActualPnL    float64 // 实际损益
}

// AllocationDecision 分配决策
type AllocationDecision struct {
	Symbol         string
	Allocation     float64
	Weight         float64
	RiskAdjustment float64
	Reason         string
}

// NewCapitalAllocator 创建资本分配器
func NewCapitalAllocator(totalCapital float64) *CapitalAllocator {
	return &CapitalAllocator{
		totalCapital:     totalCapital,
		riskTolerance:    0.7, // 默认中等风险承受度
		maxDrawdown:      0.15, // 默认15%最大回撤
		minAllocation:    totalCapital * 0.01, // 最小1%资本
		maxAllocation:    totalCapital * 0.10, // 最大10%资本
		allocationHistory: make([]AllocationRecord, 0),
	}
}

// AllocateCapital 动态分配资本
func (ca *CapitalAllocator) AllocateCapital(
	symbolStates map[string]*SymbolState,
	opportunities []*SymbolOpportunity,
	result *BacktestResult,
) map[string]*AllocationDecision {

	decisions := make(map[string]*AllocationDecision)

	// 1. 计算可用资本
	availableCapital := ca.calculateAvailableCapital(result)

	// 2. 评估每个机会的风险和收益
	opportunityScores := ca.scoreOpportunities(opportunities, symbolStates)

	// 3. 应用资本约束
	constrainedAllocations := ca.applyCapitalConstraints(opportunityScores, availableCapital)

	// 4. 执行风险平价分配
	finalAllocations := ca.riskParityAllocation(constrainedAllocations, symbolStates)

	// 5. 记录分配决策
	for symbol, decision := range finalAllocations {
		decisions[symbol] = decision

		// 记录到历史
		record := AllocationRecord{
			Timestamp:    time.Now(),
			Symbol:       symbol,
			Allocation:   decision.Allocation,
			Weight:       decision.Weight,
			RiskScore:    decision.RiskAdjustment,
			ExpectedPnL:  0.0, // 暂时设为0
			ActualPnL:    0.0,
		}
		ca.allocationHistory = append(ca.allocationHistory, record)
	}

	return decisions
}

// calculateAvailableCapital 计算可用资本
func (ca *CapitalAllocator) calculateAvailableCapital(result *BacktestResult) float64 {
	if result == nil {
		return ca.totalCapital
	}

	// 计算当前回撤
	currentDrawdown := ca.calculateCurrentDrawdown(result)

	// 根据回撤调整可用资本
	drawdownAdjustment := 1.0 - math.Min(currentDrawdown/ca.maxDrawdown, 0.8)

	availableCapital := ca.totalCapital * drawdownAdjustment * ca.riskTolerance

	return math.Max(availableCapital, ca.minAllocation)
}

// calculateCurrentDrawdown 计算当前回撤
func (ca *CapitalAllocator) calculateCurrentDrawdown(result *BacktestResult) float64 {
	if result == nil || len(result.PortfolioValues) == 0 {
		return 0.0
	}

	currentValue := result.PortfolioValues[len(result.PortfolioValues)-1]
	peakValue := result.Config.InitialCash

	// 找到历史最高值
	for _, value := range result.PortfolioValues {
		if value > peakValue {
			peakValue = value
		}
	}

	if peakValue == 0 {
		return 0.0
	}

	return (peakValue - currentValue) / peakValue
}

// scoreOpportunities 评估机会的质量和风险
func (ca *CapitalAllocator) scoreOpportunities(
	opportunities []*SymbolOpportunity,
	symbolStates map[string]*SymbolState,
) map[string]*OpportunityScore {

	scores := make(map[string]*OpportunityScore)

	for _, opp := range opportunities {
		score := &OpportunityScore{
			Symbol:         opp.Symbol,
			Confidence:     opp.Confidence,
			ExpectedReturn: ca.estimateExpectedReturn(opp, symbolStates[opp.Symbol]),
			RiskScore:      ca.calculateRiskScore(opp, symbolStates[opp.Symbol]),
			LiquidityScore: ca.calculateLiquidityScore(opp, symbolStates[opp.Symbol]),
		}

		// 计算综合得分
		score.CompositeScore = ca.calculateCompositeScore(score)

		scores[opp.Symbol] = score
	}

	return scores
}

// OpportunityScore 机会评分
type OpportunityScore struct {
	Symbol         string
	Confidence     float64
	ExpectedReturn float64
	RiskScore      float64
	LiquidityScore float64
	CompositeScore float64
}

// estimateExpectedReturn 估算预期收益
func (ca *CapitalAllocator) estimateExpectedReturn(opp *SymbolOpportunity, state *SymbolState) float64 {
	if state == nil || len(state.Data) < 20 {
		return 0.0
	}

	// 基于历史表现估算预期收益
	recentTrades := 0
	recentPnL := 0.0

	// 统计最近的表现
	for range state.Data {
		// 简化计算，使用固定周期
		recentTrades++
		recentPnL += 0.001 // 假设平均收益

		if recentTrades >= 20 {
			break
		}
	}

	if recentTrades == 0 {
		return opp.Confidence * 0.02 // 默认2%的预期收益
	}

	avgReturn := recentPnL / float64(recentTrades)
	return math.Max(avgReturn, opp.Confidence*0.01) // 至少0.1%的预期收益
}

// calculateRiskScore 计算风险评分
func (ca *CapitalAllocator) calculateRiskScore(opp *SymbolOpportunity, state *SymbolState) float64 {
	if state == nil || len(state.Data) < 30 {
		return 0.5 // 中等风险
	}

	// 计算价格波动率
	prices := make([]float64, len(state.Data))
	for i, data := range state.Data {
		prices[i] = data.Price
	}

	volatility := ca.calculateHistoricalVolatility(prices)

	// 计算最大回撤
	maxDrawdown := ca.calculateSymbolMaxDrawdown(prices)

	// 综合风险评分 (0-1, 1表示最高风险)
	riskScore := (volatility*0.4 + maxDrawdown*0.6)
	return math.Min(riskScore, 1.0)
}

// calculateLiquidityScore 计算流动性评分
func (ca *CapitalAllocator) calculateLiquidityScore(opp *SymbolOpportunity, state *SymbolState) float64 {
	if state == nil || len(state.Data) == 0 {
		return 0.5
	}

	// 基于成交量和价格变动频率评估流动性
	// 简化实现
	return 0.8 // 假设中等流动性
}

// calculateCompositeScore 计算综合得分
func (ca *CapitalAllocator) calculateCompositeScore(score *OpportunityScore) float64 {
	// 收益权重0.4, 风险权重0.3, 信心权重0.2, 流动性权重0.1
	return score.ExpectedReturn*0.4/ca.riskTolerance -
		   score.RiskScore*0.3 +
		   score.Confidence*0.2 +
		   score.LiquidityScore*0.1
}

// applyCapitalConstraints 应用资本约束
func (ca *CapitalAllocator) applyCapitalConstraints(
	scores map[string]*OpportunityScore,
	availableCapital float64,
) map[string]float64 {

	constrained := make(map[string]float64)

	// 按综合得分排序
	type scoredSymbol struct {
		symbol string
		score  float64
	}

	var sortedSymbols []scoredSymbol
	for symbol, score := range scores {
		sortedSymbols = append(sortedSymbols, scoredSymbol{symbol, score.CompositeScore})
	}

	sort.Slice(sortedSymbols, func(i, j int) bool {
		return sortedSymbols[i].score > sortedSymbols[j].score
	})

	// 应用约束分配资本
	remainingCapital := availableCapital

	for _, item := range sortedSymbols {
		if remainingCapital <= 0 {
			break
		}

		score := scores[item.symbol]

		// 基于得分计算基础分配
		baseAllocation := availableCapital * math.Max(score.CompositeScore, 0.0) * 0.1

		// 应用最小和最大约束
		allocation := math.Max(ca.minAllocation, math.Min(baseAllocation, ca.maxAllocation))
		allocation = math.Min(allocation, remainingCapital)

		if allocation >= ca.minAllocation {
			constrained[item.symbol] = allocation
			remainingCapital -= allocation
		}
	}

	return constrained
}

// riskParityAllocation 风险平价分配
func (ca *CapitalAllocator) riskParityAllocation(
	allocations map[string]float64,
	symbolStates map[string]*SymbolState,
) map[string]*AllocationDecision {

	decisions := make(map[string]*AllocationDecision)

	// 计算每个符号的风险贡献
	riskContributions := make(map[string]float64)

	totalAllocation := 0.0
	for _, alloc := range allocations {
		totalAllocation += alloc
	}

	// 计算风险平价权重
	for symbol, alloc := range allocations {
		riskScore := ca.calculateRiskScore(nil, symbolStates[symbol]) // 简化计算
		riskContributions[symbol] = riskScore * alloc
	}

	// 调整分配以实现风险平价
	for symbol, alloc := range allocations {
		riskContribution := riskContributions[symbol]

		// 风险调整因子
		riskAdjustment := 1.0
		if riskContribution > ca.volatilityBudget {
			riskAdjustment = ca.volatilityBudget / riskContribution
		}

		finalAllocation := alloc * riskAdjustment
		weight := finalAllocation / ca.totalCapital

		decisions[symbol] = &AllocationDecision{
			Symbol:         symbol,
			Allocation:     finalAllocation,
			Weight:         weight,
			RiskAdjustment: riskAdjustment,
			Reason:         "risk_parity_allocation",
		}
	}

	return decisions
}

// calculateHistoricalVolatility 计算历史波动率
func (ca *CapitalAllocator) calculateHistoricalVolatility(prices []float64) float64 {
	if len(prices) < 30 {
		return 0.02 // 默认2%波动率
	}

	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}

	sum := 0.0
	for _, ret := range returns {
		sum += ret
	}
	mean := sum / float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		variance += (ret - mean) * (ret - mean)
	}
	variance /= float64(len(returns))

	return math.Sqrt(variance)
}

// calculateSymbolMaxDrawdown 计算单个符号的最大回撤
func (ca *CapitalAllocator) calculateSymbolMaxDrawdown(prices []float64) float64 {
	if len(prices) < 10 {
		return 0.0
	}

	maxDrawdown := 0.0
	peak := prices[0]

	for _, price := range prices {
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

// UpdateRiskParameters 动态更新风险参数
func (ca *CapitalAllocator) UpdateRiskParameters(currentDrawdown float64, marketVolatility float64) {
	// 根据市场状况动态调整风险参数
	ca.currentDrawdown = currentDrawdown

	// 调整风险承受度
	if currentDrawdown > 0.10 {
		ca.riskTolerance = math.Max(0.3, ca.riskTolerance*0.9) // 降低风险承受度
	} else if currentDrawdown < 0.05 {
		ca.riskTolerance = math.Min(0.8, ca.riskTolerance*1.05) // 提高风险承受度
	}

	// 调整波动率预算
	ca.volatilityBudget = ca.riskTolerance * (1.0 - currentDrawdown)
}

// GetAllocationHistory 获取分配历史
func (ca *CapitalAllocator) GetAllocationHistory() []AllocationRecord {
	return ca.allocationHistory
}

// CalculatePortfolioRisk 计算投资组合风险
func (ca *CapitalAllocator) CalculatePortfolioRisk(allocations map[string]*AllocationDecision) float64 {
	totalRisk := 0.0
	totalWeight := 0.0

	for _, decision := range allocations {
		totalRisk += decision.RiskAdjustment * decision.Weight
		totalWeight += decision.Weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return totalRisk / totalWeight
}
