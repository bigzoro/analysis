package execution

import (
	"context"
	"fmt"
	"log"
	"analysis/internal/server/strategy/shared/execution"
	"math"
	"time"
)

// ============================================================================
// 均线策略执行器实现
// ============================================================================

// Executor 均线策略执行器
type Executor struct {
	dependencies *ExecutionDependencies
}

// NewExecutor 创建均线策略执行器
func NewExecutor(deps *ExecutionDependencies) *Executor {
	return &Executor{
		dependencies: deps,
	}
}

// GetStrategyType 获取策略类型
func (e *Executor) GetStrategyType() string {
	return "moving_average"
}

// IsEnabled 检查策略是否启用
func (e *Executor) IsEnabled(config interface{}) bool {
	maConfig, ok := config.(*MovingAverageExecutionConfig)
	if !ok {
		return false
	}
	return maConfig.Enabled && maConfig.MovingAverageEnabled
}

// ValidateExecution 预执行验证
func (e *Executor) ValidateExecution(symbol string, marketData *execution.MarketData, config interface{}) error {
	maConfig, ok := config.(*MovingAverageExecutionConfig)
	if !ok {
		return fmt.Errorf("无效的配置类型: %T", config)
	}

	// 基础验证
	if symbol == "" {
		return fmt.Errorf("交易对不能为空")
	}

	if marketData == nil {
		return fmt.Errorf("市场数据不能为空")
	}

	// 均线参数验证
	if maConfig.ShortMAPeriod <= 0 || maConfig.LongMAPeriod <= 0 {
		return fmt.Errorf("均线周期必须大于0")
	}

	if maConfig.ShortMAPeriod >= maConfig.LongMAPeriod {
		return fmt.Errorf("短期均线周期(%d)不能大于等于长期均线周期(%d)", maConfig.ShortMAPeriod, maConfig.LongMAPeriod)
	}

	// 检查是否有足够的技术指标数据
	if maConfig.MAType == "SMA" {
		if marketData.SMA5 == 0 || marketData.SMA10 == 0 || marketData.SMA20 == 0 {
			return fmt.Errorf("SMA指标数据不完整")
		}
	}

	return nil
}

