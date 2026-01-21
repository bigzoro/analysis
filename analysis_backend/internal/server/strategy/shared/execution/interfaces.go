package execution

import (
	"context"
	"time"
)

// ============================================================================
// 策略执行核心接口
// ============================================================================

// StrategyExecutor 策略执行器接口
type StrategyExecutor interface {
	// 核心执行方法
	Execute(ctx context.Context, symbol string, marketData *MarketData,
		config interface{}, context *ExecutionContext) (*ExecutionResult, error)

	// 策略类型标识
	GetStrategyType() string

	// 策略启用检查
	IsEnabled(config interface{}) bool

	// 预执行验证
	ValidateExecution(symbol string, marketData *MarketData, config interface{}) error
}

// ExecutionEngine 执行引擎接口
type ExecutionEngine interface {
	// 执行策略
	ExecuteStrategy(ctx context.Context, executor StrategyExecutor, symbol string,
		marketData *MarketData, config interface{}) (*ExecutionResult, error)

	// 批量执行策略
	ExecuteStrategies(ctx context.Context, requests []ExecutionRequest) ([]*ExecutionResult, error)

	// 取消执行
	CancelExecution(requestID string) error

	// 查询执行状态
	GetExecutionStatus(requestID string) (*ExecutionStatus, error)
}

// ExecutionRequest 执行请求
type ExecutionRequest struct {
	StrategyType string            `json:"strategy_type"` // 策略类型
	Symbol       string            `json:"symbol"`        // 交易对
	Config       interface{}       `json:"config"`        // 策略配置
	Context      *ExecutionContext `json:"context"`       // 执行上下文
}

// ExecutionStatus 执行状态
type ExecutionStatus struct {
	RequestID string           `json:"request_id"`
	Status    string           `json:"status"` // "pending", "executing", "completed", "failed", "cancelled"
	StartTime time.Time        `json:"start_time"`
	EndTime   *time.Time       `json:"end_time,omitempty"`
	Result    *ExecutionResult `json:"result,omitempty"`
	Error     string           `json:"error,omitempty"`
}

// ============================================================================
// 依赖服务接口（依赖注入用）
// ============================================================================

// MarketDataProvider 市场数据提供者接口
type MarketDataProvider interface {
	GetMarketData(symbol string) (*MarketData, error)
	GetRealTimePrice(symbol string) (float64, error)
	GetKlineData(symbol, interval string, limit int) ([]*KlineData, error)
}

// OrderManager 订单管理器接口
type OrderManager interface {
	PlaceOrder(symbol, side string, quantity, price float64) (string, error)
	CancelOrder(orderID string) error
	GetOrderStatus(orderID string) (*OrderStatus, error)
}

// RiskManager 风险管理器接口
type RiskManager interface {
	ValidateRisk(symbol string, positionSize float64) error
	CalculateStopLoss(entryPrice float64, riskPercent float64) float64
	CalculateTakeProfit(entryPrice float64, rewardPercent float64) float64
	CheckPositionLimits(symbol string, newPositionSize float64) error

	// 保证金风险管理方法
	CalculateMarginStopLoss(symbol string, marginLossPercent float64) (float64, error)
	CalculateMarginTakeProfit(symbol string, marginProfitPercent float64) (float64, error)
	CheckMarginLoss(symbol string, marginLossPercent float64) (bool, float64, error)
	GetPositionMarginInfo(symbol string) (*PositionMarginInfo, error)
	ValidateMarginStopLossConfig(marginLossPercent float64) error
}

// ConfigProvider 配置提供者接口
type ConfigProvider interface {
	GetStrategyConfig(strategyType string, userID uint) (interface{}, error)
	GetGlobalConfig(key string) (interface{}, error)
	UpdateStrategyConfig(strategyType string, userID uint, config interface{}) error
}

// ExecutionDependencies 执行依赖容器
type ExecutionDependencies struct {
	MarketDataProvider MarketDataProvider
	OrderManager       OrderManager
	RiskManager        RiskManager
	ConfigProvider     ConfigProvider
}

// ============================================================================
// 数据结构定义
// ============================================================================

// KlineData K线数据
type KlineData struct {
	OpenTime   int64   `json:"open_time"`
	OpenPrice  float64 `json:"open_price"`
	HighPrice  float64 `json:"high_price"`
	LowPrice   float64 `json:"low_price"`
	ClosePrice float64 `json:"close_price"`
	Volume     float64 `json:"volume"`
	CloseTime  int64   `json:"close_time"`
}

// OrderStatus 订单状态
type OrderStatus struct {
	OrderID     string  `json:"order_id"`
	Status      string  `json:"status"` // "new", "filled", "cancelled", "rejected"
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"` // "buy", "sell"
	Quantity    float64 `json:"quantity"`
	Price       float64 `json:"price"`
	ExecutedQty float64 `json:"executed_qty"`
	AvgPrice    float64 `json:"avg_price"`
	Fee         float64 `json:"fee"`
}

// PositionMarginInfo 持仓保证金信息
type PositionMarginInfo struct {
	Symbol            string  `json:"symbol"`
	EntryPrice        float64 `json:"entry_price"`
	PositionAmount    float64 `json:"position_amount"`
	UnrealizedProfit  float64 `json:"unrealized_profit"`
	InitialMargin     float64 `json:"initial_margin"`
	MaintMargin       float64 `json:"maint_margin"`
	Leverage          float64 `json:"leverage"`
	MarginLossPercent float64 `json:"margin_loss_percent"` // 保证金盈亏百分比
	IsIsolated        bool    `json:"is_isolated"`         // 是否逐仓模式
}
