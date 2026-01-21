package execution

import (
	"analysis/internal/server/strategy/shared/execution"
	"context"
	"fmt"
	"log"
	"time"
)

// ============================================================================
// 传统策略执行器实现
// ============================================================================

// Executor 传统策略执行器
type Executor struct {
	dependencies *ExecutionDependencies
}

// NewExecutor 创建传统策略执行器
func NewExecutor(deps *ExecutionDependencies) *Executor {
	return &Executor{
		dependencies: deps,
	}
}

// GetStrategyType 获取策略类型
func (e *Executor) GetStrategyType() string {
	return "traditional"
}

// IsEnabled 检查策略是否启用
func (e *Executor) IsEnabled(config interface{}) bool {
	traditionalConfig, ok := config.(*TraditionalExecutionConfig)
	if !ok {
		return false
	}
	return traditionalConfig.Enabled &&
		(traditionalConfig.ShortOnGainers || traditionalConfig.LongOnSmallGainers || traditionalConfig.FuturesPriceShortStrategyEnabled)
}

// ValidateExecution 预执行验证
func (e *Executor) ValidateExecution(symbol string, marketData *execution.MarketData, config interface{}) error {
	traditionalConfig, ok := config.(*TraditionalExecutionConfig)
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

	// 交易类型验证
	if traditionalConfig.TradingType != "" {
		switch traditionalConfig.TradingType {
		case "spot":
			if !marketData.HasSpot {
				return fmt.Errorf("交易类型设置为现货，但%s没有现货交易", symbol)
			}
		case "futures":
			if !marketData.HasFutures {
				return fmt.Errorf("交易类型设置为合约，但%s没有合约交易", symbol)
			}
		case "both":
			if !marketData.HasSpot && !marketData.HasFutures {
				return fmt.Errorf("交易类型设置为两者皆可，但%s既没有现货也没有合约交易", symbol)
			}
		default:
			return fmt.Errorf("无效的交易类型: %s", traditionalConfig.TradingType)
		}
	}

	// 合约涨幅排名过滤验证
	if traditionalConfig.FuturesPriceRankFilterEnabled {
		// 只有当选择合约交易或两者皆可时才进行排名验证
		if traditionalConfig.TradingType == "futures" || traditionalConfig.TradingType == "both" {
			if marketData.GainersRank > traditionalConfig.MaxFuturesPriceRank {
				return fmt.Errorf("合约涨幅排名%d超出限制%d", marketData.GainersRank, traditionalConfig.MaxFuturesPriceRank)
			}
		}
	}

	// 策略特定验证 - 增强版
	if traditionalConfig.ShortOnGainers {
		// 检查不开空条件
		if traditionalConfig.NoShortBelowMarketCap && marketData.MarketCap < traditionalConfig.MarketCapLimitShort {
			return fmt.Errorf("市值%.1f万低于%.1f万不开空", marketData.MarketCap/10000, traditionalConfig.MarketCapLimitShort/10000)
		}

		// 检查开空条件（三个条件都要满足）
		shortCondition2 := marketData.GainersRank <= traditionalConfig.GainersRankLimit
		shortCondition3 := marketData.MarketCap >= traditionalConfig.MarketCapLimitShort

		if !shortCondition2 {
			return fmt.Errorf("涨幅排名(%d)超出限制(%d)", marketData.GainersRank, traditionalConfig.GainersRankLimit)
		}
		if !shortCondition3 {
			return fmt.Errorf("市值(%.0f)低于开空限制(%.0f)", marketData.MarketCap, traditionalConfig.MarketCapLimitShort)
		}
	}

	if traditionalConfig.LongOnSmallGainers {
		// 检查开多条件（市值低于阈值且涨幅排名靠前）
		longCondition2 := marketData.MarketCap < traditionalConfig.MarketCapLimitLong
		longCondition3 := marketData.GainersRank <= traditionalConfig.LongGainersRankLimit

		if !longCondition2 {
			return fmt.Errorf("市值(%.0f)高于开多限制(%.0f)", marketData.MarketCap, traditionalConfig.MarketCapLimitLong)
		}
		if !longCondition3 {
			return fmt.Errorf("涨幅排名(%d)超出开多限制(%d)", marketData.GainersRank, traditionalConfig.LongGainersRankLimit)
		}
	}

	// 新增：合约涨幅开空策略验证
	if traditionalConfig.FuturesPriceShortStrategyEnabled {
		// 检查涨幅排名条件
		if marketData.GainersRank > traditionalConfig.FuturesPriceShortMaxRank {
			return fmt.Errorf("涨幅排名(%d)超出限制(%d)", marketData.GainersRank, traditionalConfig.FuturesPriceShortMaxRank)
		}

		// 检查是否有合约交易（必须是期货交易）
		if !marketData.HasFutures {
			return fmt.Errorf("合约涨幅开空策略需要合约交易，但%s没有合约交易", symbol)
		}

		// 注意：资金费率验证将在执行时通过validator进行，这里只做基础检查
	}

	// 风险管理验证
	if e.dependencies.RiskManager != nil {
		positionSize := 1.0 // 默认仓位大小，可以根据配置调整
		if err := e.dependencies.RiskManager.CheckPositionLimits(symbol, positionSize); err != nil {
			return fmt.Errorf("风险检查失败: %w", err)
		}
	}

	return nil
}

