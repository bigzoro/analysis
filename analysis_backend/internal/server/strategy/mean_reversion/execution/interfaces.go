package execution

import (
	"context"
	"analysis/internal/server/strategy/shared/execution"
)

// ============================================================================
// 均值回归策略执行接口
// ============================================================================

// MeanReversionExecutor 均值回归策略执行器接口
type MeanReversionExecutor interface {
	execution.StrategyExecutor

	// 具体执行方法
	ExecuteMeanReversionBuy(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *MeanReversionExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteMeanReversionSell(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *MeanReversionExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteMeanReversionExit(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *MeanReversionExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)
}

// MeanReversionExecutionConfig 均值回归策略执行配置
type MeanReversionExecutionConfig struct {
	execution.ExecutionConfig

	// 均值回归特有配置
	MeanReversionEnabled    bool    `json:"mean_reversion_enabled"`     // 均值回归策略启用
	MRBollingerBandsEnabled bool    `json:"mr_bollinger_bands_enabled"` // 布林带均值回归启用

	// 均值回归参数
	MeanReversionLookback   int     `json:"mean_reversion_lookback"`    // 回望周期
	MeanReversionThreshold  float64 `json:"mean_reversion_threshold"`   // 均值回归阈值
	MeanReversionStdDev     float64 `json:"mean_reversion_std_dev"`     // 标准差倍数

	// 布林带参数
	BollingerPeriod         int     `json:"bollinger_period"`           // 布林带周期
	BollingerStdDev         float64 `json:"bollinger_std_dev"`           // 布林带标准差倍数

	// 执行参数
	LongMultiplier          float64 `json:"long_multiplier"`            // 多头倍数
	ShortMultiplier         float64 `json:"short_multiplier"`           // 空头倍数
	MaxPositionTime         int     `json:"max_position_time"`           // 最大持仓时间（小时）
	ProfitTarget            float64 `json:"profit_target"`              // 利润目标
	LossLimit               float64 `json:"loss_limit"`                 // 损失限制
}

// MeanReversionSignal 均值回归信号
type MeanReversionSignal struct {
	Symbol       string  `json:"symbol"`
	SignalType   string  `json:"signal_type"`   // "BUY", "SELL", "EXIT"
	Confidence   float64 `json:"confidence"`    // 信号置信度 0-1
	ZScore       float64 `json:"z_score"`       // Z分数
	BollingerPosition string `json:"bollinger_position"` // "UPPER", "LOWER", "MIDDLE"
	ExpectedReturn float64 `json:"expected_return"` // 预期收益率
	RiskLevel    float64 `json:"risk_level"`    // 风险等级 0-1
}

// ExecutionDependencies 执行依赖（复用shared的定义）
type ExecutionDependencies = execution.ExecutionDependencies