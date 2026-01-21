package factory

import (
	pdb "analysis/internal/db"
	arb_execution "analysis/internal/server/strategy/arbitrage/execution"
	grid_execution "analysis/internal/server/strategy/grid_trading/execution"
	mr_execution "analysis/internal/server/strategy/mean_reversion/execution"
	ma_execution "analysis/internal/server/strategy/moving_average/execution"
	"analysis/internal/server/strategy/shared/execution"
	traditional_execution "analysis/internal/server/strategy/traditional/execution"
	"fmt"
)

// ============================================================================
// 策略工厂 - 统一创建策略执行器和配置
// ============================================================================

// StrategyFactory 策略工厂
type StrategyFactory struct {
	deps *ExecutionDependencies
}

// ExecutionDependencies 执行依赖（与shared/execution保持一致）
type ExecutionDependencies struct {
	MarketDataProvider execution.MarketDataProvider
	OrderManager       execution.OrderManager
	RiskManager        execution.RiskManager
	ConfigProvider     execution.ConfigProvider
}

// NewStrategyFactory 创建策略工厂
func NewStrategyFactory(deps *ExecutionDependencies) *StrategyFactory {
	return &StrategyFactory{
		deps: deps,
	}
}

// CreateExecutor 创建策略执行器
func (f *StrategyFactory) CreateExecutor(strategyType string, conditions pdb.StrategyConditions) (execution.StrategyExecutor, interface{}, error) {
	switch strategyType {
	case "traditional":
		config := f.buildTraditionalConfig(conditions)
		executor := traditional_execution.NewExecutor(&traditional_execution.ExecutionDependencies{
			MarketDataProvider: f.deps.MarketDataProvider,
			OrderManager:       f.deps.OrderManager,
			RiskManager:        f.deps.RiskManager,
			ConfigProvider:     f.deps.ConfigProvider,
		})
		return executor, config, nil

	case "moving_average":
		config := f.buildMovingAverageConfig(conditions)
		executor := ma_execution.NewExecutor(&ma_execution.ExecutionDependencies{
			MarketDataProvider: f.deps.MarketDataProvider,
			OrderManager:       f.deps.OrderManager,
			RiskManager:        f.deps.RiskManager,
			ConfigProvider:     f.deps.ConfigProvider,
		})
		return executor, config, nil

	case "arbitrage":
		config := f.buildArbitrageConfig(conditions)
		executor := arb_execution.NewExecutor(&arb_execution.ExecutionDependencies{
			MarketDataProvider: f.deps.MarketDataProvider,
			OrderManager:       f.deps.OrderManager,
			RiskManager:        f.deps.RiskManager,
			ConfigProvider:     f.deps.ConfigProvider,
		})
		return executor, config, nil

	case "grid_trading":
		config := f.buildGridTradingConfig(conditions)
		executor := grid_execution.NewExecutor(&grid_execution.ExecutionDependencies{
			MarketDataProvider: f.deps.MarketDataProvider,
			OrderManager:       f.deps.OrderManager,
			RiskManager:        f.deps.RiskManager,
			ConfigProvider:     f.deps.ConfigProvider,
		})
		return executor, config, nil

	case "mean_reversion":
		config := f.buildMeanReversionConfig(conditions)
		executor := mr_execution.NewExecutor(&mr_execution.ExecutionDependencies{
			MarketDataProvider: f.deps.MarketDataProvider,
			OrderManager:       f.deps.OrderManager,
			RiskManager:        f.deps.RiskManager,
			ConfigProvider:     f.deps.ConfigProvider,
		})
		return executor, config, nil

	default:
		return nil, nil, fmt.Errorf("不支持的策略类型: %s", strategyType)
	}
}

// ============================================================================
// 配置构建器方法（与router保持一致）
// ============================================================================