// Execute 执行策略
func (e *Executor) Execute(ctx context.Context, symbol string, marketData *execution.MarketData,
	config interface{}, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	traditionalConfig, ok := config.(*TraditionalExecutionConfig)
	if !ok {
		return nil, fmt.Errorf("无效的配置类型: %T", config)
	}

	log.Printf("[TraditionalExecutor] 开始执行策略: %s, 用户: %d", symbol, execContext.UserID)

	// 预执行验证
	if err := e.ValidateExecution(symbol, marketData, config); err != nil {
		log.Printf("[TraditionalExecutor] 验证失败: %v", err)
		return &execution.ExecutionResult{
			Action:    "skip",
			Reason:    fmt.Sprintf("验证失败: %v", err),
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 根据交易类型记录执行信息
	log.Printf("[TraditionalExecutor] 交易类型: %s", traditionalConfig.TradingType)
	switch traditionalConfig.TradingType {
	case "spot":
		log.Printf("[TraditionalExecutor] 将在现货市场执行交易")
	case "futures":
		log.Printf("[TraditionalExecutor] 将在合约市场执行交易")
	case "both":
		log.Printf("[TraditionalExecutor] 将根据市场条件选择现货或合约执行交易")
	default:
		log.Printf("[TraditionalExecutor] 未指定交易类型，使用默认策略")
	}

	// 记录合约涨幅排名过滤信息
	if traditionalConfig.FuturesPriceRankFilterEnabled {
		log.Printf("[TraditionalExecutor] 合约涨幅排名过滤已启用，限制前%d名，当前排名%d",
			traditionalConfig.MaxFuturesPriceRank, marketData.GainersRank)
	} else {
		log.Printf("[TraditionalExecutor] 合约涨幅排名过滤未启用")
	}

	// 记录详细的条件检查日志（类似原有实现）
	marketCap := marketData.MarketCap
	gainersRank := marketData.GainersRank

	log.Printf("[TraditionalStrategy] %s 条件检查 - ShortOnGainers:%v, LongOnSmallGainers:%v",
		symbol, traditionalConfig.ShortOnGainers, traditionalConfig.LongOnSmallGainers)
	log.Printf("[TraditionalStrategy] %s 市场数据 - rank:%d, marketCap:%.1f万",
		symbol, gainersRank, marketCap/10000)

	// 检查开空条件
	if traditionalConfig.ShortOnGainers {
		shortCondition2 := gainersRank <= traditionalConfig.GainersRankLimit
		shortCondition3 := marketCap >= traditionalConfig.MarketCapLimitShort

		log.Printf("[TraditionalStrategy] %s 开空条件检查 - rank<=%d:%v (%d<=%d), marketCap>=%.0f:%v (%.0f>=%.0f)",
			symbol, traditionalConfig.GainersRankLimit, shortCondition2, gainersRank, traditionalConfig.GainersRankLimit,
			traditionalConfig.MarketCapLimitShort, shortCondition3, marketCap, traditionalConfig.MarketCapLimitShort)

		if shortCondition2 && shortCondition3 {
			return e.ExecuteShortOnGainers(ctx, symbol, marketData, traditionalConfig, execContext)
		}
	}

	// 检查开多条件
	if traditionalConfig.LongOnSmallGainers {
		longCondition2 := marketCap < traditionalConfig.MarketCapLimitLong
		longCondition3 := gainersRank <= traditionalConfig.LongGainersRankLimit

		log.Printf("[TraditionalStrategy] %s 开多条件检查 - marketCap<%.0f:%v (%.0f<%.0f), rank<=%d:%v (%d<=%d)",
			symbol, traditionalConfig.MarketCapLimitLong, longCondition2, marketCap, traditionalConfig.MarketCapLimitLong,
			traditionalConfig.LongGainersRankLimit, longCondition3, gainersRank, traditionalConfig.LongGainersRankLimit)

		if longCondition2 && longCondition3 {
			return e.ExecuteLongOnSmallGainers(ctx, symbol, marketData, traditionalConfig, execContext)
		}
	}

	// 新增：检查合约涨幅开空策略条件
	if traditionalConfig.FuturesPriceShortStrategyEnabled {
		futuresShortCondition1 := gainersRank <= traditionalConfig.FuturesPriceShortMaxRank
		futuresShortCondition2 := marketData.HasFutures // 必须有合约交易

		log.Printf("[TraditionalStrategy] %s 合约涨幅开空条件检查 - rank<=%d:%v (%d<=%d), hasFutures:%v",
			symbol, traditionalConfig.FuturesPriceShortMaxRank, futuresShortCondition1, gainersRank, traditionalConfig.FuturesPriceShortMaxRank, futuresShortCondition2)

		if futuresShortCondition1 && futuresShortCondition2 {
			return e.ExecuteFuturesPriceShort(ctx, symbol, marketData, traditionalConfig, execContext)
		}
	}

	log.Printf("[TraditionalStrategy] %s 不符合任何传统策略条件", symbol)
	return &execution.ExecutionResult{
		Action:    "no_op",
		Reason:    "不符合传统策略条件",
		Symbol:    symbol,
		Timestamp: time.Now(),
	}, nil
}

// ExecuteShortOnGainers 执行涨幅开空策略
func (e *Executor) ExecuteShortOnGainers(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *TraditionalExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[TraditionalExecutor] 执行涨幅开空: %s, 排名: %d, 市值: %.0f",
		symbol, marketData.GainersRank, marketData.MarketCap)

	// 计算执行参数
	multiplier := config.ShortMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// 如果启用了杠杆，使用杠杆倍数覆盖基础倍数
	if config.EnableLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	// 计算风险管理参数
	var stopLossPrice, takeProfitPrice float64
	if e.dependencies.RiskManager != nil {
		stopLossPrice = e.dependencies.RiskManager.CalculateStopLoss(marketData.Price, 0.05)     // 5%止损
		takeProfitPrice = e.dependencies.RiskManager.CalculateTakeProfit(marketData.Price, 0.10) // 10%止盈
	}

	// 模拟执行（实际实现中会调用订单管理器）
	result := &execution.ExecutionResult{
		Action:          "sell",
		Reason:          fmt.Sprintf("涨幅排名第%d位，符合开空条件", marketData.GainersRank),
		Multiplier:      multiplier,
		Symbol:          symbol,
		Timestamp:       time.Now(),
		StopLossPrice:   stopLossPrice,
		TakeProfitPrice: takeProfitPrice,
		MaxPositionSize: config.MaxPositionSize,
		MaxHoldHours:    config.MaxHoldHours,
		RiskLevel:       0.6, // 中等风险
	}

	// 如果有订单管理器，执行实际下单
	if e.dependencies.OrderManager != nil {
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "sell", 100.0, marketData.Price)
		if err != nil {
			log.Printf("[TraditionalExecutor] 下单失败: %v", err)
			result.Action = "skip"
			result.Reason = fmt.Sprintf("下单失败: %v", err)
		} else {
			result.OrderID = orderID
			log.Printf("[TraditionalExecutor] 成功下单: %s", orderID)
		}
	}

	return result, nil
}

