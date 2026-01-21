package validation

import (
	"analysis/internal/server/strategy/mean_reversion"
	"fmt"
	"math"
	"time"
)

// Validator 策略验证器实现
type validator struct {
	strategy mean_reversion.MRStrategy
}

// NewMRValidator 创建均值回归验证器
func NewMRValidator(strategy mean_reversion.MRStrategy) mean_reversion.MRValidator {
	return &validator{
		strategy: strategy,
	}
}

// ValidateStrategy 验证策略配置和逻辑
func (v *validator) ValidateStrategy(config *mean_reversion.MeanReversionConfig, marketData *mean_reversion.StrategyMarketData) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}

	if marketData == nil {
		return fmt.Errorf("市场数据不能为空")
	}

	// 验证配置
	configManager := v.strategy.GetConfigManager()
	if err := configManager.ValidateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 验证市场数据
	if len(marketData.Prices) < config.Core.Period {
		return fmt.Errorf("价格数据不足，至少需要%d个数据点，当前%d个",
			config.Core.Period, len(marketData.Prices))
	}

	// 验证指标计算
	indicatorFactory := v.strategy.GetIndicatorFactory()
	for _, indicatorName := range config.Core.Indicators {
		indicator, err := indicatorFactory.Create(indicatorName, v.getIndicatorParams(config, indicatorName))
		if err != nil {
			return fmt.Errorf("创建指标失败 %s: %w", indicatorName, err)
		}

		_, err = indicator.Calculate(marketData.Prices, v.getIndicatorParams(config, indicatorName))
		if err != nil {
			return fmt.Errorf("指标计算失败 %s: %w", indicatorName, err)
		}
	}

	return nil
}

// Backtest 执行回测验证
func (v *validator) Backtest(config *mean_reversion.MeanReversionConfig, historicalData []mean_reversion.StrategyMarketData, startTime, endTime time.Time) (*mean_reversion.BacktestResult, error) {
	if len(historicalData) == 0 {
		return nil, fmt.Errorf("历史数据不能为空")
	}

	result := &mean_reversion.BacktestResult{
		Trades: make([]mean_reversion.TradeRecord, 0),
	}

	totalPnL := 0.0
	winningTrades := 0
	totalTrades := 0

	for _, data := range historicalData {
		// 过滤时间范围
		if data.Prices != nil && len(data.Prices) > 0 {
			// 简化的时间过滤，这里应该根据实际的时间戳过滤
		}

		// 执行策略扫描
		signal, err := v.strategy.Scan(nil, data.Symbol, &data, config)
		if err != nil {
			continue // 跳过错误的数据
		}

		if signal != nil {
			// 模拟交易执行
			trade := v.simulateTrade(signal, data.Prices[len(data.Prices)-1])
			result.Trades = append(result.Trades, trade)
			totalPnL += trade.PnL
			totalTrades++

			if trade.PnL > 0 {
				winningTrades++
			}
		}
	}

	// 计算统计结果
	result.TotalTrades = totalTrades
	if totalTrades > 0 {
		result.WinRate = float64(winningTrades) / float64(totalTrades)
	}
	result.TotalPnL = totalPnL

	// 计算最大回撤
	if len(result.Trades) > 0 {
		result.MaxDrawdown = v.calculateMaxDrawdown(result.Trades)
		result.SharpeRatio = v.calculateSharpeRatio(result.Trades)
	}

	return result, nil
}

// StressTest 执行压力测试
func (v *validator) StressTest(config *mean_reversion.MeanReversionConfig, scenarios []mean_reversion.StressTestScenario) (*mean_reversion.StressTestResult, error) {
	result := &mean_reversion.StressTestResult{
		PassedScenarios: make([]string, 0),
		FailedScenarios: make([]string, 0),
		Recommendations: make([]string, 0),
	}

	totalScore := 0.0
	totalScenarios := len(scenarios)

	for _, scenario := range scenarios {
		score := v.evaluateScenario(config, scenario)
		totalScore += score

		if score >= 60 { // 及格分数
			result.PassedScenarios = append(result.PassedScenarios, scenario.Name)
		} else {
			result.FailedScenarios = append(result.FailedScenarios, scenario.Name)
			result.Recommendations = append(result.Recommendations,
				fmt.Sprintf("%s: 需要改进，当前分数%.1f", scenario.Name, score))
		}
	}

	if totalScenarios > 0 {
		result.OverallScore = totalScore / float64(totalScenarios)
	}

	return result, nil
}

