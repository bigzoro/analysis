package db

import "time"

type ScheduledOrder struct {
	ID         uint   `gorm:"primaryKey"                      json:"id"`
	UserID     uint   `gorm:"index;not null"                  json:"user_id"`
	Exchange   string `gorm:"size:32;not null"                json:"exchange"` // binance_futures
	Testnet    bool   `gorm:"not null"                        json:"testnet"`
	Symbol     string `gorm:"size:32;not null"                json:"symbol"`     // ETHUSDT
	Side       string `gorm:"size:8;not null"                 json:"side"`       // BUY / SELL
	OrderType  string `gorm:"size:16;not null"                json:"order_type"` // MARKET / LIMIT
	Quantity   string `gorm:"size:64;not null"                json:"quantity"`   // 原样字符串，避免精度问题
	Price      string `gorm:"size:64"                         json:"price"`      // 限价单需要
	Leverage   int    `gorm:"not null;default:0"              json:"leverage"`   // >0 则下单前设置
	ReduceOnly bool   `gorm:"not null;default:false"          json:"reduce_only"`

	// --- Bracket (一键三连) ---
	BracketEnabled bool    `gorm:"not null;default:false"        json:"bracket_enabled"`
	TPPercent      float64 `json:"tp_percent"`
	SLPercent      float64 `json:"sl_percent"`
	TPPrice        string  `gorm:"size:64"                       json:"tp_price"`
	SLPrice        string  `gorm:"size:64"                       json:"sl_price"`
	WorkingType    string  `gorm:"size:16;not null;default:MARK_PRICE" json:"working_type"` // MARK_PRICE / CONTRACT_PRICE

	TriggerTime time.Time `gorm:"index;not null"                 json:"trigger_time"`
	Status      string    `gorm:"size:16;not null;default:pending" json:"status"` // pending/processing/sent/filled/canceled/failed
	Result      string    `gorm:"type:text"                      json:"result"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
