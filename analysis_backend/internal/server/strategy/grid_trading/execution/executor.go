package execution

import (
	"context"
	"fmt"
	"log"
	"math"
	"analysis/internal/server/strategy/shared/execution"
	"time"
)

// ============================================================================
// 网格交易策略执行器实现
// ============================================================================

// Executor 网格交易策略执行器
type Executor struct {
	dependencies *ExecutionDependencies
}

// NewExecutor 创建网格交易策略执行器
func NewExecutor(deps *ExecutionDependencies) *Executor {
	return &Executor{
		dependencies: deps,
	}
}

// GetStrategyType 获取策略类型
func (e *Executor) GetStrategyType() string {
	return "grid_trading"
}

// IsEnabled 检查策略是否启用
func (e *Executor) IsEnabled(config interface{}) bool {
	gridConfig, ok := config.(*GridTradingExecutionConfig)
	if !ok {
		return false
	}
	return gridConfig.Enabled && gridConfig.GridTradingEnabled
}

// ValidateExecution 预执行验证
func (e *Executor) ValidateExecution(symbol string, marketData *execution.MarketData, config interface{}) error {
	gridConfig, ok := config.(*GridTradingExecutionConfig)
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

	// 网格参数验证
	if gridConfig.GridUpperPrice <= 0 || gridConfig.GridLowerPrice <= 0 {
		return fmt.Errorf("网格价格上下限必须大于0")
	}

	if gridConfig.GridUpperPrice <= gridConfig.GridLowerPrice {
		return fmt.Errorf("网格上限价格必须大于下限价格")
	}

	if gridConfig.GridLevels <= 0 {
		return fmt.Errorf("网格层数必须大于0")
	}

	if gridConfig.GridInvestmentAmount <= 0 {
		return fmt.Errorf("网格投资金额必须大于0")
	}

	// 验证当前价格在网格范围内
	if marketData.Price < gridConfig.GridLowerPrice || marketData.Price > gridConfig.GridUpperPrice {
		return fmt.Errorf("当前价格(%.4f)超出网格范围[%.4f, %.4f]",
			marketData.Price, gridConfig.GridLowerPrice, gridConfig.GridUpperPrice)
	}

	return nil
}