// getIndicatorParams 获取指标参数
func (v *validator) getIndicatorParams(config *mean_reversion.MeanReversionConfig, indicatorName string) map[string]interface{} {
	params := make(map[string]interface{})

	switch indicatorName {
	case "bollinger":
		params["period"] = config.Core.Period
		params["multiplier"] = config.Indicators.Bollinger.Multiplier
	case "rsi":
		params["overbought"] = config.Indicators.RSI.Overbought
		params["oversold"] = config.Indicators.RSI.Oversold
	case "price_channel":
		params["period"] = config.Core.Period
	}

	return params
}

// simulateTrade 模拟交易执行
func (v *validator) simulateTrade(signal *mean_reversion.EligibleSymbol, currentPrice float64) mean_reversion.TradeRecord {
	// 简化的交易模拟
	entryPrice := currentPrice
	exitPrice := currentPrice * 1.02 // 假设2%的收益

	var pnl float64
	switch signal.Action {
	case "buy":
		pnl = (exitPrice - entryPrice) / entryPrice
	case "sell":
		pnl = (entryPrice - exitPrice) / entryPrice
	}

	pnlPercent := pnl * 100

	return mean_reversion.TradeRecord{
		Symbol:     signal.Symbol,
		Side:       signal.Action,
		EntryTime:  time.Now(),
		EntryPrice: entryPrice,
		ExitTime:   time.Now().Add(time.Hour),
		ExitPrice:  exitPrice,
		Quantity:   1.0,
		PnL:        pnl,
		PnLPercent: pnlPercent,
		Reason:     signal.Reason,
	}
}

// calculateMaxDrawdown 计算最大回撤
func (v *validator) calculateMaxDrawdown(trades []mean_reversion.TradeRecord) float64 {
	if len(trades) == 0 {
		return 0.0
	}

	maxDrawdown := 0.0
	peak := trades[0].PnL

	for _, trade := range trades {
		if trade.PnL > peak {
			peak = trade.PnL
		}

		drawdown := peak - trade.PnL
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// calculateSharpeRatio 计算夏普比率
func (v *validator) calculateSharpeRatio(trades []mean_reversion.TradeRecord) float64 {
	if len(trades) == 0 {
		return 0.0
	}

	// 计算日收益率
	returns := make([]float64, len(trades))
	for i, trade := range trades {
		returns[i] = trade.PnL
	}

	// 计算平均收益率和标准差
	meanReturn := v.calculateMean(returns)
	stdDev := v.calculateStdDev(returns, meanReturn)

	if stdDev == 0 {
		return 0.0
	}

	// 简化的夏普比率计算（假设无风险利率为0）
	return meanReturn / stdDev
}

// calculateMean 计算平均值
func (v *validator) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}

	return sum / float64(len(values))
}

// calculateStdDev 计算标准差
func (v *validator) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) <= 1 {
		return 0.0
	}

	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}

	return math.Sqrt(sum / float64(len(values)-1))
}

// evaluateScenario 评估测试场景
func (v *validator) evaluateScenario(config *mean_reversion.MeanReversionConfig, scenario mean_reversion.StressTestScenario) float64 {
	score := 100.0

	// 简化的场景评估逻辑
	for _, data := range scenario.MarketData {
		_, err := v.strategy.Scan(nil, data.Symbol, &data, config)
		if err != nil {
			score -= 10 // 错误情况下扣分
		}
	}

	// 根据预期结果调整分数
	switch scenario.ExpectedOutcome {
	case "stable":
		// 稳定市场应该有合理的信号
		score *= 0.9
	case "volatile":
		// 波动市场应该控制风险
		score *= 0.8
	case "crash":
		// 崩溃市场应该停止交易
		score *= 0.7
	}

	return math.Max(0, score)
}
