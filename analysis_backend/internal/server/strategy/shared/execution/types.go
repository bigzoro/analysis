package execution

import (
	"time"
)

// ============================================================================
// 策略执行相关类型定义
// ============================================================================

// ExecutionResult 执行结果
type ExecutionResult struct {
	Action     string    `json:"action"`     // 执行动作: "buy", "sell", "skip", "no_op"
	Reason     string    `json:"reason"`     // 执行理由
	Multiplier float64   `json:"multiplier"` // 杠杆倍数
	Symbol     string    `json:"symbol"`     // 交易对
	Timestamp  time.Time `json:"timestamp"`  // 执行时间

	// 风险管理相关
	StopLossPrice   float64 `json:"stop_loss_price,omitempty"`   // 止损价格
	TakeProfitPrice float64 `json:"take_profit_price,omitempty"` // 止盈价格
	MaxPositionSize float64 `json:"max_position_size,omitempty"` // 最大仓位比例
	MaxHoldHours    int     `json:"max_hold_hours,omitempty"`    // 最大持仓小时数
	RiskLevel       float64 `json:"risk_level,omitempty"`        // 风险等级 (0-1)

	// 执行详情
	OrderID       string  `json:"order_id,omitempty"`       // 订单ID
	ExecutedQty   float64 `json:"executed_qty,omitempty"`   // 执行数量
	ExecutedPrice float64 `json:"executed_price,omitempty"` // 执行价格
	Fee           float64 `json:"fee,omitempty"`            // 手续费
}

// MarketData 市场数据（执行时使用）
type MarketData struct {
	Symbol      string  `json:"symbol"`
	Price       float64 `json:"price"`        // 当前价格
	Volume      float64 `json:"volume"`       // 交易量
	MarketCap   float64 `json:"market_cap"`   // 市值
	GainersRank int     `json:"gainers_rank"` // 涨幅排名
	HasSpot     bool    `json:"has_spot"`     // 是否有现货交易
	HasFutures  bool    `json:"has_futures"`  // 是否有期货交易

	// 技术指标
	SMA5           float64 `json:"sma5,omitempty"`            // 5日均线
	SMA10          float64 `json:"sma10,omitempty"`           // 10日均线
	SMA20          float64 `json:"sma20,omitempty"`           // 20日均线
	SMA50          float64 `json:"sma50,omitempty"`           // 50日均线
	RSI            float64 `json:"rsi,omitempty"`             // RSI指标
	MACD           float64 `json:"macd,omitempty"`            // MACD指标
	BollingerUpper float64 `json:"bollinger_upper,omitempty"` // 布林带上轨
	BollingerLower float64 `json:"bollinger_lower,omitempty"` // 布林带下轨

	// 其他市场数据
	Change24h float64 `json:"change_24h"` // 24小时涨跌幅
	High24h   float64 `json:"high_24h"`   // 24小时最高价
	Low24h    float64 `json:"low_24h"`    // 24小时最低价
}

// ExecutionContext 执行上下文
type ExecutionContext struct {
	Symbol         string    `json:"symbol"`          // 交易对
	StrategyType   string    `json:"strategy_type"`   // 策略类型
	UserID         uint      `json:"user_id"`         // 用户ID
	RequestID      string    `json:"request_id"`      // 请求ID
	Timestamp      time.Time `json:"timestamp"`       // 请求时间
	TimeoutSeconds int       `json:"timeout_seconds"` // 超时时间（秒）
}

// ExecutionConfig 执行配置
type ExecutionConfig struct {
	// 基础配置
	Enabled bool `json:"enabled"` // 是否启用执行

	// 风险控制
	MaxSlippagePercent float64 `json:"max_slippage_percent"` // 最大滑点百分比
	MaxPositionSize    float64 `json:"max_position_size"`    // 最大仓位比例
	MaxHoldHours       int     `json:"max_hold_hours"`       // 最大持仓小时数
	StopLossEnabled    bool    `json:"stop_loss_enabled"`    // 是否启用止损
	TakeProfitEnabled  bool    `json:"take_profit_enabled"`  // 是否启用止盈

	// 执行参数
	TimeoutSeconds    int `json:"timeout_seconds"`     // 执行超时时间
	RetryCount        int `json:"retry_count"`         // 重试次数
	RetryDelaySeconds int `json:"retry_delay_seconds"` // 重试间隔

	// 特殊策略参数
	AllowLeverage   bool `json:"allow_leverage"`   // 是否允许杠杆
	DefaultLeverage int  `json:"default_leverage"` // 默认杠杆倍数
}
