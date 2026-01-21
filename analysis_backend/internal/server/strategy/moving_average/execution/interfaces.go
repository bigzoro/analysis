package execution

import (
	"context"
	"analysis/internal/server/strategy/shared/execution"
)

// ============================================================================
// 均线策略执行接口
// ============================================================================

// MovingAverageExecutor 均线策略执行器接口
type MovingAverageExecutor interface {
	execution.StrategyExecutor

	// 具体执行方法
	ExecuteGoldenCross(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *MovingAverageExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteDeathCross(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *MovingAverageExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)
}

// MovingAverageExecutionConfig 均线策略执行配置
type MovingAverageExecutionConfig struct {
	execution.ExecutionConfig

	// 均线策略特有配置
	MovingAverageEnabled bool   `json:"moving_average_enabled"` // 均线策略启用
	MAType               string `json:"ma_type"`                // 均线类型: "SMA", "EMA", "WMA"
	ShortMAPeriod        int    `json:"short_ma_period"`        // 短期均线周期
	LongMAPeriod         int    `json:"long_ma_period"`         // 长期均线周期
	MACrossSignal        string `json:"ma_cross_signal"`        // 交叉信号: "GOLDEN_CROSS", "DEATH_CROSS", "BOTH"
	MATrendFilter        bool   `json:"ma_trend_filter"`        // 趋势过滤启用
	MATrendDirection     string `json:"ma_trend_direction"`     // 趋势方向: "UP", "DOWN", "BOTH"
	MASignalMode         string `json:"ma_signal_mode"`         // 信号模式: "QUALITY_FIRST", "QUANTITY_FIRST"

	// 执行参数
	LongMultiplier  float64 `json:"long_multiplier"`  // 多头倍数
	ShortMultiplier float64 `json:"short_multiplier"` // 空头倍数
}

// ExecutionDependencies 执行依赖（复用shared的定义）
type ExecutionDependencies = execution.ExecutionDependencies