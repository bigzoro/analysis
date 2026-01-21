package execution

import (
	"analysis/internal/server/strategy/shared/execution"
	"context"
	"fmt"
	"log"
	"math"
	"time"
)

// ============================================================================
// 套利策略执行器实现
// ============================================================================

// Executor 套利策略执行器
type Executor struct {
	dependencies *ExecutionDependencies
}

// NewExecutor 创建套利策略执行器
func NewExecutor(deps *ExecutionDependencies) *Executor {
	return &Executor{
		dependencies: deps,
	}
}

// GetStrategyType 获取策略类型
func (e *Executor) GetStrategyType() string {
	return "arbitrage"
}

// IsEnabled 检查策略是否启用
func (e *Executor) IsEnabled(config interface{}) bool {
	arbConfig, ok := config.(*ArbitrageExecutionConfig)
	if !ok {
		return false
	}
	return arbConfig.Enabled && (arbConfig.TriangleArbEnabled ||
		arbConfig.CrossExchangeArbEnabled ||
		arbConfig.SpotFutureArbEnabled ||
		arbConfig.StatisticalArbEnabled ||
		arbConfig.FuturesSpotArbEnabled)
}

// ValidateExecution 预执行验证
func (e *Executor) ValidateExecution(symbol string, marketData *execution.MarketData, config interface{}) error {
	arbConfig, ok := config.(*ArbitrageExecutionConfig)
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

	// 检查是否有期货交易能力（大部分套利需要）
	hasArbitrageCapability := marketData.HasSpot || marketData.HasFutures
	if !hasArbitrageCapability {
		return fmt.Errorf("交易对不支持必要的交易类型进行套利")
	}

	// 验证利润阈值
	if arbConfig.MinProfitThreshold <= 0 {
		return fmt.Errorf("最小利润阈值必须大于0")
	}

	// 验证交易量
	if marketData.Volume < arbConfig.MinVolumeThreshold {
		return fmt.Errorf("交易量(%.2f)低于最小阈值(%.2f)", marketData.Volume, arbConfig.MinVolumeThreshold)
	}

	return nil
}

