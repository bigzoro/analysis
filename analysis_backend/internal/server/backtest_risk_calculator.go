package server

import (
	"fmt"
	"math"
	"math/rand"
	"sort"
)

// RiskCalculator 风险计算器
type RiskCalculator struct{}

// NewRiskCalculator 创建风险计算器
func NewRiskCalculator() *RiskCalculator {
	return &RiskCalculator{}
}

// CalculateVaR 计算VaR (Value at Risk)
func (rc *RiskCalculator) CalculateVaR(returns []float64, confidenceLevel float64, method string) (float64, error) {
	if len(returns) < 30 {
		return 0, fmt.Errorf("收益率数据不足，至少需要30个数据点")
	}

	switch method {
	case "historical":
		return rc.calculateHistoricalVaR(returns, confidenceLevel)
	case "parametric":
		return rc.calculateParametricVaR(returns, confidenceLevel)
	case "monte_carlo":
		return rc.calculateMonteCarloVaR(returns, confidenceLevel)
	default:
		return rc.calculateHistoricalVaR(returns, confidenceLevel)
	}
}

// calculateHistoricalVaR 历史模拟法VaR
func (rc *RiskCalculator) calculateHistoricalVaR(returns []float64, confidenceLevel float64) (float64, error) {
	sortedReturns := make([]float64, len(returns))
	copy(sortedReturns, returns)
	sort.Float64s(sortedReturns)

	// 计算分位数位置
	index := int(float64(len(sortedReturns)) * (1 - confidenceLevel))
	if index >= len(sortedReturns) {
		index = len(sortedReturns) - 1
	}

	varValue := -sortedReturns[index] // VaR为正值，取绝对值

	return varValue, nil
}

// calculateParametricVaR 参数法VaR
func (rc *RiskCalculator) calculateParametricVaR(returns []float64, confidenceLevel float64) (float64, error) {
	// 计算均值和标准差
	mean, stdDev := rc.calculateMeanStdDev(returns)
	if stdDev == 0 {
		return 0, fmt.Errorf("标准差为0，无法计算参数VaR")
	}

	// 使用正态分布假设
	// VaR = mean - z * stdDev (假设损失为负收益)
	zScore := rc.getZScore(confidenceLevel)
	varValue := -(mean - zScore*stdDev) // 转换为正的VaR值

	return varValue, nil
}

// calculateMonteCarloVaR 蒙特卡洛VaR
func (rc *RiskCalculator) calculateMonteCarloVaR(returns []float64, confidenceLevel float64) (float64, error) {
	if len(returns) < 10 {
		return 0, fmt.Errorf("数据不足，无法进行蒙特卡洛模拟")
	}

	// 计算历史参数
	mean, stdDev := rc.calculateMeanStdDev(returns)

	// 生成模拟收益
	simulations := 10000
	simulatedReturns := make([]float64, simulations)

	for i := 0; i < simulations; i++ {
		// 使用正态分布生成随机收益
		randomReturn := mean + stdDev*rand.NormFloat64()
		simulatedReturns[i] = randomReturn
	}

	// 排序模拟结果
	sort.Float64s(simulatedReturns)

	// 计算VaR
	index := int(float64(simulations) * (1 - confidenceLevel))
	if index >= simulations {
		index = simulations - 1
	}

	varValue := -simulatedReturns[index]

	return varValue, nil
}

// CalculateCVaR 计算CVaR (Conditional VaR)
func (rc *RiskCalculator) CalculateCVaR(returns []float64, confidenceLevel float64) (float64, error) {
	if len(returns) < 30 {
		return 0, fmt.Errorf("收益率数据不足")
	}

	// 计算VaR
	var95, err := rc.CalculateVaR(returns, confidenceLevel, "historical")
	if err != nil {
		return 0, err
	}

	// 找出超过VaR的损失
	tailLosses := make([]float64, 0)
	for _, ret := range returns {
		if -ret > var95 { // 转换为损失
			tailLosses = append(tailLosses, -ret)
		}
	}

	if len(tailLosses) == 0 {
		return var95, nil // 如果没有尾部损失，返回VaR
	}

	// 计算尾部损失的平均值
	sum := 0.0
	for _, loss := range tailLosses {
		sum += loss
	}

	cvar := sum / float64(len(tailLosses))
	return cvar, nil
}

// CalculateExpectedShortfall 计算预期亏空
func (rc *RiskCalculator) CalculateExpectedShortfall(returns []float64, confidenceLevel float64) (float64, error) {
	return rc.CalculateCVaR(returns, confidenceLevel)
}