// Execute 执行策略
func (e *Executor) Execute(ctx context.Context, symbol string, marketData *execution.MarketData,
	config interface{}, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	gridConfig, ok := config.(*GridTradingExecutionConfig)
	if !ok {
		return nil, fmt.Errorf("无效的配置类型: %T", config)
	}

	log.Printf("[GridTradingExecutor] 开始执行网格交易策略: %s, 用户: %d", symbol, execContext.UserID)

	// 预执行验证
	if err := e.ValidateExecution(symbol, marketData, config); err != nil {
		log.Printf("[GridTradingExecutor] 验证失败: %v", err)
		return &execution.ExecutionResult{
			Action:    "skip",
			Reason:    fmt.Sprintf("验证失败: %v", err),
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 分析当前网格状态
	gridAnalysis := e.analyzeGridStatus(marketData, gridConfig)

	// 根据分析结果决定执行动作
	switch gridAnalysis.Action {
	case "setup":
		return e.ExecuteGridSetup(ctx, symbol, marketData, gridConfig, execContext)
	case "rebalance":
		return e.ExecuteGridRebalance(ctx, symbol, marketData, gridConfig, execContext)
	case "exit":
		return e.ExecuteGridExit(ctx, symbol, marketData, gridConfig, execContext)
	default:
		return &execution.ExecutionResult{
			Action:    "no_op",
			Reason:    "网格状态正常，无需操作",
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}
}

// GridAnalysis 网格分析结果
type GridAnalysis struct {
	Action     string  `json:"action"`      // "setup", "rebalance", "exit", "hold"
	Reason     string  `json:"reason"`      // 分析理由
	TargetLevel int    `json:"target_level"` // 目标网格层级
	Confidence float64 `json:"confidence"`  // 置信度 0-1
}

// analyzeGridStatus 分析网格状态
func (e *Executor) analyzeGridStatus(marketData *execution.MarketData, config *GridTradingExecutionConfig) *GridAnalysis {
	currentPrice := marketData.Price

	// 计算当前价格在网格中的位置
	gridRange := config.GridUpperPrice - config.GridLowerPrice
	relativePosition := (currentPrice - config.GridLowerPrice) / gridRange

	// 计算当前应该在哪个网格层级
	currentLevel := int(math.Floor(relativePosition * float64(config.GridLevels)))

	// 边界检查
	if currentLevel < 0 {
		currentLevel = 0
	} else if currentLevel >= config.GridLevels {
		currentLevel = config.GridLevels - 1
	}

	// 检查是否需要重新平衡
	levelSize := gridRange / float64(config.GridLevels)
	idealPrice := config.GridLowerPrice + float64(currentLevel)*levelSize
	priceDeviation := math.Abs(currentPrice - idealPrice) / idealPrice

	// 决定行动
	if priceDeviation > config.GridRebalanceThreshold {
		return &GridAnalysis{
			Action:      "rebalance",
			Reason:      fmt.Sprintf("价格偏离度%.2f%%超过阈值%.2f%%", priceDeviation*100, config.GridRebalanceThreshold*100),
			TargetLevel: currentLevel,
			Confidence:  0.8,
		}
	}

	// 检查退出条件
	if e.shouldExitGrid(marketData, config) {
		return &GridAnalysis{
			Action:     "exit",
			Reason:     "满足网格退出条件",
			Confidence: 0.9,
		}
	}

	return &GridAnalysis{
		Action:     "hold",
		Reason:     "网格运行正常",
		Confidence: 1.0,
	}
}

// shouldExitGrid 检查是否应该退出网格
func (e *Executor) shouldExitGrid(marketData *execution.MarketData, config *GridTradingExecutionConfig) bool {
	switch config.GridExitCondition {
	case "PROFIT_TARGET":
		// 检查利润目标（这里需要计算实际利润）
		return false // 暂时返回false
	case "LOSS_LIMIT":
		// 检查损失限制
		if config.GridStopLossEnabled && math.Abs(marketData.Change24h) > config.GridStopLossPercent {
			return true
		}
	case "TIME_BASED":
		// 时间-based退出（需要持仓时间记录）
		return false // 暂时返回false
	}
	return false
}

// ExecuteGridSetup 执行网格设置
func (e *Executor) ExecuteGridSetup(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *GridTradingExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[GridTradingExecutor] 执行网格设置: %s", symbol)

	// 计算网格参数
	gridLevels := e.calculateGridLevels(marketData, config)

	// 计算执行参数
	multiplier := 1.0 // 网格策略默认1倍
	if config.AllowLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	result := &execution.ExecutionResult{
		Action:     "grid_setup",
		Reason:     fmt.Sprintf("设置%d层网格策略，范围[%.4f, %.4f]", len(gridLevels), config.GridLowerPrice, config.GridUpperPrice),
		Multiplier: multiplier,
		Symbol:     symbol,
		Timestamp:  time.Now(),
		RiskLevel:  0.3, // 网格策略风险相对较低
	}

	// 执行网格订单设置（模拟）
	if e.dependencies.OrderManager != nil {
		for _, level := range gridLevels {
			orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, level.Side, level.Quantity, level.Price)
			if err != nil {
				log.Printf("[GridTradingExecutor] 网格订单设置失败: %v", err)
				continue
			}
			log.Printf("[GridTradingExecutor] 网格订单设置成功: %s at %.4f", orderID, level.Price)
		}
	}

	return result, nil
}

// ExecuteGridRebalance 执行网格再平衡
func (e *Executor) ExecuteGridRebalance(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *GridTradingExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[GridTradingExecutor] 执行网格再平衡: %s", symbol)

	// 计算需要调整的仓位
	rebalanceOrders := e.calculateRebalanceOrders(marketData, config)

	// 计算执行参数
	multiplier := 1.0 // 网格策略默认1倍
	if config.AllowLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	result := &execution.ExecutionResult{
		Action:     "grid_rebalance",
		Reason:     fmt.Sprintf("网格再平衡，调整%d个仓位", len(rebalanceOrders)),
		Multiplier: multiplier,
		Symbol:     symbol,
		Timestamp:  time.Now(),
		RiskLevel:  0.4,
	}

	// 执行再平衡订单
	if e.dependencies.OrderManager != nil {
		for _, order := range rebalanceOrders {
			orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, order.Side, order.Quantity, order.Price)
			if err != nil {
				log.Printf("[GridTradingExecutor] 再平衡订单失败: %v", err)
				continue
			}
			log.Printf("[GridTradingExecutor] 再平衡订单成功: %s", orderID)
		}
	}

	return result, nil
}

// ExecuteGridExit 执行网格退出
func (e *Executor) ExecuteGridExit(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *GridTradingExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[GridTradingExecutor] 执行网格退出: %s", symbol)

	// 计算执行参数
	multiplier := 1.0 // 网格策略默认1倍
	if config.AllowLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	result := &execution.ExecutionResult{
		Action:     "grid_exit",
		Reason:     "根据退出条件关闭网格策略",
		Multiplier: multiplier,
		Symbol:     symbol,
		Timestamp:  time.Now(),
		RiskLevel:  0.2, // 退出操作风险较低
	}

	// 执行退出订单（平掉所有网格仓位）
	if e.dependencies.OrderManager != nil {
		// 这里应该获取所有网格仓位并平仓
		// 暂时模拟一个退出订单
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "sell", 100.0, marketData.Price)
		if err != nil {
			log.Printf("[GridTradingExecutor] 网格退出订单失败: %v", err)
			result.Reason = fmt.Sprintf("网格退出失败: %v", err)
		} else {
			log.Printf("[GridTradingExecutor] 网格退出订单成功: %s", orderID)
		}
	}

	return result, nil
}

