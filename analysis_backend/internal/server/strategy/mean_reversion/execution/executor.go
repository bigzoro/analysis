package execution

import (
	mean_reversion "analysis/internal/server/strategy/mean_reversion"
	"analysis/internal/server/strategy/shared/execution"
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// ============================================================================
// 均值回归策略执行器实现
// ============================================================================

// Executor 均值回归策略执行器
type Executor struct {
	dependencies *ExecutionDependencies
}

// NewExecutor 创建均值回归策略执行器
func NewExecutor(deps *ExecutionDependencies) *Executor {
	return &Executor{
		dependencies: deps,
	}
}

// GetStrategyType 获取策略类型
func (e *Executor) GetStrategyType() string {
	return "mean_reversion"
}

// IsEnabled 检查策略是否启用
func (e *Executor) IsEnabled(config interface{}) bool {
	mrConfig, ok := config.(*MeanReversionExecutionConfig)
	if !ok {
		return false
	}
	return mrConfig.Enabled && mrConfig.MeanReversionEnabled
}

// ValidateExecution 预执行验证
func (e *Executor) ValidateExecution(symbol string, marketData *execution.MarketData, config interface{}) error {
	mrConfig, ok := config.(*MeanReversionExecutionConfig)
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

	// 策略参数验证
	if mrConfig.MeanReversionLookback <= 0 {
		return fmt.Errorf("均值回归回望周期必须大于0")
	}

	if mrConfig.MeanReversionThreshold <= 0 {
		return fmt.Errorf("均值回归阈值必须大于0")
	}

	if mrConfig.BollingerPeriod <= 0 {
		return fmt.Errorf("布林带周期必须大于0")
	}

	// 验证技术指标数据可用性
	if mrConfig.MRBollingerBandsEnabled {
		// 检查布林带有没有数据（这里需要扩展市场数据结构）
		if marketData.SMA20 == 0 {
			return fmt.Errorf("布林带指标数据不完整")
		}
	}

	return nil
}

// Execute 执行策略
func (e *Executor) Execute(ctx context.Context, symbol string, marketData *execution.MarketData,
	config interface{}, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	mrConfig, ok := config.(*MeanReversionExecutionConfig)
	if !ok {
		return nil, fmt.Errorf("无效的配置类型: %T", config)
	}

	log.Printf("[MeanReversionExecutor] 开始执行均值回归策略: %s, 用户: %d", symbol, execContext.UserID)

	// 预执行验证
	if err := e.ValidateExecution(symbol, marketData, config); err != nil {
		log.Printf("[MeanReversionExecutor] 验证失败: %v", err)
		return &execution.ExecutionResult{
			Action:    "skip",
			Reason:    fmt.Sprintf("验证失败: %v", err),
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 生成均值回归信号
	signal := e.generateMeanReversionSignal(marketData, mrConfig)

	// 根据信号决定执行动作
	switch signal.SignalType {
	case "BUY":
		return e.ExecuteMeanReversionBuy(ctx, symbol, marketData, mrConfig, execContext)
	case "SELL":
		return e.ExecuteMeanReversionSell(ctx, symbol, marketData, mrConfig, execContext)
	case "EXIT":
		return e.ExecuteMeanReversionExit(ctx, symbol, marketData, mrConfig, execContext)
	default:
		return &execution.ExecutionResult{
			Action:    "no_op",
			Reason:    "未检测到有效的均值回归信号",
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}
}

// generateMeanReversionSignal 生成均值回归信号
func (e *Executor) generateMeanReversionSignal(marketData *execution.MarketData, config *MeanReversionExecutionConfig) *MeanReversionSignal {
	// 如果有完整的市场数据提供者，使用高级信号生成
	if e.dependencies.MarketDataProvider != nil {
		return e.generateAdvancedSignal(marketData, config)
	}

	// 备选方案：使用简化的信号生成
	return e.generateSimpleSignal(marketData, config)
}

// generateAdvancedSignal 使用完整的指标系统生成信号
func (e *Executor) generateAdvancedSignal(marketData *execution.MarketData, config *MeanReversionExecutionConfig) *MeanReversionSignal {

	// 获取历史价格数据用于指标计算
	klineData, err := e.dependencies.MarketDataProvider.GetKlineData(marketData.Symbol, "1h", 100)
	if err != nil {
		log.Printf("[MeanReversionExecutor] 获取K线数据失败: %v", err)
		return e.generateSimpleSignal(marketData, config)
	}

	if len(klineData) < 20 {
		return e.generateSimpleSignal(marketData, config)
	}

	// 提取价格序列
	prices := make([]float64, len(klineData))
	for i, kline := range klineData {
		prices[i] = kline.ClosePrice
	}

	// 计算各种指标信号
	var indicatorSignals []*mean_reversion.IndicatorSignal

	// 布林带指标
	if config.MRBollingerBandsEnabled {
		bollingerSignal := e.calculateBollingerSignal(prices, config)
		if bollingerSignal != nil {
			indicatorSignals = append(indicatorSignals, bollingerSignal)
		}
	}

	// RSI指标
	rsiSignal := e.calculateRSISignal(prices, config)
	if rsiSignal != nil {
		indicatorSignals = append(indicatorSignals, rsiSignal)
	}

	// 价格通道指标
	channelSignal := e.calculatePriceChannelSignal(prices, config)
	if channelSignal != nil {
		indicatorSignals = append(indicatorSignals, channelSignal)
	}

	// 处理综合信号
	if len(indicatorSignals) > 0 {
		finalSignal := e.processCombinedSignals(indicatorSignals, config)
		return finalSignal
	}

	return e.generateSimpleSignal(marketData, config)
}

// generateSimpleSignal 生成简化的均值回归信号
func (e *Executor) generateSimpleSignal(marketData *execution.MarketData, config *MeanReversionExecutionConfig) *MeanReversionSignal {
	signal := &MeanReversionSignal{
		Symbol: marketData.Symbol,
	}

	// 计算Z分数（简化的实现）
	currentPrice := marketData.Price
	meanPrice := marketData.SMA20 // 使用SMA20作为均价
	if meanPrice > 0 {
		priceDeviation := currentPrice - meanPrice
		volatility := math.Abs(marketData.Change24h) // 使用日涨跌幅作为波动率估计
		if volatility > 0 {
			signal.ZScore = priceDeviation / (meanPrice * volatility)
		}
	}

	// 计算布林带位置
	if config.MRBollingerBandsEnabled && marketData.SMA20 > 0 {
		upperBand := marketData.SMA20 * (1 + config.BollingerStdDev*0.1) // 简化的布林带计算
		lowerBand := marketData.SMA20 * (1 - config.BollingerStdDev*0.1)

		if currentPrice > upperBand {
			signal.BollingerPosition = "UPPER"
		} else if currentPrice < lowerBand {
			signal.BollingerPosition = "LOWER"
		} else {
			signal.BollingerPosition = "MIDDLE"
		}
	}

	// 决策信号类型
	if math.Abs(signal.ZScore) > config.MeanReversionThreshold {
		if signal.ZScore < -config.MeanReversionThreshold {
			// 价格显著低于均价，买入信号
			signal.SignalType = "BUY"
			signal.Confidence = math.Min(math.Abs(signal.ZScore)/config.MeanReversionThreshold, 1.0)
			signal.ExpectedReturn = math.Abs(signal.ZScore) * 0.05 // 预期5%的回归收益
		} else if signal.ZScore > config.MeanReversionThreshold {
			// 价格显著高于均价，卖出信号
			signal.SignalType = "SELL"
			signal.Confidence = math.Min(signal.ZScore/config.MeanReversionThreshold, 1.0)
			signal.ExpectedReturn = signal.ZScore * 0.05 // 预期5%的回归收益
		}
	} else if signal.BollingerPosition == "UPPER" || signal.BollingerPosition == "LOWER" {
		// 布林带信号
		if signal.BollingerPosition == "LOWER" {
			signal.SignalType = "BUY"
			signal.Confidence = 0.7
		} else {
			signal.SignalType = "SELL"
			signal.Confidence = 0.7
		}
		signal.ExpectedReturn = 0.03 // 布林带策略预期3%的收益
	} else {
		signal.SignalType = "HOLD"
		signal.Confidence = 0.5
	}

	// 计算风险等级
	signal.RiskLevel = 1.0 - signal.Confidence // 置信度越高，风险越低

	return signal
}

// calculateBollingerSignal 计算布林带信号
func (e *Executor) calculateBollingerSignal(prices []float64, config *MeanReversionExecutionConfig) *mean_reversion.IndicatorSignal {
	if len(prices) < config.BollingerPeriod {
		return nil
	}

	// 计算SMA
	sma := e.calculateSMA(prices[len(prices)-config.BollingerPeriod:])

	// 计算标准差
	stdDev := e.calculateStdDev(prices[len(prices)-config.BollingerPeriod:], sma)

	// 计算布林带
	upperBand := sma + (stdDev * config.BollingerStdDev)
	lowerBand := sma - (stdDev * config.BollingerStdDev)
	currentPrice := prices[len(prices)-1]

	// 生成信号
	var buySignal, sellSignal bool
	var baseWeight float64

	if currentPrice < lowerBand {
		buySignal = true
		baseWeight = (lowerBand - currentPrice) / (lowerBand * 0.1) // 偏离度
	} else if currentPrice > upperBand {
		sellSignal = true
		baseWeight = (currentPrice - upperBand) / (upperBand * 0.1) // 偏离度
	} else {
		baseWeight = 0.5
	}

	return &mean_reversion.IndicatorSignal{
		Type:       "bollinger",
		BuySignal:  buySignal,
		SellSignal: sellSignal,
		BaseWeight: math.Min(baseWeight, 1.0),
		Quality:    0.8,
		Confidence: 0.8,
	}
}

// calculateRSISignal 计算RSI信号
func (e *Executor) calculateRSISignal(prices []float64, config *MeanReversionExecutionConfig) *mean_reversion.IndicatorSignal {
	if len(prices) < 14 { // RSI通常使用14周期
		return nil
	}

	rsi := e.calculateRSI(prices, 14)

	var buySignal, sellSignal bool
	var baseWeight float64

	if rsi < 30 {
		buySignal = true
		baseWeight = (30 - rsi) / 30 // RSI偏离度
	} else if rsi > 70 {
		sellSignal = true
		baseWeight = (rsi - 70) / 30 // RSI偏离度
	} else {
		baseWeight = 0.5
	}

	return &mean_reversion.IndicatorSignal{
		Type:       "rsi",
		BuySignal:  buySignal,
		SellSignal: sellSignal,
		BaseWeight: math.Min(baseWeight, 1.0),
		Quality:    0.7,
		Confidence: 0.7,
	}
}

// calculatePriceChannelSignal 计算价格通道信号
func (e *Executor) calculatePriceChannelSignal(prices []float64, config *MeanReversionExecutionConfig) *mean_reversion.IndicatorSignal {
	if len(prices) < 20 {
		return nil
	}

	// 计算20周期最高价和最低价
	high := math.Inf(-1)
	low := math.Inf(1)
	for i := len(prices) - 20; i < len(prices); i++ {
		if prices[i] > high {
			high = prices[i]
		}
		if prices[i] < low {
			low = prices[i]
		}
	}

	currentPrice := prices[len(prices)-1]
	channelRange := high - low
	position := (currentPrice - low) / channelRange

	var buySignal, sellSignal bool
	var baseWeight float64

	if position < 0.2 { // 接近下轨
		buySignal = true
		baseWeight = (0.2 - position) / 0.2
	} else if position > 0.8 { // 接近上轨
		sellSignal = true
		baseWeight = (position - 0.8) / 0.2
	} else {
		baseWeight = 0.5
	}

	return &mean_reversion.IndicatorSignal{
		Type:       "price_channel",
		BuySignal:  buySignal,
		SellSignal: sellSignal,
		BaseWeight: math.Min(baseWeight, 1.0),
		Quality:    0.6,
		Confidence: 0.6,
	}
}

// processCombinedSignals 处理综合信号
func (e *Executor) processCombinedSignals(signals []*mean_reversion.IndicatorSignal, config *MeanReversionExecutionConfig) *MeanReversionSignal {
	if len(signals) == 0 {
		return &MeanReversionSignal{
			Symbol:     "unknown",
			SignalType: "HOLD",
			Confidence: 0.5,
			RiskLevel:  0.5,
		}
	}

	// 计算综合信号强度
	buyStrength := 0.0
	sellStrength := 0.0
	totalConfidence := 0.0

	for _, signal := range signals {
		totalConfidence += signal.Confidence
		if signal.BuySignal {
			buyStrength += signal.BaseWeight
		}
		if signal.SellSignal {
			sellStrength += signal.BaseWeight
		}
	}

	avgConfidence := totalConfidence / float64(len(signals))

	// 决策最终信号
	var signalType string
	var confidence float64

	if buyStrength > sellStrength && buyStrength > 0.5 {
		signalType = "BUY"
		confidence = math.Min(buyStrength/float64(len(signals)), 1.0)
	} else if sellStrength > buyStrength && sellStrength > 0.5 {
		signalType = "SELL"
		confidence = math.Min(sellStrength/float64(len(signals)), 1.0)
	} else {
		signalType = "HOLD"
		confidence = 0.5
	}

	return &MeanReversionSignal{
		Symbol:            "unknown", // 需要从调用处传递
		SignalType:        signalType,
		Confidence:        confidence * avgConfidence,
		ZScore:            0, // 高级信号不使用简化的Z分数
		BollingerPosition: "",
		ExpectedReturn:    confidence * 0.04, // 预期4%的收益
		RiskLevel:         1.0 - confidence,
	}
}

// 辅助计算函数
func (e *Executor) calculateSMA(prices []float64) float64 {
	sum := 0.0
	for _, price := range prices {
		sum += price
	}
	return sum / float64(len(prices))
}

func (e *Executor) calculateStdDev(prices []float64, mean float64) float64 {
	sum := 0.0
	for _, price := range prices {
		sum += math.Pow(price-mean, 2)
	}
	return math.Sqrt(sum / float64(len(prices)))
}

func (e *Executor) calculateRSI(prices []float64, period int) float64 {
	if len(prices) < period+1 {
		return 50 // 中性值
	}

	gains := 0.0
	losses := 0.0

	// 计算初始收益和损失
	for i := 1; i <= period; i++ {
		change := prices[i] - prices[i-1]
		if change > 0 {
			gains += change
		} else {
			losses -= change
		}
	}

	// 计算平均收益和损失
	avgGain := gains / float64(period)
	avgLoss := losses / float64(period)

	// 计算RS和RSI
	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// ExecuteMeanReversionBuy 执行均值回归买入
func (e *Executor) ExecuteMeanReversionBuy(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *MeanReversionExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[MeanReversionExecutor] 执行均值回归买入: %s, 价格: %.4f", symbol, marketData.Price)

	// 计算执行参数
	multiplier := config.LongMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// 如果启用了杠杆，使用杠杆倍数覆盖基础倍数
	if config.AllowLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	// 计算仓位大小（基于波动率调整）
	positionSize := e.calculateMeanReversionPosition(marketData, config)

	// 计算风险管理参数
	var stopLossPrice, takeProfitPrice float64
	if e.dependencies.RiskManager != nil {
		stopLossPrice = e.dependencies.RiskManager.CalculateStopLoss(marketData.Price, config.LossLimit)
		takeProfitPrice = e.dependencies.RiskManager.CalculateTakeProfit(marketData.Price, config.ProfitTarget)
	}

	result := &execution.ExecutionResult{
		Action:          "buy",
		Reason:          "均值回归信号：价格偏离均值，预期回归",
		Multiplier:      multiplier,
		Symbol:          symbol,
		Timestamp:       time.Now(),
		StopLossPrice:   stopLossPrice,
		TakeProfitPrice: takeProfitPrice,
		MaxPositionSize: positionSize,
		MaxHoldHours:    config.MaxPositionTime,
		RiskLevel:       0.4, // 中等风险
	}

	// 执行买入订单
	if e.dependencies.OrderManager != nil {
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "buy", positionSize, marketData.Price)
		if err != nil {
			log.Printf("[MeanReversionExecutor] 买入订单失败: %v", err)
			result.Action = "skip"
			result.Reason = fmt.Sprintf("买入订单失败: %v", err)
		} else {
			result.OrderID = orderID
			log.Printf("[MeanReversionExecutor] 买入订单成功: %s", orderID)
		}
	}

	return result, nil
}

// ExecuteMeanReversionSell 执行均值回归卖出
func (e *Executor) ExecuteMeanReversionSell(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *MeanReversionExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[MeanReversionExecutor] 执行均值回归卖出: %s, 价格: %.4f", symbol, marketData.Price)

	// 计算执行参数
	multiplier := config.ShortMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// 如果启用了杠杆，使用杠杆倍数覆盖基础倍数
	if config.AllowLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	// 计算仓位大小
	positionSize := e.calculateMeanReversionPosition(marketData, config)

	// 计算风险管理参数
	var stopLossPrice, takeProfitPrice float64
	if e.dependencies.RiskManager != nil {
		stopLossPrice = e.dependencies.RiskManager.CalculateStopLoss(marketData.Price, config.LossLimit)
		takeProfitPrice = e.dependencies.RiskManager.CalculateTakeProfit(marketData.Price, config.ProfitTarget)
	}

	result := &execution.ExecutionResult{
		Action:          "sell",
		Reason:          "均值回归信号：价格偏离均值，预期回归",
		Multiplier:      multiplier,
		Symbol:          symbol,
		Timestamp:       time.Now(),
		StopLossPrice:   stopLossPrice,
		TakeProfitPrice: takeProfitPrice,
		MaxPositionSize: positionSize,
		MaxHoldHours:    config.MaxPositionTime,
		RiskLevel:       0.4, // 中等风险
	}

	// 执行卖出订单
	if e.dependencies.OrderManager != nil {
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "sell", positionSize, marketData.Price)
		if err != nil {
			log.Printf("[MeanReversionExecutor] 卖出订单失败: %v", err)
			result.Action = "skip"
			result.Reason = fmt.Sprintf("卖出订单失败: %v", err)
		} else {
			result.OrderID = orderID
			log.Printf("[MeanReversionExecutor] 卖出订单成功: %s", orderID)
		}
	}

	return result, nil
}

// ExecuteMeanReversionExit 执行均值回归退出
func (e *Executor) ExecuteMeanReversionExit(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *MeanReversionExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[MeanReversionExecutor] 执行均值回归退出: %s", symbol)

	result := &execution.ExecutionResult{
		Action:    "exit",
		Reason:    "均值回归策略：达到退出条件",
		Symbol:    symbol,
		Timestamp: time.Now(),
		RiskLevel: 0.2, // 退出操作风险较低
	}

	// 执行退出订单（平仓）
	if e.dependencies.OrderManager != nil {
		// 这里应该计算需要平仓的数量
		// 暂时使用固定数量
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "sell", 100.0, marketData.Price)
		if err != nil {
			log.Printf("[MeanReversionExecutor] 退出订单失败: %v", err)
			result.Reason = fmt.Sprintf("退出订单失败: %v", err)
		} else {
			result.OrderID = orderID
			log.Printf("[MeanReversionExecutor] 退出订单成功: %s", orderID)
		}
	}

	return result, nil
}

// calculateMeanReversionPosition 计算均值回归仓位大小
func (e *Executor) calculateMeanReversionPosition(marketData *execution.MarketData, config *MeanReversionExecutionConfig) float64 {
	// 基础仓位大小
	baseSize := 100.0

	// 根据波动率调整仓位
	volatility := math.Abs(marketData.Change24h)
	if volatility > 0.05 {
		// 高波动，减少仓位
		baseSize *= 0.7
	} else if volatility < 0.01 {
		// 低波动，增加仓位
		baseSize *= 1.3
	}

	// 根据市值调整仓位
	if marketData.MarketCap > 10000000000 { // 市值大于100亿
		baseSize *= 1.2 // 大市值币种可以增加仓位
	} else if marketData.MarketCap < 1000000000 { // 市值小于10亿
		baseSize *= 0.8 // 小市值币种减少仓位
	}

	// 限制最大仓位
	if config.MaxPositionSize > 0 && baseSize > config.MaxPositionSize {
		baseSize = config.MaxPositionSize
	}

	return baseSize
}
