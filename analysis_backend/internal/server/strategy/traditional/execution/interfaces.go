package execution

import (
	"analysis/internal/server/strategy/shared/execution"
	"context"
)

// ============================================================================
// 传统策略执行接口
// ============================================================================

// TraditionalExecutor 传统策略执行器接口
type TraditionalExecutor interface {
	execution.StrategyExecutor

	// 具体执行方法
	ExecuteShortOnGainers(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *TraditionalExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteLongOnSmallGainers(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *TraditionalExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteFuturesPriceShort(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *TraditionalExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)
}

// TraditionalExecutionConfig 传统策略执行配置
type TraditionalExecutionConfig struct {
	execution.ExecutionConfig

	// 传统策略特有配置
	ShortOnGainers        bool    `json:"short_on_gainers"`          // 涨幅开空
	LongOnSmallGainers    bool    `json:"long_on_small_gainers"`     // 小幅上涨开多
	NoShortBelowMarketCap bool    `json:"no_short_below_market_cap"` // 市值低于阈值不开空
	GainersRankLimit      int     `json:"gainers_rank_limit"`        // 开空涨幅排名限制
	LongGainersRankLimit  int     `json:"long_gainers_rank_limit"`   // 开多涨幅排名限制
	MarketCapLimitShort   float64 `json:"market_cap_limit_short"`    // 开空市值限制
	MarketCapLimitLong    float64 `json:"market_cap_limit_long"`     // 开多市值限制
	ShortMultiplier       float64 `json:"short_multiplier"`          // 开空倍数
	LongMultiplier        float64 `json:"long_multiplier"`           // 开多倍数

	// 杠杆配置
	EnableLeverage  bool `json:"enable_leverage"`  // 是否启用杠杆
	DefaultLeverage int  `json:"default_leverage"` // 默认杠杆倍数
	MaxLeverage     int  `json:"max_leverage"`     // 最大杠杆倍数

	// 合约涨幅排名过滤
	FuturesPriceRankFilterEnabled bool `json:"futures_price_rank_filter_enabled"` // 合约涨幅排名过滤启用
	MaxFuturesPriceRank           int  `json:"max_futures_price_rank"`            // 合约涨幅最大排名（前N名）

	// 交易类型配置
	TradingType string `json:"trading_type"` // 交易类型: "spot", "futures", "both"

	// 新增：合约涨幅开空策略
	FuturesPriceShortStrategyEnabled bool    `json:"futures_price_short_strategy_enabled"` // 合约涨幅开空策略启用
	FuturesPriceShortMaxRank         int     `json:"futures_price_short_max_rank"`         // 合约涨幅最大排名
	FuturesPriceShortMinFundingRate  float64 `json:"futures_price_short_min_funding_rate"` // 最低资金费率要求
	FuturesPriceShortLeverage        float64 `json:"futures_price_short_leverage"`         // 开空倍数
	FuturesPriceShortMinMarketCap    float64 `json:"futures_price_short_min_market_cap"`   // 最低市值要求（万）

	// 保证金损失止损
	EnableMarginLossStopLoss  bool    `json:"enable_margin_loss_stop_loss"`  // 启用保证金损失止损
	MarginLossStopLossPercent float64 `json:"margin_loss_stop_loss_percent"` // 保证金损失止损百分比

	// 保证金盈利止盈
	EnableMarginProfitTakeProfit  bool    `json:"enable_margin_profit_take_profit"`  // 启用保证金盈利止盈
	MarginProfitTakeProfitPercent float64 `json:"margin_profit_take_profit_percent"` // 保证金盈利止盈百分比
}

// ExecutionDependencies 执行依赖
type ExecutionDependencies struct {
	MarketDataProvider execution.MarketDataProvider
	OrderManager       execution.OrderManager
	RiskManager        execution.RiskManager
	ConfigProvider     execution.ConfigProvider
}