// CalculateStressTest 执行压力测试
func (rc *RiskCalculator) CalculateStressTest(returns []float64, scenarios []StressScenario) []StressTestResult {
	results := make([]StressTestResult, len(scenarios))

	for i, scenario := range scenarios {
		result := StressTestResult{
			Scenario: scenario,
			Shock:    scenario.Shock, // 设置冲击值
		}

		// 应用压力情景到收益序列
		stressedReturns := rc.applyStressScenario(returns, scenario)

		// 计算压力下的风险指标
		var95, _ := rc.CalculateVaR(stressedReturns, 0.95, "historical")
		cvar95, _ := rc.CalculateCVaR(stressedReturns, 0.95)
		maxDrawdown := rc.calculateMaxDrawdownFromReturns(stressedReturns)
		worstReturn := rc.findWorstReturn(stressedReturns)

		// 计算预期损失（简化计算）
		if len(stressedReturns) > 0 && len(returns) > 0 {
			avgStressedReturn := 0.0
			for _, ret := range stressedReturns {
				avgStressedReturn += ret
			}
			avgStressedReturn /= float64(len(stressedReturns))

			avgNormalReturn := 0.0
			for _, ret := range returns {
				avgNormalReturn += ret
			}
			avgNormalReturn /= float64(len(returns))

			result.Loss = avgNormalReturn - avgStressedReturn
		}

		result.VaR95 = var95
		result.CVaR95 = cvar95
		result.MaxDrawdown = maxDrawdown
		result.WorstReturn = worstReturn

		results[i] = result
	}

	return results
}

// applyStressScenario 应用压力情景
func (rc *RiskCalculator) applyStressScenario(returns []float64, scenario StressScenario) []float64 {
	stressedReturns := make([]float64, len(returns))

	for i, ret := range returns {
		stressedReturn := ret

		// 应用波动率冲击
		if scenario.VolatilityShock != 0 {
			// 增加波动率
			shock := scenario.VolatilityShock * rc.generateNormalRandom()
			stressedReturn += shock
		}

		// 应用市场冲击
		if scenario.MarketShock != 0 {
			stressedReturn *= (1 + scenario.MarketShock)
		}

		// 应用流动性冲击
		if scenario.LiquidityShock != 0 {
			// 模拟流动性影响的交易成本增加
			stressedReturn -= math.Abs(stressedReturn) * scenario.LiquidityShock
		}

		stressedReturns[i] = stressedReturn
	}

	return stressedReturns
}

// CalculatePortfolioVaR 计算投资组合VaR
func (rc *RiskCalculator) CalculatePortfolioVaR(positions []Position, covarianceMatrix [][]float64, confidenceLevel float64) (float64, error) {
	if len(positions) == 0 {
		return 0, fmt.Errorf("没有持仓")
	}

	if len(covarianceMatrix) != len(positions) {
		return 0, fmt.Errorf("协方差矩阵维度不匹配")
	}

	// 计算投资组合方差
	portfolioVariance := 0.0
	for i := 0; i < len(positions); i++ {
		for j := 0; j < len(positions); j++ {
			weightI := positions[i].Weight
			weightJ := positions[j].Weight
			portfolioVariance += weightI * weightJ * covarianceMatrix[i][j]
		}
	}

	portfolioStdDev := math.Sqrt(portfolioVariance)

	// 使用正态分布假设计算VaR
	zScore := rc.getZScore(confidenceLevel)
	varValue := zScore * portfolioStdDev

	return varValue, nil
}

// CalculateBeta 计算Beta系数
func (rc *RiskCalculator) CalculateBeta(assetReturns, marketReturns []float64) (float64, error) {
	if len(assetReturns) != len(marketReturns) {
		return 0, fmt.Errorf("资产收益和市场收益长度不匹配")
	}

	if len(assetReturns) < 2 {
		return 0, fmt.Errorf("数据点不足")
	}

	// 计算协方差
	covariance := rc.calculateCovariance(assetReturns, marketReturns)

	// 计算市场收益方差
	_, marketVariance := rc.calculateMeanStdDev(marketReturns)

	if marketVariance == 0 {
		return 0, fmt.Errorf("市场收益方差为0")
	}

	beta := covariance / marketVariance
	return beta, nil
}

// CalculateSharpeRatio 计算夏普比率
func (rc *RiskCalculator) CalculateSharpeRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	meanReturn, stdDev := rc.calculateMeanStdDev(returns)
	if stdDev == 0 {
		return 0
	}

	// 年化夏普比率 (假设252个交易日)
	excessReturn := meanReturn - riskFreeRate/252
	sharpeRatio := excessReturn / stdDev * math.Sqrt(252)

	return sharpeRatio
}