// calculateGridLevels 计算网格层级
func (e *Executor) calculateGridLevels(marketData *execution.MarketData, config *GridTradingExecutionConfig) []GridPosition {
	var levels []GridPosition

	priceRange := config.GridUpperPrice - config.GridLowerPrice
	levelSize := priceRange / float64(config.GridLevels)

	// 为每个网格层级创建持仓计划
	for i := 0; i < config.GridLevels; i++ {
		levelPrice := config.GridLowerPrice + float64(i)*levelSize

		// 根据价格位置决定买卖方向
		var side string
		var quantity float64

		if levelPrice < marketData.Price {
			side = "buy"  // 在当前价格下方设置买单
			quantity = config.GridInvestmentAmount / levelPrice * 0.1 // 10%仓位
		} else {
			side = "sell" // 在当前价格上方设置卖单
			quantity = config.GridInvestmentAmount / levelPrice * 0.1 // 10%仓位
		}

		level := GridPosition{
			Symbol:    marketData.Symbol,
			Level:     i,
			Price:     levelPrice,
			Quantity:  quantity,
			Side:      side,
			Timestamp: time.Now().Unix(),
		}

		levels = append(levels, level)
	}

	return levels
}

// calculateRebalanceOrders 计算再平衡订单
func (e *Executor) calculateRebalanceOrders(marketData *execution.MarketData, config *GridTradingExecutionConfig) []GridPosition {
	// 简化的再平衡逻辑
	// 实际应该基于当前的网格持仓状态计算需要调整的订单

	return []GridPosition{
		{
			Symbol:    marketData.Symbol,
			Level:     0,
			Price:     marketData.Price * 0.98, // 在当前价格下方2%设置买单
			Quantity:  50.0,
			Side:      "buy",
			Timestamp: time.Now().Unix(),
		},
	}
}