// ExecuteLongOnSmallGainers 执行小幅上涨开多策略
func (e *Executor) ExecuteLongOnSmallGainers(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *TraditionalExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[TraditionalExecutor] 执行小幅上涨开多: %s, 排名: %d, 市值: %.0f",
		symbol, marketData.GainersRank, marketData.MarketCap)

	// 计算执行参数
	multiplier := config.LongMultiplier
	if multiplier <= 0 {
		multiplier = 1.0
	}

	// 如果启用了杠杆，使用杠杆倍数覆盖基础倍数
	if config.EnableLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	// 计算风险管理参数
	var stopLossPrice, takeProfitPrice float64
	if e.dependencies.RiskManager != nil {
		stopLossPrice = e.dependencies.RiskManager.CalculateStopLoss(marketData.Price, 0.03)     // 3%止损
		takeProfitPrice = e.dependencies.RiskManager.CalculateTakeProfit(marketData.Price, 0.15) // 15%止盈
	}

	// 模拟执行（实际实现中会调用订单管理器）
	result := &execution.ExecutionResult{
		Action:          "buy",
		Reason:          fmt.Sprintf("小幅上涨排名第%d位，符合开多条件", marketData.GainersRank),
		Multiplier:      multiplier,
		Symbol:          symbol,
		Timestamp:       time.Now(),
		StopLossPrice:   stopLossPrice,
		TakeProfitPrice: takeProfitPrice,
		MaxPositionSize: config.MaxPositionSize,
		MaxHoldHours:    config.MaxHoldHours,
		RiskLevel:       0.4, // 较低风险
	}

	// 如果有订单管理器，执行实际下单
	if e.dependencies.OrderManager != nil {
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "buy", 100.0, marketData.Price)
		if err != nil {
			log.Printf("[TraditionalExecutor] 下单失败: %v", err)
			result.Action = "skip"
			result.Reason = fmt.Sprintf("下单失败: %v", err)
		} else {
			result.OrderID = orderID
			log.Printf("[TraditionalExecutor] 成功下单: %s", orderID)
		}
	}

	return result, nil
}