// Execute 执行策略
func (e *Executor) Execute(ctx context.Context, symbol string, marketData *execution.MarketData,
	config interface{}, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	maConfig, ok := config.(*MovingAverageExecutionConfig)
	if !ok {
		return nil, fmt.Errorf("无效的配置类型: %T", config)
	}

	log.Printf("[MovingAverageExecutor] 开始执行均线策略: %s, 用户: %d", symbol, execContext.UserID)

	// 预执行验证
	if err := e.ValidateExecution(symbol, marketData, config); err != nil {
		log.Printf("[MovingAverageExecutor] 验证失败: %v", err)
		return &execution.ExecutionResult{
			Action:    "skip",
			Reason:    fmt.Sprintf("验证失败: %v", err),
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 检测交叉信号
	goldenCross, deathCross := e.detectCrossSignals(marketData, maConfig)

	// 检查趋势过滤
	if maConfig.MATrendFilter {
		trendOk := e.checkTrendFilter(marketData, maConfig)
		if !trendOk {
			return &execution.ExecutionResult{
				Action:    "skip",
				Reason:    "趋势过滤条件不满足",
				Symbol:    symbol,
				Timestamp: time.Now(),
			}, nil
		}
	}

	// 根据交叉信号和配置决定执行动作
	switch maConfig.MACrossSignal {
	case "GOLDEN_CROSS":
		if goldenCross {
			return e.ExecuteGoldenCross(ctx, symbol, marketData, maConfig, execContext)
		}
	case "DEATH_CROSS":
		if deathCross {
			return e.ExecuteDeathCross(ctx, symbol, marketData, maConfig, execContext)
		}
	case "BOTH":
		if goldenCross {
			return e.ExecuteGoldenCross(ctx, symbol, marketData, maConfig, execContext)
		}
		if deathCross {
			return e.ExecuteDeathCross(ctx, symbol, marketData, maConfig, execContext)
		}
	}

	return &execution.ExecutionResult{
		Action:    "no_op",
		Reason:    "未检测到有效的均线交叉信号",
		Symbol:    symbol,
		Timestamp: time.Now(),
	}, nil
}

// detectCrossSignals 检测交叉信号
func (e *Executor) detectCrossSignals(marketData *execution.MarketData, config *MovingAverageExecutionConfig) (goldenCross, deathCross bool) {
	// 获取K线数据进行均线计算
	if e.dependencies.MarketDataProvider == nil {
		// 如果没有市场数据提供者，使用简化的逻辑
		var shortMA, longMA float64

		switch config.MAType {
		case "SMA":
			shortMA = e.getSMAValue(marketData, config.ShortMAPeriod)
			longMA = e.getSMAValue(marketData, config.LongMAPeriod)
		case "EMA":
			// 这里可以扩展EMA逻辑
			shortMA = e.getSMAValue(marketData, config.ShortMAPeriod)
			longMA = e.getSMAValue(marketData, config.LongMAPeriod)
		default:
			shortMA = marketData.SMA5  // 默认使用SMA5
			longMA = marketData.SMA10  // 默认使用SMA10
		}

		if shortMA == 0 || longMA == 0 {
			return false, false
		}

		// 简化的交叉检测：比较短期均线和长期均线
		// 注意：这只是基本检测，实际应该比较当前和前一个周期
		goldenCross = shortMA > longMA // 短期线上穿长期线
		deathCross = shortMA < longMA  // 短期线下穿长期线

		return goldenCross, deathCross
	}

	// 使用市场数据提供者获取完整的K线数据进行精确计算
	klineData, err := e.dependencies.MarketDataProvider.GetKlineData(marketData.Symbol, "1h", config.LongMAPeriod+10)
	if err != nil {
		log.Printf("[MovingAverageExecutor] 获取K线数据失败: %v", err)
		return false, false
	}

	if len(klineData) < config.LongMAPeriod {
		log.Printf("[MovingAverageExecutor] K线数据不足，需要%d个数据点，实际%d个", config.LongMAPeriod, len(klineData))
		return false, false
	}

	// 提取收盘价
	prices := make([]float64, len(klineData))
	for i, kline := range klineData {
		prices[i] = kline.ClosePrice
	}

	// 计算均线
	shortMA := e.calculateMovingAverage(prices, config.ShortMAPeriod, config.MAType)
	longMA := e.calculateMovingAverage(prices, config.LongMAPeriod, config.MAType)

	if len(shortMA) < 2 || len(longMA) < 2 {
		return false, false
	}

	// 获取最新的均线值
	currentShortMA := shortMA[len(shortMA)-1]
	currentLongMA := longMA[len(longMA)-1]
	previousShortMA := shortMA[len(shortMA)-2]
	previousLongMA := longMA[len(longMA)-2]

	// 检测交叉信号
	// 金叉：前一个周期短期均线 <= 长期均线，当前周期短期均线 > 长期均线
	goldenCross = (previousShortMA <= previousLongMA) && (currentShortMA > currentLongMA)

	// 死叉：前一个周期短期均线 >= 长期均线，当前周期短期均线 < 长期均线
	deathCross = (previousShortMA >= previousLongMA) && (currentShortMA < currentLongMA)

	return goldenCross, deathCross
}

// calculateMovingAverage 计算移动平均线
func (e *Executor) calculateMovingAverage(prices []float64, period int, maType string) []float64 {
	if len(prices) < period {
		return nil
	}

	result := make([]float64, len(prices)-period+1)

	switch maType {
	case "SMA":
		// 简单移动平均
		for i := 0; i <= len(prices)-period; i++ {
			sum := 0.0
			for j := i; j < i+period; j++ {
				sum += prices[j]
			}
			result[i] = sum / float64(period)
		}
	case "EMA":
		// 指数移动平均
		multiplier := 2.0 / (float64(period) + 1.0)

		// 第一个EMA值使用SMA
		sum := 0.0
		for i := 0; i < period; i++ {
			sum += prices[i]
		}
		result[0] = sum / float64(period)

		// 计算后续的EMA值
		for i := 1; i < len(result); i++ {
			result[i] = (prices[i+period-1] - result[i-1]) * multiplier + result[i-1]
		}
	default:
		// 默认使用SMA
		for i := 0; i <= len(prices)-period; i++ {
			sum := 0.0
			for j := i; j < i+period; j++ {
				sum += prices[j]
			}
			result[i] = sum / float64(period)
		}
	}

	return result
}

// getSMAValue 获取SMA值
func (e *Executor) getSMAValue(marketData *execution.MarketData, period int) float64 {
	switch period {
	case 5:
		return marketData.SMA5
	case 10:
		return marketData.SMA10
	case 20:
		return marketData.SMA20
	case 50:
		return marketData.SMA50
	default:
		return 0
	}
}

// checkTrendFilter 检查趋势过滤
func (e *Executor) checkTrendFilter(marketData *execution.MarketData, config *MovingAverageExecutionConfig) bool {
	if !config.MATrendFilter {
		return true // 如果未启用趋势过滤，直接通过
	}

	// 获取更详细的趋势数据
	if e.dependencies.MarketDataProvider != nil {
		// 使用K线数据进行趋势分析
		klineData, err := e.dependencies.MarketDataProvider.GetKlineData(marketData.Symbol, "1h", 50)
		if err == nil && len(klineData) >= 20 {
			return e.analyzeTrendFromKlines(klineData, config)
		}
	}

	// 备选方案：使用简化的市场数据进行趋势判断
	switch config.MATrendDirection {
	case "UP":
		// 上涨趋势：24小时涨幅为正且价格高于SMA20
		return marketData.Change24h > 0 && marketData.Price > marketData.SMA20
	case "DOWN":
		// 下跌趋势：24小时涨幅为负且价格低于SMA20
		return marketData.Change24h < 0 && marketData.Price < marketData.SMA20
	case "BOTH":
		return true // 不限制趋势
	default:
		return true
	}
}

// analyzeTrendFromKlines 从K线数据分析趋势
func (e *Executor) analyzeTrendFromKlines(klineData []*execution.KlineData, config *MovingAverageExecutionConfig) bool {
	if len(klineData) < 20 {
		return true // 数据不足，默认通过
	}

	// 提取收盘价
	prices := make([]float64, len(klineData))
	for i, kline := range klineData {
		prices[i] = kline.ClosePrice
	}

	// 计算长期趋势（20周期）
	longMA := e.calculateMovingAverage(prices, 20, "SMA")
	if len(longMA) < 2 {
		return true
	}

	// 计算短期趋势（5周期）
	shortMA := e.calculateMovingAverage(prices, 5, "SMA")
	if len(shortMA) < 2 {
		return true
	}

	// 获取最新值
	currentPrice := prices[len(prices)-1]
	currentLongMA := longMA[len(longMA)-1]
	currentShortMA := shortMA[len(shortMA)-1]

	switch config.MATrendDirection {
	case "UP":
		// 上涨趋势：价格高于长期均线，且短期均线高于长期均线
		return currentPrice > currentLongMA && currentShortMA > currentLongMA
	case "DOWN":
		// 下跌趋势：价格低于长期均线，且短期均线低于长期均线
		return currentPrice < currentLongMA && currentShortMA < currentLongMA
	case "BOTH":
		return true // 不限制趋势
	default:
		return true
	}
}

// ExecuteGoldenCross 执行金叉信号
func (e *Executor) ExecuteGoldenCross(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *MovingAverageExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[MovingAverageExecutor] 执行金叉买入: %s, 价格: %.4f", symbol, marketData.Price)

	// 计算执行参数
	multiplier := config.LongMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// 如果启用了杠杆，使用杠杆倍数覆盖基础倍数
	if config.AllowLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	// 计算风险管理参数
	var stopLossPrice, takeProfitPrice float64
	if e.dependencies.RiskManager != nil {
		stopLossPrice = e.dependencies.RiskManager.CalculateStopLoss(marketData.Price, 0.02) // 2%止损
		takeProfitPrice = e.dependencies.RiskManager.CalculateTakeProfit(marketData.Price, 0.05) // 5%止盈
	}

	result := &execution.ExecutionResult{
		Action:          "buy",
		Reason:          fmt.Sprintf("%s 均线金叉信号，短期线上穿长期线", symbol),
		Multiplier:      multiplier,
		Symbol:          symbol,
		Timestamp:       time.Now(),
		StopLossPrice:   stopLossPrice,
		TakeProfitPrice: takeProfitPrice,
		MaxPositionSize: config.MaxPositionSize,
		MaxHoldHours:    config.MaxHoldHours,
		RiskLevel:       0.5, // 中等风险
	}

	// 执行实际下单
	if e.dependencies.OrderManager != nil {
		quantity := e.calculatePositionSize(marketData, config)
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "buy", quantity, marketData.Price)
		if err != nil {
			log.Printf("[MovingAverageExecutor] 金叉买入下单失败: %v", err)
			result.Action = "skip"
			result.Reason = fmt.Sprintf("金叉买入下单失败: %v", err)
		} else {
			result.OrderID = orderID
			log.Printf("[MovingAverageExecutor] 金叉买入成功: %s", orderID)
		}
	}

	return result, nil
}

// ExecuteDeathCross 执行死叉信号
func (e *Executor) ExecuteDeathCross(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *MovingAverageExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[MovingAverageExecutor] 执行死叉卖出: %s, 价格: %.4f", symbol, marketData.Price)

	// 计算执行参数
	multiplier := config.ShortMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// 如果启用了杠杆，使用杠杆倍数覆盖基础倍数
	if config.AllowLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	// 计算风险管理参数
	var stopLossPrice, takeProfitPrice float64
	if e.dependencies.RiskManager != nil {
		stopLossPrice = e.dependencies.RiskManager.CalculateStopLoss(marketData.Price, 0.02)   // 2%止损
		takeProfitPrice = e.dependencies.RiskManager.CalculateTakeProfit(marketData.Price, 0.03) // 3%止盈
	}

	result := &execution.ExecutionResult{
		Action:          "sell",
		Reason:          fmt.Sprintf("均线死叉信号，%d日线下穿%d日线", config.ShortMAPeriod, config.LongMAPeriod),
		Multiplier:      multiplier,
		Symbol:          symbol,
		Timestamp:       time.Now(),
		StopLossPrice:   stopLossPrice,
		TakeProfitPrice: takeProfitPrice,
		MaxPositionSize: config.MaxPositionSize,
		MaxHoldHours:    config.MaxHoldHours,
		RiskLevel:       0.5, // 中等风险
	}

	// 执行实际下单
	if e.dependencies.OrderManager != nil {
		quantity := e.calculatePositionSize(marketData, config)
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "sell", quantity, marketData.Price)
		if err != nil {
			log.Printf("[MovingAverageExecutor] 死叉卖出下单失败: %v", err)
			result.Action = "skip"
			result.Reason = fmt.Sprintf("死叉卖出下单失败: %v", err)
		} else {
			result.OrderID = orderID
			log.Printf("[MovingAverageExecutor] 死叉卖出成功: %s", orderID)
		}
	}

	return result, nil
}

// calculatePositionSize 计算仓位大小
func (e *Executor) calculatePositionSize(marketData *execution.MarketData, config *MovingAverageExecutionConfig) float64 {
	// 简化的仓位计算逻辑
	baseQuantity := 100.0 // 基础数量

	// 根据波动率调整仓位
	volatility := math.Abs(marketData.Change24h)
	if volatility > 0.05 { // 高波动
		baseQuantity *= 0.5
	} else if volatility < 0.01 { // 低波动
		baseQuantity *= 1.5
	}

	// 限制最大仓位
	if config.MaxPositionSize > 0 && baseQuantity > config.MaxPositionSize {
		baseQuantity = config.MaxPositionSize
	}

	return baseQuantity
}