package db

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// BacktestRecord 回测记录
type BacktestRecord struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	RecommendationID uint      `gorm:"column:recommendation_id;index" json:"recommendation_id"` // 关联推荐ID
	Symbol           string    `gorm:"column:symbol;size:32;index" json:"symbol"`
	BaseSymbol       string    `gorm:"column:base_symbol;size:16" json:"base_symbol"`
	RecommendedAt    time.Time `gorm:"column:recommended_at;index" json:"recommended_at"`                    // 推荐时间
	RecommendedPrice string    `gorm:"column:recommended_price;type:decimal(20,8)" json:"recommended_price"` // 推荐时价格

	// 回测结果
	PriceAfter24h *string `gorm:"column:price_after_24h;type:decimal(20,8)" json:"price_after_24h"` // 24h后价格
	PriceAfter7d  *string `gorm:"column:price_after_7d;type:decimal(20,8)" json:"price_after_7d"`   // 7天后价格
	PriceAfter30d *string `gorm:"column:price_after_30d;type:decimal(20,8)" json:"price_after_30d"` // 30天后价格

	Performance24h *float64 `gorm:"column:performance_24h;type:decimal(10,4)" json:"performance_24h"` // 24h收益率 %
	Performance7d  *float64 `gorm:"column:performance_7d;type:decimal(10,4)" json:"performance_7d"`   // 7天收益率 %
	Performance30d *float64 `gorm:"column:performance_30d;type:decimal(10,4)" json:"performance_30d"` // 30天收益率 %

	Status    string    `gorm:"column:status;size:16;default:pending" json:"status"` // pending/completed/failed
	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 指定表名
func (BacktestRecord) TableName() string {
	return "backtest_records"
}

// SimulatedTrade 模拟交易
type SimulatedTrade struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	UserID           uint   `gorm:"column:user_id;index" json:"user_id"`
	RecommendationID *uint  `gorm:"column:recommendation_id;index" json:"recommendation_id"` // 可选的推荐ID
	Symbol           string `gorm:"column:symbol;size:32;index" json:"symbol"`
	BaseSymbol       string `gorm:"column:base_symbol;size:16" json:"base_symbol"`
	Kind             string `gorm:"column:kind;size:16" json:"kind"` // spot/futures

	// 交易信息
	Side       string `gorm:"column:side;size:8" json:"side"`                           // BUY/SELL
	Quantity   string `gorm:"column:quantity;type:decimal(20,8)" json:"quantity"`       // 数量
	Price      string `gorm:"column:price;type:decimal(20,8)" json:"price"`             // 成交价格
	TotalValue string `gorm:"column:total_value;type:decimal(20,8)" json:"total_value"` // 总价值

	// 持仓信息（仅买入时）
	IsOpen               bool     `gorm:"column:is_open;default:true" json:"is_open"`                                     // 是否持仓中
	CurrentPrice         *string  `gorm:"column:current_price;type:decimal(20,8)" json:"current_price"`                   // 当前价格
	UnrealizedPnl        *string  `gorm:"column:unrealized_pnl;type:decimal(20,8)" json:"unrealized_pnl"`                 // 未实现盈亏
	UnrealizedPnlPercent *float64 `gorm:"column:unrealized_pnl_percent;type:decimal(10,4)" json:"unrealized_pnl_percent"` // 未实现盈亏百分比

	// 卖出信息（仅卖出时）
	SoldAt             *time.Time `gorm:"column:sold_at" json:"sold_at"`
	RealizedPnl        *string    `gorm:"column:realized_pnl;type:decimal(20,8)" json:"realized_pnl"`                 // 已实现盈亏
	RealizedPnlPercent *float64   `gorm:"column:realized_pnl_percent;type:decimal(10,4)" json:"realized_pnl_percent"` // 已实现盈亏百分比

	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
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
		"total":               total,
		"completed":           completed,
		"avg_performance_24h": avg24hVal,
		"avg_performance_7d":  avg7dVal,
		"avg_performance_30d": avg30dVal,
		"win_rate_24h":        winRate24h,
		"win_rate_7d":         winRate7d,
		"win_rate_30d":        winRate30d,
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