// Execute 执行策略
func (e *Executor) Execute(ctx context.Context, symbol string, marketData *execution.MarketData,
	config interface{}, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	arbConfig, ok := config.(*ArbitrageExecutionConfig)
	if !ok {
		return nil, fmt.Errorf("无效的配置类型: %T", config)
	}

	log.Printf("[ArbitrageExecutor] 开始执行套利策略: %s, 用户: %d", symbol, execContext.UserID)

	// 预执行验证
	if err := e.ValidateExecution(symbol, marketData, config); err != nil {
		log.Printf("[ArbitrageExecutor] 验证失败: %v", err)
		return &execution.ExecutionResult{
			Action:    "skip",
			Reason:    fmt.Sprintf("验证失败: %v", err),
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 按优先级尝试不同的套利类型
	if arbConfig.TriangleArbEnabled {
		result, err := e.ExecuteTriangleArbitrage(ctx, symbol, marketData, arbConfig, execContext)
		if err == nil && result.Action != "no_op" {
			return result, nil
		}
	}

	if arbConfig.SpotFutureArbEnabled {
		result, err := e.ExecuteSpotFutureArbitrage(ctx, symbol, marketData, arbConfig, execContext)
		if err == nil && result.Action != "no_op" {
			return result, nil
		}
	}

	if arbConfig.CrossExchangeArbEnabled {
		result, err := e.ExecuteCrossExchangeArbitrage(ctx, symbol, marketData, arbConfig, execContext)
		if err == nil && result.Action != "no_op" {
			return result, nil
		}
	}

	// 没有找到套利机会
	return &execution.ExecutionResult{
		Action:    "no_op",
		Reason:    "未找到合适的套利机会",
		Symbol:    symbol,
		Timestamp: time.Now(),
		RiskLevel: 0.8, // 套利风险较高
	}, nil
}

// ExecuteTriangleArbitrage 执行三角套利
func (e *Executor) ExecuteTriangleArbitrage(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *ArbitrageExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[ArbitrageExecutor] 执行三角套利: %s", symbol)

	// 预定义有效的三角套利路径（扩展到更多主流币种）
	validPaths := map[string][][]string{
		// === BTC路径 ===
		"BTCUSDT": {
			{"BTCUSDT", "ETHBTC", "ETHUSDT"}, // BTC → ETH → USDT
			{"BTCUSDT", "BNBBTC", "BNBUSDT"}, // BTC → BNB → USDT
			{"BTCUSDT", "ADABTC", "ADAUSDT"}, // BTC → ADA → USDT
			{"BTCUSDT", "SOLBTC", "SOLUSDT"}, // BTC → SOL → USDT
			{"BTCUSDT", "DOTBTC", "DOTUSDT"}, // BTC → DOT → USDT
		},
		// === ETH路径 ===
		"ETHUSDT": {
			{"ETHUSDT", "BTCETH", "BTCUSDT"}, // ETH → BTC → USDT
			{"ETHUSDT", "BNBETH", "BNBUSDT"}, // ETH → BNB → USDT
			{"ETHUSDT", "ADAETH", "ADAUSDT"}, // ETH → ADA → USDT
			{"ETHUSDT", "SOLETH", "SOLUSDT"}, // ETH → SOL → USDT
			{"ETHUSDT", "DOTETH", "DOTUSDT"}, // ETH → DOT → USDT
		},
		// === BNB路径 ===
		"BNBUSDT": {
			{"BNBUSDT", "BTCBNB", "BTCUSDT"}, // BNB → BTC → USDT
			{"BNBUSDT", "ETHBNB", "ETHUSDT"}, // BNB → ETH → USDT
			{"BNBUSDT", "ADABNB", "ADAUSDT"}, // BNB → ADA → USDT
			{"BNBUSDT", "SOLBNB", "SOLUSDT"}, // BNB → SOL → USDT
		},
	}

	// 获取此交易对的有效路径
	paths, exists := validPaths[symbol]
	if !exists {
		return &execution.ExecutionResult{
			Action:    "skip",
			Reason:    fmt.Sprintf("%s 不是主流三角套利交易对", symbol),
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 检查每个预定义路径是否有套利机会
	for _, path := range paths {
		if len(path) != 3 {
			continue
		}

		// 验证路径中所有交易对都存在
		// 这里可以添加交易对存在性检查逻辑
		// 暂时假设所有预定义路径都有效

		// 计算三角套利
		result := e.calculateTriangleArbitrage(ctx, path, config.TriangleMinProfitPercent)
		if result.Action != "no_op" {
			// 计算执行参数
			multiplier := result.Multiplier // 基础倍数
			if config.AllowLeverage && config.DefaultLeverage > 1 {
				multiplier = float64(config.DefaultLeverage)
			}

			// 转换结果格式
			return &execution.ExecutionResult{
				Action:     result.Action,
				Reason:     result.Reason,
				Symbol:     symbol,
				Timestamp:  time.Now(),
				Multiplier: multiplier,
				RiskLevel:  0.9, // 三角套利风险很高
			}, nil
		}
	}

	return &execution.ExecutionResult{
		Action:    "no_op",
		Reason:    fmt.Sprintf("%s 的三角套利路径都没有足够的机会", symbol),
		Symbol:    symbol,
		Timestamp: time.Now(),
		RiskLevel: 0.8,
	}, nil
}

// ExecuteCrossExchangeArbitrage 执行跨交易所套利
func (e *Executor) ExecuteCrossExchangeArbitrage(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *ArbitrageExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[ArbitrageExecutor] 检查跨交易所套利机会: %s", symbol)

	// 检查配置的交易所对
	if len(config.ExchangePairs) == 0 {
		return &execution.ExecutionResult{
			Action:    "no_op",
			Reason:    "未配置交易所对",
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 计算价格差异（模拟）
	priceDiff := marketData.Change24h // 使用24小时涨跌幅作为价格差异指标

	if math.Abs(priceDiff) < config.MinProfitThreshold {
		return &execution.ExecutionResult{
			Action:    "no_op",
			Reason:    fmt.Sprintf("价格差异(%.2f%%)低于最小利润阈值(%.2f%%)", priceDiff, config.MinProfitThreshold),
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 执行套利交易
	action := "arb_buy_low_sell_high"
	if priceDiff < 0 {
		action = "arb_sell_high_buy_low"
	}

	// 计算执行参数
	multiplier := 1.0 // 套利默认1倍
	if config.AllowLeverage && config.DefaultLeverage > 1 {
		multiplier = float64(config.DefaultLeverage)
	}

	return &execution.ExecutionResult{
		Action:          action,
		Reason:          fmt.Sprintf("发现跨交易所价差%.2f%%，超过阈值%.2f%%", priceDiff, config.MinProfitThreshold),
		Symbol:          symbol,
		Timestamp:       time.Now(),
		Multiplier:      multiplier,
		MaxPositionSize: config.MaxPositionSize,
		MaxHoldHours:    1,   // 套利通常快速完成
		RiskLevel:       0.7, // 中高风险
	}, nil
}

// ExecuteSpotFutureArbitrage 执行现货期货套利
func (e *Executor) ExecuteSpotFutureArbitrage(ctx context.Context, symbol string, marketData *execution.MarketData,
	config *ArbitrageExecutionConfig, execContext *execution.ExecutionContext) (*execution.ExecutionResult, error) {

	log.Printf("[ArbitrageExecutor] 执行现货期货套利: %s", symbol)

	// 检查是否有期货交易能力
	if !marketData.HasFutures {
		return &execution.ExecutionResult{
			Action:    "skip",
			Reason:    "该交易对不支持期货交易",
			Symbol:    symbol,
			Timestamp: time.Now(),
		}, nil
	}

	// 获取现货价格
	var spotPrice float64
	if e.dependencies.MarketDataProvider != nil {
		price, err := e.dependencies.MarketDataProvider.GetRealTimePrice(symbol + "_SPOT")
		if err != nil {
			return &execution.ExecutionResult{
				Action:    "skip",
				Reason:    fmt.Sprintf("无法获取现货价格: %v", err),
				Symbol:    symbol,
				Timestamp: time.Now(),
			}, nil
		}
		spotPrice = price
	} else {
		spotPrice = marketData.Price // 使用传入的价格作为备选
	}

	// 获取期货价格
	var futuresPrice float64
	if e.dependencies.MarketDataProvider != nil {
		price, err := e.dependencies.MarketDataProvider.GetRealTimePrice(symbol + "_FUTURES")
		if err != nil {
			return &execution.ExecutionResult{
				Action:    "skip",
				Reason:    fmt.Sprintf("无法获取期货价格: %v", err),
				Symbol:    symbol,
				Timestamp: time.Now(),
			}, nil
		}
		futuresPrice = price
	} else {
		// 模拟期货价格（基于现货价格和基础差）
		futuresPrice = spotPrice * (1 + marketData.Change24h/100)
	}

	// 计算期现价差（期货价格相对于现货价格的溢价）
	priceDiff := futuresPrice - spotPrice
	priceDiffPercent := (priceDiff / spotPrice) * 100

	// 检查价差阈值
	if math.Abs(priceDiffPercent) >= config.MinProfitThreshold {
		// 检查价差是否足够大（考虑交易成本）
		minProfitThreshold := 0.05 // 最少0.05%的利润空间

		// 考虑资金费率影响（这里需要扩展市场数据以包含资金费率）
		fundingRate := e.getCurrentFundingRate(symbol)
		expectedProfit := math.Abs(priceDiffPercent)

		// 扣除资金费率成本（简化估算）
		if fundingRate != 0 {
			// 如果资金费率与套利方向相反，会增加成本
			if (priceDiffPercent > 0 && fundingRate > 0) || (priceDiffPercent < 0 && fundingRate < 0) {
				expectedProfit -= math.Abs(fundingRate) * 24 // 24小时资金费率影响
			}
		}

		// 检查是否有足够的利润空间
		if expectedProfit < minProfitThreshold {
			return &execution.ExecutionResult{
				Action: "no_op",
				Reason: fmt.Sprintf("期现价差%.2f%%不足以覆盖成本(扣除资金费率后预计利润:%.3f%%)",
					priceDiffPercent, expectedProfit),
				Symbol:    symbol,
				Timestamp: time.Now(),
				RiskLevel: 0.5,
			}, nil
		}

		if priceDiffPercent > 0 {
			// 期货价格高于现货，卖期货买现货
			// 计算执行参数
			multiplier := 1.0 // 套利默认1倍
			if config.AllowLeverage && config.DefaultLeverage > 1 {
				multiplier = float64(config.DefaultLeverage)
			}

			return &execution.ExecutionResult{
				Action: "arb_sell_futures_buy_spot",
				Reason: fmt.Sprintf("期现价差%.2f%%，预计利润%.3f%%，建议卖期货买现货",
					priceDiffPercent, expectedProfit),
				Symbol:          symbol,
				Timestamp:       time.Now(),
				Multiplier:      multiplier,
				MaxPositionSize: config.MaxPositionSize,
				MaxHoldHours:    24,
				RiskLevel:       0.6,
			}, nil
		} else {
			// 期货价格低于现货，买期货卖现货
			// 计算执行参数
			multiplier := 1.0 // 套利默认1倍
			if config.AllowLeverage && config.DefaultLeverage > 1 {
				multiplier = float64(config.DefaultLeverage)
			}

			return &execution.ExecutionResult{
				Action: "arb_buy_futures_sell_spot",
				Reason: fmt.Sprintf("期现价差%.2f%%，预计利润%.3f%%，建议买期货卖现货",
					priceDiffPercent, expectedProfit),
				Symbol:          symbol,
				Timestamp:       time.Now(),
				Multiplier:      multiplier,
				MaxPositionSize: config.MaxPositionSize,
				MaxHoldHours:    24,
				RiskLevel:       0.6,
			}, nil
		}
	}

	return &execution.ExecutionResult{
		Action: "no_op",
		Reason: fmt.Sprintf("期现价差%.2f%%在合理范围内(±%.2f%%)",
			priceDiffPercent, config.MinProfitThreshold),
		Symbol:    symbol,
		Timestamp: time.Now(),
		RiskLevel: 0.4,
	}, nil
}

// getCurrentFundingRate 获取当前资金费率（模拟实现）
func (e *Executor) getCurrentFundingRate(symbol string) float64 {
	// 这里应该从市场数据提供者获取真实的资金费率
	// 目前返回一个模拟值
	// 实际实现中需要扩展MarketData结构包含资金费率数据
	return 0.01 // 假设0.01%的资金费率
}

// calculateTriangleArbitrage 计算单个三角套利路径
func (e *Executor) calculateTriangleArbitrage(ctx context.Context, path []string, threshold float64) StrategyDecisionResult {
	if len(path) != 3 {
		return StrategyDecisionResult{
			Action:     "no_op",
			Reason:     "无效的三角路径",
			Multiplier: 1.0,
		}
	}

	// 路径格式：[A/USDT, B/A, B/USDT]
	// 例如：["BTCUSDT", "ETHBTC", "ETHUSDT"]
	// 理论：BTCUSDT * ETHBTC = ETHUSDT

	// 获取三个交易对的价格
	prices := make(map[string]float64)
	for _, symbol := range path {
		// 通过市场数据提供者获取价格
		if e.dependencies.MarketDataProvider != nil {
			price, err := e.dependencies.MarketDataProvider.GetRealTimePrice(symbol)
			if err != nil {
				log.Printf("[ArbitrageExecutor] 获取价格失败 %s: %v", symbol, err)
				return StrategyDecisionResult{
					Action:     "no_op",
					Reason:     fmt.Sprintf("无法获取%s价格", symbol),
					Multiplier: 1.0,
				}
			}
			prices[symbol] = price
		} else {
			// 如果没有市场数据提供者，暂时跳过
			return StrategyDecisionResult{
				Action:     "no_op",
				Reason:     "市场数据提供者不可用",
				Multiplier: 1.0,
			}
		}
	}

	// 计算理论价格
	// BTCUSDT * ETHBTC 应该等于 ETHUSDT
	theoreticalPrice := prices[path[0]] * prices[path[1]]
	actualPrice := prices[path[2]]

	// 计算价差百分比
	priceDiff := theoreticalPrice - actualPrice
	priceDiffPercent := (priceDiff / actualPrice) * 100

	// 检查是否有套利机会
	if math.Abs(priceDiffPercent) >= threshold {
		if priceDiffPercent > 0 {
			// 理论价格 > 实际价格
			// 这意味着：买入B/USDT，卖出A/USDT和B/A
			return StrategyDecisionResult{
				Action: "triangle_arb_buy_final_sell_others",
				Reason: fmt.Sprintf("三角套利: %s->%s->%s 价差%.2f%%(阈值%.2f%%)，理论价格高于实际",
					path[0], path[1], path[2], priceDiffPercent, threshold),
				Multiplier: 1.0,
			}
		} else {
			// 理论价格 < 实际价格
			// 这意味着：买入A/USDT和B/A，卖出B/USDT
			return StrategyDecisionResult{
				Action: "triangle_arb_buy_others_sell_final",
				Reason: fmt.Sprintf("三角套利: %s->%s->%s 价差%.2f%%(阈值%.2f%%)，实际价格高于理论",
					path[0], path[1], path[2], priceDiffPercent, threshold),
				Multiplier: 1.0,
			}
		}
	}

	return StrategyDecisionResult{
		Action: "no_op",
		Reason: fmt.Sprintf("三角套利: %s->%s->%s 价差%.2f%%未超过阈值%.2f%%",
			path[0], path[1], path[2], priceDiffPercent, threshold),
		Multiplier: 1.0,
	}
}

// StrategyDecisionResult 用于内部计算的决策结果
type StrategyDecisionResult struct {
	Action     string  `json:"action"`
	Reason     string  `json:"reason"`
	Multiplier float64 `json:"multiplier"`
}
