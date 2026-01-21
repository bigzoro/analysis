package db

import "time"

type ScheduledOrder struct {
	ID               uint   `gorm:"primaryKey"                      json:"id"`
	UserID           uint   `gorm:"index;not null"                  json:"user_id"`
	Exchange         string `gorm:"size:32;not null"                json:"exchange"` // binance_futures
	Testnet          bool   `gorm:"not null"                        json:"testnet"`
	Symbol           string `gorm:"size:32;not null"                json:"symbol"`            // ETHUSDT
	Side             string `gorm:"size:8;not null"                 json:"side"`              // BUY / SELL
	OrderType        string `gorm:"size:32;not null"                json:"order_type"`        // MARKET / LIMIT / TAKE_PROFIT_MARKET / STOP_MARKET
	Quantity         string `gorm:"size:64;not null"                json:"quantity"`          // 原样字符串，避免精度问题
	AdjustedQuantity string `gorm:"size:64"                         json:"adjusted_quantity"` // 调整后的数量
	Price            string `gorm:"size:64"                         json:"price"`             // 限价单需要
	Leverage         int    `gorm:"not null;default:0"              json:"leverage"`          // >0 则下单前设置
	ReduceOnly       bool   `gorm:"not null;default:false"          json:"reduce_only"`
	StrategyID       *uint  `gorm:"index"                           json:"strategy_id"`  // 可选策略ID
	ExecutionID      *uint  `gorm:"index"                           json:"execution_id"` // 关联的策略执行ID

	// --- Bracket (一键三连) ---
	BracketEnabled  bool    `gorm:"not null;default:false"        json:"bracket_enabled"`
	TPPercent       float64 `json:"tp_percent"`        // 用户设置的止盈百分比
	SLPercent       float64 `json:"sl_percent"`        // 用户设置的止损百分比
	ActualTPPercent float64 `json:"actual_tp_percent"` // 实际使用的止盈百分比（自动调整后的）
	ActualSLPercent float64 `json:"actual_sl_percent"` // 实际使用的止损百分比（自动调整后的）
	TPPrice         string  `gorm:"size:64"                       json:"tp_price"`
	SLPrice         string  `gorm:"size:64"                       json:"sl_price"`
	WorkingType     string  `gorm:"size:16;not null;default:MARK_PRICE" json:"working_type"` // MARK_PRICE / CONTRACT_PRICE

	TriggerTime time.Time `gorm:"index;not null"                 json:"trigger_time"`
	Status      string    `gorm:"size:16;not null;default:pending" json:"status"` // pending/processing/sent/filled/canceled/failed
	Result      string    `gorm:"type:text"                      json:"result"`

	// 订单跟踪字段
	ClientOrderId   string `gorm:"size:64"                      json:"client_order_id"`   // 客户端订单ID
	ExchangeOrderId string `gorm:"size:64"                      json:"exchange_order_id"` // 交易所订单ID
	ExecutedQty     string `gorm:"size:64"                      json:"executed_quantity"` // 已执行数量
	AvgPrice        string `gorm:"size:64"                      json:"avg_price"`         // 平均成交价

	// 订单关联字段
	ParentOrderId uint   `gorm:"default:0"                      json:"parent_order_id"` // 父订单ID (平仓订单指向开仓订单)
	CloseOrderIds string `gorm:"type:text"                      json:"close_order_ids"` // 关联的平仓订单ID列表 (JSON格式)

	// 策略相关字段
	StrategyType string `gorm:"size:32"                        json:"strategy_type"` // 策略类型 (grid_trading, etc.)
	GridLevel    int    `gorm:"default:0"                       json:"grid_level"`   // 网格层级 (仅网格交易使用)

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ExternalOperation 外部操作记录（用户在官网手动操作的记录）
type ExternalOperation struct {
	ID            uint       `gorm:"primaryKey"                      json:"id"`
	Symbol        string     `gorm:"size:32;not null;index"          json:"symbol"`         // 交易对
	OperationType string     `gorm:"size:32;not null"                json:"operation_type"` // 操作类型: external_open, external_close, external_partial_close, external_add_position
	OldAmount     string     `gorm:"size:64"                         json:"old_amount"`     // 操作前的持仓数量
	NewAmount     string     `gorm:"size:64;not null"                json:"new_amount"`     // 操作后的持仓数量
	Confidence    float64    `gorm:"not null;default:0"              json:"confidence"`     // 检测置信度 0-1
	DetectedAt    time.Time  `gorm:"not null"                        json:"detected_at"`    // 检测时间
	Status        string     `gorm:"size:16;not null;default:detected" json:"status"`       // 状态: detected, processed, ignored
	ProcessedAt   *time.Time `json:"processed_at"`                                          // 处理时间
	Notes         string     `gorm:"type:text"                       json:"notes"`          // 备注信息
	UserID        uint       `gorm:"index"                           json:"user_id"`        // 关联用户ID（如果能确定）

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// OperationLog 操作日志记录
type OperationLog struct {
	ID          uint       `gorm:"primaryKey"                      json:"id"`
	UserID      uint       `gorm:"index;not null"                  json:"user_id"`
	EntityType  string     `gorm:"size:32;not null;index"          json:"entity_type"` // 操作实体类型: order, position, strategy, system
	EntityID    uint       `gorm:"index"                           json:"entity_id"`   // 实体ID
	Action      string     `gorm:"size:64;not null;index"          json:"action"`      // 操作动作: create, update, delete, sync, detect, notify
	Description string     `gorm:"type:text;not null"              json:"description"` // 操作描述
	OldValue    string     `gorm:"type:text"                       json:"old_value"`   // 操作前的值 (JSON格式)
	NewValue    string     `gorm:"type:text"                       json:"new_value"`   // 操作后的值 (JSON格式)
	IPAddress   string     `gorm:"size:45"                         json:"ip_address"`  // 操作IP地址
	UserAgent   string     `gorm:"size:512"                        json:"user_agent"`  // 用户代理
	Source      string     `gorm:"size:32;not null;default:system" json:"source"`      // 操作来源: system, user, external, api
	Level       string     `gorm:"size:16;not null;default:info"   json:"level"`       // 日志级别: debug, info, warn, error
	ErrorMsg    string     `gorm:"type:text"                       json:"error_msg"`   // 错误信息
	Metadata    string     `gorm:"type:text"                       json:"metadata"`    // 额外元数据 (JSON格式)
	ProcessedAt *time.Time `json:"processed_at"`                                       // 处理完成时间

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AuditTrail 审计追踪记录
type AuditTrail struct {
	ID           uint      `gorm:"primaryKey"                    json:"id"`
	SessionID    string    `gorm:"size:64;index"                 json:"session_id"` // 会话ID
	UserID       uint      `gorm:"index;not null"                json:"user_id"`
	Action       string    `gorm:"size:64;not null;index"        json:"action"`        // 审计动作
	ResourceType string    `gorm:"size:32;not null;index"        json:"resource_type"` // 资源类型
	ResourceID   string    `gorm:"size:128;index"                json:"resource_id"`   // 资源ID
	Details      string    `gorm:"type:text"                     json:"details"`       // 详细信息
	OldState     string    `gorm:"type:text"                     json:"old_state"`     // 操作前状态
	NewState     string    `gorm:"type:text"                     json:"new_state"`     // 操作后状态
	IPAddress    string    `gorm:"size:45"                       json:"ip_address"`
	UserAgent    string    `gorm:"size:512"                      json:"user_agent"`
	Success      bool      `gorm:"not null;default:true"        json:"success"`        // 操作是否成功
	ErrorDetails string    `gorm:"type:text"                     json:"error_details"` // 错误详情
	Timestamp    time.Time `gorm:"not null;index"                json:"timestamp"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
