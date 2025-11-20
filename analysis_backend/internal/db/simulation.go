package db

import (
	"database/sql"
	"time"

	"gorm.io/gorm"
)

// BacktestRecord 回测记录
type BacktestRecord struct {
	ID                uint      `gorm:"primaryKey" json:"id"`
	RecommendationID  uint      `gorm:"column:recommendation_id;index" json:"recommendation_id"` // 关联推荐ID
	Symbol            string    `gorm:"column:symbol;size:32;index" json:"symbol"`
	BaseSymbol        string    `gorm:"column:base_symbol;size:16" json:"base_symbol"`
	RecommendedAt     time.Time `gorm:"column:recommended_at;index" json:"recommended_at"` // 推荐时间
	RecommendedPrice  string    `gorm:"column:recommended_price;type:decimal(20,8)" json:"recommended_price"` // 推荐时价格
	
	// 回测结果
	PriceAfter24h     *string   `gorm:"column:price_after_24h;type:decimal(20,8)" json:"price_after_24h"` // 24h后价格
	PriceAfter7d      *string   `gorm:"column:price_after_7d;type:decimal(20,8)" json:"price_after_7d"`  // 7天后价格
	PriceAfter30d     *string   `gorm:"column:price_after_30d;type:decimal(20,8)" json:"price_after_30d"` // 30天后价格
	
	Performance24h    *float64  `gorm:"column:performance_24h;type:decimal(10,4)" json:"performance_24h"`  // 24h收益率 %
	Performance7d     *float64  `gorm:"column:performance_7d;type:decimal(10,4)" json:"performance_7d"`   // 7天收益率 %
	Performance30d    *float64  `gorm:"column:performance_30d;type:decimal(10,4)" json:"performance_30d"` // 30天收益率 %
	
	Status            string    `gorm:"column:status;size:16;default:pending" json:"status"` // pending/completed/failed
	CreatedAt         time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt         time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 指定表名
func (BacktestRecord) TableName() string {
	return "backtest_records"
}

// SimulatedTrade 模拟交易
type SimulatedTrade struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	UserID        uint      `gorm:"column:user_id;index" json:"user_id"`
	RecommendationID *uint  `gorm:"column:recommendation_id;index" json:"recommendation_id"` // 可选的推荐ID
	Symbol        string    `gorm:"column:symbol;size:32;index" json:"symbol"`
	BaseSymbol    string    `gorm:"column:base_symbol;size:16" json:"base_symbol"`
	Kind          string    `gorm:"column:kind;size:16" json:"kind"` // spot/futures
	
	// 交易信息
	Side          string    `gorm:"column:side;size:8" json:"side"` // BUY/SELL
	Quantity      string    `gorm:"column:quantity;type:decimal(20,8)" json:"quantity"` // 数量
	Price         string    `gorm:"column:price;type:decimal(20,8)" json:"price"` // 成交价格
	TotalValue    string    `gorm:"column:total_value;type:decimal(20,8)" json:"total_value"` // 总价值
	
	// 持仓信息（仅买入时）
	IsOpen        bool      `gorm:"column:is_open;default:true" json:"is_open"` // 是否持仓中
	CurrentPrice  *string   `gorm:"column:current_price;type:decimal(20,8)" json:"current_price"` // 当前价格
	UnrealizedPnl *string   `gorm:"column:unrealized_pnl;type:decimal(20,8)" json:"unrealized_pnl"` // 未实现盈亏
	UnrealizedPnlPercent *float64 `gorm:"column:unrealized_pnl_percent;type:decimal(10,4)" json:"unrealized_pnl_percent"` // 未实现盈亏百分比
	
	// 卖出信息（仅卖出时）
	SoldAt        *time.Time `gorm:"column:sold_at" json:"sold_at"`
	RealizedPnl   *string    `gorm:"column:realized_pnl;type:decimal(20,8)" json:"realized_pnl"` // 已实现盈亏
	RealizedPnlPercent *float64 `gorm:"column:realized_pnl_percent;type:decimal(10,4)" json:"realized_pnl_percent"` // 已实现盈亏百分比
	
	CreatedAt     time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 指定表名
func (SimulatedTrade) TableName() string {
	return "simulated_trades"
}

// CreateBacktestRecord 创建回测记录
func CreateBacktestRecord(gdb *gorm.DB, rec *BacktestRecord) error {
	return gdb.Create(rec).Error
}

// UpdateBacktestRecord 更新回测记录
func UpdateBacktestRecord(gdb *gorm.DB, rec *BacktestRecord) error {
	return gdb.Save(rec).Error
}

// GetBacktestRecords 获取回测记录
func GetBacktestRecords(gdb *gorm.DB, limit int) ([]BacktestRecord, error) {
	var records []BacktestRecord
	q := gdb.Order("recommended_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&records).Error
	return records, err
}

// GetBacktestStats 获取回测统计
func GetBacktestStats(gdb *gorm.DB) (map[string]interface{}, error) {
	var total int64
	var completed int64
	var avg24h, avg7d, avg30d sql.NullFloat64
	var winRate24h, winRate7d, winRate30d float64
	
	// 总数
	gdb.Model(&BacktestRecord{}).Count(&total)
	
	// 已完成数
	gdb.Model(&BacktestRecord{}).Where("status = ?", "completed").Count(&completed)
	
	// 平均收益率（使用 sql.NullFloat64 处理 NULL 值）
	gdb.Model(&BacktestRecord{}).
		Where("performance_24h IS NOT NULL").
		Select("AVG(performance_24h)").Scan(&avg24h)
	gdb.Model(&BacktestRecord{}).
		Where("performance_7d IS NOT NULL").
		Select("AVG(performance_7d)").Scan(&avg7d)
	gdb.Model(&BacktestRecord{}).
		Where("performance_30d IS NOT NULL").
		Select("AVG(performance_30d)").Scan(&avg30d)
	
	// 胜率（收益率>0的比例）
	var winCount24h, winCount7d, winCount30d int64
	gdb.Model(&BacktestRecord{}).
		Where("performance_24h > 0").Count(&winCount24h)
	gdb.Model(&BacktestRecord{}).
		Where("performance_7d > 0").Count(&winCount7d)
	gdb.Model(&BacktestRecord{}).
		Where("performance_30d > 0").Count(&winCount30d)
	
	if completed > 0 {
		winRate24h = float64(winCount24h) / float64(completed) * 100
		winRate7d = float64(winCount7d) / float64(completed) * 100
		winRate30d = float64(winCount30d) / float64(completed) * 100
	}
	
	// 转换 NullFloat64 为普通 float64（如果为 NULL 则返回 0）
	var avg24hVal, avg7dVal, avg30dVal float64
	if avg24h.Valid {
		avg24hVal = avg24h.Float64
	}
	if avg7d.Valid {
		avg7dVal = avg7d.Float64
	}
	if avg30d.Valid {
		avg30dVal = avg30d.Float64
	}
	
	return map[string]interface{}{
		"total": total,
		"completed": completed,
		"avg_performance_24h": avg24hVal,
		"avg_performance_7d": avg7dVal,
		"avg_performance_30d": avg30dVal,
		"win_rate_24h": winRate24h,
		"win_rate_7d": winRate7d,
		"win_rate_30d": winRate30d,
	}, nil
}

// CreateSimulatedTrade 创建模拟交易
func CreateSimulatedTrade(gdb *gorm.DB, trade *SimulatedTrade) error {
	return gdb.Create(trade).Error
}

// UpdateSimulatedTrade 更新模拟交易
func UpdateSimulatedTrade(gdb *gorm.DB, trade *SimulatedTrade) error {
	return gdb.Save(trade).Error
}

// GetSimulatedTrades 获取用户的模拟交易
func GetSimulatedTrades(gdb *gorm.DB, userID uint, isOpen *bool) ([]SimulatedTrade, error) {
	var trades []SimulatedTrade
	q := gdb.Where("user_id = ?", userID)
	if isOpen != nil {
		q = q.Where("is_open = ?", *isOpen)
	}
	err := q.Order("created_at DESC").Find(&trades).Error
	return trades, err
}

// GetSimulatedTradeByID 根据ID获取模拟交易
func GetSimulatedTradeByID(gdb *gorm.DB, id uint, userID uint) (*SimulatedTrade, error) {
	var trade SimulatedTrade
	err := gdb.Where("id = ? AND user_id = ?", id, userID).First(&trade).Error
	if err != nil {
		return nil, err
	}
	return &trade, nil
}