// CalculateSortinoRatio 计算索提诺比率
func (rc *RiskCalculator) CalculateSortinoRatio(returns []float64, riskFreeRate float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 计算下行偏差
	downsideReturns := make([]float64, 0)
	for _, ret := range returns {
		if ret < riskFreeRate/252 {
			downsideReturns = append(downsideReturns, ret-riskFreeRate/252)
		}
	}

	if len(downsideReturns) == 0 {
		return 0
	}

	// 计算下行偏差
	downsideDeviation := 0.0
	for _, dr := range downsideReturns {
		downsideDeviation += dr * dr
	}
	downsideDeviation = math.Sqrt(downsideDeviation / float64(len(downsideReturns)))

	if downsideDeviation == 0 {
		return 0
	}

	// 计算平均超额收益
	meanReturn, _ := rc.calculateMeanStdDev(returns)
	excessReturn := meanReturn - riskFreeRate/252

	sortinoRatio := excessReturn / downsideDeviation * math.Sqrt(252)
	return sortinoRatio
}

// Helper functions

func (rc *RiskCalculator) calculateMeanStdDev(data []float64) (float64, float64) {
	if len(data) == 0 {
		return 0, 0
	}

	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(len(data))

	variance := 0.0
	for _, v := range data {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(data) - 1)
	stdDev := math.Sqrt(variance)

	return mean, stdDev
}

func (rc *RiskCalculator) getZScore(confidenceLevel float64) float64 {
	// 使用正态分布的临界值
	// 这里简化处理，使用近似值
	switch confidenceLevel {
	case 0.95:
		return 1.645
	case 0.99:
		return 2.326
	case 0.999:
		return 3.090
	default:
		// 使用逆正态分布近似
		return math.Sqrt(2) * rc.inverseErf(2*confidenceLevel-1)
	}
}

func (rc *RiskCalculator) inverseErf(x float64) float64 {
	// 简化的逆误差函数近似
	// 实际实现应该使用更精确的方法
	return x * (1.0 + 0.5*x*x)
}

func (rc *RiskCalculator) generateNormalRandom() float64 {
	// 使用正态分布随机数
	return rand.NormFloat64()
}

func (rc *RiskCalculator) calculateCovariance(x, y []float64) float64 {
	if len(x) != len(y) {
		return 0
	}

	meanX, _ := rc.calculateMeanStdDev(x)
	meanY, _ := rc.calculateMeanStdDev(y)

	covariance := 0.0
	for i := 0; i < len(x); i++ {
		covariance += (x[i] - meanX) * (y[i] - meanY)
	}
	covariance /= float64(len(x) - 1)

	return covariance
}

func (rc *RiskCalculator) calculateMaxDrawdownFromReturns(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	// 转换为累积收益
	cumulative := make([]float64, len(returns)+1)
	cumulative[0] = 1.0 // 初始价值

	for i, ret := range returns {
		cumulative[i+1] = cumulative[i] * (1 + ret)
	}

	// 计算最大回撤
	maxDrawdown := 0.0
	peak := cumulative[0]

	for _, value := range cumulative {
		if value > peak {
			peak = value
		}

		drawdown := (peak - value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

func (rc *RiskCalculator) findWorstReturn(returns []float64) float64 {
	if len(returns) == 0 {
		return 0
	}

	worst := returns[0]
	for _, ret := range returns {
		if ret < worst {
			worst = ret
		}
	}

	return worst
}

// StressScenario 压力测试情景
type StressScenario struct {
	Name            string  `json:"name"`
	Description     string  `json:"description"`
	Shock           float64 `json:"shock"`            // 简化冲击值（向后兼容）
	VolatilityShock float64 `json:"volatility_shock"` // 波动率冲击
	MarketShock     float64 `json:"market_shock"`     // 市场冲击
	LiquidityShock  float64 `json:"liquidity_shock"`  // 流动性冲击
	Probability     float64 `json:"probability"`      // 发生概率
}

// StressTestResult 压力测试结果
type StressTestResult struct {
	Scenario    StressScenario `json:"scenario"`
	Shock       float64        `json:"shock"` // 冲击值（向后兼容）
	Loss        float64        `json:"loss"`  // 预期损失（向后兼容）
	VaR95       float64        `json:"var_95"`
	CVaR95      float64        `json:"cvar_95"`
	MaxDrawdown float64        `json:"max_drawdown"`
	WorstReturn float64        `json:"worst_return"`
}

// Position 持仓信息
type Position struct {
	Symbol   string  `json:"symbol"`
	Weight   float64 `json:"weight"`
	Quantity float64 `json:"quantity"`
	Price    float64 `json:"price"`
	Value    float64 `json:"value"`
}