// ExecuteFuturesPriceShort 执行合约涨幅开空策略
func (e *Executor) ExecuteFuturesPriceShort(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *TraditionalExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[TraditionalExecutor] 执行合约涨幅开空: %s, 市值: %.0f万, 排名: %d, 杠杆: %.1fx",
		symbol, marketData.MarketCap, marketData.GainersRank, config.FuturesPriceShortLeverage)

	// 使用配置中的杠杆倍数
	multiplier := config.FuturesPriceShortLeverage
	if multiplier <= 0 {
		multiplier = 3.0 // 默认3倍杠杆
	}

	// 计算风险管理参数（合约交易风险更高）
	var stopLossPrice, takeProfitPrice float64
	if e.dependencies.RiskManager != nil {
		// 检查是否启用保证金损失止损
		if config.EnableMarginLossStopLoss && config.MarginLossStopLossPercent > 0 {
			// 使用保证金损失止损
			marginStopPrice, err := e.dependencies.RiskManager.CalculateMarginStopLoss(symbol, config.MarginLossStopLossPercent)
			if err != nil {
				log.Printf("[TraditionalExecutor] 计算保证金止损价格失败，使用价格百分比止损: %v", err)
				stopLossPrice = e.dependencies.RiskManager.CalculateStopLoss(marketData.Price, 0.03) // 3%止损（合约更严格）
			} else {
				stopLossPrice = marginStopPrice
				log.Printf("[TraditionalExecutor] 使用保证金亏损%.1f%%止损: %.4f",
					config.MarginLossStopLossPercent, stopLossPrice)
			}
		} else {
			// 使用传统的价格百分比止损
			stopLossPrice = e.dependencies.RiskManager.CalculateStopLoss(marketData.Price, 0.03) // 3%止损（合约更严格）
		}

		// 检查是否启用保证金盈利止盈
		if config.EnableMarginProfitTakeProfit && config.MarginProfitTakeProfitPercent > 0 {
			marginTakeProfitPrice, err := e.dependencies.RiskManager.CalculateMarginTakeProfit(symbol, config.MarginProfitTakeProfitPercent)
			if err != nil {
				log.Printf("[TraditionalExecutor] 计算保证金止盈价格失败，使用价格百分比止盈: %v", err)
				takeProfitPrice = e.dependencies.RiskManager.CalculateTakeProfit(marketData.Price, 0.05) // 5%止盈
			} else {
				takeProfitPrice = marginTakeProfitPrice
				log.Printf("[TraditionalExecutor] 使用保证金盈利%.1f%%止盈: %.4f",
					config.MarginProfitTakeProfitPercent, takeProfitPrice)
			}
		} else {
			takeProfitPrice = e.dependencies.RiskManager.CalculateTakeProfit(marketData.Price, 0.05) // 5%止盈
		}
	}

	// 模拟执行（实际实现中会调用订单管理器）
	result := &execution.ExecutionResult{
		Action:          "short",
		Reason:          fmt.Sprintf("合约涨幅排名第%d位，资金费率符合条件，直接开空", marketData.GainersRank),
		Multiplier:      multiplier,
		Symbol:          symbol,
		Timestamp:       time.Now(),
		StopLossPrice:   stopLossPrice,
		TakeProfitPrice: takeProfitPrice,
		MaxPositionSize: config.MaxPositionSize,
		MaxHoldHours:    config.MaxHoldHours,
		RiskLevel:       0.8, // 高风险（合约杠杆交易）
	}

	// 记录执行详情
	log.Printf("[TraditionalExecutor] 合约开空执行详情 - 杠杆: %.1fx, 风险等级: %.1f",
		multiplier, result.RiskLevel)

	// 如果有订单管理器，执行实际下单
	if e.dependencies.OrderManager != nil {
		orderID, err := e.dependencies.OrderManager.PlaceOrder(symbol, "short", 100.0, marketData.Price)
		if err != nil {
			log.Printf("[TraditionalExecutor] 合约开空下单失败: %v", err)
			result.Action = "skip"
			result.Reason = fmt.Sprintf("合约开空下单失败: %v", err)
		} else {
			result.OrderID = orderID
			log.Printf("[TraditionalExecutor] 合约开空成功下单: %s", orderID)
		}
	}

	return result, nil
}
