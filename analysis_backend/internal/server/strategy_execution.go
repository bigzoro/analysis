package server

import (
	arb_core "analysis/internal/server/strategy/arbitrage/core"
	grid_core "analysis/internal/server/strategy/grid_trading/core"
	mean_reversion_scanning "analysis/internal/server/strategy/mean_reversion/scanning"
	ma_core "analysis/internal/server/strategy/moving_average/core"
	traditional_core "analysis/internal/server/strategy/traditional/core"
	"context"
	"fmt"
	"log"

	pdb "analysis/internal/db"
)

// ============================================================================
// 策略执行核心 - 只包含接口定义、注册表和基础执行逻辑
// ============================================================================

// 策略市场数据结构（用于策略执行）
type StrategyMarketData struct {
	Symbol      string  `json:"symbol"`
	MarketCap   float64 `json:"market_cap"`
	GainersRank int     `json:"gainers_rank"`
	HasSpot     bool    `json:"has_spot"`
	HasFutures  bool    `json:"has_futures"`
}

// 策略决策结果
type StrategyDecisionResult struct {
	Action     string  `json:"action"` // "buy", "sell", "skip", "no_op", "allow"
	Reason     string  `json:"reason"`
	Multiplier float64 `json:"multiplier"`
}

// 策略执行器接口
type StrategyExecutor interface {
	// 获取策略类型标识
	GetStrategyType() string

	// 执行策略判断 (完整版本，有外部依赖)
	ExecuteFull(ctx context.Context, server *Server, symbol string, marketData StrategyMarketData, conditions pdb.StrategyConditions, strategy *pdb.TradingStrategy) StrategyDecisionResult

	// 检查策略是否启用
	IsEnabled(conditions pdb.StrategyConditions) bool
}

// ============================================================================
// 策略执行器注册表
// ============================================================================

// 策略执行器注册表
type StrategyExecutorRegistry struct {
	executors map[string]StrategyExecutor
}

// 创建策略执行器注册表
func NewStrategyExecutorRegistry() *StrategyExecutorRegistry {
	registry := &StrategyExecutorRegistry{
		executors: make(map[string]StrategyExecutor),
	}

	// 注册所有策略执行器
	registry.registerExecutors()

	return registry
}

// 注册所有策略执行器
func (r *StrategyExecutorRegistry) registerExecutors() {
	// 这里将在创建时动态注册，因为执行器需要Server实例
	// 实际的注册在 StrategyExecutorRegistry 的其他方法中处理
	log.Printf("[StrategyExecutorRegistry] 初始化策略执行器注册表")
	// 注意：由于执行器需要Server实例，我们将在Server初始化时通过RegisterExecutor方法注册
}

// 获取策略执行器
func (r *StrategyExecutorRegistry) GetExecutor(strategyType string) StrategyExecutor {
	return r.executors[strategyType]
}

// 获取所有策略执行器
func (r *StrategyExecutorRegistry) GetAllExecutors() map[string]StrategyExecutor {
	return r.executors
}

// 注册策略执行器
func (r *StrategyExecutorRegistry) RegisterExecutor(strategyType string, executor StrategyExecutor) {
	r.executors[strategyType] = executor
}

// ============================================================================
// 核心策略执行逻辑
// ============================================================================

// 策略执行逻辑（重构版：使用策略执行器模式）
func executeStrategyLogic(strategy *pdb.TradingStrategy, symbol string, marketData StrategyMarketData) StrategyDecisionResult {
	conditions := strategy.Conditions

	// 调试日志：记录策略条件和市场数据
	log.Printf("[StrategyLogic] %s 检查条件 - SpotContract:%v, ShortOnGainers:%v, LongOnSmallGainers:%v, MovingAverage:%v",
		symbol, conditions.SpotContract, conditions.ShortOnGainers, conditions.LongOnSmallGainers, conditions.MovingAverageEnabled)
	log.Printf("[StrategyLogic] %s 市场数据 - rank:%d, marketCap:%.0f, hasSpot:%v, hasFutures:%v",
		symbol, marketData.GainersRank, marketData.MarketCap, marketData.HasSpot, marketData.HasFutures)

	// 阶段1: 基础条件检查（所有策略共用）
	result := executeBasicChecks(symbol, marketData, conditions)
	if result.Action != "continue" {
		return result
	}

	// 阶段2: 使用策略执行器进行具体策略判断
	return executeStrategyWithExecutors(symbol, marketData, conditions)
}

