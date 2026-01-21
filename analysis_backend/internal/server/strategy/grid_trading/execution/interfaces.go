package execution

import (
	"context"
	"analysis/internal/server/strategy/shared/execution"
)

// ============================================================================
// 网格交易策略执行接口
// ============================================================================

// GridTradingExecutor 网格交易策略执行器接口
type GridTradingExecutor interface {
	execution.StrategyExecutor

	// 具体执行方法
	ExecuteGridSetup(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *GridTradingExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteGridRebalance(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *GridTradingExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)

	ExecuteGridExit(ctx context.Context, symbol string, marketData *execution.MarketData,
		config *GridTradingExecutionConfig, context *execution.ExecutionContext) (*execution.ExecutionResult, error)
}

// GridTradingExecutionConfig 网格交易策略执行配置
type GridTradingExecutionConfig struct {
	execution.ExecutionConfig

	// 网格交易特有配置
	GridTradingEnabled   bool    `json:"grid_trading_enabled"`    // 网格交易启用
	GridUpperPrice       float64 `json:"grid_upper_price"`        // 网格上限价格
	GridLowerPrice       float64 `json:"grid_lower_price"`        // 网格下限价格
	GridLevels           int     `json:"grid_levels"`             // 网格层数
	GridProfitPercent    float64 `json:"grid_profit_percent"`     // 网格利润百分比
	GridInvestmentAmount float64 `json:"grid_investment_amount"`  // 网格投资金额

	// 风险控制
	GridStopLossEnabled  bool    `json:"grid_stop_loss_enabled"`  // 网格止损启用
	GridStopLossPercent  float64 `json:"grid_stop_loss_percent"`  // 网格止损百分比
	DynamicPositioning   bool    `json:"dynamic_positioning"`     // 动态仓位调整

	// 网格操作参数
	GridRebalanceThreshold float64 `json:"grid_rebalance_threshold"` // 网格再平衡阈值
	GridExitCondition      string  `json:"grid_exit_condition"`       // 网格退出条件: "PROFIT_TARGET", "LOSS_LIMIT", "TIME_BASED"
	GridMaxHoldTime        int     `json:"grid_max_hold_time"`        // 网格最大持仓时间（小时）
}

// GridPosition 网格持仓信息
type GridPosition struct {
	Symbol     string  `json:"symbol"`
	Level      int     `json:"level"`       // 网格层级
	Price      float64 `json:"price"`       // 持仓价格
	Quantity   float64 `json:"quantity"`    // 持仓数量
	Side       string  `json:"side"`        // 持仓方向: "buy", "sell"
	Timestamp  int64   `json:"timestamp"`   // 持仓时间戳
}

// ExecutionDependencies 执行依赖（复用shared的定义）
type ExecutionDependencies = execution.ExecutionDependencies