// AsyncBacktestRecord 异步回测记录
type AsyncBacktestRecord struct {
	ID             uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID         uint            `gorm:"column:user_id;not null;index" json:"user_id"`
	Symbol         string          `gorm:"column:symbol;size:32;not null;index" json:"symbol"`
	Strategy       string          `gorm:"column:strategy;size:32;not null" json:"strategy"`
	StartDate      string          `gorm:"column:start_date;size:10;not null" json:"start_date"`
	EndDate        string          `gorm:"column:end_date;size:10;not null" json:"end_date"`
	InitialCapital decimal.Decimal `gorm:"column:initial_capital;type:decimal(20,8);not null" json:"initial_capital"`
	PositionSize   decimal.Decimal `gorm:"column:position_size;type:decimal(8,2);not null" json:"position_size"`
	Status         string          `gorm:"column:status;size:16;default:'pending';index" json:"status"` // pending/running/completed/failed
	Result         *string         `gorm:"column:result;type:json" json:"result,omitempty"`             // 回测结果JSON字符串
	ErrorMessage   string          `gorm:"column:error_message;type:text" json:"error_message,omitempty"`
	CreatedAt      time.Time       `gorm:"column:created_at;index:idx_created_at" json:"created_at"`
	UpdatedAt      time.Time       `gorm:"column:updated_at" json:"updated_at"`
	CompletedAt    *time.Time      `gorm:"column:completed_at" json:"completed_at,omitempty"`
}

// TableName 指定表名
func (AsyncBacktestRecord) TableName() string {
	return "async_backtest_records"
}

// CreateAsyncBacktestRecord 创建异步回测记录
func CreateAsyncBacktestRecord(gdb *gorm.DB, record *AsyncBacktestRecord) error {
	return gdb.Create(record).Error
}

// UpdateAsyncBacktestRecord 更新异步回测记录
func UpdateAsyncBacktestRecord(gdb *gorm.DB, record *AsyncBacktestRecord) error {
	return gdb.Save(record).Error
}

