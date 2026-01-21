package execution

import (
	"context"
	"analysis/internal/server/strategy/shared/execution"
)

// ============================================================================
// 套利策略执行接口
// ============================================================================

// ArbitrageExecutor 套利策略执行器接口
type ArbitrageExecutor interface {
	execution.StrategyExecutor

	// 具体执行方法
	ExecuteTriangleArbitrage(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *ArbitrageExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteCrossExchangeArbitrage(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *ArbitrageExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteSpotFutureArbitrage(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *ArbitrageExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)
}

// ArbitrageExecutionConfig 套利策略执行配置
type ArbitrageExecutionConfig struct {
	execution.ExecutionConfig

	// 套利策略特有配置
	TriangleArbEnabled      bool    `json:"triangle_arb_enabled"`       // 三角套利启用
	CrossExchangeArbEnabled bool    `json:"cross_exchange_arb_enabled"` // 跨交易所套利启用
	SpotFutureArbEnabled    bool    `json:"spot_future_arb_enabled"`    // 现货期货套利启用
	StatisticalArbEnabled   bool    `json:"statistical_arb_enabled"`    // 统计套利启用
	FuturesSpotArbEnabled   bool    `json:"futures_spot_arb_enabled"`   // 期货现货套利启用

	// 套利参数
	MinProfitThreshold      float64 `json:"min_profit_threshold"`       // 最小利润阈值
	MaxSlippagePercent      float64 `json:"max_slippage_percent"`       // 最大滑点百分比
	MinVolumeThreshold      float64 `json:"min_volume_threshold"`       // 最小交易量阈值

	// 三角套利参数
	TriangleMinProfitPercent float64 `json:"triangle_min_profit_percent"` // 三角套利最小利润百分比
	TriangleMaxPathLength    int     `json:"triangle_max_path_length"`    // 三角套利最大路径长度

	// 跨交易所参数
	ExchangePairs            []string `json:"exchange_pairs"` // 交易所对列表
}

// ExecutionDependencies 执行依赖（复用shared的定义）
type ExecutionDependencies = execution.ExecutionDependencies