// buildTraditionalConfig 构建传统策略配置
func (f *StrategyFactory) buildTraditionalConfig(conditions pdb.StrategyConditions) interface{} {
	return &traditional_execution.TraditionalExecutionConfig{
		ExecutionConfig:               execution.ExecutionConfig{Enabled: true},
		ShortOnGainers:                conditions.ShortOnGainers,
		LongOnSmallGainers:            conditions.LongOnSmallGainers,
		GainersRankLimit:              conditions.GainersRankLimit,
		LongGainersRankLimit:          conditions.GainersRankLimitLong,
		MarketCapLimitShort:           conditions.MarketCapLimitShort * 10000, // 转换为万元
		MarketCapLimitLong:            conditions.MarketCapLimitLong * 10000,  // 转换为万元
		ShortMultiplier:               conditions.ShortMultiplier,
		LongMultiplier:                conditions.LongMultiplier,
		FuturesPriceRankFilterEnabled: conditions.FuturesPriceRankFilterEnabled,
		MaxFuturesPriceRank:           conditions.MaxFuturesPriceRank,
		TradingType:                   conditions.TradingType,
		// 新增：合约涨幅开空策略
		FuturesPriceShortStrategyEnabled: conditions.FuturesPriceShortStrategyEnabled,
		FuturesPriceShortMaxRank:         conditions.FuturesPriceShortMaxRank,
		FuturesPriceShortMinFundingRate:  conditions.FuturesPriceShortMinFundingRate,
		FuturesPriceShortLeverage:        conditions.FuturesPriceShortLeverage,

		// 杠杆配置
		EnableLeverage:  conditions.EnableLeverage,
		DefaultLeverage: conditions.DefaultLeverage,
		MaxLeverage:     conditions.MaxLeverage,

		// 保证金损失止损
		EnableMarginLossStopLoss:  conditions.EnableMarginLossStopLoss,
		MarginLossStopLossPercent: conditions.MarginLossStopLossPercent,

		// 保证金盈利止盈
		EnableMarginProfitTakeProfit:  conditions.EnableMarginProfitTakeProfit,
		MarginProfitTakeProfitPercent: conditions.MarginProfitTakeProfitPercent,
	}
}

// buildMovingAverageConfig 构建均线策略配置
func (f *StrategyFactory) buildMovingAverageConfig(conditions pdb.StrategyConditions) interface{} {
	return &ma_execution.MovingAverageExecutionConfig{
		ExecutionConfig:      execution.ExecutionConfig{Enabled: true},
		MovingAverageEnabled: conditions.MovingAverageEnabled,
		MAType:               conditions.MAType,
		ShortMAPeriod:        conditions.ShortMAPeriod,
		LongMAPeriod:         conditions.LongMAPeriod,
		MACrossSignal:        conditions.MACrossSignal,
		MATrendFilter:        conditions.MATrendFilter,
		MATrendDirection:     conditions.MATrendDirection,
		MASignalMode:         conditions.MASignalMode,
		LongMultiplier:       1.0, // 默认值
		ShortMultiplier:      1.0, // 默认值
	}
}

// buildArbitrageConfig 构建套利策略配置
func (f *StrategyFactory) buildArbitrageConfig(conditions pdb.StrategyConditions) interface{} {
	return &arb_execution.ArbitrageExecutionConfig{
		ExecutionConfig:         execution.ExecutionConfig{Enabled: true},
		CrossExchangeArbEnabled: conditions.CrossExchangeArbEnabled,
		SpotFutureArbEnabled:    conditions.SpotFutureArbEnabled,
		TriangleArbEnabled:      conditions.TriangleArbEnabled,
		StatisticalArbEnabled:   conditions.StatArbEnabled,
		FuturesSpotArbEnabled:   conditions.FuturesSpotArbEnabled,
		MinProfitThreshold:      conditions.SpotFutureSpread,
	}
}

// buildGridTradingConfig 构建网格交易策略配置
func (f *StrategyFactory) buildGridTradingConfig(conditions pdb.StrategyConditions) interface{} {
	return &grid_execution.GridTradingExecutionConfig{
		ExecutionConfig: execution.ExecutionConfig{Enabled: true},
		// 网格交易配置可以根据需要扩展
	}
}

// buildMeanReversionConfig 构建均值回归策略配置
func (f *StrategyFactory) buildMeanReversionConfig(conditions pdb.StrategyConditions) interface{} {
	return &mr_execution.MeanReversionExecutionConfig{
		ExecutionConfig:         execution.ExecutionConfig{Enabled: true},
		MeanReversionEnabled:    conditions.MeanReversionEnabled,
		MRBollingerBandsEnabled: conditions.MRBollingerBandsEnabled,
		MeanReversionLookback:   conditions.MRPeriod,
		MeanReversionThreshold:  conditions.MRMinReversionStrength,
		MeanReversionStdDev:     conditions.MRBollingerMultiplier,
		BollingerPeriod:         conditions.MRChannelPeriod,
		BollingerStdDev:         conditions.MRBollingerMultiplier,
		LongMultiplier:          1.0, // 默认值
		ShortMultiplier:         1.0, // 默认值
	}
}