// 执行基础条件检查
func executeBasicChecks(symbol string, marketData StrategyMarketData, conditions pdb.StrategyConditions) StrategyDecisionResult {
	// 检查是否有现货+合约条件
	if conditions.SpotContract {
		if !marketData.HasSpot || !marketData.HasFutures {
			return StrategyDecisionResult{
				Action:     "skip",
				Reason:     fmt.Sprintf("需要现货+合约，但%s现货:%v 合约:%v", symbol, marketData.HasSpot, marketData.HasFutures),
				Multiplier: 1.0,
			}
		}
	}

	// 检查是否有任何策略启用
	hasAnyStrategy := conditions.ShortOnGainers || conditions.LongOnSmallGainers ||
		conditions.FuturesPriceShortStrategyEnabled || conditions.MovingAverageEnabled ||
		conditions.CrossExchangeArbEnabled || conditions.SpotFutureArbEnabled ||
		conditions.TriangleArbEnabled || conditions.StatArbEnabled ||
		conditions.FuturesSpotArbEnabled || conditions.GridTradingEnabled ||
		conditions.MeanReversionEnabled

	if !hasAnyStrategy {
		// 如果只设置了现货+合约条件但没有其他策略，则允许执行
		if conditions.SpotContract {
			return StrategyDecisionResult{
				Action:     "allow",
				Reason:     fmt.Sprintf("%s 满足现货+合约条件，允许执行", symbol),
				Multiplier: 1.0,
			}
		}
		return StrategyDecisionResult{
			Action:     "no_op",
			Reason:     "未启用任何策略条件",
			Multiplier: 1.0,
		}
	}

	return StrategyDecisionResult{
		Action:     "continue",
		Reason:     "基础检查通过，继续策略判断",
		Multiplier: 1.0,
	}
}

// 使用策略执行器进行策略判断
func executeStrategyWithExecutors(symbol string, marketData StrategyMarketData, conditions pdb.StrategyConditions) StrategyDecisionResult {
	// 注意：这里是纯函数版本，无法访问Server实例
	// 根据策略类型判断是否需要外部依赖

	// 检查传统策略（不需要外部依赖）
	if conditions.ShortOnGainers || conditions.LongOnSmallGainers {
		log.Printf("[StrategyLogic] %s 传统策略需要外部依赖，返回allow", symbol)
		return StrategyDecisionResult{
			Action:     "allow",
			Reason:     "传统策略需要外部数据检查",
			Multiplier: 1.0,
		}
	}

	// 检查其他策略（需要外部依赖）
	if conditions.MovingAverageEnabled || conditions.GridTradingEnabled ||
		conditions.CrossExchangeArbEnabled || conditions.SpotFutureArbEnabled ||
		conditions.TriangleArbEnabled || conditions.StatArbEnabled ||
		conditions.FuturesSpotArbEnabled || conditions.MeanReversionEnabled {
		log.Printf("[StrategyLogic] %s 策略需要外部依赖，返回allow", symbol)
		return StrategyDecisionResult{
			Action:     "allow",
			Reason:     "策略需要外部数据检查",
			Multiplier: 1.0,
		}
	}

	log.Printf("[StrategyLogic] %s 不符合任何策略条件，返回no_op", symbol)
	return StrategyDecisionResult{
		Action:     "no_op",
		Reason:     "不符合任何策略条件",
		Multiplier: 1.0,
	}
}

// 使用完整策略执行器（有外部依赖）进行策略判断
func (s *Server) executeStrategyWithFullExecutors(ctx context.Context, symbol string, marketData StrategyMarketData, conditions pdb.StrategyConditions, strategy *pdb.TradingStrategy) StrategyDecisionResult {
	executors := globalStrategyRegistry.GetAllExecutors()

	// 遍历所有策略执行器，找到启用的策略并执行完整版本
	for _, executor := range executors {
		if executor.IsEnabled(conditions) {
			log.Printf("[StrategyFull] %s 执行%s策略完整版本", symbol, executor.GetStrategyType())
			result := executor.ExecuteFull(ctx, s, symbol, marketData, conditions, strategy)

			// 如果策略返回了确定的动作，则立即返回
			if result.Action != "no_op" {
				log.Printf("[StrategyFull] %s 策略%s完整版本返回: %s - %s",
					symbol, executor.GetStrategyType(), result.Action, result.Reason)
				return result
			}
		}
	}

	log.Printf("[StrategyFull] %s 所有策略完整检查均返回no_op", symbol)
	return StrategyDecisionResult{
		Action:     "no_op",
		Reason:     "所有策略完整检查均不符合条件",
		Multiplier: 1.0,
	}
}

// 全局策略执行器注册表
var globalStrategyRegistry *StrategyExecutorRegistry

// 初始化策略注册表
func init() {
	globalStrategyRegistry = NewStrategyExecutorRegistry()
}

// ============================================================================
// 策略实例化方法
// ============================================================================

// getNewMeanReversionStrategy 获取新的模块化均值回归策略
func getNewMeanReversionStrategy(db interface{}) (StrategyScanner, error) {
	log.Printf("[StrategyInit] 获取均值回归策略实例...")

	// 获取策略扫描器实例（直接使用scanning包中的全局实例）
	scanner := mean_reversion_scanning.GetMeanReversionScanner(db)
	if scanner == nil {
		return nil, fmt.Errorf("mean_reversion_scanning.GetMeanReversionScanner() 返回 nil")
	}
	log.Printf("[StrategyInit] 均值回归策略实例获取成功")

	// 获取适配器
	adapterInterface := scanner.ToStrategyScanner()
	if adapterInterface == nil {
		return nil, fmt.Errorf("scanner.ToStrategyScanner() 返回 nil")
	}
	log.Printf("[StrategyInit] 均值回归策略适配器获取成功")

	// 类型断言获取适配器
	adapter, ok := adapterInterface.(StrategyScanner)
	if !ok {
		return nil, fmt.Errorf("适配器类型断言失败: %T", adapterInterface)
	}
	log.Printf("[StrategyInit] 均值回归策略类型断言成功")

	return adapter, nil
}

