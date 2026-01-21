package db

import (
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// RecommendationPerformance 推荐历史表现追踪
type RecommendationPerformance struct {
	ID               uint   `gorm:"primaryKey" json:"id"`
	RecommendationID uint   `gorm:"column:recommendation_id;index:idx_rec_id" json:"recommendation_id"` // 关联推荐ID
	Symbol           string `gorm:"column:symbol;size:32;index" json:"symbol"`
	BaseSymbol       string `gorm:"column:base_symbol;size:16;index" json:"base_symbol"`
	Kind             string `gorm:"column:kind;size:16;index" json:"kind"` // spot/futures

	// 推荐时的数据快照
	RecommendedAt    time.Time `gorm:"column:recommended_at;index" json:"recommended_at"`                    // 推荐时间
	RecommendedPrice float64   `gorm:"column:recommended_price;type:decimal(20,8)" json:"recommended_price"` // 推荐时价格
	TotalScore       float64   `gorm:"column:total_score;type:decimal(5,2)" json:"total_score"`              // 推荐总分
	MarketScore      float64   `gorm:"column:market_score;type:decimal(5,2)" json:"market_score"`
	FlowScore        float64   `gorm:"column:flow_score;type:decimal(5,2)" json:"flow_score"`
	HeatScore        float64   `gorm:"column:heat_score;type:decimal(5,2)" json:"heat_score"`
	EventScore       float64   `gorm:"column:event_score;type:decimal(5,2)" json:"event_score"`
	SentimentScore   float64   `gorm:"column:sentiment_score;type:decimal(5,2)" json:"sentiment_score"`

	// 价格追踪（实时更新）
	Price1h       *float64   `gorm:"column:price_1h;type:decimal(20,8)" json:"price_1h"`           // 1小时后价格
	Price24h      *float64   `gorm:"column:price_24h;type:decimal(20,8)" json:"price_24h"`         // 24小时后价格（回测用）
	Price7d       *float64   `gorm:"column:price_7d;type:decimal(20,8)" json:"price_7d"`           // 7天后价格（回测用）
	Price30d      *float64   `gorm:"column:price_30d;type:decimal(20,8)" json:"price_30d"`         // 30天后价格（回测用）
	CurrentPrice  *float64   `gorm:"column:current_price;type:decimal(20,8)" json:"current_price"` // 当前价格
	LastUpdatedAt *time.Time `gorm:"column:last_updated_at" json:"last_updated_at"`                // 最后更新时间

	// 回测状态（统一回测和表现追踪）
	BacktestStatus string `gorm:"column:backtest_status;size:16;default:pending;index" json:"backtest_status"` // pending/completed/failed

	// 收益率计算
	Return1h      *float64 `gorm:"column:return_1h;type:decimal(10,4)" json:"return_1h"`           // 1h收益率 %
	Return24h     *float64 `gorm:"column:return_24h;type:decimal(10,4)" json:"return_24h"`         // 24h收益率 %
	Return7d      *float64 `gorm:"column:return_7d;type:decimal(10,4)" json:"return_7d"`           // 7天收益率 %
	Return30d     *float64 `gorm:"column:return_30d;type:decimal(10,4)" json:"return_30d"`         // 30天收益率 %
	CurrentReturn *float64 `gorm:"column:current_return;type:decimal(10,4)" json:"current_return"` // 当前收益率 %

	// 风险指标
	MaxDrawdown *float64 `gorm:"column:max_drawdown;type:decimal(10,4)" json:"max_drawdown"` // 最大回撤 %
	MaxGain     *float64 `gorm:"column:max_gain;type:decimal(10,4)" json:"max_gain"`         // 最大涨幅 %

	// 表现评级
	PerformanceRating *string `gorm:"column:performance_rating;size:16" json:"performance_rating"` // excellent/good/average/poor
	IsWin             *bool   `gorm:"column:is_win" json:"is_win"`                                 // 是否盈利（24h）

	// 策略执行信息 (新增)
	EntryPrice *float64   `gorm:"column:entry_price;type:decimal(20,8)" json:"entry_price"` // 实际入场价格
	EntryTime  *time.Time `gorm:"column:entry_time" json:"entry_time"`                      // 实际入场时间
	ExitPrice  *float64   `gorm:"column:exit_price;type:decimal(20,8)" json:"exit_price"`   // 实际出场价格
	ExitTime   *time.Time `gorm:"column:exit_time" json:"exit_time"`                        // 实际出场时间
	ExitReason string     `gorm:"column:exit_reason;size:32" json:"exit_reason"`            // 退出原因: profit/loss/time/max_hold/force

	// 策略参数快照 (新增)
	StrategyConfig  datatypes.JSON `gorm:"column:strategy_config;type:json" json:"strategy_config"`   // 策略配置参数
	EntryConditions datatypes.JSON `gorm:"column:entry_conditions;type:json" json:"entry_conditions"` // 入场条件快照
	ExitConditions  datatypes.JSON `gorm:"column:exit_conditions;type:json" json:"exit_conditions"`   // 出场条件快照

	// 策略绩效指标 (新增)
	ActualReturn          *float64 `gorm:"column:actual_return;type:decimal(10,4)" json:"actual_return"`                     // 实际收益率 (基于策略)
	HoldingPeriod         *int     `gorm:"column:holding_period" json:"holding_period"`                                      // 持仓周期(分钟)
	MaxFavorableExcursion *float64 `gorm:"column:max_favorable_excursion;type:decimal(10,4)" json:"max_favorable_excursion"` // 最大有利变动(MFE)
	MaxAdverseExcursion   *float64 `gorm:"column:max_adverse_excursion;type:decimal(10,4)" json:"max_adverse_excursion"`     // 最大不利变动(MAE)

	// 状态
	Status      string     `gorm:"column:status;size:16;default:tracking;index" json:"status"` // tracking/completed/expired
	CompletedAt *time.Time `gorm:"column:completed_at" json:"completed_at"`                    // 完成时间（30天后）

	CreatedAt time.Time `gorm:"column:created_at" json:"created_at"`
	UpdatedAt time.Time `gorm:"column:updated_at" json:"updated_at"`
}

// TableName 指定表名
func (RecommendationPerformance) TableName() string {
	return "recommendation_performances"
}

// CreateRecommendationPerformance 创建推荐表现追踪记录
func CreateRecommendationPerformance(gdb *gorm.DB, rec *RecommendationPerformance) error {
	return gdb.Create(rec).Error
}

// UpdateRecommendationPerformance 更新推荐表现追踪记录
func UpdateRecommendationPerformance(gdb *gorm.DB, rec *RecommendationPerformance) error {
	return gdb.Save(rec).Error
}

// BatchUpdateRecommendationPerformances 批量更新推荐表现追踪记录（性能优化）
func BatchUpdateRecommendationPerformances(gdb *gorm.DB, perfs []RecommendationPerformance) error {
	if len(perfs) == 0 {
		return nil
	}

	// 使用批量更新，减少数据库往返
	// 注意：GORM的Save会更新所有字段，如果只需要更新部分字段，可以使用Updates
	for _, perf := range perfs {
		if err := gdb.Save(&perf).Error; err != nil {
			return err
		}
	}

	return nil
}

// BatchUpdateRecommendationPerformancesSelective 选择性批量更新（只更新指定字段）
func BatchUpdateRecommendationPerformancesSelective(gdb *gorm.DB, updates []struct {
	ID     uint
	Fields map[string]interface{}
}) error {
	if len(updates) == 0 {
		return nil
	}

	// 使用事务批量更新
	return gdb.Transaction(func(tx *gorm.DB) error {
		for _, update := range updates {
			if err := tx.Model(&RecommendationPerformance{}).
				Where("id = ?", update.ID).
				Updates(update.Fields).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// GetRecommendationPerformance 根据推荐ID获取表现追踪记录
func GetRecommendationPerformance(gdb *gorm.DB, recommendationID uint) (*RecommendationPerformance, error) {
	var perf RecommendationPerformance
	// 使用 Find 而不是 First，避免记录不存在时的日志输出
	err := gdb.Where("recommendation_id = ?", recommendationID).Limit(1).Find(&perf).Error
	if err != nil {
		return nil, err
	}
	// 检查是否找到记录（通过主键是否为0判断）
	if perf.ID == 0 {
		return nil, nil
	}
	return &perf, nil
}

// GetRecommendationPerformanceByID 根据主键ID获取表现追踪记录
func GetRecommendationPerformanceByID(gdb *gorm.DB, id uint) (*RecommendationPerformance, error) {
	var perf RecommendationPerformance
	err := gdb.Where("id = ?", id).First(&perf).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &perf, nil
}

// GetPerformanceBySymbol 根据币种获取最新的表现追踪记录
func GetPerformanceBySymbol(gdb *gorm.DB, symbol string, limit int) ([]RecommendationPerformance, error) {
	var perfs []RecommendationPerformance
	q := gdb.Where("symbol = ?", symbol).Order("recommended_at DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&perfs).Error
	return perfs, err
}

// GetPerformanceByRecommendationIDs 批量根据推荐ID获取表现追踪记录
func GetPerformanceByRecommendationIDs(gdb *gorm.DB, recommendationIDs []uint) ([]RecommendationPerformance, error) {
	if len(recommendationIDs) == 0 {
		return []RecommendationPerformance{}, nil
	}
	var perfs []RecommendationPerformance
	err := gdb.Where("recommendation_id IN ?", recommendationIDs).
		Order("recommended_at DESC").
		Find(&perfs).Error
	return perfs, err
}

// GetTrackingPerformances 获取需要追踪的表现记录（未完成的）
func GetTrackingPerformances(gdb *gorm.DB, limit int) ([]RecommendationPerformance, error) {
	var perfs []RecommendationPerformance
	q := gdb.Where("status = ?", "tracking").Order("recommended_at ASC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&perfs).Error
	return perfs, err
}

// GetPendingBacktests 获取待更新的回测记录
func GetPendingBacktests(gdb *gorm.DB, limit int) ([]RecommendationPerformance, error) {
	var perfs []RecommendationPerformance
	now := time.Now().UTC()
	time24hAgo := now.Add(-24 * time.Hour)
	time7dAgo := now.Add(-7 * 24 * time.Hour)
	time30dAgo := now.Add(-30 * 24 * time.Hour)

	// 获取需要更新的记录：
	// 1. 回测状态为 pending 或 tracking
	// 2. 或者已经过了24小时但Return24h还是nil（即使backtest_status是completed）
	// 3. 或者已经过了7天但Return7d还是nil
	// 4. 或者已经过了30天但Return30d还是nil
	// 使用原生SQL查询以确保正确性
	err := gdb.Where(
		"recommended_at <= ? AND ("+
			"backtest_status IN (?, ?) OR "+
			"(recommended_at <= ? AND return_24h IS NULL) OR "+
			"(recommended_at <= ? AND return_7d IS NULL) OR "+
			"(recommended_at <= ? AND return_30d IS NULL)"+
			")",
		now,
		"pending", "tracking",
		time24hAgo,
		time7dAgo,
		time30dAgo,
	).
		Order("recommended_at ASC").
		Limit(limit).
		Find(&perfs).Error

	return perfs, err
}

// GetPerformancesNeedingStrategyBacktest 获取需要策略回测的记录
func GetPerformancesNeedingStrategyBacktest(gdb *gorm.DB, limit int) ([]RecommendationPerformance, error) {
	var perfs []RecommendationPerformance
	// 修改为查找所有需要策略回测的记录：backtest_status为pending或tracking的记录
	q := gdb.Where("(backtest_status = ? OR backtest_status = ?) AND recommended_at <= ?",
		"pending", "tracking", time.Now().UTC()).
		Order("recommended_at ASC")

	if limit > 0 {
		q = q.Limit(limit)
	}

	err := q.Find(&perfs).Error
	return perfs, err
}

// GetPerformancesNeedingUpdate 统一查询需要更新的记录（优化：合并查询）
// 返回需要实时更新的记录和需要回测更新的记录
func GetPerformancesNeedingUpdate(gdb *gorm.DB, limit int) (realtime []RecommendationPerformance, backtest []RecommendationPerformance, err error) {
	now := time.Now().UTC()

	// 一次性查询所有需要更新的记录
	var allPerfs []RecommendationPerformance
	q := gdb.Where("(status = ? AND (current_price IS NULL OR last_updated_at IS NULL OR last_updated_at < ?)) OR (backtest_status IN ? AND recommended_at <= ?)",
		"tracking",
		now.Add(-10*time.Minute), // 10分钟未更新
		[]string{"pending", "tracking"},
		now,
	).Order("recommended_at ASC")

	if limit > 0 {
		q = q.Limit(limit * 2) // 多查询一些，因为会分成两类
	}

	if err := q.Find(&allPerfs).Error; err != nil {
		return nil, nil, err
	}

	// 分类：实时更新 vs 回测更新
	realtimeMap := make(map[uint]bool)
	backtestMap := make(map[uint]bool)

	for _, perf := range allPerfs {
		// 需要实时更新：状态为tracking且需要更新当前价格
		needsRealtime := perf.Status == "tracking" &&
			(perf.CurrentPrice == nil || perf.LastUpdatedAt == nil || perf.LastUpdatedAt.Before(now.Add(-10*time.Minute)))

		// 需要回测更新：回测状态为pending或tracking
		needsBacktest := (perf.BacktestStatus == "pending" || perf.BacktestStatus == "tracking") &&
			perf.RecommendedAt.Before(now)

		if needsRealtime {
			realtimeMap[perf.ID] = true
		}
		if needsBacktest {
			backtestMap[perf.ID] = true
		}
	}

	// 构建结果
	realtime = make([]RecommendationPerformance, 0, len(realtimeMap))
	backtest = make([]RecommendationPerformance, 0, len(backtestMap))

	for _, perf := range allPerfs {
		if realtimeMap[perf.ID] {
			realtime = append(realtime, perf)
		}
		if backtestMap[perf.ID] {
			backtest = append(backtest, perf)
		}
	}

	// 限制数量
	if limit > 0 {
		if len(realtime) > limit {
			realtime = realtime[:limit]
		}
		if len(backtest) > limit {
			backtest = backtest[:limit]
		}
	}

	return realtime, backtest, nil
}

// GetPerformanceStats 获取表现统计（优化：使用单次查询）
func GetPerformanceStats(gdb *gorm.DB, days int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// 时间范围
	startTime := time.Now().UTC().AddDate(0, 0, -days)

	// 优化：使用单次查询获取所有统计信息（使用子查询和聚合函数）
	type StatsResult struct {
		Total        int64   `gorm:"column:total"`
		Completed24h int64   `gorm:"column:completed_24h"`
		AvgReturn24h float64 `gorm:"column:avg_return_24h"`
		AvgReturn7d  float64 `gorm:"column:avg_return_7d"`
		AvgReturn30d float64 `gorm:"column:avg_return_30d"`
		WinRate24h   float64 `gorm:"column:win_rate_24h"`
		WinRate7d    float64 `gorm:"column:win_rate_7d"`
		WinRate30d   float64 `gorm:"column:win_rate_30d"`
		MaxGain      float64 `gorm:"column:max_gain"`
		MaxDrawdown  float64 `gorm:"column:max_drawdown"`
		// 策略相关统计
		StrategyCompleted int64   `gorm:"column:strategy_completed"`
		AvgStrategyReturn float64 `gorm:"column:avg_strategy_return"`
		StrategyWinRate   float64 `gorm:"column:strategy_win_rate"`
		AvgHoldingPeriod  float64 `gorm:"column:avg_holding_period"`
	}

	var result StatsResult
	err := gdb.Model(&RecommendationPerformance{}).
		Select(`
			COUNT(*) as total,
			COUNT(return_24h) as completed_24h,
			COALESCE(AVG(return_24h), 0) as avg_return_24h,
			COALESCE(AVG(return_7d), 0) as avg_return_7d,
			COALESCE(AVG(return_30d), 0) as avg_return_30d,
			COALESCE(SUM(CASE WHEN return_24h > 0 THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(return_24h), 0), 0) as win_rate_24h,
			COALESCE(SUM(CASE WHEN return_7d > 0 THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(return_7d), 0), 0) as win_rate_7d,
			COALESCE(SUM(CASE WHEN return_30d > 0 THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(return_30d), 0), 0) as win_rate_30d,
			COALESCE(MAX(max_gain), 0) as max_gain,
			COALESCE(MIN(max_drawdown), 0) as max_drawdown,
			-- 策略相关统计
			COUNT(actual_return) as strategy_completed,
			COALESCE(AVG(actual_return), 0) as avg_strategy_return,
			COALESCE(SUM(CASE WHEN actual_return > 0 THEN 1 ELSE 0 END) * 100.0 / NULLIF(COUNT(actual_return), 0), 0) as strategy_win_rate,
			COALESCE(AVG(holding_period), 0) as avg_holding_period
		`).
		Where("recommended_at >= ?", startTime).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	stats["total"] = result.Total
	stats["completed_24h"] = result.Completed24h
	stats["avg_return_24h"] = result.AvgReturn24h
	stats["avg_return_7d"] = result.AvgReturn7d
	stats["avg_return_30d"] = result.AvgReturn30d
	stats["win_rate_24h"] = result.WinRate24h
	stats["win_rate_7d"] = result.WinRate7d
	stats["win_rate_30d"] = result.WinRate30d
	stats["max_gain"] = result.MaxGain
	stats["max_drawdown"] = result.MaxDrawdown

	// 策略相关统计
	stats["strategy_completed"] = result.StrategyCompleted
	stats["avg_strategy_return"] = result.AvgStrategyReturn
	stats["strategy_win_rate"] = result.StrategyWinRate
	stats["avg_holding_period"] = result.AvgHoldingPeriod

	return stats, nil
}

// GetFactorPerformanceStats 获取各因子表现统计（用于反馈循环）
func GetFactorPerformanceStats(gdb *gorm.DB, days int) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	startTime := time.Now().UTC().AddDate(0, 0, -days)

	// 按因子得分分组统计
	type FactorStats struct {
		FactorName string
		AvgScore   float64
		AvgReturn  float64
		WinRate    float64
		Count      int64
	}

	// 市场表现因子
	var marketStats FactorStats
	gdb.Model(&RecommendationPerformance{}).
		Where("recommended_at >= ? AND return_24h IS NOT NULL", startTime).
		Select("COALESCE(AVG(market_score), 0) as avg_score, COALESCE(AVG(return_24h), 0) as avg_return, COUNT(*) as count").
		Scan(&marketStats)
	marketStats.FactorName = "market"
	if marketStats.Count > 0 {
		var winCount int64
		gdb.Model(&RecommendationPerformance{}).
			Where("recommended_at >= ? AND return_24h IS NOT NULL AND return_24h > 0", startTime).
			Count(&winCount)
		marketStats.WinRate = float64(winCount) / float64(marketStats.Count) * 100
	}

	// 资金流因子
	var flowStats FactorStats
	gdb.Model(&RecommendationPerformance{}).
		Where("recommended_at >= ? AND return_24h IS NOT NULL", startTime).
		Select("COALESCE(AVG(flow_score), 0) as avg_score, COALESCE(AVG(return_24h), 0) as avg_return, COUNT(*) as count").
		Scan(&flowStats)
	flowStats.FactorName = "flow"
	if flowStats.Count > 0 {
		var winCount int64
		gdb.Model(&RecommendationPerformance{}).
			Where("recommended_at >= ? AND return_24h IS NOT NULL AND return_24h > 0", startTime).
			Count(&winCount)
		flowStats.WinRate = float64(winCount) / float64(flowStats.Count) * 100
	}

	// 热度因子
	var heatStats FactorStats
	gdb.Model(&RecommendationPerformance{}).
		Where("recommended_at >= ? AND return_24h IS NOT NULL", startTime).
		Select("COALESCE(AVG(heat_score), 0) as avg_score, COALESCE(AVG(return_24h), 0) as avg_return, COUNT(*) as count").
		Scan(&heatStats)
	heatStats.FactorName = "heat"
	if heatStats.Count > 0 {
		var winCount int64
		gdb.Model(&RecommendationPerformance{}).
			Where("recommended_at >= ? AND return_24h IS NOT NULL AND return_24h > 0", startTime).
			Count(&winCount)
		heatStats.WinRate = float64(winCount) / float64(heatStats.Count) * 100
	}

	// 事件因子
	var eventStats FactorStats
	gdb.Model(&RecommendationPerformance{}).
		Where("recommended_at >= ? AND return_24h IS NOT NULL", startTime).
		Select("COALESCE(AVG(event_score), 0) as avg_score, COALESCE(AVG(return_24h), 0) as avg_return, COUNT(*) as count").
		Scan(&eventStats)
	eventStats.FactorName = "event"
	if eventStats.Count > 0 {
		var winCount int64
		gdb.Model(&RecommendationPerformance{}).
			Where("recommended_at >= ? AND return_24h IS NOT NULL AND return_24h > 0", startTime).
			Count(&winCount)
		eventStats.WinRate = float64(winCount) / float64(eventStats.Count) * 100
	}

	// 情绪因子
	var sentimentStats FactorStats
	gdb.Model(&RecommendationPerformance{}).
		Where("recommended_at >= ? AND return_24h IS NOT NULL", startTime).
		Select("COALESCE(AVG(sentiment_score), 0) as avg_score, COALESCE(AVG(return_24h), 0) as avg_return, COUNT(*) as count").
		Scan(&sentimentStats)
	sentimentStats.FactorName = "sentiment"
	if sentimentStats.Count > 0 {
		var winCount int64
		gdb.Model(&RecommendationPerformance{}).
			Where("recommended_at >= ? AND return_24h IS NOT NULL AND return_24h > 0", startTime).
			Count(&winCount)
		sentimentStats.WinRate = float64(winCount) / float64(sentimentStats.Count) * 100
	}

	stats["factors"] = []FactorStats{
		marketStats,
		flowStats,
		heatStats,
		eventStats,
		sentimentStats,
	}

	return stats, nil
}