// GetAsyncBacktestRecords 获取用户的异步回测记录
func GetAsyncBacktestRecords(gdb *gorm.DB, userID uint, page, limit int, status, symbol string) ([]AsyncBacktestRecord, int64, error) {
	var records []AsyncBacktestRecord
	var total int64

	query := gdb.Where("user_id = ?", userID)

	// 添加筛选条件
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if symbol != "" {
		query = query.Where("symbol LIKE ?", "%"+symbol+"%")
	}

	// 获取总数
	if err := query.Model(&AsyncBacktestRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询 - 使用ID降序代替created_at排序，避免排序内存问题
	offset := (page - 1) * limit
	err := query.Order("id DESC").Offset(offset).Limit(limit).Find(&records).Error

	return records, total, err
}

// GetAsyncBacktestRecordByID 根据ID获取异步回测记录
func GetAsyncBacktestRecordByID(gdb *gorm.DB, id uint, userID uint) (*AsyncBacktestRecord, error) {
	var record AsyncBacktestRecord
	err := gdb.Where("id = ? AND user_id = ?", id, userID).First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

// DeleteAsyncBacktestRecord 删除异步回测记录
func DeleteAsyncBacktestRecord(gdb *gorm.DB, id uint, userID uint) error {
	return gdb.Where("id = ? AND user_id = ?", id, userID).Delete(&AsyncBacktestRecord{}).Error
}

// UpdateAsyncBacktestRecordStatus 更新异步回测记录状态
func UpdateAsyncBacktestRecordStatus(gdb *gorm.DB, id uint, userID uint, status string, result *string, errorMessage string, completedAt *time.Time) error {
	updateData := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}

	if result != nil {
		updateData["result"] = *result
	}
	if errorMessage != "" {
		updateData["error_message"] = errorMessage
	}
	if completedAt != nil {
		updateData["completed_at"] = *completedAt
	}

	return gdb.Model(&AsyncBacktestRecord{}).Where("id = ? AND user_id = ?", id, userID).Updates(updateData).Error
}

// UpdateAsyncBacktestRecordSymbol 更新异步回测记录的币种
func UpdateAsyncBacktestRecordSymbol(gdb *gorm.DB, id uint, userID uint, symbol string) error {
	updateData := map[string]interface{}{
		"symbol":     symbol,
		"updated_at": time.Now(),
	}

	return gdb.Model(&AsyncBacktestRecord{}).Where("id = ? AND user_id = ?", id, userID).Updates(updateData).Error
}

// ABTestConfig A/B测试配置
type ABTestConfig struct {
	ID            uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	TestName      string     `gorm:"column:test_name;size:100;uniqueIndex;not null" json:"test_name"`
	Description   string     `gorm:"column:description;type:text" json:"description"`
	Status        string     `gorm:"column:status;size:20;default:'active';index" json:"status"` // active/paused/completed
	Groups        string     `gorm:"column:groups;type:json;not null" json:"groups"`             // JSON string of []ABTestGroup
	TargetMetric  string     `gorm:"column:target_metric;size:50;not null" json:"target_metric"` // clicks/conversions/rating
	MinSampleSize int        `gorm:"column:min_sample_size;not null" json:"min_sample_size"`
	StartTime     time.Time  `gorm:"column:start_time;not null" json:"start_time"`
	EndTime       *time.Time `gorm:"column:end_time" json:"end_time,omitempty"`
	CreatedBy     uint       `gorm:"column:created_by;not null;index" json:"created_by"`
	Metadata      string     `gorm:"column:metadata;type:json" json:"metadata"` // JSON string of map[string]interface{}
	CreatedAt     time.Time  `gorm:"column:created_at" json:"created_at"`
	UpdatedAt     time.Time  `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 指定表名
func (ABTestConfig) TableName() string {
	return "ab_test_configs"
}

// ABTestResult A/B测试结果
type ABTestResult struct {
	ID              uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	TestName        string    `gorm:"column:test_name;size:100;not null;index" json:"test_name"`
	GroupResults    string    `gorm:"column:group_results;type:json;not null" json:"group_results"` // JSON string of []GroupResult
	BestGroup       string    `gorm:"column:best_group;size:50;not null" json:"best_group"`
	ConfidenceLevel float64   `gorm:"column:confidence_level;type:decimal(5,4);not null" json:"confidence_level"`
	StatisticalSig  float64   `gorm:"column:statistical_significance;type:decimal(5,4);not null" json:"statistical_significance"`
	ImprovementRate float64   `gorm:"column:improvement_rate;type:decimal(6,4);not null" json:"improvement_rate"`
	SampleSize      int       `gorm:"column:sample_size;not null" json:"sample_size"`
	Duration        int64     `gorm:"column:duration;not null" json:"duration"` // 存储纳秒数
	Recommendation  string    `gorm:"column:recommendation;type:text" json:"recommendation"`
	CreatedAt       time.Time `gorm:"column:created_at" json:"created_at"`
}

// TableName 指定表名
func (ABTestResult) TableName() string {
	return "ab_test_results"
}

// CreateABTestConfig 创建A/B测试配置
func CreateABTestConfig(gdb *gorm.DB, config *ABTestConfig) error {
	return gdb.Create(config).Error
}

// GetABTestConfigs 获取A/B测试配置列表
func GetABTestConfigs(gdb *gorm.DB, status string, limit int) ([]ABTestConfig, error) {
	var configs []ABTestConfig
	query := gdb.Order("created_at DESC")
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&configs).Error
	return configs, err
}

// GetABTestConfigByName 根据测试名称获取配置
func GetABTestConfigByName(gdb *gorm.DB, testName string) (*ABTestConfig, error) {
	var config ABTestConfig
	err := gdb.Where("test_name = ?", testName).First(&config).Error
	if err != nil {
		return nil, err
	}
	return &config, nil
}

// UpdateABTestConfig 更新A/B测试配置
func UpdateABTestConfig(gdb *gorm.DB, config *ABTestConfig) error {
	return gdb.Save(config).Error
}

// DeleteABTestConfig 删除A/B测试配置
func DeleteABTestConfig(gdb *gorm.DB, testName string) error {
	return gdb.Where("test_name = ?", testName).Delete(&ABTestConfig{}).Error
}

// CreateABTestResult 创建A/B测试结果
func CreateABTestResult(gdb *gorm.DB, result *ABTestResult) error {
	return gdb.Create(result).Error
}

// AsyncBacktestTrade 异步回测交易记录
type AsyncBacktestTrade struct {
	ID               uint            `gorm:"primarykey" json:"id"`
	BacktestRecordID uint            `gorm:"column:backtest_record_id;not null;index:idx_backtest_record" json:"backtest_record_id"`
	Timestamp        time.Time       `gorm:"column:timestamp;index" json:"timestamp"`
	Symbol           string          `gorm:"column:symbol;size:20;not null" json:"symbol"`
	Side             string          `gorm:"column:side;size:10;not null" json:"side"` // buy/sell
	Price            decimal.Decimal `gorm:"column:price;type:decimal(20,8);not null" json:"price"`
	Quantity         decimal.Decimal `gorm:"column:quantity;type:decimal(20,8);not null" json:"quantity"`
	Value            decimal.Decimal `gorm:"column:value;type:decimal(20,8);not null" json:"value"`              // 成交金额
	Commission       decimal.Decimal `gorm:"column:commission;type:decimal(20,8);default:0" json:"commission"`   // 手续费
	PnL              decimal.Decimal `gorm:"column:pnl;type:decimal(20,8);default:0" json:"pnl"`                 // 盈亏
	PnLPercent       decimal.Decimal `gorm:"column:pnl_percent;type:decimal(20,8);default:0" json:"pnl_percent"` // 盈亏百分比
	CreatedAt        time.Time       `gorm:"column:created_at" json:"created_at"`
}

func (AsyncBacktestTrade) TableName() string {
	return "async_backtest_trades"
}

// CreateAsyncBacktestTrades 批量创建回测交易记录
func CreateAsyncBacktestTrades(gdb *gorm.DB, trades []AsyncBacktestTrade) error {
	if len(trades) == 0 {
		return nil
	}
	return gdb.CreateInBatches(trades, 1000).Error
}

// GetAsyncBacktestTrades 分页获取回测交易记录
func GetAsyncBacktestTrades(gdb *gorm.DB, backtestRecordID uint, page, limit int, sortBy, sortOrder string) ([]AsyncBacktestTrade, int64, error) {
	var trades []AsyncBacktestTrade
	var total int64

	query := gdb.Model(&AsyncBacktestTrade{}).Where("backtest_record_id = ?", backtestRecordID)

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 排序
	if sortBy == "" {
		sortBy = "timestamp"
	}
	if sortOrder == "" {
		sortOrder = "asc"
	}
	orderClause := fmt.Sprintf("%s %s", sortBy, sortOrder)
	query = query.Order(orderClause)

	// 分页
	offset := (page - 1) * limit
	query = query.Offset(offset).Limit(limit)

	if err := query.Find(&trades).Error; err != nil {
		return nil, 0, err
	}

	return trades, total, nil
}

// DeleteAsyncBacktestTrades 删除指定回测记录的所有交易记录
func DeleteAsyncBacktestTrades(gdb *gorm.DB, backtestRecordID uint) error {
	return gdb.Where("backtest_record_id = ?", backtestRecordID).Delete(&AsyncBacktestTrade{}).Error
}

// GetABTestResults 获取A/B测试结果
func GetABTestResults(gdb *gorm.DB, testName string) ([]ABTestResult, error) {
	var results []ABTestResult
	err := gdb.Where("test_name = ?", testName).Order("created_at DESC").Find(&results).Error
	return results, err
}