// getNewGridTradingStrategy 获取新的模块化网格交易策略
func getNewGridTradingStrategy() (StrategyScanner, error) {
	log.Printf("[StrategyInit] 获取网格交易策略实例...")

	// 获取策略实例
	strategy := grid_core.GetGridTradingStrategy()
	if strategy == nil {
		return nil, fmt.Errorf("grid_core.GetGridTradingStrategy() 返回 nil")
	}
	log.Printf("[StrategyInit] 网格交易策略实例获取成功")

	// 获取适配器
	adapterInterface := strategy.ToStrategyScanner()
	if adapterInterface == nil {
		return nil, fmt.Errorf("strategy.ToStrategyScanner() 返回 nil")
	}
	log.Printf("[StrategyInit] 网格交易策略适配器获取成功")

	// 类型断言获取适配器
	adapter, ok := adapterInterface.(StrategyScanner)
	if !ok {
		return nil, fmt.Errorf("适配器类型断言失败: %T", adapterInterface)
	}
	log.Printf("[StrategyInit] 网格交易策略类型断言成功")

	return adapter, nil
}

// getNewTraditionalStrategy 获取新的模块化传统策略
func getNewTraditionalStrategy(db interface{}) (StrategyScanner, error) {
	log.Printf("[StrategyInit] 获取传统策略实例...")

	// 获取策略实例（传递数据库连接）
	strategy := traditional_core.GetTraditionalStrategy(db)
	if strategy == nil {
		return nil, fmt.Errorf("traditional_core.GetTraditionalStrategy() 返回 nil")
	}
	log.Printf("[StrategyInit] 传统策略实例获取成功")

	// 获取适配器
	adapterInterface := strategy.ToStrategyScanner()
	if adapterInterface == nil {
		return nil, fmt.Errorf("strategy.ToStrategyScanner() 返回 nil")
	}
	log.Printf("[StrategyInit] 传统策略适配器获取成功")

	// 类型断言获取适配器
	adapter, ok := adapterInterface.(StrategyScanner)
	if !ok {
		return nil, fmt.Errorf("适配器类型断言失败: %T", adapterInterface)
	}
	log.Printf("[StrategyInit] 传统策略类型断言成功")

	return adapter, nil
}

// getNewMovingAverageStrategy 获取新的模块化均线策略
func getNewMovingAverageStrategy() (StrategyScanner, error) {
	log.Printf("[StrategyInit] 获取均线策略实例...")

	// 获取策略实例
	strategy := ma_core.GetMovingAverageStrategy()
	if strategy == nil {
		return nil, fmt.Errorf("ma_core.GetMovingAverageStrategy() 返回 nil")
	}
	log.Printf("[StrategyInit] 均线策略实例获取成功")

	// 获取适配器
	adapterInterface := strategy.ToStrategyScanner()
	if adapterInterface == nil {
		return nil, fmt.Errorf("strategy.ToStrategyScanner() 返回 nil")
	}
	log.Printf("[StrategyInit] 均线策略适配器获取成功")

	// 类型断言获取适配器
	adapter, ok := adapterInterface.(StrategyScanner)
	if !ok {
		return nil, fmt.Errorf("适配器类型断言失败: %T", adapterInterface)
	}
	log.Printf("[StrategyInit] 均线策略类型断言成功")

	return adapter, nil
}

// getNewArbitrageStrategy 获取新的模块化套利策略
func getNewArbitrageStrategy() (StrategyScanner, error) {
	log.Printf("[StrategyInit] 获取套利策略实例...")

	// 获取策略实例
	strategy := arb_core.GetArbitrageStrategy()
	if strategy == nil {
		return nil, fmt.Errorf("arb_core.GetArbitrageStrategy() 返回 nil")
	}
	log.Printf("[StrategyInit] 套利策略实例获取成功")

	// 获取适配器
	adapterInterface := strategy.ToStrategyScanner()
	if adapterInterface == nil {
		return nil, fmt.Errorf("strategy.ToStrategyScanner() 返回 nil")
	}
	log.Printf("[StrategyInit] 套利策略适配器获取成功")

	// 类型断言获取适配器
	adapter, ok := adapterInterface.(StrategyScanner)
	if !ok {
		return nil, fmt.Errorf("适配器类型断言失败: %T", adapterInterface)
	}
	log.Printf("[StrategyInit] 套利策略类型断言成功")

	return adapter, nil
}
